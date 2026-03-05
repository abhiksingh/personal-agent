package cliapp

import (
	"fmt"
	"strings"

	"personalagent/runtime/internal/providerconfig"
)

func (h quickstartCommandHints) daemonStartCommand() string {
	return fmt.Sprintf(
		"personal-agent-daemon --listen-mode %s --listen-address %s --auth-token-file %s",
		quickstartShellQuote(h.ListenerMode),
		quickstartShellQuote(h.Address),
		quickstartShellQuote(h.TokenFilePath),
	)
}

func (h quickstartCommandHints) profileUseCommand() string {
	if h.ProfileActive || strings.TrimSpace(h.ProfileName) == "" {
		return ""
	}
	return fmt.Sprintf("personal-agent profile use --name %s", quickstartShellQuote(h.ProfileName))
}

func (h quickstartCommandHints) smokeCommand() string {
	return fmt.Sprintf("%s smoke", h.cliPrefix())
}

func (h quickstartCommandHints) cliPrefix() string {
	if h.ProfileActive {
		return "personal-agent"
	}
	return fmt.Sprintf(
		"personal-agent --mode %s --address %s --auth-token-file %s",
		quickstartShellQuote(h.ListenerMode),
		quickstartShellQuote(h.Address),
		quickstartShellQuote(h.TokenFilePath),
	)
}

func quickstartProviderFailureRemediation(hints quickstartCommandHints, workspaceID, providerName, endpoint, secretName string) []string {
	steps := []string{}
	if profileUse := hints.profileUseCommand(); profileUse != "" {
		steps = append(steps, fmt.Sprintf("Activate quickstart profile with: %s", profileUse))
	}
	checkCommand := fmt.Sprintf(
		"%s provider check --workspace %s --provider %s",
		hints.cliPrefix(),
		quickstartShellQuote(workspaceID),
		quickstartShellQuote(providerName),
	)
	setCommandParts := []string{
		fmt.Sprintf("%s provider set", hints.cliPrefix()),
		fmt.Sprintf("--workspace %s", quickstartShellQuote(workspaceID)),
		fmt.Sprintf("--provider %s", quickstartShellQuote(providerName)),
	}
	if strings.TrimSpace(endpoint) != "" {
		setCommandParts = append(setCommandParts, fmt.Sprintf("--endpoint %s", quickstartShellQuote(endpoint)))
	}
	if strings.TrimSpace(secretName) != "" {
		setCommandParts = append(setCommandParts, fmt.Sprintf("--api-key-secret %s", quickstartShellQuote(secretName)))
	}
	steps = append(steps,
		fmt.Sprintf("Validate provider connectivity with: %s", checkCommand),
		fmt.Sprintf("Reapply provider configuration with: %s", strings.Join(setCommandParts, " ")),
	)
	return steps
}

func quickstartModelRouteFailureRemediation(hints quickstartCommandHints, workspaceID, providerName, taskClass, modelKey string) []string {
	steps := []string{}
	if profileUse := hints.profileUseCommand(); profileUse != "" {
		steps = append(steps, fmt.Sprintf("Activate quickstart profile with: %s", profileUse))
	}
	listCommand := fmt.Sprintf(
		"%s model list --workspace %s --provider %s",
		hints.cliPrefix(),
		quickstartShellQuote(workspaceID),
		quickstartShellQuote(providerName),
	)
	selectCommand := fmt.Sprintf(
		"%s model select --workspace %s --task-class %s --provider %s --model %s",
		hints.cliPrefix(),
		quickstartShellQuote(workspaceID),
		quickstartShellQuote(taskClass),
		quickstartShellQuote(providerName),
		quickstartShellQuote(modelKey),
	)
	steps = append(steps,
		fmt.Sprintf("List available models with: %s", listCommand),
		fmt.Sprintf("Set the model route with: %s", selectCommand),
	)
	return steps
}

func quickstartShellQuote(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(trimmed, "'", "'\"'\"'") + "'"
}

func quickstartDefaultAPIKeySecretName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providerconfig.ProviderOpenAI:
		return "OPENAI_API_KEY"
	case providerconfig.ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case providerconfig.ProviderGoogle:
		return "GOOGLE_API_KEY"
	default:
		return ""
	}
}

func quickstartDefaultModelKey(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providerconfig.ProviderOpenAI:
		return "gpt-4.1-mini"
	case providerconfig.ProviderAnthropic:
		return "claude-3-5-sonnet-latest"
	case providerconfig.ProviderGoogle:
		return "gemini-2.0-flash"
	case providerconfig.ProviderOllama:
		return "llama3.2"
	default:
		return ""
	}
}
