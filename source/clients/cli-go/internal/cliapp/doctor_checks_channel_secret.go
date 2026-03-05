package cliapp

import (
	"context"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runDoctorChannelMappingReadinessCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	mappingCheck := doctorCheck{
		ID:      "channels.mappings",
		Title:   "Channel Mapping Readiness",
		Status:  doctorCheckStatusWarn,
		Summary: "Channel mapping readiness check was not evaluated.",
	}
	mappingDetails := map[string]any{}
	mappingFailures := []string{}
	mappingWarnings := []string{}
	for _, channelID := range []string{"app", "message", "voice"} {
		response, err := client.ChannelConnectorMappingsList(ctx, transport.ChannelConnectorMappingListRequest{
			WorkspaceID: state.Workspace,
			ChannelID:   channelID,
		}, state.CorrelationID)
		if err != nil {
			if channelID == "app" || channelID == "message" {
				mappingFailures = append(mappingFailures, channelID+": query_failed")
			} else {
				mappingWarnings = append(mappingWarnings, channelID+": query_failed")
			}
			continue
		}
		enabled := 0
		for _, binding := range response.Bindings {
			if binding.Enabled {
				enabled++
			}
		}
		mappingDetails[channelID] = map[string]any{
			"enabled_bindings": enabled,
			"binding_count":    len(response.Bindings),
			"fallback_policy":  response.FallbackPolicy,
		}
		if (channelID == "app" || channelID == "message") && enabled == 0 {
			mappingFailures = append(mappingFailures, channelID+": no_enabled_connectors")
		}
		if channelID == "voice" && enabled == 0 {
			mappingWarnings = append(mappingWarnings, channelID+": no_enabled_connectors")
		}
	}
	mappingCheck.Details = mappingDetails
	if len(mappingFailures) > 0 {
		mappingCheck.Status = doctorCheckStatusFail
		mappingCheck.Summary = "Required logical channels are missing enabled connector bindings."
		mappingCheck.Remediation = []string{
			"Run `personal-agent channel mapping list --workspace " + state.Workspace + " --channel app` and `--channel message`.",
			"Enable required mappings with `personal-agent channel mapping enable`.",
		}
		mappingCheck.Details["failures"] = mappingFailures
		return mappingCheck
	}
	if len(mappingWarnings) > 0 {
		mappingCheck.Status = doctorCheckStatusWarn
		mappingCheck.Summary = "Required channel mappings are configured; optional mappings have gaps."
		mappingCheck.Remediation = []string{
			"Review optional mappings via `personal-agent channel mapping list --workspace " + state.Workspace + " --channel voice`.",
		}
		mappingCheck.Details["warnings"] = mappingWarnings
		return mappingCheck
	}

	mappingCheck.Status = doctorCheckStatusPass
	mappingCheck.Summary = "Required logical channel mappings are configured."
	return mappingCheck
}

func runDoctorSecretReferenceCheck(ctx context.Context, client *transport.Client, state *doctorExecutionState) doctorCheck {
	secretCheck := doctorCheck{
		ID:      "secrets.references",
		Title:   "Secret Reference Registration",
		Status:  doctorCheckStatusWarn,
		Summary: "Secret reference registration check was not evaluated.",
	}
	referenceNames := map[string]struct{}{}
	providerList, providerListErr := client.ListProviders(ctx, transport.ProviderListRequest{
		WorkspaceID: state.Workspace,
	}, state.CorrelationID)
	if providerListErr == nil {
		for _, provider := range providerList.Providers {
			name := strings.TrimSpace(provider.APIKeySecretName)
			if name != "" {
				referenceNames[name] = struct{}{}
			}
		}
	}
	twilioConfig, twilioErr := client.TwilioGet(ctx, transport.TwilioGetRequest{
		WorkspaceID: state.Workspace,
	}, state.CorrelationID)
	if twilioErr == nil {
		if name := strings.TrimSpace(twilioConfig.AccountSIDSecretName); name != "" {
			referenceNames[name] = struct{}{}
		}
		if name := strings.TrimSpace(twilioConfig.AuthTokenSecretName); name != "" {
			referenceNames[name] = struct{}{}
		}
	}

	registered := []string{}
	missing := []string{}
	for name := range referenceNames {
		if _, err := client.GetSecretReference(ctx, state.Workspace, name, state.CorrelationID); err != nil {
			missing = append(missing, name)
			continue
		}
		registered = append(registered, name)
	}
	sort.Strings(registered)
	sort.Strings(missing)
	secretCheck.Details = map[string]any{
		"workspace_id":      state.Workspace,
		"registered_refs":   registered,
		"missing_refs":      missing,
		"checked_ref_count": len(referenceNames),
		"provider_list_error": func() string {
			if providerListErr == nil {
				return ""
			}
			return providerListErr.Error()
		}(),
		"twilio_config_error": func() string {
			if twilioErr == nil {
				return ""
			}
			return twilioErr.Error()
		}(),
	}
	switch {
	case len(referenceNames) == 0:
		secretCheck.Status = doctorCheckStatusWarn
		secretCheck.Summary = "No referenced secrets were discovered from provider/twilio configuration."
		secretCheck.Remediation = []string{
			"Register secrets via `personal-agent secret set --workspace " + state.Workspace + " --name <SECRET_NAME> --value <secret>`.",
		}
	case len(missing) > 0:
		secretCheck.Status = doctorCheckStatusFail
		secretCheck.Summary = "One or more configured secret references are missing."
		secretCheck.Remediation = []string{
			"Register missing secret refs via `personal-agent secret set --workspace " + state.Workspace + " --name <SECRET_NAME> --value <secret>`.",
		}
	default:
		secretCheck.Status = doctorCheckStatusPass
		secretCheck.Summary = "Configured secret references are registered."
	}
	return secretCheck
}
