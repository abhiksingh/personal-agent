package cliapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultWebhookAPIVersion   = "v1"
	defaultProjectNameFallback = "personalagent"
	projectNameEnvKey          = "PA_PROJECT_NAME"
)

func defaultTwilioWebhookSMSPath() string {
	return fmt.Sprintf("/%s/%s/connector/twilio/sms", resolveWebhookProjectName(), defaultWebhookAPIVersion)
}

func defaultTwilioWebhookVoicePath() string {
	return fmt.Sprintf("/%s/%s/connector/twilio/voice", resolveWebhookProjectName(), defaultWebhookAPIVersion)
}

func resolveWebhookProjectName() string {
	if override := strings.TrimSpace(os.Getenv(projectNameEnvKey)); override != "" {
		return normalizeWebhookProjectName(override)
	}
	executablePath, err := os.Executable()
	if err == nil && strings.TrimSpace(executablePath) != "" {
		return normalizeWebhookProjectName(filepath.Base(executablePath))
	}
	return normalizeWebhookProjectName(filepath.Base(os.Args[0]))
}

func normalizeWebhookProjectName(raw string) string {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	if candidate == "" {
		return defaultProjectNameFallback
	}
	candidate = strings.TrimSuffix(candidate, filepath.Ext(candidate))
	candidate = strings.TrimSuffix(candidate, "-daemon")
	candidate = strings.TrimSuffix(candidate, "_daemon")
	candidate = strings.TrimSuffix(candidate, "daemon")

	builder := strings.Builder{}
	for _, r := range candidate {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		}
	}
	normalized := strings.TrimSpace(builder.String())
	if normalized == "" {
		return defaultProjectNameFallback
	}
	return normalized
}

func normalizeWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}
