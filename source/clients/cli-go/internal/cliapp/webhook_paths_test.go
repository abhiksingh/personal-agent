package cliapp

import "testing"

func TestNormalizeWebhookProjectName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty fallback", input: "", want: defaultProjectNameFallback},
		{name: "daemon suffix stripped", input: "Personal-Agent-Daemon", want: "personalagent"},
		{name: "underscores removed", input: "personal_agent", want: "personalagent"},
		{name: "extension stripped", input: "personal-agent.exe", want: "personalagent"},
		{name: "non-alnum removed", input: "pa@prod#1", want: "paprod1"},
		{name: "all invalid fallback", input: "@@@", want: defaultProjectNameFallback},
	}

	for _, testCase := range tests {
		if got := normalizeWebhookProjectName(testCase.input); got != testCase.want {
			t.Fatalf("%s: normalizeWebhookProjectName(%q) = %q, want %q", testCase.name, testCase.input, got, testCase.want)
		}
	}
}

func TestResolveWebhookProjectNameUsesEnvOverride(t *testing.T) {
	t.Setenv(projectNameEnvKey, "Personal-Agent-Daemon")
	if got := resolveWebhookProjectName(); got != "personalagent" {
		t.Fatalf("resolveWebhookProjectName() = %q, want %q", got, "personalagent")
	}
}

func TestDefaultTwilioWebhookPathsUseResolvedProjectName(t *testing.T) {
	t.Setenv(projectNameEnvKey, "Personal-Agent-Daemon")

	if got := defaultTwilioWebhookSMSPath(); got != "/personalagent/v1/connector/twilio/sms" {
		t.Fatalf("defaultTwilioWebhookSMSPath() = %q", got)
	}
	if got := defaultTwilioWebhookVoicePath(); got != "/personalagent/v1/connector/twilio/voice" {
		t.Fatalf("defaultTwilioWebhookVoicePath() = %q", got)
	}
}
