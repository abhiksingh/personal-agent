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

const mailRuleIngestSource = "apple_mail_rule"
const mailConnectorID = "mail"

type mailIngestPersistInput struct {
	WorkspaceID      string
	Source           string
	SourceScope      string
	SourceEventID    string
	SourceCursor     string
	MessageID        string
	ThreadRef        string
	InReplyTo        string
	ReferencesHeader string
	FromAddress      string
	ToAddress        string
	Subject          string
	BodyText         string
	OccurredAt       string
}

type mailIngestPersistResult struct {
	EventID  string
	ThreadID string
	Replayed bool
}

func (s *CommTwilioService) IngestMailRuleEvent(ctx context.Context, request transport.MailRuleIngestRequest) (transport.MailRuleIngestResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.MailRuleIngestResponse{}, fmt.Errorf("comm service container db is required")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	sourceScope := resolveMailSourceScope(request.SourceScope)
	sourceEventID := firstNonEmpty(strings.TrimSpace(request.SourceEventID), strings.TrimSpace(request.MessageID))
	if strings.TrimSpace(sourceEventID) == "" {
		return transport.MailRuleIngestResponse{}, fmt.Errorf("mail ingest source_event_id or message_id is required")
	}
	occurredAt := normalizeOccurredAt(request.OccurredAt)
	sourceCursor := resolveMailSourceCursor(request.SourceCursor, occurredAt)

	persisted, err := persistMailInboundEvent(ctx, s.container.DB, mailIngestPersistInput{
		WorkspaceID:      workspace,
		Source:           mailRuleIngestSource,
		SourceScope:      sourceScope,
		SourceEventID:    sourceEventID,
		SourceCursor:     sourceCursor,
		MessageID:        strings.TrimSpace(request.MessageID),
		ThreadRef:        strings.TrimSpace(request.ThreadRef),
		InReplyTo:        strings.TrimSpace(request.InReplyTo),
		ReferencesHeader: strings.TrimSpace(request.ReferencesHeader),
		FromAddress:      strings.TrimSpace(request.FromAddress),
		ToAddress:        strings.TrimSpace(request.ToAddress),
		Subject:          strings.TrimSpace(request.Subject),
		BodyText:         strings.TrimSpace(request.BodyText),
		OccurredAt:       occurredAt,
	})
	if err != nil {
		_ = upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, mailRuleIngestSource, sourceScope, sourceCursor, "", err.Error())
		return transport.MailRuleIngestResponse{}, err
	}

	if err := upsertCommIngestCursor(ctx, s.container.DB, workspace, mailRuleIngestSource, sourceScope, sourceCursor); err != nil {
		return transport.MailRuleIngestResponse{}, err
	}
	if err := upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, mailRuleIngestSource, sourceScope, sourceCursor, persisted.EventID, ""); err != nil {
		return transport.MailRuleIngestResponse{}, err
	}

	if !persisted.Replayed {
		s.evaluateAutomationForCommEvents(ctx, true, false, persisted.EventID)
	}

	return transport.MailRuleIngestResponse{
		WorkspaceID:   workspace,
		Source:        mailRuleIngestSource,
		SourceScope:   sourceScope,
		SourceEventID: sourceEventID,
		SourceCursor:  sourceCursor,
		Accepted:      true,
		Replayed:      persisted.Replayed,
		EventID:       persisted.EventID,
		ThreadID:      persisted.ThreadID,
		MessageID:     strings.TrimSpace(request.MessageID),
	}, nil
}

func persistMailInboundEvent(ctx context.Context, db *sql.DB, input mailIngestPersistInput) (mailIngestPersistResult, error) {
	if db == nil {
		return mailIngestPersistResult{}, fmt.Errorf("db is required")
	}
	workspace := normalizeWorkspaceID(input.WorkspaceID)
	source := strings.TrimSpace(input.Source)
	sourceScope := strings.TrimSpace(input.SourceScope)
	sourceEventID := strings.TrimSpace(input.SourceEventID)
	if source == "" || sourceScope == "" || sourceEventID == "" {
		return mailIngestPersistResult{}, fmt.Errorf("workspace/source/source_scope/source_event_id are required")
	}
	fromAddress := strings.TrimSpace(input.FromAddress)
	if fromAddress == "" {
		return mailIngestPersistResult{}, fmt.Errorf("mail ingest from_address is required")
	}

	messageID := strings.TrimSpace(input.MessageID)
	threadRef := firstNonEmpty(strings.TrimSpace(input.ThreadRef), strings.TrimSpace(input.InReplyTo), messageID, sourceEventID)
	toAddress := strings.TrimSpace(input.ToAddress)
	subject := strings.TrimSpace(input.Subject)
	bodyText := strings.TrimSpace(input.BodyText)
	occurredAt := normalizeOccurredAt(input.OccurredAt)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	threadID := deterministicMessagesID("thread", workspace, source, sourceScope, threadRef)
	eventID := deterministicMessagesID("event", workspace, source, sourceEventID)
	receiptID := deterministicMessagesID("receipt", workspace, source, sourceEventID)
	payloadHash := hashMessagesPayload(
		sourceScope,
		sourceEventID,
		messageID,
		threadRef,
		fromAddress,
		toAddress,
		subject,
		bodyText,
		occurredAt,
	)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("begin mail ingest tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return mailIngestPersistResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_ingest_receipts(
			id, workspace_id, source, source_scope, source_event_id, source_cursor,
			trust_state, event_id, payload_hash, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, 'accepted', NULL, ?, ?, ?)
	`, receiptID, workspace, source, sourceScope, sourceEventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, now); err != nil {
		if !isUniqueConstraintError(err) {
			return mailIngestPersistResult{}, fmt.Errorf("insert mail ingest receipt: %w", err)
		}
		existingEventID, existingThreadID, loadErr := loadExistingCommIngestReceipt(ctx, tx, workspace, source, sourceEventID)
		if loadErr != nil {
			return mailIngestPersistResult{}, loadErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return mailIngestPersistResult{}, fmt.Errorf("commit replay mail ingest tx: %w", commitErr)
		}
		return mailIngestPersistResult{
			EventID:  firstNonEmpty(existingEventID, eventID),
			ThreadID: firstNonEmpty(existingThreadID, threadID),
			Replayed: true,
		}, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		VALUES (?, ?, 'mail', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connector_id = excluded.connector_id,
			external_ref = excluded.external_ref,
			title = excluded.title,
			updated_at = excluded.updated_at
	`, threadID, workspace, mailConnectorID, nullableText(threadRef), nullableText(firstNonEmpty(subject, fromAddress, threadRef)), now, now); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("upsert mail thread: %w", err)
	}

	combinedBody := buildMailEventBody(subject, bodyText)
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
	`, eventID, workspace, threadID, mailConnectorID, occurredAt, nullableText(combinedBody), now); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("upsert mail comm event: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		VALUES (?, ?, 'FROM', ?, NULL, 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			event_id = excluded.event_id,
			address_role = excluded.address_role,
			address_value = excluded.address_value,
			position = excluded.position
	`, deterministicMessagesID("event-address", eventID, "from", fromAddress), eventID, fromAddress, now); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("upsert mail from address: %w", err)
	}
	if toAddress != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
			VALUES (?, ?, 'TO', ?, NULL, 0, ?)
			ON CONFLICT(id) DO UPDATE SET
				event_id = excluded.event_id,
				address_role = excluded.address_role,
				address_value = excluded.address_value,
				position = excluded.position
		`, deterministicMessagesID("event-address", eventID, "to", toAddress), eventID, toAddress, now); err != nil {
			return mailIngestPersistResult{}, fmt.Errorf("upsert mail to address: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO email_event_meta(
			id, event_id, message_id, in_reply_to, references_header, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(event_id) DO UPDATE SET
			message_id = excluded.message_id,
			in_reply_to = excluded.in_reply_to,
			references_header = excluded.references_header
	`, deterministicMessagesID("email-meta", eventID), eventID, nullableText(messageID), nullableText(strings.TrimSpace(input.InReplyTo)), nullableText(strings.TrimSpace(input.ReferencesHeader)), now); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("upsert email event metadata: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_ingest_receipts
		SET event_id = ?, source_cursor = ?, payload_hash = ?, received_at = ?
		WHERE workspace_id = ? AND source = ? AND source_event_id = ?
	`, eventID, nullableText(strings.TrimSpace(input.SourceCursor)), payloadHash, occurredAt, workspace, source, sourceEventID); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("update mail ingest receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return mailIngestPersistResult{}, fmt.Errorf("commit mail ingest tx: %w", err)
	}
	return mailIngestPersistResult{
		EventID:  eventID,
		ThreadID: threadID,
		Replayed: false,
	}, nil
}

func resolveMailSourceScope(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	return "mail-rule-default"
}

func resolveMailSourceCursor(raw string, occurredAt string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(occurredAt))
	if err != nil {
		return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}
	return strconv.FormatInt(parsed.UTC().UnixNano(), 10)
}

func buildMailEventBody(subject string, body string) string {
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	switch {
	case subject != "" && body != "":
		return subject + "\n\n" + body
	case subject != "":
		return subject
	default:
		return body
	}
}
