package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

const (
	defaultTaskRunListLimit = 25
	maxTaskRunListLimit     = 200
)

func (s *AgentDelegationService) ListTaskRuns(ctx context.Context, request transport.TaskRunListRequest) (transport.TaskRunListResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.TaskRunListResponse{}, fmt.Errorf("database is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	stateFilter := normalizeTaskRunListStateFilter(request.State)
	routeResolver := newWorkflowRouteMetadataResolver(s.container)

	query := `
		SELECT
			t.id,
			COALESCE(tr.id, ''),
			t.workspace_id,
			t.title,
			t.state,
			COALESCE(tr.state, ''),
			t.priority,
			t.requested_by_actor_id,
			t.subject_principal_actor_id,
			COALESCE(tr.acting_as_actor_id, ''),
			COALESCE(tr.last_error, ''),
			t.created_at,
			t.updated_at,
			COALESCE(tr.created_at, ''),
			COALESCE(tr.updated_at, ''),
			COALESCE(tr.started_at, ''),
			COALESCE(tr.finished_at, ''),
			COALESCE(t.channel, ''),
			COALESCE((
				SELECT ts.capability_key
				FROM task_steps ts
				WHERE ts.run_id = tr.id
				  AND TRIM(COALESCE(ts.capability_key, '')) <> ''
				ORDER BY ts.step_index ASC
				LIMIT 1
			), '')
		FROM tasks t
		LEFT JOIN task_runs tr ON tr.task_id = t.id
		WHERE t.workspace_id = ?
	`
	params := []any{workspace}

	if stateFilter != "" {
		query += " AND LOWER(COALESCE(NULLIF(tr.state, ''), t.state)) = ?"
		params = append(params, stateFilter)
	}

	query += `
		ORDER BY
			COALESCE(NULLIF(tr.updated_at, ''), t.updated_at) DESC,
			t.updated_at DESC,
			t.id DESC
		LIMIT ?
	`
	params = append(params, clampTaskRunListLimit(request.Limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.TaskRunListResponse{}, fmt.Errorf("list task runs: %w", err)
	}
	defer rows.Close()

	items := make([]transport.TaskRunListItem, 0)
	type routeHint struct {
		workspaceID    string
		taskID         string
		runID          string
		stepCapability string
		taskChannel    string
	}
	routeHints := make([]routeHint, 0)
	for rows.Next() {
		var item transport.TaskRunListItem
		var taskChannel string
		var stepCapability string
		if err := rows.Scan(
			&item.TaskID,
			&item.RunID,
			&item.WorkspaceID,
			&item.Title,
			&item.TaskState,
			&item.RunState,
			&item.Priority,
			&item.RequestedByActorID,
			&item.SubjectPrincipalActorID,
			&item.ActingAsActorID,
			&item.LastError,
			&item.TaskCreatedAt,
			&item.TaskUpdatedAt,
			&item.RunCreatedAt,
			&item.RunUpdatedAt,
			&item.StartedAt,
			&item.FinishedAt,
			&taskChannel,
			&stepCapability,
		); err != nil {
			return transport.TaskRunListResponse{}, fmt.Errorf("scan task run row: %w", err)
		}
		items = append(items, item)
		items[len(items)-1].Actions = transport.ResolveTaskRunActionAvailability(item.TaskState, item.RunState)
		routeHints = append(routeHints, routeHint{
			workspaceID:    item.WorkspaceID,
			taskID:         item.TaskID,
			runID:          item.RunID,
			stepCapability: stepCapability,
			taskChannel:    taskChannel,
		})
	}
	if err := rows.Err(); err != nil {
		return transport.TaskRunListResponse{}, fmt.Errorf("iterate task run rows: %w", err)
	}
	for idx := range items {
		hint := routeHints[idx]
		items[idx].Route = routeResolver.ResolveForTaskRun(
			ctx,
			hint.workspaceID,
			hint.taskID,
			hint.runID,
			hint.stepCapability,
			hint.taskChannel,
		)
	}

	return transport.TaskRunListResponse{
		WorkspaceID: workspace,
		Items:       items,
	}, nil
}

func normalizeTaskRunListStateFilter(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func clampTaskRunListLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultTaskRunListLimit
	case limit > maxTaskRunListLimit:
		return maxTaskRunListLimit
	default:
		return limit
	}
}
