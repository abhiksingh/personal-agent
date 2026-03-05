package cliapp

import (
	"context"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runDoctorConnectivityCheck(ctx context.Context, client *transport.Client, correlationID string) (doctorCheck, transport.DaemonCapabilitiesResponse, error) {
	capabilities, capabilitiesErr := client.DaemonCapabilities(ctx, correlationID)
	if capabilitiesErr != nil {
		return doctorCheck{
			ID:      "daemon.connectivity",
			Title:   "Daemon Connectivity/Auth",
			Status:  doctorCheckStatusFail,
			Summary: "Failed to reach daemon discovery endpoint.",
			Details: doctorErrorDetails(capabilitiesErr),
			Remediation: []string{
				"Start Personal Agent Daemon and confirm --mode/--address/--auth-token values.",
				"Run `personal-agent meta capabilities` to verify control API connectivity.",
			},
		}, transport.DaemonCapabilitiesResponse{}, capabilitiesErr
	}

	return doctorCheck{
		ID:      "daemon.connectivity",
		Title:   "Daemon Connectivity/Auth",
		Status:  doctorCheckStatusPass,
		Summary: "Daemon discovery endpoint is reachable and authenticated.",
		Details: map[string]any{
			"api_version":               capabilities.APIVersion,
			"route_group_count":         len(capabilities.RouteGroups),
			"realtime_event_type_count": len(capabilities.RealtimeEventTypes),
		},
	}, capabilities, nil
}

func buildDoctorConnectivityFailureChecks(includeOptional bool) []doctorCheck {
	checks := []doctorCheck{
		{
			ID:      "daemon.lifecycle",
			Title:   "Daemon Lifecycle Health",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "workspace.context",
			Title:   "Active Workspace Context",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "providers.readiness",
			Title:   "Provider Connectivity Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "models.route_readiness",
			Title:   "Model Route Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "channels.mappings",
			Title:   "Channel Mapping Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "secrets.references",
			Title:   "Secret Reference Registration",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
		{
			ID:      "plugins.health",
			Title:   "Plugin Runtime Health",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		},
	}
	if includeOptional {
		checks = append(checks, doctorCheck{
			ID:      "tooling.optional",
			Title:   "Optional Tooling Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped because daemon connectivity/auth failed.",
		})
	}
	return checks
}

func runDoctorLifecycleHealthCheck(ctx context.Context, client *transport.Client, correlationID string) doctorCheck {
	lifecycleStatus, lifecycleErr := client.DaemonLifecycleStatus(ctx, correlationID)
	if lifecycleErr != nil {
		return doctorCheck{
			ID:      "daemon.lifecycle",
			Title:   "Daemon Lifecycle Health",
			Status:  doctorCheckStatusWarn,
			Summary: "Could not query daemon lifecycle status.",
			Details: doctorErrorDetails(lifecycleErr),
			Remediation: []string{
				"Run `personal-agent meta capabilities` and `personal-agent smoke` to verify daemon route availability.",
			},
		}
	}

	lifecycleCheck := doctorCheck{
		ID:      "daemon.lifecycle",
		Title:   "Daemon Lifecycle Health",
		Status:  doctorCheckStatusPass,
		Summary: "Daemon lifecycle and core runtime health are ready.",
		Details: map[string]any{
			"lifecycle_state": lifecycleStatus.LifecycleState,
			"core_runtime":    lifecycleStatus.HealthClassification.CoreRuntimeState,
			"plugin_runtime":  lifecycleStatus.HealthClassification.PluginRuntimeState,
			"blocking":        lifecycleStatus.HealthClassification.Blocking,
			"control_auth": map[string]any{
				"state":             strings.TrimSpace(lifecycleStatus.ControlAuth.State),
				"source":            strings.TrimSpace(lifecycleStatus.ControlAuth.Source),
				"remediation_hints": lifecycleStatus.ControlAuth.RemediationHints,
			},
			"worker_summary": lifecycleStatus.WorkerSummary,
		},
	}

	if lifecycleStatus.HealthClassification.Blocking ||
		!strings.EqualFold(strings.TrimSpace(lifecycleStatus.LifecycleState), "running") ||
		!strings.EqualFold(strings.TrimSpace(lifecycleStatus.HealthClassification.CoreRuntimeState), "ready") {
		lifecycleCheck.Status = doctorCheckStatusFail
		lifecycleCheck.Summary = "Daemon lifecycle indicates a blocking runtime state."
		lifecycleCheck.Remediation = []string{
			"Run `personal-agent meta capabilities` and inspect daemon lifecycle status for core runtime blockers.",
			"Run daemon lifecycle repair from app/CLI if available for this host mode.",
		}
	} else if !strings.EqualFold(strings.TrimSpace(lifecycleStatus.HealthClassification.PluginRuntimeState), "ready") {
		lifecycleCheck.Status = doctorCheckStatusWarn
		lifecycleCheck.Summary = "Daemon core runtime is ready, but plugin runtime reports degradation."
		lifecycleCheck.Remediation = []string{
			"Inspect plugin status via `POST /v1/channels/status` and `POST /v1/connectors/status`.",
		}
	}

	authState := strings.ToLower(strings.TrimSpace(lifecycleStatus.ControlAuth.State))
	switch authState {
	case "missing":
		lifecycleCheck.Status = doctorCheckStatusFail
		lifecycleCheck.Summary = "Daemon control auth token is missing."
		lifecycleCheck.Remediation = appendDoctorRemediation(
			lifecycleCheck.Remediation,
			lifecycleStatus.ControlAuth.RemediationHints...,
		)
	}
	return lifecycleCheck
}

func runDoctorWorkspaceContextCheck(ctx context.Context, client *transport.Client, requestedWorkspace string, correlationID string) (doctorCheck, string) {
	activeContext, contextErr := client.IdentityActiveContext(ctx, transport.IdentityActiveContextRequest{
		WorkspaceID: requestedWorkspace,
	}, correlationID)
	workspace := normalizeWorkspace(requestedWorkspace)
	if contextErr != nil {
		return doctorCheck{
			ID:      "workspace.context",
			Title:   "Active Workspace Context",
			Status:  doctorCheckStatusWarn,
			Summary: "Could not query daemon identity context; using CLI workspace fallback.",
			Details: doctorErrorDetails(contextErr),
			Remediation: []string{
				"Run `personal-agent identity context` and `personal-agent identity select-workspace --workspace <id>`.",
			},
		}, workspace
	}

	resolvedWorkspace := strings.TrimSpace(activeContext.ActiveContext.WorkspaceID)
	if strings.TrimSpace(requestedWorkspace) == "" && resolvedWorkspace != "" {
		workspace = normalizeWorkspace(resolvedWorkspace)
	}

	contextCheck := doctorCheck{
		ID:      "workspace.context",
		Title:   "Active Workspace Context",
		Status:  doctorCheckStatusPass,
		Summary: "Daemon identity context resolved workspace selection.",
		Details: map[string]any{
			"requested_workspace_id": strings.TrimSpace(requestedWorkspace),
			"resolved_workspace_id":  strings.TrimSpace(activeContext.ActiveContext.WorkspaceID),
			"principal_actor_id":     strings.TrimSpace(activeContext.ActiveContext.PrincipalActorID),
			"workspace_source":       strings.TrimSpace(activeContext.ActiveContext.WorkspaceSource),
			"principal_source":       strings.TrimSpace(activeContext.ActiveContext.PrincipalSource),
			"selection_version":      activeContext.ActiveContext.SelectionVersion,
		},
	}
	if strings.TrimSpace(activeContext.ActiveContext.WorkspaceID) == "" {
		contextCheck.Status = doctorCheckStatusFail
		contextCheck.Summary = "Daemon identity context did not return a workspace id."
		contextCheck.Remediation = []string{
			"Run `personal-agent identity bootstrap --workspace <id> --principal <actor-id>`.",
			"Run `personal-agent identity select-workspace --workspace <id>`.",
		}
	} else if strings.TrimSpace(requestedWorkspace) != "" && !strings.EqualFold(requestedWorkspace, activeContext.ActiveContext.WorkspaceID) {
		contextCheck.Status = doctorCheckStatusWarn
		contextCheck.Summary = "Requested workspace differs from daemon active context."
		contextCheck.Remediation = []string{
			"Run `personal-agent identity select-workspace --workspace " + requestedWorkspace + "`.",
		}
	}
	return contextCheck, workspace
}

func buildDoctorQuickModeSkippedChecks(includeOptional bool) []doctorCheck {
	optionalSummary := "Optional tooling checks disabled."
	if includeOptional {
		optionalSummary = "Skipped in quick mode; run full `doctor` to evaluate optional tooling checks."
	}

	return []doctorCheck{
		{
			ID:      "providers.readiness",
			Title:   "Provider Connectivity Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped in quick mode; run full `doctor` for provider diagnostics.",
			Details: map[string]any{"quick_mode": true},
		},
		{
			ID:      "models.route_readiness",
			Title:   "Model Route Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped in quick mode; run full `doctor` for model-route diagnostics.",
			Details: map[string]any{"quick_mode": true},
		},
		{
			ID:      "channels.mappings",
			Title:   "Channel Mapping Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped in quick mode; run full `doctor` for channel-mapping diagnostics.",
			Details: map[string]any{"quick_mode": true},
		},
		{
			ID:      "secrets.references",
			Title:   "Secret Reference Registration",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped in quick mode; run full `doctor` for secret-reference diagnostics.",
			Details: map[string]any{"quick_mode": true},
		},
		{
			ID:      "plugins.health",
			Title:   "Plugin Runtime Health",
			Status:  doctorCheckStatusSkipped,
			Summary: "Skipped in quick mode; run full `doctor` for plugin-runtime diagnostics.",
			Details: map[string]any{"quick_mode": true},
		},
		{
			ID:      "tooling.optional",
			Title:   "Optional Tooling Readiness",
			Status:  doctorCheckStatusSkipped,
			Summary: optionalSummary,
			Details: map[string]any{
				"quick_mode":       true,
				"include_optional": includeOptional,
			},
		},
	}
}
