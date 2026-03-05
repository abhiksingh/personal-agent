package channelcheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"personalagent/runtime/internal/endpointpolicy"
)

type TwilioRequest struct {
	Endpoint   string
	AccountSID string
	AuthToken  string
}

type TwilioResult struct {
	Endpoint   string `json:"endpoint"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Message    string `json:"message"`
}

func CheckTwilio(ctx context.Context, client *http.Client, request TwilioRequest) (TwilioResult, error) {
	accountSID := strings.TrimSpace(request.AccountSID)
	authToken := strings.TrimSpace(request.AuthToken)
	if accountSID == "" {
		return TwilioResult{}, fmt.Errorf("twilio account sid is required")
	}
	if authToken == "" {
		return TwilioResult{}, fmt.Errorf("twilio auth token is required")
	}
	if client == nil {
		return TwilioResult{
			Endpoint: strings.TrimSpace(request.Endpoint),
		}, fmt.Errorf("http client is required")
	}

	targetURL, err := buildTwilioProbeURL(request.Endpoint, accountSID)
	if err != nil {
		return TwilioResult{Endpoint: strings.TrimSpace(request.Endpoint)}, err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return TwilioResult{Endpoint: targetURL}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.SetBasicAuth(accountSID, authToken)
	httpRequest.Header.Set("Accept", "application/json")

	start := time.Now()
	response, err := client.Do(httpRequest)
	latencyMS := time.Since(start).Milliseconds()
	if err != nil {
		return TwilioResult{Endpoint: targetURL, LatencyMS: latencyMS}, fmt.Errorf("probe request failed: %w", err)
	}
	defer response.Body.Close()

	result := TwilioResult{
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

func buildTwilioProbeURL(endpoint string, accountSID string) (string, error) {
	base := strings.TrimSpace(endpoint)
	if base == "" {
		base = "https://api.twilio.com"
	}
	base = strings.TrimRight(base, "/")

	parsed, err := endpointpolicy.ParseAndValidate(base, endpointpolicy.Options{Service: "twilio endpoint"})
	if err != nil {
		return "", err
	}

	accountPath := "/2010-04-01/Accounts/" + strings.TrimSpace(accountSID) + ".json"
	switch {
	case parsed.Path == "" || parsed.Path == "/":
		parsed.Path = accountPath
	case strings.HasSuffix(parsed.Path, "/2010-04-01"):
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/Accounts/" + strings.TrimSpace(accountSID) + ".json"
	case strings.Contains(parsed.Path, "/Accounts/"):
		// caller already pointed at account endpoint path
	default:
		parsed.Path = strings.TrimRight(parsed.Path, "/") + accountPath
	}

	return parsed.String(), nil
}
