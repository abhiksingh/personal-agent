package cliapp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"

	_ "modernc.org/sqlite"
)

func TestRunTaskSubmitAndStatusCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	requester := "actor.requester"

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", requester,
		"--subject", requester,
		"--title", "CLI Task",
	}, submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("submit exit code %d, stderr=%s", submitCode, submitErr.String())
	}

	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	taskID, ok := submitResponse["task_id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("expected task_id in submit response, got %v", submitResponse["task_id"])
	}
	runID, ok := submitResponse["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id in submit response, got %v", submitResponse["run_id"])
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "status",
		"--task-id", taskID,
	}, statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("status exit code %d, stderr=%s", statusCode, statusErr.String())
	}

	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got := statusResponse["task_id"]; got != taskID {
		t.Fatalf("expected task_id %s, got %v", taskID, got)
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
		t.Fatalf("query persisted task state: %v", err)
	}
	if taskState != "queued" {
		t.Fatalf("expected persisted task state queued, got %s", taskState)
	}

	var runState string
	if err := db.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, runID).Scan(&runState); err != nil {
		t.Fatalf("query persisted run state: %v", err)
	}
	if runState != "queued" {
		t.Fatalf("expected persisted run state queued, got %s", runState)
	}
}

func TestRunTaskCancelCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	requester := "actor.requester"

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", requester,
		"--subject", requester,
		"--title", "CLI Task Cancel",
	}, submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("submit exit code %d, stderr=%s", submitCode, submitErr.String())
	}

	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	taskID, _ := submitResponse["task_id"].(string)
	runID, _ := submitResponse["run_id"].(string)
	if taskID == "" || runID == "" {
		t.Fatalf("expected task_id + run_id in submit response, got %+v", submitResponse)
	}

	cancelOut := &bytes.Buffer{}
	cancelErr := &bytes.Buffer{}
	cancelCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "cancel",
		"--run-id", runID,
		"--reason", "manual cancel test",
	}, cancelOut, cancelErr)
	if cancelCode != 0 {
		t.Fatalf("cancel exit code %d, stderr=%s", cancelCode, cancelErr.String())
	}

	var cancelResponse map[string]any
	if err := json.Unmarshal(cancelOut.Bytes(), &cancelResponse); err != nil {
		t.Fatalf("decode cancel response: %v", err)
	}
	if got := cancelResponse["task_id"]; got != taskID {
		t.Fatalf("expected cancelled task_id %s, got %v", taskID, got)
	}
	if got := cancelResponse["run_id"]; got != runID {
		t.Fatalf("expected cancelled run_id %s, got %v", runID, got)
	}
	if got := cancelResponse["cancelled"]; got != true {
		t.Fatalf("expected cancelled=true, got %v", got)
	}
	if got := cancelResponse["run_state"]; got != "cancelled" {
		t.Fatalf("expected run_state cancelled, got %v", got)
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "status",
		"--task-id", taskID,
	}, statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("status exit code %d, stderr=%s", statusCode, statusErr.String())
	}
	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got := statusResponse["state"]; got != "cancelled" {
		t.Fatalf("expected task status cancelled, got %v", got)
	}
}

func TestRunTaskRetryCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	requester := "actor.requester"

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run(withDaemonArgs(server,
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", requester,
		"--subject", requester,
		"--title", "CLI Task Retry",
	), submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("submit exit code %d, stderr=%s", submitCode, submitErr.String())
	}

	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	taskID, _ := submitResponse["task_id"].(string)
	runID, _ := submitResponse["run_id"].(string)
	if taskID == "" || runID == "" {
		t.Fatalf("expected task_id + run_id in submit response, got %+v", submitResponse)
	}

	cancelCode := run(withDaemonArgs(server,
		"task", "cancel",
		"--run-id", runID,
		"--reason", "cancel before retry",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if cancelCode != 0 {
		t.Fatalf("cancel before retry failed")
	}

	retryOut := &bytes.Buffer{}
	retryErr := &bytes.Buffer{}
	retryCode := run(withDaemonArgs(server,
		"task", "retry",
		"--run-id", runID,
		"--reason", "retry from cli",
	), retryOut, retryErr)
	if retryCode != 0 {
		t.Fatalf("retry exit code %d, stderr=%s", retryCode, retryErr.String())
	}

	var retryResponse map[string]any
	if err := json.Unmarshal(retryOut.Bytes(), &retryResponse); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if got := retryResponse["retried"]; got != true {
		t.Fatalf("expected retried=true, got %v", got)
	}
	if got := retryResponse["previous_run_id"]; got != runID {
		t.Fatalf("expected previous_run_id %s, got %v", runID, got)
	}
	retryRunID, _ := retryResponse["run_id"].(string)
	if retryRunID == "" || retryRunID == runID {
		t.Fatalf("expected new run_id in retry response, got %v", retryResponse["run_id"])
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run(withDaemonArgs(server,
		"task", "status",
		"--task-id", taskID,
	), statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("status exit code %d, stderr=%s", statusCode, statusErr.String())
	}
	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got := statusResponse["state"]; got != "queued" {
		t.Fatalf("expected task status queued after retry, got %v", got)
	}
	if got := statusResponse["run_id"]; got != retryRunID {
		t.Fatalf("expected latest run_id %s after retry, got %v", retryRunID, got)
	}
}

func TestRunTaskRequeueCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)
	requester := "actor.requester"

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run(withDaemonArgs(server,
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", requester,
		"--subject", requester,
		"--title", "CLI Task Requeue",
	), submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("submit exit code %d, stderr=%s", submitCode, submitErr.String())
	}

	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	taskID, _ := submitResponse["task_id"].(string)
	runID, _ := submitResponse["run_id"].(string)
	if taskID == "" || runID == "" {
		t.Fatalf("expected task_id + run_id in submit response, got %+v", submitResponse)
	}

	requeueOut := &bytes.Buffer{}
	requeueErr := &bytes.Buffer{}
	requeueCode := run(withDaemonArgs(server,
		"task", "requeue",
		"--run-id", runID,
		"--reason", "requeue from cli",
	), requeueOut, requeueErr)
	if requeueCode != 0 {
		t.Fatalf("requeue exit code %d, stderr=%s", requeueCode, requeueErr.String())
	}

	var requeueResponse map[string]any
	if err := json.Unmarshal(requeueOut.Bytes(), &requeueResponse); err != nil {
		t.Fatalf("decode requeue response: %v", err)
	}
	if got := requeueResponse["requeued"]; got != true {
		t.Fatalf("expected requeued=true, got %v", got)
	}
	if got := requeueResponse["previous_run_id"]; got != runID {
		t.Fatalf("expected previous_run_id %s, got %v", runID, got)
	}
	requeueRunID, _ := requeueResponse["run_id"].(string)
	if requeueRunID == "" || requeueRunID == runID {
		t.Fatalf("expected new run_id in requeue response, got %v", requeueResponse["run_id"])
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run(withDaemonArgs(server,
		"task", "status",
		"--task-id", taskID,
	), statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("status exit code %d, stderr=%s", statusCode, statusErr.String())
	}
	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got := statusResponse["state"]; got != "queued" {
		t.Fatalf("expected task status queued after requeue, got %v", got)
	}
	if got := statusResponse["run_id"]; got != requeueRunID {
		t.Fatalf("expected latest run_id %s after requeue, got %v", requeueRunID, got)
	}
}

func TestRunTaskSubmitDeniesCrossPrincipalWithoutDelegation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run(withDaemonArgs(server,
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", "actor.a",
		"--subject", "actor.b",
		"--title", "Cross principal task",
	), stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected cross-principal task submit denial without delegation")
	}
	if !strings.Contains(stderr.String(), "acting_as denied") {
		t.Fatalf("expected acting_as denial, got stderr=%s", stderr.String())
	}
}

func TestRunTaskSubmitAllowsCrossPrincipalWithDelegation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "task-runtime.db")
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

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run(withDaemonArgs(server,
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", "actor.a",
		"--subject", "actor.b",
		"--title", "Cross principal task",
	), submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("expected task submit success with delegation, code=%d stderr=%s", submitCode, submitErr.String())
	}

	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	taskID, ok := submitResponse["task_id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("expected task_id in submit response, got %v", submitResponse["task_id"])
	}
	runID, ok := submitResponse["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id in submit response, got %v", submitResponse["run_id"])
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run(withDaemonArgs(server,
		"task", "status",
		"--task-id", taskID,
	), statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("expected status success for delegated submit, code=%d stderr=%s", statusCode, statusErr.String())
	}

	var statusResponse map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusResponse); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got := statusResponse["task_id"]; got != taskID {
		t.Fatalf("expected task_id %s, got %v", taskID, got)
	}
	if got := statusResponse["state"]; got != "queued" {
		t.Fatalf("expected queued task status, got %v", got)
	}
}

func TestRunStreamCommandConsumesRealtimeEvent(t *testing.T) {
	server := startCLITestServer(t)

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", "actor.requester",
		"--subject", "actor.requester",
		"--title", "Stream cancel signal task",
	}, submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("submit exit code %d, stderr=%s", submitCode, submitErr.String())
	}
	var submitResponse map[string]any
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	runID, _ := submitResponse["run_id"].(string)
	if runID == "" {
		t.Fatalf("expected run_id from task submit, got %+v", submitResponse)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"stream",
		"--duration", "1200ms",
		"--signal-type", "cancel",
		"--run-id", runID,
		"--reason", "stream signal test",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("stream exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("\"event_type\": \"client_signal_ack\"")) {
		t.Fatalf("expected client_signal_ack event in stream output, got %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("\"accepted\": true")) {
		t.Fatalf("expected accepted=true in stream ack output, got %s", stdout.String())
	}
}

func TestRunStreamCommandNoEventsExitsCleanly(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"stream",
		"--duration", "350ms",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("stream (no events) exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stream events for idle duration, got %s", stdout.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %s", stderr.String())
	}
}

func TestRunConnectorSmokeCommand(t *testing.T) {
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"connector", "smoke",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector smoke exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector smoke response: %v", err)
	}
	if success, ok := response["success"].(bool); !ok || !success {
		t.Fatalf("expected connector smoke success=true, got %v", response["success"])
	}
}

func TestRunConnectorBridgeStatusAndSetupCommands(t *testing.T) {
	bridgeRoot := filepath.Join(t.TempDir(), "inbound-bridge")
	t.Setenv("PA_INBOUND_WATCHER_INBOX_DIR", bridgeRoot)

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run([]string{
		"connector", "bridge", "status",
		"--workspace", "ws1",
	}, statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("connector bridge status exit code %d, stderr=%s", statusCode, statusErr.String())
	}

	var statusPayload map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode connector bridge status payload: %v", err)
	}
	statusRecord, ok := statusPayload["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected status object in bridge status payload, got %v", statusPayload["status"])
	}
	if ready, ok := statusRecord["ready"].(bool); !ok || ready {
		t.Fatalf("expected bridge ready=false before setup, got %v", statusRecord["ready"])
	}

	setupOut := &bytes.Buffer{}
	setupErr := &bytes.Buffer{}
	setupCode := run([]string{
		"connector", "bridge", "setup",
		"--workspace", "ws1",
	}, setupOut, setupErr)
	if setupCode != 0 {
		t.Fatalf("connector bridge setup exit code %d, stderr=%s", setupCode, setupErr.String())
	}
	var setupPayload map[string]any
	if err := json.Unmarshal(setupOut.Bytes(), &setupPayload); err != nil {
		t.Fatalf("decode connector bridge setup payload: %v", err)
	}
	if ensureApplied, ok := setupPayload["ensure_applied"].(bool); !ok || !ensureApplied {
		t.Fatalf("expected ensure_applied=true, got %v", setupPayload["ensure_applied"])
	}
	setupStatus, ok := setupPayload["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected setup status object in payload, got %v", setupPayload["status"])
	}
	if ready, ok := setupStatus["ready"].(bool); !ok || !ready {
		t.Fatalf("expected bridge ready=true after setup, got %v", setupStatus["ready"])
	}
}

func TestRunConnectorMailHandoffCommand(t *testing.T) {
	bridgeRoot := filepath.Join(t.TempDir(), "inbound-bridge")
	t.Setenv("PA_INBOUND_WATCHER_INBOX_DIR", bridgeRoot)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"connector", "mail", "handoff",
		"--workspace", "ws1",
		"--source-scope", "mailbox://inbox",
		"--source-event-id", "mail-handoff-event-1",
		"--source-cursor", "1001",
		"--message-id", "<mail-handoff-event-1@example.com>",
		"--thread-ref", "mail-thread-1",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
		"--subject", "mail handoff test",
		"--body", "mail handoff body",
		"--occurred-at", "2026-02-26T10:00:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector mail handoff exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector mail handoff response: %v", err)
	}
	if queued, ok := response["queued"].(bool); !ok || !queued {
		t.Fatalf("expected queued=true, got %v", response["queued"])
	}
	filePath := strings.TrimSpace(fmt.Sprintf("%v", response["file_path"]))
	if filePath == "" {
		t.Fatalf("expected non-empty file_path")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected handoff file to exist at %s: %v", filePath, err)
	}
}

func TestRunConnectorCalendarHandoffCommand(t *testing.T) {
	bridgeRoot := filepath.Join(t.TempDir(), "inbound-bridge")
	t.Setenv("PA_INBOUND_WATCHER_INBOX_DIR", bridgeRoot)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"connector", "calendar", "handoff",
		"--workspace", "ws1",
		"--source-scope", "calendar://inbox",
		"--source-event-id", "calendar-handoff-event-1",
		"--source-cursor", "2001",
		"--calendar-id", "calendar-id-1",
		"--calendar-name", "Personal",
		"--event-uid", "event-uid-1",
		"--change-type", "updated",
		"--title", "Calendar handoff test",
		"--notes", "calendar handoff body",
		"--location", "Room A",
		"--starts-at", "2026-02-26T10:00:00Z",
		"--ends-at", "2026-02-26T11:00:00Z",
		"--occurred-at", "2026-02-26T09:30:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector calendar handoff exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector calendar handoff response: %v", err)
	}
	if queued, ok := response["queued"].(bool); !ok || !queued {
		t.Fatalf("expected queued=true, got %v", response["queued"])
	}
	filePath := strings.TrimSpace(fmt.Sprintf("%v", response["file_path"]))
	if filePath == "" {
		t.Fatalf("expected non-empty file_path")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected handoff file to exist at %s: %v", filePath, err)
	}
}

func TestRunConnectorBrowserHandoffCommand(t *testing.T) {
	bridgeRoot := filepath.Join(t.TempDir(), "inbound-bridge")
	t.Setenv("PA_INBOUND_WATCHER_INBOX_DIR", bridgeRoot)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"connector", "browser", "handoff",
		"--workspace", "ws1",
		"--source-scope", "safari://window/1",
		"--source-event-id", "browser-handoff-event-1",
		"--source-cursor", "3001",
		"--window-id", "window-1",
		"--tab-id", "tab-1",
		"--page-url", "https://example.com",
		"--page-title", "Example Domain",
		"--event-type", "navigation",
		"--payload", "page loaded",
		"--occurred-at", "2026-02-26T10:30:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector browser handoff exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector browser handoff response: %v", err)
	}
	if queued, ok := response["queued"].(bool); !ok || !queued {
		t.Fatalf("expected queued=true, got %v", response["queued"])
	}
	filePath := strings.TrimSpace(fmt.Sprintf("%v", response["file_path"]))
	if filePath == "" {
		t.Fatalf("expected non-empty file_path")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected handoff file to exist at %s: %v", filePath, err)
	}
}

func TestRunConnectorMailIngestCommand(t *testing.T) {
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")
	server := startCLITestServerWithDaemonServices(t, filepath.Join(t.TempDir(), "daemon-runtime.db"), nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"connector", "mail", "ingest",
		"--workspace", "ws1",
		"--source-scope", "mailbox://inbox",
		"--source-event-id", "mail-test-event-1",
		"--source-cursor", "1001",
		"--message-id", "<mail-test-event-1@example.com>",
		"--thread-ref", "mail-thread-1",
		"--from", "sender@example.com",
		"--to", "recipient@example.com",
		"--subject", "mail ingest test",
		"--body", "mail ingest body",
		"--occurred-at", "2026-02-24T15:00:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector mail ingest exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector mail ingest response: %v", err)
	}
	if got := response["source"]; got != "apple_mail_rule" {
		t.Fatalf("expected source apple_mail_rule, got %v", got)
	}
	if accepted, ok := response["accepted"].(bool); !ok || !accepted {
		t.Fatalf("expected accepted=true, got %v", response["accepted"])
	}
	if eventID := strings.TrimSpace(fmt.Sprintf("%v", response["event_id"])); eventID == "" {
		t.Fatalf("expected non-empty event_id, got %v", response["event_id"])
	}
}

func TestRunConnectorCalendarIngestCommand(t *testing.T) {
	server := startCLITestServerWithDaemonServices(t, filepath.Join(t.TempDir(), "daemon-runtime.db"), nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"connector", "calendar", "ingest",
		"--workspace", "ws1",
		"--source-scope", "calendar://inbox",
		"--source-event-id", "calendar-test-event-1",
		"--source-cursor", "2001",
		"--calendar-id", "calendar-id-1",
		"--calendar-name", "Personal",
		"--event-uid", "event-uid-1",
		"--change-type", "updated",
		"--title", "Calendar ingest test",
		"--notes", "calendar ingest body",
		"--location", "Room A",
		"--starts-at", "2026-02-24T15:00:00Z",
		"--ends-at", "2026-02-24T16:00:00Z",
		"--occurred-at", "2026-02-24T14:30:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector calendar ingest exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector calendar ingest response: %v", err)
	}
	if got := response["source"]; got != "apple_calendar_eventkit" {
		t.Fatalf("expected source apple_calendar_eventkit, got %v", got)
	}
	if accepted, ok := response["accepted"].(bool); !ok || !accepted {
		t.Fatalf("expected accepted=true, got %v", response["accepted"])
	}
	if changeType := strings.TrimSpace(fmt.Sprintf("%v", response["change_type"])); changeType != "updated" {
		t.Fatalf("expected change_type updated, got %v", response["change_type"])
	}
	if eventID := strings.TrimSpace(fmt.Sprintf("%v", response["event_id"])); eventID == "" {
		t.Fatalf("expected non-empty event_id, got %v", response["event_id"])
	}
}

func TestRunConnectorBrowserIngestCommand(t *testing.T) {
	server := startCLITestServerWithDaemonServices(t, filepath.Join(t.TempDir(), "daemon-runtime.db"), nil)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"connector", "browser", "ingest",
		"--workspace", "ws1",
		"--source-scope", "safari://window/1",
		"--source-event-id", "browser-test-event-1",
		"--source-cursor", "3001",
		"--window-id", "window-1",
		"--tab-id", "tab-1",
		"--page-url", "https://example.com",
		"--page-title", "Example Domain",
		"--event-type", "navigation",
		"--payload", "page loaded",
		"--occurred-at", "2026-02-24T15:30:00Z",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("connector browser ingest exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode connector browser ingest response: %v", err)
	}
	if got := response["source"]; got != "apple_safari_extension" {
		t.Fatalf("expected source apple_safari_extension, got %v", got)
	}
	if accepted, ok := response["accepted"].(bool); !ok || !accepted {
		t.Fatalf("expected accepted=true, got %v", response["accepted"])
	}
	if eventType := strings.TrimSpace(fmt.Sprintf("%v", response["event_type"])); eventType != "navigation" {
		t.Fatalf("expected event_type navigation, got %v", response["event_type"])
	}
	if eventID := strings.TrimSpace(fmt.Sprintf("%v", response["event_id"])); eventID == "" {
		t.Fatalf("expected non-empty event_id, got %v", response["event_id"])
	}
}

func TestRunSecretSetGetDeleteCommands(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	server := startCLITestServer(t)

	baseArgs := []string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
	}

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run(append(baseArgs,
		"secret", "set",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
		"--value", "sk-secret-123",
	), setOut, setErr)
	if setCode != 0 {
		t.Fatalf("secret set exit code %d, stderr=%s", setCode, setErr.String())
	}

	var setResponse map[string]any
	if err := json.Unmarshal(setOut.Bytes(), &setResponse); err != nil {
		t.Fatalf("decode set response: %v", err)
	}
	if got := setResponse["workspace_id"]; got != "ws1" {
		t.Fatalf("expected workspace_id ws1, got %v", got)
	}
	if got := setResponse["name"]; got != "OPENAI_API_KEY" {
		t.Fatalf("expected secret name OPENAI_API_KEY, got %v", got)
	}
	if got := setResponse["registered"]; got != true {
		t.Fatalf("expected registered=true, got %v", got)
	}
	if got := setResponse["service"]; strings.TrimSpace(fmt.Sprintf("%v", got)) == "" {
		t.Fatalf("expected non-empty service in response, got %v", got)
	}
	if got := setResponse["account"]; got != "OPENAI_API_KEY" {
		t.Fatalf("expected account OPENAI_API_KEY, got %v", got)
	}
	if _, exists := setResponse["value"]; exists {
		t.Fatalf("did not expect plaintext value in secret set output")
	}
	if _, exists := setResponse["value_masked"]; exists {
		t.Fatalf("did not expect masked value in secret set output")
	}

	getOut := &bytes.Buffer{}
	getErr := &bytes.Buffer{}
	getCode := run(append(baseArgs,
		"secret", "get",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
	), getOut, getErr)
	if getCode != 0 {
		t.Fatalf("secret get exit code %d, stderr=%s", getCode, getErr.String())
	}

	var getResponse map[string]any
	if err := json.Unmarshal(getOut.Bytes(), &getResponse); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got := getResponse["workspace_id"]; got != "ws1" {
		t.Fatalf("expected workspace_id ws1, got %v", got)
	}
	if got := getResponse["name"]; got != "OPENAI_API_KEY" {
		t.Fatalf("expected secret name OPENAI_API_KEY, got %v", got)
	}
	if got := getResponse["service"]; strings.TrimSpace(fmt.Sprintf("%v", got)) == "" {
		t.Fatalf("expected non-empty service in response, got %v", got)
	}
	if got := getResponse["account"]; got != "OPENAI_API_KEY" {
		t.Fatalf("expected account OPENAI_API_KEY, got %v", got)
	}
	if _, exists := getResponse["value"]; exists {
		t.Fatalf("did not expect plaintext value in secret get output")
	}
	if _, exists := getResponse["value_masked"]; exists {
		t.Fatalf("did not expect masked value in secret get output")
	}

	deleteOut := &bytes.Buffer{}
	deleteErr := &bytes.Buffer{}
	deleteCode := run(append(baseArgs,
		"secret", "delete",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
	), deleteOut, deleteErr)
	if deleteCode != 0 {
		t.Fatalf("secret delete exit code %d, stderr=%s", deleteCode, deleteErr.String())
	}
	var deleteResponse map[string]any
	if err := json.Unmarshal(deleteOut.Bytes(), &deleteResponse); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if got := deleteResponse["deleted"]; got != true {
		t.Fatalf("expected deleted=true, got %v", got)
	}
	if _, exists := deleteResponse["value"]; exists {
		t.Fatalf("did not expect plaintext value in secret delete output")
	}

	missingOut := &bytes.Buffer{}
	missingErr := &bytes.Buffer{}
	missingCode := run(append(baseArgs,
		"secret", "get",
		"--workspace", "ws1",
		"--name", "OPENAI_API_KEY",
	), missingOut, missingErr)
	if missingCode == 0 {
		t.Fatalf("expected secret get after delete to fail")
	}
}

func TestRunSecretSetFromFile(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	server := startCLITestServer(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "openai_token.txt")
	if err := os.WriteFile(path, []byte("  sk-file-secret \n"), 0o600); err != nil {
		t.Fatalf("write temp secret file: %v", err)
	}

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"secret", "set",
		"--workspace", "ws-file",
		"--name", "OPENAI_API_KEY",
		"--file", path,
	}, setOut, setErr)
	if setCode != 0 {
		t.Fatalf("secret set --file exit code %d, stderr=%s", setCode, setErr.String())
	}

	_, value, err := manager.Get("ws-file", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("expected stored value: %v", err)
	}
	if value != "sk-file-secret" {
		t.Fatalf("expected trimmed file secret value, got %q", value)
	}
}

func TestRunSecretCommandRequiresReachableDaemon(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", "127.0.0.1:1",
		"--auth-token", "unused",
		"secret", "set",
		"--workspace", "ws-offline",
		"--name", "OPENAI_API_KEY",
		"--value", "sk-offline",
	}, stdout, stderr)
	if exitCode == 0 {
		t.Fatalf("expected secret command to fail when daemon is unreachable")
	}
	if !strings.Contains(stderr.String(), "request failed") {
		t.Fatalf("expected transport request failure, stderr=%s", stderr.String())
	}
}

func TestManagerDeleteReturnsNotFound(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if _, err := manager.Delete("ws1", "MISSING_SECRET"); !errors.Is(err, securestore.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestRunProviderSetListAndCheckOpenAICommands(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-provider-test"); err != nil {
		t.Fatalf("seed api key secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-provider-test" {
			t.Fatalf("expected auth header from secure storage")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "provider-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), setOut, setErr)
	if setCode != 0 {
		t.Fatalf("provider set exit code %d, stderr=%s", setCode, setErr.String())
	}

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"provider", "list",
		"--workspace", "ws1",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("provider list exit code %d, stderr=%s", listCode, listErr.String())
	}

	var listResponse map[string]any
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	providers, ok := listResponse["providers"].([]any)
	if !ok || len(providers) != 1 {
		t.Fatalf("expected one provider in list response, got %v", listResponse["providers"])
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"provider", "check",
		"--workspace", "ws1",
		"--provider", "openai",
	), checkOut, checkErr)
	if checkCode != 0 {
		t.Fatalf("provider check exit code %d, stderr=%s output=%s", checkCode, checkErr.String(), checkOut.String())
	}

	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if success, ok := checkResponse["success"].(bool); !ok || !success {
		t.Fatalf("expected provider check success=true, got %v", checkResponse["success"])
	}
}

func TestRunProviderCommandsUseClientDefaultWorkspaceWhenOmitted(t *testing.T) {
	defaultWorkspace := "ws-provider-default"
	t.Setenv(cliWorkspaceEnvKey, defaultWorkspace)

	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put(defaultWorkspace, "OPENAI_API_KEY", "sk-provider-test"); err != nil {
		t.Fatalf("seed api key secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-provider-test" {
			t.Fatalf("expected auth header from secure storage")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "provider-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), setOut, setErr)
	if setCode != 0 {
		t.Fatalf("provider set exit code %d, stderr=%s output=%s", setCode, setErr.String(), setOut.String())
	}

	var setResponse transport.ProviderConfigRecord
	if err := json.Unmarshal(setOut.Bytes(), &setResponse); err != nil {
		t.Fatalf("decode set response: %v", err)
	}
	if setResponse.WorkspaceID != defaultWorkspace {
		t.Fatalf("expected provider set workspace %q, got %q", defaultWorkspace, setResponse.WorkspaceID)
	}

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"provider", "list",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("provider list exit code %d, stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}

	var listResponse transport.ProviderListResponse
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResponse.WorkspaceID != defaultWorkspace {
		t.Fatalf("expected provider list workspace %q, got %q", defaultWorkspace, listResponse.WorkspaceID)
	}
	if len(listResponse.Providers) != 1 || listResponse.Providers[0].WorkspaceID != defaultWorkspace {
		t.Fatalf("expected one provider scoped to %q, got %+v", defaultWorkspace, listResponse.Providers)
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"provider", "check",
		"--provider", "openai",
	), checkOut, checkErr)
	if checkCode != 0 {
		t.Fatalf("provider check exit code %d, stderr=%s output=%s", checkCode, checkErr.String(), checkOut.String())
	}

	var checkResponse transport.ProviderCheckResponse
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if checkResponse.WorkspaceID != defaultWorkspace {
		t.Fatalf("expected provider check workspace %q, got %q", defaultWorkspace, checkResponse.WorkspaceID)
	}
	if !checkResponse.Success {
		t.Fatalf("expected provider check success=true, got %+v", checkResponse)
	}
}

func TestRunProviderCheckOllamaWithoutSecret(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)

	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("expected /api/tags, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ollamaServer.Close()

	dbPath := filepath.Join(t.TempDir(), "provider-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	setOut := &bytes.Buffer{}
	setErr := &bytes.Buffer{}
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "ollama",
		"--endpoint", ollamaServer.URL,
	), setOut, setErr)
	if setCode != 0 {
		t.Fatalf("provider set ollama exit code %d, stderr=%s", setCode, setErr.String())
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"provider", "check",
		"--workspace", "ws1",
		"--provider", "ollama",
	), checkOut, checkErr)
	if checkCode != 0 {
		t.Fatalf("provider check ollama exit code %d, stderr=%s output=%s", checkCode, checkErr.String(), checkOut.String())
	}
}

func TestRunProviderCheckFailsWhenSecretMissing(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-provider-test"); err != nil {
		t.Fatalf("seed api key secret: %v", err)
	}

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer openAIServer.Close()

	dbPath := filepath.Join(t.TempDir(), "provider-runtime.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)
	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", openAIServer.URL,
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("expected provider set to succeed")
	}

	if _, err := manager.Delete("ws1", "OPENAI_API_KEY"); err != nil {
		t.Fatalf("delete secret: %v", err)
	}

	checkOut := &bytes.Buffer{}
	checkErr := &bytes.Buffer{}
	checkCode := run(withDaemonArgs(server,
		"provider", "check",
		"--workspace", "ws1",
		"--provider", "openai",
	), checkOut, checkErr)
	if checkCode == 0 {
		t.Fatalf("expected provider check to fail when secret missing")
	}

	var checkResponse map[string]any
	if err := json.Unmarshal(checkOut.Bytes(), &checkResponse); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if success, ok := checkResponse["success"].(bool); !ok || success {
		t.Fatalf("expected provider check success=false, got %v", checkResponse["success"])
	}
}
