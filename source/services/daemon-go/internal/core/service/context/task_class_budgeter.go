package context

import (
	"context"
	"fmt"
	"math"
	"strings"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type TaskClassBudgeter struct {
	base  *Budgeter
	store contract.ContextBudgetTelemetryStore
}

func NewTaskClassBudgeter(store contract.ContextBudgetTelemetryStore) *TaskClassBudgeter {
	return &TaskClassBudgeter{
		base:  NewBudgeter(),
		store: store,
	}
}

func (b *TaskClassBudgeter) Compute(
	ctx context.Context,
	workspaceID string,
	taskClass string,
	input types.ContextBudgetInput,
) (types.ContextBudget, error) {
	budget := b.base.Compute(input)
	if input.ContextWindow <= 0 {
		return budget, nil
	}

	multiplier := 1.0
	if b.store != nil && strings.TrimSpace(workspaceID) != "" && strings.TrimSpace(taskClass) != "" {
		profile, exists, err := b.store.GetContextBudgetTuningProfile(ctx, workspaceID, taskClass)
		if err != nil {
			return types.ContextBudget{}, fmt.Errorf("get context budget tuning profile: %w", err)
		}
		if exists && profile.RetrievalMultiplier > 0 {
			multiplier = profile.RetrievalMultiplier
		}
	}

	adjusted := int(math.Round(float64(budget.RetrievalTarget) * multiplier))
	if adjusted > budget.Remaining {
		adjusted = budget.Remaining
	}
	if adjusted < 0 {
		adjusted = 0
	}

	budget.RetrievalMultiplier = multiplier
	budget.RetrievalTarget = adjusted
	return budget, nil
}
