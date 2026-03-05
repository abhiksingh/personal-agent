package daemonruntime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/workspaceid"
)

const cliSecretOwnerType = "CLI_SECRET"

var ErrSecretReferenceNotFound = errors.New("secret reference not found")

type RegisteredSecretReference struct {
	WorkspaceID string
	Name        string
	Backend     string
	Service     string
	Account     string
	CreatedAt   time.Time
}

func (c *ServiceContainer) RegisterSecretReference(ctx context.Context, reference securestore.SecretReference) (RegisteredSecretReference, error) {
	workspaceID := normalizeWorkspaceID(reference.WorkspaceID)
	name := strings.TrimSpace(reference.Name)
	backend := strings.TrimSpace(reference.Backend)
	service := strings.TrimSpace(reference.Service)
	account := strings.TrimSpace(reference.Account)

	if name == "" {
		return RegisteredSecretReference{}, fmt.Errorf("secret name is required")
	}
	if service == "" {
		return RegisteredSecretReference{}, fmt.Errorf("secret service is required")
	}
	if account == "" {
		return RegisteredSecretReference{}, fmt.Errorf("secret account is required")
	}

	now := time.Now().UTC()
	nowString := now.Format(time.RFC3339Nano)

	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return RegisteredSecretReference{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, nowString); err != nil {
		return RegisteredSecretReference{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
	`, workspaceID, cliSecretOwnerType, name); err != nil {
		return RegisteredSecretReference{}, fmt.Errorf("clear existing secret refs: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_refs(
			id,
			workspace_id,
			owner_type,
			owner_id,
			keychain_account,
			keychain_service,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, secretReferenceRecordID(workspaceID, name, service, account), workspaceID, cliSecretOwnerType, name, account, service, nowString); err != nil {
		return RegisteredSecretReference{}, fmt.Errorf("insert secret ref: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RegisteredSecretReference{}, fmt.Errorf("commit tx: %w", err)
	}

	return RegisteredSecretReference{
		WorkspaceID: workspaceID,
		Name:        name,
		Backend:     backend,
		Service:     service,
		Account:     account,
		CreatedAt:   now,
	}, nil
}

func (c *ServiceContainer) GetSecretReference(ctx context.Context, workspaceID string, name string) (RegisteredSecretReference, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	secretName := strings.TrimSpace(name)
	if secretName == "" {
		return RegisteredSecretReference{}, fmt.Errorf("secret name is required")
	}

	found, loaded, err := c.getSecretReferenceByWorkspace(ctx, workspace, secretName)
	if err != nil {
		return RegisteredSecretReference{}, err
	}
	if !found {
		return RegisteredSecretReference{}, ErrSecretReferenceNotFound
	}
	loaded.WorkspaceID = workspace
	return loaded, nil
}

func (c *ServiceContainer) getSecretReferenceByWorkspace(
	ctx context.Context,
	workspaceID string,
	secretName string,
) (bool, RegisteredSecretReference, error) {
	var (
		service    string
		account    string
		createdRaw string
	)
	err := c.DB.QueryRowContext(ctx, `
		SELECT keychain_service, keychain_account, created_at
		FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
		LIMIT 1
	`, workspaceID, cliSecretOwnerType, secretName).Scan(&service, &account, &createdRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, RegisteredSecretReference{}, nil
	}
	if err != nil {
		return false, RegisteredSecretReference{}, fmt.Errorf("query secret ref: %w", err)
	}

	createdAt, err := parseTimestamp(createdRaw)
	if err != nil {
		return false, RegisteredSecretReference{}, err
	}

	return true, RegisteredSecretReference{
		WorkspaceID: workspaceID,
		Name:        secretName,
		Service:     service,
		Account:     account,
		CreatedAt:   createdAt,
	}, nil
}

func (c *ServiceContainer) DeleteSecretReference(ctx context.Context, workspaceID string, name string) (RegisteredSecretReference, error) {
	current, err := c.GetSecretReference(ctx, workspaceID, name)
	if err != nil {
		return RegisteredSecretReference{}, err
	}

	if _, err := c.DB.ExecContext(ctx, `
		DELETE FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
	`, current.WorkspaceID, cliSecretOwnerType, current.Name); err != nil {
		return RegisteredSecretReference{}, fmt.Errorf("delete secret ref: %w", err)
	}

	return current, nil
}

func ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, now string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, workspaceID, workspaceID, now, now); err != nil {
		return fmt.Errorf("ensure workspace %s: %w", workspaceID, err)
	}
	return nil
}

func normalizeWorkspaceID(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func secretReferenceRecordID(workspaceID string, name string, service string, account string) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{workspaceID, name, service, account}, "|")))
	return "sref_" + hex.EncodeToString(hash[:16])
}

func parseTimestamp(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("missing timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", trimmed, err)
	}
	return parsed.UTC(), nil
}
