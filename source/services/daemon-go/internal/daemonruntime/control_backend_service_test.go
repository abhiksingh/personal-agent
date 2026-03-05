package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

type controlBackendAgentStub struct {
	approveCalled  bool
	approveRequest transport.AgentApproveRequest
	approveErr     error
}

type queuedRunCancellerStub struct {
	called     bool
	runID      string
	reason     string
	returnedOK bool
	onCancel   func(runID string, reason string)
}

func (s *queuedRunCancellerStub) CancelQueuedTaskRun(runID string, reason string) bool {
	s.called = true
	s.runID = strings.TrimSpace(runID)
	s.reason = strings.TrimSpace(reason)
	if s.onCancel != nil {
		s.onCancel(s.runID, s.reason)
	}
	return s.returnedOK
}

func (s *controlBackendAgentStub) RunAgent(context.Context, transport.AgentRunRequest) (transport.AgentRunResponse, error) {
	return transport.AgentRunResponse{}, nil
}

func (s *controlBackendAgentStub) ApproveAgent(_ context.Context, request transport.AgentApproveRequest) (transport.AgentRunResponse, error) {
	s.approveCalled = true
	s.approveRequest = request
	if s.approveErr != nil {
		return transport.AgentRunResponse{}, s.approveErr
	}
	return transport.AgentRunResponse{
		TaskID:    "task-approve",
		RunID:     "run-approve",
		TaskState: "completed",
		RunState:  "completed",
	}, nil
}

func TestPersistedControlBackendSubmitTaskAndStatusPersistSQLite(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	backend, err := NewPersistedControlBackend(container, agent, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	submitResponse, err := backend.SubmitTask(context.Background(), transport.SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.requester",
		Title:                   "Persisted task",
		Description:             "task body",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if submitResponse.TaskID == "" || submitResponse.RunID == "" {
		t.Fatalf("expected non-empty task/run ids, got %+v", submitResponse)
	}
	if submitResponse.State != "queued" {
		t.Fatalf("expected queued state, got %s", submitResponse.State)
	}

	statusResponse, err := backend.TaskStatus(context.Background(), submitResponse.TaskID, "corr-status")
	if err != nil {
		t.Fatalf("task status: %v", err)
	}
	if statusResponse.TaskID != submitResponse.TaskID {
		t.Fatalf("expected task id %s, got %s", submitResponse.TaskID, statusResponse.TaskID)
	}
	if statusResponse.State != "queued" {
		t.Fatalf("expected status state queued, got %s", statusResponse.State)
	}

	var taskState string
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, submitResponse.TaskID).Scan(&taskState); err != nil {
		t.Fatalf("query persisted task: %v", err)
	}
	if taskState != "queued" {
		t.Fatalf("expected persisted task state queued, got %s", taskState)
	}

	var runState string
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, submitResponse.RunID).Scan(&runState); err != nil {
		t.Fatalf("query persisted run: %v", err)
	}
	if runState != "queued" {
		t.Fatalf("expected persisted run state queued, got %s", runState)
	}
}

func TestPersistedControlBackendCapabilitySmokeUsesCanonicalDefaults(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	backend, err := NewPersistedControlBackend(container, agent, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	response, err := backend.CapabilitySmoke(context.Background(), "corr-smoke-defaults")
	if err != nil {
		t.Fatalf("capability smoke: %v", err)
	}
	if !reflect.DeepEqual(response.Channels, transport.DefaultCapabilitySmokeChannels()) {
		t.Fatalf("expected canonical channels %+v, got %+v", transport.DefaultCapabilitySmokeChannels(), response.Channels)
	}
	if !reflect.DeepEqual(response.Connectors, transport.DefaultCapabilitySmokeConnectors()) {
		t.Fatalf("expected canonical connectors %+v, got %+v", transport.DefaultCapabilitySmokeConnectors(), response.Connectors)
	}
}

func TestPersistedControlBackendSubmitTaskPublishesQueuedLifecycleEvent(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	backend, err := NewPersistedControlBackend(container, agent, broker)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	submitResponse, err := backend.SubmitTask(context.Background(), transport.SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.requester",
		Title:                   "Persisted lifecycle task",
		Description:             "task body",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	lifecycleEvent := readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle)
	if gotCorrelation := strings.TrimSpace(lifecycleEvent.CorrelationID); gotCorrelation != "corr-submit" {
		t.Fatalf("expected lifecycle correlation corr-submit, got %s", gotCorrelation)
	}
	if gotRunID := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["run_id"])); gotRunID != submitResponse.RunID {
		t.Fatalf("expected lifecycle run_id %s, got %s", submitResponse.RunID, gotRunID)
	}
	if gotTaskID := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["task_id"])); gotTaskID != submitResponse.TaskID {
		t.Fatalf("expected lifecycle task_id %s, got %s", submitResponse.TaskID, gotTaskID)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued lifecycle_state, got %s", gotState)
	}
	if gotRunState := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["run_state"])); gotRunState != "queued" {
		t.Fatalf("expected queued run_state, got %s", gotRunState)
	}
}

func TestPersistedControlBackendSubmitTaskRejectsCrossPrincipalWithoutDelegation(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	backend, err := NewPersistedControlBackend(container, agent, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	_, err = backend.SubmitTask(context.Background(), transport.SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.subject",
		Title:                   "Persisted task",
		Description:             "task body",
	}, "corr-submit")
	if err == nil {
		t.Fatalf("expected cross-principal submit to be denied without delegation")
	}
}

func TestPersistedControlBackendTaskControlErrorsExposeTypedMapping(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	backend, err := NewPersistedControlBackend(container, agent, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	_, err = backend.CancelTask(context.Background(), transport.TaskCancelRequest{}, "corr-cancel-missing")
	assertPersistedTaskControlDomainError(t, err, http.StatusBadRequest, "missing_required_field", "task_control_validation")

	_, err = backend.TaskStatus(context.Background(), "task-missing", "corr-status-missing")
	assertPersistedTaskControlDomainError(t, err, http.StatusNotFound, "resource_not_found", "task_control_lookup")

	submitResponse, err := backend.SubmitTask(context.Background(), transport.SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.requester",
		Title:                   "Persisted domain error task",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	_, err = backend.RetryTask(context.Background(), transport.TaskRetryRequest{
		RunID: submitResponse.RunID,
	}, "corr-retry-conflict")
	assertPersistedTaskControlDomainError(t, err, http.StatusConflict, "resource_conflict", "task_control_state_conflict")
}

func TestPersistedControlBackendCancelTaskRunQueuedState(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	agent := &controlBackendAgentStub{}
	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	backend, err := NewPersistedControlBackend(container, agent, broker)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	submitResponse, err := backend.SubmitTask(context.Background(), transport.SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.requester",
		Title:                   "Cancel queued task",
		Description:             "send an email update",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	cancelResponse, err := backend.CancelTask(context.Background(), transport.TaskCancelRequest{
		RunID:  submitResponse.RunID,
		Reason: "manual cancel",
	}, "corr-cancel")
	if err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if !cancelResponse.Cancelled {
		t.Fatalf("expected cancelled response, got %+v", cancelResponse)
	}
	if cancelResponse.RunState != "cancelled" || cancelResponse.TaskState != "cancelled" {
		t.Fatalf("expected cancelled task/run states, got task=%s run=%s", cancelResponse.TaskState, cancelResponse.RunState)
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(cancelResponse.Reason)), "manual cancel") {
		t.Fatalf("expected cancellation reason to round-trip, got %q", cancelResponse.Reason)
	}

	var (
		taskState string
		runState  string
		lastError sql.NullString
	)
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, submitResponse.TaskID).Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state, last_error FROM task_runs WHERE id = ?`, submitResponse.RunID).Scan(&runState, &lastError); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if taskState != "cancelled" || runState != "cancelled" {
		t.Fatalf("expected persisted cancelled task/run state, got task=%s run=%s", taskState, runState)
	}
	if !lastError.Valid || !strings.Contains(lastError.String, "manual cancel") {
		t.Fatalf("expected persisted cancellation reason in task_runs.last_error, got %+v", lastError)
	}

	_ = readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle) // submit lifecycle
	cancelLifecycle := readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle)
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", cancelLifecycle.Payload.AsMap()["lifecycle_state"])); gotState != "cancelled" {
		t.Fatalf("expected cancelled lifecycle event state, got %s", gotState)
	}
	if strings.TrimSpace(cancelLifecycle.CorrelationID) != "corr-cancel" {
		t.Fatalf("expected cancel lifecycle correlation corr-cancel, got %s", cancelLifecycle.CorrelationID)
	}
}

func assertPersistedTaskControlDomainError(t *testing.T, err error, statusCode int, code string, category string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	var domainErr transport.TransportDomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected TransportDomainError, got %T", err)
	}
	if got := domainErr.TransportStatusCode(); got != statusCode {
		t.Fatalf("expected status %d, got %d", statusCode, got)
	}
	if got := strings.TrimSpace(domainErr.TransportErrorCode()); got != code {
		t.Fatalf("expected code %q, got %q", code, got)
	}
	details, ok := domainErr.TransportErrorDetails().(map[string]any)
	if !ok {
		t.Fatalf("expected error details map, got %T", domainErr.TransportErrorDetails())
	}
	if got := strings.TrimSpace(fmt.Sprintf("%v", details["category"])); got != category {
		t.Fatalf("expected details.category %q, got %q", category, got)
	}
}

func TestPersistedControlBackendCancelRunningTaskRunViaRuntimeCanceller(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendRunningTaskFixture(t, container.DB)

	agent := &controlBackendAgentStub{}
	backend, err := NewPersistedControlBackend(container, agent, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	canceller := &queuedRunCancellerStub{
		returnedOK: true,
		onCancel: func(runID string, reason string) {
			time.Sleep(50 * time.Millisecond)
			now := "2026-02-26T13:45:00Z"
			_, _ = container.DB.Exec(`
				UPDATE task_runs
				SET state = 'cancelled', finished_at = ?, updated_at = ?, last_error = ?
				WHERE id = ?
			`, now, now, reason, runID)
			_, _ = container.DB.Exec(`
				UPDATE tasks
				SET state = 'cancelled', updated_at = ?
				WHERE id = 'task-running-1'
			`, now)
		},
	}
	backend.SetQueuedTaskRunCanceller(canceller)

	cancelResponse, err := backend.CancelTask(context.Background(), transport.TaskCancelRequest{
		RunID:  "run-running-1",
		Reason: "running cancel test",
	}, "corr-cancel-running")
	if err != nil {
		t.Fatalf("cancel running task: %v", err)
	}
	if !canceller.called {
		t.Fatalf("expected running cancel request to invoke queued runtime canceller")
	}
	if canceller.runID != "run-running-1" {
		t.Fatalf("expected canceller run id run-running-1, got %s", canceller.runID)
	}
	if !cancelResponse.Cancelled || cancelResponse.RunState != "cancelled" {
		t.Fatalf("expected running cancel response to settle cancelled state, got %+v", cancelResponse)
	}
}

func TestPersistedControlBackendCancelTaskAlreadyTerminalRun(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendFailedTaskFixture(t, container.DB)

	backend, err := NewPersistedControlBackend(container, &controlBackendAgentStub{}, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	response, err := backend.CancelTask(context.Background(), transport.TaskCancelRequest{
		RunID:  "run-failed-1",
		Reason: "cancel terminal run",
	}, "corr-cancel-terminal")
	if err != nil {
		t.Fatalf("cancel terminal run: %v", err)
	}
	if response.Cancelled {
		t.Fatalf("expected terminal run cancel to report cancelled=false, got %+v", response)
	}
	if !response.AlreadyTerminal {
		t.Fatalf("expected terminal run cancel to report already_terminal=true, got %+v", response)
	}
	if response.RunState != "failed" || response.TaskState != "failed" {
		t.Fatalf("expected failed terminal states to remain unchanged, got task=%s run=%s", response.TaskState, response.RunState)
	}
}

func TestPersistedControlBackendCancelRunningTaskRunSettlementTimeout(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendRunningTaskFixture(t, container.DB)

	backend, err := NewPersistedControlBackend(container, &controlBackendAgentStub{}, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}
	backend.cancelSettlementTimeout = 40 * time.Millisecond
	backend.cancelSettlementPollInterval = 5 * time.Millisecond

	canceller := &queuedRunCancellerStub{returnedOK: true}
	backend.SetQueuedTaskRunCanceller(canceller)

	_, err = backend.CancelTask(context.Background(), transport.TaskCancelRequest{
		RunID:  "run-running-1",
		Reason: "running cancel timeout",
	}, "corr-cancel-timeout")
	if err == nil {
		t.Fatalf("expected timeout while waiting for running cancellation settlement")
	}
	if !strings.Contains(err.Error(), "timed out waiting for task run cancellation settlement") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if !canceller.called {
		t.Fatalf("expected running cancel timeout path to invoke queued runtime canceller")
	}
}

func TestPersistedControlBackendCancelRunningTaskRunSettlementContextCanceled(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendRunningTaskFixture(t, container.DB)

	backend, err := NewPersistedControlBackend(container, &controlBackendAgentStub{}, nil)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	canceller := &queuedRunCancellerStub{
		returnedOK: true,
		onCancel: func(_ string, _ string) {
			cancel()
		},
	}
	backend.SetQueuedTaskRunCanceller(canceller)

	_, err = backend.CancelTask(cancelCtx, transport.TaskCancelRequest{
		RunID:  "run-running-1",
		Reason: "running cancel context",
	}, "corr-cancel-context")
	if err == nil {
		t.Fatalf("expected context cancellation while waiting for running cancellation settlement")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	if !canceller.called {
		t.Fatalf("expected running cancel context path to invoke queued runtime canceller")
	}
}

func TestPersistedControlBackendRetryTaskFromFailedStateCreatesQueuedRun(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendFailedTaskFixture(t, container.DB)

	agent := &controlBackendAgentStub{}
	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	backend, err := NewPersistedControlBackend(container, agent, broker)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	retryResponse, err := backend.RetryTask(context.Background(), transport.TaskRetryRequest{
		RunID:  "run-failed-1",
		Reason: "manual retry",
	}, "corr-retry")
	if err != nil {
		t.Fatalf("retry task: %v", err)
	}
	if !retryResponse.Retried {
		t.Fatalf("expected retried response, got %+v", retryResponse)
	}
	if retryResponse.PreviousRunID != "run-failed-1" {
		t.Fatalf("expected previous run id run-failed-1, got %s", retryResponse.PreviousRunID)
	}
	if retryResponse.RunID == "" || retryResponse.RunID == "run-failed-1" {
		t.Fatalf("expected new run id, got %s", retryResponse.RunID)
	}
	if retryResponse.TaskState != "queued" || retryResponse.RunState != "queued" {
		t.Fatalf("expected queued retry response states, got task=%s run=%s", retryResponse.TaskState, retryResponse.RunState)
	}
	if !retryResponse.Actions.CanCancel || retryResponse.Actions.CanRetry || !retryResponse.Actions.CanRequeue {
		t.Fatalf("unexpected retry action availability: %+v", retryResponse.Actions)
	}

	var (
		taskState string
		oldState  string
		newState  string
	)
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = 'task-failed-1'`).Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = 'run-failed-1'`).Scan(&oldState); err != nil {
		t.Fatalf("query old run state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, retryResponse.RunID).Scan(&newState); err != nil {
		t.Fatalf("query new run state: %v", err)
	}
	if taskState != "queued" || oldState != "failed" || newState != "queued" {
		t.Fatalf("unexpected persisted retry states task=%s old=%s new=%s", taskState, oldState, newState)
	}

	var auditCount int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE event_type = 'TASK_RUN_RETRIED'`).Scan(&auditCount); err != nil {
		t.Fatalf("query retry audit count: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one retry audit entry, got %d", auditCount)
	}

	retryLifecycle := readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle)
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", retryLifecycle.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued lifecycle state for retry, got %s", gotState)
	}
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", retryLifecycle.Payload.AsMap()["lifecycle_source"])); gotSource != taskRunLifecycleSourceControlRetry {
		t.Fatalf("expected retry lifecycle source %s, got %s", taskRunLifecycleSourceControlRetry, gotSource)
	}
	if strings.TrimSpace(retryLifecycle.CorrelationID) != "corr-retry" {
		t.Fatalf("expected retry lifecycle correlation corr-retry, got %s", retryLifecycle.CorrelationID)
	}
}

func TestPersistedControlBackendRequeueTaskFromBlockedStateCreatesQueuedRun(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedControlBackendBlockedTaskFixture(t, container.DB)

	agent := &controlBackendAgentStub{}
	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	backend, err := NewPersistedControlBackend(container, agent, broker)
	if err != nil {
		t.Fatalf("new persisted control backend: %v", err)
	}

	requeueResponse, err := backend.RequeueTask(context.Background(), transport.TaskRequeueRequest{
		RunID:  "run-blocked-1",
		Reason: "manual requeue",
	}, "corr-requeue")
	if err != nil {
		t.Fatalf("requeue task: %v", err)
	}
	if !requeueResponse.Requeued {
		t.Fatalf("expected requeued response, got %+v", requeueResponse)
	}
	if requeueResponse.PreviousRunID != "run-blocked-1" {
		t.Fatalf("expected previous run id run-blocked-1, got %s", requeueResponse.PreviousRunID)
	}
	if requeueResponse.RunID == "" || requeueResponse.RunID == "run-blocked-1" {
		t.Fatalf("expected new run id, got %s", requeueResponse.RunID)
	}
	if requeueResponse.TaskState != "queued" || requeueResponse.RunState != "queued" {
		t.Fatalf("expected queued requeue response states, got task=%s run=%s", requeueResponse.TaskState, requeueResponse.RunState)
	}
	if !requeueResponse.Actions.CanCancel || requeueResponse.Actions.CanRetry || !requeueResponse.Actions.CanRequeue {
		t.Fatalf("unexpected requeue action availability: %+v", requeueResponse.Actions)
	}

	var (
		taskState        string
		previousRun      string
		newRun           string
		stepStatus       string
		approvalDecision sql.NullString
	)
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = 'task-blocked-1'`).Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = 'run-blocked-1'`).Scan(&previousRun); err != nil {
		t.Fatalf("query previous run state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, requeueResponse.RunID).Scan(&newRun); err != nil {
		t.Fatalf("query new run state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT status FROM task_steps WHERE id = 'step-blocked-1'`).Scan(&stepStatus); err != nil {
		t.Fatalf("query blocked step status: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT decision FROM approval_requests WHERE id = 'apr-blocked-1'`).Scan(&approvalDecision); err != nil {
		t.Fatalf("query approval decision: %v", err)
	}
	if taskState != "queued" || previousRun != "cancelled" || newRun != "queued" {
		t.Fatalf("unexpected persisted requeue states task=%s previous=%s new=%s", taskState, previousRun, newRun)
	}
	if stepStatus != "skipped" {
		t.Fatalf("expected blocked step status skipped after requeue, got %s", stepStatus)
	}
	if !approvalDecision.Valid || strings.ToUpper(strings.TrimSpace(approvalDecision.String)) != "CANCELLED" {
		t.Fatalf("expected blocked approval decision CANCELLED after requeue, got %+v", approvalDecision)
	}

	var auditCount int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE event_type = 'TASK_RUN_REQUEUED'`).Scan(&auditCount); err != nil {
		t.Fatalf("query requeue audit count: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one requeue audit entry, got %d", auditCount)
	}

	cancelLifecycle := readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle)
	queuedLifecycle := readPersistedControlEventByType(t, stream, realtimeEventTypeTaskRunLifecycle)
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", cancelLifecycle.Payload.AsMap()["lifecycle_source"])); gotSource != taskRunLifecycleSourceControlRequeue {
		t.Fatalf("expected requeue cancellation lifecycle source %s, got %s", taskRunLifecycleSourceControlRequeue, gotSource)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", cancelLifecycle.Payload.AsMap()["lifecycle_state"])); gotState != "cancelled" {
		t.Fatalf("expected cancelled lifecycle state for prior run, got %s", gotState)
	}
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", queuedLifecycle.Payload.AsMap()["lifecycle_source"])); gotSource != taskRunLifecycleSourceControlRequeue {
		t.Fatalf("expected requeue queued lifecycle source %s, got %s", taskRunLifecycleSourceControlRequeue, gotSource)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", queuedLifecycle.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued lifecycle state for new run, got %s", gotState)
	}
	if strings.TrimSpace(queuedLifecycle.CorrelationID) != "corr-requeue" {
		t.Fatalf("expected requeue lifecycle correlation corr-requeue, got %s", queuedLifecycle.CorrelationID)
	}
}

func TestTaskLifecycleStateNormalizationNoLongerAliasesCanceled(t *testing.T) {
	if got := normalizeTaskLifecycleState(" canceled "); got != "canceled" {
		t.Fatalf("expected canceled to remain canceled, got %q", got)
	}
	if isTerminalTaskLifecycleState("canceled") {
		t.Fatalf("expected canceled not to be treated as canonical terminal task state")
	}
	if !isTerminalTaskLifecycleState("cancelled") {
		t.Fatalf("expected cancelled to remain canonical terminal task state")
	}
}

func seedControlBackendRunningTaskFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-running-1', 'ws1', 'actor.requester', 'actor.requester', 'Running task', 'running body', 'running', 1, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-running-1', 'ws1', 'task-running-1', 'actor.requester', 'running', '2026-02-24T00:00:01Z', NULL, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed running fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}

func seedControlBackendFailedTaskFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-failed-1', 'ws1', 'actor.requester', 'actor.requester', 'Failed task', 'failed body', 'failed', 1, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-failed-1', 'ws1', 'task-failed-1', 'actor.requester', 'failed', '2026-02-24T00:00:01Z', '2026-02-24T00:00:02Z', 'step failed', '2026-02-24T00:00:01Z', '2026-02-24T00:00:02Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed failed fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}

func seedControlBackendBlockedTaskFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('task-blocked-1', 'ws1', 'actor.requester', 'actor.requester', 'Blocked task', 'blocked body', 'awaiting_approval', 1, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('run-blocked-1', 'ws1', 'task-blocked-1', 'actor.requester', 'awaiting_approval', '2026-02-24T00:00:01Z', NULL, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		 VALUES ('step-blocked-1', 'run-blocked-1', 0, 'Delete file', 'pending', 'manual', 'finder_delete', 30, 0, 0, '', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`,
		`INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale)
		 VALUES ('apr-blocked-1', 'ws1', NULL, 'step-blocked-1', 'GO AHEAD', NULL, NULL, '2026-02-24T00:00:02Z', NULL, NULL)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed blocked fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}

func readPersistedControlEventByType(t *testing.T, stream <-chan transport.RealtimeEventEnvelope, eventType string) transport.RealtimeEventEnvelope {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-stream:
			if strings.TrimSpace(event.EventType) == strings.TrimSpace(eventType) {
				return event
			}
		case <-timeout:
			t.Fatalf("timeout waiting for event type %s", eventType)
		}
	}
}
