package transport

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(writer http.ResponseWriter, statusCode int, payload any, correlationID string) {
	writeJSONWithContentType(writer, statusCode, payload, correlationID, responseContentTypeJSON)
}

func writeJSONWithContentType(writer http.ResponseWriter, statusCode int, payload any, correlationID string, contentType string) {
	trimmedContentType := strings.TrimSpace(contentType)
	if trimmedContentType == "" {
		trimmedContentType = responseContentTypeJSON
	}
	writer.Header().Set("Content-Type", trimmedContentType)
	writer.Header().Set(responseHeaderAPIVersion, responseHeaderCurrentAPIVer)
	if strings.TrimSpace(correlationID) != "" {
		writer.Header().Set(responseHeaderCorrelationID, correlationID)
	}
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeJSONErrorWithDetails(writer http.ResponseWriter, statusCode int, message string, correlationID string, details any) {
	writeJSONErrorWithDetailsAndCode(writer, statusCode, message, correlationID, details, "")
}

func writeJSONErrorWithDetailsAndCode(writer http.ResponseWriter, statusCode int, message string, correlationID string, details any, code string) {
	writeJSONWithContentType(
		writer,
		statusCode,
		buildTransportErrorEnvelopeWithCode(statusCode, message, correlationID, code, details),
		correlationID,
		responseContentTypeProblem,
	)
}

func writeJSONErrorFromError(writer http.ResponseWriter, fallbackStatusCode int, err error, correlationID string) {
	if err == nil {
		writeJSONErrorWithDetailsAndCode(
			writer,
			http.StatusInternalServerError,
			redactedTransportInternalMessage(),
			correlationID,
			defaultInternalErrorDetails(),
			"internal_error",
		)
		return
	}

	statusCode := fallbackStatusCode
	if statusCode <= 0 {
		statusCode = http.StatusInternalServerError
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = strings.TrimSpace(http.StatusText(statusCode))
	}
	code := ""
	var details any
	isDomainError := false

	if domainErr, ok := transportDomainErrorFrom(err); ok {
		isDomainError = true
		statusCode = domainErr.TransportStatusCode()
		if statusCode <= 0 {
			statusCode = fallbackStatusCode
		}
		if statusCode <= 0 {
			statusCode = http.StatusInternalServerError
		}
		code = strings.TrimSpace(domainErr.TransportErrorCode())
		details = domainErr.TransportErrorDetails()
	}
	if details == nil {
		details = defaultTransportErrorDetails(statusCode, message)
	}
	if statusCode >= http.StatusInternalServerError {
		message = redactedTransportInternalMessage()
		if strings.TrimSpace(code) == "" {
			code = "internal_error"
		}
		if !isDomainError || details == nil {
			details = defaultInternalErrorDetails()
		}
	}
	writeJSONErrorWithDetailsAndCode(writer, statusCode, message, correlationID, details, code)
}

func redactedTransportInternalMessage() string {
	return "internal server error"
}

func defaultInternalErrorDetails() map[string]any {
	return map[string]any{
		"category": "internal_error",
		"redacted": true,
	}
}
