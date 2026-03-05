package authz

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	repoauthz "personalagent/runtime/internal/core/repository/authz"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupAuthzDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "authz.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	seedIdentity(t, db)
	return db
}

func seedIdentity(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_1', 'Workspace', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_a', 'ws_1', 'human', 'Actor A', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_b', 'ws_1', 'human', 'Actor B', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_a', 'ws_1', 'actor_a', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_b', 'ws_1', 'actor_b', 'ACTIVE', '` + now + `', '` + now + `')`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed identity statement failed: %v", err)
		}
	}
}

func insertDelegationRule(t *testing.T, db *sql.DB, id, scopeType, scopeKey, status string, expiresAt *time.Time) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var expires any
	if expiresAt != nil {
		expires = expiresAt.UTC().Format(time.RFC3339Nano)
	} else {
		expires = nil
	}
	_, err := db.Exec(
		`INSERT INTO delegation_rules(
			id, workspace_id, from_actor_id, to_actor_id, scope_type, scope_key, status, created_at, expires_at
		) VALUES (?, 'ws_1', 'actor_a', 'actor_b', ?, ?, ?, ?, ?)`,
		id,
		scopeType,
		scopeKey,
		status,
		now,
		expires,
	)
	if err != nil {
		t.Fatalf("insert delegation rule: %v", err)
	}
}

func TestCanActAsAllowsSelfExecution(t *testing.T) {
	authorizer := NewActingAsAuthorizer(nil)
	decision, err := authorizer.CanActAs(context.Background(), types.ActingAsRequest{
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_a",
		ActingAsActorID:    "actor_a",
	})
	if err != nil {
		t.Fatalf("can act as self: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected self execution to be allowed")
	}
}

func TestCanActAsDeniesCrossPrincipalWithoutRule(t *testing.T) {
	db := setupAuthzDB(t)
	authorizer := NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(db))

	decision, err := authorizer.CanActAs(context.Background(), types.ActingAsRequest{
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_a",
		ActingAsActorID:    "actor_b",
	})
	if err != nil {
		t.Fatalf("can act as cross principal without rule: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected cross-principal execution to be denied without delegation")
	}
}

func TestCanActAsAllowsCrossPrincipalWithValidRule(t *testing.T) {
	db := setupAuthzDB(t)
	insertDelegationRule(t, db, "dr_valid", "EXECUTION", "", "ACTIVE", nil)
	authorizer := NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(db))

	decision, err := authorizer.CanActAs(context.Background(), types.ActingAsRequest{
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_a",
		ActingAsActorID:    "actor_b",
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		t.Fatalf("can act as with valid rule: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected cross-principal execution to be allowed with valid rule")
	}
	if decision.DelegationRuleID != "dr_valid" {
		t.Fatalf("expected matched rule dr_valid, got %s", decision.DelegationRuleID)
	}
}

func TestCanActAsDeniesExpiredDelegationRule(t *testing.T) {
	db := setupAuthzDB(t)
	expired := time.Now().UTC().Add(-1 * time.Hour)
	insertDelegationRule(t, db, "dr_expired", "EXECUTION", "", "ACTIVE", &expired)
	authorizer := NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(db))

	decision, err := authorizer.CanActAs(context.Background(), types.ActingAsRequest{
		WorkspaceID:        "ws_1",
		RequestedByActorID: "actor_a",
		ActingAsActorID:    "actor_b",
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		t.Fatalf("can act as with expired rule: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected cross-principal execution to be denied for expired rule")
	}
}
