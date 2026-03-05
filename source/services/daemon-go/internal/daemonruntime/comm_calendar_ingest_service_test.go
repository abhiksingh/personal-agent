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

type recordingCalendarEvaluator struct {
	eventIDs []string
}

func (r *recordingCalendarEvaluator) EvaluateCommEvent(_ context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	r.eventIDs = append(r.eventIDs, eventID)
	return types.CommTriggerEvaluationResult{}, nil
}

func TestCommTwilioServiceIngestCalendarChangePersistsAndEvaluates(t *testing.T) {
	db := newCalendarIngestTestDB(t)
	evaluator := &recordingCalendarEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	response, err := service.IngestCalendarChange(context.Background(), transport.CalendarChangeIngestRequest{
		WorkspaceID:   "ws1",
		SourceScope:   "calendar://work",
		SourceEventID: "calendar-change-1",
		SourceCursor:  "12001",
		CalendarID:    "calendar-work-id",
		CalendarName:  "Work",
		EventUID:      "event-uid-1",
		ChangeType:    "updated",
		Title:         "Team sync",
		Notes:         "Review project updates",
		Location:      "Room 1",
		StartsAt:      "2026-02-24T13:00:00Z",
		EndsAt:        "2026-02-24T14:00:00Z",
		OccurredAt:    "2026-02-24T12:30:00Z",
	})
	if err != nil {
		t.Fatalf("ingest calendar change: %v", err)
	}
	if !response.Accepted || response.Replayed {
		t.Fatalf("unexpected ingest response: %+v", response)
	}
	if response.Source != calendarChangeIngestSource || response.SourceScope != "calendar://work" {
		t.Fatalf("unexpected source metadata: %+v", response)
	}
	if response.EventID == "" || response.ThreadID == "" {
		t.Fatalf("expected persisted event/thread ids, got %+v", response)
	}
	if response.ChangeType != "updated" {
		t.Fatalf("expected normalized change_type updated, got %s", response.ChangeType)
	}

	assertCalendarCount(t, db, "comm_events", 1)
	assertCalendarCount(t, db, "comm_threads", 1)
	assertCalendarCount(t, db, "comm_ingest_receipts", 1)
	assertCalendarCount(t, db, "comm_ingest_cursors", 1)
	assertCalendarCount(t, db, "automation_source_subscriptions", 1)
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
	if channel != "calendar" {
		t.Fatalf("expected calendar thread channel, got %s", channel)
	}
	if threadConnector != "calendar" {
		t.Fatalf("expected calendar thread connector_id=calendar, got %s", threadConnector)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, response.EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query event connector: %v", err)
	}
	if eventConnector != "calendar" {
		t.Fatalf("expected calendar event connector_id=calendar, got %s", eventConnector)
	}

	var body string
	if err := db.QueryRow(`SELECT COALESCE(body_text, '') FROM comm_events WHERE id = ?`, response.EventID).Scan(&body); err != nil {
		t.Fatalf("query event body: %v", err)
	}
	if !strings.Contains(body, "Team sync") || !strings.Contains(body, "calendar_change=updated") {
		t.Fatalf("expected calendar event body summary, got %q", body)
	}
}

func TestCommTwilioServiceIngestCalendarChangeReplaySafe(t *testing.T) {
	db := newCalendarIngestTestDB(t)
	evaluator := &recordingCalendarEvaluator{}
	service := &CommTwilioService{
		container:      &ServiceContainer{DB: db},
		automationEval: evaluator,
	}

	request := transport.CalendarChangeIngestRequest{
		WorkspaceID:   "ws1",
		SourceScope:   "calendar://work",
		SourceEventID: "calendar-change-1",
		SourceCursor:  "12001",
		EventUID:      "event-uid-1",
		Title:         "Team sync",
		OccurredAt:    "2026-02-24T12:30:00Z",
	}

	first, err := service.IngestCalendarChange(context.Background(), request)
	if err != nil {
		t.Fatalf("first calendar ingest: %v", err)
	}
	second, err := service.IngestCalendarChange(context.Background(), request)
	if err != nil {
		t.Fatalf("second calendar ingest: %v", err)
	}

	if !first.Accepted || first.Replayed {
		t.Fatalf("unexpected first ingest response: %+v", first)
	}
	if !second.Accepted || !second.Replayed {
		t.Fatalf("unexpected replay ingest response: %+v", second)
	}

	assertCalendarCount(t, db, "comm_events", 1)
	assertCalendarCount(t, db, "comm_ingest_receipts", 1)
	if len(evaluator.eventIDs) != 1 {
		t.Fatalf("expected one automation evaluation call, got %v", evaluator.eventIDs)
	}
}

func newCalendarIngestTestDB(t *testing.T) *sql.DB {
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

func assertCalendarCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, count)
	}
}
