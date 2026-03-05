package retention

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupRetentionStoreDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "retention.db")
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
	return db
}

func countRetentionRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

func retentionRowExists(t *testing.T, db *sql.DB, table string, id string) bool {
	t.Helper()
	var one int
	err := db.QueryRow("SELECT 1 FROM "+table+" WHERE id = ?", id).Scan(&one)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query row exists for %s/%s: %v", table, id, err)
	}
	return true
}

func TestPurgeTraceDataBeforeDeletesOldRows(t *testing.T) {
	db := setupRetentionStoreDB(t)
	store := NewSQLiteRetentionStore(db)
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := db.Exec(`
		INSERT INTO run_artifacts(id, run_id, artifact_type, created_at) VALUES
			('artifact-old', 'run-old', 'log', '2026-02-01T00:00:00Z'),
			('artifact-new', 'run-new', 'log', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert run_artifacts: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO audit_log_entries(id, workspace_id, event_type, created_at) VALUES
			('audit-old', 'ws1', 'TRACE', '2026-02-01T00:00:00Z'),
			('audit-new', 'ws1', 'TRACE', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert audit_log_entries: %v", err)
	}

	deleted, err := store.PurgeTraceDataBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("purge trace data: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted rows, got %d", deleted)
	}
	if got := countRetentionRows(t, db, "run_artifacts"); got != 1 {
		t.Fatalf("expected 1 run_artifact row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "audit_log_entries"); got != 1 {
		t.Fatalf("expected 1 audit_log_entries row after purge, got %d", got)
	}
}

func TestPurgeTranscriptDataBeforeDeletesOldEventGraph(t *testing.T) {
	db := setupRetentionStoreDB(t)
	store := NewSQLiteRetentionStore(db)
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := db.Exec(`
		INSERT INTO comm_threads(id, workspace_id, channel, created_at, updated_at) VALUES
			('thread-old', 'ws1', 'mail', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z'),
			('thread-new', 'ws1', 'mail', '2026-03-03T00:00:00Z', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_threads: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, occurred_at, created_at, connector_id) VALUES
			('event-old', 'ws1', 'thread-old', 'MESSAGE', 'INBOUND', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z', 'mail'),
			('event-new', 'ws1', 'thread-new', 'MESSAGE', 'INBOUND', '2026-03-03T00:00:00Z', '2026-03-03T00:00:00Z', 'mail')
	`); err != nil {
		t.Fatalf("insert comm_events: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, created_at) VALUES
			('addr-old', 'event-old', 'FROM', 'old@example.com', '2026-02-01T00:00:00Z'),
			('addr-new', 'event-new', 'FROM', 'new@example.com', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_event_addresses: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_attachments(id, event_id, created_at) VALUES
			('attach-old', 'event-old', '2026-02-01T00:00:00Z'),
			('attach-new', 'event-new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_attachments: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO email_event_meta(id, event_id, created_at) VALUES
			('meta-old', 'event-old', '2026-02-01T00:00:00Z'),
			('meta-new', 'event-new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert email_event_meta: %v", err)
	}

	deleted, err := store.PurgeTranscriptDataBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("purge transcript data: %v", err)
	}
	if deleted < 4 {
		t.Fatalf("expected at least 4 deleted rows, got %d", deleted)
	}
	if got := countRetentionRows(t, db, "comm_events"); got != 1 {
		t.Fatalf("expected 1 comm_events row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "comm_event_addresses"); got != 1 {
		t.Fatalf("expected 1 comm_event_addresses row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "comm_attachments"); got != 1 {
		t.Fatalf("expected 1 comm_attachments row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "email_event_meta"); got != 1 {
		t.Fatalf("expected 1 email_event_meta row after purge, got %d", got)
	}
}

func TestPurgeTranscriptDataBeforeReturnsPartialCountWhenLaterStatementFails(t *testing.T) {
	db := setupRetentionStoreDB(t)
	store := NewSQLiteRetentionStore(db)
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := db.Exec(`
		INSERT INTO comm_threads(id, workspace_id, channel, created_at, updated_at) VALUES
			('thread-old-partial', 'ws1', 'mail', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z'),
			('thread-new-partial', 'ws1', 'mail', '2026-03-03T00:00:00Z', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_threads: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, occurred_at, created_at, connector_id) VALUES
			('event-old-partial', 'ws1', 'thread-old-partial', 'MESSAGE', 'INBOUND', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z', 'mail'),
			('event-new-partial', 'ws1', 'thread-new-partial', 'MESSAGE', 'INBOUND', '2026-03-03T00:00:00Z', '2026-03-03T00:00:00Z', 'mail')
	`); err != nil {
		t.Fatalf("insert comm_events: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, created_at) VALUES
			('addr-old-partial', 'event-old-partial', 'FROM', 'old@example.com', '2026-02-01T00:00:00Z'),
			('addr-new-partial', 'event-new-partial', 'FROM', 'new@example.com', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_event_addresses: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO comm_attachments(id, event_id, created_at) VALUES
			('attach-old-partial', 'event-old-partial', '2026-02-01T00:00:00Z'),
			('attach-new-partial', 'event-new-partial', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_attachments: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER fail_comm_attachments_delete_retention
		BEFORE DELETE ON comm_attachments
		BEGIN
			SELECT RAISE(ABORT, 'forced comm_attachments delete failure');
		END;
	`); err != nil {
		t.Fatalf("create comm_attachments failure trigger: %v", err)
	}

	deleted, err := store.PurgeTranscriptDataBefore(context.Background(), cutoff)
	if err == nil {
		t.Fatalf("expected transcript purge failure")
	}
	if !strings.Contains(err.Error(), "purge transcript data statement 2 failed") {
		t.Fatalf("expected deterministic statement-failure detail, got %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected first statement partial deletion count 1, got %d", deleted)
	}
	if retentionRowExists(t, db, "comm_event_addresses", "addr-old-partial") {
		t.Fatalf("expected first statement commit to purge old address")
	}
	if !retentionRowExists(t, db, "comm_events", "event-old-partial") {
		t.Fatalf("expected old transcript event to remain after second statement failure")
	}
	if !retentionRowExists(t, db, "comm_attachments", "attach-old-partial") {
		t.Fatalf("expected old attachment to remain after delete failure")
	}
}

func TestPurgeMemoryDataBeforeDeletesOldRowsAcrossMemoryTables(t *testing.T) {
	db := setupRetentionStoreDB(t)
	store := NewSQLiteRetentionStore(db)
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := db.Exec(`
		INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, created_at, updated_at) VALUES
			('mem-old', 'ws1', 'actor.1', 'conversation', 'old', '{"content":"old"}', 'ACTIVE', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z'),
			('mem-new', 'ws1', 'actor.1', 'conversation', 'new', '{"content":"new"}', 'ACTIVE', '2026-03-03T00:00:00Z', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert memory_items: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES
			('src-old', 'mem-old', 'comm_event', 'event-old', '2026-02-01T00:00:00Z'),
			('src-new', 'mem-new', 'comm_event', 'event-new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert memory_sources: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, status, created_at) VALUES
			('cand-old', 'ws1', 'actor.1', '{"content":"old"}', 'PENDING', '2026-02-01T00:00:00Z'),
			('cand-new', 'ws1', 'actor.1', '{"content":"new"}', 'PENDING', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert memory_candidates: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, created_at) VALUES
			('doc-old', 'ws1', 'actor.1', 'file://old', '2026-02-01T00:00:00Z'),
			('doc-new', 'ws1', 'actor.1', 'file://new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert context_documents: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO context_chunks(id, document_id, chunk_index, text_body, created_at) VALUES
			('chunk-old', 'doc-old', 0, 'old', '2026-02-01T00:00:00Z'),
			('chunk-new', 'doc-new', 0, 'new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert context_chunks: %v", err)
	}

	deleted, err := store.PurgeMemoryDataBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("purge memory data: %v", err)
	}
	if deleted < 5 {
		t.Fatalf("expected at least 5 deleted rows, got %d", deleted)
	}
	if got := countRetentionRows(t, db, "memory_items"); got != 1 {
		t.Fatalf("expected 1 memory_items row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "memory_sources"); got != 1 {
		t.Fatalf("expected 1 memory_sources row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "memory_candidates"); got != 1 {
		t.Fatalf("expected 1 memory_candidates row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "context_documents"); got != 1 {
		t.Fatalf("expected 1 context_documents row after purge, got %d", got)
	}
	if got := countRetentionRows(t, db, "context_chunks"); got != 1 {
		t.Fatalf("expected 1 context_chunks row after purge, got %d", got)
	}
}
