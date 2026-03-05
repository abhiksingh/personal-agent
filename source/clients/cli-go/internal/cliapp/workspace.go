package cliapp

import (
	"os"
	"strings"

	"personalagent/runtime/internal/workspaceid"
)

const (
	cliWorkspaceEnvKey = "PERSONAL_AGENT_WORKSPACE_ID"
)

func normalizeWorkspace(workspaceID string) string {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return defaultWorkspaceFromClientContext()
	}
	return trimmed
}

func defaultWorkspaceFromClientContext() string {
	workspaceFromEnv := strings.TrimSpace(os.Getenv(cliWorkspaceEnvKey))
	if workspaceFromEnv != "" {
		return workspaceid.Normalize(workspaceFromEnv)
	}
	return workspaceid.CanonicalDefault
}
