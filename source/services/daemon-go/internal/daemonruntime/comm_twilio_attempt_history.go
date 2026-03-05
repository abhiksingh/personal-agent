package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

type commAttemptHistoryQueryInput struct {
	WorkspaceID       string
	OperationID       string
	ThreadID          string
	TaskID            string
	RunID             string
	StepID            string
	Channel           string
	Status            string
	CursorAttemptedAt string
	CursorID          string
	Limit             int
}

type commAttemptRouteMetadata struct {
	RoutePhase          string
	RetryOrdinal        int
	FallbackFromChannel string
}

type commAttemptRouteRow struct {
	AttemptID   string
	OperationID string
	Channel     string
	RouteIndex  int
}

func normalizeCommAttemptHistoryLimit(raw int) (int, error) {
	if raw == 0 {
		return commAttemptHistoryDefaultLimit, nil
	}
	if raw < 0 {
		return 0, fmt.Errorf("limit must be >= 0")
	}
	if raw > commAttemptHistoryMaxLimit {
		return commAttemptHistoryMaxLimit, nil
	}
	return raw, nil
}

func parseCommAttemptHistoryCursor(raw string) (string, string, error) {
	cursor := strings.TrimSpace(raw)
	if cursor == "" {
		return "", "", nil
	}
	parts := strings.Split(cursor, "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("cursor must match <attempted_at>|<attempt_id>")
	}
	attemptedAt := strings.TrimSpace(parts[0])
	attemptID := strings.TrimSpace(parts[1])
	if attemptedAt == "" || attemptID == "" {
		return "", "", fmt.Errorf("cursor must include attempted_at and attempt_id")
	}
	if _, err := time.Parse(time.RFC3339Nano, attemptedAt); err != nil {
		return "", "", fmt.Errorf("cursor timestamp must be RFC3339Nano")
	}
	return attemptedAt, attemptID, nil
}

func encodeCommAttemptHistoryCursor(attemptedAt string, attemptID string) string {
	return strings.TrimSpace(attemptedAt) + "|" + strings.TrimSpace(attemptID)
}

func queryCommAttemptHistory(ctx context.Context, db *sql.DB, input commAttemptHistoryQueryInput) ([]transport.CommAttemptRecord, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT d.id,
		       d.workspace_id,
		       COALESCE(d.step_id, ''),
		       COALESCE(d.event_id, ''),
		       d.destination_endpoint,
		       d.idempotency_key,
		       d.channel,
		       d.status,
		       COALESCE(d.provider_receipt, ''),
		       COALESCE(d.error, ''),
		       d.attempted_at,
		       COALESCE(e.thread_id, ''),
		       COALESCE(r.task_id, ''),
		       COALESCE(r.id, '')
		FROM delivery_attempts d
		LEFT JOIN comm_events e ON e.id = d.event_id
		LEFT JOIN task_steps s ON s.id = d.step_id
		LEFT JOIN task_runs r ON r.id = s.run_id
		WHERE d.workspace_id = ?
	`)
	args := []any{input.WorkspaceID}

	if strings.TrimSpace(input.OperationID) != "" {
		queryBuilder.WriteString(" AND d.idempotency_key LIKE ?")
		args = append(args, strings.TrimSpace(input.OperationID)+"|%")
	}
	if strings.TrimSpace(input.ThreadID) != "" {
		queryBuilder.WriteString(" AND e.thread_id = ?")
		args = append(args, strings.TrimSpace(input.ThreadID))
	}
	if strings.TrimSpace(input.TaskID) != "" {
		queryBuilder.WriteString(" AND r.task_id = ?")
		args = append(args, strings.TrimSpace(input.TaskID))
	}
	if strings.TrimSpace(input.RunID) != "" {
		queryBuilder.WriteString(" AND r.id = ?")
		args = append(args, strings.TrimSpace(input.RunID))
	}
	if strings.TrimSpace(input.StepID) != "" {
		queryBuilder.WriteString(" AND d.step_id = ?")
		args = append(args, strings.TrimSpace(input.StepID))
	}
	if strings.TrimSpace(input.Channel) != "" {
		queryBuilder.WriteString(" AND d.channel = ?")
		args = append(args, strings.TrimSpace(input.Channel))
	}
	if strings.TrimSpace(input.Status) != "" {
		queryBuilder.WriteString(" AND d.status = ?")
		args = append(args, strings.TrimSpace(input.Status))
	}
	if strings.TrimSpace(input.CursorAttemptedAt) != "" && strings.TrimSpace(input.CursorID) != "" {
		queryBuilder.WriteString(" AND (d.attempted_at < ? OR (d.attempted_at = ? AND d.id < ?))")
		args = append(args, input.CursorAttemptedAt, input.CursorAttemptedAt, input.CursorID)
	}

	queryBuilder.WriteString(" ORDER BY d.attempted_at DESC, d.id DESC")
	queryBuilder.WriteString(" LIMIT ?")
	args = append(args, input.Limit+1)

	rows, err := db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query comm attempt history: %w", err)
	}
	defer rows.Close()

	attempts := make([]transport.CommAttemptRecord, 0)
	for rows.Next() {
		var attempt transport.CommAttemptRecord
		if err := rows.Scan(
			&attempt.AttemptID,
			&attempt.WorkspaceID,
			&attempt.StepID,
			&attempt.EventID,
			&attempt.DestinationEndpoint,
			&attempt.IdempotencyKey,
			&attempt.Channel,
			&attempt.Status,
			&attempt.ProviderReceipt,
			&attempt.Error,
			&attempt.AttemptedAt,
			&attempt.ThreadID,
			&attempt.TaskID,
			&attempt.RunID,
		); err != nil {
			return nil, fmt.Errorf("scan comm attempt history row: %w", err)
		}
		attempt.OperationID = parseOperationIDFromIdempotencyKey(attempt.IdempotencyKey)
		attempt.RouteIndex = parseRouteIndex(attempt.IdempotencyKey)
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comm attempt history rows: %w", err)
	}
	return attempts, nil
}

func loadOperationRouteMetadataForAttempts(
	ctx context.Context,
	db *sql.DB,
	workspaceID string,
	attempts []transport.CommAttemptRecord,
) (map[string]commAttemptRouteMetadata, error) {
	metadata := map[string]commAttemptRouteMetadata{}
	if len(attempts) == 0 {
		return metadata, nil
	}

	operationSet := map[string]struct{}{}
	for _, attempt := range attempts {
		operationID := strings.TrimSpace(attempt.OperationID)
		if operationID == "" {
			operationID = parseOperationIDFromIdempotencyKey(attempt.IdempotencyKey)
		}
		if operationID == "" {
			continue
		}
		operationSet[operationID] = struct{}{}
	}
	if len(operationSet) == 0 {
		return metadata, nil
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, idempotency_key, channel
		FROM delivery_attempts
		WHERE workspace_id = ?
		  AND (
	`)
	args := []any{workspaceID}
	operations := make([]string, 0, len(operationSet))
	for op := range operationSet {
		operations = append(operations, op)
	}
	sort.Strings(operations)
	for index, operation := range operations {
		if index > 0 {
			queryBuilder.WriteString(" OR ")
		}
		queryBuilder.WriteString("idempotency_key LIKE ?")
		args = append(args, operation+"|%")
	}
	queryBuilder.WriteString(")")

	rows, err := db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("load operation route metadata: %w", err)
	}
	defer rows.Close()

	grouped := map[string][]commAttemptRouteRow{}
	for rows.Next() {
		var attemptID string
		var idempotencyKey string
		var channel string
		if err := rows.Scan(&attemptID, &idempotencyKey, &channel); err != nil {
			return nil, fmt.Errorf("scan operation route metadata: %w", err)
		}
		operationID := parseOperationIDFromIdempotencyKey(idempotencyKey)
		if operationID == "" {
			continue
		}
		grouped[operationID] = append(grouped[operationID], commAttemptRouteRow{
			AttemptID:   attemptID,
			OperationID: operationID,
			Channel:     strings.ToLower(strings.TrimSpace(channel)),
			RouteIndex:  parseRouteIndex(idempotencyKey),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation route metadata rows: %w", err)
	}

	for _, attemptsForOperation := range grouped {
		sort.Slice(attemptsForOperation, func(i, j int) bool {
			if attemptsForOperation[i].RouteIndex == attemptsForOperation[j].RouteIndex {
				return attemptsForOperation[i].AttemptID < attemptsForOperation[j].AttemptID
			}
			return attemptsForOperation[i].RouteIndex < attemptsForOperation[j].RouteIndex
		})

		firstChannel := ""
		for _, row := range attemptsForOperation {
			if row.RouteIndex == 0 {
				firstChannel = row.Channel
				break
			}
		}
		if firstChannel == "" && len(attemptsForOperation) > 0 {
			firstChannel = attemptsForOperation[0].Channel
		}

		seenByChannel := map[string]int{}
		previousChannel := ""
		for _, row := range attemptsForOperation {
			meta := commAttemptRouteMetadata{
				RoutePhase:   "primary",
				RetryOrdinal: seenByChannel[row.Channel],
			}
			seenByChannel[row.Channel] = seenByChannel[row.Channel] + 1

			switch {
			case row.RouteIndex < 0:
				meta.RoutePhase = "unknown"
				meta.RetryOrdinal = 0
			case row.RouteIndex == 0:
				meta.RoutePhase = "primary"
				meta.RetryOrdinal = 0
			case row.Channel == firstChannel:
				meta.RoutePhase = "retry"
			default:
				meta.RoutePhase = "fallback"
				meta.FallbackFromChannel = previousChannel
			}

			metadata[row.AttemptID] = meta
			previousChannel = row.Channel
		}
	}

	return metadata, nil
}

func applyAttemptRouteMetadata(attempts []transport.CommAttemptRecord, metadata map[string]commAttemptRouteMetadata) {
	for index := range attempts {
		meta, ok := metadata[attempts[index].AttemptID]
		if !ok {
			continue
		}
		attempts[index].RoutePhase = meta.RoutePhase
		attempts[index].RetryOrdinal = meta.RetryOrdinal
		attempts[index].FallbackFromChannel = meta.FallbackFromChannel
	}
}

func parseOperationIDFromIdempotencyKey(idempotencyKey string) string {
	parts := strings.SplitN(strings.TrimSpace(idempotencyKey), "|", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func queryAttemptsByOperation(ctx context.Context, db *sql.DB, workspaceID string, operationID string) ([]transport.CommAttemptRecord, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, destination_endpoint, idempotency_key, channel,
		       status, COALESCE(provider_receipt, ''), COALESCE(error, ''), attempted_at
		FROM delivery_attempts
		WHERE workspace_id = ?
		  AND idempotency_key LIKE ?
		ORDER BY attempted_at ASC, id ASC
	`, workspaceID, operationID+"|%")
	if err != nil {
		return nil, fmt.Errorf("query attempts: %w", err)
	}
	defer rows.Close()

	attempts := make([]transport.CommAttemptRecord, 0)
	for rows.Next() {
		var attempt transport.CommAttemptRecord
		if err := rows.Scan(
			&attempt.AttemptID,
			&attempt.WorkspaceID,
			&attempt.DestinationEndpoint,
			&attempt.IdempotencyKey,
			&attempt.Channel,
			&attempt.Status,
			&attempt.ProviderReceipt,
			&attempt.Error,
			&attempt.AttemptedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		attempt.OperationID = parseOperationIDFromIdempotencyKey(attempt.IdempotencyKey)
		attempt.RouteIndex = parseRouteIndex(attempt.IdempotencyKey)
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attempts: %w", err)
	}
	return attempts, nil
}

func parseRouteIndex(idempotencyKey string) int {
	parts := strings.Split(idempotencyKey, "|")
	if len(parts) == 0 {
		return -1
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	var index int
	if _, err := fmt.Sscanf(last, "%d", &index); err != nil {
		return -1
	}
	return index
}

func parseFallbackChannels(value []string) []string {
	out := make([]string, 0, len(value))
	seen := map[string]struct{}{}
	for _, item := range value {
		normalized := normalizeDeliveryPolicyRouteChannel(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeDeliveryPolicyRouteChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "", "imessage":
		return "imessage"
	case "twilio", "sms":
		return "twilio"
	case "builtin.app", "app":
		return "builtin.app"
	default:
		return strings.ToLower(strings.TrimSpace(channel))
	}
}
