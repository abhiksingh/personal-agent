package daemonruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type stubMessagesPollDispatcher struct {
	pollResponse messagesadapter.InboundPollResponse
	pollErr      error
	pollCalls    int
}

func (s *stubMessagesPollDispatcher) CheckTwilio(_ context.Context, _ channelcheck.TwilioRequest) (channelcheck.TwilioResult, error) {
	return channelcheck.TwilioResult{}, nil
}

func (s *stubMessagesPollDispatcher) SendTwilioSMS(_ context.Context, _ twilioadapter.SMSAPIRequest) (twilioadapter.SMSAPIResponse, error) {
	return twilioadapter.SMSAPIResponse{}, nil
}

func (s *stubMessagesPollDispatcher) StartTwilioVoiceCall(_ context.Context, _ twilioadapter.VoiceCallRequest) (twilioadapter.VoiceCallResponse, error) {
	return twilioadapter.VoiceCallResponse{}, nil
}

func (s *stubMessagesPollDispatcher) SendMessages(_ context.Context, _ messagesadapter.SendRequest) (messagesadapter.SendResponse, error) {
	return messagesadapter.SendResponse{}, nil
}

func (s *stubMessagesPollDispatcher) PollMessagesInbound(_ context.Context, _ messagesadapter.InboundPollRequest) (messagesadapter.InboundPollResponse, error) {
	s.pollCalls++
	return s.pollResponse, s.pollErr
}

type recordingMessagesEvaluator struct {
	eventIDs []string
}

func (r *recordingMessagesEvaluator) EvaluateCommEvent(_ context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	r.eventIDs = append(r.eventIDs, eventID)
	return types.CommTriggerEvaluationResult{}, nil
}

func TestCommTwilioServiceIngestMessagesPersistsEventsAndCursor(t *testing.T) {
	db := newMessagesIngestTestDB(t)
	dispatch := &stubMessagesPollDispatcher{
		pollResponse: messagesadapter.InboundPollResponse{
			WorkspaceID:  "ws1",
			Source:       messagesadapter.SourceName,
			SourceScope:  "scope://messages",
			SourceDBPath: "/tmp/chat.db",
			CursorStart:  "0",
			CursorEnd:    "101",
			Polled:       1,
			Events: []messagesadapter.InboundMessageEvent{
				{
					SourceEventID:    "message-guid-1",
					SourceCursor:     "101",
					ExternalThreadID: "chat-guid-1",
					SenderAddress:    "+15555550100",
					BodyText:         "hello inbound",
					OccurredAt:       "2026-02-24T12:00:00Z",
				},
			},
		},
	}
	evaluator := &recordingMessagesEvaluator{}
	service := &CommTwilioService{
		container:       &ServiceContainer{DB: db},
		channelDispatch: dispatch,
		automationEval:  evaluator,
	}

	response, err := service.IngestMessages(context.Background(), transport.MessagesIngestRequest{
		WorkspaceID:  "ws1",
		SourceScope:  "scope://messages",
		SourceDBPath: "/tmp/chat.db",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("ingest messages: %v", err)
	}
	if response.Accepted != 1 || response.Replayed != 0 {
		t.Fatalf("unexpected ingest counters: %+v", response)
	}
	if response.CursorEnd != "101" {
		t.Fatalf("expected cursor end 101, got %s", response.CursorEnd)
	}
	if dispatch.pollCalls != 1 {
		t.Fatalf("expected one poll call, got %d", dispatch.pollCalls)
	}
	if len(response.Events) != 1 || response.Events[0].EventID == "" {
		t.Fatalf("expected persisted event record, got %+v", response.Events)
	}

	assertMessagesCount(t, db, "comm_events", 1)
	assertMessagesCount(t, db, "comm_threads", 1)
	assertMessagesCount(t, db, "comm_ingest_receipts", 1)
	assertMessagesCount(t, db, "comm_ingest_cursors", 1)
	assertMessagesCount(t, db, "automation_source_subscriptions", 1)
	var (
		threadChannel   string
		threadConnector string
		eventConnector  string
	)
	if err := db.QueryRow(`SELECT channel, connector_id FROM comm_threads WHERE id = ?`, response.Events[0].ThreadID).Scan(&threadChannel, &threadConnector); err != nil {
		t.Fatalf("query messages thread channel/connector: %v", err)
	}
	if threadChannel != "message" {
		t.Fatalf("expected messages ingest thread channel=message, got %s", threadChannel)
	}
	if threadConnector != "imessage" {
		t.Fatalf("expected messages ingest thread connector_id=imessage, got %s", threadConnector)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, response.Events[0].EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query messages event connector: %v", err)
	}
	if eventConnector != "imessage" {
		t.Fatalf("expected messages ingest event connector_id=imessage, got %s", eventConnector)
	}
	if len(evaluator.eventIDs) != 1 || evaluator.eventIDs[0] != response.Events[0].EventID {
		t.Fatalf("expected one evaluated event id, got %v", evaluator.eventIDs)
	}
}

func TestCommTwilioServiceIngestMessagesIsReplaySafe(t *testing.T) {
	db := newMessagesIngestTestDB(t)
	dispatch := &stubMessagesPollDispatcher{
		pollResponse: messagesadapter.InboundPollResponse{
			WorkspaceID:  "ws1",
			Source:       messagesadapter.SourceName,
			SourceScope:  "scope://messages",
			SourceDBPath: "/tmp/chat.db",
			CursorStart:  "0",
			CursorEnd:    "101",
			Polled:       1,
			Events: []messagesadapter.InboundMessageEvent{
				{
					SourceEventID:    "message-guid-1",
					SourceCursor:     "101",
					ExternalThreadID: "chat-guid-1",
					SenderAddress:    "+15555550100",
					BodyText:         "hello inbound",
					OccurredAt:       "2026-02-24T12:00:00Z",
				},
			},
		},
	}
	evaluator := &recordingMessagesEvaluator{}
	service := &CommTwilioService{
		container:       &ServiceContainer{DB: db},
		channelDispatch: dispatch,
		automationEval:  evaluator,
	}

	first, err := service.IngestMessages(context.Background(), transport.MessagesIngestRequest{
		WorkspaceID:  "ws1",
		SourceScope:  "scope://messages",
		SourceDBPath: "/tmp/chat.db",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("first ingest messages: %v", err)
	}
	second, err := service.IngestMessages(context.Background(), transport.MessagesIngestRequest{
		WorkspaceID:  "ws1",
		SourceScope:  "scope://messages",
		SourceDBPath: "/tmp/chat.db",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("second ingest messages: %v", err)
	}

	if first.Accepted != 1 || first.Replayed != 0 {
		t.Fatalf("unexpected first ingest counters: %+v", first)
	}
	if second.Accepted != 0 || second.Replayed != 1 {
		t.Fatalf("unexpected second ingest counters: %+v", second)
	}
	assertMessagesCount(t, db, "comm_events", 1)
	assertMessagesCount(t, db, "comm_ingest_receipts", 1)
	if len(evaluator.eventIDs) != 1 {
		t.Fatalf("expected automation eval only for first accepted event, got %v", evaluator.eventIDs)
	}
}

func newMessagesIngestTestDB(t *testing.T) *sql.DB {
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

func assertMessagesCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, count)
	}
}
