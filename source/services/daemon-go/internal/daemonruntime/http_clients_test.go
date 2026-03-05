package daemonruntime

import (
	"testing"
	"time"
)

func TestDefaultProviderChatHTTPTimeoutAllowsLongRunningTurns(t *testing.T) {
	if defaultProviderChatHTTPTimeout < 5*time.Minute {
		t.Fatalf(
			"expected chat timeout to be at least 5m, got %s",
			defaultProviderChatHTTPTimeout,
		)
	}
}

func TestNewDaemonRuntimeHTTPClientUsesChatTimeoutDefaultForNonPositiveInput(t *testing.T) {
	client := newDaemonRuntimeHTTPClient(0)
	if client.Timeout != defaultProviderProbeHTTPTimeout {
		t.Fatalf(
			"expected fallback timeout %s, got %s",
			defaultProviderProbeHTTPTimeout,
			client.Timeout,
		)
	}
}
