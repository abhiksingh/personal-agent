package daemonruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestReplayTwilioWebhookRejectsBlockedRedirectTarget(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://169.254.169.254/latest/meta-data", http.StatusFound)
	}))
	defer redirectServer.Close()

	service := &CommTwilioService{}
	_, err := service.ReplayTwilioWebhook(context.Background(), transport.TwilioWebhookReplayRequest{
		WorkspaceID:   "workspace-test",
		Kind:          "sms",
		BaseURL:       redirectServer.URL,
		SignatureMode: twilioWebhookSignatureModeBypass,
		Params: map[string]string{
			"From": "+15555550100",
			"To":   "+15555550101",
			"Body": "hello",
		},
	})
	if err == nil {
		t.Fatalf("expected blocked redirect target to fail replay")
	}
	lowered := strings.ToLower(err.Error())
	if !strings.Contains(lowered, "private") && !strings.Contains(lowered, "metadata") {
		t.Fatalf("expected redirect target policy rejection, got %v", err)
	}
}

func TestReplayTwilioWebhookAllowsSafeRedirectTarget(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer targetServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get(twilioWebhookReplayMarkerHeader)); got != "1" {
			t.Fatalf("expected replay marker header %s=1, got %q", twilioWebhookReplayMarkerHeader, got)
		}
		if got := strings.TrimSpace(r.Header.Get(twilioWebhookRequestURLOverrideHeader)); got == "" {
			t.Fatalf("expected replay request-url override header to be set")
		}
		http.Redirect(w, r, targetServer.URL+"/twilio/sms", http.StatusTemporaryRedirect)
	}))
	defer redirectServer.Close()

	service := &CommTwilioService{}
	response, err := service.ReplayTwilioWebhook(context.Background(), transport.TwilioWebhookReplayRequest{
		WorkspaceID:   "workspace-test",
		Kind:          "sms",
		BaseURL:       redirectServer.URL,
		SignatureMode: twilioWebhookSignatureModeBypass,
		Params: map[string]string{
			"From": "+15555550100",
			"To":   "+15555550101",
			"Body": "hello",
		},
	})
	if err != nil {
		t.Fatalf("expected safe redirect replay to succeed: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if response.ResponseBody != `{"ok":true}` {
		t.Fatalf("unexpected response body %q", response.ResponseBody)
	}
}

func TestReplayTwilioWebhookRejectsOversizedResponseBody(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(twilioWebhookReplayMaxResponseBytes+256))
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(oversizedBody))
	}))
	defer targetServer.Close()

	service := &CommTwilioService{}
	response, err := service.ReplayTwilioWebhook(context.Background(), transport.TwilioWebhookReplayRequest{
		WorkspaceID:   "workspace-test",
		Kind:          "sms",
		BaseURL:       targetServer.URL,
		SignatureMode: twilioWebhookSignatureModeBypass,
		Params: map[string]string{
			"From": "+15555550100",
			"To":   "+15555550101",
			"Body": "hello",
		},
	})
	if err == nil {
		t.Fatalf("expected oversized replay response body to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized replay response error, got %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected response status code to be preserved, got %d", response.StatusCode)
	}
	if len(response.ResponseBody) > int(twilioWebhookReplayMaxResponseBytes) {
		t.Fatalf("expected bounded response body <= %d bytes, got %d", twilioWebhookReplayMaxResponseBytes, len(response.ResponseBody))
	}
}
