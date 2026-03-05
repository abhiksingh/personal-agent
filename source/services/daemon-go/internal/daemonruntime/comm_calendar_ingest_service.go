package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

const calendarChangeIngestSource = "apple_calendar_eventkit"
const calendarConnectorID = "calendar"

type calendarChangeIngestPersistInput struct {
	WorkspaceID   string
	Source        string
	SourceScope   string
	SourceEventID string
	SourceCursor  string
	CalendarID    string
	CalendarName  string
	EventUID      string
	ChangeType    string
	Title         string
	Notes         string
	Location      string
	StartsAt      string
	EndsAt        string
	OccurredAt    string
}

type calendarChangeIngestPersistResult struct {
	EventID  string
	ThreadID string
	Replayed bool
}

func (s *CommTwilioService) IngestCalendarChange(ctx context.Context, request transport.CalendarChangeIngestRequest) (transport.CalendarChangeIngestResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.CalendarChangeIngestResponse{}, fmt.Errorf("comm service container db is required")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	sourceScope := resolveCalendarSourceScope(request.SourceScope, request.CalendarID)
	sourceEventID := firstNonEmpty(strings.TrimSpace(request.SourceEventID), strings.TrimSpace(request.EventUID))
	if strings.TrimSpace(sourceEventID) == "" {
		return transport.CalendarChangeIngestResponse{}, fmt.Errorf("calendar ingest source_event_id or event_uid is required")
	}
	occurredAt := normalizeOccurredAt(request.OccurredAt)
	sourceCursor := resolveCalendarSourceCursor(request.SourceCursor, occurredAt)
	changeType := normalizeCalendarChangeType(request.ChangeType)

	persisted, err := persistCalendarChangeEvent(ctx, s.container.DB, calendarChangeIngestPersistInput{
		WorkspaceID:   workspace,
		Source:        calendarChangeIngestSource,
		SourceScope:   sourceScope,
		SourceEventID: sourceEventID,
		SourceCursor:  sourceCursor,
		CalendarID:    strings.TrimSpace(request.CalendarID),
		CalendarName:  strings.TrimSpace(request.CalendarName),
		EventUID:      strings.TrimSpace(request.EventUID),
		ChangeType:    changeType,
		Title:         strings.TrimSpace(request.Title),
		Notes:         strings.TrimSpace(request.Notes),
		Location:      strings.TrimSpace(request.Location),
		StartsAt:      normalizeOptionalTimestamp(request.StartsAt),
		EndsAt:        normalizeOptionalTimestamp(request.EndsAt),
		OccurredAt:    occurredAt,
	})
	if err != nil {
		_ = upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, calendarChangeIngestSource, sourceScope, sourceCursor, "", err.Error())
		return transport.CalendarChangeIngestResponse{}, err
	}

	if err := upsertCommIngestCursor(ctx, s.container.DB, workspace, calendarChangeIngestSource, sourceScope, sourceCursor); err != nil {
		return transport.CalendarChangeIngestResponse{}, err
	}
	if err := upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, calendarChangeIngestSource, sourceScope, sourceCursor, persisted.EventID, ""); err != nil {
		return transport.CalendarChangeIngestResponse{}, err
	}

	if !persisted.Replayed {
		s.evaluateAutomationForCommEvents(ctx, true, false, persisted.EventID)
	}

	return transport.CalendarChangeIngestResponse{
		WorkspaceID:   workspace,
		Source:        calendarChangeIngestSource,
		SourceScope:   sourceScope,
		SourceEventID: sourceEventID,
		SourceCursor:  sourceCursor,
		Accepted:      true,
		Replayed:      persisted.Replayed,
		EventID:       persisted.EventID,
		ThreadID:      persisted.ThreadID,
		EventUID:      strings.TrimSpace(request.EventUID),
		ChangeType:    changeType,
	}, nil
}

func persistCalendarChangeEvent(ctx context.Context, db *sql.DB, input calendarChangeIngestPersistInput) (calendarChangeIngestPersistResult, error) {
	if db == nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("db is required")
	}
	workspace := normalizeWorkspaceID(input.WorkspaceID)
	source := strings.TrimSpace(input.Source)
	sourceScope := strings.TrimSpace(input.SourceScope)
	sourceEventID := strings.TrimSpace(input.SourceEventID)
	if source == "" || sourceScope == "" || sourceEventID == "" {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("workspace/source/source_scope/source_event_id are required")
	}

	eventUID := strings.TrimSpace(input.EventUID)
	calendarID := strings.TrimSpace(input.CalendarID)
	calendarName := strings.TrimSpace(input.CalendarName)
	changeType := normalizeCalendarChangeType(input.ChangeType)
	title := strings.TrimSpace(input.Title)
	notes := strings.TrimSpace(input.Notes)
	location := strings.TrimSpace(input.Location)
	startsAt := normalizeOptionalTimestamp(input.StartsAt)
	endsAt := normalizeOptionalTimestamp(input.EndsAt)
	occurredAt := normalizeOccurredAt(input.OccurredAt)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	threadExternalRef := firstNonEmpty(eventUID, sourceEventID)
	threadID := deterministicMessagesID("thread", workspace, source, sourceScope, threadExternalRef)
	eventID := deterministicMessagesID("event", workspace, source, sourceEventID)
	receiptID := deterministicMessagesID("receipt", workspace, source, sourceEventID)
	payloadHash := hashMessagesPayload(
		sourceScope,
		sourceEventID,
		calendarID,
		calendarName,
		eventUID,
		changeType,
		title,
		notes,
		location,
		startsAt,
		endsAt,
		occurredAt,
	)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("begin calendar ingest tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return calendarChangeIngestPersistResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_ingest_receipts(
			id, workspace_id, source, source_scope, source_event_id, source_cursor,
			trust_state, event_id, payload_hash, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, 'accepted', NULL, ?, ?, ?)
	`, receiptID, workspace, source, sourceScope, sourceEventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, now); err != nil {
		if !isUniqueConstraintError(err) {
			return calendarChangeIngestPersistResult{}, fmt.Errorf("insert calendar ingest receipt: %w", err)
		}
		existingEventID, existingThreadID, loadErr := loadExistingCommIngestReceipt(ctx, tx, workspace, source, sourceEventID)
		if loadErr != nil {
			return calendarChangeIngestPersistResult{}, loadErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return calendarChangeIngestPersistResult{}, fmt.Errorf("commit replay calendar ingest tx: %w", commitErr)
		}
		return calendarChangeIngestPersistResult{
			EventID:  firstNonEmpty(existingEventID, eventID),
			ThreadID: firstNonEmpty(existingThreadID, threadID),
			Replayed: true,
		}, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, 'calendar', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			title = excluded.title,
			updated_at = excluded.updated_at
	`, threadID, workspace, calendarConnectorID, nullableText(threadExternalRef), nullableText(firstNonEmpty(title, calendarName, threadExternalRef)), now, now); err != nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("upsert calendar thread: %w", err)
	}

	bodyText := buildCalendarEventBody(changeType, calendarName, title, notes, location, startsAt, endsAt, eventUID, sourceEventID)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(
			id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
		) VALUES (?, ?, ?, ?, 'MESSAGE', 'INBOUND', 0, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, eventID, workspace, threadID, calendarConnectorID, occurredAt, nullableText(bodyText), now); err != nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("upsert calendar comm event: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_ingest_receipts
		SET event_id = ?, source_cursor = ?, payload_hash = ?, received_at = ?
		WHERE workspace_id = ? AND source = ? AND source_event_id = ?
	`, eventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, workspace, source, sourceEventID); err != nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("update calendar ingest receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return calendarChangeIngestPersistResult{}, fmt.Errorf("commit calendar ingest tx: %w", err)
	}
	return calendarChangeIngestPersistResult{
		EventID:  eventID,
		ThreadID: threadID,
		Replayed: false,
	}, nil
}

func resolveCalendarSourceScope(raw string, calendarID string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(calendarID); trimmed != "" {
		return trimmed
	}
	return "calendar-default"
}

func resolveCalendarSourceCursor(raw string, occurredAt string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(occurredAt))
	if err != nil {
		return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}
	return strconv.FormatInt(parsed.UTC().UnixNano(), 10)
}

func normalizeCalendarChangeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "created", "create":
		return "created"
	case "deleted", "delete", "removed", "cancelled", "canceled":
		return "deleted"
	default:
		return "updated"
	}
}

func normalizeOptionalTimestamp(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.UTC().Format(time.RFC3339Nano)
}

func buildCalendarEventBody(changeType string, calendarName string, title string, notes string, location string, startsAt string, endsAt string, eventUID string, sourceEventID string) string {
	sections := []string{
		fmt.Sprintf("calendar_change=%s", normalizeCalendarChangeType(changeType)),
		fmt.Sprintf("calendar_name=%s", firstNonEmpty(strings.TrimSpace(calendarName), "default")),
		fmt.Sprintf("title=%s", firstNonEmpty(strings.TrimSpace(title), "(untitled)")),
	}
	if trimmed := strings.TrimSpace(location); trimmed != "" {
		sections = append(sections, fmt.Sprintf("location=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(startsAt); trimmed != "" {
		sections = append(sections, fmt.Sprintf("starts_at=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(endsAt); trimmed != "" {
		sections = append(sections, fmt.Sprintf("ends_at=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(eventUID); trimmed != "" {
		sections = append(sections, fmt.Sprintf("event_uid=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(sourceEventID); trimmed != "" {
		sections = append(sections, fmt.Sprintf("source_event_id=%s", trimmed))
	}

	body := strings.Join(sections, "\n")
	trimmedNotes := strings.TrimSpace(notes)
	if trimmedNotes == "" {
		return body
	}
	return body + "\n\n" + trimmedNotes
}
