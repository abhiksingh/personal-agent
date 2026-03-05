package contract

import (
	"context"
	"time"
)

type DelegationRule struct {
	ID          string
	WorkspaceID string
	FromActorID string
	ToActorID   string
	ScopeType   string
	ScopeKey    string
	Status      string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

type DelegationRuleStore interface {
	ListActiveDelegationRules(
		ctx context.Context,
		workspaceID string,
		fromActorID string,
		toActorID string,
		at time.Time,
	) ([]DelegationRule, error)
}
