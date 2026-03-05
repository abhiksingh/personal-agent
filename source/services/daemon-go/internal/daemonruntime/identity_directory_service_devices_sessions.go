package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (s *IdentityDirectoryService) ListDevices(ctx context.Context, request transport.IdentityDeviceListRequest) (transport.IdentityDeviceListResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if isReservedSystemWorkspaceID(workspaceID) {
		return transport.IdentityDeviceListResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", workspaceID)
	}
	exists, err := s.workspaceExists(ctx, workspaceID)
	if err != nil {
		return transport.IdentityDeviceListResponse{}, err
	}
	if !exists {
		return transport.IdentityDeviceListResponse{}, fmt.Errorf("workspace %q not found", workspaceID)
	}

	limit := clampIdentityDirectoryListLimit(request.Limit)
	cursorCreatedAt, cursorID, err := normalizeIdentityDirectoryCursor(request.CursorCreatedAt, request.CursorID)
	if err != nil {
		return transport.IdentityDeviceListResponse{}, err
	}

	userID := strings.TrimSpace(request.UserID)
	deviceType := strings.TrimSpace(request.DeviceType)
	platform := strings.TrimSpace(request.Platform)
	nowText := time.Now().UTC().Format(time.RFC3339Nano)

	query := `
		SELECT
			d.id,
			d.workspace_id,
			d.user_id,
			d.device_type,
			d.platform,
			COALESCE(d.label, ''),
			COALESCE(d.last_seen_at, ''),
			d.created_at,
			COUNT(s.id) AS session_total,
			COALESCE(SUM(CASE
				WHEN s.id IS NOT NULL
					AND TRIM(COALESCE(s.revoked_at, '')) = ''
					AND COALESCE(s.expires_at, '') > ?
				THEN 1 ELSE 0 END), 0) AS session_active_count,
			COALESCE(SUM(CASE
				WHEN s.id IS NOT NULL
					AND TRIM(COALESCE(s.revoked_at, '')) = ''
					AND COALESCE(s.expires_at, '') <= ?
				THEN 1 ELSE 0 END), 0) AS session_expired_count,
			COALESCE(SUM(CASE
				WHEN s.id IS NOT NULL
					AND TRIM(COALESCE(s.revoked_at, '')) <> ''
				THEN 1 ELSE 0 END), 0) AS session_revoked_count,
			COALESCE(MAX(s.started_at), '') AS session_latest_started_at
		FROM user_devices d
		LEFT JOIN device_sessions s
			ON s.workspace_id = d.workspace_id
			AND s.device_id = d.id
		WHERE d.workspace_id = ?
	`
	params := []any{nowText, nowText, workspaceID}
	if userID != "" {
		query += " AND d.user_id = ?"
		params = append(params, userID)
	}
	if deviceType != "" {
		query += " AND LOWER(COALESCE(d.device_type, '')) = ?"
		params = append(params, strings.ToLower(deviceType))
	}
	if platform != "" {
		query += " AND LOWER(COALESCE(d.platform, '')) = ?"
		params = append(params, strings.ToLower(platform))
	}
	if cursorCreatedAt != "" {
		query += " AND (d.created_at < ? OR (d.created_at = ? AND d.id < ?))"
		params = append(params, cursorCreatedAt, cursorCreatedAt, cursorID)
	}

	query += `
		GROUP BY
			d.id, d.workspace_id, d.user_id, d.device_type, d.platform, d.label, d.last_seen_at, d.created_at
		ORDER BY d.created_at DESC, d.id DESC
		LIMIT ?
	`
	params = append(params, limit+1)

	rows, err := s.db.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.IdentityDeviceListResponse{}, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	items := make([]transport.IdentityDeviceRecord, 0)
	for rows.Next() {
		var row identityDeviceRow
		if err := rows.Scan(
			&row.DeviceID,
			&row.WorkspaceID,
			&row.UserID,
			&row.DeviceType,
			&row.Platform,
			&row.Label,
			&row.LastSeenAt,
			&row.CreatedAt,
			&row.SessionTotal,
			&row.SessionActiveCount,
			&row.SessionExpiredCount,
			&row.SessionRevokedCount,
			&row.SessionLatestStartedAt,
		); err != nil {
			return transport.IdentityDeviceListResponse{}, fmt.Errorf("scan device row: %w", err)
		}
		items = append(items, transport.IdentityDeviceRecord{
			DeviceID:               row.DeviceID,
			WorkspaceID:            row.WorkspaceID,
			UserID:                 row.UserID,
			DeviceType:             row.DeviceType,
			Platform:               row.Platform,
			Label:                  row.Label,
			LastSeenAt:             row.LastSeenAt,
			CreatedAt:              row.CreatedAt,
			SessionTotal:           row.SessionTotal,
			SessionActiveCount:     row.SessionActiveCount,
			SessionExpiredCount:    row.SessionExpiredCount,
			SessionRevokedCount:    row.SessionRevokedCount,
			SessionLatestStartedAt: row.SessionLatestStartedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return transport.IdentityDeviceListResponse{}, fmt.Errorf("iterate device rows: %w", err)
	}

	response := transport.IdentityDeviceListResponse{
		WorkspaceID: workspaceID,
		UserID:      userID,
		DeviceType:  deviceType,
		Platform:    platform,
		Items:       items,
	}
	if len(items) > limit {
		response.HasMore = true
		response.NextCursorCreatedAt = items[limit-1].CreatedAt
		response.NextCursorID = items[limit-1].DeviceID
		response.Items = items[:limit]
	}
	return response, nil
}

func (s *IdentityDirectoryService) ListSessions(ctx context.Context, request transport.IdentitySessionListRequest) (transport.IdentitySessionListResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if isReservedSystemWorkspaceID(workspaceID) {
		return transport.IdentitySessionListResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", workspaceID)
	}
	exists, err := s.workspaceExists(ctx, workspaceID)
	if err != nil {
		return transport.IdentitySessionListResponse{}, err
	}
	if !exists {
		return transport.IdentitySessionListResponse{}, fmt.Errorf("workspace %q not found", workspaceID)
	}

	limit := clampIdentityDirectoryListLimit(request.Limit)
	cursorStartedAt, cursorID, err := normalizeIdentityDirectoryCursor(request.CursorStartedAt, request.CursorID)
	if err != nil {
		return transport.IdentitySessionListResponse{}, err
	}

	deviceID := strings.TrimSpace(request.DeviceID)
	userID := strings.TrimSpace(request.UserID)
	sessionHealth, err := normalizeIdentitySessionHealth(request.SessionHealth)
	if err != nil {
		return transport.IdentitySessionListResponse{}, err
	}
	nowText := time.Now().UTC().Format(time.RFC3339Nano)

	query := `
		SELECT
			s.id,
			s.workspace_id,
			s.device_id,
			d.user_id,
			d.device_type,
			d.platform,
			COALESCE(d.label, ''),
			COALESCE(d.last_seen_at, ''),
			s.started_at,
			s.expires_at,
			COALESCE(s.revoked_at, '')
		FROM device_sessions s
		JOIN user_devices d
			ON d.workspace_id = s.workspace_id
			AND d.id = s.device_id
		WHERE s.workspace_id = ?
	`
	params := []any{workspaceID}
	if deviceID != "" {
		query += " AND s.device_id = ?"
		params = append(params, deviceID)
	}
	if userID != "" {
		query += " AND d.user_id = ?"
		params = append(params, userID)
	}
	switch sessionHealth {
	case identitySessionHealthActive:
		query += " AND TRIM(COALESCE(s.revoked_at, '')) = '' AND COALESCE(s.expires_at, '') > ?"
		params = append(params, nowText)
	case identitySessionHealthExpired:
		query += " AND TRIM(COALESCE(s.revoked_at, '')) = '' AND COALESCE(s.expires_at, '') <= ?"
		params = append(params, nowText)
	case identitySessionHealthRevoked:
		query += " AND TRIM(COALESCE(s.revoked_at, '')) <> ''"
	}
	if cursorStartedAt != "" {
		query += " AND (s.started_at < ? OR (s.started_at = ? AND s.id < ?))"
		params = append(params, cursorStartedAt, cursorStartedAt, cursorID)
	}

	query += `
		ORDER BY s.started_at DESC, s.id DESC
		LIMIT ?
	`
	params = append(params, limit+1)

	rows, err := s.db.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.IdentitySessionListResponse{}, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	items := make([]transport.IdentitySessionRecord, 0)
	for rows.Next() {
		var row identitySessionRow
		if err := rows.Scan(
			&row.SessionID,
			&row.WorkspaceID,
			&row.DeviceID,
			&row.UserID,
			&row.DeviceType,
			&row.Platform,
			&row.DeviceLabel,
			&row.DeviceLastSeenAt,
			&row.StartedAt,
			&row.ExpiresAt,
			&row.RevokedAt,
		); err != nil {
			return transport.IdentitySessionListResponse{}, fmt.Errorf("scan session row: %w", err)
		}
		items = append(items, transport.IdentitySessionRecord{
			SessionID:        row.SessionID,
			WorkspaceID:      row.WorkspaceID,
			DeviceID:         row.DeviceID,
			UserID:           row.UserID,
			DeviceType:       row.DeviceType,
			Platform:         row.Platform,
			DeviceLabel:      row.DeviceLabel,
			DeviceLastSeenAt: row.DeviceLastSeenAt,
			StartedAt:        row.StartedAt,
			ExpiresAt:        row.ExpiresAt,
			RevokedAt:        row.RevokedAt,
			SessionHealth:    classifyIdentitySessionHealth(row.ExpiresAt, row.RevokedAt, now),
		})
	}
	if err := rows.Err(); err != nil {
		return transport.IdentitySessionListResponse{}, fmt.Errorf("iterate session rows: %w", err)
	}

	response := transport.IdentitySessionListResponse{
		WorkspaceID:   workspaceID,
		DeviceID:      deviceID,
		UserID:        userID,
		SessionHealth: sessionHealth,
		Items:         items,
	}
	if len(items) > limit {
		response.HasMore = true
		response.NextCursorStartedAt = items[limit-1].StartedAt
		response.NextCursorID = items[limit-1].SessionID
		response.Items = items[:limit]
	}
	return response, nil
}

func (s *IdentityDirectoryService) RevokeSession(ctx context.Context, request transport.IdentitySessionRevokeRequest) (transport.IdentitySessionRevokeResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if isReservedSystemWorkspaceID(workspaceID) {
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", workspaceID)
	}
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("session_id is required")
	}

	exists, err := s.workspaceExists(ctx, workspaceID)
	if err != nil {
		return transport.IdentitySessionRevokeResponse{}, err
	}
	if !exists {
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("workspace %q not found", workspaceID)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	record := transport.IdentitySessionRevokeResponse{}
	if err := tx.QueryRowContext(ctx, `
		SELECT
			s.workspace_id,
			s.id,
			s.device_id,
			s.started_at,
			s.expires_at,
			COALESCE(s.revoked_at, ''),
			COALESCE(d.last_seen_at, '')
		FROM device_sessions s
		JOIN user_devices d
			ON d.workspace_id = s.workspace_id
			AND d.id = s.device_id
		WHERE s.workspace_id = ?
		  AND s.id = ?
		LIMIT 1
	`, workspaceID, sessionID).Scan(
		&record.WorkspaceID,
		&record.SessionID,
		&record.DeviceID,
		&record.StartedAt,
		&record.ExpiresAt,
		&record.RevokedAt,
		&record.DeviceLastSeenAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("session %q not found in workspace %q", sessionID, workspaceID)
		}
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("load session for revoke: %w", err)
	}

	if strings.TrimSpace(record.RevokedAt) == "" {
		record.RevokedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := tx.ExecContext(ctx, `
			UPDATE device_sessions
			SET revoked_at = ?
			WHERE workspace_id = ?
			  AND id = ?
		`, record.RevokedAt, workspaceID, sessionID); err != nil {
			return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("revoke session: %w", err)
		}
		record.Idempotent = false
	} else {
		record.Idempotent = true
	}
	record.SessionHealth = identitySessionHealthRevoked

	if err := tx.Commit(); err != nil {
		return transport.IdentitySessionRevokeResponse{}, fmt.Errorf("commit tx: %w", err)
	}
	return record, nil
}

func clampIdentityDirectoryListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultIdentityDirectoryListLimit
	case limit > maxIdentityDirectoryListLimit:
		return maxIdentityDirectoryListLimit
	default:
		return limit
	}
}

func normalizeIdentityDirectoryCursor(cursorTimestamp string, cursorID string) (string, string, error) {
	ts := strings.TrimSpace(cursorTimestamp)
	id := strings.TrimSpace(cursorID)
	if ts == "" && id == "" {
		return "", "", nil
	}
	if ts == "" || id == "" {
		return "", "", fmt.Errorf("cursor timestamp and cursor_id must both be provided")
	}
	if _, err := parseTimestamp(ts); err != nil {
		return "", "", fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return ts, id, nil
}

func normalizeIdentitySessionHealth(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", nil
	}
	switch normalized {
	case identitySessionHealthActive, identitySessionHealthExpired, identitySessionHealthRevoked:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported session_health %q (allowed: active|expired|revoked)", strings.TrimSpace(raw))
	}
}

func classifyIdentitySessionHealth(expiresAt string, revokedAt string, now time.Time) string {
	if strings.TrimSpace(revokedAt) != "" {
		return identitySessionHealthRevoked
	}
	expiresAtTime, err := parseTimestamp(expiresAt)
	if err != nil {
		return identitySessionHealthActive
	}
	if !expiresAtTime.After(now) {
		return identitySessionHealthExpired
	}
	return identitySessionHealthActive
}
