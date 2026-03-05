package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type transportErrorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type transportErrorEnvelope struct {
	Error         transportErrorObject `json:"error"`
	CorrelationID string               `json:"correlation_id,omitempty"`
	Type          string               `json:"type"`
	Title         string               `json:"title"`
	Status        int                  `json:"status"`
	Detail        string               `json:"detail"`
	Instance      string               `json:"instance"`
}

type transportErrorEnvelopeDecode struct {
	Error         json.RawMessage `json:"error"`
	CorrelationID string          `json:"correlation_id"`
	Type          string          `json:"type"`
	Title         string          `json:"title"`
	Status        int             `json:"status"`
	Detail        string          `json:"detail"`
	Instance      string          `json:"instance"`
}

type transportErrorObjectDecode struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

var unconfiguredServiceDomainByID = map[string]string{
	"secret_reference":      "secrets",
	"provider":              "providers",
	"model":                 "models",
	"chat":                  "chat",
	"agent":                 "agent",
	"delegation":            "delegation",
	"comm":                  "comm",
	"twilio_channel":        "channels.twilio",
	"cloudflared_connector": "connectors.cloudflared",
	"automation":            "automation",
	"inspect":               "inspect",
	"retention":             "retention",
	"context":               "context",
	"daemon_lifecycle":      "daemon.lifecycle",
	"workflow_query":        "workflow.query",
	"ui_status":             "ui.status",
	"identity_directory":    "identity",
}

var unconfiguredServiceConfigFieldByID = map[string]string{
	"secret_reference":      "SecretReferences",
	"provider":              "Providers",
	"model":                 "Models",
	"chat":                  "Chat",
	"agent":                 "Agent",
	"delegation":            "Delegation",
	"comm":                  "Comm",
	"twilio_channel":        "Twilio",
	"cloudflared_connector": "Cloudflared",
	"automation":            "Automation",
	"inspect":               "Inspect",
	"retention":             "Retention",
	"context":               "ContextOps",
	"daemon_lifecycle":      "DaemonLifecycle",
	"workflow_query":        "WorkflowQueries",
	"ui_status":             "UIStatus",
	"identity_directory":    "IdentityDirectory",
}

func defaultTransportErrorCode(statusCode int, message string) string {
	normalizedMessage := strings.ToLower(strings.TrimSpace(message))
	switch statusCode {
	case http.StatusBadRequest:
		if strings.Contains(normalizedMessage, " is required") {
			return "missing_required_field"
		}
		if strings.HasPrefix(normalizedMessage, "invalid ") {
			return "invalid_request_payload"
		}
		return "invalid_request"
	case http.StatusUnauthorized:
		return "auth_unauthorized"
	case http.StatusForbidden:
		return "auth_forbidden"
	case http.StatusRequestEntityTooLarge:
		return "request_payload_too_large"
	case http.StatusTooManyRequests:
		return "rate_limit_exceeded"
	case http.StatusNotFound:
		return "resource_not_found"
	case http.StatusConflict:
		return "resource_conflict"
	case http.StatusNotImplemented:
		if _, _, _, _, ok := parseServiceNotConfiguredMessage(message); ok {
			return "service_not_configured"
		}
		return "not_implemented"
	default:
		if statusCode >= 500 {
			return "internal_error"
		}
		return "request_failed"
	}
}

func defaultTransportErrorDetails(statusCode int, message string) any {
	if statusCode != http.StatusNotImplemented {
		return nil
	}
	serviceID, serviceLabel, serviceDomain, configField, ok := parseServiceNotConfiguredMessage(message)
	if !ok {
		return nil
	}
	return map[string]any{
		"category": "service_not_configured",
		"service": map[string]any{
			"id":           serviceID,
			"label":        serviceLabel,
			"config_field": configField,
		},
		"domain": serviceDomain,
		"remediation": map[string]any{
			"action": "configure_server_service",
			"label":  "Configure Service Dependency",
			"hint":   fmt.Sprintf("Set ServerConfig.%s with a non-nil implementation before calling this endpoint.", configField),
		},
	}
}

func parseServiceNotConfiguredMessage(message string) (serviceID string, serviceLabel string, serviceDomain string, configField string, ok bool) {
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)
	const suffix = " service is not configured"
	if !strings.HasSuffix(lower, suffix) {
		return "", "", "", "", false
	}
	rawService := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
	if rawService == "" {
		return "", "", "", "", false
	}
	serviceID = normalizeErrorToken(rawService)
	if serviceID == "" {
		return "", "", "", "", false
	}
	serviceLabel = rawService + " service"
	serviceDomain = firstNonEmpty(unconfiguredServiceDomainByID[serviceID], serviceID)
	configField = firstNonEmpty(unconfiguredServiceConfigFieldByID[serviceID], "UnknownService")
	return serviceID, serviceLabel, serviceDomain, configField, true
}

func normalizeErrorToken(raw string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(raw)), func(r rune) bool {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= '0' && r <= '9' {
			return false
		}
		return true
	})
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "_")
}

func buildTransportErrorEnvelope(statusCode int, message string, correlationID string, details any) transportErrorEnvelope {
	return buildTransportErrorEnvelopeWithCode(statusCode, message, correlationID, "", details)
}

func buildTransportErrorEnvelopeWithCode(statusCode int, message string, correlationID string, code string, details any) transportErrorEnvelope {
	trimmedMessage := strings.TrimSpace(message)
	errorCode := strings.TrimSpace(code)
	if errorCode == "" {
		errorCode = defaultTransportErrorCode(statusCode, trimmedMessage)
	}
	trimmedCorrelationID := strings.TrimSpace(correlationID)
	return transportErrorEnvelope{
		Error: transportErrorObject{
			Code:    errorCode,
			Message: trimmedMessage,
			Details: details,
		},
		CorrelationID: trimmedCorrelationID,
		Type:          buildTransportProblemTypeURI(errorCode),
		Title:         firstNonEmpty(strings.TrimSpace(http.StatusText(statusCode)), "Request Failed"),
		Status:        statusCode,
		Detail:        trimmedMessage,
		Instance:      buildTransportProblemInstance(trimmedCorrelationID),
	}
}

func buildTransportProblemTypeURI(errorCode string) string {
	trimmed := strings.TrimSpace(errorCode)
	if trimmed == "" {
		trimmed = "request_failed"
	}
	return "https://personalagent.dev/problems/" + trimmed
}

func parseTransportProblemTypeCode(problemType string) string {
	trimmed := strings.TrimSpace(problemType)
	if trimmed == "" {
		return ""
	}
	slashIndex := strings.LastIndex(trimmed, "/")
	if slashIndex >= 0 && slashIndex+1 < len(trimmed) {
		return strings.TrimSpace(trimmed[slashIndex+1:])
	}
	return ""
}

func buildTransportProblemInstance(correlationID string) string {
	trimmed := strings.TrimSpace(correlationID)
	if trimmed == "" {
		return ""
	}
	return "/v1/errors/" + url.PathEscape(trimmed)
}

func parseTransportHTTPError(statusCode int, body []byte, fallbackCorrelationID string) HTTPError {
	bodyText := strings.TrimSpace(string(body))
	parsed := HTTPError{
		StatusCode:     statusCode,
		Body:           string(body),
		CorrelationID:  strings.TrimSpace(fallbackCorrelationID),
		Code:           defaultTransportErrorCode(statusCode, bodyText),
		Message:        bodyText,
		DetailsPayload: nil,
	}

	var envelope transportErrorEnvelopeDecode
	if err := json.Unmarshal(body, &envelope); err != nil {
		if parsed.Message == "" {
			parsed.Message = strings.TrimSpace(http.StatusText(statusCode))
		}
		return parsed
	}

	if strings.TrimSpace(parsed.CorrelationID) == "" {
		parsed.CorrelationID = strings.TrimSpace(envelope.CorrelationID)
	}

	if len(envelope.Error) > 0 {
		var object transportErrorObjectDecode
		if decodeErr := json.Unmarshal(envelope.Error, &object); decodeErr == nil {
			if strings.TrimSpace(object.Code) != "" {
				parsed.Code = strings.TrimSpace(object.Code)
			}
			if strings.TrimSpace(object.Message) != "" {
				parsed.Message = strings.TrimSpace(object.Message)
			}
			if len(object.Details) > 0 {
				parsed.DetailsPayload = object.Details
			}
		}
	}

	if strings.TrimSpace(envelope.Type) != "" && strings.TrimSpace(parsed.Code) == "" {
		parsed.Code = parseTransportProblemTypeCode(envelope.Type)
	}
	if strings.TrimSpace(envelope.Detail) != "" {
		parsed.Message = strings.TrimSpace(envelope.Detail)
	}
	if strings.TrimSpace(parsed.Message) == "" && strings.TrimSpace(envelope.Title) != "" {
		parsed.Message = strings.TrimSpace(envelope.Title)
	}

	if strings.TrimSpace(parsed.Message) == "" {
		parsed.Message = strings.TrimSpace(http.StatusText(statusCode))
	}
	if strings.TrimSpace(parsed.Code) == "" {
		parsed.Code = defaultTransportErrorCode(statusCode, parsed.Message)
	}

	return parsed
}
