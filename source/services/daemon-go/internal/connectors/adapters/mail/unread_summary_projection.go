package mail

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type unreadMailItem struct {
	EventID     string
	ThreadID    string
	ThreadTitle string
	MessageID   string
	FromAddress string
	ToAddress   string
	Subject     string
	BodyPreview string
	OccurredAt  string
}

type unreadMailSummaryResult struct {
	Limit       int
	UnreadCount int
	ThreadCount int
	Items       []unreadMailItem
}

func resolveMailSummaryLimit(input map[string]any) (int, error) {
	if len(input) == 0 {
		return defaultMailSummaryLimit, nil
	}
	raw, ok := input["limit"]
	if !ok || raw == nil {
		return defaultMailSummaryLimit, nil
	}
	parsed, err := parseMailSummaryLimit(raw)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("mail limit must be non-negative")
	}
	return normalizeMailSummaryLimit(parsed), nil
}

func parseMailSummaryLimit(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, fmt.Errorf("mail limit must be an integer")
		}
		return int(typed), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("mail limit must be an integer")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("mail limit must be an integer")
	}
}

func normalizeMailSummaryLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultMailSummaryLimit
	case limit > maxMailSummaryLimit:
		return maxMailSummaryLimit
	default:
		return limit
	}
}

func (a *Adapter) summarizeUnreadInbox(ctx context.Context, workspaceID string, limit int) (unreadMailSummaryResult, error) {
	trimmedWorkspace := strings.TrimSpace(workspaceID)
	if trimmedWorkspace == "" {
		return unreadMailSummaryResult{}, fmt.Errorf("workspace id is required")
	}
	dbPath := strings.TrimSpace(a.dbPath)
	if dbPath == "" {
		return unreadMailSummaryResult{}, fmt.Errorf("mail unread summary db path is required")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return unreadMailSummaryResult{}, fmt.Errorf("open mail unread summary db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	normalizedLimit := normalizeMailSummaryLimit(limit)
	unreadCount, err := queryMailUnreadCount(ctx, db, trimmedWorkspace)
	if err != nil {
		return unreadMailSummaryResult{}, err
	}
	threadCount, err := queryMailUnreadThreadCount(ctx, db, trimmedWorkspace)
	if err != nil {
		return unreadMailSummaryResult{}, err
	}
	items, err := queryMailUnreadItems(ctx, db, trimmedWorkspace, normalizedLimit)
	if err != nil {
		return unreadMailSummaryResult{}, err
	}
	return unreadMailSummaryResult{
		Limit:       normalizedLimit,
		UnreadCount: unreadCount,
		ThreadCount: threadCount,
		Items:       items,
	}, nil
}

func queryMailUnreadCount(ctx context.Context, db *sql.DB, workspaceID string) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM comm_events ce
		WHERE ce.workspace_id = ?
		  AND LOWER(COALESCE(ce.connector_id, '')) = 'mail'
		  AND UPPER(COALESCE(ce.event_type, '')) = 'MESSAGE'
		  AND UPPER(COALESCE(ce.direction, '')) = 'INBOUND'
		  AND ce.assistant_emitted = 0
		  AND NOT EXISTS (
			SELECT 1
			FROM comm_events newer
			WHERE newer.workspace_id = ce.workspace_id
			  AND newer.thread_id = ce.thread_id
			  AND LOWER(COALESCE(newer.connector_id, '')) = 'mail'
			  AND UPPER(COALESCE(newer.event_type, '')) = 'MESSAGE'
			  AND UPPER(COALESCE(newer.direction, '')) = 'OUTBOUND'
			  AND newer.assistant_emitted = 1
			  AND (
				newer.occurred_at > ce.occurred_at
				OR (newer.occurred_at = ce.occurred_at AND newer.id > ce.id)
			  )
		  )
	`, strings.TrimSpace(workspaceID)).Scan(&count); err != nil {
		return 0, fmt.Errorf("query unread mail count: %w", err)
	}
	return count, nil
}

func queryMailUnreadThreadCount(ctx context.Context, db *sql.DB, workspaceID string) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ce.thread_id)
		FROM comm_events ce
		WHERE ce.workspace_id = ?
		  AND LOWER(COALESCE(ce.connector_id, '')) = 'mail'
		  AND UPPER(COALESCE(ce.event_type, '')) = 'MESSAGE'
		  AND UPPER(COALESCE(ce.direction, '')) = 'INBOUND'
		  AND ce.assistant_emitted = 0
		  AND NOT EXISTS (
			SELECT 1
			FROM comm_events newer
			WHERE newer.workspace_id = ce.workspace_id
			  AND newer.thread_id = ce.thread_id
			  AND LOWER(COALESCE(newer.connector_id, '')) = 'mail'
			  AND UPPER(COALESCE(newer.event_type, '')) = 'MESSAGE'
			  AND UPPER(COALESCE(newer.direction, '')) = 'OUTBOUND'
			  AND newer.assistant_emitted = 1
			  AND (
				newer.occurred_at > ce.occurred_at
				OR (newer.occurred_at = ce.occurred_at AND newer.id > ce.id)
			  )
		  )
	`, strings.TrimSpace(workspaceID)).Scan(&count); err != nil {
		return 0, fmt.Errorf("query unread mail thread count: %w", err)
	}
	return count, nil
}

func queryMailUnreadItems(ctx context.Context, db *sql.DB, workspaceID string, limit int) ([]unreadMailItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			ce.id,
			ce.thread_id,
			COALESCE(ct.title, ''),
			COALESCE(ce.body_text, ''),
			COALESCE(ce.occurred_at, ''),
			COALESCE(em.message_id, ''),
			COALESCE((
				SELECT cea.address_value
				FROM comm_event_addresses cea
				WHERE cea.event_id = ce.id
				  AND UPPER(COALESCE(cea.address_role, '')) = 'FROM'
				ORDER BY cea.position ASC, cea.id ASC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT cea.address_value
				FROM comm_event_addresses cea
				WHERE cea.event_id = ce.id
				  AND UPPER(COALESCE(cea.address_role, '')) = 'TO'
				ORDER BY cea.position ASC, cea.id ASC
				LIMIT 1
			), '')
		FROM comm_events ce
		JOIN comm_threads ct ON ct.id = ce.thread_id
		LEFT JOIN email_event_meta em ON em.event_id = ce.id
		WHERE ce.workspace_id = ?
		  AND LOWER(COALESCE(ce.connector_id, '')) = 'mail'
		  AND UPPER(COALESCE(ce.event_type, '')) = 'MESSAGE'
		  AND UPPER(COALESCE(ce.direction, '')) = 'INBOUND'
		  AND ce.assistant_emitted = 0
		  AND NOT EXISTS (
			SELECT 1
			FROM comm_events newer
			WHERE newer.workspace_id = ce.workspace_id
			  AND newer.thread_id = ce.thread_id
			  AND LOWER(COALESCE(newer.connector_id, '')) = 'mail'
			  AND UPPER(COALESCE(newer.event_type, '')) = 'MESSAGE'
			  AND UPPER(COALESCE(newer.direction, '')) = 'OUTBOUND'
			  AND newer.assistant_emitted = 1
			  AND (
				newer.occurred_at > ce.occurred_at
				OR (newer.occurred_at = ce.occurred_at AND newer.id > ce.id)
			  )
		  )
		ORDER BY ce.occurred_at DESC, ce.id DESC
		LIMIT ?
	`, strings.TrimSpace(workspaceID), normalizeMailSummaryLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query unread mail items: %w", err)
	}
	defer rows.Close()

	items := make([]unreadMailItem, 0, normalizeMailSummaryLimit(limit))
	for rows.Next() {
		var (
			item         unreadMailItem
			combinedBody string
		)
		if err := rows.Scan(
			&item.EventID,
			&item.ThreadID,
			&item.ThreadTitle,
			&combinedBody,
			&item.OccurredAt,
			&item.MessageID,
			&item.FromAddress,
			&item.ToAddress,
		); err != nil {
			return nil, fmt.Errorf("scan unread mail row: %w", err)
		}
		subject, body := splitMailCombinedBody(combinedBody)
		if subject == "" {
			subject = strings.TrimSpace(item.ThreadTitle)
		}
		if body == "" {
			body = strings.TrimSpace(combinedBody)
		}
		item.Subject = subject
		item.BodyPreview = truncateMailPreview(body, 240)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unread mail rows: %w", err)
	}
	return items, nil
}

func splitMailCombinedBody(combined string) (string, string) {
	trimmed := strings.TrimSpace(combined)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, "\n\n", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", trimmed
}

func truncateMailPreview(body string, maxLen int) string {
	trimmed := strings.TrimSpace(body)
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen]
}

func unreadSummaryItemsOutput(items []unreadMailItem) []map[string]any {
	outputItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		outputItems = append(outputItems, map[string]any{
			"event_id":      item.EventID,
			"thread_id":     item.ThreadID,
			"message_id":    item.MessageID,
			"from":          item.FromAddress,
			"to":            item.ToAddress,
			"subject":       item.Subject,
			"body_preview":  item.BodyPreview,
			"occurred_at":   item.OccurredAt,
			"thread_title":  item.ThreadTitle,
			"unread_reason": "no newer assistant outbound mail event in thread",
		})
	}
	return outputItems
}
