package context

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type TuningLoop struct {
	store                 contract.ContextBudgetTelemetryStore
	now                   func() time.Time
	lookbackSamples       int
	minimumSamples        int
	adjustmentStep        float64
	minimumMultiplier     float64
	maximumMultiplier     float64
	increaseThreshold     float64
	decreaseThreshold     float64
	promptPressureCeiling float64
}

type TuningLoopOptions struct {
	Now                   func() time.Time
	LookbackSamples       int
	MinimumSamples        int
	AdjustmentStep        float64
	MinimumMultiplier     float64
	MaximumMultiplier     float64
	IncreaseThreshold     float64
	DecreaseThreshold     float64
	PromptPressureCeiling float64
}

func NewTuningLoop(store contract.ContextBudgetTelemetryStore, opts TuningLoopOptions) *TuningLoop {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}

	lookback := opts.LookbackSamples
	if lookback <= 0 {
		lookback = 20
	}
	minSamples := opts.MinimumSamples
	if minSamples <= 0 {
		minSamples = 5
	}
	step := opts.AdjustmentStep
	if step <= 0 {
		step = 0.1
	}
	minMultiplier := opts.MinimumMultiplier
	if minMultiplier <= 0 {
		minMultiplier = 0.5
	}
	maxMultiplier := opts.MaximumMultiplier
	if maxMultiplier <= 0 || maxMultiplier < minMultiplier {
		maxMultiplier = 1.5
	}
	increaseThreshold := opts.IncreaseThreshold
	if increaseThreshold <= 0 {
		increaseThreshold = 0.85
	}
	decreaseThreshold := opts.DecreaseThreshold
	if decreaseThreshold <= 0 {
		decreaseThreshold = 0.35
	}
	promptPressureCeiling := opts.PromptPressureCeiling
	if promptPressureCeiling <= 0 {
		promptPressureCeiling = 0.9
	}

	return &TuningLoop{
		store:                 store,
		now:                   nowFn,
		lookbackSamples:       lookback,
		minimumSamples:        minSamples,
		adjustmentStep:        step,
		minimumMultiplier:     minMultiplier,
		maximumMultiplier:     maxMultiplier,
		increaseThreshold:     increaseThreshold,
		decreaseThreshold:     decreaseThreshold,
		promptPressureCeiling: promptPressureCeiling,
	}
}

func (t *TuningLoop) RecordSample(ctx context.Context, sample types.ContextBudgetSample) error {
	if t.store == nil {
		return fmt.Errorf("context budget telemetry store is required")
	}
	return t.store.RecordContextBudgetSample(ctx, sample)
}

func (t *TuningLoop) TuneTaskClass(
	ctx context.Context,
	workspaceID string,
	taskClass string,
) (types.ContextBudgetTuningDecision, error) {
	if t.store == nil {
		return types.ContextBudgetTuningDecision{}, fmt.Errorf("context budget telemetry store is required")
	}
	if strings.TrimSpace(workspaceID) == "" {
		return types.ContextBudgetTuningDecision{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(taskClass) == "" {
		return types.ContextBudgetTuningDecision{}, fmt.Errorf("task class is required")
	}

	now := t.now().UTC()
	profile, exists, err := t.store.GetContextBudgetTuningProfile(ctx, workspaceID, taskClass)
	if err != nil {
		return types.ContextBudgetTuningDecision{}, err
	}

	currentMultiplier := 1.0
	if exists && profile.RetrievalMultiplier > 0 {
		currentMultiplier = profile.RetrievalMultiplier
	}

	samples, err := t.store.ListRecentContextBudgetSamples(ctx, workspaceID, taskClass, t.lookbackSamples)
	if err != nil {
		return types.ContextBudgetTuningDecision{}, err
	}
	if len(samples) < t.minimumSamples {
		return types.ContextBudgetTuningDecision{
			WorkspaceID:        workspaceID,
			TaskClass:          taskClass,
			PreviousMultiplier: currentMultiplier,
			NewMultiplier:      currentMultiplier,
			Changed:            false,
			Reason:             "insufficient_samples",
			SampleCount:        len(samples),
			EvaluatedAt:        now,
		}, nil
	}

	avgRetrievalUtilization, avgPromptUtilization := summarizeSamples(samples)
	nextMultiplier := currentMultiplier
	reason := "stable"

	if avgRetrievalUtilization >= t.increaseThreshold && avgPromptUtilization < t.promptPressureCeiling {
		nextMultiplier = clampFloat(currentMultiplier+t.adjustmentStep, t.minimumMultiplier, t.maximumMultiplier)
		reason = "increase_retrieval_target"
	} else if avgRetrievalUtilization <= t.decreaseThreshold || avgPromptUtilization >= t.promptPressureCeiling {
		nextMultiplier = clampFloat(currentMultiplier-t.adjustmentStep, t.minimumMultiplier, t.maximumMultiplier)
		reason = "decrease_retrieval_target"
	}

	decision := types.ContextBudgetTuningDecision{
		WorkspaceID:             workspaceID,
		TaskClass:               taskClass,
		PreviousMultiplier:      currentMultiplier,
		NewMultiplier:           nextMultiplier,
		Changed:                 math.Abs(nextMultiplier-currentMultiplier) > 0.00001,
		Reason:                  reason,
		SampleCount:             len(samples),
		AvgRetrievalUtilization: avgRetrievalUtilization,
		AvgPromptUtilization:    avgPromptUtilization,
		EvaluatedAt:             now,
	}

	if err := t.store.UpsertContextBudgetTuningProfile(ctx, types.ContextBudgetTuningProfile{
		WorkspaceID:             workspaceID,
		TaskClass:               taskClass,
		RetrievalMultiplier:     nextMultiplier,
		SampleCount:             len(samples),
		AvgRetrievalUtilization: avgRetrievalUtilization,
		AvgPromptUtilization:    avgPromptUtilization,
		UpdatedAt:               now,
	}); err != nil {
		return types.ContextBudgetTuningDecision{}, err
	}

	return decision, nil
}

func summarizeSamples(samples []types.ContextBudgetSample) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}

	totalRetrievalUtilization := 0.0
	totalPromptUtilization := 0.0
	for _, sample := range samples {
		retrievalBase := maxInt(sample.RetrievalTarget, 1)
		promptBase := maxInt(sample.ContextWindow, 1)

		totalRetrievalUtilization += float64(sample.RetrievalUsed) / float64(retrievalBase)
		totalPromptUtilization += float64(sample.TotalTokens()) / float64(promptBase)
	}

	count := float64(len(samples))
	return totalRetrievalUtilization / count, totalPromptUtilization / count
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
