package transport

import (
	"encoding/json"
	"testing"
)

func TestBuildTransportErrorEnvelopeIncludesTypedProblemFields(t *testing.T) {
	envelope := buildTransportErrorEnvelope(400, "task id is required", "corr-123", map[string]any{
		"field": "task_id",
	})

	if envelope.Error.Code != "missing_required_field" {
		t.Fatalf("expected missing_required_field code, got %q", envelope.Error.Code)
	}
	if envelope.Error.Message != "task id is required" {
		t.Fatalf("expected message, got %q", envelope.Error.Message)
	}
	if envelope.CorrelationID != "corr-123" {
		t.Fatalf("expected correlation id corr-123, got %q", envelope.CorrelationID)
	}
	if envelope.Type != "https://personalagent.dev/problems/missing_required_field" {
		t.Fatalf("expected RFC problem type uri, got %q", envelope.Type)
	}
	if envelope.Title != "Bad Request" {
		t.Fatalf("expected RFC problem title Bad Request, got %q", envelope.Title)
	}
	if envelope.Status != 400 {
		t.Fatalf("expected RFC problem status 400, got %d", envelope.Status)
	}
	if envelope.Detail != "task id is required" {
		t.Fatalf("expected RFC problem detail message, got %q", envelope.Detail)
	}
	if envelope.Instance != "/v1/errors/corr-123" {
		t.Fatalf("expected RFC problem instance /v1/errors/corr-123, got %q", envelope.Instance)
	}
}

func TestBuildTransportErrorEnvelopeWithCodeOverridesDefaultMapping(t *testing.T) {
	envelope := buildTransportErrorEnvelopeWithCode(
		400,
		"task run state \"running\" is not retryable",
		"corr-override",
		"resource_conflict",
		map[string]any{"category": "task_control_state_conflict"},
	)

	if envelope.Error.Code != "resource_conflict" {
		t.Fatalf("expected overridden error code resource_conflict, got %q", envelope.Error.Code)
	}
	if envelope.Type != "https://personalagent.dev/problems/resource_conflict" {
		t.Fatalf("expected problem type to use overridden code, got %q", envelope.Type)
	}
}

func TestParseTransportHTTPErrorTypedEnvelope(t *testing.T) {
	envelope := transportErrorEnvelope{
		Error: transportErrorObject{
			Code:    "resource_not_found",
			Message: "task not found",
			Details: map[string]any{"task_id": "task-1"},
		},
		CorrelationID: "corr-body",
		Type:          "https://personalagent.dev/problems/resource_not_found",
		Title:         "Not Found",
		Status:        404,
		Detail:        "task not found",
		Instance:      "/v1/errors/corr-body",
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	parsed := parseTransportHTTPError(404, body, "corr-header")
	if parsed.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d", parsed.StatusCode)
	}
	if parsed.Code != "resource_not_found" {
		t.Fatalf("expected code resource_not_found, got %q", parsed.Code)
	}
	if parsed.Message != "task not found" {
		t.Fatalf("expected message task not found, got %q", parsed.Message)
	}
	if parsed.CorrelationID != "corr-header" {
		t.Fatalf("expected header correlation id corr-header, got %q", parsed.CorrelationID)
	}
	if len(parsed.DetailsPayload) == 0 {
		t.Fatalf("expected details payload to be present")
	}
}

func TestParseTransportHTTPErrorFallsBackToDetailWhenTypedErrorObjectMissing(t *testing.T) {
	body := []byte(`{"detail":"unauthorized"}`)
	parsed := parseTransportHTTPError(401, body, "")

	if parsed.Code != "auth_unauthorized" {
		t.Fatalf("expected code auth_unauthorized, got %q", parsed.Code)
	}
	if parsed.Message != "unauthorized" {
		t.Fatalf("expected message unauthorized, got %q", parsed.Message)
	}
	if parsed.Error() == "" {
		t.Fatalf("expected non-empty error string")
	}
}

func TestDefaultTransportErrorCodeClassifiesServiceNotConfigured(t *testing.T) {
	if got := defaultTransportErrorCode(501, "provider service is not configured"); got != "service_not_configured" {
		t.Fatalf("expected service_not_configured code, got %q", got)
	}
	if got := defaultTransportErrorCode(501, "daemon endpoint group is registered but not implemented yet"); got != "not_implemented" {
		t.Fatalf("expected not_implemented code for generic message, got %q", got)
	}
}

func TestDefaultTransportErrorCodeClassifiesRequestEntityTooLarge(t *testing.T) {
	if got := defaultTransportErrorCode(413, "request body exceeds limit"); got != "request_payload_too_large" {
		t.Fatalf("expected request_payload_too_large code, got %q", got)
	}
}

func TestDefaultTransportErrorCodeClassifiesRateLimitExceeded(t *testing.T) {
	if got := defaultTransportErrorCode(429, "control endpoint rate limit exceeded"); got != "rate_limit_exceeded" {
		t.Fatalf("expected rate_limit_exceeded code, got %q", got)
	}
}

func TestDefaultTransportErrorDetailsIncludesServiceDomainAndRemediation(t *testing.T) {
	detailsRaw := defaultTransportErrorDetails(501, "provider service is not configured")
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected details map, got %T", detailsRaw)
	}
	if got := details["category"]; got != "service_not_configured" {
		t.Fatalf("expected category service_not_configured, got %v", got)
	}
	if got := details["domain"]; got != "providers" {
		t.Fatalf("expected domain providers, got %v", got)
	}

	serviceRaw, ok := details["service"]
	if !ok {
		t.Fatalf("expected service details object")
	}
	service, ok := serviceRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected service details map, got %T", serviceRaw)
	}
	if got := service["id"]; got != "provider" {
		t.Fatalf("expected service.id provider, got %v", got)
	}
	if got := service["config_field"]; got != "Providers" {
		t.Fatalf("expected service.config_field Providers, got %v", got)
	}

	remediationRaw, ok := details["remediation"]
	if !ok {
		t.Fatalf("expected remediation details object")
	}
	remediation, ok := remediationRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected remediation details map, got %T", remediationRaw)
	}
	if got := remediation["action"]; got != "configure_server_service" {
		t.Fatalf("expected remediation action configure_server_service, got %v", got)
	}
	if got := remediation["hint"]; got == nil || got == "" {
		t.Fatalf("expected remediation hint, got %v", got)
	}
}
