package transport

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestChatTurnItemMetadataTypedRoundTripWithExtensions(t *testing.T) {
	needsApproval := true
	guardrailCount := 2
	metadata := ChatTurnItemMetadata{
		PolicyDecision:   "ALLOW",
		PolicyReasonCode: "allowed",
		PolicyRationale: &ChatToolPolicyRationale{
			PolicyVersion: "tool_policy.v1",
			Decision:      "ALLOW",
			NeedsApproval: &needsApproval,
			Additional: map[string]any{
				"reason_detail": "connector ready",
			},
		},
		ResponseShapingProfile:        "app.default",
		ResponseShapingGuardrailCount: &guardrailCount,
		Additional: map[string]any{
			"workflow": "mail_send",
		},
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	var decoded ChatTurnItemMetadata
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if decoded.PolicyDecision != "ALLOW" || decoded.PolicyReasonCode != "allowed" {
		t.Fatalf("expected typed policy metadata fields to round-trip, got %+v", decoded)
	}
	if decoded.PolicyRationale == nil || decoded.PolicyRationale.Decision != "ALLOW" {
		t.Fatalf("expected typed policy_rationale to round-trip, got %+v", decoded.PolicyRationale)
	}
	if decoded.PolicyRationale.Additional["reason_detail"] != "connector ready" {
		t.Fatalf("expected policy_rationale extension field to round-trip, got %+v", decoded.PolicyRationale.Additional)
	}
	if decoded.Additional["workflow"] != "mail_send" {
		t.Fatalf("expected metadata extension field to round-trip, got %+v", decoded.Additional)
	}
}

func TestRealtimeEventPayloadTypedRoundTripWithExtensions(t *testing.T) {
	accepted := true
	itemCount := 3
	payload := RealtimeEventPayload{
		TaskID:     "task-1",
		RunID:      "run-1",
		Status:     "completed",
		SignalType: "cancel",
		Accepted:   &accepted,
		ItemCount:  &itemCount,
		ToolName:   "mail_send",
		Metadata:   ChatTurnItemMetadataFromMap(map[string]any{"policy_reason_code": "allowed", "trace_id": "trace-1"}),
		Additional: map[string]any{"custom_flag": "enabled"},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var decoded RealtimeEventPayload
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if decoded.TaskID != "task-1" || decoded.RunID != "run-1" {
		t.Fatalf("expected typed task/run ids to round-trip, got %+v", decoded)
	}
	if decoded.Accepted == nil || !*decoded.Accepted {
		t.Fatalf("expected accepted=true to round-trip, got %+v", decoded.Accepted)
	}
	if decoded.ItemCount == nil || *decoded.ItemCount != 3 {
		t.Fatalf("expected item_count=3 to round-trip, got %+v", decoded.ItemCount)
	}
	if decoded.Metadata.PolicyReasonCode != "allowed" {
		t.Fatalf("expected metadata policy_reason_code to round-trip, got %+v", decoded.Metadata)
	}
	if decoded.Metadata.Additional["trace_id"] != "trace-1" {
		t.Fatalf("expected metadata extension to round-trip, got %+v", decoded.Metadata.Additional)
	}
	if decoded.Additional["custom_flag"] != "enabled" {
		t.Fatalf("expected payload extension to round-trip, got %+v", decoded.Additional)
	}
}

func TestRealtimeEventPayloadDeltaPreservesWhitespace(t *testing.T) {
	payload := RealtimeEventPayload{
		ItemType: "assistant_message",
		Delta:    " hello \n",
	}

	mapped := payload.AsMap()
	delta, ok := mapped["delta"].(string)
	if !ok {
		t.Fatalf("expected mapped delta string field, got %+v", mapped["delta"])
	}
	if delta != " hello \n" {
		t.Fatalf("expected raw delta whitespace to be preserved, got %q", delta)
	}

	decoded := RealtimeEventPayloadFromMap(mapped)
	if decoded.Delta != " hello \n" {
		t.Fatalf("expected decoded delta whitespace to be preserved, got %q", decoded.Delta)
	}
}

func TestUIStatusConfigurationTypedRoundTripWithExtensions(t *testing.T) {
	configuration := UIStatusConfigurationFromMap(map[string]any{
		"status_reason":            "worker_failed",
		"primary_connector_id":     "twilio",
		"mapped_connector_ids":     []string{"twilio", "imessage"},
		"execute_path_probe_ready": true,
		"custom_key":               "custom-value",
	})

	encoded, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("marshal ui status configuration: %v", err)
	}

	var decoded UIStatusConfiguration
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal ui status configuration: %v", err)
	}

	if decoded.StatusReason != "worker_failed" || decoded.PrimaryConnectorID != "twilio" {
		t.Fatalf("expected typed ui-status configuration fields to round-trip, got %+v", decoded)
	}
	if len(decoded.MappedConnectorIDs) != 2 {
		t.Fatalf("expected mapped connector ids to round-trip, got %+v", decoded.MappedConnectorIDs)
	}
	if decoded.ExecutePathProbeReady == nil || !*decoded.ExecutePathProbeReady {
		t.Fatalf("expected execute_path_probe_ready=true, got %+v", decoded.ExecutePathProbeReady)
	}
	if decoded.Additional["custom_key"] != "custom-value" {
		t.Fatalf("expected ui-status configuration extension to round-trip, got %+v", decoded.Additional)
	}
}

func TestUIStatusTestOperationDetailsTypedRoundTripWithExtensions(t *testing.T) {
	details := UIStatusTestOperationDetailsFromMap(map[string]any{
		"plugin_id":                      "mail.daemon",
		"worker_registered":              true,
		"execute_path_probe_status_code": 503,
		"probe_error":                    "timeout",
		"custom_probe_dimension":         "primary",
	})

	encoded, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal ui status test details: %v", err)
	}

	var decoded UIStatusTestOperationDetails
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal ui status test details: %v", err)
	}

	if decoded.PluginID != "mail.daemon" {
		t.Fatalf("expected plugin_id mail.daemon, got %+v", decoded)
	}
	if decoded.WorkerRegistered == nil || !*decoded.WorkerRegistered {
		t.Fatalf("expected worker_registered=true, got %+v", decoded.WorkerRegistered)
	}
	if decoded.ExecutePathProbeCode == nil || *decoded.ExecutePathProbeCode != 503 {
		t.Fatalf("expected execute_path_probe_status_code=503, got %+v", decoded.ExecutePathProbeCode)
	}
	if decoded.ProbeError != "timeout" {
		t.Fatalf("expected probe_error timeout, got %+v", decoded)
	}
	if decoded.Additional["custom_probe_dimension"] != "primary" {
		t.Fatalf("expected details extension to round-trip, got %+v", decoded.Additional)
	}
}

func TestReadAnyIntPointerRejectsUint64Overflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	overflow := uint64(maxInt) + 1
	if got := readAnyIntPointer(overflow); got != nil {
		t.Fatalf("expected overflow uint64 to be rejected, got %v", *got)
	}

	fit := uint64(maxInt)
	got := readAnyIntPointer(fit)
	if got == nil || *got != maxInt {
		t.Fatalf("expected max int conversion for uint64(%d), got %+v", fit, got)
	}
}

func TestReadAnyIntPointerUint32ConversionSafety(t *testing.T) {
	const safeValue = uint32(42)
	got := readAnyIntPointer(safeValue)
	if got == nil || *got != int(safeValue) {
		t.Fatalf("expected uint32(%d) conversion, got %+v", safeValue, got)
	}

	maxInt := int(^uint(0) >> 1)
	maxUint32 := uint64(^uint32(0))
	if uint64(maxInt) >= maxUint32 {
		// 64-bit platforms can represent all uint32 values without overflow.
		return
	}
	overflow := uint32(maxInt) + 1
	if got := readAnyIntPointer(overflow); got != nil {
		t.Fatalf("expected overflowing uint32 to be rejected on narrow-int platforms, got %v", *got)
	}
}

func TestReadAnyIntPointerRejectsJSONNumberOverflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	overflow := strconv.FormatUint(uint64(maxInt)+1, 10)
	if got := readAnyIntPointer(json.Number(overflow)); got != nil {
		t.Fatalf("expected overflow json.Number to be rejected, got %v", *got)
	}

	fit := strconv.Itoa(maxInt)
	got := readAnyIntPointer(json.Number(fit))
	if got == nil || *got != maxInt {
		t.Fatalf("expected max int conversion for json.Number(%q), got %+v", fit, got)
	}
}

func TestReadAnyIntPointerRejectsNonIntegralAndOverflowFloats(t *testing.T) {
	if got := readAnyIntPointer(42.5); got != nil {
		t.Fatalf("expected non-integral float to be rejected, got %v", *got)
	}
	if got := readAnyIntPointer(1e100); got != nil {
		t.Fatalf("expected overflow float to be rejected, got %v", *got)
	}
	got := readAnyIntPointer(float64(42))
	if got == nil || *got != 42 {
		t.Fatalf("expected integral float conversion, got %+v", got)
	}
}
