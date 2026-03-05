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

func streamOpenAI(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildOpenAIChatURL(endpoint)
	if err != nil {
		return err
	}

	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("openai api key is required")
	}

	payload := map[string]any{
		"model":    modelKey,
		"messages": messages,
		"stream":   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal openai payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("openai status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type openAIChunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
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
		if chunkData == "[DONE]" {
			return nil
		}

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode openai stream chunk: %w", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := onDelta(choice.Delta.Content); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read openai stream: %w", err)
	}
	return nil
}

func streamOpenAINativeTools(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	tools []ToolSpec,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildOpenAIChatURL(endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("openai api key is required")
	}

	openAITools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		parameters := map[string]any{}
		if len(tool.InputSchema) > 0 {
			parameters = tool.InputSchema
		}
		openAITools = append(openAITools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": strings.TrimSpace(tool.Description),
				"parameters":  parameters,
			},
		})
	}
	if len(openAITools) == 0 {
		return streamOpenAI(ctx, httpClient, endpoint, modelKey, messages, apiKey, onDelta)
	}

	payload := map[string]any{
		"model":       modelKey,
		"messages":    messages,
		"stream":      true,
		"tools":       openAITools,
		"tool_choice": "auto",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal openai native-tools payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build openai native-tools request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai native-tools request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("openai native-tools status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type toolCallChunk struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	type openAIChunk struct {
		Choices []struct {
			Delta struct {
				Content   string          `json:"content"`
				ToolCalls []toolCallChunk `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	type openAIToolCallState struct {
		Name      string
		Arguments strings.Builder
	}

	textBuilder := &strings.Builder{}
	toolCalls := map[int]*openAIToolCallState{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		chunkData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if chunkData == "" {
			continue
		}
		if chunkData == "[DONE]" {
			break
		}

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode openai native-tools stream chunk: %w", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				textBuilder.WriteString(choice.Delta.Content)
			}
			for _, toolCall := range choice.Delta.ToolCalls {
				state, ok := toolCalls[toolCall.Index]
				if !ok {
					state = &openAIToolCallState{}
					toolCalls[toolCall.Index] = state
				}
				if strings.TrimSpace(toolCall.Function.Name) != "" {
					state.Name = strings.TrimSpace(toolCall.Function.Name)
				}
				if strings.TrimSpace(toolCall.Function.Arguments) != "" {
					state.Arguments.WriteString(toolCall.Function.Arguments)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read openai native-tools stream: %w", err)
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
			rawArguments := strings.TrimSpace(state.Arguments.String())
			if rawArguments != "" {
				_ = json.Unmarshal([]byte(rawArguments), &arguments)
			}
			directivePayload, err := json.Marshal(map[string]any{
				"type":      "tool_call",
				"tool_name": strings.TrimSpace(state.Name),
				"arguments": arguments,
			})
			if err != nil {
				return fmt.Errorf("marshal openai native-tools planner directive: %w", err)
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

func buildOpenAIChatURL(endpoint string) (string, error) {
	parsed, err := endpointpolicy.ParseAndValidate(strings.TrimSpace(endpoint), endpointpolicy.Options{
		Service: "openai provider endpoint",
	})
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		// already full endpoint
	case strings.HasSuffix(path, "/v1"):
		path = path + "/chat/completions"
	case path == "":
		path = "/v1/chat/completions"
	default:
		path = path + "/chat/completions"
	}
	parsed.Path = path
	return parsed.String(), nil
}
