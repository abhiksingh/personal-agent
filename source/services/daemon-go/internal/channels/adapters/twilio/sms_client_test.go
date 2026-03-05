package twilio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSMSSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "token" {
			t.Fatalf("unexpected basic auth: %v %s %s", ok, user, pass)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("From"); got != "+15550001111" {
			t.Fatalf("expected From number, got %s", got)
		}
		if got := r.Form.Get("To"); got != "+15550002222" {
			t.Fatalf("expected To number, got %s", got)
		}
		if got := r.Form.Get("Body"); got != "Hello from test" {
			t.Fatalf("expected Body, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"SM123","account_sid":"AC123","status":"queued","from":"+15550001111","to":"+15550002222"}`))
	}))
	defer server.Close()

	result, err := SendSMS(context.Background(), http.DefaultClient, SMSAPIRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err != nil {
		t.Fatalf("expected send success: %v", err)
	}
	if result.MessageSID != "SM123" {
		t.Fatalf("expected MessageSID SM123, got %s", result.MessageSID)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestSendSMSFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Authenticate","error_message":"invalid auth token"}`))
	}))
	defer server.Close()

	result, err := SendSMS(context.Background(), http.DefaultClient, SMSAPIRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "wrong",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err == nil {
		t.Fatalf("expected send failure")
	}
	if !strings.Contains(err.Error(), "invalid auth token") {
		t.Fatalf("expected auth token error, got %v", err)
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", result.StatusCode)
	}
}

func TestSendSMSRequiresHTTPClient(t *testing.T) {
	_, err := SendSMS(context.Background(), nil, SMSAPIRequest{
		Endpoint:   "https://api.twilio.com",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err == nil || err.Error() != "http client is required" {
		t.Fatalf("expected missing http client error, got %v", err)
	}
}

func TestSendSMSRejectsInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	_, err := SendSMS(context.Background(), http.DefaultClient, SMSAPIRequest{
		Endpoint:   "http://api.twilio.com",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err == nil {
		t.Fatalf("expected insecure non-loopback endpoint to be rejected")
	}
}

func TestSendSMSRejectsPrivateEndpointByDefault(t *testing.T) {
	_, err := SendSMS(context.Background(), http.DefaultClient, SMSAPIRequest{
		Endpoint:   "https://10.0.0.20",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err == nil {
		t.Fatalf("expected private endpoint to be rejected")
	}
}

func TestSendSMSAllowsInsecurePrivateEndpointWithExplicitOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"SM123","account_sid":"AC123","status":"queued","from":"+15550001111","to":"+15550002222"}`))
	}))
	defer server.Close()

	_, err := SendSMS(context.Background(), server.Client(), SMSAPIRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
}

func TestSendSMSRejectsOversizedProviderResponse(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(twilioProviderResponseBodyLimitBytes+128))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oversizedBody))
	}))
	defer server.Close()

	result, err := SendSMS(context.Background(), server.Client(), SMSAPIRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550001111",
		To:         "+15550002222",
		Body:       "Hello from test",
	})
	if err == nil {
		t.Fatalf("expected oversized provider response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
	if len(result.RawBody) > int(twilioProviderResponseBodyLimitBytes) {
		t.Fatalf("expected raw body to be capped at limit %d, got %d", twilioProviderResponseBodyLimitBytes, len(result.RawBody))
	}
}
