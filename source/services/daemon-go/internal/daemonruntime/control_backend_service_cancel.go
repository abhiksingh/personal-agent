package daemonruntime

import (
	"context"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (b *PersistedControlBackend) CancelTask(ctx context.Context, request transport.TaskCancelRequest, correlationID string) (transport.TaskCancelResponse, error) {
	if b.container == nil || b.container.DB == nil {
		return transport.TaskCancelResponse{}, transport.NewTaskControlBackendError("database is not configured")
	}

	record, err := b.lookupTaskRunForCancel(ctx, request)
	if err != nil {
		return transport.TaskCancelResponse{}, err
	}
	if err := b.ensureTaskRunDelegationAllowed(ctx, record); err != nil {
		return transport.TaskCancelResponse{}, err
	}

	previousTaskState := normalizeTaskLifecycleState(record.TaskState)
	previousRunState := normalizeTaskLifecycleState(record.RunState)
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "cancel requested by control api"
	}
	if isTerminalTaskLifecycleState(previousRunState) {
		return transport.TaskCancelResponse{
			WorkspaceID:       record.WorkspaceID,
			TaskID:            record.TaskID,
			RunID:             record.RunID,
			PreviousTaskState: previousTaskState,
			PreviousRunState:  previousRunState,
			TaskState:         previousTaskState,
			RunState:          previousRunState,
			Cancelled:         false,
			AlreadyTerminal:   true,
			Reason:            reason,
			CorrelationID:     correlationID,
		}, nil
	}

	finalRecord := record
	cancelled := false
	if previousRunState == "running" && b.runCanceller != nil {
		if b.runCanceller.CancelQueuedTaskRun(record.RunID, reason) {
			waited, waitErr := b.waitForTaskRunSettlement(ctx, record.RunID, b.cancelSettlementTimeout)
			if waitErr != nil {
				return transport.TaskCancelResponse{}, waitErr
			}
			finalRecord = waited
			cancelled = normalizeTaskLifecycleState(waited.RunState) == "cancelled"
			return transport.TaskCancelResponse{
				WorkspaceID:       finalRecord.WorkspaceID,
				TaskID:            finalRecord.TaskID,
				RunID:             finalRecord.RunID,
				PreviousTaskState: previousTaskState,
				PreviousRunState:  previousRunState,
				TaskState:         normalizeTaskLifecycleState(finalRecord.TaskState),
				RunState:          normalizeTaskLifecycleState(finalRecord.RunState),
				Cancelled:         cancelled,
				AlreadyTerminal:   isTerminalTaskLifecycleState(finalRecord.RunState) && !cancelled,
				Reason:            reason,
				CorrelationID:     correlationID,
			}, nil
		}
	}

	changed, cancelErr := b.cancelTaskRunRows(ctx, record.TaskID, record.RunID, reason)
	if cancelErr != nil {
		return transport.TaskCancelResponse{}, cancelErr
	}
	updated, lookupErr := b.lookupTaskRunForCancel(ctx, transport.TaskCancelRequest{
		RunID:       record.RunID,
		WorkspaceID: record.WorkspaceID,
		TaskID:      record.TaskID,
	})
	if lookupErr != nil {
		return transport.TaskCancelResponse{}, lookupErr
	}
	finalRecord = updated
	cancelled = changed && normalizeTaskLifecycleState(finalRecord.RunState) == "cancelled"

	if changed {
		publishTaskRunLifecycleEvent(
			b.eventBroker,
			correlationID,
			finalRecord.WorkspaceID,
			finalRecord.TaskID,
			finalRecord.RunID,
			normalizeTaskLifecycleState(finalRecord.TaskState),
			normalizeTaskLifecycleState(finalRecord.RunState),
			taskRunLifecycleSourceControlCancel,
			reason,
			time.Now().UTC(),
		)
	}

	return transport.TaskCancelResponse{
		WorkspaceID:       finalRecord.WorkspaceID,
		TaskID:            finalRecord.TaskID,
		RunID:             finalRecord.RunID,
		PreviousTaskState: previousTaskState,
		PreviousRunState:  previousRunState,
		TaskState:         normalizeTaskLifecycleState(finalRecord.TaskState),
		RunState:          normalizeTaskLifecycleState(finalRecord.RunState),
		Cancelled:         cancelled,
		AlreadyTerminal:   isTerminalTaskLifecycleState(finalRecord.RunState) && !cancelled,
		Reason:            reason,
		CorrelationID:     correlationID,
	}, nil
}
