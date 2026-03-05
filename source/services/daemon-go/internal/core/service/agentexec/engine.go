package agentexec

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type ConnectorSelector interface {
	SelectByCapability(capabilityKey string, preferredAdapterID string) (connectorcontract.Adapter, error)
}

type MessageDispatchRequest struct {
	WorkspaceID   string
	TaskID        string
	RunID         string
	StepIndex     int
	CorrelationID string
	SourceChannel string
	Destination   string
	MessageBody   string
	OperationID   string
}

type MessageDispatchResult struct {
	Channel         string
	ProviderReceipt string
	Summary         string
}

type MessageDispatcher interface {
	DispatchMessage(ctx context.Context, request MessageDispatchRequest) (MessageDispatchResult, error)
}

type ExecuteRequest struct {
	WorkspaceID            string
	RequestText            string
	NativeAction           *NativeAction
	RequestedByActorID     string
	SubjectActorID         string
	ActingAsActorID        string
	Origin                 types.ExecutionOrigin
	InAppApprovalConfirmed bool
	CorrelationID          string
	ApprovalPhrase         string
	PreferredAdapterID     string
}

type PersistedRunRequest struct {
	WorkspaceID        string
	TaskID             string
	RunID              string
	RequestText        string
	RequestedByActorID string
	SubjectActorID     string
	ActingAsActorID    string
	SourceChannel      string
	CorrelationID      string
	PreferredAdapterID string
}

type StepExecutionRecord struct {
	StepID        string            `json:"step_id"`
	StepIndex     int               `json:"step_index"`
	Name          string            `json:"name"`
	CapabilityKey string            `json:"capability_key"`
	AdapterID     string            `json:"adapter_id"`
	Status        string            `json:"status"`
	Summary       string            `json:"summary"`
	Evidence      map[string]string `json:"evidence,omitempty"`
}

type ExecuteResult struct {
	Workflow              string                `json:"workflow"`
	NativeAction          *NativeAction         `json:"native_action,omitempty"`
	TaskID                string                `json:"task_id"`
	RunID                 string                `json:"run_id"`
	TaskState             string                `json:"task_state"`
	RunState              string                `json:"run_state"`
	ClarificationRequired bool                  `json:"clarification_required,omitempty"`
	ClarificationPrompt   string                `json:"clarification_prompt,omitempty"`
	MissingSlots          []string              `json:"missing_slots,omitempty"`
	ApprovalRequired      bool                  `json:"approval_required,omitempty"`
	ApprovalRequestID     string                `json:"approval_request_id,omitempty"`
	StepStates            []StepExecutionRecord `json:"step_states"`
}

type ResumeRequest struct {
	WorkspaceID       string
	ApprovalRequestID string
	DecisionByActorID string
	Phrase            string
	CorrelationID     string
}

type plannedStep struct {
	StepIndex        int
	Name             string
	CapabilityKey    string
	Input            map[string]any
	MessageChannel   string
	MessageRecipient string
	MessageBody      string
}

type approvalDecisionRationale struct {
	PolicyVersion      string  `json:"policy_version"`
	CapabilityKey      string  `json:"capability_key"`
	RiskLevel          string  `json:"risk_level"`
	RiskConfidence     float64 `json:"risk_confidence"`
	RiskReason         string  `json:"risk_reason"`
	DestructiveClass   string  `json:"destructive_class,omitempty"`
	Decision           string  `json:"decision"`
	DecisionReason     string  `json:"decision_reason"`
	DecisionReasonCode string  `json:"decision_reason_code"`
	DecisionSource     string  `json:"decision_source"`
	ExecutionOrigin    string  `json:"execution_origin"`
}

type stepApprovalDecision struct {
	RequireApproval bool
	Summary         string
	Rationale       approvalDecisionRationale
}

type SQLiteExecutionEngine struct {
	db                *sql.DB
	selector          ConnectorSelector
	messageDispatcher MessageDispatcher
	now               func() time.Time
	intentInterpreter IntentInterpreter
}

func NewSQLiteExecutionEngine(db *sql.DB, selector ConnectorSelector) *SQLiteExecutionEngine {
	return &SQLiteExecutionEngine{
		db:       db,
		selector: selector,
		intentInterpreter: NewModelAssistedIntentInterpreter(
			nil,
			0,
		),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (e *SQLiteExecutionEngine) SetIntentInterpreter(interpreter IntentInterpreter) {
	if interpreter == nil {
		e.intentInterpreter = NewDeterministicIntentInterpreter()
		return
	}
	e.intentInterpreter = interpreter
}

func (e *SQLiteExecutionEngine) SetMessageDispatcher(dispatcher MessageDispatcher) {
	e.messageDispatcher = dispatcher
}

func (e *SQLiteExecutionEngine) Execute(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	if e.db == nil {
		return ExecuteResult{}, fmt.Errorf("db is required")
	}
	if e.selector == nil {
		return ExecuteResult{}, fmt.Errorf("connector selector is required")
	}

	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	executionOrigin, err := normalizeExecutionOrigin(request.Origin)
	if err != nil {
		return ExecuteResult{}, err
	}
	intent, err := e.resolveIntent(ctx, workspaceID, request.RequestText, request.NativeAction)
	if err != nil {
		return ExecuteResult{}, err
	}
	if intent.RequiresClarification() {
		return ExecuteResult{
			Workflow:              intent.Workflow,
			NativeAction:          cloneNativeAction(intent.Action),
			TaskState:             "clarification_required",
			RunState:              "clarification_required",
			ClarificationRequired: true,
			ClarificationPrompt:   strings.TrimSpace(intent.ClarificationPrompt),
			MissingSlots:          cloneStringSlice(intent.MissingSlots),
			StepStates:            []StepExecutionRecord{},
		}, nil
	}

	plannedSteps, err := planSteps(intent)
	if err != nil {
		return ExecuteResult{}, err
	}

	requestedBy := normalizeActorID(request.RequestedByActorID, "actor.requester."+workspaceID)
	subject := normalizeActorID(request.SubjectActorID, requestedBy)
	actingAs := normalizeActorID(request.ActingAsActorID, subject)
	nowText := e.now().Format(time.RFC3339Nano)

	taskID, err := randomID()
	if err != nil {
		return ExecuteResult{}, err
	}
	runID, err := randomID()
	if err != nil {
		return ExecuteResult{}, err
	}

	stepIDs := make(map[int]string, len(plannedSteps))
	for _, step := range plannedSteps {
		stepID, stepErr := randomID()
		if stepErr != nil {
			return ExecuteResult{}, stepErr
		}
		stepIDs[step.StepIndex] = stepID
	}
	if err := e.runTransaction(ctx, "execution bootstrap", func(tx *sql.Tx) error {
		if err := ensureWorkspace(ctx, tx, workspaceID, nowText); err != nil {
			return err
		}
		for _, actorID := range []string{requestedBy, subject, actingAs} {
			if err := ensureActorPrincipal(ctx, tx, workspaceID, actorID, nowText); err != nil {
				return err
			}
		}
		if err := insertTask(ctx, tx, taskID, workspaceID, requestedBy, subject, nowText, intent); err != nil {
			return err
		}
		if err := insertTaskRun(ctx, tx, runID, taskID, workspaceID, actingAs, nowText); err != nil {
			return err
		}
		if err := updateTaskState(ctx, tx, taskID, shared.TaskStatePlanning, "", nowText); err != nil {
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
		if err := updateRunState(ctx, tx, runID, shared.TaskStateRunning, nowText, "", "", nowText); err != nil {
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
			SourceChannelForExecution: sourceChannelForExecutionOrigin(executionOrigin),
			ExecutionOrigin:           executionOrigin,
			InAppApprovalConfirmed:    request.InAppApprovalConfirmed,
			ApprovalPhrase:            request.ApprovalPhrase,
			FailureMode:               stepExecutionFailureReturnError,
		},
	)
}
