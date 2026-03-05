package daemonruntime

import (
	"context"
	"time"
)

func (r *InboundWatcherRuntime) runLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		processed, err := r.pollOnce(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		if processed {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *InboundWatcherRuntime) pollOnce(ctx context.Context) (bool, error) {
	workDone := false
	var firstErr error
	workspaceID := r.currentWorkspaceID(ctx)

	processed, err := r.pollMessages(ctx, workspaceID)
	if err != nil && firstErr == nil {
		firstErr = err
	}
	if processed {
		workDone = true
	}

	for _, adapter := range r.fileAdapters {
		processed, err := r.pollFileAdapter(ctx, adapter, workspaceID)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if processed {
			workDone = true
		}
	}

	return workDone, firstErr
}
