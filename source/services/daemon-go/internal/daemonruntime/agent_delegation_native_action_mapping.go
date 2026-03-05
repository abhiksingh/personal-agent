package daemonruntime

import (
	"strings"

	"personalagent/runtime/internal/core/service/agentexec"
	"personalagent/runtime/internal/transport"
)

func agentRunResponse(result agentexec.ExecuteResult) transport.AgentRunResponse {
	stepStates := make([]transport.AgentStepState, 0, len(result.StepStates))
	for _, step := range result.StepStates {
		stepStates = append(stepStates, transport.AgentStepState{
			StepID:        step.StepID,
			StepIndex:     step.StepIndex,
			Name:          step.Name,
			CapabilityKey: step.CapabilityKey,
			AdapterID:     step.AdapterID,
			Status:        step.Status,
			Summary:       step.Summary,
			Evidence:      cloneEvidence(step.Evidence),
		})
	}

	return transport.AgentRunResponse{
		Workflow:              result.Workflow,
		NativeAction:          mapAgentNativeAction(result.NativeAction),
		TaskID:                result.TaskID,
		RunID:                 result.RunID,
		TaskState:             result.TaskState,
		RunState:              result.RunState,
		ClarificationRequired: result.ClarificationRequired,
		ClarificationPrompt:   result.ClarificationPrompt,
		MissingSlots:          cloneStringSlice(result.MissingSlots),
		ApprovalRequired:      result.ApprovalRequired,
		ApprovalRequestID:     result.ApprovalRequestID,
		StepStates:            stepStates,
	}
}

func mapAgentNativeAction(action *agentexec.NativeAction) *transport.AgentNativeAction {
	if action == nil {
		return nil
	}
	mapped := &transport.AgentNativeAction{
		Connector: action.Connector,
		Operation: action.Operation,
	}
	if action.Mail != nil {
		mapped.Mail = &transport.AgentMailAction{
			Operation: action.Mail.Operation,
			Recipient: action.Mail.Recipient,
			Subject:   action.Mail.Subject,
			Body:      action.Mail.Body,
			Limit:     action.Mail.Limit,
		}
	}
	if action.Calendar != nil {
		mapped.Calendar = &transport.AgentCalendarAction{
			Operation: action.Calendar.Operation,
			EventID:   action.Calendar.EventID,
			Title:     action.Calendar.Title,
			Notes:     action.Calendar.Notes,
		}
	}
	if action.Messages != nil {
		mapped.Messages = &transport.AgentMessagesAction{
			Operation: action.Messages.Operation,
			Channel:   action.Messages.Channel,
			Recipient: action.Messages.Recipient,
			Body:      action.Messages.Body,
		}
	}
	if action.Browser != nil {
		mapped.Browser = &transport.AgentBrowserAction{
			Operation: action.Browser.Operation,
			TargetURL: action.Browser.TargetURL,
			Query:     action.Browser.Query,
		}
	}
	if action.Finder != nil {
		mapped.Finder = &transport.AgentFinderAction{
			Operation:  action.Finder.Operation,
			TargetPath: action.Finder.TargetPath,
			Query:      action.Finder.Query,
			RootPath:   action.Finder.RootPath,
		}
	}
	return mapped
}

func mapTransportNativeAction(action *transport.AgentNativeAction) (*agentexec.NativeAction, error) {
	if action == nil {
		return nil, nil
	}
	mapped := &agentexec.NativeAction{
		Connector: strings.TrimSpace(action.Connector),
		Operation: strings.TrimSpace(action.Operation),
	}
	if action.Mail != nil {
		mapped.Mail = &agentexec.MailAction{
			Operation: strings.TrimSpace(action.Mail.Operation),
			Recipient: strings.TrimSpace(action.Mail.Recipient),
			Subject:   strings.TrimSpace(action.Mail.Subject),
			Body:      strings.TrimSpace(action.Mail.Body),
			Limit:     action.Mail.Limit,
		}
	}
	if action.Calendar != nil {
		mapped.Calendar = &agentexec.CalendarAction{
			Operation: strings.TrimSpace(action.Calendar.Operation),
			EventID:   strings.TrimSpace(action.Calendar.EventID),
			Title:     strings.TrimSpace(action.Calendar.Title),
			Notes:     strings.TrimSpace(action.Calendar.Notes),
		}
	}
	if action.Messages != nil {
		mapped.Messages = &agentexec.MessagesAction{
			Operation: strings.TrimSpace(action.Messages.Operation),
			Channel:   strings.TrimSpace(action.Messages.Channel),
			Recipient: strings.TrimSpace(action.Messages.Recipient),
			Body:      strings.TrimSpace(action.Messages.Body),
		}
	}
	if action.Browser != nil {
		mapped.Browser = &agentexec.BrowserAction{
			Operation: strings.TrimSpace(action.Browser.Operation),
			TargetURL: strings.TrimSpace(action.Browser.TargetURL),
			Query:     strings.TrimSpace(action.Browser.Query),
		}
	}
	if action.Finder != nil {
		mapped.Finder = &agentexec.FinderAction{
			Operation:  strings.TrimSpace(action.Finder.Operation),
			TargetPath: strings.TrimSpace(action.Finder.TargetPath),
			Query:      strings.TrimSpace(action.Finder.Query),
			RootPath:   strings.TrimSpace(action.Finder.RootPath),
		}
	}
	normalized, err := agentexec.IntentFromNativeAction(mapped)
	if err != nil {
		return nil, err
	}
	return cloneNativeActionFromIntent(normalized.Action), nil
}

func cloneNativeActionFromIntent(action agentexec.NativeAction) *agentexec.NativeAction {
	cloned := &agentexec.NativeAction{
		Connector: strings.TrimSpace(action.Connector),
		Operation: strings.TrimSpace(action.Operation),
	}
	if action.Mail != nil {
		cloned.Mail = &agentexec.MailAction{
			Operation: strings.TrimSpace(action.Mail.Operation),
			Recipient: strings.TrimSpace(action.Mail.Recipient),
			Subject:   strings.TrimSpace(action.Mail.Subject),
			Body:      strings.TrimSpace(action.Mail.Body),
			Limit:     action.Mail.Limit,
		}
	}
	if action.Calendar != nil {
		cloned.Calendar = &agentexec.CalendarAction{
			Operation: strings.TrimSpace(action.Calendar.Operation),
			EventID:   strings.TrimSpace(action.Calendar.EventID),
			Title:     strings.TrimSpace(action.Calendar.Title),
			Notes:     strings.TrimSpace(action.Calendar.Notes),
		}
	}
	if action.Messages != nil {
		cloned.Messages = &agentexec.MessagesAction{
			Operation: strings.TrimSpace(action.Messages.Operation),
			Channel:   strings.TrimSpace(action.Messages.Channel),
			Recipient: strings.TrimSpace(action.Messages.Recipient),
			Body:      strings.TrimSpace(action.Messages.Body),
		}
	}
	if action.Browser != nil {
		cloned.Browser = &agentexec.BrowserAction{
			Operation: strings.TrimSpace(action.Browser.Operation),
			TargetURL: strings.TrimSpace(action.Browser.TargetURL),
			Query:     strings.TrimSpace(action.Browser.Query),
		}
	}
	if action.Finder != nil {
		cloned.Finder = &agentexec.FinderAction{
			Operation:  strings.TrimSpace(action.Finder.Operation),
			TargetPath: strings.TrimSpace(action.Finder.TargetPath),
			Query:      strings.TrimSpace(action.Finder.Query),
			RootPath:   strings.TrimSpace(action.Finder.RootPath),
		}
	}
	return cloned
}

func cloneEvidence(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	return append([]string(nil), input...)
}
