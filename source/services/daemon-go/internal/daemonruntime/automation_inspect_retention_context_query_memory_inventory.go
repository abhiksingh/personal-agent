package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) QueryContextMemoryInventory(ctx context.Context, request transport.ContextMemoryInventoryRequest) (transport.ContextMemoryInventoryResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampContextQueryListLimit(request.Limit)
	cursorUpdatedAt, cursorID, err := normalizeContextQueryCursor(request.CursorUpdatedAt, request.CursorID, "cursor_updated_at")
	if err != nil {
		return transport.ContextMemoryInventoryResponse{}, err
	}

	query := `
		SELECT
			mi.id,
			mi.workspace_id,
			mi.owner_principal_actor_id,
			mi.scope_type,
			mi.key,
			mi.value_json,
			mi.status,
			COALESCE(mi.source_summary, ''),
			mi.created_at,
			mi.updated_at,
			(SELECT COUNT(*) FROM memory_sources ms WHERE ms.memory_item_id = mi.id)
		FROM memory_items mi
		WHERE mi.workspace_id = ?
	`
	params := []any{workspace}

	if ownerActorID := strings.TrimSpace(request.OwnerActorID); ownerActorID != "" {
		query += " AND mi.owner_principal_actor_id = ?"
		params = append(params, ownerActorID)
	}
	if scopeType := strings.TrimSpace(request.ScopeType); scopeType != "" {
		query += " AND LOWER(COALESCE(mi.scope_type, '')) = ?"
		params = append(params, strings.ToLower(scopeType))
	}
	if status := strings.TrimSpace(request.Status); status != "" {
		query += " AND UPPER(COALESCE(mi.status, '')) = ?"
		params = append(params, strings.ToUpper(status))
	}

	sourceType := strings.TrimSpace(request.SourceType)
	sourceRefQuery := strings.TrimSpace(request.SourceRefQuery)
	if sourceType != "" || sourceRefQuery != "" {
		query += `
			AND EXISTS (
				SELECT 1
				FROM memory_sources ms
				WHERE ms.memory_item_id = mi.id
		`
		if sourceType != "" {
			query += " AND LOWER(COALESCE(ms.source_type, '')) = ?"
			params = append(params, strings.ToLower(sourceType))
		}
		if sourceRefQuery != "" {
			query += " AND LOWER(COALESCE(ms.source_ref, '')) LIKE ?"
			params = append(params, "%"+strings.ToLower(sourceRefQuery)+"%")
		}
		query += ")"
	}
	if cursorUpdatedAt != "" {
		query += " AND ((mi.updated_at < ?) OR (mi.updated_at = ? AND mi.id < ?))"
		params = append(params, cursorUpdatedAt, cursorUpdatedAt, cursorID)
	}

	query += " ORDER BY mi.updated_at DESC, mi.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ContextMemoryInventoryResponse{}, fmt.Errorf("query context memory inventory: %w", err)
	}
	defer rows.Close()

	items := make([]transport.ContextMemoryInventoryItem, 0)
	for rows.Next() {
		var (
			item      transport.ContextMemoryInventoryItem
			valueJSON string
		)
		if err := rows.Scan(
			&item.MemoryID,
			&item.WorkspaceID,
			&item.OwnerActorID,
			&item.ScopeType,
			&item.Key,
			&valueJSON,
			&item.Status,
			&item.SourceSummary,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.SourceCount,
		); err != nil {
			return transport.ContextMemoryInventoryResponse{}, fmt.Errorf("scan context memory inventory row: %w", err)
		}
		kind, canonical, tokens := parseMemoryValueJSON(valueJSON, item.ScopeType)
		item.Kind = kind
		item.IsCanonical = canonical
		item.TokenEstimate = tokens
		item.ValueJSON = valueJSON
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.ContextMemoryInventoryResponse{}, fmt.Errorf("iterate context memory inventory rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	sourcesByMemoryID, err := listMemorySourcesForItems(ctx, s.container.DB, items)
	if err != nil {
		return transport.ContextMemoryInventoryResponse{}, err
	}
	for idx := range items {
		if sources, ok := sourcesByMemoryID[items[idx].MemoryID]; ok {
			items[idx].Sources = sources
		} else {
			items[idx].Sources = []transport.ContextMemorySourceRecord{}
		}
	}

	response := transport.ContextMemoryInventoryResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorUpdatedAt = last.UpdatedAt
		response.NextCursorID = last.MemoryID
	}
	return response, nil
}
