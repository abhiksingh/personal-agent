package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// applyTaskRunCancellationTransitionsTx applies canonical cancellation transitions
// for task run, task, pending/running steps, and pending approvals in one tx.
func applyTaskRunCancellationTransitionsTx(
	ctx context.Context,
	tx *sql.Tx,
	taskID string,
	runID string,
	reason string,
	nowText string,
) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("cancel task run rows: nil tx")
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	trimmedRunID := strings.TrimSpace(runID)
	trimmedNowText := strings.TrimSpace(nowText)
	nullableReason := taskLifecycleNullableText(reason)

	runUpdateResult, err := tx.ExecContext(ctx, `
		UPDATE task_runs
		SET state = 'cancelled',
		    finished_at = COALESCE(finished_at, ?),
		    last_error = ?,
		    updated_at = ?
		WHERE id = ?
		  AND LOWER(COALESCE(state, '')) NOT IN ('completed', 'failed', 'cancelled')
	`, trimmedNowText, nullableReason, trimmedNowText, trimmedRunID)
	if err != nil {
		return false, fmt.Errorf("mark task run cancelled: %w", err)
	}
	affectedRuns, _ := runUpdateResult.RowsAffected()

	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = 'cancelled',
		    updated_at = ?
		WHERE id = ?
		  AND LOWER(COALESCE(state, '')) NOT IN ('completed', 'failed', 'cancelled')
	`, trimmedNowText, trimmedTaskID); err != nil {
		return false, fmt.Errorf("mark task cancelled: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE task_steps
		SET status = 'skipped',
		    last_error = COALESCE(last_error, ?),
		    updated_at = ?
		WHERE run_id = ?
		  AND LOWER(COALESCE(status, '')) IN ('pending', 'running')
	`, nullableReason, trimmedNowText, trimmedRunID); err != nil {
		return false, fmt.Errorf("mark task steps skipped for cancelled run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE approval_requests
		SET decision = 'CANCELLED',
		    decided_at = COALESCE(decided_at, ?)
		WHERE decision IS NULL
		  AND (
			run_id = ?
			OR step_id IN (
				SELECT id
				FROM task_steps
				WHERE run_id = ?
			)
		  )
	`, trimmedNowText, trimmedRunID, trimmedRunID); err != nil {
		return false, fmt.Errorf("close pending approvals for cancelled run: %w", err)
	}

	return affectedRuns > 0, nil
}

func taskLifecycleNullableText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
