package approval

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
)

type SQLiteApprovalStore struct {
	db *sql.DB
}

func NewSQLiteApprovalStore(db *sql.DB) *SQLiteApprovalStore {
	return &SQLiteApprovalStore{db: db}
}

func (s *SQLiteApprovalStore) RecordApprovalGranted(ctx context.Context, record types.ApprovalRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin approval transaction: %w", err)
	}

	decidedAtText := record.DecidedAt.UTC().Format(time.RFC3339Nano)
	existingRationale := ""
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(rationale, '')
		 FROM approval_requests
		 WHERE id = ?
		   AND workspace_id = ?`,
		record.ApprovalRequestID,
		record.WorkspaceID,
	).Scan(&existingRationale); err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return fmt.Errorf("load existing approval rationale: %w", err)
	}
	decisionRationale, err := approvalDecisionRationaleJSON(existingRationale, record, decidedAtText)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	res, err := tx.ExecContext(
		ctx,
		`UPDATE approval_requests
		 SET decision = 'APPROVED',
		     decision_by_actor_id = ?,
		     decided_at = ?,
		     rationale = ?
		 WHERE id = ?
		   AND workspace_id = ?`,
		record.DecisionByActorID,
		decidedAtText,
		decisionRationale,
		record.ApprovalRequestID,
		record.WorkspaceID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update approval request: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read approval rows affected: %w", err)
	}
	if rows != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("approval request not found or not unique")
	}

	auditID, err := randomID()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"approval_request_id": record.ApprovalRequestID,
		"phrase":              record.Phrase,
	})
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("marshal approval audit payload: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type,
			actor_id, acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, ?, ?, 'APPROVAL_GRANTED', ?, ?, ?, ?, ?)`,
		auditID,
		record.WorkspaceID,
		nullIfEmpty(record.RunID),
		nullIfEmpty(record.StepID),
		record.DecisionByActorID,
		record.DecisionByActorID,
		nullIfEmpty(record.CorrelationID),
		string(payload),
		decidedAtText,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert approval audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit approval transaction: %w", err)
	}
	return nil
}

func approvalDecisionRationaleJSON(existingRationale string, record types.ApprovalRecord, decidedAtText string) (string, error) {
	trimmed := strings.TrimSpace(existingRationale)
	rationale := map[string]any{}
	if trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &rationale); err != nil || rationale == nil {
			rationale = map[string]any{}
		}
	}

	rationale["decision"] = "approved"
	rationale["decision_reason"] = "approval phrase validated"
	rationale["decision_reason_code"] = "approval_phrase_validated"
	rationale["decision_source"] = "approval_phrase"
	rationale["decision_actor_id"] = strings.TrimSpace(record.DecisionByActorID)
	rationale["decision_phrase"] = strings.TrimSpace(record.Phrase)
	rationale["decision_at"] = decidedAtText

	payload, err := json.Marshal(rationale)
	if err != nil {
		return "", fmt.Errorf("marshal approval decision rationale: %w", err)
	}
	return string(payload), nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
