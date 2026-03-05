package contract

import (
	"context"

	"personalagent/runtime/internal/core/types"
)

type ApprovalAuditStore interface {
	RecordApprovalGranted(ctx context.Context, record types.ApprovalRecord) error
}
