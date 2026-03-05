package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) QueryContextRetrievalDocuments(ctx context.Context, request transport.ContextRetrievalDocumentsRequest) (transport.ContextRetrievalDocumentsResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampContextQueryListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeContextQueryCursor(request.CursorCreatedAt, request.CursorID, "cursor_created_at")
	if err != nil {
		return transport.ContextRetrievalDocumentsResponse{}, err
	}

	query := `
		SELECT
			d.id,
			d.workspace_id,
			COALESCE(d.owner_principal_actor_id, ''),
			COALESCE(d.source_uri, ''),
			COALESCE(d.checksum, ''),
			d.created_at,
			(SELECT COUNT(*) FROM context_chunks c WHERE c.document_id = d.id)
		FROM context_documents d
		WHERE d.workspace_id = ?
	`
	params := []any{workspace}

	if ownerActorID := strings.TrimSpace(request.OwnerActorID); ownerActorID != "" {
		query += " AND d.owner_principal_actor_id = ?"
		params = append(params, ownerActorID)
	}
	if sourceURIQuery := strings.TrimSpace(request.SourceURIQuery); sourceURIQuery != "" {
		query += " AND LOWER(COALESCE(d.source_uri, '')) LIKE ?"
		params = append(params, "%"+strings.ToLower(sourceURIQuery)+"%")
	}
	if cursorCreatedAt != "" {
		query += " AND ((d.created_at < ?) OR (d.created_at = ? AND d.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}

	query += " ORDER BY d.created_at DESC, d.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ContextRetrievalDocumentsResponse{}, fmt.Errorf("query context retrieval documents: %w", err)
	}
	defer rows.Close()

	items := make([]transport.ContextRetrievalDocumentItem, 0)
	for rows.Next() {
		var item transport.ContextRetrievalDocumentItem
		if err := rows.Scan(
			&item.DocumentID,
			&item.WorkspaceID,
			&item.OwnerActorID,
			&item.SourceURI,
			&item.Checksum,
			&item.CreatedAt,
			&item.ChunkCount,
		); err != nil {
			return transport.ContextRetrievalDocumentsResponse{}, fmt.Errorf("scan context retrieval document row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.ContextRetrievalDocumentsResponse{}, fmt.Errorf("iterate context retrieval document rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.ContextRetrievalDocumentsResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.DocumentID
	}
	return response, nil
}
