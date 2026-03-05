package types

import "time"

type ActingAsRequest struct {
	WorkspaceID        string
	RequestedByActorID string
	ActingAsActorID    string
	ScopeType          string
	ScopeKey           string
	At                 time.Time
}

type ActingAsDecision struct {
	Allowed          bool
	Reason           string
	DelegationRuleID string
}
