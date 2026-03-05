package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

type queuedRuntimeExecutorStub struct {
	mu             sync.Mutex
	calls          int
	runIDs         []string
	correlationIDs []string
	inFlight       int
	maxInFlight    int
	response       transport.AgentRunResponse
	err            error
	onExecute      func(ctx context.Context, runID string, correlationID string) error
}

func (s *queuedRuntimeExecutorStub) ExecuteQueuedTaskRun(ctx context.Context, runID string, correlationID string) (transport.AgentRunResponse, error) {
	s.mu.Lock()
	s.calls++
	s.runIDs = append(s.runIDs, runID)
	s.correlationIDs = append(s.correlationIDs, correlationID)
	s.inFlight++
	if s.inFlight > s.maxInFlight {
		s.maxInFlight = s.inFlight
	}
	execHook := s.onExecute
	response := s.response
	err := s.err
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.inFlight--
		s.mu.Unlock()
	}()
	if execHook != nil {
		if hookErr := execHook(ctx, runID, correlationID); hookErr != nil {
			return transport.AgentRunResponse{}, hookErr
		}
	}
	return response, err
}

func (s *queuedRuntimeExecutorStub) snapshot() (int, []string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls, append([]string(nil), s.runIDs...), append([]string(nil), s.correlationIDs...)
}

func (s *queuedRuntimeExecutorStub) snapshotMaxInFlight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxInFlight
}

func TestQueuedTaskRuntimeDrainOnceClaimsQueuedRunAndInvokesExecutor(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-1", "run-queued-1")

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-1",
			RunID:     "run-queued-1",
			TaskState: "completed",
			RunState:  "completed",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			now := "2026-02-24T16:00:00Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = 'task-queued-1'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	calls, runIDs, correlationIDs := executor.snapshot()
	if calls != 1 {
		t.Fatalf("expected executor to be called once, got %d", calls)
	}
	if len(runIDs) != 1 || runIDs[0] != "run-queued-1" {
		t.Fatalf("unexpected run ids: %+v", runIDs)
	}
	if len(correlationIDs) != 1 || !strings.HasPrefix(correlationIDs[0], "queue-run-") {
		t.Fatalf("unexpected correlation ids: %+v", correlationIDs)
	}

	var taskState string
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, "task-queued-1").Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if taskState != "completed" {
		t.Fatalf("expected task state completed after execution, got %s", taskState)
	}
	var runState string
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, "run-queued-1").Scan(&runState); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if runState != "completed" {
		t.Fatalf("expected run state completed after execution, got %s", runState)
	}
}

func TestQueuedTaskRuntimeDrainOnceMarksClaimedRunFailedWhenExecutorReturnsError(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-2", "run-queued-2")

	executor := &queuedRuntimeExecutorStub{err: fmt.Errorf("execute failed")}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 5, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	var taskState string
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, "task-queued-2").Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if taskState != "failed" {
		t.Fatalf("expected task state failed after executor error, got %s", taskState)
	}
	var runState string
	var runError sql.NullString
	if err := container.DB.QueryRow(`SELECT state, last_error FROM task_runs WHERE id = ?`, "run-queued-2").Scan(&runState, &runError); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if runState != "failed" {
		t.Fatalf("expected run state failed after executor error, got %s", runState)
	}
	if !runError.Valid || !strings.Contains(runError.String, "execute failed") {
		t.Fatalf("expected last_error to contain executor failure, got %+v", runError)
	}
}

func TestQueuedTaskRuntimeDrainOnceProcessesQueuedRunWhenTaskStateAlreadyRunning(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-mismatch-1", "run-queued-mismatch-1")
	if _, err := container.DB.Exec(`
		UPDATE tasks
		SET state = 'running', updated_at = '2026-02-24T16:06:00Z'
		WHERE id = 'task-queued-mismatch-1'
	`); err != nil {
		t.Fatalf("set mismatched task state: %v", err)
	}

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-mismatch-1",
			RunID:     "run-queued-mismatch-1",
			TaskState: "completed",
			RunState:  "completed",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			now := "2026-02-24T16:06:30Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = 'task-queued-mismatch-1'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 6, 30, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected mismatched queued run to be processed")
	}

	var runState string
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = ?`, "run-queued-mismatch-1").Scan(&runState); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if runState != "completed" {
		t.Fatalf("expected run state completed, got %s", runState)
	}
}

func TestQueuedTaskRuntimeDrainOnceMarksUnsettledExecutorResponseFailed(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-unsettled-1", "run-queued-unsettled-1")

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-unsettled-1",
			RunID:     "run-queued-unsettled-1",
			TaskState: "running",
			RunState:  "running",
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 7, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	var (
		taskState string
		runState  string
		lastError sql.NullString
	)
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, "task-queued-unsettled-1").Scan(&taskState); err != nil {
		t.Fatalf("query task state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state, last_error FROM task_runs WHERE id = ?`, "run-queued-unsettled-1").Scan(&runState, &lastError); err != nil {
		t.Fatalf("query run state: %v", err)
	}
	if taskState != "failed" || runState != "failed" {
		t.Fatalf("expected unsettled executor response to fail task/run, got task=%s run=%s", taskState, runState)
	}
	if !lastError.Valid || !strings.Contains(lastError.String, "unsettled lifecycle state") {
		t.Fatalf("expected unsettled lifecycle failure reason, got %+v", lastError)
	}
}

func TestQueuedTaskRuntimeStartAndStopProcessesQueuedRun(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-3", "run-queued-3")

	executor := &queuedRuntimeExecutorStub{}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		PollInterval: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start queued runtime: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Stop(context.Background())
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		calls, _, _ := executor.snapshot()
		if calls > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	calls, _, _ := executor.snapshot()
	if calls == 0 {
		t.Fatalf("expected queued runtime loop to invoke executor at least once")
	}

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop queued runtime: %v", err)
	}
}

func TestQueuedTaskRuntimeDrainOncePublishesRunningAndCompletedLifecycleEvents(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-4", "run-queued-4")

	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-4",
			RunID:     "run-queued-4",
			TaskState: "completed",
			RunState:  "completed",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			now := "2026-02-24T16:10:00Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = 'task-queued-4'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		EventBroker: broker,
		Now:         func() time.Time { return time.Date(2026, time.February, 24, 16, 10, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	events := readQueuedLifecycleEvents(t, stream, 2)
	assertQueuedLifecycleEvent(t, events[0], "task-queued-4", "run-queued-4", "running", "running", "queue-run-run-queued-4")
	assertQueuedLifecycleEvent(t, events[1], "task-queued-4", "run-queued-4", "completed", "completed", "queue-run-run-queued-4")
}

func TestQueuedTaskRuntimeDrainOncePublishesAwaitingApprovalLifecycleEvent(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-5", "run-queued-5")

	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:            "task-queued-5",
			RunID:             "run-queued-5",
			TaskState:         "awaiting_approval",
			RunState:          "awaiting_approval",
			ApprovalRequired:  true,
			ApprovalRequestID: "apr-queued-5",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			now := "2026-02-24T16:11:00Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'awaiting_approval', updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'awaiting_approval', updated_at = ?
				WHERE id = 'task-queued-5'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		EventBroker: broker,
		Now:         func() time.Time { return time.Date(2026, time.February, 24, 16, 11, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	events := readQueuedLifecycleEvents(t, stream, 2)
	assertQueuedLifecycleEvent(t, events[0], "task-queued-5", "run-queued-5", "running", "running", "queue-run-run-queued-5")
	assertQueuedLifecycleEvent(t, events[1], "task-queued-5", "run-queued-5", "awaiting_approval", "awaiting_approval", "queue-run-run-queued-5")
}

func TestQueuedTaskRuntimeDrainOncePublishesFailedLifecycleEventsOnExecutorError(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-6", "run-queued-6")

	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	executor := &queuedRuntimeExecutorStub{
		err: fmt.Errorf("execute failed"),
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		EventBroker: broker,
		Now:         func() time.Time { return time.Date(2026, time.February, 24, 16, 12, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued run to be processed")
	}

	events := readQueuedLifecycleEvents(t, stream, 2)
	assertQueuedLifecycleEvent(t, events[0], "task-queued-6", "run-queued-6", "running", "running", "queue-run-run-queued-6")
	assertQueuedLifecycleEvent(t, events[1], "task-queued-6", "run-queued-6", "failed", "failed", "queue-run-run-queued-6")
	if lastError := strings.TrimSpace(fmt.Sprintf("%v", events[1].Payload.AsMap()["last_error"])); !strings.Contains(lastError, "execute failed") {
		t.Fatalf("expected failed lifecycle event to include executor error, got %v", events[1].Payload.AsMap()["last_error"])
	}
}

func TestQueuedTaskRuntimeCancelQueuedTaskRunCancelsActiveRunningRun(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-7", "run-queued-7")
	if _, err := container.DB.Exec(`
		INSERT INTO task_steps(id, run_id, step_index, name, status, interaction_level, capability_key, timeout_seconds, retry_max, retry_count, last_error, created_at, updated_at)
		VALUES ('step-queued-7', 'run-queued-7', 0, 'Send update', 'pending', 'manual', 'mail_send', 30, 0, 0, NULL, '2026-02-24T16:13:00Z', '2026-02-24T16:13:00Z')
	`); err != nil {
		t.Fatalf("insert queued runtime step fixture: %v", err)
	}
	if _, err := container.DB.Exec(`
		INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale)
		VALUES ('apr-queued-7-step', 'ws1', NULL, 'step-queued-7', 'GO AHEAD', NULL, NULL, '2026-02-24T16:13:00Z', NULL, NULL)
	`); err != nil {
		t.Fatalf("insert queued runtime step approval fixture: %v", err)
	}
	if _, err := container.DB.Exec(`
		INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale)
		VALUES ('apr-queued-7-run', 'ws1', 'run-queued-7', NULL, 'GO AHEAD', NULL, NULL, '2026-02-24T16:13:00Z', NULL, NULL)
	`); err != nil {
		t.Fatalf("insert queued runtime run approval fixture: %v", err)
	}

	broker := transport.NewEventBroker()
	subID, stream := broker.Subscribe(8)
	t.Cleanup(func() {
		broker.Unsubscribe(subID)
	})

	executor := &queuedRuntimeExecutorStub{
		onExecute: func(ctx context.Context, _ string, _ string) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		EventBroker: broker,
		Now:         func() time.Time { return time.Date(2026, time.February, 24, 16, 13, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	type drainResult struct {
		processed bool
		err       error
	}
	drainDone := make(chan drainResult, 1)
	go func() {
		processed, drainErr := runtime.drainOnce(context.Background())
		drainDone <- drainResult{processed: processed, err: drainErr}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		calls, _, _ := executor.snapshot()
		if calls > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	calls, _, _ := executor.snapshot()
	if calls == 0 {
		t.Fatalf("expected queued runtime to start run execution before cancellation")
	}

	if cancelled := runtime.CancelQueuedTaskRun("run-queued-7", "operator cancelled run"); !cancelled {
		t.Fatalf("expected active run cancellation signal to be accepted")
	}

	select {
	case result := <-drainDone:
		if result.err != nil {
			t.Fatalf("drain once after cancellation: %v", result.err)
		}
		if !result.processed {
			t.Fatalf("expected drainOnce to report processed after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for drainOnce completion after cancellation")
	}

	var (
		taskState            string
		runState             string
		lastError            sql.NullString
		stepStatus           string
		stepLastError        sql.NullString
		stepApprovalDecision sql.NullString
		runApprovalDecision  sql.NullString
	)
	if err := container.DB.QueryRow(`SELECT state FROM tasks WHERE id = ?`, "task-queued-7").Scan(&taskState); err != nil {
		t.Fatalf("query cancelled task state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT state, last_error FROM task_runs WHERE id = ?`, "run-queued-7").Scan(&runState, &lastError); err != nil {
		t.Fatalf("query cancelled run state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT status, last_error FROM task_steps WHERE id = ?`, "step-queued-7").Scan(&stepStatus, &stepLastError); err != nil {
		t.Fatalf("query cancelled step state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, "apr-queued-7-step").Scan(&stepApprovalDecision); err != nil {
		t.Fatalf("query cancelled step approval state: %v", err)
	}
	if err := container.DB.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, "apr-queued-7-run").Scan(&runApprovalDecision); err != nil {
		t.Fatalf("query cancelled run approval state: %v", err)
	}
	if taskState != "cancelled" || runState != "cancelled" {
		t.Fatalf("expected cancelled task/run state, got task=%s run=%s", taskState, runState)
	}
	if !lastError.Valid || !strings.Contains(lastError.String, "operator cancelled run") {
		t.Fatalf("expected cancelled run reason persisted in last_error, got %+v", lastError)
	}
	if stepStatus != "skipped" {
		t.Fatalf("expected cancelled run step status skipped, got %s", stepStatus)
	}
	if !stepLastError.Valid || !strings.Contains(stepLastError.String, "operator cancelled run") {
		t.Fatalf("expected cancelled step reason persisted in last_error, got %+v", stepLastError)
	}
	if !stepApprovalDecision.Valid || strings.ToUpper(strings.TrimSpace(stepApprovalDecision.String)) != "CANCELLED" {
		t.Fatalf("expected step approval decision CANCELLED, got %+v", stepApprovalDecision)
	}
	if !runApprovalDecision.Valid || strings.ToUpper(strings.TrimSpace(runApprovalDecision.String)) != "CANCELLED" {
		t.Fatalf("expected run approval decision CANCELLED, got %+v", runApprovalDecision)
	}

	events := readQueuedLifecycleEvents(t, stream, 2)
	assertQueuedLifecycleEvent(t, events[0], "task-queued-7", "run-queued-7", "running", "running", "queue-run-run-queued-7")
	assertQueuedLifecycleEvent(t, events[1], "task-queued-7", "run-queued-7", "cancelled", "cancelled", "queue-run-run-queued-7")
}

func TestQueuedTaskRuntimeDrainOnceConcurrentContentionProcessesRunOnlyOnce(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-contention-1", "run-queued-contention-1")

	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-contention-1",
			RunID:     "run-queued-contention-1",
			TaskState: "completed",
			RunState:  "completed",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			startedOnce.Do(func() { close(started) })
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			now := "2026-02-24T16:20:00Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = 'task-queued-contention-1'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 20, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	type drainResult struct {
		processed bool
		err       error
	}
	firstDrainDone := make(chan drainResult, 1)
	go func() {
		processed, drainErr := runtime.drainOnce(context.Background())
		firstDrainDone <- drainResult{processed: processed, err: drainErr}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for first drain to claim run")
	}

	secondProcessed, secondErr := runtime.drainOnce(context.Background())
	if secondErr != nil {
		t.Fatalf("second drain once during contention: %v", secondErr)
	}
	if secondProcessed {
		t.Fatalf("expected second drainOnce to skip contested run while first worker owns lease")
	}

	close(release)
	select {
	case result := <-firstDrainDone:
		if result.err != nil {
			t.Fatalf("first drain once result error: %v", result.err)
		}
		if !result.processed {
			t.Fatalf("expected first drainOnce to process run")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for first drain completion")
	}

	calls, runIDs, _ := executor.snapshot()
	if calls != 1 {
		t.Fatalf("expected single execution under contention, got %d", calls)
	}
	if len(runIDs) != 1 || runIDs[0] != "run-queued-contention-1" {
		t.Fatalf("unexpected run IDs after contention execution: %+v", runIDs)
	}
}

func TestQueuedTaskRuntimeClaimNextQueuedRunAtomicUnderContention(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-atomic-1", "run-queued-atomic-1")

	runtime, err := NewQueuedTaskRuntime(container.DB, &queuedRuntimeExecutorStub{}, QueuedTaskRuntimeOptions{
		Now: func() time.Time { return time.Date(2026, time.February, 24, 16, 30, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	type claimResult struct {
		claim queuedTaskClaim
		found bool
		err   error
	}
	const contenders = 8
	results := make(chan claimResult, contenders)
	start := make(chan struct{})
	for i := 0; i < contenders; i++ {
		go func() {
			<-start
			claim, found, claimErr := runtime.claimNextQueuedRun(context.Background())
			results <- claimResult{claim: claim, found: found, err: claimErr}
		}()
	}
	close(start)

	foundCount := 0
	for i := 0; i < contenders; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("claimNextQueuedRun under contention: %v", result.err)
			}
			if result.found {
				foundCount++
				if strings.TrimSpace(result.claim.RunID) != "run-queued-atomic-1" {
					t.Fatalf("expected claimed run run-queued-atomic-1, got %+v", result.claim)
				}
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for atomic claim contender result")
		}
	}
	if foundCount != 1 {
		t.Fatalf("expected exactly one successful claim under contention, got %d", foundCount)
	}

	var runState string
	if err := container.DB.QueryRow(`SELECT state FROM task_runs WHERE id = 'run-queued-atomic-1'`).Scan(&runState); err != nil {
		t.Fatalf("query claimed run state: %v", err)
	}
	if strings.TrimSpace(runState) != "running" {
		t.Fatalf("expected claimed run state running, got %s", runState)
	}
}

func TestQueuedTaskRuntimeStartRespectsMaxWorkersBound(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-workers-1", "run-queued-workers-1")
	for idx := 2; idx <= 4; idx++ {
		taskID := fmt.Sprintf("task-queued-workers-%d", idx)
		runID := fmt.Sprintf("run-queued-workers-%d", idx)
		seedQueuedRuntimeTaskRunOnly(t, container.DB, taskID, runID)
	}

	executor := &queuedRuntimeExecutorStub{
		onExecute: func(ctx context.Context, runID string, _ string) error {
			time.Sleep(75 * time.Millisecond)
			taskID := strings.Replace(runID, "run-", "task-", 1)
			now := time.Now().UTC().Format(time.RFC3339Nano)
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = ?
			`, now, taskID); err != nil {
				return err
			}
			return nil
		},
		response: transport.AgentRunResponse{
			TaskState: "completed",
			RunState:  "completed",
		},
	}

	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		PollInterval: 5 * time.Millisecond,
		MaxWorkers:   2,
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start queued runtime: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Stop(context.Background())
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		calls, _, _ := executor.snapshot()
		if calls >= 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	calls, _, _ := executor.snapshot()
	if calls < 4 {
		t.Fatalf("expected all queued runs to execute, got %d", calls)
	}

	maxInFlight := executor.snapshotMaxInFlight()
	if maxInFlight > 2 {
		t.Fatalf("expected execution concurrency to stay within max workers, saw %d", maxInFlight)
	}
	if maxInFlight < 2 {
		t.Fatalf("expected runtime to utilize two workers under load, saw max concurrency %d", maxInFlight)
	}
}

func TestQueuedTaskRuntimeDrainOnceReclaimsExpiredRunningLease(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedQueuedRuntimeTask(t, container.DB, "task-queued-reclaim-1", "run-queued-reclaim-1")
	if _, err := container.DB.Exec(`
		UPDATE task_runs
		SET state = 'running', started_at = '2026-02-24T16:24:00Z', updated_at = '2026-02-24T16:24:10Z'
		WHERE id = 'run-queued-reclaim-1'
	`); err != nil {
		t.Fatalf("set stale running run: %v", err)
	}
	if _, err := container.DB.Exec(`
		UPDATE tasks
		SET state = 'running', updated_at = '2026-02-24T16:24:10Z'
		WHERE id = 'task-queued-reclaim-1'
	`); err != nil {
		t.Fatalf("set stale running task: %v", err)
	}

	executor := &queuedRuntimeExecutorStub{
		response: transport.AgentRunResponse{
			TaskID:    "task-queued-reclaim-1",
			RunID:     "run-queued-reclaim-1",
			TaskState: "completed",
			RunState:  "completed",
		},
		onExecute: func(ctx context.Context, runID string, _ string) error {
			now := "2026-02-24T16:25:00Z"
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE task_runs
				SET state = 'completed', finished_at = ?, updated_at = ?, last_error = NULL
				WHERE id = ?
			`, now, now, runID); err != nil {
				return err
			}
			if _, err := container.DB.ExecContext(ctx, `
				UPDATE tasks
				SET state = 'completed', updated_at = ?
				WHERE id = 'task-queued-reclaim-1'
			`, now); err != nil {
				return err
			}
			return nil
		},
	}
	runtime, err := NewQueuedTaskRuntime(container.DB, executor, QueuedTaskRuntimeOptions{
		Now:           func() time.Time { return time.Date(2026, time.February, 24, 16, 25, 0, 0, time.UTC) },
		LeaseDuration: 20 * time.Second,
	})
	if err != nil {
		t.Fatalf("new queued runtime: %v", err)
	}

	processed, err := runtime.drainOnce(context.Background())
	if err != nil {
		t.Fatalf("drain once stale running reclaim: %v", err)
	}
	if !processed {
		t.Fatalf("expected stale running lease to be reclaimed and processed")
	}

	calls, runIDs, _ := executor.snapshot()
	if calls != 1 || len(runIDs) != 1 || runIDs[0] != "run-queued-reclaim-1" {
		t.Fatalf("expected stale running run execution, got calls=%d runIDs=%+v", calls, runIDs)
	}
}

func TestQueuedRuntimeSettledStateDoesNotTreatCanceledAsCanonical(t *testing.T) {
	if !isQueuedRuntimeSettledState("cancelled") {
		t.Fatalf("expected cancelled to be a settled queued runtime state")
	}
	if isQueuedRuntimeSettledState("canceled") {
		t.Fatalf("expected canceled to be treated as non-canonical unsettled state")
	}
}

func readQueuedLifecycleEvents(t *testing.T, stream <-chan transport.RealtimeEventEnvelope, expected int) []transport.RealtimeEventEnvelope {
	t.Helper()
	out := make([]transport.RealtimeEventEnvelope, 0, expected)
	deadline := time.After(2 * time.Second)
	for len(out) < expected {
		select {
		case event := <-stream:
			if event.EventType != realtimeEventTypeTaskRunLifecycle {
				continue
			}
			out = append(out, event)
		case <-deadline:
			t.Fatalf("timeout waiting for %d queued lifecycle events, got %d", expected, len(out))
		}
	}
	return out
}

func assertQueuedLifecycleEvent(t *testing.T, event transport.RealtimeEventEnvelope, taskID string, runID string, taskState string, runState string, correlationPrefix string) {
	t.Helper()
	if event.EventType != realtimeEventTypeTaskRunLifecycle {
		t.Fatalf("expected %s event type, got %s", realtimeEventTypeTaskRunLifecycle, event.EventType)
	}
	if !strings.HasPrefix(strings.TrimSpace(event.CorrelationID), strings.TrimSpace(correlationPrefix)) {
		t.Fatalf("expected correlation prefix %s, got %s", correlationPrefix, event.CorrelationID)
	}
	if gotTaskID := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["task_id"])); gotTaskID != taskID {
		t.Fatalf("expected payload task_id %s, got %s", taskID, gotTaskID)
	}
	if gotRunID := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["run_id"])); gotRunID != runID {
		t.Fatalf("expected payload run_id %s, got %s", runID, gotRunID)
	}
	if gotTaskState := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["task_state"])); gotTaskState != taskState {
		t.Fatalf("expected payload task_state %s, got %s", taskState, gotTaskState)
	}
	if gotRunState := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["run_state"])); gotRunState != runState {
		t.Fatalf("expected payload run_state %s, got %s", runState, gotRunState)
	}
}

func seedQueuedRuntimeTask(t *testing.T, db *sql.DB, taskID string, runID string) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at)
		 VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		 VALUES ('actor.requester', 'ws1', 'human', 'Requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		 VALUES ('wp-ws1-actor.requester', 'ws1', 'actor.requester', 'ACTIVE', '2026-02-24T00:00:00Z', '2026-02-24T00:00:00Z')`,
		fmt.Sprintf(`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('%s', 'ws1', 'actor.requester', 'actor.requester', 'Queued task', 'send an email update', 'queued', 0, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, taskID),
		fmt.Sprintf(`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('%s', 'ws1', '%s', 'actor.requester', 'queued', NULL, NULL, NULL, '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, runID, taskID),
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed queued runtime fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}

func seedQueuedRuntimeTaskRunOnly(t *testing.T, db *sql.DB, taskID string, runID string) {
	t.Helper()
	statements := []string{
		fmt.Sprintf(`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at)
		 VALUES ('%s', 'ws1', 'actor.requester', 'actor.requester', 'Queued task', 'send an email update', 'queued', 0, NULL, 'app_chat', '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, taskID),
		fmt.Sprintf(`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at)
		 VALUES ('%s', 'ws1', '%s', 'actor.requester', 'queued', NULL, NULL, NULL, '2026-02-24T00:00:01Z', '2026-02-24T00:00:01Z')`, runID, taskID),
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed queued runtime task/run fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
