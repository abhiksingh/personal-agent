package authz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

const (
	ScopeTypeAll       = "ALL"
	ScopeTypeExecution = "EXECUTION"
)

type ActingAsAuthorizer struct {
	store contract.DelegationRuleStore
}

func NewActingAsAuthorizer(store contract.DelegationRuleStore) *ActingAsAuthorizer {
	return &ActingAsAuthorizer{store: store}
}

func (a *ActingAsAuthorizer) CanActAs(ctx context.Context, req types.ActingAsRequest) (types.ActingAsDecision, error) {
	if req.WorkspaceID == "" {
		return types.ActingAsDecision{}, fmt.Errorf("workspace_id is required")
	}
	if req.RequestedByActorID == "" {
		return types.ActingAsDecision{}, fmt.Errorf("requested_by_actor_id is required")
	}
	if req.ActingAsActorID == "" {
		return types.ActingAsDecision{}, fmt.Errorf("acting_as_actor_id is required")
	}

	if req.RequestedByActorID == req.ActingAsActorID {
		return types.ActingAsDecision{Allowed: true, Reason: "self-execution"}, nil
	}

	if a.store == nil {
		return types.ActingAsDecision{Allowed: false, Reason: "no delegation store configured"}, nil
	}

	evaluationTime := req.At
	if evaluationTime.IsZero() {
		evaluationTime = time.Now().UTC()
	}

	requiredScope := strings.ToUpper(strings.TrimSpace(req.ScopeType))
	if requiredScope == "" {
		requiredScope = ScopeTypeExecution
	}

	rules, err := a.store.ListActiveDelegationRules(
		ctx,
		req.WorkspaceID,
		req.RequestedByActorID,
		req.ActingAsActorID,
		evaluationTime,
	)
	if err != nil {
		return types.ActingAsDecision{}, fmt.Errorf("evaluate delegation rules: %w", err)
	}

	for _, rule := range rules {
		if !scopeMatches(rule.ScopeType, requiredScope) {
			continue
		}
		if !scopeKeyMatches(rule.ScopeKey, req.ScopeKey) {
			continue
		}
		return types.ActingAsDecision{
			Allowed:          true,
			Reason:           "matched delegation rule",
			DelegationRuleID: rule.ID,
		}, nil
	}

	return types.ActingAsDecision{Allowed: false, Reason: "missing valid delegation rule"}, nil
}

func scopeMatches(ruleScope, requiredScope string) bool {
	r := strings.ToUpper(strings.TrimSpace(ruleScope))
	if r == ScopeTypeAll {
		return true
	}
	return r == requiredScope
}

func scopeKeyMatches(ruleKey, requiredKey string) bool {
	if strings.TrimSpace(ruleKey) == "" {
		return true
	}
	return strings.TrimSpace(ruleKey) == strings.TrimSpace(requiredKey)
}
