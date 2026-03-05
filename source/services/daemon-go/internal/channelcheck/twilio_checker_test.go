package channelcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckTwilioSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123.json" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatalf("expected basic auth")
		}
		if username != "AC123" || password != "secret" {
			t.Fatalf("unexpected basic auth credentials")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result, err := CheckTwilio(context.Background(), http.DefaultClient, TwilioRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "secret",
	})
	if err != nil {
		t.Fatalf("expected successful twilio check: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckTwilioFailsOnUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	result, err := CheckTwilio(context.Background(), http.DefaultClient, TwilioRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "bad",
	})
	if err == nil {
		t.Fatalf("expected unauthorized check to fail")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", result.StatusCode)
	}
}

func TestCheckTwilioRequiresHTTPClient(t *testing.T) {
	_, err := CheckTwilio(context.Background(), nil, TwilioRequest{
		Endpoint:   "https://api.twilio.com",
		AccountSID: "AC123",
		AuthToken:  "secret",
	})
	if err == nil || err.Error() != "http client is required" {
		t.Fatalf("expected missing http client error, got %v", err)
	}
}

func TestCheckTwilioRejectsInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	_, err := CheckTwilio(context.Background(), http.DefaultClient, TwilioRequest{
		Endpoint:   "http://api.twilio.com",
		AccountSID: "AC123",
		AuthToken:  "secret",
	})
	if err == nil {
		t.Fatalf("expected insecure non-loopback endpoint to be rejected")
	}
}

func TestCheckTwilioRejectsPrivateEndpointByDefault(t *testing.T) {
	_, err := CheckTwilio(context.Background(), http.DefaultClient, TwilioRequest{
		Endpoint:   "https://10.0.0.20",
		AccountSID: "AC123",
		AuthToken:  "secret",
	})
	if err == nil {
		t.Fatalf("expected private endpoint to be rejected")
	}
}

func TestCheckTwilioAllowsInsecurePrivateEndpointWithExplicitOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := CheckTwilio(context.Background(), server.Client(), TwilioRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "secret",
	})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
}
