package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *IdentityDirectoryService) loadWorkspaces(ctx context.Context, includeInactive bool) ([]identityWorkspaceRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			w.id,
			w.name,
			w.status,
			w.updated_at,
			COUNT(DISTINCT wp.actor_id) AS principal_count,
			COUNT(DISTINCT a.id) AS actor_count,
			COUNT(DISTINCT ah.id) AS handle_count
		FROM workspaces w
		LEFT JOIN workspace_principals wp ON wp.workspace_id = w.id
		LEFT JOIN actors a ON a.workspace_id = w.id
		LEFT JOIN actor_handles ah ON ah.workspace_id = w.id
		WHERE (? = 1 OR UPPER(w.status) = 'ACTIVE')
		GROUP BY w.id, w.name, w.status, w.updated_at
		ORDER BY CASE WHEN UPPER(w.status) = 'ACTIVE' THEN 0 ELSE 1 END, w.updated_at DESC, w.id ASC
	`, boolToInt(includeInactive))
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	items := make([]identityWorkspaceRow, 0)
	for rows.Next() {
		var item identityWorkspaceRow
		if err := rows.Scan(
			&item.WorkspaceID,
			&item.Name,
			&item.Status,
			&item.UpdatedAt,
			&item.PrincipalCount,
			&item.ActorCount,
			&item.HandleCount,
		); err != nil {
			return nil, fmt.Errorf("scan workspace row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace rows: %w", err)
	}
	return items, nil
}

func (s *IdentityDirectoryService) loadPrincipals(ctx context.Context, workspaceID string, includeInactive bool) ([]identityPrincipalRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			wp.actor_id,
			a.display_name,
			a.actor_type,
			a.status,
			wp.status
		FROM workspace_principals wp
		JOIN actors a
			ON a.id = wp.actor_id
			AND a.workspace_id = wp.workspace_id
		WHERE wp.workspace_id = ?
			AND (? = 1 OR (UPPER(wp.status) = 'ACTIVE' AND UPPER(a.status) = 'ACTIVE'))
		ORDER BY a.display_name COLLATE NOCASE ASC, wp.actor_id ASC
	`, workspaceID, boolToInt(includeInactive))
	if err != nil {
		return nil, fmt.Errorf("list principals: %w", err)
	}
	defer rows.Close()

	items := make([]identityPrincipalRow, 0)
	for rows.Next() {
		var item identityPrincipalRow
		if err := rows.Scan(
			&item.ActorID,
			&item.DisplayName,
			&item.ActorType,
			&item.ActorStatus,
			&item.PrincipalStatus,
		); err != nil {
			return nil, fmt.Errorf("scan principal row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate principal rows: %w", err)
	}
	return items, nil
}

func (s *IdentityDirectoryService) loadActorHandles(ctx context.Context, workspaceID string) (map[string][]transport.IdentityActorHandleRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			actor_id,
			channel,
			handle_value,
			is_primary,
			updated_at
		FROM actor_handles
		WHERE workspace_id = ?
		ORDER BY actor_id ASC, channel ASC, is_primary DESC, handle_value ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list actor handles: %w", err)
	}
	defer rows.Close()

	handlesByActor := make(map[string][]transport.IdentityActorHandleRecord)
	for rows.Next() {
		var item identityHandleRow
		var isPrimary int
		if err := rows.Scan(
			&item.ActorID,
			&item.Channel,
			&item.HandleValue,
			&isPrimary,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan actor handle row: %w", err)
		}
		item.IsPrimary = isPrimary != 0
		handlesByActor[item.ActorID] = append(handlesByActor[item.ActorID], transport.IdentityActorHandleRecord{
			Channel:     item.Channel,
			HandleValue: item.HandleValue,
			IsPrimary:   item.IsPrimary,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate actor handle rows: %w", err)
	}
	return handlesByActor, nil
}

func (s *IdentityDirectoryService) workspaceExists(ctx context.Context, workspaceID string) (bool, error) {
	normalizedWorkspace := normalizeWorkspaceID(workspaceID)
	if normalizedWorkspace == "" || isReservedSystemWorkspaceID(normalizedWorkspace) {
		return false, nil
	}

	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT 1
		FROM workspaces
		WHERE id = ?
		LIMIT 1
	`, normalizedWorkspace).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("lookup workspace: %w", err)
	}
	return true, nil
}

func (s *IdentityDirectoryService) principalExists(ctx context.Context, workspaceID string, actorID string) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT 1
		FROM workspace_principals wp
		JOIN actors a
			ON a.id = wp.actor_id
			AND a.workspace_id = wp.workspace_id
		WHERE wp.workspace_id = ?
			AND wp.actor_id = ?
		LIMIT 1
	`, workspaceID, strings.TrimSpace(actorID)).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("lookup principal: %w", err)
	}
	return true, nil
}

func (s *IdentityDirectoryService) firstWorkspaceID(ctx context.Context) (string, bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM workspaces
		ORDER BY CASE WHEN UPPER(status) = 'ACTIVE' THEN 0 ELSE 1 END, created_at ASC, id ASC
	`)
	if err != nil {
		return "", false, fmt.Errorf("load first workspace: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var workspaceID string
		if err := rows.Scan(&workspaceID); err != nil {
			return "", false, fmt.Errorf("scan first workspace row: %w", err)
		}
		normalizedWorkspaceID := normalizeWorkspaceID(workspaceID)
		if normalizedWorkspaceID == "" || isReservedSystemWorkspaceID(normalizedWorkspaceID) {
			continue
		}
		return normalizedWorkspaceID, true, nil
	}
	if err := rows.Err(); err != nil {
		return "", false, fmt.Errorf("iterate first workspace rows: %w", err)
	}
	return "", false, nil
}

func isReservedSystemWorkspaceID(workspaceID string) bool {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return false
	}
	_, ok := reservedSystemWorkspaceIDs[trimmed]
	return ok
}

func (s *IdentityDirectoryService) firstPrincipalActorID(ctx context.Context, workspaceID string) (string, bool, error) {
	var actorID string
	if err := s.db.QueryRowContext(ctx, `
		SELECT wp.actor_id
		FROM workspace_principals wp
		JOIN actors a
			ON a.id = wp.actor_id
			AND a.workspace_id = wp.workspace_id
		WHERE wp.workspace_id = ?
			AND UPPER(wp.status) = 'ACTIVE'
			AND UPPER(a.status) = 'ACTIVE'
		ORDER BY a.display_name COLLATE NOCASE ASC, wp.actor_id ASC
		LIMIT 1
	`, workspaceID).Scan(&actorID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("load first principal: %w", err)
	}
	return actorID, true, nil
}

type identityBootstrapAuditRecord struct {
	WorkspaceID      string
	PrincipalActorID string
	Source           string
	Timestamp        string
	Payload          map[string]any
}
