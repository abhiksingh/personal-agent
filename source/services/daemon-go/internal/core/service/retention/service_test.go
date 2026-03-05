package retention

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reporetention "personalagent/runtime/internal/core/repository/retention"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupRetentionDB(t *testing.T, now time.Time) *sql.DB {
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

	seedRetentionFixtures(t, db, now)
	return db
}

func seedRetentionFixtures(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()
	old := now.Add(-8 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	recent := now.Add(-2 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_ret', 'Retention WS', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_ret', 'ws_ret', 'human', 'Retention Actor', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_ret', 'ws_ret', 'actor_ret', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at) VALUES ('thread_ret', 'ws_ret', 'voice', 'thread-ext', 'Voice Thread', '` + recent + `', '` + recent + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_old', 'ws_ret', 'thread_ret', 'VOICE_TRANSCRIPT', 'INBOUND', 0, '` + old + `', 'old transcript', '` + old + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_new', 'ws_ret', 'thread_ret', 'VOICE_TRANSCRIPT', 'INBOUND', 0, '` + recent + `', 'new transcript', '` + recent + `')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr_old', 'event_old', 'FROM', 'old@example.com', 'Old', 0, '` + old + `')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr_new', 'event_new', 'FROM', 'new@example.com', 'New', 0, '` + recent + `')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit_old', 'ws_ret', NULL, NULL, 'TRACE', 'actor_ret', 'actor_ret', 'c1', '{}', '` + old + `')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit_new', 'ws_ret', NULL, NULL, 'TRACE', 'actor_ret', 'actor_ret', 'c2', '{}', '` + recent + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_old', 'ws_ret', 'actor_ret', 'profile', 'k_old', '{}', 'ACTIVE', 'old', '` + old + `', '` + old + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_new', 'ws_ret', 'actor_ret', 'profile', 'k_new', '{}', 'ACTIVE', 'new', '` + recent + `', '` + recent + `')`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('ms_old', 'mem_old', 'comm', 'event_old', '` + old + `')`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('ms_new', 'mem_new', 'comm', 'event_new', '` + recent + `')`,
		`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at) VALUES ('mc_old', 'ws_ret', 'actor_ret', '{}', 0.2, 'ACTIVE', '` + old + `')`,
		`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at) VALUES ('mc_new', 'ws_ret', 'actor_ret', '{}', 0.9, 'ACTIVE', '` + recent + `')`,
		`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at) VALUES ('doc_old', 'ws_ret', 'actor_ret', 'old://doc', 'x', '` + old + `')`,
		`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at) VALUES ('doc_new', 'ws_ret', 'actor_ret', 'new://doc', 'y', '` + recent + `')`,
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at) VALUES ('chunk_old', 'doc_old', 0, 'old text', 2, '` + old + `')`,
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at) VALUES ('chunk_new', 'doc_new', 0, 'new text', 2, '` + recent + `')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed retention fixture failed: %v", err)
		}
	}
}

func rowExists(t *testing.T, db *sql.DB, table, id string) bool {
	t.Helper()
	var one int
	err := db.QueryRow(`SELECT 1 FROM `+table+` WHERE id = ?`, id).Scan(&one)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query row exists for %s/%s: %v", table, id, err)
	}
	return true
}

func TestPurgeUsesSevenDayDefaults(t *testing.T) {
	now := time.Date(2026, 2, 23, 23, 30, 0, 0, time.UTC)
	db := setupRetentionDB(t, now)
	service := NewService(reporetention.NewSQLiteRetentionStore(db), func() time.Time { return now })

	result, err := service.Purge(context.Background(), nil)
	if err != nil {
		t.Fatalf("purge with default policy: %v", err)
	}
	if result.ConsistencyMode != types.RetentionPurgeConsistencyModePartialSuccess {
		t.Fatalf("expected consistency mode %q, got %q", types.RetentionPurgeConsistencyModePartialSuccess, result.ConsistencyMode)
	}
	if result.Status != types.RetentionPurgeStatusCompleted {
		t.Fatalf("expected purge status %q, got %q", types.RetentionPurgeStatusCompleted, result.Status)
	}
	if result.Failure != nil {
		t.Fatalf("expected no failure payload for successful purge, got %+v", result.Failure)
	}
	if result.TracesDeleted == 0 || result.TranscriptsDeleted == 0 || result.MemoryDeleted == 0 {
		t.Fatalf("expected deletions across all retention categories, got %+v", result)
	}

	if rowExists(t, db, "audit_log_entries", "audit_old") {
		t.Fatalf("expected old trace data to be purged")
	}
	if !rowExists(t, db, "audit_log_entries", "audit_new") {
		t.Fatalf("expected recent trace data to be retained")
	}

	if rowExists(t, db, "comm_events", "event_old") {
		t.Fatalf("expected old transcript event to be purged")
	}
	if !rowExists(t, db, "comm_events", "event_new") {
		t.Fatalf("expected recent transcript event to be retained")
	}

	if rowExists(t, db, "memory_items", "mem_old") {
		t.Fatalf("expected old memory item to be purged")
	}
	if !rowExists(t, db, "memory_items", "mem_new") {
		t.Fatalf("expected recent memory item to be retained")
	}
}

func TestPurgeSupportsPolicyOverrides(t *testing.T) {
	now := time.Date(2026, 2, 23, 23, 30, 0, 0, time.UTC)
	db := setupRetentionDB(t, now)
	service := NewService(reporetention.NewSQLiteRetentionStore(db), func() time.Time { return now })

	_, err := service.Purge(context.Background(), &types.RetentionPolicy{
		TraceDays:      1,
		TranscriptDays: 1,
		MemoryDays:     1,
	})
	if err != nil {
		t.Fatalf("purge with override policy: %v", err)
	}

	if rowExists(t, db, "audit_log_entries", "audit_new") {
		t.Fatalf("expected stricter override to purge recent trace data")
	}
	if rowExists(t, db, "comm_events", "event_new") {
		t.Fatalf("expected stricter override to purge recent transcript data")
	}
	if rowExists(t, db, "memory_items", "mem_new") {
		t.Fatalf("expected stricter override to purge recent memory data")
	}
}

func TestPurgeReturnsPartialFailureResultWhenTranscriptStatementFailsAfterPriorCommits(t *testing.T) {
	now := time.Date(2026, 2, 23, 23, 30, 0, 0, time.UTC)
	db := setupRetentionDB(t, now)
	service := NewService(reporetention.NewSQLiteRetentionStore(db), func() time.Time { return now })

	if _, err := db.Exec(`
		INSERT INTO comm_attachments(id, event_id, created_at) VALUES
			('attach_old_failure', 'event_old', '2026-02-01T00:00:00Z'),
			('attach_new_failure', 'event_new', '2026-03-03T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert comm_attachments: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER fail_retention_comm_attachments_delete
		BEFORE DELETE ON comm_attachments
		BEGIN
			SELECT RAISE(ABORT, 'forced comm_attachments delete failure');
		END;
	`); err != nil {
		t.Fatalf("create transcript failure trigger: %v", err)
	}

	result, err := service.Purge(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected typed partial-failure purge result, got error: %v", err)
	}
	if result.ConsistencyMode != types.RetentionPurgeConsistencyModePartialSuccess {
		t.Fatalf("expected consistency mode %q, got %q", types.RetentionPurgeConsistencyModePartialSuccess, result.ConsistencyMode)
	}
	if result.Status != types.RetentionPurgeStatusPartialFailure {
		t.Fatalf("expected purge status %q, got %q", types.RetentionPurgeStatusPartialFailure, result.Status)
	}
	if result.Failure == nil {
		t.Fatalf("expected failure metadata for transcript failure, got nil")
	}
	if result.Failure.Stage != types.RetentionPurgeFailureStageTranscript {
		t.Fatalf("expected failure stage %q, got %q", types.RetentionPurgeFailureStageTranscript, result.Failure.Stage)
	}
	if result.Failure.Code != types.RetentionPurgeFailureCodeStatementFailed {
		t.Fatalf("expected failure code %q, got %q", types.RetentionPurgeFailureCodeStatementFailed, result.Failure.Code)
	}
	if !strings.Contains(result.Failure.Details, "purge transcript data statement 2 failed") {
		t.Fatalf("expected deterministic transcript failure detail, got %q", result.Failure.Details)
	}
	if result.TracesDeleted == 0 {
		t.Fatalf("expected trace deletions from committed earlier phase, got %+v", result)
	}
	if result.TranscriptsDeleted == 0 {
		t.Fatalf("expected transcript partial deletions committed before failure, got %+v", result)
	}
	if result.MemoryDeleted != 0 {
		t.Fatalf("expected memory purge not to run after transcript failure, got %+v", result)
	}

	if rowExists(t, db, "audit_log_entries", "audit_old") {
		t.Fatalf("expected old trace row to be purged before transcript failure")
	}
	if rowExists(t, db, "comm_event_addresses", "addr_old") {
		t.Fatalf("expected first transcript statement commit to purge old event address")
	}
	if !rowExists(t, db, "comm_events", "event_old") {
		t.Fatalf("expected old transcript event to remain after transcript mid-sequence failure")
	}
	if !rowExists(t, db, "memory_items", "mem_old") {
		t.Fatalf("expected memory purge to be skipped after transcript failure")
	}
}
