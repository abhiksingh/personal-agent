package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func resolveAutomationFilterJSON(triggerType string, intervalSeconds int, filter *transport.AutomationCommTriggerFilter) (string, error) {
	switch triggerType {
	case "SCHEDULE":
		if filter != nil {
			return "", fmt.Errorf("--filter can only be set for ON_COMM_EVENT triggers")
		}
		config := types.ScheduleConfig{IntervalSeconds: intervalSeconds}
		if config.IntervalSeconds <= 0 {
			config.IntervalSeconds = 300
		}
		bytes, err := json.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("marshal schedule filter: %w", err)
		}
		return string(bytes), nil
	case "ON_COMM_EVENT":
		if filter == nil {
			return "{}", nil
		}
		return marshalAutomationCommTriggerFilterJSON(normalizeAutomationCommTriggerFilter(*filter)), nil
	default:
		return "", fmt.Errorf("unsupported trigger type %q", triggerType)
	}
}

type automationTriggerRecordQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func loadAutomationTriggerRecord(ctx context.Context, query automationTriggerRecordQuery, workspaceID string, triggerID string) (transport.AutomationTriggerRecord, error) {
	var (
		record      transport.AutomationTriggerRecord
		enabledFlag int
	)
	err := query.QueryRowContext(ctx, `
		SELECT
			at.id,
			at.workspace_id,
			at.directive_id,
			at.trigger_type,
			at.is_enabled,
			COALESCE(at.filter_json, '{}'),
			COALESCE(at.cooldown_seconds, 0),
			d.subject_principal_actor_id,
			d.title,
			d.instruction,
			d.status,
			at.created_at,
			at.updated_at
		FROM automation_triggers at
		JOIN directives d ON d.id = at.directive_id
		WHERE at.workspace_id = ? AND at.id = ?
	`, workspaceID, triggerID).Scan(
		&record.TriggerID,
		&record.WorkspaceID,
		&record.DirectiveID,
		&record.TriggerType,
		&enabledFlag,
		&record.FilterJSON,
		&record.CooldownSeconds,
		&record.SubjectPrincipalActor,
		&record.DirectiveTitle,
		&record.DirectiveInstruction,
		&record.DirectiveStatus,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return transport.AutomationTriggerRecord{}, err
	}
	record.Enabled = enabledFlag == 1
	return record, nil
}

func resolveAutomationUpdatedFilterJSON(
	triggerType string,
	existingFilterJSON string,
	intervalSeconds *int,
	filter *transport.AutomationCommTriggerFilter,
) (string, error) {
	if filter != nil {
		if triggerType != "ON_COMM_EVENT" {
			return "", fmt.Errorf("--filter can only be set for ON_COMM_EVENT triggers")
		}
		return marshalAutomationCommTriggerFilterJSON(normalizeAutomationCommTriggerFilter(*filter)), nil
	}
	if intervalSeconds != nil {
		if triggerType != "SCHEDULE" {
			return "", fmt.Errorf("--interval-seconds can only be set for SCHEDULE triggers")
		}
		return resolveAutomationFilterJSON(triggerType, *intervalSeconds, nil)
	}
	return strings.TrimSpace(existingFilterJSON), nil
}

func normalizeAutomationFireHistoryStatusFilter(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", "all":
		return "", nil
	case "pending", "created_task", "failed":
		return normalized, nil
	default:
		return "", fmt.Errorf("--status must be one of pending|created_task|failed")
	}
}

func automationFireOutcomeForStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "PENDING"
	case "created_task":
		return "CREATED_TASK"
	case "failed":
		return "FAILED"
	default:
		return ""
	}
}

func automationFireStatusFromOutcome(outcome string) string {
	switch strings.ToUpper(strings.TrimSpace(outcome)) {
	case "PENDING":
		return "pending"
	case "CREATED_TASK":
		return "created_task"
	case "FAILED":
		return "failed"
	case "":
		return "unknown"
	default:
		return strings.ToLower(strings.TrimSpace(outcome))
	}
}

func buildAutomationFireIdempotencyKey(workspaceID string, triggerID string, sourceEventID string) string {
	return strings.TrimSpace(workspaceID) + ":" + strings.TrimSpace(triggerID) + ":" + strings.TrimSpace(sourceEventID)
}

func automationRecordUnchanged(
	existing transport.AutomationTriggerRecord,
	subject string,
	title string,
	instruction string,
	enabled bool,
	filterJSON string,
	cooldownSeconds int,
) bool {
	return subject == existing.SubjectPrincipalActor &&
		title == existing.DirectiveTitle &&
		instruction == existing.DirectiveInstruction &&
		enabled == existing.Enabled &&
		cooldownSeconds == existing.CooldownSeconds &&
		normalizeJSONForCompare(filterJSON) == normalizeJSONForCompare(existing.FilterJSON)
}

func normalizeJSONForCompare(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return trimmed
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return trimmed
	}
	return string(normalized)
}

func automationRandomID(prefix string) (string, error) {
	id, err := daemonRandomID()
	if err != nil {
		return "", err
	}
	prefixValue := strings.TrimSpace(prefix)
	if prefixValue == "" {
		return id, nil
	}
	return prefixValue + "-" + id, nil
}
