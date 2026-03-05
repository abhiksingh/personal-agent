package authz

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"personalagent/runtime/internal/core/contract"
)

type DelegationRuleStoreSQLite struct {
	db *sql.DB
}

func NewDelegationRuleStoreSQLite(db *sql.DB) *DelegationRuleStoreSQLite {
	return &DelegationRuleStoreSQLite{db: db}
}

func (s *DelegationRuleStoreSQLite) ListActiveDelegationRules(
	ctx context.Context,
	workspaceID string,
	fromActorID string,
	toActorID string,
	at time.Time,
) ([]contract.DelegationRule, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, workspace_id, from_actor_id, to_actor_id, scope_type, COALESCE(scope_key, ''), status, created_at, expires_at
		 FROM delegation_rules
		 WHERE workspace_id = ?
		   AND from_actor_id = ?
		   AND to_actor_id = ?
		   AND status = 'ACTIVE'
		   AND (expires_at IS NULL OR expires_at > ?)`,
		workspaceID,
		fromActorID,
		toActorID,
		at.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("list active delegation rules: %w", err)
	}
	defer rows.Close()

	rules := []contract.DelegationRule{}
	for rows.Next() {
		var (
			rule          contract.DelegationRule
			createdAtText string
			expiresAtText sql.NullString
		)
		if err := rows.Scan(
			&rule.ID,
			&rule.WorkspaceID,
			&rule.FromActorID,
			&rule.ToActorID,
			&rule.ScopeType,
			&rule.ScopeKey,
			&rule.Status,
			&createdAtText,
			&expiresAtText,
		); err != nil {
			return nil, fmt.Errorf("scan delegation rule: %w", err)
		}

		createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
		if err != nil {
			return nil, fmt.Errorf("parse delegation rule created_at: %w", err)
		}
		rule.CreatedAt = createdAt

		if expiresAtText.Valid {
			expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtText.String)
			if err != nil {
				return nil, fmt.Errorf("parse delegation rule expires_at: %w", err)
			}
			rule.ExpiresAt = &expiresAt
		}

		rules = append(rules, rule)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate delegation rules: %w", rows.Err())
	}

	return rules, nil
}
