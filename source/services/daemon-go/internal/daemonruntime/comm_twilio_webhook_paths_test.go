package daemonruntime

import "testing"

func TestNormalizeDaemonTwilioWebhookProjectName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty fallback", input: "", want: twilioWebhookDefaultProjectName},
		{name: "daemon suffix stripped", input: "Personal-Agent-Daemon", want: "personalagent"},
		{name: "underscores removed", input: "personal_agent", want: "personalagent"},
		{name: "extension stripped", input: "personal-agent.exe", want: "personalagent"},
		{name: "non-alnum removed", input: "pa@prod#1", want: "paprod1"},
		{name: "all invalid fallback", input: "@@@", want: twilioWebhookDefaultProjectName},
	}

	for _, testCase := range tests {
		if got := normalizeDaemonTwilioWebhookProjectName(testCase.input); got != testCase.want {
			t.Fatalf("%s: normalizeDaemonTwilioWebhookProjectName(%q) = %q, want %q", testCase.name, testCase.input, got, testCase.want)
		}
	}
}

func TestResolveDaemonTwilioWebhookProjectNameUsesEnvOverride(t *testing.T) {
	t.Setenv(twilioWebhookProjectNameEnvKey, "Personal-Agent-Daemon")
	if got := resolveDaemonTwilioWebhookProjectName(); got != "personalagent" {
		t.Fatalf("resolveDaemonTwilioWebhookProjectName() = %q, want %q", got, "personalagent")
	}
}

func TestDefaultDaemonTwilioWebhookPathsUseResolvedProjectName(t *testing.T) {
	t.Setenv(twilioWebhookProjectNameEnvKey, "Personal-Agent-Daemon")

	if got := defaultDaemonTwilioWebhookSMSPath(); got != "/personalagent/v1/connector/twilio/sms" {
		t.Fatalf("defaultDaemonTwilioWebhookSMSPath() = %q", got)
	}
	if got := defaultDaemonTwilioWebhookVoicePath(); got != "/personalagent/v1/connector/twilio/voice" {
		t.Fatalf("defaultDaemonTwilioWebhookVoicePath() = %q", got)
	}
}
