package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSONErrorFromErrorUsesTransportDomainErrorMapping(t *testing.T) {
	err := NewTransportDomainError(http.StatusConflict, "resource_conflict", "task is already terminal", map[string]any{
		"category": "task_control_state_conflict",
	})

	recorder := httptest.NewRecorder()
	writeJSONErrorFromError(recorder, http.StatusBadRequest, err, "corr-domain")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
	parsed := parseTransportHTTPError(recorder.Code, recorder.Body.Bytes(), "corr-domain")
	if parsed.Code != "resource_conflict" {
		t.Fatalf("expected code resource_conflict, got %q", parsed.Code)
	}
	if parsed.Message != "task is already terminal" {
		t.Fatalf("expected domain message, got %q", parsed.Message)
	}
	details := decodeRawDetailsMap(t, parsed.DetailsPayload)
	if got := strings.TrimSpace(toString(details["category"])); got != "task_control_state_conflict" {
		t.Fatalf("expected details category task_control_state_conflict, got %q", got)
	}
}

func TestWriteJSONErrorFromErrorRedactsInternalFailureMessages(t *testing.T) {
	err := errors.New("dial tcp 10.0.0.9:5432: permission denied")

	recorder := httptest.NewRecorder()
	writeJSONErrorFromError(recorder, http.StatusInternalServerError, err, "corr-redact")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	parsed := parseTransportHTTPError(recorder.Code, recorder.Body.Bytes(), "corr-redact")
	if parsed.Code != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", parsed.Code)
	}
	if parsed.Message != "internal server error" {
		t.Fatalf("expected redacted internal message, got %q", parsed.Message)
	}
	if strings.Contains(recorder.Body.String(), "10.0.0.9") {
		t.Fatalf("expected internal payload to be redacted, body=%q", recorder.Body.String())
	}
	details := decodeRawDetailsMap(t, parsed.DetailsPayload)
	if got := strings.TrimSpace(toString(details["category"])); got != "internal_error" {
		t.Fatalf("expected details category internal_error, got %q", got)
	}
	if redacted, ok := details["redacted"].(bool); !ok || !redacted {
		t.Fatalf("expected details.redacted=true, got %v", details["redacted"])
	}
}

func TestWriteJSONErrorFromErrorRedactsNilErrorAsInternalFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSONErrorFromError(recorder, http.StatusBadRequest, nil, "corr-nil")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	parsed := parseTransportHTTPError(recorder.Code, recorder.Body.Bytes(), "corr-nil")
	if parsed.Code != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", parsed.Code)
	}
	if parsed.Message != "internal server error" {
		t.Fatalf("expected redacted internal message, got %q", parsed.Message)
	}
}

func TestWriteJSONErrorFromErrorKeepsClientActionableBadRequestMessage(t *testing.T) {
	err := errors.New("workspace id is required")

	recorder := httptest.NewRecorder()
	writeJSONErrorFromError(recorder, http.StatusBadRequest, err, "corr-400")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	parsed := parseTransportHTTPError(recorder.Code, recorder.Body.Bytes(), "corr-400")
	if parsed.Message != "workspace id is required" {
		t.Fatalf("expected actionable bad-request message, got %q", parsed.Message)
	}
	if parsed.Code != "missing_required_field" {
		t.Fatalf("expected missing_required_field code, got %q", parsed.Code)
	}
}

func decodeRawDetailsMap(t *testing.T, payload json.RawMessage) map[string]any {
	t.Helper()
	if len(payload) == 0 {
		t.Fatalf("expected details payload")
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode details payload: %v", err)
	}
	return decoded
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
