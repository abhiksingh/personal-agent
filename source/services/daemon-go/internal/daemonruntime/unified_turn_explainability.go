package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

type modelRouteExplainabilityService interface {
	ExplainModelRoute(ctx context.Context, request transport.ModelRouteExplainRequest) (transport.ModelRouteExplainResponse, error)
}

func (s *UnifiedTurnService) ExplainChatTurn(
	ctx context.Context,
	request transport.ChatTurnExplainRequest,
) (transport.ChatTurnExplainResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := normalizeTaskClass(request.TaskClass)
	requestedBy := strings.TrimSpace(request.RequestedByActorID)
	subject := strings.TrimSpace(request.SubjectActorID)
	actingAs := strings.TrimSpace(request.ActingAsActorID)
	if requestedBy == "" && actingAs != "" {
		requestedBy = actingAs
	}
	if actingAs == "" && requestedBy != "" {
		actingAs = requestedBy
	}
	if requestedBy == "" && actingAs == "" {
		requestedBy = "assistant"
		actingAs = "assistant"
	}

	channel := transport.ChatTurnChannelContext{
		ChannelID:   normalizeTurnChannelID(request.Channel.ChannelID),
		ConnectorID: strings.TrimSpace(request.Channel.ConnectorID),
		ThreadID:    strings.TrimSpace(request.Channel.ThreadID),
	}

	tools, err := s.resolveAvailableTools(ctx, workspace)
	if err != nil {
		return transport.ChatTurnExplainResponse{}, fmt.Errorf("resolve tool catalog: %w", err)
	}
	registry, err := s.buildToolSchemaRegistry(ctx, transport.ChatTurnRequest{
		WorkspaceID:        workspace,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    actingAs,
		Channel:            channel,
	}, workspace, tools)
	if err != nil {
		return transport.ChatTurnExplainResponse{}, fmt.Errorf("build tool schema registry: %w", err)
	}
	toolCatalog := registry.allToolCatalogEntries()
	policyDecisions := make([]transport.ChatTurnToolPolicyDecision, 0, len(registry.Entries))
	for _, entry := range registry.Entries {
		policyDecisions = append(policyDecisions, transport.ChatTurnToolPolicyDecision{
			ToolName:      strings.TrimSpace(entry.Tool.Name),
			CapabilityKey: strings.TrimSpace(entry.CapabilityKey),
			Decision:      string(entry.Policy.Decision),
			Reason:        strings.TrimSpace(entry.Policy.Reason),
		})
	}
	sort.Slice(policyDecisions, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(policyDecisions[i].ToolName)) < strings.ToLower(strings.TrimSpace(policyDecisions[j].ToolName))
	})

	route := transport.ModelRouteExplainResponse{
		WorkspaceID:      workspace,
		TaskClass:        taskClass,
		PrincipalActorID: routeExplainPrincipal(subject, actingAs, requestedBy),
		SelectedProvider: "",
		SelectedModelKey: "",
		SelectedSource:   "none",
		Summary:          "model route explainability unavailable",
		Explanations: []string{
			"The active chat model service does not expose route explainability in this runtime configuration.",
		},
		ReasonCodes:   []string{"route_explain_unavailable"},
		Decisions:     []transport.ModelRouteDecision{},
		FallbackChain: []transport.ModelRouteFallbackDecision{},
	}
	if explainer, ok := s.modelChat.(modelRouteExplainabilityService); ok {
		route, err = explainer.ExplainModelRoute(ctx, transport.ModelRouteExplainRequest{
			WorkspaceID:      workspace,
			TaskClass:        taskClass,
			PrincipalActorID: routeExplainPrincipal(subject, actingAs, requestedBy),
		})
		if err != nil {
			return transport.ChatTurnExplainResponse{}, fmt.Errorf("explain selected model route: %w", err)
		}
	}

	return transport.ChatTurnExplainResponse{
		WorkspaceID:        workspace,
		TaskClass:          taskClass,
		RequestedByActorID: requestedBy,
		SubjectActorID:     subject,
		ActingAsActorID:    actingAs,
		Channel:            channel,
		ContractVersion:    transport.ChatTurnExplainContractVersionV1,
		SelectedRoute:      route,
		ToolCatalog:        toolCatalog,
		PolicyDecisions:    policyDecisions,
	}, nil
}

func modelToolInputSchema(tool modelToolDefinition) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0)
	argumentNames := make([]string, 0, len(tool.Arguments))
	for key := range tool.Arguments {
		argumentNames = append(argumentNames, key)
	}
	sort.Strings(argumentNames)
	for _, argumentName := range argumentNames {
		spec := tool.Arguments[argumentName]
		field := map[string]any{
			"type": strings.TrimSpace(spec.Type),
		}
		if strings.TrimSpace(spec.Description) != "" {
			field["description"] = strings.TrimSpace(spec.Description)
		}
		if len(spec.EnumOptions) > 0 {
			field["enum"] = append([]string(nil), spec.EnumOptions...)
		}
		properties[argumentName] = field
		if spec.Required {
			required = append(required, argumentName)
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func routeExplainPrincipal(subject string, actingAs string, requestedBy string) string {
	if strings.TrimSpace(actingAs) != "" {
		return strings.TrimSpace(actingAs)
	}
	if strings.TrimSpace(subject) != "" {
		return strings.TrimSpace(subject)
	}
	return strings.TrimSpace(requestedBy)
}
