package cliapp

import (
	"testing"

	"personalagent/runtime/internal/workspaceid"
)

func TestNormalizeWorkspaceUsesEnvWorkspaceWhenFlagOmitted(t *testing.T) {
	t.Setenv(cliWorkspaceEnvKey, "ws-env")

	if got := normalizeWorkspace(""); got != "ws-env" {
		t.Fatalf("expected env workspace ws-env, got %q", got)
	}
}

func TestNormalizeWorkspacePreservesExplicitDefaultWorkspace(t *testing.T) {
	t.Setenv(cliWorkspaceEnvKey, "ws-env")

	if got := normalizeWorkspace("default"); got != "default" {
		t.Fatalf("expected explicit workspace default, got %q", got)
	}
}

func TestNormalizeWorkspaceFallsBackToCLIWorkspaceDefault(t *testing.T) {
	t.Setenv(cliWorkspaceEnvKey, "")

	if got := normalizeWorkspace(""); got != workspaceid.CanonicalDefault {
		t.Fatalf("expected fallback workspace %q, got %q", workspaceid.CanonicalDefault, got)
	}
}

func TestNormalizeWorkspacePreservesExplicitWorkspace(t *testing.T) {
	t.Setenv(cliWorkspaceEnvKey, "ws-env")

	if got := normalizeWorkspace("ws-explicit"); got != "ws-explicit" {
		t.Fatalf("expected explicit workspace ws-explicit, got %q", got)
	}
}

func TestNormalizeWorkspaceUsesExplicitEnvWorkspaceValue(t *testing.T) {
	t.Setenv(cliWorkspaceEnvKey, "default")

	if got := normalizeWorkspace(""); got != "default" {
		t.Fatalf("expected env workspace default, got %q", got)
	}
}
