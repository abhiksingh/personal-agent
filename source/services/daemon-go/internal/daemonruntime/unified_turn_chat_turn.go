package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

type chatTurnPhaseContext struct {
	request         transport.ChatTurnRequest
	workspace       string
	taskClass       string
	correlationID   string
	responseShaping resolvedResponseShapingPolicy
	assembly        contextAssembly
	toolRegistry    modelToolSchemaRegistry
	toolCatalog     []transport.ChatTurnToolCatalogEntry
	plannerPrompt   string
}

type chatTurnPhaseState struct {
	plannerConversationItems []transport.ChatTurnItem
	generatedItems           []transport.ChatTurnItem
	taskRunReference         transport.ChatTurnTaskRunCorrelation
	latestToolResult         transport.ChatTurnItem
	latestApproval           transport.ChatTurnItem
	toolCallCount            int
	loopStopReason           string
	lastPlannerText          string
	plannerRepairAttempts    int
	provider                 string
	modelKey                 string
}

func (s *UnifiedTurnService) ChatTurn(
	ctx context.Context,
	request transport.ChatTurnRequest,
	correlationID string,
	onToken func(delta string),
) (transport.ChatTurnResponse, error) {
	phase, state, err := s.prepareChatTurnPhases(ctx, request, correlationID)
	if err != nil {
		return transport.ChatTurnResponse{}, err
	}

	earlyResponse, err := s.runPlannerLoopPhase(ctx, phase, &state, onToken)
	if err != nil {
		return transport.ChatTurnResponse{}, err
	}
	if earlyResponse != nil {
		return *earlyResponse, nil
	}

	assistantContent, assistantMetadata := s.synthesizeFinalAssistantPhase(ctx, phase, &state, onToken)
	return s.assembleFinalTurnResponsePhase(ctx, phase, &state, assistantContent, assistantMetadata)
}

func (s *UnifiedTurnService) prepareChatTurnPhases(
	ctx context.Context,
	request transport.ChatTurnRequest,
	correlationID string,
) (chatTurnPhaseContext, chatTurnPhaseState, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := normalizeTaskClass(request.TaskClass)
	items := normalizeInputTurnItems(request.Items)
	if len(items) == 0 {
		return chatTurnPhaseContext{}, chatTurnPhaseState{}, fmt.Errorf("chat turn items are required")
	}

	request.WorkspaceID = workspace
	request.TaskClass = taskClass
	request.Channel.ChannelID = normalizeTurnChannelID(request.Channel.ChannelID)
	request.Items = items

	personaPolicy, err := s.resolvePersonaPolicy(ctx, transport.ChatPersonaPolicyRequest{
		WorkspaceID:      workspace,
		PrincipalActorID: strings.TrimSpace(request.ActingAsActorID),
		ChannelID:        request.Channel.ChannelID,
	})
	if err != nil {
		return chatTurnPhaseContext{}, chatTurnPhaseState{}, err
	}
	responseShaping := resolveResponseShapingPolicy(personaPolicy, request.Channel.ChannelID)

	assembly, err := s.assembleContext(ctx, request)
	if err != nil {
		return chatTurnPhaseContext{}, chatTurnPhaseState{}, err
	}

	tools, err := s.resolveAvailableTools(ctx, workspace)
	if err != nil {
		return chatTurnPhaseContext{}, chatTurnPhaseState{}, err
	}
	toolRegistry, err := s.buildToolSchemaRegistry(ctx, request, workspace, tools)
	if err != nil {
		return chatTurnPhaseContext{}, chatTurnPhaseState{}, err
	}
	toolCatalog := toolRegistry.plannerToolCatalogEntries()
	plannerTools := toolRegistry.plannerTools()
	plannerPrompt := buildPlannerPrompt(request.SystemPrompt, responseShaping, assembly, plannerTools, toolRegistry.Version)

	phase := chatTurnPhaseContext{
		request:         request,
		workspace:       workspace,
		taskClass:       taskClass,
		correlationID:   correlationID,
		responseShaping: responseShaping,
		assembly:        assembly,
		toolRegistry:    toolRegistry,
		toolCatalog:     toolCatalog,
		plannerPrompt:   plannerPrompt,
	}
	state := chatTurnPhaseState{
		plannerConversationItems: prepareItemsForModel(request.Items),
		generatedItems:           make([]transport.ChatTurnItem, 0, 16),
		taskRunReference:         transport.ChatTurnTaskRunCorrelation{Available: false, Source: "none"},
	}
	return phase, state, nil
}

func (s *UnifiedTurnService) runPlannerLoopPhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	onToken func(delta string),
) (*transport.ChatTurnResponse, error) {
	for {
		if state.toolCallCount >= unifiedTurnMaxToolCalls {
			state.loopStopReason = "tool_call_limit_reached"
			return nil, nil
		}

		planner, plannerText, plannerStructured, err := s.requestPlannerDirectivePhase(ctx, phase, state)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if response, handled := s.remediationResponseForModelRouteFailure(
				ctx,
				phase.request,
				phase.workspace,
				phase.taskClass,
				phase.correlationID,
				state.taskRunReference,
				state.generatedItems,
				phase.assembly,
				err,
			); handled {
				return &response, nil
			}
			return nil, err
		}

		response, handled, handleErr := s.handlePlannerNonToolPhase(
			ctx,
			phase,
			state,
			planner,
			plannerText,
			plannerStructured,
			onToken,
		)
		if handleErr != nil {
			return nil, handleErr
		}
		if handled {
			if response != nil {
				return response, nil
			}
			return nil, nil
		}

		if err := s.executeToolDirectivePhase(ctx, phase, state, planner); err != nil {
			return nil, err
		}
		if strings.EqualFold(strings.TrimSpace(state.loopStopReason), "awaiting_approval") {
			return nil, nil
		}
	}
}

func (s *UnifiedTurnService) requestPlannerDirectivePhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
) (plannerDirective, string, bool, error) {
	plannerResponse, err := s.modelChat.ChatTurn(
		ctx,
		s.buildPlannerRequestForPhase(phase, state),
		phase.correlationID,
		nil,
	)
	if err != nil {
		return plannerDirective{}, "", false, err
	}
	s.mergeModelRouteSelectionPhase(state, plannerResponse)

	plannerText := strings.TrimSpace(assistantMessageFromItems(plannerResponse.Items))
	state.lastPlannerText = plannerText
	planner, plannerStructured := parsePlannerDirective(plannerText)

	repairAttemptsForCall := 0
	for !plannerStructured &&
		state.toolCallCount == 0 &&
		strings.TrimSpace(plannerText) != "" &&
		repairAttemptsForCall < unifiedTurnPlannerRepairMaxRetries {
		repairResponse, repairErr := s.requestPlannerRepair(
			ctx,
			phase.request,
			phase.workspace,
			phase.taskClass,
			state.plannerConversationItems,
			phase.toolCatalog,
			phase.plannerPrompt,
			plannerText,
			phase.correlationID,
		)
		if repairErr != nil {
			return plannerDirective{}, "", false, repairErr
		}
		s.mergeModelRouteSelectionPhase(state, repairResponse)
		plannerText = strings.TrimSpace(assistantMessageFromItems(repairResponse.Items))
		state.lastPlannerText = plannerText
		planner, plannerStructured = parsePlannerDirective(plannerText)
		repairAttemptsForCall++
		state.plannerRepairAttempts++
	}

	return planner, plannerText, plannerStructured, nil
}

func (s *UnifiedTurnService) buildPlannerRequestForPhase(
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
) transport.ChatTurnRequest {
	request := phase.request
	request.SystemPrompt = phase.plannerPrompt
	request.ToolCatalog = phase.toolCatalog
	request.Items = append([]transport.ChatTurnItem{}, state.plannerConversationItems...)
	return request
}

func (s *UnifiedTurnService) mergeModelRouteSelectionPhase(
	state *chatTurnPhaseState,
	modelResponse transport.ChatTurnResponse,
) {
	if provider := strings.TrimSpace(modelResponse.Provider); provider != "" {
		state.provider = provider
	}
	if modelKey := strings.TrimSpace(modelResponse.ModelKey); modelKey != "" {
		state.modelKey = modelKey
	}
	if !state.taskRunReference.Available && modelResponse.TaskRunCorrelation.Available {
		state.taskRunReference = modelResponse.TaskRunCorrelation
	}
}

func (s *UnifiedTurnService) handlePlannerNonToolPhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	planner plannerDirective,
	plannerText string,
	plannerStructured bool,
	onToken func(delta string),
) (*transport.ChatTurnResponse, bool, error) {
	if plannerStructured && strings.EqualFold(strings.TrimSpace(planner.Type), "tool_call") {
		return nil, false, nil
	}

	assistantContent := resolvePlannerAssistantContentPhase(state.toolCallCount, plannerText, planner, plannerStructured)
	if assistantContent != "" {
		if onToken != nil {
			streamedContent, streamedProvider, streamedModelKey := s.recoverModelOnlyAssistantContent(
				ctx,
				phase.request,
				phase.workspace,
				phase.taskClass,
				phase.responseShaping,
				phase.assembly,
				state.plannerConversationItems,
				phase.correlationID,
				onToken,
			)
			if streamedContent != "" {
				assistantContent = streamedContent
			}
			if streamedProvider != "" {
				state.provider = streamedProvider
			}
			if streamedModelKey != "" {
				state.modelKey = streamedModelKey
			}
		}
		orchestrationMode := "model_only"
		if state.toolCallCount > 0 {
			orchestrationMode = "model_tool_loop"
		}
		assistantMetadata := withResponseShapingMetadata(map[string]any{
			"orchestration":           orchestrationMode,
			"planner_repair_attempts": state.plannerRepairAttempts,
		}, phase.responseShaping)
		response, err := s.assembleFinalTurnResponsePhase(ctx, phase, state, assistantContent, assistantMetadata)
		if err != nil {
			return nil, true, err
		}
		return &response, true, nil
	}

	if state.toolCallCount == 0 {
		stopReason := "planner_no_action"
		if !plannerStructured && strings.TrimSpace(plannerText) != "" {
			stopReason = "planner_output_invalid"
		}
		fallbackContent, fallbackProvider, fallbackModelKey := s.synthesizeModelOnlyFallbackPhase(
			ctx,
			phase,
			state,
			onToken,
		)
		if fallbackProvider != "" {
			state.provider = fallbackProvider
		}
		if fallbackModelKey != "" {
			state.modelKey = fallbackModelKey
		}
		assistantMetadata := withResponseShapingMetadata(map[string]any{
			"orchestration":           "model_only",
			"stop_reason":             stopReason,
			"planner_repair_attempts": state.plannerRepairAttempts,
			"remediation":             plannerRemediationHint(stopReason),
		}, phase.responseShaping)
		response, err := s.assembleFinalTurnResponsePhase(ctx, phase, state, fallbackContent, assistantMetadata)
		if err != nil {
			return nil, true, err
		}
		return &response, true, nil
	}

	state.loopStopReason = "planner_no_action"
	return nil, true, nil
}

func resolvePlannerAssistantContentPhase(
	toolCallCount int,
	plannerText string,
	planner plannerDirective,
	plannerStructured bool,
) string {
	if plannerStructured && strings.EqualFold(strings.TrimSpace(planner.Type), "assistant_message") {
		return strings.TrimSpace(planner.Content)
	}
	if !plannerStructured && toolCallCount > 0 && strings.TrimSpace(plannerText) != "" {
		return strings.TrimSpace(plannerText)
	}
	return ""
}

func (s *UnifiedTurnService) synthesizeModelOnlyFallbackPhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	onToken func(delta string),
) (string, string, string) {
	fallbackContent, fallbackProvider, fallbackModelKey := s.recoverModelOnlyAssistantContent(
		ctx,
		phase.request,
		phase.workspace,
		phase.taskClass,
		phase.responseShaping,
		phase.assembly,
		state.plannerConversationItems,
		phase.correlationID,
		onToken,
	)
	assistantContent := strings.TrimSpace(fallbackContent)
	if assistantContent == "" {
		assistantContent = "I couldn't determine a valid next action from this request."
	}
	return assistantContent, fallbackProvider, fallbackModelKey
}

func (s *UnifiedTurnService) executeToolDirectivePhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	planner plannerDirective,
) error {
	toolCallItem, toolResultItem, approvalItem, taskRunReference, err := s.executeToolDirective(
		ctx,
		phase.request,
		phase.workspace,
		phase.correlationID,
		phase.toolRegistry,
		planner,
	)
	if err != nil {
		return err
	}
	state.generatedItems = append(state.generatedItems, toolCallItem, toolResultItem)
	if strings.TrimSpace(approvalItem.Type) != "" {
		state.generatedItems = append(state.generatedItems, approvalItem)
	}
	if taskRunReference.Available {
		state.taskRunReference = taskRunReference
	}
	state.latestToolResult = toolResultItem
	state.latestApproval = approvalItem
	state.toolCallCount++
	state.plannerConversationItems = append(state.plannerConversationItems, toolResultAsPlannerConversationItems(toolCallItem, toolResultItem, approvalItem)...)
	if strings.EqualFold(strings.TrimSpace(toolResultItem.Status), "awaiting_approval") {
		state.loopStopReason = "awaiting_approval"
	}
	return nil
}

func (s *UnifiedTurnService) synthesizeFinalAssistantPhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	onToken func(delta string),
) (string, map[string]any) {
	responsePrompt := buildResponsePrompt(phase.request.SystemPrompt, phase.responseShaping, phase.assembly)
	responseItems := append([]transport.ChatTurnItem{}, state.plannerConversationItems...)
	responseItems = append(responseItems, transport.ChatTurnItem{
		Type:    "user_message",
		Role:    "user",
		Status:  "completed",
		Content: "Using the tool execution records above, provide the final assistant response.",
	})
	responseRequest := phase.request
	responseRequest.SystemPrompt = responsePrompt
	responseRequest.ToolCatalog = nil
	responseRequest.Items = responseItems

	responseModel, responseErr := s.modelChat.ChatTurn(ctx, responseRequest, phase.correlationID, onToken)

	assistantContent := ""
	if responseErr == nil {
		s.mergeModelRouteSelectionPhase(state, responseModel)
		assistantContent = strings.TrimSpace(assistantMessageFromItems(responseModel.Items))
	}
	if assistantContent == "" {
		assistantContent = fallbackAssistantSummary(state.latestToolResult, state.latestApproval)
		if state.loopStopReason == "tool_call_limit_reached" {
			assistantContent = fmt.Sprintf("I reached the maximum of %d tool calls for this turn. %s", unifiedTurnMaxToolCalls, assistantContent)
		}
	}

	assistantMetadata := map[string]any{
		"orchestration":           "model_tool_model_loop",
		"tool_call_count":         state.toolCallCount,
		"planner_repair_attempts": state.plannerRepairAttempts,
	}
	if stopReason := strings.TrimSpace(state.loopStopReason); stopReason != "" {
		assistantMetadata["stop_reason"] = stopReason
		if remediation := plannerRemediationHint(stopReason); remediation != nil {
			assistantMetadata["remediation"] = remediation
		}
	}
	return assistantContent, withResponseShapingMetadata(assistantMetadata, phase.responseShaping)
}

func (s *UnifiedTurnService) assembleFinalTurnResponsePhase(
	ctx context.Context,
	phase chatTurnPhaseContext,
	state *chatTurnPhaseState,
	assistantContent string,
	assistantMetadata map[string]any,
) (transport.ChatTurnResponse, error) {
	generatedItems := append([]transport.ChatTurnItem{}, state.generatedItems...)
	generatedItems = append(generatedItems, transport.ChatTurnItem{
		ItemID:   mustLocalRandomID("assistant"),
		Type:     "assistant_message",
		Role:     "assistant",
		Status:   "completed",
		Content:  strings.TrimSpace(assistantContent),
		Metadata: transport.ChatTurnItemMetadataFromMap(assistantMetadata),
	})

	response := transport.ChatTurnResponse{
		WorkspaceID:        phase.workspace,
		TaskClass:          phase.taskClass,
		Provider:           strings.TrimSpace(state.provider),
		ModelKey:           strings.TrimSpace(state.modelKey),
		CorrelationID:      phase.correlationID,
		Channel:            phase.request.Channel,
		Items:              generatedItems,
		TaskRunCorrelation: state.taskRunReference,
	}
	if strings.TrimSpace(response.TaskRunCorrelation.Source) == "" {
		response.TaskRunCorrelation = transport.ChatTurnTaskRunCorrelation{Available: false, Source: "none"}
	}
	if err := s.persistTurnItems(ctx, phase.request, response, latestUserMessageItem(phase.request.Items)); err != nil {
		return transport.ChatTurnResponse{}, err
	}
	s.recordContextSample(ctx, phase.workspace, phase.taskClass, response.ModelKey, phase.assembly, state.lastPlannerText, strings.TrimSpace(assistantContent))
	return response, nil
}
