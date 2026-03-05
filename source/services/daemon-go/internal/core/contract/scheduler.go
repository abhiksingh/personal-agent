package contract

import (
	"context"
	"time"

	"personalagent/runtime/internal/core/types"
)

type ScheduleStore interface {
	ListEnabledScheduleTriggers(ctx context.Context) ([]types.ScheduleTrigger, error)
	TryReserveScheduleFire(
		ctx context.Context,
		trigger types.ScheduleTrigger,
		sourceEventID string,
		firedAt time.Time,
	) (types.TriggerFireReservation, bool, error)
	MarkScheduleFireOutcome(ctx context.Context, fireID string, taskID string, outcome string) error
}

type ScheduledTaskCreator interface {
	CreateTaskForScheduledDirective(
		ctx context.Context,
		trigger types.ScheduleTrigger,
		sourceEventID string,
		now time.Time,
	) (string, error)
}
