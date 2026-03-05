package daemonruntime

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestCommTrustReceiptQueriesSupportFiltersPaginationAndAuditLinks(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommTrustReceiptFixtures(t, container.DB)

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	webhookPage1, err := service.ListCommWebhookReceipts(context.Background(), transport.CommWebhookReceiptListRequest{
		WorkspaceID: "ws1",
		Provider:    "twilio",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm webhook receipts page 1: %v", err)
	}
	if len(webhookPage1.Items) != 1 || webhookPage1.Items[0].ReceiptID != "wr-2" || !webhookPage1.HasMore {
		t.Fatalf("unexpected comm webhook page 1 payload: %+v", webhookPage1)
	}
	if webhookPage1.Items[0].TrustState != "accepted" || len(webhookPage1.Items[0].AuditLinks) == 0 || webhookPage1.Items[0].ThreadID != "thread-2" {
		t.Fatalf("expected webhook trust/audit/thread metadata, got %+v", webhookPage1.Items[0])
	}

	webhookPage2, err := service.ListCommWebhookReceipts(context.Background(), transport.CommWebhookReceiptListRequest{
		WorkspaceID:     "ws1",
		Provider:        "twilio",
		CursorCreatedAt: webhookPage1.NextCursorCreatedAt,
		CursorID:        webhookPage1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("list comm webhook receipts page 2: %v", err)
	}
	if len(webhookPage2.Items) != 1 || webhookPage2.Items[0].ReceiptID != "wr-1" || webhookPage2.Items[0].TrustState != "rejected" {
		t.Fatalf("unexpected comm webhook page 2 payload: %+v", webhookPage2)
	}

	webhookFiltered, err := service.ListCommWebhookReceipts(context.Background(), transport.CommWebhookReceiptListRequest{
		WorkspaceID:        "ws1",
		Provider:           "twilio",
		ProviderEventQuery: "accepted",
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("list filtered comm webhook receipts: %v", err)
	}
	if len(webhookFiltered.Items) != 1 || webhookFiltered.Items[0].ReceiptID != "wr-2" {
		t.Fatalf("unexpected filtered comm webhook payload: %+v", webhookFiltered)
	}

	ingestPage1, err := service.ListCommIngestReceipts(context.Background(), transport.CommIngestReceiptListRequest{
		WorkspaceID: "ws1",
		Source:      "apple_mail_rule",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm ingest receipts page 1: %v", err)
	}
	if len(ingestPage1.Items) != 1 || ingestPage1.Items[0].ReceiptID != "ir-2" || !ingestPage1.HasMore {
		t.Fatalf("unexpected comm ingest page 1 payload: %+v", ingestPage1)
	}
	if ingestPage1.Items[0].TrustState != "accepted" || len(ingestPage1.Items[0].AuditLinks) == 0 || ingestPage1.Items[0].ThreadID != "thread-2" {
		t.Fatalf("expected ingest trust/audit/thread metadata, got %+v", ingestPage1.Items[0])
	}

	ingestPage2, err := service.ListCommIngestReceipts(context.Background(), transport.CommIngestReceiptListRequest{
		WorkspaceID:     "ws1",
		Source:          "apple_mail_rule",
		CursorCreatedAt: ingestPage1.NextCursorCreatedAt,
		CursorID:        ingestPage1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("list comm ingest receipts page 2: %v", err)
	}
	if len(ingestPage2.Items) != 1 || ingestPage2.Items[0].ReceiptID != "ir-1" {
		t.Fatalf("unexpected comm ingest page 2 payload: %+v", ingestPage2)
	}

	ingestFiltered, err := service.ListCommIngestReceipts(context.Background(), transport.CommIngestReceiptListRequest{
		WorkspaceID: "ws1",
		TrustState:  "rejected",
		Source:      "apple_calendar_eventkit",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list filtered comm ingest receipts: %v", err)
	}
	if len(ingestFiltered.Items) != 1 || ingestFiltered.Items[0].ReceiptID != "ir-3" || ingestFiltered.Items[0].TrustState != "rejected" {
		t.Fatalf("unexpected filtered comm ingest payload: %+v", ingestFiltered)
	}
}

func TestCommTrustReceiptValidationErrors(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommTrustReceiptFixtures(t, container.DB)

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	_, err = service.ListCommWebhookReceipts(context.Background(), transport.CommWebhookReceiptListRequest{
		WorkspaceID: "ws1",
		CursorID:    "wr-2",
		Limit:       10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at is required") {
		t.Fatalf("expected cursor validation error for webhook receipts, got %v", err)
	}

	_, err = service.ListCommIngestReceipts(context.Background(), transport.CommIngestReceiptListRequest{
		WorkspaceID: "ws1",
		TrustState:  "unknown",
		Limit:       10,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported trust_state") {
		t.Fatalf("expected trust_state validation error, got %v", err)
	}

	_, err = service.ListCommIngestReceipts(context.Background(), transport.CommIngestReceiptListRequest{
		WorkspaceID:     "ws1",
		CursorCreatedAt: "not-a-timestamp",
		Limit:           10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at must be RFC3339 timestamp") {
		t.Fatalf("expected cursor timestamp validation error, got %v", err)
	}
}

func seedCommTrustReceiptFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at) VALUES ('thread-1', 'ws1', 'twilio_sms', 'thread-1', 'Thread 1', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at) VALUES ('thread-2', 'ws1', 'twilio_sms', 'thread-2', 'Thread 2', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-1', 'ws1', 'thread-1', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:01Z', 'hello one', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event-2', 'ws1', 'thread-2', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:02Z', 'hello two', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_webhook_receipts(id, workspace_id, provider, provider_event_id, signature_valid, signature_value, payload_hash, event_id, received_at, created_at) VALUES ('wr-1', 'ws1', 'twilio', 'SM-rejected-1', 0, '', 'hash-wr-1', NULL, '2026-02-25T00:00:01Z', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_webhook_receipts(id, workspace_id, provider, provider_event_id, signature_valid, signature_value, payload_hash, event_id, received_at, created_at) VALUES ('wr-2', 'ws1', 'twilio', 'SM-accepted-2', 1, 'sig-value', 'hash-wr-2', 'event-2', '2026-02-25T00:00:02Z', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_ingest_receipts(id, workspace_id, source, source_scope, source_event_id, source_cursor, trust_state, event_id, payload_hash, received_at, created_at) VALUES ('ir-1', 'ws1', 'apple_mail_rule', 'mail-rule-default', 'mail-event-1', '100', 'accepted', 'event-1', 'hash-ir-1', '2026-02-25T00:00:01Z', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_ingest_receipts(id, workspace_id, source, source_scope, source_event_id, source_cursor, trust_state, event_id, payload_hash, received_at, created_at) VALUES ('ir-2', 'ws1', 'apple_mail_rule', 'mail-rule-default', 'mail-event-2', '101', 'accepted', 'event-2', 'hash-ir-2', '2026-02-25T00:00:02Z', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_ingest_receipts(id, workspace_id, source, source_scope, source_event_id, source_cursor, trust_state, event_id, payload_hash, received_at, created_at) VALUES ('ir-3', 'ws1', 'apple_calendar_eventkit', 'calendar-default', 'calendar-event-1', '102', 'rejected', NULL, 'hash-ir-3', '2026-02-25T00:00:03Z', '2026-02-25T00:00:03Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit-wr-1', 'ws1', NULL, NULL, 'twilio_webhook_rejected_invalid_signature', NULL, NULL, NULL, '{"receipt_id":"wr-1","provider_event_id":"SM-rejected-1"}', '2026-02-25T00:00:01Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit-wr-2', 'ws1', NULL, NULL, 'twilio_webhook_received', NULL, NULL, NULL, '{"receipt_id":"wr-2","provider_event_id":"SM-accepted-2","event_id":"event-2"}', '2026-02-25T00:00:02Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit-ir-1', 'ws1', NULL, NULL, 'comm_ingest_received', NULL, NULL, NULL, '{"receipt_id":"ir-1","source_event_id":"mail-event-1","event_id":"event-1"}', '2026-02-25T00:00:01Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit-ir-2', 'ws1', NULL, NULL, 'comm_ingest_received', NULL, NULL, NULL, '{"receipt_id":"ir-2","source_event_id":"mail-event-2","event_id":"event-2"}', '2026-02-25T00:00:02Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit-ir-3', 'ws1', NULL, NULL, 'comm_ingest_rejected', NULL, NULL, NULL, '{"receipt_id":"ir-3","source_event_id":"calendar-event-1"}', '2026-02-25T00:00:03Z')`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed comm trust receipt fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
