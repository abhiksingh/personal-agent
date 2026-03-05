package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	repocommtrigger "personalagent/runtime/internal/core/repository/commtrigger"
	reposcheduler "personalagent/runtime/internal/core/repository/scheduler"
	commtriggerservice "personalagent/runtime/internal/core/service/commtrigger"
	schedulerservice "personalagent/runtime/internal/core/service/scheduler"
	"personalagent/runtime/internal/core/types"
)

const defaultAutomationSchedulePollInterval = 5 * time.Second

type AutomationRuntimeOptions struct {
	SchedulePollInterval time.Duration
	Now                  func() time.Time
}

type AutomationRuntime struct {
	db                   *sql.DB
	schedulePollInterval time.Duration
	now                  func() time.Time
	scheduleStore        *reposcheduler.SQLiteScheduleStore
	scheduleEngine       *schedulerservice.Engine
	commTriggerStore     *repocommtrigger.SQLiteCommTriggerStore
	commTriggerEngine    *commtriggerservice.Engine

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewAutomationRuntime(db *sql.DB, opts AutomationRuntimeOptions) (*AutomationRuntime, error) {
	if db == nil {
		return nil, fmt.Errorf("automation runtime db is required")
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	interval := opts.SchedulePollInterval
	if interval <= 0 {
		interval = defaultAutomationSchedulePollInterval
	}
	scheduleStore := reposcheduler.NewSQLiteScheduleStore(db)
	commTriggerStore := repocommtrigger.NewSQLiteCommTriggerStore(db)
	return &AutomationRuntime{
		db:                   db,
		schedulePollInterval: interval,
		now:                  nowFn,
		scheduleStore:        scheduleStore,
		scheduleEngine: schedulerservice.NewEngine(scheduleStore, scheduleStore, schedulerservice.Options{
			Now: nowFn,
		}),
		commTriggerStore: commTriggerStore,
		commTriggerEngine: commtriggerservice.NewEngine(commTriggerStore, commtriggerservice.Options{
			Now: nowFn,
		}),
	}, nil
}

func (r *AutomationRuntime) Start(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return nil
	}

	runCtx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.running = true
	go r.runScheduleLoop(runCtx, done)
	return nil
}

func (r *AutomationRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	r.running = false
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *AutomationRuntime) EvaluateSchedule(ctx context.Context) (types.ScheduleEvaluationResult, error) {
	if r.scheduleEngine == nil {
		return types.ScheduleEvaluationResult{}, fmt.Errorf("automation runtime schedule engine is required")
	}
	return r.scheduleEngine.EvaluateSchedules(ctx)
}

func (r *AutomationRuntime) EvaluateCommEvent(ctx context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	if r.commTriggerEngine == nil {
		return types.CommTriggerEvaluationResult{}, fmt.Errorf("automation runtime comm trigger engine is required")
	}
	trimmedEventID := strings.TrimSpace(eventID)
	if trimmedEventID == "" {
		return types.CommTriggerEvaluationResult{}, fmt.Errorf("event_id is required")
	}
	return r.commTriggerEngine.EvaluateEvent(ctx, trimmedEventID)
}

func (r *AutomationRuntime) runScheduleLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(r.schedulePollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		_, _ = r.EvaluateSchedule(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
