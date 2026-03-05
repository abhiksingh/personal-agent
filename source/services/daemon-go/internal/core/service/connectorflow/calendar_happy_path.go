package connectorflow

import (
	"context"
	"fmt"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type CalendarHappyPathService struct {
	selector ConnectorSelector
}

func NewCalendarHappyPathService(selector ConnectorSelector) *CalendarHappyPathService {
	return &CalendarHappyPathService{selector: selector}
}

func (s *CalendarHappyPathService) Execute(ctx context.Context, request types.CalendarHappyPathRequest) (types.CalendarHappyPathResult, error) {
	if s.selector == nil {
		return types.CalendarHappyPathResult{}, fmt.Errorf("connector selector is required")
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return types.CalendarHappyPathResult{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(request.RunID) == "" {
		return types.CalendarHappyPathResult{}, fmt.Errorf("run id is required")
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

	createTrace, err := s.executeCalendarStep(ctx, request.PreferredAdapterID, execCtx, "calendar_create", 0, "Create event", nil)
	if err != nil {
		return types.CalendarHappyPathResult{}, err
	}
	eventID := strings.TrimSpace(createTrace.Evidence["event_id"])
	if eventID == "" {
		return types.CalendarHappyPathResult{}, fmt.Errorf("calendar create step did not return event_id evidence")
	}

	updateTrace, err := s.executeCalendarStep(ctx, request.PreferredAdapterID, execCtx, "calendar_update", 1, "Update event", map[string]any{
		"event_id": eventID,
		"title":    "Calendar happy path event (updated)",
		"notes":    "Updated by connector happy path service.",
	})
	if err != nil {
		return types.CalendarHappyPathResult{}, err
	}
	cancelTrace, err := s.executeCalendarStep(ctx, request.PreferredAdapterID, execCtx, "calendar_cancel", 2, "Cancel event", map[string]any{
		"event_id": eventID,
	})
	if err != nil {
		return types.CalendarHappyPathResult{}, err
	}

	return types.CalendarHappyPathResult{
		CreateTrace: createTrace,
		UpdateTrace: updateTrace,
		CancelTrace: cancelTrace,
	}, nil
}

func (s *CalendarHappyPathService) executeCalendarStep(
	ctx context.Context,
	preferredAdapterID string,
	execCtx connectorcontract.ExecutionContext,
	capability string,
	stepIndex int,
	stepName string,
	input map[string]any,
) (types.ConnectorStepTrace, error) {
	adapter, err := s.selector.SelectByCapability(capability, preferredAdapterID)
	if err != nil {
		return types.ConnectorStepTrace{}, err
	}

	step := connectorcontract.TaskStep{
		ID:            fmt.Sprintf("calendar-%s-%d", capability, stepIndex),
		RunID:         execCtx.RunID,
		StepIndex:     stepIndex,
		Name:          stepName,
		Status:        shared.TaskStepStatusPending,
		CapabilityKey: capability,
		Input:         map[string]any{},
	}
	switch capability {
	case "calendar_create":
		step.Input["title"] = "Calendar happy path event"
		step.Input["notes"] = "Created by connector happy path service."
	default:
		for key, value := range input {
			step.Input[key] = value
		}
	}
	result, execErr := adapter.ExecuteStep(ctx, execCtx, step)
	if execErr != nil {
		return types.ConnectorStepTrace{}, execErr
	}
	if result.Status != shared.TaskStepStatusCompleted {
		return types.ConnectorStepTrace{}, fmt.Errorf("calendar step %s did not complete", capability)
	}

	trace := types.ConnectorStepTrace{
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
