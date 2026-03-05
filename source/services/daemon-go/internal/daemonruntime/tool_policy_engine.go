package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

type ToolPolicyDecision string

const (
	ToolPolicyDecisionAllow           ToolPolicyDecision = "ALLOW"
	ToolPolicyDecisionRequireApproval ToolPolicyDecision = "REQUIRE_APPROVAL"
	ToolPolicyDecisionDeny            ToolPolicyDecision = "DENY"
)

const toolCapabilityPolicyVersionV1 = "capability_policy.v1"

const (
	toolRiskClassLow         = "low"
	toolRiskClassExternalIO  = "external_io"
	toolRiskClassDestructive = "destructive"
	toolRiskClassUnknown     = "unknown"
)

const (
	toolApprovalModeNever     = "never"
	toolApprovalModeAlways    = "always"
	toolApprovalModeVoiceOnly = "voice_only"
)

type ToolPolicyRequest struct {
	WorkspaceID        string
	RequestedByActorID string
	ActingAsActorID    string
	ChannelID          string
	ToolName           string
	CapabilityKey      string
}

type ToolPolicyResult struct {
	Decision   ToolPolicyDecision
	Reason     string
	ReasonCode string
	Rationale  ToolPolicyRationale
}

type ToolPolicyRationale struct {
	PolicyVersion      string
	ToolName           string
	CapabilityKey      string
	RiskClass          string
	Idempotent         bool
	ApprovalMode       string
	ChannelID          string
	AllowedChannels    []string
	Decision           ToolPolicyDecision
	DecisionReason     string
	DecisionReasonCode string
	DecisionSource     string
}

type toolCapabilityPolicyMetadata struct {
	CapabilityKey   string
	RiskClass       string
	Idempotent      bool
	ApprovalMode    string
	AllowedChannels []string
}

type ToolPolicyEngine struct {
	delegation transport.DelegationService
}

var toolCapabilityPolicyCatalog = map[string]toolCapabilityPolicyMetadata{
	"mail_draft": {
		CapabilityKey:   "mail_draft",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"mail_send": {
		CapabilityKey:   "mail_send",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"mail_reply": {
		CapabilityKey:   "mail_reply",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"mail_unread_summary": {
		CapabilityKey:   "mail_unread_summary",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"calendar_create": {
		CapabilityKey:   "calendar_create",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"calendar_update": {
		CapabilityKey:   "calendar_update",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"calendar_cancel": {
		CapabilityKey:   "calendar_cancel",
		RiskClass:       toolRiskClassDestructive,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeAlways,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"browser_open": {
		CapabilityKey:   "browser_open",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"browser_extract": {
		CapabilityKey:   "browser_extract",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"browser_close": {
		CapabilityKey:   "browser_close",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"finder_find": {
		CapabilityKey:   "finder_find",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"finder_list": {
		CapabilityKey:   "finder_list",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"finder_preview": {
		CapabilityKey:   "finder_preview",
		RiskClass:       toolRiskClassLow,
		Idempotent:      true,
		ApprovalMode:    toolApprovalModeNever,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"finder_delete": {
		CapabilityKey:   "finder_delete",
		RiskClass:       toolRiskClassDestructive,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeAlways,
		AllowedChannels: []string{"app", "message", "voice"},
	},
	"channel_messages_send": {
		CapabilityKey:   "channel_messages_send",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeVoiceOnly,
		AllowedChannels: []string{"app", "message"},
	},
	"channel_twilio_sms_send": {
		CapabilityKey:   "channel_twilio_sms_send",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeVoiceOnly,
		AllowedChannels: []string{"app", "message"},
	},
	"messages_send_imessage": {
		CapabilityKey:   "messages_send_imessage",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeVoiceOnly,
		AllowedChannels: []string{"app", "message"},
	},
	"messages_send_sms": {
		CapabilityKey:   "messages_send_sms",
		RiskClass:       toolRiskClassExternalIO,
		Idempotent:      false,
		ApprovalMode:    toolApprovalModeVoiceOnly,
		AllowedChannels: []string{"app", "message"},
	},
}

func NewToolPolicyEngine(delegation transport.DelegationService) *ToolPolicyEngine {
	return &ToolPolicyEngine{delegation: delegation}
}

func (e *ToolPolicyEngine) Evaluate(ctx context.Context, request ToolPolicyRequest) (ToolPolicyResult, error) {
	toolName := normalizePolicyKey(request.ToolName)
	if toolName == "" {
		return buildToolPolicyResult(request, ToolPolicyDecisionDeny, toolCapabilityPolicyMetadata{
			CapabilityKey: "unknown",
			RiskClass:     toolRiskClassUnknown,
		}, "missing_tool_name", "tool name is required", "validation"), nil
	}

	capabilityKey := resolveToolPolicyCapabilityKey(toolName, request.CapabilityKey)
	metadata, metadataFound := toolCapabilityPolicyCatalog[capabilityKey]
	if !metadataFound {
		return buildToolPolicyResult(request, ToolPolicyDecisionDeny, toolCapabilityPolicyMetadata{
			CapabilityKey: capabilityKeyOrUnknown(capabilityKey),
			RiskClass:     toolRiskClassUnknown,
		}, "capability_metadata_missing", fmt.Sprintf("capability metadata is not defined for %s", capabilityKeyOrUnknown(capabilityKey)), "capability_policy"), nil
	}

	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	requestedBy := strings.TrimSpace(request.RequestedByActorID)
	actingAs := strings.TrimSpace(request.ActingAsActorID)
	if e.delegation != nil && requestedBy != "" && actingAs != "" && !strings.EqualFold(requestedBy, actingAs) {
		decision, err := e.delegation.CheckDelegation(ctx, transport.DelegationCheckRequest{
			WorkspaceID:        workspaceID,
			RequestedByActorID: requestedBy,
			ActingAsActorID:    actingAs,
			ScopeType:          "EXECUTION",
		})
		if err != nil {
			return ToolPolicyResult{}, fmt.Errorf("evaluate delegation policy: %w", err)
		}
		if !decision.Allowed {
			reason := strings.TrimSpace(decision.Reason)
			if reason == "" {
				reason = "delegation policy denied execution"
			}
			return buildToolPolicyResult(request, ToolPolicyDecisionDeny, metadata, "delegation_denied", reason, "delegation_policy"), nil
		}
	}

	channelID := normalizeTurnChannelID(request.ChannelID)
	if !toolPolicyChannelAllowed(metadata.AllowedChannels, channelID) {
		return buildToolPolicyResult(request, ToolPolicyDecisionDeny, metadata, "channel_not_allowed", fmt.Sprintf("capability %s is not allowed on channel %s", metadata.CapabilityKey, channelID), "capability_policy"), nil
	}

	switch strings.ToLower(strings.TrimSpace(metadata.ApprovalMode)) {
	case toolApprovalModeAlways:
		return buildToolPolicyResult(request, ToolPolicyDecisionRequireApproval, metadata, "approval_required", "capability policy requires approval before execution", "capability_policy"), nil
	case toolApprovalModeVoiceOnly:
		if channelID == "voice" {
			return buildToolPolicyResult(request, ToolPolicyDecisionRequireApproval, metadata, "approval_required_voice", "voice execution requires approval for this capability", "capability_policy"), nil
		}
	}

	return buildToolPolicyResult(request, ToolPolicyDecisionAllow, metadata, "allowed", "capability policy allows execution", "capability_policy"), nil
}

func (r ToolPolicyResult) MetadataMap() map[string]any {
	allowedChannels := append([]string(nil), r.Rationale.AllowedChannels...)
	sort.Strings(allowedChannels)
	metadata := map[string]any{
		"policy_decision":    string(r.Decision),
		"policy_reason":      strings.TrimSpace(r.Reason),
		"policy_reason_code": strings.TrimSpace(r.ReasonCode),
		"policy_rationale": map[string]any{
			"policy_version":       strings.TrimSpace(r.Rationale.PolicyVersion),
			"tool_name":            strings.TrimSpace(r.Rationale.ToolName),
			"capability_key":       strings.TrimSpace(r.Rationale.CapabilityKey),
			"risk_class":           strings.TrimSpace(r.Rationale.RiskClass),
			"idempotent":           r.Rationale.Idempotent,
			"approval_mode":        strings.TrimSpace(r.Rationale.ApprovalMode),
			"channel_id":           strings.TrimSpace(r.Rationale.ChannelID),
			"allowed_channels":     allowedChannels,
			"decision":             string(r.Rationale.Decision),
			"decision_reason":      strings.TrimSpace(r.Rationale.DecisionReason),
			"decision_reason_code": strings.TrimSpace(r.Rationale.DecisionReasonCode),
			"decision_source":      strings.TrimSpace(r.Rationale.DecisionSource),
		},
	}
	return metadata
}

func buildToolPolicyResult(
	request ToolPolicyRequest,
	decision ToolPolicyDecision,
	metadata toolCapabilityPolicyMetadata,
	reasonCode string,
	reason string,
	decisionSource string,
) ToolPolicyResult {
	channelID := normalizeTurnChannelID(request.ChannelID)
	toolName := strings.TrimSpace(request.ToolName)
	capabilityKey := strings.TrimSpace(metadata.CapabilityKey)
	if capabilityKey == "" {
		capabilityKey = capabilityKeyOrUnknown(resolveToolPolicyCapabilityKey(toolName, request.CapabilityKey))
	}
	allowedChannels := append([]string(nil), metadata.AllowedChannels...)
	sort.Strings(allowedChannels)

	rationale := ToolPolicyRationale{
		PolicyVersion:      toolCapabilityPolicyVersionV1,
		ToolName:           toolName,
		CapabilityKey:      capabilityKey,
		RiskClass:          firstNonEmptyTrimmed(strings.TrimSpace(metadata.RiskClass), toolRiskClassUnknown),
		Idempotent:         metadata.Idempotent,
		ApprovalMode:       firstNonEmptyTrimmed(strings.TrimSpace(metadata.ApprovalMode), toolApprovalModeNever),
		ChannelID:          channelID,
		AllowedChannels:    allowedChannels,
		Decision:           decision,
		DecisionReason:     strings.TrimSpace(reason),
		DecisionReasonCode: strings.TrimSpace(reasonCode),
		DecisionSource:     strings.TrimSpace(decisionSource),
	}
	return ToolPolicyResult{
		Decision:   decision,
		Reason:     strings.TrimSpace(reason),
		ReasonCode: strings.TrimSpace(reasonCode),
		Rationale:  rationale,
	}
}

func normalizePolicyKey(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, ".", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return strings.TrimSpace(normalized)
}

func resolveToolPolicyCapabilityKey(toolName string, capabilityKey string) string {
	normalizedCapability := normalizePolicyKey(capabilityKey)
	if normalizedCapability != "" {
		return normalizedCapability
	}
	normalizedTool := normalizePolicyKey(toolName)
	if normalizedTool == "" {
		return ""
	}
	return normalizedTool
}

func capabilityKeyOrUnknown(capabilityKey string) string {
	if strings.TrimSpace(capabilityKey) == "" {
		return "unknown"
	}
	return strings.TrimSpace(capabilityKey)
}

func toolPolicyChannelAllowed(allowedChannels []string, channelID string) bool {
	normalizedChannel := normalizeTurnChannelID(channelID)
	if len(allowedChannels) == 0 {
		return true
	}
	for _, channel := range allowedChannels {
		if normalizeTurnChannelID(channel) == normalizedChannel {
			return true
		}
	}
	return false
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
