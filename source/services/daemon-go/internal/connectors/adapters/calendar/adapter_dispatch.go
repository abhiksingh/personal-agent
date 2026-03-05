package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

func (a *Adapter) executeUncached(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep, stepResultPath string) (connectorcontract.StepExecutionResult, error) {
	workspaceRoot := adapterscaffold.WorkspaceRootFromStepResultPath(stepResultPath)
	stepToken := adapterhelpers.StableStepToken(execCtx, step)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	calendarName := resolveCalendarName()
	provider := "apple-calendar"
	if isCalendarAutomationDryRunEnabled() {
		provider = "apple-calendar-dry-run"
	}

	switch step.CapabilityKey {
	case CapabilityCreate:
		input, inputErr := resolveCalendarCreateInput(step)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar create input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		eventID := "event-" + stepToken
		operation, err := executeCalendarOperation(ctx, calendarOperationRequest{
			Mode:         "create",
			CalendarName: calendarName,
			EventID:      eventID,
			Title:        input.Title,
			Notes:        input.Notes,
		})
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar create automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute calendar create: %w", err)
		}
		eventRecord := calendarEventRecord{
			EventID:           eventID,
			WorkspaceID:       execCtx.WorkspaceID,
			CalendarName:      calendarName,
			Title:             input.Title,
			Notes:             input.Notes,
			Status:            "active",
			LastOperationID:   operation.OperationID,
			LastOperationMode: "create",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		eventRecordPath := calendarEventRecordPath(workspaceRoot, eventID)
		if err := localstate.WriteJSONFile(eventRecordPath, eventRecord); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar create write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar event record: %w", err)
		}
		record := buildCalendarOperationRecord(
			CapabilityCreate,
			operation.OperationID,
			eventID,
			stepToken,
			execCtx,
			now,
			operation.Transport,
			calendarName,
			input.Title,
			input.Notes,
		)
		recordPath, err := writeCalendarOperationRecord(workspaceRoot, CapabilityCreate, operation.OperationID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar create write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar create record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "calendar event created via Calendar automation",
			Retryable: false,
			Evidence: map[string]string{
				"event_id":      eventID,
				"operation_id":  operation.OperationID,
				"workspace_id":  execCtx.WorkspaceID,
				"provider":      provider,
				"transport":     operation.Transport,
				"calendar_name": calendarName,
				"event_path":    eventRecordPath,
				"record_path":   recordPath,
			},
			Output: map[string]any{
				"event_id":     eventID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilityUpdate:
		input, inputErr := resolveCalendarUpdateInput(step)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar update input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		eventRecordPath := calendarEventRecordPath(workspaceRoot, input.EventID)
		existingEvent, err := loadCalendarEventRecord(eventRecordPath)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar event update target not found",
				Retryable:   false,
				ErrorReason: "not_found",
			}, fmt.Errorf("load calendar event record: %w", err)
		}
		if strings.EqualFold(strings.TrimSpace(existingEvent.Status), "cancelled") {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar event is already cancelled",
				Retryable:   false,
				ErrorReason: "invalid_state",
			}, fmt.Errorf("calendar event %s is already cancelled", input.EventID)
		}
		resolvedTitle := strings.TrimSpace(input.Title)
		if resolvedTitle == "" {
			resolvedTitle = strings.TrimSpace(existingEvent.Title)
		}
		resolvedNotes := strings.TrimSpace(input.Notes)
		if resolvedNotes == "" {
			resolvedNotes = strings.TrimSpace(existingEvent.Notes)
		}
		operation, err := executeCalendarOperation(ctx, calendarOperationRequest{
			Mode:         "update",
			CalendarName: strings.TrimSpace(existingEvent.CalendarName),
			EventID:      input.EventID,
			Title:        resolvedTitle,
			Notes:        resolvedNotes,
		})
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar update automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute calendar update: %w", err)
		}
		existingEvent.Title = resolvedTitle
		existingEvent.Notes = resolvedNotes
		existingEvent.Status = "active"
		existingEvent.LastOperationID = operation.OperationID
		existingEvent.LastOperationMode = "update"
		existingEvent.UpdatedAt = now
		if err := localstate.WriteJSONFile(eventRecordPath, existingEvent); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar update write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar event record: %w", err)
		}
		record := buildCalendarOperationRecord(
			CapabilityUpdate,
			operation.OperationID,
			input.EventID,
			stepToken,
			execCtx,
			now,
			operation.Transport,
			existingEvent.CalendarName,
			resolvedTitle,
			resolvedNotes,
		)
		recordPath, err := writeCalendarOperationRecord(workspaceRoot, CapabilityUpdate, operation.OperationID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar update write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar update record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "calendar event updated via Calendar automation",
			Retryable: false,
			Evidence: map[string]string{
				"event_id":      input.EventID,
				"operation_id":  operation.OperationID,
				"provider":      provider,
				"transport":     operation.Transport,
				"calendar_name": existingEvent.CalendarName,
				"event_path":    eventRecordPath,
				"record_path":   recordPath,
			},
			Output: map[string]any{
				"event_id":     input.EventID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilityCancel:
		input, inputErr := resolveCalendarCancelInput(step)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar cancel input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		eventRecordPath := calendarEventRecordPath(workspaceRoot, input.EventID)
		existingEvent, err := loadCalendarEventRecord(eventRecordPath)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar event cancel target not found",
				Retryable:   false,
				ErrorReason: "not_found",
			}, fmt.Errorf("load calendar event record: %w", err)
		}
		if strings.EqualFold(strings.TrimSpace(existingEvent.Status), "cancelled") {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar event is already cancelled",
				Retryable:   false,
				ErrorReason: "invalid_state",
			}, fmt.Errorf("calendar event %s is already cancelled", input.EventID)
		}
		operation, err := executeCalendarOperation(ctx, calendarOperationRequest{
			Mode:         "cancel",
			CalendarName: strings.TrimSpace(existingEvent.CalendarName),
			EventID:      input.EventID,
			Title:        strings.TrimSpace(existingEvent.Title),
			Notes:        strings.TrimSpace(existingEvent.Notes),
		})
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar cancel automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute calendar cancel: %w", err)
		}
		existingEvent.Status = "cancelled"
		existingEvent.LastOperationID = operation.OperationID
		existingEvent.LastOperationMode = "cancel"
		existingEvent.UpdatedAt = now
		if err := localstate.WriteJSONFile(eventRecordPath, existingEvent); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar cancel write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar event record: %w", err)
		}
		record := buildCalendarOperationRecord(
			CapabilityCancel,
			operation.OperationID,
			input.EventID,
			stepToken,
			execCtx,
			now,
			operation.Transport,
			existingEvent.CalendarName,
			existingEvent.Title,
			existingEvent.Notes,
		)
		recordPath, err := writeCalendarOperationRecord(workspaceRoot, CapabilityCancel, operation.OperationID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "calendar cancel write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write calendar cancel record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "calendar event cancelled via Calendar automation",
			Retryable: false,
			Evidence: map[string]string{
				"event_id":      input.EventID,
				"operation_id":  operation.OperationID,
				"reason":        "user_request",
				"provider":      provider,
				"transport":     operation.Transport,
				"calendar_name": existingEvent.CalendarName,
				"event_path":    eventRecordPath,
				"record_path":   recordPath,
			},
			Output: map[string]any{
				"event_id":     input.EventID,
				"operation_id": operation.OperationID,
			},
		}, nil
	default:
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "unsupported calendar capability",
			Retryable:   false,
			ErrorReason: "unsupported_capability",
		}, fmt.Errorf("unsupported calendar capability: %s", step.CapabilityKey)
	}
}

func (a *Adapter) executeCalendarExecuteProbe(ctx context.Context) (connectorcontract.StepExecutionResult, error) {
	if err := executeCalendarPermissionProbe(ctx); err != nil {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "calendar execute probe failed",
			Retryable:   true,
			ErrorReason: "automation_unavailable",
		}, fmt.Errorf("execute calendar permission probe: %w", err)
	}
	return connectorcontract.StepExecutionResult{
		Status:    shared.TaskStepStatusCompleted,
		Summary:   "calendar execute probe completed",
		Retryable: false,
		Evidence: map[string]string{
			"transport": transportCalendarAppleEvents,
			"probe":     "calendar_automation",
		},
	}, nil
}
