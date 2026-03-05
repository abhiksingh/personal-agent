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

const (
	defaultCommTrustReceiptListLimit = 50
	maxCommTrustReceiptListLimit     = 200
	maxCommTrustAuditLinksPerReceipt = 20
)

func (s *CommTwilioService) ListCommWebhookReceipts(ctx context.Context, request transport.CommWebhookReceiptListRequest) (transport.CommWebhookReceiptListResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.CommWebhookReceiptListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCommTrustReceiptListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeCommTrustReceiptCursor(request.CursorCreatedAt, request.CursorID)
	if err != nil {
		return transport.CommWebhookReceiptListResponse{}, err
	}

	query := `
		SELECT
			r.id,
			r.workspace_id,
			r.provider,
			r.provider_event_id,
			r.signature_valid,
			COALESCE(r.signature_value, ''),
			COALESCE(r.payload_hash, ''),
			COALESCE(r.event_id, ''),
			COALESCE(e.thread_id, ''),
			r.received_at,
			r.created_at
		FROM comm_webhook_receipts r
		LEFT JOIN comm_events e ON e.id = r.event_id
		WHERE r.workspace_id = ?
	`
	params := []any{workspace}

	provider := strings.ToLower(strings.TrimSpace(request.Provider))
	if provider != "" {
		query += " AND LOWER(COALESCE(r.provider, '')) = ?"
		params = append(params, provider)
	}
	if providerEventID := strings.TrimSpace(request.ProviderEventID); providerEventID != "" {
		query += " AND r.provider_event_id = ?"
		params = append(params, providerEventID)
	}
	if providerEventQuery := strings.TrimSpace(request.ProviderEventQuery); providerEventQuery != "" {
		query += " AND LOWER(COALESCE(r.provider_event_id, '')) LIKE ?"
		params = append(params, "%"+strings.ToLower(providerEventQuery)+"%")
	}
	if eventID := strings.TrimSpace(request.EventID); eventID != "" {
		query += " AND COALESCE(r.event_id, '') = ?"
		params = append(params, eventID)
	}
	if cursorCreatedAt != "" {
		query += " AND ((r.created_at < ?) OR (r.created_at = ? AND r.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}
	query += " ORDER BY r.created_at DESC, r.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommWebhookReceiptListResponse{}, fmt.Errorf("list comm webhook receipts: %w", err)
	}

	items := make([]transport.CommWebhookReceiptItem, 0)
	for rows.Next() {
		var (
			item           transport.CommWebhookReceiptItem
			signatureValue string
			signatureValid int
		)
		if err := rows.Scan(
			&item.ReceiptID,
			&item.WorkspaceID,
			&item.Provider,
			&item.ProviderEventID,
			&signatureValid,
			&signatureValue,
			&item.PayloadHash,
			&item.EventID,
			&item.ThreadID,
			&item.ReceivedAt,
			&item.CreatedAt,
		); err != nil {
			return transport.CommWebhookReceiptListResponse{}, fmt.Errorf("scan comm webhook receipt row: %w", err)
		}
		item.SignatureValid = signatureValid == 1
		item.SignatureValuePresent = strings.TrimSpace(signatureValue) != ""
		if item.SignatureValid {
			item.TrustState = "accepted"
		} else {
			item.TrustState = "rejected"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return transport.CommWebhookReceiptListResponse{}, fmt.Errorf("iterate comm webhook receipt rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return transport.CommWebhookReceiptListResponse{}, fmt.Errorf("close comm webhook receipt rows: %w", err)
	}

	for index := range items {
		items[index].AuditLinks, err = listAuditLinksForReceipt(ctx, s.container.DB, workspace, map[string]string{
			"receipt_id":        items[index].ReceiptID,
			"provider_event_id": items[index].ProviderEventID,
			"message_sid":       items[index].ProviderEventID,
			"call_sid":          items[index].ProviderEventID,
			"event_id":          items[index].EventID,
		})
		if err != nil {
			return transport.CommWebhookReceiptListResponse{}, err
		}
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.CommWebhookReceiptListResponse{
		WorkspaceID: workspace,
		Provider:    provider,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.ReceiptID
	}
	return response, nil
}

func (s *CommTwilioService) ListCommIngestReceipts(ctx context.Context, request transport.CommIngestReceiptListRequest) (transport.CommIngestReceiptListResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.CommIngestReceiptListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCommTrustReceiptListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeCommTrustReceiptCursor(request.CursorCreatedAt, request.CursorID)
	if err != nil {
		return transport.CommIngestReceiptListResponse{}, err
	}
	trustState, err := normalizeIngestTrustStateFilter(request.TrustState)
	if err != nil {
		return transport.CommIngestReceiptListResponse{}, err
	}

	query := `
		SELECT
			r.id,
			r.workspace_id,
			r.source,
			r.source_scope,
			r.source_event_id,
			COALESCE(r.source_cursor, ''),
			r.trust_state,
			COALESCE(r.payload_hash, ''),
			COALESCE(r.event_id, ''),
			COALESCE(e.thread_id, ''),
			r.received_at,
			r.created_at
		FROM comm_ingest_receipts r
		LEFT JOIN comm_events e ON e.id = r.event_id
		WHERE r.workspace_id = ?
	`
	params := []any{workspace}

	source := strings.ToLower(strings.TrimSpace(request.Source))
	if source != "" {
		query += " AND LOWER(COALESCE(r.source, '')) = ?"
		params = append(params, source)
	}
	sourceScope := strings.TrimSpace(request.SourceScope)
	if sourceScope != "" {
		query += " AND COALESCE(r.source_scope, '') = ?"
		params = append(params, sourceScope)
	}
	if sourceEventID := strings.TrimSpace(request.SourceEventID); sourceEventID != "" {
		query += " AND COALESCE(r.source_event_id, '') = ?"
		params = append(params, sourceEventID)
	}
	if sourceEventQuery := strings.TrimSpace(request.SourceEventQuery); sourceEventQuery != "" {
		query += " AND LOWER(COALESCE(r.source_event_id, '')) LIKE ?"
		params = append(params, "%"+strings.ToLower(sourceEventQuery)+"%")
	}
	if trustState != "" {
		query += " AND UPPER(COALESCE(r.trust_state, '')) = ?"
		params = append(params, trustState)
	}
	if eventID := strings.TrimSpace(request.EventID); eventID != "" {
		query += " AND COALESCE(r.event_id, '') = ?"
		params = append(params, eventID)
	}
	if cursorCreatedAt != "" {
		query += " AND ((r.created_at < ?) OR (r.created_at = ? AND r.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}
	query += " ORDER BY r.created_at DESC, r.id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.CommIngestReceiptListResponse{}, fmt.Errorf("list comm ingest receipts: %w", err)
	}

	items := make([]transport.CommIngestReceiptItem, 0)
	for rows.Next() {
		var item transport.CommIngestReceiptItem
		if err := rows.Scan(
			&item.ReceiptID,
			&item.WorkspaceID,
			&item.Source,
			&item.SourceScope,
			&item.SourceEventID,
			&item.SourceCursor,
			&item.TrustState,
			&item.PayloadHash,
			&item.EventID,
			&item.ThreadID,
			&item.ReceivedAt,
			&item.CreatedAt,
		); err != nil {
			return transport.CommIngestReceiptListResponse{}, fmt.Errorf("scan comm ingest receipt row: %w", err)
		}
		item.TrustState = strings.ToLower(strings.TrimSpace(item.TrustState))
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return transport.CommIngestReceiptListResponse{}, fmt.Errorf("iterate comm ingest receipt rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return transport.CommIngestReceiptListResponse{}, fmt.Errorf("close comm ingest receipt rows: %w", err)
	}

	for index := range items {
		items[index].AuditLinks, err = listAuditLinksForReceipt(ctx, s.container.DB, workspace, map[string]string{
			"receipt_id":      items[index].ReceiptID,
			"source_event_id": items[index].SourceEventID,
			"event_id":        items[index].EventID,
		})
		if err != nil {
			return transport.CommIngestReceiptListResponse{}, err
		}
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.CommIngestReceiptListResponse{
		WorkspaceID: workspace,
		Source:      source,
		SourceScope: sourceScope,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.ReceiptID
	}
	return response, nil
}

func clampCommTrustReceiptListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultCommTrustReceiptListLimit
	case limit > maxCommTrustReceiptListLimit:
		return maxCommTrustReceiptListLimit
	default:
		return limit
	}
}

func normalizeCommTrustReceiptCursor(createdAt string, cursorID string) (string, string, error) {
	cursorCreatedAt := strings.TrimSpace(createdAt)
	resolvedID := strings.TrimSpace(cursorID)
	if cursorCreatedAt == "" {
		if resolvedID != "" {
			return "", "", fmt.Errorf("cursor_created_at is required when cursor_id is provided")
		}
		return "", "", nil
	}
	if _, err := time.Parse(time.RFC3339Nano, cursorCreatedAt); err != nil {
		return "", "", fmt.Errorf("cursor_created_at must be RFC3339 timestamp: %w", err)
	}
	if resolvedID == "" {
		resolvedID = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	}
	return cursorCreatedAt, resolvedID, nil
}

func normalizeIngestTrustStateFilter(raw string) (string, error) {
	filter := strings.ToUpper(strings.TrimSpace(raw))
	if filter == "" {
		return "", nil
	}
	switch filter {
	case "ACCEPTED", "REJECTED":
		return filter, nil
	default:
		return "", fmt.Errorf("unsupported trust_state %q (allowed: accepted|rejected)", strings.TrimSpace(raw))
	}
}

func listAuditLinksForReceipt(ctx context.Context, db *sql.DB, workspaceID string, keyValues map[string]string) ([]transport.ReceiptAuditLink, error) {
	if db == nil {
		return []transport.ReceiptAuditLink{}, nil
	}
	workspaceID = normalizeWorkspaceID(workspaceID)

	keys := make([]string, 0, len(keyValues))
	for key := range keyValues {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	patterns := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(keyValues[key])
		if value == "" {
			continue
		}
		patterns = append(patterns, `%"`+strings.ToLower(strings.TrimSpace(key))+`":"`+escapeSQLLikePattern(strings.ToLower(value))+`"%`)
	}
	if len(patterns) == 0 {
		return []transport.ReceiptAuditLink{}, nil
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, event_type, created_at
		FROM audit_log_entries
		WHERE workspace_id = ?
		  AND (
	`)
	args := []any{workspaceID}
	for index, pattern := range patterns {
		if index > 0 {
			queryBuilder.WriteString(" OR ")
		}
		queryBuilder.WriteString("LOWER(COALESCE(payload_json, '')) LIKE ? ESCAPE '\\'")
		args = append(args, pattern)
	}
	queryBuilder.WriteString(")")
	queryBuilder.WriteString(" ORDER BY created_at DESC, id DESC LIMIT ?")
	args = append(args, maxCommTrustAuditLinksPerReceipt)

	rows, err := db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list receipt audit links: %w", err)
	}
	defer rows.Close()

	links := make([]transport.ReceiptAuditLink, 0)
	for rows.Next() {
		var item transport.ReceiptAuditLink
		if err := rows.Scan(&item.AuditID, &item.EventType, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan receipt audit link row: %w", err)
		}
		links = append(links, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate receipt audit link rows: %w", err)
	}
	return links, nil
}

func escapeSQLLikePattern(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(value)
}
