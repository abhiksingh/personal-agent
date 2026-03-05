package helpers

import (
	"fmt"
	"regexp"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

var unsafeTokenPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)

func RequiredStringInput(input map[string]any, key string) (string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", fmt.Errorf("input key is required")
	}
	raw, ok := input[trimmedKey]
	if !ok {
		return "", fmt.Errorf("missing required input %s", trimmedKey)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("input %s must be a string", trimmedKey)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("input %s is required", trimmedKey)
	}
	return value, nil
}

func OptionalStringInput(input map[string]any, key string) (string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", fmt.Errorf("input key is required")
	}
	raw, ok := input[trimmedKey]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("input %s must be a string", trimmedKey)
	}
	return strings.TrimSpace(value), nil
}

func StableStepToken(execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) string {
	if token := strings.TrimSpace(execCtx.StepID); token != "" {
		return StableToken(token, "step")
	}
	if token := strings.TrimSpace(step.ID); token != "" {
		return StableToken(token, "step")
	}
	return StableToken(step.CapabilityKey, "step")
}

func StableToken(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	cleaned := unsafeTokenPattern.ReplaceAllString(trimmed, "")
	cleaned = strings.ToLower(cleaned)
	if cleaned == "" {
		return fallback
	}
	if len(cleaned) > 16 {
		return cleaned[:16]
	}
	return cleaned
}
