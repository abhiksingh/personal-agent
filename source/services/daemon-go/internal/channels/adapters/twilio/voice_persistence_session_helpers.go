package twilio

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type callSessionUpsertInput struct {
	WorkspaceID     string
	SessionID       string
	Provider        string
	ConnectorID     string
	ProviderCallID  string
	ProviderAccount string
	ThreadID        string
	Direction       string
	FromAddress     string
	ToAddress       string
	Status          string
	OccurredAt      time.Time
	Now             time.Time
}

func upsertCallSession(ctx context.Context, tx *sql.Tx, input callSessionUpsertInput) (string, error) {
	nowText := input.Now.UTC().Format(time.RFC3339Nano)
	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = input.Now.UTC()
	}
	occurredAtText := occurredAt.Format(time.RFC3339Nano)

	var (
		currentStatus sql.NullString
		startedAt     sql.NullString
		endedAt       sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
		SELECT status, started_at, ended_at
		FROM comm_call_sessions
		WHERE workspace_id = ? AND provider = ? AND provider_call_id = ?
		LIMIT 1
	`, input.WorkspaceID, input.Provider, input.ProviderCallID).Scan(&currentStatus, &startedAt, &endedAt)
	if err == sql.ErrNoRows {
		resolvedStatus := normalizeVoiceCallStatus(input.Status)
		endValue := nullableTimestampIfTerminal(resolvedStatus, occurredAtText)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_call_sessions(
				id, workspace_id, provider, connector_id, provider_call_id, provider_account_id, thread_id, direction,
				from_address, to_address, status, started_at, ended_at, last_error, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
		`, input.SessionID, input.WorkspaceID, input.Provider, strings.TrimSpace(input.ConnectorID), input.ProviderCallID, nullable(input.ProviderAccount), input.ThreadID, input.Direction, nullable(input.FromAddress), nullable(input.ToAddress), resolvedStatus, occurredAtText, endValue, nowText, nowText); err != nil {
			return "", fmt.Errorf("insert comm call session: %w", err)
		}
		return resolvedStatus, nil
	}
	if err != nil {
		return "", fmt.Errorf("load comm call session: %w", err)
	}

	current := normalizeVoiceCallStatus(currentStatus.String)
	next := resolveMonotonicCallStatus(current, input.Status)

	startValue := nullable(startedAt.String)
	if strings.TrimSpace(startedAt.String) == "" {
		startValue = occurredAtText
	}
	endValue := nullable(endedAt.String)
	if strings.TrimSpace(endedAt.String) == "" && isTerminalVoiceStatus(next) {
		endValue = occurredAtText
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_call_sessions
		SET provider_account_id = COALESCE(?, provider_account_id),
			connector_id = COALESCE(NULLIF(?, ''), connector_id),
			thread_id = ?,
			direction = ?,
			from_address = ?,
			to_address = ?,
			status = ?,
			started_at = ?,
			ended_at = ?,
			updated_at = ?
		WHERE workspace_id = ? AND provider = ? AND provider_call_id = ?
	`, nullable(input.ProviderAccount), strings.TrimSpace(input.ConnectorID), input.ThreadID, input.Direction, nullable(input.FromAddress), nullable(input.ToAddress), next, startValue, endValue, nowText, input.WorkspaceID, input.Provider, input.ProviderCallID); err != nil {
		return "", fmt.Errorf("update comm call session: %w", err)
	}
	return next, nil
}

func loadCallSessionThreadID(ctx context.Context, tx *sql.Tx, workspaceID string, providerCallID string) (string, error) {
	var threadID string
	err := tx.QueryRowContext(ctx, `
		SELECT thread_id
		FROM comm_call_sessions
		WHERE workspace_id = ? AND provider = ? AND provider_call_id = ?
		LIMIT 1
	`, workspaceID, providerNameTwilio, strings.TrimSpace(providerCallID)).Scan(&threadID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(threadID), nil
}

func resolveMonotonicCallStatus(current string, incoming string) string {
	currentNormalized := normalizeVoiceCallStatus(current)
	incomingNormalized := normalizeVoiceCallStatus(incoming)
	if currentNormalized == "" {
		return incomingNormalized
	}
	if incomingNormalized == "" {
		return currentNormalized
	}
	if currentNormalized == incomingNormalized {
		return currentNormalized
	}
	if isTerminalVoiceStatus(currentNormalized) {
		return currentNormalized
	}
	currentRank := callStatusRank(currentNormalized)
	incomingRank := callStatusRank(incomingNormalized)
	if incomingRank < currentRank {
		return currentNormalized
	}
	if incomingRank == currentRank {
		return currentNormalized
	}
	return incomingNormalized
}

func callStatusRank(status string) int {
	switch normalizeVoiceCallStatus(status) {
	case "initiated":
		return 1
	case "ringing":
		return 2
	case "in_progress":
		return 3
	case "completed", "failed", "no_answer", "busy", "canceled":
		return 4
	default:
		return 0
	}
}

func isTerminalVoiceStatus(status string) bool {
	switch normalizeVoiceCallStatus(status) {
	case "completed", "failed", "no_answer", "busy", "canceled":
		return true
	default:
		return false
	}
}

func normalizeVoiceCallStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	switch normalized {
	case "", "queued", "initiated":
		return "initiated"
	case "ringing":
		return "ringing"
	case "in_progress", "inprogress":
		return "in_progress"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "no_answer", "noanswer":
		return "no_answer"
	case "busy":
		return "busy"
	case "canceled", "cancelled":
		return "canceled"
	default:
		return normalized
	}
}

func normalizeCallDirection(direction string) string {
	normalized := strings.ToLower(strings.TrimSpace(direction))
	switch normalized {
	case "", "inbound", "inbound_api":
		return "inbound"
	case "outbound", "outbound_api", "outbound_dial":
		return "outbound"
	default:
		return normalized
	}
}

func normalizeEventDirection(direction string) string {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	switch normalized {
	case "", "INBOUND":
		return "INBOUND"
	case "OUTBOUND":
		return "OUTBOUND"
	default:
		return normalized
	}
}

func callEventDirection(callDirection string) string {
	switch normalizeCallDirection(callDirection) {
	case "outbound":
		return "OUTBOUND"
	default:
		return "INBOUND"
	}
}

func voiceExternalRef(localAddress string, remoteAddress string) string {
	return fmt.Sprintf("twilio:voice:%s:%s", normalizeAddress(localAddress), normalizeAddress(remoteAddress))
}

func nullableTimestampIfTerminal(status string, value string) any {
	if !isTerminalVoiceStatus(status) {
		return nil
	}
	return nullable(value)
}
