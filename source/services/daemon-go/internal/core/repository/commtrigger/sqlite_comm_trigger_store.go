package commtrigger

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

type SQLiteCommTriggerStore struct {
	db *sql.DB
}

func NewSQLiteCommTriggerStore(db *sql.DB) *SQLiteCommTriggerStore {
	return &SQLiteCommTriggerStore{db: db}
}

func (s *SQLiteCommTriggerStore) LoadCommEvent(ctx context.Context, eventID string) (types.CommEventRecord, error) {
	var event types.CommEventRecord
	err := s.db.QueryRowContext(
		ctx,
		`SELECT ce.id, ce.workspace_id, ce.thread_id, COALESCE(ct.channel, ''), ce.event_type,
		        ce.direction, ce.assistant_emitted, COALESCE(ce.body_text, ''),
		        COALESCE((
		          SELECT cea.address_value
		          FROM comm_event_addresses cea
		          WHERE cea.event_id = ce.id AND cea.address_role = 'FROM'
		          ORDER BY cea.position
		          LIMIT 1
		        ), '')
		 FROM comm_events ce
		 JOIN comm_threads ct ON ct.id = ce.thread_id
		 WHERE ce.id = ?`,
		eventID,
	).Scan(
		&event.EventID,
		&event.WorkspaceID,
		&event.ThreadID,
		&event.Channel,
		&event.EventType,
		&event.Direction,
		&event.AssistantEmitted,
		&event.BodyText,
		&event.SenderAddress,
	)
	if err != nil {
		return types.CommEventRecord{}, fmt.Errorf("load comm event: %w", err)
	}
	return event, nil
}

func (s *SQLiteCommTriggerStore) ListEnabledOnCommEventTriggers(ctx context.Context, workspaceID string) ([]types.OnCommEventTrigger, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT at.id, at.workspace_id, at.directive_id, d.subject_principal_actor_id, COALESCE(at.filter_json, '{}'),
		        COALESCE(d.title, ''), COALESCE(d.instruction, '')
		 FROM automation_triggers at
		 JOIN directives d ON d.id = at.directive_id
		 WHERE at.workspace_id = ?
		   AND at.trigger_type = 'ON_COMM_EVENT'
		   AND at.is_enabled = 1
		   AND d.status = 'ACTIVE'
		 ORDER BY at.created_at ASC, at.id ASC`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list on-comm-event triggers: %w", err)
	}
	defer rows.Close()

	triggers := []types.OnCommEventTrigger{}
	for rows.Next() {
		var trigger types.OnCommEventTrigger
		if err := rows.Scan(
			&trigger.TriggerID,
			&trigger.WorkspaceID,
			&trigger.DirectiveID,
			&trigger.SubjectPrincipalActor,
			&trigger.FilterJSON,
			&trigger.DirectiveTitle,
			&trigger.DirectiveInstruction,
		); err != nil {
			return nil, fmt.Errorf("scan on-comm-event trigger: %w", err)
		}
		triggers = append(triggers, trigger)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate on-comm-event triggers: %w", rows.Err())
	}
	return triggers, nil
}

func (s *SQLiteCommTriggerStore) TryReserveTriggerFire(
	ctx context.Context,
	triggerID string,
	workspaceID string,
	sourceEventID string,
	firedAt time.Time,
) (string, bool, error) {
	fireID, err := randomID()
	if err != nil {
		return "", false, err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO trigger_fires(id, workspace_id, trigger_id, source_event_id, fired_at, task_id, outcome)
		 VALUES (?, ?, ?, ?, ?, NULL, 'PENDING')`,
		fireID,
		workspaceID,
		triggerID,
		sourceEventID,
		firedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return "", false, nil
		}
		return "", false, fmt.Errorf("reserve trigger fire: %w", err)
	}
	return fireID, true, nil
}

func (s *SQLiteCommTriggerStore) MarkTriggerFireOutcome(ctx context.Context, fireID string, taskID string, outcome string) error {
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
		return fmt.Errorf("mark trigger fire outcome: %w", err)
	}
	return nil
}

func (s *SQLiteCommTriggerStore) CreateTaskForDirective(
	ctx context.Context,
	trigger types.OnCommEventTrigger,
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
	title := fmt.Sprintf("ON_COMM_EVENT %s", trigger.DirectiveID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin comm-trigger task transaction: %w", err)
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
		return "", fmt.Errorf("insert comm-trigger task: %w", err)
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
		return "", fmt.Errorf("insert comm-trigger run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit comm-trigger task transaction: %w", err)
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
