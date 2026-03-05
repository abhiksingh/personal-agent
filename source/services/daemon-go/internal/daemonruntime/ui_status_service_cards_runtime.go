package daemonruntime

import (
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func buildManagedChannelCard(
	channelID string,
	displayName string,
	category string,
	worker PluginWorkerStatus,
	workerFound bool,
	configuration map[string]any,
	defaultCapabilities []string,
) transport.ChannelStatusCard {
	status, summary, statusReason := resolveManagedChannelState(displayName, worker, workerFound)
	capabilities := append([]string{}, defaultCapabilities...)
	if workerFound {
		keys := capabilityKeys(worker.Metadata.Capabilities)
		if len(keys) > 0 {
			capabilities = keys
		}
	}
	config := map[string]any{
		"status_reason": statusReason,
	}
	for key, value := range configuration {
		config[key] = value
	}

	card := transport.ChannelStatusCard{
		ChannelID:     channelID,
		DisplayName:   displayName,
		Category:      category,
		Enabled:       true,
		Configured:    true,
		Status:        status,
		Summary:       summary,
		Configuration: config,
		Capabilities:  capabilities,
	}
	if workerFound {
		workerCard := workerStatusCard(worker)
		card.Worker = &workerCard
	}
	return card
}

func buildLogicalChannelStatusCard(
	channelID string,
	displayName string,
	bindings []transport.ChannelConnectorMappingRecord,
	connectorIndex map[string]transport.ConnectorStatusCard,
	fallbackPolicy string,
) transport.ChannelStatusCard {
	channel := strings.ToLower(strings.TrimSpace(channelID))
	label := strings.TrimSpace(displayName)
	if label == "" {
		label = channel
		if len(channel) > 0 {
			label = strings.ToUpper(channel[:1]) + channel[1:]
		}
	}
	policy := strings.ToLower(strings.TrimSpace(fallbackPolicy))
	if policy == "" {
		policy = channelConnectorFallbackPolicyPriorityOrder
	}

	channelBindings := make([]transport.ChannelConnectorMappingRecord, 0, len(bindings))
	for _, binding := range bindings {
		bindingChannel := strings.ToLower(strings.TrimSpace(binding.ChannelID))
		if bindingChannel != channel {
			continue
		}
		connectorID := normalizeChannelMappingConnectorID(binding.ConnectorID)
		if connectorID == "" {
			continue
		}
		binding.ChannelID = channel
		binding.ConnectorID = connectorID
		if binding.Priority <= 0 {
			binding.Priority = len(channelBindings) + 1
		}
		channelBindings = append(channelBindings, binding)
	}
	sort.Slice(channelBindings, func(i, j int) bool {
		if channelBindings[i].Priority == channelBindings[j].Priority {
			return channelBindings[i].ConnectorID < channelBindings[j].ConnectorID
		}
		return channelBindings[i].Priority < channelBindings[j].Priority
	})

	mappedConnectorIDs := make([]string, 0, len(channelBindings))
	enabledConnectorIDs := []string{}
	capabilitySet := map[string]struct{}{}
	type mappedConnectorSnapshot struct {
		ConnectorID string `json:"connector_id"`
		Enabled     bool   `json:"enabled"`
		Priority    int    `json:"priority"`
		Configured  bool   `json:"configured"`
		Status      string `json:"status"`
		Summary     string `json:"summary,omitempty"`
	}
	mappedConnectors := make([]mappedConnectorSnapshot, 0, len(channelBindings))

	var (
		primaryConnectorID string
		primarySummary     string
		primaryReason      string
		primaryWorker      *transport.PluginWorkerStatusCard
	)
	readyCount := 0
	notConfiguredCount := 0
	degradedCount := 0
	configuredConnectorCount := 0

	for _, binding := range channelBindings {
		mappedConnectorIDs = append(mappedConnectorIDs, binding.ConnectorID)
		if binding.Enabled {
			enabledConnectorIDs = append(enabledConnectorIDs, binding.ConnectorID)
			if primaryConnectorID == "" {
				primaryConnectorID = binding.ConnectorID
			}
		}

		connectorCard, found := connectorIndex[binding.ConnectorID]
		connectorStatus := "degraded"
		connectorSummary := "Connector status is unavailable."
		connectorReason := connectorReasonWorkerMissing
		connectorConfigured := false
		connectorCapabilities := append([]string{}, binding.Capabilities...)
		if found {
			status := strings.ToLower(strings.TrimSpace(connectorCard.Status))
			if status != "" {
				connectorStatus = status
			}
			connectorSummary = strings.TrimSpace(connectorCard.Summary)
			if connectorSummary == "" {
				connectorSummary = "Connector status is unavailable."
			}
			connectorReason = connectorStatusReason(connectorCard)
			if connectorReason == "" {
				connectorReason = connectorReasonRuntimeFailure
			}
			connectorConfigured = connectorCard.Configured
			if connectorCard.Worker != nil {
				workerCopy := *connectorCard.Worker
				if primaryWorker == nil && binding.Enabled && binding.ConnectorID == primaryConnectorID {
					primaryWorker = &workerCopy
				}
			}
			if len(connectorCard.Capabilities) > 0 {
				connectorCapabilities = append([]string{}, connectorCard.Capabilities...)
			}
		}

		for _, capability := range connectorCapabilities {
			key := strings.TrimSpace(capability)
			if key == "" {
				continue
			}
			capabilitySet[key] = struct{}{}
		}

		if binding.Enabled && binding.ConnectorID == primaryConnectorID {
			primarySummary = connectorSummary
			primaryReason = connectorReason
		}

		if binding.Enabled {
			if connectorConfigured {
				configuredConnectorCount++
			}
			switch connectorStatus {
			case "ready", "starting":
				readyCount++
			case "not_configured":
				notConfiguredCount++
			default:
				degradedCount++
			}
		}

		mappedConnectors = append(mappedConnectors, mappedConnectorSnapshot{
			ConnectorID: binding.ConnectorID,
			Enabled:     binding.Enabled,
			Priority:    binding.Priority,
			Configured:  connectorConfigured,
			Status:      connectorStatus,
			Summary:     connectorSummary,
		})
	}

	capabilities := make([]string, 0, len(capabilitySet))
	for capability := range capabilitySet {
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)

	configured := configuredConnectorCount > 0
	status := "not_configured"
	statusReason := channelReasonNotConfigured
	summary := fmt.Sprintf("%s channel has no enabled connectors.", label)

	if len(enabledConnectorIDs) > 0 {
		if readyCount > 0 {
			status = "ready"
			statusReason = channelReasonReady
			if degradedCount > 0 || notConfiguredCount > 0 {
				summary = fmt.Sprintf(
					"%s channel is available via %s; %d mapped connector(s) need attention.",
					label,
					humanizeConnectorID(primaryConnectorID),
					degradedCount+notConfiguredCount,
				)
			} else {
				summary = fmt.Sprintf("%s channel is available via %s.", label, humanizeConnectorID(primaryConnectorID))
			}
		} else if notConfiguredCount == len(enabledConnectorIDs) {
			status = "not_configured"
			statusReason = channelReasonNotConfigured
			summary = fmt.Sprintf("%s channel connectors are not configured.", label)
			if strings.TrimSpace(primarySummary) != "" {
				summary = primarySummary
			}
		} else {
			status = "degraded"
			statusReason = connectorReasonToChannelReason(primaryReason, channelReasonWorkerFailed)
			if strings.TrimSpace(primarySummary) != "" {
				summary = primarySummary
			} else {
				summary = fmt.Sprintf("%s channel connectors are degraded.", label)
			}
		}
	}

	configuration := map[string]any{
		"status_reason":         statusReason,
		"fallback_policy":       policy,
		"mapped_connector_ids":  mappedConnectorIDs,
		"enabled_connector_ids": enabledConnectorIDs,
		"mapped_connectors":     mappedConnectors,
	}
	if primaryConnectorID != "" {
		configuration["primary_connector_id"] = primaryConnectorID
	}

	card := transport.ChannelStatusCard{
		ChannelID:     channel,
		DisplayName:   label,
		Category:      channel,
		Enabled:       true,
		Configured:    configured,
		Status:        status,
		Summary:       summary,
		Configuration: configuration,
		Capabilities:  capabilities,
	}
	if primaryWorker != nil {
		card.Worker = primaryWorker
	}
	return card
}

func connectorReasonToChannelReason(reason string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case connectorReasonReady:
		return channelReasonReady
	case connectorReasonNotConfigured:
		return channelReasonNotConfigured
	case connectorReasonWorkerMissing:
		return channelReasonWorkerMissing
	case connectorReasonWorkerStarting:
		return channelReasonWorkerStarting
	case connectorReasonWorkerStopped:
		return channelReasonWorkerStopped
	case connectorReasonIngestFailure:
		return channelReasonIngestFailure
	case connectorReasonWorkerFailed, connectorReasonRuntimeFailure, connectorReasonExecutePathFailure, connectorReasonPermissionMissing, connectorReasonCloudflaredBinaryMissing, connectorReasonCloudflaredRuntimeFailure:
		return channelReasonWorkerFailed
	default:
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return channelReasonWorkerFailed
	}
}

func dedupeNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func channelHasMappedConnector(card transport.ChannelStatusCard, connectorID string) bool {
	target := normalizeChannelMappingConnectorID(connectorID)
	if target == "" || len(card.Configuration) == 0 {
		return false
	}
	if primary, ok := card.Configuration["primary_connector_id"].(string); ok && normalizeChannelMappingConnectorID(primary) == target {
		return true
	}
	rawIDs, ok := card.Configuration["mapped_connector_ids"]
	if !ok {
		return false
	}
	values, ok := rawIDs.([]string)
	if ok {
		for _, value := range values {
			if normalizeChannelMappingConnectorID(value) == target {
				return true
			}
		}
		return false
	}
	items, ok := rawIDs.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			continue
		}
		if normalizeChannelMappingConnectorID(value) == target {
			return true
		}
	}
	return false
}

type messagesIngestHealthState struct {
	SourceScope string
	LastError   string
	UpdatedAt   string
	Available   bool
}
