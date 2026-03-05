package contract

import (
	"context"
	"time"

	"personalagent/runtime/internal/core/types"
)

type CommEventTriggerStore interface {
	LoadCommEvent(ctx context.Context, eventID string) (types.CommEventRecord, error)
	ListEnabledOnCommEventTriggers(ctx context.Context, workspaceID string) ([]types.OnCommEventTrigger, error)
	TryReserveTriggerFire(
		ctx context.Context,
		triggerID string,
		workspaceID string,
		sourceEventID string,
		firedAt time.Time,
	) (fireID string, created bool, err error)
	MarkTriggerFireOutcome(ctx context.Context, fireID string, taskID string, outcome string) error
	CreateTaskForDirective(
		ctx context.Context,
		trigger types.OnCommEventTrigger,
		sourceEventID string,
		now time.Time,
	) (taskID string, err error)
}
