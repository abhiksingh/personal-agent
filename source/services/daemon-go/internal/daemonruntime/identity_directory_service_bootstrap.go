package daemonruntime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (s *IdentityDirectoryService) Bootstrap(ctx context.Context, request transport.IdentityBootstrapRequest) (transport.IdentityBootstrapResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	workspaceName := strings.TrimSpace(request.WorkspaceName)
	if workspaceName == "" {
		workspaceName = workspaceID
	}
	workspaceStatus, err := normalizeIdentityBootstrapStatus(request.WorkspaceStatus)
	if err != nil {
		return transport.IdentityBootstrapResponse{}, err
	}

	principalActorID := strings.TrimSpace(request.PrincipalActorID)
	if principalActorID == "" {
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("principal_actor_id is required")
	}
	principalDisplayName := strings.TrimSpace(request.PrincipalDisplayName)
	if principalDisplayName == "" {
		principalDisplayName = principalActorID
	}
	principalActorType := normalizeIdentityBootstrapActorType(request.PrincipalActorType)
	principalStatus, err := normalizeIdentityBootstrapStatus(request.PrincipalStatus)
	if err != nil {
		return transport.IdentityBootstrapResponse{}, err
	}

	handleChannel := ""
	handleValue := ""
	handlePrimary := false
	if request.Handle != nil {
		handleChannel = strings.TrimSpace(request.Handle.Channel)
		handleValue = strings.TrimSpace(request.Handle.HandleValue)
		handlePrimary = request.Handle.IsPrimary
		if handleChannel == "" || handleValue == "" {
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("handle channel and handle_value are required when handle is provided")
		}
	}

	source := strings.TrimSpace(request.Source)
	if source == "" {
		source = "cli"
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	response := transport.IdentityBootstrapResponse{
		WorkspaceID:      workspaceID,
		PrincipalActorID: principalActorID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingWorkspaceName string
	var existingWorkspaceStatus string
	switch err := tx.QueryRowContext(ctx, `
		SELECT name, status
		FROM workspaces
		WHERE id = ?
		LIMIT 1
	`, workspaceID).Scan(&existingWorkspaceName, &existingWorkspaceStatus); err {
	case sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO workspaces(id, name, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, workspaceID, workspaceName, workspaceStatus, now, now); err != nil {
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("insert workspace: %w", err)
		}
		response.WorkspaceCreated = true
	case nil:
		if strings.TrimSpace(existingWorkspaceName) != workspaceName || !strings.EqualFold(strings.TrimSpace(existingWorkspaceStatus), workspaceStatus) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE workspaces
				SET name = ?, status = ?, updated_at = ?
				WHERE id = ?
			`, workspaceName, workspaceStatus, now, workspaceID); err != nil {
				return transport.IdentityBootstrapResponse{}, fmt.Errorf("update workspace: %w", err)
			}
			response.WorkspaceUpdated = true
		}
	default:
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("load workspace: %w", err)
	}

	var actorWorkspaceID string
	var existingActorType string
	var existingActorDisplayName string
	var existingActorStatus string
	switch err := tx.QueryRowContext(ctx, `
		SELECT workspace_id, actor_type, display_name, status
		FROM actors
		WHERE id = ?
		LIMIT 1
	`, principalActorID).Scan(&actorWorkspaceID, &existingActorType, &existingActorDisplayName, &existingActorStatus); err {
	case sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, principalActorID, workspaceID, principalActorType, principalDisplayName, principalStatus, now, now); err != nil {
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("insert actor: %w", err)
		}
		response.PrincipalCreated = true
	case nil:
		if strings.TrimSpace(actorWorkspaceID) != workspaceID {
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("principal actor %q already belongs to workspace %q", principalActorID, actorWorkspaceID)
		}
		if strings.TrimSpace(existingActorDisplayName) != principalDisplayName ||
			!strings.EqualFold(strings.TrimSpace(existingActorType), principalActorType) ||
			!strings.EqualFold(strings.TrimSpace(existingActorStatus), principalStatus) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE actors
				SET actor_type = ?, display_name = ?, status = ?, updated_at = ?
				WHERE id = ?
				  AND workspace_id = ?
			`, principalActorType, principalDisplayName, principalStatus, now, principalActorID, workspaceID); err != nil {
				return transport.IdentityBootstrapResponse{}, fmt.Errorf("update actor: %w", err)
			}
			response.PrincipalUpdated = true
		}
	default:
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("load actor: %w", err)
	}

	var principalLinkID string
	var existingPrincipalStatus string
	switch err := tx.QueryRowContext(ctx, `
		SELECT id, status
		FROM workspace_principals
		WHERE workspace_id = ?
		  AND actor_id = ?
		LIMIT 1
	`, workspaceID, principalActorID).Scan(&principalLinkID, &existingPrincipalStatus); err {
	case sql.ErrNoRows:
		principalLinkID = identityBootstrapStableID("wp", workspaceID, principalActorID)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, principalLinkID, workspaceID, principalActorID, principalStatus, now, now); err != nil {
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("insert workspace principal: %w", err)
		}
		response.PrincipalLinked = true
	case nil:
		if !strings.EqualFold(strings.TrimSpace(existingPrincipalStatus), principalStatus) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE workspace_principals
				SET status = ?, updated_at = ?
				WHERE id = ?
			`, principalStatus, now, principalLinkID); err != nil {
				return transport.IdentityBootstrapResponse{}, fmt.Errorf("update workspace principal status: %w", err)
			}
			response.PrincipalUpdated = true
		}
	default:
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("load workspace principal link: %w", err)
	}

	if handleChannel != "" && handleValue != "" {
		var handleID string
		var handleActorID string
		var handleIsPrimary int
		var handleUpdatedAt string
		switch err := tx.QueryRowContext(ctx, `
			SELECT id, actor_id, is_primary, updated_at
			FROM actor_handles
			WHERE workspace_id = ?
			  AND channel = ?
			  AND handle_value = ?
			LIMIT 1
		`, workspaceID, handleChannel, handleValue).Scan(&handleID, &handleActorID, &handleIsPrimary, &handleUpdatedAt); err {
		case sql.ErrNoRows:
			if handlePrimary {
				if _, err := tx.ExecContext(ctx, `
					UPDATE actor_handles
					SET is_primary = 0, updated_at = ?
					WHERE workspace_id = ?
					  AND actor_id = ?
					  AND channel = ?
					  AND is_primary = 1
				`, now, workspaceID, principalActorID, handleChannel); err != nil {
					return transport.IdentityBootstrapResponse{}, fmt.Errorf("clear previous primary handles: %w", err)
				}
			}
			handleID = identityBootstrapStableID("ah", workspaceID, handleChannel, handleValue)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO actor_handles(id, workspace_id, actor_id, channel, handle_value, is_primary, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, handleID, workspaceID, principalActorID, handleChannel, handleValue, boolToInt(handlePrimary), now, now); err != nil {
				return transport.IdentityBootstrapResponse{}, fmt.Errorf("insert actor handle: %w", err)
			}
			response.HandleCreated = true
			response.Handle = &transport.IdentityActorHandleRecord{
				Channel:     handleChannel,
				HandleValue: handleValue,
				IsPrimary:   handlePrimary,
				UpdatedAt:   now,
			}
		case nil:
			if strings.TrimSpace(handleActorID) != principalActorID {
				return transport.IdentityBootstrapResponse{}, fmt.Errorf("handle %q on channel %q is already assigned to actor %q", handleValue, handleChannel, handleActorID)
			}
			if handlePrimary {
				if _, err := tx.ExecContext(ctx, `
					UPDATE actor_handles
					SET is_primary = 0, updated_at = ?
					WHERE workspace_id = ?
					  AND actor_id = ?
					  AND channel = ?
					  AND id <> ?
					  AND is_primary = 1
				`, now, workspaceID, principalActorID, handleChannel, handleID); err != nil {
					return transport.IdentityBootstrapResponse{}, fmt.Errorf("clear previous primary handles: %w", err)
				}
			}
			shouldUpdatePrimary := handlePrimary && handleIsPrimary == 0
			if shouldUpdatePrimary {
				if _, err := tx.ExecContext(ctx, `
					UPDATE actor_handles
					SET is_primary = 1, updated_at = ?
					WHERE id = ?
				`, now, handleID); err != nil {
					return transport.IdentityBootstrapResponse{}, fmt.Errorf("update actor handle primary state: %w", err)
				}
				response.HandleUpdated = true
				handleUpdatedAt = now
				handleIsPrimary = 1
			}
			response.Handle = &transport.IdentityActorHandleRecord{
				Channel:     handleChannel,
				HandleValue: handleValue,
				IsPrimary:   handleIsPrimary != 0,
				UpdatedAt:   handleUpdatedAt,
			}
		default:
			return transport.IdentityBootstrapResponse{}, fmt.Errorf("load actor handle: %w", err)
		}
	}

	response.Idempotent = !response.WorkspaceCreated &&
		!response.WorkspaceUpdated &&
		!response.PrincipalCreated &&
		!response.PrincipalUpdated &&
		!response.PrincipalLinked &&
		!response.HandleCreated &&
		!response.HandleUpdated

	auditID, err := insertIdentityBootstrapAudit(ctx, tx, identityBootstrapAuditRecord{
		WorkspaceID:      workspaceID,
		PrincipalActorID: principalActorID,
		Source:           source,
		Timestamp:        now,
		Payload: map[string]any{
			"workspace_id":       workspaceID,
			"workspace_name":     workspaceName,
			"workspace_status":   workspaceStatus,
			"principal_actor_id": principalActorID,
			"principal_status":   principalStatus,
			"source":             source,
			"workspace_created":  response.WorkspaceCreated,
			"workspace_updated":  response.WorkspaceUpdated,
			"principal_created":  response.PrincipalCreated,
			"principal_updated":  response.PrincipalUpdated,
			"principal_linked":   response.PrincipalLinked,
			"handle_created":     response.HandleCreated,
			"handle_updated":     response.HandleUpdated,
			"idempotent":         response.Idempotent,
		},
	})
	if err != nil {
		return transport.IdentityBootstrapResponse{}, err
	}
	response.AuditLogID = auditID

	if err := tx.Commit(); err != nil {
		return transport.IdentityBootstrapResponse{}, fmt.Errorf("commit identity bootstrap tx: %w", err)
	}

	activeContextResponse, err := s.SelectWorkspace(ctx, transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      workspaceID,
		PrincipalActorID: principalActorID,
		Source:           source,
	})
	if err != nil {
		return transport.IdentityBootstrapResponse{}, err
	}
	response.ActiveContext = activeContextResponse.ActiveContext
	return response, nil
}

func insertIdentityBootstrapAudit(ctx context.Context, tx *sql.Tx, record identityBootstrapAuditRecord) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("insert identity bootstrap audit: nil tx")
	}
	workspaceID := normalizeWorkspaceID(record.WorkspaceID)
	actorID := strings.TrimSpace(record.PrincipalActorID)
	timestamp := strings.TrimSpace(record.Timestamp)
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	payloadJSON := "{}"
	if record.Payload != nil {
		raw, err := json.Marshal(record.Payload)
		if err != nil {
			return "", fmt.Errorf("marshal identity bootstrap audit payload: %w", err)
		}
		payloadJSON = string(raw)
	}

	auditID := identityBootstrapAuditID(workspaceID, actorID, timestamp)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, NULL, NULL, ?, ?, ?, NULL, ?, ?)
	`, auditID, workspaceID, identityBootstrapAuditEventType, nullableText(actorID), nullableText(actorID), payloadJSON, timestamp); err != nil {
		return "", fmt.Errorf("insert identity bootstrap audit entry: %w", err)
	}
	return auditID, nil
}

func normalizeIdentityBootstrapStatus(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "" {
		return identityBootstrapDefaultStatus, nil
	}
	switch normalized {
	case "ACTIVE", "INACTIVE":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported status %q (allowed: ACTIVE|INACTIVE)", strings.TrimSpace(raw))
	}
}

func normalizeIdentityBootstrapActorType(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return identityBootstrapDefaultActorType
	}
	return normalized
}

func identityBootstrapStableID(prefix string, parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return strings.TrimSpace(prefix) + "_" + hex.EncodeToString(hash[:12])
}

func identityBootstrapAuditID(workspaceID string, principalActorID string, timestamp string) string {
	return identityBootstrapStableID("audit_identity_bootstrap", workspaceID, principalActorID, timestamp)
}
