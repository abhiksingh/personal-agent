package modelpolicy

import (
	"strings"

	"personalagent/runtime/internal/providerconfig"
)

const TaskClassDefault = "default"

type CatalogModel struct {
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
	DisplayName string `json:"display_name"`
}

var defaultCatalog = []CatalogModel{
	{
		Provider:    providerconfig.ProviderOpenAI,
		ModelKey:    "gpt-4.1",
		DisplayName: "GPT-4.1",
	},
	{
		Provider:    providerconfig.ProviderOpenAI,
		ModelKey:    "gpt-4.1-mini",
		DisplayName: "GPT-4.1 Mini",
	},
	{
		Provider:    providerconfig.ProviderAnthropic,
		ModelKey:    "claude-3-5-sonnet-latest",
		DisplayName: "Claude 3.5 Sonnet",
	},
	{
		Provider:    providerconfig.ProviderAnthropic,
		ModelKey:    "claude-3-5-haiku-latest",
		DisplayName: "Claude 3.5 Haiku",
	},
	{
		Provider:    providerconfig.ProviderGoogle,
		ModelKey:    "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash",
	},
	{
		Provider:    providerconfig.ProviderGoogle,
		ModelKey:    "gemini-2.0-flash-lite",
		DisplayName: "Gemini 2.0 Flash-Lite",
	},
	{
		Provider:    providerconfig.ProviderOllama,
		ModelKey:    "llama3.2",
		DisplayName: "Llama 3.2",
	},
	{
		Provider:    providerconfig.ProviderOllama,
		ModelKey:    "mistral",
		DisplayName: "Mistral",
	},
}

func DefaultCatalog() []CatalogModel {
	items := make([]CatalogModel, len(defaultCatalog))
	copy(items, defaultCatalog)
	return items
}

func IsSupportedModel(provider string, modelKey string) bool {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	normalizedModel := strings.TrimSpace(modelKey)
	for _, item := range defaultCatalog {
		if item.Provider == normalizedProvider && item.ModelKey == normalizedModel {
			return true
		}
	}
	return false
}
