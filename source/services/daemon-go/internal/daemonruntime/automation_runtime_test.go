package daemonruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestAutomationRuntimeScheduleLoopCreatesTaskFromEnabledTrigger(t *testing.T) {
	db := newAutomationRuntimeTestDB(t)
	workspaceID := "ws_automation_schedule"
	subjectActorID := "actor.automation.schedule"
	now := time.Date(2026, time.February, 24, 12, 0, 0, 0, time.UTC)
	nowText := now.Format(time.RFC3339Nano)

	seedAutomationPrincipal(t, db, workspaceID, subjectActorID, nowText)
	if _, err := db.Exec(`
		INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at)
		VALUES ('directive_sched_auto', ?, ?, 'Auto schedule', 'Run automatically', 'ACTIVE', ?, ?)
	`, workspaceID, subjectActorID, nowText, nowText); err != nil {
		t.Fatalf("insert directive: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at)
		VALUES ('trigger_sched_auto', ?, 'directive_sched_auto', 'SCHEDULE', 1, '{"interval_seconds":1}', NULL, ?, ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert schedule trigger: %v", err)
	}

	runtime, err := NewAutomationRuntime(db, AutomationRuntimeOptions{
		SchedulePollInterval: 10 * time.Millisecond,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new automation runtime: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start automation runtime: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Stop(context.Background())
	})

	waitForAutomationCondition(t, 2*time.Second, func() bool {
		count, countErr := queryInt(db, `SELECT COUNT(*) FROM trigger_fires WHERE trigger_id = 'trigger_sched_auto'`)
		return countErr == nil && count >= 1
	})

	triggerFireCount, err := queryInt(db, `SELECT COUNT(*) FROM trigger_fires WHERE trigger_id = 'trigger_sched_auto'`)
	if err != nil {
		t.Fatalf("count trigger fires: %v", err)
	}
	if triggerFireCount != 1 {
		t.Fatalf("expected exactly one trigger fire reservation for fixed schedule slot, got %d", triggerFireCount)
	}

	taskCount, err := queryInt(db, `SELECT COUNT(*) FROM tasks WHERE title = 'Scheduled directive directive_sched_auto'`)
	if err != nil {
		t.Fatalf("count scheduled tasks: %v", err)
	}
	if taskCount != 1 {
		t.Fatalf("expected one scheduled task, got %d", taskCount)
	}
}

func TestAutomationRuntimeEvaluateCommEventCreatesTaskAndIsReplaySafe(t *testing.T) {
	db := newAutomationRuntimeTestDB(t)
	workspaceID := "ws_automation_comm"
	subjectActorID := "actor.automation.comm"
	now := time.Date(2026, time.February, 24, 13, 0, 0, 0, time.UTC)
	nowText := now.Format(time.RFC3339Nano)

	seedAutomationPrincipal(t, db, workspaceID, subjectActorID, nowText)
	if _, err := db.Exec(`
		INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at)
		VALUES ('directive_comm_auto', ?, ?, 'Auto comm', 'Run on inbound message', 'ACTIVE', ?, ?)
	`, workspaceID, subjectActorID, nowText, nowText); err != nil {
		t.Fatalf("insert directive: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at)
		VALUES ('trigger_comm_auto', ?, 'directive_comm_auto', 'ON_COMM_EVENT', 1, '{"channels":["twilio_sms"]}', NULL, ?, ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert comm trigger: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		VALUES ('thread_comm_auto', ?, 'twilio_sms', 'thread:comm:auto', 'comm auto thread', ?, ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert comm thread: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		VALUES ('event_comm_auto', ?, 'thread_comm_auto', 'MESSAGE', 'INBOUND', 0, ?, 'automation keyword ping', ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert comm event: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		VALUES ('address_comm_auto', 'event_comm_auto', 'FROM', '+15555550123', 'sender', 0, ?)
	`, nowText); err != nil {
		t.Fatalf("insert comm event from address: %v", err)
	}

	runtime, err := NewAutomationRuntime(db, AutomationRuntimeOptions{
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new automation runtime: %v", err)
	}

	firstResult, err := runtime.EvaluateCommEvent(context.Background(), "event_comm_auto")
	if err != nil {
		t.Fatalf("evaluate comm event first run: %v", err)
	}
	if firstResult.Created != 1 {
		t.Fatalf("expected first evaluation to create one task, got %+v", firstResult)
	}

	secondResult, err := runtime.EvaluateCommEvent(context.Background(), "event_comm_auto")
	if err != nil {
		t.Fatalf("evaluate comm event replay run: %v", err)
	}
	if secondResult.Created != 0 {
		t.Fatalf("expected replay evaluation to create zero tasks, got %+v", secondResult)
	}

	triggerFireCount, err := queryInt(db, `SELECT COUNT(*) FROM trigger_fires WHERE trigger_id = 'trigger_comm_auto'`)
	if err != nil {
		t.Fatalf("count trigger fires: %v", err)
	}
	if triggerFireCount != 1 {
		t.Fatalf("expected one trigger fire row after replay-safe eval, got %d", triggerFireCount)
	}

	taskCount, err := queryInt(db, `SELECT COUNT(*) FROM tasks WHERE title = 'ON_COMM_EVENT directive_comm_auto'`)
	if err != nil {
		t.Fatalf("count comm-trigger tasks: %v", err)
	}
	if taskCount != 1 {
		t.Fatalf("expected one comm-trigger task after replay-safe eval, got %d", taskCount)
	}
}

func TestAutomationRuntimeCachesStoresAndEnginesAcrossEvaluations(t *testing.T) {
	db := newAutomationRuntimeTestDB(t)
	workspaceID := "ws_automation_cache"
	subjectActorID := "actor.automation.cache"
	now := time.Date(2026, time.February, 24, 14, 0, 0, 0, time.UTC)
	nowText := now.Format(time.RFC3339Nano)

	seedAutomationPrincipal(t, db, workspaceID, subjectActorID, nowText)
	if _, err := db.Exec(`
		INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		VALUES ('thread_cache_auto', ?, 'twilio_sms', 'thread:cache:auto', 'cache thread', ?, ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert cache comm thread: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		VALUES ('event_cache_auto', ?, 'thread_cache_auto', 'MESSAGE', 'INBOUND', 0, ?, 'cache probe', ?)
	`, workspaceID, nowText, nowText); err != nil {
		t.Fatalf("insert cache comm event: %v", err)
	}

	runtime, err := NewAutomationRuntime(db, AutomationRuntimeOptions{
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new automation runtime: %v", err)
	}

	initialScheduleStore := runtime.scheduleStore
	initialScheduleEngine := runtime.scheduleEngine
	initialCommStore := runtime.commTriggerStore
	initialCommEngine := runtime.commTriggerEngine
	if initialScheduleStore == nil || initialScheduleEngine == nil || initialCommStore == nil || initialCommEngine == nil {
		t.Fatalf("expected automation runtime stores and engines to be initialized")
	}

	for idx := 0; idx < 2; idx++ {
		if _, err := runtime.EvaluateSchedule(context.Background()); err != nil {
			t.Fatalf("evaluate schedule run %d: %v", idx+1, err)
		}
		if _, err := runtime.EvaluateCommEvent(context.Background(), "event_cache_auto"); err != nil {
			t.Fatalf("evaluate comm event run %d: %v", idx+1, err)
		}
	}

	if runtime.scheduleStore != initialScheduleStore {
		t.Fatalf("expected schedule store to be cached across evaluations")
	}
	if runtime.scheduleEngine != initialScheduleEngine {
		t.Fatalf("expected schedule engine to be cached across evaluations")
	}
	if runtime.commTriggerStore != initialCommStore {
		t.Fatalf("expected comm trigger store to be cached across evaluations")
	}
	if runtime.commTriggerEngine != initialCommEngine {
		t.Fatalf("expected comm trigger engine to be cached across evaluations")
	}
}

func seedAutomationPrincipal(t *testing.T, db *sql.DB, workspaceID string, actorID string, nowText string) {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin seed tx: %v", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(context.Background(), tx, workspaceID, nowText); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	if err := ensureDelegationActorPrincipal(context.Background(), tx, workspaceID, actorID, nowText); err != nil {
		t.Fatalf("ensure actor principal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed tx: %v", err)
	}
}

func newAutomationRuntimeTestDB(t *testing.T) *sql.DB {
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

func queryInt(db *sql.DB, query string, args ...any) (int, error) {
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func waitForAutomationCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}
