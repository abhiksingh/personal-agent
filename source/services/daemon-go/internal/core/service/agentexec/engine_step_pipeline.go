package agentexec

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	voicehandoffservice "personalagent/runtime/internal/core/service/voicehandoff"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type stepExecutionFailureMode int

const (
	stepExecutionFailureReturnError stepExecutionFailureMode = iota
	stepExecutionFailureReturnResult
)

type stepExecutionPipelineOptions struct {
	WorkspaceID               string
	TaskID                    string
	RunID                     string
	RequestedByActorID        string
	SubjectActorID            string
	ActingAsActorID           string
	CorrelationID             string
	PreferredAdapterID        string
	SourceChannelForExecution string
	ExecutionOrigin           types.ExecutionOrigin
	InAppApprovalConfirmed    bool
	ApprovalPhrase            string
	FailureMode               stepExecutionFailureMode
	PhasePrefix               string
}

func (o stepExecutionPipelineOptions) phase(name string) string {
	prefix := strings.TrimSpace(o.PhasePrefix)
	if prefix == "" {
		return strings.TrimSpace(name)
	}
	return prefix + " " + strings.TrimSpace(name)
}

func (e *SQLiteExecutionEngine) executePlannedStepsPipeline(
	ctx context.Context,
	plannedSteps []plannedStep,
	stepIDs map[int]string,
	result ExecuteResult,
	opts stepExecutionPipelineOptions,
) (ExecuteResult, error) {
	if result.StepStates == nil {
		result.StepStates = make([]StepExecutionRecord, 0, len(plannedSteps))
	}

	voiceGate := voicehandoffservice.NewGate()

	for _, step := range plannedSteps {
		stepID := stepIDs[step.StepIndex]
		if err := e.runTransaction(ctx, opts.phase("step running transition"), func(tx *sql.Tx) error {
			return updateTaskStepState(ctx, tx, stepID, shared.TaskStepStatusRunning, "", e.now().Format(time.RFC3339Nano))
		}); err != nil {
			return ExecuteResult{}, err
		}

		approvalDecision := evaluateStepApprovalDecision(
			step,
			opts.ExecutionOrigin,
			opts.InAppApprovalConfirmed,
			opts.ApprovalPhrase,
			voiceGate,
		)
		if approvalDecision.RequireApproval {
			approvalSummary := strings.TrimSpace(approvalDecision.Summary)
			if approvalSummary == "" {
				approvalSummary = "awaiting destructive approval"
			}
			approvalRequestID, approvalErr := randomID()
			if approvalErr != nil {
				return ExecuteResult{}, approvalErr
			}
			if err := e.runTransaction(ctx, opts.phase("approval transition"), func(tx *sql.Tx) error {
				nowText := e.now().Format(time.RFC3339Nano)
				if err := updateTaskStepState(ctx, tx, stepID, shared.TaskStepStatusPending, "", nowText); err != nil {
					return err
				}
				if err := createApprovalRequest(ctx, tx, approvalRequestID, opts.WorkspaceID, stepID, nowText, approvalDecision.Rationale); err != nil {
					return err
				}
				if err := updateTaskState(ctx, tx, opts.TaskID, shared.TaskStateAwaitingApproval, "", nowText); err != nil {
					return err
				}
				if err := updateRunState(ctx, tx, opts.RunID, shared.TaskStateAwaitingApproval, "", "", "", nowText); err != nil {
					return err
				}
				if err := insertApprovalRequestedAuditEvent(ctx, tx, opts.WorkspaceID, opts.RunID, stepID, opts.ActingAsActorID, opts.CorrelationID, approvalRequestID, approvalDecision.Rationale, nowText); err != nil {
					return err
				}
				return nil
			}); err != nil {
				return ExecuteResult{}, err
			}

			result.TaskState = string(shared.TaskStateAwaitingApproval)
			result.RunState = string(shared.TaskStateAwaitingApproval)
			result.ApprovalRequired = true
			result.ApprovalRequestID = approvalRequestID
			result.StepStates = append(result.StepStates, StepExecutionRecord{
				StepID:        stepID,
				StepIndex:     step.StepIndex,
				Name:          step.Name,
				CapabilityKey: step.CapabilityKey,
				Status:        string(shared.TaskStepStatusPending),
				Summary:       approvalSummary,
			})
			return result, nil
		}

		adapterID := ""
		var (
			executionResult connectorcontract.StepExecutionResult
			execErr         error
		)
		if isMessagesSendCapability(step.CapabilityKey) {
			executionResult, execErr = e.executeMessageStep(ctx, MessageDispatchRequest{
				WorkspaceID:   opts.WorkspaceID,
				TaskID:        opts.TaskID,
				RunID:         opts.RunID,
				StepIndex:     step.StepIndex,
				CorrelationID: opts.CorrelationID,
				SourceChannel: strings.TrimSpace(step.MessageChannel),
				Destination:   strings.TrimSpace(step.MessageRecipient),
				MessageBody:   strings.TrimSpace(step.MessageBody),
				OperationID:   fmt.Sprintf("taskrun-%s-step-%d", strings.TrimSpace(opts.RunID), step.StepIndex),
			})
			adapterID = "channel.dispatch"
		} else {
			adapter, err := e.selector.SelectByCapability(step.CapabilityKey, opts.PreferredAdapterID)
			if err != nil {
				errMessage := err.Error()
				if failureErr := e.recordStepAndRunFailure(ctx, opts.TaskID, opts.RunID, stepID, errMessage); failureErr != nil {
					return ExecuteResult{}, failureErr
				}
				if opts.FailureMode == stepExecutionFailureReturnResult {
					result = appendFailedStepExecutionRecord(result, step, stepID, "", errMessage)
					return result, nil
				}
				return ExecuteResult{}, err
			}

			execCtx := connectorcontract.ExecutionContext{
				WorkspaceID:      opts.WorkspaceID,
				TaskID:           opts.TaskID,
				RunID:            opts.RunID,
				StepID:           stepID,
				CorrelationID:    opts.CorrelationID,
				RequestedByActor: opts.RequestedByActorID,
				SubjectPrincipal: opts.SubjectActorID,
				ActingAsActor:    opts.ActingAsActorID,
				SourceChannel:    opts.SourceChannelForExecution,
			}
			stepContract := connectorcontract.TaskStep{
				ID:            stepID,
				RunID:         opts.RunID,
				StepIndex:     step.StepIndex,
				Name:          step.Name,
				Status:        shared.TaskStepStatusRunning,
				CapabilityKey: step.CapabilityKey,
				Input:         cloneAnyMap(step.Input),
			}

			executionResult, execErr = adapter.ExecuteStep(ctx, execCtx, stepContract)
			adapterID = adapter.Metadata().ID
		}

		if execErr != nil || executionResult.Status != shared.TaskStepStatusCompleted {
			errMessage := ""
			if execErr != nil {
				errMessage = execErr.Error()
			} else {
				errMessage = fmt.Sprintf("step returned status %s", executionResult.Status)
			}
			if failureErr := e.recordStepAndRunFailure(ctx, opts.TaskID, opts.RunID, stepID, errMessage); failureErr != nil {
				return ExecuteResult{}, failureErr
			}
			if opts.FailureMode == stepExecutionFailureReturnResult {
				result = appendFailedStepExecutionRecord(result, step, stepID, adapterID, errMessage)
				return result, nil
			}
			return ExecuteResult{}, fmt.Errorf("step %s failed: %s", step.CapabilityKey, errMessage)
		}

		if err := e.runTransaction(ctx, opts.phase("step completion transition"), func(tx *sql.Tx) error {
			nowText := e.now().Format(time.RFC3339Nano)
			if err := updateTaskStepState(ctx, tx, stepID, shared.TaskStepStatusCompleted, "", nowText); err != nil {
				return err
			}
			if err := insertStepAuditEvent(ctx, tx, opts.WorkspaceID, opts.RunID, stepID, opts.ActingAsActorID, opts.CorrelationID, executionResult, nowText); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return ExecuteResult{}, err
		}

		result.StepStates = append(result.StepStates, StepExecutionRecord{
			StepID:        stepID,
			StepIndex:     step.StepIndex,
			Name:          step.Name,
			CapabilityKey: step.CapabilityKey,
			AdapterID:     adapterID,
			Status:        string(shared.TaskStepStatusCompleted),
			Summary:       executionResult.Summary,
			Evidence:      cloneEvidence(executionResult.Evidence),
		})
	}

	finishedAt := e.now().Format(time.RFC3339Nano)
	if err := e.runTransaction(ctx, opts.phase("completion transition"), func(tx *sql.Tx) error {
		if err := updateTaskState(ctx, tx, opts.TaskID, shared.TaskStateCompleted, "", finishedAt); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, opts.RunID, shared.TaskStateCompleted, "", finishedAt, "", finishedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ExecuteResult{}, err
	}

	result.TaskState = string(shared.TaskStateCompleted)
	result.RunState = string(shared.TaskStateCompleted)
	return result, nil
}

func appendFailedStepExecutionRecord(
	result ExecuteResult,
	step plannedStep,
	stepID string,
	adapterID string,
	summary string,
) ExecuteResult {
	result.TaskState = string(shared.TaskStateFailed)
	result.RunState = string(shared.TaskStateFailed)
	record := StepExecutionRecord{
		StepID:        stepID,
		StepIndex:     step.StepIndex,
		Name:          step.Name,
		CapabilityKey: step.CapabilityKey,
		Status:        string(shared.TaskStepStatusFailed),
		Summary:       summary,
	}
	if strings.TrimSpace(adapterID) != "" {
		record.AdapterID = strings.TrimSpace(adapterID)
	}
	result.StepStates = append(result.StepStates, record)
	return result
}
