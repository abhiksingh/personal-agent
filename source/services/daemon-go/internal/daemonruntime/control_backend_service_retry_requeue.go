package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (b *PersistedControlBackend) RetryTask(ctx context.Context, request transport.TaskRetryRequest, correlationID string) (transport.TaskRetryResponse, error) {
	if b.container == nil || b.container.DB == nil {
		return transport.TaskRetryResponse{}, transport.NewTaskControlBackendError("database is not configured")
	}

	record, err := b.lookupTaskRunForCancel(ctx, transport.TaskCancelRequest{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		TaskID:      strings.TrimSpace(request.TaskID),
		RunID:       strings.TrimSpace(request.RunID),
	})
	if err != nil {
		return transport.TaskRetryResponse{}, err
	}
	if err := b.ensureTaskRunDelegationAllowed(ctx, record); err != nil {
		return transport.TaskRetryResponse{}, err
	}

	previousTaskState := normalizeTaskLifecycleState(record.TaskState)
	previousRunState := normalizeTaskLifecycleState(record.RunState)
	if previousRunState != "failed" && previousRunState != "cancelled" {
		return transport.TaskRetryResponse{}, transport.NewTaskControlStateConflictError(fmt.Sprintf("task run state %q is not retryable", previousRunState))
	}

	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "retry requested by control api"
	}
	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	newRunID, err := controlBackendRandomID()
	if err != nil {
		return transport.TaskRetryResponse{}, err
	}

	tx, err := b.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.TaskRetryResponse{}, fmt.Errorf("begin retry tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_runs(
			id, workspace_id, task_id, acting_as_actor_id,
			state, started_at, finished_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'queued', NULL, NULL, NULL, ?, ?)
	`, newRunID, record.WorkspaceID, record.TaskID, record.ActingAsActorID, nowText, nowText); err != nil {
		return transport.TaskRetryResponse{}, fmt.Errorf("insert retry task run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = 'queued', updated_at = ?
		WHERE id = ?
	`, nowText, record.TaskID); err != nil {
		return transport.TaskRetryResponse{}, fmt.Errorf("mark task queued for retry: %w", err)
	}

	if err := b.appendControlAuditEntryTx(
		ctx,
		tx,
		record.WorkspaceID,
		newRunID,
		"TASK_RUN_RETRIED",
		record.RequestedByActorID,
		record.ActingAsActorID,
		strings.TrimSpace(correlationID),
		map[string]any{
			"task_id":             record.TaskID,
			"previous_run_id":     record.RunID,
			"run_id":              newRunID,
			"previous_task_state": previousTaskState,
			"previous_run_state":  previousRunState,
			"reason":              reason,
		},
		nowText,
	); err != nil {
		return transport.TaskRetryResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return transport.TaskRetryResponse{}, fmt.Errorf("commit retry tx: %w", err)
	}

	publishTaskRunLifecycleEvent(
		b.eventBroker,
		correlationID,
		record.WorkspaceID,
		record.TaskID,
		newRunID,
		"queued",
		"queued",
		taskRunLifecycleSourceControlRetry,
		reason,
		now,
	)

	return transport.TaskRetryResponse{
		WorkspaceID:       record.WorkspaceID,
		TaskID:            record.TaskID,
		PreviousRunID:     record.RunID,
		RunID:             newRunID,
		PreviousTaskState: previousTaskState,
		PreviousRunState:  previousRunState,
		TaskState:         "queued",
		RunState:          "queued",
		Retried:           true,
		Reason:            reason,
		Actions:           transport.ResolveTaskRunActionAvailability("queued", "queued"),
		CorrelationID:     correlationID,
	}, nil
}

func (b *PersistedControlBackend) RequeueTask(ctx context.Context, request transport.TaskRequeueRequest, correlationID string) (transport.TaskRequeueResponse, error) {
	if b.container == nil || b.container.DB == nil {
		return transport.TaskRequeueResponse{}, transport.NewTaskControlBackendError("database is not configured")
	}

	record, err := b.lookupTaskRunForCancel(ctx, transport.TaskCancelRequest{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		TaskID:      strings.TrimSpace(request.TaskID),
		RunID:       strings.TrimSpace(request.RunID),
	})
	if err != nil {
		return transport.TaskRequeueResponse{}, err
	}
	if err := b.ensureTaskRunDelegationAllowed(ctx, record); err != nil {
		return transport.TaskRequeueResponse{}, err
	}

	previousTaskState := normalizeTaskLifecycleState(record.TaskState)
	previousRunState := normalizeTaskLifecycleState(record.RunState)
	switch previousRunState {
	case "queued", "planning", "awaiting_approval", "blocked":
	default:
		return transport.TaskRequeueResponse{}, transport.NewTaskControlStateConflictError(fmt.Sprintf("task run state %q is not requeueable", previousRunState))
	}

	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "requeue requested by control api"
	}

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	newRunID, err := controlBackendRandomID()
	if err != nil {
		return transport.TaskRequeueResponse{}, err
	}

	tx, err := b.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.TaskRequeueResponse{}, fmt.Errorf("begin requeue tx: %w", err)
	}
	defer tx.Rollback()

	cancelledPreviousRun, err := b.cancelTaskRunRowsTx(ctx, tx, record.TaskID, record.RunID, reason, nowText)
	if err != nil {
		return transport.TaskRequeueResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = 'queued', updated_at = ?
		WHERE id = ?
	`, nowText, record.TaskID); err != nil {
		return transport.TaskRequeueResponse{}, fmt.Errorf("mark task queued for requeue: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_runs(
			id, workspace_id, task_id, acting_as_actor_id,
			state, started_at, finished_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'queued', NULL, NULL, NULL, ?, ?)
	`, newRunID, record.WorkspaceID, record.TaskID, record.ActingAsActorID, nowText, nowText); err != nil {
		return transport.TaskRequeueResponse{}, fmt.Errorf("insert requeue task run: %w", err)
	}
	if err := b.appendControlAuditEntryTx(
		ctx,
		tx,
		record.WorkspaceID,
		newRunID,
		"TASK_RUN_REQUEUED",
		record.RequestedByActorID,
		record.ActingAsActorID,
		strings.TrimSpace(correlationID),
		map[string]any{
			"task_id":             record.TaskID,
			"previous_run_id":     record.RunID,
			"run_id":              newRunID,
			"previous_task_state": previousTaskState,
			"previous_run_state":  previousRunState,
			"cancelled_previous":  cancelledPreviousRun,
			"reason":              reason,
		},
		nowText,
	); err != nil {
		return transport.TaskRequeueResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return transport.TaskRequeueResponse{}, fmt.Errorf("commit requeue tx: %w", err)
	}

	if cancelledPreviousRun {
		publishTaskRunLifecycleEvent(
			b.eventBroker,
			correlationID,
			record.WorkspaceID,
			record.TaskID,
			record.RunID,
			"cancelled",
			"cancelled",
			taskRunLifecycleSourceControlRequeue,
			reason,
			now,
		)
	}
	publishTaskRunLifecycleEvent(
		b.eventBroker,
		correlationID,
		record.WorkspaceID,
		record.TaskID,
		newRunID,
		"queued",
		"queued",
		taskRunLifecycleSourceControlRequeue,
		reason,
		now,
	)

	return transport.TaskRequeueResponse{
		WorkspaceID:       record.WorkspaceID,
		TaskID:            record.TaskID,
		PreviousRunID:     record.RunID,
		RunID:             newRunID,
		PreviousTaskState: previousTaskState,
		PreviousRunState:  previousRunState,
		TaskState:         "queued",
		RunState:          "queued",
		Requeued:          true,
		Reason:            reason,
		Actions:           transport.ResolveTaskRunActionAvailability("queued", "queued"),
		CorrelationID:     correlationID,
	}, nil
}
