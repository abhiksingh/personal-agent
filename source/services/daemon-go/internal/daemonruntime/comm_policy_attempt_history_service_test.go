package daemonruntime

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

func TestSetCommPolicyUpdatesExistingPolicyByID(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	created, err := service.SetCommPolicy(context.Background(), transport.CommPolicySetRequest{
		WorkspaceID:      "ws1",
		SourceChannel:    "message",
		EndpointPattern:  "+1555%",
		PrimaryChannel:   "imessage",
		RetryCount:       1,
		FallbackChannels: []string{"sms"},
		IsDefault:        true,
	})
	if err != nil {
		t.Fatalf("create comm policy: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created policy id")
	}

	updated, err := service.SetCommPolicy(context.Background(), transport.CommPolicySetRequest{
		PolicyID:         created.ID,
		WorkspaceID:      "ws1",
		SourceChannel:    "message",
		EndpointPattern:  "+1555%",
		PrimaryChannel:   "sms",
		RetryCount:       0,
		FallbackChannels: []string{"twilio"},
		IsDefault:        true,
	})
	if err != nil {
		t.Fatalf("update comm policy: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected updated policy id to remain %q, got %q", created.ID, updated.ID)
	}
	if updated.Policy.PrimaryChannel != "twilio" || updated.Policy.RetryCount != 0 {
		t.Fatalf("unexpected updated policy payload: %+v", updated.Policy)
	}
	if len(updated.Policy.FallbackChannels) != 0 {
		t.Fatalf("unexpected updated fallback channels: %+v", updated.Policy.FallbackChannels)
	}

	listed, err := service.ListCommPolicies(context.Background(), transport.CommPolicyListRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list comm policies: %v", err)
	}
	if len(listed.Policies) != 1 {
		t.Fatalf("expected one policy row after update, got %d", len(listed.Policies))
	}
	if listed.Policies[0].ID != created.ID {
		t.Fatalf("expected updated policy row id %q, got %q", created.ID, listed.Policies[0].ID)
	}
	if listed.Policies[0].Policy.PrimaryChannel != "twilio" {
		t.Fatalf("expected listed policy primary_channel=twilio, got %+v", listed.Policies[0].Policy)
	}
}

func TestSendCommDerivesReplyDestinationFromThreadContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommSendThreadFixture(t, container.DB, commSendThreadFixture{
		WorkspaceID:  "ws1",
		ThreadID:     "thread-reply-imessage",
		Channel:      "message",
		ConnectorID:  "imessage",
		ExternalRef:  "twilio:sms:+15559990000:+15550001111",
		EventID:      "event-reply-imessage",
		EventDir:     "INBOUND",
		FromAddress:  "+15550001111",
		ToAddress:    "+15559990000",
		OccurredAt:   "2026-02-26T00:00:00Z",
		EventCreated: "2026-02-26T00:00:00Z",
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}
	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			MessageID:   "msg-1",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	service.channelDispatch = dispatch

	response, err := service.SendComm(context.Background(), transport.CommSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-reply-imessage",
		Message:     "reply from thread context",
	})
	if err != nil {
		t.Fatalf("send comm with thread context: %v", err)
	}
	if !response.Success || !response.Result.Delivered {
		t.Fatalf("expected successful thread-context send response, got %+v", response)
	}
	if response.ResolvedSourceChannel != "imessage" {
		t.Fatalf("expected resolved source channel imessage, got %q", response.ResolvedSourceChannel)
	}
	if response.ResolvedConnectorID != "imessage" {
		t.Fatalf("expected resolved connector_id imessage, got %q", response.ResolvedConnectorID)
	}
	if response.ResolvedDestination != "+15550001111" {
		t.Fatalf("expected resolved destination +15550001111, got %q", response.ResolvedDestination)
	}
	if dispatch.messagesCalls != 1 {
		t.Fatalf("expected one messages dispatch call, got %d", dispatch.messagesCalls)
	}
	if dispatch.lastMessagesReq.Destination != "+15550001111" {
		t.Fatalf("expected derived destination to route through messages dispatcher, got %q", dispatch.lastMessagesReq.Destination)
	}
}

func TestSendCommConnectorHintTargetsTwilioRoute(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			MessageID:   "should-not-be-used",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	service.channelDispatch = dispatch

	response, err := service.SendComm(context.Background(), transport.CommSendRequest{
		WorkspaceID:   "ws1",
		OperationID:   "op-twilio-hint",
		SourceChannel: "message",
		ConnectorID:   "twilio",
		Destination:   "+15550002222",
		Message:       "route to twilio hint",
	})
	if err != nil {
		t.Fatalf("send comm with twilio connector hint: %v", err)
	}
	if !response.Success || !response.Result.Delivered {
		t.Fatalf("expected successful twilio-hint send response, got %+v", response)
	}
	if response.ResolvedSourceChannel != "twilio" {
		t.Fatalf("expected resolved source channel twilio, got %q", response.ResolvedSourceChannel)
	}
	if response.ResolvedConnectorID != "twilio" {
		t.Fatalf("expected resolved connector_id twilio, got %q", response.ResolvedConnectorID)
	}
	if response.Result.Channel != "twilio" {
		t.Fatalf("expected twilio-hint send to dispatch via twilio route, got %q", response.Result.Channel)
	}
	if dispatch.messagesCalls != 0 {
		t.Fatalf("expected no messages dispatch calls for twilio-hinted route, got %d", dispatch.messagesCalls)
	}
}

func TestSendCommAcceptsSMSAliasSourceChannel(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			MessageID:   "should-not-be-used",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	service.channelDispatch = dispatch

	response, err := service.SendComm(context.Background(), transport.CommSendRequest{
		WorkspaceID:   "ws1",
		OperationID:   "op-sms-alias",
		SourceChannel: "sms",
		Destination:   "+15550003333",
		Message:       "route using sms alias",
	})
	if err != nil {
		t.Fatalf("send comm with sms alias source: %v", err)
	}
	if !response.Success || !response.Result.Delivered {
		t.Fatalf("expected successful sms-alias send response, got %+v", response)
	}
	if response.ResolvedSourceChannel != "sms" {
		t.Fatalf("expected resolved source channel sms, got %q", response.ResolvedSourceChannel)
	}
	if response.ResolvedConnectorID != "twilio" {
		t.Fatalf("expected resolved connector_id twilio, got %q", response.ResolvedConnectorID)
	}
	if response.Result.Channel != "twilio" {
		t.Fatalf("expected sms-alias send to dispatch via twilio route, got %q", response.Result.Channel)
	}
	if dispatch.messagesCalls != 0 {
		t.Fatalf("expected no messages dispatch calls for sms alias route, got %d", dispatch.messagesCalls)
	}
}

func TestSendCommRejectsConnectorHintMismatchForThreadContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommSendThreadFixture(t, container.DB, commSendThreadFixture{
		WorkspaceID: "ws1",
		ThreadID:    "thread-mismatch",
		Channel:     "message",
		ConnectorID: "imessage",
		ExternalRef: "twilio:sms:+15559990000:+15550001111",
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	_, err = service.SendComm(context.Background(), transport.CommSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-mismatch",
		ConnectorID: "twilio",
		Message:     "should fail",
	})
	if err == nil {
		t.Fatalf("expected connector/thread mismatch validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "does not match thread") {
		t.Fatalf("expected mismatch error copy, got %v", err)
	}
}

func TestSendCommRequiresDestinationWhenThreadContextCannotResolveReplyTarget(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommSendThreadFixture(t, container.DB, commSendThreadFixture{
		WorkspaceID: "ws1",
		ThreadID:    "thread-no-destination",
		Channel:     "message",
		ConnectorID: "imessage",
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	_, err = service.SendComm(context.Background(), transport.CommSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-no-destination",
		Message:     "missing destination",
	})
	if err == nil {
		t.Fatalf("expected missing destination derivation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cannot derive reply destination") {
		t.Fatalf("expected destination derivation failure copy, got %v", err)
	}
}

func TestListCommAttemptsSupportsContextFiltersCursorAndFallbackMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommAttemptHistoryFixtures(t, container.DB)

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	page1, err := service.ListCommAttempts(context.Background(), transport.CommAttemptsRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list comm attempts page1: %v", err)
	}
	if len(page1.Attempts) != 1 {
		t.Fatalf("expected one attempt row on page1, got %d", len(page1.Attempts))
	}
	if !page1.HasMore || page1.NextCursor == "" {
		t.Fatalf("expected page1 cursor metadata, got %+v", page1)
	}
	if page1.Attempts[0].AttemptID != "attempt-3" {
		t.Fatalf("expected newest attempt first, got %+v", page1.Attempts[0])
	}
	if page1.Attempts[0].TaskID != "task-1" || page1.Attempts[0].RunID != "run-1" || page1.Attempts[0].StepID != "step-1" || page1.Attempts[0].ThreadID != "thread-1" {
		t.Fatalf("expected context ids on attempts row, got %+v", page1.Attempts[0])
	}
	if page1.Attempts[0].RoutePhase != "fallback" || page1.Attempts[0].FallbackFromChannel != "imessage" {
		t.Fatalf("expected fallback metadata on newest attempt, got %+v", page1.Attempts[0])
	}

	page2, err := service.ListCommAttempts(context.Background(), transport.CommAttemptsRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
		Cursor:      page1.NextCursor,
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("list comm attempts page2: %v", err)
	}
	if len(page2.Attempts) != 2 {
		t.Fatalf("expected two rows on page2, got %d", len(page2.Attempts))
	}
	if page2.Attempts[0].AttemptID != "attempt-2" || page2.Attempts[0].RoutePhase != "retry" || page2.Attempts[0].RetryOrdinal != 1 {
		t.Fatalf("expected retry metadata on attempt-2, got %+v", page2.Attempts[0])
	}

	filteredByTask, err := service.ListCommAttempts(context.Background(), transport.CommAttemptsRequest{
		WorkspaceID: "ws1",
		TaskID:      "task-1",
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("list comm attempts by task: %v", err)
	}
	if len(filteredByTask.Attempts) != 3 {
		t.Fatalf("expected three attempts for task-1, got %d", len(filteredByTask.Attempts))
	}
	for _, attempt := range filteredByTask.Attempts {
		if attempt.TaskID != "task-1" || attempt.OperationID != "op-1" {
			t.Fatalf("unexpected task-filtered attempt row: %+v", attempt)
		}
	}
}

func TestListCommAttemptsLegacyOperationPathPreservesRouteOrder(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedCommAttemptHistoryFixtures(t, container.DB)

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm service: %v", err)
	}

	response, err := service.ListCommAttempts(context.Background(), transport.CommAttemptsRequest{
		WorkspaceID: "ws1",
		OperationID: "op-1",
	})
	if err != nil {
		t.Fatalf("list legacy comm attempts path: %v", err)
	}
	if len(response.Attempts) != 3 {
		t.Fatalf("expected three attempts on legacy path, got %d", len(response.Attempts))
	}
	if response.Attempts[0].AttemptID != "attempt-1" || response.Attempts[1].AttemptID != "attempt-2" || response.Attempts[2].AttemptID != "attempt-3" {
		t.Fatalf("expected legacy attempt ordering by route, got %+v", response.Attempts)
	}
	if response.Attempts[0].RoutePhase != "primary" || response.Attempts[1].RoutePhase != "retry" || response.Attempts[2].RoutePhase != "fallback" {
		t.Fatalf("expected route-phase metadata on legacy attempts, got %+v", response.Attempts)
	}
}

func seedCommAttemptHistoryFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'PERSON', 'Requester', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('principal.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-1', 'ws1', 'actor.requester', 'actor.requester', 'Send update', '', 'completed', 0, NULL, 'messages', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-1', 'ws1', 'task-1', 'actor.requester', 'completed', '2026-02-25T00:00:00Z', '2026-02-25T00:00:04Z', '', '2026-02-25T00:00:00Z', '2026-02-25T00:00:04Z')`,
		`INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		 VALUES ('step-1', 'run-1', 0, 'Send message', 'completed', 'NONE', 'messages_send_sms', NULL, 0, 0, '', '2026-02-25T00:00:00Z', '2026-02-25T00:00:04Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread-1', 'ws1', 'message', 'thread:1', 'Thread 1', '2026-02-25T00:00:00Z', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread-2', 'ws1', 'message', 'thread:2', 'Thread 2', '2026-02-25T00:00:00Z', '2026-02-25T00:00:04Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-1', 'ws1', 'thread-1', 'MESSAGE', 'OUTBOUND', 1, '2026-02-25T00:00:01Z', 'outbound to thread 1', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-2', 'ws1', 'thread-2', 'MESSAGE', 'OUTBOUND', 1, '2026-02-25T00:00:04Z', 'outbound to thread 2', '2026-02-25T00:00:04Z')`,
		`INSERT INTO delivery_attempts(id, workspace_id, step_id, event_id, destination_endpoint, idempotency_key, channel, provider_receipt, status, error, attempted_at)
		 VALUES ('attempt-1', 'ws1', 'step-1', 'event-1', '+15550001111', 'op-1|+15550001111|imessage|0', 'imessage', '', 'failed', 'imessage unavailable', '2026-02-25T00:00:01Z')`,
		`INSERT INTO delivery_attempts(id, workspace_id, step_id, event_id, destination_endpoint, idempotency_key, channel, provider_receipt, status, error, attempted_at)
		 VALUES ('attempt-2', 'ws1', 'step-1', 'event-1', '+15550001111', 'op-1|+15550001111|imessage|1', 'imessage', '', 'failed', 'imessage unavailable', '2026-02-25T00:00:02Z')`,
		`INSERT INTO delivery_attempts(id, workspace_id, step_id, event_id, destination_endpoint, idempotency_key, channel, provider_receipt, status, error, attempted_at)
		 VALUES ('attempt-3', 'ws1', 'step-1', 'event-1', '+15550001111', 'op-1|+15550001111|sms|2', 'sms', 'SM123', 'sent', '', '2026-02-25T00:00:03Z')`,
		`INSERT INTO delivery_attempts(id, workspace_id, step_id, event_id, destination_endpoint, idempotency_key, channel, provider_receipt, status, error, attempted_at)
		 VALUES ('attempt-4', 'ws1', NULL, 'event-2', '+15550002222', 'op-2|+15550002222|sms|0', 'sms', 'SM999', 'sent', '', '2026-02-25T00:00:04Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed comm attempt history fixtures: %v\nstatement: %s", err, statement)
		}
	}
}

type commSendThreadFixture struct {
	WorkspaceID  string
	ThreadID     string
	Channel      string
	ConnectorID  string
	ExternalRef  string
	EventID      string
	EventDir     string
	FromAddress  string
	ToAddress    string
	OccurredAt   string
	EventCreated string
}

func seedCommSendThreadFixture(t *testing.T, db *sql.DB, fixture commSendThreadFixture) {
	t.Helper()

	workspaceID := strings.TrimSpace(fixture.WorkspaceID)
	if workspaceID == "" {
		workspaceID = "ws1"
	}
	threadID := strings.TrimSpace(fixture.ThreadID)
	if threadID == "" {
		t.Fatalf("thread id is required for comm send thread fixture")
	}
	channel := strings.TrimSpace(fixture.Channel)
	if channel == "" {
		channel = "message"
	}
	connectorID := strings.TrimSpace(fixture.ConnectorID)
	externalRef := strings.TrimSpace(fixture.ExternalRef)
	occurredAt := strings.TrimSpace(fixture.OccurredAt)
	if occurredAt == "" {
		occurredAt = "2026-02-26T00:00:00Z"
	}
	eventCreated := strings.TrimSpace(fixture.EventCreated)
	if eventCreated == "" {
		eventCreated = occurredAt
	}

	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, eventCreated, eventCreated); err != nil {
		t.Fatalf("seed comm send fixture workspace: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			channel = excluded.channel,
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			updated_at = excluded.updated_at
	`, threadID, workspaceID, channel, connectorID, nullableText(externalRef), "fixture thread", eventCreated, eventCreated); err != nil {
		t.Fatalf("seed comm send fixture thread: %v", err)
	}

	eventID := strings.TrimSpace(fixture.EventID)
	if eventID == "" {
		return
	}
	eventDir := strings.ToUpper(strings.TrimSpace(fixture.EventDir))
	if eventDir == "" {
		eventDir = "INBOUND"
	}
	if _, err := db.Exec(`
		INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		VALUES (?, ?, ?, ?, 'MESSAGE', ?, 0, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			direction = excluded.direction,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, eventID, workspaceID, threadID, connectorID, eventDir, occurredAt, "fixture event", eventCreated); err != nil {
		t.Fatalf("seed comm send fixture event: %v", err)
	}

	fromAddress := strings.TrimSpace(fixture.FromAddress)
	toAddress := strings.TrimSpace(fixture.ToAddress)
	if fromAddress != "" {
		if _, err := db.Exec(`
			INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
			VALUES (?, ?, 'FROM', ?, NULL, 0, ?)
			ON CONFLICT(id) DO UPDATE SET address_value = excluded.address_value
		`, "fixture-address-from-"+eventID, eventID, fromAddress, eventCreated); err != nil {
			t.Fatalf("seed comm send fixture from address: %v", err)
		}
	}
	if toAddress != "" {
		if _, err := db.Exec(`
			INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
			VALUES (?, ?, 'TO', ?, NULL, 0, ?)
			ON CONFLICT(id) DO UPDATE SET address_value = excluded.address_value
		`, "fixture-address-to-"+eventID, eventID, toAddress, eventCreated); err != nil {
			t.Fatalf("seed comm send fixture to address: %v", err)
		}
	}
}
