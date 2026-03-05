package providercheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCheckOpenAISuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected path /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("expected bearer token header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Check(ctx, server.Client(), Request{
		Provider: "openai",
		Endpoint: server.URL,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckRequiresHTTPClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, nil, Request{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "sk-test",
	})
	if err == nil || err.Error() != "http client is required" {
		t.Fatalf("expected missing http client error, got %v", err)
	}
}

func TestCheckOpenAIRequiresAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, http.DefaultClient, Request{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
	})
	if err == nil {
		t.Fatalf("expected error when api key is missing")
	}
}

func TestCheckOllamaSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("expected path /api/tags, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Check(ctx, server.Client(), Request{
		Provider: "ollama",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckAnthropicSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected path /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "anthropic-test" {
			t.Fatalf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Fatalf("expected anthropic-version header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Check(ctx, server.Client(), Request{
		Provider: "anthropic",
		Endpoint: server.URL,
		APIKey:   "anthropic-test",
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckGoogleSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models" {
			t.Fatalf("expected path /v1beta/models, got %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "google-test" {
			t.Fatalf("expected x-goog-api-key header")
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatalf("expected no authorization bearer header for google")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Check(ctx, server.Client(), Request{
		Provider: "google",
		Endpoint: server.URL,
		APIKey:   "google-test",
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestDiscoverOpenAIParsesModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected path /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("expected bearer token header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"},{"id":"gpt-4.1"}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Discover(ctx, server.Client(), Request{
		Provider: "openai",
		Endpoint: server.URL,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
	expected := []string{"gpt-4.1", "gpt-4.1-mini"}
	if !slices.Equal(result.Models, expected) {
		t.Fatalf("expected models %v, got %v", expected, result.Models)
	}
}

func TestDiscoverRequiresHTTPClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Discover(ctx, nil, Request{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "sk-test",
	})
	if err == nil || err.Error() != "http client is required" {
		t.Fatalf("expected missing http client error, got %v", err)
	}
}

func TestDiscoverOllamaParsesModelNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("expected path /api/tags, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"},{"name":"mistral"}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Discover(ctx, server.Client(), Request{
		Provider: "ollama",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	expected := []string{"llama3.2:latest", "mistral"}
	if !slices.Equal(result.Models, expected) {
		t.Fatalf("expected models %v, got %v", expected, result.Models)
	}
}

func TestDiscoverGoogleNormalizesModelPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models" {
			t.Fatalf("expected path /v1beta/models, got %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "google-test" {
			t.Fatalf("expected x-goog-api-key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-2.0-flash"}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Discover(ctx, server.Client(), Request{
		Provider: "google",
		Endpoint: server.URL,
		APIKey:   "google-test",
	})
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	expected := []string{"gemini-2.0-flash"}
	if !slices.Equal(result.Models, expected) {
		t.Fatalf("expected models %v, got %v", expected, result.Models)
	}
}

func TestDiscoverRequiresAPIKeyWhenProviderNeedsOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Discover(ctx, http.DefaultClient, Request{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
	})
	if err == nil {
		t.Fatalf("expected api key requirement error")
	}
}

func TestCheckRejectsInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, http.DefaultClient, Request{
		Provider: "openai",
		Endpoint: "http://api.openai.com/v1",
		APIKey:   "sk-test",
	})
	if err == nil {
		t.Fatalf("expected insecure non-loopback endpoint to be rejected")
	}
}

func TestCheckRejectsPrivateEndpointByDefault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, http.DefaultClient, Request{
		Provider: "openai",
		Endpoint: "https://10.0.0.20/v1",
		APIKey:   "sk-test",
	})
	if err == nil {
		t.Fatalf("expected private endpoint to be rejected")
	}
}

func TestCheckAllowsInsecurePrivateEndpointWithExplicitOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, server.Client(), Request{
		Provider: "openai",
		Endpoint: server.URL,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
}

func TestCheckRejectsRedirectToMetadataTarget(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://169.254.169.254/latest/meta-data", http.StatusFound)
	}))
	defer redirectServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Check(ctx, redirectServer.Client(), Request{
		Provider: "openai",
		Endpoint: redirectServer.URL,
		APIKey:   "sk-test",
	})
	if err == nil {
		t.Fatalf("expected metadata redirect to be rejected")
	}
	lowered := strings.ToLower(err.Error())
	if !strings.Contains(lowered, "private") && !strings.Contains(lowered, "metadata") {
		t.Fatalf("expected security redirect rejection, got %v", err)
	}
}

func TestDiscoverRejectsRedirectToPrivateTarget(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://10.20.30.40/v1/models", http.StatusFound)
	}))
	defer redirectServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Discover(ctx, redirectServer.Client(), Request{
		Provider: "openai",
		Endpoint: redirectServer.URL,
		APIKey:   "sk-test",
	})
	if err == nil {
		t.Fatalf("expected private redirect to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "private") {
		t.Fatalf("expected private redirect rejection message, got %v", err)
	}
}
