package daemonruntime

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
)

func TestValidateTwilioWebhookReplayTargetAllowsLoopbackByDefault(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(context.Background(), "http://127.0.0.1:8088/personalagent/v1/connector/twilio/sms", "")
	if err != nil {
		t.Fatalf("expected loopback replay target to be allowed: %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetRejectsPrivateIPByDefault(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(context.Background(), "http://10.0.0.20:8088/personalagent/v1/connector/twilio/sms", "")
	if err == nil {
		t.Fatalf("expected private replay target to be rejected")
	}
}

func TestValidateTwilioWebhookReplayTargetRejectsMetadataByDefault(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(context.Background(), "http://169.254.169.254/latest/meta-data", "")
	if err == nil {
		t.Fatalf("expected metadata replay target to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "metadata") {
		t.Fatalf("expected metadata rejection message, got %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetAllowsPrivateIPWhenAllowlisted(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(
		context.Background(),
		"http://10.0.0.20:8088/personalagent/v1/connector/twilio/sms",
		"10.0.0.20",
	)
	if err != nil {
		t.Fatalf("expected allowlisted private replay target to be accepted: %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetAllowsCIDRAllowlist(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(
		context.Background(),
		"http://10.11.12.13:8088/personalagent/v1/connector/twilio/sms",
		"10.0.0.0/8",
	)
	if err != nil {
		t.Fatalf("expected CIDR-allowlisted replay target to be accepted: %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetRejectsUnsupportedScheme(t *testing.T) {
	err := validateTwilioWebhookReplayTarget(context.Background(), "ftp://10.0.0.20/resource", "")
	if err == nil {
		t.Fatalf("expected unsupported replay target scheme to be rejected")
	}
}

func TestValidateTwilioWebhookReplayTargetRejectsHostnameResolvingPrivateAddress(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.0.0.20")}, nil
	}
	err := validateTwilioWebhookReplayTargetWithLookup(
		context.Background(),
		"https://twilio-replay.example/personalagent/v1/connector/twilio/sms",
		"",
		lookup,
	)
	if err == nil {
		t.Fatalf("expected hostname resolving to private address to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "resolves to private") {
		t.Fatalf("expected private resolution rejection message, got %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetAllowsHostnameWithPublicResolution(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("198.51.100.22")}, nil
	}
	err := validateTwilioWebhookReplayTargetWithLookup(
		context.Background(),
		"https://twilio-replay.example/personalagent/v1/connector/twilio/sms",
		"",
		lookup,
	)
	if err != nil {
		t.Fatalf("expected public hostname resolution to be allowed: %v", err)
	}
}

func TestValidateTwilioWebhookReplayTargetRejectsLookupFailure(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return nil, errors.New("lookup failed")
	}
	err := validateTwilioWebhookReplayTargetWithLookup(
		context.Background(),
		"https://twilio-replay.example/personalagent/v1/connector/twilio/sms",
		"",
		lookup,
	)
	if err == nil {
		t.Fatalf("expected lookup failure to be returned")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "lookup failed") {
		t.Fatalf("expected lookup failure detail, got %v", err)
	}
}
