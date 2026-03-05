package transport

import "testing"

func TestNormalizeTaskLifecycleStateDoesNotAliasCanceled(t *testing.T) {
	if got := normalizeTaskLifecycleState(" canceled "); got != "canceled" {
		t.Fatalf("expected canceled to remain canceled, got %q", got)
	}
	if isTerminalTaskLifecycleState("canceled") {
		t.Fatalf("expected canceled not to be treated as canonical terminal state")
	}
	if !isTerminalTaskLifecycleState("cancelled") {
		t.Fatalf("expected cancelled to remain canonical terminal state")
	}
}

func TestNormalizeTaskRunActionStateDoesNotAliasCanceled(t *testing.T) {
	if got := normalizeTaskRunActionState(" canceled "); got != "canceled" {
		t.Fatalf("expected canceled to remain canceled for action-state normalization, got %q", got)
	}
}

func TestResolveTaskRunActionAvailabilityDoesNotTreatCanceledAsCancelled(t *testing.T) {
	availability := ResolveTaskRunActionAvailability("", "canceled")
	if availability.CanCancel || availability.CanRetry || availability.CanRequeue {
		t.Fatalf("expected no canonical actions for non-canonical canceled state, got %+v", availability)
	}

	canonical := ResolveTaskRunActionAvailability("", "cancelled")
	if !canonical.CanRetry || canonical.CanCancel || canonical.CanRequeue {
		t.Fatalf("expected canonical cancelled actions, got %+v", canonical)
	}
}
