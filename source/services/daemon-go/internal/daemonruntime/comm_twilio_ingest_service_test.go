package daemonruntime

import (
	"context"
	"testing"

	"personalagent/runtime/internal/core/types"
)

type recordingAutomationEvaluator struct {
	eventIDs []string
}

func (r *recordingAutomationEvaluator) EvaluateCommEvent(_ context.Context, eventID string) (types.CommTriggerEvaluationResult, error) {
	r.eventIDs = append(r.eventIDs, eventID)
	return types.CommTriggerEvaluationResult{}, nil
}

func TestEvaluateAutomationForCommEventsRunsOnlyForAcceptedNonReplayEvents(t *testing.T) {
	evaluator := &recordingAutomationEvaluator{}
	service := &CommTwilioService{
		automationEval: evaluator,
	}

	service.evaluateAutomationForCommEvents(context.Background(), true, false, "event-1", "", "event-2")
	if len(evaluator.eventIDs) != 2 {
		t.Fatalf("expected two evaluated event ids, got %v", evaluator.eventIDs)
	}
	if evaluator.eventIDs[0] != "event-1" || evaluator.eventIDs[1] != "event-2" {
		t.Fatalf("unexpected evaluated event ids: %v", evaluator.eventIDs)
	}

	service.evaluateAutomationForCommEvents(context.Background(), false, false, "event-3")
	service.evaluateAutomationForCommEvents(context.Background(), true, true, "event-4")
	if len(evaluator.eventIDs) != 2 {
		t.Fatalf("expected accepted/replayed gating to block extra evaluations, got %v", evaluator.eventIDs)
	}
}
