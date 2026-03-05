package scheduler

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	reposcheduler "personalagent/runtime/internal/core/repository/scheduler"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupSchedulerDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "scheduler.db")
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

	seedSchedulerFixtures(t, db)
	return db
}

func seedSchedulerFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_sched', 'Scheduler Workspace', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_sched', 'ws_sched', 'human', 'Scheduler Actor', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_sched', 'ws_sched', 'actor_sched', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at) VALUES ('directive_sched', 'ws_sched', 'actor_sched', 'Schedule me', 'Run schedule', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at) VALUES ('trigger_sched', 'ws_sched', 'directive_sched', 'SCHEDULE', 1, '{"interval_seconds":60}', NULL, '` + now + `', '` + now + `')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed scheduler fixture statement failed: %v", err)
		}
	}
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

func TestEvaluateSchedulesCreatesTaskOncePerSlot(t *testing.T) {
	db := setupSchedulerDB(t)
	store := reposcheduler.NewSQLiteScheduleStore(db)

	currentTime := time.Date(2026, 2, 23, 22, 0, 30, 0, time.UTC)
	engine := NewEngine(store, store, Options{Now: func() time.Time { return currentTime }})

	ctx := context.Background()
	result1, err := engine.EvaluateSchedules(ctx)
	if err != nil {
		t.Fatalf("evaluate schedules first run: %v", err)
	}
	if result1.Created != 1 {
		t.Fatalf("expected first run to create one task, got %d", result1.Created)
	}

	result2, err := engine.EvaluateSchedules(ctx)
	if err != nil {
		t.Fatalf("evaluate schedules second run: %v", err)
	}
	if result2.Created != 0 {
		t.Fatalf("expected second run in same slot to create zero tasks, got %d", result2.Created)
	}
	if result2.Skipped != 1 {
		t.Fatalf("expected one skipped trigger due to idempotency, got %d", result2.Skipped)
	}

	currentTime = currentTime.Add(65 * time.Second)
	result3, err := engine.EvaluateSchedules(ctx)
	if err != nil {
		t.Fatalf("evaluate schedules third run: %v", err)
	}
	if result3.Created != 1 {
		t.Fatalf("expected next slot to create one task, got %d", result3.Created)
	}

	if tasks := countRows(t, db, "tasks"); tasks != 2 {
		t.Fatalf("expected 2 scheduled tasks, got %d", tasks)
	}
	if fires := countRows(t, db, "trigger_fires"); fires != 2 {
		t.Fatalf("expected 2 trigger fires, got %d", fires)
	}
}

func TestEvaluateSchedulesDedupesEquivalentTriggers(t *testing.T) {
	db := setupSchedulerDB(t)
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO directives(id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at)
		 VALUES ('directive_sched_dup', 'ws_sched', 'actor_sched', 'Schedule me', 'Run schedule', 'ACTIVE', ?, ?)`,
		nowText,
		nowText,
	); err != nil {
		t.Fatalf("insert duplicate directive: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO automation_triggers(id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at)
		 VALUES ('trigger_sched_dup', 'ws_sched', 'directive_sched_dup', 'SCHEDULE', 1, '{"interval_seconds":60}', NULL, ?, ?)`,
		nowText,
		nowText,
	); err != nil {
		t.Fatalf("insert duplicate trigger: %v", err)
	}

	store := reposcheduler.NewSQLiteScheduleStore(db)
	engine := NewEngine(store, store, Options{
		Now: func() time.Time { return time.Date(2026, 2, 23, 22, 0, 30, 0, time.UTC) },
	})

	result, err := engine.EvaluateSchedules(context.Background())
	if err != nil {
		t.Fatalf("evaluate schedules with duplicate triggers: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected one task for duplicate-equivalent triggers, got %d", result.Created)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected one skipped duplicate trigger, got %d", result.Skipped)
	}
	if tasks := countRows(t, db, "tasks"); tasks != 1 {
		t.Fatalf("expected one scheduled task after dedupe, got %d", tasks)
	}
	if fires := countRows(t, db, "trigger_fires"); fires != 1 {
		t.Fatalf("expected one trigger fire after dedupe, got %d", fires)
	}
}
