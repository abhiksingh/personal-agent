package cliapp

import (
	"sort"
	"strings"
)

func buildCompletionCommandIndex(schema cliSchemaDocument) completionCommandIndex {
	index := completionCommandIndex{
		rootCommands:      make([]string, 0, len(schema.Commands)),
		globalFlags:       make([]string, 0, len(schema.GlobalFlags)),
		knownPaths:        make([]string, 0, len(schema.Commands)),
		subcommandsByPath: make(map[string][]string, len(schema.Commands)),
		flagsByPath:       make(map[string][]string, len(schema.Commands)),
	}
	for _, flagSchema := range schema.GlobalFlags {
		name := strings.TrimSpace(flagSchema.Name)
		if name == "" {
			continue
		}
		index.globalFlags = append(index.globalFlags, name)
	}
	index.globalFlags = sanitizeCompletionTokens(index.globalFlags)

	for _, command := range schema.Commands {
		commandName := strings.TrimSpace(command.Name)
		if commandName == "" {
			continue
		}
		index.rootCommands = append(index.rootCommands, commandName)
		collectCompletionCommand(&index, nil, command)
	}
	index.rootCommands = sanitizeCompletionTokens(index.rootCommands)
	index.knownPaths = sanitizeCompletionTokens(index.knownPaths)
	for pathKey, subcommands := range index.subcommandsByPath {
		index.subcommandsByPath[pathKey] = sanitizeCompletionTokens(subcommands)
	}
	for pathKey, flags := range index.flagsByPath {
		index.flagsByPath[pathKey] = sanitizeCompletionTokens(flags)
	}

	return index
}

func collectCompletionCommand(index *completionCommandIndex, parentPath []string, command cliCommandSchema) {
	commandName := strings.TrimSpace(command.Name)
	if commandName == "" {
		return
	}
	path := append(append([]string{}, parentPath...), commandName)
	pathKey := strings.Join(path, " ")
	index.knownPaths = append(index.knownPaths, pathKey)

	if len(command.Subcommands) > 0 {
		subcommands := index.subcommandsByPath[pathKey]
		for _, subcommand := range command.Subcommands {
			subcommandName := strings.TrimSpace(subcommand.Name)
			if subcommandName == "" {
				continue
			}
			subcommands = append(subcommands, subcommandName)
		}
		index.subcommandsByPath[pathKey] = subcommands
	}

	flags := append([]string{}, expandCompletionFlagPatterns(command.RequiredFlags)...)
	flags = append(flags, completionContextualFlagHints(pathKey)...)
	if len(flags) > 0 {
		index.flagsByPath[pathKey] = append(index.flagsByPath[pathKey], flags...)
	}

	for _, subcommand := range command.Subcommands {
		collectCompletionCommand(index, path, subcommand)
	}
}

func sortedCompletionMapKeys(valuesByPath map[string][]string) []string {
	keys := make([]string, 0, len(valuesByPath))
	for pathKey := range valuesByPath {
		if strings.TrimSpace(pathKey) == "" {
			continue
		}
		keys = append(keys, pathKey)
	}
	sort.Strings(keys)
	return keys
}

func sanitizeCompletionTokens(tokens []string) []string {
	normalized := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func expandCompletionFlagPatterns(rawFlags []string) []string {
	expanded := make([]string, 0, len(rawFlags))
	for _, rawFlag := range rawFlags {
		for _, candidate := range strings.Split(strings.TrimSpace(rawFlag), "|") {
			trimmed := strings.TrimSpace(candidate)
			if strings.HasPrefix(trimmed, "--") {
				expanded = append(expanded, trimmed)
			}
		}
	}
	return expanded
}

func completionContextualFlagHints(pathKey string) []string {
	switch strings.TrimSpace(pathKey) {
	case "completion":
		return []string{"--shell"}
	case "task submit":
		return []string{"--description", "--requested-by", "--subject", "--task-class", "--title", "--workspace"}
	case "task status":
		return []string{"--task-id"}
	case "task cancel", "task retry", "task requeue":
		return []string{"--reason", "--run-id", "--task-id", "--workspace"}
	case "provider set":
		return []string{"--api-key-secret", "--clear-api-key", "--endpoint", "--provider", "--workspace"}
	case "provider list":
		return []string{"--workspace"}
	case "provider check":
		return []string{"--provider", "--workspace"}
	case "model list", "model discover":
		return []string{"--provider", "--workspace"}
	case "model add":
		return []string{"--enabled", "--model", "--provider", "--workspace"}
	case "model remove", "model enable", "model disable":
		return []string{"--model", "--provider", "--workspace"}
	case "model select":
		return []string{"--model", "--provider", "--task-class", "--workspace"}
	case "model policy", "model resolve":
		return []string{"--task-class", "--workspace"}
	case "comm send":
		return []string{"--connector-id", "--destination", "--event-id", "--imessage-failures", "--message", "--operation-id", "--sms-failures", "--source-channel", "--step-id", "--workspace"}
	case "comm attempts":
		return []string{"--operation-id", "--workspace"}
	case "comm policy set":
		return []string{"--endpoint-pattern", "--fallback-channels", "--is-default", "--primary-channel", "--retry-count", "--source-channel", "--workspace"}
	case "comm policy list":
		return []string{"--source-channel", "--workspace"}
	case "connector bridge setup", "connector bridge status":
		return []string{"--inbox-dir", "--workspace"}
	case "connector mail ingest", "connector mail handoff":
		return []string{"--body", "--from", "--inbox-dir", "--message-id", "--occurred-at", "--source-cursor", "--source-event-id", "--source-scope", "--subject", "--thread-ref", "--to", "--workspace"}
	case "connector calendar ingest", "connector calendar handoff":
		return []string{"--calendar-id", "--calendar-name", "--change-type", "--ends-at", "--event-uid", "--inbox-dir", "--location", "--notes", "--occurred-at", "--source-cursor", "--source-event-id", "--source-scope", "--starts-at", "--title", "--workspace"}
	case "connector browser ingest", "connector browser handoff":
		return []string{"--event-type", "--inbox-dir", "--occurred-at", "--page-title", "--page-url", "--payload", "--source-cursor", "--source-event-id", "--source-scope", "--tab-id", "--window-id", "--workspace"}
	case "connector cloudflared exec":
		return []string{"--arg", "--workspace"}
	case "connector cloudflared version":
		return []string{"--workspace"}
	case "connector twilio set":
		return []string{"--account-sid", "--account-sid-file", "--account-sid-secret", "--auth-token", "--auth-token-file", "--auth-token-secret", "--endpoint", "--sms-number", "--voice-number", "--workspace"}
	case "connector twilio get", "connector twilio check":
		return []string{"--workspace"}
	case "connector twilio sms-chat":
		return []string{"--interactive", "--message", "--operation-id", "--to", "--workspace"}
	case "connector twilio start-call":
		return []string{"--from", "--to", "--twiml-url", "--workspace"}
	case "connector twilio call-status":
		return []string{"--call-sid", "--limit", "--workspace"}
	case "connector twilio transcript":
		return []string{"--call-sid", "--limit", "--thread-id", "--workspace"}
	case "connector twilio ingest-sms":
		return []string{"--account-sid", "--body", "--from", "--message-sid", "--request-url", "--signature", "--skip-signature", "--to", "--workspace"}
	case "connector twilio ingest-voice":
		return []string{"--account-sid", "--call-sid", "--call-status", "--direction", "--from", "--provider-event-id", "--request-url", "--signature", "--skip-signature", "--to", "--transcript", "--transcript-assistant-emitted", "--transcript-direction", "--workspace"}
	case "connector twilio webhook serve":
		return []string{"--assistant-max-history", "--assistant-replies", "--assistant-reply-timeout", "--assistant-system", "--assistant-task-class", "--cloudflared-mode", "--cloudflared-startup-timeout", "--listen", "--run-for", "--signature-mode", "--sms-path", "--voice-fallback", "--voice-greeting", "--voice-path", "--voice-response-mode", "--workspace"}
	case "connector twilio webhook replay":
		return []string{"--base-url", "--fixture", "--http-timeout", "--request-url", "--signature-mode", "--sms-path", "--voice-path", "--workspace"}
	case "identity workspaces":
		return []string{"--include-inactive"}
	case "identity principals":
		return []string{"--include-inactive", "--workspace"}
	case "identity context":
		return []string{"--workspace"}
	case "identity select-workspace":
		return []string{"--principal", "--source", "--workspace"}
	case "identity bootstrap":
		return []string{"--actor-type", "--display-name", "--handle-channel", "--handle-primary", "--handle-value", "--principal", "--principal-status", "--source", "--workspace", "--workspace-name", "--workspace-status"}
	case "identity devices":
		return []string{"--cursor-created-at", "--cursor-id", "--device-type", "--limit", "--platform", "--user-id", "--workspace"}
	case "identity sessions":
		return []string{"--cursor-id", "--cursor-started-at", "--device-id", "--limit", "--session-health", "--user-id", "--workspace"}
	case "identity revoke-session":
		return []string{"--session-id", "--workspace"}
	default:
		return nil
	}
}
