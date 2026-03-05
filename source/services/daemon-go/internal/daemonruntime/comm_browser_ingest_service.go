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

const browserEventIngestSource = "apple_safari_extension"
const browserConnectorID = "browser"

type browserEventIngestPersistInput struct {
	WorkspaceID   string
	Source        string
	SourceScope   string
	SourceEventID string
	SourceCursor  string
	WindowID      string
	TabID         string
	PageURL       string
	PageTitle     string
	EventType     string
	PayloadText   string
	OccurredAt    string
}

type browserEventIngestPersistResult struct {
	EventID  string
	ThreadID string
	Replayed bool
}

func (s *CommTwilioService) IngestBrowserEvent(ctx context.Context, request transport.BrowserEventIngestRequest) (transport.BrowserEventIngestResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.BrowserEventIngestResponse{}, fmt.Errorf("comm service container db is required")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	sourceScope := resolveBrowserSourceScope(request.SourceScope, request.WindowID)
	occurredAt := normalizeOccurredAt(request.OccurredAt)
	sourceCursor := resolveBrowserSourceCursor(request.SourceCursor, occurredAt)
	sourceEventID := resolveBrowserSourceEventID(request.SourceEventID, request.TabID, sourceCursor, request.PageURL)
	if strings.TrimSpace(sourceEventID) == "" {
		return transport.BrowserEventIngestResponse{}, fmt.Errorf("browser ingest source_event_id is required")
	}
	eventType := normalizeBrowserEventType(request.EventType)

	persisted, err := persistBrowserInboundEvent(ctx, s.container.DB, browserEventIngestPersistInput{
		WorkspaceID:   workspace,
		Source:        browserEventIngestSource,
		SourceScope:   sourceScope,
		SourceEventID: sourceEventID,
		SourceCursor:  sourceCursor,
		WindowID:      strings.TrimSpace(request.WindowID),
		TabID:         strings.TrimSpace(request.TabID),
		PageURL:       strings.TrimSpace(request.PageURL),
		PageTitle:     strings.TrimSpace(request.PageTitle),
		EventType:     eventType,
		PayloadText:   strings.TrimSpace(request.PayloadText),
		OccurredAt:    occurredAt,
	})
	if err != nil {
		_ = upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, browserEventIngestSource, sourceScope, sourceCursor, "", err.Error())
		return transport.BrowserEventIngestResponse{}, err
	}

	if err := upsertCommIngestCursor(ctx, s.container.DB, workspace, browserEventIngestSource, sourceScope, sourceCursor); err != nil {
		return transport.BrowserEventIngestResponse{}, err
	}
	if err := upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, browserEventIngestSource, sourceScope, sourceCursor, persisted.EventID, ""); err != nil {
		return transport.BrowserEventIngestResponse{}, err
	}

	if !persisted.Replayed {
		s.evaluateAutomationForCommEvents(ctx, true, false, persisted.EventID)
	}

	return transport.BrowserEventIngestResponse{
		WorkspaceID:   workspace,
		Source:        browserEventIngestSource,
		SourceScope:   sourceScope,
		SourceEventID: sourceEventID,
		SourceCursor:  sourceCursor,
		Accepted:      true,
		Replayed:      persisted.Replayed,
		EventID:       persisted.EventID,
		ThreadID:      persisted.ThreadID,
		EventType:     eventType,
		PageURL:       strings.TrimSpace(request.PageURL),
	}, nil
}

func persistBrowserInboundEvent(ctx context.Context, db *sql.DB, input browserEventIngestPersistInput) (browserEventIngestPersistResult, error) {
	if db == nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("db is required")
	}
	workspace := normalizeWorkspaceID(input.WorkspaceID)
	source := strings.TrimSpace(input.Source)
	sourceScope := strings.TrimSpace(input.SourceScope)
	sourceEventID := strings.TrimSpace(input.SourceEventID)
	if source == "" || sourceScope == "" || sourceEventID == "" {
		return browserEventIngestPersistResult{}, fmt.Errorf("workspace/source/source_scope/source_event_id are required")
	}

	windowID := strings.TrimSpace(input.WindowID)
	tabID := strings.TrimSpace(input.TabID)
	pageURL := strings.TrimSpace(input.PageURL)
	pageTitle := strings.TrimSpace(input.PageTitle)
	eventType := normalizeBrowserEventType(input.EventType)
	payloadText := strings.TrimSpace(input.PayloadText)
	occurredAt := normalizeOccurredAt(input.OccurredAt)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	threadExternalRef := firstNonEmpty(tabID, windowID, pageURL, sourceEventID)
	threadID := deterministicMessagesID("thread", workspace, source, sourceScope, threadExternalRef)
	eventID := deterministicMessagesID("event", workspace, source, sourceEventID)
	receiptID := deterministicMessagesID("receipt", workspace, source, sourceEventID)
	payloadHash := hashMessagesPayload(
		sourceScope,
		sourceEventID,
		windowID,
		tabID,
		pageURL,
		pageTitle,
		eventType,
		payloadText,
		occurredAt,
	)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("begin browser ingest tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return browserEventIngestPersistResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_ingest_receipts(
			id, workspace_id, source, source_scope, source_event_id, source_cursor,
			trust_state, event_id, payload_hash, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, 'accepted', NULL, ?, ?, ?)
	`, receiptID, workspace, source, sourceScope, sourceEventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, now); err != nil {
		if !isUniqueConstraintError(err) {
			return browserEventIngestPersistResult{}, fmt.Errorf("insert browser ingest receipt: %w", err)
		}
		existingEventID, existingThreadID, loadErr := loadExistingCommIngestReceipt(ctx, tx, workspace, source, sourceEventID)
		if loadErr != nil {
			return browserEventIngestPersistResult{}, loadErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return browserEventIngestPersistResult{}, fmt.Errorf("commit replay browser ingest tx: %w", commitErr)
		}
		return browserEventIngestPersistResult{
			EventID:  firstNonEmpty(existingEventID, eventID),
			ThreadID: firstNonEmpty(existingThreadID, threadID),
			Replayed: true,
		}, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, 'browser', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			title = excluded.title,
			updated_at = excluded.updated_at
	`, threadID, workspace, browserConnectorID, nullableText(threadExternalRef), nullableText(firstNonEmpty(pageTitle, pageURL, threadExternalRef)), now, now); err != nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("upsert browser thread: %w", err)
	}

	bodyText := buildBrowserEventBody(eventType, pageURL, pageTitle, windowID, tabID, payloadText, sourceEventID)
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
	`, eventID, workspace, threadID, browserConnectorID, occurredAt, nullableText(bodyText), now); err != nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("upsert browser comm event: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_ingest_receipts
		SET event_id = ?, source_cursor = ?, payload_hash = ?, received_at = ?
		WHERE workspace_id = ? AND source = ? AND source_event_id = ?
	`, eventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, workspace, source, sourceEventID); err != nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("update browser ingest receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return browserEventIngestPersistResult{}, fmt.Errorf("commit browser ingest tx: %w", err)
	}
	return browserEventIngestPersistResult{
		EventID:  eventID,
		ThreadID: threadID,
		Replayed: false,
	}, nil
}

func resolveBrowserSourceScope(raw string, windowID string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(windowID); trimmed != "" {
		return trimmed
	}
	return "safari-extension-default"
}

func resolveBrowserSourceCursor(raw string, occurredAt string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(occurredAt))
	if err != nil {
		return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}
	return strconv.FormatInt(parsed.UTC().UnixNano(), 10)
}

func resolveBrowserSourceEventID(raw string, tabID string, sourceCursor string, pageURL string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	tab := strings.TrimSpace(tabID)
	cursor := strings.TrimSpace(sourceCursor)
	if tab != "" && cursor != "" {
		return tab + "@" + cursor
	}
	if cursor != "" && strings.TrimSpace(pageURL) != "" {
		return deterministicMessagesID("browser-event", strings.TrimSpace(pageURL), cursor)
	}
	return ""
}

func normalizeBrowserEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "navigation", "navigate":
		return "navigation"
	case "message", "bridge_message", "extension_message":
		return "message"
	case "tab_closed", "closed":
		return "tab_closed"
	case "tab_opened", "opened":
		return "tab_opened"
	default:
		return "update"
	}
}

func buildBrowserEventBody(eventType string, pageURL string, pageTitle string, windowID string, tabID string, payloadText string, sourceEventID string) string {
	sections := []string{
		fmt.Sprintf("browser_event_type=%s", normalizeBrowserEventType(eventType)),
		fmt.Sprintf("page_url=%s", firstNonEmpty(strings.TrimSpace(pageURL), "(unknown)")),
		fmt.Sprintf("page_title=%s", firstNonEmpty(strings.TrimSpace(pageTitle), "(untitled)")),
	}
	if trimmed := strings.TrimSpace(windowID); trimmed != "" {
		sections = append(sections, fmt.Sprintf("window_id=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(tabID); trimmed != "" {
		sections = append(sections, fmt.Sprintf("tab_id=%s", trimmed))
	}
	if trimmed := strings.TrimSpace(sourceEventID); trimmed != "" {
		sections = append(sections, fmt.Sprintf("source_event_id=%s", trimmed))
	}

	body := strings.Join(sections, "\n")
	trimmedPayload := strings.TrimSpace(payloadText)
	if trimmedPayload == "" {
		return body
	}
	return body + "\n\n" + trimmedPayload
}
