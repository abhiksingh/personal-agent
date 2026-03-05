package daemonruntime

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	defaultWorkerDispatchOperationTimeout     = 3 * time.Second
	defaultWorkerDispatchMaxRetries           = 2
	defaultWorkerDispatchRetryBackoffBase     = 50 * time.Millisecond
	defaultWorkerDispatchRetryBackoffMax      = 500 * time.Millisecond
	defaultWorkerDispatchRetryJitterFraction  = 0.2
	defaultWorkerDispatchCircuitOpenThreshold = 4
	defaultWorkerDispatchCircuitOpenCooldown  = 5 * time.Second
)

type workerDispatchResilienceOptions struct {
	OperationTimeout     time.Duration
	MaxRetries           int
	RetryBackoffBase     time.Duration
	RetryBackoffMax      time.Duration
	RetryJitterFraction  float64
	CircuitOpenThreshold int
	CircuitOpenCooldown  time.Duration
	Now                  func() time.Time
	Rand                 *rand.Rand
}

type workerDispatchResilience struct {
	operationTimeout     time.Duration
	maxRetries           int
	retryBackoffBase     time.Duration
	retryBackoffMax      time.Duration
	retryJitterFraction  float64
	circuitOpenThreshold int
	circuitOpenCooldown  time.Duration
	now                  func() time.Time

	rngMu sync.Mutex
	rng   *rand.Rand

	circuitMu    sync.Mutex
	circuitState map[string]workerDispatchCircuitState
}

type workerDispatchCircuitState struct {
	failures  int
	openUntil time.Time
}

type workerDispatchCircuitOpenError struct {
	WorkerID string
	Until    time.Time
}

func defaultWorkerDispatchResilience() *workerDispatchResilience {
	return newWorkerDispatchResilience(workerDispatchResilienceOptions{
		OperationTimeout:     defaultWorkerDispatchOperationTimeout,
		MaxRetries:           defaultWorkerDispatchMaxRetries,
		RetryBackoffBase:     defaultWorkerDispatchRetryBackoffBase,
		RetryBackoffMax:      defaultWorkerDispatchRetryBackoffMax,
		RetryJitterFraction:  defaultWorkerDispatchRetryJitterFraction,
		CircuitOpenThreshold: defaultWorkerDispatchCircuitOpenThreshold,
		CircuitOpenCooldown:  defaultWorkerDispatchCircuitOpenCooldown,
	})
}

func (e *workerDispatchCircuitOpenError) Error() string {
	if e == nil {
		return "worker dispatch circuit is open"
	}
	workerID := strings.TrimSpace(e.WorkerID)
	if workerID == "" {
		workerID = "unknown-worker"
	}
	return fmt.Sprintf("worker dispatch circuit open for %s until %s", workerID, e.Until.UTC().Format(time.RFC3339))
}

func newWorkerDispatchResilience(opts workerDispatchResilienceOptions) *workerDispatchResilience {
	operationTimeout := opts.OperationTimeout
	if operationTimeout <= 0 {
		operationTimeout = defaultWorkerDispatchOperationTimeout
	}
	maxRetries := opts.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	retryBackoffBase := opts.RetryBackoffBase
	if retryBackoffBase <= 0 {
		retryBackoffBase = defaultWorkerDispatchRetryBackoffBase
	}
	retryBackoffMax := opts.RetryBackoffMax
	if retryBackoffMax <= 0 {
		retryBackoffMax = defaultWorkerDispatchRetryBackoffMax
	}
	if retryBackoffMax < retryBackoffBase {
		retryBackoffMax = retryBackoffBase
	}
	retryJitterFraction := opts.RetryJitterFraction
	if retryJitterFraction < 0 {
		retryJitterFraction = 0
	}
	if retryJitterFraction > 1 {
		retryJitterFraction = 1
	}

	circuitOpenThreshold := opts.CircuitOpenThreshold
	if circuitOpenThreshold <= 0 {
		circuitOpenThreshold = defaultWorkerDispatchCircuitOpenThreshold
	}
	circuitOpenCooldown := opts.CircuitOpenCooldown
	if circuitOpenCooldown <= 0 {
		circuitOpenCooldown = defaultWorkerDispatchCircuitOpenCooldown
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	rng := opts.Rand
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return &workerDispatchResilience{
		operationTimeout:     operationTimeout,
		maxRetries:           maxRetries,
		retryBackoffBase:     retryBackoffBase,
		retryBackoffMax:      retryBackoffMax,
		retryJitterFraction:  retryJitterFraction,
		circuitOpenThreshold: circuitOpenThreshold,
		circuitOpenCooldown:  circuitOpenCooldown,
		now:                  now,
		rng:                  rng,
		circuitState:         map[string]workerDispatchCircuitState{},
	}
}

func (r *workerDispatchResilience) operationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if r == nil || r.operationTimeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, r.operationTimeout)
}

func (r *workerDispatchResilience) allow(workerID string) error {
	if r == nil {
		return nil
	}
	trimmedWorkerID := strings.TrimSpace(workerID)
	if trimmedWorkerID == "" {
		return nil
	}

	now := r.now()
	r.circuitMu.Lock()
	defer r.circuitMu.Unlock()
	state, ok := r.circuitState[trimmedWorkerID]
	if !ok {
		return nil
	}
	if state.openUntil.IsZero() || !now.Before(state.openUntil) {
		state.openUntil = time.Time{}
		r.circuitState[trimmedWorkerID] = state
		return nil
	}
	return &workerDispatchCircuitOpenError{
		WorkerID: trimmedWorkerID,
		Until:    state.openUntil,
	}
}

func (r *workerDispatchResilience) recordSuccess(workerID string) {
	if r == nil {
		return
	}
	trimmedWorkerID := strings.TrimSpace(workerID)
	if trimmedWorkerID == "" {
		return
	}
	r.circuitMu.Lock()
	defer r.circuitMu.Unlock()
	r.circuitState[trimmedWorkerID] = workerDispatchCircuitState{}
}

func (r *workerDispatchResilience) recordFailure(workerID string, retryable bool) {
	if r == nil || !retryable {
		return
	}
	trimmedWorkerID := strings.TrimSpace(workerID)
	if trimmedWorkerID == "" {
		return
	}

	r.circuitMu.Lock()
	defer r.circuitMu.Unlock()
	state := r.circuitState[trimmedWorkerID]
	state.failures++
	if state.failures >= r.circuitOpenThreshold {
		state.failures = 0
		state.openUntil = r.now().Add(r.circuitOpenCooldown)
	}
	r.circuitState[trimmedWorkerID] = state
}

func (r *workerDispatchResilience) attempts() int {
	if r == nil {
		return 1
	}
	if r.maxRetries <= 0 {
		return 1
	}
	return r.maxRetries + 1
}

func (r *workerDispatchResilience) sleepBackoff(ctx context.Context, attempt int) error {
	if r == nil {
		return nil
	}
	delay := r.backoff(attempt)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *workerDispatchResilience) backoff(attempt int) time.Duration {
	if r == nil {
		return 0
	}
	if attempt < 0 {
		attempt = 0
	}
	backoff := r.retryBackoffBase
	for i := 0; i < attempt; i++ {
		backoff *= 2
		if backoff >= r.retryBackoffMax {
			backoff = r.retryBackoffMax
			break
		}
	}

	jitterFraction := r.retryJitterFraction
	if jitterFraction <= 0 {
		return backoff
	}
	minFactor := 1 - jitterFraction
	maxFactor := 1 + jitterFraction

	r.rngMu.Lock()
	factor := minFactor + (r.rng.Float64() * (maxFactor - minFactor))
	r.rngMu.Unlock()
	if factor <= 0 {
		factor = 1
	}
	return time.Duration(float64(backoff) * factor)
}
