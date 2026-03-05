package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) QueryContextMemoryCandidates(ctx context.Context, request transport.ContextMemoryCandidatesRequest) (transport.ContextMemoryCandidatesResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampContextQueryListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeContextQueryCursor(request.CursorCreatedAt, request.CursorID, "cursor_created_at")
	if err != nil {
		return transport.ContextMemoryCandidatesResponse{}, err
	}

	query := `
		SELECT
			id,
			workspace_id,
			owner_principal_actor_id,
			candidate_json,
			score,
			status,
			created_at
		FROM memory_candidates
		WHERE workspace_id = ?
	`
	params := []any{workspace}

	if ownerActorID := strings.TrimSpace(request.OwnerActorID); ownerActorID != "" {
		query += " AND owner_principal_actor_id = ?"
		params = append(params, ownerActorID)
	}
	if status := strings.TrimSpace(request.Status); status != "" {
		query += " AND UPPER(COALESCE(status, '')) = ?"
		params = append(params, strings.ToUpper(status))
	}
	if cursorCreatedAt != "" {
		query += " AND ((created_at < ?) OR (created_at = ? AND id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ContextMemoryCandidatesResponse{}, fmt.Errorf("query context memory candidates: %w", err)
	}
	defer rows.Close()

	items := make([]transport.ContextMemoryCandidateItem, 0)
	for rows.Next() {
		var (
			item    transport.ContextMemoryCandidateItem
			score   sql.NullFloat64
			payload string
		)
		if err := rows.Scan(
			&item.CandidateID,
			&item.WorkspaceID,
			&item.OwnerActorID,
			&payload,
			&score,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return transport.ContextMemoryCandidatesResponse{}, fmt.Errorf("scan context memory candidate row: %w", err)
		}
		if score.Valid {
			value := score.Float64
			item.Score = &value
		}
		item.CandidateJSON = payload
		item.CandidateKind, item.TokenEstimate, item.SourceIDs, item.SourceRefs = parseMemoryCandidateJSON(payload)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.ContextMemoryCandidatesResponse{}, fmt.Errorf("iterate context memory candidate rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.ContextMemoryCandidatesResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.CandidateID
	}
	return response, nil
}
