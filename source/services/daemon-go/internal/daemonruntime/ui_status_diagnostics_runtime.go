package daemonruntime

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

type connectorPermissionProbeOutcome struct {
	response transport.ConnectorPermissionResponse
	metadata map[string]any
}

func (s *UIStatusService) ListChannelDiagnostics(ctx context.Context, request transport.ChannelDiagnosticsRequest) (transport.ChannelDiagnosticsResponse, error) {
	statusResponse, err := s.ListChannelStatus(ctx, transport.ChannelStatusRequest{
		WorkspaceID: request.WorkspaceID,
	})
	if err != nil {
		return transport.ChannelDiagnosticsResponse{}, err
	}

	channelIDFilter := strings.TrimSpace(request.ChannelID)
	diagnostics := make([]transport.ChannelDiagnosticsSummary, 0, len(statusResponse.Channels))
	for _, card := range statusResponse.Channels {
		if channelIDFilter != "" && !strings.EqualFold(card.ChannelID, channelIDFilter) {
			continue
		}
		workerHealth := workerHealthSnapshot(card.Worker)
		actions := append([]transport.DiagnosticsRemediationAction{}, card.RemediationActions...)
		if len(actions) == 0 {
			actions = channelRemediationActions(card, workerHealth)
		}
		diagnostics = append(diagnostics, transport.ChannelDiagnosticsSummary{
			ChannelID:          card.ChannelID,
			DisplayName:        card.DisplayName,
			Category:           card.Category,
			Configured:         card.Configured,
			Status:             card.Status,
			Summary:            card.Summary,
			WorkerHealth:       workerHealth,
			RemediationActions: actions,
		})
	}

	return transport.ChannelDiagnosticsResponse{
		WorkspaceID: statusResponse.WorkspaceID,
		Diagnostics: diagnostics,
	}, nil
}

func (s *UIStatusService) ListConnectorDiagnostics(ctx context.Context, request transport.ConnectorDiagnosticsRequest) (transport.ConnectorDiagnosticsResponse, error) {
	statusResponse, err := s.ListConnectorStatus(ctx, transport.ConnectorStatusRequest{
		WorkspaceID: request.WorkspaceID,
	})
	if err != nil {
		return transport.ConnectorDiagnosticsResponse{}, err
	}

	connectorIDFilter := strings.TrimSpace(request.ConnectorID)
	diagnostics := make([]transport.ConnectorDiagnosticsSummary, 0, len(statusResponse.Connectors))
	for _, card := range statusResponse.Connectors {
		if connectorIDFilter != "" && !strings.EqualFold(card.ConnectorID, connectorIDFilter) {
			continue
		}
		workerHealth := workerHealthSnapshot(card.Worker)
		actions := append([]transport.DiagnosticsRemediationAction{}, card.RemediationActions...)
		if len(actions) == 0 {
			actions = connectorRemediationActions(card, workerHealth)
		}
		diagnostics = append(diagnostics, transport.ConnectorDiagnosticsSummary{
			ConnectorID:        card.ConnectorID,
			PluginID:           card.PluginID,
			DisplayName:        card.DisplayName,
			Configured:         card.Configured,
			Status:             card.Status,
			Summary:            card.Summary,
			WorkerHealth:       workerHealth,
			RemediationActions: actions,
		})
	}

	return transport.ConnectorDiagnosticsResponse{
		WorkspaceID: statusResponse.WorkspaceID,
		Diagnostics: diagnostics,
	}, nil
}

func (s *UIStatusService) RequestConnectorPermission(ctx context.Context, request transport.ConnectorPermissionRequest) (transport.ConnectorPermissionResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	connectorID := strings.ToLower(strings.TrimSpace(request.ConnectorID))
	if connectorID == "" {
		return transport.ConnectorPermissionResponse{}, fmt.Errorf("connector_id is required")
	}

	spec, ok := connectorPermissionProbeSpecs[connectorID]
	if !ok {
		return transport.ConnectorPermissionResponse{}, fmt.Errorf("unsupported connector permission request: %s", connectorID)
	}

	var outcome connectorPermissionProbeOutcome
	if connectorID == "imessage" {
		outcome = requestMessagesConnectorPermission(ctx, workspace, spec)
	} else {
		output, err := runConnectorPermissionCommand(ctx, "osascript", connectorPermissionProbeArgs(spec)...)
		if connectorPermissionProbeNeedsWarmLaunch(spec, output, err) {
			_, _ = runConnectorPermissionCommand(ctx, "open", "-a", spec.launchApp)
			// Allow LaunchServices to warm the target app before retrying the probe.
			time.Sleep(350 * time.Millisecond)
			output, err = runConnectorPermissionCommand(ctx, "osascript", connectorPermissionProbeArgs(spec)...)
		}
		permissionState, message := classifyConnectorPermissionProbeResult(spec, output, err)
		outcome = connectorPermissionProbeOutcome{
			response: transport.ConnectorPermissionResponse{
				WorkspaceID:     workspace,
				ConnectorID:     spec.connectorID,
				PermissionState: permissionState,
				Message:         message,
			},
		}
	}
	if persistErr := s.persistConnectorPermissionState(
		ctx,
		workspace,
		outcome.response.ConnectorID,
		outcome.response.PermissionState,
		outcome.response.Message,
		outcome.metadata,
	); persistErr != nil {
		return transport.ConnectorPermissionResponse{}, persistErr
	}
	return outcome.response, nil
}

func requestMessagesConnectorPermission(
	ctx context.Context,
	workspace string,
	spec connectorPermissionProbeSpec,
) connectorPermissionProbeOutcome {
	automationOutput, automationErr := runConnectorPermissionCommand(ctx, "osascript", connectorPermissionProbeArgs(spec)...)
	if connectorPermissionProbeNeedsWarmLaunch(spec, automationOutput, automationErr) {
		_, _ = runConnectorPermissionCommand(ctx, "open", "-a", spec.launchApp)
		// Allow LaunchServices to warm the target app before retrying the probe.
		time.Sleep(350 * time.Millisecond)
		automationOutput, automationErr = runConnectorPermissionCommand(ctx, "osascript", connectorPermissionProbeArgs(spec)...)
	}
	automationState, _ := classifyConnectorPermissionProbeResult(spec, automationOutput, automationErr)
	automationDetail := connectorPermissionProbeDetail(automationOutput, automationErr)

	status := runMessagesStatusProbe(messagesadapter.StatusRequest{})
	fullDiskState, fullDiskDetail, sourcePath := classifyMessagesFullDiskAccessState(status)
	permissionState := combineConnectorPermissionStates(automationState, fullDiskState)
	message := messagesPermissionProbeMessage(
		permissionState,
		automationState,
		fullDiskState,
		sourcePath,
		automationDetail,
		fullDiskDetail,
	)

	return connectorPermissionProbeOutcome{
		response: transport.ConnectorPermissionResponse{
			WorkspaceID:     workspace,
			ConnectorID:     spec.connectorID,
			PermissionState: permissionState,
			Message:         message,
		},
		metadata: map[string]any{
			"messages_automation_permission_state": normalizeConnectorPermissionState(automationState),
			"messages_full_disk_permission_state":  normalizeConnectorPermissionState(fullDiskState),
			"messages_source_db_path":              sourcePath,
			"messages_automation_probe_detail":     automationDetail,
			"messages_full_disk_probe_detail":      fullDiskDetail,
		},
	}
}

func classifyMessagesFullDiskAccessState(status messagesadapter.StatusResponse) (string, string, string) {
	sourcePath := strings.TrimSpace(status.SourceDBPath)
	if sourcePath == "" {
		sourcePath = messagesadapter.ResolveSourceDBPath("")
	}
	if status.Ready {
		return "granted", "Messages chat database is readable.", sourcePath
	}
	detail := strings.TrimSpace(status.Error)
	if detail == "" {
		detail = "Messages chat database is not readable."
	}
	if connectorLikelyPermissionDeniedError(detail) {
		return "missing", detail, sourcePath
	}
	return "unknown", detail, sourcePath
}

func combineConnectorPermissionStates(states ...string) string {
	allGranted := true
	for _, state := range states {
		normalized := normalizeConnectorPermissionState(state)
		if normalized == "missing" {
			return "missing"
		}
		if normalized != "granted" {
			allGranted = false
		}
	}
	if allGranted {
		return "granted"
	}
	return "unknown"
}

func connectorPermissionProbeDetail(output string, runErr error) string {
	detail := strings.TrimSpace(output)
	if detail != "" {
		return detail
	}
	if runErr != nil {
		return strings.TrimSpace(runErr.Error())
	}
	return "ok"
}

func messagesPermissionProbeMessage(
	permissionState string,
	automationState string,
	fullDiskState string,
	sourcePath string,
	automationDetail string,
	fullDiskDetail string,
) string {
	normalizedPermission := normalizeConnectorPermissionState(permissionState)
	normalizedAutomation := normalizeConnectorPermissionState(automationState)
	normalizedFullDisk := normalizeConnectorPermissionState(fullDiskState)
	switch {
	case normalizedPermission == "granted":
		return "iMessage connector permissions are available for Personal Agent Daemon (Automation + Full Disk Access)."
	case normalizedAutomation == "missing" && normalizedFullDisk == "missing":
		return fmt.Sprintf(
			"iMessage connector still needs Automation and Full Disk Access for Personal Agent Daemon. Open System Settings > Privacy & Security > Automation and Full Disk Access, allow Personal Agent Daemon, then retry. Automation probe: %s. Full Disk Access probe (%s): %s.",
			automationDetail,
			sourcePath,
			fullDiskDetail,
		)
	case normalizedAutomation == "missing":
		return fmt.Sprintf(
			"iMessage connector automation access is not granted. Open System Settings > Privacy & Security > Automation and allow Personal Agent Daemon. Automation probe: %s. Full Disk Access probe (%s): %s.",
			automationDetail,
			sourcePath,
			fullDiskDetail,
		)
	case normalizedFullDisk == "missing":
		return fmt.Sprintf(
			"iMessage connector still needs Full Disk Access for Personal Agent Daemon to read %s. Open System Settings > Privacy & Security > Full Disk Access and allow Personal Agent Daemon. Full Disk Access probe: %s. Automation probe: %s.",
			sourcePath,
			fullDiskDetail,
			automationDetail,
		)
	default:
		return fmt.Sprintf(
			"iMessage connector permission probe did not complete cleanly. Verify Automation and Full Disk Access for Personal Agent Daemon. Automation probe: %s. Full Disk Access probe (%s): %s.",
			automationDetail,
			sourcePath,
			fullDiskDetail,
		)
	}
}

func connectorPermissionProbeNeedsWarmLaunch(spec connectorPermissionProbeSpec, output string, runErr error) bool {
	if runErr == nil || strings.TrimSpace(spec.launchApp) == "" {
		return false
	}
	detail := strings.ToLower(strings.TrimSpace(uiFirstNonEmpty(output, runErr.Error())))
	if detail == "" {
		return false
	}
	return strings.Contains(detail, "-600") ||
		strings.Contains(detail, "application isn") ||
		strings.Contains(detail, "isn't running")
}

func connectorPermissionProbeArgs(spec connectorPermissionProbeSpec) []string {
	args := make([]string, 0, len(spec.appleScript)*2)
	for _, line := range spec.appleScript {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		args = append(args, "-e", trimmed)
	}
	return args
}

func classifyConnectorPermissionProbeResult(spec connectorPermissionProbeSpec, output string, runErr error) (string, string) {
	normalizedOutput := strings.TrimSpace(output)
	detail := strings.ToLower(strings.TrimSpace(uiFirstNonEmpty(normalizedOutput, daemonErrorString(runErr))))
	switch {
	case strings.Contains(detail, "not authorized"), strings.Contains(detail, "-1743"), strings.Contains(detail, "not permitted"):
		return "missing", fmt.Sprintf(
			"%s automation access is not granted. Open System Settings > %s and allow Personal Agent Daemon.",
			spec.displayName,
			spec.systemTarget,
		)
	}

	if runErr == nil && connectorPermissionProbeOutputLooksGranted(normalizedOutput) {
		return "granted", fmt.Sprintf(
			"%s permission request dispatched via Personal Agent Daemon. Confirm in %s if access is still missing.",
			spec.displayName,
			spec.systemTarget,
		)
	}
	if runErr == nil {
		return "unknown", fmt.Sprintf(
			"%s permission probe returned an unexpected result: %s. Confirm in System Settings > %s and retry.",
			spec.displayName,
			uiFirstNonEmpty(normalizedOutput, "no probe output"),
			spec.systemTarget,
		)
	}
	switch {
	case strings.Contains(detail, "application isn"), strings.Contains(detail, "-600"):
		return "missing", fmt.Sprintf(
			"%s is unavailable or could not be launched while requesting permission.",
			spec.displayName,
		)
	case strings.Contains(detail, "executable file not found"), strings.Contains(detail, "no such file or directory"):
		return "unknown", "Native permission request tooling is unavailable on this runtime. Use System Settings to grant access manually."
	default:
		message := strings.TrimSpace(normalizedOutput)
		if message == "" {
			message = strings.TrimSpace(runErr.Error())
		}
		return "unknown", fmt.Sprintf(
			"%s permission request did not complete cleanly via daemon runtime: %s",
			spec.displayName,
			message,
		)
	}
}

func connectorPermissionProbeOutputLooksGranted(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	if _, err := strconv.Atoi(trimmed); err == nil {
		return true
	}
	switch strings.ToLower(trimmed) {
	case "true", "ok":
		return true
	default:
		return false
	}
}

func daemonErrorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
