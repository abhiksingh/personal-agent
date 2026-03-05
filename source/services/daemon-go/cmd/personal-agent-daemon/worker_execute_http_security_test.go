package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewWorkerExecuteHTTPServerUsesDoSHardeningTimeouts(t *testing.T) {
	server := newWorkerExecuteHTTPServer(http.NewServeMux())

	if server.ReadHeaderTimeout != workerExecuteReadHeaderTimeout {
		t.Fatalf("expected read header timeout %s, got %s", workerExecuteReadHeaderTimeout, server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != workerExecuteReadTimeout {
		t.Fatalf("expected read timeout %s, got %s", workerExecuteReadTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != workerExecuteWriteTimeout {
		t.Fatalf("expected write timeout %s, got %s", workerExecuteWriteTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != workerExecuteIdleTimeout {
		t.Fatalf("expected idle timeout %s, got %s", workerExecuteIdleTimeout, server.IdleTimeout)
	}
}

func TestDecodeWorkerExecuteJSONPayloadRejectsMalformedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://worker/execute", strings.NewReader(`{"operation":"health","payload":`))
	recorder := httptest.NewRecorder()

	var payload channelWorkerExecuteRequest
	statusCode, err := decodeWorkerExecuteJSONPayload(recorder, request, &payload, "execute")
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", statusCode)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid execute payload") {
		t.Fatalf("expected malformed payload rejection, got %v", err)
	}
}

func TestDecodeWorkerExecuteJSONPayloadRejectsTrailingContent(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"http://worker/execute",
		strings.NewReader(`{"operation":"health","payload":{}}{"operation":"extra"}`),
	)
	recorder := httptest.NewRecorder()

	var payload channelWorkerExecuteRequest
	statusCode, err := decodeWorkerExecuteJSONPayload(recorder, request, &payload, "execute")
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", statusCode)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "trailing content") {
		t.Fatalf("expected trailing content rejection, got %v", err)
	}
}

func TestDecodeWorkerExecuteJSONPayloadRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"http://worker/execute",
		strings.NewReader(`{"operation":"health","payload":{},"unexpected":true}`),
	)
	recorder := httptest.NewRecorder()

	var payload channelWorkerExecuteRequest
	statusCode, err := decodeWorkerExecuteJSONPayload(recorder, request, &payload, "execute")
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", statusCode)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestDecodeWorkerExecuteJSONPayloadRejectsOversizedBody(t *testing.T) {
	oversizedOperation := strings.Repeat("o", int(workerExecuteMaxBodyBytes)+256)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://worker/execute",
		strings.NewReader(`{"operation":"`+oversizedOperation+`","payload":{}}`),
	)
	recorder := httptest.NewRecorder()

	var payload channelWorkerExecuteRequest
	statusCode, err := decodeWorkerExecuteJSONPayload(recorder, request, &payload, "execute")
	if statusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", statusCode)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exceeds") {
		t.Fatalf("expected oversized payload rejection, got %v", err)
	}
}

func TestDecodeWorkerExecuteJSONPayloadAcceptsValidBody(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"http://worker/execute",
		strings.NewReader(`{"operation":"app_chat_status","payload":{}}`),
	)
	recorder := httptest.NewRecorder()

	var payload channelWorkerExecuteRequest
	statusCode, err := decodeWorkerExecuteJSONPayload(recorder, request, &payload, "execute")
	if err != nil {
		t.Fatalf("expected valid payload decode, got %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if payload.Operation != "app_chat_status" {
		t.Fatalf("expected operation to decode, got %+v", payload)
	}
}
