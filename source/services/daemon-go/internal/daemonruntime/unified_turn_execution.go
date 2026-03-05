package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *UnifiedTurnService) executeToolDirective(
	ctx context.Context,
	request transport.ChatTurnRequest,
	workspace string,
	correlationID string,
	registry modelToolSchemaRegistry,
	planner plannerDirective,
) (
	transport.ChatTurnItem,
	transport.ChatTurnItem,
	transport.ChatTurnItem,
	transport.ChatTurnTaskRunCorrelation,
	error,
) {
	toolEntry, found := registry.findTool(planner.ToolName)
	toolCallID := mustLocalRandomID("toolcall")
	toolCallItem := transport.ChatTurnItem{
		ItemID:     mustLocalRandomID("item"),
		Type:       "tool_call",
		Status:     "started",
		ToolName:   strings.TrimSpace(planner.ToolName),
		ToolCallID: toolCallID,
		Arguments:  cloneAnyMap(planner.Arguments),
	}
	toolResultItem := transport.ChatTurnItem{
		ItemID:     mustLocalRandomID("item"),
		Type:       "tool_result",
		Status:     "failed",
		ToolName:   strings.TrimSpace(planner.ToolName),
		ToolCallID: toolCallID,
		Output:     map[string]any{},
	}
	approvalItem := transport.ChatTurnItem{}
	taskRunReference := transport.ChatTurnTaskRunCorrelation{Available: false, Source: "none"}

	if !found {
		toolResultItem.ErrorCode = "unsupported_tool"
		toolResultItem.ErrorMessage = fmt.Sprintf("tool %q is not available", strings.TrimSpace(planner.ToolName))
		toolResultItem.Status = "denied"
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}
	toolDef := toolEntry.Tool

	if validationErr := validateToolArgumentsAgainstRegistry(toolEntry, planner.Arguments); validationErr != nil {
		toolResultItem.ErrorCode = "tool_schema_validation_failed"
		toolResultItem.Metadata = transport.ChatTurnItemMetadataFromMap(map[string]any{
			"schema_registry_version": strings.TrimSpace(registry.Version),
			"validation_error":        strings.TrimSpace(validationErr.Error()),
		})
		if typed, ok := validationErr.(modelToolArgumentValidationError); ok {
			toolResultItem.ErrorCode = strings.TrimSpace(typed.Code)
			toolResultItem.Metadata.Set("validation_error_code", strings.TrimSpace(typed.Code))
			toolResultItem.Metadata.Set("validation_argument", strings.TrimSpace(typed.Argument))
			toolResultItem.Metadata.Set("validation_expected", strings.TrimSpace(typed.Expected))
		}
		toolResultItem.ErrorMessage = validationErr.Error()
		toolResultItem.Status = "failed"
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}

	policyDecision := toolEntry.Policy
	policyMetadata := policyDecision.MetadataMap()
	toolCallItem.Status = strings.ToLower(strings.TrimSpace(string(policyDecision.Decision)))
	toolCallItem.Metadata = transport.ChatTurnItemMetadataFromMap(cloneAnyMap(policyMetadata))
	toolResultItem.Metadata = transport.ChatTurnItemMetadataFromMap(cloneAnyMap(policyMetadata))

	if policyDecision.Decision == ToolPolicyDecisionDeny {
		toolResultItem.Status = "denied"
		toolResultItem.ErrorCode = "policy_denied"
		toolResultItem.ErrorMessage = policyDecision.Reason
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}
	if s.agent == nil {
		toolResultItem.Status = "failed"
		toolResultItem.ErrorCode = "tool_executor_unavailable"
		toolResultItem.ErrorMessage = "agent tool executor is not configured"
		toolResultItem.Metadata = mergeToolItemMetadata(
			transport.ChatTurnItemMetadataFromMap(toolFailureRemediationHint(toolDef, toolResultItem.ErrorCode)),
			policyMetadata,
		)
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}

	policyRequiresApproval := policyDecision.Decision == ToolPolicyDecisionRequireApproval
	if toolDef.BuildNativeAction == nil {
		toolResultItem.Status = "failed"
		toolResultItem.ErrorCode = "tool_execution_failed"
		toolResultItem.ErrorMessage = "tool is missing native action builder"
		toolResultItem.Metadata = mergeToolItemMetadata(
			transport.ChatTurnItemMetadataFromMap(toolFailureRemediationHint(toolDef, toolResultItem.ErrorCode)),
			policyMetadata,
		)
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}
	nativeAction, buildErr := toolDef.BuildNativeAction(planner.Arguments)
	if buildErr != nil {
		toolResultItem.Status = "failed"
		toolResultItem.ErrorCode = "invalid_tool_arguments"
		toolResultItem.ErrorMessage = buildErr.Error()
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}
	agentResult, runErr := s.agent.RunAgent(ctx, transport.AgentRunRequest{
		WorkspaceID:        workspace,
		NativeAction:       nativeAction,
		RequestedByActorID: strings.TrimSpace(request.RequestedByActorID),
		SubjectActorID:     strings.TrimSpace(request.SubjectActorID),
		ActingAsActorID:    strings.TrimSpace(request.ActingAsActorID),
		Origin:             executionOriginForTurnChannel(request.Channel.ChannelID),
		CorrelationID:      correlationID,
	})
	if runErr != nil {
		toolResultItem.Status = "failed"
		toolResultItem.ErrorCode = "tool_execution_failed"
		toolResultItem.ErrorMessage = runErr.Error()
		toolResultItem.Metadata = mergeToolItemMetadata(
			transport.ChatTurnItemMetadataFromMap(toolFailureRemediationHint(toolDef, toolResultItem.ErrorCode)),
			policyMetadata,
		)
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}

	status := strings.ToLower(strings.TrimSpace(agentResult.RunState))
	if status == "" {
		status = "completed"
	}
	toolResultItem.Status = status
	toolResultItem.Output = map[string]any{
		"workflow":               strings.TrimSpace(agentResult.Workflow),
		"task_id":                strings.TrimSpace(agentResult.TaskID),
		"run_id":                 strings.TrimSpace(agentResult.RunID),
		"task_state":             strings.TrimSpace(agentResult.TaskState),
		"run_state":              strings.TrimSpace(agentResult.RunState),
		"clarification_required": agentResult.ClarificationRequired,
		"clarification_prompt":   strings.TrimSpace(agentResult.ClarificationPrompt),
		"approval_required":      agentResult.ApprovalRequired,
		"approval_request_id":    strings.TrimSpace(agentResult.ApprovalRequestID),
		"native_action":          nativeAction,
	}
	if strings.TrimSpace(agentResult.TaskID) != "" && strings.TrimSpace(agentResult.RunID) != "" {
		taskRunReference = transport.ChatTurnTaskRunCorrelation{
			Available: true,
			Source:    "agent_run",
			TaskID:    strings.TrimSpace(agentResult.TaskID),
			RunID:     strings.TrimSpace(agentResult.RunID),
			TaskState: strings.TrimSpace(agentResult.TaskState),
			RunState:  strings.TrimSpace(agentResult.RunState),
		}
	}
	if policyRequiresApproval {
		approvalRequestID := strings.TrimSpace(agentResult.ApprovalRequestID)
		if !agentResult.ApprovalRequired || approvalRequestID == "" {
			reason := strings.TrimSpace(policyDecision.Reason)
			if reason == "" {
				reason = "tool policy requires approval"
			}
			toolResultItem.Status = "failed"
			toolResultItem.ErrorCode = "approval_required_unmet"
			toolResultItem.ErrorMessage = fmt.Sprintf("tool policy requires approval but executor response was not approval-gated: %s", reason)
			toolResultItem.Output["policy_requires_approval"] = true
			toolResultItem.Output["policy_reason"] = reason
			return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
		}
		toolResultItem.Status = "awaiting_approval"
		approvalItem = transport.ChatTurnItem{
			ItemID:            mustLocalRandomID("item"),
			Type:              "approval_request",
			Status:            "awaiting_approval",
			ApprovalRequestID: approvalRequestID,
			Content:           "Approval is required before execution can continue.",
			Metadata: transport.ChatTurnItemMetadataFromMap(map[string]any{
				"workflow":      strings.TrimSpace(agentResult.Workflow),
				"policy_reason": strings.TrimSpace(policyDecision.Reason),
			}),
		}
		approvalItem.Metadata = mergeToolItemMetadata(approvalItem.Metadata, policyMetadata)
		toolResultItem.ApprovalRequestID = approvalRequestID
		return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
	}
	if agentResult.ApprovalRequired {
		approvalItem = transport.ChatTurnItem{
			ItemID:            mustLocalRandomID("item"),
			Type:              "approval_request",
			Status:            "awaiting_approval",
			ApprovalRequestID: strings.TrimSpace(agentResult.ApprovalRequestID),
			Content:           "Approval is required before execution can continue.",
			Metadata: transport.ChatTurnItemMetadataFromMap(map[string]any{
				"workflow": strings.TrimSpace(agentResult.Workflow),
			}),
		}
		approvalItem.Metadata = mergeToolItemMetadata(approvalItem.Metadata, policyMetadata)
		toolResultItem.ApprovalRequestID = strings.TrimSpace(agentResult.ApprovalRequestID)
	}

	return toolCallItem, toolResultItem, approvalItem, taskRunReference, nil
}

func toolResultAsPlannerConversationItems(call transport.ChatTurnItem, result transport.ChatTurnItem, approval transport.ChatTurnItem) []transport.ChatTurnItem {
	payload := map[string]any{
		"tool_call": map[string]any{
			"tool_name":    call.ToolName,
			"tool_call_id": call.ToolCallID,
			"arguments":    call.Arguments,
			"status":       call.Status,
		},
		"tool_result": map[string]any{
			"status":              result.Status,
			"output":              result.Output,
			"error_code":          result.ErrorCode,
			"error_message":       result.ErrorMessage,
			"approval_request_id": result.ApprovalRequestID,
		},
	}
	if strings.TrimSpace(approval.Type) != "" {
		payload["approval_request"] = map[string]any{
			"approval_request_id": approval.ApprovalRequestID,
			"status":              approval.Status,
		}
	}
	serialized, _ := json.Marshal(payload)
	return []transport.ChatTurnItem{
		{
			Type:    "assistant_message",
			Role:    "assistant",
			Status:  "completed",
			Content: "Tool execution record: " + string(serialized),
		},
		{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Choose the next planner action for this turn (assistant_message or tool_call).",
		},
	}
}

func toolResultAsConversationItems(call transport.ChatTurnItem, result transport.ChatTurnItem, approval transport.ChatTurnItem) []transport.ChatTurnItem {
	payload := map[string]any{
		"tool_call": map[string]any{
			"tool_name":    call.ToolName,
			"tool_call_id": call.ToolCallID,
			"arguments":    call.Arguments,
			"status":       call.Status,
		},
		"tool_result": map[string]any{
			"status":              result.Status,
			"output":              result.Output,
			"error_code":          result.ErrorCode,
			"error_message":       result.ErrorMessage,
			"approval_request_id": result.ApprovalRequestID,
		},
	}
	if strings.TrimSpace(approval.Type) != "" {
		payload["approval_request"] = map[string]any{
			"approval_request_id": approval.ApprovalRequestID,
			"status":              approval.Status,
		}
	}
	serialized, _ := json.Marshal(payload)
	return []transport.ChatTurnItem{
		{
			Type:    "assistant_message",
			Role:    "assistant",
			Status:  "completed",
			Content: "Tool execution record: " + string(serialized),
		},
		{
			Type:    "user_message",
			Role:    "user",
			Status:  "completed",
			Content: "Using the tool execution record above, provide the final assistant response.",
		},
	}
}

func fallbackAssistantSummary(result transport.ChatTurnItem, approval transport.ChatTurnItem) string {
	if strings.TrimSpace(approval.ApprovalRequestID) != "" {
		return fmt.Sprintf("Execution is waiting for approval (request %s).", strings.TrimSpace(approval.ApprovalRequestID))
	}
	if strings.TrimSpace(result.ErrorMessage) != "" {
		return fmt.Sprintf("Tool execution failed: %s", strings.TrimSpace(result.ErrorMessage))
	}
	if strings.EqualFold(strings.TrimSpace(result.Status), "completed") {
		return "Tool execution completed successfully."
	}
	return "Tool execution finished with no additional details."
}

func mergeToolItemMetadata(base transport.ChatTurnItemMetadata, overlay map[string]any) transport.ChatTurnItemMetadata {
	merged := base.AsMap()
	for key, value := range overlay {
		merged[key] = value
	}
	return transport.ChatTurnItemMetadataFromMap(merged)
}
