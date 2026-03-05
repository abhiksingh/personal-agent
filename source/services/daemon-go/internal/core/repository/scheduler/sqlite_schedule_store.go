package scheduler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
)

type SQLiteScheduleStore struct {
	db *sql.DB
}

func NewSQLiteScheduleStore(db *sql.DB) *SQLiteScheduleStore {
	return &SQLiteScheduleStore{db: db}
}

func (s *SQLiteScheduleStore) ListEnabledScheduleTriggers(ctx context.Context) ([]types.ScheduleTrigger, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT at.id, at.workspace_id, at.directive_id, COALESCE(at.filter_json, '{}'), d.subject_principal_actor_id,
		        COALESCE(d.title, ''), COALESCE(d.instruction, '')
		 FROM automation_triggers at
		 JOIN directives d ON d.id = at.directive_id
		 WHERE at.trigger_type = 'SCHEDULE'
		   AND at.is_enabled = 1
		   AND d.status = 'ACTIVE'
		 ORDER BY at.created_at ASC, at.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled schedule triggers: %w", err)
	}
	defer rows.Close()

	triggers := []types.ScheduleTrigger{}
	for rows.Next() {
		var trigger types.ScheduleTrigger
		if err := rows.Scan(
			&trigger.TriggerID,
			&trigger.WorkspaceID,
			&trigger.DirectiveID,
			&trigger.FilterJSON,
			&trigger.SubjectPrincipalActor,
			&trigger.DirectiveTitle,
			&trigger.DirectiveInstruction,
		); err != nil {
			return nil, fmt.Errorf("scan schedule trigger: %w", err)
		}
		triggers = append(triggers, trigger)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate schedule triggers: %w", rows.Err())
	}
	return triggers, nil
}

func (s *SQLiteScheduleStore) TryReserveScheduleFire(
	ctx context.Context,
	trigger types.ScheduleTrigger,
	sourceEventID string,
	firedAt time.Time,
) (types.TriggerFireReservation, bool, error) {
	fireID, err := randomID()
	if err != nil {
		return types.TriggerFireReservation{}, false, err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO trigger_fires(
			id, workspace_id, trigger_id, source_event_id, fired_at, task_id, outcome
		) VALUES (?, ?, ?, ?, ?, NULL, 'PENDING')`,
		fireID,
		trigger.WorkspaceID,
		trigger.TriggerID,
		sourceEventID,
		firedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return types.TriggerFireReservation{}, false, nil
		}
		return types.TriggerFireReservation{}, false, fmt.Errorf("reserve schedule fire: %w", err)
	}

	return types.TriggerFireReservation{
		FireID:        fireID,
		WorkspaceID:   trigger.WorkspaceID,
		TriggerID:     trigger.TriggerID,
		SourceEventID: sourceEventID,
		FiredAt:       firedAt.UTC(),
	}, true, nil
}

func (s *SQLiteScheduleStore) MarkScheduleFireOutcome(ctx context.Context, fireID string, taskID string, outcome string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE trigger_fires
		 SET task_id = ?, outcome = ?
		 WHERE id = ?`,
		nullIfEmpty(taskID),
		outcome,
		fireID,
	)
	if err != nil {
		return fmt.Errorf("mark schedule fire outcome: %w", err)
	}
	return nil
}

func (s *SQLiteScheduleStore) CreateTaskForScheduledDirective(
	ctx context.Context,
	trigger types.ScheduleTrigger,
	sourceEventID string,
	now time.Time,
) (string, error) {
	taskID, err := randomID()
	if err != nil {
		return "", err
	}
	runID, err := randomID()
	if err != nil {
		return "", err
	}

	nowText := now.UTC().Format(time.RFC3339Nano)
	title := fmt.Sprintf("Scheduled directive %s", trigger.DirectiveID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin scheduled task transaction: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO tasks(
			id, workspace_id, requested_by_actor_id, subject_principal_actor_id,
			title, description, state, priority, deadline_at, channel,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'queued', 0, NULL, 'automation', ?, ?)`,
		taskID,
		trigger.WorkspaceID,
		trigger.SubjectPrincipalActor,
		trigger.SubjectPrincipalActor,
		title,
		sourceEventID,
		nowText,
		nowText,
	); err != nil {
		_ = tx.Rollback()
		return "", fmt.Errorf("insert scheduled task: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO task_runs(
			id, workspace_id, task_id, acting_as_actor_id,
			state, started_at, finished_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'queued', NULL, NULL, NULL, ?, ?)`,
		runID,
		trigger.WorkspaceID,
		taskID,
		trigger.SubjectPrincipalActor,
		nowText,
		nowText,
	); err != nil {
		_ = tx.Rollback()
		return "", fmt.Errorf("insert scheduled task run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit scheduled task transaction: %w", err)
	}

	return taskID, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
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
