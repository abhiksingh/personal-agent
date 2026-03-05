package daemonruntime

import (
	"context"
	"database/sql"
	"testing"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func TestInspectRunIncludesRouteMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedInspectRunRouteFixtures(t, container.DB)
	configureOllamaProviderForRouteTests(t, container)

	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	response, err := service.InspectRun(context.Background(), transport.InspectRunRequest{
		RunID: "run-inspect-route",
	})
	if err != nil {
		t.Fatalf("inspect run: %v", err)
	}
	if response.Run.RunID != "run-inspect-route" {
		t.Fatalf("expected inspect run id run-inspect-route, got %+v", response.Run)
	}
	if !response.Route.Available {
		t.Fatalf("expected inspect run route to be available, got %+v", response.Route)
	}
	if response.Route.TaskClass != "browser" || response.Route.TaskClassSource != "step_capability" {
		t.Fatalf("expected browser task-class route metadata derived from step capability, got %+v", response.Route)
	}
	if response.Route.Provider != providerconfig.ProviderOllama || response.Route.ModelKey == "" || response.Route.RouteSource == "" {
		t.Fatalf("expected populated provider/model/source route metadata, got %+v", response.Route)
	}
}

func seedInspectRunRouteFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.subject', 'ws1', 'human', 'Subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.subject', 'ws1', 'actor.subject', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-inspect-route', 'ws1', 'actor.requester', 'actor.subject', 'Inspect route task', 'inspect route metadata', 'running', 2, NULL, 'browser', '2026-02-24T00:00:01Z', '2026-02-24T00:00:02Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-inspect-route', 'ws1', 'task-inspect-route', 'actor.subject', 'running', '2026-02-24T00:00:01Z', NULL, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:02Z')`,
		`INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		 VALUES ('step-inspect-route', 'run-inspect-route', 1, 'Open browser', 'completed', 'auto', 'browser_open', 30, 0, 0, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:02Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed inspect run route fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
