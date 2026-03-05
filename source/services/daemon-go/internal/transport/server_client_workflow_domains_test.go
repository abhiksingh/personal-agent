package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestTransportChatTurnUsesCanonicalTurnItemsContract(t *testing.T) {
	chat := &chatServiceStub{
		response: ChatTurnResponse{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []ChatTurnItem{
				{
					ItemID:  "assistant-2",
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: "I can help with that.",
				},
				{
					ItemID:       "tool-call-1",
					Type:         "tool_call",
					Status:       "completed",
					ToolName:     "mail_send",
					ToolCallID:   "tc-1",
					Arguments:    map[string]any{"recipient": "sam@example.com"},
					ErrorCode:    "",
					ErrorMessage: "",
				},
				{
					ItemID:            "tool-result-1",
					Type:              "tool_result",
					Status:            "awaiting_approval",
					ToolName:          "mail_send",
					ToolCallID:        "tc-1",
					Output:            map[string]any{"approval_required": true, "approval_request_id": "approval-1"},
					ApprovalRequestID: "approval-1",
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	response, err := client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "system_message", Role: "system", Status: "completed", Content: "you are helpful"},
			{Type: "user_message", Role: "user", Status: "completed", Content: "email sam that we are on track"},
		},
	}, "corr-chat-act")
	if err != nil {
		t.Fatalf("chat act turn: %v", err)
	}

	if len(response.Items) != 3 {
		t.Fatalf("expected 3 turn items, got %+v", response.Items)
	}
	if response.Items[1].Type != "tool_call" || response.Items[2].Type != "tool_result" {
		t.Fatalf("expected tool call/result turn items, got %+v", response.Items)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", response.Items[2].Output["approval_required"])) != "true" {
		t.Fatalf("expected tool_result output approval_required=true, got %+v", response.Items[2].Output)
	}
}

func TestTransportChatTurnPublishesRealtimeLifecycleEvents(t *testing.T) {
	chat := &chatServiceStub{
		streamDeltas: []string{"hello", " ", "world"},
		response: ChatTurnResponse{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []ChatTurnItem{
				{
					ItemID:  "assistant-1",
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: "hello world",
				},
				{
					ItemID:     "tool-call-evt",
					Type:       "tool_call",
					Status:     "completed",
					ToolName:   "mail_send",
					ToolCallID: "tc-evt",
					Arguments:  map[string]any{"recipient": "sam@example.com"},
					Metadata: ChatTurnItemMetadataFromMap(map[string]any{
						"policy_decision":    "ALLOW",
						"policy_reason_code": "allowed",
					}),
				},
				{
					ItemID:            "tool-result-evt",
					Type:              "tool_result",
					Status:            "completed",
					ToolName:          "mail_send",
					ToolCallID:        "tc-evt",
					Output:            map[string]any{"delivered": true},
					ApprovalRequestID: "",
					Metadata: ChatTurnItemMetadataFromMap(map[string]any{
						"policy_decision":    "ALLOW",
						"policy_reason_code": "allowed",
					}),
				},
			},
			TaskRunCorrelation: ChatTurnTaskRunCorrelation{
				Available: true,
				Source:    "audit",
				TaskID:    "task-chat-1",
				RunID:     "run-chat-1",
				TaskState: "running",
				RunState:  "running",
			},
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-chat-stream")
	if err != nil {
		t.Fatalf("connect realtime stream: %v", err)
	}
	defer stream.Close()

	_, err = client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Status: "completed", Content: "hello"},
		},
	}, "corr-chat-stream")
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}

	deltaSeen := false
	deltaWhitespaceSeen := false
	streamedDeltas := make([]string, 0, 4)
	toolStartedSeen := false
	toolOutputSeen := false
	toolCompletedSeen := false
	completedSeen := false
	for attempt := 0; attempt < 24 && (!deltaSeen || !toolStartedSeen || !toolOutputSeen || !toolCompletedSeen || !completedSeen); attempt++ {
		event, recvErr := stream.Receive()
		if recvErr != nil {
			t.Fatalf("receive realtime event: %v", recvErr)
		}
		switch event.EventType {
		case "turn_item_delta", "turn_item_started", "turn_item_completed", "tool_call_started", "tool_call_output", "tool_call_completed", "chat_completed", "chat_error":
			if strings.TrimSpace(event.ContractVersion) != ChatRealtimeLifecycleContractVersionV2 {
				t.Fatalf("expected realtime contract version %q for %s, got %q", ChatRealtimeLifecycleContractVersionV2, event.EventType, event.ContractVersion)
			}
			if strings.TrimSpace(event.LifecycleSchemaVersion) != ChatTurnItemSchemaVersionV1 {
				t.Fatalf("expected lifecycle schema version %q for %s, got %q", ChatTurnItemSchemaVersionV1, event.EventType, event.LifecycleSchemaVersion)
			}
		}
		switch event.EventType {
		case "turn_item_delta":
			deltaValue, ok := event.Payload.AsMap()["delta"]
			if !ok {
				t.Fatalf("expected turn_item_delta payload to include delta")
			}
			delta, ok := deltaValue.(string)
			if !ok {
				delta = fmt.Sprintf("%v", deltaValue)
			}
			if delta == "" {
				t.Fatalf("expected non-empty turn_item_delta payload")
			}
			if strings.TrimSpace(delta) == "" {
				deltaWhitespaceSeen = true
			} else {
				deltaSeen = true
			}
			streamedDeltas = append(streamedDeltas, delta)
		case "tool_call_started":
			if gotTool := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["tool_name"])); gotTool != "mail_send" {
				t.Fatalf("expected tool_call_started tool_name mail_send, got %q", gotTool)
			}
			metadata, ok := event.Payload.AsMap()["metadata"].(map[string]any)
			if !ok || strings.TrimSpace(fmt.Sprintf("%v", metadata["policy_decision"])) != "ALLOW" {
				t.Fatalf("expected tool_call_started metadata with policy_decision=ALLOW, got %+v", event.Payload.AsMap()["metadata"])
			}
			toolStartedSeen = true
		case "tool_call_output":
			if gotStatus := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["status"])); gotStatus != "completed" {
				t.Fatalf("expected tool_call_output status completed, got %q", gotStatus)
			}
			metadata, ok := event.Payload.AsMap()["metadata"].(map[string]any)
			if !ok || strings.TrimSpace(fmt.Sprintf("%v", metadata["policy_reason_code"])) != "allowed" {
				t.Fatalf("expected tool_call_output metadata with policy_reason_code=allowed, got %+v", event.Payload.AsMap()["metadata"])
			}
			toolOutputSeen = true
		case "tool_call_completed":
			if gotStatus := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["status"])); gotStatus != "completed" {
				t.Fatalf("expected tool_call_completed status completed, got %q", gotStatus)
			}
			metadata, ok := event.Payload.AsMap()["metadata"].(map[string]any)
			if !ok || strings.TrimSpace(fmt.Sprintf("%v", metadata["policy_reason_code"])) != "allowed" {
				t.Fatalf("expected tool_call_completed metadata with policy_reason_code=allowed, got %+v", event.Payload.AsMap()["metadata"])
			}
			toolCompletedSeen = true
		case "chat_completed":
			if gotItemCount := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["item_count"])); gotItemCount != "3" {
				t.Fatalf("expected chat_completed item_count=3, got %q", gotItemCount)
			}
			if gotTaskID := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["task_id"])); gotTaskID != "task-chat-1" {
				t.Fatalf("expected chat_completed task_id task-chat-1, got %q", gotTaskID)
			}
			completedSeen = true
		}
	}

	if !deltaSeen {
		t.Fatalf("expected realtime turn_item_delta event")
	}
	if !deltaWhitespaceSeen {
		t.Fatalf("expected realtime turn_item_delta stream to preserve whitespace-only deltas, got %+v", streamedDeltas)
	}
	if strings.Join(streamedDeltas, "") != "hello world" {
		t.Fatalf("expected streamed deltas to preserve spacing, got %+v", streamedDeltas)
	}
	if !toolStartedSeen {
		t.Fatalf("expected realtime tool_call_started event")
	}
	if !toolOutputSeen {
		t.Fatalf("expected realtime tool_call_output event")
	}
	if !toolCompletedSeen {
		t.Fatalf("expected realtime tool_call_completed event")
	}
	if !completedSeen {
		t.Fatalf("expected realtime chat_completed event")
	}
}

func TestTransportChatTurnPublishesRealtimeLifecycleEventsForMultipleToolCalls(t *testing.T) {
	chat := &chatServiceStub{
		response: ChatTurnResponse{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []ChatTurnItem{
				{
					ItemID:     "tool-call-1",
					Type:       "tool_call",
					Status:     "completed",
					ToolName:   "mail_send",
					ToolCallID: "tc-1",
					Arguments:  map[string]any{"recipient": "sam@example.com"},
				},
				{
					ItemID:     "tool-result-1",
					Type:       "tool_result",
					Status:     "completed",
					ToolName:   "mail_send",
					ToolCallID: "tc-1",
					Output:     map[string]any{"delivered": true},
				},
				{
					ItemID:     "tool-call-2",
					Type:       "tool_call",
					Status:     "completed",
					ToolName:   "calendar_create",
					ToolCallID: "tc-2",
					Arguments:  map[string]any{"title": "Ship review"},
				},
				{
					ItemID:     "tool-result-2",
					Type:       "tool_result",
					Status:     "completed",
					ToolName:   "calendar_create",
					ToolCallID: "tc-2",
					Output:     map[string]any{"event_id": "evt-1"},
				},
				{
					ItemID:  "assistant-final",
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: "Done.",
				},
			},
			TaskRunCorrelation: ChatTurnTaskRunCorrelation{
				Available: true,
				Source:    "audit",
				TaskID:    "task-chat-2",
				RunID:     "run-chat-2",
				TaskState: "completed",
				RunState:  "completed",
			},
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-chat-stream-multi")
	if err != nil {
		t.Fatalf("connect realtime stream: %v", err)
	}
	defer stream.Close()

	_, err = client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Status: "completed", Content: "run two tools"},
		},
	}, "corr-chat-stream-multi")
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}

	toolStartedCount := 0
	toolOutputCount := 0
	toolCompletedCount := 0
	completedSeen := false
	for attempt := 0; attempt < 48 && !completedSeen; attempt++ {
		event, recvErr := stream.Receive()
		if recvErr != nil {
			t.Fatalf("receive realtime event: %v", recvErr)
		}
		switch event.EventType {
		case "turn_item_delta", "turn_item_started", "turn_item_completed", "tool_call_started", "tool_call_output", "tool_call_completed", "chat_completed", "chat_error":
			if strings.TrimSpace(event.ContractVersion) != ChatRealtimeLifecycleContractVersionV2 {
				t.Fatalf("expected realtime contract version %q for %s, got %q", ChatRealtimeLifecycleContractVersionV2, event.EventType, event.ContractVersion)
			}
			if strings.TrimSpace(event.LifecycleSchemaVersion) != ChatTurnItemSchemaVersionV1 {
				t.Fatalf("expected lifecycle schema version %q for %s, got %q", ChatTurnItemSchemaVersionV1, event.EventType, event.LifecycleSchemaVersion)
			}
		}
		switch event.EventType {
		case "tool_call_started":
			toolStartedCount++
		case "tool_call_output":
			toolOutputCount++
		case "tool_call_completed":
			toolCompletedCount++
		case "chat_completed":
			if gotItemCount := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["item_count"])); gotItemCount != "5" {
				t.Fatalf("expected chat_completed item_count=5, got %q", gotItemCount)
			}
			completedSeen = true
		}
	}
	if !completedSeen {
		t.Fatalf("expected realtime chat_completed event for multi-tool response")
	}
	if toolStartedCount != 2 {
		t.Fatalf("expected 2 tool_call_started events, got %d", toolStartedCount)
	}
	if toolOutputCount != 2 {
		t.Fatalf("expected 2 tool_call_output events, got %d", toolOutputCount)
	}
	if toolCompletedCount != 2 {
		t.Fatalf("expected 2 tool_call_completed events, got %d", toolCompletedCount)
	}
}

func TestTransportChatTurnFailurePublishesRealtimeChatError(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat: &chatServiceStub{
			turnErr: errors.New("chat backend unavailable"),
		},
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-chat-error")
	if err != nil {
		t.Fatalf("connect realtime stream: %v", err)
	}
	defer stream.Close()

	_, err = client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Status: "completed", Content: "hello"},
		},
	}, "corr-chat-error")
	if err == nil {
		t.Fatalf("expected chat turn failure")
	}

	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", httpErr.StatusCode)
	}

	event, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive realtime error event: %v", err)
	}
	for event.EventType != "chat_error" {
		event, err = stream.Receive()
		if err != nil {
			t.Fatalf("receive chat_error follow-up event: %v", err)
		}
	}
	if strings.TrimSpace(event.ContractVersion) != ChatRealtimeLifecycleContractVersionV2 {
		t.Fatalf("expected realtime contract version %q for chat_error, got %q", ChatRealtimeLifecycleContractVersionV2, event.ContractVersion)
	}
	if strings.TrimSpace(event.LifecycleSchemaVersion) != ChatTurnItemSchemaVersionV1 {
		t.Fatalf("expected lifecycle schema version %q for chat_error, got %q", ChatTurnItemSchemaVersionV1, event.LifecycleSchemaVersion)
	}
	if gotMessage := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["message"])); !strings.Contains(gotMessage, "unavailable") {
		t.Fatalf("expected chat_error message to mention unavailable, got %q", gotMessage)
	}
}

func TestTransportChannelAndConnectorDiagnosticsRoutes(t *testing.T) {
	uiStatus := &uiStatusServiceStub{
		channelDiagnosticsResponse: ChannelDiagnosticsResponse{
			WorkspaceID: "ws1",
			Diagnostics: []ChannelDiagnosticsSummary{
				{
					ChannelID:   "app_chat",
					DisplayName: "App Chat",
					Category:    "local",
					Configured:  true,
					Status:      "ready",
					WorkerHealth: WorkerHealthSnapshot{
						Registered: true,
						Worker: &PluginWorkerStatusCard{
							PluginID: "app_chat.daemon",
							State:    "running",
						},
					},
					RemediationActions: []DiagnosticsRemediationAction{
						{
							Identifier:  "refresh_channel_status",
							Label:       "Refresh Channel Status",
							Intent:      "refresh_status",
							Destination: "/v1/channels/status",
							Enabled:     true,
							Recommended: false,
						},
					},
				},
			},
		},
		connectorDiagnosticsResponse: ConnectorDiagnosticsResponse{
			WorkspaceID: "ws1",
			Diagnostics: []ConnectorDiagnosticsSummary{
				{
					ConnectorID: "calendar",
					PluginID:    "calendar.daemon",
					DisplayName: "Calendar Connector",
					Configured:  true,
					Status:      "failed",
					WorkerHealth: WorkerHealthSnapshot{
						Registered: true,
						Worker: &PluginWorkerStatusCard{
							PluginID: "calendar.daemon",
							State:    "failed",
						},
					},
					RemediationActions: []DiagnosticsRemediationAction{
						{
							Identifier:  "repair_daemon_runtime",
							Label:       "Run Daemon Repair",
							Intent:      "daemon_lifecycle_control",
							Destination: "/v1/daemon/lifecycle/control",
							Enabled:     true,
							Recommended: true,
						},
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "ui-status-token",
		UIStatus:     uiStatus,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "ui-status-token",
	})
	if err != nil {
		t.Fatalf("create ui diagnostics client: %v", err)
	}

	channelDiagnostics, err := client.ChannelDiagnostics(context.Background(), ChannelDiagnosticsRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app_chat",
	}, "corr-channel-diagnostics")
	if err != nil {
		t.Fatalf("channel diagnostics: %v", err)
	}
	if len(channelDiagnostics.Diagnostics) != 1 || channelDiagnostics.Diagnostics[0].ChannelID != "app_chat" {
		t.Fatalf("unexpected channel diagnostics payload: %+v", channelDiagnostics)
	}
	if len(channelDiagnostics.Diagnostics[0].RemediationActions) != 1 ||
		channelDiagnostics.Diagnostics[0].RemediationActions[0].Identifier != "refresh_channel_status" ||
		channelDiagnostics.Diagnostics[0].RemediationActions[0].Intent != "refresh_status" {
		t.Fatalf("expected channel diagnostics remediation actions to round-trip typed metadata, got %+v", channelDiagnostics.Diagnostics[0].RemediationActions)
	}
	if uiStatus.lastChannelDiagReq.WorkspaceID != "ws1" || uiStatus.lastChannelDiagReq.ChannelID != "app_chat" {
		t.Fatalf("unexpected channel diagnostics request payload: %+v", uiStatus.lastChannelDiagReq)
	}

	connectorDiagnostics, err := client.ConnectorDiagnostics(context.Background(), ConnectorDiagnosticsRequest{
		WorkspaceID: "ws1",
		ConnectorID: "calendar",
	}, "corr-connector-diagnostics")
	if err != nil {
		t.Fatalf("connector diagnostics: %v", err)
	}
	if len(connectorDiagnostics.Diagnostics) != 1 || connectorDiagnostics.Diagnostics[0].ConnectorID != "calendar" {
		t.Fatalf("unexpected connector diagnostics payload: %+v", connectorDiagnostics)
	}
	if len(connectorDiagnostics.Diagnostics[0].RemediationActions) != 1 ||
		connectorDiagnostics.Diagnostics[0].RemediationActions[0].Identifier != "repair_daemon_runtime" ||
		connectorDiagnostics.Diagnostics[0].RemediationActions[0].Intent != "daemon_lifecycle_control" {
		t.Fatalf("expected connector diagnostics remediation actions to round-trip typed metadata, got %+v", connectorDiagnostics.Diagnostics[0].RemediationActions)
	}
	if uiStatus.lastConnectorDiagReq.WorkspaceID != "ws1" || uiStatus.lastConnectorDiagReq.ConnectorID != "calendar" {
		t.Fatalf("unexpected connector diagnostics request payload: %+v", uiStatus.lastConnectorDiagReq)
	}
}

func TestTransportChannelAndConnectorStatusRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "ui-status-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "ui-status-token",
	})
	if err != nil {
		t.Fatalf("create ui status client: %v", err)
	}

	_, err = client.ChannelConnectorMappingsList(context.Background(), ChannelConnectorMappingListRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
	}, "corr-channel-mapping-list")
	if err == nil {
		t.Fatalf("expected channel connector mapping list error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ChannelConnectorMappingUpsert(context.Background(), ChannelConnectorMappingUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
		ConnectorID: "twilio",
		Enabled:     true,
		Priority:    1,
	}, "corr-channel-mapping-upsert")
	if err == nil {
		t.Fatalf("expected channel connector mapping upsert error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ChannelStatus(context.Background(), ChannelStatusRequest{WorkspaceID: "ws1"}, "corr-channel-status")
	if err == nil {
		t.Fatalf("expected channel status error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ConnectorStatus(context.Background(), ConnectorStatusRequest{WorkspaceID: "ws1"}, "corr-connector-status")
	if err == nil {
		t.Fatalf("expected connector status error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ConnectorPermissionRequest(context.Background(), ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	}, "corr-connector-permission")
	if err == nil {
		t.Fatalf("expected connector permission request error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ChannelConfigUpsert(context.Background(), ChannelConfigUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app_chat",
	}, "corr-channel-config-upsert")
	if err == nil {
		t.Fatalf("expected channel config upsert error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ConnectorConfigUpsert(context.Background(), ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	}, "corr-connector-config-upsert")
	if err == nil {
		t.Fatalf("expected connector config upsert error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ChannelTestOperation(context.Background(), ChannelTestOperationRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app_chat",
	}, "corr-channel-test")
	if err == nil {
		t.Fatalf("expected channel test error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ConnectorTestOperation(context.Background(), ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	}, "corr-connector-test")
	if err == nil {
		t.Fatalf("expected connector test error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportChannelAndConnectorDiagnosticsRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "ui-status-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "ui-status-token",
	})
	if err != nil {
		t.Fatalf("create ui diagnostics client: %v", err)
	}

	_, err = client.ChannelDiagnostics(context.Background(), ChannelDiagnosticsRequest{
		WorkspaceID: "ws1",
	}, "corr-channel-diagnostics")
	if err == nil {
		t.Fatalf("expected channel diagnostics error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ConnectorDiagnostics(context.Background(), ConnectorDiagnosticsRequest{
		WorkspaceID: "ws1",
	}, "corr-connector-diagnostics")
	if err == nil {
		t.Fatalf("expected connector diagnostics error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportCloudflaredConnectorRoutes(t *testing.T) {
	cloudflared := &cloudflaredConnectorServiceStub{
		versionResponse: CloudflaredVersionResponse{
			WorkspaceID: "ws1",
			Available:   true,
			BinaryPath:  "/usr/local/bin/cloudflared",
			Version:     "2026.2.0",
			ExitCode:    0,
			DryRun:      false,
		},
		execResponse: CloudflaredExecResponse{
			WorkspaceID: "ws1",
			Success:     true,
			BinaryPath:  "/usr/local/bin/cloudflared",
			Args:        []string{"version"},
			ExitCode:    0,
			Stdout:      "cloudflared version 2026.2.0",
			DurationMS:  5,
			DryRun:      false,
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "cloudflared-token",
		Cloudflared:  cloudflared,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "cloudflared-token",
	})
	if err != nil {
		t.Fatalf("create cloudflared client: %v", err)
	}

	version, err := client.CloudflaredVersion(context.Background(), CloudflaredVersionRequest{
		WorkspaceID: "ws1",
	}, "corr-cloudflared-version")
	if err != nil {
		t.Fatalf("cloudflared version: %v", err)
	}
	if !version.Available || version.Version != "2026.2.0" {
		t.Fatalf("unexpected cloudflared version payload: %+v", version)
	}
	if cloudflared.lastVersionReq.WorkspaceID != "ws1" {
		t.Fatalf("expected cloudflared version workspace ws1, got %+v", cloudflared.lastVersionReq)
	}

	execResponse, err := client.CloudflaredExec(context.Background(), CloudflaredExecRequest{
		WorkspaceID: "ws1",
		Args:        []string{"version"},
		TimeoutMS:   3000,
	}, "corr-cloudflared-exec")
	if err != nil {
		t.Fatalf("cloudflared exec: %v", err)
	}
	if !execResponse.Success || execResponse.ExitCode != 0 {
		t.Fatalf("unexpected cloudflared exec payload: %+v", execResponse)
	}
	if cloudflared.lastExecReq.WorkspaceID != "ws1" || len(cloudflared.lastExecReq.Args) != 1 || cloudflared.lastExecReq.Args[0] != "version" {
		t.Fatalf("unexpected cloudflared exec request payload: %+v", cloudflared.lastExecReq)
	}
}

func TestTransportCloudflaredConnectorRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "cloudflared-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "cloudflared-token",
	})
	if err != nil {
		t.Fatalf("create cloudflared client: %v", err)
	}

	_, err = client.CloudflaredVersion(context.Background(), CloudflaredVersionRequest{
		WorkspaceID: "ws1",
	}, "corr-cloudflared-version")
	if err == nil {
		t.Fatalf("expected cloudflared version error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportApprovalInboxRoute(t *testing.T) {
	workflowQueries := &workflowQueryServiceStub{
		approvalInboxResponse: ApprovalInboxResponse{
			WorkspaceID: "ws1",
			Approvals: []ApprovalInboxItem{
				{
					ApprovalRequestID:       "apr-1",
					WorkspaceID:             "ws1",
					State:                   "pending",
					RiskLevel:               "destructive",
					RiskRationale:           "Destructive action requires explicit GO AHEAD approval before execution.",
					RequestedAt:             "2026-02-24T00:00:00Z",
					RequestedByActorID:      "actor.requester",
					SubjectPrincipalActorID: "actor.subject",
					ActingAsActorID:         "actor.subject",
					Route: WorkflowRouteMetadata{
						Available:       true,
						TaskClass:       "finder",
						Provider:        "ollama",
						ModelKey:        "llama3.2",
						TaskClassSource: "step_capability",
						RouteSource:     "fallback_enabled",
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "workflow-query-token",
		WorkflowQueries: workflowQueries,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	response, err := client.ApprovalInbox(context.Background(), ApprovalInboxRequest{
		WorkspaceID:  "ws1",
		IncludeFinal: true,
		Limit:        20,
		State:        "pending",
	}, "corr-approval-inbox")
	if err != nil {
		t.Fatalf("approval inbox: %v", err)
	}
	if len(response.Approvals) != 1 || response.Approvals[0].ApprovalRequestID != "apr-1" {
		t.Fatalf("unexpected approval inbox payload: %+v", response)
	}
	if !response.Approvals[0].Route.Available ||
		response.Approvals[0].Route.Provider != "ollama" ||
		response.Approvals[0].Route.ModelKey != "llama3.2" ||
		response.Approvals[0].Route.TaskClass != "finder" {
		t.Fatalf("expected approval inbox route metadata to round-trip, got %+v", response.Approvals[0].Route)
	}
	if workflowQueries.lastApprovalInboxReq.WorkspaceID != "ws1" ||
		!workflowQueries.lastApprovalInboxReq.IncludeFinal ||
		workflowQueries.lastApprovalInboxReq.Limit != 20 ||
		workflowQueries.lastApprovalInboxReq.State != "pending" {
		t.Fatalf("unexpected approval inbox request payload: %+v", workflowQueries.lastApprovalInboxReq)
	}
}

func TestTransportApprovalInboxRouteNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "workflow-query-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	_, err = client.ApprovalInbox(context.Background(), ApprovalInboxRequest{
		WorkspaceID: "ws1",
	}, "corr-approval-inbox")
	if err == nil {
		t.Fatalf("expected approval inbox to fail when workflow query service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportTaskRunListRoute(t *testing.T) {
	workflowQueries := &workflowQueryServiceStub{
		taskRunListResponse: TaskRunListResponse{
			WorkspaceID: "ws1",
			Items: []TaskRunListItem{
				{
					TaskID:                  "task-1",
					RunID:                   "run-1",
					WorkspaceID:             "ws1",
					Title:                   "Sample task",
					TaskState:               "running",
					RunState:                "running",
					Priority:                2,
					RequestedByActorID:      "actor.requester",
					SubjectPrincipalActorID: "actor.subject",
					ActingAsActorID:         "actor.subject",
					LastError:               "connector timeout",
					TaskCreatedAt:           "2026-02-24T00:00:00Z",
					TaskUpdatedAt:           "2026-02-24T00:00:05Z",
					RunCreatedAt:            "2026-02-24T00:00:01Z",
					RunUpdatedAt:            "2026-02-24T00:00:05Z",
					StartedAt:               "2026-02-24T00:00:02Z",
					Actions: TaskRunActionAvailability{
						CanCancel:  true,
						CanRetry:   false,
						CanRequeue: false,
					},
					Route: WorkflowRouteMetadata{
						Available:       true,
						TaskClass:       "chat",
						Provider:        "ollama",
						ModelKey:        "llama3.2",
						TaskClassSource: "task_channel",
						RouteSource:     "fallback_enabled",
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "workflow-query-token",
		WorkflowQueries: workflowQueries,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	response, err := client.TaskRunList(context.Background(), TaskRunListRequest{
		WorkspaceID: "ws1",
		State:       "running",
		Limit:       15,
	}, "corr-task-run-list")
	if err != nil {
		t.Fatalf("task run list: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].TaskID != "task-1" {
		t.Fatalf("unexpected task run list payload: %+v", response)
	}
	if !response.Items[0].Route.Available ||
		response.Items[0].Route.TaskClass != "chat" ||
		response.Items[0].Route.Provider != "ollama" ||
		response.Items[0].Route.ModelKey != "llama3.2" {
		t.Fatalf("expected task run list route metadata to round-trip, got %+v", response.Items[0].Route)
	}
	if !response.Items[0].Actions.CanCancel || response.Items[0].Actions.CanRetry || response.Items[0].Actions.CanRequeue {
		t.Fatalf("expected task run list action metadata to round-trip, got %+v", response.Items[0].Actions)
	}
	if workflowQueries.lastTaskRunListReq.WorkspaceID != "ws1" ||
		workflowQueries.lastTaskRunListReq.State != "running" ||
		workflowQueries.lastTaskRunListReq.Limit != 15 {
		t.Fatalf("unexpected task run list request payload: %+v", workflowQueries.lastTaskRunListReq)
	}
}

func TestTransportTaskStatusResponseDefaultsActionAvailabilityMetadata(t *testing.T) {
	backend := &taskStatusContractBackendStub{
		statusResponse: TaskStatusResponse{
			TaskID:    "task-contract-status",
			RunID:     "run-contract-status",
			State:     "queued",
			RunState:  "queued",
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := startTestServerWithBackend(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "task-action-contract-token",
	}, backend)

	request, err := http.NewRequest(http.MethodGet, "http://"+server.Address()+"/v1/tasks/task-contract-status", nil)
	if err != nil {
		t.Fatalf("build task status request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer task-action-contract-token")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute task status request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected task status 200, got %d", response.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode task status payload: %v", err)
	}
	assertTaskActionAvailabilityPayload(t, payload, false, false, false)
}

func TestTransportTaskRunListResponseDefaultsActionAvailabilityMetadata(t *testing.T) {
	workflowQueries := &workflowQueryServiceStub{
		taskRunListResponse: TaskRunListResponse{
			WorkspaceID: "ws1",
			Items: []TaskRunListItem{
				{
					TaskID:                  "task-contract-list",
					RunID:                   "run-contract-list",
					WorkspaceID:             "ws1",
					Title:                   "Contract list item",
					TaskState:               "queued",
					RunState:                "queued",
					Priority:                1,
					RequestedByActorID:      "actor.requester",
					SubjectPrincipalActorID: "actor.subject",
					TaskCreatedAt:           "2026-02-26T00:00:00Z",
					TaskUpdatedAt:           "2026-02-26T00:00:00Z",
				},
			},
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "task-action-contract-token",
		WorkflowQueries: workflowQueries,
	})

	requestBody, err := json.Marshal(TaskRunListRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("marshal task run list request: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+server.Address()+"/v1/tasks/list", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("build task run list request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer task-action-contract-token")
	request.Header.Set("Content-Type", "application/json")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute task run list request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected task run list 200, got %d", response.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode task run list payload: %v", err)
	}
	itemsRaw, ok := payload["items"]
	if !ok {
		t.Fatalf("expected items array in task run list payload")
	}
	items, ok := itemsRaw.([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected non-empty items array, got %T (%+v)", itemsRaw, itemsRaw)
	}
	firstItemRaw, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first task run item object, got %T", items[0])
	}
	assertTaskActionAvailabilityPayload(t, firstItemRaw, false, false, false)
}

func TestTransportTaskRunListRouteNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "workflow-query-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	_, err = client.TaskRunList(context.Background(), TaskRunListRequest{
		WorkspaceID: "ws1",
	}, "corr-task-run-list")
	if err == nil {
		t.Fatalf("expected task run list to fail when workflow query service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportCommInboxRoutes(t *testing.T) {
	workflowQueries := &workflowQueryServiceStub{
		commThreadResponse: CommThreadListResponse{
			WorkspaceID: "ws1",
			Items: []CommThreadListItem{
				{
					ThreadID:             "thread-1",
					WorkspaceID:          "ws1",
					Channel:              "message",
					ConnectorID:          "twilio",
					Title:                "Thread One",
					LastEventID:          "event-1",
					LastEventType:        "MESSAGE",
					LastDirection:        "INBOUND",
					LastOccurredAt:       "2026-02-25T00:00:02Z",
					ParticipantAddresses: []string{"+15555550100"},
					EventCount:           2,
					CreatedAt:            "2026-02-25T00:00:00Z",
					UpdatedAt:            "2026-02-25T00:00:02Z",
				},
			},
			HasMore:    true,
			NextCursor: "2026-02-25T00:00:02Z|thread-1",
		},
		commEventResponse: CommEventTimelineResponse{
			WorkspaceID: "ws1",
			Items: []CommEventTimelineItem{
				{
					EventID:          "event-1",
					WorkspaceID:      "ws1",
					ThreadID:         "thread-1",
					Channel:          "message",
					ConnectorID:      "twilio",
					EventType:        "MESSAGE",
					Direction:        "INBOUND",
					AssistantEmitted: false,
					BodyText:         "hello",
					OccurredAt:       "2026-02-25T00:00:02Z",
					CreatedAt:        "2026-02-25T00:00:02Z",
					Addresses: []CommEventAddressItem{
						{Role: "FROM", Value: "+15555550100", Position: 0},
					},
				},
			},
			HasMore:    false,
			NextCursor: "",
		},
		commCallResponse: CommCallSessionListResponse{
			WorkspaceID: "ws1",
			Items: []CommCallSessionListItem{
				{
					SessionID:      "call-1",
					WorkspaceID:    "ws1",
					Provider:       "twilio",
					ConnectorID:    "twilio",
					ProviderCallID: "CA123",
					ThreadID:       "thread-voice-1",
					Direction:      "inbound",
					FromAddress:    "+15555550101",
					ToAddress:      "+15555550002",
					Status:         "in_progress",
					UpdatedAt:      "2026-02-25T00:00:03Z",
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "workflow-query-token",
		WorkflowQueries: workflowQueries,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	threadResponse, err := client.CommThreadList(context.Background(), CommThreadListRequest{
		WorkspaceID: "ws1",
		Channel:     "message",
		ConnectorID: "twilio",
		Cursor:      "2026-02-25T00:00:05Z|thread-9",
		Limit:       10,
	}, "corr-comm-threads")
	if err != nil {
		t.Fatalf("comm thread list: %v", err)
	}
	if len(threadResponse.Items) != 1 || threadResponse.Items[0].ThreadID != "thread-1" {
		t.Fatalf("unexpected comm thread response: %+v", threadResponse)
	}
	if workflowQueries.lastCommThreadReq.WorkspaceID != "ws1" ||
		workflowQueries.lastCommThreadReq.Channel != "message" ||
		workflowQueries.lastCommThreadReq.ConnectorID != "twilio" ||
		workflowQueries.lastCommThreadReq.Cursor != "2026-02-25T00:00:05Z|thread-9" ||
		workflowQueries.lastCommThreadReq.Limit != 10 {
		t.Fatalf("unexpected comm thread request: %+v", workflowQueries.lastCommThreadReq)
	}

	eventResponse, err := client.CommEventTimeline(context.Background(), CommEventTimelineRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
		ConnectorID: "twilio",
		Limit:       25,
	}, "corr-comm-events")
	if err != nil {
		t.Fatalf("comm event list: %v", err)
	}
	if len(eventResponse.Items) != 1 || eventResponse.Items[0].EventID != "event-1" {
		t.Fatalf("unexpected comm event response: %+v", eventResponse)
	}
	if workflowQueries.lastCommEventReq.WorkspaceID != "ws1" ||
		workflowQueries.lastCommEventReq.ThreadID != "thread-1" ||
		workflowQueries.lastCommEventReq.ConnectorID != "twilio" ||
		workflowQueries.lastCommEventReq.Limit != 25 {
		t.Fatalf("unexpected comm event request: %+v", workflowQueries.lastCommEventReq)
	}

	callResponse, err := client.CommCallSessionList(context.Background(), CommCallSessionListRequest{
		WorkspaceID: "ws1",
		ConnectorID: "twilio",
		Status:      "in_progress",
		Limit:       5,
	}, "corr-comm-calls")
	if err != nil {
		t.Fatalf("comm call-session list: %v", err)
	}
	if len(callResponse.Items) != 1 || callResponse.Items[0].SessionID != "call-1" {
		t.Fatalf("unexpected comm call-session response: %+v", callResponse)
	}
	if workflowQueries.lastCommCallReq.WorkspaceID != "ws1" ||
		workflowQueries.lastCommCallReq.ConnectorID != "twilio" ||
		workflowQueries.lastCommCallReq.Status != "in_progress" ||
		workflowQueries.lastCommCallReq.Limit != 5 {
		t.Fatalf("unexpected comm call-session request: %+v", workflowQueries.lastCommCallReq)
	}
}

func TestTransportCommInboxRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "workflow-query-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "workflow-query-token",
	})
	if err != nil {
		t.Fatalf("create workflow query client: %v", err)
	}

	_, err = client.CommThreadList(context.Background(), CommThreadListRequest{
		WorkspaceID: "ws1",
	}, "corr-comm-threads")
	if err == nil {
		t.Fatalf("expected comm thread list to fail when workflow query service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportIdentityDirectoryRoutes(t *testing.T) {
	identity := &identityDirectoryServiceStub{
		workspacesResponse: IdentityWorkspacesResponse{
			ActiveContext: IdentityActiveContext{
				WorkspaceID:       "ws1",
				PrincipalActorID:  "actor.requester",
				WorkspaceSource:   "selected",
				PrincipalSource:   "derived",
				LastUpdatedAt:     "2026-02-25T00:00:00Z",
				WorkspaceResolved: true,
				MutationSource:    "app",
				MutationReason:    "explicit_select_workspace",
				SelectionVersion:  3,
			},
			Workspaces: []IdentityWorkspaceRecord{
				{
					WorkspaceID:    "ws1",
					Name:           "Workspace One",
					Status:         "ACTIVE",
					PrincipalCount: 2,
					ActorCount:     2,
					HandleCount:    1,
					UpdatedAt:      "2026-02-25T00:00:00Z",
					IsActive:       true,
				},
			},
		},
		principalsResponse: IdentityPrincipalsResponse{
			WorkspaceID: "ws1",
			ActiveContext: IdentityActiveContext{
				WorkspaceID:       "ws1",
				PrincipalActorID:  "actor.requester",
				WorkspaceSource:   "selected",
				PrincipalSource:   "selected",
				LastUpdatedAt:     "2026-02-25T00:00:00Z",
				WorkspaceResolved: true,
				MutationSource:    "app",
				MutationReason:    "explicit_select_workspace",
				SelectionVersion:  3,
			},
			Principals: []IdentityPrincipalRecord{
				{
					ActorID:         "actor.requester",
					DisplayName:     "Requester",
					ActorType:       "human",
					ActorStatus:     "ACTIVE",
					PrincipalStatus: "ACTIVE",
					IsActive:        true,
					Handles: []IdentityActorHandleRecord{
						{
							Channel:     "imessage",
							HandleValue: "+15550000001",
							IsPrimary:   true,
							UpdatedAt:   "2026-02-25T00:00:00Z",
						},
					},
				},
			},
		},
		activeContextResponse: IdentityActiveContextResponse{
			ActiveContext: IdentityActiveContext{
				WorkspaceID:       "ws1",
				PrincipalActorID:  "actor.requester",
				WorkspaceSource:   "selected",
				PrincipalSource:   "selected",
				LastUpdatedAt:     "2026-02-25T00:00:00Z",
				WorkspaceResolved: true,
				MutationSource:    "app",
				MutationReason:    "explicit_select_workspace",
				SelectionVersion:  3,
			},
		},
		selectContextResponse: IdentityActiveContextResponse{
			ActiveContext: IdentityActiveContext{
				WorkspaceID:       "ws2",
				PrincipalActorID:  "actor.approver",
				WorkspaceSource:   "selected",
				PrincipalSource:   "selected",
				LastUpdatedAt:     "2026-02-25T00:00:01Z",
				WorkspaceResolved: true,
				MutationSource:    "cli",
				MutationReason:    "explicit_select_workspace",
				SelectionVersion:  4,
			},
		},
		bootstrapResponse: IdentityBootstrapResponse{
			WorkspaceID:      "ws3",
			PrincipalActorID: "actor.bootstrap",
			WorkspaceCreated: true,
			PrincipalCreated: true,
			PrincipalLinked:  true,
			HandleCreated:    true,
			Idempotent:       false,
			AuditLogID:       "audit-bootstrap-1",
			Handle: &IdentityActorHandleRecord{
				Channel:     "message",
				HandleValue: "+15550000003",
				IsPrimary:   true,
				UpdatedAt:   "2026-02-25T00:00:01Z",
			},
			ActiveContext: IdentityActiveContext{
				WorkspaceID:       "ws3",
				PrincipalActorID:  "actor.bootstrap",
				WorkspaceSource:   "selected",
				PrincipalSource:   "selected",
				LastUpdatedAt:     "2026-02-25T00:00:01Z",
				WorkspaceResolved: true,
				MutationSource:    "cli",
				MutationReason:    "explicit_select_workspace",
				SelectionVersion:  5,
			},
		},
		devicesResponse: IdentityDeviceListResponse{
			WorkspaceID: "ws1",
			Items: []IdentityDeviceRecord{
				{
					DeviceID:               "device-1",
					WorkspaceID:            "ws1",
					UserID:                 "user-1",
					DeviceType:             "phone",
					Platform:               "ios",
					Label:                  "Requester iPhone",
					LastSeenAt:             "2026-02-25T00:10:00Z",
					CreatedAt:              "2026-02-25T00:00:00Z",
					SessionTotal:           2,
					SessionActiveCount:     1,
					SessionExpiredCount:    0,
					SessionRevokedCount:    1,
					SessionLatestStartedAt: "2026-02-25T00:05:00Z",
				},
			},
		},
		sessionsResponse: IdentitySessionListResponse{
			WorkspaceID:   "ws1",
			SessionHealth: "active",
			Items: []IdentitySessionRecord{
				{
					SessionID:        "session-1",
					WorkspaceID:      "ws1",
					DeviceID:         "device-1",
					UserID:           "user-1",
					DeviceType:       "phone",
					Platform:         "ios",
					DeviceLabel:      "Requester iPhone",
					DeviceLastSeenAt: "2026-02-25T00:10:00Z",
					StartedAt:        "2026-02-25T00:05:00Z",
					ExpiresAt:        "2099-01-01T00:00:00Z",
					SessionHealth:    "active",
				},
			},
		},
		sessionRevokeResponse: IdentitySessionRevokeResponse{
			WorkspaceID:      "ws1",
			SessionID:        "session-1",
			DeviceID:         "device-1",
			StartedAt:        "2026-02-25T00:05:00Z",
			ExpiresAt:        "2099-01-01T00:00:00Z",
			RevokedAt:        "2026-02-25T00:20:00Z",
			DeviceLastSeenAt: "2026-02-25T00:10:00Z",
			SessionHealth:    "revoked",
			Idempotent:       false,
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:      ListenerModeTCP,
		Address:           "127.0.0.1:0",
		AuthToken:         "identity-token",
		IdentityDirectory: identity,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "identity-token",
	})
	if err != nil {
		t.Fatalf("create identity directory client: %v", err)
	}

	workspaces, err := client.IdentityWorkspaces(context.Background(), IdentityWorkspacesRequest{IncludeInactive: true}, "corr-identity-workspaces")
	if err != nil {
		t.Fatalf("identity workspaces: %v", err)
	}
	if len(workspaces.Workspaces) != 1 || workspaces.Workspaces[0].WorkspaceID != "ws1" {
		t.Fatalf("unexpected identity workspaces response: %+v", workspaces)
	}
	if !identity.lastWorkspacesRequest.IncludeInactive {
		t.Fatalf("expected include_inactive request to round-trip")
	}

	principals, err := client.IdentityPrincipals(context.Background(), IdentityPrincipalsRequest{WorkspaceID: "ws1"}, "corr-identity-principals")
	if err != nil {
		t.Fatalf("identity principals: %v", err)
	}
	if len(principals.Principals) != 1 || len(principals.Principals[0].Handles) != 1 {
		t.Fatalf("unexpected identity principals response: %+v", principals)
	}
	if identity.lastPrincipalsRequest.WorkspaceID != "ws1" {
		t.Fatalf("unexpected principals request payload: %+v", identity.lastPrincipalsRequest)
	}

	activeContext, err := client.IdentityActiveContext(context.Background(), IdentityActiveContextRequest{WorkspaceID: "ws1"}, "corr-identity-context")
	if err != nil {
		t.Fatalf("identity context: %v", err)
	}
	if activeContext.ActiveContext.WorkspaceID != "ws1" || !activeContext.ActiveContext.WorkspaceResolved {
		t.Fatalf("unexpected identity context response: %+v", activeContext)
	}
	if activeContext.ActiveContext.MutationSource != "app" || activeContext.ActiveContext.MutationReason != "explicit_select_workspace" || activeContext.ActiveContext.SelectionVersion != 3 {
		t.Fatalf("expected mutation metadata in active context response, got %+v", activeContext.ActiveContext)
	}
	if identity.lastActiveContextReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected active context request payload: %+v", identity.lastActiveContextReq)
	}

	selectedContext, err := client.IdentitySelectWorkspace(context.Background(), IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws2",
		PrincipalActorID: "actor.approver",
		Source:           "app",
	}, "corr-identity-select")
	if err != nil {
		t.Fatalf("identity select workspace: %v", err)
	}
	if selectedContext.ActiveContext.WorkspaceID != "ws2" || selectedContext.ActiveContext.PrincipalActorID != "actor.approver" {
		t.Fatalf("unexpected identity select workspace response: %+v", selectedContext)
	}
	if selectedContext.ActiveContext.MutationSource != "cli" || selectedContext.ActiveContext.SelectionVersion != 4 {
		t.Fatalf("expected selected context mutation metadata, got %+v", selectedContext.ActiveContext)
	}
	if identity.lastSelectWorkspaceReq.WorkspaceID != "ws2" || identity.lastSelectWorkspaceReq.PrincipalActorID != "actor.approver" {
		t.Fatalf("unexpected select workspace request payload: %+v", identity.lastSelectWorkspaceReq)
	}

	bootstrap, err := client.IdentityBootstrap(context.Background(), IdentityBootstrapRequest{
		WorkspaceID:          "ws3",
		WorkspaceName:        "Workspace Three",
		PrincipalActorID:     "actor.bootstrap",
		PrincipalDisplayName: "Bootstrap User",
		PrincipalActorType:   "human",
		PrincipalStatus:      "ACTIVE",
		Handle: &IdentityBootstrapHandle{
			Channel:     "message",
			HandleValue: "+15550000003",
			IsPrimary:   true,
		},
		Source: "cli",
	}, "corr-identity-bootstrap")
	if err != nil {
		t.Fatalf("identity bootstrap: %v", err)
	}
	if bootstrap.WorkspaceID != "ws3" || bootstrap.PrincipalActorID != "actor.bootstrap" || !bootstrap.WorkspaceCreated || !bootstrap.HandleCreated {
		t.Fatalf("unexpected identity bootstrap response: %+v", bootstrap)
	}
	if identity.lastBootstrapRequest.WorkspaceID != "ws3" || identity.lastBootstrapRequest.PrincipalActorID != "actor.bootstrap" {
		t.Fatalf("unexpected bootstrap request payload: %+v", identity.lastBootstrapRequest)
	}
	if identity.lastBootstrapRequest.Handle == nil || identity.lastBootstrapRequest.Handle.HandleValue != "+15550000003" {
		t.Fatalf("unexpected bootstrap handle payload: %+v", identity.lastBootstrapRequest.Handle)
	}

	devices, err := client.IdentityDevices(context.Background(), IdentityDeviceListRequest{
		WorkspaceID: "ws1",
		UserID:      "user-1",
		Limit:       10,
	}, "corr-identity-devices")
	if err != nil {
		t.Fatalf("identity devices: %v", err)
	}
	if len(devices.Items) != 1 || devices.Items[0].DeviceID != "device-1" || devices.Items[0].SessionActiveCount != 1 {
		t.Fatalf("unexpected identity devices response: %+v", devices)
	}
	if identity.lastDevicesRequest.WorkspaceID != "ws1" || identity.lastDevicesRequest.UserID != "user-1" {
		t.Fatalf("unexpected devices request payload: %+v", identity.lastDevicesRequest)
	}

	sessions, err := client.IdentitySessions(context.Background(), IdentitySessionListRequest{
		WorkspaceID:   "ws1",
		DeviceID:      "device-1",
		SessionHealth: "active",
		Limit:         5,
	}, "corr-identity-sessions")
	if err != nil {
		t.Fatalf("identity sessions: %v", err)
	}
	if len(sessions.Items) != 1 || sessions.Items[0].SessionID != "session-1" || sessions.Items[0].SessionHealth != "active" {
		t.Fatalf("unexpected identity sessions response: %+v", sessions)
	}
	if identity.lastSessionsRequest.DeviceID != "device-1" || identity.lastSessionsRequest.SessionHealth != "active" {
		t.Fatalf("unexpected sessions request payload: %+v", identity.lastSessionsRequest)
	}

	revoke, err := client.IdentitySessionRevoke(context.Background(), IdentitySessionRevokeRequest{
		WorkspaceID: "ws1",
		SessionID:   "session-1",
	}, "corr-identity-revoke")
	if err != nil {
		t.Fatalf("identity session revoke: %v", err)
	}
	if revoke.SessionID != "session-1" || revoke.SessionHealth != "revoked" || revoke.Idempotent {
		t.Fatalf("unexpected identity session revoke response: %+v", revoke)
	}
	if identity.lastSessionRevokeReq.WorkspaceID != "ws1" || identity.lastSessionRevokeReq.SessionID != "session-1" {
		t.Fatalf("unexpected session revoke request payload: %+v", identity.lastSessionRevokeReq)
	}
}

func TestTransportIdentityDirectoryRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "identity-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "identity-token",
	})
	if err != nil {
		t.Fatalf("create identity directory client: %v", err)
	}

	_, err = client.IdentityWorkspaces(context.Background(), IdentityWorkspacesRequest{}, "corr-identity-workspaces")
	if err == nil {
		t.Fatalf("expected identity workspaces route to fail when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.IdentitySessions(context.Background(), IdentitySessionListRequest{
		WorkspaceID: "ws1",
	}, "corr-identity-sessions")
	if err == nil {
		t.Fatalf("expected identity sessions route to fail when service is not configured")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError for sessions, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501 for identity sessions, got %d", httpErr.StatusCode)
	}

	_, err = client.IdentityBootstrap(context.Background(), IdentityBootstrapRequest{
		WorkspaceID:      "ws1",
		PrincipalActorID: "actor.requester",
	}, "corr-identity-bootstrap")
	if err == nil {
		t.Fatalf("expected identity bootstrap route to fail when service is not configured")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError for identity bootstrap, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501 for identity bootstrap, got %d", httpErr.StatusCode)
	}
}

func TestTransportDelegationRoutesAllowDenyAndRevoke(t *testing.T) {
	delegation := &delegationServiceStub{
		grantResponse: DelegationRuleRecord{
			ID:          "rule-1",
			WorkspaceID: "ws1",
			FromActorID: "actor.requester",
			ToActorID:   "actor.approver",
			ScopeType:   "EXECUTION",
			Status:      "ACTIVE",
			CreatedAt:   "2026-02-25T00:00:00Z",
		},
		listResponse: DelegationListResponse{
			WorkspaceID: "ws1",
			Rules: []DelegationRuleRecord{
				{
					ID:          "rule-1",
					WorkspaceID: "ws1",
					FromActorID: "actor.requester",
					ToActorID:   "actor.approver",
					ScopeType:   "EXECUTION",
					Status:      "ACTIVE",
					CreatedAt:   "2026-02-25T00:00:00Z",
				},
			},
		},
		revokeResponse: DelegationRevokeResponse{
			WorkspaceID: "ws1",
			RuleID:      "rule-1",
			Status:      "REVOKED",
		},
		checkResponse: DelegationCheckResponse{
			WorkspaceID:        "ws1",
			RequestedByActorID: "actor.requester",
			ActingAsActorID:    "actor.approver",
			Allowed:            false,
			Reason:             "missing valid delegation rule",
			ReasonCode:         "missing_delegation_rule",
		},
		capabilityUpsertResp: CapabilityGrantRecord{
			GrantID:       "grant-1",
			WorkspaceID:   "ws1",
			ActorID:       "actor.requester",
			CapabilityKey: "messages_send_sms",
			ScopeJSON:     `{"channel":"sms"}`,
			Status:        "ACTIVE",
			CreatedAt:     "2026-02-25T00:00:04Z",
			ExpiresAt:     "2026-02-27T00:00:00Z",
		},
		capabilityListResp: CapabilityGrantListResponse{
			WorkspaceID: "ws1",
			Items: []CapabilityGrantRecord{
				{
					GrantID:       "grant-1",
					WorkspaceID:   "ws1",
					ActorID:       "actor.requester",
					CapabilityKey: "messages_send_sms",
					ScopeJSON:     `{"channel":"sms"}`,
					Status:        "ACTIVE",
					CreatedAt:     "2026-02-25T00:00:04Z",
					ExpiresAt:     "2026-02-27T00:00:00Z",
				},
			},
			HasMore:             true,
			NextCursorCreatedAt: "2026-02-25T00:00:04Z",
			NextCursorID:        "grant-1",
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "delegation-token",
		Delegation:   delegation,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "delegation-token",
	})
	if err != nil {
		t.Fatalf("create delegation client: %v", err)
	}

	grant, err := client.DelegationGrant(context.Background(), DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.approver",
		ScopeType:   "EXECUTION",
	}, "corr-delegation-grant")
	if err != nil {
		t.Fatalf("delegation grant: %v", err)
	}
	if grant.ID != "rule-1" {
		t.Fatalf("unexpected delegation grant response: %+v", grant)
	}
	if delegation.lastGrantReq.ScopeType != "EXECUTION" {
		t.Fatalf("unexpected delegation grant request payload: %+v", delegation.lastGrantReq)
	}

	check, err := client.DelegationCheck(context.Background(), DelegationCheckRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.approver",
		ScopeType:          "EXECUTION",
	}, "corr-delegation-check")
	if err != nil {
		t.Fatalf("delegation check: %v", err)
	}
	if check.Allowed || check.ReasonCode != "missing_delegation_rule" {
		t.Fatalf("expected deny reason response for delegation check, got %+v", check)
	}
	if delegation.lastCheckReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected delegation check request payload: %+v", delegation.lastCheckReq)
	}

	list, err := client.DelegationList(context.Background(), DelegationListRequest{WorkspaceID: "ws1"}, "corr-delegation-list")
	if err != nil {
		t.Fatalf("delegation list: %v", err)
	}
	if len(list.Rules) != 1 || list.Rules[0].ID != "rule-1" {
		t.Fatalf("unexpected delegation list response: %+v", list)
	}
	if delegation.lastListReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected delegation list request payload: %+v", delegation.lastListReq)
	}

	revoke, err := client.DelegationRevoke(context.Background(), DelegationRevokeRequest{
		WorkspaceID: "ws1",
		RuleID:      "rule-1",
	}, "corr-delegation-revoke")
	if err != nil {
		t.Fatalf("delegation revoke: %v", err)
	}
	if revoke.Status != "REVOKED" {
		t.Fatalf("expected revoked response status, got %+v", revoke)
	}
	if delegation.lastRevokeReq.RuleID != "rule-1" {
		t.Fatalf("unexpected delegation revoke request payload: %+v", delegation.lastRevokeReq)
	}

	capabilityUpsert, err := client.CapabilityGrantUpsert(context.Background(), CapabilityGrantUpsertRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "messages_send_sms",
		ScopeJSON:     `{"channel":"sms"}`,
		Status:        "ACTIVE",
		ExpiresAt:     "2026-02-27T00:00:00Z",
	}, "corr-capability-upsert")
	if err != nil {
		t.Fatalf("capability grant upsert: %v", err)
	}
	if capabilityUpsert.GrantID != "grant-1" || capabilityUpsert.CapabilityKey != "messages_send_sms" {
		t.Fatalf("unexpected capability grant upsert response: %+v", capabilityUpsert)
	}
	if delegation.lastCapabilityUpsertReq.ActorID != "actor.requester" || delegation.lastCapabilityUpsertReq.CapabilityKey != "messages_send_sms" {
		t.Fatalf("unexpected capability grant upsert request payload: %+v", delegation.lastCapabilityUpsertReq)
	}

	capabilityList, err := client.CapabilityGrantList(context.Background(), CapabilityGrantListRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "messages_send_sms",
		Status:        "ACTIVE",
		Limit:         1,
	}, "corr-capability-list")
	if err != nil {
		t.Fatalf("capability grant list: %v", err)
	}
	if len(capabilityList.Items) != 1 || capabilityList.Items[0].GrantID != "grant-1" {
		t.Fatalf("unexpected capability grant list response: %+v", capabilityList)
	}
	if capabilityList.NextCursorCreatedAt == "" || capabilityList.NextCursorID == "" || !capabilityList.HasMore {
		t.Fatalf("expected capability grant cursor metadata, got %+v", capabilityList)
	}
	if delegation.lastCapabilityListReq.WorkspaceID != "ws1" || delegation.lastCapabilityListReq.Status != "ACTIVE" {
		t.Fatalf("unexpected capability grant list request payload: %+v", delegation.lastCapabilityListReq)
	}
}

func TestTransportDelegationRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "delegation-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "delegation-token",
	})
	if err != nil {
		t.Fatalf("create delegation client: %v", err)
	}

	_, err = client.DelegationGrant(context.Background(), DelegationGrantRequest{
		WorkspaceID: "ws1",
		FromActorID: "actor.requester",
		ToActorID:   "actor.approver",
	}, "corr-delegation-grant")
	if err == nil {
		t.Fatalf("expected delegation grant to fail when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportAutomationUpdateDeleteRoutes(t *testing.T) {
	enabled := false
	cooldown := 0
	intervalSeconds := 600

	automation := &automationServiceStub{
		updateResponse: AutomationUpdateResponse{
			Trigger: AutomationTriggerRecord{
				TriggerID:             "trg-1",
				WorkspaceID:           "ws1",
				DirectiveID:           "dir-1",
				TriggerType:           "SCHEDULE",
				Enabled:               false,
				FilterJSON:            `{"interval_seconds":600}`,
				CooldownSeconds:       0,
				SubjectPrincipalActor: "actor.requester",
				DirectiveTitle:        "Updated schedule automation",
				DirectiveInstruction:  "updated instruction",
				DirectiveStatus:       "ACTIVE",
				CreatedAt:             "2026-02-24T00:00:00Z",
				UpdatedAt:             "2026-02-24T00:00:10Z",
			},
			Updated:    true,
			Idempotent: false,
		},
		deleteResponse: AutomationDeleteResponse{
			WorkspaceID: "ws1",
			TriggerID:   "trg-1",
			DirectiveID: "dir-1",
			Deleted:     true,
			Idempotent:  false,
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "automation-token",
		Automation:   automation,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "automation-token",
	})
	if err != nil {
		t.Fatalf("create automation client: %v", err)
	}

	updateResponse, err := client.AutomationUpdate(context.Background(), AutomationUpdateRequest{
		WorkspaceID:     "ws1",
		TriggerID:       "trg-1",
		Title:           "Updated schedule automation",
		Instruction:     "updated instruction",
		IntervalSeconds: &intervalSeconds,
		CooldownSeconds: &cooldown,
		Enabled:         &enabled,
	}, "corr-automation-update")
	if err != nil {
		t.Fatalf("automation update: %v", err)
	}
	if !updateResponse.Updated || updateResponse.Idempotent {
		t.Fatalf("expected updated non-idempotent response, got %+v", updateResponse)
	}
	if updateResponse.Trigger.TriggerID != "trg-1" || updateResponse.Trigger.FilterJSON != `{"interval_seconds":600}` {
		t.Fatalf("unexpected automation update payload: %+v", updateResponse)
	}
	if automation.lastUpdateReq.WorkspaceID != "ws1" || automation.lastUpdateReq.TriggerID != "trg-1" {
		t.Fatalf("unexpected automation update request payload: %+v", automation.lastUpdateReq)
	}
	if automation.lastUpdateReq.IntervalSeconds == nil || *automation.lastUpdateReq.IntervalSeconds != 600 {
		t.Fatalf("expected interval_seconds=600, got %+v", automation.lastUpdateReq.IntervalSeconds)
	}
	if automation.lastUpdateReq.Enabled == nil || *automation.lastUpdateReq.Enabled {
		t.Fatalf("expected enabled=false, got %+v", automation.lastUpdateReq.Enabled)
	}

	deleteResponse, err := client.AutomationDelete(context.Background(), AutomationDeleteRequest{
		WorkspaceID: "ws1",
		TriggerID:   "trg-1",
	}, "corr-automation-delete")
	if err != nil {
		t.Fatalf("automation delete: %v", err)
	}
	if !deleteResponse.Deleted || deleteResponse.Idempotent {
		t.Fatalf("expected deleted non-idempotent response, got %+v", deleteResponse)
	}
	if automation.lastDeleteReq.WorkspaceID != "ws1" || automation.lastDeleteReq.TriggerID != "trg-1" {
		t.Fatalf("unexpected automation delete request payload: %+v", automation.lastDeleteReq)
	}
}

func TestTransportAutomationFireHistoryRoute(t *testing.T) {
	automation := &automationServiceStub{
		fireHistoryResponse: AutomationFireHistoryResponse{
			WorkspaceID: "ws1",
			Fires: []AutomationFireHistoryRecord{
				{
					FireID:            "fire-1",
					WorkspaceID:       "ws1",
					TriggerID:         "trg-1",
					TriggerType:       "SCHEDULE",
					DirectiveID:       "dir-1",
					Status:            "created_task",
					Outcome:           "CREATED_TASK",
					IdempotencyKey:    "ws1:trg-1:schedule:2026-02-24T00:00:00Z",
					IdempotencySignal: "schedule:2026-02-24T00:00:00Z",
					FiredAt:           "2026-02-24T00:00:00Z",
					TaskID:            "task-1",
					RunID:             "run-1",
					Route: WorkflowRouteMetadata{
						Available:       true,
						TaskClass:       "chat",
						Provider:        "ollama",
						ModelKey:        "llama3.2",
						TaskClassSource: "run_step_capability",
						RouteSource:     "fallback_enabled",
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "automation-token",
		Automation:   automation,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "automation-token",
	})
	if err != nil {
		t.Fatalf("create automation client: %v", err)
	}

	response, err := client.AutomationFireHistory(context.Background(), AutomationFireHistoryRequest{
		WorkspaceID: "ws1",
		TriggerID:   "trg-1",
		Status:      "created_task",
		Limit:       10,
	}, "corr-automation-fire-history")
	if err != nil {
		t.Fatalf("automation fire-history: %v", err)
	}
	if response.WorkspaceID != "ws1" || len(response.Fires) != 1 || response.Fires[0].FireID != "fire-1" {
		t.Fatalf("unexpected automation fire-history payload: %+v", response)
	}
	if !response.Fires[0].Route.Available ||
		response.Fires[0].Route.TaskClass == "" ||
		response.Fires[0].Route.Provider == "" ||
		response.Fires[0].Route.ModelKey == "" {
		t.Fatalf("expected automation fire-history route metadata to round-trip, got %+v", response.Fires[0].Route)
	}
	if automation.lastFireHistoryReq.WorkspaceID != "ws1" || automation.lastFireHistoryReq.TriggerID != "trg-1" {
		t.Fatalf("unexpected automation fire-history request payload: %+v", automation.lastFireHistoryReq)
	}
}

func TestTransportAutomationCommTriggerMetadataValidateRoutes(t *testing.T) {
	automation := &automationServiceStub{
		metadataResponse: AutomationCommTriggerMetadataResponse{
			TriggerType: "ON_COMM_EVENT",
			RequiredDefaults: AutomationCommTriggerRequiredDefaults{
				EventType:        "MESSAGE",
				Direction:        "INBOUND",
				AssistantEmitted: false,
			},
			IdempotencyKeyFields: []string{"workspace_id", "trigger_id", "source_event_id"},
			FilterDefaults: AutomationCommTriggerFilter{
				Channels:          []string{},
				PrincipalActorIDs: []string{},
				SenderAllowlist:   []string{},
				ThreadIDs:         []string{},
				Keywords: AutomationCommTriggerKeywordFilter{
					ContainsAny:  []string{},
					ContainsAll:  []string{},
					ExactPhrases: []string{},
				},
			},
			FilterSchema: []AutomationCommTriggerFilterFieldSchema{
				{Field: "channels", ValueType: "string[]", MatchSemantics: "case_insensitive_equals_any", Description: "channel include list"},
			},
			Compatibility: AutomationCommTriggerMetadataCompatibility{
				PrincipalFilterBehavior: "principal_actor_ids compare against trigger subject_principal_actor_id",
				KeywordMatchBehavior:    "keyword matching is case-insensitive substring matching",
			},
		},
		validateResponse: AutomationCommTriggerValidateResponse{
			Valid:                true,
			TriggerType:          "ON_COMM_EVENT",
			RequiredDefaults:     AutomationCommTriggerRequiredDefaults{EventType: "MESSAGE", Direction: "INBOUND", AssistantEmitted: false},
			NormalizedFilter:     AutomationCommTriggerFilter{Channels: []string{"twilio_sms"}, PrincipalActorIDs: []string{"actor.requester"}, SenderAllowlist: []string{}, ThreadIDs: []string{}, Keywords: AutomationCommTriggerKeywordFilter{ContainsAny: []string{"hello"}, ContainsAll: []string{}, ExactPhrases: []string{}}},
			NormalizedFilterJSON: `{"channels":["twilio_sms"],"principal_actor_ids":["actor.requester"],"sender_allowlist":[],"thread_ids":[],"keywords":{"contains_any":["hello"],"contains_all":[],"exact_phrases":[]}}`,
			Errors:               []AutomationCommTriggerValidationIssue{},
			Warnings:             []AutomationCommTriggerValidationIssue{},
			Compatibility: AutomationCommTriggerValidationCompatibility{
				Compatible:                  true,
				SubjectActorID:              "actor.requester",
				SubjectMatchesPrincipalRule: true,
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "automation-token",
		Automation:   automation,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "automation-token",
	})
	if err != nil {
		t.Fatalf("create automation client: %v", err)
	}

	metadataResponse, err := client.AutomationCommTriggerMetadata(context.Background(), AutomationCommTriggerMetadataRequest{
		WorkspaceID: "ws1",
	}, "corr-automation-comm-metadata")
	if err != nil {
		t.Fatalf("automation comm-trigger metadata: %v", err)
	}
	if metadataResponse.TriggerType != "ON_COMM_EVENT" || metadataResponse.RequiredDefaults.EventType != "MESSAGE" {
		t.Fatalf("unexpected automation comm-trigger metadata payload: %+v", metadataResponse)
	}
	if len(metadataResponse.IdempotencyKeyFields) != 3 {
		t.Fatalf("expected idempotency contract fields, got %+v", metadataResponse.IdempotencyKeyFields)
	}
	if automation.lastCommMetadataReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected metadata request payload: %+v", automation.lastCommMetadataReq)
	}

	validateResponse, err := client.AutomationCommTriggerValidate(context.Background(), AutomationCommTriggerValidateRequest{
		WorkspaceID:    "ws1",
		SubjectActorID: "actor.requester",
		Filter: &AutomationCommTriggerFilter{
			Channels:          []string{"twilio_sms"},
			PrincipalActorIDs: []string{"actor.requester"},
			SenderAllowlist:   []string{},
			ThreadIDs:         []string{},
			Keywords:          AutomationCommTriggerKeywordFilter{ContainsAny: []string{"hello"}, ContainsAll: []string{}, ExactPhrases: []string{}},
		},
	}, "corr-automation-comm-validate")
	if err != nil {
		t.Fatalf("automation comm-trigger validate: %v", err)
	}
	if !validateResponse.Valid || validateResponse.TriggerType != "ON_COMM_EVENT" {
		t.Fatalf("unexpected automation comm-trigger validate payload: %+v", validateResponse)
	}
	if automation.lastCommValidateReq.SubjectActorID != "actor.requester" {
		t.Fatalf("unexpected validate request payload: %+v", automation.lastCommValidateReq)
	}
	if automation.lastCommValidateReq.Filter == nil || len(automation.lastCommValidateReq.Filter.Channels) != 1 {
		t.Fatalf("expected validate filter payload to round-trip, got %+v", automation.lastCommValidateReq.Filter)
	}
}

func TestTransportAutomationCommTriggerValidateRejectsLegacyFilterJSONInput(t *testing.T) {
	automation := &automationServiceStub{}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "automation-token",
		Automation:   automation,
	})

	body := bytes.NewBufferString(`{"workspace_id":"ws1","subject_actor_id":"actor.requester","filter_json":"{\"channels\":[\"twilio_sms\"]}"}`)
	request, err := http.NewRequest(http.MethodPost, "http://"+server.Address()+"/v1/automation/comm-trigger/validate", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer automation-token")
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for legacy filter_json payload, got %d", response.StatusCode)
	}
}

func TestTransportAutomationUpdateDeleteRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "automation-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "automation-token",
	})
	if err != nil {
		t.Fatalf("create automation client: %v", err)
	}

	_, err = client.AutomationUpdate(context.Background(), AutomationUpdateRequest{
		WorkspaceID: "ws1",
		TriggerID:   "trg-1",
	}, "corr-automation-update")
	if err == nil {
		t.Fatalf("expected automation update to fail when automation service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.AutomationDelete(context.Background(), AutomationDeleteRequest{
		WorkspaceID: "ws1",
		TriggerID:   "trg-1",
	}, "corr-automation-delete")
	if err == nil {
		t.Fatalf("expected automation delete to fail when automation service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.AutomationFireHistory(context.Background(), AutomationFireHistoryRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-automation-fire-history")
	if err == nil {
		t.Fatalf("expected automation fire-history to fail when automation service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.AutomationCommTriggerMetadata(context.Background(), AutomationCommTriggerMetadataRequest{
		WorkspaceID: "ws1",
	}, "corr-automation-comm-metadata")
	if err == nil {
		t.Fatalf("expected automation comm-trigger metadata to fail when automation service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.AutomationCommTriggerValidate(context.Background(), AutomationCommTriggerValidateRequest{
		WorkspaceID: "ws1",
	}, "corr-automation-comm-validate")
	if err == nil {
		t.Fatalf("expected automation comm-trigger validate to fail when automation service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportInspectLogsQueryAndStreamRoutes(t *testing.T) {
	inspectStub := &inspectServiceStub{
		queryResponse: InspectLogQueryResponse{
			WorkspaceID: "ws1",
			Logs: []InspectLogRecord{
				{
					LogID:     "audit-1",
					EventType: "STEP_EXECUTED",
					Status:    "completed",
					CreatedAt: "2026-02-24T00:00:00Z",
					Route: WorkflowRouteMetadata{
						Available:       true,
						TaskClass:       "browser",
						Provider:        "ollama",
						ModelKey:        "llama3.2",
						TaskClassSource: "step_capability",
						RouteSource:     "fallback_enabled",
					},
				},
			},
		},
		streamResponse: InspectLogStreamResponse{
			WorkspaceID: "ws1",
			Logs: []InspectLogRecord{
				{
					LogID:     "audit-2",
					EventType: "APPROVAL_GRANTED",
					Status:    "approved",
					CreatedAt: "2026-02-24T00:00:01Z",
					Route: WorkflowRouteMetadata{
						Available:       true,
						TaskClass:       "finder",
						Provider:        "ollama",
						ModelKey:        "llama3.2",
						TaskClassSource: "step_capability",
						RouteSource:     "fallback_enabled",
					},
				},
			},
			CursorCreatedAt: "2026-02-24T00:00:01Z",
			CursorID:        "audit-2",
			TimedOut:        false,
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "inspect-token",
		Inspect:      inspectStub,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "inspect-token",
	})
	if err != nil {
		t.Fatalf("create inspect client: %v", err)
	}

	queryResponse, err := client.InspectLogsQuery(context.Background(), InspectLogQueryRequest{
		WorkspaceID: "ws1",
		Limit:       25,
	}, "corr-inspect-query")
	if err != nil {
		t.Fatalf("inspect logs query: %v", err)
	}
	if len(queryResponse.Logs) != 1 || queryResponse.Logs[0].LogID != "audit-1" {
		t.Fatalf("unexpected inspect logs query payload: %+v", queryResponse)
	}
	if !queryResponse.Logs[0].Route.Available ||
		queryResponse.Logs[0].Route.Provider != "ollama" ||
		queryResponse.Logs[0].Route.ModelKey != "llama3.2" {
		t.Fatalf("expected inspect query route metadata to round-trip, got %+v", queryResponse.Logs[0].Route)
	}
	if inspectStub.lastQueryReq.WorkspaceID != "ws1" || inspectStub.lastQueryReq.Limit != 25 {
		t.Fatalf("unexpected inspect query request: %+v", inspectStub.lastQueryReq)
	}

	streamResponse, err := client.InspectLogsStream(context.Background(), InspectLogStreamRequest{
		WorkspaceID: "ws1",
		TimeoutMS:   1500,
	}, "corr-inspect-stream")
	if err != nil {
		t.Fatalf("inspect logs stream: %v", err)
	}
	if len(streamResponse.Logs) != 1 || streamResponse.Logs[0].LogID != "audit-2" {
		t.Fatalf("unexpected inspect logs stream payload: %+v", streamResponse)
	}
	if !streamResponse.Logs[0].Route.Available ||
		streamResponse.Logs[0].Route.Provider != "ollama" ||
		streamResponse.Logs[0].Route.ModelKey != "llama3.2" {
		t.Fatalf("expected inspect stream route metadata to round-trip, got %+v", streamResponse.Logs[0].Route)
	}
	if inspectStub.lastStreamReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected inspect stream request: %+v", inspectStub.lastStreamReq)
	}
}

func TestTransportInspectRunRouteIncludesRouteMetadata(t *testing.T) {
	inspectStub := &inspectServiceStub{
		inspectRunResponse: InspectRunResponse{
			Task: InspectTask{
				TaskID:      "task-1",
				WorkspaceID: "ws1",
			},
			Run: InspectTaskRun{
				RunID:       "run-1",
				WorkspaceID: "ws1",
				TaskID:      "task-1",
			},
			Route: WorkflowRouteMetadata{
				Available:       true,
				TaskClass:       "browser",
				Provider:        "ollama",
				ModelKey:        "llama3.2",
				TaskClassSource: "step_capability",
				RouteSource:     "fallback_enabled",
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "inspect-token",
		Inspect:      inspectStub,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "inspect-token",
	})
	if err != nil {
		t.Fatalf("create inspect client: %v", err)
	}

	response, err := client.InspectRun(context.Background(), InspectRunRequest{
		RunID: "run-1",
	}, "corr-inspect-run")
	if err != nil {
		t.Fatalf("inspect run: %v", err)
	}
	if response.Run.RunID != "run-1" {
		t.Fatalf("unexpected inspect run payload: %+v", response.Run)
	}
	if !response.Route.Available ||
		response.Route.Provider != "ollama" ||
		response.Route.ModelKey != "llama3.2" ||
		response.Route.TaskClass != "browser" {
		t.Fatalf("expected inspect run route metadata to round-trip, got %+v", response.Route)
	}
	if inspectStub.lastInspectRunReq.RunID != "run-1" {
		t.Fatalf("unexpected inspect run request payload: %+v", inspectStub.lastInspectRunReq)
	}
}

func TestTransportInspectLogsRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "inspect-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "inspect-token",
	})
	if err != nil {
		t.Fatalf("create inspect client: %v", err)
	}

	_, err = client.InspectLogsQuery(context.Background(), InspectLogQueryRequest{
		WorkspaceID: "ws1",
	}, "corr-inspect-query")
	if err == nil {
		t.Fatalf("expected inspect logs query to fail when inspect service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.InspectLogsStream(context.Background(), InspectLogStreamRequest{
		WorkspaceID: "ws1",
	}, "corr-inspect-stream")
	if err == nil {
		t.Fatalf("expected inspect logs stream to fail when inspect service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportContextQueryRoutes(t *testing.T) {
	contextStub := &contextOpsServiceStub{
		memoryInventoryResponse: ContextMemoryInventoryResponse{
			WorkspaceID: "ws1",
			Items: []ContextMemoryInventoryItem{
				{
					MemoryID:    "mem-1",
					WorkspaceID: "ws1",
					Status:      "ACTIVE",
					UpdatedAt:   "2026-02-25T00:00:00Z",
					Sources: []ContextMemorySourceRecord{
						{SourceID: "src-1", SourceType: "comm_event", SourceRef: "event-1", CreatedAt: "2026-02-25T00:00:00Z"},
					},
				},
			},
			HasMore:             true,
			NextCursorUpdatedAt: "2026-02-25T00:00:00Z",
			NextCursorID:        "mem-1",
		},
		memoryCandidatesResponse: ContextMemoryCandidatesResponse{
			WorkspaceID: "ws1",
			Items: []ContextMemoryCandidateItem{
				{
					CandidateID:   "cand-1",
					WorkspaceID:   "ws1",
					OwnerActorID:  "actor.subject",
					Status:        "PENDING",
					CandidateJSON: `{"kind":"summary","token_estimate":32}`,
					CandidateKind: "summary",
					CreatedAt:     "2026-02-25T00:00:01Z",
				},
			},
		},
		retrievalDocumentsResponse: ContextRetrievalDocumentsResponse{
			WorkspaceID: "ws1",
			Items: []ContextRetrievalDocumentItem{
				{
					DocumentID:  "doc-1",
					WorkspaceID: "ws1",
					SourceURI:   "memory://doc/1",
					ChunkCount:  2,
					CreatedAt:   "2026-02-25T00:00:02Z",
				},
			},
		},
		retrievalChunksResponse: ContextRetrievalChunksResponse{
			WorkspaceID: "ws1",
			DocumentID:  "doc-1",
			Items: []ContextRetrievalChunkItem{
				{
					ChunkID:     "chunk-1",
					WorkspaceID: "ws1",
					DocumentID:  "doc-1",
					ChunkIndex:  0,
					TextBody:    "hello world",
					CreatedAt:   "2026-02-25T00:00:03Z",
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "context-token",
		ContextOps:   contextStub,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "context-token",
	})
	if err != nil {
		t.Fatalf("create context client: %v", err)
	}

	memoryInventory, err := client.ContextMemoryInventory(context.Background(), ContextMemoryInventoryRequest{
		WorkspaceID:  "ws1",
		OwnerActorID: "actor.subject",
		Limit:        10,
	}, "corr-context-memory-inventory")
	if err != nil {
		t.Fatalf("context memory inventory: %v", err)
	}
	if memoryInventory.WorkspaceID != "ws1" || len(memoryInventory.Items) != 1 || memoryInventory.Items[0].MemoryID != "mem-1" {
		t.Fatalf("unexpected context memory inventory payload: %+v", memoryInventory)
	}
	if contextStub.lastMemoryInventoryReq.OwnerActorID != "actor.subject" {
		t.Fatalf("unexpected context memory inventory request payload: %+v", contextStub.lastMemoryInventoryReq)
	}

	memoryCandidates, err := client.ContextMemoryCandidates(context.Background(), ContextMemoryCandidatesRequest{
		WorkspaceID:  "ws1",
		OwnerActorID: "actor.subject",
		Limit:        10,
	}, "corr-context-memory-candidates")
	if err != nil {
		t.Fatalf("context memory candidates: %v", err)
	}
	if memoryCandidates.WorkspaceID != "ws1" || len(memoryCandidates.Items) != 1 || memoryCandidates.Items[0].CandidateID != "cand-1" {
		t.Fatalf("unexpected context memory candidates payload: %+v", memoryCandidates)
	}
	if contextStub.lastMemoryCandidatesReq.OwnerActorID != "actor.subject" {
		t.Fatalf("unexpected context memory candidates request payload: %+v", contextStub.lastMemoryCandidatesReq)
	}

	retrievalDocuments, err := client.ContextRetrievalDocuments(context.Background(), ContextRetrievalDocumentsRequest{
		WorkspaceID:    "ws1",
		SourceURIQuery: "memory://doc",
		Limit:          10,
	}, "corr-context-retrieval-documents")
	if err != nil {
		t.Fatalf("context retrieval documents: %v", err)
	}
	if retrievalDocuments.WorkspaceID != "ws1" || len(retrievalDocuments.Items) != 1 || retrievalDocuments.Items[0].DocumentID != "doc-1" {
		t.Fatalf("unexpected context retrieval documents payload: %+v", retrievalDocuments)
	}
	if contextStub.lastRetrievalDocumentsReq.SourceURIQuery != "memory://doc" {
		t.Fatalf("unexpected context retrieval documents request payload: %+v", contextStub.lastRetrievalDocumentsReq)
	}

	retrievalChunks, err := client.ContextRetrievalChunks(context.Background(), ContextRetrievalChunksRequest{
		WorkspaceID: "ws1",
		DocumentID:  "doc-1",
		Limit:       10,
	}, "corr-context-retrieval-chunks")
	if err != nil {
		t.Fatalf("context retrieval chunks: %v", err)
	}
	if retrievalChunks.WorkspaceID != "ws1" || len(retrievalChunks.Items) != 1 || retrievalChunks.Items[0].ChunkID != "chunk-1" {
		t.Fatalf("unexpected context retrieval chunks payload: %+v", retrievalChunks)
	}
	if contextStub.lastRetrievalChunksReq.DocumentID != "doc-1" {
		t.Fatalf("unexpected context retrieval chunks request payload: %+v", contextStub.lastRetrievalChunksReq)
	}
}

func TestTransportContextQueryRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "context-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "context-token",
	})
	if err != nil {
		t.Fatalf("create context client: %v", err)
	}

	assertNotImplemented := func(err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected route to fail when context service is not configured")
		}
		var httpErr HTTPError
		if !errors.As(err, &httpErr) {
			t.Fatalf("expected HTTPError, got %T", err)
		}
		if httpErr.StatusCode != 501 {
			t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
		}
	}

	_, err = client.ContextMemoryInventory(context.Background(), ContextMemoryInventoryRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-context-memory-inventory")
	assertNotImplemented(err)

	_, err = client.ContextMemoryCandidates(context.Background(), ContextMemoryCandidatesRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-context-memory-candidates")
	assertNotImplemented(err)

	_, err = client.ContextRetrievalDocuments(context.Background(), ContextRetrievalDocumentsRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-context-retrieval-documents")
	assertNotImplemented(err)

	_, err = client.ContextRetrievalChunks(context.Background(), ContextRetrievalChunksRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-context-retrieval-chunks")
	assertNotImplemented(err)
}

func TestTransportSubmitReplayUsesCorrelationIDIdempotency(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "replay-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "replay-token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	request := SubmitTaskRequest{
		WorkspaceID:             "ws-replay",
		RequestedByActorID:      "actor-requester",
		SubjectPrincipalActorID: "actor-subject",
		Title:                   "Replay-safe task",
		TaskClass:               "chat",
	}
	first, err := client.SubmitTask(context.Background(), request, "corr-replay")
	if err != nil {
		t.Fatalf("submit first request: %v", err)
	}
	second, err := client.SubmitTask(context.Background(), request, "corr-replay")
	if err != nil {
		t.Fatalf("submit replay request: %v", err)
	}
	if first.TaskID != second.TaskID {
		t.Fatalf("expected replay submit to return same task id, got %s and %s", first.TaskID, second.TaskID)
	}
	if first.RunID != second.RunID {
		t.Fatalf("expected replay submit to return same run id, got %s and %s", first.RunID, second.RunID)
	}

	third, err := client.SubmitTask(context.Background(), request, "corr-replay-new")
	if err != nil {
		t.Fatalf("submit with new correlation id: %v", err)
	}
	if third.TaskID == first.TaskID {
		t.Fatalf("expected new correlation id to create a new task id")
	}
}

func TestTransportServerAndClientOverUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on windows")
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pa-%d.sock", time.Now().UTC().UnixNano()))
	t.Cleanup(func() { _ = os.Remove(socketPath) })
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeUnix,
		Address:      socketPath,
		AuthToken:    "unix-token",
	})
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("expected unix socket path to exist: %v", err)
	}
	if !strings.Contains(server.Address(), socketPath) {
		t.Fatalf("expected server address to include socket path %s, got %s", socketPath, server.Address())
	}

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeUnix,
		Address:      socketPath,
		AuthToken:    "unix-token",
	})
	if err != nil {
		t.Fatalf("create unix client: %v", err)
	}

	smoke, err := client.CapabilitySmoke(context.Background(), "corr-unix")
	if err != nil {
		t.Fatalf("run capability smoke over unix socket: %v", err)
	}
	if !smoke.Healthy {
		t.Fatalf("expected healthy transport smoke response")
	}
}

func TestRealtimeClientConnectionReceiveWithTimeoutThenReceiveDoesNotPanic(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		<-r.Context().Done()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial test websocket: %v", err)
	}
	stream := &RealtimeClientConnection{conn: conn}
	t.Cleanup(func() { _ = stream.Close() })

	_, err = stream.ReceiveWithTimeout(20 * time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout/read error")
	}

	_, err = stream.Receive()
	if err == nil {
		t.Fatalf("expected cached terminal read error on repeated receive")
	}
}

func TestRealtimeClientConnectionReceiveAfterCloseErrorDoesNotPanic(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(50*time.Millisecond))
		_ = conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial test websocket: %v", err)
	}
	stream := &RealtimeClientConnection{conn: conn}
	t.Cleanup(func() { _ = stream.Close() })

	_, err = stream.Receive()
	if err == nil {
		t.Fatalf("expected close/read error")
	}

	_, err = stream.Receive()
	if err == nil {
		t.Fatalf("expected cached terminal read error on repeated receive")
	}
}

func TestTransportRegistersDaemonDomainEndpointGroups(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "domain-token",
	})

	baseURL := "http://" + server.Address()
	client := &http.Client{Timeout: 2 * time.Second}
	unknownRoutes := []string{
		"/v1",
		"/v1/unknown-group",
		"/v1/secrets",
		"/v1/providers",
		"/v1/models/unknown",
		"/v1/chat/unknown",
		"/v1/agent/unknown",
		"/v1/delegation/unknown",
		"/v1/identity/unknown",
		"/v1/comm/unknown",
		"/v1/automation/unknown",
		"/v1/inspect/unknown",
		"/v1/retention/unknown",
		"/v1/context/unknown",
		"/v1/channels/twilio",
		"/v1/connectors/cloudflared",
	}
	for _, path := range unknownRoutes {
		request, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			t.Fatalf("build request for %s: %v", path, err)
		}
		request.Header.Set("Authorization", "Bearer domain-token")
		request.Header.Set(responseHeaderCorrelationID, "corr-unknown-route")

		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("request %s: %v", path, err)
		}
		if response.StatusCode != http.StatusNotFound {
			_ = response.Body.Close()
			t.Fatalf("expected 404 for %s, got %d", path, response.StatusCode)
		}
		if got := strings.TrimSpace(response.Header.Get(responseHeaderCorrelationID)); got == "" {
			_ = response.Body.Close()
			t.Fatalf("expected %s response header for %s", responseHeaderCorrelationID, path)
		}
		if got := strings.TrimSpace(response.Header.Get(responseHeaderAPIVersion)); got != responseHeaderCurrentAPIVer {
			_ = response.Body.Close()
			t.Fatalf("expected %s=%s for %s, got %q", responseHeaderAPIVersion, responseHeaderCurrentAPIVer, path, got)
		}
		if got := strings.TrimSpace(response.Header.Get("Content-Type")); !strings.HasPrefix(got, responseContentTypeProblem) {
			_ = response.Body.Close()
			t.Fatalf("expected problem+json content type for %s, got %q", path, got)
		}

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			_ = response.Body.Close()
			t.Fatalf("decode %s response: %v", path, err)
		}
		_ = response.Body.Close()

		if got := strings.TrimSpace(fmt.Sprint(payload["path"])); got != path {
			t.Fatalf("expected path %s, got %q", path, got)
		}
		if got := strings.TrimSpace(fmt.Sprint(payload["method"])); got != http.MethodGet {
			t.Fatalf("expected method GET, got %q", got)
		}
		errorObjectRaw, ok := payload["error"]
		if !ok {
			t.Fatalf("expected typed error object for %s", path)
		}
		errorObject, ok := errorObjectRaw.(map[string]any)
		if !ok {
			t.Fatalf("expected error object map for %s, got %T", path, errorObjectRaw)
		}
		if strings.TrimSpace(fmt.Sprint(errorObject["code"])) == "" {
			t.Fatalf("expected error.code for %s", path)
		}
		if got := strings.TrimSpace(fmt.Sprint(errorObject["message"])); !strings.Contains(strings.ToLower(got), "unknown control route") {
			t.Fatalf("expected unknown-route error.message for %s, got %q", path, got)
		}
		if strings.TrimSpace(fmt.Sprint(payload["correlation_id"])) == "" {
			t.Fatalf("expected correlation_id for %s", path)
		}
		if strings.TrimSpace(fmt.Sprint(payload["type"])) == "" {
			t.Fatalf("expected RFC problem type for %s", path)
		}
		if strings.TrimSpace(fmt.Sprint(payload["title"])) == "" {
			t.Fatalf("expected RFC problem title for %s", path)
		}
		statusCode, ok := payload["status"].(float64)
		if !ok || int(statusCode) != http.StatusNotFound {
			t.Fatalf("expected RFC problem status %d for %s, got %v", http.StatusNotFound, path, payload["status"])
		}
		if strings.TrimSpace(fmt.Sprint(payload["detail"])) == "" {
			t.Fatalf("expected RFC problem detail for %s", path)
		}
		if strings.TrimSpace(fmt.Sprint(payload["instance"])) == "" {
			t.Fatalf("expected RFC problem instance for %s", path)
		}
	}
}

func TestTransportControlRateLimitReturnsTyped429AndResetsAfterWindow(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:                ListenerModeTCP,
		Address:                     "127.0.0.1:0",
		AuthToken:                   "rate-limit-token",
		ControlRateLimitWindow:      120 * time.Millisecond,
		ControlRateLimitMaxRequests: 2,
	})

	baseURL := "http://" + server.Address()
	client := &http.Client{Timeout: 2 * time.Second}

	doChatTurnRequest := func() (int, http.Header, map[string]any) {
		t.Helper()

		request, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/turn", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			t.Fatalf("build chat turn request: %v", err)
		}
		request.Header.Set("Authorization", "Bearer rate-limit-token")
		request.Header.Set("Content-Type", "application/json")

		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("chat turn request failed: %v", err)
		}
		defer response.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode chat turn response: %v", err)
		}
		return response.StatusCode, response.Header.Clone(), payload
	}

	for attempt := 1; attempt <= 2; attempt++ {
		statusCode, _, payload := doChatTurnRequest()
		if statusCode == http.StatusTooManyRequests {
			t.Fatalf("expected attempt %d to stay below rate limit, payload=%v", attempt, payload)
		}
	}

	statusCode, headers, payload := doChatTurnRequest()
	if statusCode != http.StatusTooManyRequests {
		t.Fatalf("expected third request to be rate-limited with 429, got %d payload=%v", statusCode, payload)
	}
	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	if retryAfter == "" {
		t.Fatalf("expected Retry-After header on rate-limited response")
	}
	if _, err := strconv.Atoi(retryAfter); err != nil {
		t.Fatalf("expected Retry-After header to be integer seconds, got %q", retryAfter)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in 429 payload")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != "rate_limit_exceeded" {
		t.Fatalf("expected error.code rate_limit_exceeded, got %q", got)
	}

	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected error.details in 429 payload")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error.details map, got %T", detailsRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["endpoint"])); got != controlRateLimitKeyChatTurn {
		t.Fatalf("expected details.endpoint=%s, got %q", controlRateLimitKeyChatTurn, got)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["scope_type"])); got == "" {
		t.Fatalf("expected details.scope_type to be populated")
	}
	if got := strings.TrimSpace(fmt.Sprint(details["scope_key"])); got == "" {
		t.Fatalf("expected details.scope_key to be populated")
	}
	if got := strings.TrimSpace(fmt.Sprint(details["bucket_key"])); got == "" {
		t.Fatalf("expected details.bucket_key to be populated")
	}
	if got, ok := details["limit"].(float64); !ok || int(got) != 2 {
		t.Fatalf("expected details.limit=2, got %v", details["limit"])
	}
	if got, ok := details["remaining"].(float64); !ok || int(got) != 0 {
		t.Fatalf("expected details.remaining=0, got %v", details["remaining"])
	}
	if got, ok := details["retry_after_seconds"].(float64); !ok || int(got) <= 0 {
		t.Fatalf("expected positive details.retry_after_seconds, got %v", details["retry_after_seconds"])
	}
	resetAt := strings.TrimSpace(fmt.Sprint(details["reset_at"]))
	if resetAt == "" {
		t.Fatalf("expected details.reset_at to be populated")
	}
	if _, err := time.Parse(time.RFC3339Nano, resetAt); err != nil {
		t.Fatalf("expected RFC3339 reset_at timestamp, got %q err=%v", resetAt, err)
	}

	time.Sleep(150 * time.Millisecond)
	statusCode, _, payload = doChatTurnRequest()
	if statusCode == http.StatusTooManyRequests {
		t.Fatalf("expected rate limit window reset after wait, got payload=%v", payload)
	}
}

func TestTransportControlRateLimitRejectsTrustedScopeHeaders(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:                ListenerModeTCP,
		Address:                     "127.0.0.1:0",
		AuthToken:                   "rate-limit-token",
		ControlRateLimitWindow:      200 * time.Millisecond,
		ControlRateLimitMaxRequests: 1,
	})

	baseURL := "http://" + server.Address()
	client := &http.Client{Timeout: 2 * time.Second}

	doChatTurn := func(actorID string, trustedActorID string) (int, map[string]any) {
		t.Helper()
		body := bytes.NewBufferString(fmt.Sprintf(`{"workspace_id":"ws1","requested_by_actor_id":"%s","items":[]}`, actorID))
		request, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/turn", body)
		if err != nil {
			t.Fatalf("build chat turn request: %v", err)
		}
		request.Header.Set("Authorization", "Bearer rate-limit-token")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(controlRateLimitTrustedWorkspaceHeader, "ws1")
		if strings.TrimSpace(trustedActorID) != "" {
			request.Header.Set(controlRateLimitTrustedActorHeader, trustedActorID)
		}

		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("chat turn request failed: %v", err)
		}
		defer response.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode chat turn response: %v", err)
		}
		return response.StatusCode, payload
	}

	statusCode, payload := doChatTurn("actor.alpha", "actor.alpha")
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected trusted scope headers to be rejected with 400, got %d payload=%v", statusCode, payload)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in 400 payload")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected error.details in 400 payload")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error.details map, got %T", detailsRaw)
	}
	unsupportedRaw, ok := details["unsupported_headers"]
	if !ok {
		t.Fatalf("expected unsupported_headers details in 400 payload")
	}
	unsupportedHeaders, ok := unsupportedRaw.([]any)
	if !ok || len(unsupportedHeaders) == 0 {
		t.Fatalf("expected unsupported_headers array in 400 payload, got %T %v", unsupportedRaw, unsupportedRaw)
	}
}

func TestTransportControlRateLimitIgnoresUntrustedPayloadScopeKeys(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:                ListenerModeTCP,
		Address:                     "127.0.0.1:0",
		AuthToken:                   "rate-limit-token",
		ControlRateLimitWindow:      200 * time.Millisecond,
		ControlRateLimitMaxRequests: 1,
	})

	baseURL := "http://" + server.Address()
	client := &http.Client{Timeout: 2 * time.Second}

	doChatTurn := func(actorID string) (int, map[string]any) {
		t.Helper()
		body := bytes.NewBufferString(fmt.Sprintf(`{"workspace_id":"ws1","requested_by_actor_id":"%s","items":[]}`, actorID))
		request, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/turn", body)
		if err != nil {
			t.Fatalf("build chat turn request: %v", err)
		}
		request.Header.Set("Authorization", "Bearer rate-limit-token")
		request.Header.Set("Content-Type", "application/json")
		// Intentionally omit trusted scope headers to ensure payload actor IDs are not trusted.

		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("chat turn request failed: %v", err)
		}
		defer response.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode chat turn response: %v", err)
		}
		return response.StatusCode, payload
	}

	statusCode, payload := doChatTurn("actor.alpha")
	if statusCode == http.StatusTooManyRequests {
		t.Fatalf("expected first actor.alpha request to remain below rate limit, payload=%v", payload)
	}
	statusCode, payload = doChatTurn("actor.beta")
	if statusCode != http.StatusTooManyRequests {
		t.Fatalf("expected actor.beta to share token-fingerprint bucket without trusted headers, got %d payload=%v", statusCode, payload)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in 429 payload")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected error.details in 429 payload")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error.details map, got %T", detailsRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["scope_type"])); got != controlRateLimitScopeTypeToken {
		t.Fatalf("expected scope_type token fallback, got %q", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["scope_source"])); got != "token_fingerprint" {
		t.Fatalf("expected scope_source token_fingerprint, got %q", got)
	}
}

func TestTransportControlRateLimitCoversHighRiskMutatingRoutes(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:                ListenerModeTCP,
		Address:                     "127.0.0.1:0",
		AuthToken:                   "rate-limit-token",
		ControlRateLimitWindow:      300 * time.Millisecond,
		ControlRateLimitMaxRequests: 1,
	})

	baseURL := "http://" + server.Address()
	client := &http.Client{Timeout: 2 * time.Second}

	doRequest := func(method string, path string, body string) (int, http.Header, map[string]any) {
		t.Helper()
		request, err := http.NewRequest(method, baseURL+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("build request %s %s: %v", method, path, err)
		}
		request.Header.Set("Authorization", "Bearer rate-limit-token")
		request.Header.Set("Content-Type", "application/json")

		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("execute request %s %s: %v", method, path, err)
		}
		defer response.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode response %s %s: %v", method, path, err)
		}
		return response.StatusCode, response.Header.Clone(), payload
	}

	cases := []struct {
		name        string
		method      string
		path        string
		payload     string
		endpointKey string
	}{
		{name: "chat turn", method: http.MethodPost, path: "/v1/chat/turn", payload: `{"workspace_id":"ws1","requested_by_actor_id":"actor.chat","items":[]}`, endpointKey: controlRateLimitKeyChatTurn},
		{name: "agent run", method: http.MethodPost, path: "/v1/agent/run", payload: `{"workspace_id":"ws1","request_text":"run","requested_by_actor_id":"actor.agent"}`, endpointKey: controlRateLimitKeyAgentRun},
		{name: "agent approve", method: http.MethodPost, path: "/v1/agent/approve", payload: `{"workspace_id":"ws1","approval_request_id":"apr-1","decision_by_actor_id":"actor.approver"}`, endpointKey: controlRateLimitKeyAgentApprove},
		{name: "task submit", method: http.MethodPost, path: "/v1/tasks", payload: `{"workspace_id":"ws1","requested_by_actor_id":"actor.req","subject_principal_actor_id":"actor.sub","title":"rate limited task"}`, endpointKey: controlRateLimitKeyTaskSubmit},
		{name: "task cancel", method: http.MethodPost, path: "/v1/tasks/cancel", payload: `{"workspace_id":"ws1","task_id":"task-missing","run_id":"run-missing"}`, endpointKey: controlRateLimitKeyTaskCancel},
		{name: "task retry", method: http.MethodPost, path: "/v1/tasks/retry", payload: `{"workspace_id":"ws1","task_id":"task-missing","run_id":"run-missing"}`, endpointKey: controlRateLimitKeyTaskRetry},
		{name: "task requeue", method: http.MethodPost, path: "/v1/tasks/requeue", payload: `{"workspace_id":"ws1","task_id":"task-missing","run_id":"run-missing"}`, endpointKey: controlRateLimitKeyTaskRequeue},
		{name: "automation create", method: http.MethodPost, path: "/v1/automation/create", payload: `{"workspace_id":"ws1","subject_actor_id":"actor.auto","trigger_type":"SCHEDULE","interval_seconds":60,"enabled":true}`, endpointKey: controlRateLimitKeyAutomationCreate},
		{name: "automation update", method: http.MethodPost, path: "/v1/automation/update", payload: `{"workspace_id":"ws1","trigger_id":"trigger-1","subject_actor_id":"actor.auto","enabled":true}`, endpointKey: controlRateLimitKeyAutomationUpdate},
		{name: "automation delete", method: http.MethodPost, path: "/v1/automation/delete", payload: `{"workspace_id":"ws1","trigger_id":"trigger-1"}`, endpointKey: controlRateLimitKeyAutomationDelete},
		{name: "automation run schedule", method: http.MethodPost, path: "/v1/automation/run/schedule", payload: `{"at":"2026-03-03T12:00:00Z"}`, endpointKey: controlRateLimitKeyAutomationRunSchedule},
		{name: "automation run comm event", method: http.MethodPost, path: "/v1/automation/run/comm-event", payload: `{"workspace_id":"ws1","event_id":"evt-1"}`, endpointKey: controlRateLimitKeyAutomationRunCommEvent},
		{name: "daemon lifecycle control", method: http.MethodPost, path: "/v1/daemon/lifecycle/control", payload: `{"action":"restart"}`, endpointKey: controlRateLimitKeyDaemonLifecycleControl},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, _, payload := doRequest(tc.method, tc.path, tc.payload)
			if statusCode == http.StatusTooManyRequests {
				t.Fatalf("expected first request to remain below rate limit, payload=%v", payload)
			}

			statusCode, headers, payload := doRequest(tc.method, tc.path, tc.payload)
			if statusCode != http.StatusTooManyRequests {
				t.Fatalf("expected second request to be rate-limited with 429, got %d payload=%v", statusCode, payload)
			}
			if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter == "" {
				t.Fatalf("expected Retry-After header on rate-limited response")
			}

			errorObjectRaw, ok := payload["error"]
			if !ok {
				t.Fatalf("expected typed error object in 429 payload")
			}
			errorObject, ok := errorObjectRaw.(map[string]any)
			if !ok {
				t.Fatalf("expected error object map, got %T", errorObjectRaw)
			}
			if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != "rate_limit_exceeded" {
				t.Fatalf("expected rate_limit_exceeded code, got %q", got)
			}

			detailsRaw, ok := errorObject["details"]
			if !ok {
				t.Fatalf("expected error.details in 429 payload")
			}
			details, ok := detailsRaw.(map[string]any)
			if !ok {
				t.Fatalf("expected error.details map, got %T", detailsRaw)
			}
			if got := strings.TrimSpace(fmt.Sprint(details["endpoint"])); got != tc.endpointKey {
				t.Fatalf("expected details.endpoint=%s, got %q", tc.endpointKey, got)
			}
			if got := strings.TrimSpace(fmt.Sprint(details["scope_type"])); got == "" {
				t.Fatalf("expected non-empty details.scope_type, got %q", got)
			}
			if got := strings.TrimSpace(fmt.Sprint(details["scope_key"])); got == "" {
				t.Fatalf("expected non-empty details.scope_key, got %q", got)
			}
			if got := strings.TrimSpace(fmt.Sprint(details["bucket_key"])); got == "" {
				t.Fatalf("expected non-empty details.bucket_key, got %q", got)
			}
			if got, ok := details["retry_after_seconds"].(float64); !ok || int(got) <= 0 {
				t.Fatalf("expected positive details.retry_after_seconds, got %v", details["retry_after_seconds"])
			}
		})
	}
}

func TestTransportSuccessResponsesIncludeCorrelationAndAPIVersionHeaders(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "header-token",
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+server.Address()+"/v1/capabilities/smoke", nil)
	if err != nil {
		t.Fatalf("build smoke request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer header-token")
	request.Header.Set(responseHeaderCorrelationID, "corr-success-header")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute smoke request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if got := strings.TrimSpace(response.Header.Get(responseHeaderCorrelationID)); got != "corr-success-header" {
		t.Fatalf("expected %s header corr-success-header, got %q", responseHeaderCorrelationID, got)
	}
	if got := strings.TrimSpace(response.Header.Get(responseHeaderAPIVersion)); got != responseHeaderCurrentAPIVer {
		t.Fatalf("expected %s=%s, got %q", responseHeaderAPIVersion, responseHeaderCurrentAPIVer, got)
	}
	if got := strings.TrimSpace(response.Header.Get("Content-Type")); !strings.HasPrefix(got, responseContentTypeJSON) {
		t.Fatalf("expected application/json content type, got %q", got)
	}
}

func TestTransportMetaCapabilitiesRoute(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "meta-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "meta-token",
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.DaemonCapabilities(context.Background(), "corr-meta-capabilities")
	if err != nil {
		t.Fatalf("meta capabilities request failed: %v", err)
	}

	if strings.TrimSpace(response.APIVersion) != responseHeaderCurrentAPIVer {
		t.Fatalf("expected api_version=%s, got %q", responseHeaderCurrentAPIVer, response.APIVersion)
	}
	if strings.TrimSpace(response.CorrelationID) != "corr-meta-capabilities" {
		t.Fatalf("expected correlation id corr-meta-capabilities, got %q", response.CorrelationID)
	}
	if len(response.RouteGroups) == 0 {
		t.Fatalf("expected non-empty route groups")
	}
	if len(response.RealtimeEventTypes) == 0 {
		t.Fatalf("expected non-empty realtime event types")
	}
	if strings.TrimSpace(response.RealtimeBackpressure.OverflowPolicy) == "" {
		t.Fatalf("expected realtime backpressure overflow policy")
	}
	if response.RealtimeBackpressure.SlowSubscriberConsecutiveLimit <= 0 {
		t.Fatalf("expected positive realtime backpressure drop limit, got %d", response.RealtimeBackpressure.SlowSubscriberConsecutiveLimit)
	}
	if response.RealtimeDiagnostics.ActiveSubscribers < 0 {
		t.Fatalf("expected non-negative realtime diagnostics subscriber count, got %d", response.RealtimeDiagnostics.ActiveSubscribers)
	}
	if len(response.ClientSignalTypes) == 0 {
		t.Fatalf("expected non-empty client signal types")
	}
	if len(response.ProtocolModes) == 0 {
		t.Fatalf("expected non-empty protocol modes")
	}
	if len(response.TransportListenerModes) == 0 {
		t.Fatalf("expected non-empty transport listener modes")
	}
}

func TestTransportSecretReferenceEndpointsReturnMetadataOnly(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:     ListenerModeTCP,
		Address:          "127.0.0.1:0",
		AuthToken:        "secret-token",
		SecretReferences: NewInMemorySecretReferenceService(),
	})

	baseURL := "http://" + server.Address()
	httpClient := &http.Client{Timeout: 2 * time.Second}

	doRequest := func(method string, path string, body []byte) map[string]any {
		t.Helper()

		var reader *bytes.Reader
		if body == nil {
			reader = bytes.NewReader([]byte{})
		} else {
			reader = bytes.NewReader(body)
		}
		request, err := http.NewRequest(method, baseURL+path, reader)
		if err != nil {
			t.Fatalf("build %s %s request: %v", method, path, err)
		}
		request.Header.Set("Authorization", "Bearer secret-token")
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}

		response, err := httpClient.Do(request)
		if err != nil {
			t.Fatalf("%s %s request failed: %v", method, path, err)
		}
		defer response.Body.Close()

		if response.StatusCode >= 400 {
			var payload map[string]any
			_ = json.NewDecoder(response.Body).Decode(&payload)
			t.Fatalf("%s %s expected success, got status=%d payload=%v", method, path, response.StatusCode, payload)
		}

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode %s %s response: %v", method, path, err)
		}
		return payload
	}

	postPayload := []byte(`{
		"workspace_id":"ws1",
		"name":"OPENAI_API_KEY",
		"backend":"memory",
		"service":"personal-agent.ws1",
		"account":"OPENAI_API_KEY"
	}`)
	created := doRequest(http.MethodPost, "/v1/secrets/refs", postPayload)
	loaded := doRequest(http.MethodGet, "/v1/secrets/refs/ws1/OPENAI_API_KEY", nil)
	deleted := doRequest(http.MethodDelete, "/v1/secrets/refs/ws1/OPENAI_API_KEY", nil)

	for label, payload := range map[string]map[string]any{
		"created": created,
		"loaded":  loaded,
		"deleted": deleted,
	} {
		if _, exists := payload["value"]; exists {
			t.Fatalf("did not expect plaintext value field in %s payload: %v", label, payload)
		}
		if _, exists := payload["value_masked"]; exists {
			t.Fatalf("did not expect masked value field in %s payload: %v", label, payload)
		}

		record, ok := payload["reference"].(map[string]any)
		if !ok {
			t.Fatalf("expected reference record in %s payload, got %v", label, payload["reference"])
		}
		if record["workspace_id"] != "ws1" || record["name"] != "OPENAI_API_KEY" {
			t.Fatalf("unexpected reference values in %s payload: %v", label, record)
		}
		if record["service"] != "personal-agent.ws1" || record["account"] != "OPENAI_API_KEY" {
			t.Fatalf("unexpected service/account values in %s payload: %v", label, record)
		}
	}

	if deleted["deleted"] != true {
		t.Fatalf("expected deleted=true on delete payload, got %v", deleted["deleted"])
	}
}

func assertTaskActionAvailabilityPayload(
	t *testing.T,
	payload map[string]any,
	wantCanCancel bool,
	wantCanRetry bool,
	wantCanRequeue bool,
) {
	t.Helper()
	actionsRaw, ok := payload["actions"]
	if !ok {
		t.Fatalf("expected actions object in payload: %+v", payload)
	}
	actions, ok := actionsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected actions object map, got %T", actionsRaw)
	}

	canCancelRaw, cancelExists := actions["can_cancel"]
	canRetryRaw, retryExists := actions["can_retry"]
	canRequeueRaw, requeueExists := actions["can_requeue"]
	if !cancelExists || !retryExists || !requeueExists {
		t.Fatalf("expected actions can_cancel/can_retry/can_requeue keys, got %+v", actions)
	}

	canCancel, ok := canCancelRaw.(bool)
	if !ok {
		t.Fatalf("expected actions.can_cancel bool, got %T", canCancelRaw)
	}
	canRetry, ok := canRetryRaw.(bool)
	if !ok {
		t.Fatalf("expected actions.can_retry bool, got %T", canRetryRaw)
	}
	canRequeue, ok := canRequeueRaw.(bool)
	if !ok {
		t.Fatalf("expected actions.can_requeue bool, got %T", canRequeueRaw)
	}
	if canCancel != wantCanCancel || canRetry != wantCanRetry || canRequeue != wantCanRequeue {
		t.Fatalf(
			"unexpected actions values can_cancel=%v can_retry=%v can_requeue=%v want=%v/%v/%v",
			canCancel,
			canRetry,
			canRequeue,
			wantCanCancel,
			wantCanRetry,
			wantCanRequeue,
		)
	}
}

type configFieldDescriptorContractExpectation struct {
	Required    bool
	EnumOptions []string
	Secret      bool
	WriteOnly   bool
	HelpText    string
}

func assertConfigFieldDescriptorPayload(
	t *testing.T,
	descriptor map[string]any,
	expectation configFieldDescriptorContractExpectation,
) {
	t.Helper()

	requiredRaw, requiredExists := descriptor["required"]
	enumRaw, enumExists := descriptor["enum_options"]
	secretRaw, secretExists := descriptor["secret"]
	writeOnlyRaw, writeOnlyExists := descriptor["write_only"]
	helpTextRaw, helpTextExists := descriptor["help_text"]
	if !requiredExists || !enumExists || !secretExists || !writeOnlyExists || !helpTextExists {
		t.Fatalf(
			"expected descriptor keys required/enum_options/secret/write_only/help_text, got %+v",
			descriptor,
		)
	}

	required, ok := requiredRaw.(bool)
	if !ok {
		t.Fatalf("expected descriptor.required bool, got %T", requiredRaw)
	}
	if required != expectation.Required {
		t.Fatalf("unexpected descriptor.required=%v want=%v", required, expectation.Required)
	}

	enumOptionsRaw, ok := enumRaw.([]any)
	if !ok {
		t.Fatalf("expected descriptor.enum_options array, got %T (%+v)", enumRaw, enumRaw)
	}
	enumOptions := make([]string, 0, len(enumOptionsRaw))
	for _, raw := range enumOptionsRaw {
		enumOptions = append(enumOptions, strings.TrimSpace(fmt.Sprint(raw)))
	}
	if len(enumOptions) != len(expectation.EnumOptions) {
		t.Fatalf("unexpected descriptor.enum_options length=%d want=%d (%+v)", len(enumOptions), len(expectation.EnumOptions), enumOptions)
	}
	for index := range expectation.EnumOptions {
		if enumOptions[index] != expectation.EnumOptions[index] {
			t.Fatalf(
				"unexpected descriptor.enum_options[%d]=%q want=%q",
				index,
				enumOptions[index],
				expectation.EnumOptions[index],
			)
		}
	}

	secret, ok := secretRaw.(bool)
	if !ok {
		t.Fatalf("expected descriptor.secret bool, got %T", secretRaw)
	}
	if secret != expectation.Secret {
		t.Fatalf("unexpected descriptor.secret=%v want=%v", secret, expectation.Secret)
	}

	writeOnly, ok := writeOnlyRaw.(bool)
	if !ok {
		t.Fatalf("expected descriptor.write_only bool, got %T", writeOnlyRaw)
	}
	if writeOnly != expectation.WriteOnly {
		t.Fatalf("unexpected descriptor.write_only=%v want=%v", writeOnly, expectation.WriteOnly)
	}

	helpText, ok := helpTextRaw.(string)
	if !ok {
		t.Fatalf("expected descriptor.help_text string, got %T", helpTextRaw)
	}
	if helpText != expectation.HelpText {
		t.Fatalf("unexpected descriptor.help_text=%q want=%q", helpText, expectation.HelpText)
	}
}
