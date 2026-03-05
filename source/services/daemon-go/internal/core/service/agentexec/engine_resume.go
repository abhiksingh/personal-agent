package agentexec

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	repoapproval "personalagent/runtime/internal/core/repository/approval"
	repoauthz "personalagent/runtime/internal/core/repository/authz"
	approvalservice "personalagent/runtime/internal/core/service/approval"
	authzservice "personalagent/runtime/internal/core/service/authz"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

func (e *SQLiteExecutionEngine) ResumeAfterApproval(ctx context.Context, request ResumeRequest) (ExecuteResult, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if strings.TrimSpace(request.ApprovalRequestID) == "" {
		return ExecuteResult{}, fmt.Errorf("approval request id is required")
	}
	if strings.TrimSpace(request.DecisionByActorID) == "" {
		return ExecuteResult{}, fmt.Errorf("decision actor id is required")
	}

	row := e.db.QueryRowContext(ctx, `
		SELECT
			ts.run_id,
			ar.step_id,
			tr.task_id,
			tr.acting_as_actor_id,
			t.requested_by_actor_id,
			t.subject_principal_actor_id,
			ts.step_index,
			ts.name,
			ts.capability_key,
			COALESCE(ts.input_json, '')
		FROM approval_requests ar
		JOIN task_steps ts ON ts.id = ar.step_id
		JOIN task_runs tr ON tr.id = ts.run_id
		JOIN tasks t ON t.id = tr.task_id
		WHERE ar.id = ? AND ar.workspace_id = ?
	`, request.ApprovalRequestID, workspaceID)

	var (
		runID            string
		stepID           string
		taskID           string
		actingAsActorID  string
		requestedByActor string
		subjectActor     string
		stepIndex        int
		stepName         string
		capabilityKey    string
		stepInputJSON    string
	)
	if err := row.Scan(
		&runID,
		&stepID,
		&taskID,
		&actingAsActorID,
		&requestedByActor,
		&subjectActor,
		&stepIndex,
		&stepName,
		&capabilityKey,
		&stepInputJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return ExecuteResult{}, fmt.Errorf("approval request not found")
		}
		return ExecuteResult{}, fmt.Errorf("load approval context: %w", err)
	}

	if err := e.authorizeApprovalDecisionActor(ctx, workspaceID, actingAsActorID, strings.TrimSpace(request.DecisionByActorID)); err != nil {
		return ExecuteResult{}, err
	}

	approvalSvc := approvalservice.NewService(repoapproval.NewSQLiteApprovalStore(e.db), e.now)
	if err := approvalSvc.ConfirmDestructiveApproval(ctx, types.ApprovalConfirmationRequest{
		WorkspaceID:       workspaceID,
		ApprovalRequestID: request.ApprovalRequestID,
		DecisionByActorID: strings.TrimSpace(request.DecisionByActorID),
		RunID:             runID,
		StepID:            stepID,
		Phrase:            request.Phrase,
		CorrelationID:     request.CorrelationID,
	}); err != nil {
		return ExecuteResult{}, err
	}

	if err := e.runTransaction(ctx, "resume running transition", func(tx *sql.Tx) error {
		nowText := e.now().Format(time.RFC3339Nano)
		if err := updateTaskState(ctx, tx, taskID, shared.TaskStateRunning, "", nowText); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, runID, shared.TaskStateRunning, "", "", "", nowText); err != nil {
			return err
		}
		if err := updateTaskStepState(ctx, tx, stepID, shared.TaskStepStatusRunning, "", nowText); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ExecuteResult{}, err
	}

	adapter, err := e.selector.SelectByCapability(capabilityKey, "")
	if err != nil {
		if failureErr := e.recordStepAndRunFailure(ctx, taskID, runID, stepID, err.Error()); failureErr != nil {
			return ExecuteResult{}, failureErr
		}
		return ExecuteResult{}, err
	}

	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID:       workspaceID,
		TaskID:            taskID,
		RunID:             runID,
		StepID:            stepID,
		CorrelationID:     request.CorrelationID,
		RequestedByActor:  requestedByActor,
		SubjectPrincipal:  subjectActor,
		ActingAsActor:     actingAsActorID,
		SourceChannel:     "cli_chat",
		ApprovalReference: request.ApprovalRequestID,
	}
	stepContract := connectorcontract.TaskStep{
		ID:            stepID,
		RunID:         runID,
		StepIndex:     stepIndex,
		Name:          stepName,
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: capabilityKey,
		Input:         parseStepInputJSON(stepInputJSON),
	}

	executionResult, execErr := adapter.ExecuteStep(ctx, execCtx, stepContract)
	if execErr != nil || executionResult.Status != shared.TaskStepStatusCompleted {
		errMessage := ""
		if execErr != nil {
			errMessage = execErr.Error()
		} else {
			errMessage = fmt.Sprintf("step returned status %s", executionResult.Status)
		}
		if failureErr := e.recordStepAndRunFailure(ctx, taskID, runID, stepID, errMessage); failureErr != nil {
			return ExecuteResult{}, failureErr
		}
		return ExecuteResult{}, fmt.Errorf("resume step failed: %s", errMessage)
	}

	finishedAt := e.now().Format(time.RFC3339Nano)
	if err := e.runTransaction(ctx, "resume completion transition", func(tx *sql.Tx) error {
		if err := updateTaskStepState(ctx, tx, stepID, shared.TaskStepStatusCompleted, "", finishedAt); err != nil {
			return err
		}
		if err := updateTaskState(ctx, tx, taskID, shared.TaskStateCompleted, "", finishedAt); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, runID, shared.TaskStateCompleted, "", finishedAt, "", finishedAt); err != nil {
			return err
		}
		if err := insertStepAuditEvent(ctx, tx, workspaceID, runID, stepID, actingAsActorID, request.CorrelationID, executionResult, finishedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ExecuteResult{}, err
	}

	return ExecuteResult{
		Workflow:  workflowFromCapability(capabilityKey),
		TaskID:    taskID,
		RunID:     runID,
		TaskState: string(shared.TaskStateCompleted),
		RunState:  string(shared.TaskStateCompleted),
		StepStates: []StepExecutionRecord{
			{
				StepID:        stepID,
				StepIndex:     stepIndex,
				Name:          stepName,
				CapabilityKey: capabilityKey,
				AdapterID:     adapter.Metadata().ID,
				Status:        string(shared.TaskStepStatusCompleted),
				Summary:       executionResult.Summary,
				Evidence:      cloneEvidence(executionResult.Evidence),
			},
		},
	}, nil
}

func (e *SQLiteExecutionEngine) authorizeApprovalDecisionActor(ctx context.Context, workspaceID string, actingAsActorID string, decisionByActorID string) error {
	resolvedActingAs := strings.TrimSpace(actingAsActorID)
	resolvedDecisionActor := strings.TrimSpace(decisionByActorID)
	if resolvedDecisionActor == resolvedActingAs {
		return nil
	}
	authorizer := authzservice.NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(e.db))
	decision, err := authorizer.CanActAs(ctx, types.ActingAsRequest{
		WorkspaceID:        workspaceID,
		RequestedByActorID: resolvedActingAs,
		ActingAsActorID:    resolvedDecisionActor,
		ScopeType:          "APPROVAL",
	})
	if err != nil {
		return fmt.Errorf("evaluate approval delegation: %w", err)
	}
	if !decision.Allowed {
		return fmt.Errorf("approval denied: decision actor %s is not authorized for acting_as %s", resolvedDecisionActor, resolvedActingAs)
	}
	return nil
}
