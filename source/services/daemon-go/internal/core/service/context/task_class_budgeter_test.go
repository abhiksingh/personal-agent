package context

import (
	"context"
	"testing"

	"personalagent/runtime/internal/core/types"
)

func TestTaskClassBudgeterAppliesTaskClassMultiplier(t *testing.T) {
	store := newInMemoryTelemetryStore()
	store.profiles["ws-1:chat"] = types.ContextBudgetTuningProfile{
		WorkspaceID:         "ws-1",
		TaskClass:           "chat",
		RetrievalMultiplier: 0.5,
	}

	budgeter := NewTaskClassBudgeter(store)
	budget, err := budgeter.Compute(context.Background(), "ws-1", "chat", types.ContextBudgetInput{
		ContextWindow: 32000,
		OutputLimit:   4096,
	})
	if err != nil {
		t.Fatalf("compute budget: %v", err)
	}

	if budget.RetrievalMultiplier != 0.5 {
		t.Fatalf("expected retrieval multiplier 0.5, got %v", budget.RetrievalMultiplier)
	}
	if budget.RetrievalTarget != 11552 {
		t.Fatalf("expected tuned retrieval target 11552, got %d", budget.RetrievalTarget)
	}
}

func TestTaskClassBudgeterDefaultsToBaseWhenNoProfileExists(t *testing.T) {
	budgeter := NewTaskClassBudgeter(newInMemoryTelemetryStore())
	budget, err := budgeter.Compute(context.Background(), "ws-1", "chat", types.ContextBudgetInput{
		ContextWindow: 32000,
		OutputLimit:   4096,
	})
	if err != nil {
		t.Fatalf("compute budget: %v", err)
	}

	if budget.RetrievalMultiplier != 1.0 {
		t.Fatalf("expected retrieval multiplier 1.0, got %v", budget.RetrievalMultiplier)
	}
	if budget.RetrievalTarget != 23104 {
		t.Fatalf("expected base retrieval target 23104, got %d", budget.RetrievalTarget)
	}
}
