package context

import (
	"testing"

	"personalagent/runtime/internal/core/types"
)

func TestComputeMatchesSpecBudgetFormula(t *testing.T) {
	budgeter := NewBudgeter()
	budget := budgeter.Compute(types.ContextBudgetInput{
		ContextWindow: 128000,
		OutputLimit:   4096,
		DeepAnalysis:  false,
	})

	if budget.OutputReserve != 4096 {
		t.Fatalf("expected output reserve 4096, got %d", budget.OutputReserve)
	}
	if budget.SystemReserve != 12800 {
		t.Fatalf("expected system reserve 12800, got %d", budget.SystemReserve)
	}
	if budget.SafetyReserve != 6400 {
		t.Fatalf("expected safety reserve 6400, got %d", budget.SafetyReserve)
	}
	if budget.Remaining != 104704 {
		t.Fatalf("expected remaining 104704, got %d", budget.Remaining)
	}
	if budget.RetrievalTarget != 24000 {
		t.Fatalf("expected retrieval target 24000, got %d", budget.RetrievalTarget)
	}
}

func TestComputeDeepAnalysisUsesFullRemainingBudget(t *testing.T) {
	budgeter := NewBudgeter()
	budget := budgeter.Compute(types.ContextBudgetInput{
		ContextWindow: 32000,
		OutputLimit:   4096,
		DeepAnalysis:  true,
	})

	if budget.RetrievalTarget != budget.Remaining {
		t.Fatalf("expected deep analysis retrieval target %d, got %d", budget.Remaining, budget.RetrievalTarget)
	}
}
