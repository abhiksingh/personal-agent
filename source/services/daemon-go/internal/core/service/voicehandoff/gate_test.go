package voicehandoff

import (
	"testing"

	"personalagent/runtime/internal/core/types"
)

func TestEvaluateBlocksVoiceDestructiveWithoutInAppApproval(t *testing.T) {
	gate := NewGate()
	decision := gate.Evaluate(types.VoiceHandoffInput{
		Origin:                 types.ExecutionOriginVoice,
		DestructiveAction:      true,
		InAppApprovalConfirmed: false,
	})
	if decision.AllowExecution {
		t.Fatalf("expected voice destructive action without approval to be blocked")
	}
	if decision.NextState != types.VoiceHandoffStateAwaitingApproval {
		t.Fatalf("expected awaiting_approval state, got %s", decision.NextState)
	}
}

func TestEvaluateAllowsVoiceDestructiveWithInAppApproval(t *testing.T) {
	gate := NewGate()
	decision := gate.Evaluate(types.VoiceHandoffInput{
		Origin:                 types.ExecutionOriginVoice,
		DestructiveAction:      true,
		InAppApprovalConfirmed: true,
	})
	if !decision.AllowExecution {
		t.Fatalf("expected approved voice destructive action to proceed")
	}
	if decision.NextState != types.VoiceHandoffStateRunning {
		t.Fatalf("expected running state, got %s", decision.NextState)
	}
}

func TestEvaluateAllowsNonVoiceDestructiveWithoutHandoff(t *testing.T) {
	gate := NewGate()
	decision := gate.Evaluate(types.VoiceHandoffInput{
		Origin:                 types.ExecutionOriginApp,
		DestructiveAction:      true,
		InAppApprovalConfirmed: false,
	})
	if !decision.AllowExecution {
		t.Fatalf("expected non-voice destructive action handoff gate to allow (approval handled elsewhere)")
	}
}
