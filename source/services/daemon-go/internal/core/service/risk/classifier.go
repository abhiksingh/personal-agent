package risk

import (
	"strings"

	"personalagent/runtime/internal/core/types"
)

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) ClassifyAction(actionSummary string) types.RiskClassification {
	action := strings.ToLower(strings.TrimSpace(actionSummary))
	if action == "" {
		return types.RiskClassification{
			Risk:       types.ActionRiskUnknown,
			Confidence: 0.0,
			Rationale:  "empty action summary",
		}
	}

	destructiveTerms := []string{
		"delete", "remove", "erase", "destroy", "cancel", "send", "submit",
		"transfer", "overwrite", "drop", "archive", "truncate",
	}
	for _, term := range destructiveTerms {
		if strings.Contains(action, term) {
			return types.RiskClassification{
				Risk:       types.ActionRiskDestructive,
				Confidence: 0.90,
				Rationale:  "matched destructive term: " + term,
			}
		}
	}

	reversibleTerms := []string{
		"list", "read", "fetch", "view", "summarize", "draft", "preview",
	}
	for _, term := range reversibleTerms {
		if strings.Contains(action, term) {
			return types.RiskClassification{
				Risk:       types.ActionRiskReversible,
				Confidence: 0.85,
				Rationale:  "matched reversible term: " + term,
			}
		}
	}

	return types.RiskClassification{
		Risk:       types.ActionRiskUnknown,
		Confidence: 0.45,
		Rationale:  "no known risk pattern matched",
	}
}
