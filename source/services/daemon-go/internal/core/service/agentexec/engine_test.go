package agentexec

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"
	shared "personalagent/runtime/internal/shared/contracts"

	_ "modernc.org/sqlite"
)

type engineStubModelIntentExtractor struct {
	candidate ModelIntentCandidate
	err       error
}

func (s engineStubModelIntentExtractor) ExtractIntent(_ context.Context, _ string, _ string) (ModelIntentCandidate, error) {
	if s.err != nil {
		return ModelIntentCandidate{}, s.err
	}
	return s.candidate, nil
}

type engineStubMessageDispatcher struct {
	requests []MessageDispatchRequest
	result   MessageDispatchResult
	err      error
}

func (s *engineStubMessageDispatcher) DispatchMessage(_ context.Context, request MessageDispatchRequest) (MessageDispatchResult, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return MessageDispatchResult{}, s.err
	}
	return s.result, nil
}

type engineSQLiteRoundTripMessageDispatcher struct {
	db    *sql.DB
	calls int
}

func (s *engineSQLiteRoundTripMessageDispatcher) DispatchMessage(ctx context.Context, _ MessageDispatchRequest) (MessageDispatchResult, error) {
	s.calls++
	if s.db == nil {
		return MessageDispatchResult{}, fmt.Errorf("db is required")
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count); err != nil {
		return MessageDispatchResult{}, err
	}
	return MessageDispatchResult{
		Channel:         "sms",
		ProviderReceipt: "SM-ROUNDTRIP",
		Summary:         "message dispatched via sms",
	}, nil
}

type engineSQLiteRoundTripFinderAdapter struct {
	db    *sql.DB
	calls int
}

func (s *engineSQLiteRoundTripFinderAdapter) Metadata() shared.AdapterMetadata {
	return shared.AdapterMetadata{
		ID:          "finder.roundtrip",
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Finder Roundtrip",
		Version:     "test",
		Capabilities: []shared.CapabilityDescriptor{
			{
				Key: "finder_delete",
			},
		},
	}
}

func (s *engineSQLiteRoundTripFinderAdapter) HealthCheck(context.Context) error {
	return nil
}

func (s *engineSQLiteRoundTripFinderAdapter) ExecuteStep(ctx context.Context, _ shared.ExecutionContext, _ shared.TaskStep) (shared.StepExecutionResult, error) {
	s.calls++
	if s.db == nil {
		return shared.StepExecutionResult{}, fmt.Errorf("db is required")
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count); err != nil {
		return shared.StepExecutionResult{}, err
	}
	return shared.StepExecutionResult{
		Status:  shared.TaskStepStatusCompleted,
		Summary: "finder delete completed",
		Evidence: map[string]string{
			"task_count": fmt.Sprintf("%d", count),
		},
	}, nil
}

func setupAgentExecDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "agent-exec.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func buildSelector(t *testing.T) *connectorregistry.Registry {
	t.Helper()
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")
	registry := connectorregistry.New()
	if err := registry.Register(mailadapter.NewAdapter("mail.exec")); err != nil {
		t.Fatalf("register mail adapter: %v", err)
	}
	if err := registry.Register(calendaradapter.NewAdapter("calendar.exec")); err != nil {
		t.Fatalf("register calendar adapter: %v", err)
	}
	if err := registry.Register(browseradapter.NewAdapter("browser.exec")); err != nil {
		t.Fatalf("register browser adapter: %v", err)
	}
	if err := registry.Register(finderadapter.NewAdapter("finder.exec")); err != nil {
		t.Fatalf("register finder adapter: %v", err)
	}
	return registry
}

func TestExecutePersistsCompletedRunForBrowserIntent(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:   "ws1",
		RequestText:   "open https://example.com and summarize it",
		CorrelationID: "corr-agent-browser",
	})
	if err != nil {
		t.Fatalf("execute browser intent: %v", err)
	}
	if result.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow, got %s", result.Workflow)
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.StepStates) != 3 {
		t.Fatalf("expected three steps, got %d", len(result.StepStates))
	}
}

func TestExecuteUsesTypedNativeActionForSingleBrowserToolOperation(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: WorkflowBrowser,
			Operation: "open",
			Browser: &BrowserAction{
				Operation: "open",
				TargetURL: "https://example.com/path",
			},
		},
		CorrelationID: "corr-agent-browser-native-open",
	})
	if err != nil {
		t.Fatalf("execute browser native action: %v", err)
	}
	if result.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow, got %s", result.Workflow)
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.StepStates) != 1 {
		t.Fatalf("expected one step for single open operation, got %d", len(result.StepStates))
	}
	if result.StepStates[0].CapabilityKey != "browser_open" {
		t.Fatalf("expected browser_open capability, got %s", result.StepStates[0].CapabilityKey)
	}
}

func TestPlanStepsBrowserOpenExtractCloseCarriesExtractQuery(t *testing.T) {
	steps, err := planSteps(Intent{
		Workflow:  WorkflowBrowser,
		TargetURL: "https://example.com",
		Action: NativeAction{
			Connector: WorkflowBrowser,
			Operation: "open_extract_close",
			Browser: &BrowserAction{
				Operation: "open_extract_close",
				TargetURL: "https://example.com",
				Query:     "summarize this page",
			},
		},
	})
	if err != nil {
		t.Fatalf("plan browser open_extract_close steps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected three browser steps, got %d", len(steps))
	}
	if steps[1].CapabilityKey != "browser_extract" {
		t.Fatalf("expected browser_extract second step, got %s", steps[1].CapabilityKey)
	}
	query, ok := steps[1].Input["query"].(string)
	if !ok || query != "summarize this page" {
		t.Fatalf("expected extract step query to be preserved, got %#v", steps[1].Input["query"])
	}
}

func TestExecuteUsesModelAssistedIntentWhenDeterministicParserCannotResolve(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))
	engine.SetIntentInterpreter(NewModelAssistedIntentInterpreter(engineStubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:   WorkflowCalendar,
			Confidence: 0.95,
		},
	}, 0.6))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:   "ws1",
		RequestText:   "please follow up with the team on this",
		CorrelationID: "corr-agent-model-intent",
	})
	if err != nil {
		t.Fatalf("execute model-assisted intent: %v", err)
	}
	if result.Workflow != WorkflowCalendar {
		t.Fatalf("expected model-assisted workflow calendar, got %s", result.Workflow)
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
}

func TestExecuteFallsBackToDeterministicWhenModelIntentIsInvalid(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))
	engine.SetIntentInterpreter(NewModelAssistedIntentInterpreter(engineStubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:   WorkflowFinder,
			Confidence: 0.95,
		},
	}, 0.6))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:   "ws1",
		RequestText:   "open https://example.com and summarize it",
		CorrelationID: "corr-agent-model-fallback",
	})
	if err != nil {
		t.Fatalf("execute deterministic fallback intent: %v", err)
	}
	if result.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow via deterministic fallback, got %s", result.Workflow)
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
}

func TestExecuteReturnsClarificationWithoutPersistingTaskForMissingFinderPath(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file now",
	})
	if err != nil {
		t.Fatalf("execute finder clarification intent: %v", err)
	}
	if !result.ClarificationRequired {
		t.Fatalf("expected clarification_required=true")
	}
	if result.TaskState != "clarification_required" || result.RunState != "clarification_required" {
		t.Fatalf("expected clarification_required states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.MissingSlots) == 0 {
		t.Fatalf("expected missing slots in clarification response")
	}
	if result.TaskID != "" || result.RunID != "" {
		t.Fatalf("did not expect task/run ids for clarification response")
	}

	var taskCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&taskCount); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected no persisted tasks for clarification response, got %d", taskCount)
	}
}

func TestExecuteReturnsClarificationForMessagesIntent(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: `send a text to +15550001111: "hello"`,
	})
	if err != nil {
		t.Fatalf("execute messages clarification intent: %v", err)
	}
	if result.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow, got %s", result.Workflow)
	}
	if !result.ClarificationRequired {
		t.Fatalf("expected clarification_required=true")
	}
	if result.NativeAction == nil || result.NativeAction.Messages == nil {
		t.Fatalf("expected messages native action in clarification response")
	}
}

func TestExecuteDispatchesResolvedMessagesWorkflow(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))
	dispatcher := &engineStubMessageDispatcher{
		result: MessageDispatchResult{
			Channel:         "sms",
			ProviderReceipt: "SM123",
			Summary:         "message dispatched via sms",
		},
	}
	engine.SetMessageDispatcher(dispatcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: `send an sms to +15550001111: "hello"`,
	})
	if err != nil {
		t.Fatalf("execute resolved messages intent: %v", err)
	}
	if result.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow, got %s", result.Workflow)
	}
	if result.ClarificationRequired {
		t.Fatalf("did not expect clarification for resolved messages workflow")
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.StepStates) != 1 {
		t.Fatalf("expected one messages step, got %d", len(result.StepStates))
	}
	if result.StepStates[0].CapabilityKey != "messages_send_sms" {
		t.Fatalf("expected messages_send_sms capability, got %s", result.StepStates[0].CapabilityKey)
	}
	if result.StepStates[0].AdapterID != "channel.dispatch" {
		t.Fatalf("expected channel.dispatch adapter id, got %s", result.StepStates[0].AdapterID)
	}
	if len(dispatcher.requests) != 1 {
		t.Fatalf("expected one message dispatch request, got %d", len(dispatcher.requests))
	}
	if dispatcher.requests[0].SourceChannel != "sms" {
		t.Fatalf("expected source channel sms, got %s", dispatcher.requests[0].SourceChannel)
	}
	if dispatcher.requests[0].Destination != "+15550001111" {
		t.Fatalf("expected destination +15550001111, got %s", dispatcher.requests[0].Destination)
	}
	if dispatcher.requests[0].MessageBody != "hello" {
		t.Fatalf("expected message body hello, got %s", dispatcher.requests[0].MessageBody)
	}
}

func TestExecuteMessageDispatchCanReenterSQLiteWithSingleWriterPool(t *testing.T) {
	db := setupAgentExecDB(t)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))
	dispatcher := &engineSQLiteRoundTripMessageDispatcher{db: db}
	engine.SetMessageDispatcher(dispatcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: "messages",
			Operation: "send_message",
			Messages: &MessagesAction{
				Operation: "send_message",
				Channel:   "sms",
				Recipient: "+15550001111",
				Body:      "hello roundtrip",
			},
		},
		CorrelationID: "corr-agent-sqlite-roundtrip-dispatch",
	})
	if err != nil {
		t.Fatalf("execute messages workflow with sqlite roundtrip dispatcher: %v", err)
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if dispatcher.calls != 1 {
		t.Fatalf("expected one sqlite roundtrip dispatch call, got %d", dispatcher.calls)
	}
}

func TestExecutePausesForFinderApproval(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder intent: %v", err)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval_required=true")
	}
	if result.ApprovalRequestID == "" {
		t.Fatalf("expected approval request id")
	}
	if result.TaskState != "awaiting_approval" || result.RunState != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval states, got task=%s run=%s", result.TaskState, result.RunState)
	}

	var rationaleRaw string
	if err := db.QueryRow(`SELECT rationale FROM approval_requests WHERE id = ?`, result.ApprovalRequestID).Scan(&rationaleRaw); err != nil {
		t.Fatalf("load approval rationale: %v", err)
	}
	rationale := map[string]any{}
	if err := json.Unmarshal([]byte(rationaleRaw), &rationale); err != nil {
		t.Fatalf("parse approval rationale json: %v", err)
	}
	if rationale["capability_key"] != "finder_delete" {
		t.Fatalf("expected approval rationale capability_key finder_delete, got %#v", rationale["capability_key"])
	}
	if rationale["risk_level"] != "destructive" {
		t.Fatalf("expected approval rationale risk_level destructive, got %#v", rationale["risk_level"])
	}
	if rationale["decision_reason_code"] != "missing_approval_phrase" {
		t.Fatalf("expected approval rationale decision_reason_code missing_approval_phrase, got %#v", rationale["decision_reason_code"])
	}

	var auditPayloadRaw string
	if err := db.QueryRow(`
		SELECT payload_json
		FROM audit_log_entries
		WHERE run_id = ?
		  AND event_type = 'APPROVAL_REQUESTED'
		ORDER BY created_at DESC
		LIMIT 1
	`, result.RunID).Scan(&auditPayloadRaw); err != nil {
		t.Fatalf("load approval audit payload: %v", err)
	}
	auditPayload := map[string]any{}
	if err := json.Unmarshal([]byte(auditPayloadRaw), &auditPayload); err != nil {
		t.Fatalf("parse approval audit payload json: %v", err)
	}
	rationalePayload, ok := auditPayload["rationale"].(map[string]any)
	if !ok {
		t.Fatalf("expected approval audit payload rationale object, got %#v", auditPayload["rationale"])
	}
	if rationalePayload["capability_key"] != "finder_delete" {
		t.Fatalf("expected approval audit rationale capability_key finder_delete, got %#v", rationalePayload["capability_key"])
	}
}

func TestExecutePausesForCalendarCancelApproval(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: WorkflowCalendar,
			Operation: calendarOperationCancel,
			Calendar: &CalendarAction{
				Operation: calendarOperationCancel,
				EventID:   "event-team-sync-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute calendar intent: %v", err)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval_required=true")
	}
	if result.ApprovalRequestID == "" {
		t.Fatalf("expected approval request id")
	}
	if result.TaskState != "awaiting_approval" || result.RunState != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.StepStates) == 0 {
		t.Fatalf("expected step states to include pending calendar cancel step")
	}
	lastStep := result.StepStates[len(result.StepStates)-1]
	if lastStep.CapabilityKey != "calendar_cancel" {
		t.Fatalf("expected pending capability calendar_cancel, got %s", lastStep.CapabilityKey)
	}

	var rationaleRaw string
	if err := db.QueryRow(`SELECT rationale FROM approval_requests WHERE id = ?`, result.ApprovalRequestID).Scan(&rationaleRaw); err != nil {
		t.Fatalf("load approval rationale: %v", err)
	}
	rationale := map[string]any{}
	if err := json.Unmarshal([]byte(rationaleRaw), &rationale); err != nil {
		t.Fatalf("parse approval rationale json: %v", err)
	}
	if rationale["capability_key"] != "calendar_cancel" {
		t.Fatalf("expected capability_key calendar_cancel, got %#v", rationale["capability_key"])
	}
	if rationale["destructive_class"] != "calendar_cancel" {
		t.Fatalf("expected destructive_class calendar_cancel, got %#v", rationale["destructive_class"])
	}
}

func TestExecuteCalendarRunsWhenGoAheadPhraseProvided(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createResult, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: WorkflowCalendar,
			Operation: calendarOperationCreate,
			Calendar: &CalendarAction{
				Operation: calendarOperationCreate,
				Title:     "Team sync",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute calendar create prerequisite: %v", err)
	}
	if len(createResult.StepStates) != 1 {
		t.Fatalf("expected one calendar create step, got %d", len(createResult.StepStates))
	}
	eventID := strings.TrimSpace(createResult.StepStates[0].Evidence["event_id"])
	if eventID == "" {
		t.Fatalf("expected create step evidence to include event_id")
	}

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:    "ws1",
		ApprovalPhrase: "GO AHEAD",
		NativeAction: &NativeAction{
			Connector: WorkflowCalendar,
			Operation: calendarOperationCancel,
			Calendar: &CalendarAction{
				Operation: calendarOperationCancel,
				EventID:   eventID,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute calendar intent with approval phrase: %v", err)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false when GO AHEAD is provided")
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
}

func TestExecuteCalendarUpdateTargetsStableEventID(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createResult, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: WorkflowCalendar,
			Operation: calendarOperationCreate,
			Calendar: &CalendarAction{
				Operation: calendarOperationCreate,
				Title:     "Project review",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute calendar create prerequisite: %v", err)
	}
	if len(createResult.StepStates) != 1 {
		t.Fatalf("expected one calendar create step, got %d", len(createResult.StepStates))
	}
	eventID := strings.TrimSpace(createResult.StepStates[0].Evidence["event_id"])
	if eventID == "" {
		t.Fatalf("expected create step evidence to include event_id")
	}

	updateResult, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		NativeAction: &NativeAction{
			Connector: WorkflowCalendar,
			Operation: calendarOperationUpdate,
			Calendar: &CalendarAction{
				Operation: calendarOperationUpdate,
				EventID:   eventID,
				Title:     "Project review (updated)",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute calendar update: %v", err)
	}
	if updateResult.TaskState != "completed" || updateResult.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", updateResult.TaskState, updateResult.RunState)
	}
	if len(updateResult.StepStates) != 1 {
		t.Fatalf("expected one calendar update step, got %d", len(updateResult.StepStates))
	}
	if updateResult.StepStates[0].CapabilityKey != "calendar_update" {
		t.Fatalf("expected calendar_update capability, got %s", updateResult.StepStates[0].CapabilityKey)
	}
	if updateResult.StepStates[0].Evidence["event_id"] != eventID {
		t.Fatalf("expected update to target event_id %q, got %q", eventID, updateResult.StepStates[0].Evidence["event_id"])
	}
}

func TestExecuteCalendarCreateDoesNotRequireApproval(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "schedule event with the team",
	})
	if err != nil {
		t.Fatalf("execute calendar create intent: %v", err)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false for calendar create operation")
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	if len(result.StepStates) != 1 {
		t.Fatalf("expected one calendar step for create operation, got %d", len(result.StepStates))
	}
	if result.StepStates[0].CapabilityKey != "calendar_create" {
		t.Fatalf("expected calendar_create capability, got %s", result.StepStates[0].CapabilityKey)
	}
}

func TestExecuteMailSendPlansSingleStep(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "send an email update to sam@example.com",
	})
	if err != nil {
		t.Fatalf("execute mail send intent: %v", err)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false for mail send operation")
	}
	if len(result.StepStates) != 1 {
		t.Fatalf("expected one mail step for send operation, got %d", len(result.StepStates))
	}
	if result.StepStates[0].CapabilityKey != "mail_send" {
		t.Fatalf("expected mail_send capability, got %s", result.StepStates[0].CapabilityKey)
	}
}

func TestPlanStepsMailUnreadSummaryUsesCanonicalCapability(t *testing.T) {
	steps, err := planSteps(Intent{
		Workflow: WorkflowMail,
		Action: NativeAction{
			Connector: WorkflowMail,
			Operation: mailOperationSummarizeUnread,
			Mail: &MailAction{
				Operation: mailOperationSummarizeUnread,
				Limit:     7,
			},
		},
	})
	if err != nil {
		t.Fatalf("plan mail unread summary steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one summary step, got %d", len(steps))
	}
	if steps[0].CapabilityKey != "mail_unread_summary" {
		t.Fatalf("expected mail_unread_summary capability, got %s", steps[0].CapabilityKey)
	}
	limit, ok := steps[0].Input["limit"].(int)
	if !ok || limit != 7 {
		t.Fatalf("expected limit=7 in step input, got %#v", steps[0].Input["limit"])
	}
}

func TestPlanStepsFinderFindUsesCanonicalCapability(t *testing.T) {
	steps, err := planSteps(Intent{
		Workflow: WorkflowFinder,
		Action: NativeAction{
			Connector: WorkflowFinder,
			Operation: finderOperationFind,
			Finder: &FinderAction{
				Operation: finderOperationFind,
				Query:     "budget report",
				RootPath:  "/tmp",
			},
		},
	})
	if err != nil {
		t.Fatalf("plan finder find steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one finder find step, got %d", len(steps))
	}
	if steps[0].CapabilityKey != "finder_find" {
		t.Fatalf("expected finder_find capability, got %s", steps[0].CapabilityKey)
	}
	if got := steps[0].Input["query"]; got != "budget report" {
		t.Fatalf("expected query input budget report, got %#v", got)
	}
	if got := steps[0].Input["root_path"]; got != "/tmp" {
		t.Fatalf("expected root_path /tmp, got %#v", got)
	}
}

func TestPlanStepsFinderPreviewUsesCanonicalCapability(t *testing.T) {
	steps, err := planSteps(Intent{
		Workflow: WorkflowFinder,
		Action: NativeAction{
			Connector: WorkflowFinder,
			Operation: finderOperationPreview,
			Finder: &FinderAction{
				Operation:  finderOperationPreview,
				TargetPath: "/tmp/report.txt",
			},
		},
	})
	if err != nil {
		t.Fatalf("plan finder preview steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one finder preview step, got %d", len(steps))
	}
	if steps[0].CapabilityKey != "finder_preview" {
		t.Fatalf("expected finder_preview capability, got %s", steps[0].CapabilityKey)
	}
}

func TestExecuteFinderListDoesNotAutoPlanDelete(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "list files /tmp",
	})
	if err != nil {
		t.Fatalf("execute finder list intent: %v", err)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false for finder list operation")
	}
	if len(result.StepStates) != 1 {
		t.Fatalf("expected one finder step for list operation, got %d", len(result.StepStates))
	}
	if result.StepStates[0].CapabilityKey != "finder_list" {
		t.Fatalf("expected finder_list capability, got %s", result.StepStates[0].CapabilityKey)
	}
}

func TestExecuteVoiceDestructiveRequiresInAppHandoffEvenWithGoAheadPhrase(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:            "ws1",
		RequestText:            "delete file /tmp/test.txt",
		Origin:                 types.ExecutionOriginVoice,
		ApprovalPhrase:         "GO AHEAD",
		InAppApprovalConfirmed: false,
	})
	if err != nil {
		t.Fatalf("execute voice destructive intent: %v", err)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval_required=true for unconfirmed voice destructive action")
	}
	if result.TaskState != "awaiting_approval" || result.RunState != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval states, got task=%s run=%s", result.TaskState, result.RunState)
	}
	foundVoiceHandoffSummary := false
	for _, state := range result.StepStates {
		if strings.Contains(strings.ToLower(state.Summary), "in-app approval handoff") {
			foundVoiceHandoffSummary = true
			break
		}
	}
	if !foundVoiceHandoffSummary {
		t.Fatalf("expected voice handoff summary, got %+v", result.StepStates)
	}
}

func TestExecuteVoiceDestructiveAllowsExecutionWhenInAppHandoffConfirmed(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID:            "ws1",
		RequestText:            "delete file /tmp/test.txt",
		Origin:                 types.ExecutionOriginVoice,
		InAppApprovalConfirmed: true,
	})
	if err != nil {
		t.Fatalf("execute confirmed voice destructive intent: %v", err)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false when voice handoff is confirmed")
	}
	if result.TaskState != "completed" || result.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", result.TaskState, result.RunState)
	}
}

func TestResumeAfterApprovalCompletesRun(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pending, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder intent: %v", err)
	}
	if !pending.ApprovalRequired {
		t.Fatalf("expected approval required")
	}

	resumed, err := engine.ResumeAfterApproval(ctx, ResumeRequest{
		WorkspaceID:       "ws1",
		ApprovalRequestID: pending.ApprovalRequestID,
		DecisionByActorID: "actor.requester.ws1",
		Phrase:            "GO AHEAD",
		CorrelationID:     "corr-approve",
	})
	if err != nil {
		t.Fatalf("resume after approval: %v", err)
	}
	if resumed.TaskState != "completed" || resumed.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", resumed.TaskState, resumed.RunState)
	}
}

func TestResumeAfterApprovalCanReenterSQLiteWithSingleWriterPool(t *testing.T) {
	db := setupAgentExecDB(t)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	registry := connectorregistry.New()
	finderAdapter := &engineSQLiteRoundTripFinderAdapter{db: db}
	if err := registry.Register(finderAdapter); err != nil {
		t.Fatalf("register finder roundtrip adapter: %v", err)
	}
	engine := NewSQLiteExecutionEngine(db, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pending, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder delete intent: %v", err)
	}
	if !pending.ApprovalRequired {
		t.Fatalf("expected approval required")
	}

	resumed, err := engine.ResumeAfterApproval(ctx, ResumeRequest{
		WorkspaceID:       "ws1",
		ApprovalRequestID: pending.ApprovalRequestID,
		DecisionByActorID: "actor.requester.ws1",
		Phrase:            "GO AHEAD",
		CorrelationID:     "corr-resume-roundtrip",
	})
	if err != nil {
		t.Fatalf("resume after approval with roundtrip adapter: %v", err)
	}
	if resumed.TaskState != "completed" || resumed.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", resumed.TaskState, resumed.RunState)
	}
	if finderAdapter.calls != 1 {
		t.Fatalf("expected one finder adapter execute call, got %d", finderAdapter.calls)
	}
}

func TestResumeAfterApprovalRejectsNonExactPhrase(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pending, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder intent: %v", err)
	}

	_, err = engine.ResumeAfterApproval(ctx, ResumeRequest{
		WorkspaceID:       "ws1",
		ApprovalRequestID: pending.ApprovalRequestID,
		DecisionByActorID: "actor.requester.ws1",
		Phrase:            "go ahead",
	})
	if err == nil {
		t.Fatalf("expected non-exact phrase rejection")
	}

	var decision sql.NullString
	queryErr := db.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, pending.ApprovalRequestID).Scan(&decision)
	if queryErr != nil {
		t.Fatalf("query approval decision: %v", queryErr)
	}
	if decision.Valid {
		t.Fatalf("expected no approval decision for rejected phrase, got %s", decision.String)
	}
}

func TestResumeAfterApprovalRejectsUnauthorizedDecisionActor(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pending, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder intent: %v", err)
	}
	if !pending.ApprovalRequired {
		t.Fatalf("expected approval required")
	}

	_, err = engine.ResumeAfterApproval(ctx, ResumeRequest{
		WorkspaceID:       "ws1",
		ApprovalRequestID: pending.ApprovalRequestID,
		DecisionByActorID: "actor.approver",
		Phrase:            "GO AHEAD",
	})
	if err == nil {
		t.Fatalf("expected unauthorized decision actor rejection")
	}
	if !strings.Contains(err.Error(), "approval denied") {
		t.Fatalf("expected approval denied error, got %v", err)
	}

	var decision sql.NullString
	queryErr := db.QueryRow(`SELECT decision FROM approval_requests WHERE id = ?`, pending.ApprovalRequestID).Scan(&decision)
	if queryErr != nil {
		t.Fatalf("query approval decision: %v", queryErr)
	}
	if decision.Valid {
		t.Fatalf("expected no approval decision for unauthorized actor, got %s", decision.String)
	}
}

func TestResumeAfterApprovalAllowsDelegatedDecisionActor(t *testing.T) {
	db := setupAgentExecDB(t)
	engine := NewSQLiteExecutionEngine(db, buildSelector(t))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pending, err := engine.Execute(ctx, ExecuteRequest{
		WorkspaceID: "ws1",
		RequestText: "delete file /tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("execute finder intent: %v", err)
	}
	if !pending.ApprovalRequired {
		t.Fatalf("expected approval required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
	`, "actor.approver", "ws1", "actor.approver", now, now); err != nil {
		t.Fatalf("insert delegated approval actor: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'ACTIVE', ?, ?)
	`, "wp.approver", "ws1", "actor.approver", now, now); err != nil {
		t.Fatalf("insert delegated approval principal: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO delegation_rules(
			id, workspace_id, from_actor_id, to_actor_id, scope_type, scope_key, status, created_at, expires_at
		) VALUES (?, ?, ?, ?, 'APPROVAL', NULL, 'ACTIVE', ?, NULL)
	`, "dr.approval", "ws1", "actor.requester.ws1", "actor.approver", now); err != nil {
		t.Fatalf("insert approval delegation rule: %v", err)
	}

	resumed, err := engine.ResumeAfterApproval(ctx, ResumeRequest{
		WorkspaceID:       "ws1",
		ApprovalRequestID: pending.ApprovalRequestID,
		DecisionByActorID: "actor.approver",
		Phrase:            "GO AHEAD",
		CorrelationID:     "corr-delegated-approve",
	})
	if err != nil {
		t.Fatalf("resume after delegated approval: %v", err)
	}
	if resumed.TaskState != "completed" || resumed.RunState != "completed" {
		t.Fatalf("expected completed states, got task=%s run=%s", resumed.TaskState, resumed.RunState)
	}
}
