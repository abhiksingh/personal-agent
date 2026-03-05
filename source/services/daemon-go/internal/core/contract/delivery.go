package contract

import (
	"context"
	"time"

	"personalagent/runtime/internal/core/types"
)

type DeliveryStore interface {
	ResolveDeliveryPolicy(
		ctx context.Context,
		workspaceID string,
		sourceChannel string,
		destinationEndpoint string,
	) (types.ChannelDeliveryPolicy, bool, error)
	ReserveDeliveryAttempt(ctx context.Context, attempt types.DeliveryAttemptRecord) (types.DeliveryAttemptRecord, bool, error)
	MarkDeliveryAttemptResult(
		ctx context.Context,
		attemptID string,
		status types.DeliveryAttemptStatus,
		providerReceipt string,
		errorText string,
		attemptedAt time.Time,
	) error
}

type DeliverySender interface {
	Send(ctx context.Context, channel string, request types.DeliveryRequest, idempotencyKey string) (providerReceipt string, err error)
}
