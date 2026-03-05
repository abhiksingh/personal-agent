package contract

import (
	"context"

	"personalagent/runtime/internal/core/types"
)

type ContextBudgetTelemetryStore interface {
	RecordContextBudgetSample(ctx context.Context, sample types.ContextBudgetSample) error
	ListRecentContextBudgetSamples(ctx context.Context, workspaceID string, taskClass string, limit int) ([]types.ContextBudgetSample, error)
	GetContextBudgetTuningProfile(ctx context.Context, workspaceID string, taskClass string) (types.ContextBudgetTuningProfile, bool, error)
	UpsertContextBudgetTuningProfile(ctx context.Context, profile types.ContextBudgetTuningProfile) error
}
