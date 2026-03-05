package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"personalagent/runtime/internal/endpointpolicy"
)

type VoiceCallRequest struct {
	Endpoint   string
	AccountSID string
	AuthToken  string
	From       string
	To         string
	TwiMLURL   string
}

type VoiceCallResponse struct {
	Endpoint     string
	StatusCode   int
	CallSID      string
	AccountSID   string
	Status       string
	Direction    string
	From         string
	To           string
	ErrorCode    int
	ErrorMessage string
	RawBody      string
}

type voiceCallWireResponse struct {
	SID          string `json:"sid"`
	AccountSID   string `json:"account_sid"`
	Status       string `json:"status"`
	Direction    string `json:"direction"`
	From         string `json:"from"`
	To           string `json:"to"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Message      string `json:"message"`
}

func StartVoiceCall(ctx context.Context, client *http.Client, request VoiceCallRequest) (VoiceCallResponse, error) {
	accountSID := strings.TrimSpace(request.AccountSID)
	authToken := strings.TrimSpace(request.AuthToken)
	fromNumber := strings.TrimSpace(request.From)
	toNumber := strings.TrimSpace(request.To)
	twimlURL := strings.TrimSpace(request.TwiMLURL)
	if accountSID == "" {
		return VoiceCallResponse{}, fmt.Errorf("twilio account sid is required")
	}
	if authToken == "" {
		return VoiceCallResponse{}, fmt.Errorf("twilio auth token is required")
	}
	if fromNumber == "" {
		return VoiceCallResponse{}, fmt.Errorf("twilio from number is required")
	}
	if toNumber == "" {
		return VoiceCallResponse{}, fmt.Errorf("twilio destination number is required")
	}
	if twimlURL == "" {
		return VoiceCallResponse{}, fmt.Errorf("twiml url is required")
	}
	if client == nil {
		return VoiceCallResponse{}, fmt.Errorf("http client is required")
	}

	targetURL, err := buildVoiceCallsURL(request.Endpoint, accountSID)
	if err != nil {
		return VoiceCallResponse{}, err
	}
	form := url.Values{}
	form.Set("To", toNumber)
	form.Set("From", fromNumber)
	form.Set("Url", twimlURL)

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return VoiceCallResponse{}, fmt.Errorf("build twilio voice call request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.SetBasicAuth(accountSID, authToken)

	start := time.Now()
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return VoiceCallResponse{}, fmt.Errorf("twilio voice call request failed after %s: %w", time.Since(start).Round(time.Millisecond), err)
	}
	defer httpResponse.Body.Close()

	payload, truncated, err := readBoundedTwilioProviderResponseBody(httpResponse.Body)
	if err != nil {
		return VoiceCallResponse{}, fmt.Errorf("read twilio voice call response: %w", err)
	}

	wire := voiceCallWireResponse{}
	if len(payload) > 0 && !truncated {
		_ = json.Unmarshal(payload, &wire)
	}

	result := VoiceCallResponse{
		Endpoint:     targetURL,
		StatusCode:   httpResponse.StatusCode,
		CallSID:      strings.TrimSpace(wire.SID),
		AccountSID:   strings.TrimSpace(wire.AccountSID),
		Status:       strings.TrimSpace(wire.Status),
		Direction:    strings.TrimSpace(wire.Direction),
		From:         strings.TrimSpace(wire.From),
		To:           strings.TrimSpace(wire.To),
		ErrorCode:    wire.ErrorCode,
		ErrorMessage: firstNonEmpty(strings.TrimSpace(wire.ErrorMessage), strings.TrimSpace(wire.Message)),
		RawBody:      strings.TrimSpace(string(payload)),
	}
	if truncated {
		result.ErrorMessage = fmt.Sprintf(
			"twilio voice call response exceeded max size of %d bytes",
			twilioProviderResponseBodyLimitBytes,
		)
		return result, fmt.Errorf("%s", result.ErrorMessage)
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		message := firstNonEmpty(result.ErrorMessage, fmt.Sprintf("twilio voice call request returned status %d", httpResponse.StatusCode))
		return result, fmt.Errorf("%s", message)
	}
	if result.CallSID == "" {
		return result, fmt.Errorf("twilio voice call response missing sid")
	}
	return result, nil
}

func buildVoiceCallsURL(endpoint string, accountSID string) (string, error) {
	base := strings.TrimSpace(endpoint)
	if base == "" {
		base = defaultEndpoint
	}
	parsed, err := endpointpolicy.ParseAndValidate(base, endpointpolicy.Options{Service: "twilio endpoint"})
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, "/2010-04-01/Accounts", strings.TrimSpace(accountSID), "Calls.json")
	return parsed.String(), nil
}
