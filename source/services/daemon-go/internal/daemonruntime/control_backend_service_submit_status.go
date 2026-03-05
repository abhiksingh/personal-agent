package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	repoauthz "personalagent/runtime/internal/core/repository/authz"
	authzservice "personalagent/runtime/internal/core/service/authz"
	coretypes "personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (b *PersistedControlBackend) SubmitTask(ctx context.Context, request transport.SubmitTaskRequest, correlationID string) (transport.SubmitTaskResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	requestedBy := strings.TrimSpace(request.RequestedByActorID)
	subject := strings.TrimSpace(request.SubjectPrincipalActorID)
	title := strings.TrimSpace(request.Title)
	description := strings.TrimSpace(request.Description)
	if requestedBy == "" {
		return transport.SubmitTaskResponse{}, errors.New("requested_by_actor_id is required")
	}
	if subject == "" {
		return transport.SubmitTaskResponse{}, errors.New("subject_principal_actor_id is required")
	}
	if title == "" {
		return transport.SubmitTaskResponse{}, errors.New("title is required")
	}
	authorizer := authzservice.NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(b.container.DB))
	decision, err := authorizer.CanActAs(ctx, coretypes.ActingAsRequest{
		WorkspaceID:        workspaceID,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    subject,
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		return transport.SubmitTaskResponse{}, err
	}
	if !decision.Allowed {
		return transport.SubmitTaskResponse{}, fmt.Errorf("acting_as denied: %s", decision.Reason)
	}

	taskID, err := controlBackendRandomID()
	if err != nil {
		return transport.SubmitTaskResponse{}, err
	}
	runID, err := controlBackendRandomID()
	if err != nil {
		return transport.SubmitTaskResponse{}, err
	}
	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)

	tx, err := b.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.SubmitTaskResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(ctx, tx, workspaceID, nowText); err != nil {
		return transport.SubmitTaskResponse{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspaceID, requestedBy, nowText); err != nil {
		return transport.SubmitTaskResponse{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspaceID, subject, nowText); err != nil {
		return transport.SubmitTaskResponse{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tasks(
			id, workspace_id, requested_by_actor_id, subject_principal_actor_id,
			title, description, state, priority, deadline_at, channel, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'queued', 0, NULL, 'app_chat', ?, ?)
	`, taskID, workspaceID, requestedBy, subject, title, controlBackendNullableText(description), nowText, nowText); err != nil {
		return transport.SubmitTaskResponse{}, fmt.Errorf("insert task: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_runs(
			id, workspace_id, task_id, acting_as_actor_id,
			state, started_at, finished_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'queued', NULL, NULL, NULL, ?, ?)
	`, runID, workspaceID, taskID, subject, nowText, nowText); err != nil {
		return transport.SubmitTaskResponse{}, fmt.Errorf("insert task run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return transport.SubmitTaskResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	response := transport.SubmitTaskResponse{
		TaskID:        taskID,
		RunID:         runID,
		State:         "queued",
		CorrelationID: correlationID,
	}

	if b.eventBroker != nil {
		_ = b.eventBroker.Publish(transport.RealtimeEventEnvelope{
			EventID:       controlBackendMustRandomID(),
			EventType:     "task_submitted",
			OccurredAt:    now,
			CorrelationID: correlationID,
			Payload: transport.RealtimeEventPayload{
				TaskID: taskID,
				RunID:  runID,
				State:  "queued",
			},
		})
		publishTaskRunLifecycleEvent(
			b.eventBroker,
			correlationID,
			workspaceID,
			taskID,
			runID,
			"queued",
			"queued",
			taskRunLifecycleSourceControlSubmit,
			"",
			now,
		)
	}

	return response, nil
}

func (b *PersistedControlBackend) TaskStatus(ctx context.Context, taskID string, correlationID string) (transport.TaskStatusResponse, error) {
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return transport.TaskStatusResponse{}, transport.NewTaskControlMissingReferenceError("task id is required")
	}

	var (
		state      string
		updatedRaw string
		runID      string
		runState   string
		lastError  string
	)
	err := b.container.DB.QueryRowContext(
		ctx,
		`
		SELECT
			COALESCE(t.state, ''),
			t.updated_at,
			COALESCE(tr.id, ''),
			COALESCE(tr.state, ''),
			COALESCE(tr.last_error, '')
		FROM tasks t
		LEFT JOIN task_runs tr ON tr.id = (
			SELECT tr2.id
			FROM task_runs tr2
			WHERE tr2.task_id = t.id
			ORDER BY tr2.created_at DESC, tr2.id DESC
			LIMIT 1
		)
		WHERE t.id = ?
		`,
		trimmedTaskID,
	).Scan(&state, &updatedRaw, &runID, &runState, &lastError)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return transport.TaskStatusResponse{}, transport.NewTaskControlNotFoundError(fmt.Sprintf("task not found: %s", trimmedTaskID))
		}
		return transport.TaskStatusResponse{}, fmt.Errorf("query task status: %w", err)
	}

	updatedAt, parseErr := time.Parse(time.RFC3339Nano, updatedRaw)
	if parseErr != nil {
		return transport.TaskStatusResponse{}, fmt.Errorf("parse task updated_at: %w", parseErr)
	}
	return transport.TaskStatusResponse{
		TaskID:        trimmedTaskID,
		State:         strings.TrimSpace(state),
		RunID:         strings.TrimSpace(runID),
		RunState:      normalizeTaskLifecycleState(runState),
		LastError:     strings.TrimSpace(lastError),
		Actions:       transport.ResolveTaskRunActionAvailability(state, runState),
		UpdatedAt:     updatedAt.UTC(),
		CorrelationID: correlationID,
	}, nil
}
