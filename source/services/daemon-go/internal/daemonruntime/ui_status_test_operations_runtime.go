package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func (s *UIStatusService) TestChannelOperation(ctx context.Context, request transport.ChannelTestOperationRequest) (transport.ChannelTestOperationResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	channelID := strings.ToLower(strings.TrimSpace(request.ChannelID))
	if channelID == "" {
		return transport.ChannelTestOperationResponse{}, fmt.Errorf("channel_id is required")
	}
	operation, err := normalizeUITestOperation(request.Operation)
	if err != nil {
		return transport.ChannelTestOperationResponse{}, err
	}

	pluginID, ok := channelPluginForID(channelID)
	if !ok {
		return transport.ChannelTestOperationResponse{}, fmt.Errorf("unsupported channel test target: %s", channelID)
	}

	workers := listWorkerStatusByPluginID(s.container)
	worker, found := workers[pluginID]
	logicalChannelID, _ := normalizeLogicalChannelID(channelID, true)
	if found &&
		worker.Kind != shared.AdapterKindChannel &&
		!(logicalChannelID == "voice" && worker.Kind == shared.AdapterKindConnector) &&
		!(logicalChannelID == "message" && worker.Kind == shared.AdapterKindConnector) &&
		!(logicalChannelID == "app" && worker.Kind == shared.AdapterKindConnector) {
		found = false
	}

	success := found && worker.State == PluginWorkerStateRunning
	status := "failed"
	summary := fmt.Sprintf("%s channel worker is unavailable.", channelID)
	if found && worker.State != PluginWorkerStateRunning {
		status = "degraded"
		summary = fmt.Sprintf("%s channel worker is %s.", channelID, worker.State)
	}
	if success {
		status = "ok"
		summary = fmt.Sprintf("%s channel worker is healthy.", channelID)
	}

	details := transport.UIStatusTestOperationDetails{
		PluginID:         pluginID,
		WorkerRegistered: boolPtr(found),
		WorkerState:      "",
	}
	if found {
		details.WorkerState = string(worker.State)
	}

	if logicalChannelID == "voice" {
		twilioConfig, configured, loadErr := s.loadTwilioConfig(ctx, workspace)
		if loadErr != nil {
			return transport.ChannelTestOperationResponse{}, loadErr
		}
		details.Configured = boolPtr(configured)
		details.CredentialsConfigured = boolPtr(twilioConfig.CredentialsConfigured)
		details.Endpoint = strings.TrimSpace(twilioConfig.Endpoint)
		if !configured {
			success = false
			status = "not_configured"
			summary = fmt.Sprintf("%s channel is not configured.", channelID)
		} else if !twilioConfig.CredentialsConfigured {
			success = false
			status = "degraded"
			summary = fmt.Sprintf("%s channel credentials are incomplete.", channelID)
		}
	}

	return transport.ChannelTestOperationResponse{
		WorkspaceID: workspace,
		ChannelID:   channelID,
		Operation:   operation,
		Success:     success,
		Status:      status,
		Summary:     summary,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Details:     details,
	}, nil
}

func (s *UIStatusService) TestConnectorOperation(ctx context.Context, request transport.ConnectorTestOperationRequest) (transport.ConnectorTestOperationResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	connectorID := strings.ToLower(strings.TrimSpace(request.ConnectorID))
	if connectorID == "" {
		return transport.ConnectorTestOperationResponse{}, fmt.Errorf("connector_id is required")
	}
	operation, err := normalizeUITestOperation(request.Operation)
	if err != nil {
		return transport.ConnectorTestOperationResponse{}, err
	}

	pluginID, ok := connectorPluginForID(connectorID)
	if !ok {
		return transport.ConnectorTestOperationResponse{}, fmt.Errorf("unsupported connector test target: %s", connectorID)
	}

	workers := listWorkerStatusByPluginID(s.container)
	worker, found := workers[pluginID]
	normalizedConnectorID := normalizeChannelMappingConnectorID(connectorID)
	if normalizedConnectorID == "" {
		normalizedConnectorID = connectorID
	}
	if found &&
		worker.Kind != shared.AdapterKindConnector &&
		!(normalizedConnectorID == "twilio" && worker.Kind == shared.AdapterKindChannel) &&
		!(normalizedConnectorID == "imessage" && worker.Kind == shared.AdapterKindChannel) &&
		!(normalizedConnectorID == "builtin.app" && worker.Kind == shared.AdapterKindChannel) {
		found = false
	}

	success := found && worker.State == PluginWorkerStateRunning
	status := "failed"
	summary := fmt.Sprintf("%s connector worker is unavailable.", connectorID)
	if found && worker.State != PluginWorkerStateRunning {
		status = "degraded"
		summary = fmt.Sprintf("%s connector worker is %s.", connectorID, worker.State)
	}
	if success {
		status = "ok"
		summary = fmt.Sprintf("%s connector worker is healthy.", connectorID)
	}

	details := transport.UIStatusTestOperationDetails{
		PluginID:         pluginID,
		WorkerRegistered: boolPtr(found),
		WorkerState:      "",
	}
	if found {
		details.WorkerState = string(worker.State)
	}

	if found &&
		worker.State == PluginWorkerStateRunning &&
		normalizedConnectorID != "cloudflared" &&
		strings.TrimSpace(cloudflaredWorkerExecAddress(worker.Metadata)) != "" &&
		strings.TrimSpace(worker.execAuthToken) != "" {
		probe, probeErr := runConnectorExecutePathProbe(ctx, workspace, normalizedConnectorID, worker)
		details.ExecutePathReady = boolPtr(probe.Ready)
		if probe.StatusCode > 0 {
			details.ExecutePathProbeCode = intPtr(probe.StatusCode)
		}
		if strings.TrimSpace(probe.Error) != "" {
			details.ExecutePathProbeErr = strings.TrimSpace(probe.Error)
		}
		if probeErr != nil || !probe.Ready {
			success = false
			status = "degraded"
			probeFailure := strings.TrimSpace(uiFirstNonEmpty(probe.Error, daemonErrorString(probeErr)))
			if probeFailure == "" {
				probeFailure = "connector execute endpoint probe failed"
			}
			summary = fmt.Sprintf("%s connector execute endpoint probe failed: %s", connectorID, probeFailure)
		}
	}

	if connectorID == "cloudflared" && found && worker.State == PluginWorkerStateRunning {
		versionResponse, probeErr := runCloudflaredConnectorVersionProbe(ctx, workspace, worker)
		if probeErr != nil {
			success = false
			status = "failed"
			summary = "cloudflared connector probe failed"
			details.ProbeError = probeErr.Error()
		} else {
			details.Available = boolPtr(versionResponse.Available)
			details.BinaryPath = strings.TrimSpace(versionResponse.BinaryPath)
			details.DryRun = boolPtr(versionResponse.DryRun)
			details.Stdout = strings.TrimSpace(versionResponse.Stdout)
			details.Stderr = strings.TrimSpace(versionResponse.Stderr)
			if !versionResponse.Available {
				success = false
				status = "failed"
				summary = "cloudflared binary is not available"
			}
		}
	}
	if normalizedConnectorID == "twilio" {
		twilioConfig, configured, loadErr := s.loadTwilioConfig(ctx, workspace)
		if loadErr != nil {
			return transport.ConnectorTestOperationResponse{}, loadErr
		}
		details.Configured = boolPtr(configured)
		details.CredentialsConfigured = boolPtr(twilioConfig.CredentialsConfigured)
		details.Endpoint = strings.TrimSpace(twilioConfig.Endpoint)
		details.SMSNumber = strings.TrimSpace(twilioConfig.SMSNumber)
		details.VoiceNumber = strings.TrimSpace(twilioConfig.VoiceNumber)
		if !configured {
			success = false
			status = "not_configured"
			summary = "twilio connector is not configured."
		} else if !twilioConfig.CredentialsConfigured {
			success = false
			status = "degraded"
			summary = "twilio connector credentials are incomplete."
		}
	}

	return transport.ConnectorTestOperationResponse{
		WorkspaceID: workspace,
		ConnectorID: connectorID,
		Operation:   operation,
		Success:     success,
		Status:      status,
		Summary:     summary,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Details:     details,
	}, nil
}
