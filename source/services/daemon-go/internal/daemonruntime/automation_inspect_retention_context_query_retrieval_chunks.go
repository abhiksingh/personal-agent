package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) QueryContextRetrievalChunks(ctx context.Context, request transport.ContextRetrievalChunksRequest) (transport.ContextRetrievalChunksResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampContextQueryListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeContextQueryCursor(request.CursorCreatedAt, request.CursorID, "cursor_created_at")
	if err != nil {
		return transport.ContextRetrievalChunksResponse{}, err
	}

	query := `
		SELECT
			c.id,
			d.workspace_id,
			c.document_id,
			COALESCE(d.owner_principal_actor_id, ''),
			COALESCE(d.source_uri, ''),
			c.chunk_index,
			c.text_body,
			COALESCE(c.token_count, 0),
			c.created_at
		FROM context_chunks c
		JOIN context_documents d ON d.id = c.document_id
		WHERE d.workspace_id = ?
	`
	params := []any{workspace}

	documentID := strings.TrimSpace(request.DocumentID)
	if documentID != "" {
		query += " AND c.document_id = ?"
		params = append(params, documentID)
	}
	if ownerActorID := strings.TrimSpace(request.OwnerActorID); ownerActorID != "" {
		query += " AND d.owner_principal_actor_id = ?"
		params = append(params, ownerActorID)
	}
	if sourceURIQuery := strings.TrimSpace(request.SourceURIQuery); sourceURIQuery != "" {
		query += " AND LOWER(COALESCE(d.source_uri, '')) LIKE ?"
		params = append(params, "%"+strings.ToLower(sourceURIQuery)+"%")
	}
	if chunkTextQuery := strings.TrimSpace(request.ChunkTextQuery); chunkTextQuery != "" {
		query += " AND LOWER(COALESCE(c.text_body, '')) LIKE ?"
		params = append(params, "%"+strings.ToLower(chunkTextQuery)+"%")
	}
	if cursorCreatedAt != "" {
		query += " AND ((c.created_at < ?) OR (c.created_at = ? AND c.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}

	query += " ORDER BY c.created_at DESC, c.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ContextRetrievalChunksResponse{}, fmt.Errorf("query context retrieval chunks: %w", err)
	}
	defer rows.Close()

	items := make([]transport.ContextRetrievalChunkItem, 0)
	for rows.Next() {
		var item transport.ContextRetrievalChunkItem
		if err := rows.Scan(
			&item.ChunkID,
			&item.WorkspaceID,
			&item.DocumentID,
			&item.OwnerActorID,
			&item.SourceURI,
			&item.ChunkIndex,
			&item.TextBody,
			&item.TokenCount,
			&item.CreatedAt,
		); err != nil {
			return transport.ContextRetrievalChunksResponse{}, fmt.Errorf("scan context retrieval chunk row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.ContextRetrievalChunksResponse{}, fmt.Errorf("iterate context retrieval chunk rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.ContextRetrievalChunksResponse{
		WorkspaceID: workspace,
		DocumentID:  documentID,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.ChunkID
	}
	return response, nil
}
