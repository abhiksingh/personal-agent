package commtrigger

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	repocomm "personalagent/runtime/internal/core/repository/commtrigger"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupCommTriggerDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "commtrigger.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	seedCommTriggerFixtures(t, db)
	return db
}

func seedCommTriggerFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_comm', 'Comm Workspace', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_comm', 'ws_comm', 'human', 'Comm Actor', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_comm', 'ws_comm', 'actor_comm', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at) VALUES ('directive_comm', 'ws_comm', 'actor_comm', 'Comm Trigger', 'Handle inbound messages', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at) VALUES ('trigger_comm', 'ws_comm', 'directive_comm', 'ON_COMM_EVENT', 1, '{"channels":["imessage"]}', NULL, '` + now + `', '` + now + `')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at) VALUES ('thread_1', 'ws_comm', 'imessage', 'ext_1', 'Thread 1', '` + now + `', '` + now + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_inbound', 'ws_comm', 'thread_1', 'MESSAGE', 'INBOUND', 0, '` + now + `', 'Please handle this urgent request', '` + now + `')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr_from_1', 'event_inbound', 'FROM', 'sender@example.com', 'Sender', 0, '` + now + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_outbound', 'ws_comm', 'thread_1', 'MESSAGE', 'OUTBOUND', 0, '` + now + `', 'Outbound message', '` + now + `')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed comm-trigger fixture failed: %v", err)
		}
	}
}

func countTableRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count rows for %s: %v", table, err)
	}
	return count
}

func TestEvaluateEventCreatesTaskAndIsIdempotent(t *testing.T) {
	db := setupCommTriggerDB(t)
	store := repocomm.NewSQLiteCommTriggerStore(db)
	engine := NewEngine(store, Options{Now: func() time.Time { return time.Date(2026, 2, 23, 22, 10, 0, 0, time.UTC) }})

	ctx := context.Background()
	result1, err := engine.EvaluateEvent(ctx, "event_inbound")
	if err != nil {
		t.Fatalf("evaluate inbound event first run: %v", err)
	}
	if result1.Created != 1 {
		t.Fatalf("expected first run to create one task, got %d", result1.Created)
	}

	result2, err := engine.EvaluateEvent(ctx, "event_inbound")
	if err != nil {
		t.Fatalf("evaluate inbound event second run: %v", err)
	}
	if result2.Created != 0 {
		t.Fatalf("expected second run to create zero tasks due to idempotency, got %d", result2.Created)
	}

	if tasks := countTableRows(t, db, "tasks"); tasks != 1 {
		t.Fatalf("expected exactly one task from idempotent event processing, got %d", tasks)
	}
}

func TestEvaluateEventSkipsWhenDefaultFiltersFail(t *testing.T) {
	db := setupCommTriggerDB(t)
	store := repocomm.NewSQLiteCommTriggerStore(db)
	engine := NewEngine(store, Options{})

	ctx := context.Background()
	result, err := engine.EvaluateEvent(ctx, "event_outbound")
	if err != nil {
		t.Fatalf("evaluate outbound event: %v", err)
	}
	if result.Created != 0 {
		t.Fatalf("expected no task creation when default filters fail")
	}
	if tasks := countTableRows(t, db, "tasks"); tasks != 0 {
		t.Fatalf("expected no tasks for outbound event, got %d", tasks)
	}
}

func TestEvaluateEventDedupesEquivalentTriggers(t *testing.T) {
	db := setupCommTriggerDB(t)
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at)
		 VALUES ('directive_comm_dup', 'ws_comm', 'actor_comm', 'Comm Trigger', 'Handle inbound messages', 'ACTIVE', ?, ?)`,
		nowText,
		nowText,
	); err != nil {
		t.Fatalf("insert duplicate comm directive: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at)
		 VALUES ('trigger_comm_dup', 'ws_comm', 'directive_comm_dup', 'ON_COMM_EVENT', 1, '{"channels":["imessage"]}', NULL, ?, ?)`,
		nowText,
		nowText,
	); err != nil {
		t.Fatalf("insert duplicate comm trigger: %v", err)
	}

	store := repocomm.NewSQLiteCommTriggerStore(db)
	engine := NewEngine(store, Options{Now: func() time.Time { return time.Date(2026, 2, 23, 22, 10, 0, 0, time.UTC) }})
	result, err := engine.EvaluateEvent(context.Background(), "event_inbound")
	if err != nil {
		t.Fatalf("evaluate inbound event with duplicate triggers: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected one task for duplicate-equivalent comm triggers, got %d", result.Created)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected one skipped duplicate comm trigger, got %d", result.Skipped)
	}
	if tasks := countTableRows(t, db, "tasks"); tasks != 1 {
		t.Fatalf("expected one task after comm-trigger dedupe, got %d", tasks)
	}
	if fires := countTableRows(t, db, "trigger_fires"); fires != 1 {
		t.Fatalf("expected one trigger fire after comm-trigger dedupe, got %d", fires)
	}
}
