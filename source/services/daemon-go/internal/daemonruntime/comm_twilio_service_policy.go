package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) ListCommAttempts(ctx context.Context, request transport.CommAttemptsRequest) (transport.CommAttemptsResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	operationID := strings.TrimSpace(request.OperationID)
	threadID := strings.TrimSpace(request.ThreadID)
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	stepID := strings.TrimSpace(request.StepID)
	channel := strings.ToLower(strings.TrimSpace(request.Channel))
	status := strings.ToLower(strings.TrimSpace(request.Status))

	if operationID == "" &&
		threadID == "" &&
		taskID == "" &&
		runID == "" &&
		stepID == "" {
		return transport.CommAttemptsResponse{}, fmt.Errorf("--operation-id or one of --thread-id/--task-id/--run-id/--step-id is required")
	}

	if operationID != "" &&
		threadID == "" &&
		taskID == "" &&
		runID == "" &&
		stepID == "" &&
		channel == "" &&
		status == "" &&
		strings.TrimSpace(request.Cursor) == "" &&
		request.Limit <= 0 {
		attempts, err := queryAttemptsByOperation(ctx, s.container.DB, workspace, operationID)
		if err != nil {
			return transport.CommAttemptsResponse{}, err
		}
		metadata, err := loadOperationRouteMetadataForAttempts(ctx, s.container.DB, workspace, attempts)
		if err != nil {
			return transport.CommAttemptsResponse{}, err
		}
		applyAttemptRouteMetadata(attempts, metadata)
		return transport.CommAttemptsResponse{
			WorkspaceID: workspace,
			OperationID: operationID,
			Attempts:    attempts,
		}, nil
	}

	limit, err := normalizeCommAttemptHistoryLimit(request.Limit)
	if err != nil {
		return transport.CommAttemptsResponse{}, err
	}
	cursorAttemptedAt, cursorID, err := parseCommAttemptHistoryCursor(request.Cursor)
	if err != nil {
		return transport.CommAttemptsResponse{}, err
	}

	queryInput := commAttemptHistoryQueryInput{
		WorkspaceID:       workspace,
		OperationID:       operationID,
		ThreadID:          threadID,
		TaskID:            taskID,
		RunID:             runID,
		StepID:            stepID,
		Channel:           channel,
		Status:            status,
		CursorAttemptedAt: cursorAttemptedAt,
		CursorID:          cursorID,
		Limit:             limit,
	}
	attempts, err := queryCommAttemptHistory(ctx, s.container.DB, queryInput)
	if err != nil {
		return transport.CommAttemptsResponse{}, err
	}

	hasMore := false
	nextCursor := ""
	if len(attempts) > limit {
		hasMore = true
		attempts = attempts[:limit]
	}
	if hasMore && len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		nextCursor = encodeCommAttemptHistoryCursor(last.AttemptedAt, last.AttemptID)
	}

	metadata, err := loadOperationRouteMetadataForAttempts(ctx, s.container.DB, workspace, attempts)
	if err != nil {
		return transport.CommAttemptsResponse{}, err
	}
	applyAttemptRouteMetadata(attempts, metadata)
	return transport.CommAttemptsResponse{
		WorkspaceID: workspace,
		OperationID: operationID,
		ThreadID:    threadID,
		TaskID:      taskID,
		RunID:       runID,
		StepID:      stepID,
		HasMore:     hasMore,
		NextCursor:  nextCursor,
		Attempts:    attempts,
	}, nil
}

func (s *CommTwilioService) SetCommPolicy(ctx context.Context, request transport.CommPolicySetRequest) (transport.CommPolicyRecord, error) {
	policyID := strings.TrimSpace(request.PolicyID)
	updateExisting := policyID != ""
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	source := strings.ToLower(strings.TrimSpace(request.SourceChannel))
	if source == "" {
		return transport.CommPolicyRecord{}, fmt.Errorf("--source-channel is required")
	}

	policy := types.ChannelDeliveryPolicy{
		PrimaryChannel:   normalizeDeliveryPolicyRouteChannel(request.PrimaryChannel),
		RetryCount:       request.RetryCount,
		FallbackChannels: parseFallbackChannels(request.FallbackChannels),
	}
	policy = normalizeDeliveryPolicy(policy)
	if policy.RetryCount < 0 {
		return transport.CommPolicyRecord{}, fmt.Errorf("--retry-count must be >= 0")
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return transport.CommPolicyRecord{}, fmt.Errorf("marshal policy: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.CommPolicyRecord{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return transport.CommPolicyRecord{}, err
	}

	createdAt := now
	if updateExisting {
		err := tx.QueryRowContext(ctx, `
			SELECT created_at
			FROM channel_delivery_policies
			WHERE id = ? AND workspace_id = ?
			LIMIT 1
		`, policyID, workspace).Scan(&createdAt)
		if err == sql.ErrNoRows {
			return transport.CommPolicyRecord{}, fmt.Errorf("policy %q was not found in workspace %q", policyID, workspace)
		}
		if err != nil {
			return transport.CommPolicyRecord{}, fmt.Errorf("load policy for update: %w", err)
		}
	} else {
		policyID, err = daemonRandomID()
		if err != nil {
			return transport.CommPolicyRecord{}, err
		}
	}

	if request.IsDefault {
		query := `
			UPDATE channel_delivery_policies
			SET is_default = 0
			WHERE workspace_id = ? AND channel = ?
		`
		params := []any{workspace, source}
		if updateExisting {
			query += " AND id <> ?"
			params = append(params, policyID)
		}
		if _, err := tx.ExecContext(ctx, query, params...); err != nil {
			return transport.CommPolicyRecord{}, fmt.Errorf("clear default policies: %w", err)
		}
	}

	endpointPattern := strings.TrimSpace(request.EndpointPattern)
	if updateExisting {
		result, err := tx.ExecContext(ctx, `
			UPDATE channel_delivery_policies
			SET channel = ?,
			    endpoint_pattern = ?,
			    policy_json = ?,
			    is_default = ?,
			    updated_at = ?
			WHERE id = ? AND workspace_id = ?
		`, source, nullableText(endpointPattern), string(policyJSON), boolToInt(request.IsDefault), now, policyID, workspace)
		if err != nil {
			return transport.CommPolicyRecord{}, fmt.Errorf("update policy: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return transport.CommPolicyRecord{}, fmt.Errorf("check update rows: %w", err)
		}
		if rows == 0 {
			return transport.CommPolicyRecord{}, fmt.Errorf("policy %q was not found in workspace %q", policyID, workspace)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO channel_delivery_policies(
				id, workspace_id, channel, endpoint_pattern, policy_json, is_default, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, policyID, workspace, source, nullableText(endpointPattern), string(policyJSON), boolToInt(request.IsDefault), now, now); err != nil {
			return transport.CommPolicyRecord{}, fmt.Errorf("insert policy: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return transport.CommPolicyRecord{}, fmt.Errorf("commit tx: %w", err)
	}

	return transport.CommPolicyRecord{
		ID:              policyID,
		WorkspaceID:     workspace,
		SourceChannel:   source,
		EndpointPattern: endpointPattern,
		IsDefault:       request.IsDefault,
		Policy:          policy,
		CreatedAt:       createdAt,
		UpdatedAt:       now,
	}, nil
}

func (s *CommTwilioService) ListCommPolicies(ctx context.Context, request transport.CommPolicyListRequest) (transport.CommPolicyListResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)

	query := `
		SELECT id, workspace_id, channel, COALESCE(endpoint_pattern, ''), policy_json, is_default, created_at, updated_at
		FROM channel_delivery_policies
		WHERE workspace_id = ?
	`
	params := []any{workspace}
	if strings.TrimSpace(request.SourceChannel) != "" {
		query += " AND channel = ?"
		params = append(params, strings.ToLower(strings.TrimSpace(request.SourceChannel)))
	}
	query += " ORDER BY is_default DESC, updated_at DESC"

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommPolicyListResponse{}, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	policies := make([]transport.CommPolicyRecord, 0)
	for rows.Next() {
		var (
			item        transport.CommPolicyRecord
			policyJSON  string
			defaultFlag int
		)
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.SourceChannel,
			&item.EndpointPattern,
			&policyJSON,
			&defaultFlag,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return transport.CommPolicyListResponse{}, fmt.Errorf("scan policy: %w", err)
		}
		item.IsDefault = defaultFlag == 1
		if err := json.Unmarshal([]byte(policyJSON), &item.Policy); err != nil {
			return transport.CommPolicyListResponse{}, fmt.Errorf("decode policy json: %w", err)
		}
		item.Policy = normalizeDeliveryPolicy(item.Policy)
		policies = append(policies, item)
	}
	if err := rows.Err(); err != nil {
		return transport.CommPolicyListResponse{}, fmt.Errorf("iterate policies: %w", err)
	}

	return transport.CommPolicyListResponse{
		WorkspaceID: workspace,
		Policies:    policies,
	}, nil
}

func normalizeDeliveryPolicy(policy types.ChannelDeliveryPolicy) types.ChannelDeliveryPolicy {
	policy.PrimaryChannel = normalizeDeliveryPolicyRouteChannel(policy.PrimaryChannel)
	if policy.RetryCount < 0 {
		policy.RetryCount = 0
	}

	fallback := make([]string, 0, len(policy.FallbackChannels))
	seen := map[string]struct{}{}
	if policy.PrimaryChannel != "" {
		seen[policy.PrimaryChannel] = struct{}{}
	}
	for _, channel := range policy.FallbackChannels {
		normalized := normalizeDeliveryPolicyRouteChannel(channel)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		fallback = append(fallback, normalized)
	}
	policy.FallbackChannels = fallback
	if policy.PrimaryChannel == "" {
		policy.PrimaryChannel = "imessage"
	}
	return policy
}
