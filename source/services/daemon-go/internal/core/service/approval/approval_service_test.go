package approval

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	repoapproval "personalagent/runtime/internal/core/repository/approval"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupApprovalDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "approval.db")
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

	seedApprovalFixtures(t, db)
	return db
}

func seedApprovalFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws_app', 'Approval WS', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_req', 'ws_app', 'human', 'Requester', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor_approver', 'ws_app', 'human', 'Approver', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_req', 'ws_app', 'actor_req', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp_approver', 'ws_app', 'actor_approver', 'ACTIVE', '` + now + `', '` + now + `')`,
		`INSERT INTO tasks(id, workspace_id, requested_by_actor_id, subject_principal_actor_id, title, description, state, priority, deadline_at, channel, created_at, updated_at) VALUES ('task_app', 'ws_app', 'actor_req', 'actor_req', 'Task', 'desc', 'awaiting_approval', 0, NULL, 'app', '` + now + `', '` + now + `')`,
		`INSERT INTO task_runs(id, workspace_id, task_id, acting_as_actor_id, state, started_at, finished_at, last_error, created_at, updated_at) VALUES ('run_app', 'ws_app', 'task_app', 'actor_req', 'awaiting_approval', NULL, NULL, NULL, '` + now + `', '` + now + `')`,
		`INSERT INTO approval_requests(id, workspace_id, run_id, step_id, requested_phrase, decision, decision_by_actor_id, requested_at, decided_at, rationale) VALUES ('approval_1', 'ws_app', 'run_app', NULL, 'GO AHEAD', NULL, NULL, '` + now + `', NULL, NULL)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed approval fixtures: %v", err)
		}
	}
}

func TestConfirmDestructiveApprovalRejectsNonExactPhrase(t *testing.T) {
	db := setupApprovalDB(t)
	service := NewService(repoapproval.NewSQLiteApprovalStore(db), nil)

	err := service.ConfirmDestructiveApproval(context.Background(), types.ApprovalConfirmationRequest{
		WorkspaceID:       "ws_app",
		ApprovalRequestID: "approval_1",
		DecisionByActorID: "actor_approver",
		Phrase:            "go ahead",
		RunID:             "run_app",
	})
	if err == nil {
		t.Fatalf("expected non-exact phrase to be rejected")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_log_entries`).Scan(&count); err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no audit rows for rejected phrase, got %d", count)
	}
}

func TestConfirmDestructiveApprovalRecordsApprovalAndAudit(t *testing.T) {
	db := setupApprovalDB(t)
	service := NewService(repoapproval.NewSQLiteApprovalStore(db), func() time.Time {
		return time.Date(2026, 2, 23, 23, 0, 0, 0, time.UTC)
	})

	err := service.ConfirmDestructiveApproval(context.Background(), types.ApprovalConfirmationRequest{
		WorkspaceID:       "ws_app",
		ApprovalRequestID: "approval_1",
		DecisionByActorID: "actor_approver",
		Phrase:            types.DestructiveApprovalPhrase,
		RunID:             "run_app",
		CorrelationID:     "corr-123",
	})
	if err != nil {
		t.Fatalf("confirm destructive approval: %v", err)
	}

	var decision string
	var decisionBy string
	var rationaleRaw string
	if err := db.QueryRow(`SELECT decision, decision_by_actor_id, COALESCE(rationale, '') FROM approval_requests WHERE id = 'approval_1'`).Scan(&decision, &decisionBy, &rationaleRaw); err != nil {
		t.Fatalf("load approval request decision: %v", err)
	}
	if decision != "APPROVED" || decisionBy != "actor_approver" {
		t.Fatalf("unexpected approval request state decision=%s decision_by=%s", decision, decisionBy)
	}
	rationale := map[string]any{}
	if err := json.Unmarshal([]byte(rationaleRaw), &rationale); err != nil {
		t.Fatalf("parse approval rationale json: %v", err)
	}
	if rationale["decision_reason"] != "approval phrase validated" {
		t.Fatalf("expected decision_reason approval phrase validated, got %#v", rationale["decision_reason"])
	}
	if rationale["decision_reason_code"] != "approval_phrase_validated" {
		t.Fatalf("expected decision_reason_code approval_phrase_validated, got %#v", rationale["decision_reason_code"])
	}

	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_log_entries WHERE event_type = 'APPROVAL_GRANTED'`).Scan(&auditCount); err != nil {
		t.Fatalf("count approval audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one approval audit log, got %d", auditCount)
	}
}
