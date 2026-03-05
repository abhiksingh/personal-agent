package contracts

import (
	"encoding/json"
	"testing"
)

func TestTaskValidateRequiresIdentityFields(t *testing.T) {
	task := Task{}
	if err := task.Validate(); err == nil {
		t.Fatalf("expected validation error for empty task")
	}

	task = Task{
		ID:                 "task_1",
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_req",
		SubjectPrincipalID: "actor_subj",
		State:              TaskStateQueued,
	}
	if err := task.Validate(); err != nil {
		t.Fatalf("expected valid task, got error: %v", err)
	}
}

func TestTaskStateTerminalStates(t *testing.T) {
	if !TaskStateCompleted.IsTerminal() {
		t.Fatalf("expected completed to be terminal")
	}
	if TaskStateRunning.IsTerminal() {
		t.Fatalf("expected running to be non-terminal")
	}
}

func TestTaskJSONUsesCanonicalFieldNames(t *testing.T) {
	task := Task{
		ID:                 "task_1",
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_req",
		SubjectPrincipalID: "actor_subj",
		State:              TaskStateQueued,
	}
	bytes, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("unmarshal task json: %v", err)
	}

	if _, ok := payload["requested_by_actor_id"]; !ok {
		t.Fatalf("expected requested_by_actor_id json field")
	}
	if _, ok := payload["subject_principal_actor_id"]; !ok {
		t.Fatalf("expected subject_principal_actor_id json field")
	}
}
