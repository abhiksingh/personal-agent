package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func buildPlannerPrompt(basePrompt string, policy resolvedResponseShapingPolicy, assembly contextAssembly, tools []modelToolDefinition, toolSchemaVersion string) string {
	lines := []string{
		strings.TrimSpace(basePrompt),
		"",
		"Unified-turn orchestration mode:",
		"Return exactly one JSON object with no markdown.",
		"Valid planner object shapes:",
		`{"type":"assistant_message","content":"<reply>"}`,
		`{"type":"tool_call","tool_name":"<tool>","arguments":{...}}`,
		"",
		"Persona style prompt:",
		strings.TrimSpace(policy.StylePrompt),
		"",
		"Persona guardrails:",
	}
	for _, guardrail := range policy.Guardrails {
		lines = append(lines, "- "+strings.TrimSpace(guardrail))
	}
	lines = append(
		lines,
		"",
		"Response-shaping channel profile:",
		fmt.Sprintf("%s (%s)", strings.TrimSpace(policy.ProfileID), strings.TrimSpace(policy.ChannelID)),
		"",
		"Channel response instructions:",
	)
	for _, instruction := range policy.ChannelInstructions {
		lines = append(lines, "- "+strings.TrimSpace(instruction))
	}
	if strings.TrimSpace(assembly.Summary) != "" {
		lines = append(lines, "", "Retrieved context summary:", strings.TrimSpace(assembly.Summary))
	}
	if strings.TrimSpace(toolSchemaVersion) != "" {
		lines = append(lines, "", "Tool schema registry version:", strings.TrimSpace(toolSchemaVersion))
	}
	lines = append(lines, "", "Available tools (JSON schema):", serializeModelTools(tools))
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func buildPlannerRepairPrompt(plannerPrompt string) string {
	lines := []string{
		strings.TrimSpace(plannerPrompt),
		"",
		"Planner repair mode:",
		"The previous planner response was invalid.",
		"Return exactly one valid planner JSON object that matches the allowed shapes.",
		"Do not return markdown or explanatory prose.",
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *UnifiedTurnService) requestPlannerRepair(
	ctx context.Context,
	request transport.ChatTurnRequest,
	workspace string,
	taskClass string,
	plannerConversationItems []transport.ChatTurnItem,
	toolCatalog []transport.ChatTurnToolCatalogEntry,
	plannerPrompt string,
	invalidPlannerText string,
	correlationID string,
) (transport.ChatTurnResponse, error) {
	repairItems := append([]transport.ChatTurnItem{}, plannerConversationItems...)
	repairItems = append(repairItems, transport.ChatTurnItem{
		Type:    "user_message",
		Role:    "user",
		Status:  "completed",
		Content: fmt.Sprintf("Repair this invalid planner output and return valid planner JSON only:\n%s", strings.TrimSpace(invalidPlannerText)),
	})
	return s.modelChat.ChatTurn(ctx, transport.ChatTurnRequest{
		WorkspaceID:        workspace,
		TaskClass:          taskClass,
		RequestedByActorID: strings.TrimSpace(request.RequestedByActorID),
		SubjectActorID:     strings.TrimSpace(request.SubjectActorID),
		ActingAsActorID:    strings.TrimSpace(request.ActingAsActorID),
		ProviderOverride:   strings.TrimSpace(request.ProviderOverride),
		ModelOverride:      strings.TrimSpace(request.ModelOverride),
		SystemPrompt:       buildPlannerRepairPrompt(plannerPrompt),
		Channel:            request.Channel,
		ToolCatalog:        toolCatalog,
		Items:              repairItems,
	}, correlationID, nil)
}

func (s *UnifiedTurnService) remediationResponseForModelRouteFailure(
	ctx context.Context,
	request transport.ChatTurnRequest,
	workspace string,
	taskClass string,
	correlationID string,
	taskRunReference transport.ChatTurnTaskRunCorrelation,
	generatedItems []transport.ChatTurnItem,
	assembly contextAssembly,
	plannerErr error,
) (transport.ChatTurnResponse, bool) {
	remediation := modelRouteRemediationHint(plannerErr)
	if remediation == nil {
		return transport.ChatTurnResponse{}, false
	}
	assistantContent := remediationSummary(remediation, "No ready chat route is configured for tool orchestration.")
	generated := append([]transport.ChatTurnItem{}, generatedItems...)
	generated = append(generated, transport.ChatTurnItem{
		ItemID:  mustLocalRandomID("assistant"),
		Type:    "assistant_message",
		Role:    "assistant",
		Status:  "completed",
		Content: assistantContent,
		Metadata: transport.ChatTurnItemMetadataFromMap(map[string]any{
			"orchestration": "model_only",
			"stop_reason":   "model_route_unavailable",
			"remediation":   remediation,
		}),
	})
	response := transport.ChatTurnResponse{
		WorkspaceID:        workspace,
		TaskClass:          taskClass,
		Provider:           strings.TrimSpace(request.ProviderOverride),
		ModelKey:           strings.TrimSpace(request.ModelOverride),
		CorrelationID:      correlationID,
		Channel:            request.Channel,
		Items:              generated,
		TaskRunCorrelation: taskRunReference,
	}
	if strings.TrimSpace(response.TaskRunCorrelation.Source) == "" {
		response.TaskRunCorrelation = transport.ChatTurnTaskRunCorrelation{Available: false, Source: "none"}
	}
	if err := s.persistTurnItems(ctx, request, response, latestUserMessageItem(request.Items)); err != nil {
		return transport.ChatTurnResponse{}, false
	}
	s.recordContextSample(ctx, workspace, taskClass, response.ModelKey, assembly, strings.TrimSpace(plannerErr.Error()), assistantContent)
	return response, true
}

func buildResponsePrompt(basePrompt string, policy resolvedResponseShapingPolicy, assembly contextAssembly) string {
	lines := []string{
		strings.TrimSpace(basePrompt),
		"",
		"Response synthesis mode:",
		"Use tool output facts to answer the user directly and safely.",
		"If execution is blocked or approval is required, explain exactly what is needed next.",
		"",
		"Persona style prompt:",
		strings.TrimSpace(policy.StylePrompt),
		"",
		"Persona guardrails:",
	}
	for _, guardrail := range policy.Guardrails {
		lines = append(lines, "- "+strings.TrimSpace(guardrail))
	}
	lines = append(
		lines,
		"",
		"Response-shaping channel profile:",
		fmt.Sprintf("%s (%s)", strings.TrimSpace(policy.ProfileID), strings.TrimSpace(policy.ChannelID)),
		"",
		"Channel response instructions:",
	)
	for _, instruction := range policy.ChannelInstructions {
		lines = append(lines, "- "+strings.TrimSpace(instruction))
	}
	if strings.TrimSpace(assembly.Summary) != "" {
		lines = append(lines, "", "Retrieved context summary:", strings.TrimSpace(assembly.Summary))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *UnifiedTurnService) recoverModelOnlyAssistantContent(
	ctx context.Context,
	request transport.ChatTurnRequest,
	workspace string,
	taskClass string,
	policy resolvedResponseShapingPolicy,
	assembly contextAssembly,
	conversationItems []transport.ChatTurnItem,
	correlationID string,
	onToken func(delta string),
) (string, string, string) {
	if s == nil || s.modelChat == nil {
		return "", "", ""
	}
	items := append([]transport.ChatTurnItem{}, conversationItems...)
	if len(items) == 0 {
		items = prepareItemsForModel(request.Items)
	}
	recoveryResponse, err := s.modelChat.ChatTurn(ctx, transport.ChatTurnRequest{
		WorkspaceID:        workspace,
		TaskClass:          taskClass,
		RequestedByActorID: strings.TrimSpace(request.RequestedByActorID),
		SubjectActorID:     strings.TrimSpace(request.SubjectActorID),
		ActingAsActorID:    strings.TrimSpace(request.ActingAsActorID),
		ProviderOverride:   strings.TrimSpace(request.ProviderOverride),
		ModelOverride:      strings.TrimSpace(request.ModelOverride),
		SystemPrompt:       buildResponsePrompt(request.SystemPrompt, policy, assembly),
		Channel:            request.Channel,
		Items:              items,
	}, correlationID, onToken)
	if err != nil {
		return "", "", ""
	}
	return strings.TrimSpace(assistantMessageFromItems(recoveryResponse.Items)),
		strings.TrimSpace(recoveryResponse.Provider),
		strings.TrimSpace(recoveryResponse.ModelKey)
}

func serializeModelTools(tools []modelToolDefinition) string {
	if len(tools) == 0 {
		return "[]"
	}
	serialized := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		properties := map[string]any{}
		required := make([]string, 0)
		keys := make([]string, 0, len(tool.Arguments))
		for key := range tool.Arguments {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			spec := tool.Arguments[key]
			field := map[string]any{"type": spec.Type}
			if strings.TrimSpace(spec.Description) != "" {
				field["description"] = strings.TrimSpace(spec.Description)
			}
			if len(spec.EnumOptions) > 0 {
				field["enum"] = append([]string(nil), spec.EnumOptions...)
			}
			properties[key] = field
			if spec.Required {
				required = append(required, key)
			}
		}
		serialized = append(serialized, map[string]any{
			"name":            tool.Name,
			"description":     tool.Description,
			"capability_keys": append([]string(nil), tool.CapabilityKeys...),
			"input_schema": map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		})
	}
	payload, err := json.Marshal(serialized)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func parsePlannerDirective(raw string) (plannerDirective, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return plannerDirective{}, false
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	var directive plannerDirective
	if err := json.Unmarshal([]byte(trimmed), &directive); err != nil {
		return plannerDirective{}, false
	}
	directive.Type = strings.ToLower(strings.TrimSpace(directive.Type))
	if directive.Type == "" {
		return plannerDirective{}, false
	}
	if directive.Type == "assistant_message" {
		return directive, true
	}
	if directive.Type == "tool_call" && strings.TrimSpace(directive.ToolName) != "" {
		if directive.Arguments == nil {
			directive.Arguments = map[string]any{}
		}
		return directive, true
	}
	return plannerDirective{}, false
}

func requiredStringArgument(arguments map[string]any, key string) (string, error) {
	value, ok := arguments[key]
	if !ok {
		return "", fmt.Errorf("missing required argument %s", key)
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be string", key)
	}
	trimmed := strings.TrimSpace(stringValue)
	if trimmed == "" {
		return "", fmt.Errorf("argument %s must not be empty", key)
	}
	return trimmed, nil
}

func optionalStringArgument(arguments map[string]any, key string) string {
	value, ok := arguments[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}

func optionalIntegerArgument(arguments map[string]any, key string) (int, error) {
	value, ok := arguments[key]
	if !ok || value == nil {
		return 0, nil
	}
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, fmt.Errorf("argument %s must be integer", key)
		}
		return int(typed), nil
	default:
		return 0, fmt.Errorf("argument %s must be integer", key)
	}
}

func firstCapabilityKey(tool modelToolDefinition) string {
	if len(tool.CapabilityKeys) == 0 {
		return ""
	}
	return strings.TrimSpace(tool.CapabilityKeys[0])
}

func normalizeInputTurnItems(items []transport.ChatTurnItem) []transport.ChatTurnItem {
	if len(items) == 0 {
		return []transport.ChatTurnItem{}
	}
	cloned := make([]transport.ChatTurnItem, 0, len(items))
	for _, item := range items {
		typeValue := strings.ToLower(strings.TrimSpace(item.Type))
		roleValue := strings.ToLower(strings.TrimSpace(item.Role))
		if typeValue == "" {
			continue
		}
		if roleValue == "" {
			switch typeValue {
			case "user_message":
				roleValue = "user"
			case "assistant_message":
				roleValue = "assistant"
			case "system_message":
				roleValue = "system"
			}
		}
		item.Type = typeValue
		item.Role = roleValue
		if strings.TrimSpace(item.Status) == "" {
			item.Status = "completed"
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func prepareItemsForModel(items []transport.ChatTurnItem) []transport.ChatTurnItem {
	prepared := make([]transport.ChatTurnItem, 0, len(items))
	for _, item := range items {
		typeValue := strings.ToLower(strings.TrimSpace(item.Type))
		switch typeValue {
		case "user_message", "assistant_message", "system_message":
			if strings.TrimSpace(item.Content) == "" {
				continue
			}
			prepared = append(prepared, transport.ChatTurnItem{
				Type:    typeValue,
				Role:    strings.ToLower(strings.TrimSpace(item.Role)),
				Status:  "completed",
				Content: strings.TrimSpace(item.Content),
			})
		}
	}
	return prepared
}

func assistantMessageFromItems(items []transport.ChatTurnItem) string {
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		if strings.ToLower(strings.TrimSpace(item.Type)) != "assistant_message" {
			continue
		}
		if content := strings.TrimSpace(item.Content); content != "" {
			return content
		}
	}
	return ""
}

func latestUserMessageItem(items []transport.ChatTurnItem) transport.ChatTurnItem {
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		if strings.ToLower(strings.TrimSpace(item.Type)) != "user_message" {
			continue
		}
		if strings.TrimSpace(item.Content) == "" {
			continue
		}
		item.ItemID = mustLocalRandomID("item")
		item.Status = "completed"
		return item
	}
	return transport.ChatTurnItem{}
}
