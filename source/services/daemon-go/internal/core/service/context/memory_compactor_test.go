package context

import (
	"testing"
	"time"

	"personalagent/runtime/internal/core/types"
)

func TestCompactKeepsCanonicalAndSummarizesStaleConversationalMemory(t *testing.T) {
	now := time.Date(2026, 2, 23, 23, 50, 0, 0, time.UTC)
	compactor := NewMemoryCompactor(func() time.Time { return now })

	records := []types.MemoryRecord{
		{
			ID:            "fact_1",
			Kind:          "fact",
			Status:        "ACTIVE",
			IsCanonical:   true,
			TokenEstimate: 300,
			LastUpdatedAt: now.Add(-24 * time.Hour),
		},
		{
			ID:            "chat_old_1",
			Kind:          "conversation",
			Status:        "ACTIVE",
			TokenEstimate: 700,
			SourceRef:     "event:1",
			LastUpdatedAt: now.Add(-10 * 24 * time.Hour),
		},
		{
			ID:            "chat_old_2",
			Kind:          "conversation",
			Status:        "ACTIVE",
			TokenEstimate: 600,
			SourceRef:     "event:2",
			LastUpdatedAt: now.Add(-9 * 24 * time.Hour),
		},
		{
			ID:            "chat_new",
			Kind:          "conversation",
			Status:        "ACTIVE",
			TokenEstimate: 200,
			LastUpdatedAt: now.Add(-1 * 24 * time.Hour),
		},
	}

	result := compactor.Compact(records, types.MemoryCompactionConfig{
		TokenThreshold: 1000,
		StaleAfter:     7 * 24 * time.Hour,
	})

	if len(result.Summaries) != 1 {
		t.Fatalf("expected one summary, got %d", len(result.Summaries))
	}
	if len(result.DroppedIDs) != 2 {
		t.Fatalf("expected two dropped stale records, got %d", len(result.DroppedIDs))
	}

	keptIDs := map[string]bool{}
	for _, kept := range result.KeptRecords {
		keptIDs[kept.ID] = true
	}
	if !keptIDs["fact_1"] || !keptIDs["chat_new"] {
		t.Fatalf("expected canonical and recent records to be kept")
	}
}

func TestCompactExcludesDisabledAndDeletedMemory(t *testing.T) {
	now := time.Date(2026, 2, 23, 23, 50, 0, 0, time.UTC)
	compactor := NewMemoryCompactor(func() time.Time { return now })

	records := []types.MemoryRecord{
		{ID: "active_1", Status: "ACTIVE", TokenEstimate: 50, LastUpdatedAt: now},
		{ID: "disabled_1", Status: "DISABLED", TokenEstimate: 500, LastUpdatedAt: now},
		{ID: "deleted_1", Status: "DELETED", TokenEstimate: 500, LastUpdatedAt: now},
	}

	result := compactor.Compact(records, types.MemoryCompactionConfig{TokenThreshold: 1000})
	if result.OriginalTokens != 50 {
		t.Fatalf("expected only active tokens to count, got %d", result.OriginalTokens)
	}
	if len(result.KeptRecords) != 1 || result.KeptRecords[0].ID != "active_1" {
		t.Fatalf("expected only active records to be kept")
	}
}
