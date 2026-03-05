package daemonruntime

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewTwilioWebhookHTTPServerConfiguresSlowRequestTimeouts(t *testing.T) {
	server := newTwilioWebhookHTTPServer(http.NewServeMux())

	if server.ReadHeaderTimeout != defaultTwilioWebhookReadHeaderTimeout {
		t.Fatalf("expected read header timeout %s, got %s", defaultTwilioWebhookReadHeaderTimeout, server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != defaultTwilioWebhookReadTimeout {
		t.Fatalf("expected read timeout %s, got %s", defaultTwilioWebhookReadTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != defaultTwilioWebhookWriteTimeout {
		t.Fatalf("expected write timeout %s, got %s", defaultTwilioWebhookWriteTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != defaultTwilioWebhookIdleTimeout {
		t.Fatalf("expected idle timeout %s, got %s", defaultTwilioWebhookIdleTimeout, server.IdleTimeout)
	}
}

func TestParseTwilioWebhookFormRejectsOversizedBody(t *testing.T) {
	body := "Body=" + strings.Repeat("a", int(defaultTwilioWebhookMaxRequestBytes)+1)
	request := httptest.NewRequest(http.MethodPost, "/webhook/sms", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	statusCode, err := parseTwilioWebhookForm(recorder, request)
	if err == nil {
		t.Fatalf("expected oversized body parse error")
	}
	if statusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, statusCode)
	}
}

func TestParseTwilioWebhookFormRejectsExcessiveFormFields(t *testing.T) {
	values := url.Values{}
	for index := 0; index <= defaultTwilioWebhookMaxFormFields; index++ {
		values.Set(fmt.Sprintf("field_%d", index), "value")
	}

	request := httptest.NewRequest(http.MethodPost, "/webhook/sms", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	statusCode, err := parseTwilioWebhookForm(recorder, request)
	if err == nil {
		t.Fatalf("expected form field limit parse error")
	}
	if statusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, statusCode)
	}
}

func TestParseTwilioWebhookFormAcceptsBoundedPayload(t *testing.T) {
	values := url.Values{
		"Body": []string{"hello"},
		"From": []string{"+15550000001"},
		"To":   []string{"+15550000002"},
	}
	request := httptest.NewRequest(http.MethodPost, "/webhook/sms", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	statusCode, err := parseTwilioWebhookForm(recorder, request)
	if err != nil {
		t.Fatalf("expected bounded payload parse success, got %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, statusCode)
	}
	if got := request.PostForm.Get("Body"); got != "hello" {
		t.Fatalf("expected Body=hello, got %q", got)
	}
}
