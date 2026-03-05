package providercheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"personalagent/runtime/internal/endpointpolicy"
	"personalagent/runtime/internal/providerconfig"
)

type Request struct {
	Provider string
	Endpoint string
	APIKey   string
}

type Result struct {
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Message    string `json:"message"`
}

type DiscoverResult struct {
	Provider   string   `json:"provider"`
	Endpoint   string   `json:"endpoint"`
	StatusCode int      `json:"status_code"`
	LatencyMS  int64    `json:"latency_ms"`
	Message    string   `json:"message"`
	Models     []string `json:"models"`
}

func Check(ctx context.Context, client *http.Client, request Request) (Result, error) {
	provider, err := providerconfig.NormalizeProvider(request.Provider)
	if err != nil {
		return Result{}, err
	}
	if client == nil {
		return Result{
			Provider: provider,
			Endpoint: strings.TrimSpace(request.Endpoint),
		}, fmt.Errorf("http client is required")
	}

	targetURL, err := buildProbeURL(provider, request.Endpoint)
	if err != nil {
		return Result{
			Provider: provider,
			Endpoint: strings.TrimSpace(request.Endpoint),
		}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return Result{
			Provider: provider,
			Endpoint: targetURL,
		}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")

	trimmedAPIKey := strings.TrimSpace(request.APIKey)
	if providerconfig.ProviderRequiresAPIKey(provider) && trimmedAPIKey == "" {
		return Result{
			Provider: provider,
			Endpoint: targetURL,
		}, fmt.Errorf("%s provider requires api key", provider)
	}
	if trimmedAPIKey != "" {
		applyProviderAuthHeaders(httpRequest, provider, trimmedAPIKey)
	}
	policyOptions := endpointpolicy.Options{Service: fmt.Sprintf("%s provider endpoint", provider)}
	if _, err := endpointpolicy.ParseAndValidateResolved(ctx, targetURL, policyOptions); err != nil {
		return Result{
			Provider: provider,
			Endpoint: targetURL,
		}, err
	}
	guardedClient := withEndpointPolicyRedirectGuard(client, policyOptions)

	start := time.Now()
	response, err := guardedClient.Do(httpRequest)
	latencyMS := time.Since(start).Milliseconds()
	if err != nil {
		return Result{
			Provider:  provider,
			Endpoint:  targetURL,
			LatencyMS: latencyMS,
		}, fmt.Errorf("probe request failed: %w", err)
	}
	defer response.Body.Close()

	result := Result{
		Provider:   provider,
		Endpoint:   targetURL,
		StatusCode: response.StatusCode,
		LatencyMS:  latencyMS,
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result.Message = "ok"
		return result, nil
	}
	result.Message = fmt.Sprintf("unexpected status %d", response.StatusCode)
	return result, errors.New(result.Message)
}

func Discover(ctx context.Context, client *http.Client, request Request) (DiscoverResult, error) {
	provider, err := providerconfig.NormalizeProvider(request.Provider)
	if err != nil {
		return DiscoverResult{}, err
	}
	if client == nil {
		return DiscoverResult{
			Provider: provider,
			Endpoint: strings.TrimSpace(request.Endpoint),
		}, fmt.Errorf("http client is required")
	}

	targetURL, err := buildProbeURL(provider, request.Endpoint)
	if err != nil {
		return DiscoverResult{
			Provider: provider,
			Endpoint: strings.TrimSpace(request.Endpoint),
		}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return DiscoverResult{
			Provider: provider,
			Endpoint: targetURL,
		}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")

	trimmedAPIKey := strings.TrimSpace(request.APIKey)
	if providerconfig.ProviderRequiresAPIKey(provider) && trimmedAPIKey == "" {
		return DiscoverResult{
			Provider: provider,
			Endpoint: targetURL,
		}, fmt.Errorf("%s provider requires api key", provider)
	}
	if trimmedAPIKey != "" {
		applyProviderAuthHeaders(httpRequest, provider, trimmedAPIKey)
	}
	policyOptions := endpointpolicy.Options{Service: fmt.Sprintf("%s provider endpoint", provider)}
	if _, err := endpointpolicy.ParseAndValidateResolved(ctx, targetURL, policyOptions); err != nil {
		return DiscoverResult{
			Provider: provider,
			Endpoint: targetURL,
		}, err
	}
	guardedClient := withEndpointPolicyRedirectGuard(client, policyOptions)

	start := time.Now()
	response, err := guardedClient.Do(httpRequest)
	latencyMS := time.Since(start).Milliseconds()
	if err != nil {
		return DiscoverResult{
			Provider:  provider,
			Endpoint:  targetURL,
			LatencyMS: latencyMS,
		}, fmt.Errorf("discovery request failed: %w", err)
	}
	defer response.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	result := DiscoverResult{
		Provider:   provider,
		Endpoint:   targetURL,
		StatusCode: response.StatusCode,
		LatencyMS:  latencyMS,
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(string(bodyBytes))
		if message == "" {
			message = fmt.Sprintf("unexpected status %d", response.StatusCode)
		}
		result.Message = message
		return result, errors.New(result.Message)
	}

	models, err := decodeModelKeys(provider, bodyBytes)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.Models = models
	if len(models) == 0 {
		result.Message = "no models discovered"
	} else {
		result.Message = "ok"
	}
	return result, nil
}

func buildProbeURL(provider string, endpoint string) (string, error) {
	base := strings.TrimSpace(endpoint)
	if base == "" {
		base = providerconfig.DefaultEndpoint(provider)
	}
	base = strings.TrimRight(base, "/")

	parsed, err := endpointpolicy.ParseAndValidate(base, endpointpolicy.Options{
		Service: fmt.Sprintf("%s provider endpoint", provider),
	})
	if err != nil {
		return "", err
	}

	switch provider {
	case providerconfig.ProviderOpenAI:
		switch {
		case strings.HasSuffix(parsed.Path, "/models"):
			// already points to models endpoint
		case strings.HasSuffix(parsed.Path, "/v1"):
			parsed.Path = parsed.Path + "/models"
		case parsed.Path == "" || parsed.Path == "/":
			parsed.Path = "/v1/models"
		default:
			parsed.Path = strings.TrimRight(parsed.Path, "/") + "/models"
		}
	case providerconfig.ProviderAnthropic:
		switch {
		case strings.HasSuffix(parsed.Path, "/models"):
			// already points to models endpoint
		case strings.HasSuffix(parsed.Path, "/v1"):
			parsed.Path = parsed.Path + "/models"
		case parsed.Path == "" || parsed.Path == "/":
			parsed.Path = "/v1/models"
		default:
			parsed.Path = strings.TrimRight(parsed.Path, "/") + "/models"
		}
	case providerconfig.ProviderGoogle:
		switch {
		case strings.HasSuffix(parsed.Path, "/models"):
			// already points to models endpoint
		case parsed.Path == "" || parsed.Path == "/":
			parsed.Path = "/v1beta/models"
		default:
			parsed.Path = strings.TrimRight(parsed.Path, "/") + "/models"
		}
	case providerconfig.ProviderOllama:
		switch {
		case strings.HasSuffix(parsed.Path, "/api/tags"):
			// already points to tags endpoint
		case parsed.Path == "" || parsed.Path == "/":
			parsed.Path = "/api/tags"
		default:
			parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/tags"
		}
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}

	return parsed.String(), nil
}

func applyProviderAuthHeaders(request *http.Request, provider string, apiKey string) {
	switch provider {
	case providerconfig.ProviderAnthropic:
		request.Header.Set("x-api-key", apiKey)
		request.Header.Set("anthropic-version", "2023-06-01")
	case providerconfig.ProviderGoogle:
		request.Header.Set("x-goog-api-key", apiKey)
	default:
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func withEndpointPolicyRedirectGuard(client *http.Client, options endpointpolicy.Options) *http.Client {
	guarded := *client
	existingCheckRedirect := client.CheckRedirect
	guarded.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if _, err := endpointpolicy.ParseAndValidateResolved(request.Context(), request.URL.String(), options); err != nil {
			return err
		}
		if existingCheckRedirect != nil {
			return existingCheckRedirect(request, via)
		}
		return nil
	}
	return &guarded
}

func decodeModelKeys(provider string, payload []byte) ([]string, error) {
	type discoveryObject map[string]any

	collectFromItem := func(item any, out map[string]struct{}) {
		switch value := item.(type) {
		case string:
			key := normalizeDiscoveredModelKey(provider, value)
			if key != "" {
				out[key] = struct{}{}
			}
		case discoveryObject:
			addFromDiscoveryObject(provider, value, out)
		case map[string]any:
			addFromDiscoveryObject(provider, discoveryObject(value), out)
		}
	}

	collectFromArray := func(items []any, out map[string]struct{}) {
		for _, item := range items {
			collectFromItem(item, out)
		}
	}

	seen := map[string]struct{}{}

	var object map[string]any
	if err := json.Unmarshal(payload, &object); err == nil {
		if data, ok := object["data"].([]any); ok {
			collectFromArray(data, seen)
		}
		if models, ok := object["models"].([]any); ok {
			collectFromArray(models, seen)
		}
		if results, ok := object["results"].([]any); ok {
			collectFromArray(results, seen)
		}
	}

	if len(seen) == 0 {
		var array []any
		if err := json.Unmarshal(payload, &array); err == nil {
			collectFromArray(array, seen)
		}
	}

	models := make([]string, 0, len(seen))
	for model := range seen {
		models = append(models, model)
	}
	slices.Sort(models)
	if len(models) == 0 {
		return nil, fmt.Errorf("no model identifiers found in discovery payload")
	}
	return models, nil
}

func addFromDiscoveryObject(provider string, object map[string]any, out map[string]struct{}) {
	for _, key := range []string{"id", "model", "name"} {
		if raw, ok := object[key]; ok {
			normalized := normalizeDiscoveredModelKey(provider, fmt.Sprintf("%v", raw))
			if normalized != "" {
				out[normalized] = struct{}{}
				return
			}
		}
	}
}

func normalizeDiscoveredModelKey(provider string, raw string) string {
	modelKey := strings.TrimSpace(raw)
	if modelKey == "" {
		return ""
	}
	if provider == providerconfig.ProviderGoogle && strings.HasPrefix(modelKey, "models/") {
		modelKey = strings.TrimPrefix(modelKey, "models/")
	}
	return strings.TrimSpace(modelKey)
}
