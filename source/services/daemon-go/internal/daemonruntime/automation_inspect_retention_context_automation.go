package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	repocommtrigger "personalagent/runtime/internal/core/repository/commtrigger"
	reposcheduler "personalagent/runtime/internal/core/repository/scheduler"
	commtriggerservice "personalagent/runtime/internal/core/service/commtrigger"
	schedulerservice "personalagent/runtime/internal/core/service/scheduler"
	"personalagent/runtime/internal/transport"
)

type simulatedCommEventInput struct {
	WorkspaceID      string
	EventID          string
	ThreadID         string
	Channel          string
	Body             string
	Sender           string
	EventType        string
	Direction        string
	AssistantEmitted bool
	OccurredAt       time.Time
}

func (s *AutomationInspectRetentionContextService) CreateAutomation(ctx context.Context, request transport.AutomationCreateRequest) (transport.AutomationTriggerRecord, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	subject := strings.TrimSpace(request.SubjectActorID)
	if subject == "" {
		subject = "actor.automation"
	}

	triggerType := strings.ToUpper(strings.TrimSpace(request.TriggerType))
	if triggerType != "SCHEDULE" && triggerType != "ON_COMM_EVENT" {
		return transport.AutomationTriggerRecord{}, fmt.Errorf("--trigger-type must be SCHEDULE or ON_COMM_EVENT")
	}

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)

	directiveID := strings.TrimSpace(request.DirectiveID)
	if directiveID == "" {
		id, err := automationRandomID("dir")
		if err != nil {
			return transport.AutomationTriggerRecord{}, err
		}
		directiveID = id
	}

	triggerID := strings.TrimSpace(request.TriggerID)
	if triggerID == "" {
		id, err := automationRandomID("trg")
		if err != nil {
			return transport.AutomationTriggerRecord{}, err
		}
		triggerID = id
	}

	title := strings.TrimSpace(request.Title)
	if title == "" {
		if triggerType == "SCHEDULE" {
			title = "Scheduled automation"
		} else {
			title = "Communication automation"
		}
	}

	instruction := strings.TrimSpace(request.Instruction)
	if instruction == "" {
		instruction = "Execute automated task"
	}

	resolvedFilterJSON, err := resolveAutomationFilterJSON(triggerType, request.IntervalSeconds, request.Filter)
	if err != nil {
		return transport.AutomationTriggerRecord{}, err
	}

	var cooldownValue any
	if request.CooldownSeconds > 0 {
		cooldownValue = request.CooldownSeconds
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.AutomationTriggerRecord{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureDelegationWorkspace(ctx, tx, workspace, nowText); err != nil {
		return transport.AutomationTriggerRecord{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspace, subject, nowText); err != nil {
		return transport.AutomationTriggerRecord{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO directives(
			id, workspace_id, subject_principal_actor_id, title, instruction, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
	`, directiveID, workspace, subject, title, instruction, nowText, nowText); err != nil {
		return transport.AutomationTriggerRecord{}, fmt.Errorf("insert directive: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO automation_triggers(
			id, workspace_id, directive_id, trigger_type, is_enabled, filter_json, cooldown_seconds, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, triggerID, workspace, directiveID, triggerType, boolToInt(request.Enabled), resolvedFilterJSON, cooldownValue, nowText, nowText); err != nil {
		return transport.AutomationTriggerRecord{}, fmt.Errorf("insert automation trigger: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return transport.AutomationTriggerRecord{}, fmt.Errorf("commit tx: %w", err)
	}

	return transport.AutomationTriggerRecord{
		TriggerID:             triggerID,
		WorkspaceID:           workspace,
		DirectiveID:           directiveID,
		TriggerType:           triggerType,
		Enabled:               request.Enabled,
		FilterJSON:            resolvedFilterJSON,
		CooldownSeconds:       maxInt(0, request.CooldownSeconds),
		SubjectPrincipalActor: subject,
		DirectiveTitle:        title,
		DirectiveInstruction:  instruction,
		DirectiveStatus:       "ACTIVE",
		CreatedAt:             nowText,
		UpdatedAt:             nowText,
	}, nil
}

func (s *AutomationInspectRetentionContextService) ListAutomation(ctx context.Context, request transport.AutomationListRequest) (transport.AutomationListResponse, error) {
	query := `
		SELECT
			at.id,
			at.workspace_id,
			at.directive_id,
			at.trigger_type,
			at.is_enabled,
			COALESCE(at.filter_json, '{}'),
			COALESCE(at.cooldown_seconds, 0),
			d.subject_principal_actor_id,
			d.title,
			d.instruction,
			d.status,
			at.created_at,
			at.updated_at
		FROM automation_triggers at
		JOIN directives d ON d.id = at.directive_id
		WHERE at.workspace_id = ?
	`
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	params := []any{workspace}

	if strings.TrimSpace(request.TriggerType) != "" {
		query += " AND at.trigger_type = ?"
		params = append(params, strings.ToUpper(strings.TrimSpace(request.TriggerType)))
	}
	if !request.IncludeDisabled {
		query += " AND at.is_enabled = 1"
	}
	query += " ORDER BY at.created_at DESC"

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.AutomationListResponse{}, fmt.Errorf("list automation triggers: %w", err)
	}
	defer rows.Close()

	triggers := make([]transport.AutomationTriggerRecord, 0)
	for rows.Next() {
		var (
			item        transport.AutomationTriggerRecord
			enabledFlag int
		)
		if err := rows.Scan(
			&item.TriggerID,
			&item.WorkspaceID,
			&item.DirectiveID,
			&item.TriggerType,
			&enabledFlag,
			&item.FilterJSON,
			&item.CooldownSeconds,
			&item.SubjectPrincipalActor,
			&item.DirectiveTitle,
			&item.DirectiveInstruction,
			&item.DirectiveStatus,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return transport.AutomationListResponse{}, fmt.Errorf("scan automation trigger: %w", err)
		}
		item.Enabled = enabledFlag == 1
		triggers = append(triggers, item)
	}
	if err := rows.Err(); err != nil {
		return transport.AutomationListResponse{}, fmt.Errorf("iterate automation triggers: %w", err)
	}

	return transport.AutomationListResponse{
		WorkspaceID: workspace,
		Triggers:    triggers,
	}, nil
}

func (s *AutomationInspectRetentionContextService) ListAutomationFireHistory(ctx context.Context, request transport.AutomationFireHistoryRequest) (transport.AutomationFireHistoryResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	triggerID := strings.TrimSpace(request.TriggerID)
	routeResolver := newWorkflowRouteMetadataResolver(s.container)
	statusFilter, err := normalizeAutomationFireHistoryStatusFilter(request.Status)
	if err != nil {
		return transport.AutomationFireHistoryResponse{}, err
	}

	limit := request.Limit
	switch {
	case limit <= 0:
		limit = 50
	case limit > 200:
		limit = 200
	}

	query := `
		SELECT
			tf.id,
			tf.workspace_id,
			tf.trigger_id,
			at.trigger_type,
			at.directive_id,
			tf.source_event_id,
			tf.fired_at,
			COALESCE(tf.task_id, ''),
			COALESCE(tf.outcome, ''),
			COALESCE((
				SELECT tr.id
				FROM task_runs tr
				WHERE tr.task_id = tf.task_id
				ORDER BY COALESCE(tr.started_at, tr.created_at) DESC, tr.id DESC
				LIMIT 1
			), '')
		FROM trigger_fires tf
		JOIN automation_triggers at ON at.id = tf.trigger_id
		WHERE tf.workspace_id = ?
	`
	params := []any{workspace}
	if triggerID != "" {
		query += " AND tf.trigger_id = ?"
		params = append(params, triggerID)
	}
	if statusFilter != "" {
		query += " AND UPPER(COALESCE(tf.outcome, '')) = ?"
		params = append(params, automationFireOutcomeForStatusFilter(statusFilter))
	}
	query += " ORDER BY tf.fired_at DESC, tf.id DESC LIMIT ?"
	params = append(params, limit)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.AutomationFireHistoryResponse{}, fmt.Errorf("list automation fire history: %w", err)
	}
	defer rows.Close()

	fires := make([]transport.AutomationFireHistoryRecord, 0)
	type fireRouteHint struct {
		workspaceID string
		taskID      string
		runID       string
		triggerType string
	}
	routeHints := make([]fireRouteHint, 0)
	for rows.Next() {
		var (
			item          transport.AutomationFireHistoryRecord
			sourceEventID string
			outcome       string
		)
		if err := rows.Scan(
			&item.FireID,
			&item.WorkspaceID,
			&item.TriggerID,
			&item.TriggerType,
			&item.DirectiveID,
			&sourceEventID,
			&item.FiredAt,
			&item.TaskID,
			&outcome,
			&item.RunID,
		); err != nil {
			return transport.AutomationFireHistoryResponse{}, fmt.Errorf("scan automation fire history row: %w", err)
		}
		item.TaskID = strings.TrimSpace(item.TaskID)
		item.RunID = strings.TrimSpace(item.RunID)
		item.Outcome = strings.TrimSpace(outcome)
		item.Status = automationFireStatusFromOutcome(item.Outcome)
		item.IdempotencySignal = strings.TrimSpace(sourceEventID)
		item.IdempotencyKey = buildAutomationFireIdempotencyKey(item.WorkspaceID, item.TriggerID, item.IdempotencySignal)
		fires = append(fires, item)
		routeHints = append(routeHints, fireRouteHint{
			workspaceID: item.WorkspaceID,
			taskID:      item.TaskID,
			runID:       item.RunID,
			triggerType: item.TriggerType,
		})
	}
	if err := rows.Err(); err != nil {
		return transport.AutomationFireHistoryResponse{}, fmt.Errorf("iterate automation fire history: %w", err)
	}
	for idx := range fires {
		hint := routeHints[idx]
		fires[idx].Route = routeResolver.ResolveForAutomationFire(
			ctx,
			hint.workspaceID,
			hint.taskID,
			hint.runID,
			hint.triggerType,
		)
	}

	return transport.AutomationFireHistoryResponse{
		WorkspaceID: workspace,
		Fires:       fires,
	}, nil
}

func (s *AutomationInspectRetentionContextService) UpdateAutomation(ctx context.Context, request transport.AutomationUpdateRequest) (transport.AutomationUpdateResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	triggerID := strings.TrimSpace(request.TriggerID)
	if triggerID == "" {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("--trigger-id is required")
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	existing, err := loadAutomationTriggerRecord(ctx, tx, workspace, triggerID)
	if err == sql.ErrNoRows {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("automation trigger not found")
	}
	if err != nil {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("load automation trigger: %w", err)
	}

	if request.IntervalSeconds != nil {
		if *request.IntervalSeconds <= 0 {
			return transport.AutomationUpdateResponse{}, fmt.Errorf("--interval-seconds must be positive")
		}
		if existing.TriggerType != "SCHEDULE" {
			return transport.AutomationUpdateResponse{}, fmt.Errorf("--interval-seconds can only be set for SCHEDULE triggers")
		}
	}
	if request.CooldownSeconds != nil && *request.CooldownSeconds < 0 {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("--cooldown-seconds cannot be negative")
	}

	subject := existing.SubjectPrincipalActor
	if trimmed := strings.TrimSpace(request.SubjectActorID); trimmed != "" {
		subject = trimmed
	}

	title := existing.DirectiveTitle
	if trimmed := strings.TrimSpace(request.Title); trimmed != "" {
		title = trimmed
	}

	instruction := existing.DirectiveInstruction
	if trimmed := strings.TrimSpace(request.Instruction); trimmed != "" {
		instruction = trimmed
	}

	resolvedFilterJSON, err := resolveAutomationUpdatedFilterJSON(
		existing.TriggerType,
		existing.FilterJSON,
		request.IntervalSeconds,
		request.Filter,
	)
	if err != nil {
		return transport.AutomationUpdateResponse{}, err
	}

	enabled := existing.Enabled
	if request.Enabled != nil {
		enabled = *request.Enabled
	}

	cooldownSeconds := existing.CooldownSeconds
	if request.CooldownSeconds != nil {
		cooldownSeconds = *request.CooldownSeconds
	}
	if cooldownSeconds < 0 {
		cooldownSeconds = 0
	}

	if automationRecordUnchanged(existing, subject, title, instruction, enabled, resolvedFilterJSON, cooldownSeconds) {
		return transport.AutomationUpdateResponse{
			Trigger:    existing,
			Updated:    false,
			Idempotent: true,
		}, nil
	}

	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if err := ensureDelegationWorkspace(ctx, tx, workspace, nowText); err != nil {
		return transport.AutomationUpdateResponse{}, err
	}
	if err := ensureDelegationActorPrincipal(ctx, tx, workspace, subject, nowText); err != nil {
		return transport.AutomationUpdateResponse{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE directives
		SET subject_principal_actor_id = ?, title = ?, instruction = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, subject, title, instruction, nowText, existing.DirectiveID, workspace); err != nil {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("update directive: %w", err)
	}

	var cooldownValue any
	if cooldownSeconds > 0 {
		cooldownValue = cooldownSeconds
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE automation_triggers
		SET is_enabled = ?, filter_json = ?, cooldown_seconds = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, boolToInt(enabled), resolvedFilterJSON, cooldownValue, nowText, triggerID, workspace); err != nil {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("update automation trigger: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return transport.AutomationUpdateResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	updated := existing
	updated.Enabled = enabled
	updated.FilterJSON = resolvedFilterJSON
	updated.CooldownSeconds = cooldownSeconds
	updated.SubjectPrincipalActor = subject
	updated.DirectiveTitle = title
	updated.DirectiveInstruction = instruction
	updated.UpdatedAt = nowText
	return transport.AutomationUpdateResponse{
		Trigger:    updated,
		Updated:    true,
		Idempotent: false,
	}, nil
}

func (s *AutomationInspectRetentionContextService) DeleteAutomation(ctx context.Context, request transport.AutomationDeleteRequest) (transport.AutomationDeleteResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	triggerID := strings.TrimSpace(request.TriggerID)
	if triggerID == "" {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("--trigger-id is required")
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var directiveID string
	if err := tx.QueryRowContext(ctx, `
		SELECT directive_id
		FROM automation_triggers
		WHERE id = ? AND workspace_id = ?
	`, triggerID, workspace).Scan(&directiveID); err == sql.ErrNoRows {
		return transport.AutomationDeleteResponse{
			WorkspaceID: workspace,
			TriggerID:   triggerID,
			Deleted:     false,
			Idempotent:  true,
		}, nil
	} else if err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("load automation trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM trigger_fires
		WHERE workspace_id = ? AND trigger_id = ?
	`, workspace, triggerID); err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("delete trigger fires: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM automation_triggers
		WHERE id = ? AND workspace_id = ?
	`, triggerID, workspace); err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("delete automation trigger: %w", err)
	}

	var remaining int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM automation_triggers
		WHERE workspace_id = ? AND directive_id = ?
	`, workspace, directiveID).Scan(&remaining); err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("count directive trigger usage: %w", err)
	}
	if remaining == 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM directives
			WHERE id = ? AND workspace_id = ?
		`, directiveID, workspace); err != nil {
			return transport.AutomationDeleteResponse{}, fmt.Errorf("delete directive: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return transport.AutomationDeleteResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	return transport.AutomationDeleteResponse{
		WorkspaceID: workspace,
		TriggerID:   triggerID,
		DirectiveID: directiveID,
		Deleted:     true,
		Idempotent:  false,
	}, nil
}

func (s *AutomationInspectRetentionContextService) RunAutomationSchedule(ctx context.Context, request transport.AutomationRunScheduleRequest) (transport.AutomationRunScheduleResponse, error) {
	runAt := time.Now().UTC()
	if strings.TrimSpace(request.At) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.At))
		if err != nil {
			return transport.AutomationRunScheduleResponse{}, fmt.Errorf("invalid --at timestamp: %w", err)
		}
		runAt = parsed.UTC()
	}

	store := reposcheduler.NewSQLiteScheduleStore(s.container.DB)
	engine := schedulerservice.NewEngine(store, store, schedulerservice.Options{
		Now: func() time.Time { return runAt },
	})
	result, err := engine.EvaluateSchedules(ctx)
	if err != nil {
		return transport.AutomationRunScheduleResponse{}, err
	}

	return transport.AutomationRunScheduleResponse{
		At:     runAt.Format(time.RFC3339Nano),
		Result: result,
	}, nil
}

func (s *AutomationInspectRetentionContextService) RunAutomationCommEvent(ctx context.Context, request transport.AutomationRunCommEventRequest) (transport.AutomationRunCommEventResponse, error) {
	eventID := strings.TrimSpace(request.EventID)
	if eventID == "" {
		return transport.AutomationRunCommEventResponse{}, fmt.Errorf("--event-id is required")
	}

	occurredAt := time.Now().UTC()
	if strings.TrimSpace(request.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.OccurredAt))
		if err != nil {
			return transport.AutomationRunCommEventResponse{}, fmt.Errorf("invalid --occurred-at timestamp: %w", err)
		}
		occurredAt = parsed.UTC()
	}

	if request.SeedEvent {
		if err := ensureSimulatedCommEvent(ctx, s.container.DB, simulatedCommEventInput{
			WorkspaceID:      normalizeWorkspaceID(request.WorkspaceID),
			EventID:          eventID,
			ThreadID:         strings.TrimSpace(request.ThreadID),
			Channel:          strings.TrimSpace(request.Channel),
			Body:             strings.TrimSpace(request.Body),
			Sender:           strings.TrimSpace(request.Sender),
			EventType:        strings.TrimSpace(request.EventType),
			Direction:        strings.TrimSpace(request.Direction),
			AssistantEmitted: request.AssistantEmitted,
			OccurredAt:       occurredAt,
		}); err != nil {
			return transport.AutomationRunCommEventResponse{}, err
		}
	}

	store := repocommtrigger.NewSQLiteCommTriggerStore(s.container.DB)
	engine := commtriggerservice.NewEngine(store, commtriggerservice.Options{})
	result, err := engine.EvaluateEvent(ctx, eventID)
	if err != nil {
		return transport.AutomationRunCommEventResponse{}, err
	}

	return transport.AutomationRunCommEventResponse{
		EventID:     eventID,
		SeededEvent: request.SeedEvent,
		Result:      result,
	}, nil
}
