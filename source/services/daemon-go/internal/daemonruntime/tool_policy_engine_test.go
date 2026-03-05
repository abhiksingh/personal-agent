package daemonruntime

import (
	"context"
	"testing"

	"personalagent/runtime/internal/transport"
)

type toolPolicyDelegationStub struct {
	checkResponse transport.DelegationCheckResponse
	checkErr      error
	checkRequests []transport.DelegationCheckRequest
}

func (s *toolPolicyDelegationStub) GrantDelegation(context.Context, transport.DelegationGrantRequest) (transport.DelegationRuleRecord, error) {
	return transport.DelegationRuleRecord{}, nil
}

func (s *toolPolicyDelegationStub) ListDelegations(context.Context, transport.DelegationListRequest) (transport.DelegationListResponse, error) {
	return transport.DelegationListResponse{}, nil
}

func (s *toolPolicyDelegationStub) RevokeDelegation(context.Context, transport.DelegationRevokeRequest) (transport.DelegationRevokeResponse, error) {
	return transport.DelegationRevokeResponse{}, nil
}

func (s *toolPolicyDelegationStub) CheckDelegation(_ context.Context, request transport.DelegationCheckRequest) (transport.DelegationCheckResponse, error) {
	s.checkRequests = append(s.checkRequests, request)
	if s.checkErr != nil {
		return transport.DelegationCheckResponse{}, s.checkErr
	}
	return s.checkResponse, nil
}

func (s *toolPolicyDelegationStub) UpsertCapabilityGrant(context.Context, transport.CapabilityGrantUpsertRequest) (transport.CapabilityGrantRecord, error) {
	return transport.CapabilityGrantRecord{}, nil
}

func (s *toolPolicyDelegationStub) ListCapabilityGrants(context.Context, transport.CapabilityGrantListRequest) (transport.CapabilityGrantListResponse, error) {
	return transport.CapabilityGrantListResponse{}, nil
}

func TestToolPolicyEngineAllowForNonDestructiveTool(t *testing.T) {
	engine := NewToolPolicyEngine(nil)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.a",
		ActingAsActorID:    "actor.a",
		ChannelID:          "app",
		ToolName:           "mail_send",
		CapabilityKey:      "mail_send",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if result.Decision != ToolPolicyDecisionAllow {
		t.Fatalf("expected ALLOW decision, got %+v", result)
	}
	if result.ReasonCode != "allowed" {
		t.Fatalf("expected reason_code=allowed, got %+v", result)
	}
}

func TestToolPolicyEngineRequiresApprovalForDestructiveVoiceTool(t *testing.T) {
	engine := NewToolPolicyEngine(nil)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.a",
		ActingAsActorID:    "actor.a",
		ChannelID:          "voice",
		ToolName:           "finder_delete",
		CapabilityKey:      "finder_delete",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if result.Decision != ToolPolicyDecisionRequireApproval {
		t.Fatalf("expected REQUIRE_APPROVAL decision, got %+v", result)
	}
	if result.ReasonCode != "approval_required" {
		t.Fatalf("expected reason_code=approval_required, got %+v", result)
	}
}

func TestToolPolicyEngineDeniesWhenDelegationCheckFails(t *testing.T) {
	delegation := &toolPolicyDelegationStub{checkResponse: transport.DelegationCheckResponse{
		Allowed: false,
		Reason:  "delegation missing",
	}}
	engine := NewToolPolicyEngine(delegation)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.delegate",
		ChannelID:          "app",
		ToolName:           "mail_send",
		CapabilityKey:      "mail_send",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if result.Decision != ToolPolicyDecisionDeny {
		t.Fatalf("expected DENY decision, got %+v", result)
	}
	if result.ReasonCode != "delegation_denied" {
		t.Fatalf("expected reason_code=delegation_denied, got %+v", result)
	}
	if len(delegation.checkRequests) != 1 {
		t.Fatalf("expected one delegation check request, got %d", len(delegation.checkRequests))
	}
}

func TestToolPolicyEngineAllowsDelegatedExecutionWhenRuleExists(t *testing.T) {
	delegation := &toolPolicyDelegationStub{checkResponse: transport.DelegationCheckResponse{
		Allowed: true,
		Reason:  "delegation rule matched",
	}}
	engine := NewToolPolicyEngine(delegation)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.delegate",
		ChannelID:          "message",
		ToolName:           "message_send",
		CapabilityKey:      "channel.messages.send",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if result.Decision != ToolPolicyDecisionAllow {
		t.Fatalf("expected ALLOW decision, got %+v", result)
	}
	if result.ReasonCode != "allowed" {
		t.Fatalf("expected reason_code=allowed, got %+v", result)
	}
	if len(delegation.checkRequests) != 1 {
		t.Fatalf("expected one delegation check request, got %d", len(delegation.checkRequests))
	}
}

func TestToolPolicyEngineDeniesWhenChannelIsDisallowedByCapabilityPolicy(t *testing.T) {
	engine := NewToolPolicyEngine(nil)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.a",
		ActingAsActorID:    "actor.a",
		ChannelID:          "voice",
		ToolName:           "messages_send_sms",
		CapabilityKey:      "messages_send_sms",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if result.Decision != ToolPolicyDecisionDeny {
		t.Fatalf("expected DENY decision for voice-restricted capability, got %+v", result)
	}
	if result.ReasonCode != "channel_not_allowed" {
		t.Fatalf("expected reason_code=channel_not_allowed, got %+v", result)
	}
}

func TestToolPolicyEngineMetadataMapIncludesDeterministicRationale(t *testing.T) {
	engine := NewToolPolicyEngine(nil)
	result, err := engine.Evaluate(context.Background(), ToolPolicyRequest{
		WorkspaceID:        "ws1",
		RequestedByActorID: "actor.a",
		ActingAsActorID:    "actor.a",
		ChannelID:          "voice",
		ToolName:           "finder_delete",
		CapabilityKey:      "finder_delete",
	})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	metadata := result.MetadataMap()
	if metadata["policy_decision"] != string(ToolPolicyDecisionRequireApproval) {
		t.Fatalf("expected metadata policy_decision REQUIRE_APPROVAL, got %+v", metadata)
	}
	if metadata["policy_reason_code"] != "approval_required" {
		t.Fatalf("expected metadata policy_reason_code approval_required, got %+v", metadata)
	}
	rationale, ok := metadata["policy_rationale"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy_rationale map, got %+v", metadata)
	}
	if rationale["policy_version"] != toolCapabilityPolicyVersionV1 {
		t.Fatalf("expected policy_version %s, got %+v", toolCapabilityPolicyVersionV1, rationale)
	}
	if rationale["capability_key"] != "finder_delete" {
		t.Fatalf("expected capability_key finder_delete, got %+v", rationale)
	}
	if rationale["decision_reason_code"] != "approval_required" {
		t.Fatalf("expected decision_reason_code approval_required, got %+v", rationale)
	}
}
