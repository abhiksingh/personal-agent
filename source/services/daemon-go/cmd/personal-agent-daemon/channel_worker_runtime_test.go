package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	localbridge "personalagent/runtime/internal/channels/adapters/localbridge"
	shared "personalagent/runtime/internal/shared/contracts"
)

func TestLoadDaemonPluginWorkersEmbeddedManifestIncludesAppChatChannelWorker(t *testing.T) {
	workers, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "", "")
	if err != nil {
		t.Fatalf("load daemon plugin workers: %v", err)
	}
	channelWorkers := 0
	for _, worker := range workers {
		if worker.Kind == shared.AdapterKindChannel {
			channelWorkers++
		}
	}
	if channelWorkers != 1 {
		t.Fatalf("expected one channel worker in embedded manifest, got %d", channelWorkers)
	}

	found := false
	for _, worker := range workers {
		if worker.Kind == shared.AdapterKindChannel && worker.PluginID == "app_chat.daemon" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected app_chat.daemon channel worker to be registered")
	}
}

func TestChannelWorkerCapabilitiesForNonTwilioWorkers(t *testing.T) {
	appCapabilities := channelWorkerCapabilities(channelWorkerTypeAppChat)
	if len(appCapabilities) == 0 {
		t.Fatalf("expected app chat capabilities")
	}

	if capabilities := channelWorkerCapabilities("messages"); len(capabilities) != 0 {
		t.Fatalf("expected no messages capabilities in channel worker runtime, got %+v", capabilities)
	}
}

func TestChannelWorkerRuntimeAppChatSendUsesLocalTransportArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("PA_CHANNEL_DATA_DIR", dataDir)

	runtime := &channelWorkerRuntime{}
	payload, err := json.Marshal(localbridge.AppChatSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "manual-thread",
		Message:     "hello app chat",
	})
	if err != nil {
		t.Fatalf("marshal app chat payload: %v", err)
	}

	result, err := runtime.executeChannelWorkerOperation(context.Background(), "app_chat_send", payload)
	if err != nil {
		t.Fatalf("execute app_chat_send: %v", err)
	}
	response, ok := result.(localbridge.AppChatSendResponse)
	if !ok {
		t.Fatalf("expected app chat send response type, got %T", result)
	}
	if response.Status != "sent" {
		t.Fatalf("expected sent status, got %s", response.Status)
	}
	if response.Transport == "" {
		t.Fatalf("expected transport marker")
	}
	if response.RecordPath == "" {
		t.Fatalf("expected record path")
	}
	if _, err := os.Stat(response.RecordPath); err != nil {
		t.Fatalf("expected app chat record at %s: %v", response.RecordPath, err)
	}
}

func TestChannelWorkerRuntimeUnsupportedOperation(t *testing.T) {
	runtime := &channelWorkerRuntime{}
	_, err := runtime.executeChannelWorkerOperation(context.Background(), "unsupported", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}
}

func TestDecodeChannelWorkerPayloadRequiresPayload(t *testing.T) {
	var request localbridge.AppChatSendRequest
	err := decodeChannelWorkerPayload(nil, &request)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "payload is required") {
		t.Fatalf("expected payload required error, got %v", err)
	}
}

func TestWriteChannelWorkerErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeChannelWorkerError(recorder, http.StatusBadRequest, fmt.Errorf("invalid channel payload"))
	response := recorder.Result()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload channelWorkerExecuteResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode channel worker error payload: %v", err)
	}
	if payload.Error == "" || !strings.Contains(payload.Error, "invalid channel payload") {
		t.Fatalf("expected encoded error message, got %+v", payload)
	}
}
