package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"testing"

	coretypes "personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func TestAgentDelegationServiceExecuteQueuedTaskRunRejectsUnauthorizedCrossPrincipal(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRunForAuthTest(t, container, queuedRunAuthFixture{
		taskID:      "task-auth-deny",
		runID:       "run-auth-deny",
		requestedBy: "actor.requester",
		subject:     "actor.subject",
		actingAs:    "actor.subject",
		title:       "Queued auth check",
		description: "send an email update",
	})

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	_, execErr := service.ExecuteQueuedTaskRun(context.Background(), "run-auth-deny", "corr-auth-deny")
	if execErr == nil {
		t.Fatalf("expected queued run execution to be denied without delegation")
	}
	if !strings.Contains(execErr.Error(), "acting_as denied") {
		t.Fatalf("expected acting_as denied error, got %v", execErr)
	}
}

func TestAgentDelegationServiceExecuteQueuedTaskRunAllowsDelegatedCrossPrincipal(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRunForAuthTest(t, container, queuedRunAuthFixture{
		taskID:      "task-auth-allow",
		runID:       "run-auth-allow",
		requestedBy: "actor.requester",
		subject:     "actor.subject",
		actingAs:    "actor.subject",
		title:       "",
		description: "",
	})
	if _, err := container.DB.Exec(`
		INSERT INTO delegation_rules(
			id, workspace_id, from_actor_id, to_actor_id, scope_type, scope_key, status, created_at, expires_at
		) VALUES (?, ?, ?, ?, 'EXECUTION', NULL, 'ACTIVE', ?, NULL)
	`, "dr-execution-allow", "ws1", "actor.requester", "actor.subject", "2026-02-24T00:00:02Z"); err != nil {
		t.Fatalf("insert execution delegation rule: %v", err)
	}

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	result, execErr := service.ExecuteQueuedTaskRun(context.Background(), "run-auth-allow", "corr-auth-allow")
	if execErr != nil {
		t.Fatalf("execute delegated queued task run: %v", execErr)
	}
	if result.RunState != "failed" || result.TaskState != "failed" {
		t.Fatalf("expected delegated run with empty request text to fail deterministically, got task=%s run=%s", result.TaskState, result.RunState)
	}
}

type stubAgentMessagesCommService struct {
	requests []transport.CommSendRequest
}

func (s *stubAgentMessagesCommService) SendComm(_ context.Context, request transport.CommSendRequest) (transport.CommSendResponse, error) {
	s.requests = append(s.requests, request)
	return transport.CommSendResponse{
		WorkspaceID: request.WorkspaceID,
		OperationID: request.OperationID,
		Success:     true,
		Result: coretypes.DeliveryResult{
			Delivered:       true,
			Channel:         strings.TrimSpace(request.SourceChannel),
			ProviderReceipt: "receipt-123",
		},
	}, nil
}

func (s *stubAgentMessagesCommService) ListCommAttempts(context.Context, transport.CommAttemptsRequest) (transport.CommAttemptsResponse, error) {
	return transport.CommAttemptsResponse{}, nil
}

func (s *stubAgentMessagesCommService) SetCommPolicy(context.Context, transport.CommPolicySetRequest) (transport.CommPolicyRecord, error) {
	return transport.CommPolicyRecord{}, nil
}

func (s *stubAgentMessagesCommService) ListCommPolicies(context.Context, transport.CommPolicyListRequest) (transport.CommPolicyListResponse, error) {
	return transport.CommPolicyListResponse{}, nil
}

func (s *stubAgentMessagesCommService) ListCommWebhookReceipts(context.Context, transport.CommWebhookReceiptListRequest) (transport.CommWebhookReceiptListResponse, error) {
	return transport.CommWebhookReceiptListResponse{}, nil
}

func (s *stubAgentMessagesCommService) ListCommIngestReceipts(context.Context, transport.CommIngestReceiptListRequest) (transport.CommIngestReceiptListResponse, error) {
	return transport.CommIngestReceiptListResponse{}, nil
}

func (s *stubAgentMessagesCommService) IngestMessages(context.Context, transport.MessagesIngestRequest) (transport.MessagesIngestResponse, error) {
	return transport.MessagesIngestResponse{}, nil
}

func (s *stubAgentMessagesCommService) IngestMailRuleEvent(context.Context, transport.MailRuleIngestRequest) (transport.MailRuleIngestResponse, error) {
	return transport.MailRuleIngestResponse{}, nil
}

func (s *stubAgentMessagesCommService) IngestCalendarChange(context.Context, transport.CalendarChangeIngestRequest) (transport.CalendarChangeIngestResponse, error) {
	return transport.CalendarChangeIngestResponse{}, nil
}

func (s *stubAgentMessagesCommService) IngestBrowserEvent(context.Context, transport.BrowserEventIngestRequest) (transport.BrowserEventIngestResponse, error) {
	return transport.BrowserEventIngestResponse{}, nil
}

func TestAgentDelegationServiceRunAgentDispatchesResolvedMessagesWorkflow(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	commStub := &stubAgentMessagesCommService{}
	service.SetCommService(commStub)

	response, runErr := service.RunAgent(context.Background(), transport.AgentRunRequest{
		WorkspaceID: "ws1",
		RequestText: `send an sms to +15550001111: "hello from daemon"`,
	})
	if runErr != nil {
		t.Fatalf("run agent messages workflow: %v", runErr)
	}
	if response.ClarificationRequired {
		t.Fatalf("did not expect clarification for resolved messages workflow")
	}
	if response.TaskState != "completed" || response.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", response.TaskState, response.RunState)
	}
	if len(response.StepStates) != 1 {
		t.Fatalf("expected one messages step, got %d", len(response.StepStates))
	}
	if response.StepStates[0].CapabilityKey != "messages_send_sms" {
		t.Fatalf("expected messages_send_sms capability, got %s", response.StepStates[0].CapabilityKey)
	}
	if len(commStub.requests) != 1 {
		t.Fatalf("expected one comm send request, got %d", len(commStub.requests))
	}
	if commStub.requests[0].SourceChannel != "sms" {
		t.Fatalf("expected sms source channel, got %s", commStub.requests[0].SourceChannel)
	}
	if commStub.requests[0].Destination != "+15550001111" {
		t.Fatalf("expected destination +15550001111, got %s", commStub.requests[0].Destination)
	}
	if commStub.requests[0].Message != "hello from daemon" {
		t.Fatalf("expected message body, got %s", commStub.requests[0].Message)
	}
}

func TestAgentDelegationServiceRunAgentExecutesTypedNativeActionPayload(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	commStub := &stubAgentMessagesCommService{}
	service.SetCommService(commStub)

	response, runErr := service.RunAgent(context.Background(), transport.AgentRunRequest{
		WorkspaceID: "ws1",
		NativeAction: &transport.AgentNativeAction{
			Connector: "messages",
			Operation: "send_message",
			Messages: &transport.AgentMessagesAction{
				Operation: "send_message",
				Channel:   "sms",
				Recipient: "+15550001111",
				Body:      "hello typed payload",
			},
		},
	})
	if runErr != nil {
		t.Fatalf("run typed native action: %v", runErr)
	}
	if response.ClarificationRequired {
		t.Fatalf("did not expect clarification for typed native action")
	}
	if response.TaskState != "completed" || response.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", response.TaskState, response.RunState)
	}
	if len(response.StepStates) != 1 || response.StepStates[0].CapabilityKey != "messages_send_sms" {
		t.Fatalf("expected single messages_send_sms step, got %+v", response.StepStates)
	}
	if len(commStub.requests) != 1 {
		t.Fatalf("expected one comm send request, got %d", len(commStub.requests))
	}
	if commStub.requests[0].Message != "hello typed payload" {
		t.Fatalf("unexpected message body %q", commStub.requests[0].Message)
	}
}

func TestAgentDelegationServiceGrantDelegationValidatesScopeAndPrincipalConstraints(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	_, err = service.GrantDelegation(context.Background(), transport.DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.requester",
		ScopeType:   "EXECUTION",
	})
	if err == nil || !strings.Contains(err.Error(), "self delegation") {
		t.Fatalf("expected self-delegation denial, got %v", err)
	}

	_, err = service.GrantDelegation(context.Background(), transport.DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.approver",
		ScopeType:   "INVALID_SCOPE",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported scope_type") {
		t.Fatalf("expected scope_type validation failure, got %v", err)
	}

	_, err = service.GrantDelegation(context.Background(), transport.DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.approver",
		ScopeType:   "ALL",
		ScopeKey:    "chat",
	})
	if err == nil || !strings.Contains(err.Error(), "scope_key cannot be set") {
		t.Fatalf("expected scope_key rejection for ALL scope, got %v", err)
	}
}

func TestAgentDelegationServiceGrantAndRevokePersistAuditEntries(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	grant, err := service.GrantDelegation(context.Background(), transport.DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.approver",
		ScopeType:   "EXECUTION",
	})
	if err != nil {
		t.Fatalf("grant delegation: %v", err)
	}

	var grantAuditCount int
	if err := container.DB.QueryRow(`
		SELECT COUNT(*)
		FROM audit_log_entries
		WHERE workspace_id = 'ws1' AND event_type = 'DELEGATION_RULE_GRANTED'
	`).Scan(&grantAuditCount); err != nil {
		t.Fatalf("query delegation grant audit count: %v", err)
	}
	if grantAuditCount != 1 {
		t.Fatalf("expected one delegation grant audit entry, got %d", grantAuditCount)
	}

	revoke, err := service.RevokeDelegation(context.Background(), transport.DelegationRevokeRequest{
		WorkspaceID: "ws1",
		RuleID:      grant.ID,
	})
	if err != nil {
		t.Fatalf("revoke delegation: %v", err)
	}
	if revoke.Status != "REVOKED" {
		t.Fatalf("expected revoked status, got %+v", revoke)
	}

	var revokeAuditCount int
	if err := container.DB.QueryRow(`
		SELECT COUNT(*)
		FROM audit_log_entries
		WHERE workspace_id = 'ws1' AND event_type = 'DELEGATION_RULE_REVOKED'
	`).Scan(&revokeAuditCount); err != nil {
		t.Fatalf("query delegation revoke audit count: %v", err)
	}
	if revokeAuditCount != 1 {
		t.Fatalf("expected one delegation revoke audit entry, got %d", revokeAuditCount)
	}
}

func TestAgentDelegationServiceCheckDelegationReturnsDenyReasonCodes(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	deniedRequested, err := service.CheckDelegation(context.Background(), transport.DelegationCheckRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.approver",
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		t.Fatalf("check delegation missing requested principal: %v", err)
	}
	if deniedRequested.Allowed || deniedRequested.ReasonCode != "requested_by_not_workspace_principal" {
		t.Fatalf("expected requested_by principal deny reason, got %+v", deniedRequested)
	}

	nowText := "2026-02-25T00:00:00Z"
	tx, err := container.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := ensureDelegationWorkspace(context.Background(), tx, "ws1", nowText); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if err := ensureDelegationActorPrincipal(context.Background(), tx, "ws1", "actor.requester", nowText); err != nil {
		t.Fatalf("seed requester principal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	deniedActingAs, err := service.CheckDelegation(context.Background(), transport.DelegationCheckRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.approver",
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		t.Fatalf("check delegation missing acting_as principal: %v", err)
	}
	if deniedActingAs.Allowed || deniedActingAs.ReasonCode != "acting_as_not_workspace_principal" {
		t.Fatalf("expected acting_as principal deny reason, got %+v", deniedActingAs)
	}

	tx, err = container.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := ensureDelegationActorPrincipal(context.Background(), tx, "ws1", "actor.approver", nowText); err != nil {
		t.Fatalf("seed approver principal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	deniedNoRule, err := service.CheckDelegation(context.Background(), transport.DelegationCheckRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.approver",
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		t.Fatalf("check delegation missing rule: %v", err)
	}
	if deniedNoRule.Allowed || deniedNoRule.ReasonCode != "missing_delegation_rule" {
		t.Fatalf("expected missing delegation rule deny reason, got %+v", deniedNoRule)
	}
}

type queuedRunAuthFixture struct {
	taskID      string
	runID       string
	requestedBy string
	subject     string
	actingAs    string
	title       string
	description string
}

func seedQueuedRunForAuthTest(t *testing.T, container *ServiceContainer, fixture queuedRunAuthFixture) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at)
		 VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		fmt.Sprintf(`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		 VALUES ('%s', 'ws1', 'human', '%s', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`, fixture.requestedBy, fixture.requestedBy),
		fmt.Sprintf(`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		 VALUES ('%s', 'ws1', 'human', '%s', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`, fixture.subject, fixture.subject),
		fmt.Sprintf(`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		 VALUES ('wp-ws1-%s', 'ws1', '%s', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`, fixture.requestedBy, fixture.requestedBy),
		fmt.Sprintf(`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		 VALUES ('wp-ws1-%s', 'ws1', '%s', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`, fixture.subject, fixture.subject),
		fmt.Sprintf(`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('%s', 'ws1', '%s', '%s', ?, ?, 'queued', 0, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, fixture.taskID, fixture.requestedBy, fixture.subject),
		fmt.Sprintf(`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('%s', 'ws1', '%s', '%s', 'queued', NULL, NULL, NULL, '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, fixture.runID, fixture.taskID, fixture.actingAs),
	}

	for idx, statement := range statements {
		switch idx {
		case 5:
			if _, err := container.DB.Exec(statement, fixture.title, fixture.description); err != nil {
				t.Fatalf("seed queued auth fixture failed: %v\nstatement: %s", err, statement)
			}
		default:
			if _, err := container.DB.Exec(statement); err != nil {
				t.Fatalf("seed queued auth fixture failed: %v\nstatement: %s", err, statement)
			}
		}
	}
}
