package transport

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProviderOpenAPIAdaptersRoundTrip(t *testing.T) {
	request := ProviderSetRequest{
		WorkspaceID:      "ws1",
		Provider:         "openai",
		Endpoint:         "https://api.openai.com/v1",
		APIKeySecretName: "openai_api_key",
		ClearAPIKey:      true,
	}
	requestRoundTrip := fromOpenAPIProviderSetRequest(toOpenAPIProviderSetRequest(request))
	if requestRoundTrip != request {
		t.Fatalf("expected provider set request round-trip parity, got %+v", requestRoundTrip)
	}

	record := ProviderConfigRecord{
		WorkspaceID:      "ws1",
		Provider:         "openai",
		Endpoint:         "https://api.openai.com/v1",
		APIKeySecretName: "openai_api_key",
		APIKeyConfigured: true,
		UpdatedAt:        time.Date(2026, 3, 3, 10, 5, 0, 0, time.UTC),
	}
	recordRoundTrip := fromOpenAPIProviderConfigRecord(toOpenAPIProviderConfigRecord(record))
	if !recordRoundTrip.UpdatedAt.Equal(record.UpdatedAt) {
		t.Fatalf("expected provider config updated_at to round-trip, got %s", recordRoundTrip.UpdatedAt)
	}
	recordRoundTrip.UpdatedAt = time.Time{}
	record.UpdatedAt = time.Time{}
	if recordRoundTrip != record {
		t.Fatalf("expected provider config round-trip parity, got %+v", recordRoundTrip)
	}
}

func TestTaskRunListOpenAPIAdaptersRoundTrip(t *testing.T) {
	payload := TaskRunListResponse{
		WorkspaceID: "ws1",
		Items: []TaskRunListItem{
			{
				TaskID:                  "task-1",
				RunID:                   "run-1",
				WorkspaceID:             "ws1",
				Title:                   "Task One",
				TaskState:               "running",
				RunState:                "running",
				Priority:                1,
				RequestedByActorID:      "actor.requester",
				SubjectPrincipalActorID: "actor.subject",
				ActingAsActorID:         "actor.subject",
				LastError:               "",
				TaskCreatedAt:           "2026-03-01T00:00:00Z",
				TaskUpdatedAt:           "2026-03-01T00:00:05Z",
				RunCreatedAt:            "2026-03-01T00:00:00Z",
				RunUpdatedAt:            "2026-03-01T00:00:05Z",
				StartedAt:               "2026-03-01T00:00:01Z",
				FinishedAt:              "",
				Actions: TaskRunActionAvailability{
					CanCancel:  true,
					CanRetry:   false,
					CanRequeue: true,
				},
				Route: WorkflowRouteMetadata{
					Available:       true,
					TaskClass:       "chat",
					TaskClassSource: "task_channel",
					Provider:        "openai",
					ModelKey:        "gpt-4.1-mini",
					RouteSource:     "policy",
					Notes:           "selected by workspace policy",
				},
			},
		},
	}
	roundTrip := fromOpenAPITaskRunListResponse(toOpenAPITaskRunListResponse(payload))
	if !reflect.DeepEqual(roundTrip, payload) {
		t.Fatalf("expected task run list round-trip parity, got %+v", roundTrip)
	}
}

func TestChatTurnOpenAPIAdaptersRoundTrip(t *testing.T) {
	request := ChatTurnRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.requester",
		SubjectActorID:     "actor.subject",
		ActingAsActorID:    "actor.subject",
		ProviderOverride:   "openai",
		ModelOverride:      "gpt-4.1-mini",
		SystemPrompt:       "Be concise.",
		Channel: ChatTurnChannelContext{
			ChannelID:   "app",
			ConnectorID: "builtin.app",
			ThreadID:    "thread-1",
		},
		ToolCatalog: []ChatTurnToolCatalogEntry{
			{
				Name:           "mail_send",
				Description:    "Send an email",
				CapabilityKeys: []string{"mail.send"},
				InputSchema: map[string]any{
					"type": "object",
				},
			},
		},
		Items: []ChatTurnItem{
			{
				ItemID:   "item-1",
				Type:     "user_message",
				Role:     "user",
				Status:   "completed",
				Content:  "send an update",
				Metadata: ChatTurnItemMetadataFromMap(map[string]any{"source": "test"}),
			},
		},
	}
	requestRoundTrip := fromOpenAPIChatTurnRequest(toOpenAPIChatTurnRequest(request))
	if !reflect.DeepEqual(requestRoundTrip, request) {
		t.Fatalf("expected chat turn request round-trip parity, got %+v", requestRoundTrip)
	}

	response := ChatTurnResponse{
		WorkspaceID:                  "ws1",
		TaskClass:                    "chat",
		Provider:                     "openai",
		ModelKey:                     "gpt-4.1-mini",
		CorrelationID:                "corr-chat",
		ContractVersion:              ChatTurnContractVersionV2,
		TurnItemSchemaVersion:        ChatTurnItemSchemaVersionV1,
		RealtimeEventContractVersion: ChatRealtimeLifecycleContractVersionV2,
		Channel:                      request.Channel,
		Items:                        request.Items,
		TaskRunCorrelation: ChatTurnTaskRunCorrelation{
			Available: true,
			Source:    "turn_ledger",
			TaskID:    "task-1",
			RunID:     "run-1",
			TaskState: "running",
			RunState:  "running",
		},
	}
	responseRoundTrip := fromOpenAPIChatTurnResponse(toOpenAPIChatTurnResponse(response))
	if !reflect.DeepEqual(responseRoundTrip, response) {
		t.Fatalf("expected chat turn response round-trip parity, got %+v", responseRoundTrip)
	}
}

func TestChatTurnOpenAPIAdapterOmitsOptionalToolCatalogWhenEmpty(t *testing.T) {
	request := ChatTurnRequest{
		WorkspaceID: "ws1",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Content: "hello"},
		},
	}

	openapiRequest := toOpenAPIChatTurnRequest(request)
	if openapiRequest.ToolCatalog != nil {
		t.Fatalf("expected nil tool_catalog pointer when optional catalog is empty")
	}

	payload, err := json.Marshal(openapiRequest)
	if err != nil {
		t.Fatalf("marshal chat turn openapi request: %v", err)
	}
	if strings.Contains(string(payload), "tool_catalog") {
		t.Fatalf("expected tool_catalog to be omitted from JSON payload, got %s", string(payload))
	}
}
