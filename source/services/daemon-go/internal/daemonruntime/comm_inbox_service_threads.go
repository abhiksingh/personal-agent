package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AgentDelegationService) ListCommThreads(ctx context.Context, request transport.CommThreadListRequest) (transport.CommThreadListResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.CommThreadListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCommInboxListLimit(request.Limit)
	cursorTime, cursorID, err := parseCommInboxCursor(request.Cursor)
	if err != nil {
		return transport.CommThreadListResponse{}, err
	}

	query := `
		SELECT
			t.id,
			t.workspace_id,
			COALESCE(t.channel, ''),
			COALESCE(t.connector_id, ''),
			COALESCE(t.external_ref, ''),
			COALESCE(t.title, ''),
			COALESCE((
				SELECT ce.id
				FROM comm_events ce
				WHERE ce.thread_id = t.id
				ORDER BY ce.occurred_at DESC, ce.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT ce.event_type
				FROM comm_events ce
				WHERE ce.thread_id = t.id
				ORDER BY ce.occurred_at DESC, ce.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT ce.direction
				FROM comm_events ce
				WHERE ce.thread_id = t.id
				ORDER BY ce.occurred_at DESC, ce.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT ce.occurred_at
				FROM comm_events ce
				WHERE ce.thread_id = t.id
				ORDER BY ce.occurred_at DESC, ce.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT ce.body_text
				FROM comm_events ce
				WHERE ce.thread_id = t.id
				ORDER BY ce.occurred_at DESC, ce.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT COUNT(*)
				FROM comm_events ce
				WHERE ce.thread_id = t.id
			), 0),
			t.created_at,
			t.updated_at
		FROM comm_threads t
		WHERE t.workspace_id = ?
	`
	params := []any{workspace}

	if strings.TrimSpace(request.Channel) != "" {
		query += " AND LOWER(COALESCE(t.channel, '')) = ?"
		params = append(params, strings.ToLower(strings.TrimSpace(request.Channel)))
	}
	if connectorID := strings.TrimSpace(request.ConnectorID); connectorID != "" {
		query += " AND LOWER(COALESCE(t.connector_id, '')) = ?"
		params = append(params, strings.ToLower(connectorID))
	}
	if search := strings.TrimSpace(request.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query += `
			AND (
				LOWER(COALESCE(t.title, '')) LIKE ?
				OR LOWER(COALESCE(t.external_ref, '')) LIKE ?
				OR EXISTS (
					SELECT 1
					FROM comm_events ce
					LEFT JOIN comm_event_addresses cea ON cea.event_id = ce.id
					WHERE ce.thread_id = t.id
					  AND (
						LOWER(COALESCE(ce.body_text, '')) LIKE ?
						OR LOWER(COALESCE(cea.address_value, '')) LIKE ?
					  )
				)
			)
		`
		params = append(params, like, like, like, like)
	}
	if cursorTime != "" && cursorID != "" {
		query += " AND ((t.updated_at < ?) OR (t.updated_at = ? AND t.id < ?))"
		params = append(params, cursorTime, cursorTime, cursorID)
	}

	query += " ORDER BY t.updated_at DESC, t.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommThreadListResponse{}, fmt.Errorf("list comm threads: %w", err)
	}
	defer rows.Close()

	items := make([]transport.CommThreadListItem, 0)
	for rows.Next() {
		var (
			item       transport.CommThreadListItem
			eventCount int
		)
		if err := rows.Scan(
			&item.ThreadID,
			&item.WorkspaceID,
			&item.Channel,
			&item.ConnectorID,
			&item.ExternalRef,
			&item.Title,
			&item.LastEventID,
			&item.LastEventType,
			&item.LastDirection,
			&item.LastOccurredAt,
			&item.LastBodyPreview,
			&eventCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return transport.CommThreadListResponse{}, fmt.Errorf("scan comm thread row: %w", err)
		}
		item.EventCount = eventCount
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.CommThreadListResponse{}, fmt.Errorf("iterate comm thread rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	for idx := range items {
		participants, err := listThreadParticipantAddresses(ctx, s.container.DB, workspace, items[idx].ThreadID)
		if err != nil {
			return transport.CommThreadListResponse{}, err
		}
		items[idx].ParticipantAddresses = participants
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCommInboxCursor(last.UpdatedAt, last.ThreadID)
	}
	return transport.CommThreadListResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
		NextCursor:  nextCursor,
	}, nil
}
