package daemonruntime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

func TestAutomationCommTriggerMetadataContract(t *testing.T) {
	service, _ := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	response, err := service.AutomationCommTriggerMetadata(ctx, transport.AutomationCommTriggerMetadataRequest{
		WorkspaceID: "ws1",
	})
	if err != nil {
		t.Fatalf("automation comm-trigger metadata: %v", err)
	}
	if response.TriggerType != "ON_COMM_EVENT" {
		t.Fatalf("expected trigger type ON_COMM_EVENT, got %q", response.TriggerType)
	}
	if response.RequiredDefaults.EventType != "MESSAGE" || response.RequiredDefaults.Direction != "INBOUND" || response.RequiredDefaults.AssistantEmitted {
		t.Fatalf("unexpected required defaults payload: %+v", response.RequiredDefaults)
	}
	if len(response.IdempotencyKeyFields) != 3 {
		t.Fatalf("expected idempotency key field contract, got %+v", response.IdempotencyKeyFields)
	}
	if response.Compatibility.PrincipalFilterBehavior == "" || response.Compatibility.KeywordMatchBehavior == "" {
		t.Fatalf("expected compatibility semantics metadata, got %+v", response.Compatibility)
	}
	if len(response.FilterSchema) == 0 {
		t.Fatalf("expected filter schema metadata entries")
	}
}

func TestAutomationCommTriggerValidateNormalizesTypedFilterAndReportsSubjectMismatch(t *testing.T) {
	service, _ := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	response, err := service.AutomationCommTriggerValidate(ctx, transport.AutomationCommTriggerValidateRequest{
		WorkspaceID:    "ws1",
		SubjectActorID: "actor.approver",
		Filter: &transport.AutomationCommTriggerFilter{
			Channels:          []string{" Twilio_SMS ", "twilio_sms"},
			PrincipalActorIDs: []string{"actor.requester"},
			SenderAllowlist:   []string{},
			ThreadIDs:         []string{},
			Keywords: transport.AutomationCommTriggerKeywordFilter{
				ContainsAny: []string{" Hello ", "hello"},
			},
		},
	})
	if err != nil {
		t.Fatalf("automation comm-trigger validate: %v", err)
	}
	if !response.Valid {
		t.Fatalf("expected valid response, got %+v", response)
	}
	if response.Compatibility.Compatible {
		t.Fatalf("expected incompatible compatibility status when subject actor mismatches principal filter")
	}
	if response.Compatibility.SubjectMatchesPrincipalRule {
		t.Fatalf("expected subject/principal compatibility mismatch")
	}
	if len(response.Warnings) == 0 {
		t.Fatalf("expected warning for subject/principal mismatch, got %+v", response.Warnings)
	}
	if len(response.NormalizedFilter.Channels) != 1 || response.NormalizedFilter.Channels[0] != "twilio_sms" {
		t.Fatalf("expected normalized channel list, got %+v", response.NormalizedFilter.Channels)
	}
	if len(response.NormalizedFilter.Keywords.ContainsAny) != 1 || response.NormalizedFilter.Keywords.ContainsAny[0] != "hello" {
		t.Fatalf("expected normalized keyword list, got %+v", response.NormalizedFilter.Keywords.ContainsAny)
	}
	if response.NormalizedFilterJSON == "" {
		t.Fatalf("expected normalized filter json")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(response.NormalizedFilterJSON), &decoded); err != nil {
		t.Fatalf("expected normalized filter json to parse: %v", err)
	}
}

func TestAutomationCommTriggerValidateBroadFilterWarningWhenFilterUnset(t *testing.T) {
	service, _ := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	response, err := service.AutomationCommTriggerValidate(ctx, transport.AutomationCommTriggerValidateRequest{
		WorkspaceID:    "ws1",
		SubjectActorID: "actor.requester",
	})
	if err != nil {
		t.Fatalf("automation comm-trigger validate: %v", err)
	}
	if !response.Valid {
		t.Fatalf("expected valid response for unset typed filter, got %+v", response)
	}
	if len(response.Warnings) == 0 || response.Warnings[0].Code != "broad_filter_match" {
		t.Fatalf("expected broad_filter_match warning, got %+v", response.Warnings)
	}
}

func TestAutomationUpdateAndIdempotentNoop(t *testing.T) {
	service, _ := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:     "ws1",
		SubjectActorID:  "actor.requester",
		TriggerType:     "SCHEDULE",
		Title:           "Original schedule",
		Instruction:     "original instruction",
		IntervalSeconds: 120,
		CooldownSeconds: 15,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	noopResponse, err := service.UpdateAutomation(ctx, transport.AutomationUpdateRequest{
		WorkspaceID: "ws1",
		TriggerID:   created.TriggerID,
	})
	if err != nil {
		t.Fatalf("noop update automation: %v", err)
	}
	if noopResponse.Updated || !noopResponse.Idempotent {
		t.Fatalf("expected idempotent noop update, got %+v", noopResponse)
	}

	enabled := false
	cooldown := 0
	interval := 600
	updatedResponse, err := service.UpdateAutomation(ctx, transport.AutomationUpdateRequest{
		WorkspaceID:     "ws1",
		TriggerID:       created.TriggerID,
		Title:           "Updated schedule",
		Instruction:     "updated instruction",
		Enabled:         &enabled,
		CooldownSeconds: &cooldown,
		IntervalSeconds: &interval,
	})
	if err != nil {
		t.Fatalf("update automation: %v", err)
	}
	if !updatedResponse.Updated || updatedResponse.Idempotent {
		t.Fatalf("expected updated non-idempotent response, got %+v", updatedResponse)
	}
	if updatedResponse.Trigger.DirectiveTitle != "Updated schedule" {
		t.Fatalf("expected updated title, got %s", updatedResponse.Trigger.DirectiveTitle)
	}
	if updatedResponse.Trigger.DirectiveInstruction != "updated instruction" {
		t.Fatalf("expected updated instruction, got %s", updatedResponse.Trigger.DirectiveInstruction)
	}
	if updatedResponse.Trigger.Enabled {
		t.Fatalf("expected trigger to be disabled after update")
	}
	if updatedResponse.Trigger.CooldownSeconds != 0 {
		t.Fatalf("expected cooldown reset to 0, got %d", updatedResponse.Trigger.CooldownSeconds)
	}
	if updatedResponse.Trigger.FilterJSON != `{"interval_seconds":600}` {
		t.Fatalf("expected interval filter update, got %s", updatedResponse.Trigger.FilterJSON)
	}

	replayUpdateResponse, err := service.UpdateAutomation(ctx, transport.AutomationUpdateRequest{
		WorkspaceID:     "ws1",
		TriggerID:       created.TriggerID,
		Title:           "Updated schedule",
		Instruction:     "updated instruction",
		Enabled:         &enabled,
		CooldownSeconds: &cooldown,
		IntervalSeconds: &interval,
	})
	if err != nil {
		t.Fatalf("repeat update automation: %v", err)
	}
	if replayUpdateResponse.Updated || !replayUpdateResponse.Idempotent {
		t.Fatalf("expected idempotent repeat update, got %+v", replayUpdateResponse)
	}
}

func TestAutomationUpdateValidationRejectsIntervalForCommTrigger(t *testing.T) {
	service, _ := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:    "ws1",
		SubjectActorID: "actor.requester",
		TriggerType:    "ON_COMM_EVENT",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	interval := 10
	_, err = service.UpdateAutomation(ctx, transport.AutomationUpdateRequest{
		WorkspaceID:     "ws1",
		TriggerID:       created.TriggerID,
		IntervalSeconds: &interval,
	})
	if err == nil {
		t.Fatalf("expected interval update for ON_COMM_EVENT to fail")
	}
}

func TestAutomationDeleteRemovesTriggerDirectiveAndIsIdempotent(t *testing.T) {
	service, container := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:     "ws1",
		SubjectActorID:  "actor.requester",
		TriggerType:     "SCHEDULE",
		Title:           "Delete me",
		Instruction:     "delete me",
		IntervalSeconds: 60,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	deleteResponse, err := service.DeleteAutomation(ctx, transport.AutomationDeleteRequest{
		WorkspaceID: "ws1",
		TriggerID:   created.TriggerID,
	})
	if err != nil {
		t.Fatalf("delete automation: %v", err)
	}
	if !deleteResponse.Deleted || deleteResponse.Idempotent {
		t.Fatalf("expected deleted non-idempotent response, got %+v", deleteResponse)
	}
	if deleteResponse.DirectiveID != created.DirectiveID {
		t.Fatalf("expected directive id %s, got %s", created.DirectiveID, deleteResponse.DirectiveID)
	}

	assertTableCount(t, container, "automation_triggers", "id = ?", created.TriggerID, 0)
	assertTableCount(t, container, "directives", "id = ?", created.DirectiveID, 0)

	replayDeleteResponse, err := service.DeleteAutomation(ctx, transport.AutomationDeleteRequest{
		WorkspaceID: "ws1",
		TriggerID:   created.TriggerID,
	})
	if err != nil {
		t.Fatalf("repeat delete automation: %v", err)
	}
	if replayDeleteResponse.Deleted || !replayDeleteResponse.Idempotent {
		t.Fatalf("expected idempotent repeat delete, got %+v", replayDeleteResponse)
	}
}

func TestAutomationDeletePreservesDirectiveWhenOtherTriggersReferenceIt(t *testing.T) {
	service, container := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:     "ws1",
		SubjectActorID:  "actor.requester",
		TriggerType:     "SCHEDULE",
		Title:           "Shared directive",
		Instruction:     "shared instruction",
		IntervalSeconds: 120,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := container.DB.Exec(`
		INSERT INTO automation_triggers(
			id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "trg-shared-secondary", created.WorkspaceID, created.DirectiveID, "SCHEDULE", 1, `{"interval_seconds":300}`, nil, nowText, nowText); err != nil {
		t.Fatalf("insert second trigger on same directive: %v", err)
	}

	deleteResponse, err := service.DeleteAutomation(ctx, transport.AutomationDeleteRequest{
		WorkspaceID: created.WorkspaceID,
		TriggerID:   created.TriggerID,
	})
	if err != nil {
		t.Fatalf("delete automation trigger with shared directive: %v", err)
	}
	if !deleteResponse.Deleted || deleteResponse.Idempotent {
		t.Fatalf("expected deleted non-idempotent response, got %+v", deleteResponse)
	}

	assertTableCount(t, container, "automation_triggers", "id = ?", created.TriggerID, 0)
	assertTableCount(t, container, "automation_triggers", "id = ?", "trg-shared-secondary", 1)
	assertTableCount(t, container, "directives", "id = ?", created.DirectiveID, 1)
}

func TestAutomationFireHistoryListsRecentRecordsWithTaskRunLinkage(t *testing.T) {
	service, container := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()
	configureOllamaProviderForRouteTests(t, container)

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:     "ws1",
		SubjectActorID:  "actor.requester",
		TriggerType:     "SCHEDULE",
		Title:           "History schedule",
		Instruction:     "history instruction",
		IntervalSeconds: 60,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	runAt := time.Now().UTC()
	if _, err := service.RunAutomationSchedule(ctx, transport.AutomationRunScheduleRequest{
		At: runAt.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("run schedule automation: %v", err)
	}

	pendingFireID, err := automationRandomID("fire")
	if err != nil {
		t.Fatalf("generate pending fire id: %v", err)
	}
	failedFireID, err := automationRandomID("fire")
	if err != nil {
		t.Fatalf("generate failed fire id: %v", err)
	}
	insertTriggerFireRow(t, container, pendingFireID, created.WorkspaceID, created.TriggerID, "manual:pending", runAt.Add(1*time.Second), "", "PENDING")
	insertTriggerFireRow(t, container, failedFireID, created.WorkspaceID, created.TriggerID, "manual:failed", runAt.Add(2*time.Second), "", "FAILED")

	response, err := service.ListAutomationFireHistory(ctx, transport.AutomationFireHistoryRequest{
		WorkspaceID: created.WorkspaceID,
		TriggerID:   created.TriggerID,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("list automation fire history: %v", err)
	}
	if response.WorkspaceID != created.WorkspaceID {
		t.Fatalf("expected workspace %s, got %s", created.WorkspaceID, response.WorkspaceID)
	}
	if len(response.Fires) < 3 {
		t.Fatalf("expected at least 3 fire records, got %d", len(response.Fires))
	}

	var (
		pendingFound     bool
		failedFound      bool
		createdTaskFound bool
	)
	for _, item := range response.Fires {
		if item.FireID == pendingFireID {
			pendingFound = true
			if item.Status != "pending" || item.IdempotencySignal != "manual:pending" {
				t.Fatalf("unexpected pending fire record: %+v", item)
			}
			if item.Route.TaskClassSource != "trigger_type" {
				t.Fatalf("expected trigger_type route source metadata for pending fire, got %+v", item.Route)
			}
		}
		if item.FireID == failedFireID {
			failedFound = true
			if item.Status != "failed" || item.IdempotencySignal != "manual:failed" {
				t.Fatalf("unexpected failed fire record: %+v", item)
			}
			if item.Route.TaskClassSource != "trigger_type" {
				t.Fatalf("expected trigger_type route source metadata for failed fire, got %+v", item.Route)
			}
		}
		if item.Status == "created_task" {
			if item.TaskID == "" || item.RunID == "" {
				t.Fatalf("expected task/run linkage for created_task fire, got %+v", item)
			}
			if !item.Route.Available {
				t.Fatalf("expected created_task fire to include available route metadata, got %+v", item.Route)
			}
			if item.Route.Provider == "" || item.Route.ModelKey == "" || item.Route.RouteSource == "" {
				t.Fatalf("expected created_task fire to include provider/model/source metadata, got %+v", item.Route)
			}
			createdTaskFound = true
		}
		if item.IdempotencyKey == "" {
			t.Fatalf("expected idempotency key on fire record: %+v", item)
		}
		if item.Route.TaskClass == "" || item.Route.TaskClassSource == "" || item.Route.RouteSource == "" {
			t.Fatalf("expected deterministic route metadata on fire record, got %+v", item.Route)
		}
	}
	if !pendingFound {
		t.Fatalf("expected pending fire record %s in response", pendingFireID)
	}
	if !failedFound {
		t.Fatalf("expected failed fire record %s in response", failedFireID)
	}
	if !createdTaskFound {
		t.Fatalf("expected created_task fire record in response")
	}
}

func TestAutomationFireHistoryFiltersStatusAndReturnsEmptyDeterministically(t *testing.T) {
	service, container := newAutomationInspectServiceForTriggerTests(t)
	ctx := context.Background()
	configureOllamaProviderForRouteTests(t, container)

	created, err := service.CreateAutomation(ctx, transport.AutomationCreateRequest{
		WorkspaceID:     "ws1",
		SubjectActorID:  "actor.requester",
		TriggerType:     "SCHEDULE",
		Title:           "History filter schedule",
		Instruction:     "history filter instruction",
		IntervalSeconds: 120,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create automation: %v", err)
	}

	failedFireID, err := automationRandomID("fire")
	if err != nil {
		t.Fatalf("generate failed fire id: %v", err)
	}
	insertTriggerFireRow(t, container, failedFireID, created.WorkspaceID, created.TriggerID, "manual:failed-only", time.Now().UTC(), "", "FAILED")

	failedOnly, err := service.ListAutomationFireHistory(ctx, transport.AutomationFireHistoryRequest{
		WorkspaceID: created.WorkspaceID,
		TriggerID:   created.TriggerID,
		Status:      "failed",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list failed-only fire history: %v", err)
	}
	if len(failedOnly.Fires) != 1 || failedOnly.Fires[0].FireID != failedFireID || failedOnly.Fires[0].Status != "failed" {
		t.Fatalf("unexpected failed-only fire history payload: %+v", failedOnly)
	}
	if failedOnly.Fires[0].Route.TaskClass == "" || failedOnly.Fires[0].Route.TaskClassSource == "" || failedOnly.Fires[0].Route.RouteSource == "" {
		t.Fatalf("expected deterministic route metadata on filtered fire history row, got %+v", failedOnly.Fires[0].Route)
	}

	emptyResult, err := service.ListAutomationFireHistory(ctx, transport.AutomationFireHistoryRequest{
		WorkspaceID: created.WorkspaceID,
		TriggerID:   "trigger-does-not-exist",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("list fire history empty result: %v", err)
	}
	if emptyResult.WorkspaceID != created.WorkspaceID {
		t.Fatalf("expected workspace %s for empty result, got %s", created.WorkspaceID, emptyResult.WorkspaceID)
	}
	if len(emptyResult.Fires) != 0 {
		t.Fatalf("expected empty fire history result, got %+v", emptyResult)
	}

	if _, err := service.ListAutomationFireHistory(ctx, transport.AutomationFireHistoryRequest{
		WorkspaceID: created.WorkspaceID,
		Status:      "not-a-real-status",
		Limit:       10,
	}); err == nil {
		t.Fatalf("expected invalid status filter to fail")
	}
}

func newAutomationInspectServiceForTriggerTests(t *testing.T) (*AutomationInspectRetentionContextService, *ServiceContainer) {
	t.Helper()
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}
	return service, container
}

func assertTableCount(t *testing.T, container *ServiceContainer, table string, where string, arg any, expected int) {
	t.Helper()
	query := "SELECT COUNT(1) FROM " + table + " WHERE " + where
	var count int
	if err := container.DB.QueryRow(query, arg).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d row(s) in %s, got %d", expected, table, count)
	}
}

func insertTriggerFireRow(
	t *testing.T,
	container *ServiceContainer,
	fireID string,
	workspaceID string,
	triggerID string,
	sourceEventID string,
	firedAt time.Time,
	taskID string,
	outcome string,
) {
	t.Helper()
	if _, err := container.DB.Exec(`
		INSERT INTO trigger_fires(id, workspace_id, trigger_id, source_event_id, fired_at, task_id, outcome)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fireID, workspaceID, triggerID, sourceEventID, firedAt.UTC().Format(time.RFC3339Nano), nullableTriggerTaskID(taskID), outcome); err != nil {
		t.Fatalf("insert trigger fire row: %v", err)
	}
}

func nullableTriggerTaskID(taskID string) any {
	if taskID == "" {
		return nil
	}
	return taskID
}
