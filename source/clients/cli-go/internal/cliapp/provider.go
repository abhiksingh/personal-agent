package cliapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"personalagent/runtime/internal/filesecurity"
	"personalagent/runtime/internal/persistence/migrator"
	"personalagent/runtime/internal/runtimepaths"

	_ "modernc.org/sqlite"
)

func resolveRuntimeDBPath(explicitPath string) (string, error) {
	trimmed := strings.TrimSpace(explicitPath)
	if trimmed != "" {
		return trimmed, nil
	}

	envPath := strings.TrimSpace(os.Getenv("PERSONAL_AGENT_DB"))
	if envPath != "" {
		return envPath, nil
	}

	defaultPath, err := runtimepaths.DefaultDBPath()
	if err != nil {
		return "", err
	}
	return defaultPath, nil
}

func openRuntimeDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("db path is required")
	}
	if err := filesecurity.EnsurePrivateDir(filepath.Dir(dbPath)); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if _, err := migrator.Apply(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	if err := filesecurity.EnsurePrivateFile(dbPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("harden db file permissions: %w", err)
	}
	return db, nil
}
