package chatruntime

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"

	"personalagent/runtime/internal/providerconfig"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

type StreamRequest struct {
	Provider                string
	Endpoint                string
	ModelKey                string
	APIKey                  string
	Messages                []Message
	ToolSpecs               []ToolSpec
	PreferNativeToolCalling bool
}

const (
	defaultStreamScannerInitialBufferBytes = 4 * 1024
	defaultStreamScannerMaxBufferBytes     = 16 * 1024 * 1024
	defaultAnthropicMaxTokens              = 4096
)

func configureStreamScanner(scanner *bufio.Scanner) {
	scanner.Buffer(
		make([]byte, defaultStreamScannerInitialBufferBytes),
		defaultStreamScannerMaxBufferBytes,
	)
}

func StreamAssistantResponse(
	ctx context.Context,
	httpClient *http.Client,
	request StreamRequest,
	onDelta func(delta string) error,
) error {
	if httpClient == nil {
		return fmt.Errorf("http client is required")
	}
	if onDelta == nil {
		return fmt.Errorf("onDelta callback is required")
	}

	provider, err := providerconfig.NormalizeProvider(request.Provider)
	if err != nil {
		return err
	}
	endpoint := strings.TrimSpace(request.Endpoint)
	if endpoint == "" {
		endpoint = providerconfig.DefaultEndpoint(provider)
	}

	switch provider {
	case providerconfig.ProviderOpenAI:
		if request.PreferNativeToolCalling && len(request.ToolSpecs) > 0 {
			return streamOpenAINativeTools(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.ToolSpecs, request.APIKey, onDelta)
		}
		return streamOpenAI(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.APIKey, onDelta)
	case providerconfig.ProviderAnthropic:
		if request.PreferNativeToolCalling && len(request.ToolSpecs) > 0 {
			return streamAnthropicNativeTools(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.ToolSpecs, request.APIKey, onDelta)
		}
		return streamAnthropic(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.APIKey, onDelta)
	case providerconfig.ProviderGoogle:
		if request.PreferNativeToolCalling && len(request.ToolSpecs) > 0 {
			return streamGoogleNativeTools(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.ToolSpecs, request.APIKey, onDelta)
		}
		return streamGoogle(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.APIKey, onDelta)
	case providerconfig.ProviderOllama:
		if request.PreferNativeToolCalling && len(request.ToolSpecs) > 0 {
			return streamOllamaNativeTools(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.ToolSpecs, request.APIKey, onDelta)
		}
		return streamOllama(ctx, httpClient, endpoint, request.ModelKey, request.Messages, request.APIKey, onDelta)
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
}
