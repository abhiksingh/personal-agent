package context

import "personalagent/runtime/internal/core/types"

type Budgeter struct{}

func NewBudgeter() *Budgeter {
	return &Budgeter{}
}

func (b *Budgeter) Compute(input types.ContextBudgetInput) types.ContextBudget {
	if input.ContextWindow <= 0 {
		return types.ContextBudget{}
	}

	outputReserve := maxInt(1024, percentOf(input.ContextWindow, 15))
	if input.OutputLimit > 0 && outputReserve > input.OutputLimit {
		outputReserve = input.OutputLimit
	}

	systemReserve := maxInt(1500, percentOf(input.ContextWindow, 10))
	safetyReserve := maxInt(512, percentOf(input.ContextWindow, 5))

	remaining := input.ContextWindow - outputReserve - systemReserve - safetyReserve
	if remaining < 0 {
		remaining = 0
	}

	retrievalTarget := minInt(24000, remaining)
	if input.DeepAnalysis {
		retrievalTarget = remaining
	}

	return types.ContextBudget{
		OutputReserve:       outputReserve,
		SystemReserve:       systemReserve,
		SafetyReserve:       safetyReserve,
		Remaining:           remaining,
		RetrievalTarget:     retrievalTarget,
		RetrievalMultiplier: 1.0,
	}
}

func percentOf(total, pct int) int {
	return (total * pct) / 100
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
