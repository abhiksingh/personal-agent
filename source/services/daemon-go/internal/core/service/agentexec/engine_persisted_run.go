package agentexec

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

func (e *SQLiteExecutionEngine) ExecutePersistedRun(ctx context.Context, request PersistedRunRequest) (ExecuteResult, error) {
	if e.db == nil {
		return ExecuteResult{}, fmt.Errorf("db is required")
	}
	if e.selector == nil {
		return ExecuteResult{}, fmt.Errorf("connector selector is required")
	}

	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	requestedBy := strings.TrimSpace(request.RequestedByActorID)
	subject := strings.TrimSpace(request.SubjectActorID)
	actingAs := strings.TrimSpace(request.ActingAsActorID)
	requestText := strings.TrimSpace(request.RequestText)
	sourceChannel := strings.TrimSpace(request.SourceChannel)
	if sourceChannel == "" {
		sourceChannel = "daemon_queue"
	}
	if taskID == "" {
		return ExecuteResult{}, fmt.Errorf("task id is required")
	}
	if runID == "" {
		return ExecuteResult{}, fmt.Errorf("run id is required")
	}
	if requestedBy == "" || subject == "" || actingAs == "" {
		return ExecuteResult{}, fmt.Errorf("requested_by, subject, and acting_as are required")
	}
	if requestText == "" {
		errMessage := "queued run request text is empty"
		if err := e.markPersistedRunFailed(ctx, taskID, runID, errMessage); err != nil {
			return ExecuteResult{}, err
		}
		return ExecuteResult{
			TaskID:    taskID,
			RunID:     runID,
			TaskState: string(shared.TaskStateFailed),
			RunState:  string(shared.TaskStateFailed),
			StepStates: []StepExecutionRecord{
				{
					Status:  string(shared.TaskStepStatusFailed),
					Summary: errMessage,
				},
			},
		}, nil
	}

	intent, err := e.interpretIntent(ctx, workspaceID, requestText)
	if err != nil {
		errMessage := fmt.Sprintf("intent interpretation failed: %v", err)
		if markErr := e.markPersistedRunFailed(ctx, taskID, runID, errMessage); markErr != nil {
			return ExecuteResult{}, markErr
		}
		return ExecuteResult{
			TaskID:    taskID,
			RunID:     runID,
			TaskState: string(shared.TaskStateFailed),
			RunState:  string(shared.TaskStateFailed),
			StepStates: []StepExecutionRecord{
				{
					Status:  string(shared.TaskStepStatusFailed),
					Summary: errMessage,
				},
			},
		}, nil
	}

	if intent.RequiresClarification() {
		nowText := e.now().Format(time.RFC3339Nano)
		lastError := fmt.Sprintf("clarification required: %s", strings.Join(intent.MissingSlots, ", "))
		if err := e.runTransaction(ctx, "clarification transition", func(tx *sql.Tx) error {
			if err := updateTaskState(ctx, tx, taskID, shared.TaskStateBlocked, "", nowText); err != nil {
				return err
			}
			if err := updateRunState(ctx, tx, runID, shared.TaskStateBlocked, nowText, "", lastError, nowText); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return ExecuteResult{}, err
		}

		return ExecuteResult{
			Workflow:              intent.Workflow,
			NativeAction:          cloneNativeAction(intent.Action),
			TaskID:                taskID,
			RunID:                 runID,
			TaskState:             string(shared.TaskStateBlocked),
			RunState:              string(shared.TaskStateBlocked),
			ClarificationRequired: true,
			ClarificationPrompt:   strings.TrimSpace(intent.ClarificationPrompt),
			MissingSlots:          cloneStringSlice(intent.MissingSlots),
			StepStates:            []StepExecutionRecord{},
		}, nil
	}

	plannedSteps, err := planSteps(intent)
	if err != nil {
		errMessage := fmt.Sprintf("plan steps failed: %v", err)
		if markErr := e.markPersistedRunFailed(ctx, taskID, runID, errMessage); markErr != nil {
			return ExecuteResult{}, markErr
		}
		return ExecuteResult{
			Workflow:  intent.Workflow,
			TaskID:    taskID,
			RunID:     runID,
			TaskState: string(shared.TaskStateFailed),
			RunState:  string(shared.TaskStateFailed),
			StepStates: []StepExecutionRecord{
				{
					Status:  string(shared.TaskStepStatusFailed),
					Summary: errMessage,
				},
			},
		}, nil
	}

	nowText := e.now().Format(time.RFC3339Nano)
	stepIDs := make(map[int]string, len(plannedSteps))
	for _, step := range plannedSteps {
		stepID, stepErr := randomID()
		if stepErr != nil {
			return ExecuteResult{}, stepErr
		}
		stepIDs[step.StepIndex] = stepID
	}
	if err := e.runTransaction(ctx, "persisted bootstrap", func(tx *sql.Tx) error {
		if err := updateTaskState(ctx, tx, taskID, shared.TaskStatePlanning, "", nowText); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, runID, shared.TaskStateRunning, nowText, "", "", nowText); err != nil {
			return err
		}
		for _, step := range plannedSteps {
			if err := insertTaskStep(ctx, tx, stepIDs[step.StepIndex], runID, step, shared.TaskStepStatusPending, nowText); err != nil {
				return err
			}
		}
		if err := updateTaskState(ctx, tx, taskID, shared.TaskStateRunning, "", nowText); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ExecuteResult{}, err
	}

	result := ExecuteResult{
		Workflow:     intent.Workflow,
		NativeAction: cloneNativeAction(intent.Action),
		TaskID:       taskID,
		RunID:        runID,
		TaskState:    string(shared.TaskStateRunning),
		RunState:     string(shared.TaskStateRunning),
		StepStates:   make([]StepExecutionRecord, 0, len(plannedSteps)),
	}
	return e.executePlannedStepsPipeline(
		ctx,
		plannedSteps,
		stepIDs,
		result,
		stepExecutionPipelineOptions{
			WorkspaceID:               workspaceID,
			TaskID:                    taskID,
			RunID:                     runID,
			RequestedByActorID:        requestedBy,
			SubjectActorID:            subject,
			ActingAsActorID:           actingAs,
			CorrelationID:             request.CorrelationID,
			PreferredAdapterID:        request.PreferredAdapterID,
			SourceChannelForExecution: sourceChannel,
			ExecutionOrigin:           executionOriginFromSourceChannel(sourceChannel),
			FailureMode:               stepExecutionFailureReturnResult,
			PhasePrefix:               "persisted",
		},
	)
}

func (e *SQLiteExecutionEngine) markPersistedRunFailed(ctx context.Context, taskID string, runID string, errMessage string) error {
	if strings.TrimSpace(taskID) == "" || strings.TrimSpace(runID) == "" {
		return nil
	}
	nowText := e.now().Format(time.RFC3339Nano)
	return e.runTransaction(ctx, "persisted failed-state transition", func(tx *sql.Tx) error {
		if err := updateTaskState(ctx, tx, strings.TrimSpace(taskID), shared.TaskStateFailed, errMessage, nowText); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, strings.TrimSpace(runID), shared.TaskStateFailed, nowText, nowText, errMessage, nowText); err != nil {
			return err
		}
		return nil
	})
}

func (e *SQLiteExecutionEngine) interpretIntent(ctx context.Context, workspaceID string, request string) (Intent, error) {
	if e.intentInterpreter == nil {
		return InterpretIntent(request)
	}
	return e.intentInterpreter.Interpret(ctx, workspaceID, request)
}

func (e *SQLiteExecutionEngine) resolveIntent(ctx context.Context, workspaceID string, requestText string, nativeAction *NativeAction) (Intent, error) {
	if nativeAction != nil {
		return IntentFromNativeAction(nativeAction)
	}
	return e.interpretIntent(ctx, workspaceID, requestText)
}
