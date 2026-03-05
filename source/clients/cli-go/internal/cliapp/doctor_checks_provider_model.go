package cliapp

import (
	"context"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runDoctorProviderReadinessCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	providerCheck := doctorCheck{
		ID:      "providers.readiness",
		Title:   "Provider Connectivity Readiness",
		Status:  doctorCheckStatusWarn,
		Summary: "Provider readiness check was not evaluated.",
	}
	providerResults, providerErr := client.CheckProviders(ctx, transport.ProviderCheckRequest{
		WorkspaceID: state.Workspace,
	}, state.CorrelationID)
	if providerErr != nil {
		providerCheck.Summary = "Provider readiness check request failed."
		providerCheck.Details = doctorErrorDetails(providerErr)
		providerCheck.Remediation = []string{
			"Run `personal-agent provider list --workspace " + state.Workspace + "`.",
			"Run `personal-agent provider check --workspace " + state.Workspace + "`.",
		}
		return providerCheck
	}

	readyProviders := []string{}
	failedProviders := []string{}
	for _, result := range providerResults.Results {
		if result.Success {
			readyProviders = append(readyProviders, result.Provider)
		} else {
			failedProviders = append(failedProviders, result.Provider)
		}
	}
	sort.Strings(readyProviders)
	sort.Strings(failedProviders)
	providerCheck.Details = map[string]any{
		"workspace_id":     providerResults.WorkspaceID,
		"overall_success":  providerResults.Success,
		"ready_providers":  readyProviders,
		"failed_providers": failedProviders,
		"result_count":     len(providerResults.Results),
	}
	switch {
	case len(providerResults.Results) == 0:
		providerCheck.Status = doctorCheckStatusFail
		providerCheck.Summary = "Provider readiness returned no providers."
		providerCheck.Remediation = []string{
			"Configure at least one provider via `personal-agent provider set --workspace " + state.Workspace + " ...`.",
		}
	case len(readyProviders) == 0:
		providerCheck.Status = doctorCheckStatusFail
		providerCheck.Summary = "No providers are ready for model routing."
		providerCheck.Remediation = []string{
			"Configure provider credentials and endpoint via `personal-agent provider set`.",
			"Validate provider connectivity via `personal-agent provider check --workspace " + state.Workspace + "`.",
		}
	case len(failedProviders) > 0:
		providerCheck.Status = doctorCheckStatusWarn
		providerCheck.Summary = "Some providers are not ready, but at least one provider is available."
		providerCheck.Remediation = []string{
			"Run provider-specific checks: `personal-agent provider check --workspace " + state.Workspace + " --provider <name>`.",
		}
	default:
		providerCheck.Status = doctorCheckStatusPass
		providerCheck.Summary = "Provider readiness checks succeeded."
	}
	return providerCheck
}

func runDoctorModelRouteReadinessCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	modelRouteCheck := doctorCheck{
		ID:      "models.route_readiness",
		Title:   "Model Route Readiness",
		Status:  doctorCheckStatusWarn,
		Summary: "Model route readiness check was not evaluated.",
	}
	modelRoute, modelRouteErr := client.ResolveModelRoute(ctx, transport.ModelResolveRequest{
		WorkspaceID: state.Workspace,
		TaskClass:   "chat",
	}, state.CorrelationID)
	if modelRouteErr != nil {
		modelRouteCheck.Status = doctorCheckStatusFail
		modelRouteCheck.Summary = "Could not resolve a model route for task class `chat`."
		modelRouteCheck.Details = doctorErrorDetails(modelRouteErr)
		modelRouteCheck.Remediation = []string{
			"Configure provider/model routing via `personal-agent provider set`, `model add`, and `model select`.",
		}
		return modelRouteCheck
	}

	modelRouteCheck.Details = map[string]any{
		"workspace_id": modelRoute.WorkspaceID,
		"task_class":   modelRoute.TaskClass,
		"provider":     modelRoute.Provider,
		"model_key":    modelRoute.ModelKey,
		"source":       modelRoute.Source,
		"notes":        modelRoute.Notes,
	}
	if strings.TrimSpace(modelRoute.Provider) == "" || strings.TrimSpace(modelRoute.ModelKey) == "" {
		modelRouteCheck.Status = doctorCheckStatusFail
		modelRouteCheck.Summary = "Resolved model route payload is missing provider/model identifiers."
		modelRouteCheck.Remediation = []string{
			"Set an explicit route with `personal-agent model select --workspace " + state.Workspace + " --task-class chat --provider <provider> --model <model>`.",
		}
		return modelRouteCheck
	}

	modelRouteCheck.Status = doctorCheckStatusPass
	modelRouteCheck.Summary = "Model route resolves for task class `chat`."
	return modelRouteCheck
}
