package transport

import (
	"errors"
	"net/http"
	"strings"
)

// TransportDomainError exposes deterministic transport envelope metadata.
type TransportDomainError interface {
	error
	TransportStatusCode() int
	TransportErrorCode() string
	TransportErrorDetails() any
}

type transportDomainError struct {
	statusCode int
	code       string
	message    string
	details    any
	cause      error
}

func (e *transportDomainError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *transportDomainError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *transportDomainError) TransportStatusCode() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	if e.statusCode <= 0 {
		return http.StatusInternalServerError
	}
	return e.statusCode
}

func (e *transportDomainError) TransportErrorCode() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.code)
}

func (e *transportDomainError) TransportErrorDetails() any {
	if e == nil {
		return nil
	}
	return e.details
}

func NewTransportDomainError(statusCode int, code string, message string, details any) error {
	normalizedStatus := statusCode
	if normalizedStatus <= 0 {
		normalizedStatus = http.StatusInternalServerError
	}
	normalizedMessage := strings.TrimSpace(message)
	if normalizedMessage == "" {
		normalizedMessage = strings.TrimSpace(http.StatusText(normalizedStatus))
	}
	return &transportDomainError{
		statusCode: normalizedStatus,
		code:       strings.TrimSpace(code),
		message:    normalizedMessage,
		details:    details,
	}
}

func WrapTransportDomainError(statusCode int, code string, message string, details any, cause error) error {
	domainErr := NewTransportDomainError(statusCode, code, message, details)
	typed, _ := domainErr.(*transportDomainError)
	if typed == nil {
		return domainErr
	}
	typed.cause = cause
	return typed
}

func transportDomainErrorFrom(err error) (TransportDomainError, bool) {
	var domainErr TransportDomainError
	if err == nil {
		return nil, false
	}
	if !errors.As(err, &domainErr) {
		return nil, false
	}
	return domainErr, true
}
