package daemonruntime

import (
	"personalagent/runtime/internal/transport"
	"strings"
)

func workerStatusCard(status PluginWorkerStatus) transport.PluginWorkerStatusCard {
	return transport.PluginWorkerStatusCard{
		PluginID:           status.PluginID,
		Kind:               string(status.Kind),
		State:              string(status.State),
		ProcessID:          status.ProcessID,
		RestartCount:       status.RestartCount,
		LastError:          strings.TrimSpace(status.LastError),
		LastErrorSource:    strings.TrimSpace(status.LastErrorSource),
		LastErrorOperation: strings.TrimSpace(status.LastErrorOperation),
		LastErrorStderr:    strings.TrimSpace(status.LastErrorStderr),
		LastHeartbeat:      formatTimeCard(status.LastHeartbeat),
		LastTransition:     formatTimeCard(status.LastTransition),
	}
}

func listWorkerStatusByPluginID(container *ServiceContainer) map[string]PluginWorkerStatus {
	out := map[string]PluginWorkerStatus{}
	if container == nil || container.PluginSupervisor == nil {
		return out
	}
	for _, worker := range container.PluginSupervisor.ListWorkers() {
		pluginID := strings.TrimSpace(worker.PluginID)
		if pluginID == "" {
			continue
		}
		out[pluginID] = worker
	}
	return out
}

func workerHealthSnapshot(worker *transport.PluginWorkerStatusCard) transport.WorkerHealthSnapshot {
	if worker == nil {
		return transport.WorkerHealthSnapshot{
			Registered: false,
		}
	}
	workerCopy := *worker
	return transport.WorkerHealthSnapshot{
		Registered: true,
		Worker:     &workerCopy,
	}
}

func channelRemediationActions(card transport.ChannelStatusCard, worker transport.WorkerHealthSnapshot) []transport.DiagnosticsRemediationAction {
	status := strings.ToLower(strings.TrimSpace(card.Status))
	requiresSetup := !card.Configured || status == "not_configured"
	degraded := status != "ready"
	statusReason := channelStatusReason(card)
	ingestFailure := strings.EqualFold(strings.TrimSpace(card.ChannelID), "message") && statusReason == channelReasonIngestFailure
	workerIssue := !worker.Registered || workerStateNeedsRepair(worker.Worker)

	actions := []transport.DiagnosticsRemediationAction{
		{
			Identifier:  "refresh_channel_status",
			Label:       "Refresh Channel Status",
			Intent:      "refresh_status",
			Destination: "/v1/channels/status",
			Parameters: map[string]string{
				"workspace_scope": "current",
				"channel_id":      card.ChannelID,
			},
			Enabled:     true,
			Recommended: degraded,
		},
		{
			Identifier:  "open_channel_setup",
			Label:       "Open Channel Setup",
			Intent:      "navigate",
			Destination: "ui://configuration/channels/" + card.ChannelID,
			Parameters: map[string]string{
				"channel_id": card.ChannelID,
			},
			Enabled:     true,
			Recommended: requiresSetup || ingestFailure,
			Reason: func() string {
				if ingestFailure {
					return "Inbound iMessage ingest is failing. Verify Messages data access and source path."
				}
				return ""
			}(),
		},
		{
			Identifier:  "open_channel_logs",
			Label:       "Open Inspect Logs",
			Intent:      "navigate",
			Destination: "ui://inspect/logs?scope=channel:" + card.ChannelID,
			Parameters: map[string]string{
				"scope":      "channel",
				"channel_id": card.ChannelID,
			},
			Enabled:     true,
			Recommended: degraded,
		},
	}

	if ingestFailure {
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "repair_messages_ingest_access",
			Label:       "Fix Messages Ingest Access",
			Intent:      "navigate",
			Destination: "ui://configuration/channels/message",
			Parameters: map[string]string{
				"channel_id": "message",
			},
			Enabled:     true,
			Recommended: true,
			Reason:      "Grant Personal Agent Daemon access to Messages data and confirm the source database path is readable.",
		})
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "open_channel_system_settings",
			Label:       "Open Full Disk Access",
			Intent:      "open_system_settings",
			Destination: connectorSystemSettingsTarget("imessage"),
			Parameters: map[string]string{
				"channel_id":   "message",
				"connector_id": "imessage",
			},
			Enabled:     true,
			Recommended: true,
			Reason:      "iMessage connector ingest requires Full Disk Access to read the Messages chat database.",
		})
	}

	if channelHasMappedConnector(card, "twilio") && requiresSetup {
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "configure_twilio_channel",
			Label:       "Configure Twilio Credentials",
			Intent:      "navigate",
			Destination: "ui://configuration/channels/twilio",
			Parameters: map[string]string{
				"channel_family": "twilio",
			},
			Enabled:     true,
			Recommended: true,
		})
	}

	if workerIssue {
		enabled := true
		reason := ""
		if workerStateStarting(worker.Worker) {
			enabled = false
			reason = "Worker is already starting/restarting; wait for startup to complete before repair."
		}
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "repair_daemon_runtime",
			Label:       "Run Daemon Repair",
			Intent:      "daemon_lifecycle_control",
			Destination: "/v1/daemon/lifecycle/control",
			Parameters: map[string]string{
				"action": "repair",
			},
			Enabled:     enabled,
			Recommended: !requiresSetup,
			Reason:      reason,
		})
	}

	return actions
}

func connectorRemediationActions(card transport.ConnectorStatusCard, worker transport.WorkerHealthSnapshot) []transport.DiagnosticsRemediationAction {
	status := strings.ToLower(strings.TrimSpace(card.Status))
	degraded := status != "ready"
	requiresSetup := !card.Configured || status == "not_configured"
	statusReason := connectorStatusReason(card)
	executePathFailure := statusReason == connectorReasonExecutePathFailure
	workerIssue := !worker.Registered || workerStateNeedsRepair(worker.Worker) || executePathFailure
	permissionMissing := statusReason == connectorReasonPermissionMissing
	messagesIngestFailure := statusReason == connectorReasonIngestFailure
	cloudflaredBinaryMissing := statusReason == connectorReasonCloudflaredBinaryMissing
	requestPermissionEnabled := connectorSupportsPermissionPrompt(card.ConnectorID)

	actions := []transport.DiagnosticsRemediationAction{
		{
			Identifier:  "refresh_connector_status",
			Label:       "Refresh Connector Status",
			Intent:      "refresh_status",
			Destination: "/v1/connectors/status",
			Parameters: map[string]string{
				"workspace_scope": "current",
				"connector_id":    card.ConnectorID,
			},
			Enabled:     true,
			Recommended: degraded,
		},
		{
			Identifier:  "open_connector_logs",
			Label:       "Open Inspect Logs",
			Intent:      "navigate",
			Destination: "ui://inspect/logs?scope=connector:" + card.ConnectorID,
			Parameters: map[string]string{
				"scope":        "connector",
				"connector_id": card.ConnectorID,
			},
			Enabled:     true,
			Recommended: degraded,
		},
	}

	if requestPermissionEnabled {
		requestPermissionLabel := "Request Permission"
		requestPermissionReason := ""
		if normalizeChannelMappingConnectorID(card.ConnectorID) == "imessage" {
			requestPermissionLabel = "Request Messages Automation Permission"
			requestPermissionReason = "Requests Messages automation access. Full Disk Access is also required for inbound chat database ingestion."
		}
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "request_connector_permission",
			Label:       requestPermissionLabel,
			Intent:      "request_permission",
			Destination: "ui://connectors/request-permission/" + card.ConnectorID,
			Parameters: map[string]string{
				"connector_id": card.ConnectorID,
			},
			Enabled:     true,
			Recommended: permissionMissing,
			Reason:      requestPermissionReason,
		})
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "open_connector_system_settings",
			Label:       "Open System Settings",
			Intent:      "open_system_settings",
			Destination: connectorPermissionSystemSettingsTarget(card.ConnectorID),
			Parameters: map[string]string{
				"connector_id": card.ConnectorID,
			},
			Enabled:     true,
			Recommended: permissionMissing,
		})
	} else if cloudflaredBinaryMissing {
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "install_cloudflared_connector",
			Label:       "Open Cloudflared Setup",
			Intent:      "navigate",
			Destination: "ui://configuration/connectors/cloudflared",
			Parameters: map[string]string{
				"connector_id": card.ConnectorID,
			},
			Enabled:     true,
			Recommended: true,
			Reason:      "Install or configure a reachable cloudflared binary and refresh connector status.",
		})
	}
	if strings.EqualFold(strings.TrimSpace(card.ConnectorID), "twilio") && requiresSetup {
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "configure_twilio_connector",
			Label:       "Configure Twilio Connector",
			Intent:      "navigate",
			Destination: "ui://configuration/connectors/twilio",
			Parameters: map[string]string{
				"connector_id": "twilio",
			},
			Enabled:     true,
			Recommended: true,
			Reason:      "Twilio SMS and voice require one shared connector configuration.",
		})
	}
	connectorID := normalizeChannelMappingConnectorID(card.ConnectorID)
	if connectorID == "imessage" {
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "configure_imessage_connector",
			Label:       "Configure iMessage Connector",
			Intent:      "navigate",
			Destination: "ui://configuration/connectors/imessage",
			Parameters: map[string]string{
				"connector_id": "imessage",
			},
			Enabled:     true,
			Recommended: requiresSetup || messagesIngestFailure,
			Reason: func() string {
				if messagesIngestFailure {
					return "iMessage inbound ingest is failing; review connector setup and local Messages database permissions."
				}
				return ""
			}(),
		})
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "open_imessage_system_settings",
			Label:       "Open Full Disk Access",
			Intent:      "open_system_settings",
			Destination: connectorSystemSettingsTarget("imessage"),
			Parameters: map[string]string{
				"connector_id": "imessage",
			},
			Enabled:     true,
			Recommended: messagesIngestFailure || requiresSetup,
			Reason:      "iMessage connector may require Full Disk Access for chat database ingestion.",
		})
	}

	if workerIssue && !permissionMissing {
		enabled := true
		reason := ""
		if workerStateStarting(worker.Worker) {
			enabled = false
			reason = "Worker is already starting/restarting; wait for startup to complete before repair."
		}
		actions = append(actions, transport.DiagnosticsRemediationAction{
			Identifier:  "repair_daemon_runtime",
			Label:       "Run Daemon Repair",
			Intent:      "daemon_lifecycle_control",
			Destination: "/v1/daemon/lifecycle/control",
			Parameters: map[string]string{
				"action": "repair",
			},
			Enabled:     enabled,
			Recommended: true,
			Reason:      reason,
		})
	}

	return actions
}
