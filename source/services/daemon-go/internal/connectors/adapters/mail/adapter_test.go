package mail

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/persistence/migrator"
)

func TestExecuteStepDraftSendReplyPersistedAndIdempotent(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("mail.test")
	baseCtx := connectorcontract.ExecutionContext{
		WorkspaceID:      "ws_mail",
		TaskID:           "task-1",
		RunID:            "run-1",
		CorrelationID:    "corr-1",
		RequestedByActor: "actor.requester",
		SubjectPrincipal: "actor.subject",
		ActingAsActor:    "actor.subject",
	}

	draftCtx := baseCtx
	draftCtx.StepID = "step-draft-1"
	draftStep := connectorcontract.TaskStep{
		ID:            "step-draft-1",
		CapabilityKey: CapabilityDraft,
		Name:          "Draft email",
		Input: map[string]any{
			"recipient": "sam@example.com",
			"subject":   "Draft update",
			"body":      "Draft body",
		},
	}
	draftFirst, err := adapter.ExecuteStep(context.Background(), draftCtx, draftStep)
	if err != nil {
		t.Fatalf("execute draft step: %v", err)
	}
	draftSecond, err := adapter.ExecuteStep(context.Background(), draftCtx, draftStep)
	if err != nil {
		t.Fatalf("execute draft step second time: %v", err)
	}
	if draftFirst.Evidence["draft_id"] == "" {
		t.Fatalf("expected draft_id evidence")
	}
	if draftFirst.Evidence["draft_id"] != draftSecond.Evidence["draft_id"] {
		t.Fatalf("expected idempotent draft_id, got %q vs %q", draftFirst.Evidence["draft_id"], draftSecond.Evidence["draft_id"])
	}
	if draftFirst.Evidence["provider"] != "apple-mail-dry-run" {
		t.Fatalf("expected apple-mail-dry-run provider, got %q", draftFirst.Evidence["provider"])
	}
	if draftFirst.Evidence["transport"] != "mail_dry_run" {
		t.Fatalf("expected mail_dry_run transport, got %q", draftFirst.Evidence["transport"])
	}
	if _, err := os.Stat(draftFirst.Evidence["record_path"]); err != nil {
		t.Fatalf("expected draft record file to exist: %v", err)
	}

	sendCtx := baseCtx
	sendCtx.StepID = "step-send-1"
	sendStep := connectorcontract.TaskStep{
		ID:            "step-send-1",
		CapabilityKey: CapabilitySend,
		Name:          "Send email",
		Input: map[string]any{
			"recipient": "sam@example.com",
			"subject":   "Status update",
			"body":      "Send body",
		},
	}
	sendResult, err := adapter.ExecuteStep(context.Background(), sendCtx, sendStep)
	if err != nil {
		t.Fatalf("execute send step: %v", err)
	}
	if sendResult.Evidence["message_id"] == "" {
		t.Fatalf("expected message_id evidence")
	}
	if _, err := os.Stat(sendResult.Evidence["record_path"]); err != nil {
		t.Fatalf("expected send record file to exist: %v", err)
	}

	replyCtx := baseCtx
	replyCtx.StepID = "step-reply-1"
	replyStep := connectorcontract.TaskStep{
		ID:            "step-reply-1",
		CapabilityKey: CapabilityReply,
		Name:          "Reply to thread",
		Input: map[string]any{
			"recipient": "sam@example.com",
			"subject":   "Re: Thread",
			"body":      "Reply body",
		},
	}
	replyResult, err := adapter.ExecuteStep(context.Background(), replyCtx, replyStep)
	if err != nil {
		t.Fatalf("execute reply step: %v", err)
	}
	if replyResult.Evidence["reply_id"] == "" || replyResult.Evidence["thread_id"] == "" {
		t.Fatalf("expected reply evidence to include reply_id and thread_id")
	}
	if _, err := os.Stat(replyResult.Evidence["record_path"]); err != nil {
		t.Fatalf("expected reply record file to exist: %v", err)
	}
}

func TestExecuteStepUnreadSummaryReturnsUnreadItems(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")

	dbPath := setupMailUnreadSummaryDB(t)
	adapter := NewAdapterWithDBPath("mail.test", dbPath)
	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws_mail",
		TaskID:      "task-mail-unread",
		RunID:       "run-mail-unread",
		StepID:      "step-mail-unread",
	}
	step := connectorcontract.TaskStep{
		ID:            "step-mail-unread",
		CapabilityKey: CapabilityUnreadSummary,
		Name:          "Summarize unread inbox mail",
		Input: map[string]any{
			"limit": 1,
		},
	}

	first, err := adapter.ExecuteStep(context.Background(), execCtx, step)
	if err != nil {
		t.Fatalf("execute unread summary step: %v", err)
	}
	second, err := adapter.ExecuteStep(context.Background(), execCtx, step)
	if err != nil {
		t.Fatalf("execute unread summary step second time: %v", err)
	}

	if first.Status != "completed" {
		t.Fatalf("expected completed status, got %s", first.Status)
	}
	if first.Evidence["unread_count"] != "2" {
		t.Fatalf("expected unread_count=2 evidence, got %q", first.Evidence["unread_count"])
	}
	if first.Evidence["returned_count"] != "1" {
		t.Fatalf("expected returned_count=1 evidence, got %q", first.Evidence["returned_count"])
	}
	if first.Summary != second.Summary {
		t.Fatalf("expected cached summary text to match first run, got %q vs %q", first.Summary, second.Summary)
	}
	if first.Evidence["unread_count"] != second.Evidence["unread_count"] {
		t.Fatalf("expected cached unread_count evidence to match first run, got %q vs %q", first.Evidence["unread_count"], second.Evidence["unread_count"])
	}

	unreadCount, err := outputInt(first.Output["unread_count"])
	if err != nil {
		t.Fatalf("parse output unread_count: %v", err)
	}
	if unreadCount != 2 {
		t.Fatalf("expected output unread_count=2, got %d", unreadCount)
	}
	items, ok := first.Output["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected output items []map[string]any, got %T", first.Output["items"])
	}
	if len(items) != 1 {
		t.Fatalf("expected one unread item due limit=1, got %d", len(items))
	}
	if items[0]["event_id"] != "event-unread-2" {
		t.Fatalf("expected most recent unread event event-unread-2, got %#v", items[0]["event_id"])
	}
}

func outputInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

func TestExecuteStepRejectsUnsupportedCapability(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("mail.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_mail",
		StepID:      "step-unsupported",
	}, connectorcontract.TaskStep{
		ID:            "step-unsupported",
		CapabilityKey: "mail_unknown",
		Name:          "Unsupported mail step",
	})
	if err == nil {
		t.Fatalf("expected unsupported capability error")
	}
}

func TestHealthCheckRequiresDarwinUnlessDryRun(t *testing.T) {
	adapter := NewAdapter("mail.test")
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")
	if err := adapter.HealthCheck(context.Background()); err != nil {
		t.Fatalf("expected dry-run health check to pass, got %v", err)
	}
}

func setupMailUnreadSummaryDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_mail', 'WS Mail', 'ACTIVE', '2026-02-28T00:00:00Z', '2026-02-28T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at) VALUES ('thread-read', 'ws_mail', 'mail', 'mail', 'thread-read', 'Read thread', '2026-02-28T00:00:00Z', '2026-02-28T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at) VALUES ('thread-unread-1', 'ws_mail', 'mail', 'mail', 'thread-unread-1', 'Unread thread one', '2026-02-28T00:00:00Z', '2026-02-28T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at) VALUES ('thread-unread-2', 'ws_mail', 'mail', 'mail', 'thread-unread-2', 'Unread thread two', '2026-02-28T00:00:00Z', '2026-02-28T00:00:00Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-read-in', 'ws_mail', 'thread-read', 'mail', 'MESSAGE', 'INBOUND', 0, '2026-02-28T09:00:00Z', 'Read Subject\n\nalready handled body', '2026-02-28T09:00:00Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-read-out', 'ws_mail', 'thread-read', 'mail', 'MESSAGE', 'OUTBOUND', 1, '2026-02-28T10:00:00Z', 'assistant reply', '2026-02-28T10:00:00Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-unread-1', 'ws_mail', 'thread-unread-1', 'mail', 'MESSAGE', 'INBOUND', 0, '2026-02-28T11:00:00Z', 'Unread Subject One\n\nbody one', '2026-02-28T11:00:00Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-unread-2', 'ws_mail', 'thread-unread-2', 'mail', 'MESSAGE', 'INBOUND', 0, '2026-02-28T12:00:00Z', 'Unread Subject Two\n\nbody two', '2026-02-28T12:00:00Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr-read-from', 'event-read-in', 'FROM', 'reader@example.com', NULL, 0, '2026-02-28T09:00:00Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr-unread1-from', 'event-unread-1', 'FROM', 'one@example.com', NULL, 0, '2026-02-28T11:00:00Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr-unread2-from', 'event-unread-2', 'FROM', 'two@example.com', NULL, 0, '2026-02-28T12:00:00Z')`,
		`INSERT INTO email_event_meta(id, event_id, message_id, in_reply_to, references_header, created_at) VALUES ('meta-unread-1', 'event-unread-1', '<unread-1@example.com>', NULL, NULL, '2026-02-28T11:00:00Z')`,
		`INSERT INTO email_event_meta(id, event_id, message_id, in_reply_to, references_header, created_at) VALUES ('meta-unread-2', 'event-unread-2', '<unread-2@example.com>', NULL, NULL, '2026-02-28T12:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed unread summary fixture: %v\nstatement: %s", err, statement)
		}
	}

	return dbPath
}
