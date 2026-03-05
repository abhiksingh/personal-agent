package daemonruntime

import (
	"context"
	"database/sql"
	"testing"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func TestListTaskRunsReturnsPrincipalStateAndErrorMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedTaskRunListFixtures(t, container.DB)
	configureOllamaProviderForRouteTests(t, container)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	response, err := service.ListTaskRuns(context.Background(), transport.TaskRunListRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list task runs: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 task/run rows, got %d", len(response.Items))
	}

	first := response.Items[0]
	if first.TaskID != "task-running" || first.RunID != "run-running" {
		t.Fatalf("expected newest running task/run first, got %+v", first)
	}
	if first.TaskState != "running" || first.RunState != "running" {
		t.Fatalf("expected running state metadata, got task=%s run=%s", first.TaskState, first.RunState)
	}
	if !first.Actions.CanCancel || first.Actions.CanRetry || first.Actions.CanRequeue {
		t.Fatalf("expected running action availability {cancel:true,retry:false,requeue:false}, got %+v", first.Actions)
	}
	if first.Priority != 3 {
		t.Fatalf("expected task priority 3, got %d", first.Priority)
	}
	if first.RequestedByActorID != "actor.requester" || first.SubjectPrincipalActorID != "actor.subject" || first.ActingAsActorID != "actor.subject" {
		t.Fatalf("expected principal context metadata, got %+v", first)
	}
	if first.LastError != "connector timeout" {
		t.Fatalf("expected last_error metadata, got %q", first.LastError)
	}
	if !first.Route.Available {
		t.Fatalf("expected route metadata to be available, got %+v", first.Route)
	}
	if first.Route.Provider != providerconfig.ProviderOllama || first.Route.ModelKey == "" {
		t.Fatalf("expected ollama route metadata, got %+v", first.Route)
	}
	if first.Route.TaskClass != "chat" || first.Route.TaskClassSource != "run_task_channel" {
		t.Fatalf("expected deterministic route task-class metadata, got %+v", first.Route)
	}
	if first.Route.RouteSource == "" {
		t.Fatalf("expected route source to be populated, got %+v", first.Route)
	}
}

func TestListTaskRunsStateFilterReturnsOnlyMatchingRows(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedTaskRunListFixtures(t, container.DB)
	configureOllamaProviderForRouteTests(t, container)

	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	filtered, err := service.ListTaskRuns(context.Background(), transport.TaskRunListRequest{
		WorkspaceID: "ws1",
		State:       "completed",
	})
	if err != nil {
		t.Fatalf("list task runs state filter: %v", err)
	}
	if len(filtered.Items) != 1 {
		t.Fatalf("expected 1 completed row, got %d", len(filtered.Items))
	}
	if filtered.Items[0].TaskID != "task-completed" || filtered.Items[0].RunState != "completed" {
		t.Fatalf("unexpected filtered row: %+v", filtered.Items[0])
	}
	if filtered.Items[0].Actions.CanCancel || filtered.Items[0].Actions.CanRetry || filtered.Items[0].Actions.CanRequeue {
		t.Fatalf("expected completed action availability to disable controls, got %+v", filtered.Items[0].Actions)
	}
	if filtered.Items[0].Route.TaskClass == "" || filtered.Items[0].Route.TaskClassSource == "" || filtered.Items[0].Route.RouteSource == "" {
		t.Fatalf("expected filtered row to include deterministic route metadata, got %+v", filtered.Items[0].Route)
	}
}

func seedTaskRunListFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.subject', 'ws1', 'human', 'Subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.subject', 'ws1', 'actor.subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-completed', 'ws1', 'actor.requester', 'actor.subject', 'Completed task', 'done', 'completed', 1, NULL, 'app', '2026-02-24T00:00:01Z', '2026-02-24T00:00:03Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-completed', 'ws1', 'task-completed', 'actor.subject', 'completed', '2026-02-24T00:00:01Z', '2026-02-24T00:00:03Z', '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:03Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-running', 'ws1', 'actor.requester', 'actor.subject', 'Running task', 'still running', 'running', 3, NULL, 'app', '2026-02-24T00:00:04Z', '2026-02-24T00:00:06Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-running', 'ws1', 'task-running', 'actor.subject', 'running', '2026-02-24T00:00:05Z', NULL, 'connector timeout', '2026-02-24T00:00:05Z', '2026-02-24T00:00:06Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed task run list fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}

func configureOllamaProviderForRouteTests(t *testing.T, container *ServiceContainer) {
	t.Helper()
	if container == nil || container.ProviderConfigStore == nil {
		t.Fatalf("provider config store is not configured")
	}
	if _, err := container.ProviderConfigStore.Upsert(context.Background(), providerconfig.UpsertInput{
		WorkspaceID: "ws1",
		Provider:    providerconfig.ProviderOllama,
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("configure ollama provider: %v", err)
	}
}
