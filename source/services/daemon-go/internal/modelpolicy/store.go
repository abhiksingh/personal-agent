package modelpolicy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/workspaceid"
)

var (
	ErrModelNotFound         = errors.New("model not found in catalog")
	ErrRoutingPolicyNotFound = errors.New("routing policy not found")
)

type CatalogEntry struct {
	WorkspaceID string    `json:"workspace_id"`
	Provider    string    `json:"provider"`
	ModelKey    string    `json:"model_key"`
	Enabled     bool      `json:"enabled"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoutingPolicy struct {
	WorkspaceID string    `json:"workspace_id"`
	TaskClass   string    `json:"task_class"`
	Provider    string    `json:"provider"`
	ModelKey    string    `json:"model_key"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SQLiteStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *SQLiteStore) SeedDefaults(ctx context.Context, workspaceID string) error {
	workspace := normalizeWorkspaceID(workspaceID)
	now := s.now().Format(time.RFC3339Nano)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspace, now); err != nil {
		return err
	}

	for _, model := range DefaultCatalog() {
		if tombstoned, err := isCatalogEntryTombstoned(ctx, tx, workspace, model.Provider, model.ModelKey); err != nil {
			return err
		} else if tombstoned {
			continue
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model_catalog_entries (
				id,
				workspace_id,
				provider,
				model_key,
				enabled,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, 1, ?, ?)
			ON CONFLICT(workspace_id, provider, model_key) DO NOTHING
		`, catalogEntryID(workspace, model.Provider, model.ModelKey), workspace, model.Provider, model.ModelKey, now, now); err != nil {
			return fmt.Errorf("seed default model %s/%s: %w", model.Provider, model.ModelKey, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListCatalog(ctx context.Context, workspaceID string, provider string) ([]CatalogEntry, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	if err := s.SeedDefaults(ctx, workspace); err != nil {
		return nil, err
	}

	query := `
		SELECT provider, model_key, enabled, updated_at
		FROM model_catalog_entries
		WHERE workspace_id = ?
	`
	args := []any{workspace}
	if strings.TrimSpace(provider) != "" {
		normalizedProvider, err := providerconfig.NormalizeProvider(provider)
		if err != nil {
			return nil, err
		}
		query += " AND provider = ?"
		args = append(args, normalizedProvider)
	}
	query += " ORDER BY provider ASC, model_key ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query model catalog: %w", err)
	}
	defer rows.Close()

	entries := make([]CatalogEntry, 0)
	for rows.Next() {
		var (
			providerValue string
			modelKey      string
			enabledValue  int
			updatedAtRaw  string
		)
		if err := rows.Scan(&providerValue, &modelKey, &enabledValue, &updatedAtRaw); err != nil {
			return nil, fmt.Errorf("scan model catalog row: %w", err)
		}
		updatedAt, err := parseTimestamp(updatedAtRaw)
		if err != nil {
			return nil, err
		}
		entries = append(entries, CatalogEntry{
			WorkspaceID: workspace,
			Provider:    providerValue,
			ModelKey:    modelKey,
			Enabled:     enabledValue == 1,
			UpdatedAt:   updatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model catalog rows: %w", err)
	}
	return entries, nil
}

func (s *SQLiteStore) AddCatalogEntry(
	ctx context.Context,
	workspaceID string,
	provider string,
	modelKey string,
	enabled bool,
) (CatalogEntry, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedProvider, normalizedModel, err := normalizeProviderModel(provider, modelKey)
	if err != nil {
		return CatalogEntry{}, err
	}

	if err := s.SeedDefaults(ctx, workspace); err != nil {
		return CatalogEntry{}, err
	}

	now := s.now().Format(time.RFC3339Nano)
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspace, now); err != nil {
		return CatalogEntry{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO model_catalog_entries (
			id,
			workspace_id,
			provider,
			model_key,
			enabled,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, provider, model_key) DO UPDATE SET
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, catalogEntryID(workspace, normalizedProvider, normalizedModel), workspace, normalizedProvider, normalizedModel, enabledInt, now, now); err != nil {
		return CatalogEntry{}, fmt.Errorf("upsert model catalog entry: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM model_catalog_tombstones
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
	`, workspace, normalizedProvider, normalizedModel); err != nil {
		return CatalogEntry{}, fmt.Errorf("clear model catalog tombstone: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return CatalogEntry{}, fmt.Errorf("commit tx: %w", err)
	}
	return s.GetCatalogEntry(ctx, workspace, normalizedProvider, normalizedModel)
}

func (s *SQLiteStore) RemoveCatalogEntry(
	ctx context.Context,
	workspaceID string,
	provider string,
	modelKey string,
) (CatalogEntry, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedProvider, normalizedModel, err := normalizeProviderModel(provider, modelKey)
	if err != nil {
		return CatalogEntry{}, err
	}

	entry, err := s.GetCatalogEntry(ctx, workspace, normalizedProvider, normalizedModel)
	if err != nil {
		return CatalogEntry{}, err
	}

	now := s.now().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspace, now); err != nil {
		return CatalogEntry{}, err
	}

	result, err := tx.ExecContext(ctx, `
		DELETE FROM model_catalog_entries
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
	`, workspace, normalizedProvider, normalizedModel)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("delete model catalog entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("read deleted row count: %w", err)
	}
	if affected == 0 {
		return CatalogEntry{}, ErrModelNotFound
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM model_routing_policies
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
	`, workspace, normalizedProvider, normalizedModel); err != nil {
		return CatalogEntry{}, fmt.Errorf("delete model routing policies for removed model: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO model_catalog_tombstones (
			id,
			workspace_id,
			provider,
			model_key,
			created_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, provider, model_key) DO NOTHING
	`, catalogTombstoneID(workspace, normalizedProvider, normalizedModel), workspace, normalizedProvider, normalizedModel, now); err != nil {
		return CatalogEntry{}, fmt.Errorf("insert model catalog tombstone: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return CatalogEntry{}, fmt.Errorf("commit tx: %w", err)
	}

	entry.UpdatedAt, err = parseTimestamp(now)
	if err != nil {
		return CatalogEntry{}, err
	}
	return entry, nil
}

func (s *SQLiteStore) SetModelEnabled(ctx context.Context, workspaceID string, provider string, modelKey string, enabled bool) (CatalogEntry, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedProvider, normalizedModel, err := normalizeProviderModel(provider, modelKey)
	if err != nil {
		return CatalogEntry{}, err
	}

	if err := s.SeedDefaults(ctx, workspace); err != nil {
		return CatalogEntry{}, err
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE model_catalog_entries
		SET enabled = ?, updated_at = ?
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
	`, enabledInt, s.now().Format(time.RFC3339Nano), workspace, normalizedProvider, normalizedModel)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("update model enablement: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("read updated row count: %w", err)
	}
	if affected == 0 {
		return CatalogEntry{}, ErrModelNotFound
	}
	return s.GetCatalogEntry(ctx, workspace, normalizedProvider, normalizedModel)
}

func (s *SQLiteStore) GetCatalogEntry(ctx context.Context, workspaceID string, provider string, modelKey string) (CatalogEntry, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedProvider, normalizedModel, err := normalizeProviderModel(provider, modelKey)
	if err != nil {
		return CatalogEntry{}, err
	}

	if err := s.SeedDefaults(ctx, workspace); err != nil {
		return CatalogEntry{}, err
	}

	var (
		enabledValue int
		updatedAtRaw string
	)
	err = s.db.QueryRowContext(ctx, `
		SELECT enabled, updated_at
		FROM model_catalog_entries
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
	`, workspace, normalizedProvider, normalizedModel).Scan(&enabledValue, &updatedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return CatalogEntry{}, ErrModelNotFound
	}
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("query model catalog entry: %w", err)
	}

	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogEntry{}, err
	}
	return CatalogEntry{
		WorkspaceID: workspace,
		Provider:    normalizedProvider,
		ModelKey:    normalizedModel,
		Enabled:     enabledValue == 1,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *SQLiteStore) SetRoutingPolicy(ctx context.Context, workspaceID string, taskClass string, provider string, modelKey string) (RoutingPolicy, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedTaskClass := normalizeTaskClass(taskClass)
	normalizedProvider, normalizedModel, err := normalizeProviderModel(provider, modelKey)
	if err != nil {
		return RoutingPolicy{}, err
	}

	entry, err := s.GetCatalogEntry(ctx, workspace, normalizedProvider, normalizedModel)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if !entry.Enabled {
		return RoutingPolicy{}, fmt.Errorf("model %s/%s is disabled", normalizedProvider, normalizedModel)
	}

	now := s.now().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO model_routing_policies (
			id,
			workspace_id,
			task_class,
			provider,
			model_key,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, task_class) DO UPDATE SET
			provider = excluded.provider,
			model_key = excluded.model_key,
			updated_at = excluded.updated_at
	`, routingPolicyID(workspace, normalizedTaskClass), workspace, normalizedTaskClass, normalizedProvider, normalizedModel, now, now); err != nil {
		return RoutingPolicy{}, fmt.Errorf("upsert routing policy: %w", err)
	}
	return s.GetRoutingPolicy(ctx, workspace, normalizedTaskClass)
}

func (s *SQLiteStore) GetRoutingPolicy(ctx context.Context, workspaceID string, taskClass string) (RoutingPolicy, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedTaskClass := normalizeTaskClass(taskClass)

	var (
		providerValue string
		modelValue    string
		updatedAtRaw  string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT provider, model_key, updated_at
		FROM model_routing_policies
		WHERE workspace_id = ? AND task_class = ?
	`, workspace, normalizedTaskClass).Scan(&providerValue, &modelValue, &updatedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, ErrRoutingPolicyNotFound
	}
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("query routing policy: %w", err)
	}

	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return RoutingPolicy{}, err
	}
	return RoutingPolicy{
		WorkspaceID: workspace,
		TaskClass:   normalizedTaskClass,
		Provider:    providerValue,
		ModelKey:    modelValue,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *SQLiteStore) ListRoutingPolicies(ctx context.Context, workspaceID string) ([]RoutingPolicy, error) {
	workspace := normalizeWorkspaceID(workspaceID)

	rows, err := s.db.QueryContext(ctx, `
		SELECT task_class, provider, model_key, updated_at
		FROM model_routing_policies
		WHERE workspace_id = ?
		ORDER BY task_class ASC
	`, workspace)
	if err != nil {
		return nil, fmt.Errorf("query routing policies: %w", err)
	}
	defer rows.Close()

	policies := make([]RoutingPolicy, 0)
	for rows.Next() {
		var (
			taskClass    string
			providerName string
			modelKey     string
			updatedAtRaw string
		)
		if err := rows.Scan(&taskClass, &providerName, &modelKey, &updatedAtRaw); err != nil {
			return nil, fmt.Errorf("scan routing policy row: %w", err)
		}
		updatedAt, err := parseTimestamp(updatedAtRaw)
		if err != nil {
			return nil, err
		}
		policies = append(policies, RoutingPolicy{
			WorkspaceID: workspace,
			TaskClass:   taskClass,
			Provider:    providerName,
			ModelKey:    modelKey,
			UpdatedAt:   updatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing policy rows: %w", err)
	}
	return policies, nil
}

func normalizeProviderModel(provider string, modelKey string) (string, string, error) {
	normalizedProvider, err := providerconfig.NormalizeProvider(provider)
	if err != nil {
		return "", "", err
	}
	normalizedModel := strings.TrimSpace(modelKey)
	if normalizedModel == "" {
		return "", "", fmt.Errorf("model key is required")
	}
	return normalizedProvider, normalizedModel, nil
}

func isCatalogEntryTombstoned(ctx context.Context, tx *sql.Tx, workspaceID string, provider string, modelKey string) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM model_catalog_tombstones
		WHERE workspace_id = ? AND provider = ? AND model_key = ?
		LIMIT 1
	`, workspaceID, provider, modelKey).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query model catalog tombstone: %w", err)
	}
	return true, nil
}

func ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, now string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces (
			id,
			name,
			status,
			created_at,
			updated_at
		) VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("missing timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", value, err)
	}
	return parsed, nil
}

func normalizeWorkspaceID(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func normalizeTaskClass(taskClass string) string {
	trimmed := strings.ToLower(strings.TrimSpace(taskClass))
	if trimmed == "" {
		return TaskClassDefault
	}
	return trimmed
}

func catalogEntryID(workspaceID string, provider string, modelKey string) string {
	return fmt.Sprintf("model-catalog:%s:%s:%s", workspaceID, provider, modelKey)
}

func catalogTombstoneID(workspaceID string, provider string, modelKey string) string {
	return fmt.Sprintf("model-catalog-tombstone:%s:%s:%s", workspaceID, provider, modelKey)
}

func routingPolicyID(workspaceID string, taskClass string) string {
	return fmt.Sprintf("model-routing:%s:%s", workspaceID, taskClass)
}
