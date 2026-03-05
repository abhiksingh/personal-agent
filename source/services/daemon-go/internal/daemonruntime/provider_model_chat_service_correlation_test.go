package daemonruntime

import (
	"context"
	"database/sql"
	"testing"
)

func TestResolveChatTurnTaskRunCorrelationFromAuditLog(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedChatTurnCorrelationFixtures(t, container.DB)

	service := NewProviderModelChatService(container)
	correlation := service.resolveChatTurnTaskRunCorrelation(context.Background(), "corr-chat-1")

	if !correlation.Available {
		t.Fatalf("expected available=true, got %+v", correlation)
	}
	if correlation.Source != "audit_log_entry" {
		t.Fatalf("expected source audit_log_entry, got %s", correlation.Source)
	}
	if correlation.TaskID != "task-chat-1" || correlation.RunID != "run-chat-1" {
		t.Fatalf("expected task/run correlation task-chat-1/run-chat-1, got %+v", correlation)
	}
	if correlation.TaskState != "running" || correlation.RunState != "running" {
		t.Fatalf("expected running states, got task=%s run=%s", correlation.TaskState, correlation.RunState)
	}
}

func TestResolveChatTurnTaskRunCorrelationMissingReturnsDeterministicNone(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	correlation := service.resolveChatTurnTaskRunCorrelation(context.Background(), "corr-missing")
	if correlation.Available {
		t.Fatalf("expected available=false, got %+v", correlation)
	}
	if correlation.Source != "none" {
		t.Fatalf("expected source none, got %s", correlation.Source)
	}
	if correlation.TaskID != "" || correlation.RunID != "" {
		t.Fatalf("expected empty task/run fields, got %+v", correlation)
	}
}

func seedChatTurnCorrelationFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-chat-1', 'ws1', 'actor.requester', 'actor.requester', 'Chat correlated task', 'desc', 'running', 2, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:03Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-chat-1', 'ws1', 'task-chat-1', 'actor.requester', 'running', '2026-02-24T00:00:02Z', NULL, '', '2026-02-24T00:00:02Z', '2026-02-24T00:00:03Z')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at)
		 VALUES ('audit-chat-1', 'ws1', 'run-chat-1', NULL, 'task_run_lifecycle', 'actor.requester', 'actor.requester', 'corr-chat-1', '{}', '2026-02-24T00:00:03Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed chat correlation fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
