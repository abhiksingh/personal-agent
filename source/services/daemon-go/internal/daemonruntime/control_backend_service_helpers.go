package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	repoauthz "personalagent/runtime/internal/core/repository/authz"
	authzservice "personalagent/runtime/internal/core/service/authz"
	coretypes "personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type persistedTaskRunRecord struct {
	WorkspaceID             string
	TaskID                  string
	RunID                   string
	TaskState               string
	RunState                string
	RequestedByActorID      string
	SubjectPrincipalActorID string
	ActingAsActorID         string
	LastError               string
}

func (b *PersistedControlBackend) lookupTaskRunForCancel(ctx context.Context, request transport.TaskCancelRequest) (persistedTaskRunRecord, error) {
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if taskID == "" && runID == "" {
		return persistedTaskRunRecord{}, transport.NewTaskControlMissingReferenceError("task_id or run_id is required")
	}

	var (
		query string
		args  []any
	)
	switch {
	case runID != "":
		query = `
			SELECT
				tr.workspace_id,
				tr.task_id,
				tr.id,
				COALESCE(t.state, ''),
				COALESCE(tr.state, ''),
				COALESCE(t.requested_by_actor_id, ''),
				COALESCE(t.subject_principal_actor_id, ''),
				COALESCE(tr.acting_as_actor_id, ''),
				COALESCE(tr.last_error, '')
			FROM task_runs tr
			JOIN tasks t ON t.id = tr.task_id
			WHERE tr.id = ?
		`
		args = append(args, runID)
	case taskID != "":
		query = `
			SELECT
				tr.workspace_id,
				tr.task_id,
				tr.id,
				COALESCE(t.state, ''),
				COALESCE(tr.state, ''),
				COALESCE(t.requested_by_actor_id, ''),
				COALESCE(t.subject_principal_actor_id, ''),
				COALESCE(tr.acting_as_actor_id, ''),
				COALESCE(tr.last_error, '')
			FROM task_runs tr
			JOIN tasks t ON t.id = tr.task_id
			WHERE tr.task_id = ?
			ORDER BY tr.created_at DESC, tr.id DESC
			LIMIT 1
		`
		args = append(args, taskID)
	}

	var record persistedTaskRunRecord
	err := b.container.DB.QueryRowContext(ctx, query, args...).Scan(
		&record.WorkspaceID,
		&record.TaskID,
		&record.RunID,
		&record.TaskState,
		&record.RunState,
		&record.RequestedByActorID,
		&record.SubjectPrincipalActorID,
		&record.ActingAsActorID,
		&record.LastError,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistedTaskRunRecord{}, transport.NewTaskControlNotFoundError("task run not found")
		}
		return persistedTaskRunRecord{}, fmt.Errorf("lookup task run: %w", err)
	}
	if workspaceID != "" && !strings.EqualFold(strings.TrimSpace(record.WorkspaceID), workspaceID) {
		return persistedTaskRunRecord{}, transport.NewTaskControlReferenceMismatchError("workspace mismatch for task/run")
	}
	if taskID != "" && !strings.EqualFold(strings.TrimSpace(record.TaskID), taskID) {
		return persistedTaskRunRecord{}, transport.NewTaskControlReferenceMismatchError("task and run id mismatch")
	}

	record.TaskState = normalizeTaskLifecycleState(record.TaskState)
	record.RunState = normalizeTaskLifecycleState(record.RunState)
	record.WorkspaceID = strings.TrimSpace(record.WorkspaceID)
	record.TaskID = strings.TrimSpace(record.TaskID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.RequestedByActorID = strings.TrimSpace(record.RequestedByActorID)
	record.SubjectPrincipalActorID = strings.TrimSpace(record.SubjectPrincipalActorID)
	record.ActingAsActorID = strings.TrimSpace(record.ActingAsActorID)
	record.LastError = strings.TrimSpace(record.LastError)
	return record, nil
}

func (b *PersistedControlBackend) cancelTaskRunRows(ctx context.Context, taskID string, runID string, reason string) (bool, error) {
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := b.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin cancel tx: %w", err)
	}
	defer tx.Rollback()

	affectedRuns, err := b.cancelTaskRunRowsTx(ctx, tx, taskID, runID, reason, nowText)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit cancel tx: %w", err)
	}
	return affectedRuns, nil
}

func (b *PersistedControlBackend) cancelTaskRunRowsTx(ctx context.Context, tx *sql.Tx, taskID string, runID string, reason string, nowText string) (bool, error) {
	affectedRuns, err := applyTaskRunCancellationTransitionsTx(ctx, tx, taskID, runID, reason, nowText)
	if err != nil {
		return false, err
	}
	return affectedRuns, nil
}

func (b *PersistedControlBackend) waitForTaskRunSettlement(ctx context.Context, runID string, timeout time.Duration) (persistedTaskRunRecord, error) {
	if timeout <= 0 {
		timeout = defaultControlCancelSettlementTimeout
	}
	pollInterval := b.cancelSettlementPollInterval
	if pollInterval <= 0 {
		pollInterval = defaultControlCancelSettlementPollInterval
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	done := workerWaitContextDone(ctx)
	request := transport.TaskCancelRequest{RunID: strings.TrimSpace(runID)}
	for {
		select {
		case <-done:
			return persistedTaskRunRecord{}, ctx.Err()
		default:
		}

		record, err := b.lookupTaskRunForCancel(ctx, request)
		if err != nil {
			return persistedTaskRunRecord{}, err
		}
		if isTerminalTaskLifecycleState(record.RunState) {
			return record, nil
		}

		select {
		case <-done:
			return persistedTaskRunRecord{}, ctx.Err()
		case <-timer.C:
			return persistedTaskRunRecord{}, fmt.Errorf("timed out waiting for task run cancellation settlement")
		case <-ticker.C:
		}
	}
}

func (b *PersistedControlBackend) ensureTaskRunDelegationAllowed(ctx context.Context, record persistedTaskRunRecord) error {
	if b == nil || b.container == nil || b.container.DB == nil {
		return transport.NewTaskControlBackendError("database is not configured")
	}
	workspaceID := normalizeWorkspaceID(record.WorkspaceID)
	requestedBy := strings.TrimSpace(record.RequestedByActorID)
	actingAs := strings.TrimSpace(record.ActingAsActorID)
	if workspaceID == "" || requestedBy == "" || actingAs == "" {
		return transport.NewTaskControlBackendError("task/run principal context is incomplete")
	}

	authorizer := authzservice.NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(b.container.DB))
	decision, err := authorizer.CanActAs(ctx, coretypes.ActingAsRequest{
		WorkspaceID:        workspaceID,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    actingAs,
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return transport.NewTaskControlForbiddenError(fmt.Sprintf("acting_as denied: %s", decision.Reason))
	}
	return nil
}

func (b *PersistedControlBackend) appendControlAuditEntryTx(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	runID string,
	eventType string,
	actorID string,
	actingAsActorID string,
	correlationID string,
	payload map[string]any,
	createdAt string,
) error {
	if tx == nil {
		return fmt.Errorf("append control audit entry: nil tx")
	}
	trimmedEventType := strings.TrimSpace(eventType)
	if trimmedEventType == "" {
		return fmt.Errorf("append control audit entry: event type is required")
	}
	createdAtText := strings.TrimSpace(createdAt)
	if createdAtText == "" {
		createdAtText = time.Now().UTC().Format(time.RFC3339Nano)
	}
	auditID, err := controlBackendRandomID()
	if err != nil {
		return err
	}

	var payloadJSON any = nil
	if len(payload) > 0 {
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("marshal control audit payload: %w", marshalErr)
		}
		payloadJSON = string(encoded)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id, correlation_id, payload_json, created_at
		) VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?)
	`,
		auditID,
		strings.TrimSpace(workspaceID),
		controlBackendNullableText(strings.TrimSpace(runID)),
		trimmedEventType,
		controlBackendNullableText(strings.TrimSpace(actorID)),
		controlBackendNullableText(strings.TrimSpace(actingAsActorID)),
		controlBackendNullableText(strings.TrimSpace(correlationID)),
		payloadJSON,
		createdAtText,
	); err != nil {
		return fmt.Errorf("insert control audit entry: %w", err)
	}
	return nil
}

func controlBackendRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func controlBackendMustRandomID() string {
	id, err := controlBackendRandomID()
	if err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
	}
	return id
}

func controlBackendNullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func normalizeTaskLifecycleState(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isTerminalTaskLifecycleState(state string) bool {
	switch normalizeTaskLifecycleState(state) {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}
