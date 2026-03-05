package daemonruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

func TestProviderModelChatServiceCheckProvidersUsesConfiguredHTTPTimeout(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)
	service.providerProbeClient = &http.Client{Timeout: 40 * time.Millisecond}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    server.URL,
	}); err != nil {
		t.Fatalf("set provider config: %v", err)
	}

	start := time.Now()
	response, err := service.CheckProviders(context.Background(), transport.ProviderCheckRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("check providers: %v", err)
	}
	if response.Success {
		t.Fatalf("expected timeout probe to mark response success=false, got %+v", response)
	}
	if len(response.Results) != 1 {
		t.Fatalf("expected one provider result, got %d", len(response.Results))
	}
	if response.Results[0].Success {
		t.Fatalf("expected timeout probe result success=false, got %+v", response.Results[0])
	}
	message := strings.ToLower(strings.TrimSpace(response.Results[0].Message))
	if !strings.Contains(message, "timeout") && !strings.Contains(message, "deadline") {
		t.Fatalf("expected timeout/deadline error message, got %q", response.Results[0].Message)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("expected configured timeout to fail before server delay, elapsed=%s", elapsed)
	}
}
