package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"
)

type unifiedModelChatStub struct {
	responses       []transport.ChatTurnResponse
	errs            []error
	requests        []transport.ChatTurnRequest
	blockOnCall     int
	explainResponse transport.ModelRouteExplainResponse
	explainErr      error
	explainRequests []transport.ModelRouteExplainRequest
}

func (s *unifiedModelChatStub) ChatTurn(ctx context.Context, request transport.ChatTurnRequest, correlationID string, onToken func(delta string)) (transport.ChatTurnResponse, error) {
	s.requests = append(s.requests, request)
	call := len(s.requests)
	if s.blockOnCall == call {
		<-ctx.Done()
		return transport.ChatTurnResponse{}, ctx.Err()
	}
	if call <= len(s.errs) && s.errs[call-1] != nil {
		return transport.ChatTurnResponse{}, s.errs[call-1]
	}
	if call > len(s.responses) {
		return transport.ChatTurnResponse{}, errors.New("unexpected model call")
	}
	response := s.responses[call-1]
	if strings.TrimSpace(response.CorrelationID) == "" {
		response.CorrelationID = strings.TrimSpace(correlationID)
	}
	if onToken != nil && call == len(s.responses) {
		onToken("ok")
	}
	return response, nil
}

func (s *unifiedModelChatStub) ExplainModelRoute(_ context.Context, request transport.ModelRouteExplainRequest) (transport.ModelRouteExplainResponse, error) {
	s.explainRequests = append(s.explainRequests, request)
	if s.explainErr != nil {
		return transport.ModelRouteExplainResponse{}, s.explainErr
	}
	if strings.TrimSpace(s.explainResponse.WorkspaceID) == "" {
		s.explainResponse.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	}
	if strings.TrimSpace(s.explainResponse.TaskClass) == "" {
		s.explainResponse.TaskClass = strings.TrimSpace(request.TaskClass)
	}
	if strings.TrimSpace(s.explainResponse.PrincipalActorID) == "" {
		s.explainResponse.PrincipalActorID = strings.TrimSpace(request.PrincipalActorID)
	}
	return s.explainResponse, nil
}

type unifiedAgentStub struct {
	responses []transport.AgentRunResponse
	errs      []error
	requests  []transport.AgentRunRequest
}

func (s *unifiedAgentStub) RunAgent(_ context.Context, request transport.AgentRunRequest) (transport.AgentRunResponse, error) {
	s.requests = append(s.requests, request)
	call := len(s.requests)
	if call <= len(s.errs) && s.errs[call-1] != nil {
		return transport.AgentRunResponse{}, s.errs[call-1]
	}
	if call > len(s.responses) {
		return transport.AgentRunResponse{}, nil
	}
	return s.responses[call-1], nil
}

func (s *unifiedAgentStub) ApproveAgent(_ context.Context, _ transport.AgentApproveRequest) (transport.AgentRunResponse, error) {
	return transport.AgentRunResponse{}, nil
}

type unifiedUIStatusStub struct {
	channelResponse   transport.ChannelStatusResponse
	connectorResponse transport.ConnectorStatusResponse
}

func (s *unifiedUIStatusStub) ListChannelConnectorMappings(context.Context, transport.ChannelConnectorMappingListRequest) (transport.ChannelConnectorMappingListResponse, error) {
	return transport.ChannelConnectorMappingListResponse{}, nil
}

func (s *unifiedUIStatusStub) UpsertChannelConnectorMapping(context.Context, transport.ChannelConnectorMappingUpsertRequest) (transport.ChannelConnectorMappingUpsertResponse, error) {
	return transport.ChannelConnectorMappingUpsertResponse{}, nil
}

func (s *unifiedUIStatusStub) ListChannelStatus(_ context.Context, _ transport.ChannelStatusRequest) (transport.ChannelStatusResponse, error) {
	return s.channelResponse, nil
}

func (s *unifiedUIStatusStub) ListConnectorStatus(_ context.Context, _ transport.ConnectorStatusRequest) (transport.ConnectorStatusResponse, error) {
	return s.connectorResponse, nil
}

func (s *unifiedUIStatusStub) ListChannelDiagnostics(context.Context, transport.ChannelDiagnosticsRequest) (transport.ChannelDiagnosticsResponse, error) {
	return transport.ChannelDiagnosticsResponse{}, nil
}

func (s *unifiedUIStatusStub) ListConnectorDiagnostics(context.Context, transport.ConnectorDiagnosticsRequest) (transport.ConnectorDiagnosticsResponse, error) {
	return transport.ConnectorDiagnosticsResponse{}, nil
}

func (s *unifiedUIStatusStub) RequestConnectorPermission(context.Context, transport.ConnectorPermissionRequest) (transport.ConnectorPermissionResponse, error) {
	return transport.ConnectorPermissionResponse{}, nil
}

func (s *unifiedUIStatusStub) UpsertChannelConfig(context.Context, transport.ChannelConfigUpsertRequest) (transport.ChannelConfigUpsertResponse, error) {
	return transport.ChannelConfigUpsertResponse{}, nil
}

func (s *unifiedUIStatusStub) UpsertConnectorConfig(context.Context, transport.ConnectorConfigUpsertRequest) (transport.ConnectorConfigUpsertResponse, error) {
	return transport.ConnectorConfigUpsertResponse{}, nil
}

func (s *unifiedUIStatusStub) TestChannelOperation(context.Context, transport.ChannelTestOperationRequest) (transport.ChannelTestOperationResponse, error) {
	return transport.ChannelTestOperationResponse{}, nil
}

func (s *unifiedUIStatusStub) TestConnectorOperation(context.Context, transport.ConnectorTestOperationRequest) (transport.ConnectorTestOperationResponse, error) {
	return transport.ConnectorTestOperationResponse{}, nil
}

type unifiedDelegationStub struct {
	checkResponse transport.DelegationCheckResponse
	checkErr      error
	checkRequests []transport.DelegationCheckRequest
}

func (s *unifiedDelegationStub) GrantDelegation(context.Context, transport.DelegationGrantRequest) (transport.DelegationRuleRecord, error) {
	return transport.DelegationRuleRecord{}, nil
}

func (s *unifiedDelegationStub) ListDelegations(context.Context, transport.DelegationListRequest) (transport.DelegationListResponse, error) {
	return transport.DelegationListResponse{}, nil
}

func (s *unifiedDelegationStub) RevokeDelegation(context.Context, transport.DelegationRevokeRequest) (transport.DelegationRevokeResponse, error) {
	return transport.DelegationRevokeResponse{}, nil
}

func (s *unifiedDelegationStub) CheckDelegation(_ context.Context, request transport.DelegationCheckRequest) (transport.DelegationCheckResponse, error) {
	s.checkRequests = append(s.checkRequests, request)
	if s.checkErr != nil {
		return transport.DelegationCheckResponse{}, s.checkErr
	}
	return s.checkResponse, nil
}

func (s *unifiedDelegationStub) UpsertCapabilityGrant(context.Context, transport.CapabilityGrantUpsertRequest) (transport.CapabilityGrantRecord, error) {
	return transport.CapabilityGrantRecord{}, nil
}

func (s *unifiedDelegationStub) ListCapabilityGrants(context.Context, transport.CapabilityGrantListRequest) (transport.CapabilityGrantListResponse, error) {
	return transport.CapabilityGrantListResponse{}, nil
}

func newUnifiedTurnTestContainer(t *testing.T) *ServiceContainer {
	t.Helper()
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new secret manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})
	return container
}

func newUnifiedTurnTestService(t *testing.T, container *ServiceContainer, model transport.ChatService, agent transport.AgentService, ui transport.UIStatusService, delegation transport.DelegationService) *UnifiedTurnService {
	t.Helper()
	service, err := NewUnifiedTurnService(container, model, agent, ui, delegation)
	if err != nil {
		t.Fatalf("new unified turn service: %v", err)
	}
	return service
}

func seedWorkspacePrincipal(t *testing.T, db *sql.DB, workspaceID string, actorID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, actorID, workspaceID, actorID, now, now); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(workspace_id, actor_id) DO UPDATE SET updated_at = excluded.updated_at
	`, "principal."+actorID, workspaceID, actorID, now, now); err != nil {
		t.Fatalf("seed workspace principal: %v", err)
	}
}

func toolNamesByID(tools []modelToolDefinition) map[string]struct{} {
	names := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		normalized := strings.TrimSpace(tool.Name)
		if normalized == "" {
			continue
		}
		names[normalized] = struct{}{}
	}
	return names
}

func findToolByName(tools []modelToolDefinition, name string) (modelToolDefinition, bool) {
	target := strings.TrimSpace(name)
	for _, tool := range tools {
		if strings.EqualFold(strings.TrimSpace(tool.Name), target) {
			return tool, true
		}
	}
	return modelToolDefinition{}, false
}

func historyContainsType(items []transport.ChatTurnHistoryRecord, itemType string) bool {
	target := strings.ToLower(strings.TrimSpace(itemType))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Item.Type)) == target {
			return true
		}
	}
	return false
}

func TestUnifiedTurnServiceResolveAvailableToolsSupportsDotCapabilityKeys(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	service := newUnifiedTurnTestService(t, container, &unifiedModelChatStub{}, &unifiedAgentStub{}, &unifiedUIStatusStub{
		connectorResponse: transport.ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail.send"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "calendar",
					Capabilities:    []string{"calendar.create", "calendar.cancel"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "browser",
					Capabilities:    []string{"browser.open"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "finder",
					Capabilities:    []string{"finder.delete"},
					ActionReadiness: "ready",
				},
			},
		},
	}, nil)

	tools, err := service.resolveAvailableTools(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("resolve available tools: %v", err)
	}
	toolNames := toolNamesByID(tools)
	for _, expected := range []string{"mail_send", "calendar_create", "calendar_cancel", "browser_open", "finder_delete"} {
		if _, ok := toolNames[expected]; !ok {
			t.Fatalf("expected resolved tool %s, got %+v", expected, tools)
		}
	}
}

func TestUnifiedTurnServiceResolveAvailableToolsSkipsBlockedConnectorCapabilities(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	service := newUnifiedTurnTestService(t, container, &unifiedModelChatStub{}, &unifiedAgentStub{}, &unifiedUIStatusStub{
		connectorResponse: transport.ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail_send"},
					ActionReadiness: "blocked",
				},
				{
					ConnectorID:     "twilio",
					Capabilities:    []string{"channel.twilio.sms.send"},
					ActionReadiness: "ready",
				},
			},
		},
	}, nil)

	tools, err := service.resolveAvailableTools(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("resolve available tools: %v", err)
	}
	toolNames := toolNamesByID(tools)
	if _, ok := toolNames["mail_send"]; ok {
		t.Fatalf("expected blocked mail connector capability to be excluded, got %+v", tools)
	}
	if _, ok := toolNames["message_send"]; !ok {
		t.Fatalf("expected ready message_send capability to remain available, got %+v", tools)
	}
}

func TestUnifiedTurnServiceResolveAvailableToolsIncludesExpandedConnectorCapabilities(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	service := newUnifiedTurnTestService(t, container, &unifiedModelChatStub{}, &unifiedAgentStub{}, &unifiedUIStatusStub{
		connectorResponse: transport.ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail.draft", "mail.reply", "mail.unread_summary"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "calendar",
					Capabilities:    []string{"calendar.update"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "browser",
					Capabilities:    []string{"browser.extract", "browser.close"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "finder",
					Capabilities:    []string{"finder.preview", "finder.find"},
					ActionReadiness: "ready",
				},
				{
					ConnectorID:     "twilio",
					Capabilities:    []string{"channel.twilio.sms.send"},
					ActionReadiness: "ready",
				},
			},
		},
	}, nil)

	tools, err := service.resolveAvailableTools(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("resolve available tools: %v", err)
	}
	toolNames := toolNamesByID(tools)
	for _, expected := range []string{
		"mail_draft",
		"mail_reply",
		"mail_unread_summary",
		"calendar_update",
		"browser_extract",
		"browser_close",
		"finder_find",
		"finder_preview",
		"message_send",
	} {
		if _, ok := toolNames[expected]; !ok {
			t.Fatalf("expected resolved tool %s, got %+v", expected, tools)
		}
	}
	if _, ok := toolNames["mail_send"]; ok {
		t.Fatalf("did not expect mail_send when capability inventory only includes draft+reply, got %+v", tools)
	}
}

func TestUnifiedTurnServiceResolveAvailableToolsDeduplicatesAliasCapabilities(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	service := newUnifiedTurnTestService(t, container, &unifiedModelChatStub{}, &unifiedAgentStub{}, &unifiedUIStatusStub{
		connectorResponse: transport.ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []transport.ConnectorStatusCard{
				{
					ConnectorID:     "mail",
					Capabilities:    []string{"mail.send", "mail_send"},
					ActionReadiness: "ready",
				},
			},
		},
	}, nil)

	tools, err := service.resolveAvailableTools(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("resolve available tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected exactly one tool from aliased mail.send/mail_send capabilities, got %+v", tools)
	}
	tool, found := findToolByName(tools, "mail_send")
	if !found {
		t.Fatalf("expected mail_send tool, got %+v", tools)
	}
	if len(tool.CapabilityKeys) != 1 || tool.CapabilityKeys[0] != "mail_send" {
		t.Fatalf("expected canonical deduplicated capability key mail_send, got %+v", tool.CapabilityKeys)
	}
}

func TestBuildMailNativeActionUnreadSummarySupportsOptionalLimit(t *testing.T) {
	action, err := buildMailNativeAction("summarize_unread", map[string]any{
		"limit": float64(3),
	})
	if err != nil {
		t.Fatalf("build unread summary native action: %v", err)
	}
	if action == nil || action.Mail == nil {
		t.Fatalf("expected mail native action payload")
	}
	if action.Mail.Operation != "summarize_unread" {
		t.Fatalf("expected summarize_unread operation, got %+v", action.Mail)
	}
	if action.Mail.Limit != 3 {
		t.Fatalf("expected limit=3, got %d", action.Mail.Limit)
	}
	if strings.TrimSpace(action.Mail.Body) != "" {
		t.Fatalf("expected empty body for summarize_unread, got %q", action.Mail.Body)
	}
}

func TestUnifiedTurnServiceBuildToolSchemaRegistryUsesPolicyInputsForPlannerVisibility(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	service := newUnifiedTurnTestService(t, container, &unifiedModelChatStub{}, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	messageTool, ok := modelToolDefinitionFromCapability("messages_send_sms")
	if !ok {
		t.Fatalf("expected messages_send_sms tool definition")
	}
	registry, err := service.buildToolSchemaRegistry(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		Channel: transport.ChatTurnChannelContext{
			ChannelID: "voice",
		},
		RequestedByActorID: "actor.requested",
		ActingAsActorID:    "actor.requested",
	}, "ws1", []modelToolDefinition{messageTool})
	if err != nil {
		t.Fatalf("build tool schema registry: %v", err)
	}
	if strings.TrimSpace(registry.Version) != modelToolSchemaRegistryVersionV1 {
		t.Fatalf("expected registry version %s, got %q", modelToolSchemaRegistryVersionV1, registry.Version)
	}
	if len(registry.Entries) != 1 {
		t.Fatalf("expected one registry entry, got %+v", registry.Entries)
	}
	if registry.Entries[0].Policy.Decision != ToolPolicyDecisionDeny || registry.Entries[0].Policy.ReasonCode != "channel_not_allowed" {
		t.Fatalf("expected voice message_send to be denied by channel constraints, got %+v", registry.Entries[0].Policy)
	}
	if registry.Entries[0].PlannerVisible {
		t.Fatalf("expected denied capability to be hidden from planner catalog, got %+v", registry.Entries[0])
	}
	if len(registry.plannerToolCatalogEntries()) != 0 {
		t.Fatalf("expected planner-visible tool catalog to omit denied entries, got %+v", registry.plannerToolCatalogEntries())
	}
	if len(registry.allToolCatalogEntries()) != 1 {
		t.Fatalf("expected explainability catalog to include denied entries, got %+v", registry.allToolCatalogEntries())
	}
}

func TestBuildCalendarNativeActionUpdateRequiresEventID(t *testing.T) {
	_, err := buildCalendarNativeAction("update", map[string]any{
		"title": "Updated title",
	})
	if err == nil {
		t.Fatalf("expected missing event_id error for calendar update")
	}
}

func TestBuildCalendarNativeActionUpdateIncludesEventFields(t *testing.T) {
	action, err := buildCalendarNativeAction("update", map[string]any{
		"event_id": "event-team-sync-1",
		"title":    "Team sync (updated)",
		"notes":    "Bring roadmap.",
	})
	if err != nil {
		t.Fatalf("build calendar update native action: %v", err)
	}
	if action == nil || action.Calendar == nil {
		t.Fatalf("expected calendar native action payload")
	}
	if action.Calendar.Operation != "update" {
		t.Fatalf("expected update operation, got %+v", action.Calendar)
	}
	if action.Calendar.EventID != "event-team-sync-1" {
		t.Fatalf("expected event_id event-team-sync-1, got %q", action.Calendar.EventID)
	}
	if action.Calendar.Notes != "Bring roadmap." {
		t.Fatalf("expected notes to be preserved, got %q", action.Calendar.Notes)
	}
}

func TestBuildMessageNativeActionAcceptsCanonicalLogicalChannel(t *testing.T) {
	action, err := buildMessageNativeAction(map[string]any{
		"channel":   "message",
		"recipient": "+15550001111",
		"body":      "hello",
	})
	if err != nil {
		t.Fatalf("build message native action: %v", err)
	}
	if action == nil || action.Messages == nil {
		t.Fatalf("expected messages native action payload")
	}
	if action.Messages.Channel != "imessage" {
		t.Fatalf("expected message alias to normalize to imessage, got %q", action.Messages.Channel)
	}
}

func TestBuildMessageNativeActionRejectsNonCanonicalChannelAliases(t *testing.T) {
	cases := []string{"twilio", "text", "i_message"}
	for _, channel := range cases {
		_, err := buildMessageNativeAction(map[string]any{
			"channel":   channel,
			"recipient": "+15550002222",
			"body":      "hello",
		})
		if err == nil {
			t.Fatalf("expected alias %q to be rejected", channel)
		}
		if !strings.Contains(err.Error(), `unsupported message channel "`) || !strings.Contains(err.Error(), "allowed: message|imessage|sms") {
			t.Fatalf("expected deterministic canonical-channel validation error for %q, got %v", channel, err)
		}
	}
}

func TestBuildBrowserNativeActionExtractSupportsOptionalQuery(t *testing.T) {
	action, err := buildBrowserNativeAction("extract", map[string]any{
		"url":   "https://example.com",
		"query": "summarize key facts",
	})
	if err != nil {
		t.Fatalf("build browser extract native action: %v", err)
	}
	if action == nil || action.Browser == nil {
		t.Fatalf("expected browser native action payload")
	}
	if action.Browser.Operation != "extract" {
		t.Fatalf("expected extract operation, got %+v", action.Browser)
	}
	if action.Browser.Query != "summarize key facts" {
		t.Fatalf("expected query propagation, got %q", action.Browser.Query)
	}
}

func TestBuildFinderNativeActionFindRequiresQuery(t *testing.T) {
	_, err := buildFinderNativeAction("find", map[string]any{
		"root_path": "/tmp",
	})
	if err == nil {
		t.Fatalf("expected missing query error for finder find")
	}
}

func TestBuildFinderNativeActionPreviewSupportsQueryResolution(t *testing.T) {
	action, err := buildFinderNativeAction("preview", map[string]any{
		"query":     "travel checklist",
		"root_path": "/tmp",
	})
	if err != nil {
		t.Fatalf("build finder preview native action: %v", err)
	}
	if action == nil || action.Finder == nil {
		t.Fatalf("expected finder native action payload")
	}
	if action.Finder.Operation != "preview" {
		t.Fatalf("expected preview operation, got %+v", action.Finder)
	}
	if action.Finder.Query != "travel checklist" {
		t.Fatalf("expected query propagation, got %q", action.Finder.Query)
	}
	if action.Finder.RootPath != "/tmp" {
		t.Fatalf("expected root_path propagation, got %q", action.Finder.RootPath)
	}
}

func TestUnifiedTurnServiceChatTurnModelToolModelSuccess(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)

	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"We are on track."}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Done. I sent the update to Sam.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{{
		Workflow:  "send_email",
		TaskID:    "task-1",
		RunID:     "run-1",
		TaskState: "completed",
		RunState:  "completed",
	}}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:     "mail",
			Capabilities:    []string{"mail_send"},
			ActionReadiness: "ready",
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.requested",
		SubjectActorID:     "actor.requested",
		ActingAsActorID:    "actor.requested",
		Channel: transport.ChatTurnChannelContext{
			ChannelID:   "message",
			ConnectorID: "twilio",
			ThreadID:    "thread-1",
		},
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Please email Sam that we're on track.",
		}},
	}, "corr-unified-success", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}

	if len(model.requests) != 2 {
		t.Fatalf("expected two model calls (planner + response), got %d", len(model.requests))
	}
	if !strings.Contains(model.requests[0].SystemPrompt, `"name":"mail_send"`) {
		t.Fatalf("expected planner prompt to advertise mail_send tool, got %q", model.requests[0].SystemPrompt)
	}
	if strings.Contains(model.requests[0].SystemPrompt, `"name":"finder_delete"`) {
		t.Fatalf("expected unavailable tool finder_delete to be omitted from planner prompt, got %q", model.requests[0].SystemPrompt)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one tool execution call, got %d", len(agent.requests))
	}
	if strings.TrimSpace(agent.requests[0].Origin) != "app" {
		t.Fatalf("expected message-channel tool execution origin app, got %q", agent.requests[0].Origin)
	}
	if agent.requests[0].NativeAction == nil || agent.requests[0].NativeAction.Mail == nil {
		t.Fatalf("expected tool execution to carry mail native action payload, got %+v", agent.requests[0])
	}
	if strings.TrimSpace(agent.requests[0].NativeAction.Mail.Recipient) != "sam@example.com" {
		t.Fatalf("expected tool native action recipient sam@example.com, got %+v", agent.requests[0].NativeAction.Mail)
	}

	toolCallSeen := false
	toolResultSeen := false
	assistantSeen := false
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_call":
			toolCallSeen = true
		case "tool_result":
			if strings.ToLower(strings.TrimSpace(item.Status)) != "completed" {
				t.Fatalf("expected completed tool_result status, got %+v", item)
			}
			if strings.TrimSpace(item.ErrorCode) != "" {
				t.Fatalf("expected no tool_result error_code, got %+v", item)
			}
			toolResultSeen = true
		case "assistant_message":
			if strings.Contains(strings.ToLower(item.Content), "sent") {
				assistantSeen = true
			}
		}
	}
	if !toolCallSeen || !toolResultSeen || !assistantSeen {
		t.Fatalf("expected tool_call/tool_result/final assistant in response, got %+v", response.Items)
	}
	if !response.TaskRunCorrelation.Available || response.TaskRunCorrelation.TaskID != "task-1" || response.TaskRunCorrelation.RunID != "run-1" {
		t.Fatalf("unexpected task/run correlation payload: %+v", response.TaskRunCorrelation)
	}

	history, err := service.QueryChatTurnHistory(context.Background(), transport.ChatTurnHistoryRequest{
		WorkspaceID:   "ws1",
		CorrelationID: "corr-unified-success",
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("query chat turn history: %v", err)
	}
	if len(history.Items) == 0 {
		t.Fatalf("expected persisted turn ledger records")
	}
}

func TestUnifiedTurnServiceChatTurnRepairsMalformedPlannerOutputBeforeToolExecution(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Broken planner output"`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Recovered planner output"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Done. I sent the recovered update.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{{
		Workflow:  "send_email",
		TaskID:    "task-repair",
		RunID:     "run-repair",
		TaskState: "completed",
		RunState:  "completed",
	}}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:     "mail",
			Capabilities:    []string{"mail_send"},
			ActionReadiness: "ready",
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send Sam an update.",
		}},
	}, "corr-planner-repair-success", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(model.requests) != 3 {
		t.Fatalf("expected planner + repair + final synthesis calls, got %d", len(model.requests))
	}
	if len(model.requests[0].ToolCatalog) != 1 || strings.TrimSpace(model.requests[0].ToolCatalog[0].Name) != "mail_send" {
		t.Fatalf("expected typed tool catalog in planner request, got %+v", model.requests[0].ToolCatalog)
	}
	if len(model.requests[1].ToolCatalog) != 1 || strings.TrimSpace(model.requests[1].ToolCatalog[0].Name) != "mail_send" {
		t.Fatalf("expected typed tool catalog in planner repair request, got %+v", model.requests[1].ToolCatalog)
	}
	if len(model.requests[2].ToolCatalog) != 1 || strings.TrimSpace(model.requests[2].ToolCatalog[0].Name) != "mail_send" {
		t.Fatalf("expected typed tool catalog in post-tool planner request, got %+v", model.requests[2].ToolCatalog)
	}
	if !strings.Contains(model.requests[1].SystemPrompt, "Planner repair mode:") {
		t.Fatalf("expected second model call to be planner repair mode, got %q", model.requests[1].SystemPrompt)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one tool execution after planner repair, got %d", len(agent.requests))
	}

	foundCompletedToolResult := false
	finalAssistant := transport.ChatTurnItem{}
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_result":
			if strings.ToLower(strings.TrimSpace(item.Status)) == "completed" {
				foundCompletedToolResult = true
			}
		case "assistant_message":
			finalAssistant = item
		}
	}
	if !foundCompletedToolResult {
		t.Fatalf("expected completed tool_result after planner repair, got %+v", response.Items)
	}
	if !strings.Contains(strings.ToLower(finalAssistant.Content), "sent the recovered update") {
		t.Fatalf("expected repaired final assistant content, got %+v", finalAssistant)
	}
	if fmt.Sprintf("%v", finalAssistant.Metadata.AsMap()["planner_repair_attempts"]) != "1" {
		t.Fatalf("expected planner_repair_attempts=1, got %+v", finalAssistant.Metadata)
	}
}

func TestUnifiedTurnServiceChatTurnFallsBackAfterPlannerRepairRetriesExhausted(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"broken"`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send"`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `still not valid planner json`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Fallback response after planner repair retries.",
			}},
		},
	}}
	agent := &unifiedAgentStub{}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:     "mail",
			Capabilities:    []string{"mail_send"},
			ActionReadiness: "ready",
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send Sam an update.",
		}},
	}, "corr-planner-repair-exhausted", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(model.requests) != 4 {
		t.Fatalf("expected planner + two repair retries + model-only recovery, got %d", len(model.requests))
	}
	if !strings.Contains(model.requests[1].SystemPrompt, "Planner repair mode:") || !strings.Contains(model.requests[2].SystemPrompt, "Planner repair mode:") {
		t.Fatalf("expected repair-mode prompts for planner retries, got prompts[1]=%q prompts[2]=%q", model.requests[1].SystemPrompt, model.requests[2].SystemPrompt)
	}
	if !strings.Contains(model.requests[3].SystemPrompt, "Response synthesis mode:") {
		t.Fatalf("expected final model-only recovery prompt, got %q", model.requests[3].SystemPrompt)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("expected no tool executions when planner repair retries are exhausted, got %d", len(agent.requests))
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one fallback assistant item, got %+v", response.Items)
	}
	assistant := response.Items[0]
	if strings.ToLower(strings.TrimSpace(assistant.Type)) != "assistant_message" {
		t.Fatalf("expected assistant_message item, got %+v", assistant)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", assistant.Metadata.AsMap()["stop_reason"])) != "planner_output_invalid" {
		t.Fatalf("expected stop_reason planner_output_invalid, got %+v", assistant.Metadata)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", assistant.Metadata.AsMap()["planner_repair_attempts"])) != "2" {
		t.Fatalf("expected planner_repair_attempts=2, got %+v", assistant.Metadata)
	}
	remediation, ok := assistant.Metadata.AsMap()["remediation"].(map[string]any)
	if !ok {
		t.Fatalf("expected planner remediation metadata, got %+v", assistant.Metadata)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", remediation["code"])) != "planner_output_invalid" {
		t.Fatalf("expected remediation code planner_output_invalid, got %+v", remediation)
	}
}

func TestUnifiedTurnServiceChatTurnSupportsIterativeToolLoop(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Shipping update"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"calendar_create","arguments":{"title":"Ship review"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Done. I sent the update and created the calendar event.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{
		{
			Workflow:  "send_email",
			TaskID:    "task-1",
			RunID:     "run-1",
			TaskState: "completed",
			RunState:  "completed",
		},
		{
			Workflow:  "calendar_create",
			TaskID:    "task-2",
			RunID:     "run-2",
			TaskState: "completed",
			RunState:  "completed",
		},
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{
			{
				ConnectorID:     "mail",
				Capabilities:    []string{"mail_send"},
				ActionReadiness: "ready",
			},
			{
				ConnectorID:     "calendar",
				Capabilities:    []string{"calendar_create"},
				ActionReadiness: "ready",
			},
		},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Email Sam and create a ship review event.",
		}},
	}, "corr-iterative-loop", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected two tool executions, got %d", len(agent.requests))
	}
	if agent.requests[0].NativeAction == nil || agent.requests[0].NativeAction.Mail == nil {
		t.Fatalf("expected first tool request to include mail native action payload, got %+v", agent.requests[0])
	}
	if strings.TrimSpace(agent.requests[0].NativeAction.Mail.Recipient) != "sam@example.com" {
		t.Fatalf("expected first tool recipient sam@example.com, got %+v", agent.requests[0].NativeAction.Mail)
	}
	if agent.requests[1].NativeAction == nil || agent.requests[1].NativeAction.Calendar == nil {
		t.Fatalf("expected second tool request to include calendar native action payload, got %+v", agent.requests[1])
	}
	if strings.TrimSpace(agent.requests[1].NativeAction.Calendar.Title) != "Ship review" {
		t.Fatalf("expected calendar title Ship review, got %+v", agent.requests[1].NativeAction.Calendar)
	}
	if len(model.requests) != 3 {
		t.Fatalf("expected three planner calls for two tools plus final assistant, got %d", len(model.requests))
	}

	toolCallCount := 0
	toolResultCount := 0
	finalAssistant := transport.ChatTurnItem{}
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_call":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		case "assistant_message":
			finalAssistant = item
		}
	}
	if toolCallCount != 2 || toolResultCount != 2 {
		t.Fatalf("expected two tool_call and two tool_result items, got %+v", response.Items)
	}
	if !strings.Contains(strings.ToLower(finalAssistant.Content), "created the calendar event") {
		t.Fatalf("expected final assistant summary from iterative planner, got %+v", finalAssistant)
	}
}

func TestUnifiedTurnServiceChatTurnNaturalLanguageEmailAndBrowserToolChain(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Sharing the browser findings shortly."}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"browser_open","arguments":{"url":"https://example.com"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"browser_extract","arguments":{"url":"https://example.com","query":"summarize key details"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Done. I emailed Sam and shared the page summary.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{
		{Workflow: "send_email", TaskID: "task-mail", RunID: "run-mail", TaskState: "completed", RunState: "completed"},
		{Workflow: "browser_open", TaskID: "task-open", RunID: "run-open", TaskState: "completed", RunState: "completed"},
		{Workflow: "browser_extract", TaskID: "task-extract", RunID: "run-extract", TaskState: "completed", RunState: "completed"},
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{
			{ConnectorID: "mail", Capabilities: []string{"mail_send"}, ActionReadiness: "ready"},
			{ConnectorID: "browser", Capabilities: []string{"browser_open", "browser_extract"}, ActionReadiness: "ready"},
		},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Email Sam and browse example.com to summarize key details.",
		}},
	}, "corr-e2e-email-browser", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 3 {
		t.Fatalf("expected three tool executions for email+browser chain, got %d", len(agent.requests))
	}

	toolCallCount := 0
	toolResultCount := 0
	assistantCount := 0
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_call":
			toolCallCount++
		case "tool_result":
			toolResultCount++
			if strings.ToLower(strings.TrimSpace(item.Status)) != "completed" {
				t.Fatalf("expected completed tool_result, got %+v", item)
			}
		case "assistant_message":
			assistantCount++
		}
	}
	if toolCallCount != 3 || toolResultCount != 3 || assistantCount == 0 {
		t.Fatalf("expected tool chain items in response, got %+v", response.Items)
	}

	history, err := service.QueryChatTurnHistory(context.Background(), transport.ChatTurnHistoryRequest{
		WorkspaceID:   "ws1",
		CorrelationID: "corr-e2e-email-browser",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("query chat turn history: %v", err)
	}
	if !historyContainsType(history.Items, "user_message") ||
		!historyContainsType(history.Items, "tool_call") ||
		!historyContainsType(history.Items, "tool_result") ||
		!historyContainsType(history.Items, "assistant_message") {
		t.Fatalf("expected persisted user->tool_call->tool_result->assistant chain, got %+v", history.Items)
	}
}

func TestUnifiedTurnServiceChatTurnNaturalLanguageEmailFailureEdge(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Status update"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "I could not send that email because the connector failed.",
			}},
		},
	}}
	agent := &unifiedAgentStub{errs: []error{errors.New("mail connector unavailable")}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "mail",
			Capabilities: []string{"mail_send"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Email Sam a quick status update.",
		}},
	}, "corr-e2e-email-failure", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one email tool execution, got %d", len(agent.requests))
	}

	foundFailure := false
	foundAssistant := false
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_result":
			if strings.ToLower(strings.TrimSpace(item.ErrorCode)) == "tool_execution_failed" && strings.ToLower(strings.TrimSpace(item.Status)) == "failed" {
				remediation := item.Metadata
				if strings.TrimSpace(fmt.Sprintf("%v", remediation.AsMap()["code"])) != "tool_execution_failure" {
					t.Fatalf("expected tool_result remediation metadata, got %+v", item)
				}
				foundFailure = true
			}
		case "assistant_message":
			foundAssistant = true
		}
	}
	if !foundFailure || !foundAssistant {
		t.Fatalf("expected failure edge tool_result + assistant message, got %+v", response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnNaturalLanguageBrowserApprovalEdge(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"browser_open","arguments":{"url":"https://example.com"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"finder_delete","arguments":{"path":"/tmp/example.txt"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Browser step completed; waiting for approval before deleting the file.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{
		{Workflow: "browser_open", TaskID: "task-open", RunID: "run-open", TaskState: "completed", RunState: "completed"},
		{Workflow: "finder_delete", TaskID: "task-delete", RunID: "run-delete", TaskState: "awaiting_approval", RunState: "awaiting_approval", ApprovalRequired: true, ApprovalRequestID: "approval-delete-1"},
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{
			{ConnectorID: "browser", Capabilities: []string{"browser_open"}, ActionReadiness: "ready"},
			{ConnectorID: "finder", Capabilities: []string{"finder_delete"}, ActionReadiness: "ready"},
		},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Open example.com and then delete /tmp/example.txt.",
		}},
	}, "corr-e2e-browser-approval", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 2 {
		t.Fatalf("expected browser + approval-gated delete execution requests, got %d", len(agent.requests))
	}

	foundAwaitingApprovalResult := false
	foundApprovalItem := false
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_result":
			if strings.ToLower(strings.TrimSpace(item.Status)) == "awaiting_approval" && strings.TrimSpace(item.ApprovalRequestID) == "approval-delete-1" {
				foundAwaitingApprovalResult = true
			}
		case "approval_request":
			if strings.TrimSpace(item.ApprovalRequestID) == "approval-delete-1" {
				foundApprovalItem = true
			}
		}
	}
	if !foundAwaitingApprovalResult || !foundApprovalItem {
		t.Fatalf("expected approval edge turn items, got %+v", response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnReturnsModelRouteRemediationOnPlannerRouteFailure(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{errs: []error{errors.New(`no enabled action-capable models with ready provider configuration for task class "chat" in workspace "ws1"`)}}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send Sam a status update.",
		}},
	}, "corr-model-route-remediation", nil)
	if err != nil {
		t.Fatalf("expected remediation response instead of hard error, got %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected single remediation assistant item, got %+v", response.Items)
	}
	assistant := response.Items[0]
	if strings.ToLower(strings.TrimSpace(assistant.Type)) != "assistant_message" {
		t.Fatalf("expected assistant_message remediation item, got %+v", assistant)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", assistant.Metadata.AsMap()["stop_reason"])) != "model_route_unavailable" {
		t.Fatalf("expected stop_reason model_route_unavailable, got %+v", assistant.Metadata)
	}
	remediation, ok := assistant.Metadata.AsMap()["remediation"].(map[string]any)
	if !ok {
		t.Fatalf("expected remediation metadata map, got %+v", assistant.Metadata)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", remediation["code"])) != "model_route_unavailable" {
		t.Fatalf("expected remediation code model_route_unavailable, got %+v", remediation)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", remediation["primary_action"])) != "open_models" {
		t.Fatalf("expected remediation primary_action open_models, got %+v", remediation)
	}
}

func TestUnifiedTurnServiceChatTurnStopsAtToolCallLimit(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Loop step 1"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Loop step 2"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Loop step 3"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Loop step 4"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{
		{Workflow: "send_email", TaskID: "task-1", RunID: "run-1", TaskState: "completed", RunState: "completed"},
		{Workflow: "send_email", TaskID: "task-2", RunID: "run-2", TaskState: "completed", RunState: "completed"},
		{Workflow: "send_email", TaskID: "task-3", RunID: "run-3", TaskState: "completed", RunState: "completed"},
		{Workflow: "send_email", TaskID: "task-4", RunID: "run-4", TaskState: "completed", RunState: "completed"},
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:     "mail",
			Capabilities:    []string{"mail_send"},
			ActionReadiness: "ready",
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Keep sending updates.",
		}},
	}, "corr-tool-limit", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 4 {
		t.Fatalf("expected exactly four tool executions before limit stop, got %d", len(agent.requests))
	}

	var (
		toolCallCount   int
		toolResultCount int
		finalAssistant  transport.ChatTurnItem
	)
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_call":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		case "assistant_message":
			finalAssistant = item
		}
	}
	if toolCallCount != 4 || toolResultCount != 4 {
		t.Fatalf("expected four tool_call and four tool_result items before stop, got %+v", response.Items)
	}
	if !strings.Contains(strings.ToLower(finalAssistant.Content), "maximum of 4 tool calls") {
		t.Fatalf("expected final assistant to mention tool-call limit, got %+v", finalAssistant)
	}
	stopReason := strings.TrimSpace(fmt.Sprintf("%v", finalAssistant.Metadata.AsMap()["stop_reason"]))
	if stopReason != "tool_call_limit_reached" {
		t.Fatalf("expected stop_reason=tool_call_limit_reached, got %+v", finalAssistant.Metadata)
	}
}

func TestUnifiedTurnServiceChatTurnRejectsInvalidToolArguments(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"body":"Missing recipient"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "I need the recipient before sending.",
			}},
		},
	}}
	agent := &unifiedAgentStub{}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "mail",
			Capabilities: []string{"mail_send"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send the message.",
		}},
	}, "corr-invalid-args", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("expected no tool execution when arguments are invalid, got %d calls", len(agent.requests))
	}

	foundInvalid := false
	for _, item := range response.Items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "tool_result" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ErrorCode)) == toolSchemaErrorMissingRequiredArgument {
			foundInvalid = true
			if strings.ToLower(strings.TrimSpace(item.Status)) != "failed" {
				t.Fatalf("expected failed status for invalid tool args, got %+v", item)
			}
			if strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["validation_error_code"])) != toolSchemaErrorMissingRequiredArgument {
				t.Fatalf("expected validation_error_code %s, got %+v", toolSchemaErrorMissingRequiredArgument, item.Metadata)
			}
			if strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["schema_registry_version"])) != modelToolSchemaRegistryVersionV1 {
				t.Fatalf("expected schema_registry_version %s, got %+v", modelToolSchemaRegistryVersionV1, item.Metadata)
			}
		}
	}
	if !foundInvalid {
		t.Fatalf("expected %s result item, got %+v", toolSchemaErrorMissingRequiredArgument, response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnRejectsUnknownToolArgumentsWithSchemaErrorCode(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Hello","priority":"high"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "I need to adjust the arguments before retrying.",
			}},
		},
	}}
	agent := &unifiedAgentStub{}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "mail",
			Capabilities: []string{"mail_send"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send the message with highest priority.",
		}},
	}, "corr-invalid-unknown-arg", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 0 {
		t.Fatalf("expected no tool execution when arguments include unknown fields, got %d calls", len(agent.requests))
	}

	foundUnknownArgumentError := false
	for _, item := range response.Items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "tool_result" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ErrorCode)) != toolSchemaErrorUnknownArgument {
			continue
		}
		foundUnknownArgumentError = true
		if strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["validation_error_code"])) != toolSchemaErrorUnknownArgument {
			t.Fatalf("expected validation_error_code %s, got %+v", toolSchemaErrorUnknownArgument, item.Metadata)
		}
		if strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["validation_argument"])) != "priority" {
			t.Fatalf("expected validation_argument priority, got %+v", item.Metadata)
		}
	}
	if !foundUnknownArgumentError {
		t.Fatalf("expected %s result item, got %+v", toolSchemaErrorUnknownArgument, response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnToolExecutionFailure(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"mail_send","arguments":{"recipient":"sam@example.com","body":"Hello"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "I could not complete that action.",
			}},
		},
	}}
	agent := &unifiedAgentStub{errs: []error{errors.New("connector unavailable")}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "mail",
			Capabilities: []string{"mail_send"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Send email",
		}},
	}, "corr-tool-failure", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one tool execution request, got %d", len(agent.requests))
	}

	foundToolFailure := false
	for _, item := range response.Items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "tool_result" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ErrorCode)) == "tool_execution_failed" {
			foundToolFailure = true
			if strings.ToLower(strings.TrimSpace(item.Status)) != "failed" {
				t.Fatalf("expected failed tool_result status for execution failure, got %+v", item)
			}
		}
	}
	if !foundToolFailure {
		t.Fatalf("expected tool_execution_failed result item, got %+v", response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnPolicyRequireApprovalFlow(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"finder_delete","arguments":{"path":"/tmp/test.txt"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Waiting for approval before deleting.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{{
		Workflow:          "finder_delete",
		TaskID:            "task-delete",
		RunID:             "run-delete",
		TaskState:         "awaiting_approval",
		RunState:          "awaiting_approval",
		ApprovalRequired:  true,
		ApprovalRequestID: "approval-1",
	}}}
	delegation := &unifiedDelegationStub{checkResponse: transport.DelegationCheckResponse{
		Allowed: true,
		Reason:  "delegated",
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "finder",
			Capabilities: []string{"finder_delete"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, delegation)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.requested",
		ActingAsActorID:    "actor.delegated",
		Channel: transport.ChatTurnChannelContext{
			ChannelID: "voice",
		},
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Delete that file.",
		}},
	}, "corr-require-approval", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(delegation.checkRequests) != 1 {
		t.Fatalf("expected delegated execution check, got %d checks", len(delegation.checkRequests))
	}
	if len(agent.requests) != 1 {
		t.Fatalf("expected one tool execution request, got %d", len(agent.requests))
	}
	if strings.TrimSpace(agent.requests[0].Origin) != "voice" {
		t.Fatalf("expected voice-channel tool execution origin voice, got %q", agent.requests[0].Origin)
	}

	toolCallDecisionSeen := false
	toolCallRationaleSeen := false
	toolApprovalResultSeen := false
	toolResultRationaleSeen := false
	approvalItemSeen := false
	approvalRationaleSeen := false
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "tool_call":
			policyDecision := strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["policy_decision"]))
			policyReasonCode := strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["policy_reason_code"]))
			if strings.TrimSpace(item.ToolName) == "finder_delete" && policyDecision == "REQUIRE_APPROVAL" && policyReasonCode == "approval_required" {
				toolCallDecisionSeen = true
			}
			rationale, ok := item.Metadata.AsMap()["policy_rationale"].(map[string]any)
			if ok && strings.TrimSpace(fmt.Sprintf("%v", rationale["capability_key"])) == "finder_delete" {
				toolCallRationaleSeen = true
			}
		case "tool_result":
			if strings.TrimSpace(item.ApprovalRequestID) == "approval-1" && strings.ToLower(strings.TrimSpace(item.Status)) == "awaiting_approval" {
				toolApprovalResultSeen = true
			}
			policyReasonCode := strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["policy_reason_code"]))
			if policyReasonCode == "approval_required" {
				toolResultRationaleSeen = true
			}
		case "approval_request":
			if strings.TrimSpace(item.ApprovalRequestID) == "approval-1" {
				approvalItemSeen = true
			}
			rationale, ok := item.Metadata.AsMap()["policy_rationale"].(map[string]any)
			if ok && strings.TrimSpace(fmt.Sprintf("%v", rationale["decision_reason_code"])) == "approval_required" {
				approvalRationaleSeen = true
			}
		}
	}
	if !toolCallDecisionSeen || !toolCallRationaleSeen || !toolApprovalResultSeen || !toolResultRationaleSeen || !approvalItemSeen || !approvalRationaleSeen {
		t.Fatalf("expected policy decision + approval flow items, got %+v", response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnPolicyRequireApprovalUnmet(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"tool_call","tool_name":"finder_delete","arguments":{"path":"/tmp/test.txt"}}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Execution failed policy checks.",
			}},
		},
	}}
	agent := &unifiedAgentStub{responses: []transport.AgentRunResponse{{
		Workflow:  "finder_delete",
		TaskID:    "task-delete",
		RunID:     "run-delete",
		TaskState: "completed",
		RunState:  "completed",
	}}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:  "finder",
			Capabilities: []string{"finder_delete"},
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, agent, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Delete this file.",
		}},
	}, "corr-unmet-approval", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}

	foundUnmet := false
	for _, item := range response.Items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "tool_result" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ErrorCode)) == "approval_required_unmet" {
			foundUnmet = true
			if strings.ToLower(strings.TrimSpace(item.Status)) != "failed" {
				t.Fatalf("expected failed status for unmet approval policy, got %+v", item)
			}
			if strings.TrimSpace(fmt.Sprintf("%v", item.Metadata.AsMap()["policy_reason_code"])) != "approval_required" {
				t.Fatalf("expected policy_reason_code approval_required for unmet approval item, got %+v", item.Metadata)
			}
		}
	}
	if !foundUnmet {
		t.Fatalf("expected approval_required_unmet tool result item, got %+v", response.Items)
	}
}

func TestUnifiedTurnServiceChatTurnRespectsCancellation(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{blockOnCall: 1}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.ChatTurn(ctx, transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Hello",
		}},
	}, "corr-cancel", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestUnifiedTurnServiceChatTurnPlannerEmptyFallsBackToModelOnlyResponse(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "ollama",
			ModelKey:    "gpt-oss:20b",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "",
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-5.2",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "Hello there.",
			}},
		},
	}}
	ui := &unifiedUIStatusStub{connectorResponse: transport.ConnectorStatusResponse{
		WorkspaceID: "ws1",
		Connectors: []transport.ConnectorStatusCard{{
			ConnectorID:     "mail",
			Capabilities:    []string{"mail_send"},
			ActionReadiness: "ready",
		}},
	}}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, ui, nil)

	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Say hello.",
		}},
	}, "corr-planner-empty-fallback", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(model.requests) != 2 {
		t.Fatalf("expected planner + fallback model calls, got %d", len(model.requests))
	}
	if !strings.Contains(model.requests[0].SystemPrompt, "Unified-turn orchestration mode:") {
		t.Fatalf("expected planner prompt for first model call, got %q", model.requests[0].SystemPrompt)
	}
	if !strings.Contains(model.requests[1].SystemPrompt, "Response synthesis mode:") {
		t.Fatalf("expected response prompt for fallback model call, got %q", model.requests[1].SystemPrompt)
	}
	if len(model.requests[0].ToolCatalog) != 1 || strings.TrimSpace(model.requests[0].ToolCatalog[0].Name) != "mail_send" {
		t.Fatalf("expected planner request to include typed tool catalog, got %+v", model.requests[0].ToolCatalog)
	}
	if len(model.requests[1].ToolCatalog) != 0 {
		t.Fatalf("expected response synthesis request to omit tool catalog, got %+v", model.requests[1].ToolCatalog)
	}
	if response.Provider != "openai" || response.ModelKey != "gpt-5.2" {
		t.Fatalf("expected fallback response provider/model, got %s/%s", response.Provider, response.ModelKey)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected single model-only assistant item, got %+v", response.Items)
	}
	item := response.Items[0]
	if strings.ToLower(strings.TrimSpace(item.Type)) != "assistant_message" {
		t.Fatalf("expected assistant_message item, got %+v", item)
	}
	if strings.TrimSpace(item.Content) != "Hello there." {
		t.Fatalf("expected fallback content, got %+v", item)
	}
	if strings.Contains(strings.ToLower(item.Content), "valid next action") {
		t.Fatalf("expected fallback message to be replaced by recovery response, got %+v", item)
	}
}

func TestUnifiedTurnServiceChatTurnStreamsModelOnlyResponseWhenTokenCallbackProvided(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "ollama",
			ModelKey:    "gpt-oss:20b",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"assistant_message","content":"I can help with that."}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "ollama",
			ModelKey:    "gpt-oss:20b",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "I can help with that.",
			}},
		},
	}}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	streamed := strings.Builder{}
	response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "What can you do?",
		}},
	}, "corr-model-only-streaming", func(delta string) {
		streamed.WriteString(delta)
	})
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(model.requests) != 2 {
		t.Fatalf("expected planner + streamed synthesis model calls, got %d", len(model.requests))
	}
	if !strings.Contains(model.requests[0].SystemPrompt, "Unified-turn orchestration mode:") {
		t.Fatalf("expected planner prompt for first model call, got %q", model.requests[0].SystemPrompt)
	}
	if !strings.Contains(model.requests[1].SystemPrompt, "Response synthesis mode:") {
		t.Fatalf("expected response synthesis prompt for second model call, got %q", model.requests[1].SystemPrompt)
	}
	if streamed.String() != "ok" {
		t.Fatalf("expected streamed token callback content, got %q", streamed.String())
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected a single assistant response item, got %+v", response.Items)
	}
	if got := strings.TrimSpace(response.Items[0].Content); got != "I can help with that." {
		t.Fatalf("expected streamed synthesis assistant content, got %q", got)
	}
}

func TestUnifiedTurnServiceAppliesPersonaAndContextBudgetTelemetry(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	seedWorkspacePrincipal(t, container.DB, "ws1", "actor.a")

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := container.DB.Exec(`
		INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?)
	`, "mem-1", "ws1", "actor.a", "fact", "travel.preference", `{"airline":"window"}`, "memory://travel.preference", now, now); err != nil {
		t.Fatalf("seed memory item: %v", err)
	}

	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Provider:    "openai",
		ModelKey:    "gpt-4.1-mini",
		Items: []transport.ChatTurnItem{{
			Type:    "assistant_message",
			Role:    "assistant",
			Status:  "completed",
			Content: `{"type":"assistant_message","content":"Acknowledged."}`,
		}},
	}}}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	_, err := service.UpsertChatPersonaPolicy(context.Background(), transport.ChatPersonaPolicyUpsertRequest{
		WorkspaceID:      "ws1",
		PrincipalActorID: "actor.a",
		ChannelID:        "app",
		StylePrompt:      "Respond like a precise concierge.",
		Guardrails:       []string{"Do not speculate."},
	})
	if err != nil {
		t.Fatalf("upsert persona policy: %v", err)
	}

	_, err = service.ChatTurn(context.Background(), transport.ChatTurnRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.a",
		SubjectActorID:     "actor.a",
		ActingAsActorID:    "actor.a",
		Channel: transport.ChatTurnChannelContext{
			ChannelID: "app",
		},
		Items: []transport.ChatTurnItem{{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Remember my travel preferences.",
		}},
	}, "corr-persona-context", nil)
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if len(model.requests) != 1 {
		t.Fatalf("expected single model call for model-only reply, got %d", len(model.requests))
	}
	plannerPrompt := model.requests[0].SystemPrompt
	if !strings.Contains(plannerPrompt, "Respond like a precise concierge.") {
		t.Fatalf("expected persona style prompt in planner prompt, got %q", plannerPrompt)
	}
	if !strings.Contains(plannerPrompt, "travel.preference") {
		t.Fatalf("expected retrieved memory context in planner prompt, got %q", plannerPrompt)
	}

	var sampleCount int
	if err := container.DB.QueryRow(`
		SELECT COUNT(*)
		FROM context_budget_samples
		WHERE workspace_id = 'ws1'
	`).Scan(&sampleCount); err != nil {
		t.Fatalf("count context budget samples: %v", err)
	}
	if sampleCount == 0 {
		t.Fatalf("expected context budget telemetry sample to be recorded")
	}
}

func TestUnifiedTurnServicePersonaPolicyPrecedenceByScope(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	seedWorkspacePrincipal(t, container.DB, "ws1", "actor.a")
	model := &unifiedModelChatStub{}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	upserts := []transport.ChatPersonaPolicyUpsertRequest{
		{
			WorkspaceID: "ws1",
			StylePrompt: "Workspace default style",
			Guardrails:  []string{"Workspace guardrail"},
		},
		{
			WorkspaceID:      "ws1",
			PrincipalActorID: "actor.a",
			StylePrompt:      "Principal default style",
			Guardrails:       []string{"Principal guardrail"},
		},
		{
			WorkspaceID: "ws1",
			ChannelID:   "message",
			StylePrompt: "Message channel default style",
			Guardrails:  []string{"Message guardrail"},
		},
		{
			WorkspaceID:      "ws1",
			PrincipalActorID: "actor.a",
			ChannelID:        "message",
			StylePrompt:      "Principal+message style",
			Guardrails:       []string{"Principal+message guardrail"},
		},
	}
	for _, upsert := range upserts {
		if _, err := service.UpsertChatPersonaPolicy(context.Background(), upsert); err != nil {
			t.Fatalf("upsert persona policy %+v: %v", upsert, err)
		}
	}

	testCases := []struct {
		name      string
		principal string
		channel   string
		wantStyle string
	}{
		{
			name:      "principal and channel match",
			principal: "actor.a",
			channel:   "message",
			wantStyle: "Principal+message style",
		},
		{
			name:      "principal-only fallback",
			principal: "actor.a",
			channel:   "app",
			wantStyle: "Principal default style",
		},
		{
			name:      "channel-only fallback",
			principal: "actor.b",
			channel:   "message",
			wantStyle: "Message channel default style",
		},
		{
			name:      "workspace fallback",
			principal: "actor.b",
			channel:   "voice",
			wantStyle: "Workspace default style",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := service.GetChatPersonaPolicy(context.Background(), transport.ChatPersonaPolicyRequest{
				WorkspaceID:      "ws1",
				PrincipalActorID: tc.principal,
				ChannelID:        tc.channel,
			})
			if err != nil {
				t.Fatalf("get persona policy: %v", err)
			}
			if got := strings.TrimSpace(policy.StylePrompt); got != tc.wantStyle {
				t.Fatalf("expected style %q, got %q", tc.wantStyle, got)
			}
			if got := strings.TrimSpace(policy.Source); got != "persisted" {
				t.Fatalf("expected persisted source, got %q", got)
			}
		})
	}
}

func TestUnifiedTurnServiceAppliesChannelResponseShapingProfilesAcrossCanonicalChannels(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{
		responses: []transport.ChatTurnResponse{
			{
				WorkspaceID: "ws1",
				TaskClass:   "chat",
				Provider:    "openai",
				ModelKey:    "gpt-5",
				Items: []transport.ChatTurnItem{{
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: `{"type":"assistant_message","content":"App reply."}`,
				}},
			},
			{
				WorkspaceID: "ws1",
				TaskClass:   "chat",
				Provider:    "openai",
				ModelKey:    "gpt-5",
				Items: []transport.ChatTurnItem{{
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: `{"type":"assistant_message","content":"Message reply."}`,
				}},
			},
			{
				WorkspaceID: "ws1",
				TaskClass:   "chat",
				Provider:    "openai",
				ModelKey:    "gpt-5",
				Items: []transport.ChatTurnItem{{
					Type:    "assistant_message",
					Role:    "assistant",
					Status:  "completed",
					Content: `{"type":"assistant_message","content":"Voice reply."}`,
				}},
			},
		},
	}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	channelCases := []struct {
		channelID   string
		wantProfile string
	}{
		{channelID: "app", wantProfile: "app.default"},
		{channelID: "message", wantProfile: "message.compact"},
		{channelID: "voice", wantProfile: "voice.spoken"},
	}

	for _, tc := range channelCases {
		response, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Channel: transport.ChatTurnChannelContext{
				ChannelID: tc.channelID,
			},
			Items: []transport.ChatTurnItem{{
				Type:    "user_message",
				Role:    "user",
				Status:  "completed",
				Content: "Hello",
			}},
		}, "corr-response-shaping-"+tc.channelID, nil)
		if err != nil {
			t.Fatalf("chat turn %s: %v", tc.channelID, err)
		}
		if len(response.Items) != 1 {
			t.Fatalf("expected one assistant item for %s, got %+v", tc.channelID, response.Items)
		}
		profile := strings.TrimSpace(fmt.Sprintf("%v", response.Items[0].Metadata.AsMap()["response_shaping_profile"]))
		if profile != tc.wantProfile {
			t.Fatalf("expected response_shaping_profile %q for %s, got %q (metadata=%+v)", tc.wantProfile, tc.channelID, profile, response.Items[0].Metadata)
		}
	}

	if len(model.requests) != len(channelCases) {
		t.Fatalf("expected %d model requests, got %d", len(channelCases), len(model.requests))
	}
	promptExpectations := []struct {
		index               int
		wantProfileFragment string
	}{
		{index: 0, wantProfileFragment: "app.default (app)"},
		{index: 1, wantProfileFragment: "message.compact (message)"},
		{index: 2, wantProfileFragment: "voice.spoken (voice)"},
	}
	for _, expectation := range promptExpectations {
		prompt := model.requests[expectation.index].SystemPrompt
		if !strings.Contains(prompt, "Response-shaping channel profile:") {
			t.Fatalf("expected response-shaping section in prompt[%d], got %q", expectation.index, prompt)
		}
		if !strings.Contains(prompt, expectation.wantProfileFragment) {
			t.Fatalf("expected profile fragment %q in prompt[%d], got %q", expectation.wantProfileFragment, expectation.index, prompt)
		}
	}
}

func TestUnifiedTurnServiceHistoryPaginationUsesCursor(t *testing.T) {
	container := newUnifiedTurnTestContainer(t)
	model := &unifiedModelChatStub{responses: []transport.ChatTurnResponse{
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"assistant_message","content":"First reply"}`,
			}},
		},
		{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []transport.ChatTurnItem{{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: `{"type":"assistant_message","content":"Second reply"}`,
			}},
		},
	}}
	service := newUnifiedTurnTestService(t, container, model, &unifiedAgentStub{}, &unifiedUIStatusStub{}, nil)

	for _, turn := range []struct {
		message       string
		correlationID string
	}{
		{message: "first", correlationID: "corr-history-1"},
		{message: "second", correlationID: "corr-history-2"},
	} {
		if _, err := service.ChatTurn(context.Background(), transport.ChatTurnRequest{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Items: []transport.ChatTurnItem{{
				Type:    "user_message",
				Role:    "user",
				Status:  "completed",
				Content: turn.message,
			}},
		}, turn.correlationID, nil); err != nil {
			t.Fatalf("chat turn %s: %v", turn.correlationID, err)
		}
	}

	page1, err := service.QueryChatTurnHistory(context.Background(), transport.ChatTurnHistoryRequest{
		WorkspaceID: "ws1",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("query history page 1: %v", err)
	}
	if len(page1.Items) != 1 || !page1.HasMore {
		t.Fatalf("expected first page to include one item with has_more=true, got %+v", page1)
	}
	if strings.TrimSpace(page1.NextCursorCreatedAt) == "" || strings.TrimSpace(page1.NextCursorItemID) == "" {
		t.Fatalf("expected non-empty pagination cursor in page 1, got %+v", page1)
	}

	page2, err := service.QueryChatTurnHistory(context.Background(), transport.ChatTurnHistoryRequest{
		WorkspaceID:     "ws1",
		BeforeCreatedAt: page1.NextCursorCreatedAt,
		BeforeItemID:    page1.NextCursorItemID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("query history page 2: %v", err)
	}
	if len(page2.Items) == 0 {
		t.Fatalf("expected second page to include at least one item")
	}
	if page2.Items[0].RecordID == page1.Items[0].RecordID {
		t.Fatalf("expected second page item to differ from first page item")
	}
}
