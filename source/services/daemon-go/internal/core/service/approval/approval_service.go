package approval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type Service struct {
	store contract.ApprovalAuditStore
	now   func() time.Time
}

func NewService(store contract.ApprovalAuditStore, nowFn func() time.Time) *Service {
	now := nowFn
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, now: now}
}

func (s *Service) ConfirmDestructiveApproval(ctx context.Context, req types.ApprovalConfirmationRequest) error {
	if s.store == nil {
		return fmt.Errorf("approval store is required")
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if strings.TrimSpace(req.ApprovalRequestID) == "" {
		return fmt.Errorf("approval_request_id is required")
	}
	if strings.TrimSpace(req.DecisionByActorID) == "" {
		return fmt.Errorf("decision_by_actor_id is required")
	}
	if strings.TrimSpace(req.Phrase) != types.DestructiveApprovalPhrase {
		return fmt.Errorf("approval phrase must be exact %q", types.DestructiveApprovalPhrase)
	}

	record := types.ApprovalRecord{
		WorkspaceID:       req.WorkspaceID,
		ApprovalRequestID: req.ApprovalRequestID,
		DecisionByActorID: req.DecisionByActorID,
		RunID:             req.RunID,
		StepID:            req.StepID,
		CorrelationID:     req.CorrelationID,
		Phrase:            req.Phrase,
		DecidedAt:         s.now(),
	}

	return s.store.RecordApprovalGranted(ctx, record)
}
