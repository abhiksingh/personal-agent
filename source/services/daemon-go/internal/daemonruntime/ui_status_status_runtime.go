package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/channelconfig"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

func (s *UIStatusService) ListChannelStatus(ctx context.Context, request transport.ChannelStatusRequest) (transport.ChannelStatusResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	mappings, err := s.ListChannelConnectorMappings(ctx, transport.ChannelConnectorMappingListRequest{
		WorkspaceID: workspace,
	})
	if err != nil {
		return transport.ChannelStatusResponse{}, err
	}
	connectorStatus, err := s.ListConnectorStatus(ctx, transport.ConnectorStatusRequest{
		WorkspaceID: workspace,
	})
	if err != nil {
		return transport.ChannelStatusResponse{}, err
	}
	connectorIndex := map[string]transport.ConnectorStatusCard{}
	for _, card := range connectorStatus.Connectors {
		connectorID := normalizeChannelMappingConnectorID(card.ConnectorID)
		if connectorID == "" {
			continue
		}
		connectorIndex[connectorID] = card
	}

	type logicalChannelSeed struct {
		ChannelID   string
		DisplayName string
	}

	seeds := []logicalChannelSeed{
		{
			ChannelID:   "app",
			DisplayName: "App",
		},
		{
			ChannelID:   "message",
			DisplayName: "Message",
		},
		{
			ChannelID:   "voice",
			DisplayName: "Voice",
		},
	}

	channels := make([]transport.ChannelStatusCard, 0, len(seeds))
	for _, seed := range seeds {
		card := buildLogicalChannelStatusCard(seed.ChannelID, seed.DisplayName, mappings.Bindings, connectorIndex, mappings.FallbackPolicy)
		if seed.ChannelID == "message" && channelHasMappedConnector(card, "imessage") {
			enriched, enrichErr := s.applyMessagesIngestHealth(ctx, workspace, card)
			if enrichErr != nil {
				return transport.ChannelStatusResponse{}, enrichErr
			}
			card = enriched
		}
		card.ConfigFieldDescriptors = channelConfigFieldDescriptors(seed.ChannelID)
		workerHealth := workerHealthSnapshot(card.Worker)
		card.RemediationActions = channelRemediationActions(card, workerHealth)
		card.ActionReadiness, card.ActionBlockers = channelActionReadiness(card)
		channels = append(channels, card)
	}

	return transport.ChannelStatusResponse{
		WorkspaceID: workspace,
		Channels:    channels,
	}, nil
}

func (s *UIStatusService) ListConnectorStatus(ctx context.Context, request transport.ConnectorStatusRequest) (transport.ConnectorStatusResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	workers := listWorkerStatusByPluginID(s.container)
	messagesConnectorBound, err := s.isConnectorBoundToChannel(ctx, workspace, "message", "imessage")
	if err != nil {
		return transport.ConnectorStatusResponse{}, err
	}
	appConnectorBound, err := s.isConnectorBoundToChannel(ctx, workspace, "app", "builtin.app")
	if err != nil {
		return transport.ConnectorStatusResponse{}, err
	}
	twilioConfig, twilioConfigured, err := s.loadTwilioConfig(ctx, workspace)
	if err != nil {
		return transport.ConnectorStatusResponse{}, err
	}

	type connectorSeed struct {
		ConnectorID  string
		PluginID     string
		DisplayName  string
		Capabilities []string
	}

	seeds := []connectorSeed{
		{ConnectorID: "builtin.app", PluginID: appChatWorkerPluginID, DisplayName: "App Connector", Capabilities: []string{"channel.app_chat.send", "channel.app_chat.status"}},
		{ConnectorID: "imessage", PluginID: messagesWorkerPluginID, DisplayName: "iMessage Connector", Capabilities: []string{"channel.messages.send", "channel.messages.status", "channel.messages.ingest_poll"}},
		{ConnectorID: "twilio", PluginID: twilioWorkerPluginID, DisplayName: "Twilio Connector", Capabilities: []string{"channel.twilio.check", "channel.twilio.sms.send", "channel.twilio.voice.start_call"}},
		{ConnectorID: "mail", PluginID: "mail.daemon", DisplayName: "Mail Connector", Capabilities: []string{"mail.draft", "mail.send", "mail.reply", "mail.unread_summary"}},
		{ConnectorID: "calendar", PluginID: "calendar.daemon", DisplayName: "Calendar Connector", Capabilities: []string{"calendar.create", "calendar.update", "calendar.cancel"}},
		{ConnectorID: "browser", PluginID: "browser.daemon", DisplayName: "Browser Connector", Capabilities: []string{"browser.open", "browser.extract", "browser.close"}},
		{ConnectorID: "finder", PluginID: "finder.daemon", DisplayName: "Finder Connector", Capabilities: []string{"finder.list", "finder.preview", "finder.delete"}},
		{ConnectorID: "cloudflared", PluginID: CloudflaredConnectorPluginID, DisplayName: "Cloudflared Connector", Capabilities: []string{CloudflaredConnectorCapabilityVersion, CloudflaredConnectorCapabilityExec}},
	}

	seenPluginIDs := map[string]struct{}{}
	for _, seed := range seeds {
		seenPluginIDs[seed.PluginID] = struct{}{}
	}

	if s.container != nil && s.container.ConnectorRegistry != nil {
		metadata := s.container.ConnectorRegistry.ListMetadata()
		for _, entry := range metadata {
			if _, exists := seenPluginIDs[entry.ID]; exists {
				continue
			}
			connectorID := normalizeChannelMappingConnectorID(connectorIDFromPluginID(entry.ID))
			if connectorID == "" {
				connectorID = connectorIDFromPluginID(entry.ID)
			}
			seeds = append(seeds, connectorSeed{
				ConnectorID:  connectorID,
				PluginID:     entry.ID,
				DisplayName:  strings.TrimSpace(entry.DisplayName),
				Capabilities: capabilityKeys(entry.Capabilities),
			})
			seenPluginIDs[entry.ID] = struct{}{}
		}
	}

	for pluginID, worker := range workers {
		if worker.Kind != shared.AdapterKindConnector {
			continue
		}
		if _, exists := seenPluginIDs[pluginID]; exists {
			continue
		}
		connectorID := normalizeChannelMappingConnectorID(connectorIDFromPluginID(pluginID))
		if connectorID == "" {
			connectorID = connectorIDFromPluginID(pluginID)
		}
		seeds = append(seeds, connectorSeed{
			ConnectorID:  connectorID,
			PluginID:     pluginID,
			DisplayName:  strings.TrimSpace(worker.Metadata.DisplayName),
			Capabilities: capabilityKeys(worker.Metadata.Capabilities),
		})
		seenPluginIDs[pluginID] = struct{}{}
	}

	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].ConnectorID == seeds[j].ConnectorID {
			return seeds[i].PluginID < seeds[j].PluginID
		}
		return seeds[i].ConnectorID < seeds[j].ConnectorID
	})

	bridgeStatus := InspectInboundWatcherBridge("")
	bridgeSourceByConnector := map[string]InboundWatcherBridgeSourceStatus{}
	if source, ok := InboundWatcherBridgeSourceByID(bridgeStatus, "mail"); ok {
		bridgeSourceByConnector["mail"] = source
	}
	if source, ok := InboundWatcherBridgeSourceByID(bridgeStatus, "calendar"); ok {
		bridgeSourceByConnector["calendar"] = source
	}
	if source, ok := InboundWatcherBridgeSourceByID(bridgeStatus, "browser"); ok {
		bridgeSourceByConnector["browser"] = source
	}

	connectors := make([]transport.ConnectorStatusCard, 0, len(seeds))
	for _, seed := range seeds {
		seedConnectorID := normalizeChannelMappingConnectorID(seed.ConnectorID)
		if seedConnectorID == "" {
			seedConnectorID = strings.ToLower(strings.TrimSpace(seed.ConnectorID))
		}
		worker, found := workers[seed.PluginID]
		if found &&
			worker.Kind != shared.AdapterKindConnector &&
			!(seedConnectorID == "twilio" && worker.Kind == shared.AdapterKindChannel) &&
			!(seedConnectorID == "imessage" && worker.Kind == shared.AdapterKindChannel) &&
			!(seedConnectorID == "builtin.app" && worker.Kind == shared.AdapterKindChannel) {
			found = false
		}
		status, summary, statusReason := resolveConnectorCardState(worker, found)
		status, summary, statusReason, runtimeConfiguration := assessConnectorRuntimeStatus(ctx, workspace, seedConnectorID, status, summary, statusReason, worker, found)
		displayName := strings.TrimSpace(seed.DisplayName)
		if displayName == "" {
			displayName = humanizeConnectorID(seedConnectorID)
		}

		configured := true
		storedConfiguration, storedConfigErr := s.loadStoredConnectorPermissionConfiguration(ctx, workspace, seedConnectorID)
		if storedConfigErr != nil {
			return transport.ConnectorStatusResponse{}, fmt.Errorf("load stored connector config: %w", storedConfigErr)
		}
		configuration := cloneUIAnyMap(storedConfiguration)
		configuration["plugin_id"] = seed.PluginID
		configuration["status_reason"] = statusReason
		if connectorSupportsPermissionPrompt(seedConnectorID) {
			if permissionState, ok := normalizeConnectorPermissionStateAny(configuration["permission_state"]); ok {
				configuration["permission_state"] = permissionState
			} else {
				configuration["permission_state"] = "unknown"
			}
		}
		for key, value := range runtimeConfiguration {
			configuration[key] = value
		}
		if bridgeSource, ok := bridgeSourceByConnector[seedConnectorID]; ok {
			bridgeConfiguration := map[string]any{
				"source":             bridgeSource.Source,
				"ready":              bridgeSource.Ready,
				"inbox_root":         bridgeStatus.InboxRoot,
				"pending_dir":        bridgeSource.Pending.Path,
				"pending_exists":     bridgeSource.Pending.Exists,
				"pending_writable":   bridgeSource.Pending.Writable,
				"processed_dir":      bridgeSource.Processed.Path,
				"processed_exists":   bridgeSource.Processed.Exists,
				"processed_writable": bridgeSource.Processed.Writable,
				"failed_dir":         bridgeSource.Failed.Path,
				"failed_exists":      bridgeSource.Failed.Exists,
				"failed_writable":    bridgeSource.Failed.Writable,
			}
			bridgeErrors := []string{}
			for _, detail := range []string{
				strings.TrimSpace(bridgeSource.Pending.Error),
				strings.TrimSpace(bridgeSource.Processed.Error),
				strings.TrimSpace(bridgeSource.Failed.Error),
			} {
				if detail != "" {
					bridgeErrors = append(bridgeErrors, detail)
				}
			}
			if len(bridgeErrors) > 0 {
				bridgeConfiguration["error"] = strings.Join(bridgeErrors, "; ")
			}
			configuration["local_ingest_bridge_ready"] = bridgeSource.Ready
			configuration["local_ingest_bridge"] = bridgeConfiguration
		}

		if seedConnectorID == "builtin.app" {
			configured = appConnectorBound
			configuration["bound_to_channel"] = appConnectorBound
			configuration["channel_id"] = "app"
			switch {
			case !appConnectorBound:
				status = "not_configured"
				summary = "App connector is not bound to the app channel."
				statusReason = connectorReasonNotConfigured
			case !found:
				status = "degraded"
				summary = "App connector worker is not registered with daemon plugin supervisor."
				statusReason = connectorReasonWorkerMissing
			}
			configuration["status_reason"] = statusReason
		}
		if seedConnectorID == "twilio" {
			configured = twilioConfigured
			configuration["account_sid_secret_name"] = twilioConfig.AccountSIDSecretName
			configuration["auth_token_secret_name"] = twilioConfig.AuthTokenSecretName
			configuration["endpoint"] = uiFirstNonEmpty(strings.TrimSpace(twilioConfig.Endpoint), channelconfig.DefaultTwilioEndpoint())
			configuration["sms_number"] = strings.TrimSpace(twilioConfig.SMSNumber)
			configuration["voice_number"] = strings.TrimSpace(twilioConfig.VoiceNumber)
			configuration["credentials_configured"] = twilioConfig.CredentialsConfigured
			configuration["account_sid_configured"] = twilioConfig.AccountSIDConfigured
			configuration["auth_token_configured"] = twilioConfig.AuthTokenConfigured
			switch {
			case !twilioConfigured:
				status = "not_configured"
				summary = "Twilio connector is not configured for this workspace."
				statusReason = connectorReasonNotConfigured
			case !twilioConfig.CredentialsConfigured:
				status = "degraded"
				summary = "Twilio connector configuration exists but credential references are incomplete."
				statusReason = connectorReasonCredentialsIncomplete
			case !found:
				status = "degraded"
				summary = "Twilio connector worker is not registered with daemon plugin supervisor."
				statusReason = connectorReasonWorkerMissing
			}
			configuration["status_reason"] = statusReason
		}
		if seedConnectorID == "imessage" {
			configured = messagesConnectorBound
			configuration["source"] = "apple_messages_chatdb"
			configuration["bound_to_channel"] = messagesConnectorBound
			configuration["channel_id"] = "message"
			switch {
			case !messagesConnectorBound:
				status = "not_configured"
				summary = "iMessage connector is not bound to the message channel."
				statusReason = connectorReasonNotConfigured
			case !found:
				status = "degraded"
				summary = "iMessage connector worker is not registered with daemon plugin supervisor."
				statusReason = connectorReasonWorkerMissing
			}
			ingestState, ingestErr := s.loadMessagesIngestHealthState(ctx, workspace)
			if ingestErr != nil {
				return transport.ConnectorStatusResponse{}, ingestErr
			}
			if strings.TrimSpace(ingestState.SourceScope) != "" {
				configuration["ingest_source_scope"] = ingestState.SourceScope
			}
			if strings.TrimSpace(ingestState.UpdatedAt) != "" {
				configuration["ingest_updated_at"] = ingestState.UpdatedAt
			}
			if ingestLastError := strings.TrimSpace(ingestState.LastError); ingestLastError != "" && messagesConnectorBound {
				recovered, recoveryErr := s.recoverMessagesIngestPermissionDeniedState(ctx, workspace, ingestState, configuration)
				if recoveryErr != nil {
					return transport.ConnectorStatusResponse{}, recoveryErr
				}
				if !recovered {
					status = "degraded"
					configuration["ingest_last_error"] = ingestLastError
					if connectorLikelyPermissionDeniedError(ingestLastError) {
						statusReason = connectorReasonPermissionMissing
						configuration["permission_state"] = "missing"
						summary = "iMessage connector cannot read chat database. Grant Personal Agent Daemon Full Disk Access and retry."
					} else {
						statusReason = connectorReasonIngestFailure
						summary = "iMessage inbound ingest failed: " + truncateDiagnosticsText(ingestLastError, 220)
					}
				}
			}
			configuration["status_reason"] = statusReason
		}

		card := transport.ConnectorStatusCard{
			ConnectorID:            seedConnectorID,
			PluginID:               seed.PluginID,
			DisplayName:            displayName,
			Enabled:                true,
			Configured:             configured,
			Status:                 status,
			Summary:                summary,
			Configuration:          configuration,
			ConfigFieldDescriptors: connectorConfigFieldDescriptors(seedConnectorID),
			Capabilities:           append([]string{}, seed.Capabilities...),
		}
		if found {
			workerCard := workerStatusCard(worker)
			card.Worker = &workerCard
			if len(card.Capabilities) == 0 {
				card.Capabilities = capabilityKeys(worker.Metadata.Capabilities)
			}
		}
		workerHealth := workerHealthSnapshot(card.Worker)
		card.RemediationActions = connectorRemediationActions(card, workerHealth)
		card.ActionReadiness, card.ActionBlockers = connectorActionReadiness(card)
		connectors = append(connectors, card)
	}

	return transport.ConnectorStatusResponse{
		WorkspaceID: workspace,
		Connectors:  connectors,
	}, nil
}
