package daemonruntime

import (
	"context"
	"database/sql"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestListCommThreadsSupportsFiltersAndCursor(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommInboxFixtures(t, container.DB)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	page1, err := service.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: "ws1",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm threads page1: %v", err)
	}
	if len(page1.Items) != 1 {
		t.Fatalf("expected one thread on first page, got %d", len(page1.Items))
	}
	if page1.Items[0].ThreadID != "thread-new" {
		t.Fatalf("expected newest thread first, got %+v", page1.Items[0])
	}
	if page1.Items[0].ConnectorID != "twilio" {
		t.Fatalf("expected connector_id twilio on newest thread, got %+v", page1.Items[0])
	}
	if !page1.HasMore || page1.NextCursor == "" {
		t.Fatalf("expected next cursor for first page, got %+v", page1)
	}
	if len(page1.Items[0].ParticipantAddresses) == 0 {
		t.Fatalf("expected participant addresses on thread row, got %+v", page1.Items[0])
	}

	page2, err := service.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: "ws1",
		Cursor:      page1.NextCursor,
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm threads page2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Fatalf("expected one thread on second page, got %d", len(page2.Items))
	}
	if page2.Items[0].ThreadID == page1.Items[0].ThreadID {
		t.Fatalf("expected cursor to advance to next thread, got duplicate %+v", page2.Items[0])
	}

	filtered, err := service.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: "ws1",
		Channel:     "mail",
		ConnectorID: "mail",
		Query:       "mail watcher",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list comm threads filtered: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ThreadID != "thread-mail" {
		t.Fatalf("expected filtered mail thread only, got %+v", filtered.Items)
	}
}

func TestListCommEventsSupportsFiltersAndCursor(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommInboxFixtures(t, container.DB)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	page1, err := service.ListCommEvents(context.Background(), transport.CommEventTimelineRequest{
		WorkspaceID: "ws1",
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("list comm events page1: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("expected two events on first page, got %d", len(page1.Items))
	}
	if page1.Items[0].EventID != "event-new-2" {
		t.Fatalf("expected newest event first, got %+v", page1.Items[0])
	}
	if page1.Items[0].ConnectorID != "twilio" {
		t.Fatalf("expected connector_id twilio on newest event, got %+v", page1.Items[0])
	}
	if !page1.HasMore || page1.NextCursor == "" {
		t.Fatalf("expected event next cursor, got %+v", page1)
	}
	if len(page1.Items[0].Addresses) == 0 {
		t.Fatalf("expected event addresses to be populated, got %+v", page1.Items[0])
	}

	page2, err := service.ListCommEvents(context.Background(), transport.CommEventTimelineRequest{
		WorkspaceID: "ws1",
		Cursor:      page1.NextCursor,
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("list comm events page2: %v", err)
	}
	if len(page2.Items) == 0 {
		t.Fatalf("expected second page events, got none")
	}
	if page2.Items[0].EventID == page1.Items[0].EventID {
		t.Fatalf("expected cursor to advance event timeline, got duplicate %+v", page2.Items[0])
	}

	filtered, err := service.ListCommEvents(context.Background(), transport.CommEventTimelineRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-new",
		ConnectorID: "twilio",
		Direction:   "INBOUND",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list comm events filtered: %v", err)
	}
	if len(filtered.Items) == 0 {
		t.Fatalf("expected filtered events for thread-new")
	}
	for _, item := range filtered.Items {
		if item.ThreadID != "thread-new" || item.Direction != "INBOUND" || item.ConnectorID != "twilio" {
			t.Fatalf("unexpected filtered event row: %+v", item)
		}
	}
}

func TestListCommCallSessionsSupportsFiltersAndCursor(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommInboxFixtures(t, container.DB)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	page1, err := service.ListCommCallSessions(context.Background(), transport.CommCallSessionListRequest{
		WorkspaceID: "ws1",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm call sessions page1: %v", err)
	}
	if len(page1.Items) != 1 {
		t.Fatalf("expected one call-session row on first page, got %d", len(page1.Items))
	}
	if page1.Items[0].SessionID != "call-new" {
		t.Fatalf("expected newest call-session first, got %+v", page1.Items[0])
	}
	if page1.Items[0].ConnectorID != "twilio" {
		t.Fatalf("expected connector_id twilio on newest call session, got %+v", page1.Items[0])
	}
	if !page1.HasMore || page1.NextCursor == "" {
		t.Fatalf("expected call-session next cursor, got %+v", page1)
	}

	page2, err := service.ListCommCallSessions(context.Background(), transport.CommCallSessionListRequest{
		WorkspaceID: "ws1",
		Cursor:      page1.NextCursor,
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm call sessions page2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Fatalf("expected one call-session row on second page, got %d", len(page2.Items))
	}
	if page2.Items[0].SessionID == page1.Items[0].SessionID {
		t.Fatalf("expected cursor to advance call-session list, got duplicate %+v", page2.Items[0])
	}

	filtered, err := service.ListCommCallSessions(context.Background(), transport.CommCallSessionListRequest{
		WorkspaceID: "ws1",
		ConnectorID: "twilio",
		Status:      "completed",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list comm call sessions filtered: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].SessionID != "call-old" {
		t.Fatalf("expected only completed call-session row, got %+v", filtered.Items)
	}
}

func seedCommInboxFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		 VALUES ('thread-new', 'ws1', 'message', 'twilio', 'thread:new', 'Newest Thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		 VALUES ('thread-mail', 'ws1', 'mail', 'mail', 'thread:mail', 'Mail watcher thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		 VALUES ('thread-old', 'ws1', 'voice', 'twilio', 'thread:old', 'Older voice thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-old-1', 'ws1', 'thread-old', 'twilio', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:01Z', 'voice transcript old', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-mail-1', 'ws1', 'thread-mail', 'mail', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:02Z', 'mail watcher token', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-new-1', 'ws1', 'thread-new', 'twilio', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:02Z', 'hello from thread new', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-new-2', 'ws1', 'thread-new', 'twilio', 'MESSAGE', 'OUTBOUND', 1, '2026-02-25T00:00:03Z', 'assistant response', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-old-1', 'event-old-1', 'FROM', '+15555550009', 'Caller', 0, '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-mail-1', 'event-mail-1', 'FROM', 'sender@example.com', 'Sender', 0, '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-new-1', 'event-new-1', 'FROM', '+15555550010', 'Person A', 0, '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-new-2', 'event-new-1', 'TO', '+15555550002', 'Agent', 1, '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-new-3', 'event-new-2', 'FROM', '+15555550002', 'Agent', 0, '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-new-4', 'event-new-2', 'TO', '+15555550010', 'Person A', 1, '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_call_sessions(id, workspace_id, provider, connector_id, provider_call_id, thread_id, direction, from_address, to_address, status, started_at, ended_at, created_at, updated_at)
		 VALUES ('call-old', 'ws1', 'twilio', 'twilio', 'CAOLD', 'thread-old', 'inbound', '+15555550009', '+15555550002', 'completed', '2026-02-25T00:00:00Z', '2026-02-25T00:00:01Z', '2026-02-25T00:00:00Z', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_call_sessions(id, workspace_id, provider, connector_id, provider_call_id, thread_id, direction, from_address, to_address, status, started_at, ended_at, created_at, updated_at)
		 VALUES ('call-new', 'ws1', 'twilio', 'twilio', 'CANEW', 'thread-new', 'outbound', '+15555550002', '+15555550010', 'in_progress', '2026-02-25T00:00:02Z', NULL, '2026-02-25T00:00:02Z', '2026-02-25T00:00:03Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed comm inbox fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
