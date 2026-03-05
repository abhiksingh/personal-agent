package daemonruntime

import (
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func catalogLookupKey(provider string, modelKey string) string {
	return strings.TrimSpace(provider) + "::" + strings.TrimSpace(modelKey)
}

func providerPriority(provider string) int {
	switch provider {
	case providerconfig.ProviderOpenAI:
		return 0
	case providerconfig.ProviderAnthropic:
		return 1
	case providerconfig.ProviderGoogle:
		return 2
	case providerconfig.ProviderOllama:
		return 3
	default:
		return 99
	}
}

func normalizeTaskClass(taskClass string) string {
	normalized := strings.ToLower(strings.TrimSpace(taskClass))
	if normalized == "" {
		return modelpolicy.TaskClassDefault
	}
	return normalized
}

func providerConfigRecord(config providerconfig.Config) transport.ProviderConfigRecord {
	return transport.ProviderConfigRecord{
		WorkspaceID:      config.WorkspaceID,
		Provider:         config.Provider,
		Endpoint:         config.Endpoint,
		APIKeySecretName: config.APIKeySecretName,
		APIKeyConfigured: config.APIKeyConfigured,
		UpdatedAt:        config.UpdatedAt,
	}
}

func modelCatalogEntryRecord(entry modelpolicy.CatalogEntry) transport.ModelCatalogEntryRecord {
	return transport.ModelCatalogEntryRecord{
		WorkspaceID: entry.WorkspaceID,
		Provider:    entry.Provider,
		ModelKey:    entry.ModelKey,
		Enabled:     entry.Enabled,
		UpdatedAt:   entry.UpdatedAt,
	}
}

func modelRoutingPolicyRecord(policy modelpolicy.RoutingPolicy) transport.ModelRoutingPolicyRecord {
	return transport.ModelRoutingPolicyRecord{
		WorkspaceID: policy.WorkspaceID,
		TaskClass:   policy.TaskClass,
		Provider:    policy.Provider,
		ModelKey:    policy.ModelKey,
		UpdatedAt:   policy.UpdatedAt,
	}
}
