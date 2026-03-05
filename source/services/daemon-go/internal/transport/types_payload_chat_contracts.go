package transport

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChatToolPolicyRationale captures externally-consumed policy metadata fields while
// preserving unknown future keys in Additional.
type ChatToolPolicyRationale struct {
	PolicyVersion     string         `json:"policy_version,omitempty"`
	Decision          string         `json:"decision,omitempty"`
	ReasonCode        string         `json:"reason_code,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	CapabilityKey     string         `json:"capability_key,omitempty"`
	CapabilityName    string         `json:"capability_name,omitempty"`
	RiskClass         string         `json:"risk_class,omitempty"`
	Idempotency       string         `json:"idempotency,omitempty"`
	ApprovalMode      string         `json:"approval_mode,omitempty"`
	ChannelConstraint string         `json:"channel_constraint,omitempty"`
	NeedsApproval     *bool          `json:"needs_approval,omitempty"`
	Additional        map[string]any `json:"-"`
}

func (r ChatToolPolicyRationale) IsZero() bool {
	return r.PolicyVersion == "" &&
		r.Decision == "" &&
		r.ReasonCode == "" &&
		r.Reason == "" &&
		r.CapabilityKey == "" &&
		r.CapabilityName == "" &&
		r.RiskClass == "" &&
		r.Idempotency == "" &&
		r.ApprovalMode == "" &&
		r.ChannelConstraint == "" &&
		r.NeedsApproval == nil &&
		len(r.Additional) == 0
}

func (r ChatToolPolicyRationale) AsMap() map[string]any {
	result := cloneAnyMapShallow(r.Additional)
	setStringField(result, "policy_version", r.PolicyVersion)
	setStringField(result, "decision", r.Decision)
	setStringField(result, "reason_code", r.ReasonCode)
	setStringField(result, "reason", r.Reason)
	setStringField(result, "capability_key", r.CapabilityKey)
	setStringField(result, "capability_name", r.CapabilityName)
	setStringField(result, "risk_class", r.RiskClass)
	setStringField(result, "idempotency", r.Idempotency)
	setStringField(result, "approval_mode", r.ApprovalMode)
	setStringField(result, "channel_constraint", r.ChannelConstraint)
	if r.NeedsApproval != nil {
		result["needs_approval"] = *r.NeedsApproval
	}
	return result
}

func ChatToolPolicyRationaleFromMap(value map[string]any) ChatToolPolicyRationale {
	if len(value) == 0 {
		return ChatToolPolicyRationale{}
	}
	result := ChatToolPolicyRationale{
		PolicyVersion:     readAnyString(value["policy_version"]),
		Decision:          readAnyString(value["decision"]),
		ReasonCode:        readAnyString(value["reason_code"]),
		Reason:            readAnyString(value["reason"]),
		CapabilityKey:     readAnyString(value["capability_key"]),
		CapabilityName:    readAnyString(value["capability_name"]),
		RiskClass:         readAnyString(value["risk_class"]),
		Idempotency:       readAnyString(value["idempotency"]),
		ApprovalMode:      readAnyString(value["approval_mode"]),
		ChannelConstraint: readAnyString(value["channel_constraint"]),
		NeedsApproval:     readAnyBoolPointer(value["needs_approval"]),
	}
	result.Additional = removeKnownKeys(value,
		"policy_version",
		"decision",
		"reason_code",
		"reason",
		"capability_key",
		"capability_name",
		"risk_class",
		"idempotency",
		"approval_mode",
		"channel_constraint",
		"needs_approval",
	)
	return result
}

func (r ChatToolPolicyRationale) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.AsMap())
}

func (r *ChatToolPolicyRationale) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("nil ChatToolPolicyRationale")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*r = ChatToolPolicyRationale{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = ChatToolPolicyRationaleFromMap(decoded)
	return nil
}

// ChatMetadataRemediation captures deterministic remediation hints with extensibility.
type ChatMetadataRemediation struct {
	Code            string         `json:"code,omitempty"`
	Domain          string         `json:"domain,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	PrimaryAction   string         `json:"primary_action,omitempty"`
	SecondaryAction string         `json:"secondary_action,omitempty"`
	Additional      map[string]any `json:"-"`
}

func (r ChatMetadataRemediation) IsZero() bool {
	return r.Code == "" && r.Domain == "" && r.Summary == "" && r.PrimaryAction == "" && r.SecondaryAction == "" && len(r.Additional) == 0
}

func (r ChatMetadataRemediation) AsMap() map[string]any {
	result := cloneAnyMapShallow(r.Additional)
	setStringField(result, "code", r.Code)
	setStringField(result, "domain", r.Domain)
	setStringField(result, "summary", r.Summary)
	setStringField(result, "primary_action", r.PrimaryAction)
	setStringField(result, "secondary_action", r.SecondaryAction)
	return result
}

func ChatMetadataRemediationFromMap(value map[string]any) ChatMetadataRemediation {
	if len(value) == 0 {
		return ChatMetadataRemediation{}
	}
	result := ChatMetadataRemediation{
		Code:            readAnyString(value["code"]),
		Domain:          readAnyString(value["domain"]),
		Summary:         readAnyString(value["summary"]),
		PrimaryAction:   readAnyString(value["primary_action"]),
		SecondaryAction: readAnyString(value["secondary_action"]),
	}
	result.Additional = removeKnownKeys(value,
		"code",
		"domain",
		"summary",
		"primary_action",
		"secondary_action",
	)
	return result
}

func (r ChatMetadataRemediation) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.AsMap())
}

func (r *ChatMetadataRemediation) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("nil ChatMetadataRemediation")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*r = ChatMetadataRemediation{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = ChatMetadataRemediationFromMap(decoded)
	return nil
}

// ChatTurnItemMetadata types external tool/result metadata fields and keeps unknown
// fields in Additional for deterministic forward compatibility.
type ChatTurnItemMetadata struct {
	PolicyDecision                  string                   `json:"policy_decision,omitempty"`
	PolicyReasonCode                string                   `json:"policy_reason_code,omitempty"`
	PolicyRationale                 *ChatToolPolicyRationale `json:"policy_rationale,omitempty"`
	ValidationErrorCode             string                   `json:"validation_error_code,omitempty"`
	ValidationArgument              string                   `json:"validation_argument,omitempty"`
	ValidationExpected              string                   `json:"validation_expected,omitempty"`
	SchemaRegistryVersion           string                   `json:"schema_registry_version,omitempty"`
	ResponseShapingChannel          string                   `json:"response_shaping_channel,omitempty"`
	ResponseShapingProfile          string                   `json:"response_shaping_profile,omitempty"`
	ResponseShapingGuardrailCount   *int                     `json:"response_shaping_guardrail_count,omitempty"`
	ResponseShapingInstructionCount *int                     `json:"response_shaping_instruction_count,omitempty"`
	StopReason                      string                   `json:"stop_reason,omitempty"`
	PlannerRepairAttempts           *int                     `json:"planner_repair_attempts,omitempty"`
	Code                            string                   `json:"code,omitempty"`
	Domain                          string                   `json:"domain,omitempty"`
	Summary                         string                   `json:"summary,omitempty"`
	PrimaryAction                   string                   `json:"primary_action,omitempty"`
	SecondaryAction                 string                   `json:"secondary_action,omitempty"`
	Remediation                     *ChatMetadataRemediation `json:"remediation,omitempty"`
	Additional                      map[string]any           `json:"-"`
}

func (m ChatTurnItemMetadata) IsZero() bool {
	return m.PolicyDecision == "" &&
		m.PolicyReasonCode == "" &&
		(m.PolicyRationale == nil || m.PolicyRationale.IsZero()) &&
		m.ValidationErrorCode == "" &&
		m.ValidationArgument == "" &&
		m.ValidationExpected == "" &&
		m.SchemaRegistryVersion == "" &&
		m.ResponseShapingChannel == "" &&
		m.ResponseShapingProfile == "" &&
		m.ResponseShapingGuardrailCount == nil &&
		m.ResponseShapingInstructionCount == nil &&
		m.StopReason == "" &&
		m.PlannerRepairAttempts == nil &&
		m.Code == "" &&
		m.Domain == "" &&
		m.Summary == "" &&
		m.PrimaryAction == "" &&
		m.SecondaryAction == "" &&
		(m.Remediation == nil || m.Remediation.IsZero()) &&
		len(m.Additional) == 0
}

func (m *ChatTurnItemMetadata) ensureAdditionalMap() {
	if m.Additional == nil {
		m.Additional = map[string]any{}
	}
}

func (m *ChatTurnItemMetadata) Set(key string, value any) {
	if m == nil {
		return
	}
	normalized := strings.TrimSpace(key)
	switch normalized {
	case "policy_decision":
		m.PolicyDecision = readAnyString(value)
	case "policy_reason_code":
		m.PolicyReasonCode = readAnyString(value)
	case "policy_rationale":
		if decoded := readAnyMap(value); len(decoded) > 0 {
			rationale := ChatToolPolicyRationaleFromMap(decoded)
			m.PolicyRationale = &rationale
		} else {
			m.PolicyRationale = nil
		}
	case "validation_error_code":
		m.ValidationErrorCode = readAnyString(value)
	case "validation_argument":
		m.ValidationArgument = readAnyString(value)
	case "validation_expected":
		m.ValidationExpected = readAnyString(value)
	case "schema_registry_version":
		m.SchemaRegistryVersion = readAnyString(value)
	case "response_shaping_channel":
		m.ResponseShapingChannel = readAnyString(value)
	case "response_shaping_profile":
		m.ResponseShapingProfile = readAnyString(value)
	case "response_shaping_guardrail_count":
		m.ResponseShapingGuardrailCount = readAnyIntPointer(value)
	case "response_shaping_instruction_count":
		m.ResponseShapingInstructionCount = readAnyIntPointer(value)
	case "stop_reason":
		m.StopReason = readAnyString(value)
	case "planner_repair_attempts":
		m.PlannerRepairAttempts = readAnyIntPointer(value)
	case "code":
		m.Code = readAnyString(value)
	case "domain":
		m.Domain = readAnyString(value)
	case "summary":
		m.Summary = readAnyString(value)
	case "primary_action":
		m.PrimaryAction = readAnyString(value)
	case "secondary_action":
		m.SecondaryAction = readAnyString(value)
	case "remediation":
		if decoded := readAnyMap(value); len(decoded) > 0 {
			remediation := ChatMetadataRemediationFromMap(decoded)
			m.Remediation = &remediation
		} else {
			m.Remediation = nil
		}
	default:
		m.ensureAdditionalMap()
		m.Additional[normalized] = value
	}
}

func (m ChatTurnItemMetadata) AsMap() map[string]any {
	result := cloneAnyMapShallow(m.Additional)
	setStringField(result, "policy_decision", m.PolicyDecision)
	setStringField(result, "policy_reason_code", m.PolicyReasonCode)
	if m.PolicyRationale != nil && !m.PolicyRationale.IsZero() {
		result["policy_rationale"] = m.PolicyRationale.AsMap()
	}
	setStringField(result, "validation_error_code", m.ValidationErrorCode)
	setStringField(result, "validation_argument", m.ValidationArgument)
	setStringField(result, "validation_expected", m.ValidationExpected)
	setStringField(result, "schema_registry_version", m.SchemaRegistryVersion)
	setStringField(result, "response_shaping_channel", m.ResponseShapingChannel)
	setStringField(result, "response_shaping_profile", m.ResponseShapingProfile)
	setIntPointerField(result, "response_shaping_guardrail_count", m.ResponseShapingGuardrailCount)
	setIntPointerField(result, "response_shaping_instruction_count", m.ResponseShapingInstructionCount)
	setStringField(result, "stop_reason", m.StopReason)
	setIntPointerField(result, "planner_repair_attempts", m.PlannerRepairAttempts)
	setStringField(result, "code", m.Code)
	setStringField(result, "domain", m.Domain)
	setStringField(result, "summary", m.Summary)
	setStringField(result, "primary_action", m.PrimaryAction)
	setStringField(result, "secondary_action", m.SecondaryAction)
	if m.Remediation != nil && !m.Remediation.IsZero() {
		result["remediation"] = m.Remediation.AsMap()
	}
	return result
}

func ChatTurnItemMetadataFromMap(value map[string]any) ChatTurnItemMetadata {
	if len(value) == 0 {
		return ChatTurnItemMetadata{}
	}
	result := ChatTurnItemMetadata{}
	for key, item := range value {
		result.Set(key, item)
	}
	result.Additional = removeKnownKeys(value,
		"policy_decision",
		"policy_reason_code",
		"policy_rationale",
		"validation_error_code",
		"validation_argument",
		"validation_expected",
		"schema_registry_version",
		"response_shaping_channel",
		"response_shaping_profile",
		"response_shaping_guardrail_count",
		"response_shaping_instruction_count",
		"stop_reason",
		"planner_repair_attempts",
		"code",
		"domain",
		"summary",
		"primary_action",
		"secondary_action",
		"remediation",
	)
	return result
}

func (m ChatTurnItemMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.AsMap())
}

func (m *ChatTurnItemMetadata) UnmarshalJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf("nil ChatTurnItemMetadata")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*m = ChatTurnItemMetadata{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = ChatTurnItemMetadataFromMap(decoded)
	return nil
}
