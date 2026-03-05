package daemonruntime

import (
	"context"
	"fmt"
	"time"
)

type pluginWorkerStatusLookupFunc func() (PluginWorkerStatus, bool)
type pluginWorkerStatusReadyFunc func(status PluginWorkerStatus) bool

func waitForPluginWorkerStatus(
	ctx context.Context,
	lookup pluginWorkerStatusLookupFunc,
	isReady pluginWorkerStatusReadyFunc,
	pollInterval time.Duration,
	timeout time.Duration,
	timeoutErr error,
) (PluginWorkerStatus, error) {
	if lookup == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin worker status lookup is required")
	}
	if isReady == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin worker readiness predicate is required")
	}
	if pollInterval <= 0 {
		pollInterval = 10 * time.Millisecond
	}
	if timeout < 0 {
		timeout = 0
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	done := workerWaitContextDone(ctx)
	for {
		select {
		case <-done:
			return PluginWorkerStatus{}, ctx.Err()
		default:
		}

		status, ok := lookup()
		if ok && isReady(status) {
			return status, nil
		}

		select {
		case <-done:
			return PluginWorkerStatus{}, ctx.Err()
		case <-timer.C:
			if timeoutErr != nil {
				return PluginWorkerStatus{}, timeoutErr
			}
			return PluginWorkerStatus{}, fmt.Errorf("timed out waiting for plugin worker status")
		case <-ticker.C:
		}
	}
}

func workerWaitContextDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}
