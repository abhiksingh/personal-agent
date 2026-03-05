package cliapp

import (
	"context"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runDoctorPluginRuntimeHealthCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	pluginCheck := doctorCheck{
		ID:      "plugins.health",
		Title:   "Plugin Runtime Health",
		Status:  doctorCheckStatusWarn,
		Summary: "Plugin runtime health check was not evaluated.",
	}

	channelStatus, channelErr := state.loadChannelStatus(ctx, client)
	connectorStatus, connectorErr := state.loadConnectorStatus(ctx, client)
	if channelErr != nil || connectorErr != nil || channelStatus == nil || connectorStatus == nil {
		pluginCheck.Status = doctorCheckStatusWarn
		pluginCheck.Summary = "Could not query channel/connector status for plugin health."
		pluginCheck.Details = map[string]any{
			"channel_status_error": func() string {
				if channelErr == nil {
					return ""
				}
				return channelErr.Error()
			}(),
			"connector_status_error": func() string {
				if connectorErr == nil {
					return ""
				}
				return connectorErr.Error()
			}(),
		}
		pluginCheck.Remediation = []string{
			"Run `personal-agent channel mapping list --workspace " + state.Workspace + "` and inspect status APIs.",
		}
		return pluginCheck
	}

	criticalChannelFailures := []string{}
	nonReadyChannels := []string{}
	nonReadyConnectors := []string{}
	for _, card := range channelStatus.Channels {
		channelID := strings.TrimSpace(card.ChannelID)
		if !strings.EqualFold(strings.TrimSpace(card.Status), "ready") {
			nonReadyChannels = append(nonReadyChannels, channelID+":"+strings.TrimSpace(card.Status))
			if channelID == "app" || channelID == "message" {
				criticalChannelFailures = append(criticalChannelFailures, channelID+":"+strings.TrimSpace(card.Status))
			}
		}
	}
	for _, card := range connectorStatus.Connectors {
		if !strings.EqualFold(strings.TrimSpace(card.Status), "ready") {
			nonReadyConnectors = append(nonReadyConnectors, strings.TrimSpace(card.ConnectorID)+":"+strings.TrimSpace(card.Status))
		}
	}
	sort.Strings(nonReadyChannels)
	sort.Strings(nonReadyConnectors)
	pluginCheck.Details = map[string]any{
		"non_ready_channels":   nonReadyChannels,
		"non_ready_connectors": nonReadyConnectors,
		"channel_count":        len(channelStatus.Channels),
		"connector_count":      len(connectorStatus.Connectors),
	}
	switch {
	case len(criticalChannelFailures) > 0:
		pluginCheck.Status = doctorCheckStatusFail
		pluginCheck.Summary = "Critical logical channels are not ready."
		pluginCheck.Remediation = []string{
			"Repair required channels (`app`, `message`) and inspect connector status/remediation actions.",
		}
		pluginCheck.Details["critical_failures"] = criticalChannelFailures
	case len(nonReadyChannels) > 0 || len(nonReadyConnectors) > 0:
		pluginCheck.Status = doctorCheckStatusWarn
		pluginCheck.Summary = "Some channels/connectors are degraded, but critical channels are ready."
		pluginCheck.Remediation = []string{
			"Review non-ready channel/connector cards and execute recommended remediation actions.",
		}
	default:
		pluginCheck.Status = doctorCheckStatusPass
		pluginCheck.Summary = "All channels/connectors are ready."
	}
	return pluginCheck
}

func runDoctorOptionalToolingCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	optionalCheck := doctorCheck{
		ID:      "tooling.optional",
		Title:   "Optional Tooling Readiness",
		Status:  doctorCheckStatusSkipped,
		Summary: "Optional tooling checks disabled.",
	}
	if !state.IncludeOptional {
		return optionalCheck
	}

	optionalCheck.Status = doctorCheckStatusPass
	optionalCheck.Summary = "Optional tooling checks passed."
	optionalDetails := map[string]any{}
	optionalWarnings := []string{}

	connectorStatus, connectorErr := state.loadConnectorStatus(ctx, client)
	if connectorErr == nil && connectorStatus != nil {
		if cloudflaredCard, found := doctorFindConnectorCard(connectorStatus.Connectors, "cloudflared"); found {
			optionalDetails["cloudflared_status"] = strings.TrimSpace(cloudflaredCard.Status)
			if !strings.EqualFold(strings.TrimSpace(cloudflaredCard.Status), "ready") {
				optionalWarnings = append(optionalWarnings, "cloudflared:"+strings.TrimSpace(cloudflaredCard.Status))
			}
		} else {
			optionalWarnings = append(optionalWarnings, "cloudflared:not_found")
		}

		bridgeNotReady := []string{}
		for _, connectorID := range []string{"mail", "calendar", "browser"} {
			card, found := doctorFindConnectorCard(connectorStatus.Connectors, connectorID)
			if !found {
				bridgeNotReady = append(bridgeNotReady, connectorID+":not_found")
				continue
			}
			ready, ok := card.Configuration["local_ingest_bridge_ready"].(bool)
			if !ok || !ready {
				bridgeNotReady = append(bridgeNotReady, connectorID+":not_ready")
			}
		}
		sort.Strings(bridgeNotReady)
		optionalDetails["local_ingest_bridge_not_ready"] = bridgeNotReady
		if len(bridgeNotReady) > 0 {
			optionalWarnings = append(optionalWarnings, "local_ingest_bridge:not_ready")
		}
	} else {
		optionalWarnings = append(optionalWarnings, "connector_status:query_failed")
	}

	optionalDetails["warnings"] = optionalWarnings
	optionalCheck.Details = optionalDetails
	if len(optionalWarnings) > 0 {
		optionalCheck.Status = doctorCheckStatusWarn
		optionalCheck.Summary = "Optional tooling checks detected readiness gaps."
		optionalCheck.Remediation = []string{
			"Run `personal-agent connector bridge setup --workspace " + state.Workspace + "` to ensure local watcher queue paths.",
			"Run `personal-agent connector cloudflared version --workspace " + state.Workspace + "` for cloudflared runtime diagnostics.",
		}
	}
	return optionalCheck
}
