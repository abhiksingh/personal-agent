package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) InspectRun(ctx context.Context, request transport.InspectRunRequest) (transport.InspectRunResponse, error) {
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		return transport.InspectRunResponse{}, fmt.Errorf("--run-id is required")
	}

	var run transport.InspectTaskRun
	err := s.container.DB.QueryRowContext(ctx, `
		SELECT id, workspace_id, task_id, acting_as_actor_id, state,
		       COALESCE(started_at, ''), COALESCE(finished_at, ''), COALESCE(last_error, ''),
		       created_at, updated_at
		FROM task_runs
		WHERE id = ?
	`, runID).Scan(
		&run.RunID,
		&run.WorkspaceID,
		&run.TaskID,
		&run.ActingAsActorID,
		&run.State,
		&run.StartedAt,
		&run.FinishedAt,
		&run.LastError,
		&run.CreatedAt,
		&run.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return transport.InspectRunResponse{}, fmt.Errorf("run not found")
	}
	if err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("load run: %w", err)
	}

	var task transport.InspectTask
	if err := s.container.DB.QueryRowContext(ctx, `
		SELECT id, workspace_id, requested_by_actor_id, subject_principal_actor_id,
		       title, COALESCE(description, ''), state, priority,
		       COALESCE(deadline_at, ''), COALESCE(channel, ''), created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, run.TaskID).Scan(
		&task.TaskID,
		&task.WorkspaceID,
		&task.RequestedByActorID,
		&task.SubjectPrincipalActor,
		&task.Title,
		&task.Description,
		&task.State,
		&task.Priority,
		&task.DeadlineAt,
		&task.Channel,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("load task: %w", err)
	}

	steps := make([]transport.InspectStep, 0)
	stepRows, err := s.container.DB.QueryContext(ctx, `
		SELECT id, run_id, step_index, name, status,
		       COALESCE(interaction_level, ''), COALESCE(capability_key, ''),
		       COALESCE(timeout_seconds, 0), retry_max, retry_count,
		       COALESCE(last_error, ''), created_at, updated_at
		FROM task_steps
		WHERE run_id = ?
		ORDER BY step_index ASC
	`, run.RunID)
	if err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("list task steps: %w", err)
	}
	defer stepRows.Close()

	for stepRows.Next() {
		var step transport.InspectStep
		if err := stepRows.Scan(
			&step.StepID,
			&step.RunID,
			&step.StepIndex,
			&step.Name,
			&step.Status,
			&step.InteractionLevel,
			&step.CapabilityKey,
			&step.TimeoutSeconds,
			&step.RetryMax,
			&step.RetryCount,
			&step.LastError,
			&step.CreatedAt,
			&step.UpdatedAt,
		); err != nil {
			return transport.InspectRunResponse{}, fmt.Errorf("scan task step: %w", err)
		}
		steps = append(steps, step)
	}
	if err := stepRows.Err(); err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("iterate task steps: %w", err)
	}

	artifacts := make([]transport.InspectRunArtifact, 0)
	artifactRows, err := s.container.DB.QueryContext(ctx, `
		SELECT id, run_id, COALESCE(step_id, ''), artifact_type,
		       COALESCE(uri, ''), COALESCE(content_hash, ''), created_at
		FROM run_artifacts
		WHERE run_id = ?
		ORDER BY created_at ASC, id ASC
	`, run.RunID)
	if err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("list run artifacts: %w", err)
	}
	defer artifactRows.Close()

	for artifactRows.Next() {
		var artifact transport.InspectRunArtifact
		if err := artifactRows.Scan(
			&artifact.ArtifactID,
			&artifact.RunID,
			&artifact.StepID,
			&artifact.ArtifactType,
			&artifact.URI,
			&artifact.ContentHash,
			&artifact.CreatedAt,
		); err != nil {
			return transport.InspectRunResponse{}, fmt.Errorf("scan run artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := artifactRows.Err(); err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("iterate run artifacts: %w", err)
	}

	audits := make([]transport.InspectAuditEntry, 0)
	auditRows, err := s.container.DB.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(run_id, ''), COALESCE(step_id, ''), event_type,
		       COALESCE(actor_id, ''), COALESCE(acting_as_actor_id, ''), COALESCE(correlation_id, ''),
		       COALESCE(payload_json, ''), created_at
		FROM audit_log_entries
		WHERE run_id = ?
		   OR step_id IN (SELECT id FROM task_steps WHERE run_id = ?)
		ORDER BY created_at ASC, id ASC
	`, run.RunID, run.RunID)
	if err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("list audit entries: %w", err)
	}
	defer auditRows.Close()

	for auditRows.Next() {
		var item transport.InspectAuditEntry
		if err := auditRows.Scan(
			&item.AuditID,
			&item.WorkspaceID,
			&item.RunID,
			&item.StepID,
			&item.EventType,
			&item.ActorID,
			&item.ActingAsActorID,
			&item.CorrelationID,
			&item.PayloadJSON,
			&item.CreatedAt,
		); err != nil {
			return transport.InspectRunResponse{}, fmt.Errorf("scan audit entry: %w", err)
		}
		audits = append(audits, item)
	}
	if err := auditRows.Err(); err != nil {
		return transport.InspectRunResponse{}, fmt.Errorf("iterate audit entries: %w", err)
	}

	return transport.InspectRunResponse{
		Task:         task,
		Run:          run,
		Steps:        steps,
		Artifacts:    artifacts,
		AuditEntries: audits,
		Route: func() transport.WorkflowRouteMetadata {
			routeResolver := newWorkflowRouteMetadataResolver(s.container)
			stepCapability := ""
			for _, step := range steps {
				candidate := strings.TrimSpace(step.CapabilityKey)
				if candidate != "" {
					stepCapability = candidate
					break
				}
			}
			return routeResolver.ResolveForTaskRun(
				ctx,
				run.WorkspaceID,
				task.TaskID,
				run.RunID,
				stepCapability,
				task.Channel,
			)
		}(),
	}, nil
}

func (s *AutomationInspectRetentionContextService) InspectTranscript(ctx context.Context, request transport.InspectTranscriptRequest) (transport.InspectTranscriptResponse, error) {
	query := `
		SELECT
			ce.id,
			ce.workspace_id,
			ce.thread_id,
			COALESCE(ct.channel, ''),
			ce.event_type,
			ce.direction,
			ce.assistant_emitted,
			COALESCE(ce.body_text, ''),
			COALESCE((
				SELECT cea.address_value
				FROM comm_event_addresses cea
				WHERE cea.event_id = ce.id
				  AND cea.address_role = 'FROM'
				ORDER BY cea.position ASC
				LIMIT 1
			), ''),
			ce.occurred_at,
			ce.created_at
		FROM comm_events ce
		JOIN comm_threads ct ON ct.id = ce.thread_id
		WHERE ce.workspace_id = ?
	`
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	params := []any{workspace}
	if strings.TrimSpace(request.ThreadID) != "" {
		query += " AND ce.thread_id = ?"
		params = append(params, strings.TrimSpace(request.ThreadID))
	}
	query += " ORDER BY ce.occurred_at DESC, ce.id DESC LIMIT ?"
	params = append(params, maxInt(1, request.Limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.InspectTranscriptResponse{}, fmt.Errorf("list transcript events: %w", err)
	}
	defer rows.Close()

	events := make([]transport.InspectTranscriptEvent, 0)
	for rows.Next() {
		var (
			event         transport.InspectTranscriptEvent
			assistantFlag int
		)
		if err := rows.Scan(
			&event.EventID,
			&event.WorkspaceID,
			&event.ThreadID,
			&event.Channel,
			&event.EventType,
			&event.Direction,
			&assistantFlag,
			&event.BodyText,
			&event.SenderAddress,
			&event.OccurredAt,
			&event.CreatedAt,
		); err != nil {
			return transport.InspectTranscriptResponse{}, fmt.Errorf("scan transcript event: %w", err)
		}
		event.AssistantEmitted = assistantFlag == 1
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return transport.InspectTranscriptResponse{}, fmt.Errorf("iterate transcript events: %w", err)
	}

	return transport.InspectTranscriptResponse{
		WorkspaceID: workspace,
		Events:      events,
	}, nil
}

func (s *AutomationInspectRetentionContextService) InspectMemory(ctx context.Context, request transport.InspectMemoryRequest) (transport.InspectMemoryResponse, error) {
	query := `
		SELECT
			mi.id,
			mi.workspace_id,
			mi.owner_principal_actor_id,
			mi.scope_type,
			mi.key,
			mi.value_json,
			mi.status,
			COALESCE(mi.source_summary, ''),
			mi.created_at,
			mi.updated_at,
			(SELECT COUNT(*) FROM memory_sources ms WHERE ms.memory_item_id = mi.id)
		FROM memory_items mi
		WHERE mi.workspace_id = ?
	`
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	params := []any{workspace}
	if strings.TrimSpace(request.OwnerActor) != "" {
		query += " AND mi.owner_principal_actor_id = ?"
		params = append(params, strings.TrimSpace(request.OwnerActor))
	}
	if strings.TrimSpace(request.Status) != "" {
		query += " AND mi.status = ?"
		params = append(params, strings.ToUpper(strings.TrimSpace(request.Status)))
	}
	query += " ORDER BY mi.updated_at DESC, mi.id DESC LIMIT ?"
	params = append(params, maxInt(1, request.Limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.InspectMemoryResponse{}, fmt.Errorf("list memory items: %w", err)
	}
	defer rows.Close()

	items := make([]transport.InspectMemoryItem, 0)
	for rows.Next() {
		var (
			item      transport.InspectMemoryItem
			valueJSON string
		)
		if err := rows.Scan(
			&item.MemoryID,
			&item.WorkspaceID,
			&item.OwnerActorID,
			&item.ScopeType,
			&item.Key,
			&valueJSON,
			&item.Status,
			&item.SourceSummary,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.SourceCount,
		); err != nil {
			return transport.InspectMemoryResponse{}, fmt.Errorf("scan memory item: %w", err)
		}
		kind, canonical, tokens := parseMemoryValueJSON(valueJSON, item.ScopeType)
		item.Kind = kind
		item.IsCanonical = canonical
		item.TokenEstimate = tokens
		item.ValueJSON = valueJSON
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return transport.InspectMemoryResponse{}, fmt.Errorf("iterate memory items: %w", err)
	}

	return transport.InspectMemoryResponse{
		WorkspaceID: workspace,
		Items:       items,
	}, nil
}

func (s *AutomationInspectRetentionContextService) QueryInspectLogs(ctx context.Context, request transport.InspectLogQueryRequest) (transport.InspectLogQueryResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampInspectLogLimit(request.Limit)
	routeResolver := newWorkflowRouteMetadataResolver(s.container)

	beforeCreatedAt, err := normalizeInspectLogTimestamp(request.BeforeCreatedAt)
	if err != nil {
		return transport.InspectLogQueryResponse{}, fmt.Errorf("invalid --before-created-at: %w", err)
	}
	options := inspectLogsQueryOptions{
		workspaceID:     workspace,
		runID:           strings.TrimSpace(request.RunID),
		eventType:       strings.TrimSpace(request.EventType),
		beforeCreatedAt: beforeCreatedAt,
		beforeID:        strings.TrimSpace(request.BeforeID),
		limit:           limit,
		orderDescending: true,
	}

	logs, err := s.queryInspectLogs(ctx, options, routeResolver)
	if err != nil {
		return transport.InspectLogQueryResponse{}, err
	}

	response := transport.InspectLogQueryResponse{
		WorkspaceID: workspace,
		Logs:        logs,
	}
	if len(logs) > 0 {
		last := logs[len(logs)-1]
		response.NextCursorCreatedAt = last.CreatedAt
		response.NextCursorID = last.LogID
	}
	return response, nil
}

func (s *AutomationInspectRetentionContextService) StreamInspectLogs(ctx context.Context, request transport.InspectLogStreamRequest) (transport.InspectLogStreamResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := clampInspectLogLimit(request.Limit)
	timeout := clampInspectLogTimeout(request.TimeoutMS)
	pollInterval := clampInspectLogPollInterval(request.PollIntervalMS)
	routeResolver := newWorkflowRouteMetadataResolver(s.container)

	cursorCreatedAt, err := normalizeInspectLogTimestamp(request.CursorCreatedAt)
	if err != nil {
		return transport.InspectLogStreamResponse{}, fmt.Errorf("invalid --cursor-created-at: %w", err)
	}
	cursorID := strings.TrimSpace(request.CursorID)

	deadline := time.Now().UTC().Add(timeout)
	for {
		logs, queryErr := s.queryInspectLogs(ctx, inspectLogsQueryOptions{
			workspaceID:     workspace,
			runID:           strings.TrimSpace(request.RunID),
			eventType:       strings.TrimSpace(request.EventType),
			sinceCreatedAt:  cursorCreatedAt,
			sinceID:         cursorID,
			limit:           limit,
			orderDescending: true,
		}, routeResolver)
		if queryErr != nil {
			return transport.InspectLogStreamResponse{}, queryErr
		}
		if len(logs) > 0 {
			latest := logs[0]
			return transport.InspectLogStreamResponse{
				WorkspaceID:     workspace,
				Logs:            logs,
				CursorCreatedAt: latest.CreatedAt,
				CursorID:        latest.LogID,
				TimedOut:        false,
			}, nil
		}

		if timeout <= 0 || time.Now().UTC().After(deadline) {
			return transport.InspectLogStreamResponse{
				WorkspaceID:     workspace,
				Logs:            []transport.InspectLogRecord{},
				CursorCreatedAt: cursorCreatedAt,
				CursorID:        cursorID,
				TimedOut:        true,
			}, nil
		}

		select {
		case <-ctx.Done():
			return transport.InspectLogStreamResponse{}, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

type inspectLogsQueryOptions struct {
	workspaceID     string
	runID           string
	eventType       string
	beforeCreatedAt string
	beforeID        string
	sinceCreatedAt  string
	sinceID         string
	limit           int
	orderDescending bool
}

func (s *AutomationInspectRetentionContextService) queryInspectLogs(
	ctx context.Context,
	options inspectLogsQueryOptions,
	routeResolver *workflowRouteMetadataResolver,
) ([]transport.InspectLogRecord, error) {
	query := `
		SELECT al.id, al.workspace_id, COALESCE(al.run_id, ''), COALESCE(al.step_id, ''), al.event_type,
		       COALESCE(al.actor_id, ''), COALESCE(al.acting_as_actor_id, ''), COALESCE(al.correlation_id, ''),
		       COALESCE(al.payload_json, ''), al.created_at, COALESCE(ts.capability_key, '')
		FROM audit_log_entries al
		LEFT JOIN task_steps ts ON ts.id = al.step_id
		WHERE al.workspace_id = ?
	`
	params := []any{options.workspaceID}

	if strings.TrimSpace(options.runID) != "" {
		query += " AND al.run_id = ?"
		params = append(params, strings.TrimSpace(options.runID))
	}
	if strings.TrimSpace(options.eventType) != "" {
		query += " AND al.event_type = ?"
		params = append(params, strings.ToUpper(strings.TrimSpace(options.eventType)))
	}
	if strings.TrimSpace(options.beforeCreatedAt) != "" {
		beforeID := strings.TrimSpace(options.beforeID)
		if beforeID == "" {
			beforeID = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
		}
		query += " AND (al.created_at < ? OR (al.created_at = ? AND al.id < ?))"
		params = append(params, options.beforeCreatedAt, options.beforeCreatedAt, beforeID)
	}
	if strings.TrimSpace(options.sinceCreatedAt) != "" {
		sinceID := strings.TrimSpace(options.sinceID)
		if sinceID == "" {
			sinceID = ""
		}
		query += " AND (al.created_at > ? OR (al.created_at = ? AND al.id > ?))"
		params = append(params, options.sinceCreatedAt, options.sinceCreatedAt, sinceID)
	}

	order := "ASC"
	if options.orderDescending {
		order = "DESC"
	}
	query += " ORDER BY al.created_at " + order + ", al.id " + order + " LIMIT ?"
	params = append(params, clampInspectLogLimit(options.limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("query inspect logs: %w", err)
	}
	defer rows.Close()

	type inspectLogRow struct {
		logID           string
		workspaceID     string
		runID           string
		stepID          string
		eventType       string
		actorID         string
		actingAsActorID string
		correlationID   string
		payloadJSON     string
		createdAt       string
		stepCapability  string
	}
	rawRows := make([]inspectLogRow, 0)
	for rows.Next() {
		var row inspectLogRow
		if err := rows.Scan(
			&row.logID,
			&row.workspaceID,
			&row.runID,
			&row.stepID,
			&row.eventType,
			&row.actorID,
			&row.actingAsActorID,
			&row.correlationID,
			&row.payloadJSON,
			&row.createdAt,
			&row.stepCapability,
		); err != nil {
			return nil, fmt.Errorf("scan inspect log: %w", err)
		}
		rawRows = append(rawRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inspect logs: %w", err)
	}

	logs := make([]transport.InspectLogRecord, 0, len(rawRows))
	for _, row := range rawRows {
		route := transport.WorkflowRouteMetadata{
			Available:       false,
			TaskClass:       modelpolicy.TaskClassDefault,
			TaskClassSource: "default",
			RouteSource:     "none",
		}
		if routeResolver != nil {
			route = routeResolver.ResolveForTaskRun(
				ctx,
				row.workspaceID,
				"",
				row.runID,
				row.stepCapability,
				"",
			)
		}
		logs = append(logs, buildInspectLogRecord(
			row.logID,
			row.workspaceID,
			row.runID,
			row.stepID,
			row.eventType,
			row.actorID,
			row.actingAsActorID,
			row.correlationID,
			row.payloadJSON,
			row.createdAt,
			route,
		))
	}
	return logs, nil
}

func buildInspectLogRecord(
	logID string,
	workspaceID string,
	runID string,
	stepID string,
	eventType string,
	actorID string,
	actingAsActorID string,
	correlationID string,
	payloadJSON string,
	createdAt string,
	route transport.WorkflowRouteMetadata,
) transport.InspectLogRecord {
	payload, metadata, parseErr := parseInspectLogPayload(payloadJSON)
	if parseErr {
		metadata["payload_parse_error"] = "invalid json"
	}

	status := deriveInspectLogStatus(eventType, payload)
	inputSummary := deriveInspectLogInputSummary(payload)
	outputSummary := deriveInspectLogOutputSummary(payload)

	return transport.InspectLogRecord{
		LogID:           logID,
		WorkspaceID:     workspaceID,
		RunID:           runID,
		StepID:          stepID,
		EventType:       eventType,
		Status:          status,
		InputSummary:    inputSummary,
		OutputSummary:   outputSummary,
		CorrelationID:   correlationID,
		ActorID:         actorID,
		ActingAsActorID: actingAsActorID,
		CreatedAt:       createdAt,
		Metadata:        metadata,
		Route:           route,
	}
}

type inspectLogKnownPayload struct {
	Status            *json.RawMessage `json:"status"`
	Summary           *json.RawMessage `json:"summary"`
	Error             *json.RawMessage `json:"error"`
	PluginID          *json.RawMessage `json:"plugin_id"`
	Kind              *json.RawMessage `json:"kind"`
	State             *json.RawMessage `json:"state"`
	ProcessID         *json.RawMessage `json:"process_id"`
	RestartCount      *json.RawMessage `json:"restart_count"`
	ApprovalRequestID *json.RawMessage `json:"approval_request_id"`
	MessageSID        *json.RawMessage `json:"message_sid"`
	CallSID           *json.RawMessage `json:"call_sid"`
	ProviderEventID   *json.RawMessage `json:"provider_event_id"`
	Accepted          *json.RawMessage `json:"accepted"`
	Replayed          *json.RawMessage `json:"replayed"`
}

func parseInspectLogPayload(payloadJSON string) (inspectLogKnownPayload, map[string]any, bool) {
	payload := inspectLogKnownPayload{}
	metadata := map[string]any{}
	if strings.TrimSpace(payloadJSON) == "" {
		return payload, metadata, false
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return inspectLogKnownPayload{}, metadata, true
	}
	for _, field := range []struct {
		Key string
		Raw *json.RawMessage
	}{
		{Key: "plugin_id", Raw: payload.PluginID},
		{Key: "kind", Raw: payload.Kind},
		{Key: "state", Raw: payload.State},
		{Key: "process_id", Raw: payload.ProcessID},
		{Key: "restart_count", Raw: payload.RestartCount},
		{Key: "approval_request_id", Raw: payload.ApprovalRequestID},
		{Key: "message_sid", Raw: payload.MessageSID},
		{Key: "call_sid", Raw: payload.CallSID},
		{Key: "provider_event_id", Raw: payload.ProviderEventID},
		{Key: "accepted", Raw: payload.Accepted},
		{Key: "replayed", Raw: payload.Replayed},
	} {
		if field.Raw == nil {
			continue
		}
		var value any
		if err := json.Unmarshal(*field.Raw, &value); err != nil {
			continue
		}
		metadata[field.Key] = value
	}
	return payload, metadata, false
}

func deriveInspectLogStatus(eventType string, payload inspectLogKnownPayload) string {
	if status := strings.ToLower(strings.TrimSpace(inspectLogRawValueAsString(payload.Status))); status != "" {
		return status
	}
	switch strings.ToUpper(strings.TrimSpace(eventType)) {
	case "APPROVAL_REQUESTED":
		return "awaiting_approval"
	case "APPROVAL_GRANTED":
		return "approved"
	case "PLUGIN_HEALTH_TIMEOUT", "PLUGIN_WORKER_RESTART_LIMIT_REACHED":
		return "failed"
	case "PLUGIN_WORKER_STARTED", "PLUGIN_HANDSHAKE_ACCEPTED":
		return "running"
	default:
		return "info"
	}
}

func deriveInspectLogInputSummary(payload inspectLogKnownPayload) string {
	for _, field := range []struct {
		Key string
		Raw *json.RawMessage
	}{
		{Key: "approval_request_id", Raw: payload.ApprovalRequestID},
		{Key: "plugin_id", Raw: payload.PluginID},
		{Key: "message_sid", Raw: payload.MessageSID},
		{Key: "call_sid", Raw: payload.CallSID},
		{Key: "provider_event_id", Raw: payload.ProviderEventID},
	} {
		if value := strings.TrimSpace(inspectLogRawValueAsString(field.Raw)); value != "" {
			return field.Key + "=" + value
		}
	}
	return ""
}

func deriveInspectLogOutputSummary(payload inspectLogKnownPayload) string {
	if summary := strings.TrimSpace(inspectLogRawValueAsString(payload.Summary)); summary != "" {
		return summary
	}
	if errText := strings.TrimSpace(inspectLogRawValueAsString(payload.Error)); errText != "" {
		return errText
	}
	if status := strings.TrimSpace(inspectLogRawValueAsString(payload.Status)); status != "" {
		return "status=" + status
	}
	if accepted := strings.TrimSpace(inspectLogRawValueAsString(payload.Accepted)); accepted != "" {
		replayed := strings.TrimSpace(inspectLogRawValueAsString(payload.Replayed))
		if replayed != "" {
			return "accepted=" + accepted + " replayed=" + replayed
		}
		return "accepted=" + accepted
	}
	return ""
}

func inspectLogRawValueAsString(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var value any
	if err := json.Unmarshal(*raw, &value); err != nil {
		return ""
	}
	return valueAsString(value)
}

func valueAsString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func normalizeInspectLogTimestamp(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(time.RFC3339Nano), nil
}

func clampInspectLogLimit(limit int) int {
	resolved := maxInt(1, limit)
	if resolved > 200 {
		return 200
	}
	if limit <= 0 {
		return 50
	}
	return resolved
}

func clampInspectLogTimeout(timeoutMS int64) time.Duration {
	if timeoutMS <= 0 {
		return 4 * time.Second
	}
	if timeoutMS > 30000 {
		timeoutMS = 30000
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

func clampInspectLogPollInterval(intervalMS int64) time.Duration {
	if intervalMS <= 0 {
		return 250 * time.Millisecond
	}
	if intervalMS < 100 {
		intervalMS = 100
	}
	if intervalMS > 5000 {
		intervalMS = 5000
	}
	return time.Duration(intervalMS) * time.Millisecond
}
