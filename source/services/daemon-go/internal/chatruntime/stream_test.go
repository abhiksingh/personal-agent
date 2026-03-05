package chatruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamAssistantResponseOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-openai" {
			t.Fatalf("expected authorization header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "openai",
		Endpoint: server.URL,
		ModelKey: "gpt-4.1-mini",
		APIKey:   "sk-openai",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream openai response: %v", err)
	}
	if builder.String() != "Hello world" {
		t.Fatalf("unexpected stream output %q", builder.String())
	}
}

func TestStreamAssistantResponseOpenAIPreservesWhitespaceOnlyDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Line 1\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"\\n\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"- item\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "openai",
		Endpoint: server.URL,
		ModelKey: "gpt-4.1-mini",
		APIKey:   "sk-openai",
		Messages: []Message{{Role: "user", Content: "list"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream openai whitespace response: %v", err)
	}
	if builder.String() != "Line 1\n- item" {
		t.Fatalf("expected markdown-preserving output, got %q", builder.String())
	}
}

func TestStreamAssistantResponseOpenAIHandlesLargeSingleChunk(t *testing.T) {
	largeContent := strings.Repeat("x", (1024*1024)+(128*1024))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunkBytes, err := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{
					"delta": map[string]any{
						"content": largeContent,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("marshal chunk: %v", err)
		}
		_, _ = w.Write([]byte("data: " + string(chunkBytes) + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "openai",
		Endpoint: server.URL,
		ModelKey: "gpt-4.1-mini",
		APIKey:   "sk-openai",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream openai large chunk response: %v", err)
	}
	if builder.String() != largeContent {
		t.Fatalf("expected large content length %d, got %d", len(largeContent), len(builder.String()))
	}
}

func TestStreamAssistantResponseRequiresHTTPClient(t *testing.T) {
	err := StreamAssistantResponse(context.Background(), nil, StreamRequest{
		Provider: "openai",
		ModelKey: "gpt-4.1-mini",
		APIKey:   "sk-openai",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "http client is required") {
		t.Fatalf("expected missing http client error, got %v", err)
	}
}

func TestStreamURLBuildersRejectInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	for _, builder := range streamURLBuildersForSecurityTests() {
		builder := builder
		t.Run(builder.name, func(t *testing.T) {
			_, err := builder.build("http://api.example.com")
			if err == nil {
				t.Fatalf("expected insecure non-loopback endpoint to be rejected")
			}
		})
	}
}

func TestStreamURLBuildersRejectPrivateEndpointByDefault(t *testing.T) {
	for _, builder := range streamURLBuildersForSecurityTests() {
		builder := builder
		t.Run(builder.name, func(t *testing.T) {
			_, err := builder.build("https://10.0.0.20")
			if err == nil {
				t.Fatalf("expected private endpoint to be rejected")
			}
		})
	}
}

func TestStreamURLBuildersAllowInsecurePrivateEndpointWithExplicitOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	for _, builder := range streamURLBuildersForSecurityTests() {
		builder := builder
		t.Run(builder.name, func(t *testing.T) {
			builtURL, err := builder.build("http://10.0.0.20")
			if err != nil {
				t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
			}
			if strings.TrimSpace(builtURL) == "" {
				t.Fatalf("expected non-empty URL")
			}
		})
	}
}

func TestStreamAssistantResponseOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("expected /api/chat, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"Hi\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\" there\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"\"},\"done\":true}\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "ollama",
		Endpoint: server.URL,
		ModelKey: "llama3.2",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream ollama response: %v", err)
	}
	if builder.String() != "Hi there" {
		t.Fatalf("unexpected stream output %q", builder.String())
	}
}

func TestStreamAssistantResponseOllamaCanonicalizesCumulativeChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("expected /api/chat, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"I\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"I can\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"I can help\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"\"},\"done\":true}\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "ollama",
		Endpoint: server.URL,
		ModelKey: "gpt-oss:20b",
		Messages: []Message{{Role: "user", Content: "help"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream ollama cumulative response: %v", err)
	}
	if builder.String() != "I can help" {
		t.Fatalf("unexpected cumulative stream output %q", builder.String())
	}
}

func TestStreamAssistantResponseAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("expected /v1/messages, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := map[string]any{}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			t.Fatalf("decode anthropic request body: %v", err)
		}
		maxTokens, ok := payload["max_tokens"].(float64)
		if !ok || int(maxTokens) != defaultAnthropicMaxTokens {
			t.Fatalf("expected max_tokens=%d, got payload=%s", defaultAnthropicMaxTokens, string(bodyBytes))
		}
		if r.Header.Get("x-api-key") != "sk-anthropic" {
			t.Fatalf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Fatalf("expected anthropic-version header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "anthropic",
		Endpoint: server.URL,
		ModelKey: "claude-3-5-sonnet-latest",
		APIKey:   "sk-anthropic",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream anthropic response: %v", err)
	}
	if builder.String() != "Hello world" {
		t.Fatalf("unexpected stream output %q", builder.String())
	}
}

func TestStreamAssistantResponseGoogle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-2.0-flash:streamGenerateContent" {
			t.Fatalf("expected google stream path, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("expected alt=sse query, got %q", got)
		}
		if r.Header.Get("x-goog-api-key") != "google-key" {
			t.Fatalf("expected x-goog-api-key header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hi\"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" there\"}]}}]}\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider: "google",
		Endpoint: server.URL + "/v1beta",
		ModelKey: "gemini-2.0-flash",
		APIKey:   "google-key",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream google response: %v", err)
	}
	if builder.String() != "Hi there" {
		t.Fatalf("unexpected stream output %q", builder.String())
	}
}

func TestStreamAssistantResponseOpenAINativeToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := string(bodyBytes)
		if !strings.Contains(payload, `"tools"`) {
			t.Fatalf("expected openai payload to include tools, got %s", payload)
		}
		if !strings.Contains(payload, `"tool_choice":"auto"`) {
			t.Fatalf("expected openai payload to include tool_choice=auto, got %s", payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"mail_send\",\"arguments\":\"{\\\"recipient\\\":\\\"sam@example.com\\\"\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\",\\\"body\\\":\\\"hello\\\"}\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider:                "openai",
		Endpoint:                server.URL,
		ModelKey:                "gpt-4.1-mini",
		APIKey:                  "sk-openai",
		PreferNativeToolCalling: true,
		ToolSpecs: []ToolSpec{
			{
				Name:        "mail_send",
				Description: "Send an email.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"recipient": map[string]any{"type": "string"},
						"body":      map[string]any{"type": "string"},
					},
				},
			},
		},
		Messages: []Message{{Role: "user", Content: "send email"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream openai native tool response: %v", err)
	}
	expected := `{"arguments":{"body":"hello","recipient":"sam@example.com"},"tool_name":"mail_send","type":"tool_call"}`
	if builder.String() != expected {
		t.Fatalf("unexpected native tool output %q", builder.String())
	}
}

func TestStreamAssistantResponseAnthropicNativeToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("expected /v1/messages, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := string(bodyBytes)
		if !strings.Contains(payload, `"tools"`) {
			t.Fatalf("expected anthropic payload to include tools, got %s", payload)
		}
		decoded := map[string]any{}
		if err := json.Unmarshal(bodyBytes, &decoded); err != nil {
			t.Fatalf("decode anthropic native-tools payload: %v", err)
		}
		maxTokens, ok := decoded["max_tokens"].(float64)
		if !ok || int(maxTokens) != defaultAnthropicMaxTokens {
			t.Fatalf("expected max_tokens=%d, got payload=%s", defaultAnthropicMaxTokens, payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"name\":\"mail_send\",\"input\":{}}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"recipient\\\":\\\"sam@example.com\\\"\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\",\\\"body\\\":\\\"hello\\\"}\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider:                "anthropic",
		Endpoint:                server.URL,
		ModelKey:                "claude-3-5-sonnet-latest",
		APIKey:                  "anthropic-key",
		PreferNativeToolCalling: true,
		ToolSpecs: []ToolSpec{
			{
				Name: "mail_send",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"recipient": map[string]any{"type": "string"},
						"body":      map[string]any{"type": "string"},
					},
				},
			},
		},
		Messages: []Message{{Role: "user", Content: "send email"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream anthropic native tool response: %v", err)
	}
	expected := `{"arguments":{"body":"hello","recipient":"sam@example.com"},"tool_name":"mail_send","type":"tool_call"}`
	if builder.String() != expected {
		t.Fatalf("unexpected anthropic native tool output %q", builder.String())
	}
}

func TestStreamAssistantResponseGoogleNativeToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-2.0-flash:streamGenerateContent" {
			t.Fatalf("expected google stream path, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := string(bodyBytes)
		if !strings.Contains(payload, `"functionDeclarations"`) {
			t.Fatalf("expected google payload to include function declarations, got %s", payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"browser_open\",\"args\":{\"url\":\"https://example.com\"}}}]}}]}\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider:                "google",
		Endpoint:                server.URL + "/v1beta",
		ModelKey:                "gemini-2.0-flash",
		APIKey:                  "google-key",
		PreferNativeToolCalling: true,
		ToolSpecs: []ToolSpec{
			{
				Name: "browser_open",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{"type": "string"},
					},
				},
			},
		},
		Messages: []Message{{Role: "user", Content: "open example.com"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream google native tool response: %v", err)
	}
	expected := `{"arguments":{"url":"https://example.com"},"tool_name":"browser_open","type":"tool_call"}`
	if builder.String() != expected {
		t.Fatalf("unexpected google native tool output %q", builder.String())
	}
}

func TestStreamAssistantResponseNativeToolCallingFallsBackWhenNoValidToolSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("expected /v1/messages, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := string(bodyBytes)
		if strings.Contains(payload, `"tools"`) {
			t.Fatalf("expected unsupported provider payload to omit tools, got %s", payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"fallback\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" response\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider:                "anthropic",
		Endpoint:                server.URL,
		ModelKey:                "claude-3-5-sonnet-latest",
		APIKey:                  "anthropic-key",
		PreferNativeToolCalling: true,
		ToolSpecs: []ToolSpec{
			{Name: "  ", InputSchema: map[string]any{"type": "object"}},
		},
		Messages: []Message{{Role: "user", Content: "send email"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream provider fallback response: %v", err)
	}
	if builder.String() != "fallback response" {
		t.Fatalf("unexpected fallback output %q", builder.String())
	}
}

func TestStreamAssistantResponseOllamaNativeToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("expected /api/chat, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		payload := string(bodyBytes)
		if !strings.Contains(payload, `"tools"`) {
			t.Fatalf("expected ollama native tool payload to include tools, got %s", payload)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"message\":{\"tool_calls\":[{\"function\":{\"name\":\"browser_open\",\"arguments\":{\"url\":\"https://example.com\"}}}]},\"done\":true}\n"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	builder := &strings.Builder{}
	err := StreamAssistantResponse(ctx, server.Client(), StreamRequest{
		Provider:                "ollama",
		Endpoint:                server.URL,
		ModelKey:                "llama3.2",
		PreferNativeToolCalling: true,
		ToolSpecs: []ToolSpec{
			{
				Name: "browser_open",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{"type": "string"},
					},
				},
			},
		},
		Messages: []Message{{Role: "user", Content: "open example.com"}},
	}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream ollama native tool response: %v", err)
	}
	expected := `{"arguments":{"url":"https://example.com"},"tool_name":"browser_open","type":"tool_call"}`
	if builder.String() != expected {
		t.Fatalf("unexpected ollama native tool output %q", builder.String())
	}
}

type streamURLBuilder struct {
	name  string
	build func(endpoint string) (string, error)
}

func streamURLBuildersForSecurityTests() []streamURLBuilder {
	return []streamURLBuilder{
		{
			name:  "openai",
			build: buildOpenAIChatURL,
		},
		{
			name:  "anthropic",
			build: buildAnthropicChatURL,
		},
		{
			name: "google",
			build: func(endpoint string) (string, error) {
				return buildGoogleChatURL(endpoint, "gemini-2.0-flash")
			},
		},
		{
			name:  "ollama",
			build: buildOllamaChatURL,
		},
	}
}
