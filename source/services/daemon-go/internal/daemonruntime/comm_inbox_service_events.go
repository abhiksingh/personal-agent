package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AgentDelegationService) ListCommEvents(ctx context.Context, request transport.CommEventTimelineRequest) (transport.CommEventTimelineResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.CommEventTimelineResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCommInboxListLimit(request.Limit)
	cursorTime, cursorID, err := parseCommInboxCursor(request.Cursor)
	if err != nil {
		return transport.CommEventTimelineResponse{}, err
	}

	query := `
		SELECT
			ce.id,
			ce.workspace_id,
			ce.thread_id,
			COALESCE(ct.channel, ''),
			COALESCE(ce.connector_id, ''),
			ce.event_type,
			ce.direction,
			ce.assistant_emitted,
			COALESCE(ce.body_text, ''),
			ce.occurred_at,
			ce.created_at
		FROM comm_events ce
		JOIN comm_threads ct ON ct.id = ce.thread_id
		WHERE ce.workspace_id = ?
	`
	params := []any{workspace}

	threadID := strings.TrimSpace(request.ThreadID)
	if threadID != "" {
		query += " AND ce.thread_id = ?"
		params = append(params, threadID)
	}
	if channel := strings.TrimSpace(request.Channel); channel != "" {
		query += " AND LOWER(COALESCE(ct.channel, '')) = ?"
		params = append(params, strings.ToLower(channel))
	}
	if connectorID := strings.TrimSpace(request.ConnectorID); connectorID != "" {
		query += " AND LOWER(COALESCE(ce.connector_id, '')) = ?"
		params = append(params, strings.ToLower(connectorID))
	}
	if eventType := strings.TrimSpace(request.EventType); eventType != "" {
		query += " AND UPPER(COALESCE(ce.event_type, '')) = ?"
		params = append(params, strings.ToUpper(eventType))
	}
	if direction := strings.TrimSpace(request.Direction); direction != "" {
		query += " AND UPPER(COALESCE(ce.direction, '')) = ?"
		params = append(params, strings.ToUpper(direction))
	}
	if search := strings.TrimSpace(request.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query += `
			AND (
				LOWER(COALESCE(ce.body_text, '')) LIKE ?
				OR EXISTS (
					SELECT 1
					FROM comm_event_addresses cea
					WHERE cea.event_id = ce.id
					  AND LOWER(COALESCE(cea.address_value, '')) LIKE ?
				)
			)
		`
		params = append(params, like, like)
	}
	if cursorTime != "" && cursorID != "" {
		query += " AND ((ce.occurred_at < ?) OR (ce.occurred_at = ? AND ce.id < ?))"
		params = append(params, cursorTime, cursorTime, cursorID)
	}

	query += " ORDER BY ce.occurred_at DESC, ce.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommEventTimelineResponse{}, fmt.Errorf("list comm events: %w", err)
	}
	defer rows.Close()

	items := make([]transport.CommEventTimelineItem, 0)
	for rows.Next() {
		var (
			item          transport.CommEventTimelineItem
			assistantFlag int
		)
		if err := rows.Scan(
			&item.EventID,
			&item.WorkspaceID,
			&item.ThreadID,
			&item.Channel,
			&item.ConnectorID,
			&item.EventType,
			&item.Direction,
			&assistantFlag,
			&item.BodyText,
			&item.OccurredAt,
			&item.CreatedAt,
		); err != nil {
			return transport.CommEventTimelineResponse{}, fmt.Errorf("scan comm event row: %w", err)
		}
		item.AssistantEmitted = assistantFlag == 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.CommEventTimelineResponse{}, fmt.Errorf("iterate comm event rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	addressesByEvent, err := listCommEventAddresses(ctx, s.container.DB, items)
	if err != nil {
		return transport.CommEventTimelineResponse{}, err
	}
	for idx := range items {
		if addresses, ok := addressesByEvent[items[idx].EventID]; ok {
			items[idx].Addresses = addresses
		}
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCommInboxCursor(last.OccurredAt, last.EventID)
	}
	return transport.CommEventTimelineResponse{
		WorkspaceID: workspace,
		ThreadID:    threadID,
		Items:       items,
		HasMore:     hasMore,
		NextCursor:  nextCursor,
	}, nil
}
