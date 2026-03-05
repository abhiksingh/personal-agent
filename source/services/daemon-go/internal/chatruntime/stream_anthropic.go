package chatruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"personalagent/runtime/internal/endpointpolicy"
)

func streamAnthropicNativeTools(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	tools []ToolSpec,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildAnthropicChatURL(endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("anthropic api key is required")
	}

	anthropicTools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		inputSchema := map[string]any{}
		if len(tool.InputSchema) > 0 {
			inputSchema = tool.InputSchema
		}
		anthropicTools = append(anthropicTools, map[string]any{
			"name":         name,
			"description":  strings.TrimSpace(tool.Description),
			"input_schema": inputSchema,
		})
	}
	if len(anthropicTools) == 0 {
		return streamAnthropic(ctx, httpClient, endpoint, modelKey, messages, apiKey, onDelta)
	}

	systemPrompt, payloadMessages := buildAnthropicMessages(messages)
	payload := map[string]any{
		"model":       modelKey,
		"messages":    payloadMessages,
		"stream":      true,
		"max_tokens":  defaultAnthropicMaxTokens,
		"tools":       anthropicTools,
		"tool_choice": map[string]any{"type": "auto"},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["system"] = systemPrompt
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal anthropic native-tools payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build anthropic native-tools request: %w", err)
	}
	req.Header.Set("x-api-key", strings.TrimSpace(apiKey))
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic native-tools request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("anthropic native-tools status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type anthropicChunk struct {
		Type         string `json:"type"`
		Index        int    `json:"index"`
		ContentBlock struct {
			Type  string         `json:"type"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content_block"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	type anthropicToolCallState struct {
		Name      string
		Arguments strings.Builder
		Input     map[string]any
	}

	textBuilder := &strings.Builder{}
	toolCalls := map[int]*anthropicToolCallState{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		chunkData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if chunkData == "" {
			continue
		}

		var chunk anthropicChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode anthropic native-tools stream chunk: %w", err)
		}
		if chunk.Type == "message_stop" {
			break
		}
		switch chunk.Type {
		case "content_block_start":
			if chunk.ContentBlock.Type != "tool_use" {
				continue
			}
			state, ok := toolCalls[chunk.Index]
			if !ok {
				state = &anthropicToolCallState{}
				toolCalls[chunk.Index] = state
			}
			if strings.TrimSpace(chunk.ContentBlock.Name) != "" {
				state.Name = strings.TrimSpace(chunk.ContentBlock.Name)
			}
			if len(chunk.ContentBlock.Input) > 0 {
				state.Input = chunk.ContentBlock.Input
			}
		case "content_block_delta":
			if chunk.Delta.Type == "text_delta" && chunk.Delta.Text != "" {
				textBuilder.WriteString(chunk.Delta.Text)
				continue
			}
			if chunk.Delta.Type != "input_json_delta" || strings.TrimSpace(chunk.Delta.PartialJSON) == "" {
				continue
			}
			state, ok := toolCalls[chunk.Index]
			if !ok {
				state = &anthropicToolCallState{}
				toolCalls[chunk.Index] = state
			}
			state.Arguments.WriteString(chunk.Delta.PartialJSON)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read anthropic native-tools stream: %w", err)
	}

	if len(toolCalls) > 0 {
		firstIndex := -1
		for index := range toolCalls {
			if firstIndex == -1 || index < firstIndex {
				firstIndex = index
			}
		}
		state := toolCalls[firstIndex]
		if state != nil && strings.TrimSpace(state.Name) != "" {
			arguments := map[string]any{}
			for key, value := range state.Input {
				arguments[key] = value
			}
			rawArguments := strings.TrimSpace(state.Arguments.String())
			if rawArguments != "" {
				decoded := map[string]any{}
				if err := json.Unmarshal([]byte(rawArguments), &decoded); err == nil {
					for key, value := range decoded {
						arguments[key] = value
					}
				}
			}
			directivePayload, err := json.Marshal(map[string]any{
				"type":      "tool_call",
				"tool_name": strings.TrimSpace(state.Name),
				"arguments": arguments,
			})
			if err != nil {
				return fmt.Errorf("marshal anthropic native-tools planner directive: %w", err)
			}
			return onDelta(string(directivePayload))
		}
	}

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil
	}
	return onDelta(content)
}

func streamAnthropic(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildAnthropicChatURL(endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("anthropic api key is required")
	}

	systemPrompt, payloadMessages := buildAnthropicMessages(messages)
	payload := map[string]any{
		"model":      modelKey,
		"messages":   payloadMessages,
		"stream":     true,
		"max_tokens": defaultAnthropicMaxTokens,
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["system"] = systemPrompt
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal anthropic payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", strings.TrimSpace(apiKey))
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("anthropic status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type anthropicChunk struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		chunkData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if chunkData == "" {
			continue
		}

		var chunk anthropicChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode anthropic stream chunk: %w", err)
		}
		if chunk.Type == "message_stop" {
			return nil
		}
		if chunk.Type == "content_block_delta" && chunk.Delta.Type == "text_delta" && chunk.Delta.Text != "" {
			if err := onDelta(chunk.Delta.Text); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read anthropic stream: %w", err)
	}
	return nil
}

func buildAnthropicChatURL(endpoint string) (string, error) {
	parsed, err := endpointpolicy.ParseAndValidate(strings.TrimSpace(endpoint), endpointpolicy.Options{
		Service: "anthropic provider endpoint",
	})
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/messages"):
		// already full endpoint
	case strings.HasSuffix(path, "/v1"):
		path = path + "/messages"
	case path == "":
		path = "/v1/messages"
	default:
		path = path + "/messages"
	}
	parsed.Path = path
	return parsed.String(), nil
}

func buildAnthropicMessages(messages []Message) (string, []map[string]string) {
	systemSegments := make([]string, 0)
	payloadMessages := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			systemSegments = append(systemSegments, content)
		case "assistant", "model":
			payloadMessages = append(payloadMessages, map[string]string{
				"role":    "assistant",
				"content": content,
			})
		default:
			payloadMessages = append(payloadMessages, map[string]string{
				"role":    "user",
				"content": content,
			})
		}
	}
	return strings.Join(systemSegments, "\n\n"), payloadMessages
}
