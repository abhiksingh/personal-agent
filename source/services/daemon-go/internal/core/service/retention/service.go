package retention

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type Service struct {
	store contract.RetentionStore
	now   func() time.Time
}

func NewService(store contract.RetentionStore, nowFn func() time.Time) *Service {
	now := nowFn
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, now: now}
}

func (s *Service) Purge(ctx context.Context, policy *types.RetentionPolicy) (types.RetentionPurgeResult, error) {
	if s.store == nil {
		return types.RetentionPurgeResult{}, fmt.Errorf("retention store is required")
	}

	effective := types.DefaultRetentionPolicy()
	if policy != nil {
		effective.TraceDays = positiveOrDefault(policy.TraceDays, effective.TraceDays)
		effective.TranscriptDays = positiveOrDefault(policy.TranscriptDays, effective.TranscriptDays)
		effective.MemoryDays = positiveOrDefault(policy.MemoryDays, effective.MemoryDays)
	}

	now := s.now().UTC()
	cutoffs := types.RetentionCutoffs{
		TraceBefore:      now.Add(-time.Duration(effective.TraceDays) * 24 * time.Hour),
		TranscriptBefore: now.Add(-time.Duration(effective.TranscriptDays) * 24 * time.Hour),
		MemoryBefore:     now.Add(-time.Duration(effective.MemoryDays) * 24 * time.Hour),
	}
	result := types.RetentionPurgeResult{
		ConsistencyMode: types.RetentionPurgeConsistencyModePartialSuccess,
		Status:          types.RetentionPurgeStatusCompleted,
	}

	traceDeleted, err := s.store.PurgeTraceDataBefore(ctx, cutoffs.TraceBefore)
	result.TracesDeleted = traceDeleted
	if err != nil {
		return retentionPartialFailureResult(result, types.RetentionPurgeFailureStageTrace, err), nil
	}
	transcriptDeleted, err := s.store.PurgeTranscriptDataBefore(ctx, cutoffs.TranscriptBefore)
	result.TranscriptsDeleted = transcriptDeleted
	if err != nil {
		return retentionPartialFailureResult(result, types.RetentionPurgeFailureStageTranscript, err), nil
	}
	memoryDeleted, err := s.store.PurgeMemoryDataBefore(ctx, cutoffs.MemoryBefore)
	result.MemoryDeleted = memoryDeleted
	if err != nil {
		return retentionPartialFailureResult(result, types.RetentionPurgeFailureStageMemory, err), nil
	}

	return result, nil
}

func positiveOrDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func retentionPartialFailureResult(result types.RetentionPurgeResult, stage types.RetentionPurgeFailureStage, purgeErr error) types.RetentionPurgeResult {
	result.Status = types.RetentionPurgeStatusPartialFailure
	details := "retention purge statement failed"
	if trimmed := strings.TrimSpace(purgeErr.Error()); trimmed != "" {
		details = trimmed
	}
	result.Failure = &types.RetentionPurgeFailure{
		Stage:   stage,
		Code:    types.RetentionPurgeFailureCodeStatementFailed,
		Details: details,
	}
	return result
}
