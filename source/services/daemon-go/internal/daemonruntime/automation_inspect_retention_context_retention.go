package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	contextrepo "personalagent/runtime/internal/core/repository/context"
	reporetention "personalagent/runtime/internal/core/repository/retention"
	contextservice "personalagent/runtime/internal/core/service/context"
	retentionservice "personalagent/runtime/internal/core/service/retention"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) PurgeRetention(ctx context.Context, request transport.RetentionPurgeRequest) (transport.RetentionPurgeResponse, error) {
	var policy *types.RetentionPolicy
	if request.TraceDays > 0 || request.TranscriptDays > 0 || request.MemoryDays > 0 {
		policy = &types.RetentionPolicy{
			TraceDays:      request.TraceDays,
			TranscriptDays: request.TranscriptDays,
			MemoryDays:     request.MemoryDays,
		}
	}

	service := retentionservice.NewService(reporetention.NewSQLiteRetentionStore(s.container.DB), nil)
	result, err := service.Purge(ctx, policy)
	if err != nil {
		return transport.RetentionPurgeResponse{}, err
	}

	effective := types.DefaultRetentionPolicy()
	if policy != nil {
		effective.TraceDays = positiveIntOrDefault(policy.TraceDays, effective.TraceDays)
		effective.TranscriptDays = positiveIntOrDefault(policy.TranscriptDays, effective.TranscriptDays)
		effective.MemoryDays = positiveIntOrDefault(policy.MemoryDays, effective.MemoryDays)
	}

	return transport.RetentionPurgeResponse{
		EffectivePolicy: effective,
		Result:          result,
	}, nil
}

func (s *AutomationInspectRetentionContextService) CompactRetentionMemory(ctx context.Context, request transport.RetentionCompactMemoryRequest) (transport.RetentionCompactMemoryResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	owner := strings.TrimSpace(request.OwnerActor)
	if owner == "" {
		return transport.RetentionCompactMemoryResponse{}, fmt.Errorf("--owner is required")
	}

	records, err := loadMemoryRecordsForCompaction(ctx, s.container.DB, workspace, owner, maxInt(1, request.Limit))
	if err != nil {
		return transport.RetentionCompactMemoryResponse{}, err
	}

	compactor := contextservice.NewMemoryCompactor(nil)
	result := compactor.Compact(records, types.MemoryCompactionConfig{
		TokenThreshold: maxInt(1, request.TokenThreshold),
		StaleAfter:     time.Duration(maxInt(1, request.StaleAfterHours)) * time.Hour,
	})

	applied := false
	createdSummaries := make([]string, 0)
	if request.Apply {
		ids, err := applyMemoryCompaction(ctx, s.container.DB, workspace, owner, result)
		if err != nil {
			return transport.RetentionCompactMemoryResponse{}, err
		}
		applied = true
		createdSummaries = ids
	}

	return transport.RetentionCompactMemoryResponse{
		WorkspaceID:       workspace,
		OwnerActorID:      owner,
		Applied:           applied,
		CreatedSummaryIDs: createdSummaries,
		Result:            result,
	}, nil
}

func (s *AutomationInspectRetentionContextService) ListContextSamples(ctx context.Context, request transport.ContextSamplesRequest) (transport.ContextSamplesResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := strings.TrimSpace(request.TaskClass)
	if taskClass == "" {
		return transport.ContextSamplesResponse{}, fmt.Errorf("--task-class is required")
	}

	store := contextrepo.NewSQLiteBudgetTelemetryStore(s.container.DB)
	samples, err := store.ListRecentContextBudgetSamples(ctx, workspace, taskClass, maxInt(1, request.Limit))
	if err != nil {
		return transport.ContextSamplesResponse{}, err
	}
	profile, exists, err := store.GetContextBudgetTuningProfile(ctx, workspace, taskClass)
	if err != nil {
		return transport.ContextSamplesResponse{}, err
	}

	return transport.ContextSamplesResponse{
		WorkspaceID:  workspace,
		TaskClass:    taskClass,
		Samples:      samples,
		Profile:      profile,
		ProfileFound: exists,
	}, nil
}

func (s *AutomationInspectRetentionContextService) TuneContext(ctx context.Context, request transport.ContextTuneRequest) (transport.ContextTuneResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := strings.TrimSpace(request.TaskClass)
	if taskClass == "" {
		return transport.ContextTuneResponse{}, fmt.Errorf("--task-class is required")
	}

	store := contextrepo.NewSQLiteBudgetTelemetryStore(s.container.DB)
	loop := contextservice.NewTuningLoop(store, contextservice.TuningLoopOptions{})
	decision, err := loop.TuneTaskClass(ctx, workspace, taskClass)
	if err != nil {
		return transport.ContextTuneResponse{}, err
	}
	return decision, nil
}

func loadMemoryRecordsForCompaction(ctx context.Context, db *sql.DB, workspaceID string, ownerActorID string, limit int) ([]types.MemoryRecord, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, scope_type, status, value_json, COALESCE(source_summary, ''), updated_at
		FROM memory_items
		WHERE workspace_id = ?
		  AND owner_principal_actor_id = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT ?
	`, workspaceID, ownerActorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list memory records: %w", err)
	}
	defer rows.Close()

	records := make([]types.MemoryRecord, 0)
	for rows.Next() {
		var (
			record    types.MemoryRecord
			scopeType string
			valueJSON string
			updatedAt string
		)
		if err := rows.Scan(&record.ID, &scopeType, &record.Status, &valueJSON, &record.SourceRef, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan memory record: %w", err)
		}
		record.Kind, record.IsCanonical, record.TokenEstimate = parseMemoryValueJSON(valueJSON, scopeType)
		record.Content = valueJSON
		record.LastUpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory records: %w", err)
	}
	return records, nil
}

func applyMemoryCompaction(ctx context.Context, db *sql.DB, workspaceID string, ownerActorID string, result types.MemoryCompactionResult) ([]string, error) {
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if len(result.DroppedIDs) == 0 && len(result.Summaries) == 0 {
		return []string{}, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin memory compaction apply tx: %w", err)
	}
	defer tx.Rollback()

	if len(result.DroppedIDs) > 0 {
		const droppedChunkSize = 128
		for start := 0; start < len(result.DroppedIDs); start += droppedChunkSize {
			end := start + droppedChunkSize
			if end > len(result.DroppedIDs) {
				end = len(result.DroppedIDs)
			}
			if err := disableDroppedMemoryRecordsTx(ctx, tx, workspaceID, ownerActorID, result.DroppedIDs[start:end], nowText); err != nil {
				return nil, err
			}
		}
	}

	summaryIDs := make([]string, 0, len(result.Summaries))
	for index, summary := range result.Summaries {
		id, err := automationRandomID("memsum")
		if err != nil {
			return nil, fmt.Errorf("generate summary id: %w", err)
		}
		summaryPayload, err := json.Marshal(map[string]any{
			"kind":           "summary",
			"is_canonical":   false,
			"token_estimate": summary.TokenEstimate,
			"content":        summary.Content,
			"source_ids":     summary.SourceIDs,
			"source_refs":    summary.SourceRefs,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal summary payload: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO memory_items(
				id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?)
		`, id, workspaceID, ownerActorID, "conversation", fmt.Sprintf("summary_%d_%s", index, summary.SummaryID), string(summaryPayload), strings.Join(summary.SourceRefs, ","), nowText, nowText); err != nil {
			return nil, fmt.Errorf("insert summary memory item: %w", err)
		}
		summaryIDs = append(summaryIDs, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit memory compaction apply tx: %w", err)
	}
	return summaryIDs, nil
}

func disableDroppedMemoryRecordsTx(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	ownerActorID string,
	droppedIDs []string,
	nowText string,
) error {
	if len(droppedIDs) == 0 {
		return nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(droppedIDs)), ",")
	query := "UPDATE memory_items SET status = 'DISABLED', updated_at = ? WHERE workspace_id = ? AND owner_principal_actor_id = ? AND id IN (" + placeholders + ")"
	params := make([]any, 0, 3+len(droppedIDs))
	params = append(params, nowText, workspaceID, ownerActorID)
	for _, id := range droppedIDs {
		params = append(params, id)
	}
	if _, err := tx.ExecContext(ctx, query, params...); err != nil {
		return fmt.Errorf("disable dropped memory records: %w", err)
	}
	return nil
}

func parseMemoryValueJSON(valueJSON string, fallbackKind string) (kind string, isCanonical bool, tokenEstimate int) {
	kind = strings.TrimSpace(fallbackKind)
	if kind == "" {
		kind = "memory"
	}
	isCanonical = false
	tokenEstimate = 0

	var payload map[string]any
	if err := json.Unmarshal([]byte(valueJSON), &payload); err != nil {
		return kind, isCanonical, estimateTokenCount(valueJSON)
	}

	if candidate, ok := payload["kind"].(string); ok && strings.TrimSpace(candidate) != "" {
		kind = strings.TrimSpace(candidate)
	}
	if candidate, ok := payload["is_canonical"].(bool); ok {
		isCanonical = candidate
	}
	if candidate, ok := payload["token_estimate"].(float64); ok {
		tokenEstimate = int(candidate)
	}
	if tokenEstimate <= 0 {
		if content, ok := payload["content"].(string); ok {
			tokenEstimate = estimateTokenCount(content)
		} else {
			tokenEstimate = estimateTokenCount(valueJSON)
		}
	}

	return kind, isCanonical, tokenEstimate
}

func estimateTokenCount(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	return maxInt(1, len(trimmed)/4)
}

func positiveIntOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
