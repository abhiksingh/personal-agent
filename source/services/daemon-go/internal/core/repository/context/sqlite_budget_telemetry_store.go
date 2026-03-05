package contextrepo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
)

type SQLiteBudgetTelemetryStore struct {
	db *sql.DB
}

func NewSQLiteBudgetTelemetryStore(db *sql.DB) *SQLiteBudgetTelemetryStore {
	return &SQLiteBudgetTelemetryStore{db: db}
}

func (s *SQLiteBudgetTelemetryStore) RecordContextBudgetSample(ctx context.Context, sample types.ContextBudgetSample) error {
	if strings.TrimSpace(sample.WorkspaceID) == "" {
		return errors.New("workspace id is required")
	}
	if strings.TrimSpace(sample.TaskClass) == "" {
		return errors.New("task class is required")
	}
	if sample.ContextWindow <= 0 {
		return errors.New("context window must be > 0")
	}
	if sample.RetrievalTarget < 0 || sample.RetrievalUsed < 0 {
		return errors.New("retrieval values must be >= 0")
	}
	if sample.PromptTokens < 0 || sample.CompletionTokens < 0 {
		return errors.New("token values must be >= 0")
	}

	sampleID := strings.TrimSpace(sample.SampleID)
	if sampleID == "" {
		generatedID, err := randomID()
		if err != nil {
			return err
		}
		sampleID = generatedID
	}

	recordedAt := sample.RecordedAt.UTC()
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO context_budget_samples(
			id,
			workspace_id,
			task_class,
			model_key,
			context_window,
			output_limit,
			deep_analysis,
			remaining_budget,
			retrieval_target,
			retrieval_used,
			prompt_tokens,
			completion_tokens,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sampleID,
		sample.WorkspaceID,
		sample.TaskClass,
		nullIfEmpty(sample.ModelKey),
		sample.ContextWindow,
		nullInt(sample.OutputLimit),
		boolToInt(sample.DeepAnalysis),
		sample.RemainingBudget,
		sample.RetrievalTarget,
		sample.RetrievalUsed,
		sample.PromptTokens,
		sample.CompletionTokens,
		recordedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert context budget sample: %w", err)
	}

	return nil
}

func (s *SQLiteBudgetTelemetryStore) ListRecentContextBudgetSamples(
	ctx context.Context,
	workspaceID string,
	taskClass string,
	limit int,
) ([]types.ContextBudgetSample, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id,
			workspace_id,
			task_class,
			COALESCE(model_key, ''),
			context_window,
			COALESCE(output_limit, 0),
			deep_analysis,
			remaining_budget,
			retrieval_target,
			retrieval_used,
			prompt_tokens,
			completion_tokens,
			created_at
		 FROM context_budget_samples
		 WHERE workspace_id = ?
		   AND task_class = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		workspaceID,
		taskClass,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list context budget samples: %w", err)
	}
	defer rows.Close()

	samples := []types.ContextBudgetSample{}
	for rows.Next() {
		var sample types.ContextBudgetSample
		var deepAnalysis int
		var createdAt string
		if scanErr := rows.Scan(
			&sample.SampleID,
			&sample.WorkspaceID,
			&sample.TaskClass,
			&sample.ModelKey,
			&sample.ContextWindow,
			&sample.OutputLimit,
			&deepAnalysis,
			&sample.RemainingBudget,
			&sample.RetrievalTarget,
			&sample.RetrievalUsed,
			&sample.PromptTokens,
			&sample.CompletionTokens,
			&createdAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan context budget sample: %w", scanErr)
		}
		sample.DeepAnalysis = deepAnalysis == 1
		sample.RecordedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		samples = append(samples, sample)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate context budget samples: %w", rows.Err())
	}

	return samples, nil
}

func (s *SQLiteBudgetTelemetryStore) GetContextBudgetTuningProfile(
	ctx context.Context,
	workspaceID string,
	taskClass string,
) (types.ContextBudgetTuningProfile, bool, error) {
	var profile types.ContextBudgetTuningProfile
	var updatedAt string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			workspace_id,
			task_class,
			retrieval_multiplier,
			sample_count,
			avg_retrieval_utilization,
			avg_prompt_utilization,
			updated_at
		 FROM context_budget_tuning_profiles
		 WHERE workspace_id = ? AND task_class = ?`,
		workspaceID,
		taskClass,
	).Scan(
		&profile.WorkspaceID,
		&profile.TaskClass,
		&profile.RetrievalMultiplier,
		&profile.SampleCount,
		&profile.AvgRetrievalUtilization,
		&profile.AvgPromptUtilization,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return types.ContextBudgetTuningProfile{}, false, nil
	}
	if err != nil {
		return types.ContextBudgetTuningProfile{}, false, fmt.Errorf("get context budget tuning profile: %w", err)
	}
	profile.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return profile, true, nil
}

func (s *SQLiteBudgetTelemetryStore) UpsertContextBudgetTuningProfile(
	ctx context.Context,
	profile types.ContextBudgetTuningProfile,
) error {
	if strings.TrimSpace(profile.WorkspaceID) == "" {
		return errors.New("workspace id is required")
	}
	if strings.TrimSpace(profile.TaskClass) == "" {
		return errors.New("task class is required")
	}
	if profile.RetrievalMultiplier <= 0 {
		return errors.New("retrieval multiplier must be > 0")
	}

	updatedAt := profile.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO context_budget_tuning_profiles(
			workspace_id,
			task_class,
			retrieval_multiplier,
			sample_count,
			avg_retrieval_utilization,
			avg_prompt_utilization,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, task_class)
		DO UPDATE SET
			retrieval_multiplier = excluded.retrieval_multiplier,
			sample_count = excluded.sample_count,
			avg_retrieval_utilization = excluded.avg_retrieval_utilization,
			avg_prompt_utilization = excluded.avg_prompt_utilization,
			updated_at = excluded.updated_at`,
		profile.WorkspaceID,
		profile.TaskClass,
		profile.RetrievalMultiplier,
		profile.SampleCount,
		profile.AvgRetrievalUtilization,
		profile.AvgPromptUtilization,
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert context budget tuning profile: %w", err)
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
