package connectorflow

import (
	"context"
	"testing"

	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/types"
)

func TestExecuteCalendarHappyPathCreateUpdateCancelWithTraceEvidence(t *testing.T) {
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	registry := connectorregistry.New()
	if err := registry.Register(calendaradapter.NewAdapter("calendar.mock")); err != nil {
		t.Fatalf("register calendar adapter: %v", err)
	}

	service := NewCalendarHappyPathService(registry)
	result, err := service.Execute(context.Background(), types.CalendarHappyPathRequest{
		WorkspaceID:      "ws_calendar",
		RunID:            "run_calendar_happy_path",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_calendar_happy_path",
	})
	if err != nil {
		t.Fatalf("execute calendar happy path: %v", err)
	}

	if result.CreateTrace.CapabilityKey != "calendar_create" {
		t.Fatalf("expected create capability calendar_create, got %s", result.CreateTrace.CapabilityKey)
	}
	if result.UpdateTrace.CapabilityKey != "calendar_update" {
		t.Fatalf("expected update capability calendar_update, got %s", result.UpdateTrace.CapabilityKey)
	}
	if result.CancelTrace.CapabilityKey != "calendar_cancel" {
		t.Fatalf("expected cancel capability calendar_cancel, got %s", result.CancelTrace.CapabilityKey)
	}

	if result.CreateTrace.AdapterID != "calendar.mock" || result.UpdateTrace.AdapterID != "calendar.mock" || result.CancelTrace.AdapterID != "calendar.mock" {
		t.Fatalf("expected all steps to execute via calendar.mock adapter")
	}

	if result.CreateTrace.Evidence["event_id"] == "" {
		t.Fatalf("expected create trace evidence to include event_id")
	}
	if result.UpdateTrace.Evidence["event_id"] == "" {
		t.Fatalf("expected update trace evidence to include event_id")
	}
	if result.CancelTrace.Evidence["event_id"] == "" {
		t.Fatalf("expected cancel trace evidence to include event_id")
	}
	if result.UpdateTrace.Evidence["event_id"] != result.CreateTrace.Evidence["event_id"] {
		t.Fatalf("expected update to target created event_id")
	}
	if result.CancelTrace.Evidence["event_id"] != result.CreateTrace.Evidence["event_id"] {
		t.Fatalf("expected cancel to target created event_id")
	}
}
