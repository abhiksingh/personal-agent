package writequeue

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Handler func(ctx context.Context, payload json.RawMessage) error

type Queue struct {
	db                *sql.DB
	handlers          map[string]Handler
	now               func() time.Time
	inProgressTimeout time.Duration
	retryBackoff      func(attempt int) time.Duration
}

type Options struct {
	Now               func() time.Time
	InProgressTimeout time.Duration
	RetryBackoff      func(attempt int) time.Duration
}

type claimedWrite struct {
	ID          string
	Operation   string
	PayloadJSON string
	Attempts    int
	MaxAttempts int
}

const completionPendingMarker = "__handler_succeeded_completion_pending__"

func New(db *sql.DB, opts Options) *Queue {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}

	timeout := opts.InProgressTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	backoff := opts.RetryBackoff
	if backoff == nil {
		backoff = func(attempt int) time.Duration {
			if attempt <= 0 {
				return 0
			}
			return time.Duration(attempt) * time.Second
		}
	}

	return &Queue{
		db:                db,
		handlers:          map[string]Handler{},
		now:               nowFn,
		inProgressTimeout: timeout,
		retryBackoff:      backoff,
	}
}

func (q *Queue) Register(operation string, handler Handler) error {
	if operation == "" {
		return fmt.Errorf("operation is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if _, exists := q.handlers[operation]; exists {
		return fmt.Errorf("handler already registered for operation %s", operation)
	}
	q.handlers[operation] = handler
	return nil
}

func (q *Queue) Enqueue(ctx context.Context, operation string, payload any, maxAttempts int) (string, error) {
	if operation == "" {
		return "", fmt.Errorf("operation is required")
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	now := q.now().Format(time.RFC3339Nano)
	id, err := newWriteID()
	if err != nil {
		return "", err
	}

	_, execErr := q.db.ExecContext(
		ctx,
		`INSERT INTO persistence_write_queue(
			id, operation, payload_json, status, attempt_count, max_attempts,
			available_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, 'pending', 0, ?, ?, NULL, ?, ?)`,
		id,
		operation,
		string(payloadBytes),
		maxAttempts,
		now,
		now,
		now,
	)
	if execErr != nil {
		return "", fmt.Errorf("enqueue write: %w", execErr)
	}

	return id, nil
}

func (q *Queue) Run(ctx context.Context, idleSleep time.Duration) error {
	if idleSleep <= 0 {
		idleSleep = 100 * time.Millisecond
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		processed, err := q.RunOnce(ctx)
		if err != nil {
			return err
		}
		if processed {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(idleSleep):
		}
	}
}

func (q *Queue) RunOnce(ctx context.Context) (bool, error) {
	if err := q.requeueStaleInProgress(ctx); err != nil {
		return false, err
	}

	finalizedCompletion, err := q.finalizeCompletionPending(ctx)
	if err != nil {
		return false, err
	}
	if finalizedCompletion {
		return true, nil
	}

	job, claimed, err := q.claimNext(ctx)
	if err != nil {
		return false, err
	}
	if !claimed {
		return false, nil
	}

	handler, ok := q.handlers[job.Operation]
	if !ok {
		if markErr := q.markFailed(ctx, job.ID, fmt.Sprintf("no handler for operation %s", job.Operation)); markErr != nil {
			return true, markErr
		}
		return true, nil
	}

	handlerErr := handler(ctx, json.RawMessage(job.PayloadJSON))
	if handlerErr != nil {
		if retryErr := q.retryOrFail(ctx, job, handlerErr.Error()); retryErr != nil {
			return true, retryErr
		}
		return true, nil
	}

	if err := q.markHandlerSucceeded(ctx, job.ID); err != nil {
		return true, err
	}
	if err := q.markCompleted(ctx, job.ID); err != nil {
		return true, err
	}

	return true, nil
}

func (q *Queue) markHandlerSucceeded(ctx context.Context, id string) error {
	now := q.now().Format(time.RFC3339Nano)
	result, err := q.db.ExecContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET last_error = ?, updated_at = ?
		 WHERE id = ? AND status = 'in_progress'`,
		completionPendingMarker,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark write handler_succeeded: %w", err)
	}
	rows, rowsErr := result.RowsAffected()
	if rowsErr == nil && rows == 0 {
		return fmt.Errorf("mark write handler_succeeded: no in_progress row for id %s", id)
	}
	return nil
}

func (q *Queue) markCompleted(ctx context.Context, id string) error {
	now := q.now().Format(time.RFC3339Nano)
	result, err := q.db.ExecContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET status = 'completed', last_error = NULL, updated_at = ?
		 WHERE id = ? AND status = 'in_progress' AND last_error = ?`,
		now,
		id,
		completionPendingMarker,
	)
	if err != nil {
		return fmt.Errorf("mark write completed: %w", err)
	}
	rows, rowsErr := result.RowsAffected()
	if rowsErr == nil && rows == 0 {
		return fmt.Errorf("mark write completed: no completion_pending row for id %s", id)
	}
	return nil
}

func (q *Queue) finalizeCompletionPending(ctx context.Context) (bool, error) {
	var id string
	if err := q.db.QueryRowContext(
		ctx,
		`SELECT id
		 FROM persistence_write_queue
		 WHERE status = 'in_progress' AND last_error = ?
		 ORDER BY updated_at ASC, created_at ASC, id ASC
		 LIMIT 1`,
		completionPendingMarker,
	).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("select completion_pending write: %w", err)
	}
	if err := q.markCompleted(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (q *Queue) claimNext(ctx context.Context) (claimedWrite, bool, error) {
	now := q.now().Format(time.RFC3339Nano)
	row := q.db.QueryRowContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET status = 'in_progress',
		     attempt_count = attempt_count + 1,
		     updated_at = ?
		 WHERE id = (
		   SELECT id
		   FROM persistence_write_queue
		   WHERE status = 'pending' AND available_at <= ?
		   ORDER BY created_at
		   LIMIT 1
		 )
		 RETURNING id, operation, payload_json, attempt_count, max_attempts`,
		now,
		now,
	)

	job := claimedWrite{}
	if err := row.Scan(&job.ID, &job.Operation, &job.PayloadJSON, &job.Attempts, &job.MaxAttempts); err != nil {
		if err == sql.ErrNoRows {
			return claimedWrite{}, false, nil
		}
		return claimedWrite{}, false, fmt.Errorf("claim next write: %w", err)
	}

	return job, true, nil
}

func (q *Queue) retryOrFail(ctx context.Context, job claimedWrite, errMsg string) error {
	now := q.now()
	status := "pending"
	availableAt := now
	if job.Attempts >= job.MaxAttempts {
		status = "failed"
	}
	if status == "pending" {
		availableAt = now.Add(q.retryBackoff(job.Attempts))
	}

	_, err := q.db.ExecContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET status = ?, available_at = ?, last_error = ?, updated_at = ?
		 WHERE id = ?`,
		status,
		availableAt.Format(time.RFC3339Nano),
		errMsg,
		now.Format(time.RFC3339Nano),
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("update retry state for write %s: %w", job.ID, err)
	}
	return nil
}

func (q *Queue) markFailed(ctx context.Context, id, errMsg string) error {
	now := q.now().Format(time.RFC3339Nano)
	_, err := q.db.ExecContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET status = 'failed', last_error = ?, updated_at = ?
		 WHERE id = ?`,
		errMsg,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark write failed: %w", err)
	}
	return nil
}

func (q *Queue) requeueStaleInProgress(ctx context.Context) error {
	threshold := q.now().Add(-q.inProgressTimeout).Format(time.RFC3339Nano)
	now := q.now().Format(time.RFC3339Nano)
	_, err := q.db.ExecContext(
		ctx,
		`UPDATE persistence_write_queue
		 SET status = 'pending',
		     available_at = ?,
		     last_error = CASE
		       WHEN last_error IS NULL OR last_error = '' THEN 'recovered stale in_progress write'
		       ELSE last_error
		     END,
		     updated_at = ?
		 WHERE status = 'in_progress'
		   AND updated_at <= ?
		   AND COALESCE(last_error, '') <> ?`,
		now,
		now,
		threshold,
		completionPendingMarker,
	)
	if err != nil {
		return fmt.Errorf("requeue stale writes: %w", err)
	}
	return nil
}

func newWriteID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate write id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
