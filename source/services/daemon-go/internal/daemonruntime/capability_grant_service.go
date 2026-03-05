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

const (
	defaultCapabilityGrantListLimit = 50
	maxCapabilityGrantListLimit     = 200
)

func (s *AgentDelegationService) UpsertCapabilityGrant(ctx context.Context, request transport.CapabilityGrantUpsertRequest) (transport.CapabilityGrantRecord, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	grantID := strings.TrimSpace(request.GrantID)
	actorID := strings.TrimSpace(request.ActorID)
	capabilityKey := strings.TrimSpace(request.CapabilityKey)
	scopeJSONInput := strings.TrimSpace(request.ScopeJSON)
	if scopeJSONInput != "" && !json.Valid([]byte(scopeJSONInput)) {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("scope_json must be valid json")
	}

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(ctx, tx, workspace, nowText); err != nil {
		return transport.CapabilityGrantRecord{}, err
	}

	record, found, err := loadCapabilityGrantForUpsert(ctx, tx, workspace, grantID, actorID, capabilityKey)
	if err != nil {
		return transport.CapabilityGrantRecord{}, err
	}
	if found {
		grantID = record.GrantID
		if actorID == "" {
			actorID = record.ActorID
		}
		if capabilityKey == "" {
			capabilityKey = record.CapabilityKey
		}
	}

	if actorID == "" || capabilityKey == "" {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("actor_id and capability_key are required")
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspace, actorID, nowText); err != nil {
		return transport.CapabilityGrantRecord{}, err
	}
	if active, principalErr := isActiveWorkspacePrincipal(ctx, tx, workspace, actorID); principalErr != nil {
		return transport.CapabilityGrantRecord{}, principalErr
	} else if !active {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("capability grant denied: actor_id %q is not an active workspace principal", actorID)
	}

	scopeJSON := record.ScopeJSON
	if scopeJSONInput != "" {
		scopeJSON = scopeJSONInput
	}
	status, err := normalizeCapabilityGrantStatus(request.Status, record.Status)
	if err != nil {
		return transport.CapabilityGrantRecord{}, err
	}
	expiresAt, err := resolveCapabilityGrantExpiresAt(request.ExpiresAt, record.ExpiresAt, status, now)
	if err != nil {
		return transport.CapabilityGrantRecord{}, err
	}

	createdAt := record.CreatedAt
	changeType := "updated"
	if !found {
		changeType = "created"
		createdAt = nowText
		if grantID == "" {
			grantID, err = delegationRandomID()
			if err != nil {
				return transport.CapabilityGrantRecord{}, err
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO capability_grants(
				id, workspace_id, actor_id, capability_key, scope_json, status, created_at, expires_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, grantID, workspace, actorID, capabilityKey, delegationNullableText(scopeJSON), status, createdAt, delegationNullableText(expiresAt)); err != nil {
			return transport.CapabilityGrantRecord{}, fmt.Errorf("insert capability grant: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE capability_grants
			SET actor_id = ?,
			    capability_key = ?,
			    scope_json = ?,
			    status = ?,
			    expires_at = ?
			WHERE id = ? AND workspace_id = ?
		`, actorID, capabilityKey, delegationNullableText(scopeJSON), status, delegationNullableText(expiresAt), grantID, workspace); err != nil {
			return transport.CapabilityGrantRecord{}, fmt.Errorf("update capability grant: %w", err)
		}
	}

	if err := appendDelegationAuditEntry(ctx, tx, workspace, "CAPABILITY_GRANT_UPSERTED", actorID, actorID, map[string]any{
		"grant_id":         grantID,
		"actor_id":         actorID,
		"capability_key":   capabilityKey,
		"status":           status,
		"expires_at":       expiresAt,
		"scope_json":       scopeJSON,
		"capability_event": changeType,
	}); err != nil {
		return transport.CapabilityGrantRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return transport.CapabilityGrantRecord{}, fmt.Errorf("commit tx: %w", err)
	}

	return transport.CapabilityGrantRecord{
		GrantID:       grantID,
		WorkspaceID:   workspace,
		ActorID:       actorID,
		CapabilityKey: capabilityKey,
		ScopeJSON:     scopeJSON,
		Status:        status,
		CreatedAt:     createdAt,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *AgentDelegationService) ListCapabilityGrants(ctx context.Context, request transport.CapabilityGrantListRequest) (transport.CapabilityGrantListResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.CapabilityGrantListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampCapabilityGrantListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeCapabilityGrantCursor(request.CursorCreatedAt, request.CursorID)
	if err != nil {
		return transport.CapabilityGrantListResponse{}, err
	}

	query := `
		SELECT
			id,
			workspace_id,
			actor_id,
			capability_key,
			COALESCE(scope_json, ''),
			status,
			created_at,
			COALESCE(expires_at, '')
		FROM capability_grants
		WHERE workspace_id = ?
	`
	params := []any{workspace}

	if actorID := strings.TrimSpace(request.ActorID); actorID != "" {
		query += " AND actor_id = ?"
		params = append(params, actorID)
	}
	if capabilityKey := strings.TrimSpace(request.CapabilityKey); capabilityKey != "" {
		query += " AND LOWER(COALESCE(capability_key, '')) = ?"
		params = append(params, strings.ToLower(capabilityKey))
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
		return transport.CapabilityGrantListResponse{}, fmt.Errorf("list capability grants: %w", err)
	}
	defer rows.Close()

	items := make([]transport.CapabilityGrantRecord, 0)
	for rows.Next() {
		var item transport.CapabilityGrantRecord
		if err := rows.Scan(
			&item.GrantID,
			&item.WorkspaceID,
			&item.ActorID,
			&item.CapabilityKey,
			&item.ScopeJSON,
			&item.Status,
			&item.CreatedAt,
			&item.ExpiresAt,
		); err != nil {
			return transport.CapabilityGrantListResponse{}, fmt.Errorf("scan capability grant row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.CapabilityGrantListResponse{}, fmt.Errorf("iterate capability grant rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	response := transport.CapabilityGrantListResponse{
		WorkspaceID: workspace,
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.GrantID
	}
	return response, nil
}

func loadCapabilityGrantForUpsert(
	ctx context.Context,
	queryable interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	workspaceID string,
	grantID string,
	actorID string,
	capabilityKey string,
) (transport.CapabilityGrantRecord, bool, error) {
	workspaceID = normalizeWorkspaceID(workspaceID)
	trimmedGrantID := strings.TrimSpace(grantID)
	if trimmedGrantID != "" {
		record, found, err := loadCapabilityGrantByID(ctx, queryable, workspaceID, trimmedGrantID)
		if err != nil {
			return transport.CapabilityGrantRecord{}, false, err
		}
		if !found {
			return transport.CapabilityGrantRecord{}, false, fmt.Errorf("capability grant %q not found", trimmedGrantID)
		}
		return record, true, nil
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(capabilityKey) == "" {
		return transport.CapabilityGrantRecord{}, false, nil
	}
	return loadLatestCapabilityGrant(ctx, queryable, workspaceID, actorID, capabilityKey)
}

func loadCapabilityGrantByID(
	ctx context.Context,
	queryable interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	workspaceID string,
	grantID string,
) (transport.CapabilityGrantRecord, bool, error) {
	var record transport.CapabilityGrantRecord
	err := queryable.QueryRowContext(ctx, `
		SELECT id, workspace_id, actor_id, capability_key, COALESCE(scope_json, ''), status, created_at, COALESCE(expires_at, '')
		FROM capability_grants
		WHERE workspace_id = ? AND id = ?
		LIMIT 1
	`, workspaceID, strings.TrimSpace(grantID)).Scan(
		&record.GrantID,
		&record.WorkspaceID,
		&record.ActorID,
		&record.CapabilityKey,
		&record.ScopeJSON,
		&record.Status,
		&record.CreatedAt,
		&record.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return transport.CapabilityGrantRecord{}, false, nil
	}
	if err != nil {
		return transport.CapabilityGrantRecord{}, false, fmt.Errorf("load capability grant by id: %w", err)
	}
	return record, true, nil
}

func loadLatestCapabilityGrant(
	ctx context.Context,
	queryable interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	workspaceID string,
	actorID string,
	capabilityKey string,
) (transport.CapabilityGrantRecord, bool, error) {
	var record transport.CapabilityGrantRecord
	err := queryable.QueryRowContext(ctx, `
		SELECT id, workspace_id, actor_id, capability_key, COALESCE(scope_json, ''), status, created_at, COALESCE(expires_at, '')
		FROM capability_grants
		WHERE workspace_id = ? AND actor_id = ? AND LOWER(COALESCE(capability_key, '')) = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, workspaceID, strings.TrimSpace(actorID), strings.ToLower(strings.TrimSpace(capabilityKey))).Scan(
		&record.GrantID,
		&record.WorkspaceID,
		&record.ActorID,
		&record.CapabilityKey,
		&record.ScopeJSON,
		&record.Status,
		&record.CreatedAt,
		&record.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return transport.CapabilityGrantRecord{}, false, nil
	}
	if err != nil {
		return transport.CapabilityGrantRecord{}, false, fmt.Errorf("load latest capability grant: %w", err)
	}
	return record, true, nil
}

func normalizeCapabilityGrantStatus(raw string, fallback string) (string, error) {
	status := strings.ToUpper(strings.TrimSpace(raw))
	if status == "" {
		status = strings.ToUpper(strings.TrimSpace(fallback))
	}
	if status == "" {
		status = "ACTIVE"
	}
	switch status {
	case "ACTIVE", "DISABLED", "REVOKED":
		return status, nil
	default:
		return "", fmt.Errorf("unsupported capability grant status %q (allowed: ACTIVE|DISABLED|REVOKED)", strings.TrimSpace(raw))
	}
}

func resolveCapabilityGrantExpiresAt(raw string, fallback string, status string, now time.Time) (string, error) {
	resolved := strings.TrimSpace(fallback)
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		parsed, err := time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid expires_at: %w", err)
		}
		resolved = parsed.UTC().Format(time.RFC3339Nano)
	}
	if resolved != "" {
		parsed, err := time.Parse(time.RFC3339Nano, resolved)
		if err != nil {
			return "", fmt.Errorf("invalid expires_at: %w", err)
		}
		if strings.EqualFold(status, "ACTIVE") && !parsed.UTC().After(now.UTC()) {
			return "", fmt.Errorf("capability grant denied: expires_at must be in the future for ACTIVE grants")
		}
	}
	return resolved, nil
}

func clampCapabilityGrantListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultCapabilityGrantListLimit
	case limit > maxCapabilityGrantListLimit:
		return maxCapabilityGrantListLimit
	default:
		return limit
	}
}

func normalizeCapabilityGrantCursor(createdAt string, cursorID string) (string, string, error) {
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
