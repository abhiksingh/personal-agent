package calendar

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

func TestExecuteStepCreateUpdateCancelUsesStableEventIdentity(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("calendar.test")
	baseCtx := connectorcontract.ExecutionContext{
		WorkspaceID:      "ws_calendar",
		TaskID:           "task-1",
		RunID:            "run-1",
		CorrelationID:    "corr-1",
		RequestedByActor: "actor.requester",
		SubjectPrincipal: "actor.subject",
		ActingAsActor:    "actor.subject",
	}

	createCtx := baseCtx
	createCtx.StepID = "step-create-1"
	createStep := connectorcontract.TaskStep{
		ID:            "step-create-1",
		CapabilityKey: CapabilityCreate,
		Name:          "Create event",
		Input: map[string]any{
			"title": "Create event title",
			"notes": "Create event notes",
		},
	}
	createFirst, err := adapter.ExecuteStep(context.Background(), createCtx, createStep)
	if err != nil {
		t.Fatalf("execute create step: %v", err)
	}
	createSecond, err := adapter.ExecuteStep(context.Background(), createCtx, createStep)
	if err != nil {
		t.Fatalf("execute create step second time: %v", err)
	}
	if createFirst.Evidence["event_id"] == "" {
		t.Fatalf("expected event_id evidence")
	}
	if createFirst.Evidence["event_id"] != createSecond.Evidence["event_id"] {
		t.Fatalf("expected idempotent event_id, got %q vs %q", createFirst.Evidence["event_id"], createSecond.Evidence["event_id"])
	}
	if createFirst.Evidence["provider"] != "apple-calendar-dry-run" {
		t.Fatalf("expected apple-calendar-dry-run provider, got %q", createFirst.Evidence["provider"])
	}
	eventID := createFirst.Evidence["event_id"]
	if strings.TrimSpace(eventID) == "" {
		t.Fatalf("expected non-empty event_id")
	}
	if _, err := os.Stat(createFirst.Evidence["record_path"]); err != nil {
		t.Fatalf("expected create record file to exist: %v", err)
	}
	eventPath := createFirst.Evidence["event_path"]
	if _, err := os.Stat(eventPath); err != nil {
		t.Fatalf("expected create event record file to exist: %v", err)
	}

	updateCtx := baseCtx
	updateCtx.StepID = "step-update-1"
	updateStep := connectorcontract.TaskStep{
		ID:            "step-update-1",
		CapabilityKey: CapabilityUpdate,
		Name:          "Update event",
		Input: map[string]any{
			"event_id": eventID,
			"title":    "Update event title",
			"notes":    "Update event notes",
		},
	}
	updateResult, err := adapter.ExecuteStep(context.Background(), updateCtx, updateStep)
	if err != nil {
		t.Fatalf("execute update step: %v", err)
	}
	if updateResult.Evidence["event_id"] != eventID {
		t.Fatalf("expected update event_id=%q, got %q", eventID, updateResult.Evidence["event_id"])
	}
	if _, err := os.Stat(updateResult.Evidence["record_path"]); err != nil {
		t.Fatalf("expected update record file to exist: %v", err)
	}

	cancelCtx := baseCtx
	cancelCtx.StepID = "step-cancel-1"
	cancelStep := connectorcontract.TaskStep{
		ID:            "step-cancel-1",
		CapabilityKey: CapabilityCancel,
		Name:          "Cancel event",
		Input: map[string]any{
			"event_id": eventID,
		},
	}
	cancelResult, err := adapter.ExecuteStep(context.Background(), cancelCtx, cancelStep)
	if err != nil {
		t.Fatalf("execute cancel step: %v", err)
	}
	if cancelResult.Evidence["event_id"] != eventID {
		t.Fatalf("expected cancel event_id=%q, got %q", eventID, cancelResult.Evidence["event_id"])
	}
	if _, err := os.Stat(cancelResult.Evidence["record_path"]); err != nil {
		t.Fatalf("expected cancel record file to exist: %v", err)
	}
	if cancelResult.Evidence["event_path"] != eventPath {
		t.Fatalf("expected cancel event_path to match create event_path")
	}

	eventRaw, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("read calendar event record: %v", err)
	}
	var eventRecord calendarEventRecord
	if err := json.Unmarshal(eventRaw, &eventRecord); err != nil {
		t.Fatalf("decode calendar event record: %v", err)
	}
	if eventRecord.EventID != eventID {
		t.Fatalf("expected persisted event_id=%q, got %q", eventID, eventRecord.EventID)
	}
	if eventRecord.Title != "Update event title" {
		t.Fatalf("expected persisted updated title, got %q", eventRecord.Title)
	}
	if eventRecord.Status != "cancelled" {
		t.Fatalf("expected persisted cancelled status, got %q", eventRecord.Status)
	}
}

func TestExecuteStepUpdateRequiresExistingEventID(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("calendar.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_calendar",
		StepID:      "step-update-unknown",
	}, connectorcontract.TaskStep{
		ID:            "step-update-unknown",
		CapabilityKey: CapabilityUpdate,
		Name:          "Update event",
		Input: map[string]any{
			"event_id": "event-missing-1",
			"title":    "new title",
		},
	})
	if err == nil {
		t.Fatalf("expected update to fail when event_id does not exist")
	}
}

func TestExecuteStepCancelRejectsMissingEventID(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("calendar.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_calendar",
		StepID:      "step-cancel-missing",
	}, connectorcontract.TaskStep{
		ID:            "step-cancel-missing",
		CapabilityKey: CapabilityCancel,
		Name:          "Cancel event",
		Input: map[string]any{
			"title": "ignored",
		},
	})
	if err == nil {
		t.Fatalf("expected cancel to fail when event_id is missing")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "event_id") {
		t.Fatalf("expected missing event_id error, got %v", err)
	}
}

func TestExecuteStepRejectsUnsupportedCapability(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	adapter := NewAdapter("calendar.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_calendar",
		StepID:      "step-unsupported",
	}, connectorcontract.TaskStep{
		ID:            "step-unsupported",
		CapabilityKey: "calendar_unknown",
		Name:          "Unsupported calendar step",
	})
	if err == nil {
		t.Fatalf("expected unsupported capability error")
	}
}

func TestExecuteStepExecuteProbeBypassesStepResultCache(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("calendar.test")
	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws_calendar",
		TaskID:      "task-probe",
		RunID:       "run-probe",
		StepID:      "connector.execute_probe",
	}
	step := connectorcontract.TaskStep{
		ID:            "connector.execute_probe",
		CapabilityKey: capabilityExecuteProbe,
		Name:          "Connector Execute Probe",
		Input: map[string]any{
			"url": "https://example.com",
		},
	}

	first, err := adapter.ExecuteStep(context.Background(), execCtx, step)
	if err != nil {
		t.Fatalf("execute probe first call: %v", err)
	}
	if first.Status != "completed" {
		t.Fatalf("expected probe completed status, got %s", first.Status)
	}
	if first.Evidence["probe"] != "calendar_automation" {
		t.Fatalf("expected probe evidence marker, got %+v", first.Evidence["probe"])
	}

	second, err := adapter.ExecuteStep(context.Background(), execCtx, step)
	if err != nil {
		t.Fatalf("execute probe second call: %v", err)
	}
	if second.Status != "completed" {
		t.Fatalf("expected probe completed status on second call, got %s", second.Status)
	}

	stepResultPath := localstate.StepResultPath("calendar", execCtx.WorkspaceID, execCtx, step)
	if _, statErr := os.Stat(stepResultPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected execute probe to bypass step-result caching at %s, stat err=%v", stepResultPath, statErr)
	}
}
