package cliapp

import (
	"context"
	"database/sql"
	"fmt"
)

func ensureDelegationWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, nowText string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func ensureDelegationActorPrincipal(ctx context.Context, tx *sql.Tx, workspaceID string, actorID string, nowText string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, actorID, workspaceID, actorID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure actor %s: %w", actorID, err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(workspace_id, actor_id) DO UPDATE SET updated_at = excluded.updated_at
	`, "wp-"+workspaceID+"-"+actorID, workspaceID, actorID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure workspace principal %s: %w", actorID, err)
	}
	return nil
}
