package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func normalizeDaemonPluginLifecycleKindFilter(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "":
		return "", nil
	case "channel", "connector":
		return normalized, nil
	default:
		return "", fmt.Errorf("kind must be one of channel|connector")
	}
}

func normalizeDaemonPluginLifecycleStateFilter(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "":
		return "", nil
	case string(PluginWorkerStateRegistered),
		string(PluginWorkerStateStarting),
		string(PluginWorkerStateRunning),
		string(PluginWorkerStateRestarting),
		string(PluginWorkerStateStopped),
		string(PluginWorkerStateFailed):
		return normalized, nil
	default:
		return "", fmt.Errorf("state must be one of registered|starting|running|restarting|stopped|failed")
	}
}

func normalizeDaemonPluginLifecycleEventTypeFilter(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "" {
		return "", nil
	}
	for _, eventType := range daemonPluginLifecycleEventTypes() {
		if normalized == eventType {
			return normalized, nil
		}
	}
	return "", fmt.Errorf(
		"event_type must be one of %s",
		strings.Join(daemonPluginLifecycleEventTypes(), "|"),
	)
}

func normalizeDaemonPluginLifecycleCursor(createdAt string, id string) (string, string, error) {
	cursorCreatedAt := strings.TrimSpace(createdAt)
	cursorID := strings.TrimSpace(id)
	if cursorCreatedAt == "" {
		if cursorID != "" {
			return "", "", fmt.Errorf("cursor_created_at is required when cursor_id is provided")
		}
		return "", "", nil
	}
	if _, err := time.Parse(time.RFC3339Nano, cursorCreatedAt); err != nil {
		return "", "", fmt.Errorf("cursor_created_at must be RFC3339 timestamp: %w", err)
	}
	if cursorID == "" {
		cursorID = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	}
	return cursorCreatedAt, cursorID, nil
}

func clampDaemonPluginLifecycleHistoryLimit(raw int) int {
	switch {
	case raw <= 0:
		return daemonPluginLifecycleHistoryDefaultLimit
	case raw > daemonPluginLifecycleHistoryMaxLimit:
		return daemonPluginLifecycleHistoryMaxLimit
	default:
		return raw
	}
}

func daemonPluginLifecycleEventTypes() []string {
	return []string{
		pluginEventWorkerStarted,
		pluginEventHandshakeAccepted,
		pluginEventHealthTimeout,
		pluginEventWorkerRestarting,
		pluginEventWorkerExited,
		pluginEventWorkerStopped,
		pluginEventWorkerRestartLimit,
	}
}

func (s *DaemonLifecycleService) queryDaemonPluginLifecycleAuditRows(
	ctx context.Context,
	workspaceID string,
	pluginID string,
	eventType string,
	cursorCreatedAt string,
	cursorID string,
	limit int,
) ([]daemonPluginLifecycleAuditRow, error) {
	query := `
		SELECT
			al.id,
			al.workspace_id,
			al.event_type,
			COALESCE(al.correlation_id, ''),
			COALESCE(al.payload_json, ''),
			al.created_at
		FROM audit_log_entries al
		WHERE al.workspace_id = ?
	`
	params := []any{workspaceID}

	if strings.TrimSpace(eventType) != "" {
		query += " AND al.event_type = ?"
		params = append(params, strings.TrimSpace(eventType))
	} else {
		eventTypes := daemonPluginLifecycleEventTypes()
		query += " AND al.event_type IN (" + strings.TrimSuffix(strings.Repeat("?,", len(eventTypes)), ",") + ")"
		for _, value := range eventTypes {
			params = append(params, value)
		}
	}
	if strings.TrimSpace(pluginID) != "" {
		query += " AND al.correlation_id = ?"
		params = append(params, strings.TrimSpace(pluginID))
	}
	if strings.TrimSpace(cursorCreatedAt) != "" {
		query += " AND (al.created_at < ? OR (al.created_at = ? AND al.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}

	query += " ORDER BY al.created_at DESC, al.id DESC LIMIT ?"
	params = append(params, limit)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("query daemon plugin lifecycle history: %w", err)
	}
	defer rows.Close()

	items := make([]daemonPluginLifecycleAuditRow, 0)
	for rows.Next() {
		var row daemonPluginLifecycleAuditRow
		if err := rows.Scan(
			&row.AuditID,
			&row.WorkspaceID,
			&row.EventType,
			&row.CorrelationID,
			&row.PayloadJSON,
			&row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan daemon plugin lifecycle row: %w", err)
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daemon plugin lifecycle rows: %w", err)
	}
	return items, nil
}

func daemonPluginLifecycleRecordFromAuditRow(row daemonPluginLifecycleAuditRow) transport.DaemonPluginLifecycleHistoryRecord {
	payload := daemonPluginLifecycleAuditPayload{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.PayloadJSON)), &payload)

	pluginID := strings.TrimSpace(payload.PluginID)
	if pluginID == "" {
		pluginID = strings.TrimSpace(row.CorrelationID)
	}
	kind := strings.ToLower(strings.TrimSpace(payload.Kind))
	if kind == "" {
		kind = "unknown"
	}
	state := strings.ToLower(strings.TrimSpace(payload.State))
	if state == "" {
		state = "unknown"
	}
	errorText := strings.TrimSpace(payload.Error)
	eventType := strings.TrimSpace(row.EventType)
	restartEvent := daemonPluginLifecycleIsRestartEvent(eventType)
	failureEvent := daemonPluginLifecycleIsFailureEvent(eventType)
	recoveryEvent := daemonPluginLifecycleIsRecoveryEvent(eventType, payload.RestartCount)

	return transport.DaemonPluginLifecycleHistoryRecord{
		AuditID:          strings.TrimSpace(row.AuditID),
		WorkspaceID:      strings.TrimSpace(row.WorkspaceID),
		PluginID:         pluginID,
		Kind:             kind,
		State:            state,
		EventType:        eventType,
		ProcessID:        payload.ProcessID,
		RestartCount:     payload.RestartCount,
		Reason:           daemonPluginLifecycleReason(eventType, errorText, recoveryEvent),
		Error:            errorText,
		ErrorSource:      strings.TrimSpace(payload.ErrorSource),
		ErrorOperation:   strings.TrimSpace(payload.ErrorOperation),
		ErrorStderr:      strings.TrimSpace(payload.ErrorStderr),
		RestartEvent:     restartEvent,
		FailureEvent:     failureEvent,
		RecoveryEvent:    recoveryEvent,
		LastHeartbeatAt:  strings.TrimSpace(payload.LastHeartbeatAt),
		LastTransitionAt: strings.TrimSpace(payload.LastTransitionAt),
		OccurredAt:       strings.TrimSpace(row.CreatedAt),
	}
}

func daemonPluginLifecycleIsRestartEvent(eventType string) bool {
	return strings.EqualFold(strings.TrimSpace(eventType), pluginEventWorkerRestarting)
}

func daemonPluginLifecycleIsFailureEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case pluginEventHealthTimeout, pluginEventWorkerExited, pluginEventWorkerRestartLimit:
		return true
	default:
		return false
	}
}

func daemonPluginLifecycleIsRecoveryEvent(eventType string, restartCount int) bool {
	return strings.EqualFold(strings.TrimSpace(eventType), pluginEventHandshakeAccepted) && restartCount > 0
}

func daemonPluginLifecycleReason(eventType string, errorText string, recoveryEvent bool) string {
	switch strings.TrimSpace(eventType) {
	case pluginEventWorkerStarted:
		return "worker_started"
	case pluginEventHandshakeAccepted:
		if recoveryEvent {
			return "worker_recovered"
		}
		return "handshake_accepted"
	case pluginEventHealthTimeout:
		return "health_timeout"
	case pluginEventWorkerRestarting:
		if strings.TrimSpace(errorText) != "" {
			return "restart_after_error"
		}
		return "restart_requested"
	case pluginEventWorkerExited:
		if strings.TrimSpace(errorText) != "" {
			return "worker_exited_error"
		}
		return "worker_exited"
	case pluginEventWorkerStopped:
		return "worker_stopped"
	case pluginEventWorkerRestartLimit:
		return "restart_limit_reached"
	default:
		normalized := strings.ToLower(strings.TrimSpace(eventType))
		if normalized == "" {
			return "unknown"
		}
		return normalized
	}
}

func formatRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
