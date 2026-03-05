package context

import (
	"context"
	"fmt"
	"testing"
	"time"

	"personalagent/runtime/internal/core/types"
)

type inMemoryTelemetryStore struct {
	samples  map[string][]types.ContextBudgetSample
	profiles map[string]types.ContextBudgetTuningProfile
}

func newInMemoryTelemetryStore() *inMemoryTelemetryStore {
	return &inMemoryTelemetryStore{
		samples:  map[string][]types.ContextBudgetSample{},
		profiles: map[string]types.ContextBudgetTuningProfile{},
	}
}

func (s *inMemoryTelemetryStore) RecordContextBudgetSample(_ context.Context, sample types.ContextBudgetSample) error {
	key := profileKey(sample.WorkspaceID, sample.TaskClass)
	s.samples[key] = append([]types.ContextBudgetSample{sample}, s.samples[key]...)
	return nil
}

func (s *inMemoryTelemetryStore) ListRecentContextBudgetSamples(_ context.Context, workspaceID string, taskClass string, limit int) ([]types.ContextBudgetSample, error) {
	key := profileKey(workspaceID, taskClass)
	samples := s.samples[key]
	if limit <= 0 || limit >= len(samples) {
		copyOf := make([]types.ContextBudgetSample, len(samples))
		copy(copyOf, samples)
		return copyOf, nil
	}
	copyOf := make([]types.ContextBudgetSample, limit)
	copy(copyOf, samples[:limit])
	return copyOf, nil
}

func (s *inMemoryTelemetryStore) GetContextBudgetTuningProfile(_ context.Context, workspaceID string, taskClass string) (types.ContextBudgetTuningProfile, bool, error) {
	profile, ok := s.profiles[profileKey(workspaceID, taskClass)]
	if !ok {
		return types.ContextBudgetTuningProfile{}, false, nil
	}
	return profile, true, nil
}

func (s *inMemoryTelemetryStore) UpsertContextBudgetTuningProfile(_ context.Context, profile types.ContextBudgetTuningProfile) error {
	s.profiles[profileKey(profile.WorkspaceID, profile.TaskClass)] = profile
	return nil
}

func profileKey(workspaceID string, taskClass string) string {
	return fmt.Sprintf("%s:%s", workspaceID, taskClass)
}

func TestTuneTaskClassIncreasesRetrievalMultiplierWhenUsageIsHigh(t *testing.T) {
	store := newInMemoryTelemetryStore()
	store.profiles["ws-1:chat"] = types.ContextBudgetTuningProfile{
		WorkspaceID:         "ws-1",
		TaskClass:           "chat",
		RetrievalMultiplier: 1.0,
	}

	for i := 0; i < 6; i++ {
		if err := store.RecordContextBudgetSample(context.Background(), types.ContextBudgetSample{
			WorkspaceID:      "ws-1",
			TaskClass:        "chat",
			ContextWindow:    128000,
			RetrievalTarget:  24000,
			RetrievalUsed:    22000,
			PromptTokens:     30000,
			CompletionTokens: 1200,
		}); err != nil {
			t.Fatalf("record sample: %v", err)
		}
	}

	loop := NewTuningLoop(store, TuningLoopOptions{Now: func() time.Time {
		return time.Date(2026, 2, 23, 22, 30, 0, 0, time.UTC)
	}})

	decision, err := loop.TuneTaskClass(context.Background(), "ws-1", "chat")
	if err != nil {
		t.Fatalf("tune task class: %v", err)
	}
	if !decision.Changed {
		t.Fatalf("expected multiplier to change")
	}
	if decision.NewMultiplier != 1.1 {
		t.Fatalf("expected multiplier 1.1, got %v", decision.NewMultiplier)
	}
	if decision.Reason != "increase_retrieval_target" {
		t.Fatalf("expected increase reason, got %s", decision.Reason)
	}
}

func TestTuneTaskClassDecreasesRetrievalMultiplierWhenUsageIsLow(t *testing.T) {
	store := newInMemoryTelemetryStore()
	store.profiles["ws-1:automation"] = types.ContextBudgetTuningProfile{
		WorkspaceID:         "ws-1",
		TaskClass:           "automation",
		RetrievalMultiplier: 1.0,
	}

	for i := 0; i < 6; i++ {
		if err := store.RecordContextBudgetSample(context.Background(), types.ContextBudgetSample{
			WorkspaceID:      "ws-1",
			TaskClass:        "automation",
			ContextWindow:    128000,
			RetrievalTarget:  24000,
			RetrievalUsed:    1000,
			PromptTokens:     12000,
			CompletionTokens: 300,
		}); err != nil {
			t.Fatalf("record sample: %v", err)
		}
	}

	loop := NewTuningLoop(store, TuningLoopOptions{})
	decision, err := loop.TuneTaskClass(context.Background(), "ws-1", "automation")
	if err != nil {
		t.Fatalf("tune task class: %v", err)
	}
	if !decision.Changed {
		t.Fatalf("expected multiplier to change")
	}
	if decision.NewMultiplier != 0.9 {
		t.Fatalf("expected multiplier 0.9, got %v", decision.NewMultiplier)
	}
	if decision.Reason != "decrease_retrieval_target" {
		t.Fatalf("expected decrease reason, got %s", decision.Reason)
	}
}

func TestTuneTaskClassRequiresMinimumSamples(t *testing.T) {
	store := newInMemoryTelemetryStore()
	_ = store.RecordContextBudgetSample(context.Background(), types.ContextBudgetSample{
		WorkspaceID:      "ws-1",
		TaskClass:        "chat",
		ContextWindow:    128000,
		RetrievalTarget:  24000,
		RetrievalUsed:    22000,
		PromptTokens:     30000,
		CompletionTokens: 1200,
	})

	loop := NewTuningLoop(store, TuningLoopOptions{})
	decision, err := loop.TuneTaskClass(context.Background(), "ws-1", "chat")
	if err != nil {
		t.Fatalf("tune task class: %v", err)
	}
	if decision.Changed {
		t.Fatalf("expected no change with insufficient samples")
	}
	if decision.Reason != "insufficient_samples" {
		t.Fatalf("expected insufficient_samples reason, got %s", decision.Reason)
	}
}
