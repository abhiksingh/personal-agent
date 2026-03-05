package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func loadCommIngestCursor(ctx context.Context, db *sql.DB, workspaceID string, source string, sourceScope string) (string, error) {
	var cursor string
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(cursor_value, '')
		FROM comm_ingest_cursors
		WHERE workspace_id = ? AND source = ? AND source_scope = ?
		LIMIT 1
	`, workspaceID, source, sourceScope).Scan(&cursor)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load comm ingest cursor: %w", err)
	}
	return strings.TrimSpace(cursor), nil
}

func upsertCommIngestCursor(ctx context.Context, db *sql.DB, workspaceID string, source string, sourceScope string, cursorValue string) error {
	workspace := normalizeWorkspaceID(workspaceID)
	source = strings.TrimSpace(source)
	sourceScope = strings.TrimSpace(sourceScope)
	cursorValue = strings.TrimSpace(cursorValue)
	if source == "" || sourceScope == "" || cursorValue == "" {
		return fmt.Errorf("workspace/source/source_scope/cursor_value are required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin cursor upsert tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return err
	}

	var existingCursor string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(cursor_value, '')
		FROM comm_ingest_cursors
		WHERE workspace_id = ? AND source = ? AND source_scope = ?
		LIMIT 1
	`, workspace, source, sourceScope).Scan(&existingCursor)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("query existing ingest cursor: %w", err)
	}

	if err == sql.ErrNoRows {
		cursorID := deterministicMessagesID("cursor", workspace, source, sourceScope)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_ingest_cursors(
				id, workspace_id, source, source_scope, cursor_value, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, cursorID, workspace, source, sourceScope, cursorValue, now, now); err != nil {
			return fmt.Errorf("insert ingest cursor: %w", err)
		}
	} else if compareCursorValue(cursorValue, strings.TrimSpace(existingCursor)) > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE comm_ingest_cursors
			SET cursor_value = ?, updated_at = ?
			WHERE workspace_id = ? AND source = ? AND source_scope = ?
		`, cursorValue, now, workspace, source, sourceScope); err != nil {
			return fmt.Errorf("update ingest cursor: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ingest cursor upsert: %w", err)
	}
	return nil
}

func upsertAutomationSourceSubscription(
	ctx context.Context,
	db *sql.DB,
	workspaceID string,
	source string,
	sourceScope string,
	lastCursor string,
	lastEventID string,
	lastError string,
) error {
	workspace := normalizeWorkspaceID(workspaceID)
	source = strings.TrimSpace(source)
	sourceScope = strings.TrimSpace(sourceScope)
	if source == "" || sourceScope == "" {
		return fmt.Errorf("source and source_scope are required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	subscriptionID := deterministicMessagesID("source-subscription", workspace, source, sourceScope)
	configJSON, _ := json.Marshal(map[string]any{
		"source": source,
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin source subscription tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO automation_source_subscriptions(
			id, workspace_id, source, source_scope, status, config_json,
			last_cursor, last_event_id, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, source, source_scope) DO UPDATE SET
			status = excluded.status,
			config_json = excluded.config_json,
			last_cursor = excluded.last_cursor,
			last_event_id = excluded.last_event_id,
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, subscriptionID, workspace, source, sourceScope, string(configJSON), nullableText(strings.TrimSpace(lastCursor)), nullableText(strings.TrimSpace(lastEventID)), nullableText(strings.TrimSpace(lastError)), now, now); err != nil {
		return fmt.Errorf("upsert automation source subscription: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit source subscription tx: %w", err)
	}
	return nil
}

func compareCursorValue(left string, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == right {
		return 0
	}
	if left == "" {
		return -1
	}
	if right == "" {
		return 1
	}
	leftInt, leftErr := strconv.ParseInt(left, 10, 64)
	rightInt, rightErr := strconv.ParseInt(right, 10, 64)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftInt < rightInt:
			return -1
		case leftInt > rightInt:
			return 1
		default:
			return 0
		}
	}
	if left < right {
		return -1
	}
	return 1
}
