package daemonruntime

import (
	"context"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestUnifiedTurnServiceExplainChatTurnReturnsRouteToolsAndPolicyDecisions(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{
		explainResponse: transport.ModelRouteExplainResponse{
			WorkspaceID:      "ws1",
			TaskClass:        "chat",
			PrincipalActorID: "actor.user",
			SelectedProvider: "openai",
			SelectedModelKey: "gpt-4.1-mini",
			SelectedSource:   "task_policy",
			Summary:          "task policy selected openai/gpt-4.1-mini",
			Explanations: []string{
				"task-class policy selected openai/gpt-4.1-mini",
			},
		},
	}
	ui := &unifiedUIStatusStub{
		connectorResponse: transport.ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail.send"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "finder",
					Capabilities:    []string{"finder.delete"},
					ActionReadiness: "ready",
				},
			},
		},
	}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, ui, &unifiedDelegationStub{
		checkResponse: transport.DelegationCheckResponse{Allowed: true},
	})

	response, err := service.ExplainChatTurn(context.Background(), transport.ChatTurnExplainRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.user",
		ActingAsActorID:    "actor.user",
		Channel: transport.ChatTurnChannelContext{
			ChannelID: "voice",
		},
	})
	if err != nil {
		t.Fatalf("explain chat turn: %v", err)
	}

	if strings.TrimSpace(response.ContractVersion) != transport.ChatTurnExplainContractVersionV1 {
		t.Fatalf("expected explain contract version %q, got %q", transport.ChatTurnExplainContractVersionV1, response.ContractVersion)
	}
	if strings.TrimSpace(response.SelectedRoute.SelectedProvider) != "openai" {
		t.Fatalf("expected selected provider openai, got %q", response.SelectedRoute.SelectedProvider)
	}
	if strings.TrimSpace(response.SelectedRoute.SelectedModelKey) != "gpt-4.1-mini" {
		t.Fatalf("expected selected model key gpt-4.1-mini, got %q", response.SelectedRoute.SelectedModelKey)
	}
	if len(response.ToolCatalog) != 2 {
		t.Fatalf("expected 2 tool catalog entries, got %d (%+v)", len(response.ToolCatalog), response.ToolCatalog)
	}

	decisionByTool := map[string]transport.ChatTurnToolPolicyDecision{}
	for _, decision := range response.PolicyDecisions {
		decisionByTool[strings.ToLower(strings.TrimSpace(decision.ToolName))] = decision
	}
	if got := strings.TrimSpace(decisionByTool["mail_send"].Decision); got != "ALLOW" {
		t.Fatalf("expected mail_send policy ALLOW, got %q", got)
	}
	if got := strings.TrimSpace(decisionByTool["finder_delete"].Decision); got != "REQUIRE_APPROVAL" {
		t.Fatalf("expected finder_delete policy REQUIRE_APPROVAL on voice channel, got %q", got)
	}
}
