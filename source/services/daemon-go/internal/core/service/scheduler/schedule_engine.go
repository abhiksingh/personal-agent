package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type Engine struct {
	store   contract.ScheduleStore
	creator contract.ScheduledTaskCreator
	now     func() time.Time
}

type Options struct {
	Now func() time.Time
}

func NewEngine(store contract.ScheduleStore, creator contract.ScheduledTaskCreator, opts Options) *Engine {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Engine{store: store, creator: creator, now: nowFn}
}

func (e *Engine) EvaluateSchedules(ctx context.Context) (types.ScheduleEvaluationResult, error) {
	if e.store == nil {
		return types.ScheduleEvaluationResult{}, fmt.Errorf("schedule store is required")
	}
	if e.creator == nil {
		return types.ScheduleEvaluationResult{}, fmt.Errorf("scheduled task creator is required")
	}

	triggers, err := e.store.ListEnabledScheduleTriggers(ctx)
	if err != nil {
		return types.ScheduleEvaluationResult{}, err
	}
	dedupedTriggers, duplicateCount := dedupeScheduleTriggers(triggers)

	result := types.ScheduleEvaluationResult{
		Skipped: duplicateCount,
	}
	now := e.now().UTC()
	for _, trigger := range dedupedTriggers {
		result.Processed++
		config := parseScheduleConfig(trigger.FilterJSON)
		slot := now.Truncate(time.Duration(config.IntervalSeconds) * time.Second)
		sourceEventID := "schedule:" + slot.Format(time.RFC3339)

		reservation, created, err := e.store.TryReserveScheduleFire(ctx, trigger, sourceEventID, now)
		if err != nil {
			result.Failed++
			continue
		}
		if !created {
			result.Skipped++
			continue
		}

		taskID, err := e.creator.CreateTaskForScheduledDirective(ctx, trigger, sourceEventID, now)
		if err != nil {
			_ = e.store.MarkScheduleFireOutcome(ctx, reservation.FireID, "", "FAILED")
			result.Failed++
			continue
		}

		if err := e.store.MarkScheduleFireOutcome(ctx, reservation.FireID, taskID, "CREATED_TASK"); err != nil {
			result.Failed++
			continue
		}

		result.Created++
	}

	return result, nil
}

func parseScheduleConfig(raw string) types.ScheduleConfig {
	config := types.ScheduleConfig{IntervalSeconds: 300}
	if raw == "" {
		return config
	}
	_ = json.Unmarshal([]byte(raw), &config)
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = 300
	}
	return config
}

func dedupeScheduleTriggers(triggers []types.ScheduleTrigger) ([]types.ScheduleTrigger, int) {
	if len(triggers) <= 1 {
		return triggers, 0
	}

	unique := make([]types.ScheduleTrigger, 0, len(triggers))
	indexByKey := make(map[string]int, len(triggers))
	duplicateCount := 0
	for _, trigger := range triggers {
		key := scheduleTriggerDedupKey(trigger)
		if existingIndex, exists := indexByKey[key]; exists {
			duplicateCount++
			if strings.Compare(trigger.TriggerID, unique[existingIndex].TriggerID) < 0 {
				unique[existingIndex] = trigger
			}
			continue
		}
		indexByKey[key] = len(unique)
		unique = append(unique, trigger)
	}
	return unique, duplicateCount
}

func scheduleTriggerDedupKey(trigger types.ScheduleTrigger) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(trigger.WorkspaceID)),
		strings.ToLower(strings.TrimSpace(trigger.SubjectPrincipalActor)),
		normalizeScheduleTriggerJSON(trigger.FilterJSON),
		strings.ToLower(strings.TrimSpace(trigger.DirectiveTitle)),
		strings.ToLower(strings.TrimSpace(trigger.DirectiveInstruction)),
	}, "|")
}

func normalizeScheduleTriggerJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return trimmed
	}
	return string(normalized)
}
