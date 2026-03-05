package approval

import (
	"testing"

	"personalagent/runtime/internal/core/types"
)

func TestEvaluateRequiresApprovalForDestructiveActions(t *testing.T) {
	gate := NewGate(0.70)
	decision := gate.Evaluate(types.RiskClassification{Risk: types.ActionRiskDestructive, Confidence: 0.95})
	if !decision.RequireApproval {
		t.Fatalf("expected destructive action to require approval")
	}
}

func TestEvaluateAllowsReversibleHighConfidence(t *testing.T) {
	gate := NewGate(0.70)
	decision := gate.Evaluate(types.RiskClassification{Risk: types.ActionRiskReversible, Confidence: 0.90})
	if decision.RequireApproval {
		t.Fatalf("expected reversible high-confidence action to be allowed")
	}
}

func TestEvaluateRequiresApprovalForLowConfidence(t *testing.T) {
	gate := NewGate(0.70)
	decision := gate.Evaluate(types.RiskClassification{Risk: types.ActionRiskReversible, Confidence: 0.40})
	if !decision.RequireApproval {
		t.Fatalf("expected low-confidence classification to require approval")
	}
}

func TestEvaluateRequiresApprovalForUnknownRisk(t *testing.T) {
	gate := NewGate(0.70)
	decision := gate.Evaluate(types.RiskClassification{Risk: types.ActionRiskUnknown, Confidence: 0.99})
	if !decision.RequireApproval {
		t.Fatalf("expected unknown risk to require approval")
	}
}
