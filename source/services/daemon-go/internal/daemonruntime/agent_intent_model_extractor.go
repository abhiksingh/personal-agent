package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"personalagent/runtime/internal/chatruntime"
	"personalagent/runtime/internal/core/service/agentexec"
	"personalagent/runtime/internal/modelpolicy"
)

const intentExtractionSystemPrompt = `You classify a user request into one workflow.
Allowed workflows: mail, calendar, messages, browser, finder.
Return JSON only with this exact schema:
{"workflow":"mail|calendar|messages|browser|finder","target_url":"","target_path":"","target_query":"","message_channel":"","message_recipient":"","message_body":"","confidence":0.0,"rationale":""}
Rules:
- browser requires an http/https target_url when available.
- finder should include absolute target_path when available, otherwise include target_query.
- messages should include message_channel (imessage|sms), message_recipient, and message_body when available.
- Use confidence between 0.0 and 1.0.
- Do not include markdown or extra keys.`

type daemonIntentModelExtractor struct {
	container        *ServiceContainer
	chatStreamClient *http.Client
}

var _ agentexec.ModelIntentExtractor = (*daemonIntentModelExtractor)(nil)

func newDaemonIntentModelExtractor(container *ServiceContainer) *daemonIntentModelExtractor {
	return &daemonIntentModelExtractor{
		container:        container,
		chatStreamClient: newDaemonRuntimeHTTPClient(defaultProviderChatHTTPTimeout),
	}
}

func (e *daemonIntentModelExtractor) ExtractIntent(ctx context.Context, workspaceID string, request string) (agentexec.ModelIntentCandidate, error) {
	if e == nil || e.container == nil {
		return agentexec.ModelIntentCandidate{}, fmt.Errorf("service container is required")
	}
	workspace := normalizeWorkspaceID(workspaceID)
	trimmedRequest := strings.TrimSpace(request)
	if trimmedRequest == "" {
		return agentexec.ModelIntentCandidate{}, fmt.Errorf("request is required")
	}

	routeResolver := NewProviderModelChatService(e.container)
	route, err := routeResolver.resolveModelRoute(ctx, workspace, modelpolicy.TaskClassDefault)
	if err != nil {
		return agentexec.ModelIntentCandidate{}, err
	}

	providerConfig, err := e.container.ProviderConfigStore.Get(ctx, workspace, route.Provider)
	if err != nil {
		return agentexec.ModelIntentCandidate{}, fmt.Errorf("load provider config for %s: %w", route.Provider, err)
	}

	apiKey := ""
	if secretName := strings.TrimSpace(providerConfig.APIKeySecretName); secretName != "" {
		_, secretValue, resolveErr := e.container.SecretResolver.ResolveSecret(ctx, workspace, secretName)
		if resolveErr != nil {
			return agentexec.ModelIntentCandidate{}, fmt.Errorf("resolve secret %q: %w", secretName, resolveErr)
		}
		apiKey = secretValue
	}

	builder := &strings.Builder{}
	if err := chatruntime.StreamAssistantResponse(ctx, e.chatHTTPClient(), chatruntime.StreamRequest{
		Provider: route.Provider,
		Endpoint: providerConfig.Endpoint,
		ModelKey: route.ModelKey,
		APIKey:   apiKey,
		Messages: []chatruntime.Message{
			{Role: "system", Content: intentExtractionSystemPrompt},
			{Role: "user", Content: trimmedRequest},
		},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	}); err != nil {
		return agentexec.ModelIntentCandidate{}, err
	}

	return parseIntentExtractionPayload(builder.String())
}

func (e *daemonIntentModelExtractor) chatHTTPClient() *http.Client {
	if e != nil && e.chatStreamClient != nil {
		return e.chatStreamClient
	}
	return newDaemonRuntimeHTTPClient(defaultProviderChatHTTPTimeout)
}

func parseIntentExtractionPayload(raw string) (agentexec.ModelIntentCandidate, error) {
	jsonPayload, err := extractJSONObject(raw)
	if err != nil {
		return agentexec.ModelIntentCandidate{}, err
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonPayload), &decoded); err != nil {
		return agentexec.ModelIntentCandidate{}, fmt.Errorf("decode model intent payload: %w", err)
	}

	return agentexec.ModelIntentCandidate{
		Workflow:         strings.TrimSpace(asString(decoded["workflow"])),
		TargetURL:        strings.TrimSpace(asString(decoded["target_url"])),
		TargetPath:       strings.TrimSpace(asString(decoded["target_path"])),
		TargetQuery:      strings.TrimSpace(asString(decoded["target_query"])),
		MessageChannel:   strings.TrimSpace(asString(decoded["message_channel"])),
		MessageRecipient: strings.TrimSpace(asString(decoded["message_recipient"])),
		MessageBody:      strings.TrimSpace(asString(decoded["message_body"])),
		Confidence:       asFloat(decoded["confidence"]),
		Rationale:        strings.TrimSpace(asString(decoded["rationale"])),
	}, nil
}

func extractJSONObject(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("model intent output is empty")
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("model intent output did not include JSON object")
	}
	return trimmed[start : end+1], nil
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func asFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
