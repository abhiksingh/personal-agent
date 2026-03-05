package daemonruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type recordingMailEvaluator struct {
	eventIDs []string
}

func (r *recordingMailEvaluator) EvaluateCommEvent(_ context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	r.eventIDs = append(r.eventIDs, eventID)
	return types.CommTriggerEvaluationResult{}, nil
}

func TestCommTwilioServiceIngestMailRuleEventPersistsAndEvaluates(t *testing.T) {
	db := newMailIngestTestDB(t)
	evaluator := &recordingMailEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	response, err := service.IngestMailRuleEvent(context.Background(), transport.MailRuleIngestRequest{
		WorkspaceID:      "ws1",
		SourceScope:      "mailbox://inbox",
		SourceEventID:    "mail-event-1",
		MessageID:        "<mail-event-1@example.com>",
		ThreadRef:        "thread-external-1",
		FromAddress:      "sender@example.com",
		ToAddress:        "recipient@example.com",
		Subject:          "Inbound subject",
		BodyText:         "Inbound body",
		OccurredAt:       "2026-02-24T12:34:56Z",
		ReferencesHeader: "<thread-root@example.com>",
		InReplyTo:        "<thread-root@example.com>",
	})
	if err != nil {
		t.Fatalf("ingest mail rule event: %v", err)
	}
	if !response.Accepted || response.Replayed {
		t.Fatalf("unexpected ingest response: %+v", response)
	}
	if response.EventID == "" || response.ThreadID == "" {
		t.Fatalf("expected event/thread ids in response: %+v", response)
	}
	if response.Source != mailRuleIngestSource || response.SourceScope != "mailbox://inbox" {
		t.Fatalf("unexpected source metadata: %+v", response)
	}

	assertMailCount(t, db, "comm_events", 1)
	assertMailCount(t, db, "comm_threads", 1)
	assertMailCount(t, db, "comm_ingest_receipts", 1)
	assertMailCount(t, db, "comm_ingest_cursors", 1)
	assertMailCount(t, db, "automation_source_subscriptions", 1)
	assertMailCount(t, db, "email_event_meta", 1)
	if len(evaluator.eventIDs) != 1 || evaluator.eventIDs[0] != response.EventID {
		t.Fatalf("expected automation evaluation once for accepted event, got %v", evaluator.eventIDs)
	}

	var (
		channel         string
		threadConnector string
		eventConnector  string
	)
	if err := db.QueryRow(`SELECT channel, connector_id FROM comm_threads WHERE id = ?`, response.ThreadID).Scan(&channel, &threadConnector); err != nil {
		t.Fatalf("query thread channel/connector: %v", err)
	}
	if channel != "mail" {
		t.Fatalf("expected mail channel, got %s", channel)
	}
	if threadConnector != "mail" {
		t.Fatalf("expected mail thread connector_id=mail, got %s", threadConnector)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, response.EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query event connector: %v", err)
	}
	if eventConnector != "mail" {
		t.Fatalf("expected mail event connector_id=mail, got %s", eventConnector)
	}
}

func TestCommTwilioServiceIngestMailRuleEventReplaySafe(t *testing.T) {
	db := newMailIngestTestDB(t)
	evaluator := &recordingMailEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	request := transport.MailRuleIngestRequest{
		WorkspaceID:   "ws1",
		SourceScope:   "mailbox://inbox",
		SourceEventID: "mail-event-1",
		MessageID:     "<mail-event-1@example.com>",
		FromAddress:   "sender@example.com",
		ToAddress:     "recipient@example.com",
		Subject:       "Inbound subject",
		BodyText:      "Inbound body",
		OccurredAt:    "2026-02-24T12:34:56Z",
	}

	first, err := service.IngestMailRuleEvent(context.Background(), request)
	if err != nil {
		t.Fatalf("first mail ingest: %v", err)
	}
	second, err := service.IngestMailRuleEvent(context.Background(), request)
	if err != nil {
		t.Fatalf("second mail ingest: %v", err)
	}

	if !first.Accepted || first.Replayed {
		t.Fatalf("unexpected first ingest response: %+v", first)
	}
	if !second.Accepted || !second.Replayed {
		t.Fatalf("unexpected replay ingest response: %+v", second)
	}

	assertMailCount(t, db, "comm_events", 1)
	assertMailCount(t, db, "comm_ingest_receipts", 1)
	if len(evaluator.eventIDs) != 1 {
		t.Fatalf("expected one automation evaluation call, got %v", evaluator.eventIDs)
	}
}

func newMailIngestTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := openRuntimeDB(context.Background(), filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatalf("open runtime db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func assertMailCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, count)
	}
}
