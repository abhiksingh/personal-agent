package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"personalagent/runtime/internal/transport"
)

const (
	defaultQueuedTaskPollInterval     = 750 * time.Millisecond
	defaultQueuedTaskMaxWorkers       = 2
	defaultQueuedTaskLeaseDuration    = 30 * time.Second
	defaultQueuedTaskLeaseHeartbeat   = 10 * time.Second
	queuedTaskRunLeaseReclaimedReason = "reclaimed expired queued-task lease"
)

type queuedTaskRunExecutor interface {
	ExecuteQueuedTaskRun(ctx context.Context, runID string, correlationID string) (transport.AgentRunResponse, error)
}

type QueuedTaskRuntimeOptions struct {
	PollInterval   time.Duration
	MaxWorkers     int
	LeaseDuration  time.Duration
	LeaseHeartbeat time.Duration
	Now            func() time.Time
	EventBroker    *transport.EventBroker
}

type queuedTaskClaim struct {
	WorkspaceID string
	TaskID      string
	RunID       string
}

type QueuedTaskRuntime struct {
	db           *sql.DB
	executor     queuedTaskRunExecutor
	pollInterval time.Duration
	maxWorkers   int
	leaseTTL     time.Duration
	leaseBeat    time.Duration
	now          func() time.Time
	eventBroker  *transport.EventBroker

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}

	activeRunCancels map[string]context.CancelFunc
	cancelReasons    map[string]string
}

func NewQueuedTaskRuntime(db *sql.DB, executor queuedTaskRunExecutor, opts QueuedTaskRuntimeOptions) (*QueuedTaskRuntime, error) {
	if db == nil {
		return nil, fmt.Errorf("queued task runtime db is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("queued task runtime executor is required")
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	interval := opts.PollInterval
	if interval <= 0 {
		interval = defaultQueuedTaskPollInterval
	}
	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = defaultQueuedTaskMaxWorkers
	}
	leaseTTL := opts.LeaseDuration
	if leaseTTL <= 0 {
		leaseTTL = defaultQueuedTaskLeaseDuration
	}
	leaseBeat := opts.LeaseHeartbeat
	if leaseBeat <= 0 {
		leaseBeat = defaultQueuedTaskLeaseHeartbeat
	}
	if leaseBeat >= leaseTTL {
		leaseBeat = leaseTTL / 2
		if leaseBeat <= 0 {
			leaseBeat = time.Second
		}
	}
	return &QueuedTaskRuntime{
		db:               db,
		executor:         executor,
		pollInterval:     interval,
		maxWorkers:       maxWorkers,
		leaseTTL:         leaseTTL,
		leaseBeat:        leaseBeat,
		now:              nowFn,
		eventBroker:      opts.EventBroker,
		activeRunCancels: map[string]context.CancelFunc{},
		cancelReasons:    map[string]string{},
	}, nil
}

func (r *QueuedTaskRuntime) CancelQueuedTaskRun(runID string, reason string) bool {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return false
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "cancel requested by control api"
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancelReasons == nil {
		r.cancelReasons = map[string]string{}
	}
	r.cancelReasons[trimmedRunID] = trimmedReason
	if r.activeRunCancels == nil {
		r.activeRunCancels = map[string]context.CancelFunc{}
	}
	cancel := r.activeRunCancels[trimmedRunID]
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (r *QueuedTaskRuntime) Start(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return nil
	}

	runCtx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.running = true
	go r.runLoop(runCtx, done)
	return nil
}

func (r *QueuedTaskRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	r.running = false
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *QueuedTaskRuntime) runLoop(ctx context.Context, done chan struct{}) {
	defer close(done)

	workerCount := r.maxWorkers
	if workerCount <= 0 {
		workerCount = 1
	}

	var workerWG sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			r.runWorkerLoop(ctx)
		}()
	}
	workerWG.Wait()
}

func (r *QueuedTaskRuntime) runWorkerLoop(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		processed, err := r.drainOnce(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		if processed {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *QueuedTaskRuntime) drainOnce(ctx context.Context) (bool, error) {
	claim, found, err := r.claimNextQueuedRun(ctx)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	correlationID := queuedRunCorrelationID(claim.RunID)
	publishTaskRunLifecycleEvent(
		r.eventBroker,
		correlationID,
		claim.WorkspaceID,
		claim.TaskID,
		claim.RunID,
		"running",
		"running",
		taskRunLifecycleSourceQueuedTaskWorker,
		"",
		r.now(),
	)

	execCtx, execCancel := context.WithCancel(ctx)
	r.setActiveRun(claim.RunID, execCancel)
	defer r.clearActiveRun(claim.RunID)
	leaseDone := make(chan error, 1)
	go func() {
		leaseDone <- r.maintainClaimLease(execCtx, claim.RunID)
	}()

	execResponse, execErr := r.executor.ExecuteQueuedTaskRun(execCtx, claim.RunID, correlationID)
	execCancel()
	leaseErr := <-leaseDone
	if leaseErr != nil && execErr == nil {
		execErr = leaseErr
	}
	cancelReason := r.consumeCancelReason(claim.RunID)
	if strings.TrimSpace(cancelReason) != "" {
		markCtx, markCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer markCancel()
		if markErr := r.markClaimedRunCancelled(markCtx, claim.TaskID, claim.RunID, cancelReason); markErr != nil {
			return true, markErr
		}
		publishTaskRunLifecycleEvent(
			r.eventBroker,
			correlationID,
			claim.WorkspaceID,
			claim.TaskID,
			claim.RunID,
			"cancelled",
			"cancelled",
			taskRunLifecycleSourceQueuedTaskWorker,
			cancelReason,
			r.now(),
		)
		return true, nil
	}
	if execErr != nil {
		if errors.Is(execErr, context.Canceled) {
			return true, nil
		}
		errMessage := fmt.Sprintf("queued runtime execution failed: %s", strings.TrimSpace(execErr.Error()))
		markErr := r.markClaimedRunFailed(ctx, claim.TaskID, claim.RunID, errMessage)
		if markErr != nil {
			return true, markErr
		}
		publishTaskRunLifecycleEvent(
			r.eventBroker,
			correlationID,
			claim.WorkspaceID,
			claim.TaskID,
			claim.RunID,
			"failed",
			"failed",
			taskRunLifecycleSourceQueuedTaskWorker,
			errMessage,
			r.now(),
		)
		return true, nil
	}

	taskState := strings.TrimSpace(execResponse.TaskState)
	runState := strings.TrimSpace(execResponse.RunState)
	if taskState == "" || runState == "" {
		resolvedTaskState, resolvedRunState, resolveErr := r.loadTaskRunLifecycleState(ctx, claim.TaskID, claim.RunID)
		if resolveErr != nil {
			return true, resolveErr
		}
		if taskState == "" {
			taskState = resolvedTaskState
		}
		if runState == "" {
			runState = resolvedRunState
		}
	}
	if !isQueuedRuntimeSettledState(taskState) || !isQueuedRuntimeSettledState(runState) {
		errMessage := fmt.Sprintf(
			"queued runtime executor returned unsettled lifecycle state task=%s run=%s",
			taskState,
			runState,
		)
		markErr := r.markClaimedRunFailed(ctx, claim.TaskID, claim.RunID, errMessage)
		if markErr != nil {
			return true, markErr
		}
		publishTaskRunLifecycleEvent(
			r.eventBroker,
			correlationID,
			claim.WorkspaceID,
			claim.TaskID,
			claim.RunID,
			"failed",
			"failed",
			taskRunLifecycleSourceQueuedTaskWorker,
			errMessage,
			r.now(),
		)
		return true, nil
	}

	publishTaskRunLifecycleEvent(
		r.eventBroker,
		correlationID,
		claim.WorkspaceID,
		claim.TaskID,
		claim.RunID,
		taskState,
		runState,
		taskRunLifecycleSourceQueuedTaskWorker,
		"",
		r.now(),
	)

	return true, nil
}

func (r *QueuedTaskRuntime) setActiveRun(runID string, cancel context.CancelFunc) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" || cancel == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeRunCancels == nil {
		r.activeRunCancels = map[string]context.CancelFunc{}
	}
	r.activeRunCancels[trimmedRunID] = cancel
}

func (r *QueuedTaskRuntime) clearActiveRun(runID string) {
	trimmed := strings.TrimSpace(runID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeRunCancels != nil {
		delete(r.activeRunCancels, trimmed)
	}
	delete(r.cancelReasons, trimmed)
}

func (r *QueuedTaskRuntime) consumeCancelReason(runID string) string {
	trimmed := strings.TrimSpace(runID)
	r.mu.Lock()
	defer r.mu.Unlock()
	reason := strings.TrimSpace(r.cancelReasons[trimmed])
	delete(r.cancelReasons, trimmed)
	return reason
}

func (r *QueuedTaskRuntime) claimNextQueuedRun(ctx context.Context) (queuedTaskClaim, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return queuedTaskClaim{}, false, fmt.Errorf("begin queued-run claim tx: %w", err)
	}
	defer tx.Rollback()

	claimState := ""
	claimCutoff := r.now().Add(-1 * r.leaseTTL).Format(time.RFC3339Nano)
	nowText := r.now().Format(time.RFC3339Nano)
	var claim queuedTaskClaim
	err = tx.QueryRowContext(ctx, `
		WITH candidate AS (
			SELECT tr.workspace_id, tr.task_id, tr.id, COALESCE(tr.state, '') AS previous_state
			FROM task_runs tr
			WHERE tr.state = 'queued'
			   OR (tr.state = 'running' AND tr.updated_at <= ?)
			ORDER BY CASE WHEN tr.state = 'queued' THEN 0 ELSE 1 END, tr.created_at ASC
			LIMIT 1
		)
		UPDATE task_runs
		SET state = 'running',
		    started_at = COALESCE(started_at, ?),
		    updated_at = ?
		WHERE id = (SELECT id FROM candidate)
		  AND (
			state = 'queued'
			OR (state = 'running' AND updated_at <= ?)
		  )
		RETURNING
			workspace_id,
			task_id,
			id,
			COALESCE((SELECT previous_state FROM candidate), '')
	`, claimCutoff, nowText, nowText, claimCutoff).Scan(&claim.WorkspaceID, &claim.TaskID, &claim.RunID, &claimState)
	if err != nil {
		if err == sql.ErrNoRows {
			if commitErr := tx.Commit(); commitErr != nil {
				return queuedTaskClaim{}, false, fmt.Errorf("commit empty queued-run claim tx: %w", commitErr)
			}
			return queuedTaskClaim{}, false, nil
		}
		return queuedTaskClaim{}, false, fmt.Errorf("claim queued run %s: %w", claim.RunID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = 'running',
		    updated_at = ?
		WHERE id = ?
	`, nowText, claim.TaskID); err != nil {
		return queuedTaskClaim{}, false, fmt.Errorf("promote queued task %s to running: %w", claim.TaskID, err)
	}
	if strings.EqualFold(strings.TrimSpace(claimState), "running") {
		if _, err := tx.ExecContext(ctx, `
			UPDATE task_runs
			SET last_error = COALESCE(last_error, ?)
			WHERE id = ?
		`, nullableQueuedText(queuedTaskRunLeaseReclaimedReason), claim.RunID); err != nil {
			return queuedTaskClaim{}, false, fmt.Errorf("record queued-run lease reclaim %s: %w", claim.RunID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return queuedTaskClaim{}, false, fmt.Errorf("commit queued-run claim tx: %w", err)
	}
	return claim, true, nil
}

func (r *QueuedTaskRuntime) maintainClaimLease(ctx context.Context, runID string) error {
	beat := r.leaseBeat
	if beat <= 0 {
		return nil
	}
	ticker := time.NewTicker(beat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		heartbeatCtx, heartbeatCancel := context.WithTimeout(context.Background(), 2*time.Second)
		heartbeatErr := r.refreshClaimLease(heartbeatCtx, runID)
		heartbeatCancel()
		if heartbeatErr != nil {
			return heartbeatErr
		}
	}
}

func (r *QueuedTaskRuntime) refreshClaimLease(ctx context.Context, runID string) error {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("refresh queued-run lease: run id is required")
	}
	nowText := r.now().Format(time.RFC3339Nano)
	result, err := r.db.ExecContext(ctx, `
		UPDATE task_runs
		SET updated_at = ?
		WHERE id = ?
		  AND state = 'running'
	`, nowText, trimmedRunID)
	if err != nil {
		return fmt.Errorf("refresh queued-run lease %s: %w", trimmedRunID, err)
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		return nil
	}

	state, stateErr := r.loadTaskRunState(ctx, trimmedRunID)
	if stateErr != nil {
		return fmt.Errorf("refresh queued-run lease %s: lease update not applied and state lookup failed: %w", trimmedRunID, stateErr)
	}
	if isQueuedRuntimeSettledState(state) {
		return nil
	}
	return fmt.Errorf("refresh queued-run lease %s: lease update not applied while state=%s", trimmedRunID, state)
}

func (r *QueuedTaskRuntime) markClaimedRunFailed(ctx context.Context, taskID string, runID string, errMessage string) error {
	nowText := r.now().Format(time.RFC3339Nano)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin queued-run fail tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE task_runs
		SET state = 'failed',
		    finished_at = COALESCE(finished_at, ?),
		    last_error = ?,
		    updated_at = ?
		WHERE id = ?
	`, nowText, nullableQueuedText(errMessage), nowText, strings.TrimSpace(runID)); err != nil {
		return fmt.Errorf("mark task run failed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = 'failed',
		    updated_at = ?
		WHERE id = ?
	`, nowText, strings.TrimSpace(taskID)); err != nil {
		return fmt.Errorf("mark task failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit queued-run fail tx: %w", err)
	}
	return nil
}

func (r *QueuedTaskRuntime) markClaimedRunCancelled(ctx context.Context, taskID string, runID string, reason string) error {
	nowText := r.now().Format(time.RFC3339Nano)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin queued-run cancel tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := applyTaskRunCancellationTransitionsTx(ctx, tx, taskID, runID, reason, nowText); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit queued-run cancel tx: %w", err)
	}
	return nil
}

func (r *QueuedTaskRuntime) loadTaskRunLifecycleState(ctx context.Context, taskID string, runID string) (string, string, error) {
	var (
		taskState string
		runState  string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(t.state, ''), COALESCE(tr.state, '')
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE tr.id = ? AND t.id = ?
	`, strings.TrimSpace(runID), strings.TrimSpace(taskID)).Scan(&taskState, &runState)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("task run not found while resolving lifecycle state: %s", strings.TrimSpace(runID))
		}
		return "", "", fmt.Errorf("load task/run lifecycle state: %w", err)
	}
	return strings.TrimSpace(taskState), strings.TrimSpace(runState), nil
}

func (r *QueuedTaskRuntime) loadTaskRunState(ctx context.Context, runID string) (string, error) {
	var runState string
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(state, '')
		FROM task_runs
		WHERE id = ?
	`, strings.TrimSpace(runID)).Scan(&runState)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task run not found while resolving run state: %s", strings.TrimSpace(runID))
		}
		return "", fmt.Errorf("load task run state: %w", err)
	}
	return strings.TrimSpace(runState), nil
}

func isQueuedRuntimeSettledState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "awaiting_approval", "completed", "failed", "blocked", "cancelled":
		return true
	default:
		return false
	}
}

func queuedRunCorrelationID(runID string) string {
	trimmed := strings.TrimSpace(runID)
	if len(trimmed) > 12 {
		trimmed = trimmed[:12]
	}
	if trimmed == "" {
		trimmed = "unknown"
	}
	return "queue-run-" + trimmed
}

func nullableQueuedText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
