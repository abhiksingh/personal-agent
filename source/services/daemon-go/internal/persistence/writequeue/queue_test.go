package writequeue

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

type probePayload struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

func setupQueueTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "queue.db")
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

	if _, err := db.Exec(`CREATE TABLE writer_probe (id TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create writer_probe table: %v", err)
	}

	return db
}

func registerProbeHandler(t *testing.T, q *Queue, db *sql.DB) {
	t.Helper()
	err := q.Register("insert_probe", func(ctx context.Context, payload json.RawMessage) error {
		p := probePayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, `INSERT INTO writer_probe(id, value) VALUES (?, ?)`, p.ID, p.Value)
		return err
	})
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}
}

func queueStatus(t *testing.T, db *sql.DB, id string) (string, int) {
	t.Helper()
	var status string
	var attempts int
	if err := db.QueryRow(`SELECT status, attempt_count FROM persistence_write_queue WHERE id = ?`, id).Scan(&status, &attempts); err != nil {
		t.Fatalf("load queue row status: %v", err)
	}
	return status, attempts
}

func TestRunOnceProcessesPendingWrite(t *testing.T) {
	db := setupQueueTestDB(t)
	q := New(db, Options{})
	registerProbeHandler(t, q, db)

	ctx := context.Background()
	id, err := q.Enqueue(ctx, "insert_probe", probePayload{ID: "p1", Value: "ok"}, 3)
	if err != nil {
		t.Fatalf("enqueue write: %v", err)
	}

	processed, err := q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !processed {
		t.Fatalf("expected one write to be processed")
	}

	var value string
	if err := db.QueryRow(`SELECT value FROM writer_probe WHERE id = ?`, "p1").Scan(&value); err != nil {
		t.Fatalf("verify inserted probe row: %v", err)
	}
	if value != "ok" {
		t.Fatalf("unexpected probe value %q", value)
	}

	status, attempts := queueStatus(t, db, id)
	if status != "completed" {
		t.Fatalf("expected completed status, got %s", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt_count=1, got %d", attempts)
	}
}

func TestRunOnceHandlerCanReenterSQLiteWithSingleWriterPool(t *testing.T) {
	db := setupQueueTestDB(t)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	q := New(db, Options{})
	if err := q.Register("insert_probe", func(ctx context.Context, payload json.RawMessage) error {
		p := probePayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		var queued int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM persistence_write_queue`).Scan(&queued); err != nil {
			return err
		}
		if queued <= 0 {
			return context.DeadlineExceeded
		}
		_, err := db.ExecContext(ctx, `INSERT INTO writer_probe(id, value) VALUES (?, ?)`, p.ID, p.Value)
		return err
	}); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := q.Enqueue(ctx, "insert_probe", probePayload{ID: "p-reentrant", Value: "ok"}, 3)
	if err != nil {
		t.Fatalf("enqueue write: %v", err)
	}

	processed, err := q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !processed {
		t.Fatalf("expected one write to be processed")
	}

	status, attempts := queueStatus(t, db, id)
	if status != "completed" {
		t.Fatalf("expected completed status, got %s", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt_count=1, got %d", attempts)
	}
}

func TestRunOnceRetriesAfterHandlerError(t *testing.T) {
	db := setupQueueTestDB(t)
	q := New(db, Options{RetryBackoff: func(attempt int) time.Duration { return 0 }})

	failFirst := true
	err := q.Register("insert_probe", func(ctx context.Context, payload json.RawMessage) error {
		if failFirst {
			failFirst = false
			return context.DeadlineExceeded
		}
		p := probePayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, `INSERT INTO writer_probe(id, value) VALUES (?, ?)`, p.ID, p.Value)
		return err
	})
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	ctx := context.Background()
	id, err := q.Enqueue(ctx, "insert_probe", probePayload{ID: "p2", Value: "retry_ok"}, 3)
	if err != nil {
		t.Fatalf("enqueue write: %v", err)
	}

	processed, err := q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("first run once: %v", err)
	}
	if !processed {
		t.Fatalf("expected first run to process queue item")
	}
	status1, attempts1 := queueStatus(t, db, id)
	if status1 != "pending" || attempts1 != 1 {
		t.Fatalf("expected pending after first failure, got status=%s attempts=%d", status1, attempts1)
	}

	processed, err = q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("second run once: %v", err)
	}
	if !processed {
		t.Fatalf("expected second run to process queue item")
	}

	status2, attempts2 := queueStatus(t, db, id)
	if status2 != "completed" || attempts2 != 2 {
		t.Fatalf("expected completed after retry, got status=%s attempts=%d", status2, attempts2)
	}
}

func TestRunOnceReplaysStaleInProgressWrites(t *testing.T) {
	base := time.Date(2026, 2, 23, 22, 0, 0, 0, time.UTC)
	now := base
	db := setupQueueTestDB(t)
	q := New(db, Options{
		Now:               func() time.Time { return now },
		InProgressTimeout: 2 * time.Second,
	})
	registerProbeHandler(t, q, db)

	ctx := context.Background()
	id, err := q.Enqueue(ctx, "insert_probe", probePayload{ID: "p3", Value: "replayed"}, 3)
	if err != nil {
		t.Fatalf("enqueue write: %v", err)
	}

	if _, err := db.Exec(
		`UPDATE persistence_write_queue
		 SET status = 'in_progress', updated_at = ?, available_at = ?
		 WHERE id = ?`,
		base.Add(-10*time.Second).Format(time.RFC3339Nano),
		base.Add(-10*time.Second).Format(time.RFC3339Nano),
		id,
	); err != nil {
		t.Fatalf("simulate stale in-progress row: %v", err)
	}

	now = base.Add(5 * time.Second)
	processed, err := q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once with replay: %v", err)
	}
	if !processed {
		t.Fatalf("expected replayed write to be processed")
	}

	var value string
	if err := db.QueryRow(`SELECT value FROM writer_probe WHERE id = ?`, "p3").Scan(&value); err != nil {
		t.Fatalf("verify replayed write result: %v", err)
	}
	if value != "replayed" {
		t.Fatalf("unexpected replayed value %q", value)
	}

	status, attempts := queueStatus(t, db, id)
	if status != "completed" {
		t.Fatalf("expected completed after replay, got %s", status)
	}
	if attempts != 1 {
		t.Fatalf("expected attempt_count=1 after replay processing, got %d", attempts)
	}
}

func TestRunOnceFinalizesAckPendingWithoutReexecutingHandlerAfterCompletionFailure(t *testing.T) {
	db := setupQueueTestDB(t)
	q := New(db, Options{})
	registerProbeHandler(t, q, db)

	ctx := context.Background()
	id, err := q.Enqueue(ctx, "insert_probe", probePayload{ID: "p4", Value: "ack_pending_once"}, 3)
	if err != nil {
		t.Fatalf("enqueue write: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TRIGGER fail_queue_complete
		BEFORE UPDATE OF status ON persistence_write_queue
		WHEN OLD.status = 'in_progress' AND OLD.last_error = '` + completionPendingMarker + `' AND NEW.status = 'completed'
		BEGIN
			SELECT RAISE(ABORT, 'forced completion failure');
		END;
	`); err != nil {
		t.Fatalf("create completion failure trigger: %v", err)
	}

	processed, runErr := q.RunOnce(ctx)
	if runErr == nil {
		t.Fatalf("expected completion mark failure")
	}
	if !processed {
		t.Fatalf("expected run to process claimed write before completion failure")
	}

	var insertedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM writer_probe WHERE id = ?`, "p4").Scan(&insertedCount); err != nil {
		t.Fatalf("count inserted probe rows: %v", err)
	}
	if insertedCount != 1 {
		t.Fatalf("expected exactly one handler side effect after first run, got %d", insertedCount)
	}

	statusAfterFailure, attemptsAfterFailure := queueStatus(t, db, id)
	if statusAfterFailure != "in_progress" || attemptsAfterFailure != 1 {
		t.Fatalf("expected in_progress attempt_count=1 after completion failure, got status=%s attempts=%d", statusAfterFailure, attemptsAfterFailure)
	}
	var lastError string
	if err := db.QueryRow(`SELECT COALESCE(last_error, '') FROM persistence_write_queue WHERE id = ?`, id).Scan(&lastError); err != nil {
		t.Fatalf("load queue last_error: %v", err)
	}
	if lastError != completionPendingMarker {
		t.Fatalf("expected completion pending marker after failure, got %q", lastError)
	}

	if _, err := db.Exec(`DROP TRIGGER fail_queue_complete`); err != nil {
		t.Fatalf("drop completion failure trigger: %v", err)
	}

	processed, err = q.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once finalize ack_pending: %v", err)
	}
	if !processed {
		t.Fatalf("expected finalize run to process ack_pending write")
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM writer_probe WHERE id = ?`, "p4").Scan(&insertedCount); err != nil {
		t.Fatalf("count inserted probe rows after finalize: %v", err)
	}
	if insertedCount != 1 {
		t.Fatalf("expected no duplicate handler side effects after finalize, got %d", insertedCount)
	}

	statusFinal, attemptsFinal := queueStatus(t, db, id)
	if statusFinal != "completed" || attemptsFinal != 1 {
		t.Fatalf("expected completed attempt_count=1 after ack finalize, got status=%s attempts=%d", statusFinal, attemptsFinal)
	}
}
