package contextrepo

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupContextBudgetDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "context-budget.db")
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO workspaces(id, name, status, created_at, updated_at)
		 VALUES ('ws_ctx', 'Context Workspace', 'ACTIVE', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	return db
}

func TestRecordAndListContextBudgetSamples(t *testing.T) {
	db := setupContextBudgetDB(t)
	store := NewSQLiteBudgetTelemetryStore(db)
	ctx := context.Background()

	firstAt := time.Date(2026, 2, 23, 21, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(3 * time.Minute)

	if err := store.RecordContextBudgetSample(ctx, types.ContextBudgetSample{
		SampleID:         "sample-1",
		WorkspaceID:      "ws_ctx",
		TaskClass:        "chat",
		ModelKey:         "gpt-5-mini",
		ContextWindow:    128000,
		OutputLimit:      4096,
		DeepAnalysis:     false,
		RemainingBudget:  100000,
		RetrievalTarget:  24000,
		RetrievalUsed:    12000,
		PromptTokens:     22000,
		CompletionTokens: 1200,
		RecordedAt:       firstAt,
	}); err != nil {
		t.Fatalf("record first sample: %v", err)
	}

	if err := store.RecordContextBudgetSample(ctx, types.ContextBudgetSample{
		SampleID:         "sample-2",
		WorkspaceID:      "ws_ctx",
		TaskClass:        "chat",
		ModelKey:         "gpt-5-mini",
		ContextWindow:    128000,
		OutputLimit:      4096,
		DeepAnalysis:     false,
		RemainingBudget:  100000,
		RetrievalTarget:  24000,
		RetrievalUsed:    20000,
		PromptTokens:     28000,
		CompletionTokens: 1800,
		RecordedAt:       secondAt,
	}); err != nil {
		t.Fatalf("record second sample: %v", err)
	}

	samples, err := store.ListRecentContextBudgetSamples(ctx, "ws_ctx", "chat", 10)
	if err != nil {
		t.Fatalf("list recent samples: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[0].SampleID != "sample-2" {
		t.Fatalf("expected newest sample first, got %s", samples[0].SampleID)
	}
}

func TestUpsertAndGetContextBudgetTuningProfile(t *testing.T) {
	db := setupContextBudgetDB(t)
	store := NewSQLiteBudgetTelemetryStore(db)
	ctx := context.Background()

	if _, exists, err := store.GetContextBudgetTuningProfile(ctx, "ws_ctx", "chat"); err != nil {
		t.Fatalf("get missing profile: %v", err)
	} else if exists {
		t.Fatalf("expected missing profile")
	}

	first := types.ContextBudgetTuningProfile{
		WorkspaceID:             "ws_ctx",
		TaskClass:               "chat",
		RetrievalMultiplier:     1.1,
		SampleCount:             8,
		AvgRetrievalUtilization: 0.84,
		AvgPromptUtilization:    0.42,
		UpdatedAt:               time.Date(2026, 2, 23, 21, 0, 0, 0, time.UTC),
	}
	if err := store.UpsertContextBudgetTuningProfile(ctx, first); err != nil {
		t.Fatalf("upsert first profile: %v", err)
	}

	second := types.ContextBudgetTuningProfile{
		WorkspaceID:             "ws_ctx",
		TaskClass:               "chat",
		RetrievalMultiplier:     0.9,
		SampleCount:             14,
		AvgRetrievalUtilization: 0.33,
		AvgPromptUtilization:    0.21,
		UpdatedAt:               time.Date(2026, 2, 23, 22, 0, 0, 0, time.UTC),
	}
	if err := store.UpsertContextBudgetTuningProfile(ctx, second); err != nil {
		t.Fatalf("upsert second profile: %v", err)
	}

	stored, exists, err := store.GetContextBudgetTuningProfile(ctx, "ws_ctx", "chat")
	if err != nil {
		t.Fatalf("get stored profile: %v", err)
	}
	if !exists {
		t.Fatalf("expected stored profile to exist")
	}
	if stored.RetrievalMultiplier != 0.9 {
		t.Fatalf("expected multiplier 0.9, got %v", stored.RetrievalMultiplier)
	}
	if stored.SampleCount != 14 {
		t.Fatalf("expected sample count 14, got %d", stored.SampleCount)
	}
}
