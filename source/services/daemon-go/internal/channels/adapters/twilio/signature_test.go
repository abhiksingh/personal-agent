package twilio

import (
	"errors"
	"testing"
)

func TestValidateRequestSignatureSuccess(t *testing.T) {
	params := map[string]string{
		"Body":       "Hello from webhook",
		"From":       "+15555550123",
		"MessageSid": "SM123",
		"To":         "+15555550999",
	}
	signature, err := ComputeRequestSignature("auth-token", "https://agent.local/webhook/personal-agent/v1/connector/twilio/sms", params)
	if err != nil {
		t.Fatalf("compute signature: %v", err)
	}

	if err := ValidateRequestSignature("auth-token", "https://agent.local/webhook/personal-agent/v1/connector/twilio/sms", params, signature); err != nil {
		t.Fatalf("expected signature validation success: %v", err)
	}
}

func TestValidateRequestSignatureMismatch(t *testing.T) {
	params := map[string]string{
		"Body":       "Hello from webhook",
		"From":       "+15555550123",
		"MessageSid": "SM123",
		"To":         "+15555550999",
	}
	err := ValidateRequestSignature("auth-token", "https://agent.local/webhook/personal-agent/v1/connector/twilio/sms", params, "bad-signature")
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}
