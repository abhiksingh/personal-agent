package cliapp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"

	_ "modernc.org/sqlite"
)

func TestRunTaskLifecycleCommandsUseExplicitWorkspaceGuardWhenProvided(t *testing.T) {
	workspaceGuard := "ws-task-default"
	server := startCLITestServer(t)

	submitTask := func(title string) string {
		submitOut := &bytes.Buffer{}
		submitErr := &bytes.Buffer{}
		submitCode := run(withDaemonArgs(server,
			"task", "submit",
			"--workspace", "ws1",
			"--requested-by", "actor.cli",
			"--subject", "actor.cli",
			"--title", title,
			"--description", "workspace-guard-regression",
			"--task-class", "automation",
		), submitOut, submitErr)
		if submitCode != 0 {
			t.Fatalf("task submit failed: code=%d stderr=%s output=%s", submitCode, submitErr.String(), submitOut.String())
		}
		var submitResponse transport.SubmitTaskResponse
		if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
			t.Fatalf("decode task submit response: %v", err)
		}
		if strings.TrimSpace(submitResponse.TaskID) == "" {
			t.Fatalf("expected task_id in submit response")
		}
		return submitResponse.TaskID
	}

	cancelTaskID := submitTask("cancel-mismatch")
	cancelOut := &bytes.Buffer{}
	cancelErr := &bytes.Buffer{}
	cancelCode := run(withDaemonArgs(server,
		"task", "cancel",
		"--workspace", workspaceGuard,
		"--task-id", cancelTaskID,
	), cancelOut, cancelErr)
	if cancelCode == 0 {
		t.Fatalf("expected task cancel workspace guard failure, output=%s", cancelOut.String())
	}
	if !strings.Contains(cancelErr.String(), "workspace mismatch") {
		t.Fatalf("expected workspace mismatch for task cancel, got stderr=%s", cancelErr.String())
	}

	retryTaskID := submitTask("retry-mismatch")
	cancelRetrySeedOut := &bytes.Buffer{}
	cancelRetrySeedErr := &bytes.Buffer{}
	cancelRetrySeedCode := run(withDaemonArgs(server,
		"task", "cancel",
		"--workspace", "ws1",
		"--task-id", retryTaskID,
	), cancelRetrySeedOut, cancelRetrySeedErr)
	if cancelRetrySeedCode != 0 {
		t.Fatalf("seed cancel for retry failed: code=%d stderr=%s output=%s", cancelRetrySeedCode, cancelRetrySeedErr.String(), cancelRetrySeedOut.String())
	}

	retryOut := &bytes.Buffer{}
	retryErr := &bytes.Buffer{}
	retryCode := run(withDaemonArgs(server,
		"task", "retry",
		"--workspace", workspaceGuard,
		"--task-id", retryTaskID,
	), retryOut, retryErr)
	if retryCode == 0 {
		t.Fatalf("expected task retry workspace guard failure, output=%s", retryOut.String())
	}
	if !strings.Contains(retryErr.String(), "workspace mismatch") {
		t.Fatalf("expected workspace mismatch for task retry, got stderr=%s", retryErr.String())
	}

	requeueTaskID := submitTask("requeue-mismatch")
	requeueOut := &bytes.Buffer{}
	requeueErr := &bytes.Buffer{}
	requeueCode := run(withDaemonArgs(server,
		"task", "requeue",
		"--workspace", workspaceGuard,
		"--task-id", requeueTaskID,
	), requeueOut, requeueErr)
	if requeueCode == 0 {
		t.Fatalf("expected task requeue workspace guard failure, output=%s", requeueOut.String())
	}
	if !strings.Contains(requeueErr.String(), "workspace mismatch") {
		t.Fatalf("expected workspace mismatch for task requeue, got stderr=%s", requeueErr.String())
	}
}

func TestRunConnectorTwilioSetGetAndCheckCommands(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123.json" {
			t.Fatalf("expected account probe path, got %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Fatalf("expected basic auth")
		}
		if user != "AC123" || pass != "twilio-secret-token" {
			t.Fatalf("unexpected basic auth credentials")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-channel.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-secret-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), setOut, setErr)
	if setCode != 0 {
		t.Fatalf("connector twilio set failed: code=%d stderr=%s output=%s", setCode, setErr.String(), setOut.String())
	}

	var setResponse map[string]any
	if err := json.Unmarshal(setOut.Bytes(), &setResponse); err != nil {
		t.Fatalf("decode connector twilio set response: %v", err)
	}
	if got := setResponse["account_sid_secret_name"]; got != "TWILIO_ACCOUNT_SID" {
		t.Fatalf("expected default account sid secret name, got %v", got)
	}
	if got := setResponse["auth_token_secret_name"]; got != "TWILIO_AUTH_TOKEN" {
		t.Fatalf("expected default auth token secret name, got %v", got)
	}
	if got := setResponse["sms_number"]; got != "+15555550001" {
		t.Fatalf("expected sms number, got %v", got)
	}
	if got := setResponse["voice_number"]; got != "+15555550002" {
		t.Fatalf("expected voice number, got %v", got)
	}
	if configured, ok := setResponse["credentials_configured"].(bool); !ok || !configured {
		t.Fatalf("expected credentials configured=true, got %v", setResponse["credentials_configured"])
	}

	_, accountSID, err := manager.Get("ws1", "TWILIO_ACCOUNT_SID")
	if err != nil {
		t.Fatalf("expected stored twilio account sid secret: %v", err)
	}
	if accountSID != "AC123" {
		t.Fatalf("expected stored account sid AC123, got %q", accountSID)
	}
	_, authToken, err := manager.Get("ws1", "TWILIO_AUTH_TOKEN")
	if err != nil {
		t.Fatalf("expected stored twilio auth token secret: %v", err)
	}
	if authToken != "twilio-secret-token" {
		t.Fatalf("expected stored auth token value, got %q", authToken)
	}

	getOut := &bytes.Buffer{}
	getErr := &bytes.Buffer{}
	getCode := run(withDaemonArgs(server,
		"connector", "twilio", "get",
		"--workspace", "ws1",
	), getOut, getErr)
	if getCode != 0 {
		t.Fatalf("connector twilio get failed: code=%d stderr=%s output=%s", getCode, getErr.String(), getOut.String())
	}

	var getResponse map[string]any
	if err := json.Unmarshal(getOut.Bytes(), &getResponse); err != nil {
		t.Fatalf("decode connector twilio get response: %v", err)
	}
	if got := getResponse["endpoint"]; got != twilioServer.URL {
		t.Fatalf("expected endpoint %s, got %v", twilioServer.URL, got)
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"connector", "twilio", "check",
		"--workspace", "ws1",
	), checkOut, checkErr)
	if checkCode != 0 {
		t.Fatalf("connector twilio check failed: code=%d stderr=%s output=%s", checkCode, checkErr.String(), checkOut.String())
	}

	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode connector twilio check response: %v", err)
	}
	if success, ok := checkResponse["success"].(bool); !ok || !success {
		t.Fatalf("expected twilio check success=true, got %v", checkResponse["success"])
	}
	result, ok := checkResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected twilio check result payload")
	}
	if got := result["status_code"]; got != float64(http.StatusOK) {
		t.Fatalf("expected status code %d, got %v", http.StatusOK, got)
	}
}

func TestRunChannelMappingCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "channel-mapping.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"channel", "mapping", "list",
		"--workspace", "ws1",
		"--channel", "message",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("channel mapping list failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}

	var listResponse transport.ChannelConnectorMappingListResponse
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode channel mapping list response: %v", err)
	}
	if listResponse.WorkspaceID != "ws1" || listResponse.ChannelID != "message" {
		t.Fatalf("unexpected channel mapping list identifiers: %+v", listResponse)
	}
	if len(listResponse.Bindings) < 2 {
		t.Fatalf("expected seeded message channel mappings, got %+v", listResponse.Bindings)
	}

	prioritizeOut := &bytes.Buffer{}
	prioritizeErr := &bytes.Buffer{}
	prioritizeCode := run(withDaemonArgs(server,
		"channel", "mapping", "prioritize",
		"--workspace", "ws1",
		"--channel", "message",
		"--connector", "twilio",
		"--priority", "1",
	), prioritizeOut, prioritizeErr)
	if prioritizeCode != 0 {
		t.Fatalf("channel mapping prioritize failed: code=%d stderr=%s output=%s", prioritizeCode, prioritizeErr.String(), prioritizeOut.String())
	}
	var prioritizeResponse transport.ChannelConnectorMappingUpsertResponse
	if err := json.Unmarshal(prioritizeOut.Bytes(), &prioritizeResponse); err != nil {
		t.Fatalf("decode channel mapping prioritize response: %v", err)
	}
	if prioritizeResponse.ChannelID != "message" || prioritizeResponse.ConnectorID != "twilio" || prioritizeResponse.Priority != 1 {
		t.Fatalf("unexpected channel mapping prioritize payload: %+v", prioritizeResponse)
	}

	disableOut := &bytes.Buffer{}
	disableErr := &bytes.Buffer{}
	disableCode := run(withDaemonArgs(server,
		"channel", "mapping", "disable",
		"--workspace", "ws1",
		"--channel", "message",
		"--connector", "twilio",
	), disableOut, disableErr)
	if disableCode != 0 {
		t.Fatalf("channel mapping disable failed: code=%d stderr=%s output=%s", disableCode, disableErr.String(), disableOut.String())
	}
	var disableResponse transport.ChannelConnectorMappingUpsertResponse
	if err := json.Unmarshal(disableOut.Bytes(), &disableResponse); err != nil {
		t.Fatalf("decode channel mapping disable response: %v", err)
	}
	if disableResponse.Enabled {
		t.Fatalf("expected twilio mapping to be disabled, got %+v", disableResponse)
	}

	enableOut := &bytes.Buffer{}
	enableErr := &bytes.Buffer{}
	enableCode := run(withDaemonArgs(server,
		"channel", "mapping", "enable",
		"--workspace", "ws1",
		"--channel", "message",
		"--connector", "twilio",
		"--priority", "1",
	), enableOut, enableErr)
	if enableCode != 0 {
		t.Fatalf("channel mapping enable failed: code=%d stderr=%s output=%s", enableCode, enableErr.String(), enableOut.String())
	}
	var enableResponse transport.ChannelConnectorMappingUpsertResponse
	if err := json.Unmarshal(enableOut.Bytes(), &enableResponse); err != nil {
		t.Fatalf("decode channel mapping enable response: %v", err)
	}
	if !enableResponse.Enabled || enableResponse.Priority != 1 {
		t.Fatalf("expected twilio mapping enabled with priority=1, got %+v", enableResponse)
	}

	finalListOut := &bytes.Buffer{}
	finalListErr := &bytes.Buffer{}
	finalListCode := run(withDaemonArgs(server,
		"channel", "mapping", "list",
		"--workspace", "ws1",
		"--channel", "message",
	), finalListOut, finalListErr)
	if finalListCode != 0 {
		t.Fatalf("channel mapping final list failed: code=%d stderr=%s output=%s", finalListCode, finalListErr.String(), finalListOut.String())
	}
	var finalList transport.ChannelConnectorMappingListResponse
	if err := json.Unmarshal(finalListOut.Bytes(), &finalList); err != nil {
		t.Fatalf("decode channel mapping final list: %v", err)
	}
	twilioBinding, found := findChannelMappingBinding(finalList.Bindings, "twilio")
	if !found {
		t.Fatalf("expected twilio mapping in final list: %+v", finalList.Bindings)
	}
	if !twilioBinding.Enabled || twilioBinding.Priority != 1 {
		t.Fatalf("expected twilio binding enabled with priority=1, got %+v", twilioBinding)
	}
}

func findChannelMappingBinding(bindings []transport.ChannelConnectorMappingRecord, connectorID string) (transport.ChannelConnectorMappingRecord, bool) {
	target := strings.ToLower(strings.TrimSpace(connectorID))
	for _, binding := range bindings {
		if strings.ToLower(strings.TrimSpace(binding.ConnectorID)) == target {
			return binding, true
		}
	}
	return transport.ChannelConnectorMappingRecord{}, false
}

func TestRunChannelTwilioCheckFailsOnUnauthorized(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	if _, err := manager.Put("ws1", "TWILIO_ACCOUNT_SID", "AC123"); err != nil {
		t.Fatalf("seed account sid secret: %v", err)
	}
	if _, err := manager.Put("ws1", "TWILIO_AUTH_TOKEN", "bad-token"); err != nil {
		t.Fatalf("seed auth token secret: %v", err)
	}

	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-channel.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid-secret", "TWILIO_ACCOUNT_SID",
		"--auth-token-secret", "TWILIO_AUTH_TOKEN",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio set with existing secrets to succeed")
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"connector", "twilio", "check",
		"--workspace", "ws1",
	), checkOut, checkErr)
	if checkCode == 0 {
		t.Fatalf("expected unauthorized twilio check to fail")
	}

	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode twilio check response: %v", err)
	}
	if success, ok := checkResponse["success"].(bool); !ok || success {
		t.Fatalf("expected success=false for unauthorized check, got %v", checkResponse["success"])
	}
}

func TestRunChannelTwilioIngestSMSPersistsEventAndReplaySafe(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", "https://api.twilio.test",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	requestURL := "https://agent.local/webhook" + defaultTwilioWebhookSMSPath()
	params := map[string]string{
		"From":       "+15555550999",
		"To":         "+15555550001",
		"Body":       "hello from webhook",
		"MessageSid": "SMINBOUND1",
		"AccountSid": "AC123",
	}
	signature, err := twilioadapter.ComputeRequestSignature("twilio-auth-token", requestURL, params)
	if err != nil {
		t.Fatalf("compute signature: %v", err)
	}

	ingestOut := &bytes.Buffer{}
	ingestErr := &bytes.Buffer{}
	ingestCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-sms",
		"--workspace", "ws1",
		"--request-url", requestURL,
		"--signature", signature,
		"--from", "+15555550999",
		"--to", "+15555550001",
		"--body", "hello from webhook",
		"--message-sid", "SMINBOUND1",
		"--account-sid", "AC123",
	), ingestOut, ingestErr)
	if ingestCode != 0 {
		t.Fatalf("ingest-sms failed: code=%d stderr=%s output=%s", ingestCode, ingestErr.String(), ingestOut.String())
	}

	var ingestResponse map[string]any
	if err := json.Unmarshal(ingestOut.Bytes(), &ingestResponse); err != nil {
		t.Fatalf("decode ingest response: %v", err)
	}
	if accepted, ok := ingestResponse["accepted"].(bool); !ok || !accepted {
		t.Fatalf("expected accepted=true, got %v", ingestResponse["accepted"])
	}
	if replayed, ok := ingestResponse["replayed"].(bool); !ok || replayed {
		t.Fatalf("expected replayed=false, got %v", ingestResponse["replayed"])
	}
	firstEventID, _ := ingestResponse["event_id"].(string)
	if firstEventID == "" {
		t.Fatalf("expected event_id in ingest response")
	}

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	replayCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-sms",
		"--workspace", "ws1",
		"--request-url", requestURL,
		"--signature", signature,
		"--from", "+15555550999",
		"--to", "+15555550001",
		"--body", "hello from webhook",
		"--message-sid", "SMINBOUND1",
		"--account-sid", "AC123",
	), replayOut, replayErr)
	if replayCode != 0 {
		t.Fatalf("ingest-sms replay failed: code=%d stderr=%s output=%s", replayCode, replayErr.String(), replayOut.String())
	}

	var replayResponse map[string]any
	if err := json.Unmarshal(replayOut.Bytes(), &replayResponse); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if replayed, ok := replayResponse["replayed"].(bool); !ok || !replayed {
		t.Fatalf("expected replayed=true, got %v", replayResponse["replayed"])
	}
	if got := replayResponse["event_id"]; got != firstEventID {
		t.Fatalf("expected replay event_id %s, got %v", firstEventID, got)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_events`).Scan(&count); err != nil {
		t.Fatalf("count comm_events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one comm_event, got %d", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_webhook_receipts`).Scan(&count); err != nil {
		t.Fatalf("count comm_webhook_receipts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one comm_webhook_receipt, got %d", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_provider_messages`).Scan(&count); err != nil {
		t.Fatalf("count comm_provider_messages: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one comm_provider_message, got %d", count)
	}
}

func TestRunChannelTwilioIngestSMSRejectsInvalidSignatureAndAudits(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	ingestOut := &bytes.Buffer{}
	ingestErr := &bytes.Buffer{}
	ingestCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-sms",
		"--workspace", "ws1",
		"--request-url", "https://agent.local/webhook"+defaultTwilioWebhookSMSPath(),
		"--signature", "invalid",
		"--from", "+15555550999",
		"--to", "+15555550001",
		"--body", "hello from webhook",
		"--message-sid", "SMINVALID1",
		"--account-sid", "AC123",
	), ingestOut, ingestErr)
	if ingestCode == 0 {
		t.Fatalf("expected invalid signature ingest to fail")
	}

	var ingestResponse map[string]any
	if err := json.Unmarshal(ingestOut.Bytes(), &ingestResponse); err != nil {
		t.Fatalf("decode invalid signature ingest response: %v", err)
	}
	if accepted, ok := ingestResponse["accepted"].(bool); !ok || accepted {
		t.Fatalf("expected accepted=false, got %v", ingestResponse["accepted"])
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var commEventCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_events`).Scan(&commEventCount); err != nil {
		t.Fatalf("count comm_events: %v", err)
	}
	if commEventCount != 0 {
		t.Fatalf("expected no comm events on invalid signature, got %d", commEventCount)
	}

	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE event_type = 'twilio_webhook_rejected_invalid_signature'`).Scan(&auditCount); err != nil {
		t.Fatalf("count audit entries: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one rejection audit entry, got %d", auditCount)
	}
}

func TestRunChannelTwilioStartCallAndIngestVoiceLifecycle(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Calls.json" {
			t.Fatalf("unexpected Twilio path: %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "twilio-auth-token" {
			t.Fatalf("unexpected basic auth credentials")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("From"); got != "+15555550002" {
			t.Fatalf("expected From +15555550002, got %s", got)
		}
		if got := r.Form.Get("To"); got != "+15555550999" {
			t.Fatalf("expected To +15555550999, got %s", got)
		}
		if got := r.Form.Get("Url"); got != "https://agent.local/twiml/voice" {
			t.Fatalf("expected Url to match twiml url, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"CA123","account_sid":"AC123","status":"queued","direction":"outbound-api","from":"+15555550002","to":"+15555550999"}`))
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-voice.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	startOut := &bytes.Buffer{}
	startErr := &bytes.Buffer{}
	startCode := run(withDaemonArgs(server,
		"connector", "twilio", "start-call",
		"--workspace", "ws1",
		"--to", "+15555550999",
		"--twiml-url", "https://agent.local/twiml/voice",
	), startOut, startErr)
	if startCode != 0 {
		t.Fatalf("start-call failed: code=%d stderr=%s output=%s", startCode, startErr.String(), startOut.String())
	}

	var startResponse map[string]any
	if err := json.Unmarshal(startOut.Bytes(), &startResponse); err != nil {
		t.Fatalf("decode start-call response: %v", err)
	}
	if got := startResponse["call_sid"]; got != "CA123" {
		t.Fatalf("expected call_sid CA123, got %v", got)
	}
	if got := startResponse["status"]; got != "initiated" {
		t.Fatalf("expected status initiated, got %v", got)
	}

	requestURL := "https://agent.local/webhook" + defaultTwilioWebhookVoicePath()
	signature1, err := twilioadapter.ComputeRequestSignature("twilio-auth-token", requestURL, map[string]string{
		"CallSid":    "CA123",
		"CallStatus": "ringing",
		"From":       "+15555550002",
		"To":         "+15555550999",
		"Direction":  "outbound-api",
		"AccountSid": "AC123",
	})
	if err != nil {
		t.Fatalf("compute voice signature1: %v", err)
	}

	ingest1Out := &bytes.Buffer{}
	ingest1Err := &bytes.Buffer{}
	ingest1Code := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--provider-event-id", "voice-cb-1",
		"--request-url", requestURL,
		"--signature", signature1,
		"--call-sid", "CA123",
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "ringing",
	), ingest1Out, ingest1Err)
	if ingest1Code != 0 {
		t.Fatalf("ingest-voice ringing failed: code=%d stderr=%s output=%s", ingest1Code, ingest1Err.String(), ingest1Out.String())
	}

	var ingest1Response map[string]any
	if err := json.Unmarshal(ingest1Out.Bytes(), &ingest1Response); err != nil {
		t.Fatalf("decode ingest-voice ringing response: %v", err)
	}
	if got := ingest1Response["call_status"]; got != "ringing" {
		t.Fatalf("expected call_status ringing, got %v", got)
	}

	signature2, err := twilioadapter.ComputeRequestSignature("twilio-auth-token", requestURL, map[string]string{
		"CallSid":      "CA123",
		"CallStatus":   "in-progress",
		"From":         "+15555550002",
		"To":           "+15555550999",
		"Direction":    "outbound-api",
		"AccountSid":   "AC123",
		"SpeechResult": "Hello agent",
	})
	if err != nil {
		t.Fatalf("compute voice signature2: %v", err)
	}

	ingest2Out := &bytes.Buffer{}
	ingest2Err := &bytes.Buffer{}
	ingest2Code := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--provider-event-id", "voice-cb-2",
		"--request-url", requestURL,
		"--signature", signature2,
		"--call-sid", "CA123",
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "in-progress",
		"--transcript", "Hello agent",
		"--transcript-direction", "INBOUND",
	), ingest2Out, ingest2Err)
	if ingest2Code != 0 {
		t.Fatalf("ingest-voice in-progress failed: code=%d stderr=%s output=%s", ingest2Code, ingest2Err.String(), ingest2Out.String())
	}

	var ingest2Response map[string]any
	if err := json.Unmarshal(ingest2Out.Bytes(), &ingest2Response); err != nil {
		t.Fatalf("decode ingest-voice in-progress response: %v", err)
	}
	if got := ingest2Response["call_status"]; got != "in_progress" {
		t.Fatalf("expected call_status in_progress, got %v", got)
	}
	if transcriptEventID, _ := ingest2Response["transcript_event_id"].(string); transcriptEventID == "" {
		t.Fatalf("expected transcript_event_id in response")
	}

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	replayCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--provider-event-id", "voice-cb-2",
		"--request-url", requestURL,
		"--signature", signature2,
		"--call-sid", "CA123",
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "in-progress",
		"--transcript", "Hello agent",
	), replayOut, replayErr)
	if replayCode != 0 {
		t.Fatalf("ingest-voice replay failed: code=%d stderr=%s output=%s", replayCode, replayErr.String(), replayOut.String())
	}
	var replayResponse map[string]any
	if err := json.Unmarshal(replayOut.Bytes(), &replayResponse); err != nil {
		t.Fatalf("decode ingest-voice replay response: %v", err)
	}
	if replayed, ok := replayResponse["replayed"].(bool); !ok || !replayed {
		t.Fatalf("expected replayed=true, got %v", replayResponse["replayed"])
	}

	signature3, err := twilioadapter.ComputeRequestSignature("twilio-auth-token", requestURL, map[string]string{
		"CallSid":    "CA123",
		"CallStatus": "completed",
		"From":       "+15555550002",
		"To":         "+15555550999",
		"Direction":  "outbound-api",
		"AccountSid": "AC123",
	})
	if err != nil {
		t.Fatalf("compute voice signature3: %v", err)
	}

	completeOut := &bytes.Buffer{}
	completeErr := &bytes.Buffer{}
	completeCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--provider-event-id", "voice-cb-3",
		"--request-url", requestURL,
		"--signature", signature3,
		"--call-sid", "CA123",
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "completed",
	), completeOut, completeErr)
	if completeCode != 0 {
		t.Fatalf("ingest-voice completed failed: code=%d stderr=%s output=%s", completeCode, completeErr.String(), completeOut.String())
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var finalStatus string
	if err := db.QueryRow(`SELECT status FROM comm_call_sessions WHERE workspace_id = 'ws1' AND provider_call_id = 'CA123'`).Scan(&finalStatus); err != nil {
		t.Fatalf("query call session status: %v", err)
	}
	if finalStatus != "completed" {
		t.Fatalf("expected final call status completed, got %s", finalStatus)
	}

	var transcriptCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_events WHERE event_type = 'VOICE_TRANSCRIPT'`).Scan(&transcriptCount); err != nil {
		t.Fatalf("count transcript events: %v", err)
	}
	if transcriptCount != 1 {
		t.Fatalf("expected one transcript event, got %d", transcriptCount)
	}
}

func TestRunChannelTwilioWebhookReplayCommand(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-replay.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(daemonServer,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	expectedRequestURL := "https://agent.local/webhook" + defaultTwilioWebhookSMSPath()
	fixture := map[string]any{
		"kind": "sms",
		"params": map[string]string{
			"From":       "+15555550999",
			"To":         "+15555550001",
			"Body":       "fixture message",
			"MessageSid": "SMFIXTURE1",
			"AccountSid": "AC123",
		},
		"request_url": expectedRequestURL,
	}
	fixtureBytes, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	fixturePath := filepath.Join(t.TempDir(), "sms-fixture.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	expectedSignature, err := twilioadapter.ComputeRequestSignature("twilio-auth-token", expectedRequestURL, fixture["params"].(map[string]string))
	if err != nil {
		t.Fatalf("compute expected signature: %v", err)
	}

	var (
		gotSignature  string
		gotRequestURL string
	)
	replayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != defaultTwilioWebhookSMSPath() {
			t.Fatalf("unexpected replay path: %s", r.URL.Path)
		}
		gotSignature = strings.TrimSpace(r.Header.Get("X-Twilio-Signature"))
		gotRequestURL = strings.TrimSpace(r.Header.Get("X-Twilio-Request-URL"))
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse replay form: %v", err)
		}
		if got := r.Form.Get("MessageSid"); got != "SMFIXTURE1" {
			t.Fatalf("expected MessageSid SMFIXTURE1, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer replayServer.Close()

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	replayCode := run(withDaemonArgs(daemonServer,
		"connector", "twilio", "webhook", "replay",
		"--workspace", "ws1",
		"--fixture", fixturePath,
		"--base-url", replayServer.URL,
		"--signature-mode", "strict",
	), replayOut, replayErr)
	if replayCode != 0 {
		t.Fatalf("webhook replay failed: code=%d stderr=%s output=%s", replayCode, replayErr.String(), replayOut.String())
	}

	if gotSignature != expectedSignature {
		t.Fatalf("expected signature %s, got %s", expectedSignature, gotSignature)
	}
	if gotRequestURL != expectedRequestURL {
		t.Fatalf("expected request url %s, got %s", expectedRequestURL, gotRequestURL)
	}

	var replayResponse map[string]any
	if err := json.Unmarshal(replayOut.Bytes(), &replayResponse); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if got := replayResponse["status_code"]; got != float64(http.StatusOK) {
		t.Fatalf("expected replay status code 200, got %v", got)
	}
}

func TestRunChannelTwilioWebhookServeHandlesSMSAndVoice(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-serve.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	serveOut := &bytes.Buffer{}
	serveErr := &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		code := run(withDaemonArgs(server,
			"--timeout", "4s",
			"connector", "twilio", "webhook", "serve",
			"--workspace", "ws1",
			"--listen", address,
			"--signature-mode", "bypass",
			"--run-for", "1200ms",
		), serveOut, serveErr)
		done <- code
	}()

	time.Sleep(200 * time.Millisecond)

	smsResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookSMSPath(), url.Values{
		"From":       {"+15555550999"},
		"To":         {"+15555550001"},
		"Body":       {"harness sms"},
		"MessageSid": {"SMHARNESS1"},
		"AccountSid": {"AC123"},
	})
	if err != nil {
		t.Fatalf("post sms webhook: %v", err)
	}
	defer smsResponse.Body.Close()
	if smsResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(smsResponse.Body)
		t.Fatalf("expected sms webhook 200, got %d body=%s", smsResponse.StatusCode, strings.TrimSpace(string(body)))
	}

	voiceResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookVoicePath(), url.Values{
		"CallSid":    {"CAHARNESS1"},
		"AccountSid": {"AC123"},
		"From":       {"+15555550002"},
		"To":         {"+15555550999"},
		"Direction":  {"outbound-api"},
		"CallStatus": {"ringing"},
	})
	if err != nil {
		t.Fatalf("post voice webhook: %v", err)
	}
	defer voiceResponse.Body.Close()
	if voiceResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(voiceResponse.Body)
		t.Fatalf("expected voice webhook 200, got %d body=%s", voiceResponse.StatusCode, strings.TrimSpace(string(body)))
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("webhook serve command failed: code=%d stderr=%s output=%s", code, serveErr.String(), serveOut.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for webhook serve command to exit")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_webhook_receipts`).Scan(&count); err != nil {
		t.Fatalf("count webhook receipts: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two webhook receipts, got %d", count)
	}

	var callSessionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_call_sessions`).Scan(&callSessionCount); err != nil {
		t.Fatalf("count call sessions: %v", err)
	}
	if callSessionCount != 1 {
		t.Fatalf("expected one call session, got %d", callSessionCount)
	}
}

func TestRunChannelTwilioWebhookServeCloudflaredMissingFallsBackToLocal(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-cloudflared-fallback.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	t.Setenv("PATH", t.TempDir())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	serveOut := &bytes.Buffer{}
	serveErr := &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		code := run(withDaemonArgs(server,
			"--timeout", "4s",
			"connector", "twilio", "webhook", "serve",
			"--workspace", "ws1",
			"--listen", address,
			"--signature-mode", "bypass",
			"--cloudflared-mode", "auto",
			"--run-for", "1200ms",
		), serveOut, serveErr)
		done <- code
	}()

	time.Sleep(200 * time.Millisecond)
	webhookResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookSMSPath(), url.Values{
		"From":       {"+15555550999"},
		"To":         {"+15555550001"},
		"Body":       {"fallback check"},
		"MessageSid": {"SMFALLBACK1"},
		"AccountSid": {"AC123"},
	})
	if err != nil {
		t.Fatalf("post sms webhook: %v", err)
	}
	defer webhookResponse.Body.Close()
	if webhookResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(webhookResponse.Body)
		t.Fatalf("expected sms webhook 200, got %d body=%s", webhookResponse.StatusCode, strings.TrimSpace(string(body)))
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("webhook serve command failed: code=%d stderr=%s output=%s", code, serveErr.String(), serveOut.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for webhook serve command to exit")
	}

	payload := map[string]any{}
	if err := json.Unmarshal(serveOut.Bytes(), &payload); err != nil {
		t.Fatalf("decode webhook serve response: %v", err)
	}
	expectedLocalSMSURL := "http://" + address + defaultTwilioWebhookSMSPath()
	if got := payload["sms_webhook_url"]; got != expectedLocalSMSURL {
		t.Fatalf("expected local sms webhook url %q, got %v", expectedLocalSMSURL, got)
	}
	if got := payload["local_sms_webhook_url"]; got != expectedLocalSMSURL {
		t.Fatalf("expected local_sms_webhook_url %q, got %v", expectedLocalSMSURL, got)
	}
	warningText, _ := payload["warning"].(string)
	if !strings.Contains(strings.ToLower(warningText), "cloudflared is not installed") {
		t.Fatalf("expected cloudflared missing warning, got %q", warningText)
	}
}

func TestRunChannelTwilioWebhookServeSMSAssistantReply(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-webhook-chat"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected OpenAI path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"agent\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" reply\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	var (
		twilioSendCount int
		twilioTo        string
		twilioBody      string
	)
	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			t.Fatalf("unexpected Twilio path: %s", r.URL.Path)
		}
		twilioSendCount++
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse twilio send form: %v", err)
		}
		twilioTo = r.Form.Get("To")
		twilioBody = r.Form.Get("Body")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"SMAUTOREPLY1","account_sid":"AC123","status":"queued","from":"+15555550001","to":"+15555550999"}`))
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-assistant-sms.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)
	providerSetCode := run(withDaemonArgs(daemonServer,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if providerSetCode != 0 {
		t.Fatalf("expected provider setup to succeed")
	}

	modelSelectCode := run(withDaemonArgs(daemonServer,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "openai",
		"--model", "gpt-4.1-mini",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if modelSelectCode != 0 {
		t.Fatalf("expected model select to succeed")
	}

	twilioSetCode := run(withDaemonArgs(daemonServer,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if twilioSetCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	serveOut := &bytes.Buffer{}
	serveErr := &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		code := run(withDaemonArgs(daemonServer,
			"--timeout", "5s",
			"connector", "twilio", "webhook", "serve",
			"--workspace", "ws1",
			"--listen", address,
			"--signature-mode", "bypass",
			"--assistant-replies=true",
			"--assistant-task-class", "chat",
			"--assistant-reply-timeout", "4s",
			"--run-for", "1500ms",
		), serveOut, serveErr)
		done <- code
	}()

	time.Sleep(200 * time.Millisecond)

	webhookResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookSMSPath(), url.Values{
		"From":       {"+15555550999"},
		"To":         {"+15555550001"},
		"Body":       {"please reply"},
		"MessageSid": {"SMAUTOIN1"},
		"AccountSid": {"AC123"},
	})
	if err != nil {
		t.Fatalf("post sms webhook: %v", err)
	}
	defer webhookResponse.Body.Close()
	if webhookResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(webhookResponse.Body)
		t.Fatalf("expected sms webhook 200, got %d body=%s", webhookResponse.StatusCode, strings.TrimSpace(string(body)))
	}
	bodyBytes, err := io.ReadAll(webhookResponse.Body)
	if err != nil {
		t.Fatalf("read webhook response: %v", err)
	}
	var webhookPayload map[string]any
	if err := json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
		t.Fatalf("decode webhook payload: %v", err)
	}
	if got := webhookPayload["assistant_reply"]; got != "agent reply" {
		t.Fatalf("expected assistant reply, got %v", got)
	}
	if got, ok := webhookPayload["assistant_delivered"].(bool); !ok || !got {
		t.Fatalf("expected assistant_delivered=true, got %v", webhookPayload["assistant_delivered"])
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("webhook serve command failed: code=%d stderr=%s output=%s", code, serveErr.String(), serveOut.String())
		}
	case <-time.After(6 * time.Second):
		t.Fatalf("timeout waiting for webhook serve command to exit")
	}

	if twilioSendCount != 1 {
		t.Fatalf("expected one outbound assistant sms send, got %d", twilioSendCount)
	}
	if twilioTo != "+15555550999" {
		t.Fatalf("expected assistant reply destination +15555550999, got %s", twilioTo)
	}
	if twilioBody != "agent reply" {
		t.Fatalf("expected assistant reply body 'agent reply', got %q", twilioBody)
	}
}

func TestRunChannelTwilioWebhookServeVoiceTwiMLResponse(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-webhook-voice"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected OpenAI path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"voice\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" response\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-voice-twiml.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)
	providerSetCode := run(withDaemonArgs(daemonServer,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if providerSetCode != 0 {
		t.Fatalf("expected provider setup to succeed")
	}

	modelSelectCode := run(withDaemonArgs(daemonServer,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "openai",
		"--model", "gpt-4.1-mini",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if modelSelectCode != 0 {
		t.Fatalf("expected model select to succeed")
	}

	twilioSetCode := run(withDaemonArgs(daemonServer,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if twilioSetCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	serveOut := &bytes.Buffer{}
	serveErr := &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		code := run(withDaemonArgs(daemonServer,
			"--timeout", "5s",
			"connector", "twilio", "webhook", "serve",
			"--workspace", "ws1",
			"--listen", address,
			"--signature-mode", "bypass",
			"--assistant-replies=true",
			"--voice-response-mode", "twiml",
			"--assistant-task-class", "chat",
			"--assistant-reply-timeout", "4s",
			"--run-for", "1500ms",
		), serveOut, serveErr)
		done <- code
	}()

	time.Sleep(200 * time.Millisecond)

	webhookResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookVoicePath(), url.Values{
		"CallSid":      {"CATWIML1"},
		"AccountSid":   {"AC123"},
		"From":         {"+15555550999"},
		"To":           {"+15555550002"},
		"Direction":    {"inbound"},
		"CallStatus":   {"in-progress"},
		"SpeechResult": {"hello voice"},
	})
	if err != nil {
		t.Fatalf("post voice webhook: %v", err)
	}
	defer webhookResponse.Body.Close()
	if webhookResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(webhookResponse.Body)
		t.Fatalf("expected voice webhook 200, got %d body=%s", webhookResponse.StatusCode, strings.TrimSpace(string(body)))
	}
	contentType := webhookResponse.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/xml") {
		t.Fatalf("expected application/xml content type, got %q", contentType)
	}
	bodyBytes, err := io.ReadAll(webhookResponse.Body)
	if err != nil {
		t.Fatalf("read voice webhook response: %v", err)
	}
	bodyText := strings.TrimSpace(string(bodyBytes))
	if !strings.Contains(bodyText, "<Response>") || !strings.Contains(bodyText, "<Gather") {
		t.Fatalf("expected TwiML gather response, got %q", bodyText)
	}
	if !strings.Contains(bodyText, "voice response") {
		t.Fatalf("expected generated assistant voice response in TwiML body, got %q", bodyText)
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("webhook serve command failed: code=%d stderr=%s output=%s", code, serveErr.String(), serveOut.String())
		}
	case <-time.After(6 * time.Second):
		t.Fatalf("timeout waiting for webhook serve command to exit")
	}
}

func TestRunChannelTwilioSMSChatAndTranscriptCommands(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	setTestChatInput(t, "hello from cli\nsecond message\n/exit\n")
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-twilio-chat"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	sendCount := 0
	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendCount++
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			t.Fatalf("unexpected Twilio path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"sid":"SMCHAT%d","account_sid":"AC123","status":"queued","from":"+15555550001","to":"+15555550999"}`, sendCount)))
	}))
	defer twilioServer.Close()
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected OpenAI path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"chat\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" reply\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-sms-chat.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	providerSetCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if providerSetCode != 0 {
		t.Fatalf("expected provider setup to succeed")
	}
	modelSelectCode := run(withDaemonArgs(server,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "openai",
		"--model", "gpt-4.1-mini",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if modelSelectCode != 0 {
		t.Fatalf("expected model select to succeed")
	}
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	chatOut := &bytes.Buffer{}
	chatErr := &bytes.Buffer{}
	chatCode := run(withDaemonArgs(server,
		"connector", "twilio", "sms-chat",
		"--workspace", "ws1",
		"--to", "+15555550999",
		"--interactive=true",
	), chatOut, chatErr)
	if chatCode != 0 {
		t.Fatalf("sms-chat failed: code=%d stderr=%s output=%s", chatCode, chatErr.String(), chatOut.String())
	}

	var chatResponse map[string]any
	if err := json.Unmarshal(chatOut.Bytes(), &chatResponse); err != nil {
		t.Fatalf("decode sms-chat response: %v", err)
	}
	turns, ok := chatResponse["turns"].([]any)
	if !ok || len(turns) != 2 {
		t.Fatalf("expected 2 chat turns, got %v", chatResponse["turns"])
	}
	for _, raw := range turns {
		turn, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected turn shape: %T", raw)
		}
		if success, ok := turn["success"].(bool); !ok || !success {
			t.Fatalf("expected turn success=true, got %v", turn["success"])
		}
		if reply, ok := turn["assistant_reply"].(string); !ok || strings.TrimSpace(reply) == "" {
			t.Fatalf("expected assistant_reply for twilio sms chat turn, got %v", turn["assistant_reply"])
		}
		if assistantErr, _ := turn["assistant_error"].(string); strings.TrimSpace(assistantErr) != "" {
			t.Fatalf("expected empty assistant_error for successful turn, got %q", assistantErr)
		}
	}

	transcriptOut := &bytes.Buffer{}
	transcriptErr := &bytes.Buffer{}
	transcriptCode := run(withDaemonArgs(server,
		"connector", "twilio", "transcript",
		"--workspace", "ws1",
		"--limit", "10",
	), transcriptOut, transcriptErr)
	if transcriptCode != 0 {
		t.Fatalf("transcript command failed: code=%d stderr=%s output=%s", transcriptCode, transcriptErr.String(), transcriptOut.String())
	}

	var transcriptResponse map[string]any
	if err := json.Unmarshal(transcriptOut.Bytes(), &transcriptResponse); err != nil {
		t.Fatalf("decode transcript response: %v", err)
	}
	events, ok := transcriptResponse["events"].([]any)
	if !ok || len(events) < 2 {
		t.Fatalf("expected at least 2 transcript events, got %v", transcriptResponse["events"])
	}
}

func TestRunChannelTwilioCallStatusAndTranscriptForCall(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Calls.json" {
			t.Fatalf("unexpected Twilio path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"CAWORKFLOW1","account_sid":"AC123","status":"queued","direction":"outbound-api","from":"+15555550002","to":"+15555550999"}`))
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-call-workflow.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-auth-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected twilio setup to succeed")
	}

	startCode := run(withDaemonArgs(server,
		"connector", "twilio", "start-call",
		"--workspace", "ws1",
		"--to", "+15555550999",
		"--twiml-url", "https://agent.local/twiml/voice",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if startCode != 0 {
		t.Fatalf("expected start-call to succeed")
	}

	ingestCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--skip-signature=true",
		"--provider-event-id", "workflow-voice-cb-1",
		"--call-sid", "CAWORKFLOW1",
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "in-progress",
		"--transcript", "workflow transcript",
		"--transcript-direction", "INBOUND",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if ingestCode != 0 {
		t.Fatalf("expected ingest-voice to succeed")
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run(withDaemonArgs(server,
		"connector", "twilio", "call-status",
		"--workspace", "ws1",
		"--call-sid", "CAWORKFLOW1",
	), statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("call-status failed: code=%d stderr=%s output=%s", statusCode, statusErr.String(), statusOut.String())
	}

	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode call-status response: %v", err)
	}
	sessions, ok := statusResponse["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected one call session record, got %v", statusResponse["sessions"])
	}
	session, ok := sessions[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected session payload shape: %T", sessions[0])
	}
	if got := session["status"]; got != "in_progress" {
		t.Fatalf("expected call status in_progress, got %v", got)
	}

	transcriptOut := &bytes.Buffer{}
	transcriptErr := &bytes.Buffer{}
	transcriptCode := run(withDaemonArgs(server,
		"connector", "twilio", "transcript",
		"--workspace", "ws1",
		"--call-sid", "CAWORKFLOW1",
		"--limit", "10",
	), transcriptOut, transcriptErr)
	if transcriptCode != 0 {
		t.Fatalf("transcript call-sid filter failed: code=%d stderr=%s output=%s", transcriptCode, transcriptErr.String(), transcriptOut.String())
	}

	var transcriptResponse map[string]any
	if err := json.Unmarshal(transcriptOut.Bytes(), &transcriptResponse); err != nil {
		t.Fatalf("decode transcript response: %v", err)
	}
	events, ok := transcriptResponse["events"].([]any)
	if !ok || len(events) == 0 {
		t.Fatalf("expected transcript events for call sid, got %v", transcriptResponse["events"])
	}
}

func TestRunModelSelectAndResolveCommands(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-model-test"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "model-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	openAISetCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", "https://api.openai.com/v1",
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if openAISetCode != 0 {
		t.Fatalf("expected openai provider setup to succeed")
	}

	ollamaSetCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--endpoint", "http://127.0.0.1:11434",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if ollamaSetCode != 0 {
		t.Fatalf("expected ollama provider setup to succeed")
	}

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"model", "list",
		"--workspace", "ws1",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("model list exit code %d stderr=%s", listCode, listErr.String())
	}

	var listResponse map[string]any
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode model list response: %v", err)
	}
	models, ok := listResponse["models"].([]any)
	if !ok || len(models) < 4 {
		t.Fatalf("expected seeded model catalog entries, got %v", listResponse["models"])
	}

	selectOut := &bytes.Buffer{}
	selectErr := &bytes.Buffer{}
	selectCode := run(withDaemonArgs(server,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "ollama",
		"--model", "llama3.2",
	), selectOut, selectErr)
	if selectCode != 0 {
		t.Fatalf("model select exit code %d stderr=%s", selectCode, selectErr.String())
	}

	resolveOut := &bytes.Buffer{}
	resolveErr := &bytes.Buffer{}
	resolveCode := run(withDaemonArgs(server,
		"model", "resolve",
		"--workspace", "ws1",
		"--task-class", "chat",
	), resolveOut, resolveErr)
	if resolveCode != 0 {
		t.Fatalf("model resolve exit code %d stderr=%s output=%s", resolveCode, resolveErr.String(), resolveOut.String())
	}

	var resolveResponse map[string]any
	if err := json.Unmarshal(resolveOut.Bytes(), &resolveResponse); err != nil {
		t.Fatalf("decode resolve response: %v", err)
	}
	if got := resolveResponse["provider"]; got != "ollama" {
		t.Fatalf("expected provider ollama, got %v", got)
	}
	if got := resolveResponse["model_key"]; got != "llama3.2" {
		t.Fatalf("expected model llama3.2, got %v", got)
	}
	if got := resolveResponse["source"]; got != "task_class_policy" {
		t.Fatalf("expected source task_class_policy, got %v", got)
	}
}

func TestRunModelResolveFallbackToEnabledProvider(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	dbPath := filepath.Join(t.TempDir(), "model-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	ollamaSetCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--endpoint", "http://127.0.0.1:11434",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if ollamaSetCode != 0 {
		t.Fatalf("expected ollama provider setup to succeed")
	}

	resolveOut := &bytes.Buffer{}
	resolveErr := &bytes.Buffer{}
	resolveCode := run(withDaemonArgs(server,
		"model", "resolve",
		"--workspace", "ws1",
		"--task-class", "assistant",
	), resolveOut, resolveErr)
	if resolveCode != 0 {
		t.Fatalf("model resolve exit code %d stderr=%s output=%s", resolveCode, resolveErr.String(), resolveOut.String())
	}

	var resolveResponse map[string]any
	if err := json.Unmarshal(resolveOut.Bytes(), &resolveResponse); err != nil {
		t.Fatalf("decode resolve response: %v", err)
	}
	if got := resolveResponse["provider"]; got != "ollama" {
		t.Fatalf("expected fallback provider ollama, got %v", got)
	}
	if got := resolveResponse["source"]; got != "fallback_enabled" {
		t.Fatalf("expected fallback source, got %v", got)
	}
}

func TestRunModelDiscoverAddRemoveCommands(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("expected ollama tags endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"},{"name":"mistral"}]}`))
	}))
	defer ollamaServer.Close()

	dbPath := filepath.Join(t.TempDir(), "model-discover.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	providerSetCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--endpoint", ollamaServer.URL,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if providerSetCode != 0 {
		t.Fatalf("expected ollama provider setup to succeed")
	}

	discoverOut := &bytes.Buffer{}
	discoverErr := &bytes.Buffer{}
	discoverCode := run(withDaemonArgs(server,
		"model", "discover",
		"--workspace", "ws1",
		"--provider", "ollama",
	), discoverOut, discoverErr)
	if discoverCode != 0 {
		t.Fatalf("model discover exit code %d stderr=%s output=%s", discoverCode, discoverErr.String(), discoverOut.String())
	}

	var discoverResponse map[string]any
	if err := json.Unmarshal(discoverOut.Bytes(), &discoverResponse); err != nil {
		t.Fatalf("decode model discover response: %v", err)
	}
	results, ok := discoverResponse["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected one provider discovery result, got %v", discoverResponse["results"])
	}

	addOut := &bytes.Buffer{}
	addErr := &bytes.Buffer{}
	addCode := run(withDaemonArgs(server,
		"model", "add",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--model", "llama3.2-custom",
		"--enabled=true",
	), addOut, addErr)
	if addCode != 0 {
		t.Fatalf("model add exit code %d stderr=%s output=%s", addCode, addErr.String(), addOut.String())
	}

	listAfterAddOut := &bytes.Buffer{}
	listAfterAddErr := &bytes.Buffer{}
	listAfterAddCode := run(withDaemonArgs(server,
		"model", "list",
		"--workspace", "ws1",
		"--provider", "ollama",
	), listAfterAddOut, listAfterAddErr)
	if listAfterAddCode != 0 {
		t.Fatalf("model list after add exit code %d stderr=%s", listAfterAddCode, listAfterAddErr.String())
	}

	var listAfterAddResponse map[string]any
	if err := json.Unmarshal(listAfterAddOut.Bytes(), &listAfterAddResponse); err != nil {
		t.Fatalf("decode list after add response: %v", err)
	}
	modelsAfterAdd, ok := listAfterAddResponse["models"].([]any)
	if !ok {
		t.Fatalf("expected models array after add")
	}
	if !modelListContainsModel(modelsAfterAdd, "ollama", "llama3.2-custom") {
		t.Fatalf("expected custom model to appear after add, got %v", listAfterAddResponse["models"])
	}

	removeOut := &bytes.Buffer{}
	removeErr := &bytes.Buffer{}
	removeCode := run(withDaemonArgs(server,
		"model", "remove",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--model", "llama3.2-custom",
	), removeOut, removeErr)
	if removeCode != 0 {
		t.Fatalf("model remove exit code %d stderr=%s output=%s", removeCode, removeErr.String(), removeOut.String())
	}

	listAfterRemoveOut := &bytes.Buffer{}
	listAfterRemoveErr := &bytes.Buffer{}
	listAfterRemoveCode := run(withDaemonArgs(server,
		"model", "list",
		"--workspace", "ws1",
		"--provider", "ollama",
	), listAfterRemoveOut, listAfterRemoveErr)
	if listAfterRemoveCode != 0 {
		t.Fatalf("model list after remove exit code %d stderr=%s", listAfterRemoveCode, listAfterRemoveErr.String())
	}

	var listAfterRemoveResponse map[string]any
	if err := json.Unmarshal(listAfterRemoveOut.Bytes(), &listAfterRemoveResponse); err != nil {
		t.Fatalf("decode list after remove response: %v", err)
	}
	modelsAfterRemove, ok := listAfterRemoveResponse["models"].([]any)
	if !ok {
		t.Fatalf("expected models array after remove")
	}
	if modelListContainsModel(modelsAfterRemove, "ollama", "llama3.2-custom") {
		t.Fatalf("expected custom model to be removed, got %v", listAfterRemoveResponse["models"])
	}
}

func TestRunChatOneShotOpenAIStreaming(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-chat-test"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected openai chat endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "chat-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("provider set openai failed")
	}

	chatOut := &bytes.Buffer{}
	chatErr := &bytes.Buffer{}
	chatCode := run(withDaemonArgs(server,
		"chat",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--message", "hello",
	), chatOut, chatErr)
	if chatCode != 0 {
		t.Fatalf("chat command failed: code=%d stderr=%s output=%s", chatCode, chatErr.String(), chatOut.String())
	}
	if !strings.Contains(chatOut.String(), "hello world") {
		t.Fatalf("expected streamed response in output, got %q", chatOut.String())
	}
}

func TestRunChatInteractiveMultiTurn(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-chat-test"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	requestCount := 0
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n"))
		} else {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"second\"}}]}\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "chat-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("provider set openai failed")
	}

	setTestChatInput(t, "hello\nagain\n/exit\n")

	chatOut := &bytes.Buffer{}
	chatErr := &bytes.Buffer{}
	chatCode := run(withDaemonArgs(server,
		"chat",
		"--workspace", "ws1",
		"--task-class", "chat",
	), chatOut, chatErr)
	if chatCode != 0 {
		t.Fatalf("chat interactive failed: code=%d stderr=%s output=%s", chatCode, chatErr.String(), chatOut.String())
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 chat requests, got %d", requestCount)
	}
	if !strings.Contains(chatOut.String(), "first") || !strings.Contains(chatOut.String(), "second") {
		t.Fatalf("expected streamed turn outputs, got %q", chatOut.String())
	}
}

func TestRunAgentCommandPersistsTaskRunAndSteps(t *testing.T) {
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")

	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "open https://example.com and summarize",
	), stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("agent run exit code %d, stderr=%s output=%s", exitCode, stderr.String(), stdout.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode agent response: %v", err)
	}
	if got := response["workflow"]; got != "browser" {
		t.Fatalf("expected browser workflow, got %v", got)
	}
	taskID, ok := response["task_id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("expected task_id in response")
	}
	runID, ok := response["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id in response")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	var taskState string
	if err := db.QueryRow(`SELECT state FROM tasks WHERE id = ?`, taskID).Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if taskState != "completed" {
		t.Fatalf("expected task state completed, got %s", taskState)
	}

	var runState string
	if err := db.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, runID).Scan(&runState); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if runState != "completed" {
		t.Fatalf("expected run state completed, got %s", runState)
	}

	var completedSteps int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_steps WHERE run_id = ? AND status = 'completed'`, runID).Scan(&completedSteps); err != nil {
		t.Fatalf("count completed steps: %v", err)
	}
	if completedSteps != 3 {
		t.Fatalf("expected 3 completed steps, got %d", completedSteps)
	}
}

func TestRunAgentCommandRejectsUnknownIntent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "do something magical",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected agent run to fail for unknown intent")
	}
	if !strings.Contains(stderr.String(), "unable to determine intent") {
		t.Fatalf("expected unknown intent error, got stderr=%s", stderr.String())
	}
}

func TestRunAgentCommandReturnsClarificationForMissingFinderPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file now",
	), stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("expected clarification response, code=%d stderr=%s output=%s", exitCode, stderr.String(), stdout.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode clarification response: %v", err)
	}
	if got, _ := response["workflow"].(string); got != "finder" {
		t.Fatalf("expected finder workflow, got %v", response["workflow"])
	}
	if required, ok := response["clarification_required"].(bool); !ok || !required {
		t.Fatalf("expected clarification_required=true, got %v", response["clarification_required"])
	}
	if state, _ := response["task_state"].(string); state != "clarification_required" {
		t.Fatalf("expected task_state clarification_required, got %v", response["task_state"])
	}
	missingSlots, ok := response["missing_slots"].([]any)
	if !ok || len(missingSlots) == 0 {
		t.Fatalf("expected missing_slots in clarification response, got %v", response["missing_slots"])
	}
}

func TestRunAgentApprovalFlow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	runOut := &bytes.Buffer{}
	runErr := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/test.txt",
	), runOut, runErr)
	if runCode != 0 {
		t.Fatalf("agent destructive run failed: code=%d stderr=%s output=%s", runCode, runErr.String(), runOut.String())
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if required, ok := runResponse["approval_required"].(bool); !ok || !required {
		t.Fatalf("expected approval_required=true, got %v", runResponse["approval_required"])
	}
	approvalID, ok := runResponse["approval_request_id"].(string)
	if !ok || approvalID == "" {
		t.Fatalf("expected approval_request_id")
	}
	runID, ok := runResponse["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id")
	}

	approveOut := &bytes.Buffer{}
	approveErr := &bytes.Buffer{}
	approveCode := run(withDaemonArgs(server,
		"agent", "approve",
		"--workspace", "ws1",
		"--approval-id", approvalID,
		"--phrase", "GO AHEAD",
		"--actor-id", "actor.requester",
	), approveOut, approveErr)
	if approveCode != 0 {
		t.Fatalf("agent approve failed: code=%d stderr=%s output=%s", approveCode, approveErr.String(), approveOut.String())
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	var runState string
	if err := db.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, runID).Scan(&runState); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if runState != "completed" {
		t.Fatalf("expected completed run state after approval, got %s", runState)
	}

	var approvalDecision sql.NullString
	if err := db.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, approvalID).Scan(&approvalDecision); err != nil {
		t.Fatalf("query approval decision: %v", err)
	}
	if !approvalDecision.Valid || approvalDecision.String != "APPROVED" {
		t.Fatalf("expected approval decision APPROVED, got %v", approvalDecision)
	}
}

func TestRunAgentVoiceDestructiveRequiresInAppHandoff(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	runOut := &bytes.Buffer{}
	runErr := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/test.txt",
		"--origin", "voice",
		"--approval-phrase", "GO AHEAD",
	), runOut, runErr)
	if runCode != 0 {
		t.Fatalf("voice destructive run failed: code=%d stderr=%s output=%s", runCode, runErr.String(), runOut.String())
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if required, ok := runResponse["approval_required"].(bool); !ok || !required {
		t.Fatalf("expected approval_required=true for unconfirmed voice destructive action, got %v", runResponse["approval_required"])
	}
	if state, _ := runResponse["task_state"].(string); state != "awaiting_approval" {
		t.Fatalf("expected task_state awaiting_approval, got %v", runResponse["task_state"])
	}

	stepStates, ok := runResponse["step_states"].([]any)
	if !ok || len(stepStates) == 0 {
		t.Fatalf("expected step_states in run response, got %v", runResponse["step_states"])
	}
	foundVoiceHandoffSummary := false
	for _, rawStep := range stepStates {
		step, ok := rawStep.(map[string]any)
		if !ok {
			continue
		}
		summary, _ := step["summary"].(string)
		if strings.Contains(strings.ToLower(summary), "in-app approval handoff") {
			foundVoiceHandoffSummary = true
			break
		}
	}
	if !foundVoiceHandoffSummary {
		t.Fatalf("expected voice handoff summary, got %v", runResponse["step_states"])
	}
}

func TestRunAgentVoiceDestructiveAllowsConfirmedInAppHandoff(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	runOut := &bytes.Buffer{}
	runErr := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/test.txt",
		"--origin", "voice",
		"--in-app-approval-confirmed=true",
	), runOut, runErr)
	if runCode != 0 {
		t.Fatalf("voice destructive confirmed run failed: code=%d stderr=%s output=%s", runCode, runErr.String(), runOut.String())
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if state, _ := runResponse["task_state"].(string); state != "completed" {
		t.Fatalf("expected task_state completed, got %v", runResponse["task_state"])
	}
	if runResponse["approval_required"] == true {
		t.Fatalf("expected no pending approval for confirmed voice handoff")
	}
}

func TestRunAgentApproveRejectsNonExactPhrase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	runOut := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/test.txt",
	), runOut, &bytes.Buffer{})
	if runCode != 0 {
		t.Fatalf("expected pending destructive run")
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	approvalID := runResponse["approval_request_id"].(string)

	approveErr := &bytes.Buffer{}
	approveCode := run(withDaemonArgs(server,
		"agent", "approve",
		"--workspace", "ws1",
		"--approval-id", approvalID,
		"--phrase", "go ahead",
		"--actor-id", "actor.requester",
	), &bytes.Buffer{}, approveErr)
	if approveCode == 0 {
		t.Fatalf("expected non-exact phrase to fail")
	}
	if !strings.Contains(approveErr.String(), "approval phrase must be exact") {
		t.Fatalf("expected exact phrase validation error, got %s", approveErr.String())
	}
}

func TestRunAgentApproveRejectsUnauthorizedDecisionActor(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	runOut := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/test.txt",
	), runOut, &bytes.Buffer{})
	if runCode != 0 {
		t.Fatalf("expected pending destructive run")
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	approvalID := runResponse["approval_request_id"].(string)

	approveErr := &bytes.Buffer{}
	approveCode := run(withDaemonArgs(server,
		"agent", "approve",
		"--workspace", "ws1",
		"--approval-id", approvalID,
		"--phrase", "GO AHEAD",
		"--actor-id", "actor.approver",
	), &bytes.Buffer{}, approveErr)
	if approveCode == 0 {
		t.Fatalf("expected unauthorized approver to fail")
	}
	if !strings.Contains(approveErr.String(), "approval denied") {
		t.Fatalf("expected approval denied error, got %s", approveErr.String())
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	var approvalDecision sql.NullString
	if err := db.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, approvalID).Scan(&approvalDecision); err != nil {
		t.Fatalf("query approval decision: %v", err)
	}
	if approvalDecision.Valid {
		t.Fatalf("expected no approval decision for unauthorized actor, got %v", approvalDecision.String)
	}
}

func TestRunDelegationGrantCheckRevokeFlow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegation-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	grantOut := &bytes.Buffer{}
	grantErr := &bytes.Buffer{}
	grantCode := run(withDaemonArgs(server,
		"delegation", "grant",
		"--workspace", "ws1",
		"--from", "actor.a",
		"--to", "actor.b",
		"--scope-type", "EXECUTION",
	), grantOut, grantErr)
	if grantCode != 0 {
		t.Fatalf("delegation grant failed: code=%d stderr=%s", grantCode, grantErr.String())
	}

	var grantResponse map[string]any
	if err := json.Unmarshal(grantOut.Bytes(), &grantResponse); err != nil {
		t.Fatalf("decode grant response: %v", err)
	}
	ruleID, ok := grantResponse["id"].(string)
	if !ok || ruleID == "" {
		t.Fatalf("expected rule id in grant response")
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"delegation", "check",
		"--workspace", "ws1",
		"--requested-by", "actor.a",
		"--acting-as", "actor.b",
	), checkOut, checkErr)
	if checkCode != 0 {
		t.Fatalf("delegation check failed: code=%d stderr=%s", checkCode, checkErr.String())
	}
	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if allowed, ok := checkResponse["allowed"].(bool); !ok || !allowed {
		t.Fatalf("expected allowed delegation decision, got %v", checkResponse["allowed"])
	}

	revokeOut := &bytes.Buffer{}
	revokeErr := &bytes.Buffer{}
	revokeCode := run(withDaemonArgs(server,
		"delegation", "revoke",
		"--workspace", "ws1",
		"--rule-id", ruleID,
	), revokeOut, revokeErr)
	if revokeCode != 0 {
		t.Fatalf("delegation revoke failed: code=%d stderr=%s", revokeCode, revokeErr.String())
	}
}

func TestRunDelegationCheckReturnsDenyReasonAfterRevoke(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegation-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	grantOut := &bytes.Buffer{}
	grantCode := run(withDaemonArgs(server,
		"delegation", "grant",
		"--workspace", "ws1",
		"--from", "actor.a",
		"--to", "actor.b",
		"--scope-type", "EXECUTION",
	), grantOut, &bytes.Buffer{})
	if grantCode != 0 {
		t.Fatalf("expected delegation grant success")
	}

	var grantResponse map[string]any
	if err := json.Unmarshal(grantOut.Bytes(), &grantResponse); err != nil {
		t.Fatalf("decode grant response: %v", err)
	}
	ruleID, _ := grantResponse["id"].(string)
	if strings.TrimSpace(ruleID) == "" {
		t.Fatalf("expected delegation rule id in grant response")
	}

	revokeCode := run(withDaemonArgs(server,
		"delegation", "revoke",
		"--workspace", "ws1",
		"--rule-id", ruleID,
	), &bytes.Buffer{}, &bytes.Buffer{})
	if revokeCode != 0 {
		t.Fatalf("expected delegation revoke success")
	}

	checkOut := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"delegation", "check",
		"--workspace", "ws1",
		"--requested-by", "actor.a",
		"--acting-as", "actor.b",
		"--scope-type", "EXECUTION",
	), checkOut, &bytes.Buffer{})
	if checkCode != 0 {
		t.Fatalf("expected delegation check command success")
	}

	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if allowed, _ := checkResponse["allowed"].(bool); allowed {
		t.Fatalf("expected delegation check to deny after revoke, got %+v", checkResponse)
	}
	if reasonCode, _ := checkResponse["reason_code"].(string); reasonCode != "missing_delegation_rule" {
		t.Fatalf("expected missing_delegation_rule reason code, got %+v", checkResponse)
	}
}

func TestRunDelegationGrantRejectsInvalidScopeType(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegation-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"delegation", "grant",
		"--workspace", "ws1",
		"--from", "actor.a",
		"--to", "actor.b",
		"--scope-type", "INVALID_SCOPE",
	), &bytes.Buffer{}, stderr)
	if exitCode == 0 {
		t.Fatalf("expected invalid scope type to fail")
	}
	if !strings.Contains(stderr.String(), "unsupported scope_type") {
		t.Fatalf("expected unsupported scope_type error, got %s", stderr.String())
	}
}

func TestRunAgentDeniesCrossPrincipalWithoutDelegation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegation-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "send an email update to sam@example.com",
		"--requested-by", "actor.a",
		"--subject", "actor.b",
		"--acting-as", "actor.b",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected cross-principal agent run denial without delegation")
	}
	if !strings.Contains(stderr.String(), "acting_as denied") {
		t.Fatalf("expected acting_as denial, got stderr=%s", stderr.String())
	}
}

func TestRunAgentAllowsCrossPrincipalWithDelegation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegation-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	grantCode := run(withDaemonArgs(server,
		"delegation", "grant",
		"--workspace", "ws1",
		"--from", "actor.a",
		"--to", "actor.b",
		"--scope-type", "EXECUTION",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if grantCode != 0 {
		t.Fatalf("expected delegation grant success")
	}

	runOut := &bytes.Buffer{}
	runErr := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "send an email update to sam@example.com",
		"--requested-by", "actor.a",
		"--subject", "actor.b",
		"--acting-as", "actor.b",
	), runOut, runErr)
	if runCode != 0 {
		t.Fatalf("expected agent run success with delegation, code=%d stderr=%s", runCode, runErr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := response["workflow"]; got != "mail" {
		t.Fatalf("expected mail workflow, got %v", got)
	}
}
