package connectorflow

import (
	"context"
	"fmt"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type ConnectorSelector interface {
	SelectByCapability(capabilityKey string, preferredAdapterID string) (connectorcontract.Adapter, error)
}

type MailHappyPathService struct {
	selector ConnectorSelector
}

func NewMailHappyPathService(selector ConnectorSelector) *MailHappyPathService {
	return &MailHappyPathService{selector: selector}
}

func (s *MailHappyPathService) Execute(ctx context.Context, request types.MailHappyPathRequest) (types.MailHappyPathResult, error) {
	if s.selector == nil {
		return types.MailHappyPathResult{}, fmt.Errorf("connector selector is required")
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return types.MailHappyPathResult{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(request.RunID) == "" {
		return types.MailHappyPathResult{}, fmt.Errorf("run id is required")
	}

	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID:      request.WorkspaceID,
		RunID:            request.RunID,
		CorrelationID:    request.CorrelationID,
		RequestedByActor: request.RequestedByActor,
		SubjectPrincipal: request.SubjectPrincipal,
		ActingAsActor:    request.ActingAsActor,
		SourceChannel:    "app_chat",
	}

	draftTrace, err := s.executeMailStep(ctx, request.PreferredAdapterID, execCtx, "mail_draft", 0, "Draft email")
	if err != nil {
		return types.MailHappyPathResult{}, err
	}
	sendTrace, err := s.executeMailStep(ctx, request.PreferredAdapterID, execCtx, "mail_send", 1, "Send email")
	if err != nil {
		return types.MailHappyPathResult{}, err
	}
	replyTrace, err := s.executeMailStep(ctx, request.PreferredAdapterID, execCtx, "mail_reply", 2, "Reply to thread")
	if err != nil {
		return types.MailHappyPathResult{}, err
	}

	return types.MailHappyPathResult{
		DraftTrace: draftTrace,
		SendTrace:  sendTrace,
		ReplyTrace: replyTrace,
	}, nil
}

func (s *MailHappyPathService) executeMailStep(
	ctx context.Context,
	preferredAdapterID string,
	execCtx connectorcontract.ExecutionContext,
	capability string,
	stepIndex int,
	stepName string,
) (types.MailStepTrace, error) {
	adapter, err := s.selector.SelectByCapability(capability, preferredAdapterID)
	if err != nil {
		return types.MailStepTrace{}, err
	}

	step := connectorcontract.TaskStep{
		ID:            fmt.Sprintf("mail-%s-%d", capability, stepIndex),
		RunID:         execCtx.RunID,
		StepIndex:     stepIndex,
		Name:          stepName,
		Status:        shared.TaskStepStatusPending,
		CapabilityKey: capability,
		Input: map[string]any{
			"recipient": "recipient@example.com",
			"subject":   "Connector happy path update",
			"body":      "Connector happy path mail body.",
		},
	}
	result, execErr := adapter.ExecuteStep(ctx, execCtx, step)
	if execErr != nil {
		return types.MailStepTrace{}, execErr
	}
	if result.Status != shared.TaskStepStatusCompleted {
		return types.MailStepTrace{}, fmt.Errorf("mail step %s did not complete", capability)
	}

	trace := types.MailStepTrace{
		StepName:      step.Name,
		CapabilityKey: capability,
		AdapterID:     adapter.Metadata().ID,
		Summary:       result.Summary,
		Evidence:      map[string]string{},
	}
	for key, value := range result.Evidence {
		trace.Evidence[key] = value
	}
	return trace, nil
}
