package agentexec

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/workspaceid"
)

func insertTask(ctx context.Context, tx *sql.Tx, taskID string, workspaceID string, requestedBy string, subject string, nowText string, intent Intent) error {
	title := fmt.Sprintf("CLI intent: %s", intent.Workflow)
	description := intent.RawRequest
	_, err := tx.ExecContext(ctx, `
		INSERT INTO tasks(
			id, workspace_id, requested_by_actor_id, subject_principal_actor_id,
			title, description, state, priority, deadline_at, channel,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 0, NULL, 'cli', ?, ?)
	`, taskID, workspaceID, requestedBy, subject, title, description, string(shared.TaskStateQueued), nowText, nowText)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func insertTaskRun(ctx context.Context, tx *sql.Tx, runID string, taskID string, workspaceID string, actingAs string, nowText string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO task_runs(
			id, workspace_id, task_id, acting_as_actor_id,
			state, started_at, finished_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, ?, ?)
	`, runID, workspaceID, taskID, actingAs, string(shared.TaskStateQueued), nowText, nowText)
	if err != nil {
		return fmt.Errorf("insert task run: %w", err)
	}
	return nil
}

func insertTaskStep(ctx context.Context, tx *sql.Tx, stepID string, runID string, step plannedStep, status shared.TaskStepStatus, nowText string) error {
	inputJSON := ""
	if len(step.Input) > 0 {
		payload, err := json.Marshal(step.Input)
		if err != nil {
			return fmt.Errorf("marshal task step input: %w", err)
		}
		inputJSON = string(payload)
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO task_steps(
			id, run_id, step_index, name, status,
			interaction_level, capability_key, timeout_seconds,
			retry_max, retry_count, input_json, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, NULL, ?, NULL, 0, 0, ?, NULL, ?, ?)
	`, stepID, runID, step.StepIndex, step.Name, string(status), step.CapabilityKey, nullableText(inputJSON), nowText, nowText)
	if err != nil {
		return fmt.Errorf("insert task step: %w", err)
	}
	return nil
}

func updateTaskState(ctx context.Context, tx *sql.Tx, taskID string, state shared.TaskState, lastError string, updatedAt string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET state = ?, updated_at = ?
		WHERE id = ?
	`, string(state), updatedAt, taskID)
	if err != nil {
		return fmt.Errorf("update task state: %w", err)
	}
	return nil
}

func updateRunState(ctx context.Context, tx *sql.Tx, runID string, state shared.TaskState, startedAt string, finishedAt string, lastError string, updatedAt string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE task_runs
		SET state = ?,
		    started_at = COALESCE(?, started_at),
		    finished_at = COALESCE(?, finished_at),
		    last_error = ?,
		    updated_at = ?
		WHERE id = ?
	`, string(state), nullableText(startedAt), nullableText(finishedAt), nullableText(lastError), updatedAt, runID)
	if err != nil {
		return fmt.Errorf("update run state: %w", err)
	}
	return nil
}

func updateTaskStepState(ctx context.Context, tx *sql.Tx, stepID string, status shared.TaskStepStatus, lastError string, updatedAt string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE task_steps
		SET status = ?, last_error = ?, updated_at = ?
		WHERE id = ?
	`, string(status), nullableText(lastError), updatedAt, stepID)
	if err != nil {
		return fmt.Errorf("update task step state: %w", err)
	}
	return nil
}

func insertStepAuditEvent(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	runID string,
	stepID string,
	actingAs string,
	correlationID string,
	executionResult connectorcontract.StepExecutionResult,
	nowText string,
) error {
	auditID, err := randomID()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"status":   executionResult.Status,
		"summary":  executionResult.Summary,
		"evidence": executionResult.Evidence,
		"output":   executionResult.Output,
	})
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id,
			acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, ?, ?, 'STEP_EXECUTED', NULL, ?, ?, ?, ?)
	`, auditID, workspaceID, runID, stepID, actingAs, nullableText(correlationID), string(payload), nowText)
	if err != nil {
		return fmt.Errorf("insert audit log entry: %w", err)
	}
	return nil
}

func createApprovalRequest(
	ctx context.Context,
	tx *sql.Tx,
	approvalRequestID string,
	workspaceID string,
	stepID string,
	nowText string,
	rationale approvalDecisionRationale,
) error {
	rationaleJSON, err := json.Marshal(rationale)
	if err != nil {
		return fmt.Errorf("marshal approval rationale: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO approval_requests(
			id, workspace_id, run_id, step_id, requested_phrase, decision,
			decision_by_actor_id, requested_at, decided_at, rationale
		) VALUES (?, ?, NULL, ?, ?, NULL, NULL, ?, NULL, ?)
	`, approvalRequestID, workspaceID, stepID, types.DestructiveApprovalPhrase, nowText, string(rationaleJSON))
	if err != nil {
		return fmt.Errorf("insert approval request: %w", err)
	}
	return nil
}

func insertApprovalRequestedAuditEvent(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	runID string,
	stepID string,
	actingAs string,
	correlationID string,
	approvalRequestID string,
	rationale approvalDecisionRationale,
	nowText string,
) error {
	auditID, err := randomID()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"approval_request_id": approvalRequestID,
		"requested_phrase":    types.DestructiveApprovalPhrase,
		"capability_key":      strings.TrimSpace(rationale.CapabilityKey),
		"rationale":           rationale,
	})
	if err != nil {
		return fmt.Errorf("marshal approval-request payload: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id,
			acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, ?, ?, 'APPROVAL_REQUESTED', NULL, ?, ?, ?, ?)
	`, auditID, workspaceID, runID, stepID, actingAs, nullableText(correlationID), string(payload), nowText)
	if err != nil {
		return fmt.Errorf("insert approval requested audit event: %w", err)
	}
	return nil
}

func ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, nowText string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func ensureActorPrincipal(ctx context.Context, tx *sql.Tx, workspaceID string, actorID string, nowText string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at)
		VALUES (?, ?, 'human', ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, actorID, workspaceID, actorID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure actor %s: %w", actorID, err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(workspace_id, actor_id) DO UPDATE SET updated_at = excluded.updated_at
	`, "wp-"+workspaceID+"-"+actorID, workspaceID, actorID, nowText, nowText)
	if err != nil {
		return fmt.Errorf("ensure workspace principal %s: %w", actorID, err)
	}
	return nil
}

func normalizeWorkspaceID(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func normalizeActorID(actorID string, fallback string) string {
	trimmed := strings.TrimSpace(actorID)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
