package agentexec

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubModelIntentExtractor struct {
	candidate ModelIntentCandidate
	err       error
}

func (s stubModelIntentExtractor) ExtractIntent(_ context.Context, _ string, _ string) (ModelIntentCandidate, error) {
	if s.err != nil {
		return ModelIntentCandidate{}, s.err
	}
	return s.candidate, nil
}

func TestInterpretIntentBrowser(t *testing.T) {
	intent, err := InterpretIntent("open https://example.com please")
	if err != nil {
		t.Fatalf("interpret browser intent: %v", err)
	}
	if intent.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow, got %s", intent.Workflow)
	}
	if intent.RequiresClarification() {
		t.Fatalf("did not expect clarification for fully specified browser intent")
	}
	if intent.Action.Browser == nil || intent.Action.Browser.TargetURL != "https://example.com" {
		t.Fatalf("expected native browser action with URL, got %+v", intent.Action.Browser)
	}
	if strings.TrimSpace(intent.Action.Browser.Query) == "" {
		t.Fatalf("expected browser intent query context to be populated")
	}
}

func TestInterpretIntentBrowserMissingURLReturnsClarification(t *testing.T) {
	intent, err := InterpretIntent("open the website for docs")
	if err != nil {
		t.Fatalf("interpret browser clarification intent: %v", err)
	}
	if intent.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow, got %s", intent.Workflow)
	}
	if !intent.RequiresClarification() {
		t.Fatalf("expected clarification for browser request without URL")
	}
	if !containsString(intent.MissingSlots, slotTargetURL) {
		t.Fatalf("expected missing slot %q, got %v", slotTargetURL, intent.MissingSlots)
	}
}

func TestInterpretIntentMail(t *testing.T) {
	intent, err := InterpretIntent("send an email update to the team")
	if err != nil {
		t.Fatalf("interpret mail intent: %v", err)
	}
	if intent.Workflow != WorkflowMail {
		t.Fatalf("expected mail workflow, got %s", intent.Workflow)
	}
	if intent.Action.Mail == nil {
		t.Fatalf("expected mail native action payload")
	}
	if intent.Action.Mail.Operation != mailOperationSend {
		t.Fatalf("expected mail operation %q, got %q", mailOperationSend, intent.Action.Mail.Operation)
	}
}

func TestInterpretIntentMailReplyOperation(t *testing.T) {
	intent, err := InterpretIntent("reply to that email thread")
	if err != nil {
		t.Fatalf("interpret mail reply intent: %v", err)
	}
	if intent.Workflow != WorkflowMail {
		t.Fatalf("expected mail workflow, got %s", intent.Workflow)
	}
	if intent.Action.Mail == nil {
		t.Fatalf("expected mail native action payload")
	}
	if intent.Action.Mail.Operation != mailOperationReply {
		t.Fatalf("expected mail operation %q, got %q", mailOperationReply, intent.Action.Mail.Operation)
	}
}

func TestInterpretIntentMailUnreadSummaryOperation(t *testing.T) {
	intent, err := InterpretIntent("summarize unread emails 3")
	if err != nil {
		t.Fatalf("interpret mail unread summary intent: %v", err)
	}
	if intent.Workflow != WorkflowMail {
		t.Fatalf("expected mail workflow, got %s", intent.Workflow)
	}
	if intent.Action.Mail == nil {
		t.Fatalf("expected mail native action payload")
	}
	if intent.Action.Mail.Operation != mailOperationSummarizeUnread {
		t.Fatalf("expected mail operation %q, got %q", mailOperationSummarizeUnread, intent.Action.Mail.Operation)
	}
	if intent.Action.Mail.Limit != 3 {
		t.Fatalf("expected extracted unread summary limit=3, got %d", intent.Action.Mail.Limit)
	}
	if strings.TrimSpace(intent.Action.Mail.Body) != "" {
		t.Fatalf("expected summarize_unread body to be empty, got %q", intent.Action.Mail.Body)
	}
}

func TestInterpretIntentCalendarCancelOperation(t *testing.T) {
	intent, err := InterpretIntent("cancel the calendar event tomorrow")
	if err != nil {
		t.Fatalf("interpret calendar cancel intent: %v", err)
	}
	if intent.Workflow != WorkflowCalendar {
		t.Fatalf("expected calendar workflow, got %s", intent.Workflow)
	}
	if intent.Action.Calendar == nil {
		t.Fatalf("expected calendar native action payload")
	}
	if intent.Action.Calendar.Operation != calendarOperationCancel {
		t.Fatalf("expected calendar operation %q, got %q", calendarOperationCancel, intent.Action.Calendar.Operation)
	}
	if !intent.RequiresClarification() {
		t.Fatalf("expected calendar cancel intent to require clarification without event_id")
	}
	if !containsString(intent.MissingSlots, slotCalendarEventID) {
		t.Fatalf("expected missing slot %q, got %v", slotCalendarEventID, intent.MissingSlots)
	}
}

func TestInterpretIntentCalendarCancelWithEventIDIsExecutable(t *testing.T) {
	intent, err := InterpretIntent("cancel calendar event id event-team-sync-1")
	if err != nil {
		t.Fatalf("interpret calendar cancel intent with event id: %v", err)
	}
	if intent.Workflow != WorkflowCalendar {
		t.Fatalf("expected calendar workflow, got %s", intent.Workflow)
	}
	if intent.Action.Calendar == nil {
		t.Fatalf("expected calendar native action payload")
	}
	if intent.Action.Calendar.Operation != calendarOperationCancel {
		t.Fatalf("expected calendar operation %q, got %q", calendarOperationCancel, intent.Action.Calendar.Operation)
	}
	if intent.Action.Calendar.EventID != "event-team-sync-1" {
		t.Fatalf("expected parsed event_id event-team-sync-1, got %q", intent.Action.Calendar.EventID)
	}
	if intent.RequiresClarification() {
		t.Fatalf("did not expect clarification when cancel includes event_id")
	}
}

func TestInterpretIntentFinderMissingTargetInformationReturnsClarification(t *testing.T) {
	intent, err := InterpretIntent("delete file now")
	if err != nil {
		t.Fatalf("interpret finder clarification intent: %v", err)
	}
	if intent.Workflow != WorkflowFinder {
		t.Fatalf("expected finder workflow, got %s", intent.Workflow)
	}
	if !intent.RequiresClarification() {
		t.Fatalf("expected clarification for finder request without path/query")
	}
	if !containsString(intent.MissingSlots, slotFinderQuery) {
		t.Fatalf("expected missing slot %q, got %v", slotFinderQuery, intent.MissingSlots)
	}
}

func TestInterpretIntentFinderFindUsesQueryWithoutAbsolutePath(t *testing.T) {
	intent, err := InterpretIntent("find budget report")
	if err != nil {
		t.Fatalf("interpret finder find intent: %v", err)
	}
	if intent.Workflow != WorkflowFinder {
		t.Fatalf("expected finder workflow, got %s", intent.Workflow)
	}
	if intent.Action.Finder == nil {
		t.Fatalf("expected finder native action payload")
	}
	if intent.Action.Finder.Operation != finderOperationFind {
		t.Fatalf("expected finder operation %q, got %q", finderOperationFind, intent.Action.Finder.Operation)
	}
	if intent.Action.Finder.Query != "budget report" {
		t.Fatalf("expected finder query \"budget report\", got %q", intent.Action.Finder.Query)
	}
	if intent.RequiresClarification() {
		t.Fatalf("did not expect clarification for finder query request")
	}
}

func TestInterpretIntentMessagesReturnsSchemaAndClarification(t *testing.T) {
	intent, err := InterpretIntent(`send a text to +15550001111: "hello there"`)
	if err != nil {
		t.Fatalf("interpret messages intent: %v", err)
	}
	if intent.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow, got %s", intent.Workflow)
	}
	if intent.Action.Messages == nil {
		t.Fatalf("expected messages native action payload")
	}
	if intent.Action.Messages.Recipient != "+15550001111" {
		t.Fatalf("expected extracted recipient, got %q", intent.Action.Messages.Recipient)
	}
	if intent.Action.Messages.Body != "hello there" {
		t.Fatalf("expected extracted body, got %q", intent.Action.Messages.Body)
	}
	if !intent.RequiresClarification() {
		t.Fatalf("expected clarification for messages workflow")
	}
	if !containsString(intent.MissingSlots, slotMessageChannel) {
		t.Fatalf("expected missing slot %q, got %v", slotMessageChannel, intent.MissingSlots)
	}
}

func TestInterpretIntentMessagesWithExplicitChannelIsExecutable(t *testing.T) {
	intent, err := InterpretIntent(`send an sms to +15550001111: "hello there"`)
	if err != nil {
		t.Fatalf("interpret messages intent: %v", err)
	}
	if intent.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow, got %s", intent.Workflow)
	}
	if intent.Action.Messages == nil {
		t.Fatalf("expected messages native action payload")
	}
	if intent.Action.Messages.Channel != "sms" {
		t.Fatalf("expected resolved sms channel, got %q", intent.Action.Messages.Channel)
	}
	if intent.RequiresClarification() {
		t.Fatalf("did not expect clarification for fully specified messages intent")
	}
}

func TestModelAssistedIntentInterpreterUsesModelCandidateWhenValid(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:   WorkflowBrowser,
			TargetURL:  "https://example.com",
			Confidence: 0.95,
		},
	}, 0.6)

	intent, err := interpreter.Interpret(context.Background(), "ws1", "please help with this request")
	if err != nil {
		t.Fatalf("interpret model-assisted intent: %v", err)
	}
	if intent.Workflow != WorkflowBrowser {
		t.Fatalf("expected browser workflow from model candidate, got %s", intent.Workflow)
	}
	if intent.TargetURL != "https://example.com" {
		t.Fatalf("expected model target_url, got %s", intent.TargetURL)
	}
}

func TestModelAssistedIntentInterpreterReturnsMessagesClarificationCandidate(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:         WorkflowMessages,
			MessageRecipient: "+15550002222",
			MessageBody:      "hello from model",
			Confidence:       0.92,
		},
	}, 0.6)

	intent, err := interpreter.Interpret(context.Background(), "ws1", "send bob a text")
	if err != nil {
		t.Fatalf("interpret model-assisted messages intent: %v", err)
	}
	if intent.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow from model candidate, got %s", intent.Workflow)
	}
	if !intent.RequiresClarification() {
		t.Fatalf("expected messages clarification when channel is missing")
	}
	if !containsString(intent.MissingSlots, slotMessageChannel) {
		t.Fatalf("expected missing channel slot, got %v", intent.MissingSlots)
	}
}

func TestModelAssistedIntentInterpreterUsesResolvedMessagesChannelWhenPresent(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:         WorkflowMessages,
			MessageChannel:   "imessage",
			MessageRecipient: "+15550002222",
			MessageBody:      "hello from model",
			Confidence:       0.92,
		},
	}, 0.6)

	intent, err := interpreter.Interpret(context.Background(), "ws1", "send bob a text")
	if err != nil {
		t.Fatalf("interpret model-assisted messages intent: %v", err)
	}
	if intent.Workflow != WorkflowMessages {
		t.Fatalf("expected messages workflow from model candidate, got %s", intent.Workflow)
	}
	if intent.RequiresClarification() {
		t.Fatalf("expected executable messages intent when channel is provided")
	}
	if intent.Action.Messages == nil || intent.Action.Messages.Channel != "imessage" {
		t.Fatalf("expected resolved imessage channel, got %+v", intent.Action.Messages)
	}
}

func TestModelAssistedIntentInterpreterFallsBackToDeterministicOnExtractorError(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		err: errors.New("model unavailable"),
	}, 0.6)

	intent, err := interpreter.Interpret(context.Background(), "ws1", "send an email update")
	if err != nil {
		t.Fatalf("expected deterministic fallback success: %v", err)
	}
	if intent.Workflow != WorkflowMail {
		t.Fatalf("expected deterministic fallback workflow mail, got %s", intent.Workflow)
	}
}

func TestModelAssistedIntentInterpreterFallsBackToDeterministicOnLowConfidence(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		candidate: ModelIntentCandidate{
			Workflow:   WorkflowBrowser,
			TargetURL:  "https://example.com",
			Confidence: 0.2,
		},
	}, 0.6)

	intent, err := interpreter.Interpret(context.Background(), "ws1", "send an email update")
	if err != nil {
		t.Fatalf("expected deterministic fallback success: %v", err)
	}
	if intent.Workflow != WorkflowMail {
		t.Fatalf("expected mail workflow from fallback, got %s", intent.Workflow)
	}
}

func TestModelAssistedIntentInterpreterPreservesDeterministicFailureWhenUnresolved(t *testing.T) {
	interpreter := NewModelAssistedIntentInterpreter(stubModelIntentExtractor{
		err: errors.New("model unavailable"),
	}, 0.6)

	_, err := interpreter.Interpret(context.Background(), "ws1", "just do something")
	if err == nil {
		t.Fatalf("expected unresolved intent error")
	}
	if !strings.Contains(err.Error(), "unable to determine intent") {
		t.Fatalf("expected unresolved intent error, got %v", err)
	}
}

func TestInterpretIntentRegressionCorpus(t *testing.T) {
	cases := []struct {
		name              string
		request           string
		workflow          string
		requiresClarify   bool
		expectedConnector string
		expectedOperation string
	}{
		{
			name:              "mail",
			request:           "email the team an update",
			workflow:          WorkflowMail,
			requiresClarify:   false,
			expectedConnector: WorkflowMail,
			expectedOperation: mailOperationSend,
		},
		{
			name:              "calendar",
			request:           "schedule event with the team",
			workflow:          WorkflowCalendar,
			requiresClarify:   false,
			expectedConnector: WorkflowCalendar,
			expectedOperation: calendarOperationCreate,
		},
		{
			name:              "messages",
			request:           "send a text to +15550003333",
			workflow:          WorkflowMessages,
			requiresClarify:   true,
			expectedConnector: WorkflowMessages,
			expectedOperation: "send_message",
		},
		{
			name:              "browser",
			request:           "open https://example.com",
			workflow:          WorkflowBrowser,
			requiresClarify:   false,
			expectedConnector: WorkflowBrowser,
			expectedOperation: "open_extract_close",
		},
		{
			name:              "finder",
			request:           "delete file /tmp/demo.txt",
			workflow:          WorkflowFinder,
			requiresClarify:   false,
			expectedConnector: WorkflowFinder,
			expectedOperation: finderOperationDelete,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			intent, err := InterpretIntent(tc.request)
			if err != nil {
				t.Fatalf("interpret intent: %v", err)
			}
			if intent.Workflow != tc.workflow {
				t.Fatalf("expected workflow %s, got %s", tc.workflow, intent.Workflow)
			}
			if intent.RequiresClarification() != tc.requiresClarify {
				t.Fatalf("expected requiresClarification=%t, got %t", tc.requiresClarify, intent.RequiresClarification())
			}
			if intent.Action.Connector != tc.expectedConnector {
				t.Fatalf("expected connector %s, got %s", tc.expectedConnector, intent.Action.Connector)
			}
			if intent.Action.Operation != tc.expectedOperation {
				t.Fatalf("expected operation %s, got %s", tc.expectedOperation, intent.Action.Operation)
			}
		})
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
