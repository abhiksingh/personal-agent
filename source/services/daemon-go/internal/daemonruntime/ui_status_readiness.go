package daemonruntime

import (
	"strings"

	"personalagent/runtime/internal/transport"
)

func channelActionReadiness(card transport.ChannelStatusCard) (string, []transport.ActionReadinessBlocker) {
	status := strings.ToLower(strings.TrimSpace(card.Status))
	reason := channelStatusReason(card)
	remediationActionID := firstRecommendedRemediationActionID(card.RemediationActions)
	summary := strings.TrimSpace(card.Summary)

	switch {
	case status == "ready" && reason == channelReasonReady:
		return "ready", nil
	case status == "not_configured" || reason == channelReasonNotConfigured:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "config_incomplete",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel configuration is incomplete."),
				RemediationAction: remediationActionID,
			},
		}
	case reason == channelReasonWorkerMissing || reason == channelReasonWorkerStopped || reason == channelReasonWorkerFailed:
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "worker_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel worker is unavailable."),
				RemediationAction: remediationActionID,
			},
		}
	case reason == channelReasonIngestFailure:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "permission_missing",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel ingest permissions are missing."),
				RemediationAction: remediationActionID,
			},
		}
	case status == "starting" || reason == channelReasonWorkerStarting:
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "worker_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel worker is still starting."),
				RemediationAction: remediationActionID,
			},
		}
	case status == "degraded":
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "worker_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel runtime is degraded."),
				RemediationAction: remediationActionID,
			},
		}
	default:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "config_incomplete",
				Message:           fallbackReadinessBlockerMessage(summary, "Channel action prerequisites are not ready."),
				RemediationAction: remediationActionID,
			},
		}
	}
}

func connectorActionReadiness(card transport.ConnectorStatusCard) (string, []transport.ActionReadinessBlocker) {
	status := strings.ToLower(strings.TrimSpace(card.Status))
	reason := connectorStatusReason(card)
	remediationActionID := firstRecommendedRemediationActionID(card.RemediationActions)
	summary := strings.TrimSpace(card.Summary)

	switch {
	case status == "ready" && reason == connectorReasonReady:
		return "ready", nil
	case status == "not_configured" || reason == connectorReasonNotConfigured || reason == connectorReasonCloudflaredBinaryMissing:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "config_incomplete",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector configuration is incomplete."),
				RemediationAction: remediationActionID,
			},
		}
	case reason == connectorReasonCredentialsIncomplete:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "credentials_missing",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector credentials are incomplete."),
				RemediationAction: remediationActionID,
			},
		}
	case reason == connectorReasonPermissionMissing:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "permission_missing",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector permissions are missing."),
				RemediationAction: remediationActionID,
			},
		}
	case status == "starting" || reason == connectorReasonWorkerStarting:
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "worker_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector worker is still starting."),
				RemediationAction: remediationActionID,
			},
		}
	case reason == connectorReasonExecutePathFailure:
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "execute_path_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector execute endpoint is unavailable."),
				RemediationAction: remediationActionID,
			},
		}
	case status == "stopped" || status == "failed" || status == "degraded" ||
		reason == connectorReasonWorkerMissing || reason == connectorReasonWorkerStopped || reason == connectorReasonWorkerFailed ||
		reason == connectorReasonRuntimeFailure || reason == connectorReasonCloudflaredRuntimeFailure:
		return "degraded", []transport.ActionReadinessBlocker{
			{
				Code:              "worker_unavailable",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector runtime is degraded."),
				RemediationAction: remediationActionID,
			},
		}
	default:
		return "blocked", []transport.ActionReadinessBlocker{
			{
				Code:              "config_incomplete",
				Message:           fallbackReadinessBlockerMessage(summary, "Connector action prerequisites are not ready."),
				RemediationAction: remediationActionID,
			},
		}
	}
}

func firstRecommendedRemediationActionID(actions []transport.DiagnosticsRemediationAction) string {
	for _, action := range actions {
		identifier := strings.TrimSpace(action.Identifier)
		if identifier == "" {
			continue
		}
		if action.Recommended {
			return identifier
		}
	}
	for _, action := range actions {
		identifier := strings.TrimSpace(action.Identifier)
		if identifier != "" {
			return identifier
		}
	}
	return ""
}

func fallbackReadinessBlockerMessage(summary string, fallback string) string {
	if strings.TrimSpace(summary) != "" {
		return summary
	}
	return strings.TrimSpace(fallback)
}
