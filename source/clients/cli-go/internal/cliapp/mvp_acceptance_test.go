package cliapp

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/securestore"
)

func TestCLIMVPAcceptanceSuite(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	secretServer := startCLITestServer(t)

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if r.Header.Get("Authorization") == "" {
				t.Fatalf("expected openai authorization header")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"mvp\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" response\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer openAIServer.Close()

	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollamaServer.Close()

	dbPath := filepath.Join(t.TempDir(), "mvp-acceptance.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)

	// 1) Secure credential onboarding.
	secretSet := runCLIJSON(t, []string{
		"--mode", "tcp",
		"--address", secretServer.Address(),
		"--auth-token", "cli-test-token",
		"secret", "set",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
		"--value", "sk-mvp-suite",
	})
	if secretSet["name"] != "OPENAI_API_KEY" {
		t.Fatalf("expected secret name in set response, got %v", secretSet["name"])
	}
	if secretSet["registered"] != true {
		t.Fatalf("expected registered=true in set response, got %v", secretSet["registered"])
	}
	if _, exists := secretSet["value"]; exists {
		t.Fatalf("did not expect plaintext value in secret set response")
	}
	if _, exists := secretSet["value_masked"]; exists {
		t.Fatalf("did not expect masked value in secret set response")
	}

	secretGet := runCLIJSON(t, []string{
		"--mode", "tcp",
		"--address", secretServer.Address(),
		"--auth-token", "cli-test-token",
		"secret", "get",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
	})
	if secretGet["name"] != "OPENAI_API_KEY" {
		t.Fatalf("expected secret get name OPENAI_API_KEY, got %v", secretGet["name"])
	}
	if _, exists := secretGet["value"]; exists {
		t.Fatalf("did not expect plaintext value in secret get response")
	}
	if _, exists := secretGet["value_masked"]; exists {
		t.Fatalf("did not expect masked value in secret get response")
	}

	// 2) Provider/model setup and checks.
	runCLIJSON(t, withDaemonArgs(daemonServer,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	))
	runCLIJSON(t, withDaemonArgs(daemonServer,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--endpoint", ollamaServer.URL,
	))

	providerCheck := runCLIJSON(t, withDaemonArgs(daemonServer,
		"provider", "check",
		"--workspace", "ws1",
	))
	if success, ok := providerCheck["success"].(bool); !ok || !success {
		t.Fatalf("expected provider checks success=true, got %v", providerCheck["success"])
	}

	runCLIJSON(t, withDaemonArgs(daemonServer,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "openai",
		"--model", "gpt-4.1-mini",
	))
	resolvedModel := runCLIJSON(t, withDaemonArgs(daemonServer,
		"model", "resolve",
		"--workspace", "ws1",
		"--task-class", "chat",
	))
	if resolvedModel["provider"] != "openai" || resolvedModel["model_key"] != "gpt-4.1-mini" {
		t.Fatalf("expected openai/gpt-4.1-mini model resolution, got provider=%v model=%v", resolvedModel["provider"], resolvedModel["model_key"])
	}

	// 3) Chat with streaming response.
	chatOutput := runCLIText(t, withDaemonArgs(daemonServer,
		"chat",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--message", "hello suite",
	))
	if !strings.Contains(chatOutput, "mvp response") {
		t.Fatalf("expected chat output to contain streamed response, got %q", chatOutput)
	}

	// 4) Agent action execution and 5) destructive approval flow.
	normalRun := runCLIJSON(t, withDaemonArgs(daemonServer,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "send an email update to sam@example.com",
	))
	if normalRun["workflow"] != "mail" {
		t.Fatalf("expected mail workflow for normal execution, got %v", normalRun["workflow"])
	}

	destructiveRun := runCLIJSON(t, withDaemonArgs(daemonServer,
		"agent", "run",
		"--workspace", "ws1",
		"--request", "delete file /tmp/mvp.txt",
	))
	if required, ok := destructiveRun["approval_required"].(bool); !ok || !required {
		t.Fatalf("expected approval_required=true for destructive run, got %v", destructiveRun["approval_required"])
	}
	approvalID, ok := destructiveRun["approval_request_id"].(string)
	if !ok || approvalID == "" {
		t.Fatalf("expected approval_request_id in destructive run response")
	}

	approve := runCLIJSON(t, withDaemonArgs(daemonServer,
		"agent", "approve",
		"--workspace", "ws1",
		"--approval-id", approvalID,
		"--phrase", "GO AHEAD",
		"--actor-id", "actor.requester",
	))
	if approve["run_state"] != "completed" {
		t.Fatalf("expected approved run_state=completed, got %v", approve["run_state"])
	}

	// 6) Communication and automation workflows with idempotency.
	operationID := "mvp-op-comm-001"
	commSend := runCLIJSON(t, withDaemonArgs(daemonServer,
		"comm", "send",
		"--workspace", "ws1",
		"--operation-id", operationID,
		"--source-channel", "message",
		"--destination", "+15555550123",
		"--message", "status",
		"--imessage-failures", "2",
	))
	commResult, ok := commSend["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected comm result payload")
	}
	if commResult["Channel"] != "twilio" {
		t.Fatalf("expected comm fallback channel twilio, got %v", commResult["Channel"])
	}

	attempts := runCLIJSON(t, withDaemonArgs(daemonServer,
		"comm", "attempts",
		"--workspace", "ws1",
		"--operation-id", operationID,
	))
	loggedAttempts, ok := attempts["attempts"].([]any)
	if !ok || len(loggedAttempts) != 3 {
		t.Fatalf("expected 3 logged comm attempts, got %v", attempts["attempts"])
	}

	runCLIJSON(t, withDaemonArgs(daemonServer,
		"automation", "create",
		"--workspace", "ws1",
		"--subject", "actor.requester",
		"--trigger-type", "ON_COMM_EVENT",
		"--filter", `{"channels":["message"]}`,
	))

	automationRun1 := runCLIJSON(t, withDaemonArgs(daemonServer,
		"automation", "run", "comm-event",
		"--workspace", "ws1",
		"--event-id", "mvp-comm-event-1",
		"--channel", "message",
		"--body", "please handle this",
		"--sender", "sender@example.com",
	))
	autoResult1, ok := automationRun1["result"].(map[string]any)
	if !ok || autoResult1["Created"] != float64(1) {
		t.Fatalf("expected first automation comm-event run Created=1, got %v", automationRun1["result"])
	}

	automationRun2 := runCLIJSON(t, withDaemonArgs(daemonServer,
		"automation", "run", "comm-event",
		"--workspace", "ws1",
		"--event-id", "mvp-comm-event-1",
		"--channel", "message",
		"--body", "please handle this",
		"--sender", "sender@example.com",
	))
	autoResult2, ok := automationRun2["result"].(map[string]any)
	if !ok || autoResult2["Created"] != float64(0) {
		t.Fatalf("expected replay automation comm-event run Created=0, got %v", automationRun2["result"])
	}
}

func TestCLITwilioAcceptanceSuite(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2010-04-01/Accounts/AC123/Messages.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sid":"SMACC1","account_sid":"AC123","status":"queued","from":"+15555550001","to":"+15555550999"}`))
		case "/2010-04-01/Accounts/AC123/Calls.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sid":"CAACC1","account_sid":"AC123","status":"queued","direction":"outbound-api","from":"+15555550002","to":"+15555550999"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-acceptance.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)

	runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-acceptance-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	))

	smsChat := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "sms-chat",
		"--workspace", "ws1",
		"--to", "+15555550999",
		"--message", "twilio acceptance ping",
	))
	turns, ok := smsChat["turns"].([]any)
	if !ok || len(turns) != 1 {
		t.Fatalf("expected exactly one sms chat turn, got %v", smsChat["turns"])
	}
	firstTurn, ok := turns[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected sms turn shape: %T", turns[0])
	}
	if success, ok := firstTurn["success"].(bool); !ok || !success {
		t.Fatalf("expected sms turn success=true, got %v", firstTurn["success"])
	}

	inboundSMS := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "ingest-sms",
		"--workspace", "ws1",
		"--skip-signature=true",
		"--from", "+15555550999",
		"--to", "+15555550001",
		"--body", "inbound acceptance message",
		"--message-sid", "SMINACC1",
		"--account-sid", "AC123",
	))
	if accepted, ok := inboundSMS["accepted"].(bool); !ok || !accepted {
		t.Fatalf("expected inbound sms accepted=true, got %v", inboundSMS["accepted"])
	}

	startCall := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "start-call",
		"--workspace", "ws1",
		"--to", "+15555550999",
		"--twiml-url", "https://agent.local/twiml/voice",
	))
	callSID, ok := startCall["call_sid"].(string)
	if !ok || callSID == "" {
		t.Fatalf("expected call_sid in start-call response, got %v", startCall["call_sid"])
	}

	voiceCallback := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "ingest-voice",
		"--workspace", "ws1",
		"--skip-signature=true",
		"--provider-event-id", "twilio-acceptance-voice-1",
		"--call-sid", callSID,
		"--account-sid", "AC123",
		"--from", "+15555550002",
		"--to", "+15555550999",
		"--direction", "outbound-api",
		"--call-status", "in-progress",
		"--transcript", "hello from voice acceptance",
		"--transcript-direction", "INBOUND",
	))
	if status, ok := voiceCallback["call_status"]; !ok || status != "in_progress" {
		t.Fatalf("expected voice callback status in_progress, got %v", voiceCallback["call_status"])
	}

	callStatus := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "call-status",
		"--workspace", "ws1",
		"--call-sid", callSID,
	))
	sessions, ok := callStatus["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected one call status record, got %v", callStatus["sessions"])
	}
	session, ok := sessions[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected call status session payload shape: %T", sessions[0])
	}
	if session["status"] != "in_progress" {
		t.Fatalf("expected call session status in_progress, got %v", session["status"])
	}

	transcript := runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "transcript",
		"--workspace", "ws1",
		"--call-sid", callSID,
		"--limit", "20",
	))
	events, ok := transcript["events"].([]any)
	if !ok || len(events) == 0 {
		t.Fatalf("expected transcript events, got %v", transcript["events"])
	}
	foundVoiceTranscript := false
	for _, raw := range events {
		event, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if event["event_type"] == "VOICE_TRANSCRIPT" {
			foundVoiceTranscript = true
			break
		}
	}
	if !foundVoiceTranscript {
		t.Fatalf("expected VOICE_TRANSCRIPT event in transcript output, got %v", transcript["events"])
	}
}

func TestCLITwilioConversationalWebhookAcceptanceSuite(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-webhook-acceptance"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"acceptance\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" reply\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer openAIServer.Close()

	var outboundSendCount int
	twilioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			http.NotFound(w, r)
			return
		}
		outboundSendCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sid":"SMWEBHOOKACC1","account_sid":"AC123","status":"queued","from":"+15555550001","to":"+15555550999"}`))
	}))
	defer twilioServer.Close()

	dbPath := filepath.Join(t.TempDir(), "twilio-webhook-acceptance.db")
	daemonServer := startCLITestServerWithDaemonServices(t, dbPath, manager)

	runCLIJSON(t, withDaemonArgs(daemonServer,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	))
	runCLIJSON(t, withDaemonArgs(daemonServer,
		"model", "select",
		"--workspace", "ws1",
		"--task-class", "chat",
		"--provider", "openai",
		"--model", "gpt-4.1-mini",
	))
	runCLIJSON(t, withDaemonArgs(daemonServer,
		"connector", "twilio", "set",
		"--workspace", "ws1",
		"--account-sid", "AC123",
		"--auth-token", "twilio-acceptance-token",
		"--sms-number", "+15555550001",
		"--voice-number", "+15555550002",
		"--endpoint", twilioServer.URL,
	))

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
			"--timeout", "6s",
			"connector", "twilio", "webhook", "serve",
			"--workspace", "ws1",
			"--listen", address,
			"--signature-mode", "bypass",
			"--assistant-replies=true",
			"--assistant-task-class", "chat",
			"--voice-response-mode", "twiml",
			"--run-for", "1800ms",
		), serveOut, serveErr)
		done <- code
	}()

	time.Sleep(200 * time.Millisecond)

	smsResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookSMSPath(), url.Values{
		"From":       {"+15555550999"},
		"To":         {"+15555550001"},
		"Body":       {"webhook conversational acceptance"},
		"MessageSid": {"SMWEBHOOKIN1"},
		"AccountSid": {"AC123"},
	})
	if err != nil {
		t.Fatalf("post sms webhook: %v", err)
	}
	defer smsResponse.Body.Close()
	if smsResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(smsResponse.Body)
		t.Fatalf("expected sms webhook status 200, got %d body=%s", smsResponse.StatusCode, strings.TrimSpace(string(body)))
	}
	smsBody, err := io.ReadAll(smsResponse.Body)
	if err != nil {
		t.Fatalf("read sms webhook response: %v", err)
	}
	var smsPayload map[string]any
	if err := json.Unmarshal(smsBody, &smsPayload); err != nil {
		t.Fatalf("decode sms webhook response: %v output=%s", err, strings.TrimSpace(string(smsBody)))
	}
	if got := smsPayload["assistant_reply"]; got != "acceptance reply" {
		t.Fatalf("expected assistant_reply='acceptance reply', got %v", got)
	}

	voiceResponse, err := http.PostForm("http://"+address+defaultTwilioWebhookVoicePath(), url.Values{
		"CallSid":      {"CAWEBHOOKACC1"},
		"AccountSid":   {"AC123"},
		"From":         {"+15555550999"},
		"To":           {"+15555550002"},
		"Direction":    {"inbound"},
		"CallStatus":   {"in-progress"},
		"SpeechResult": {"hello voice conversational acceptance"},
	})
	if err != nil {
		t.Fatalf("post voice webhook: %v", err)
	}
	defer voiceResponse.Body.Close()
	if voiceResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(voiceResponse.Body)
		t.Fatalf("expected voice webhook status 200, got %d body=%s", voiceResponse.StatusCode, strings.TrimSpace(string(body)))
	}
	voiceBody, err := io.ReadAll(voiceResponse.Body)
	if err != nil {
		t.Fatalf("read voice webhook response: %v", err)
	}
	voiceText := strings.TrimSpace(string(voiceBody))
	if !strings.Contains(voiceText, "<Response>") || !strings.Contains(voiceText, "<Gather") {
		t.Fatalf("expected twiml response body, got %q", voiceText)
	}
	if !strings.Contains(voiceText, "acceptance reply") {
		t.Fatalf("expected generated assistant content in twiml response, got %q", voiceText)
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("webhook serve command failed: code=%d stderr=%s output=%s", code, serveErr.String(), serveOut.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for webhook serve command to exit")
	}

	if outboundSendCount != 1 {
		t.Fatalf("expected one outbound assistant sms send, got %d", outboundSendCount)
	}
}

func runCLIJSON(t *testing.T, args []string) map[string]any {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(args, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("cli command failed: args=%v code=%d stderr=%s output=%s", args, exitCode, stderr.String(), stdout.String())
	}

	payload := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode cli json output failed: args=%v err=%v output=%s", args, err, stdout.String())
	}
	return payload
}

func runCLIText(t *testing.T, args []string) string {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(args, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("cli command failed: args=%v code=%d stderr=%s output=%s", args, exitCode, stderr.String(), stdout.String())
	}
	return stdout.String()
}
