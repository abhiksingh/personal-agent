package types

type ActionRisk string

const (
	ActionRiskReversible  ActionRisk = "reversible"
	ActionRiskDestructive ActionRisk = "destructive"
	ActionRiskUnknown     ActionRisk = "unknown"
)

type RiskClassification struct {
	Risk       ActionRisk
	Confidence float64
	Rationale  string
}

type ApprovalGateDecision struct {
	RequireApproval bool
	Decision        string
	Reason          string
}
