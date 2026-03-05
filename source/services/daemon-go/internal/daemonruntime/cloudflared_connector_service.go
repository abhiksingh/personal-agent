package daemonruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

type CloudflaredConnectorService struct {
	container       *ServiceContainer
	httpClient      *http.Client
	restartBackoff  time.Duration
	restartDeadline time.Duration
}

var _ transport.CloudflaredConnectorService = (*CloudflaredConnectorService)(nil)

func NewCloudflaredConnectorService(container *ServiceContainer) (*CloudflaredConnectorService, error) {
	if container == nil || container.PluginSupervisor == nil {
		return nil, fmt.Errorf("service container with plugin supervisor is required")
	}
	return &CloudflaredConnectorService{
		container:       container,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
		restartBackoff:  50 * time.Millisecond,
		restartDeadline: 5 * time.Second,
	}, nil
}

func (s *CloudflaredConnectorService) CloudflaredVersion(ctx context.Context, request transport.CloudflaredVersionRequest) (transport.CloudflaredVersionResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	var response transport.CloudflaredVersionResponse
	if err := s.executeCloudflaredOperation(
		ctx,
		CloudflaredConnectorCapabilityVersion,
		CloudflaredConnectorOperationVersion,
		map[string]any{"workspace_id": workspace},
		&response,
	); err != nil {
		return transport.CloudflaredVersionResponse{}, err
	}
	response.WorkspaceID = workspace
	return response, nil
}

func (s *CloudflaredConnectorService) CloudflaredExec(ctx context.Context, request transport.CloudflaredExecRequest) (transport.CloudflaredExecResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	var response transport.CloudflaredExecResponse
	if err := s.executeCloudflaredOperation(
		ctx,
		CloudflaredConnectorCapabilityExec,
		CloudflaredConnectorOperationExec,
		map[string]any{
			"workspace_id": workspace,
			"args":         request.Args,
			"timeout_ms":   request.TimeoutMS,
		},
		&response,
	); err != nil {
		return transport.CloudflaredExecResponse{}, err
	}
	response.WorkspaceID = workspace
	return response, nil
}

func (s *CloudflaredConnectorService) executeCloudflaredOperation(ctx context.Context, capability string, operation string, payload any, output any) error {
	status, err := s.resolveCloudflaredWorker(ctx, capability)
	if err != nil {
		return err
	}
	address := cloudflaredWorkerExecAddress(status.Metadata)
	if address == "" {
		return fmt.Errorf("cloudflared connector worker has no execution endpoint")
	}
	authToken := strings.TrimSpace(status.execAuthToken)
	if authToken == "" {
		return fmt.Errorf("cloudflared connector worker has no daemon-issued auth token")
	}

	requestBody, err := json.Marshal(cloudflaredWorkerRPCRequest{
		Operation: strings.TrimSpace(operation),
		Payload:   payload,
	})
	if err != nil {
		return fmt.Errorf("marshal cloudflared worker request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/execute", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("build cloudflared worker request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+authToken)

	httpResponse, err := s.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	body, truncated, err := readBoundedHTTPResponseBody(httpResponse.Body, daemonWorkerRPCResponseBodyLimitBytes)
	if err != nil {
		return fmt.Errorf("read cloudflared worker response: %w", err)
	}
	if truncated {
		return fmt.Errorf(
			"cloudflared worker response exceeded max size of %d bytes",
			daemonWorkerRPCResponseBodyLimitBytes,
		)
	}
	if httpResponse.StatusCode >= 400 {
		message := strings.TrimSpace(extractWorkerErrorMessage(body))
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		if message == "" {
			message = httpResponse.Status
		}
		return fmt.Errorf("cloudflared worker execute failed: %s", message)
	}

	var response cloudflaredWorkerRPCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode cloudflared worker response: %w", err)
	}
	if strings.TrimSpace(response.Error) != "" {
		return fmt.Errorf("%s", strings.TrimSpace(response.Error))
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return fmt.Errorf("cloudflared worker response missing result")
	}
	if err := json.Unmarshal(response.Result, output); err != nil {
		return fmt.Errorf("decode cloudflared worker result: %w", err)
	}
	return nil
}

func (s *CloudflaredConnectorService) resolveCloudflaredWorker(ctx context.Context, capability string) (PluginWorkerStatus, error) {
	if s.container == nil || s.container.PluginSupervisor == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin supervisor is required")
	}
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return PluginWorkerStatus{}, fmt.Errorf("cloudflared connector capability is required")
	}

	status, ok := s.container.PluginSupervisor.WorkerStatus(CloudflaredConnectorPluginID)
	if !ok {
		return PluginWorkerStatus{}, fmt.Errorf("cloudflared connector worker is not registered")
	}
	if status.Kind != shared.AdapterKindConnector {
		return PluginWorkerStatus{}, fmt.Errorf("cloudflared worker has unexpected kind %q", status.Kind)
	}
	if s.cloudflaredWorkerUsable(status, capability) {
		return status, nil
	}

	if err := s.container.PluginSupervisor.RestartWorker(ctx, CloudflaredConnectorPluginID); err != nil {
		return PluginWorkerStatus{}, fmt.Errorf("cloudflared connector worker restart failed: %w", err)
	}
	return s.waitForCloudflaredWorker(ctx, capability)
}

func (s *CloudflaredConnectorService) waitForCloudflaredWorker(ctx context.Context, capability string) (PluginWorkerStatus, error) {
	return waitForPluginWorkerStatus(
		ctx,
		func() (PluginWorkerStatus, bool) {
			return s.container.PluginSupervisor.WorkerStatus(CloudflaredConnectorPluginID)
		},
		func(status PluginWorkerStatus) bool {
			return s.cloudflaredWorkerUsable(status, capability)
		},
		s.restartBackoff,
		s.restartDeadline,
		fmt.Errorf("timed out waiting for cloudflared worker restart"),
	)
}

func (s *CloudflaredConnectorService) cloudflaredWorkerUsable(status PluginWorkerStatus, capability string) bool {
	if status.Kind != shared.AdapterKindConnector {
		return false
	}
	if status.State != PluginWorkerStateRunning {
		return false
	}
	if cloudflaredWorkerExecAddress(status.Metadata) == "" {
		return false
	}
	return supportsCapability(status.Metadata, capability)
}

func cloudflaredWorkerExecAddress(metadata shared.AdapterMetadata) string {
	if len(metadata.Runtime) == 0 {
		return ""
	}
	return strings.TrimSpace(metadata.Runtime[connectorRuntimeExecAddressKey])
}

type cloudflaredWorkerRPCRequest struct {
	Operation string `json:"operation"`
	Payload   any    `json:"payload,omitempty"`
}

type cloudflaredWorkerRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error,omitempty"`
}
