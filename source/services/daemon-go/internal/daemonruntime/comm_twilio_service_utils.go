package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

func ensureCommWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, now string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func daemonRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func lookupThreadByProviderMessage(ctx context.Context, db *sql.DB, workspace string, providerMessageID string) (string, error) {
	var threadID string
	err := db.QueryRowContext(ctx, `
		SELECT ce.thread_id
		FROM comm_provider_messages cpm
		JOIN comm_events ce ON ce.id = cpm.event_id
		WHERE cpm.workspace_id = ?
		  AND cpm.provider = 'twilio'
		  AND cpm.provider_message_id = ?
		LIMIT 1
	`, workspace, strings.TrimSpace(providerMessageID)).Scan(&threadID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(threadID), nil
}

func lookupThreadByCallSID(ctx context.Context, db *sql.DB, workspace string, callSID string) (string, error) {
	var threadID string
	err := db.QueryRowContext(ctx, `
		SELECT thread_id
		FROM comm_call_sessions
		WHERE workspace_id = ? AND provider_call_id = ?
		LIMIT 1
	`, workspace, strings.TrimSpace(callSID)).Scan(&threadID)
	if err != nil {
		return "", fmt.Errorf("load call session thread: %w", err)
	}
	return strings.TrimSpace(threadID), nil
}
