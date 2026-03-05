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

func normalizeOllamaTextDelta(previous string, rawChunk string) (delta string, next string) {
	if rawChunk == "" {
		return "", previous
	}
	if previous == "" {
		return rawChunk, rawChunk
	}
	if strings.HasPrefix(rawChunk, previous) {
		return rawChunk[len(previous):], rawChunk
	}
	if strings.HasSuffix(previous, rawChunk) {
		return "", previous
	}

	previousRunes := []rune(previous)
	chunkRunes := []rune(rawChunk)
	maxOverlap := len(previousRunes)
	if len(chunkRunes) < maxOverlap {
		maxOverlap = len(chunkRunes)
	}
	for overlap := maxOverlap; overlap > 0; overlap-- {
		if string(previousRunes[len(previousRunes)-overlap:]) == string(chunkRunes[:overlap]) {
			deltaRunes := chunkRunes[overlap:]
			if len(deltaRunes) == 0 {
				return "", previous
			}
			delta = string(deltaRunes)
			return delta, previous + delta
		}
	}

	return rawChunk, previous + rawChunk
}

func streamOllama(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildOllamaChatURL(endpoint)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"model":    modelKey,
		"messages": messages,
		"stream":   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ollama payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build ollama request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("ollama status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type ollamaChunk struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	emittedText := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk ollamaChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return fmt.Errorf("decode ollama stream chunk: %w", err)
		}
		delta, nextText := normalizeOllamaTextDelta(emittedText, chunk.Message.Content)
		emittedText = nextText
		if delta != "" {
			if err := onDelta(delta); err != nil {
				return err
			}
		}
		if chunk.Done {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read ollama stream: %w", err)
	}
	return nil
}

func streamOllamaNativeTools(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	tools []ToolSpec,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildOllamaChatURL(endpoint)
	if err != nil {
		return err
	}

	ollamaTools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		parameters := map[string]any{}
		if len(tool.InputSchema) > 0 {
			parameters = tool.InputSchema
		}
		ollamaTools = append(ollamaTools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": strings.TrimSpace(tool.Description),
				"parameters":  parameters,
			},
		})
	}
	if len(ollamaTools) == 0 {
		return streamOllama(ctx, httpClient, endpoint, modelKey, messages, apiKey, onDelta)
	}

	payload := map[string]any{
		"model":    modelKey,
		"messages": messages,
		"stream":   true,
		"tools":    ollamaTools,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ollama native-tools payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build ollama native-tools request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama native-tools request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("ollama native-tools status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type ollamaToolCallChunk struct {
		Function struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		} `json:"function"`
	}
	type ollamaChunk struct {
		Message struct {
			Content   string                `json:"content"`
			ToolCalls []ollamaToolCallChunk `json:"tool_calls"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	textBuilder := &strings.Builder{}
	toolName := ""
	toolArgs := map[string]any{}
	emittedText := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk ollamaChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return fmt.Errorf("decode ollama native-tools stream chunk: %w", err)
		}
		delta, nextText := normalizeOllamaTextDelta(emittedText, chunk.Message.Content)
		emittedText = nextText
		if delta != "" {
			textBuilder.WriteString(delta)
		}
		if len(chunk.Message.ToolCalls) > 0 {
			first := chunk.Message.ToolCalls[0]
			if strings.TrimSpace(first.Function.Name) != "" {
				toolName = strings.TrimSpace(first.Function.Name)
			}
			if len(first.Function.Arguments) > 0 {
				toolArgs = first.Function.Arguments
			}
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read ollama native-tools stream: %w", err)
	}

	if strings.TrimSpace(toolName) != "" {
		directivePayload, err := json.Marshal(map[string]any{
			"type":      "tool_call",
			"tool_name": toolName,
			"arguments": toolArgs,
		})
		if err != nil {
			return fmt.Errorf("marshal ollama native-tools planner directive: %w", err)
		}
		return onDelta(string(directivePayload))
	}

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil
	}
	return onDelta(content)
}

func buildOllamaChatURL(endpoint string) (string, error) {
	parsed, err := endpointpolicy.ParseAndValidate(strings.TrimSpace(endpoint), endpointpolicy.Options{
		Service: "ollama provider endpoint",
	})
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/api/chat"):
		// already full endpoint
	case path == "":
		path = "/api/chat"
	default:
		path = path + "/api/chat"
	}
	parsed.Path = path
	return parsed.String(), nil
}
