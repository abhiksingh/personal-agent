package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	authzservice "personalagent/runtime/internal/core/service/authz"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *AgentDelegationService) GrantDelegation(ctx context.Context, request transport.DelegationGrantRequest) (transport.DelegationRuleRecord, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.DelegationRuleRecord{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	from := strings.TrimSpace(request.FromActorID)
	to := strings.TrimSpace(request.ToActorID)
	if from == "" || to == "" {
		return transport.DelegationRuleRecord{}, fmt.Errorf("--from and --to are required")
	}
	if from == to {
		return transport.DelegationRuleRecord{}, fmt.Errorf("delegation denied: self delegation is not allowed")
	}

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	expiresText := ""
	if strings.TrimSpace(request.ExpiresAt) != "" {
		parsed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.ExpiresAt))
		if parseErr != nil {
			return transport.DelegationRuleRecord{}, fmt.Errorf("invalid --expires-at: %w", parseErr)
		}
		if !parsed.UTC().After(now) {
			return transport.DelegationRuleRecord{}, fmt.Errorf("delegation denied: expires_at must be in the future")
		}
		expiresText = parsed.UTC().Format(time.RFC3339Nano)
	}
	scopeType, err := normalizeDelegationScopeType(strings.TrimSpace(request.ScopeType), true)
	if err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	scopeKey := strings.TrimSpace(request.ScopeKey)
	if scopeType == authzservice.ScopeTypeAll && scopeKey != "" {
		return transport.DelegationRuleRecord{}, fmt.Errorf("delegation denied: scope_key cannot be set when scope_type=ALL")
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.DelegationRuleRecord{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(ctx, tx, workspace, nowText); err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspace, from, nowText); err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspace, to, nowText); err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	if active, principalErr := isActiveWorkspacePrincipal(ctx, tx, workspace, from); principalErr != nil {
		return transport.DelegationRuleRecord{}, principalErr
	} else if !active {
		return transport.DelegationRuleRecord{}, fmt.Errorf("delegation denied: from_actor_id %q is not an active workspace principal", from)
	}
	if active, principalErr := isActiveWorkspacePrincipal(ctx, tx, workspace, to); principalErr != nil {
		return transport.DelegationRuleRecord{}, principalErr
	} else if !active {
		return transport.DelegationRuleRecord{}, fmt.Errorf("delegation denied: to_actor_id %q is not an active workspace principal", to)
	}

	ruleID, err := delegationRandomID()
	if err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO delegation_rules(
			id, workspace_id, from_actor_id, to_actor_id, scope_type, scope_key, status, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
	`, ruleID, workspace, from, to, scopeType, delegationNullableText(scopeKey), nowText, delegationNullableText(expiresText)); err != nil {
		return transport.DelegationRuleRecord{}, fmt.Errorf("insert delegation rule: %w", err)
	}
	if err := appendDelegationAuditEntry(ctx, tx, workspace, "DELEGATION_RULE_GRANTED", from, to, map[string]any{
		"rule_id":       ruleID,
		"from_actor_id": from,
		"to_actor_id":   to,
		"scope_type":    scopeType,
		"scope_key":     scopeKey,
		"expires_at":    expiresText,
	}); err != nil {
		return transport.DelegationRuleRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return transport.DelegationRuleRecord{}, fmt.Errorf("commit tx: %w", err)
	}

	return transport.DelegationRuleRecord{
		ID:          ruleID,
		WorkspaceID: workspace,
		FromActorID: from,
		ToActorID:   to,
		ScopeType:   scopeType,
		ScopeKey:    scopeKey,
		Status:      "ACTIVE",
		CreatedAt:   nowText,
		ExpiresAt:   expiresText,
	}, nil
}

func (s *AgentDelegationService) ListDelegations(ctx context.Context, request transport.DelegationListRequest) (transport.DelegationListResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.DelegationListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	query := `
		SELECT id, workspace_id, from_actor_id, to_actor_id, scope_type,
		       COALESCE(scope_key, ''), status, created_at, COALESCE(expires_at, '')
		FROM delegation_rules
		WHERE workspace_id = ?
	`
	params := []any{workspace}
	if strings.TrimSpace(request.FromActorID) != "" {
		query += " AND from_actor_id = ?"
		params = append(params, strings.TrimSpace(request.FromActorID))
	}
	if strings.TrimSpace(request.ToActorID) != "" {
		query += " AND to_actor_id = ?"
		params = append(params, strings.TrimSpace(request.ToActorID))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.DelegationListResponse{}, fmt.Errorf("list delegation rules: %w", err)
	}
	defer rows.Close()

	rules := make([]transport.DelegationRuleRecord, 0)
	for rows.Next() {
		var rule transport.DelegationRuleRecord
		if err := rows.Scan(
			&rule.ID,
			&rule.WorkspaceID,
			&rule.FromActorID,
			&rule.ToActorID,
			&rule.ScopeType,
			&rule.ScopeKey,
			&rule.Status,
			&rule.CreatedAt,
			&rule.ExpiresAt,
		); err != nil {
			return transport.DelegationListResponse{}, fmt.Errorf("scan delegation rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return transport.DelegationListResponse{}, fmt.Errorf("iterate delegation rules: %w", err)
	}

	return transport.DelegationListResponse{
		WorkspaceID: workspace,
		Rules:       rules,
	}, nil
}

func (s *AgentDelegationService) RevokeDelegation(ctx context.Context, request transport.DelegationRevokeRequest) (transport.DelegationRevokeResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.DelegationRevokeResponse{}, fmt.Errorf("database is not configured")
	}
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	ruleID := strings.TrimSpace(request.RuleID)
	if ruleID == "" {
		return transport.DelegationRevokeResponse{}, fmt.Errorf("--rule-id is required")
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.DelegationRevokeResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var fromActorID string
	var toActorID string
	if err := tx.QueryRowContext(ctx, `
		SELECT from_actor_id, to_actor_id
		FROM delegation_rules
		WHERE id = ? AND workspace_id = ?
	`, ruleID, workspace).Scan(&fromActorID, &toActorID); err != nil {
		if err == sql.ErrNoRows {
			return transport.DelegationRevokeResponse{}, fmt.Errorf("delegation rule not found")
		}
		return transport.DelegationRevokeResponse{}, fmt.Errorf("load delegation rule: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE delegation_rules
		SET status = 'REVOKED'
		WHERE id = ? AND workspace_id = ?
	`, ruleID, workspace); err != nil {
		return transport.DelegationRevokeResponse{}, fmt.Errorf("revoke delegation rule: %w", err)
	}
	if err := appendDelegationAuditEntry(ctx, tx, workspace, "DELEGATION_RULE_REVOKED", fromActorID, toActorID, map[string]any{
		"rule_id":       ruleID,
		"from_actor_id": fromActorID,
		"to_actor_id":   toActorID,
		"status":        "REVOKED",
	}); err != nil {
		return transport.DelegationRevokeResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return transport.DelegationRevokeResponse{}, fmt.Errorf("commit tx: %w", err)
	}
	return transport.DelegationRevokeResponse{
		WorkspaceID: workspace,
		RuleID:      ruleID,
		Status:      "REVOKED",
	}, nil
}

func (s *AgentDelegationService) CheckDelegation(ctx context.Context, request transport.DelegationCheckRequest) (transport.DelegationCheckResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	requestedBy := strings.TrimSpace(request.RequestedByActorID)
	actingAs := strings.TrimSpace(request.ActingAsActorID)
	if requestedBy == "" || actingAs == "" {
		return transport.DelegationCheckResponse{}, fmt.Errorf("--requested-by and --acting-as are required")
	}
	scopeType, scopeErr := normalizeDelegationScopeType(strings.TrimSpace(request.ScopeType), true)
	if scopeErr != nil {
		return transport.DelegationCheckResponse{
			WorkspaceID:        workspace,
			RequestedByActorID: requestedBy,
			ActingAsActorID:    actingAs,
			Allowed:            false,
			Reason:             scopeErr.Error(),
			ReasonCode:         "invalid_scope_type",
		}, nil
	}
	scopeKey := strings.TrimSpace(request.ScopeKey)
	if scopeType == authzservice.ScopeTypeAll && scopeKey != "" {
		return transport.DelegationCheckResponse{
			WorkspaceID:        workspace,
			RequestedByActorID: requestedBy,
			ActingAsActorID:    actingAs,
			Allowed:            false,
			Reason:             "scope_key is not allowed when scope_type=ALL",
			ReasonCode:         "scope_key_not_allowed_for_all",
		}, nil
	}
	if s.container != nil && s.container.DB != nil {
		if active, err := isActiveWorkspacePrincipal(ctx, s.container.DB, workspace, requestedBy); err != nil {
			return transport.DelegationCheckResponse{}, err
		} else if !active {
			return transport.DelegationCheckResponse{
				WorkspaceID:        workspace,
				RequestedByActorID: requestedBy,
				ActingAsActorID:    actingAs,
				Allowed:            false,
				Reason:             "requested_by actor is not an active workspace principal",
				ReasonCode:         "requested_by_not_workspace_principal",
			}, nil
		}
		if active, err := isActiveWorkspacePrincipal(ctx, s.container.DB, workspace, actingAs); err != nil {
			return transport.DelegationCheckResponse{}, err
		} else if !active {
			return transport.DelegationCheckResponse{
				WorkspaceID:        workspace,
				RequestedByActorID: requestedBy,
				ActingAsActorID:    actingAs,
				Allowed:            false,
				Reason:             "acting_as actor is not an active workspace principal",
				ReasonCode:         "acting_as_not_workspace_principal",
			}, nil
		}
	}

	decision, err := s.authorizer.CanActAs(ctx, types.ActingAsRequest{
		WorkspaceID:        workspace,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    actingAs,
		ScopeType:          scopeType,
		ScopeKey:           scopeKey,
	})
	if err != nil {
		return transport.DelegationCheckResponse{}, err
	}

	return transport.DelegationCheckResponse{
		WorkspaceID:        workspace,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    actingAs,
		Allowed:            decision.Allowed,
		Reason:             decision.Reason,
		ReasonCode:         delegationDecisionReasonCode(decision),
		DelegationRuleID:   decision.DelegationRuleID,
	}, nil
}

func normalizeDelegationScopeType(raw string, allowDefault bool) (string, error) {
	scope := strings.ToUpper(strings.TrimSpace(raw))
	if scope == "" && allowDefault {
		scope = authzservice.ScopeTypeExecution
	}
	switch scope {
	case authzservice.ScopeTypeExecution, "APPROVAL", authzservice.ScopeTypeAll:
		return scope, nil
	default:
		return "", fmt.Errorf("delegation denied: unsupported scope_type %q (allowed: EXECUTION|APPROVAL|ALL)", strings.TrimSpace(raw))
	}
}

func delegationDecisionReasonCode(decision types.ActingAsDecision) string {
	reason := strings.ToLower(strings.TrimSpace(decision.Reason))
	switch reason {
	case "self-execution":
		return "self_execution"
	case "matched delegation rule":
		return "delegation_rule_matched"
	case "missing valid delegation rule":
		return "missing_delegation_rule"
	case "no delegation store configured":
		return "delegation_store_unavailable"
	default:
		if decision.Allowed {
			return "allowed"
		}
		return "denied"
	}
}

func normalizeActorID(actorID string, fallback string) string {
	trimmed := strings.TrimSpace(actorID)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
