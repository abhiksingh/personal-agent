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

const defaultEndpoint = "https://api.twilio.com"

type SMSAPIRequest struct {
	Endpoint   string
	AccountSID string
	AuthToken  string
	From       string
	To         string
	Body       string
}

type SMSAPIResponse struct {
	Endpoint     string
	StatusCode   int
	MessageSID   string
	AccountSID   string
	Status       string
	From         string
	To           string
	ErrorCode    int
	ErrorMessage string
	RawBody      string
}

type smsWireResponse struct {
	SID          string `json:"sid"`
	AccountSID   string `json:"account_sid"`
	Status       string `json:"status"`
	From         string `json:"from"`
	To           string `json:"to"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Message      string `json:"message"`
}

func SendSMS(ctx context.Context, client *http.Client, request SMSAPIRequest) (SMSAPIResponse, error) {
	accountSID := strings.TrimSpace(request.AccountSID)
	authToken := strings.TrimSpace(request.AuthToken)
	fromNumber := strings.TrimSpace(request.From)
	toNumber := strings.TrimSpace(request.To)
	messageBody := strings.TrimSpace(request.Body)
	if accountSID == "" {
		return SMSAPIResponse{}, fmt.Errorf("twilio account sid is required")
	}
	if authToken == "" {
		return SMSAPIResponse{}, fmt.Errorf("twilio auth token is required")
	}
	if fromNumber == "" {
		return SMSAPIResponse{}, fmt.Errorf("twilio from number is required")
	}
	if toNumber == "" {
		return SMSAPIResponse{}, fmt.Errorf("twilio destination number is required")
	}
	if messageBody == "" {
		return SMSAPIResponse{}, fmt.Errorf("twilio message body is required")
	}
	if client == nil {
		return SMSAPIResponse{}, fmt.Errorf("http client is required")
	}

	targetURL, err := buildSMSURL(request.Endpoint, accountSID)
	if err != nil {
		return SMSAPIResponse{}, err
	}
	form := url.Values{}
	form.Set("To", toNumber)
	form.Set("From", fromNumber)
	form.Set("Body", messageBody)

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return SMSAPIResponse{}, fmt.Errorf("build twilio sms request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.SetBasicAuth(accountSID, authToken)

	start := time.Now()
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return SMSAPIResponse{}, fmt.Errorf("twilio sms request failed after %s: %w", time.Since(start).Round(time.Millisecond), err)
	}
	defer httpResponse.Body.Close()

	payload, truncated, err := readBoundedTwilioProviderResponseBody(httpResponse.Body)
	if err != nil {
		return SMSAPIResponse{}, fmt.Errorf("read twilio sms response: %w", err)
	}

	wire := smsWireResponse{}
	if len(payload) > 0 && !truncated {
		_ = json.Unmarshal(payload, &wire)
	}

	result := SMSAPIResponse{
		Endpoint:     targetURL,
		StatusCode:   httpResponse.StatusCode,
		MessageSID:   strings.TrimSpace(wire.SID),
		AccountSID:   strings.TrimSpace(wire.AccountSID),
		Status:       strings.TrimSpace(wire.Status),
		From:         strings.TrimSpace(wire.From),
		To:           strings.TrimSpace(wire.To),
		ErrorCode:    wire.ErrorCode,
		ErrorMessage: firstNonEmpty(strings.TrimSpace(wire.ErrorMessage), strings.TrimSpace(wire.Message)),
		RawBody:      strings.TrimSpace(string(payload)),
	}
	if truncated {
		result.ErrorMessage = fmt.Sprintf(
			"twilio sms response exceeded max size of %d bytes",
			twilioProviderResponseBodyLimitBytes,
		)
		return result, fmt.Errorf("%s", result.ErrorMessage)
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		message := firstNonEmpty(result.ErrorMessage, fmt.Sprintf("twilio sms request returned status %d", httpResponse.StatusCode))
		return result, fmt.Errorf("%s", message)
	}
	if result.MessageSID == "" {
		return result, fmt.Errorf("twilio sms response missing sid")
	}
	return result, nil
}

func buildSMSURL(endpoint string, accountSID string) (string, error) {
	base := strings.TrimSpace(endpoint)
	if base == "" {
		base = defaultEndpoint
	}
	parsed, err := endpointpolicy.ParseAndValidate(base, endpointpolicy.Options{Service: "twilio endpoint"})
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, "/2010-04-01/Accounts", strings.TrimSpace(accountSID), "Messages.json")
	return parsed.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
