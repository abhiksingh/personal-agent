package daemonruntime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func persistMessagesInboundEvent(ctx context.Context, db *sql.DB, input messagesIngestPersistInput) (messagesIngestPersistResult, error) {
	if db == nil {
		return messagesIngestPersistResult{}, fmt.Errorf("db is required")
	}
	workspace := normalizeWorkspaceID(input.WorkspaceID)
	source := strings.TrimSpace(input.Source)
	if source == "" {
		return messagesIngestPersistResult{}, fmt.Errorf("messages ingest source is required")
	}
	sourceScope := strings.TrimSpace(input.SourceScope)
	if sourceScope == "" {
		return messagesIngestPersistResult{}, fmt.Errorf("messages ingest source scope is required")
	}
	sourceEventID := strings.TrimSpace(input.SourceEventID)
	if sourceEventID == "" {
		return messagesIngestPersistResult{}, fmt.Errorf("messages source_event_id is required")
	}

	senderAddress := strings.TrimSpace(input.SenderAddress)
	localAddress := strings.TrimSpace(input.LocalAddress)
	bodyText := strings.TrimSpace(input.BodyText)
	occurredAtText := normalizeOccurredAt(input.OccurredAt)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	threadExternalRef := firstNonEmpty(strings.TrimSpace(input.ExternalThreadID), senderAddress, sourceEventID)
	threadID := deterministicMessagesID("thread", workspace, source, sourceScope, threadExternalRef)
	eventID := deterministicMessagesID("event", workspace, source, sourceEventID)
	receiptID := deterministicMessagesID("receipt", workspace, source, sourceEventID)
	payloadHash := hashMessagesPayload(sourceScope, sourceEventID, senderAddress, localAddress, bodyText, occurredAtText)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return messagesIngestPersistResult{}, fmt.Errorf("begin messages ingest tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return messagesIngestPersistResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_ingest_receipts(
			id, workspace_id, source, source_scope, source_event_id, source_cursor,
			trust_state, event_id, payload_hash, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, 'accepted', NULL, ?, ?, ?)
	`, receiptID, workspace, source, sourceScope, sourceEventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAtText, now); err != nil {
		if !isUniqueConstraintError(err) {
			return messagesIngestPersistResult{}, fmt.Errorf("insert comm ingest receipt: %w", err)
		}
		existingEventID, existingThreadID, loadErr := loadExistingCommIngestReceipt(ctx, tx, workspace, source, sourceEventID)
		if loadErr != nil {
			return messagesIngestPersistResult{}, loadErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return messagesIngestPersistResult{}, fmt.Errorf("commit replay messages ingest tx: %w", commitErr)
		}
		return messagesIngestPersistResult{
			EventID:  firstNonEmpty(existingEventID, eventID),
			ThreadID: firstNonEmpty(existingThreadID, threadID),
			Replayed: true,
		}, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, 'message', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			title = excluded.title,
			updated_at = excluded.updated_at
	`, threadID, workspace, messagesConnectorID, nullableText(threadExternalRef), nullableText(firstNonEmpty(senderAddress, threadExternalRef)), now, now); err != nil {
		return messagesIngestPersistResult{}, fmt.Errorf("upsert messages thread: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(
			id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
		) VALUES (?, ?, ?, ?, 'MESSAGE', 'INBOUND', 0, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, eventID, workspace, threadID, messagesConnectorID, occurredAtText, nullableText(bodyText), now); err != nil {
		return messagesIngestPersistResult{}, fmt.Errorf("upsert messages comm event: %w", err)
	}

	if senderAddress != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
			VALUES (?, ?, 'FROM', ?, NULL, 0, ?)
			ON CONFLICT(id) DO UPDATE SET
				event_id = excluded.event_id,
				address_role = excluded.address_role,
				address_value = excluded.address_value,
				position = excluded.position
		`, deterministicMessagesID("event-address", eventID, "from", senderAddress), eventID, senderAddress, now); err != nil {
			return messagesIngestPersistResult{}, fmt.Errorf("upsert messages from address: %w", err)
		}
	}
	if localAddress != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
			VALUES (?, ?, 'TO', ?, NULL, 0, ?)
			ON CONFLICT(id) DO UPDATE SET
				event_id = excluded.event_id,
				address_role = excluded.address_role,
				address_value = excluded.address_value,
				position = excluded.position
		`, deterministicMessagesID("event-address", eventID, "to", localAddress), eventID, localAddress, now); err != nil {
			return messagesIngestPersistResult{}, fmt.Errorf("upsert messages to address: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_ingest_receipts
		SET event_id = ?, source_cursor = ?, payload_hash = ?, received_at = ?
		WHERE workspace_id = ? AND source = ? AND source_event_id = ?
	`, eventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAtText, workspace, source, sourceEventID); err != nil {
		return messagesIngestPersistResult{}, fmt.Errorf("update messages ingest receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return messagesIngestPersistResult{}, fmt.Errorf("commit messages ingest tx: %w", err)
	}
	return messagesIngestPersistResult{
		EventID:  eventID,
		ThreadID: threadID,
		Replayed: false,
	}, nil
}

func loadExistingCommIngestReceipt(ctx context.Context, tx *sql.Tx, workspaceID string, source string, sourceEventID string) (string, string, error) {
	var eventID string
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(event_id, '')
		FROM comm_ingest_receipts
		WHERE workspace_id = ? AND source = ? AND source_event_id = ?
		LIMIT 1
	`, workspaceID, source, sourceEventID).Scan(&eventID)
	if err != nil {
		return "", "", fmt.Errorf("load existing comm ingest receipt: %w", err)
	}

	threadID := ""
	if strings.TrimSpace(eventID) != "" {
		_ = tx.QueryRowContext(ctx, `
			SELECT COALESCE(thread_id, '')
			FROM comm_events
			WHERE id = ?
			LIMIT 1
		`, eventID).Scan(&threadID)
	}
	return strings.TrimSpace(eventID), strings.TrimSpace(threadID), nil
}

func normalizeOccurredAt(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return parsed.UTC().Format(time.RFC3339Nano)
}

func deterministicMessagesID(prefix string, parts ...string) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(strings.TrimSpace(prefix)))
	for _, part := range parts {
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(strings.TrimSpace(part)))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func hashMessagesPayload(parts ...string) string {
	digest := sha256.New()
	for _, part := range parts {
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(strings.TrimSpace(part)))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "unique constraint failed")
}
