package endpointpolicy

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

func TestParseAndValidateAcceptsHTTPSPublicEndpoint(t *testing.T) {
	parsed, err := ParseAndValidate("https://api.openai.com/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected https endpoint to be accepted: %v", err)
	}
	if parsed.Host != "api.openai.com" {
		t.Fatalf("expected host api.openai.com, got %q", parsed.Host)
	}
}

func TestParseAndValidateAllowsHTTPLoopbackByDefault(t *testing.T) {
	parsed, err := ParseAndValidate("http://127.0.0.1:11434", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected loopback http endpoint to be accepted: %v", err)
	}
	if parsed.Host != "127.0.0.1:11434" {
		t.Fatalf("expected host 127.0.0.1:11434, got %q", parsed.Host)
	}
}

func TestParseAndValidateRejectsHTTPNonLoopbackByDefault(t *testing.T) {
	_, err := ParseAndValidate("http://api.openai.com/v1", Options{Service: "provider endpoint"})
	if err == nil {
		t.Fatalf("expected non-loopback http endpoint to be rejected")
	}
}

func TestParseAndValidateRejectsPrivateHostByDefault(t *testing.T) {
	_, err := ParseAndValidate("https://192.168.1.10/v1", Options{Service: "provider endpoint"})
	if err == nil {
		t.Fatalf("expected private host endpoint to be rejected")
	}
}

func TestParseAndValidateAllowsInsecureAndPrivateWithExplicitOptIns(t *testing.T) {
	t.Setenv(EnvAllowInsecureEndpoints, "1")
	t.Setenv(EnvAllowPrivateEndpoints, "1")

	parsed, err := ParseAndValidate("http://192.168.1.10/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
	if parsed.Host != "192.168.1.10" {
		t.Fatalf("expected host 192.168.1.10, got %q", parsed.Host)
	}
}

func TestParseAndValidateResolvedRejectsPrivateResolvedAddressByDefault(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.0.0.7")}, nil
	}

	parsed, err := ParseAndValidate("https://models.example.com/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	err = validateResolvedURLWithLookup(context.Background(), parsed, Options{Service: "provider endpoint"}, lookup)
	if err == nil {
		t.Fatalf("expected private resolved address to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "resolves to private") {
		t.Fatalf("expected resolved private rejection message, got %v", err)
	}
}

func TestParseAndValidateResolvedAllowsPublicResolvedAddress(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("198.51.100.27")}, nil
	}

	parsed, err := ParseAndValidate("https://models.example.com/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	if err := validateResolvedURLWithLookup(context.Background(), parsed, Options{Service: "provider endpoint"}, lookup); err != nil {
		t.Fatalf("expected public resolved address to be accepted: %v", err)
	}
}

func TestParseAndValidateResolvedAllowsPrivateResolvedAddressWithOptIn(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.0.0.7")}, nil
	}

	parsed, err := ParseAndValidate("https://models.example.com/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	if err := validateResolvedURLWithLookup(context.Background(), parsed, Options{
		Service:      "provider endpoint",
		AllowPrivate: true,
	}, lookup); err != nil {
		t.Fatalf("expected private resolved address with opt-in to be accepted: %v", err)
	}
}

func TestParseAndValidateResolvedRejectsLookupFailure(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return nil, errors.New("lookup failed")
	}

	parsed, err := ParseAndValidate("https://models.example.com/v1", Options{Service: "provider endpoint"})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	err = validateResolvedURLWithLookup(context.Background(), parsed, Options{Service: "provider endpoint"}, lookup)
	if err == nil {
		t.Fatalf("expected lookup failure to be returned")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "lookup failed") {
		t.Fatalf("expected lookup failure message, got %v", err)
	}
}

func TestValidateResolvedURLRejectsNilTarget(t *testing.T) {
	err := ValidateResolvedURL(context.Background(), nil, Options{Service: "provider endpoint"})
	if err == nil {
		t.Fatalf("expected nil target to be rejected")
	}
}

func TestValidateResolvedURLSkipsLookupForLoopbackHostnames(t *testing.T) {
	lookupCalled := false
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		lookupCalled = true
		return nil, nil
	}
	target, err := url.Parse("http://localhost:8088/health")
	if err != nil {
		t.Fatalf("parse loopback target: %v", err)
	}
	if err := validateResolvedURLWithLookup(context.Background(), target, Options{Service: "provider endpoint"}, lookup); err != nil {
		t.Fatalf("expected localhost target to pass without lookup: %v", err)
	}
	if lookupCalled {
		t.Fatalf("expected lookup to be skipped for loopback host")
	}
}

func TestValidateResolvedURLRejectsPrivateLiteralAddressByDefault(t *testing.T) {
	target, err := url.Parse("https://10.0.0.20/v1")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	err = ValidateResolvedURL(context.Background(), target, Options{Service: "provider endpoint"})
	if err == nil {
		t.Fatalf("expected private literal address to be rejected")
	}
}
