package daemonruntime

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestListApprovalInboxDefaultsToPendingAndIncludesRiskContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedApprovalInboxFixtures(t, container.DB)
	configureOllamaProviderForRouteTests(t, container)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	response, err := service.ListApprovalInbox(context.Background(), transport.ApprovalInboxRequest{
		WorkspaceID: "ws1",
	})
	if err != nil {
		t.Fatalf("list approval inbox: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Approvals) != 1 {
		t.Fatalf("expected 1 pending approval by default, got %d", len(response.Approvals))
	}

	item := response.Approvals[0]
	if item.ApprovalRequestID != "apr-pending" {
		t.Fatalf("expected pending approval apr-pending, got %s", item.ApprovalRequestID)
	}
	if item.State != "pending" {
		t.Fatalf("expected state pending, got %s", item.State)
	}
	if item.RiskLevel != "destructive" {
		t.Fatalf("expected destructive risk level, got %s", item.RiskLevel)
	}
	if item.RiskRationale == "" {
		t.Fatalf("expected risk rationale to be populated")
	}
	if !strings.Contains(strings.ToLower(item.DecisionRationale), "go ahead") {
		t.Fatalf("expected typed decision rationale to mention go ahead, got %q", item.DecisionRationale)
	}
	if item.StepCapabilityKey != "finder_delete" {
		t.Fatalf("expected finder_delete capability, got %s", item.StepCapabilityKey)
	}
	if item.RequestedByActorID != "actor.requester" || item.SubjectPrincipalActorID != "actor.subject" || item.ActingAsActorID != "actor.subject" {
		t.Fatalf("expected principal context metadata, got %+v", item)
	}
	if !item.Route.Available {
		t.Fatalf("expected route metadata to be available, got %+v", item.Route)
	}
	if item.Route.TaskClass != "finder" || item.Route.TaskClassSource != "step_capability" {
		t.Fatalf("expected finder route metadata derived from step capability, got %+v", item.Route)
	}
	if item.Route.Provider == "" || item.Route.ModelKey == "" || item.Route.RouteSource == "" {
		t.Fatalf("expected populated provider/model route metadata, got %+v", item.Route)
	}
}

func TestListApprovalInboxIncludeFinalAndStateFilter(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedApprovalInboxFixtures(t, container.DB)
	configureOllamaProviderForRouteTests(t, container)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	allResponse, err := service.ListApprovalInbox(context.Background(), transport.ApprovalInboxRequest{
		WorkspaceID:  "ws1",
		IncludeFinal: true,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("list approval inbox include final: %v", err)
	}
	if len(allResponse.Approvals) != 2 {
		t.Fatalf("expected 2 approvals when include_final=true, got %d", len(allResponse.Approvals))
	}
	if allResponse.Approvals[0].State != "pending" || allResponse.Approvals[1].State != "final" {
		t.Fatalf("expected pending before final, got states %s then %s", allResponse.Approvals[0].State, allResponse.Approvals[1].State)
	}

	finalResponse, err := service.ListApprovalInbox(context.Background(), transport.ApprovalInboxRequest{
		WorkspaceID: "ws1",
		State:       "final",
	})
	if err != nil {
		t.Fatalf("list approval inbox state=final: %v", err)
	}
	if len(finalResponse.Approvals) != 1 {
		t.Fatalf("expected 1 final approval, got %d", len(finalResponse.Approvals))
	}
	if finalResponse.Approvals[0].ApprovalRequestID != "apr-final" {
		t.Fatalf("expected final approval apr-final, got %s", finalResponse.Approvals[0].ApprovalRequestID)
	}
	if finalResponse.Approvals[0].State != "final" {
		t.Fatalf("expected final state, got %s", finalResponse.Approvals[0].State)
	}
	if finalResponse.Approvals[0].Decision != "approved" {
		t.Fatalf("expected normalized decision approved, got %s", finalResponse.Approvals[0].Decision)
	}
	if finalResponse.Approvals[0].Route.TaskClass == "" ||
		finalResponse.Approvals[0].Route.TaskClassSource == "" ||
		finalResponse.Approvals[0].Route.RouteSource == "" {
		t.Fatalf("expected final approval row to include deterministic route metadata, got %+v", finalResponse.Approvals[0].Route)
	}
}

func TestListApprovalInboxRejectsInvalidStateFilter(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	_, err = service.ListApprovalInbox(context.Background(), transport.ApprovalInboxRequest{
		WorkspaceID: "ws1",
		State:       "unknown",
	})
	if err == nil {
		t.Fatalf("expected invalid state filter error")
	}
}

func seedApprovalInboxFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.subject', 'ws1', 'human', 'Subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.subject', 'ws1', 'actor.subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-pending', 'ws1', 'actor.requester', 'actor.subject', 'Delete temp file', 'pending approval', 'awaiting_approval', 3, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-pending', 'ws1', 'task-pending', 'actor.subject', 'awaiting_approval', '2026-02-24T00:00:01Z', NULL, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		 VALUES ('step-pending', 'run-pending', 2, 'Delete /tmp/pending.txt', 'pending', 'manual', 'finder_delete', 30, 0, 0, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale)
		 VALUES ('apr-pending', 'ws1', NULL, 'step-pending', 'GO AHEAD', NULL, NULL, '2026-02-24T00:00:02Z', NULL, '{"policy_version":"capability_risk_v1","capability_key":"finder_delete","risk_level":"destructive","risk_confidence":0.99,"risk_reason":"capability finder_delete classified as destructive (file_delete)","destructive_class":"file_delete","decision":"require_confirm","decision_reason":"destructive capability requires explicit GO AHEAD approval","decision_reason_code":"missing_approval_phrase","decision_source":"approval_phrase","execution_origin":"cli"}')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-final', 'ws1', 'actor.requester', 'actor.subject', 'Delete stale report', 'finalized approval', 'completed', 2, NULL, 'app_chat', '2026-02-24T00:00:03Z', '2026-02-24T00:00:04Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-final', 'ws1', 'task-final', 'actor.subject', 'completed', '2026-02-24T00:00:03Z', '2026-02-24T00:00:05Z', '', '2026-02-24T00:00:03Z', '2026-02-24T00:00:05Z')`,
		`INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		 VALUES ('step-final', 'run-final', 2, 'Delete /tmp/final.txt', 'completed', 'manual', 'finder_delete', 30, 0, 0, '', '2026-02-24T00:00:03Z', '2026-02-24T00:00:05Z')`,
		`INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale)
		 VALUES ('apr-final', 'ws1', NULL, 'step-final', 'GO AHEAD', 'APPROVED', 'actor.subject', '2026-02-24T00:00:04Z', '2026-02-24T00:00:06Z', 'approval phrase validated')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed approval inbox fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
