package daemonruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestProbeConnectorExecutePathReturnsErrorOnServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"Browser connector execute failed: Allow JavaScript from Apple Events is disabled."}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"browser",
		testExecuteProbeWorker(server.URL),
	)
	if err == nil {
		t.Fatalf("expected execute probe error for 500 response")
	}
	if result.Ready {
		t.Fatalf("expected ready=false for 500 response")
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, result.StatusCode)
	}
	if !strings.Contains(strings.ToLower(result.Error), "allow javascript from apple events") {
		t.Fatalf("expected error detail from response body, got %q", result.Error)
	}
}

func TestProbeConnectorExecutePathTreatsUnsupportedProbeAsReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"unsupported connector execute probe operation \"__connector_execute_probe__\""}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"mail",
		testExecuteProbeWorker(server.URL),
	)
	if err != nil {
		t.Fatalf("expected unsupported-probe response to classify as reachable, got %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected ready=true for unsupported probe operation")
	}
	if result.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, result.StatusCode)
	}
}

func TestProbeConnectorExecutePathTreatsUnsupportedProbeAsReachableWhenStatus500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"unsupported calendar capability: __connector_execute_probe__"}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"calendar",
		testExecuteProbeWorker(server.URL),
	)
	if err != nil {
		t.Fatalf("expected unsupported probe response to classify as reachable, got %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected ready=true for unsupported probe operation on status 500")
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, result.StatusCode)
	}
}

func TestProbeConnectorExecutePathTreatsValidationInputErrorAsReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"browser step input is required"}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"browser",
		testExecuteProbeWorker(server.URL),
	)
	if err != nil {
		t.Fatalf("expected probe validation input response to classify as reachable, got %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected ready=true for probe validation input response")
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, result.StatusCode)
	}
}

func TestProbeConnectorExecutePathReturnsErrorOnNonProbeClientFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"browser native action target_url is required"}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"browser",
		testExecuteProbeWorker(server.URL),
	)
	if err == nil {
		t.Fatalf("expected execute probe error for non-probe 400 failure")
	}
	if result.Ready {
		t.Fatalf("expected ready=false for non-probe 400 failure")
	}
	if result.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, result.StatusCode)
	}
}

func TestProbeConnectorExecutePathReturnsErrorWhenSuccessBodyContainsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"error":"connector runtime execute failed"}`))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"calendar",
		testExecuteProbeWorker(server.URL),
	)
	if err == nil {
		t.Fatalf("expected execute probe error when success body includes error field")
	}
	if result.Ready {
		t.Fatalf("expected ready=false when success body includes error field")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, result.StatusCode)
	}
}

func TestProbeConnectorExecutePathRejectsOversizedResponseBody(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(daemonWorkerRPCResponseBodyLimitBytes+256))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(oversizedBody))
	}))
	t.Cleanup(server.Close)

	result, err := probeConnectorExecutePath(
		context.Background(),
		"ws1",
		"browser",
		testExecuteProbeWorker(server.URL),
	)
	if err == nil {
		t.Fatalf("expected oversized probe response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, result.StatusCode)
	}
}

func testExecuteProbeWorker(serverURL string) PluginWorkerStatus {
	return PluginWorkerStatus{
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: strings.TrimPrefix(strings.TrimSpace(serverURL), "http://"),
			},
		},
		execAuthToken: "worker-token",
	}
}
