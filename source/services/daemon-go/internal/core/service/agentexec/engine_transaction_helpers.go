package agentexec

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

func (e *SQLiteExecutionEngine) runTransaction(
	ctx context.Context,
	phase string,
	fn func(tx *sql.Tx) error,
) error {
	if e.db == nil {
		return fmt.Errorf("db is required")
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		if strings.TrimSpace(phase) == "" {
			return fmt.Errorf("begin tx: %w", err)
		}
		return fmt.Errorf("begin %s tx: %w", strings.TrimSpace(phase), err)
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		if strings.TrimSpace(phase) == "" {
			return fmt.Errorf("commit tx: %w", err)
		}
		return fmt.Errorf("commit %s tx: %w", strings.TrimSpace(phase), err)
	}
	return nil
}

func (e *SQLiteExecutionEngine) recordStepAndRunFailure(
	ctx context.Context,
	taskID string,
	runID string,
	stepID string,
	errMessage string,
) error {
	failedAt := e.now().Format(time.RFC3339Nano)
	return e.runTransaction(ctx, "failure transition", func(tx *sql.Tx) error {
		if strings.TrimSpace(stepID) != "" {
			if err := updateTaskStepState(ctx, tx, strings.TrimSpace(stepID), shared.TaskStepStatusFailed, errMessage, failedAt); err != nil {
				return err
			}
		}
		if err := updateTaskState(ctx, tx, strings.TrimSpace(taskID), shared.TaskStateFailed, errMessage, failedAt); err != nil {
			return err
		}
		if err := updateRunState(ctx, tx, strings.TrimSpace(runID), shared.TaskStateFailed, "", failedAt, errMessage, failedAt); err != nil {
			return err
		}
		return nil
	})
}
