package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type workerDispatchAttemptSpec[T any] struct {
	resilience      *workerDispatchResilience
	workerID        string
	execute         func(context.Context) (T, error)
	isRetryable     func(error) bool
	recoverWorker   func(context.Context)
	exhaustedErrMsg string
}

func executeWorkerDispatchWithResilience[T any](ctx context.Context, spec workerDispatchAttemptSpec[T]) (T, error) {
	var zero T
	if spec.execute == nil {
		return zero, fmt.Errorf("worker dispatch execute callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if spec.resilience == nil {
		return spec.execute(ctx)
	}

	if err := spec.resilience.allow(spec.workerID); err != nil {
		return zero, err
	}

	attempts := spec.resilience.attempts()
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		attemptCtx, cancel := spec.resilience.operationContext(ctx)
		result, err := spec.execute(attemptCtx)
		cancel()
		if err == nil {
			spec.resilience.recordSuccess(spec.workerID)
			return result, nil
		}
		lastErr = err

		retryable := true
		if spec.isRetryable != nil {
			retryable = spec.isRetryable(err)
		}
		spec.resilience.recordFailure(spec.workerID, retryable)
		if !retryable || attempt == attempts-1 {
			return zero, err
		}

		if spec.recoverWorker != nil {
			spec.recoverWorker(ctx)
		}
		if sleepErr := spec.resilience.sleepBackoff(ctx, attempt); sleepErr != nil {
			return zero, sleepErr
		}
	}

	if lastErr != nil {
		return zero, lastErr
	}
	message := strings.TrimSpace(spec.exhaustedErrMsg)
	if message == "" {
		message = "worker dispatch failed without explicit error"
	}
	return zero, errors.New(message)
}
