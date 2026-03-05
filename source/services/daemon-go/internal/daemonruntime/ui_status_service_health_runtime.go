package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

func (s *UIStatusService) recoverMessagesIngestPermissionDeniedState(
	ctx context.Context,
	workspaceID string,
	state messagesIngestHealthState,
	configuration map[string]any,
) (bool, error) {
	lastError := strings.TrimSpace(state.LastError)
	if !connectorLikelyPermissionDeniedError(lastError) {
		return false, nil
	}

	probeRequest := messagesadapter.StatusRequest{}
	sourceScope := strings.TrimSpace(state.SourceScope)
	if sourceScope != "" && strings.Contains(sourceScope, "/") {
		probeRequest.SourceDBPath = sourceScope
	}

	status := runMessagesStatusProbe(probeRequest)
	fullDiskState, fullDiskDetail, sourcePath := classifyMessagesFullDiskAccessState(status)
	configuration["messages_source_db_path"] = sourcePath
	configuration["messages_full_disk_permission_state"] = normalizeConnectorPermissionState(fullDiskState)
	configuration["messages_full_disk_probe_detail"] = fullDiskDetail

	if normalizeConnectorPermissionState(fullDiskState) != "granted" {
		return false, nil
	}

	if err := s.clearMessagesIngestHealthError(ctx, workspaceID); err != nil {
		return false, err
	}
	if err := s.persistConnectorPermissionState(
		ctx,
		workspaceID,
		"imessage",
		"granted",
		"iMessage connector permissions are available for Personal Agent Daemon (Automation + Full Disk Access).",
		map[string]any{
			"messages_full_disk_permission_state": "granted",
			"messages_full_disk_probe_detail":     fullDiskDetail,
			"messages_source_db_path":             sourcePath,
		},
	); err != nil {
		return false, err
	}

	configuration["permission_state"] = "granted"
	delete(configuration, "ingest_last_error")
	return true, nil
}

func (s *UIStatusService) applyMessagesIngestHealth(
	ctx context.Context,
	workspaceID string,
	card transport.ChannelStatusCard,
) (transport.ChannelStatusCard, error) {
	state, err := s.loadMessagesIngestHealthState(ctx, workspaceID)
	if err != nil {
		return transport.ChannelStatusCard{}, err
	}
	if card.Configuration == nil {
		card.Configuration = map[string]any{}
	}
	if strings.TrimSpace(state.SourceScope) != "" {
		card.Configuration["ingest_source_scope"] = state.SourceScope
	}
	if strings.TrimSpace(state.UpdatedAt) != "" {
		card.Configuration["ingest_updated_at"] = state.UpdatedAt
	}

	lastError := strings.TrimSpace(state.LastError)
	if lastError == "" {
		if _, exists := card.Configuration["status_reason"]; !exists {
			card.Configuration["status_reason"] = channelReasonReady
		}
		delete(card.Configuration, "ingest_last_error")
		return card, nil
	}

	card.Configuration["status_reason"] = channelReasonIngestFailure
	card.Configuration["ingest_last_error"] = lastError
	card.Status = "degraded"

	errorSummary := truncateDiagnosticsText(lastError, 220)
	baseSummary := strings.TrimSpace(card.Summary)
	if baseSummary == "" {
		card.Summary = "iMessage inbound ingest failed: " + errorSummary
		return card, nil
	}
	card.Summary = baseSummary + " Inbound ingest failed: " + errorSummary
	return card, nil
}

func applyMessagesChannelBindingState(card transport.ChannelStatusCard, bound bool) transport.ChannelStatusCard {
	if card.Configuration == nil {
		card.Configuration = map[string]any{}
	}
	card.Configuration["bound_connector"] = "imessage"
	card.Configuration["bound_to_channel"] = bound
	if bound {
		return card
	}
	card.Configured = false
	card.Status = "not_configured"
	card.Summary = "iMessage connector is not bound to the message channel."
	card.Configuration["status_reason"] = channelReasonNotConfigured
	return card
}

func (s *UIStatusService) isConnectorBoundToChannel(
	ctx context.Context,
	workspaceID string,
	channelID string,
	connectorID string,
) (bool, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return false, fmt.Errorf("database is not configured")
	}
	workspace := normalizeWorkspaceID(workspaceID)
	channel := strings.ToLower(strings.TrimSpace(channelID))
	connector := strings.ToLower(strings.TrimSpace(connectorID))
	if workspace == "" || channel == "" || connector == "" {
		return false, nil
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN connector_id = ? AND enabled = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN enabled = 1 THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM channel_connector_bindings
		WHERE workspace_id = ?
		  AND channel_id = ?
	`)

	args := []any{connector, workspace, channel}

	var (
		matchingEnabledCount int
		totalEnabledCount    int
		rowCount             int
	)
	if err := s.container.DB.QueryRowContext(ctx, query, args...).Scan(&matchingEnabledCount, &totalEnabledCount, &rowCount); err != nil {
		if strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no such table: channel_connector_bindings") {
			return false, nil
		}
		return false, fmt.Errorf("load channel connector binding: %w", err)
	}
	if rowCount == 0 {
		return false, nil
	}
	if totalEnabledCount == 0 {
		return false, nil
	}
	return matchingEnabledCount > 0, nil
}

func (s *UIStatusService) loadMessagesIngestHealthState(
	ctx context.Context,
	workspaceID string,
) (messagesIngestHealthState, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return messagesIngestHealthState{}, fmt.Errorf("database is not configured")
	}
	state := messagesIngestHealthState{}
	workspace := normalizeWorkspaceID(workspaceID)
	err := s.container.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE(source_scope, ''),
			COALESCE(last_error, ''),
			COALESCE(updated_at, '')
		FROM automation_source_subscriptions
		WHERE workspace_id = ?
		  AND source = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, workspace, messagesadapter.SourceName).Scan(
		&state.SourceScope,
		&state.LastError,
		&state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return messagesIngestHealthState{}, fmt.Errorf("load messages ingest health: %w", err)
	}
	state.Available = true
	return state, nil
}

func (s *UIStatusService) clearMessagesIngestHealthError(ctx context.Context, workspaceID string) error {
	if s == nil || s.container == nil || s.container.DB == nil {
		return fmt.Errorf("database is not configured")
	}
	workspace := normalizeWorkspaceID(workspaceID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.container.DB.ExecContext(ctx, `
		UPDATE automation_source_subscriptions
		SET last_error = '', updated_at = ?
		WHERE workspace_id = ?
		  AND source = ?
		  AND TRIM(COALESCE(last_error, '')) <> ''
	`, now, workspace, messagesadapter.SourceName); err != nil {
		return fmt.Errorf("clear messages ingest health error: %w", err)
	}
	return nil
}

func resolveManagedChannelState(displayName string, worker PluginWorkerStatus, workerFound bool) (string, string, string) {
	label := strings.TrimSpace(displayName)
	if label == "" {
		label = "Channel"
	}
	if !workerFound {
		return "degraded", fmt.Sprintf("%s worker is not registered with daemon plugin supervisor.", label), channelReasonWorkerMissing
	}

	switch worker.State {
	case PluginWorkerStateRunning:
		return "ready", fmt.Sprintf("%s worker is running.", label), channelReasonReady
	case PluginWorkerStateRegistered, PluginWorkerStateStarting, PluginWorkerStateRestarting:
		return "ready", fmt.Sprintf("%s worker is starting; daemon startup is in progress.", label), channelReasonWorkerStarting
	case PluginWorkerStateStopped:
		return "degraded", fmt.Sprintf("%s worker is stopped.", label), channelReasonWorkerStopped
	case PluginWorkerStateFailed:
		if strings.TrimSpace(worker.LastError) != "" {
			return "degraded", fmt.Sprintf("%s worker failed: %s", label, strings.TrimSpace(worker.LastError)), channelReasonWorkerFailed
		}
		return "degraded", fmt.Sprintf("%s worker is in failed state.", label), channelReasonWorkerFailed
	default:
		return "degraded", fmt.Sprintf("%s worker state: %s", label, strings.ToLower(strings.TrimSpace(string(worker.State)))), channelReasonWorkerFailed
	}
}

func twilioCardCapabilities(mode string, worker PluginWorkerStatus, workerFound bool) []string {
	filtered := []string{}
	if workerFound {
		all := capabilityKeys(worker.Metadata.Capabilities)
		for _, key := range all {
			keyLower := strings.ToLower(strings.TrimSpace(key))
			if keyLower == "" {
				continue
			}
			if mode == "voice" {
				if strings.Contains(keyLower, "voice") {
					filtered = append(filtered, key)
				}
				continue
			}
			if strings.Contains(keyLower, "sms") || strings.Contains(keyLower, "check") {
				filtered = append(filtered, key)
			}
		}
	}
	if len(filtered) > 0 {
		return filtered
	}
	if mode == "voice" {
		return []string{"channel.twilio.voice.start_call"}
	}
	return []string{"channel.twilio.check", "channel.twilio.sms.send"}
}

func resolveTwilioCardState(configured bool, credentialsConfigured bool, worker PluginWorkerStatus, workerFound bool) (string, string) {
	if !configured {
		return "not_configured", "Twilio channel is not configured for this workspace."
	}
	if !credentialsConfigured {
		return "degraded", "Twilio channel configuration exists but credential references are incomplete."
	}
	if !workerFound {
		return "degraded", "Twilio worker is not registered with daemon plugin supervisor."
	}

	switch worker.State {
	case PluginWorkerStateRunning:
		return "ready", "Twilio worker is running."
	case PluginWorkerStateRegistered, PluginWorkerStateStarting, PluginWorkerStateRestarting:
		return "ready", "Twilio worker is starting; daemon startup is in progress."
	case PluginWorkerStateStopped:
		return "degraded", "Twilio worker is stopped."
	case PluginWorkerStateFailed:
		if strings.TrimSpace(worker.LastError) != "" {
			return "degraded", "Twilio worker failed: " + strings.TrimSpace(worker.LastError)
		}
		return "degraded", "Twilio worker is in failed state."
	default:
		return "degraded", "Twilio worker state: " + strings.ToLower(strings.TrimSpace(string(worker.State)))
	}
}

func resolveConnectorCardState(worker PluginWorkerStatus, workerFound bool) (string, string, string) {
	if !workerFound {
		return "degraded", "Connector worker is not registered with daemon plugin supervisor.", connectorReasonWorkerMissing
	}

	switch worker.State {
	case PluginWorkerStateRunning:
		return "ready", "Connector worker is running.", connectorReasonReady
	case PluginWorkerStateRegistered, PluginWorkerStateStarting, PluginWorkerStateRestarting:
		return "starting", "Connector worker is starting.", connectorReasonWorkerStarting
	case PluginWorkerStateStopped:
		return "stopped", "Connector worker is stopped.", connectorReasonWorkerStopped
	case PluginWorkerStateFailed:
		if strings.TrimSpace(worker.LastError) != "" {
			return "failed", "Connector worker failed: " + strings.TrimSpace(worker.LastError), connectorReasonWorkerFailed
		}
		return "failed", "Connector worker is in failed state.", connectorReasonWorkerFailed
	default:
		return "degraded", "Connector worker state: " + strings.ToLower(strings.TrimSpace(string(worker.State))), connectorReasonRuntimeFailure
	}
}
