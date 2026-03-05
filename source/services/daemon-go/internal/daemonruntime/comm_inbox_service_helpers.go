package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func clampCommInboxListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultCommInboxListLimit
	case limit > maxCommInboxListLimit:
		return maxCommInboxListLimit
	default:
		return limit
	}
}

func encodeCommInboxCursor(sortValue string, identifier string) string {
	sortValue = strings.TrimSpace(sortValue)
	identifier = strings.TrimSpace(identifier)
	if sortValue == "" || identifier == "" {
		return ""
	}
	return sortValue + "|" + identifier
}

func parseCommInboxCursor(raw string) (string, string, error) {
	cursor := strings.TrimSpace(raw)
	if cursor == "" {
		return "", "", nil
	}
	parts := strings.SplitN(cursor, "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid --cursor format")
	}
	sortValue := strings.TrimSpace(parts[0])
	identifier := strings.TrimSpace(parts[1])
	if sortValue == "" || identifier == "" {
		return "", "", fmt.Errorf("invalid --cursor format")
	}
	return sortValue, identifier, nil
}

func listThreadParticipantAddresses(ctx context.Context, db *sql.DB, workspaceID string, threadID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT TRIM(COALESCE(cea.address_value, '')) AS address_value
		FROM comm_event_addresses cea
		JOIN comm_events ce ON ce.id = cea.event_id
		WHERE ce.workspace_id = ?
		  AND ce.thread_id = ?
		  AND TRIM(COALESCE(cea.address_value, '')) <> ''
		GROUP BY TRIM(COALESCE(cea.address_value, ''))
		ORDER BY address_value COLLATE NOCASE ASC
		LIMIT 16
	`, workspaceID, strings.TrimSpace(threadID))
	if err != nil {
		return nil, fmt.Errorf("list comm thread participants: %w", err)
	}
	defer rows.Close()

	participants := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan comm thread participant: %w", err)
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			participants = append(participants, trimmed)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comm thread participants: %w", err)
	}
	return participants, nil
}

func listCommEventAddresses(ctx context.Context, db *sql.DB, events []transport.CommEventTimelineItem) (map[string][]transport.CommEventAddressItem, error) {
	if len(events) == 0 {
		return map[string][]transport.CommEventAddressItem{}, nil
	}

	ids := make([]string, 0, len(events))
	args := make([]any, 0, len(events))
	for _, item := range events {
		eventID := strings.TrimSpace(item.EventID)
		if eventID == "" {
			continue
		}
		ids = append(ids, eventID)
		args = append(args, eventID)
	}
	if len(ids) == 0 {
		return map[string][]transport.CommEventAddressItem{}, nil
	}

	query := `
		SELECT event_id, address_role, COALESCE(address_value, ''), COALESCE(display_name, ''), position
		FROM comm_event_addresses
		WHERE event_id IN (` + sqlListPlaceholders(len(ids)) + `)
		ORDER BY event_id ASC, position ASC, id ASC
	`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list comm event addresses: %w", err)
	}
	defer rows.Close()

	results := make(map[string][]transport.CommEventAddressItem, len(ids))
	for rows.Next() {
		var (
			eventID string
			item    transport.CommEventAddressItem
		)
		if err := rows.Scan(&eventID, &item.Role, &item.Value, &item.Display, &item.Position); err != nil {
			return nil, fmt.Errorf("scan comm event address: %w", err)
		}
		trimmedID := strings.TrimSpace(eventID)
		if trimmedID == "" {
			continue
		}
		results[trimmedID] = append(results[trimmedID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comm event addresses: %w", err)
	}

	return results, nil
}

func sqlListPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}
