package daemonruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

type connectorExecuteProbeResult struct {
	Ready      bool
	StatusCode int
	Error      string
}

var runConnectorExecutePathProbe = func(ctx context.Context, workspaceID string, connectorID string, worker PluginWorkerStatus) (connectorExecuteProbeResult, error) {
	return probeConnectorExecutePath(ctx, workspaceID, connectorID, worker)
}

func probeConnectorExecutePath(
	ctx context.Context,
	workspaceID string,
	connectorID string,
	worker PluginWorkerStatus,
) (connectorExecuteProbeResult, error) {
	result := connectorExecuteProbeResult{}
	address := strings.TrimSpace(cloudflaredWorkerExecAddress(worker.Metadata))
	if address == "" {
		result.Error = "connector worker has no execution endpoint"
		return result, fmt.Errorf("%s", result.Error)
	}
	authToken := strings.TrimSpace(worker.execAuthToken)
	if authToken == "" {
		result.Error = "connector worker has no daemon-issued auth token"
		return result, fmt.Errorf("%s", result.Error)
	}

	payload, err := buildConnectorExecuteProbePayload(workspaceID, connectorID)
	if err != nil {
		result.Error = strings.TrimSpace(err.Error())
		return result, err
	}

	probeCtx := ctx
	if probeCtx == nil {
		probeCtx = context.Background()
	}
	request, err := http.NewRequestWithContext(probeCtx, http.MethodPost, "http://"+address+"/execute", bytes.NewReader(payload))
	if err != nil {
		result.Error = strings.TrimSpace(err.Error())
		return result, fmt.Errorf("build connector execute probe request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+authToken)

	httpClient := &http.Client{Timeout: 1500 * time.Millisecond}
	response, err := httpClient.Do(request)
	if err != nil {
		result.Error = strings.TrimSpace(err.Error())
		return result, fmt.Errorf("connector execute probe request failed: %w", err)
	}
	defer response.Body.Close()

	result.StatusCode = response.StatusCode
	responseBody, truncated, readErr := readBoundedHTTPResponseBody(response.Body, daemonWorkerRPCResponseBodyLimitBytes)
	if readErr != nil {
		result.Error = strings.TrimSpace(readErr.Error())
		return result, fmt.Errorf("read connector execute probe response: %w", readErr)
	}
	if truncated {
		result.Error = fmt.Sprintf(
			"connector execute probe response exceeded max size of %d bytes",
			daemonWorkerRPCResponseBodyLimitBytes,
		)
		return result, fmt.Errorf("%s", result.Error)
	}
	if message := strings.TrimSpace(extractWorkerErrorMessage(responseBody)); message != "" {
		result.Error = message
	}
	if result.Error == "" && response.StatusCode >= 400 {
		if bodyText := strings.TrimSpace(string(responseBody)); bodyText != "" {
			result.Error = bodyText
		} else {
			result.Error = strings.TrimSpace(response.Status)
		}
	}

	message := strings.TrimSpace(result.Error)
	if response.StatusCode >= 400 && message == "" {
		message = strings.TrimSpace(response.Status)
	}
	if connectorExecuteProbeValidationReachable(response.StatusCode, message) {
		result.Ready = true
		return result, nil
	}
	if response.StatusCode >= 500 {
		return result, fmt.Errorf("connector execute probe failed: %s", message)
	}
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return result, fmt.Errorf("connector execute probe failed: %s", message)
	}
	if response.StatusCode >= 400 {
		return result, fmt.Errorf("connector execute probe failed: %s", message)
	}
	if strings.TrimSpace(result.Error) != "" {
		return result, fmt.Errorf("connector execute probe failed: %s", message)
	}

	result.Ready = true
	return result, nil
}

func connectorExecuteProbeValidationReachable(statusCode int, detail string) bool {
	if statusCode < 400 {
		return false
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(detail))
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "__connector_execute_probe__") &&
		strings.Contains(normalized, "unsupported") {
		return true
	}
	if strings.Contains(normalized, "step input is required") {
		return true
	}
	if strings.Contains(normalized, "missing required input") {
		return true
	}
	return false
}

func buildConnectorExecuteProbePayload(workspaceID string, connectorID string) ([]byte, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	normalizedConnectorID := normalizeChannelMappingConnectorID(connectorID)
	if normalizedConnectorID == "" {
		normalizedConnectorID = strings.ToLower(strings.TrimSpace(connectorID))
	}
	if normalizedConnectorID == "builtin.app" || normalizedConnectorID == "imessage" || normalizedConnectorID == "twilio" {
		return json.Marshal(cloudflaredWorkerRPCRequest{
			Operation: "__connector_execute_probe__",
			Payload: map[string]any{
				"workspace_id": workspace,
			},
		})
	}
	return json.Marshal(workerExecuteRequest{
		ExecutionContext: connectorcontract.ExecutionContext{
			WorkspaceID: workspace,
			TaskID:      "connector.execute_probe",
			RunID:       "connector.execute_probe",
			StepID:      "connector.execute_probe",
		},
		Step: connectorcontract.TaskStep{
			ID:            "connector.execute_probe",
			RunID:         "connector.execute_probe",
			StepIndex:     0,
			Name:          "Connector Execute Probe",
			Status:        shared.TaskStepStatusRunning,
			CapabilityKey: "__connector_execute_probe__",
			Input: map[string]any{
				"url": "https://example.com",
			},
		},
	})
}
