package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"personalagent/runtime/internal/daemonruntime"
)

func TestLoadWorkerExecAuthTokenFromEnv(t *testing.T) {
	t.Setenv(daemonruntime.WorkerExecAuthTokenEnvVar, "")
	if _, err := loadWorkerExecAuthTokenFromEnv(); err == nil {
		t.Fatalf("expected error when worker auth token env is missing")
	}

	t.Setenv(daemonruntime.WorkerExecAuthTokenEnvVar, "worker-token")
	token, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		t.Fatalf("load worker auth token: %v", err)
	}
	if token != "worker-token" {
		t.Fatalf("expected worker-token, got %q", token)
	}
}

func TestAuthorizeWorkerExecuteRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://worker/execute", nil)
	request.Header.Set("Authorization", "Bearer worker-token")
	if !authorizeWorkerExecuteRequest(request, "worker-token") {
		t.Fatalf("expected authorization success")
	}

	request.Header.Set("Authorization", "Bearer wrong-token")
	if authorizeWorkerExecuteRequest(request, "worker-token") {
		t.Fatalf("expected authorization failure for wrong token")
	}

	request.Header.Del("Authorization")
	if authorizeWorkerExecuteRequest(request, "worker-token") {
		t.Fatalf("expected authorization failure for missing token")
	}
}

func TestWriteWorkerUnauthorized(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeWorkerUnauthorized(recorder)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 status, got %d", recorder.Code)
	}
	if recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("expected WWW-Authenticate Bearer header")
	}

	payload := map[string]any{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload["error"] != "unauthorized" {
		t.Fatalf("expected unauthorized error payload, got %+v", payload)
	}
}
