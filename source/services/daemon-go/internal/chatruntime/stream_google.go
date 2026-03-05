package chatruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"personalagent/runtime/internal/endpointpolicy"
)

func streamGoogleNativeTools(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	tools []ToolSpec,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildGoogleChatURL(endpoint, modelKey)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("google api key is required")
	}

	functionDeclarations := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		parameters := map[string]any{}
		if len(tool.InputSchema) > 0 {
			parameters = tool.InputSchema
		}
		functionDeclarations = append(functionDeclarations, map[string]any{
			"name":        name,
			"description": strings.TrimSpace(tool.Description),
			"parameters":  parameters,
		})
	}
	if len(functionDeclarations) == 0 {
		return streamGoogle(ctx, httpClient, endpoint, modelKey, messages, apiKey, onDelta)
	}

	systemPrompt, payloadContents := buildGoogleContents(messages)
	payload := map[string]any{
		"contents": payloadContents,
		"tools": []map[string]any{
			{
				"functionDeclarations": functionDeclarations,
			},
		},
		"toolConfig": map[string]any{
			"functionCallingConfig": map[string]any{
				"mode": "AUTO",
			},
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{
					"text": systemPrompt,
				},
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal google native-tools payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build google native-tools request: %w", err)
	}
	req.Header.Set("x-goog-api-key", strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("google native-tools request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("google native-tools status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type googleChunk struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	textBuilder := &strings.Builder{}
	toolName := ""
	toolArguments := map[string]any{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		chunkData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if chunkData == "" || chunkData == "[DONE]" {
			continue
		}

		var chunk googleChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode google native-tools stream chunk: %w", err)
		}
		for _, candidate := range chunk.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					textBuilder.WriteString(part.Text)
				}
				if strings.TrimSpace(part.FunctionCall.Name) != "" {
					toolName = strings.TrimSpace(part.FunctionCall.Name)
					if len(part.FunctionCall.Args) > 0 {
						toolArguments = part.FunctionCall.Args
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read google native-tools stream: %w", err)
	}

	if strings.TrimSpace(toolName) != "" {
		directivePayload, err := json.Marshal(map[string]any{
			"type":      "tool_call",
			"tool_name": toolName,
			"arguments": toolArguments,
		})
		if err != nil {
			return fmt.Errorf("marshal google native-tools planner directive: %w", err)
		}
		return onDelta(string(directivePayload))
	}

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil
	}
	return onDelta(content)
}

func streamGoogle(
	ctx context.Context,
	httpClient *http.Client,
	endpoint string,
	modelKey string,
	messages []Message,
	apiKey string,
	onDelta func(delta string) error,
) error {
	chatURL, err := buildGoogleChatURL(endpoint, modelKey)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("google api key is required")
	}

	systemPrompt, payloadContents := buildGoogleContents(messages)
	payload := map[string]any{
		"contents": payloadContents,
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{
					"text": systemPrompt,
				},
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal google payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build google request: %w", err)
	}
	req.Header.Set("x-goog-api-key", strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("google request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("google status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	scanner := bufio.NewScanner(response.Body)
	configureStreamScanner(scanner)

	type googleChunk struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		chunkData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if chunkData == "" || chunkData == "[DONE]" {
			continue
		}

		var chunk googleChunk
		if err := json.Unmarshal([]byte(chunkData), &chunk); err != nil {
			return fmt.Errorf("decode google stream chunk: %w", err)
		}
		for _, candidate := range chunk.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text == "" {
					continue
				}
				if err := onDelta(part.Text); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read google stream: %w", err)
	}
	return nil
}

func buildGoogleChatURL(endpoint string, modelKey string) (string, error) {
	parsed, err := endpointpolicy.ParseAndValidate(strings.TrimSpace(endpoint), endpointpolicy.Options{
		Service: "google provider endpoint",
	})
	if err != nil {
		return "", err
	}

	normalizedModel := strings.TrimSpace(modelKey)
	if normalizedModel == "" {
		return "", fmt.Errorf("google model key is required")
	}
	escapedModel := url.PathEscape(normalizedModel)

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.Contains(path, ":streamGenerateContent"):
		// already full endpoint
	case path == "":
		path = "/v1beta/models/" + escapedModel + ":streamGenerateContent"
	case strings.HasSuffix(path, "/models"):
		path = path + "/" + escapedModel + ":streamGenerateContent"
	case strings.Contains(path, "/models/"):
		path = path + ":streamGenerateContent"
	default:
		path = path + "/models/" + escapedModel + ":streamGenerateContent"
	}
	parsed.Path = path

	query := parsed.Query()
	if strings.TrimSpace(query.Get("alt")) == "" {
		query.Set("alt", "sse")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func buildGoogleContents(messages []Message) (string, []map[string]any) {
	systemSegments := make([]string, 0)
	payloadContents := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			systemSegments = append(systemSegments, content)
		case "assistant", "model":
			payloadContents = append(payloadContents, map[string]any{
				"role": "model",
				"parts": []map[string]string{
					{
						"text": content,
					},
				},
			})
		default:
			payloadContents = append(payloadContents, map[string]any{
				"role": "user",
				"parts": []map[string]string{
					{
						"text": content,
					},
				},
			})
		}
	}
	return strings.Join(systemSegments, "\n\n"), payloadContents
}
