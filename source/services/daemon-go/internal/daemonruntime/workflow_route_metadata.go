package daemonruntime

import (
	"context"
	"database/sql"
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/transport"
)

type workflowTaskClassHint struct {
	TaskClass string
	Source    string
}

type workflowRouteCacheEntry struct {
	Available   bool
	TaskClass   string
	Provider    string
	ModelKey    string
	RouteSource string
	Notes       string
}

type workflowRouteMetadataResolver struct {
	container     *ServiceContainer
	routeCache    map[string]workflowRouteCacheEntry
	runHintCache  map[string]workflowTaskClassHint
	taskHintCache map[string]workflowTaskClassHint
}

func newWorkflowRouteMetadataResolver(container *ServiceContainer) *workflowRouteMetadataResolver {
	return &workflowRouteMetadataResolver{
		container:     container,
		routeCache:    map[string]workflowRouteCacheEntry{},
		runHintCache:  map[string]workflowTaskClassHint{},
		taskHintCache: map[string]workflowTaskClassHint{},
	}
}

func (r *workflowRouteMetadataResolver) ResolveForTaskRun(
	ctx context.Context,
	workspaceID string,
	taskID string,
	runID string,
	stepCapability string,
	taskChannel string,
) transport.WorkflowRouteMetadata {
	hint := taskClassHintFromCapability(stepCapability, "step_capability")
	if hint.TaskClass == "" && strings.TrimSpace(runID) != "" {
		hint = r.taskClassHintForRun(ctx, runID)
	}
	if hint.TaskClass == "" && strings.TrimSpace(taskID) != "" {
		hint = r.taskClassHintForTask(ctx, taskID)
	}
	if hint.TaskClass == "" {
		hint = taskClassHintFromTaskChannel(taskChannel)
	}
	if hint.TaskClass == "" {
		hint = defaultTaskClassHint("default")
	}
	return r.resolveForTaskClass(ctx, workspaceID, hint)
}

func (r *workflowRouteMetadataResolver) ResolveForAutomationFire(
	ctx context.Context,
	workspaceID string,
	taskID string,
	runID string,
	triggerType string,
) transport.WorkflowRouteMetadata {
	if strings.TrimSpace(runID) != "" || strings.TrimSpace(taskID) != "" {
		return r.ResolveForTaskRun(ctx, workspaceID, taskID, runID, "", "")
	}
	hint := taskClassHintFromTriggerType(triggerType)
	if hint.TaskClass == "" {
		hint = defaultTaskClassHint("default")
	}
	return r.resolveForTaskClass(ctx, workspaceID, hint)
}

func (r *workflowRouteMetadataResolver) resolveForTaskClass(
	ctx context.Context,
	workspaceID string,
	hint workflowTaskClassHint,
) transport.WorkflowRouteMetadata {
	workspace := normalizeWorkspaceID(workspaceID)
	taskClass := normalizeTaskClass(hint.TaskClass)
	taskClassSource := normalizeTaskClassSource(hint.Source)
	cacheKey := workspace + "|" + taskClass

	entry, ok := r.routeCache[cacheKey]
	if !ok {
		entry = workflowRouteCacheEntry{
			Available:   false,
			TaskClass:   taskClass,
			RouteSource: "none",
		}

		if r.container == nil || r.container.ModelPolicyStore == nil || r.container.ProviderConfigStore == nil {
			entry.RouteSource = "service_unavailable"
		} else {
			resolved, err := NewProviderModelChatService(r.container).resolveModelRoute(ctx, workspace, taskClass)
			if err != nil {
				entry.RouteSource = "resolve_error"
				entry.Notes = strings.TrimSpace(err.Error())
			} else {
				entry.Available = true
				entry.Provider = strings.TrimSpace(resolved.Provider)
				entry.ModelKey = strings.TrimSpace(resolved.ModelKey)
				entry.RouteSource = normalizeRouteSource(resolved.Source)
				entry.Notes = strings.TrimSpace(resolved.Notes)
			}
		}
		r.routeCache[cacheKey] = entry
	}

	return transport.WorkflowRouteMetadata{
		Available:       entry.Available,
		TaskClass:       entry.TaskClass,
		Provider:        entry.Provider,
		ModelKey:        entry.ModelKey,
		TaskClassSource: taskClassSource,
		RouteSource:     entry.RouteSource,
		Notes:           entry.Notes,
	}
}

func (r *workflowRouteMetadataResolver) taskClassHintForRun(ctx context.Context, runID string) workflowTaskClassHint {
	run := strings.TrimSpace(runID)
	if run == "" {
		return workflowTaskClassHint{}
	}
	if cached, ok := r.runHintCache[run]; ok {
		return cached
	}

	if r.container == nil || r.container.DB == nil {
		hint := defaultTaskClassHint("run_lookup_unavailable")
		r.runHintCache[run] = hint
		return hint
	}

	var capabilityKey string
	var channel string
	err := r.container.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE((
				SELECT ts.capability_key
				FROM task_steps ts
				WHERE ts.run_id = tr.id
				  AND TRIM(COALESCE(ts.capability_key, '')) <> ''
				ORDER BY ts.step_index ASC
				LIMIT 1
			), ''),
			COALESCE(t.channel, '')
		FROM task_runs tr
		LEFT JOIN tasks t ON t.id = tr.task_id
		WHERE tr.id = ?
	`, run).Scan(&capabilityKey, &channel)
	if err == sql.ErrNoRows {
		hint := defaultTaskClassHint("run_not_found")
		r.runHintCache[run] = hint
		return hint
	}
	if err != nil {
		hint := defaultTaskClassHint("run_lookup_error")
		r.runHintCache[run] = hint
		return hint
	}

	hint := taskClassHintFromCapability(capabilityKey, "run_step_capability")
	if hint.TaskClass == "" {
		hint = taskClassHintFromTaskChannelWithSource(channel, "run_task_channel")
	}
	if hint.TaskClass == "" {
		hint = defaultTaskClassHint("run_default")
	}
	r.runHintCache[run] = hint
	return hint
}

func (r *workflowRouteMetadataResolver) taskClassHintForTask(ctx context.Context, taskID string) workflowTaskClassHint {
	task := strings.TrimSpace(taskID)
	if task == "" {
		return workflowTaskClassHint{}
	}
	if cached, ok := r.taskHintCache[task]; ok {
		return cached
	}

	if r.container == nil || r.container.DB == nil {
		hint := defaultTaskClassHint("task_lookup_unavailable")
		r.taskHintCache[task] = hint
		return hint
	}

	var capabilityKey string
	var channel string
	err := r.container.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE((
				SELECT ts.capability_key
				FROM task_runs tr
				JOIN task_steps ts ON ts.run_id = tr.id
				WHERE tr.task_id = t.id
				  AND TRIM(COALESCE(ts.capability_key, '')) <> ''
				ORDER BY COALESCE(NULLIF(tr.updated_at, ''), tr.created_at) DESC, ts.step_index ASC
				LIMIT 1
			), ''),
			COALESCE(t.channel, '')
		FROM tasks t
		WHERE t.id = ?
	`, task).Scan(&capabilityKey, &channel)
	if err == sql.ErrNoRows {
		hint := defaultTaskClassHint("task_not_found")
		r.taskHintCache[task] = hint
		return hint
	}
	if err != nil {
		hint := defaultTaskClassHint("task_lookup_error")
		r.taskHintCache[task] = hint
		return hint
	}

	hint := taskClassHintFromCapability(capabilityKey, "task_step_capability")
	if hint.TaskClass == "" {
		hint = taskClassHintFromTaskChannelWithSource(channel, "task_channel")
	}
	if hint.TaskClass == "" {
		hint = defaultTaskClassHint("task_default")
	}
	r.taskHintCache[task] = hint
	return hint
}

func taskClassHintFromCapability(capabilityKey string, source string) workflowTaskClassHint {
	key := strings.ToLower(strings.TrimSpace(capabilityKey))
	if key == "" {
		return workflowTaskClassHint{}
	}

	switch {
	case strings.HasPrefix(key, "mail_"), strings.HasPrefix(key, "mail."):
		return workflowTaskClassHint{TaskClass: "mail", Source: source}
	case strings.HasPrefix(key, "calendar_"), strings.HasPrefix(key, "calendar."):
		return workflowTaskClassHint{TaskClass: "calendar", Source: source}
	case strings.HasPrefix(key, "browser_"), strings.HasPrefix(key, "browser."):
		return workflowTaskClassHint{TaskClass: "browser", Source: source}
	case strings.HasPrefix(key, "finder_"), strings.HasPrefix(key, "finder."):
		return workflowTaskClassHint{TaskClass: "finder", Source: source}
	case strings.HasPrefix(key, "messages_"), strings.HasPrefix(key, "message_"), strings.HasPrefix(key, "messages."):
		return workflowTaskClassHint{TaskClass: "messages", Source: source}
	case strings.HasPrefix(key, "channel.messages"), strings.Contains(key, "imessage"), strings.Contains(key, "twilio"), strings.Contains(key, "sms"), strings.Contains(key, "voice"):
		return workflowTaskClassHint{TaskClass: "messages", Source: source}
	case strings.HasPrefix(key, "chat_"), strings.HasPrefix(key, "chat."), strings.HasPrefix(key, "comm_"), strings.HasPrefix(key, "comm."):
		return workflowTaskClassHint{TaskClass: "chat", Source: source}
	}

	parts := strings.FieldsFunc(key, func(r rune) bool {
		switch r {
		case '_', '.', '/', ':':
			return true
		default:
			return false
		}
	})
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		return workflowTaskClassHint{
			TaskClass: strings.TrimSpace(parts[0]),
			Source:    source,
		}
	}
	return workflowTaskClassHint{}
}

func taskClassHintFromTaskChannel(channel string) workflowTaskClassHint {
	return taskClassHintFromTaskChannelWithSource(channel, "task_channel")
}

func taskClassHintFromTaskChannelWithSource(channel string, source string) workflowTaskClassHint {
	normalized := strings.ToLower(strings.TrimSpace(channel))
	switch normalized {
	case "app":
		return workflowTaskClassHint{TaskClass: "chat", Source: source}
	case "message", "voice":
		return workflowTaskClassHint{TaskClass: "messages", Source: source}
	case "mail":
		return workflowTaskClassHint{TaskClass: "mail", Source: source}
	case "calendar":
		return workflowTaskClassHint{TaskClass: "calendar", Source: source}
	case "browser":
		return workflowTaskClassHint{TaskClass: "browser", Source: source}
	case "finder":
		return workflowTaskClassHint{TaskClass: "finder", Source: source}
	default:
		return workflowTaskClassHint{}
	}
}

func taskClassHintFromTriggerType(triggerType string) workflowTaskClassHint {
	normalized := strings.ToUpper(strings.TrimSpace(triggerType))
	switch normalized {
	case "ON_COMM_EVENT":
		return workflowTaskClassHint{TaskClass: "chat", Source: "trigger_type"}
	case "SCHEDULE":
		return workflowTaskClassHint{TaskClass: modelpolicy.TaskClassDefault, Source: "trigger_type"}
	default:
		return workflowTaskClassHint{}
	}
}

func defaultTaskClassHint(source string) workflowTaskClassHint {
	return workflowTaskClassHint{
		TaskClass: modelpolicy.TaskClassDefault,
		Source:    normalizeTaskClassSource(source),
	}
}

func normalizeTaskClassSource(raw string) string {
	source := strings.ToLower(strings.TrimSpace(raw))
	if source == "" {
		return "default"
	}
	return source
}

func normalizeRouteSource(raw string) string {
	source := strings.ToLower(strings.TrimSpace(raw))
	if source == "" {
		return "resolved"
	}
	return source
}
