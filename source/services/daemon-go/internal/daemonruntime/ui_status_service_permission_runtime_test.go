package daemonruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestProbeCloudflaredConnectorVersionRejectsOversizedResponseBody(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(daemonWorkerRPCResponseBodyLimitBytes+256))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(oversizedBody))
	}))
	defer server.Close()

	worker := PluginWorkerStatus{
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
		},
		execAuthToken: "worker-token",
	}

	_, err := probeCloudflaredConnectorVersion(context.Background(), "ws1", worker)
	if err == nil {
		t.Fatalf("expected oversized cloudflared version probe response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}
