package daemonruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func assessConnectorRuntimeStatus(
	ctx context.Context,
	workspaceID string,
	connectorID string,
	status string,
	summary string,
	statusReason string,
	worker PluginWorkerStatus,
	workerFound bool,
) (string, string, string, map[string]any) {
	configuration := map[string]any{}
	normalizedConnectorID := strings.ToLower(strings.TrimSpace(connectorID))
	if normalizedConnectorID == "" {
		return status, summary, statusReason, configuration
	}

	if workerFound {
		if permissionSummary, ok := classifyConnectorPermissionFailure(normalizedConnectorID, worker.LastError); ok {
			status = "degraded"
			summary = permissionSummary
			statusReason = connectorReasonPermissionMissing
			configuration["permission_state"] = "missing"
		}
	}

	if workerFound &&
		worker.State == PluginWorkerStateRunning &&
		normalizedConnectorID != "cloudflared" &&
		strings.TrimSpace(cloudflaredWorkerExecAddress(worker.Metadata)) != "" &&
		strings.TrimSpace(worker.execAuthToken) != "" {
		probe, probeErr := runConnectorExecutePathProbe(ctx, workspaceID, normalizedConnectorID, worker)
		configuration["execute_path_probe_ready"] = probe.Ready
		if probe.StatusCode > 0 {
			configuration["execute_path_probe_status_code"] = probe.StatusCode
		}
		if strings.TrimSpace(probe.Error) != "" {
			configuration["execute_path_probe_error"] = strings.TrimSpace(probe.Error)
		}
		if probeErr != nil || !probe.Ready {
			status = "degraded"
			probeFailure := strings.TrimSpace(uiFirstNonEmpty(probe.Error, daemonErrorString(probeErr)))
			if probeFailure == "" {
				probeFailure = "connector execute endpoint probe failed"
			}
			if connectorSupportsPermissionPrompt(normalizedConnectorID) && connectorLikelyPermissionDeniedError(probeFailure) {
				statusReason = connectorReasonPermissionMissing
				configuration["permission_state"] = "missing"
				if spec, ok := connectorPermissionProbeSpecs[normalizedConnectorID]; ok {
					summary = fmt.Sprintf(
						"%s connector permissions are missing. Open System Settings > %s and allow Personal Agent Daemon.",
						spec.displayName,
						spec.systemTarget,
					)
				} else {
					summary = "Connector permissions are missing. Open System Settings and allow Personal Agent Daemon."
				}
			} else {
				statusReason = connectorReasonExecutePathFailure
				summary = "Connector execute endpoint probe failed: " + probeFailure
			}
		}
	}

	if normalizedConnectorID != "cloudflared" || !workerFound || worker.State != PluginWorkerStateRunning {
		return status, summary, statusReason, configuration
	}

	probe, err := runCloudflaredConnectorVersionProbe(ctx, workspaceID, worker)
	if err != nil {
		status = "degraded"
		statusReason = connectorReasonCloudflaredRuntimeFailure
		summary = "Cloudflared runtime probe failed via daemon worker: " + strings.TrimSpace(err.Error())
		configuration["cloudflared_probe_error"] = strings.TrimSpace(err.Error())
		return status, summary, statusReason, configuration
	}

	configuration["cloudflared_available"] = probe.Available
	configuration["cloudflared_binary_path"] = strings.TrimSpace(probe.BinaryPath)
	configuration["cloudflared_dry_run"] = probe.DryRun
	configuration["cloudflared_exit_code"] = probe.ExitCode
	if errText := strings.TrimSpace(probe.Error); errText != "" {
		configuration["cloudflared_error"] = errText
	}

	if probe.Available {
		status = "ready"
		statusReason = connectorReasonReady
		if version := strings.TrimSpace(probe.Version); version != "" {
			summary = "Cloudflared runtime is available: " + version
		} else {
			summary = "Cloudflared runtime is available."
		}
		return status, summary, statusReason, configuration
	}

	status = "degraded"
	if cloudflaredVersionIndicatesMissingBinary(probe) {
		statusReason = connectorReasonCloudflaredBinaryMissing
		binaryPath := strings.TrimSpace(probe.BinaryPath)
		if binaryPath == "" {
			binaryPath = "cloudflared"
		}
		summary = "Cloudflared binary is unavailable or not executable at `" + binaryPath + "`."
		return status, summary, statusReason, configuration
	}

	statusReason = connectorReasonCloudflaredRuntimeFailure
	if errText := strings.TrimSpace(probe.Error); errText != "" {
		summary = "Cloudflared runtime check failed: " + errText
	} else if stderr := strings.TrimSpace(probe.Stderr); stderr != "" {
		summary = "Cloudflared runtime check failed: " + stderr
	} else {
		summary = "Cloudflared runtime check failed."
	}
	return status, summary, statusReason, configuration
}

func classifyConnectorPermissionFailure(connectorID string, lastError string) (string, bool) {
	spec, ok := connectorPermissionProbeSpecs[strings.ToLower(strings.TrimSpace(connectorID))]
	if !ok {
		return "", false
	}
	if !connectorLikelyPermissionDeniedError(lastError) {
		return "", false
	}
	return fmt.Sprintf(
		"%s automation access is not granted. Open System Settings > %s and allow Personal Agent Daemon.",
		spec.displayName,
		spec.systemTarget,
	), true
}

func connectorLikelyPermissionDeniedError(value string) bool {
	detail := strings.ToLower(strings.TrimSpace(value))
	if detail == "" {
		return false
	}
	return strings.Contains(detail, "not authorized") ||
		strings.Contains(detail, "not permitted") ||
		strings.Contains(detail, "not allowed") ||
		strings.Contains(detail, "permission denied") ||
		strings.Contains(detail, "operation not permitted") ||
		strings.Contains(detail, "allow javascript from apple events") ||
		strings.Contains(detail, "send apple events") ||
		strings.Contains(detail, "-1743")
}

func normalizeConnectorPermissionState(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "granted":
		return "granted"
	case "missing":
		return "missing"
	default:
		return "unknown"
	}
}

func normalizeConnectorPermissionStateAny(value any) (string, bool) {
	stringValue, ok := value.(string)
	if !ok {
		return "", false
	}
	normalized := normalizeConnectorPermissionState(stringValue)
	return normalized, normalized != "unknown" || strings.EqualFold(strings.TrimSpace(stringValue), "unknown")
}

func (s *UIStatusService) loadStoredConnectorPermissionConfiguration(
	ctx context.Context,
	workspaceID string,
	connectorID string,
) (map[string]any, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedConnectorID := strings.ToLower(strings.TrimSpace(connectorID))
	if normalizedConnectorID == "" {
		return map[string]any{}, nil
	}
	connectorType := uiConnectorConfigPrefix + normalizedConnectorID
	config, err := s.readUIConfigByID(ctx, workspace, uiConfigRecordID(workspace, connectorType))
	if err != nil {
		return nil, err
	}
	if len(config) == 0 {
		return map[string]any{}, nil
	}
	return cloneUIAnyMap(config), nil
}

func (s *UIStatusService) persistConnectorPermissionState(
	ctx context.Context,
	workspaceID string,
	connectorID string,
	permissionState string,
	message string,
	extra map[string]any,
) error {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedConnectorID := strings.ToLower(strings.TrimSpace(connectorID))
	if normalizedConnectorID == "" {
		return fmt.Errorf("connector_id is required")
	}
	metadata := map[string]any{
		"permission_state":           normalizeConnectorPermissionState(permissionState),
		"permission_last_checked_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if trimmedMessage := strings.TrimSpace(message); trimmedMessage != "" {
		metadata["permission_last_message"] = trimmedMessage
	}
	for key, value := range cloneUIAnyMap(extra) {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		metadata[trimmedKey] = value
	}
	if _, _, err := s.upsertUIConfig(ctx, workspace, uiConnectorConfigPrefix+normalizedConnectorID, metadata, true); err != nil {
		return fmt.Errorf("persist connector permission state: %w", err)
	}
	return nil
}

func probeCloudflaredConnectorVersion(ctx context.Context, workspaceID string, worker PluginWorkerStatus) (transport.CloudflaredVersionResponse, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	address := cloudflaredWorkerExecAddress(worker.Metadata)
	if strings.TrimSpace(address) == "" {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("cloudflared connector worker has no execution endpoint")
	}
	authToken := strings.TrimSpace(worker.execAuthToken)
	if authToken == "" {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("cloudflared connector worker has no daemon-issued auth token")
	}

	payload, err := json.Marshal(cloudflaredWorkerRPCRequest{
		Operation: CloudflaredConnectorOperationVersion,
		Payload: map[string]any{
			"workspace_id": workspace,
		},
	})
	if err != nil {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("marshal cloudflared version probe: %w", err)
	}

	probeCtx := ctx
	if probeCtx == nil {
		probeCtx = context.Background()
	}
	request, err := http.NewRequestWithContext(probeCtx, http.MethodPost, "http://"+address+"/execute", bytes.NewReader(payload))
	if err != nil {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("build cloudflared version probe request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+authToken)

	httpClient := &http.Client{Timeout: 2 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return transport.CloudflaredVersionResponse{}, err
	}
	defer response.Body.Close()

	responseBody, truncated, err := readBoundedHTTPResponseBody(response.Body, daemonWorkerRPCResponseBodyLimitBytes)
	if err != nil {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("read cloudflared version probe response: %w", err)
	}
	if truncated {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf(
			"cloudflared version probe response exceeded max size of %d bytes",
			daemonWorkerRPCResponseBodyLimitBytes,
		)
	}
	if response.StatusCode >= 400 {
		message := strings.TrimSpace(extractWorkerErrorMessage(responseBody))
		if message == "" {
			message = strings.TrimSpace(string(responseBody))
		}
		if message == "" {
			message = response.Status
		}
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("cloudflared version probe failed: %s", message)
	}

	var rpcResponse cloudflaredWorkerRPCResponse
	if err := json.Unmarshal(responseBody, &rpcResponse); err != nil {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("decode cloudflared version probe envelope: %w", err)
	}
	if strings.TrimSpace(rpcResponse.Error) != "" {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("%s", strings.TrimSpace(rpcResponse.Error))
	}
	if len(rpcResponse.Result) == 0 || string(rpcResponse.Result) == "null" {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("cloudflared version probe response missing result")
	}

	var probe transport.CloudflaredVersionResponse
	if err := json.Unmarshal(rpcResponse.Result, &probe); err != nil {
		return transport.CloudflaredVersionResponse{}, fmt.Errorf("decode cloudflared version probe result: %w", err)
	}
	probe.WorkspaceID = workspace
	return probe, nil
}

func cloudflaredVersionIndicatesMissingBinary(probe transport.CloudflaredVersionResponse) bool {
	if probe.Available {
		return false
	}
	detail := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		probe.Error,
		probe.Stderr,
		probe.Stdout,
	}, " ")))
	return strings.Contains(detail, "executable file not found") ||
		strings.Contains(detail, "no such file or directory") ||
		strings.Contains(detail, "command not found") ||
		strings.Contains(detail, "cannot find the file")
}
