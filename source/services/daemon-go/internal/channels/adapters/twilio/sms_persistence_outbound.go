package twilio

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (p *SMSPersistence) PersistOutboundSMS(ctx context.Context, input OutboundSMSInput) (OutboundSMSResult, error) {
	if p.db == nil {
		return OutboundSMSResult{}, fmt.Errorf("db is required")
	}
	workspaceID := normalizeWorkspace(input.WorkspaceID)
	providerMessageID := strings.TrimSpace(input.ProviderMessage)
	fromAddress := normalizeAddress(input.FromAddress)
	toAddress := normalizeAddress(input.ToAddress)
	if providerMessageID == "" {
		return OutboundSMSResult{}, fmt.Errorf("provider message id is required")
	}
	if fromAddress == "" {
		return OutboundSMSResult{}, fmt.Errorf("from address is required")
	}
	if toAddress == "" {
		return OutboundSMSResult{}, fmt.Errorf("to address is required")
	}

	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = p.now().UTC()
	}
	now := p.now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	occurredAtText := occurredAt.Format(time.RFC3339Nano)

	threadExternalRef := smsExternalRef(fromAddress, toAddress)
	threadID := deterministicID("thread", workspaceID, providerNameTwilio, channelNameSMS, threadExternalRef)
	eventID := deterministicID("event", workspaceID, providerNameTwilio, channelNameSMS, "outbound", providerMessageID)
	payloadJSON := marshalPayloadAnyMap(input.ProviderPayload)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return OutboundSMSResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, nowText); err != nil {
		return OutboundSMSResult{}, err
	}
	if err := upsertThread(ctx, tx, threadID, workspaceID, "message", connectorIDTwilio, threadExternalRef, toAddress, nowText); err != nil {
		return OutboundSMSResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(
			id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
		) VALUES (?, ?, ?, ?, 'MESSAGE', 'OUTBOUND', 1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, eventID, workspaceID, threadID, connectorIDTwilio, occurredAtText, nullable(strings.TrimSpace(input.BodyText)), nowText); err != nil {
		return OutboundSMSResult{}, fmt.Errorf("upsert outbound comm event: %w", err)
	}

	if err := upsertEventAddresses(ctx, tx, eventID, fromAddress, toAddress, nowText); err != nil {
		return OutboundSMSResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_provider_messages(
			id, workspace_id, event_id, provider, provider_message_id, provider_account_id, channel, direction,
			from_address, to_address, status, payload_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'OUTBOUND', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, provider, provider_message_id) DO UPDATE SET
			event_id = excluded.event_id,
			provider_account_id = excluded.provider_account_id,
			channel = excluded.channel,
			direction = excluded.direction,
			from_address = excluded.from_address,
			to_address = excluded.to_address,
			status = excluded.status,
			payload_json = excluded.payload_json,
			updated_at = excluded.updated_at
	`, deterministicID("provider-message", workspaceID, providerNameTwilio, providerMessageID), workspaceID, eventID, providerNameTwilio, providerMessageID, nullable(strings.TrimSpace(input.ProviderAccount)), channelNameSMS, fromAddress, toAddress, nullable(strings.TrimSpace(input.ProviderStatus)), nullable(payloadJSON), nowText, nowText); err != nil {
		return OutboundSMSResult{}, fmt.Errorf("upsert outbound provider message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return OutboundSMSResult{}, fmt.Errorf("commit outbound sms tx: %w", err)
	}
	return OutboundSMSResult{EventID: eventID, ThreadID: threadID}, nil
}
