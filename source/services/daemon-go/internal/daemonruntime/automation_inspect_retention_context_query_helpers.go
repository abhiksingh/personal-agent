package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func clampContextQueryListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultContextQueryListLimit
	case limit > maxContextQueryListLimit:
		return maxContextQueryListLimit
	default:
		return limit
	}
}

func normalizeContextQueryCursor(timestamp string, id string, fieldName string) (string, string, error) {
	cursorTimestamp := strings.TrimSpace(timestamp)
	cursorID := strings.TrimSpace(id)
	if cursorTimestamp == "" {
		if cursorID != "" {
			return "", "", fmt.Errorf("%s is required when cursor_id is provided", fieldName)
		}
		return "", "", nil
	}
	if _, err := time.Parse(time.RFC3339Nano, cursorTimestamp); err != nil {
		return "", "", fmt.Errorf("%s must be RFC3339 timestamp: %w", fieldName, err)
	}
	if cursorID == "" {
		cursorID = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	}
	return cursorTimestamp, cursorID, nil
}

func listMemorySourcesForItems(
	ctx context.Context,
	db *sql.DB,
	items []transport.ContextMemoryInventoryItem,
) (map[string][]transport.ContextMemorySourceRecord, error) {
	if len(items) == 0 {
		return map[string][]transport.ContextMemorySourceRecord{}, nil
	}

	ids := make([]string, 0, len(items))
	args := make([]any, 0, len(items))
	for _, item := range items {
		memoryID := strings.TrimSpace(item.MemoryID)
		if memoryID == "" {
			continue
		}
		ids = append(ids, memoryID)
		args = append(args, memoryID)
	}
	if len(ids) == 0 {
		return map[string][]transport.ContextMemorySourceRecord{}, nil
	}

	query := `
		SELECT id, memory_item_id, source_type, source_ref, created_at
		FROM memory_sources
		WHERE memory_item_id IN (` + sqlListPlaceholders(len(ids)) + `)
		ORDER BY created_at DESC, id DESC
	`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memory sources for inventory rows: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]transport.ContextMemorySourceRecord, len(ids))
	for rows.Next() {
		var (
			record   transport.ContextMemorySourceRecord
			memoryID string
		)
		if err := rows.Scan(
			&record.SourceID,
			&memoryID,
			&record.SourceType,
			&record.SourceRef,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan memory source row: %w", err)
		}
		result[memoryID] = append(result[memoryID], record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory source rows: %w", err)
	}
	return result, nil
}

func parseMemoryCandidateJSON(payload string) (string, int, []string, []string) {
	kind := ""
	tokenEstimate := 0
	sourceIDs := []string{}
	sourceRefs := []string{}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &decoded); err != nil {
		return kind, tokenEstimate, sourceIDs, sourceRefs
	}

	kind = strings.TrimSpace(valueAsString(decoded["kind"]))
	if kind == "" {
		kind = strings.TrimSpace(valueAsString(decoded["scope_type"]))
	}

	if value, ok := decoded["token_estimate"].(float64); ok {
		tokenEstimate = int(value)
	}
	sourceIDs = normalizeStringList(decoded["source_ids"])
	sourceRefs = normalizeStringList(decoded["source_refs"])
	return kind, tokenEstimate, sourceIDs, sourceRefs
}

func normalizeStringList(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			trimmed := strings.TrimSpace(valueAsString(value))
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return []string{}
		}
		parts := strings.Split(trimmed, ",")
		out := make([]string, 0, len(parts))
		for _, value := range parts {
			entry := strings.TrimSpace(value)
			if entry != "" {
				out = append(out, entry)
			}
		}
		return out
	default:
		return []string{}
	}
}
