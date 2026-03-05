package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrInvalidSignature = errors.New("invalid twilio signature")

func ComputeRequestSignature(authToken string, requestURL string, params map[string]string) (string, error) {
	token := strings.TrimSpace(authToken)
	if token == "" {
		return "", fmt.Errorf("twilio auth token is required")
	}
	urlValue := strings.TrimSpace(requestURL)
	if urlValue == "" {
		return "", fmt.Errorf("request URL is required")
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString(urlValue)
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(params[key])
	}

	mac := hmac.New(sha1.New, []byte(token))
	if _, err := mac.Write([]byte(builder.String())); err != nil {
		return "", fmt.Errorf("compute twilio signature digest: %w", err)
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func ValidateRequestSignature(authToken string, requestURL string, params map[string]string, signature string) error {
	expected, err := ComputeRequestSignature(authToken, requestURL, params)
	if err != nil {
		return err
	}
	received := strings.TrimSpace(signature)
	if received == "" {
		return fmt.Errorf("%w: missing signature", ErrInvalidSignature)
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(received)) != 1 {
		return fmt.Errorf("%w: signature mismatch", ErrInvalidSignature)
	}
	return nil
}
