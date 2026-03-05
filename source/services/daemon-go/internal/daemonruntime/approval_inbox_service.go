package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

const (
	defaultApprovalInboxLimit = 25
	maxApprovalInboxLimit     = 200
)

type typedApprovalRationale struct {
	PolicyVersion      string  `json:"policy_version"`
	CapabilityKey      string  `json:"capability_key"`
	RiskLevel          string  `json:"risk_level"`
	RiskConfidence     float64 `json:"risk_confidence"`
	RiskReason         string  `json:"risk_reason"`
	DestructiveClass   string  `json:"destructive_class,omitempty"`
	Decision           string  `json:"decision"`
	DecisionReason     string  `json:"decision_reason"`
	DecisionReasonCode string  `json:"decision_reason_code"`
	DecisionSource     string  `json:"decision_source"`
	ExecutionOrigin    string  `json:"execution_origin"`
}

var _ transport.WorkflowQueryService = (*AgentDelegationService)(nil)

func (s *AgentDelegationService) ListApprovalInbox(ctx context.Context, request transport.ApprovalInboxRequest) (transport.ApprovalInboxResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.ApprovalInboxResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	stateFilter, err := normalizeApprovalInboxStateFilter(request.State)
	if err != nil {
		return transport.ApprovalInboxResponse{}, err
	}
	routeResolver := newWorkflowRouteMetadataResolver(s.container)

	query := `
		SELECT
			ar.id,
			ar.workspace_id,
			COALESCE(ar.run_id, ''),
			COALESCE(ar.step_id, ''),
			COALESCE(ar.requested_phrase, ''),
			COALESCE(ar.decision, ''),
			COALESCE(ar.decision_by_actor_id, ''),
			ar.requested_at,
			COALESCE(ar.decided_at, ''),
			COALESCE(ar.rationale, ''),
			COALESCE(tr.id, ''),
			COALESCE(tr.state, ''),
			COALESCE(tr.acting_as_actor_id, ''),
			COALESCE(t.id, ''),
			COALESCE(t.title, ''),
			COALESCE(t.state, ''),
			COALESCE(t.requested_by_actor_id, ''),
			COALESCE(t.subject_principal_actor_id, ''),
			COALESCE(t.channel, ''),
			COALESCE(ts.name, ''),
			COALESCE(ts.capability_key, '')
		FROM approval_requests ar
		LEFT JOIN task_steps ts ON ts.id = ar.step_id
		LEFT JOIN task_runs tr ON tr.id = COALESCE(ar.run_id, ts.run_id)
		LEFT JOIN tasks t ON t.id = tr.task_id
		WHERE ar.workspace_id = ?
	`
	params := []any{workspace}

	switch stateFilter {
	case "pending":
		query += " AND ar.decision IS NULL"
	case "final":
		query += " AND ar.decision IS NOT NULL"
	default:
		if !request.IncludeFinal {
			query += " AND ar.decision IS NULL"
		}
	}

	query += `
		ORDER BY
			CASE WHEN ar.decision IS NULL THEN 0 ELSE 1 END ASC,
			COALESCE(ar.decided_at, ar.requested_at) DESC,
			ar.requested_at DESC,
			ar.id DESC
		LIMIT ?
	`
	params = append(params, clampApprovalInboxLimit(request.Limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ApprovalInboxResponse{}, fmt.Errorf("list approval inbox: %w", err)
	}
	defer rows.Close()

	items := make([]transport.ApprovalInboxItem, 0)
	type routeHint struct {
		workspaceID    string
		taskID         string
		runID          string
		stepCapability string
		taskChannel    string
	}
	routeHints := make([]routeHint, 0)
	for rows.Next() {
		var (
			item            transport.ApprovalInboxItem
			approvalRunID   string
			resolvedRunID   string
			decisionRaw     string
			stepCapability  string
			requestedPhrase string
			taskChannel     string
		)
		if err := rows.Scan(
			&item.ApprovalRequestID,
			&item.WorkspaceID,
			&approvalRunID,
			&item.StepID,
			&requestedPhrase,
			&decisionRaw,
			&item.DecisionByActorID,
			&item.RequestedAt,
			&item.DecidedAt,
			&item.DecisionRationale,
			&resolvedRunID,
			&item.RunState,
			&item.ActingAsActorID,
			&item.TaskID,
			&item.TaskTitle,
			&item.TaskState,
			&item.RequestedByActorID,
			&item.SubjectPrincipalActorID,
			&taskChannel,
			&item.StepName,
			&stepCapability,
		); err != nil {
			return transport.ApprovalInboxResponse{}, fmt.Errorf("scan approval inbox row: %w", err)
		}

		item.RunID = firstNonEmpty(resolvedRunID, approvalRunID)
		item.StepCapabilityKey = strings.TrimSpace(stepCapability)
		item.RequestedPhrase = strings.TrimSpace(requestedPhrase)
		item.State = approvalInboxState(decisionRaw)
		item.Decision = normalizeApprovalDecision(decisionRaw)
		item.RiskLevel, item.RiskRationale, item.DecisionRationale = approvalRiskMetadata(item.RequestedPhrase, item.StepCapabilityKey, item.DecisionRationale)
		items = append(items, item)
		routeHints = append(routeHints, routeHint{
			workspaceID:    item.WorkspaceID,
			taskID:         item.TaskID,
			runID:          item.RunID,
			stepCapability: item.StepCapabilityKey,
			taskChannel:    taskChannel,
		})
	}
	if err := rows.Err(); err != nil {
		return transport.ApprovalInboxResponse{}, fmt.Errorf("iterate approval inbox rows: %w", err)
	}
	for idx := range items {
		hint := routeHints[idx]
		items[idx].Route = routeResolver.ResolveForTaskRun(
			ctx,
			hint.workspaceID,
			hint.taskID,
			hint.runID,
			hint.stepCapability,
			hint.taskChannel,
		)
	}

	return transport.ApprovalInboxResponse{
		WorkspaceID: workspace,
		Approvals:   items,
	}, nil
}

func normalizeApprovalInboxStateFilter(raw string) (string, error) {
	state := strings.ToLower(strings.TrimSpace(raw))
	switch state {
	case "", "pending", "final":
		return state, nil
	default:
		return "", fmt.Errorf("--state must be one of: pending|final")
	}
}

func clampApprovalInboxLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultApprovalInboxLimit
	case limit > maxApprovalInboxLimit:
		return maxApprovalInboxLimit
	default:
		return limit
	}
}

func approvalInboxState(decision string) string {
	if strings.TrimSpace(decision) == "" {
		return "pending"
	}
	return "final"
}

func normalizeApprovalDecision(decision string) string {
	switch strings.ToUpper(strings.TrimSpace(decision)) {
	case "":
		return ""
	case "APPROVED", "ACCEPTED":
		return "approved"
	case "REJECTED", "DENIED":
		return "rejected"
	default:
		return strings.ToLower(strings.TrimSpace(decision))
	}
}

func approvalRiskMetadata(requestedPhrase string, capability string, rawRationale string) (string, string, string) {
	trimmedRationale := strings.TrimSpace(rawRationale)
	if typed, ok := parseTypedApprovalRationale(trimmedRationale); ok {
		riskLevel := strings.TrimSpace(typed.RiskLevel)
		if riskLevel == "" {
			riskLevel = "policy"
		}
		riskReason := firstNonEmpty(strings.TrimSpace(typed.RiskReason), strings.TrimSpace(typed.DecisionReason))
		if riskReason == "" {
			riskReason = "Policy requested approval before execution."
		}
		decisionRationale := firstNonEmpty(strings.TrimSpace(typed.DecisionReason), trimmedRationale)
		return riskLevel, riskReason, decisionRationale
	}

	lowerCapability := strings.ToLower(strings.TrimSpace(capability))
	if strings.TrimSpace(requestedPhrase) == types.DestructiveApprovalPhrase || strings.Contains(lowerCapability, "delete") || strings.Contains(lowerCapability, "cancel") {
		return "destructive", "Destructive action requires explicit GO AHEAD approval before execution.", trimmedRationale
	}
	if strings.TrimSpace(capability) != "" {
		return "policy", fmt.Sprintf("Policy requested approval for capability %s.", strings.TrimSpace(capability)), trimmedRationale
	}
	return "policy", "Policy requested approval before execution.", trimmedRationale
}

func parseTypedApprovalRationale(raw string) (typedApprovalRationale, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return typedApprovalRationale{}, false
	}
	parsed := typedApprovalRationale{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return typedApprovalRationale{}, false
	}
	if strings.TrimSpace(parsed.RiskLevel) == "" && strings.TrimSpace(parsed.DecisionReason) == "" {
		return typedApprovalRationale{}, false
	}
	return parsed, true
}
