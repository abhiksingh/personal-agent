package twilio

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/workspaceid"
)

func upsertThread(ctx context.Context, tx *sql.Tx, threadID string, workspaceID string, channel string, connectorID string, externalRef string, title string, now string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			channel = excluded.channel,
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			title = excluded.title,
			updated_at = excluded.updated_at
	`, threadID, workspaceID, channel, strings.TrimSpace(connectorID), externalRef, nullable(title), now, now); err != nil {
		return fmt.Errorf("upsert comm thread: %w", err)
	}
	return nil
}

func upsertEventAddresses(ctx context.Context, tx *sql.Tx, eventID string, fromAddress string, toAddress string, now string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		VALUES (?, ?, 'FROM', ?, NULL, 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			event_id = excluded.event_id,
			address_role = excluded.address_role,
			address_value = excluded.address_value,
			position = excluded.position
	`, deterministicID("event-address", eventID, "from", fromAddress), eventID, fromAddress, now); err != nil {
		return fmt.Errorf("upsert from address: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		VALUES (?, ?, 'TO', ?, NULL, 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			event_id = excluded.event_id,
			address_role = excluded.address_role,
			address_value = excluded.address_value,
			position = excluded.position
	`, deterministicID("event-address", eventID, "to", toAddress), eventID, toAddress, now); err != nil {
		return fmt.Errorf("upsert to address: %w", err)
	}
	return nil
}

func ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, now string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now); err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

type existingReceipt struct {
	ReceiptID string
	EventID   string
}

func loadExistingReceipt(ctx context.Context, tx *sql.Tx, workspaceID string, provider string, providerEventID string) (existingReceipt, error) {
	var (
		receiptID string
		eventID   sql.NullString
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT id, event_id
		FROM comm_webhook_receipts
		WHERE workspace_id = ? AND provider = ? AND provider_event_id = ?
		LIMIT 1
	`, workspaceID, provider, providerEventID).Scan(&receiptID, &eventID); err != nil {
		return existingReceipt{}, fmt.Errorf("load existing receipt: %w", err)
	}
	return existingReceipt{
		ReceiptID: strings.TrimSpace(receiptID),
		EventID:   strings.TrimSpace(eventID.String),
	}, nil
}

func loadEventThreadID(ctx context.Context, tx *sql.Tx, eventID string) (string, error) {
	var threadID string
	if err := tx.QueryRowContext(ctx, `
		SELECT thread_id FROM comm_events WHERE id = ? LIMIT 1
	`, strings.TrimSpace(eventID)).Scan(&threadID); err != nil {
		return "", fmt.Errorf("load event thread id: %w", err)
	}
	return strings.TrimSpace(threadID), nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func normalizeWorkspace(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func normalizeAddress(address string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(address)), " ", "")
}

func smsExternalRef(localAddress string, remoteAddress string) string {
	return fmt.Sprintf("twilio:sms:%s:%s", normalizeAddress(localAddress), normalizeAddress(remoteAddress))
}

func deterministicID(parts ...string) string {
	joined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:16])
}

func hashPayload(values map[string]string, fallback string) string {
	if len(values) == 0 {
		trimmed := strings.TrimSpace(fallback)
		if trimmed == "" {
			return ""
		}
		sum := sha256.Sum256([]byte(trimmed))
		return hex.EncodeToString(sum[:])
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(values[key])
		builder.WriteString("&")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func marshalPayloadStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(encoded))
}

func marshalPayloadAnyMap(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(encoded))
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
