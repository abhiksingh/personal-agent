package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

func isActiveWorkspacePrincipal(ctx context.Context, queryable interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID string, actorID string) (bool, error) {
	var exists int
	if err := queryable.QueryRowContext(ctx, `
		SELECT 1
		FROM workspace_principals wp
		JOIN actors a
			ON a.id = wp.actor_id
			AND a.workspace_id = wp.workspace_id
		WHERE wp.workspace_id = ?
			AND wp.actor_id = ?
			AND UPPER(wp.status) = 'ACTIVE'
			AND UPPER(a.status) = 'ACTIVE'
		LIMIT 1
	`, workspaceID, strings.TrimSpace(actorID)).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("lookup workspace principal %q: %w", actorID, err)
	}
	return true, nil
}

func appendDelegationAuditEntry(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	eventType string,
	actorID string,
	actingAsActorID string,
	payload map[string]any,
) error {
	if tx == nil {
		return fmt.Errorf("append delegation audit entry: nil tx")
	}
	if strings.TrimSpace(eventType) == "" {
		return fmt.Errorf("append delegation audit entry: event type is required")
	}
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	auditID, err := delegationRandomID()
	if err != nil {
		return err
	}

	var payloadJSON any = nil
	if len(payload) > 0 {
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("marshal delegation audit payload: %w", marshalErr)
		}
		payloadJSON = string(encoded)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, NULL, NULL, ?, ?, ?, NULL, ?, ?)
	`, auditID, workspaceID, strings.TrimSpace(eventType), delegationNullableText(strings.TrimSpace(actorID)), delegationNullableText(strings.TrimSpace(actingAsActorID)), payloadJSON, nowText); err != nil {
		return fmt.Errorf("insert delegation audit entry: %w", err)
	}
	return nil
}

func delegationRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate delegation id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func delegationNullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
