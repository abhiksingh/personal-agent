package risk

import (
	"testing"

	"personalagent/runtime/internal/core/types"
)

func TestClassifyActionDestructive(t *testing.T) {
	classifier := NewClassifier()
	result := classifier.ClassifyAction("Delete old files")
	if result.Risk != types.ActionRiskDestructive {
		t.Fatalf("expected destructive risk, got %s", result.Risk)
	}
}

func TestClassifyActionReversible(t *testing.T) {
	classifier := NewClassifier()
	result := classifier.ClassifyAction("List unread emails")
	if result.Risk != types.ActionRiskReversible {
		t.Fatalf("expected reversible risk, got %s", result.Risk)
	}
}

func TestClassifyActionUnknown(t *testing.T) {
	classifier := NewClassifier()
	result := classifier.ClassifyAction("maybe check something")
	if result.Risk != types.ActionRiskUnknown {
		t.Fatalf("expected unknown risk, got %s", result.Risk)
	}
}
