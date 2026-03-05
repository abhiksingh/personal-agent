package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AgentDelegationService) ListCommCallSessions(ctx context.Context, request transport.CommCallSessionListRequest) (transport.CommCallSessionListResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.CommCallSessionListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCommInboxListLimit(request.Limit)
	cursorTime, cursorID, err := parseCommInboxCursor(request.Cursor)
	if err != nil {
		return transport.CommCallSessionListResponse{}, err
	}

	query := `
		SELECT
			id,
			workspace_id,
			provider,
			COALESCE(connector_id, ''),
			provider_call_id,
			thread_id,
			direction,
			COALESCE(from_address, ''),
			COALESCE(to_address, ''),
			status,
			COALESCE(started_at, ''),
			COALESCE(ended_at, ''),
			updated_at
		FROM comm_call_sessions
		WHERE workspace_id = ?
	`
	params := []any{workspace}

	if threadID := strings.TrimSpace(request.ThreadID); threadID != "" {
		query += " AND thread_id = ?"
		params = append(params, threadID)
	}
	if provider := strings.TrimSpace(request.Provider); provider != "" {
		query += " AND LOWER(COALESCE(provider, '')) = ?"
		params = append(params, strings.ToLower(provider))
	}
	if connectorID := strings.TrimSpace(request.ConnectorID); connectorID != "" {
		query += " AND LOWER(COALESCE(connector_id, '')) = ?"
		params = append(params, strings.ToLower(connectorID))
	}
	if direction := strings.TrimSpace(request.Direction); direction != "" {
		query += " AND LOWER(COALESCE(direction, '')) = ?"
		params = append(params, strings.ToLower(direction))
	}
	if status := strings.TrimSpace(request.Status); status != "" {
		query += " AND LOWER(COALESCE(status, '')) = ?"
		params = append(params, strings.ToLower(status))
	}
	if callID := strings.TrimSpace(request.ProviderCallID); callID != "" {
		query += " AND provider_call_id = ?"
		params = append(params, callID)
	}
	if search := strings.TrimSpace(request.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query += `
			AND (
				LOWER(COALESCE(provider_call_id, '')) LIKE ?
				OR LOWER(COALESCE(from_address, '')) LIKE ?
				OR LOWER(COALESCE(to_address, '')) LIKE ?
			)
		`
		params = append(params, like, like, like)
	}
	if cursorTime != "" && cursorID != "" {
		query += " AND ((updated_at < ?) OR (updated_at = ? AND id < ?))"
		params = append(params, cursorTime, cursorTime, cursorID)
	}

	query += " ORDER BY updated_at DESC, id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommCallSessionListResponse{}, fmt.Errorf("list comm call sessions: %w", err)
	}
	defer rows.Close()

	items := make([]transport.CommCallSessionListItem, 0)
	for rows.Next() {
		var item transport.CommCallSessionListItem
		if err := rows.Scan(
			&item.SessionID,
			&item.WorkspaceID,
			&item.Provider,
			&item.ConnectorID,
			&item.ProviderCallID,
			&item.ThreadID,
			&item.Direction,
			&item.FromAddress,
			&item.ToAddress,
			&item.Status,
			&item.StartedAt,
			&item.EndedAt,
			&item.UpdatedAt,
		); err != nil {
			return transport.CommCallSessionListResponse{}, fmt.Errorf("scan comm call session row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.CommCallSessionListResponse{}, fmt.Errorf("iterate comm call session rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCommInboxCursor(last.UpdatedAt, last.SessionID)
	}

	return transport.CommCallSessionListResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
		NextCursor:  nextCursor,
	}, nil
}
