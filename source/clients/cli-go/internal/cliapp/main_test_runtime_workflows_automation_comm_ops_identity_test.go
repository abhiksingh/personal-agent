package cliapp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"

	_ "modernc.org/sqlite"
)

func TestRunAutomationScheduleCreateListAndIdempotentRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "automation-schedule.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	createOut := &bytes.Buffer{}
	createErr := &bytes.Buffer{}
	createCode := run(withDaemonArgs(server,
		"automation", "create",
		"--workspace", "ws1",
		"--subject", "actor.auto",
		"--trigger-type", "SCHEDULE",
		"--title", "Schedule Trigger",
		"--instruction", "Run schedule task",
		"--interval-seconds", "60",
	), createOut, createErr)
	if createCode != 0 {
		t.Fatalf("automation create schedule failed: code=%d stderr=%s output=%s", createCode, createErr.String(), createOut.String())
	}

	var createResponse map[string]any
	if err := json.Unmarshal(createOut.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if got := createResponse["trigger_type"]; got != "SCHEDULE" {
		t.Fatalf("expected SCHEDULE trigger type, got %v", got)
	}
	triggerID, ok := createResponse["trigger_id"].(string)
	if !ok || triggerID == "" {
		t.Fatalf("expected trigger_id in create response")
	}

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"automation", "list",
		"--workspace", "ws1",
		"--trigger-type", "SCHEDULE",
		"--include-disabled=false",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("automation list schedule failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}

	var listResponse map[string]any
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	triggers, ok := listResponse["triggers"].([]any)
	if !ok || len(triggers) != 1 {
		t.Fatalf("expected one listed schedule trigger, got %v", listResponse["triggers"])
	}
	trigger, ok := triggers[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected trigger payload shape: %T", triggers[0])
	}
	if got := trigger["trigger_id"]; got != triggerID {
		t.Fatalf("expected listed trigger_id %s, got %v", triggerID, got)
	}

	slotTime := "2026-02-24T20:00:30Z"
	run1Out := &bytes.Buffer{}
	run1Err := &bytes.Buffer{}
	run1Code := run(withDaemonArgs(server,
		"automation", "run", "schedule",
		"--at", slotTime,
	), run1Out, run1Err)
	if run1Code != 0 {
		t.Fatalf("automation run schedule first failed: code=%d stderr=%s output=%s", run1Code, run1Err.String(), run1Out.String())
	}

	var run1Response map[string]any
	if err := json.Unmarshal(run1Out.Bytes(), &run1Response); err != nil {
		t.Fatalf("decode first run response: %v", err)
	}
	run1Result, ok := run1Response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected first run result payload")
	}
	if got := run1Result["Created"]; got != float64(1) {
		t.Fatalf("expected first schedule run Created=1, got %v", got)
	}

	run2Out := &bytes.Buffer{}
	run2Err := &bytes.Buffer{}
	run2Code := run(withDaemonArgs(server,
		"automation", "run", "schedule",
		"--at", slotTime,
	), run2Out, run2Err)
	if run2Code != 0 {
		t.Fatalf("automation run schedule replay failed: code=%d stderr=%s output=%s", run2Code, run2Err.String(), run2Out.String())
	}

	var run2Response map[string]any
	if err := json.Unmarshal(run2Out.Bytes(), &run2Response); err != nil {
		t.Fatalf("decode second run response: %v", err)
	}
	run2Result, ok := run2Response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected second run result payload")
	}
	if got := run2Result["Created"]; got != float64(0) {
		t.Fatalf("expected second schedule run Created=0 due to idempotency, got %v", got)
	}
	if got := run2Result["Skipped"]; got != float64(1) {
		t.Fatalf("expected second schedule run Skipped=1, got %v", got)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var taskCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE workspace_id = 'ws1'`).Scan(&taskCount); err != nil {
		t.Fatalf("query task count: %v", err)
	}
	if taskCount != 1 {
		t.Fatalf("expected exactly 1 scheduled task from replay-safe run, got %d", taskCount)
	}

	var fireCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM trigger_fires WHERE workspace_id = 'ws1'`).Scan(&fireCount); err != nil {
		t.Fatalf("query trigger fire count: %v", err)
	}
	if fireCount != 1 {
		t.Fatalf("expected exactly 1 schedule trigger fire from replay-safe run, got %d", fireCount)
	}
}

func TestRunAutomationCommEventCreateAndIdempotentRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "automation-comm.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	createOut := &bytes.Buffer{}
	createErr := &bytes.Buffer{}
	createCode := run(withDaemonArgs(server,
		"automation", "create",
		"--workspace", "ws1",
		"--subject", "actor.auto",
		"--trigger-type", "ON_COMM_EVENT",
		"--title", "Comm Trigger",
		"--instruction", "Handle inbound comm events",
		"--filter", `{"channels":["imessage"]}`,
	), createOut, createErr)
	if createCode != 0 {
		t.Fatalf("automation create comm trigger failed: code=%d stderr=%s output=%s", createCode, createErr.String(), createOut.String())
	}

	var createResponse map[string]any
	if err := json.Unmarshal(createOut.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if got := createResponse["trigger_type"]; got != "ON_COMM_EVENT" {
		t.Fatalf("expected ON_COMM_EVENT trigger type, got %v", got)
	}

	runArgs := withDaemonArgs(server,
		"automation", "run", "comm-event",
		"--workspace", "ws1",
		"--event-id", "event-cli-001",
		"--thread-id", "thread-cli-001",
		"--channel", "imessage",
		"--body", "Please follow up on this",
		"--sender", "sender@example.com",
		"--event-type", "MESSAGE",
		"--direction", "INBOUND",
		"--assistant-emitted=false",
	)

	run1Out := &bytes.Buffer{}
	run1Err := &bytes.Buffer{}
	run1Code := run(runArgs, run1Out, run1Err)
	if run1Code != 0 {
		t.Fatalf("automation run comm-event first failed: code=%d stderr=%s output=%s", run1Code, run1Err.String(), run1Out.String())
	}

	var run1Response map[string]any
	if err := json.Unmarshal(run1Out.Bytes(), &run1Response); err != nil {
		t.Fatalf("decode first comm-event run response: %v", err)
	}
	run1Result, ok := run1Response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected first comm-event run result payload")
	}
	if got := run1Result["Created"]; got != float64(1) {
		t.Fatalf("expected first comm-event run Created=1, got %v", got)
	}

	run2Out := &bytes.Buffer{}
	run2Err := &bytes.Buffer{}
	run2Code := run(runArgs, run2Out, run2Err)
	if run2Code != 0 {
		t.Fatalf("automation run comm-event replay failed: code=%d stderr=%s output=%s", run2Code, run2Err.String(), run2Out.String())
	}

	var run2Response map[string]any
	if err := json.Unmarshal(run2Out.Bytes(), &run2Response); err != nil {
		t.Fatalf("decode second comm-event run response: %v", err)
	}
	run2Result, ok := run2Response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected second comm-event run result payload")
	}
	if got := run2Result["Created"]; got != float64(0) {
		t.Fatalf("expected second comm-event run Created=0 due to idempotency, got %v", got)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var taskCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE workspace_id = 'ws1'`).Scan(&taskCount); err != nil {
		t.Fatalf("query task count: %v", err)
	}
	if taskCount != 1 {
		t.Fatalf("expected exactly 1 task from replay-safe comm-event run, got %d", taskCount)
	}

	var fireCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM trigger_fires WHERE workspace_id = 'ws1'`).Scan(&fireCount); err != nil {
		t.Fatalf("query trigger fire count: %v", err)
	}
	if fireCount != 1 {
		t.Fatalf("expected exactly 1 trigger fire from replay-safe comm-event run, got %d", fireCount)
	}
}

func TestRunCommSendFallbackAndIdempotentReplay(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comm-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	operationID := "op-fallback-001"

	sendArgs := withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--destination", "+15555550001",
		"--message", "status update",
		"--imessage-failures", "2",
	)

	sendOut := &bytes.Buffer{}
	sendErr := &bytes.Buffer{}
	sendCode := run(sendArgs, sendOut, sendErr)
	if sendCode != 0 {
		t.Fatalf("comm send fallback failed: code=%d stderr=%s output=%s", sendCode, sendErr.String(), sendOut.String())
	}

	var sendResponse map[string]any
	if err := json.Unmarshal(sendOut.Bytes(), &sendResponse); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	if success, ok := sendResponse["success"].(bool); !ok || !success {
		t.Fatalf("expected send success=true, got %v", sendResponse["success"])
	}
	sendResult, ok := sendResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected send result payload")
	}
	if got := sendResult["Channel"]; got != "twilio" {
		t.Fatalf("expected fallback delivery channel twilio, got %v", got)
	}
	sendAttempts, ok := sendResult["Attempts"].([]any)
	if !ok || len(sendAttempts) != 3 {
		t.Fatalf("expected 3 attempts (imessage retry + twilio), got %v", sendResult["Attempts"])
	}

	attemptsOut := &bytes.Buffer{}
	attemptsErr := &bytes.Buffer{}
	attemptsCode := run(withDaemonArgs(server,
		"comm", "attempts",
		"--workspace", "ws1",
		"--operation-id", operationID,
	), attemptsOut, attemptsErr)
	if attemptsCode != 0 {
		t.Fatalf("comm attempts failed: code=%d stderr=%s output=%s", attemptsCode, attemptsErr.String(), attemptsOut.String())
	}

	var attemptsResponse map[string]any
	if err := json.Unmarshal(attemptsOut.Bytes(), &attemptsResponse); err != nil {
		t.Fatalf("decode attempts response: %v", err)
	}
	attempts, ok := attemptsResponse["attempts"].([]any)
	if !ok || len(attempts) != 3 {
		t.Fatalf("expected 3 logged attempts, got %v", attemptsResponse["attempts"])
	}

	expected := []struct {
		channel    string
		status     string
		routeIndex float64
	}{
		{channel: "imessage", status: "failed", routeIndex: 0},
		{channel: "imessage", status: "failed", routeIndex: 1},
		{channel: "twilio", status: "sent", routeIndex: 2},
	}
	for index, item := range attempts {
		attempt, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("attempt %d has unexpected shape: %T", index, item)
		}
		if got := attempt["channel"]; got != expected[index].channel {
			t.Fatalf("attempt %d channel expected %s got %v", index, expected[index].channel, got)
		}
		if got := attempt["status"]; got != expected[index].status {
			t.Fatalf("attempt %d status expected %s got %v", index, expected[index].status, got)
		}
		if got := attempt["route_index"]; got != expected[index].routeIndex {
			t.Fatalf("attempt %d route_index expected %v got %v", index, expected[index].routeIndex, got)
		}
	}

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	replayCode := run(withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--destination", "+15555550001",
		"--message", "status update",
	), replayOut, replayErr)
	if replayCode != 0 {
		t.Fatalf("comm replay send failed: code=%d stderr=%s output=%s", replayCode, replayErr.String(), replayOut.String())
	}

	var replayResponse map[string]any
	if err := json.Unmarshal(replayOut.Bytes(), &replayResponse); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	replayResult, ok := replayResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected replay result payload")
	}
	if replay, ok := replayResult["IdempotentReplay"].(bool); !ok || !replay {
		t.Fatalf("expected idempotent replay=true, got %v", replayResult["IdempotentReplay"])
	}

	replayAttemptsOut := &bytes.Buffer{}
	replayAttemptsErr := &bytes.Buffer{}
	replayAttemptsCode := run(withDaemonArgs(server,
		"comm", "attempts",
		"--workspace", "ws1",
		"--operation-id", operationID,
	), replayAttemptsOut, replayAttemptsErr)
	if replayAttemptsCode != 0 {
		t.Fatalf("comm attempts replay check failed: code=%d stderr=%s output=%s", replayAttemptsCode, replayAttemptsErr.String(), replayAttemptsOut.String())
	}

	var replayAttemptsResponse map[string]any
	if err := json.Unmarshal(replayAttemptsOut.Bytes(), &replayAttemptsResponse); err != nil {
		t.Fatalf("decode replay attempts response: %v", err)
	}
	replayAttempts, ok := replayAttemptsResponse["attempts"].([]any)
	if !ok || len(replayAttempts) != 3 {
		t.Fatalf("expected replay to preserve exactly 3 logged attempts, got %v", replayAttemptsResponse["attempts"])
	}
}

func TestRunCommSendUsesSourceChannelForSMS(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comm-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	operationID := "op-sms-routing-001"

	sendOut := &bytes.Buffer{}
	sendErr := &bytes.Buffer{}
	sendCode := run(withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--connector-id", "twilio",
		"--destination", "+15555550002",
		"--message", "hello",
	), sendOut, sendErr)
	if sendCode != 0 {
		t.Fatalf("comm send sms route failed: code=%d stderr=%s output=%s", sendCode, sendErr.String(), sendOut.String())
	}

	var sendResponse map[string]any
	if err := json.Unmarshal(sendOut.Bytes(), &sendResponse); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	sendResult, ok := sendResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected send result payload")
	}
	if got := sendResult["Channel"]; got != "twilio" {
		t.Fatalf("expected sms source to route via twilio, got %v", got)
	}

	attemptsOut := &bytes.Buffer{}
	attemptsErr := &bytes.Buffer{}
	attemptsCode := run(withDaemonArgs(server,
		"comm", "attempts",
		"--workspace", "ws1",
		"--operation-id", operationID,
	), attemptsOut, attemptsErr)
	if attemptsCode != 0 {
		t.Fatalf("comm attempts failed: code=%d stderr=%s output=%s", attemptsCode, attemptsErr.String(), attemptsOut.String())
	}

	var attemptsResponse map[string]any
	if err := json.Unmarshal(attemptsOut.Bytes(), &attemptsResponse); err != nil {
		t.Fatalf("decode attempts response: %v", err)
	}
	attempts, ok := attemptsResponse["attempts"].([]any)
	if !ok || len(attempts) != 1 {
		t.Fatalf("expected one twilio attempt, got %v", attemptsResponse["attempts"])
	}
	attempt, ok := attempts[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected attempt shape: %T", attempts[0])
	}
	if got := attempt["channel"]; got != "twilio" {
		t.Fatalf("expected attempt channel twilio, got %v", got)
	}
	if got := attempt["route_index"]; got != float64(0) {
		t.Fatalf("expected route_index 0, got %v", got)
	}
}

func TestRunCommSendUsesTwilioWhenConfiguredAndPreservesThread(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	requestCount := 0
	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			t.Fatalf("expected Twilio messages path, got %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "twilio-auth-token" {
			t.Fatalf("unexpected Twilio basic auth credentials: ok=%v user=%s pass=%s", ok, user, pass)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse twilio form: %v", err)
		}
		if got := r.Form.Get("From"); got != "+15555550001" {
			t.Fatalf("expected from number +15555550001, got %s", got)
		}
		if got := r.Form.Get("To"); got != "+15555550999" {
			t.Fatalf("expected destination +15555550999, got %s", got)
		}
		if got := r.Form.Get("Body"); got != "reply from agent" {
			t.Fatalf("expected body, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"SMOUTBOUND1","account_sid":"AC123","status":"queued","from":"+15555550001","to":"+15555550999"}`))
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "comm-runtime-twilio.db")
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

	ingestOut := &bytes.Buffer{}
	ingestErr := &bytes.Buffer{}
	ingestCode := run(withDaemonArgs(server,
		"connector", "twilio", "ingest-sms",
		"--workspace", "ws1",
		"--skip-signature=true",
		"--from", "+15555550999",
		"--to", "+15555550001",
		"--body", "hello inbound",
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
	inboundThreadID, _ := ingestResponse["thread_id"].(string)
	if inboundThreadID == "" {
		t.Fatalf("expected inbound thread id")
	}

	operationID := "op-twilio-sms-001"
	sendOut := &bytes.Buffer{}
	sendErr := &bytes.Buffer{}
	sendCode := run(withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--connector-id", "twilio",
		"--destination", "+15555550999",
		"--message", "reply from agent",
	), sendOut, sendErr)
	if sendCode != 0 {
		t.Fatalf("comm send twilio failed: code=%d stderr=%s output=%s", sendCode, sendErr.String(), sendOut.String())
	}
	var sendResponse map[string]any
	if err := json.Unmarshal(sendOut.Bytes(), &sendResponse); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	sendResult, ok := sendResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected send result payload")
	}
	if got := sendResult["Channel"]; got != "twilio" {
		t.Fatalf("expected send channel twilio, got %v", got)
	}
	if got := sendResult["ProviderReceipt"]; got != "SMOUTBOUND1" {
		t.Fatalf("expected provider receipt SMOUTBOUND1, got %v", got)
	}

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	replayCode := run(withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--connector-id", "twilio",
		"--destination", "+15555550999",
		"--message", "reply from agent",
	), replayOut, replayErr)
	if replayCode != 0 {
		t.Fatalf("comm send twilio replay failed: code=%d stderr=%s output=%s", replayCode, replayErr.String(), replayOut.String())
	}
	var replayResponse map[string]any
	if err := json.Unmarshal(replayOut.Bytes(), &replayResponse); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	replayResult, ok := replayResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected replay result payload")
	}
	if replay, ok := replayResult["IdempotentReplay"].(bool); !ok || !replay {
		t.Fatalf("expected idempotent replay=true, got %v", replayResult["IdempotentReplay"])
	}

	if requestCount != 1 {
		t.Fatalf("expected one outbound Twilio request due replay safety, got %d", requestCount)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var attemptCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_attempts WHERE idempotency_key LIKE ?`, operationID+"|%").Scan(&attemptCount); err != nil {
		t.Fatalf("count delivery attempts: %v", err)
	}
	if attemptCount != 1 {
		t.Fatalf("expected one delivery attempt row, got %d", attemptCount)
	}

	var outboundThreadID string
	if err := db.QueryRow(`
		SELECT ce.thread_id
		FROM comm_provider_messages cpm
		JOIN comm_events ce ON ce.id = cpm.event_id
		WHERE cpm.workspace_id = 'ws1'
		  AND cpm.provider = 'twilio'
		  AND cpm.provider_message_id = 'SMOUTBOUND1'
		LIMIT 1
	`).Scan(&outboundThreadID); err != nil {
		t.Fatalf("query outbound thread id: %v", err)
	}
	if outboundThreadID != inboundThreadID {
		t.Fatalf("expected outbound thread %s to match inbound thread %s", outboundThreadID, inboundThreadID)
	}
}

func TestRunCommPolicyOverrideRoutesImessageToSMS(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comm-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	policySetOut := &bytes.Buffer{}
	policySetErr := &bytes.Buffer{}
	policySetCode := run(withDaemonArgs(server,
		"comm", "policy", "set",
		"--workspace", "ws1",
		"--source-channel", "message",
		"--primary-channel", "twilio",
		"--retry-count", "0",
		"--fallback-channels", "",
	), policySetOut, policySetErr)
	if policySetCode != 0 {
		t.Fatalf("comm policy set failed: code=%d stderr=%s output=%s", policySetCode, policySetErr.String(), policySetOut.String())
	}

	policyListOut := &bytes.Buffer{}
	policyListErr := &bytes.Buffer{}
	policyListCode := run(withDaemonArgs(server,
		"comm", "policy", "list",
		"--workspace", "ws1",
		"--source-channel", "message",
	), policyListOut, policyListErr)
	if policyListCode != 0 {
		t.Fatalf("comm policy list failed: code=%d stderr=%s output=%s", policyListCode, policyListErr.String(), policyListOut.String())
	}

	var listResponse map[string]any
	if err := json.Unmarshal(policyListOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode policy list response: %v", err)
	}
	policies, ok := listResponse["policies"].([]any)
	if !ok || len(policies) < 1 {
		t.Fatalf("expected at least one policy, got %v", listResponse["policies"])
	}
	first, ok := policies[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected policy shape: %T", policies[0])
	}
	policy, ok := first["policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy payload map")
	}
	if got := policy["primary_channel"]; got != "twilio" {
		t.Fatalf("expected primary channel twilio, got %v", got)
	}

	operationID := "op-policy-override-001"
	sendOut := &bytes.Buffer{}
	sendErr := &bytes.Buffer{}
	sendCode := run(withDaemonArgs(server,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--destination", "+15555550003",
		"--message", "policy route",
		"--imessage-failures", "5",
	), sendOut, sendErr)
	if sendCode != 0 {
		t.Fatalf("comm send with policy override failed: code=%d stderr=%s output=%s", sendCode, sendErr.String(), sendOut.String())
	}

	var sendResponse map[string]any
	if err := json.Unmarshal(sendOut.Bytes(), &sendResponse); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	sendResult, ok := sendResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected send result payload")
	}
	if got := sendResult["Channel"]; got != "twilio" {
		t.Fatalf("expected policy-overridden channel twilio, got %v", got)
	}

	attemptsOut := &bytes.Buffer{}
	attemptsErr := &bytes.Buffer{}
	attemptsCode := run(withDaemonArgs(server,
		"comm", "attempts",
		"--workspace", "ws1",
		"--operation-id", operationID,
	), attemptsOut, attemptsErr)
	if attemptsCode != 0 {
		t.Fatalf("comm attempts failed: code=%d stderr=%s output=%s", attemptsCode, attemptsErr.String(), attemptsOut.String())
	}

	var attemptsResponse map[string]any
	if err := json.Unmarshal(attemptsOut.Bytes(), &attemptsResponse); err != nil {
		t.Fatalf("decode attempts response: %v", err)
	}
	attempts, ok := attemptsResponse["attempts"].([]any)
	if !ok || len(attempts) != 1 {
		t.Fatalf("expected single attempt with twilio override, got %v", attemptsResponse["attempts"])
	}
	attempt, ok := attempts[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected attempt shape: %T", attempts[0])
	}
	if got := attempt["channel"]; got != "twilio" {
		t.Fatalf("expected twilio attempt channel, got %v", got)
	}
	if got := attempt["route_index"]; got != float64(0) {
		t.Fatalf("expected route_index 0, got %v", got)
	}
}

func TestRunInspectCommandsExposeRunTranscriptAndMemoryData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "inspect-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	runOut := &bytes.Buffer{}
	runErr := &bytes.Buffer{}
	runCode := run(withDaemonArgs(server,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "send an email update to sam@example.com",
	), runOut, runErr)
	if runCode != 0 {
		t.Fatalf("seed agent run failed: code=%d stderr=%s output=%s", runCode, runErr.String(), runOut.String())
	}

	var runResponse map[string]any
	if err := json.Unmarshal(runOut.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode seed run response: %v", err)
	}
	runID, ok := runResponse["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id in seed response")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var stepID string
	if err := db.QueryRow(`SELECT id FROM task_steps WHERE run_id = ? ORDER BY step_index ASC LIMIT 1`, runID).Scan(&stepID); err != nil {
		t.Fatalf("query step id: %v", err)
	}
	now := time.Date(2026, 2, 24, 3, 15, 0, 0, time.UTC).Format(time.RFC3339Nano)

	if _, err := db.Exec(
		`INSERT INTO run_artifacts(id, run_id, step_id, artifact_type, uri, content_hash, created_at)
		 VALUES ('artifact_inspect_1', ?, ?, 'connector_output', 'file:///tmp/output.txt', 'hash-1', ?)`,
		runID,
		stepID,
		now,
	); err != nil {
		t.Fatalf("insert run artifact fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at)
		 VALUES ('audit_inspect_1', 'ws1', ?, ?, 'STEP_COMPLETED', 'actor.requester', 'actor.requester', 'corr-1', '{"ok":true}', ?)`,
		runID,
		stepID,
		now,
	); err != nil {
		t.Fatalf("insert audit fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread_inspect_1', 'ws1', 'imessage', 'thread-ext-1', 'Inspect Thread', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert transcript thread fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event_inspect_1', 'ws1', 'thread_inspect_1', 'MESSAGE', 'INBOUND', 0, ?, 'Need status update', ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert transcript event fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr_inspect_1', 'event_inspect_1', 'FROM', 'sender@example.com', 'Sender', 0, ?)`,
		now,
	); err != nil {
		t.Fatalf("insert transcript address fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at)
		 VALUES ('mem_inspect_1', 'ws1', 'actor.requester', 'conversation', 'k1', '{"kind":"conversation","token_estimate":180,"content":"remember this"}', 'ACTIVE', 'event_inspect_1', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert memory fixture: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at)
		 VALUES ('ms_inspect_1', 'mem_inspect_1', 'comm', 'event_inspect_1', ?)`,
		now,
	); err != nil {
		t.Fatalf("insert memory source fixture: %v", err)
	}

	inspectRunOut := &bytes.Buffer{}
	inspectRunErr := &bytes.Buffer{}
	inspectRunCode := run(withDaemonArgs(server,
		"inspect", "run",
		"--run-id", runID,
	), inspectRunOut, inspectRunErr)
	if inspectRunCode != 0 {
		t.Fatalf("inspect run failed: code=%d stderr=%s output=%s", inspectRunCode, inspectRunErr.String(), inspectRunOut.String())
	}

	var inspectRunResponse map[string]any
	if err := json.Unmarshal(inspectRunOut.Bytes(), &inspectRunResponse); err != nil {
		t.Fatalf("decode inspect run response: %v", err)
	}
	if artifacts, ok := inspectRunResponse["artifacts"].([]any); !ok || len(artifacts) != 1 {
		t.Fatalf("expected one artifact in inspect run response, got %v", inspectRunResponse["artifacts"])
	}
	audits, ok := inspectRunResponse["audit_entries"].([]any)
	if !ok || len(audits) == 0 {
		t.Fatalf("expected audit entries in inspect run response, got %v", inspectRunResponse["audit_entries"])
	}
	foundInsertedAudit := false
	for _, entry := range audits {
		row, castOK := entry.(map[string]any)
		if !castOK {
			continue
		}
		if row["audit_id"] == "audit_inspect_1" {
			foundInsertedAudit = true
			break
		}
	}
	if !foundInsertedAudit {
		t.Fatalf("expected inserted audit fixture to be present, got %v", audits)
	}
	if steps, ok := inspectRunResponse["steps"].([]any); !ok || len(steps) == 0 {
		t.Fatalf("expected at least one step in inspect run response, got %v", inspectRunResponse["steps"])
	}

	inspectTranscriptOut := &bytes.Buffer{}
	inspectTranscriptErr := &bytes.Buffer{}
	inspectTranscriptCode := run(withDaemonArgs(server,
		"inspect", "transcript",
		"--workspace", "ws1",
		"--thread-id", "thread_inspect_1",
		"--limit", "5",
	), inspectTranscriptOut, inspectTranscriptErr)
	if inspectTranscriptCode != 0 {
		t.Fatalf("inspect transcript failed: code=%d stderr=%s output=%s", inspectTranscriptCode, inspectTranscriptErr.String(), inspectTranscriptOut.String())
	}

	var transcriptResponse map[string]any
	if err := json.Unmarshal(inspectTranscriptOut.Bytes(), &transcriptResponse); err != nil {
		t.Fatalf("decode transcript response: %v", err)
	}
	events, ok := transcriptResponse["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected one transcript event, got %v", transcriptResponse["events"])
	}

	inspectMemoryOut := &bytes.Buffer{}
	inspectMemoryErr := &bytes.Buffer{}
	inspectMemoryCode := run(withDaemonArgs(server,
		"inspect", "memory",
		"--workspace", "ws1",
		"--owner", "actor.requester",
		"--status", "ACTIVE",
		"--limit", "5",
	), inspectMemoryOut, inspectMemoryErr)
	if inspectMemoryCode != 0 {
		t.Fatalf("inspect memory failed: code=%d stderr=%s output=%s", inspectMemoryCode, inspectMemoryErr.String(), inspectMemoryOut.String())
	}

	var memoryResponse map[string]any
	if err := json.Unmarshal(inspectMemoryOut.Bytes(), &memoryResponse); err != nil {
		t.Fatalf("decode memory response: %v", err)
	}
	items, ok := memoryResponse["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one memory item, got %v", memoryResponse["items"])
	}
}

func TestRunRetentionPurgeAndCompactMemoryCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "retention-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	now := time.Now().UTC()
	old := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339Nano)
	recent := now.Add(-2 * 24 * time.Hour).Format(time.RFC3339Nano)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := openRuntimeDB(ctx, dbPath); err != nil {
		t.Fatalf("initialize runtime db: %v", err)
	}

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_ret', 'Retention WS', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_ret', 'ws_ret', 'human', 'Retention Actor', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_ret', 'ws_ret', 'actor_ret', 'ACTIVE', '` + recent + `', '` + recent + `')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at) VALUES ('thread_ret', 'ws_ret', 'imessage', 'ext_ret', 'Retention Thread', '` + recent + `', '` + recent + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_old_ret', 'ws_ret', 'thread_ret', 'MESSAGE', 'INBOUND', 0, '` + old + `', 'old message', '` + old + `')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at) VALUES ('event_new_ret', 'ws_ret', 'thread_ret', 'MESSAGE', 'INBOUND', 0, '` + recent + `', 'new message', '` + recent + `')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr_old_ret', 'event_old_ret', 'FROM', 'old@example.com', 'Old', 0, '` + old + `')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at) VALUES ('addr_new_ret', 'event_new_ret', 'FROM', 'new@example.com', 'New', 0, '` + recent + `')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit_old_ret', 'ws_ret', NULL, NULL, 'TRACE', 'actor_ret', 'actor_ret', 'corr1', '{}', '` + old + `')`,
		`INSERT INTO audit_log_entries(id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at) VALUES ('audit_new_ret', 'ws_ret', NULL, NULL, 'TRACE', 'actor_ret', 'actor_ret', 'corr2', '{}', '` + recent + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_old_ret', 'ws_ret', 'actor_ret', 'conversation', 'old', '{"kind":"conversation","token_estimate":500}', 'ACTIVE', 'old', '` + old + `', '` + old + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_new_ret', 'ws_ret', 'actor_ret', 'conversation', 'new', '{"kind":"conversation","token_estimate":150}', 'ACTIVE', 'new', '` + recent + `', '` + recent + `')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed retention fixture: %v", err)
		}
	}

	purgeOut := &bytes.Buffer{}
	purgeErr := &bytes.Buffer{}
	purgeCode := run(withDaemonArgs(server,
		"retention", "purge",
	), purgeOut, purgeErr)
	if purgeCode != 0 {
		t.Fatalf("retention purge failed: code=%d stderr=%s output=%s", purgeCode, purgeErr.String(), purgeOut.String())
	}

	var purgeResponse map[string]any
	if err := json.Unmarshal(purgeOut.Bytes(), &purgeResponse); err != nil {
		t.Fatalf("decode purge response: %v", err)
	}
	result, ok := purgeResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected purge result payload")
	}
	if result["TracesDeleted"] == float64(0) || result["TranscriptsDeleted"] == float64(0) || result["MemoryDeleted"] == float64(0) {
		t.Fatalf("expected deletions across retention categories, got %v", result)
	}
	if consistency, ok := result["ConsistencyMode"].(string); !ok || consistency != "partial_success" {
		t.Fatalf("expected ConsistencyMode=partial_success, got %v", result["ConsistencyMode"])
	}
	if status, ok := result["Status"].(string); !ok || status != "completed" {
		t.Fatalf("expected Status=completed, got %v", result["Status"])
	}
	if failure, exists := result["Failure"]; exists && failure != nil {
		t.Fatalf("expected no failure metadata for successful purge, got %v", failure)
	}

	var remainingOldTrace int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE id = 'audit_old_ret'`).Scan(&remainingOldTrace); err != nil {
		t.Fatalf("query old trace: %v", err)
	}
	if remainingOldTrace != 0 {
		t.Fatalf("expected old trace row to be purged")
	}
	var remainingNewTrace int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE id = 'audit_new_ret'`).Scan(&remainingNewTrace); err != nil {
		t.Fatalf("query new trace: %v", err)
	}
	if remainingNewTrace != 1 {
		t.Fatalf("expected recent trace row to remain")
	}

	compactSeed := []string{
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_compact_fact', 'ws_ret', 'actor_ret', 'profile', 'fact', '{"kind":"fact","is_canonical":true,"token_estimate":300}', 'ACTIVE', '', '` + recent + `', '` + recent + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_compact_old1', 'ws_ret', 'actor_ret', 'conversation', 'old1', '{"kind":"conversation","token_estimate":700}', 'ACTIVE', 'event_old', '` + old + `', '` + old + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_compact_old2', 'ws_ret', 'actor_ret', 'conversation', 'old2', '{"kind":"conversation","token_estimate":600}', 'ACTIVE', 'event_old', '` + old + `', '` + old + `')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('mem_compact_new', 'ws_ret', 'actor_ret', 'conversation', 'new1', '{"kind":"conversation","token_estimate":200}', 'ACTIVE', 'event_new', '` + recent + `', '` + recent + `')`,
	}
	for _, stmt := range compactSeed {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed compaction fixture: %v", err)
		}
	}

	previewOut := &bytes.Buffer{}
	previewErr := &bytes.Buffer{}
	previewCode := run(withDaemonArgs(server,
		"retention", "compact-memory",
		"--workspace", "ws_ret",
		"--owner", "actor_ret",
		"--token-threshold", "1000",
		"--stale-after-hours", "168",
	), previewOut, previewErr)
	if previewCode != 0 {
		t.Fatalf("retention compact-memory preview failed: code=%d stderr=%s output=%s", previewCode, previewErr.String(), previewOut.String())
	}

	var previewResponse map[string]any
	if err := json.Unmarshal(previewOut.Bytes(), &previewResponse); err != nil {
		t.Fatalf("decode compaction preview response: %v", err)
	}
	if applied, ok := previewResponse["applied"].(bool); !ok || applied {
		t.Fatalf("expected preview applied=false, got %v", previewResponse["applied"])
	}
	previewResult, ok := previewResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected compaction result payload")
	}
	droppedIDs, ok := previewResult["DroppedIDs"].([]any)
	if !ok || len(droppedIDs) < 2 {
		t.Fatalf("expected stale dropped ids in preview, got %v", previewResult["DroppedIDs"])
	}

	applyOut := &bytes.Buffer{}
	applyErr := &bytes.Buffer{}
	applyCode := run(withDaemonArgs(server,
		"retention", "compact-memory",
		"--workspace", "ws_ret",
		"--owner", "actor_ret",
		"--token-threshold", "1000",
		"--stale-after-hours", "168",
		"--apply=true",
	), applyOut, applyErr)
	if applyCode != 0 {
		t.Fatalf("retention compact-memory apply failed: code=%d stderr=%s output=%s", applyCode, applyErr.String(), applyOut.String())
	}

	var applyResponse map[string]any
	if err := json.Unmarshal(applyOut.Bytes(), &applyResponse); err != nil {
		t.Fatalf("decode compaction apply response: %v", err)
	}
	if applied, ok := applyResponse["applied"].(bool); !ok || !applied {
		t.Fatalf("expected apply=true, got %v", applyResponse["applied"])
	}
	createdSummaries, ok := applyResponse["created_summary_ids"].([]any)
	if !ok || len(createdSummaries) == 0 {
		t.Fatalf("expected created summary ids when apply=true, got %v", applyResponse["created_summary_ids"])
	}

	var droppedStatusCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_items WHERE id IN ('mem_compact_old1','mem_compact_old2') AND status = 'DISABLED'`).Scan(&droppedStatusCount); err != nil {
		t.Fatalf("query dropped status rows: %v", err)
	}
	if droppedStatusCount != 2 {
		t.Fatalf("expected dropped records to be disabled, got %d", droppedStatusCount)
	}
}

func TestRunContextSamplesAndTuneCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "context-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := openRuntimeDB(ctx, dbPath); err != nil {
		t.Fatalf("initialize runtime db: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 2, 24, 5, 0, 0, 0, time.UTC)
	nowText := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_ctx_cli', 'Ctx', 'ACTIVE', ?, ?)`, nowText, nowText); err != nil {
		t.Fatalf("seed context workspace: %v", err)
	}
	for i := 0; i < 5; i++ {
		createdAt := now.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano)
		if _, err := db.Exec(
			`INSERT INTO context_budget_samples(
				id, workspace_id, task_class, model_key, context_window, output_limit, deep_analysis,
				remaining_budget, retrieval_target, retrieval_used, prompt_tokens, completion_tokens, created_at
			) VALUES (?, 'ws_ctx_cli', 'chat', 'gpt-5-mini', 10000, 1000, 0, 9000, 1000, 900, 600, 100, ?)`,
			fmt.Sprintf("sample_ctx_%d", i),
			createdAt,
		); err != nil {
			t.Fatalf("seed context sample %d: %v", i, err)
		}
	}

	samplesOut := &bytes.Buffer{}
	samplesErr := &bytes.Buffer{}
	samplesCode := run(withDaemonArgs(server,
		"context", "samples",
		"--workspace", "ws_ctx_cli",
		"--task-class", "chat",
		"--limit", "10",
	), samplesOut, samplesErr)
	if samplesCode != 0 {
		t.Fatalf("context samples failed: code=%d stderr=%s output=%s", samplesCode, samplesErr.String(), samplesOut.String())
	}

	var samplesResponse map[string]any
	if err := json.Unmarshal(samplesOut.Bytes(), &samplesResponse); err != nil {
		t.Fatalf("decode context samples response: %v", err)
	}
	samples, ok := samplesResponse["samples"].([]any)
	if !ok || len(samples) != 5 {
		t.Fatalf("expected 5 context samples, got %v", samplesResponse["samples"])
	}

	tuneOut := &bytes.Buffer{}
	tuneErr := &bytes.Buffer{}
	tuneCode := run(withDaemonArgs(server,
		"context", "tune",
		"--workspace", "ws_ctx_cli",
		"--task-class", "chat",
	), tuneOut, tuneErr)
	if tuneCode != 0 {
		t.Fatalf("context tune failed: code=%d stderr=%s output=%s", tuneCode, tuneErr.String(), tuneOut.String())
	}

	var tuneResponse map[string]any
	if err := json.Unmarshal(tuneOut.Bytes(), &tuneResponse); err != nil {
		t.Fatalf("decode context tune response: %v", err)
	}
	if changed, ok := tuneResponse["Changed"].(bool); !ok || !changed {
		t.Fatalf("expected tuning decision to change multiplier, got %v", tuneResponse["Changed"])
	}
	if reason, ok := tuneResponse["Reason"].(string); !ok || reason == "" {
		t.Fatalf("expected tuning reason, got %v", tuneResponse["Reason"])
	}
}

func TestRunIdentityCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := openRuntimeDB(ctx, dbPath); err != nil {
		t.Fatalf("initialize runtime db: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 2, 26, 3, 0, 0, 0, time.UTC)
	earlier := now.Format(time.RFC3339Nano)
	later := now.Add(1 * time.Minute).Format(time.RFC3339Nano)
	latest := now.Add(2 * time.Minute).Format(time.RFC3339Nano)

	seedStatements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_identity_alpha', 'Identity Alpha', 'ACTIVE', '` + earlier + `', '` + earlier + `')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_identity_beta', 'Identity Beta', 'ACTIVE', '` + later + `', '` + later + `')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_identity_archived', 'Identity Archived', 'INACTIVE', '` + latest + `', '` + latest + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.identity.alpha', 'ws_identity_alpha', 'HUMAN', 'Alpha User', 'ACTIVE', '` + earlier + `', '` + earlier + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_identity_alpha', 'ws_identity_alpha', 'actor.identity.alpha', 'ACTIVE', '` + earlier + `', '` + earlier + `')`,
		`INSERT INTO actor_handles(id, workspace_id, actor_id, channel, handle_value, is_primary, created_at, updated_at) VALUES ('ah_identity_alpha', 'ws_identity_alpha', 'actor.identity.alpha', 'app', 'alpha@app', 1, '` + earlier + `', '` + earlier + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.identity.beta', 'ws_identity_beta', 'HUMAN', 'Beta User', 'ACTIVE', '` + later + `', '` + later + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_identity_beta', 'ws_identity_beta', 'actor.identity.beta', 'ACTIVE', '` + later + `', '` + later + `')`,
	}
	for _, statement := range seedStatements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed identity fixtures: %v\nstatement: %s", err, statement)
		}
	}

	workspacesOut := &bytes.Buffer{}
	workspacesErr := &bytes.Buffer{}
	workspacesCode := run(withDaemonArgs(server,
		"identity", "workspaces",
		"--include-inactive=true",
	), workspacesOut, workspacesErr)
	if workspacesCode != 0 {
		t.Fatalf("identity workspaces failed: code=%d stderr=%s output=%s", workspacesCode, workspacesErr.String(), workspacesOut.String())
	}

	var workspacesResponse transport.IdentityWorkspacesResponse
	if err := json.Unmarshal(workspacesOut.Bytes(), &workspacesResponse); err != nil {
		t.Fatalf("decode identity workspaces response: %v", err)
	}
	if len(workspacesResponse.Workspaces) < 3 {
		t.Fatalf("expected at least 3 workspaces in identity listing, got %d", len(workspacesResponse.Workspaces))
	}

	foundInactive := false
	for _, record := range workspacesResponse.Workspaces {
		if strings.EqualFold(strings.TrimSpace(record.Status), "INACTIVE") {
			foundInactive = true
			break
		}
	}
	if !foundInactive {
		t.Fatalf("expected inactive workspace in include-inactive listing: %+v", workspacesResponse.Workspaces)
	}

	contextOut := &bytes.Buffer{}
	contextErr := &bytes.Buffer{}
	contextCode := run(withDaemonArgs(server,
		"identity", "context",
	), contextOut, contextErr)
	if contextCode != 0 {
		t.Fatalf("identity context failed: code=%d stderr=%s output=%s", contextCode, contextErr.String(), contextOut.String())
	}

	var contextResponse transport.IdentityActiveContextResponse
	if err := json.Unmarshal(contextOut.Bytes(), &contextResponse); err != nil {
		t.Fatalf("decode identity context response: %v", err)
	}
	if strings.TrimSpace(contextResponse.ActiveContext.WorkspaceID) == "" {
		t.Fatalf("expected active context workspace id, got %+v", contextResponse.ActiveContext)
	}

	principalsOut := &bytes.Buffer{}
	principalsErr := &bytes.Buffer{}
	principalsCode := run(withDaemonArgs(server,
		"identity", "principals",
		"--workspace", "ws_identity_alpha",
	), principalsOut, principalsErr)
	if principalsCode != 0 {
		t.Fatalf("identity principals failed: code=%d stderr=%s output=%s", principalsCode, principalsErr.String(), principalsOut.String())
	}

	var principalsResponse transport.IdentityPrincipalsResponse
	if err := json.Unmarshal(principalsOut.Bytes(), &principalsResponse); err != nil {
		t.Fatalf("decode identity principals response: %v", err)
	}
	if principalsResponse.WorkspaceID != "ws_identity_alpha" {
		t.Fatalf("expected principals workspace ws_identity_alpha, got %q", principalsResponse.WorkspaceID)
	}
	if len(principalsResponse.Principals) != 1 || principalsResponse.Principals[0].ActorID != "actor.identity.alpha" {
		t.Fatalf("unexpected principals response: %+v", principalsResponse.Principals)
	}
	if len(principalsResponse.Principals[0].Handles) != 1 || principalsResponse.Principals[0].Handles[0].HandleValue != "alpha@app" {
		t.Fatalf("expected actor handle payload in principals response: %+v", principalsResponse.Principals[0].Handles)
	}

	selectOut := &bytes.Buffer{}
	selectErr := &bytes.Buffer{}
	selectCode := run(withDaemonArgs(server,
		"identity", "select-workspace",
		"--workspace", "ws_identity_beta",
		"--principal", "actor.identity.beta",
	), selectOut, selectErr)
	if selectCode != 0 {
		t.Fatalf("identity select-workspace failed: code=%d stderr=%s output=%s", selectCode, selectErr.String(), selectOut.String())
	}

	var selectResponse transport.IdentityActiveContextResponse
	if err := json.Unmarshal(selectOut.Bytes(), &selectResponse); err != nil {
		t.Fatalf("decode identity select-workspace response: %v", err)
	}
	if selectResponse.ActiveContext.WorkspaceID != "ws_identity_beta" || selectResponse.ActiveContext.PrincipalActorID != "actor.identity.beta" {
		t.Fatalf("unexpected selected identity context: %+v", selectResponse.ActiveContext)
	}
	if selectResponse.ActiveContext.SelectionVersion <= 0 || strings.TrimSpace(selectResponse.ActiveContext.MutationSource) == "" || strings.TrimSpace(selectResponse.ActiveContext.MutationReason) == "" {
		t.Fatalf("expected selection metadata in selected identity context: %+v", selectResponse.ActiveContext)
	}

	contextAfterSelectOut := &bytes.Buffer{}
	contextAfterSelectErr := &bytes.Buffer{}
	contextAfterSelectCode := run(withDaemonArgs(server,
		"identity", "context",
	), contextAfterSelectOut, contextAfterSelectErr)
	if contextAfterSelectCode != 0 {
		t.Fatalf("identity context after select failed: code=%d stderr=%s output=%s", contextAfterSelectCode, contextAfterSelectErr.String(), contextAfterSelectOut.String())
	}

	var contextAfterSelect transport.IdentityActiveContextResponse
	if err := json.Unmarshal(contextAfterSelectOut.Bytes(), &contextAfterSelect); err != nil {
		t.Fatalf("decode context-after-select response: %v", err)
	}
	if contextAfterSelect.ActiveContext.WorkspaceID != "ws_identity_beta" || contextAfterSelect.ActiveContext.PrincipalActorID != "actor.identity.beta" {
		t.Fatalf("expected selected context to persist, got %+v", contextAfterSelect.ActiveContext)
	}
	if contextAfterSelect.ActiveContext.SelectionVersion != selectResponse.ActiveContext.SelectionVersion {
		t.Fatalf("expected selection_version to persist across context reads: selected=%d context=%d", selectResponse.ActiveContext.SelectionVersion, contextAfterSelect.ActiveContext.SelectionVersion)
	}
}

func TestRunIdentityBootstrapCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity-bootstrap-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := openRuntimeDB(ctx, dbPath); err != nil {
		t.Fatalf("initialize runtime db: %v", err)
	}

	firstOut := &bytes.Buffer{}
	firstErr := &bytes.Buffer{}
	firstCode := run(withDaemonArgs(server,
		"identity", "bootstrap",
		"--workspace", "ws_cli_bootstrap",
		"--workspace-name", "CLI Bootstrap Workspace",
		"--principal", "actor.cli.bootstrap",
		"--display-name", "CLI Bootstrap User",
		"--actor-type", "human",
		"--principal-status", "ACTIVE",
		"--handle-channel", "message",
		"--handle-value", "+15551112222",
		"--handle-primary=true",
	), firstOut, firstErr)
	if firstCode != 0 {
		t.Fatalf("identity bootstrap first call failed: code=%d stderr=%s output=%s", firstCode, firstErr.String(), firstOut.String())
	}

	var firstResponse transport.IdentityBootstrapResponse
	if err := json.Unmarshal(firstOut.Bytes(), &firstResponse); err != nil {
		t.Fatalf("decode first identity bootstrap response: %v", err)
	}
	if !firstResponse.WorkspaceCreated || !firstResponse.PrincipalCreated || !firstResponse.PrincipalLinked || !firstResponse.HandleCreated {
		t.Fatalf("expected create flags on first bootstrap call, got %+v", firstResponse)
	}
	if firstResponse.Idempotent {
		t.Fatalf("expected first bootstrap call to be non-idempotent")
	}
	if firstResponse.ActiveContext.WorkspaceID != "ws_cli_bootstrap" || firstResponse.ActiveContext.PrincipalActorID != "actor.cli.bootstrap" {
		t.Fatalf("unexpected active context in bootstrap response: %+v", firstResponse.ActiveContext)
	}

	secondOut := &bytes.Buffer{}
	secondErr := &bytes.Buffer{}
	secondCode := run(withDaemonArgs(server,
		"identity", "bootstrap",
		"--workspace", "ws_cli_bootstrap",
		"--workspace-name", "CLI Bootstrap Workspace",
		"--principal", "actor.cli.bootstrap",
		"--display-name", "CLI Bootstrap User",
		"--actor-type", "human",
		"--principal-status", "ACTIVE",
		"--handle-channel", "message",
		"--handle-value", "+15551112222",
		"--handle-primary=true",
	), secondOut, secondErr)
	if secondCode != 0 {
		t.Fatalf("identity bootstrap second call failed: code=%d stderr=%s output=%s", secondCode, secondErr.String(), secondOut.String())
	}

	var secondResponse transport.IdentityBootstrapResponse
	if err := json.Unmarshal(secondOut.Bytes(), &secondResponse); err != nil {
		t.Fatalf("decode second identity bootstrap response: %v", err)
	}
	if !secondResponse.Idempotent {
		t.Fatalf("expected second bootstrap call to be idempotent, got %+v", secondResponse)
	}
	if secondResponse.WorkspaceCreated || secondResponse.PrincipalCreated || secondResponse.PrincipalLinked || secondResponse.HandleCreated || secondResponse.HandleUpdated {
		t.Fatalf("expected second bootstrap call to avoid duplicate inserts, got %+v", secondResponse)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	assertCount := func(query string, expected int) {
		t.Helper()
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil {
			t.Fatalf("query count failed: %v\nquery: %s", err, query)
		}
		if count != expected {
			t.Fatalf("unexpected count=%d expected=%d query=%s", count, expected, query)
		}
	}
	assertCount(`SELECT COUNT(*) FROM workspaces WHERE id = 'ws_cli_bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM actors WHERE id = 'actor.cli.bootstrap' AND workspace_id = 'ws_cli_bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM workspace_principals WHERE workspace_id = 'ws_cli_bootstrap' AND actor_id = 'actor.cli.bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM actor_handles WHERE workspace_id = 'ws_cli_bootstrap' AND actor_id = 'actor.cli.bootstrap' AND channel = 'message' AND handle_value = '+15551112222'`, 1)
}

func TestRunIdentityDeviceSessionCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity-device-session-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := openRuntimeDB(ctx, dbPath); err != nil {
		t.Fatalf("initialize runtime db: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 2, 26, 6, 0, 0, 0, time.UTC)
	nowText := now.Format(time.RFC3339Nano)
	seedStatements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_identity_inventory', 'Identity Inventory', 'ACTIVE', '` + nowText + `', '` + nowText + `')`,
		`INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES ('user.identity.one', 'one@example.com', 'Identity One', 'ACTIVE', '` + nowText + `', '` + nowText + `')`,
		`INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES ('user.identity.two', 'two@example.com', 'Identity Two', 'ACTIVE', '` + nowText + `', '` + nowText + `')`,
		`INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES ('device.identity.one', 'ws_identity_inventory', 'user.identity.one', 'phone', 'ios', 'Identity Phone', '2026-02-26T06:02:00Z', '2026-02-26T06:00:00Z')`,
		`INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES ('device.identity.two', 'ws_identity_inventory', 'user.identity.two', 'desktop', 'macos', 'Identity Desktop', '2026-02-26T06:03:00Z', '2026-02-26T06:01:00Z')`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session.identity.active', 'ws_identity_inventory', 'device.identity.one', 'hash-active', '2026-02-26T06:00:10Z', '2099-01-01T00:00:00Z', NULL)`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session.identity.expired', 'ws_identity_inventory', 'device.identity.one', 'hash-expired', '2026-02-25T05:00:00Z', '2026-02-25T05:30:00Z', NULL)`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session.identity.revoked', 'ws_identity_inventory', 'device.identity.two', 'hash-revoked', '2026-02-26T05:00:00Z', '2099-01-01T00:00:00Z', '2026-02-26T05:30:00Z')`,
	}
	for _, statement := range seedStatements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed identity device/session fixtures: %v\nstatement: %s", err, statement)
		}
	}

	devicesPage1Out := &bytes.Buffer{}
	devicesPage1Err := &bytes.Buffer{}
	devicesPage1Code := run(withDaemonArgs(server,
		"identity", "devices",
		"--workspace", "ws_identity_inventory",
		"--limit", "1",
	), devicesPage1Out, devicesPage1Err)
	if devicesPage1Code != 0 {
		t.Fatalf("identity devices page 1 failed: code=%d stderr=%s output=%s", devicesPage1Code, devicesPage1Err.String(), devicesPage1Out.String())
	}

	var devicesPage1 transport.IdentityDeviceListResponse
	if err := json.Unmarshal(devicesPage1Out.Bytes(), &devicesPage1); err != nil {
		t.Fatalf("decode identity devices page 1 response: %v", err)
	}
	if devicesPage1.WorkspaceID != "ws_identity_inventory" || len(devicesPage1.Items) != 1 || !devicesPage1.HasMore {
		t.Fatalf("unexpected identity devices page 1 response: %+v", devicesPage1)
	}
	if strings.TrimSpace(devicesPage1.NextCursorCreatedAt) == "" || strings.TrimSpace(devicesPage1.NextCursorID) == "" {
		t.Fatalf("expected devices pagination cursor metadata, got %+v", devicesPage1)
	}

	devicesPage2Out := &bytes.Buffer{}
	devicesPage2Err := &bytes.Buffer{}
	devicesPage2Code := run(withDaemonArgs(server,
		"identity", "devices",
		"--workspace", "ws_identity_inventory",
		"--cursor-created-at", devicesPage1.NextCursorCreatedAt,
		"--cursor-id", devicesPage1.NextCursorID,
		"--limit", "5",
	), devicesPage2Out, devicesPage2Err)
	if devicesPage2Code != 0 {
		t.Fatalf("identity devices page 2 failed: code=%d stderr=%s output=%s", devicesPage2Code, devicesPage2Err.String(), devicesPage2Out.String())
	}

	var devicesPage2 transport.IdentityDeviceListResponse
	if err := json.Unmarshal(devicesPage2Out.Bytes(), &devicesPage2); err != nil {
		t.Fatalf("decode identity devices page 2 response: %v", err)
	}
	if len(devicesPage2.Items) == 0 {
		t.Fatalf("expected non-empty identity devices page 2 response")
	}

	sessionsOut := &bytes.Buffer{}
	sessionsErr := &bytes.Buffer{}
	sessionsCode := run(withDaemonArgs(server,
		"identity", "sessions",
		"--workspace", "ws_identity_inventory",
		"--device-id", "device.identity.one",
		"--session-health", "active",
		"--limit", "5",
	), sessionsOut, sessionsErr)
	if sessionsCode != 0 {
		t.Fatalf("identity sessions failed: code=%d stderr=%s output=%s", sessionsCode, sessionsErr.String(), sessionsOut.String())
	}

	var sessionsResponse transport.IdentitySessionListResponse
	if err := json.Unmarshal(sessionsOut.Bytes(), &sessionsResponse); err != nil {
		t.Fatalf("decode identity sessions response: %v", err)
	}
	if sessionsResponse.WorkspaceID != "ws_identity_inventory" || len(sessionsResponse.Items) != 1 || sessionsResponse.Items[0].SessionID != "session.identity.active" {
		t.Fatalf("unexpected identity sessions response: %+v", sessionsResponse)
	}

	revokeFirstOut := &bytes.Buffer{}
	revokeFirstErr := &bytes.Buffer{}
	revokeFirstCode := run(withDaemonArgs(server,
		"identity", "revoke-session",
		"--workspace", "ws_identity_inventory",
		"--session-id", "session.identity.active",
	), revokeFirstOut, revokeFirstErr)
	if revokeFirstCode != 0 {
		t.Fatalf("identity revoke-session first call failed: code=%d stderr=%s output=%s", revokeFirstCode, revokeFirstErr.String(), revokeFirstOut.String())
	}

	var revokeFirst transport.IdentitySessionRevokeResponse
	if err := json.Unmarshal(revokeFirstOut.Bytes(), &revokeFirst); err != nil {
		t.Fatalf("decode identity revoke-session first response: %v", err)
	}
	if revokeFirst.SessionID != "session.identity.active" || revokeFirst.SessionHealth != "revoked" || revokeFirst.Idempotent {
		t.Fatalf("unexpected first identity revoke-session response: %+v", revokeFirst)
	}

	revokeSecondOut := &bytes.Buffer{}
	revokeSecondErr := &bytes.Buffer{}
	revokeSecondCode := run(withDaemonArgs(server,
		"identity", "revoke-session",
		"--workspace", "ws_identity_inventory",
		"--session-id", "session.identity.active",
	), revokeSecondOut, revokeSecondErr)
	if revokeSecondCode != 0 {
		t.Fatalf("identity revoke-session second call failed: code=%d stderr=%s output=%s", revokeSecondCode, revokeSecondErr.String(), revokeSecondOut.String())
	}

	var revokeSecond transport.IdentitySessionRevokeResponse
	if err := json.Unmarshal(revokeSecondOut.Bytes(), &revokeSecond); err != nil {
		t.Fatalf("decode identity revoke-session second response: %v", err)
	}
	if !revokeSecond.Idempotent || revokeSecond.SessionHealth != "revoked" {
		t.Fatalf("unexpected second identity revoke-session response: %+v", revokeSecond)
	}
}

func TestRunIdentitySelectWorkspaceRequiresWorkspace(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"identity", "select-workspace",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected identity select-workspace without --workspace to fail")
	}
	if !strings.Contains(stderr.String(), "--workspace is required") {
		t.Fatalf("expected required workspace error, got stderr=%s", stderr.String())
	}
}

func TestRunIdentityBootstrapRequiresPrincipal(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"identity", "bootstrap",
		"--workspace", "ws1",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected identity bootstrap without --principal to fail")
	}
	if !strings.Contains(stderr.String(), "--principal is required") {
		t.Fatalf("expected required principal error, got stderr=%s", stderr.String())
	}
}

func TestRunIdentityRevokeSessionRequiresSessionID(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"identity", "revoke-session",
		"--workspace", "ws1",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected identity revoke-session without --session-id to fail")
	}
	if !strings.Contains(stderr.String(), "--session-id is required") {
		t.Fatalf("expected required session id error, got stderr=%s", stderr.String())
	}
}
