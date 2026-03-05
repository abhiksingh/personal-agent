package daemonruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type recordingBrowserEvaluator struct {
	eventIDs []string
}

func (r *recordingBrowserEvaluator) EvaluateCommEvent(_ context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	r.eventIDs = append(r.eventIDs, eventID)
	return types.CommTriggerEvaluationResult{}, nil
}

func TestCommTwilioServiceIngestBrowserEventPersistsAndEvaluates(t *testing.T) {
	db := newBrowserIngestTestDB(t)
	evaluator := &recordingBrowserEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	response, err := service.IngestBrowserEvent(context.Background(), transport.BrowserEventIngestRequest{
		WorkspaceID:   "ws1",
		SourceScope:   "safari://window/1",
		SourceEventID: "browser-event-1",
		SourceCursor:  "13001",
		WindowID:      "window-1",
		TabID:         "tab-1",
		PageURL:       "https://example.com",
		PageTitle:     "Example Domain",
		EventType:     "navigation",
		PayloadText:   "page loaded",
		OccurredAt:    "2026-02-24T13:00:00Z",
	})
	if err != nil {
		t.Fatalf("ingest browser event: %v", err)
	}
	if !response.Accepted || response.Replayed {
		t.Fatalf("unexpected ingest response: %+v", response)
	}
	if response.Source != browserEventIngestSource || response.SourceScope != "safari://window/1" {
		t.Fatalf("unexpected source metadata: %+v", response)
	}
	if response.EventID == "" || response.ThreadID == "" {
		t.Fatalf("expected persisted event/thread ids, got %+v", response)
	}
	if response.EventType != "navigation" {
		t.Fatalf("expected event_type navigation, got %s", response.EventType)
	}

	assertBrowserCount(t, db, "comm_events", 1)
	assertBrowserCount(t, db, "comm_threads", 1)
	assertBrowserCount(t, db, "comm_ingest_receipts", 1)
	assertBrowserCount(t, db, "comm_ingest_cursors", 1)
	assertBrowserCount(t, db, "automation_source_subscriptions", 1)
	if len(evaluator.eventIDs) != 1 || evaluator.eventIDs[0] != response.EventID {
		t.Fatalf("expected one automation evaluation call, got %v", evaluator.eventIDs)
	}

	var (
		channel         string
		threadConnector string
		eventConnector  string
	)
	if err := db.QueryRow(`SELECT channel, connector_id FROM comm_threads WHERE id = ?`, response.ThreadID).Scan(&channel, &threadConnector); err != nil {
		t.Fatalf("query thread channel/connector: %v", err)
	}
	if channel != "browser" {
		t.Fatalf("expected browser thread channel, got %s", channel)
	}
	if threadConnector != "browser" {
		t.Fatalf("expected browser thread connector_id=browser, got %s", threadConnector)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, response.EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query event connector: %v", err)
	}
	if eventConnector != "browser" {
		t.Fatalf("expected browser event connector_id=browser, got %s", eventConnector)
	}

	var body string
	if err := db.QueryRow(`SELECT COALESCE(body_text, '') FROM comm_events WHERE id = ?`, response.EventID).Scan(&body); err != nil {
		t.Fatalf("query event body: %v", err)
	}
	if !strings.Contains(body, "browser_event_type=navigation") || !strings.Contains(body, "https://example.com") {
		t.Fatalf("expected browser event body summary, got %q", body)
	}
}

func TestCommTwilioServiceIngestBrowserEventReplaySafe(t *testing.T) {
	db := newBrowserIngestTestDB(t)
	evaluator := &recordingBrowserEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	request := transport.BrowserEventIngestRequest{
		WorkspaceID:   "ws1",
		SourceScope:   "safari://window/1",
		SourceEventID: "browser-event-1",
		SourceCursor:  "13001",
		TabID:         "tab-1",
		PageURL:       "https://example.com",
		EventType:     "navigation",
		OccurredAt:    "2026-02-24T13:00:00Z",
	}

	first, err := service.IngestBrowserEvent(context.Background(), request)
	if err != nil {
		t.Fatalf("first browser ingest: %v", err)
	}
	second, err := service.IngestBrowserEvent(context.Background(), request)
	if err != nil {
		t.Fatalf("second browser ingest: %v", err)
	}

	if !first.Accepted || first.Replayed {
		t.Fatalf("unexpected first ingest response: %+v", first)
	}
	if !second.Accepted || !second.Replayed {
		t.Fatalf("unexpected replay ingest response: %+v", second)
	}

	assertBrowserCount(t, db, "comm_events", 1)
	assertBrowserCount(t, db, "comm_ingest_receipts", 1)
	if len(evaluator.eventIDs) != 1 {
		t.Fatalf("expected one automation evaluation call, got %v", evaluator.eventIDs)
	}
}

func newBrowserIngestTestDB(t *testing.T) *sql.DB {
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

func assertBrowserCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, count)
	}
}
