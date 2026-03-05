package transport

import (
	"encoding/json"
	"testing"
)

func TestAutomationTriggerRecordMarshalIncludesCooldownSecondsWhenZero(t *testing.T) {
	record := AutomationTriggerRecord{
		TriggerID:             "trigger-1",
		WorkspaceID:           "ws-1",
		DirectiveID:           "directive-1",
		TriggerType:           "SCHEDULE",
		Enabled:               true,
		FilterJSON:            "{\"interval_seconds\":60}",
		CooldownSeconds:       0,
		SubjectPrincipalActor: "actor.default",
		DirectiveTitle:        "Check Inbox",
		DirectiveInstruction:  "Look for priority messages",
		DirectiveStatus:       "ACTIVE",
		CreatedAt:             "2026-02-25T00:00:00Z",
		UpdatedAt:             "2026-02-25T00:00:00Z",
	}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal automation trigger record: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal automation trigger json: %v", err)
	}

	value, exists := decoded["cooldown_seconds"]
	if !exists {
		t.Fatalf("expected cooldown_seconds key in payload, got %s", string(payload))
	}
	cooldown, ok := value.(float64)
	if !ok || cooldown != 0 {
		t.Fatalf("expected cooldown_seconds=0, got %#v", value)
	}
}
