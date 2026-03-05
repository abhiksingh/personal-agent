package contract

import "personalagent/runtime/internal/core/types"

type RiskClassifier interface {
	ClassifyAction(actionSummary string) types.RiskClassification
}
