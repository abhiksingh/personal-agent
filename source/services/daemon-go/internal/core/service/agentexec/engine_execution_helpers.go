package agentexec

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	voicehandoffservice "personalagent/runtime/internal/core/service/voicehandoff"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

func normalizeExecutionOrigin(origin types.ExecutionOrigin) (types.ExecutionOrigin, error) {
	normalized := strings.ToLower(strings.TrimSpace(string(origin)))
	if normalized == "" {
		return types.ExecutionOriginCLI, nil
	}
	switch types.ExecutionOrigin(normalized) {
	case types.ExecutionOriginApp, types.ExecutionOriginCLI, types.ExecutionOriginVoice:
		return types.ExecutionOrigin(normalized), nil
	default:
		return "", fmt.Errorf("unsupported execution origin %q", origin)
	}
}

func executionOriginFromSourceChannel(sourceChannel string) types.ExecutionOrigin {
	switch strings.ToLower(strings.TrimSpace(sourceChannel)) {
	case "voice":
		return types.ExecutionOriginVoice
	case "app_chat", "app":
		return types.ExecutionOriginApp
	default:
		return types.ExecutionOriginCLI
	}
}

func sourceChannelForExecutionOrigin(origin types.ExecutionOrigin) string {
	switch origin {
	case types.ExecutionOriginApp:
		return "app_chat"
	case types.ExecutionOriginVoice:
		return "voice"
	default:
		return "cli_chat"
	}
}

func evaluateStepApprovalDecision(
	step plannedStep,
	executionOrigin types.ExecutionOrigin,
	inAppApprovalConfirmed bool,
	approvalPhrase string,
	voiceGate *voicehandoffservice.Gate,
) stepApprovalDecision {
	capability := strings.ToLower(strings.TrimSpace(step.CapabilityKey))
	rationale := approvalDecisionRationale{
		PolicyVersion:      "capability_risk_v1",
		CapabilityKey:      capability,
		RiskLevel:          "reversible",
		RiskConfidence:     0.92,
		RiskReason:         "capability classified as reversible by policy",
		Decision:           "allow",
		DecisionReason:     "reversible capability does not require approval",
		DecisionReasonCode: "reversible_capability",
		DecisionSource:     "capability_policy",
		ExecutionOrigin:    string(executionOrigin),
	}

	destructiveClass, destructive := destructiveClassForCapability(capability)
	if !destructive {
		return stepApprovalDecision{
			RequireApproval: false,
			Rationale:       rationale,
		}
	}

	rationale.RiskLevel = "destructive"
	rationale.RiskConfidence = 0.99
	rationale.RiskReason = fmt.Sprintf("capability %s classified as destructive (%s)", capability, destructiveClass)
	rationale.DestructiveClass = destructiveClass
	rationale.Decision = "require_confirm"
	rationale.DecisionReason = "destructive capability requires explicit GO AHEAD approval"
	rationale.DecisionReasonCode = "destructive_capability"
	rationale.DecisionSource = "capability_policy"

	if executionOrigin == types.ExecutionOriginVoice {
		resolvedVoiceGate := voiceGate
		if resolvedVoiceGate == nil {
			resolvedVoiceGate = voicehandoffservice.NewGate()
		}
		voiceDecision := resolvedVoiceGate.Evaluate(types.VoiceHandoffInput{
			Origin:                 executionOrigin,
			DestructiveAction:      true,
			InAppApprovalConfirmed: inAppApprovalConfirmed,
		})
		if voiceDecision.RequireInAppApproval {
			reason := strings.TrimSpace(voiceDecision.Reason)
			if reason == "" {
				reason = "voice destructive action requires in-app approval handoff"
			}
			rationale.DecisionReason = reason
			rationale.DecisionReasonCode = "voice_in_app_handoff_required"
			rationale.DecisionSource = "voice_handoff"
			return stepApprovalDecision{
				RequireApproval: true,
				Summary:         reason,
				Rationale:       rationale,
			}
		}

		rationale.Decision = "allow"
		rationale.DecisionReason = "voice in-app approval confirmed"
		rationale.DecisionReasonCode = "voice_in_app_confirmed"
		rationale.DecisionSource = "voice_handoff"
		return stepApprovalDecision{
			RequireApproval: false,
			Rationale:       rationale,
		}
	}

	if strings.TrimSpace(approvalPhrase) != types.DestructiveApprovalPhrase {
		rationale.DecisionReason = "destructive capability requires explicit GO AHEAD approval"
		rationale.DecisionReasonCode = "missing_approval_phrase"
		rationale.DecisionSource = "approval_phrase"
		return stepApprovalDecision{
			RequireApproval: true,
			Summary:         "awaiting destructive approval",
			Rationale:       rationale,
		}
	}

	rationale.Decision = "allow"
	rationale.DecisionReason = "exact GO AHEAD phrase supplied before execution"
	rationale.DecisionReasonCode = "preapproved_go_ahead"
	rationale.DecisionSource = "approval_phrase"
	return stepApprovalDecision{
		RequireApproval: false,
		Rationale:       rationale,
	}
}

func destructiveClassForCapability(capability string) (string, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(capability))
	switch trimmed {
	case "":
		return "", false
	case "finder_delete":
		return "file_delete", true
	case "calendar_cancel":
		return "calendar_cancel", true
	}
	if strings.HasSuffix(trimmed, "_delete") {
		return "delete_operation", true
	}
	if strings.HasSuffix(trimmed, "_cancel") {
		return "cancel_operation", true
	}
	return "", false
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func cloneEvidence(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func parseStepInputJSON(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil
	}
	return cloneAnyMap(payload)
}

func isMessagesSendCapability(capability string) bool {
	trimmed := strings.TrimSpace(capability)
	return strings.HasPrefix(trimmed, "messages_send_")
}

func (e *SQLiteExecutionEngine) executeMessageStep(ctx context.Context, request MessageDispatchRequest) (connectorcontract.StepExecutionResult, error) {
	if e.messageDispatcher == nil {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "messages dispatch is not configured",
			Retryable:   false,
			ErrorReason: "dispatcher_unavailable",
		}, fmt.Errorf("messages dispatch is not configured")
	}
	channel := normalizeMessageChannel(request.SourceChannel)
	destination := strings.TrimSpace(request.Destination)
	body := strings.TrimSpace(request.MessageBody)
	if channel == "" {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "messages channel is required",
			Retryable:   false,
			ErrorReason: "invalid_request",
		}, fmt.Errorf("messages channel is required")
	}
	if destination == "" {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "messages recipient is required",
			Retryable:   false,
			ErrorReason: "invalid_request",
		}, fmt.Errorf("messages recipient is required")
	}
	if body == "" {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "messages body is required",
			Retryable:   false,
			ErrorReason: "invalid_request",
		}, fmt.Errorf("messages body is required")
	}

	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		operationID = fmt.Sprintf("taskrun-%s-step-%d", strings.TrimSpace(request.RunID), request.StepIndex)
	}

	dispatchResult, err := e.messageDispatcher.DispatchMessage(ctx, MessageDispatchRequest{
		WorkspaceID:   strings.TrimSpace(request.WorkspaceID),
		TaskID:        strings.TrimSpace(request.TaskID),
		RunID:         strings.TrimSpace(request.RunID),
		StepIndex:     request.StepIndex,
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		SourceChannel: channel,
		Destination:   destination,
		MessageBody:   body,
		OperationID:   operationID,
	})
	if err != nil {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "messages dispatch failed",
			Retryable:   true,
			ErrorReason: "dispatch_failed",
			Evidence: map[string]string{
				"channel":     channel,
				"destination": destination,
				"operation":   operationID,
			},
		}, err
	}

	resolvedChannel := normalizeMessageChannel(dispatchResult.Channel)
	if resolvedChannel == "" {
		resolvedChannel = channel
	}
	summary := strings.TrimSpace(dispatchResult.Summary)
	if summary == "" {
		summary = fmt.Sprintf("message dispatched via %s", resolvedChannel)
	}
	evidence := map[string]string{
		"channel":     resolvedChannel,
		"destination": destination,
		"operation":   operationID,
	}
	if receipt := strings.TrimSpace(dispatchResult.ProviderReceipt); receipt != "" {
		evidence["provider_receipt"] = receipt
	}

	return connectorcontract.StepExecutionResult{
		Status:    shared.TaskStepStatusCompleted,
		Summary:   summary,
		Retryable: false,
		Evidence:  evidence,
		Output: map[string]any{
			"channel":          resolvedChannel,
			"destination":      destination,
			"provider_receipt": strings.TrimSpace(dispatchResult.ProviderReceipt),
			"operation_id":     operationID,
		},
	}, nil
}

func workflowFromCapability(capability string) string {
	switch {
	case strings.HasPrefix(capability, "mail_"):
		return WorkflowMail
	case strings.HasPrefix(capability, "calendar_"):
		return WorkflowCalendar
	case strings.HasPrefix(capability, "messages_"):
		return WorkflowMessages
	case strings.HasPrefix(capability, "browser_"):
		return WorkflowBrowser
	case strings.HasPrefix(capability, "finder_"):
		return WorkflowFinder
	default:
		return "unknown"
	}
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func cloneNativeAction(action NativeAction) *NativeAction {
	if strings.TrimSpace(action.Connector) == "" &&
		strings.TrimSpace(action.Operation) == "" &&
		action.Mail == nil &&
		action.Calendar == nil &&
		action.Messages == nil &&
		action.Browser == nil &&
		action.Finder == nil {
		return nil
	}
	clone := action
	if action.Mail != nil {
		mail := *action.Mail
		clone.Mail = &mail
	}
	if action.Calendar != nil {
		calendar := *action.Calendar
		clone.Calendar = &calendar
	}
	if action.Messages != nil {
		messages := *action.Messages
		clone.Messages = &messages
	}
	if action.Browser != nil {
		browser := *action.Browser
		clone.Browser = &browser
	}
	if action.Finder != nil {
		finder := *action.Finder
		clone.Finder = &finder
	}
	return &clone
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}
