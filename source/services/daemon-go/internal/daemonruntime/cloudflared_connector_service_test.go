package daemonruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

type cloudflaredServiceSupervisorStub struct {
	status PluginWorkerStatus
}

func (s *cloudflaredServiceSupervisorStub) SetHooks(_ PluginLifecycleHooks) {}

func (s *cloudflaredServiceSupervisorStub) RegisterWorker(_ PluginWorkerSpec) error { return nil }

func (s *cloudflaredServiceSupervisorStub) ListWorkers() []PluginWorkerStatus {
	return []PluginWorkerStatus{s.status}
}

func (s *cloudflaredServiceSupervisorStub) WorkerStatus(pluginID string) (PluginWorkerStatus, bool) {
	if strings.TrimSpace(pluginID) != CloudflaredConnectorPluginID {
		return PluginWorkerStatus{}, false
	}
	return s.status, true
}

func (s *cloudflaredServiceSupervisorStub) RestartWorker(_ context.Context, _ string) error {
	return nil
}

func (s *cloudflaredServiceSupervisorStub) StopWorker(_ context.Context, _ string) error {
	return nil
}

func (s *cloudflaredServiceSupervisorStub) Start(_ context.Context) error { return nil }

func (s *cloudflaredServiceSupervisorStub) Stop(_ context.Context) error { return nil }

func TestCloudflaredConnectorServiceRejectsOversizedWorkerResponse(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(daemonWorkerRPCResponseBodyLimitBytes+256))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(oversizedBody))
	}))
	defer server.Close()

	status := PluginWorkerStatus{
		PluginID: CloudflaredConnectorPluginID,
		Kind:     shared.AdapterKindConnector,
		State:    PluginWorkerStateRunning,
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
			Capabilities: []shared.CapabilityDescriptor{
				{Key: CloudflaredConnectorCapabilityVersion},
			},
		},
		execAuthToken: "worker-token",
	}
	supervisor := &cloudflaredServiceSupervisorStub{status: status}
	service := &CloudflaredConnectorService{
		container:       &ServiceContainer{PluginSupervisor: supervisor},
		httpClient:      server.Client(),
		restartBackoff:  time.Millisecond,
		restartDeadline: 2 * time.Second,
	}

	_, err := service.CloudflaredVersion(context.Background(), transport.CloudflaredVersionRequest{WorkspaceID: "ws1"})
	if err == nil {
		t.Fatalf("expected oversized worker response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}
