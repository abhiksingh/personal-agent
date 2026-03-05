package contract

import (
	"context"
	"time"
)

type RetentionStore interface {
	PurgeTraceDataBefore(ctx context.Context, cutoff time.Time) (int64, error)
	PurgeTranscriptDataBefore(ctx context.Context, cutoff time.Time) (int64, error)
	PurgeMemoryDataBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
