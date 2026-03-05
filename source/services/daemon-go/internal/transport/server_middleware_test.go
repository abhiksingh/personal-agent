package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireAuthorizedMethodRejectsWrongMethod(t *testing.T) {
	server := &Server{config: ServerConfig{AuthToken: "token"}}

	request := httptest.NewRequest(http.MethodGet, "http://localhost/v1/tasks", nil)
	recorder := httptest.NewRecorder()

	_, ok := server.requireAuthorizedMethod(recorder, request, http.MethodPost)
	if ok {
		t.Fatalf("expected request to be rejected")
	}
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, got)
	}
}

func TestRequireAuthorizedMethodRejectsUnauthorizedRequest(t *testing.T) {
	server := &Server{config: ServerConfig{AuthToken: "token"}}

	request := httptest.NewRequest(http.MethodPost, "http://localhost/v1/tasks", nil)
	recorder := httptest.NewRecorder()

	_, ok := server.requireAuthorizedMethod(recorder, request, http.MethodPost)
	if ok {
		t.Fatalf("expected request to be unauthorized")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}

func TestRequireAuthorizedMethodAcceptsAuthorizedRequest(t *testing.T) {
	server := &Server{config: ServerConfig{AuthToken: "token"}}

	request := httptest.NewRequest(http.MethodPost, "http://localhost/v1/tasks", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()

	correlationID, ok := server.requireAuthorizedMethod(recorder, request, http.MethodPost)
	if !ok {
		t.Fatalf("expected request to be authorized")
	}
	if strings.TrimSpace(correlationID) == "" {
		t.Fatalf("expected correlation id to be assigned")
	}
}

func TestAuthorizeBearerToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://localhost/v1/tasks", nil)
	request.Header.Set("Authorization", "Bearer token")
	if !authorizeBearerToken(request, "token") {
		t.Fatalf("expected authorization success for exact token match")
	}

	request.Header.Set("Authorization", "Bearer token-plus")
	if authorizeBearerToken(request, "token") {
		t.Fatalf("expected authorization failure for mismatched token")
	}

	request.Header.Set("Authorization", "Bearer tok")
	if authorizeBearerToken(request, "token") {
		t.Fatalf("expected authorization failure for wrong length token")
	}

	request.Header.Del("Authorization")
	if authorizeBearerToken(request, "token") {
		t.Fatalf("expected authorization failure for missing token")
	}

	if authorizeBearerToken(nil, "token") {
		t.Fatalf("expected authorization failure for nil request")
	}
}

func TestDecodeJSONBodyStrictRejectsUnknownFieldsAndTrailingContent(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	unknownFieldReq := httptest.NewRequest(http.MethodPost, "http://localhost/test", strings.NewReader(`{"name":"ok","extra":"nope"}`))
	var unknownFieldPayload payload
	if err := decodeJSONBodyStrict(unknownFieldReq.Body, &unknownFieldPayload); err == nil {
		t.Fatalf("expected unknown field decode error")
	}

	trailingReq := httptest.NewRequest(http.MethodPost, "http://localhost/test", strings.NewReader(`{"name":"ok"}{"second":"payload"}`))
	var trailingPayload payload
	if err := decodeJSONBodyStrict(trailingReq.Body, &trailingPayload); err == nil {
		t.Fatalf("expected trailing content decode error")
	}
}

func TestDecodeRequestBodyWritesCustomInvalidPayloadError(t *testing.T) {
	server := &Server{config: ServerConfig{AuthToken: "token"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://localhost/test", strings.NewReader(`{"invalid":`))

	var payload map[string]any
	ok := server.decodeRequestBody(recorder, request, "corr-test", "bad payload", &payload)
	if ok {
		t.Fatalf("expected decode failure")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var envelope transportErrorEnvelope
	decoder := json.NewDecoder(bytes.NewReader(recorder.Body.Bytes()))
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := strings.TrimSpace(envelope.Error.Message); got != "bad payload" {
		t.Fatalf("expected error.message %q, got %q", "bad payload", got)
	}
}

func TestDecodeRequestBodyRejectsOversizedPayloadWithTyped413Error(t *testing.T) {
	server := &Server{config: ServerConfig{AuthToken: "token", RequestBodyBytesLimit: 24}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://localhost/test", strings.NewReader(`{"name":"abcdefghijklmnopqrstuvwxyz"}`))

	var payload map[string]any
	ok := server.decodeRequestBody(recorder, request, "corr-too-large", "bad payload", &payload)
	if ok {
		t.Fatalf("expected oversized payload decode failure")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413 for oversized payload, got %d", recorder.Code)
	}

	var envelope transportErrorEnvelope
	if err := json.NewDecoder(bytes.NewReader(recorder.Body.Bytes())).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := strings.TrimSpace(envelope.Error.Code); got != "request_payload_too_large" {
		t.Fatalf("expected error.code request_payload_too_large, got %q", got)
	}
	if !strings.Contains(strings.TrimSpace(envelope.Error.Message), "24") {
		t.Fatalf("expected oversized payload message to include byte limit, got %q", envelope.Error.Message)
	}
}
