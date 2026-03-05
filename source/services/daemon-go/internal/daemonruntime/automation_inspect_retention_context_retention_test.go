package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	coretypes "personalagent/runtime/internal/core/types"
)

func newRetentionCompactionTestDB(t *testing.T) *sql.DB {
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

func TestApplyMemoryCompactionChunksDroppedUpdatesAndInsertsSummary(t *testing.T) {
	db := newRetentionCompactionTestDB(t)
	ctx := context.Background()
	now := "2026-03-04T12:00:00Z"

	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES (?, ?, 'ACTIVE', ?, ?)
	`, "ws1", "ws1", now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
	`, "actor.1", "ws1", "actor.1", now, now); err != nil {
		t.Fatalf("insert actor: %v", err)
	}

	droppedIDs := make([]string, 0, 260)
	for index := 0; index < 260; index++ {
		id := fmt.Sprintf("mem-%03d", index)
		droppedIDs = append(droppedIDs, id)
		if _, err := db.Exec(`
			INSERT INTO memory_items(
				id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at
			) VALUES (?, ?, ?, 'conversation', ?, ?, 'ACTIVE', NULL, ?, ?)
		`, id, "ws1", "actor.1", fmt.Sprintf("k-%03d", index), fmt.Sprintf(`{"content":"memory-%03d"}`, index), now, now); err != nil {
			t.Fatalf("insert memory item %s: %v", id, err)
		}
	}

	result := coretypes.MemoryCompactionResult{
		DroppedIDs: droppedIDs,
		Summaries: []coretypes.MemorySummary{
			{
				SummaryID:     "summary-260",
				SourceIDs:     droppedIDs[:4],
				SourceRefs:    []string{"event://1", "event://2"},
				Content:       "Summarized stale memories.",
				TokenEstimate: 64,
			},
		},
	}

	summaryIDs, err := applyMemoryCompaction(ctx, db, "ws1", "actor.1", result)
	if err != nil {
		t.Fatalf("apply memory compaction: %v", err)
	}
	if len(summaryIDs) != 1 {
		t.Fatalf("expected one summary id, got %d", len(summaryIDs))
	}

	var disabledCount int
	if err := db.QueryRow(`
		SELECT COUNT(1)
		FROM memory_items
		WHERE workspace_id = ? AND owner_principal_actor_id = ? AND id LIKE 'mem-%' AND status = 'DISABLED'
	`, "ws1", "actor.1").Scan(&disabledCount); err != nil {
		t.Fatalf("count disabled memory items: %v", err)
	}
	if disabledCount != 260 {
		t.Fatalf("expected 260 disabled memory records, got %d", disabledCount)
	}

	var summaryStatus string
	if err := db.QueryRow(`SELECT status FROM memory_items WHERE id = ?`, summaryIDs[0]).Scan(&summaryStatus); err != nil {
		t.Fatalf("query inserted summary memory item: %v", err)
	}
	if summaryStatus != "ACTIVE" {
		t.Fatalf("expected summary memory item status ACTIVE, got %s", summaryStatus)
	}
}

func TestApplyMemoryCompactionReturnsNoSummaryIDsWhenNoSummaries(t *testing.T) {
	db := newRetentionCompactionTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES (?, ?, 'ACTIVE', ?, ?)
	`, "ws1", "ws1", now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
	`, "actor.1", "ws1", "actor.1", now, now); err != nil {
		t.Fatalf("insert actor: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO memory_items(
			id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at
		) VALUES (?, ?, ?, 'conversation', ?, ?, 'ACTIVE', NULL, ?, ?)
	`, "mem-1", "ws1", "actor.1", "k-1", `{"content":"memory-1"}`, now, now); err != nil {
		t.Fatalf("insert memory item: %v", err)
	}

	summaryIDs, err := applyMemoryCompaction(ctx, db, "ws1", "actor.1", coretypes.MemoryCompactionResult{
		DroppedIDs: []string{"mem-1"},
		Summaries:  nil,
	})
	if err != nil {
		t.Fatalf("apply memory compaction: %v", err)
	}
	if len(summaryIDs) != 0 {
		t.Fatalf("expected no summary ids, got %d", len(summaryIDs))
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM memory_items WHERE id = ?`, "mem-1").Scan(&status); err != nil {
		t.Fatalf("query memory status: %v", err)
	}
	if status != "DISABLED" {
		t.Fatalf("expected dropped memory item to be DISABLED, got %s", status)
	}
}

func TestApplyMemoryCompactionRollsBackDroppedDisablesWhenSummaryInsertFails(t *testing.T) {
	db := newRetentionCompactionTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES (?, ?, 'ACTIVE', ?, ?)
	`, "ws1", "ws1", now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
	`, "actor.1", "ws1", "actor.1", now, now); err != nil {
		t.Fatalf("insert actor: %v", err)
	}

	for _, id := range []string{"mem-1", "mem-2", "mem-3"} {
		if _, err := db.Exec(`
			INSERT INTO memory_items(
				id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at
			) VALUES (?, ?, ?, 'conversation', ?, ?, 'ACTIVE', NULL, ?, ?)
		`, id, "ws1", "actor.1", id, `{"content":"memory"}`, now, now); err != nil {
			t.Fatalf("insert memory item %s: %v", id, err)
		}
	}

	if _, err := db.Exec(`
		CREATE TRIGGER fail_memory_summary_insert
		BEFORE INSERT ON memory_items
		WHEN NEW.key LIKE 'summary_%'
		BEGIN
			SELECT RAISE(ABORT, 'forced summary insert failure');
		END;
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	_, err := applyMemoryCompaction(ctx, db, "ws1", "actor.1", coretypes.MemoryCompactionResult{
		DroppedIDs: []string{"mem-1", "mem-2", "mem-3"},
		Summaries: []coretypes.MemorySummary{
			{
				SummaryID:     "summary-fail",
				SourceIDs:     []string{"mem-1", "mem-2"},
				SourceRefs:    []string{"event://x"},
				Content:       "Should fail insert",
				TokenEstimate: 16,
			},
		},
	})
	if err == nil {
		t.Fatalf("expected summary insert failure")
	}

	var disabledCount int
	if err := db.QueryRow(`
		SELECT COUNT(1)
		FROM memory_items
		WHERE workspace_id = ? AND owner_principal_actor_id = ? AND id IN ('mem-1','mem-2','mem-3') AND status = 'DISABLED'
	`, "ws1", "actor.1").Scan(&disabledCount); err != nil {
		t.Fatalf("count disabled memory items: %v", err)
	}
	if disabledCount != 0 {
		t.Fatalf("expected rollback to keep dropped records ACTIVE, found %d disabled", disabledCount)
	}

	var summaryCount int
	if err := db.QueryRow(`
		SELECT COUNT(1)
		FROM memory_items
		WHERE workspace_id = ? AND owner_principal_actor_id = ? AND key LIKE 'summary_%'
	`, "ws1", "actor.1").Scan(&summaryCount); err != nil {
		t.Fatalf("count summary memory items: %v", err)
	}
	if summaryCount != 0 {
		t.Fatalf("expected no summary items after rollback, got %d", summaryCount)
	}
}
