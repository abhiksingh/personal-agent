package transport

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewTransportDomainErrorNormalizesValues(t *testing.T) {
	err := NewTransportDomainError(0, " resource_conflict ", " ", map[string]any{
		"category": "task_control_state_conflict",
	})
	var domainErr TransportDomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected TransportDomainError, got %T", err)
	}
	if got := domainErr.TransportStatusCode(); got != http.StatusInternalServerError {
		t.Fatalf("expected fallback status 500, got %d", got)
	}
	if got := strings.TrimSpace(domainErr.TransportErrorCode()); got != "resource_conflict" {
		t.Fatalf("expected normalized code resource_conflict, got %q", got)
	}
	if got := strings.TrimSpace(domainErr.Error()); got != "Internal Server Error" {
		t.Fatalf("expected fallback message Internal Server Error, got %q", got)
	}
}

func TestWrapTransportDomainErrorSupportsUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := WrapTransportDomainError(http.StatusNotFound, "resource_not_found", "task run not found", map[string]any{
		"category": "task_control_lookup",
	}, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause to be discoverable")
	}
	var domainErr TransportDomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected TransportDomainError, got %T", err)
	}
	if got := domainErr.TransportStatusCode(); got != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", got)
	}
}
