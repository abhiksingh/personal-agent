package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"personalagent/runtime/internal/persistence/migrations"
)

type migrationFile struct {
	Name string
	Body string
}

func listMigrations() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrations.Files, "sql")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, readErr := fs.ReadFile(migrations.Files, "sql/"+entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), readErr)
		}
		files = append(files, migrationFile{Name: entry.Name(), Body: string(content)})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);`
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	versions := map[string]struct{}{}
	for rows.Next() {
		var version string
		if scanErr := rows.Scan(&version); scanErr != nil {
			return nil, fmt.Errorf("scan applied migration version: %w", scanErr)
		}
		versions[version] = struct{}{}
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", rows.Err())
	}
	return versions, nil
}

func versionFromName(name string) string {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return name
	}
	return parts[0]
}

// Apply applies pending migrations in deterministic filename order.
func Apply(ctx context.Context, db *sql.DB) ([]string, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}

	migrationsToApply, err := listMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return nil, err
	}

	appliedNow := make([]string, 0)
	for _, migration := range migrationsToApply {
		version := versionFromName(migration.Name)
		if _, exists := applied[version]; exists {
			continue
		}

		tx, beginErr := db.BeginTx(ctx, nil)
		if beginErr != nil {
			return nil, fmt.Errorf("begin migration %s: %w", migration.Name, beginErr)
		}

		if _, execErr := tx.ExecContext(ctx, migration.Body); execErr != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("apply migration %s: %w", migration.Name, execErr)
		}

		if _, insErr := tx.ExecContext(
			ctx,
			"INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)",
			version,
			time.Now().UTC().Format(time.RFC3339Nano),
		); insErr != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("record migration %s: %w", migration.Name, insErr)
		}

		if commitErr := tx.Commit(); commitErr != nil {
			return nil, fmt.Errorf("commit migration %s: %w", migration.Name, commitErr)
		}

		appliedNow = append(appliedNow, migration.Name)
	}

	return appliedNow, nil
}
