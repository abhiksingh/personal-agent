package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

type unifiedTurnCrossChannelAcceptanceScenario struct {
	name                       string
	correlationID              string
	channel                    transport.ChatTurnChannelContext
	connectorCards             []transport.ConnectorStatusCard
	modelResponses             []transport.ChatTurnResponse
	agentResponses             []transport.AgentRunResponse
	agentErrs                  []error
	expectedPlannerToolNames   []string
	expectedAgentCalls         int
	expectedOrigin             string
	expectedToolResultStatus   string
	expectedToolResultError    string
	expectedRemediationCode    string
	expectedApprovalRequestID  string
	expectHistoryApprovalEntry bool
}

func TestUnifiedTurnServiceCrossChannelAcceptanceMatrix(t *testing.T) {
	scenarios := []unifiedTurnCrossChannelAcceptanceScenario{
		{
			name:          "app channel readiness edge emits unsupported_tool and preserves canonical lifecycle",
			correlationID: "corr-cross-channel-app-readiness",
			channel: transport.ChatTurnChannelContext{
				ChannelID:   "app",
				ConnectorID: "builtin.app",
			},
			connectorCards: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail_send"},
					ActionReadiness: "blocked",
				},
			},
			modelResponses: []transport.ChatTurnResponse{
				plannerToolCallResponse(`{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"status update"}}`),
				plannerAssistantResponse("acknowledged"),
			},
			expectedPlannerToolNames: []string{},
			expectedAgentCalls:       0,
			expectedToolResultStatus: "denied",
			expectedToolResultError:  "unsupported_tool",
		},
		{
			name:          "message channel failure edge maps to tool_execution_failed with app origin",
			correlationID: "corr-cross-channel-message-failure",
			channel: transport.ChatTurnChannelContext{
				ChannelID:   "message",
				ConnectorID: "twilio",
				ThreadID:    "thread-cross-channel-message",
			},
			connectorCards: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "twilio",
					Capabilities:    []string{"channel.twilio.sms.send"},
					ActionReadiness: "ready",
				},
			},
			modelResponses: []transport.ChatTurnResponse{
				plannerToolCallResponse(`{"type":"tool_call","tool_name":"message_send","arguments":{"channel":"sms","recipient":"+15555550000","body":"send update"}}`),
				plannerAssistantResponse("unable to complete"),
			},
			agentErrs:                []error{errors.New("twilio gateway unavailable")},
			expectedPlannerToolNames: []string{"message_send"},
			expectedAgentCalls:       1,
			expectedOrigin:           "app",
			expectedToolResultStatus: "failed",
			expectedToolResultError:  "tool_execution_failed",
			expectedRemediationCode:  "tool_execution_failure",
		},
		{
			name:          "voice channel approval edge emits approval_request and voice origin",
			correlationID: "corr-cross-channel-voice-approval",
			channel: transport.ChatTurnChannelContext{
				ChannelID:   "voice",
				ConnectorID: "twilio",
			},
			connectorCards: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "finder",
					Capabilities:    []string{"finder_delete"},
					ActionReadiness: "ready",
				},
			},
			modelResponses: []transport.ChatTurnResponse{
				plannerToolCallResponse(`{"type":"tool_call","tool_name":"finder_delete","arguments":{"path":"/tmp/cross-channel-voice.txt"}}`),
				plannerAssistantResponse("waiting"),
			},
			agentResponses: []transport.AgentRunResponse{
				{
					Workflow:          "finder_delete",
					TaskID:            "task-cross-channel-voice",
					RunID:             "run-cross-channel-voice",
					TaskState:         "awaiting_approval",
					RunState:          "awaiting_approval",
					ApprovalRequired:  true,
					ApprovalRequestID: "approval-cross-channel-voice",
				},
			},
			expectedPlannerToolNames:   []string{"finder_delete"},
			expectedAgentCalls:         1,
			expectedOrigin:             "voice",
			expectedToolResultStatus:   "awaiting_approval",
			expectedApprovalRequestID:  "approval-cross-channel-voice",
			expectHistoryApprovalEntry: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			container := newUnifiedTurnTestContainer(t)
			model := &unifiedModelChatStub{responses: scenario.modelResponses}
			agent := &unifiedAgentStub{responses: scenario.agentResponses, errs: scenario.agentErrs}
			ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
				WorkspaceID: "ws1",
				Connectors:  scenario.connectorCards,
			}}
			service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

			response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
				WorkspaceID:        "ws1",
				TaskClass:          "chat",
				RequestedByActorID: "actor.requester",
				SubjectActorID:     "actor.requester",
				ActingAsActorID:    "actor.requester",
				Channel:            scenario.channel,
				Items: []transport.ChatTurnItem{{
					Type:    "user_message",
					Role:    "user",
					Status:  "completed",
					Content: "complete this action",
				}},
			}, scenario.correlationID, nil)
			if err != nil {
				t.Fatalf("chat turn: %v", err)
			}

			if got := strings.TrimSpace(response.Channel.ChannelID); got != normalizeTurnChannelID(scenario.channel.ChannelID) {
				t.Fatalf("expected normalized response channel %q, got %q", normalizeTurnChannelID(scenario.channel.ChannelID), got)
			}
			if len(model.requests) != len(scenario.modelResponses) {
				t.Fatalf("expected %d model calls, got %d", len(scenario.modelResponses), len(model.requests))
			}
			assertPlannerToolCatalogNames(t, model.requests[0].ToolCatalog, scenario.expectedPlannerToolNames)

			if len(agent.requests) != scenario.expectedAgentCalls {
				t.Fatalf("expected %d agent requests, got %d", scenario.expectedAgentCalls, len(agent.requests))
			}
			if scenario.expectedOrigin != "" {
				if len(agent.requests) == 0 {
					t.Fatalf("expected at least one agent request to assert origin %q", scenario.expectedOrigin)
				}
				if got := strings.TrimSpace(agent.requests[0].Origin); got != strings.TrimSpace(scenario.expectedOrigin) {
					t.Fatalf("expected first agent origin %q, got %q", scenario.expectedOrigin, got)
				}
			}

			assertCanonicalLifecycleShape(t, response.Items)
			toolResult := mustFirstTurnItemByType(t, response.Items, "tool_result")
			if got := strings.ToLower(strings.TrimSpace(toolResult.Status)); got != strings.ToLower(strings.TrimSpace(scenario.expectedToolResultStatus)) {
				t.Fatalf("expected tool_result status %q, got %q (%+v)", scenario.expectedToolResultStatus, got, toolResult)
			}
			if got := strings.ToLower(strings.TrimSpace(toolResult.ErrorCode)); got != strings.ToLower(strings.TrimSpace(scenario.expectedToolResultError)) {
				t.Fatalf("expected tool_result error_code %q, got %q (%+v)", scenario.expectedToolResultError, got, toolResult)
			}
			if strings.TrimSpace(scenario.expectedRemediationCode) != "" {
				if got := strings.TrimSpace(fmt.Sprintf("%v", toolResult.Metadata.AsMap()["code"])); got != strings.TrimSpace(scenario.expectedRemediationCode) {
					t.Fatalf("expected remediation code %q, got %q (metadata=%+v)", scenario.expectedRemediationCode, got, toolResult.Metadata)
				}
			}
			if strings.TrimSpace(scenario.expectedApprovalRequestID) != "" {
				if got := strings.TrimSpace(toolResult.ApprovalRequestID); got != strings.TrimSpace(scenario.expectedApprovalRequestID) {
					t.Fatalf("expected tool_result approval_request_id %q, got %q", scenario.expectedApprovalRequestID, got)
				}
				approvalItem := mustApprovalRequestItem(t, response.Items, scenario.expectedApprovalRequestID)
				if strings.ToLower(strings.TrimSpace(approvalItem.Status)) != "awaiting_approval" {
					t.Fatalf("expected approval_request awaiting_approval status, got %+v", approvalItem)
				}
			}

			history, err := service.QueryChatTurnHistory(context.Background(), transport.ChatTurnHistoryRequest{
				WorkspaceID:   "ws1",
				CorrelationID: scenario.correlationID,
				Limit:         20,
			})
			if err != nil {
				t.Fatalf("query chat history: %v", err)
			}
			if !historyContainsType(history.Items, "user_message") ||
				!historyContainsType(history.Items, "tool_call") ||
				!historyContainsType(history.Items, "tool_result") ||
				!historyContainsType(history.Items, "assistant_message") {
				t.Fatalf("expected persisted canonical history chain, got %+v", history.Items)
			}
			if scenario.expectHistoryApprovalEntry && !historyContainsType(history.Items, "approval_request") {
				t.Fatalf("expected approval_request in persisted history, got %+v", history.Items)
			}
		})
	}
}

func plannerToolCallResponse(content string) transport.ChatTurnResponse {
	return transport.ChatTurnResponse{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Provider:    "openai",
		ModelKey:    "gpt-4.1-mini",
		Items: []transport.ChatTurnItem{{
			Type:    "assistant_message",
			Role:    "assistant",
			Status:  "completed",
			Content: content,
		}},
	}
}

func plannerAssistantResponse(content string) transport.ChatTurnResponse {
	return transport.ChatTurnResponse{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Provider:    "openai",
		ModelKey:    "gpt-4.1-mini",
		Items: []transport.ChatTurnItem{{
			Type:    "assistant_message",
			Role:    "assistant",
			Status:  "completed",
			Content: content,
		}},
	}
}

func assertPlannerToolCatalogNames(t *testing.T, catalog []transport.ChatTurnToolCatalogEntry, expected []string) {
	t.Helper()
	if len(catalog) != len(expected) {
		t.Fatalf("expected planner tool catalog size %d, got %d (%+v)", len(expected), len(catalog), catalog)
	}
	for _, want := range expected {
		found := false
		for _, entry := range catalog {
			if strings.EqualFold(strings.TrimSpace(entry.Name), strings.TrimSpace(want)) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected planner tool %q in catalog %+v", want, catalog)
		}
	}
}

func assertCanonicalLifecycleShape(t *testing.T, items []transport.ChatTurnItem) {
	t.Helper()
	toolCallIndex := firstTurnItemIndexByType(items, "tool_call")
	toolResultIndex := firstTurnItemIndexByType(items, "tool_result")
	assistantIndex := firstTurnItemIndexByType(items, "assistant_message")
	if toolCallIndex < 0 || toolResultIndex < 0 || assistantIndex < 0 {
		t.Fatalf("expected tool_call/tool_result/assistant_message lifecycle items, got %+v", items)
	}
	if !(toolCallIndex < toolResultIndex && toolResultIndex < assistantIndex) {
		t.Fatalf("expected lifecycle order tool_call -> tool_result -> assistant_message, got %+v", items)
	}
	assistant := items[assistantIndex]
	if strings.ToLower(strings.TrimSpace(assistant.Status)) != "completed" {
		t.Fatalf("expected final assistant_message completed, got %+v", assistant)
	}
	if strings.TrimSpace(assistant.Content) == "" {
		t.Fatalf("expected non-empty final assistant_message content")
	}
}

func firstTurnItemIndexByType(items []transport.ChatTurnItem, itemType string) int {
	target := strings.ToLower(strings.TrimSpace(itemType))
	for index, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Type)) == target {
			return index
		}
	}
	return -1
}

func mustFirstTurnItemByType(t *testing.T, items []transport.ChatTurnItem, itemType string) transport.ChatTurnItem {
	t.Helper()
	target := strings.ToLower(strings.TrimSpace(itemType))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Type)) == target {
			return item
		}
	}
	t.Fatalf("expected turn item type %q in %+v", itemType, items)
	return transport.ChatTurnItem{}
}

func mustApprovalRequestItem(t *testing.T, items []transport.ChatTurnItem, approvalRequestID string) transport.ChatTurnItem {
	t.Helper()
	targetID := strings.TrimSpace(approvalRequestID)
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "approval_request" {
			continue
		}
		if strings.TrimSpace(item.ApprovalRequestID) == targetID {
			return item
		}
	}
	t.Fatalf("expected approval_request item with id %q in %+v", approvalRequestID, items)
	return transport.ChatTurnItem{}
}
