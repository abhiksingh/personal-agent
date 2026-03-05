package transport

import (
	"encoding/json"
	"strings"
	"testing"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func TestTaskSubmitOpenAPIRequestAdapterRoundTrip(t *testing.T) {
	request := SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.subject",
		Title:                   "task title",
		Description:             "task description",
		TaskClass:               "chat",
	}

	openapiRequest := toOpenAPITaskSubmitRequest(request)
	roundTrip := fromOpenAPITaskSubmitRequest(openapiRequest)

	if roundTrip != request {
		t.Fatalf("expected round-trip request parity, got %+v", roundTrip)
	}
}

func TestTaskSubmitOpenAPIRequestAdapterOmitsEmptyOptionalFields(t *testing.T) {
	request := SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.subject",
		Title:                   "task title",
	}

	openapiRequest := toOpenAPITaskSubmitRequest(request)
	if openapiRequest.Description != nil {
		t.Fatalf("expected nil description pointer for empty optional field")
	}
	if openapiRequest.TaskClass != nil {
		t.Fatalf("expected nil task_class pointer for empty optional field")
	}

	body, err := json.Marshal(openapiRequest)
	if err != nil {
		t.Fatalf("marshal openapi request: %v", err)
	}
	if strings.Contains(string(body), "description") {
		t.Fatalf("expected description to be omitted from JSON payload, got %s", string(body))
	}
	if strings.Contains(string(body), "task_class") {
		t.Fatalf("expected task_class to be omitted from JSON payload, got %s", string(body))
	}
}

func TestTaskSubmitOpenAPIResponseAdapterRoundTrip(t *testing.T) {
	openapiResponse := openapitypes.TaskSubmitResponse{
		TaskId:        "task_123",
		RunId:         "run_456",
		State:         "queued",
		CorrelationId: "corr_789",
	}

	legacy := fromOpenAPITaskSubmitResponse(openapiResponse)
	roundTrip := toOpenAPITaskSubmitResponse(legacy)

	if roundTrip != openapiResponse {
		t.Fatalf("expected round-trip response parity, got %+v", roundTrip)
	}
}
