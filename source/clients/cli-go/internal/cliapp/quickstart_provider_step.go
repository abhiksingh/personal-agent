package cliapp

import (
	"context"
	"strings"
	"time"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func quickstartConfigureProviderStep(ctx context.Context, client *transport.Client, input quickstartProviderConfigInput) quickstartStep {
	providerName, err := providerconfig.NormalizeProvider(input.Provider)
	if err != nil {
		return quickstartStep{
			ID:      "provider.configure",
			Title:   "Provider Configuration",
			Status:  quickstartStepStatusFail,
			Summary: "Provider value is invalid.",
			Details: map[string]any{
				"provider": input.Provider,
				"error":    err.Error(),
			},
			Remediation: []string{
				"Set --provider to one of: openai|anthropic|google|ollama.",
			},
		}
	}

	secretName := strings.TrimSpace(input.APIKeySecretName)
	if providerconfig.ProviderRequiresAPIKey(providerName) {
		resolvedValue, resolveErr := resolveSecretValue(input.APIKey, input.APIKeyFile)
		if resolveErr != nil {
			return quickstartStep{
				ID:      "provider.configure",
				Title:   "Provider Configuration",
				Status:  quickstartStepStatusFail,
				Summary: "Provider requires API key material but none was provided.",
				Details: map[string]any{
					"provider": providerName,
					"error":    resolveErr.Error(),
				},
				Remediation: []string{
					"Provide --api-key or --api-key-file for the selected provider.",
				},
			}
		}
		if secretName == "" {
			secretName = quickstartDefaultAPIKeySecretName(providerName)
		}

		manager, managerErr := newSecretManager()
		if managerErr != nil {
			return quickstartStep{
				ID:      "provider.configure",
				Title:   "Provider Configuration",
				Status:  quickstartStepStatusFail,
				Summary: "Failed to initialize secure-store secret manager.",
				Details: map[string]any{
					"provider": providerName,
					"error":    managerErr.Error(),
				},
				Remediation: []string{
					"Verify secure-store access and rerun quickstart.",
				},
			}
		}
		ref, putErr := manager.Put(input.WorkspaceID, secretName, resolvedValue)
		if putErr != nil {
			return quickstartStep{
				ID:      "provider.configure",
				Title:   "Provider Configuration",
				Status:  quickstartStepStatusFail,
				Summary: "Failed to write provider API key to secure storage.",
				Details: map[string]any{
					"provider":    providerName,
					"secret_name": secretName,
					"error":       putErr.Error(),
				},
				Remediation: []string{
					"Ensure secure-store write access, then rerun quickstart.",
				},
			}
		}
		if _, upsertErr := client.UpsertSecretReference(ctx, transport.SecretReferenceUpsertRequest{
			WorkspaceID: ref.WorkspaceID,
			Name:        ref.Name,
			Backend:     ref.Backend,
			Service:     ref.Service,
			Account:     ref.Account,
		}, input.CorrelationIDBase+".secret_ref"); upsertErr != nil {
			return quickstartStep{
				ID:      "provider.configure",
				Title:   "Provider Configuration",
				Status:  quickstartStepStatusFail,
				Summary: "Failed to register provider API key secret reference with daemon.",
				Details: map[string]any{
					"provider":    providerName,
					"secret_name": secretName,
					"error":       doctorErrorDetails(upsertErr),
				},
				Remediation: []string{
					"Confirm daemon secret-reference APIs are reachable and rerun quickstart.",
				},
			}
		}
	}

	record, setErr := client.SetProvider(ctx, transport.ProviderSetRequest{
		WorkspaceID:      input.WorkspaceID,
		Provider:         providerName,
		Endpoint:         input.Endpoint,
		APIKeySecretName: secretName,
	}, input.CorrelationIDBase+".set")
	if setErr != nil {
		return quickstartStep{
			ID:      "provider.configure",
			Title:   "Provider Configuration",
			Status:  quickstartStepStatusFail,
			Summary: "Daemon rejected provider configuration.",
			Details: map[string]any{
				"provider": providerName,
				"error":    doctorErrorDetails(setErr),
			},
			Remediation: quickstartProviderFailureRemediation(input.CommandHints, input.WorkspaceID, providerName, input.Endpoint, secretName),
		}
	}

	return quickstartStep{
		ID:      "provider.configure",
		Title:   "Provider Configuration",
		Status:  quickstartStepStatusPass,
		Summary: "Provider configuration applied.",
		Details: map[string]any{
			"workspace_id":        record.WorkspaceID,
			"provider":            record.Provider,
			"endpoint":            record.Endpoint,
			"api_key_secret_name": record.APIKeySecretName,
			"api_key_configured":  record.APIKeyConfigured,
			"updated_at":          record.UpdatedAt.Format(time.RFC3339Nano),
		},
	}
}
