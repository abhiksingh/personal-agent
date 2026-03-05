package approval

import "personalagent/runtime/internal/core/types"

type Gate struct {
	LowConfidenceThreshold float64
}

func NewGate(lowConfidenceThreshold float64) *Gate {
	threshold := lowConfidenceThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.70
	}
	return &Gate{LowConfidenceThreshold: threshold}
}

func (g *Gate) Evaluate(classification types.RiskClassification) types.ApprovalGateDecision {
	if classification.Risk == types.ActionRiskDestructive {
		return types.ApprovalGateDecision{
			RequireApproval: true,
			Decision:        "REQUIRE_CONFIRM",
			Reason:          "destructive action requires approval",
		}
	}

	if classification.Risk == types.ActionRiskUnknown {
		return types.ApprovalGateDecision{
			RequireApproval: true,
			Decision:        "REQUIRE_CONFIRM",
			Reason:          "unknown risk defaults to approval",
		}
	}

	if classification.Confidence < g.LowConfidenceThreshold {
		return types.ApprovalGateDecision{
			RequireApproval: true,
			Decision:        "REQUIRE_CONFIRM",
			Reason:          "low-confidence risk classification defaults to approval",
		}
	}

	return types.ApprovalGateDecision{
		RequireApproval: false,
		Decision:        "ALLOW",
		Reason:          "reversible action with sufficient confidence",
	}
}
