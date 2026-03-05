package types

import "time"

const DestructiveApprovalPhrase = "GO AHEAD"

type ApprovalConfirmationRequest struct {
	WorkspaceID       string
	ApprovalRequestID string
	DecisionByActorID string
	Phrase            string
	RunID             string
	StepID            string
	CorrelationID     string
}

type ApprovalRecord struct {
	WorkspaceID       string
	ApprovalRequestID string
	DecisionByActorID string
	RunID             string
	StepID            string
	CorrelationID     string
	Phrase            string
	DecidedAt         time.Time
}
