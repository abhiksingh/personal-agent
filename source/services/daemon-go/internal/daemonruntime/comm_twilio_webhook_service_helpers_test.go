package daemonruntime

import (
	"net/http/httptest"
	"strings"
	"testing"

	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
)

func TestResolveDaemonTwilioSignatureRequestURLUsesCanonicalByDefault(t *testing.T) {
	request := httptest.NewRequest("POST", "http://webhook.example.local/personalagent/v1/connector/twilio/sms?foo=1", nil)

	resolved := resolveDaemonTwilioSignatureRequestURL(request)
	expected := "http://webhook.example.local/personalagent/v1/connector/twilio/sms?foo=1"
	if resolved != expected {
		t.Fatalf("expected canonical request url %q, got %q", expected, resolved)
	}
}

func TestResolveDaemonTwilioSignatureRequestURLIgnoresSpoofedOverrideWithoutReplayMarker(t *testing.T) {
	request := httptest.NewRequest("POST", "http://webhook.example.local/personalagent/v1/connector/twilio/sms", nil)
	request.Header.Set(twilioWebhookRequestURLOverrideHeader, "https://evil.example.com/hijack")
	request.RemoteAddr = "198.51.100.20:40000"

	resolved := resolveDaemonTwilioSignatureRequestURL(request)
	if resolved != "http://webhook.example.local/personalagent/v1/connector/twilio/sms" {
		t.Fatalf("expected spoofed override to be ignored, got %q", resolved)
	}
}

func TestResolveDaemonTwilioSignatureRequestURLAllowsLoopbackReplayOverride(t *testing.T) {
	request := httptest.NewRequest("POST", "http://127.0.0.1:8088/personalagent/v1/connector/twilio/sms", nil)
	request.Header.Set(twilioWebhookRequestURLOverrideHeader, "https://public.example.com/webhook/sms")
	request.Header.Set(twilioWebhookReplayMarkerHeader, "1")
	request.RemoteAddr = "127.0.0.1:41000"

	resolved := resolveDaemonTwilioSignatureRequestURL(request)
	if resolved != "https://public.example.com/webhook/sms" {
		t.Fatalf("expected replay override to be honored for loopback-marked replay, got %q", resolved)
	}
}

func TestSpoofedTwilioRequestURLOverrideCannotDriveStrictSignatureValidation(t *testing.T) {
	params := map[string]string{
		"From": "+15555550100",
		"To":   "+15555550101",
		"Body": "hello",
	}

	request := httptest.NewRequest("POST", "http://webhook.example.local/personalagent/v1/connector/twilio/sms", nil)
	request.Header.Set(twilioWebhookRequestURLOverrideHeader, "https://evil.example.com/hijack")
	request.RemoteAddr = "198.51.100.21:42000"

	resolved := resolveDaemonTwilioSignatureRequestURL(request)
	spoofedSignature, err := twilioadapter.ComputeRequestSignature("auth-token", "https://evil.example.com/hijack", params)
	if err != nil {
		t.Fatalf("compute spoofed signature: %v", err)
	}
	err = twilioadapter.ValidateRequestSignature("auth-token", resolved, params, spoofedSignature)
	if err == nil {
		t.Fatalf("expected strict validation failure for spoofed request-url override signature")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "invalid twilio signature") {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}
