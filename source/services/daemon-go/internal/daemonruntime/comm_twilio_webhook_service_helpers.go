package daemonruntime

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func resolveWebhookStatusCode(statusCode int) int {
	if statusCode <= 0 {
		return http.StatusBadRequest
	}
	return statusCode
}

func writeWebhookJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeWebhookTwiML(writer http.ResponseWriter, statusCode int, twimlBody string) {
	payload := strings.TrimSpace(twimlBody)
	if payload == "" {
		payload = buildTwiMLEmptyResponse()
	}
	writer.Header().Set("Content-Type", "application/xml")
	writer.WriteHeader(statusCode)
	_, _ = io.WriteString(writer, payload)
}

func newTwilioWebhookHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: defaultTwilioWebhookReadHeaderTimeout,
		ReadTimeout:       defaultTwilioWebhookReadTimeout,
		WriteTimeout:      defaultTwilioWebhookWriteTimeout,
		IdleTimeout:       defaultTwilioWebhookIdleTimeout,
	}
}

func parseTwilioWebhookForm(writer http.ResponseWriter, request *http.Request) (int, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, defaultTwilioWebhookMaxRequestBytes)
	if err := request.ParseForm(); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return http.StatusRequestEntityTooLarge, fmt.Errorf(
				"request body exceeds %d bytes",
				defaultTwilioWebhookMaxRequestBytes,
			)
		}
		return http.StatusBadRequest, err
	}
	if len(request.PostForm) > defaultTwilioWebhookMaxFormFields {
		return http.StatusRequestEntityTooLarge, fmt.Errorf(
			"request form field count exceeds limit %d",
			defaultTwilioWebhookMaxFormFields,
		)
	}
	return http.StatusOK, nil
}

func defaultDaemonTwilioWebhookSMSPath() string {
	return fmt.Sprintf("/%s/%s/connector/twilio/sms", resolveDaemonTwilioWebhookProjectName(), twilioWebhookAPIVersion)
}

func defaultDaemonTwilioWebhookVoicePath() string {
	return fmt.Sprintf("/%s/%s/connector/twilio/voice", resolveDaemonTwilioWebhookProjectName(), twilioWebhookAPIVersion)
}

func resolveDaemonTwilioWebhookProjectName() string {
	if override := strings.TrimSpace(os.Getenv(twilioWebhookProjectNameEnvKey)); override != "" {
		return normalizeDaemonTwilioWebhookProjectName(override)
	}
	executablePath, err := os.Executable()
	if err == nil && strings.TrimSpace(executablePath) != "" {
		return normalizeDaemonTwilioWebhookProjectName(filepath.Base(executablePath))
	}
	return normalizeDaemonTwilioWebhookProjectName(filepath.Base(os.Args[0]))
}

func normalizeDaemonTwilioWebhookProjectName(raw string) string {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	if candidate == "" {
		return twilioWebhookDefaultProjectName
	}
	candidate = strings.TrimSuffix(candidate, filepath.Ext(candidate))
	candidate = strings.TrimSuffix(candidate, "-daemon")
	candidate = strings.TrimSuffix(candidate, "_daemon")
	candidate = strings.TrimSuffix(candidate, "daemon")

	builder := strings.Builder{}
	for _, r := range candidate {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		}
	}
	normalized := strings.TrimSpace(builder.String())
	if normalized == "" {
		return twilioWebhookDefaultProjectName
	}
	return normalized
}

func normalizeDaemonWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func webhookFormToMap(values url.Values) map[string]string {
	out := map[string]string{}
	for key, list := range values {
		if len(list) == 0 {
			continue
		}
		out[key] = list[0]
	}
	return out
}

func parseWebhookTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func normalizeTwilioAssistantOperationID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		generated, err := daemonRandomID()
		if err != nil {
			return "twilio-assistant-fallback"
		}
		return "twilio-assistant-" + generated
	}
	return trimmed
}

func isTerminalVoiceCallStatus(status string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(status)), "-", "_")
	switch normalized {
	case "completed", "failed", "no_answer", "busy", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func buildTwiMLGatherResponse(actionURL string, sayText string, fallbackText string) string {
	action := xmlEscapeText(strings.TrimSpace(actionURL))
	prompt := xmlEscapeText(strings.TrimSpace(sayText))
	fallback := xmlEscapeText(strings.TrimSpace(fallbackText))
	if fallback == "" {
		fallback = xmlEscapeText(defaultTwilioWebhookVoiceFallback)
	}
	return fmt.Sprintf(
		"<Response><Gather input=\"speech\" speechTimeout=\"auto\" method=\"POST\" action=\"%s\"><Say>%s</Say></Gather><Say>%s</Say></Response>",
		action,
		prompt,
		fallback,
	)
}

func buildTwiMLEmptyResponse() string {
	return "<Response></Response>"
}

func xmlEscapeText(value string) string {
	buffer := &bytes.Buffer{}
	_ = xml.EscapeText(buffer, []byte(value))
	return buffer.String()
}
