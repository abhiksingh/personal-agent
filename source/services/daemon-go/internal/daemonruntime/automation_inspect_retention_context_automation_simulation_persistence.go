package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func ensureSimulatedCommEvent(ctx context.Context, db *sql.DB, input simulatedCommEventInput) error {
	workspace := normalizeWorkspaceID(input.WorkspaceID)
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		return fmt.Errorf("event id is required")
	}

	threadID := strings.TrimSpace(input.ThreadID)
	if threadID == "" {
		threadID = "thread-" + eventID
	}

	channel := strings.TrimSpace(input.Channel)
	if channel == "" {
		channel = "imessage"
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		body = "automation test event"
	}
	sender := strings.TrimSpace(input.Sender)
	if sender == "" {
		sender = "sender@example.com"
	}
	eventType := strings.ToUpper(strings.TrimSpace(input.EventType))
	if eventType == "" {
		eventType = "MESSAGE"
	}
	direction := strings.ToUpper(strings.TrimSpace(input.Direction))
	if direction == "" {
		direction = "INBOUND"
	}
	connectorID := simulatedConnectorIDForChannel(channel)

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = now
	}
	occurredAtText := occurredAt.Format(time.RFC3339Nano)
	addressID := "addr-from-" + eventID

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(ctx, tx, workspace, nowText); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			channel = excluded.channel,
			connector_id = excluded.connector_id,
			updated_at = excluded.updated_at
	`, threadID, workspace, channel, connectorID, threadID, threadID, nowText, nowText); err != nil {
		return fmt.Errorf("upsert comm thread: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, eventID, workspace, threadID, connectorID, eventType, direction, boolToInt(input.AssistantEmitted), occurredAtText, body, nowText); err != nil {
		return fmt.Errorf("upsert comm event: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		VALUES (?, ?, 'FROM', ?, 'Simulation Sender', 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			event_id = excluded.event_id,
			address_role = excluded.address_role,
			address_value = excluded.address_value,
			display_name = excluded.display_name,
			position = excluded.position
	`, addressID, eventID, sender, nowText); err != nil {
		return fmt.Errorf("upsert comm event address: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func simulatedConnectorIDForChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "twilio", "voice":
		return "twilio"
	case "message":
		return "imessage"
	case "mail":
		return "mail"
	case "calendar":
		return "calendar"
	case "browser":
		return "browser"
	case "finder":
		return "finder"
	case "app":
		return "builtin.app"
	default:
		return strings.ToLower(strings.TrimSpace(channel))
	}
}
