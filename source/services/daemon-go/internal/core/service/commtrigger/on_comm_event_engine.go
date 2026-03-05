package commtrigger

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
	store contract.CommEventTriggerStore
	now   func() time.Time
}

type Options struct {
	Now func() time.Time
}

func NewEngine(store contract.CommEventTriggerStore, opts Options) *Engine {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Engine{store: store, now: nowFn}
}

func (e *Engine) EvaluateEvent(ctx context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	if e.store == nil {
		return types.CommTriggerEvaluationResult{}, fmt.Errorf("comm event trigger store is required")
	}
	if strings.TrimSpace(eventID) == "" {
		return types.CommTriggerEvaluationResult{}, fmt.Errorf("event_id is required")
	}

	event, err := e.store.LoadCommEvent(ctx, eventID)
	if err != nil {
		return types.CommTriggerEvaluationResult{}, err
	}

	result := types.CommTriggerEvaluationResult{}
	if !matchesRequiredDefaults(event) {
		result.Skipped = 1
		return result, nil
	}

	triggers, err := e.store.ListEnabledOnCommEventTriggers(ctx, event.WorkspaceID)
	if err != nil {
		return types.CommTriggerEvaluationResult{}, err
	}
	dedupedTriggers, duplicateCount := dedupeOnCommEventTriggers(triggers)

	result.Skipped += duplicateCount
	now := e.now().UTC()
	for _, trigger := range dedupedTriggers {
		result.Processed++
		filter := parseTriggerFilter(trigger.FilterJSON)
		if !matchesOptionalFilters(filter, trigger, event) {
			result.Skipped++
			continue
		}
		result.Matched++

		fireID, created, err := e.store.TryReserveTriggerFire(ctx, trigger.TriggerID, trigger.WorkspaceID, event.EventID, now)
		if err != nil {
			result.Failed++
			continue
		}
		if !created {
			result.Skipped++
			continue
		}

		taskID, err := e.store.CreateTaskForDirective(ctx, trigger, event.EventID, now)
		if err != nil {
			_ = e.store.MarkTriggerFireOutcome(ctx, fireID, "", "FAILED")
			result.Failed++
			continue
		}

		if err := e.store.MarkTriggerFireOutcome(ctx, fireID, taskID, "CREATED_TASK"); err != nil {
			result.Failed++
			continue
		}

		result.Created++
	}

	return result, nil
}

func matchesRequiredDefaults(event types.CommEventRecord) bool {
	if strings.ToUpper(strings.TrimSpace(event.EventType)) != "MESSAGE" {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(event.Direction)) != "INBOUND" {
		return false
	}
	if event.AssistantEmitted {
		return false
	}
	return true
}

func parseTriggerFilter(raw string) types.CommEventTriggerFilter {
	filter := types.CommEventTriggerFilter{}
	if strings.TrimSpace(raw) == "" {
		return filter
	}
	_ = json.Unmarshal([]byte(raw), &filter)
	return filter
}

func matchesOptionalFilters(filter types.CommEventTriggerFilter, trigger types.OnCommEventTrigger, event types.CommEventRecord) bool {
	if len(filter.Channels) > 0 && !containsCaseInsensitive(filter.Channels, event.Channel) {
		return false
	}
	if len(filter.PrincipalActorIDs) > 0 && !containsCaseInsensitive(filter.PrincipalActorIDs, trigger.SubjectPrincipalActor) {
		return false
	}
	if len(filter.SenderAllowlist) > 0 && !containsCaseInsensitive(filter.SenderAllowlist, event.SenderAddress) {
		return false
	}
	if len(filter.ThreadIDs) > 0 && !containsCaseInsensitive(filter.ThreadIDs, event.ThreadID) {
		return false
	}

	body := strings.ToLower(event.BodyText)
	for _, phrase := range filter.Keywords.ExactPhrases {
		if strings.Contains(body, strings.ToLower(phrase)) {
			return true
		}
	}

	if len(filter.Keywords.ContainsAll) > 0 {
		for _, term := range filter.Keywords.ContainsAll {
			if !strings.Contains(body, strings.ToLower(term)) {
				return false
			}
		}
	}

	if len(filter.Keywords.ContainsAny) > 0 {
		matchedAny := false
		for _, term := range filter.Keywords.ContainsAny {
			if strings.Contains(body, strings.ToLower(term)) {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return false
		}
	}

	if len(filter.Keywords.ExactPhrases) > 0 {
		matchedPhrase := false
		for _, phrase := range filter.Keywords.ExactPhrases {
			if strings.Contains(body, strings.ToLower(phrase)) {
				matchedPhrase = true
				break
			}
		}
		if !matchedPhrase {
			return false
		}
	}

	return true
}

func containsCaseInsensitive(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func dedupeOnCommEventTriggers(triggers []types.OnCommEventTrigger) ([]types.OnCommEventTrigger, int) {
	if len(triggers) <= 1 {
		return triggers, 0
	}

	unique := make([]types.OnCommEventTrigger, 0, len(triggers))
	indexByKey := make(map[string]int, len(triggers))
	duplicateCount := 0
	for _, trigger := range triggers {
		key := onCommEventTriggerDedupKey(trigger)
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

func onCommEventTriggerDedupKey(trigger types.OnCommEventTrigger) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(trigger.WorkspaceID)),
		strings.ToLower(strings.TrimSpace(trigger.SubjectPrincipalActor)),
		normalizeOnCommEventTriggerJSON(trigger.FilterJSON),
		strings.ToLower(strings.TrimSpace(trigger.DirectiveTitle)),
		strings.ToLower(strings.TrimSpace(trigger.DirectiveInstruction)),
	}, "|")
}

func normalizeOnCommEventTriggerJSON(raw string) string {
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
