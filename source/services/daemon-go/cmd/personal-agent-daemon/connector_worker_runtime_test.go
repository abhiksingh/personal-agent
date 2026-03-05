package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	shared "personalagent/runtime/internal/shared/contracts"

	_ "modernc.org/sqlite"
)

func TestLoadDaemonPluginWorkersEmbeddedManifestIncludesMessagesAndTwilio(t *testing.T) {
	workers, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "", "")
	if err != nil {
		t.Fatalf("load daemon plugin workers: %v", err)
	}
	if len(workers) != 8 {
		t.Fatalf("expected 8 embedded plugin workers, got %d", len(workers))
	}

	byID := map[string]bool{}
	for _, worker := range workers {
		byID[worker.PluginID] = true
	}
	if !byID["messages.daemon"] {
		t.Fatalf("expected messages.daemon connector worker to be registered")
	}
	if !byID["twilio.daemon"] {
		t.Fatalf("expected twilio.daemon connector worker to be registered")
	}
	if !byID["app_chat.daemon"] {
		t.Fatalf("expected app_chat.daemon channel worker to be registered")
	}
}

func TestLoadDaemonPluginWorkersIncludeDBPathForConnectorsOnly(t *testing.T) {
	workers, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "/tmp/runtime.db", "")
	if err != nil {
		t.Fatalf("load daemon plugin workers: %v", err)
	}
	if len(workers) == 0 {
		t.Fatalf("expected connector workers")
	}
	for _, worker := range workers {
		hasDBArg := false
		for idx := 0; idx < len(worker.Args)-1; idx++ {
			if worker.Args[idx] == "--db" && worker.Args[idx+1] == "/tmp/runtime.db" {
				hasDBArg = true
				break
			}
		}
		if worker.Kind == shared.AdapterKindConnector && !hasDBArg {
			t.Fatalf("expected connector worker %s args to include --db /tmp/runtime.db, got %v", worker.PluginID, worker.Args)
		}
		if worker.Kind == shared.AdapterKindChannel && hasDBArg {
			t.Fatalf("expected channel worker %s args to omit --db, got %v", worker.PluginID, worker.Args)
		}
	}
}

func TestExecuteTwilioConnectorWorkerOperationUnsupported(t *testing.T) {
	_, err := executeTwilioConnectorWorkerOperation(context.Background(), "unsupported", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}
}

func TestExecuteTwilioConnectorWorkerOperationDecodeFailure(t *testing.T) {
	_, err := executeTwilioConnectorWorkerOperation(context.Background(), "twilio_check", json.RawMessage(`{`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "decode payload") {
		t.Fatalf("expected payload decode error, got %v", err)
	}
}

func TestExecuteTwilioConnectorWorkerOperationUsesConfiguredHTTPTimeout(t *testing.T) {
	originalClient := twilioWorkerHTTPClient
	twilioWorkerHTTPClient = &http.Client{Timeout: 40 * time.Millisecond}
	t.Cleanup(func() {
		twilioWorkerHTTPClient = originalClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload, err := json.Marshal(channelcheck.TwilioRequest{
		Endpoint:   server.URL,
		AccountSID: "AC123",
		AuthToken:  "auth-token",
	})
	if err != nil {
		t.Fatalf("marshal twilio check payload: %v", err)
	}

	start := time.Now()
	_, err = executeTwilioConnectorWorkerOperation(context.Background(), "twilio_check", payload)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error for delayed twilio check")
	}
	errorText := strings.ToLower(err.Error())
	if !strings.Contains(errorText, "timeout") && !strings.Contains(errorText, "deadline") {
		t.Fatalf("expected timeout/deadline error, got %v", err)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("expected configured timeout to fail before delayed response, elapsed=%s", elapsed)
	}
}

func TestDecodeChannelConnectorPayloadRequiresPayload(t *testing.T) {
	var request channelcheck.TwilioRequest
	err := decodeChannelConnectorPayload(nil, &request)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "payload is required") {
		t.Fatalf("expected missing payload error, got %v", err)
	}
}

func TestExecuteMessagesConnectorWorkerOperationUnsupported(t *testing.T) {
	_, err := executeMessagesConnectorWorkerOperation(context.Background(), "unsupported", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}
}

func TestExecuteMessagesConnectorWorkerOperationSendUsesDryRun(t *testing.T) {
	t.Setenv("PA_MESSAGES_SEND_DRY_RUN", "1")
	payload, err := json.Marshal(messagesadapter.SendRequest{
		WorkspaceID: "ws1",
		Destination: "+15555550999",
		Message:     "hello messages",
	})
	if err != nil {
		t.Fatalf("marshal messages payload: %v", err)
	}

	result, err := executeMessagesConnectorWorkerOperation(context.Background(), "messages_send", payload)
	if err != nil {
		t.Fatalf("execute messages_send: %v", err)
	}
	response, ok := result.(messagesadapter.SendResponse)
	if !ok {
		t.Fatalf("expected messages send response type, got %T", result)
	}
	if response.Status != "sent" {
		t.Fatalf("expected sent status, got %s", response.Status)
	}
	if response.Transport != "messages_dry_run" {
		t.Fatalf("expected messages dry-run transport, got %s", response.Transport)
	}
}

func TestExecuteMessagesConnectorWorkerOperationPollInboundReturnsRows(t *testing.T) {
	chatDBPath := filepath.Join(t.TempDir(), "chat.db")
	createMessagesConnectorFixtureDB(t, chatDBPath)

	payload, err := json.Marshal(messagesadapter.InboundPollRequest{
		WorkspaceID:  "ws1",
		SourceDBPath: chatDBPath,
		SinceCursor:  "0",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("marshal messages poll payload: %v", err)
	}

	result, err := executeMessagesConnectorWorkerOperation(context.Background(), "messages_poll_inbound", payload)
	if err != nil {
		t.Fatalf("execute messages_poll_inbound: %v", err)
	}
	response, ok := result.(messagesadapter.InboundPollResponse)
	if !ok {
		t.Fatalf("expected messages poll response type, got %T", result)
	}
	if response.Polled != 1 {
		t.Fatalf("expected one inbound event, got %d", response.Polled)
	}
	if len(response.Events) != 1 {
		t.Fatalf("expected one event in payload, got %d", len(response.Events))
	}
	if response.Events[0].SourceEventID != "imessage-guid-1" {
		t.Fatalf("unexpected source event id: %s", response.Events[0].SourceEventID)
	}
}

func TestWriteWorkerErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeWorkerError(recorder, http.StatusBadRequest, fmt.Errorf("invalid connector payload"))
	response := recorder.Result()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode worker error payload: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(payload["error"])); got == "" || !strings.Contains(got, "invalid connector payload") {
		t.Fatalf("expected encoded error message, got %v", payload)
	}
}

func createMessagesConnectorFixtureDB(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	statements := []string{
		`CREATE TABLE message (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT,
			text TEXT,
			date INTEGER,
			is_from_me INTEGER,
			handle_id INTEGER,
			service TEXT
		);`,
		`CREATE TABLE chat (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT
		);`,
		`CREATE TABLE chat_message_join (
			chat_id INTEGER,
			message_id INTEGER
		);`,
		`CREATE TABLE handle (
			ROWID INTEGER PRIMARY KEY,
			id TEXT
		);`,
		`INSERT INTO handle(ROWID, id) VALUES (1, '+15555550100');`,
		`INSERT INTO chat(ROWID, guid) VALUES (1, 'chat-guid-1');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (1, 'imessage-guid-1', 'hello inbound', 1000000000, 0, 1, 'iMessage');`,
		`INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 1);`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec fixture statement: %v", err)
		}
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod fixture db: %v", err)
	}
}
