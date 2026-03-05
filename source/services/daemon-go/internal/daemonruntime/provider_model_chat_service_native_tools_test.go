package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestChatRuntimeToolSpecsFromCatalogBuildsNativeSpecs(t *testing.T) {
	specs, enabled := chatRuntimeToolSpecsFromCatalog([]transport.ChatTurnToolCatalogEntry{{
		Name:        "browser_open",
		Description: "Open a web page.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "Target URL.",
				},
			},
			"required": []string{"url"},
		},
	}})
	if !enabled {
		t.Fatalf("expected native tool parsing enabled when typed catalog is provided")
	}
	if len(specs) != 1 {
		t.Fatalf("expected one parsed tool spec, got %+v", specs)
	}
	if specs[0].Name != "browser_open" {
		t.Fatalf("expected tool browser_open, got %+v", specs[0])
	}
	if specs[0].InputSchema["type"] != "object" {
		t.Fatalf("expected object input schema, got %+v", specs[0].InputSchema)
	}
}

func putProviderAPIKey(t *testing.T, container *ServiceContainer, workspace string, secretName string, value string) {
	t.Helper()
	resolver, ok := container.SecretResolver.(*managerSecretResolver)
	if !ok || resolver == nil || resolver.manager == nil {
		t.Fatalf("expected manager-backed secret resolver, got %T", container.SecretResolver)
	}
	if _, err := resolver.manager.Put(workspace, secretName, value); err != nil {
		t.Fatalf("put provider secret %s: %v", secretName, err)
	}
}

func TestProviderModelChatServiceChatTurnUsesTypedToolCatalogForNativeToolCallingHostedProviders(t *testing.T) {
	cases := []struct {
		name                 string
		provider             string
		modelKey             string
		apiKeySecretName     string
		expectedPayloadField string
		validatePath         func(t *testing.T, r *http.Request)
		responseBody         string
		expectedToolName     string
	}{
		{
			name:                 "anthropic",
			provider:             "anthropic",
			modelKey:             "claude-3-5-sonnet-latest",
			apiKeySecretName:     "ANTHROPIC_API_KEY",
			expectedPayloadField: `"tools"`,
			validatePath: func(t *testing.T, r *http.Request) {
				t.Helper()
				if r.URL.Path != "/v1/messages" {
					t.Fatalf("expected /v1/messages path, got %s", r.URL.Path)
				}
			},
			responseBody:     "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"name\":\"browser_open\",\"input\":{}}}\n\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"url\\\":\\\"https://example.com\\\"}\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n",
			expectedToolName: "browser_open",
		},
		{
			name:                 "google",
			provider:             "google",
			modelKey:             "gemini-2.0-flash",
			apiKeySecretName:     "GOOGLE_API_KEY",
			expectedPayloadField: `"functionDeclarations"`,
			validatePath: func(t *testing.T, r *http.Request) {
				t.Helper()
				if r.URL.Path != "/v1beta/models/gemini-2.0-flash:streamGenerateContent" {
					t.Fatalf("expected google stream path, got %s", r.URL.Path)
				}
				if got := strings.TrimSpace(r.URL.Query().Get("alt")); got != "sse" {
					t.Fatalf("expected alt=sse query, got %q", got)
				}
			},
			responseBody:     "data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"browser_open\",\"args\":{\"url\":\"https://example.com\"}}}]}}]}\n\n",
			expectedToolName: "browser_open",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requestBodies := make([]string, 0, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.validatePath(t, r)
				bodyBytes, _ := io.ReadAll(r.Body)
				requestBodies = append(requestBodies, string(bodyBytes))
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			container := newLifecycleTestContainer(t, nil)
			service := NewProviderModelChatService(container)
			putProviderAPIKey(t, container, "ws1", tc.apiKeySecretName, fmt.Sprintf("%s-key", tc.provider))

			endpoint := server.URL
			if strings.EqualFold(tc.provider, "google") {
				endpoint = server.URL + "/v1beta"
			}
			if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
				WorkspaceID:      "ws1",
				Provider:         tc.provider,
				Endpoint:         endpoint,
				APIKeySecretName: tc.apiKeySecretName,
				ClearAPIKey:      false,
			}); err != nil {
				t.Fatalf("set %s provider: %v", tc.provider, err)
			}

			response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
				WorkspaceID:      "ws1",
				TaskClass:        "chat",
				ProviderOverride: tc.provider,
				ModelOverride:    tc.modelKey,
				SystemPrompt:     "Use the best available strategy to satisfy the request.",
				ToolCatalog: []transport.ChatTurnToolCatalogEntry{{
					Name:        "browser_open",
					Description: "Open a web page.",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"url": map[string]any{
								"type": "string",
							},
						},
						"required": []string{"url"},
					},
				}},
				Items: []transport.ChatTurnItem{{
					Type:    "user_message",
					Role:    "user",
					Status:  "completed",
					Content: "Open example.com",
				}},
			}, "corr-native-tool-"+tc.provider, nil)
			if err != nil {
				t.Fatalf("chat turn: %v", err)
			}
			if len(requestBodies) != 1 {
				t.Fatalf("expected one provider request body capture, got %d", len(requestBodies))
			}
			if !strings.Contains(requestBodies[0], tc.expectedPayloadField) {
				t.Fatalf("expected provider payload to include %s, got %s", tc.expectedPayloadField, requestBodies[0])
			}
			if len(response.Items) != 1 {
				t.Fatalf("expected one assistant item, got %+v", response.Items)
			}

			var directive map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(response.Items[0].Content)), &directive); err != nil {
				t.Fatalf("expected native tool planner directive json, got %q (err=%v)", response.Items[0].Content, err)
			}
			if strings.TrimSpace(directive["type"].(string)) != "tool_call" {
				t.Fatalf("expected type tool_call, got %+v", directive)
			}
			if strings.TrimSpace(directive["tool_name"].(string)) != tc.expectedToolName {
				t.Fatalf("expected tool_name %s, got %+v", tc.expectedToolName, directive)
			}
		})
	}
}

func TestProviderModelChatServiceChatTurnUsesTypedToolCatalogForNativeToolCalling(t *testing.T) {
	requestBodies := make([]string, 0, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("expected /api/chat path, got %s", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		requestBodies = append(requestBodies, string(bodyBytes))
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"message\":{\"tool_calls\":[{\"function\":{\"name\":\"browser_open\",\"arguments\":{\"url\":\"https://example.com\"}}}]},\"done\":true}\n"))
	}))
	defer server.Close()

	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    server.URL,
	}); err != nil {
		t.Fatalf("set ollama provider: %v", err)
	}

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID:  "ws1",
		TaskClass:    "chat",
		SystemPrompt: "Use the best available strategy to satisfy the request.",
		ToolCatalog: []transport.ChatTurnToolCatalogEntry{{
			Name:        "browser_open",
			Description: "Open a web page.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"url"},
			},
		}},
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Open example.com",
		}},
	}, "corr-native-tool-ollama", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one provider request body capture, got %d", len(requestBodies))
	}
	if !strings.Contains(requestBodies[0], `"tools"`) {
		t.Fatalf("expected provider request to include native tools payload, got %s", requestBodies[0])
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one assistant item, got %+v", response.Items)
	}

	var directive map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(response.Items[0].Content)), &directive); err != nil {
		t.Fatalf("expected native tool planner directive json, got %q (err=%v)", response.Items[0].Content, err)
	}
	if strings.TrimSpace(directive["type"].(string)) != "tool_call" {
		t.Fatalf("expected type tool_call, got %+v", directive)
	}
	if strings.TrimSpace(directive["tool_name"].(string)) != "browser_open" {
		t.Fatalf("expected tool_name browser_open, got %+v", directive)
	}
}
