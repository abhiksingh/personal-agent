package twilio

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (p *SMSPersistence) PersistInboundSMS(ctx context.Context, input InboundSMSInput) (InboundSMSResult, error) {
	if p.db == nil {
		return InboundSMSResult{}, fmt.Errorf("db is required")
	}
	workspaceID := normalizeWorkspace(input.WorkspaceID)
	providerEventID := strings.TrimSpace(input.ProviderEventID)
	fromAddress := normalizeAddress(input.FromAddress)
	toAddress := normalizeAddress(input.ToAddress)
	bodyText := strings.TrimSpace(input.BodyText)
	if providerEventID == "" {
		return InboundSMSResult{}, fmt.Errorf("provider event id is required")
	}
	if fromAddress == "" {
		return InboundSMSResult{}, fmt.Errorf("from address is required")
	}
	if toAddress == "" {
		return InboundSMSResult{}, fmt.Errorf("to address is required")
	}
	if !input.SignatureValid {
		return InboundSMSResult{}, fmt.Errorf("signature must be validated before persisting inbound sms")
	}

	receivedAt := input.ReceivedAt.UTC()
	if receivedAt.IsZero() {
		receivedAt = p.now().UTC()
	}
	now := p.now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	receivedAtText := receivedAt.Format(time.RFC3339Nano)

	threadExternalRef := smsExternalRef(toAddress, fromAddress)
	threadID := deterministicID("thread", workspaceID, providerNameTwilio, channelNameSMS, threadExternalRef)
	eventID := deterministicID("event", workspaceID, providerNameTwilio, channelNameSMS, "inbound", providerEventID)
	receiptID := deterministicID("webhook", workspaceID, providerNameTwilio, providerEventID)
	payloadHash := hashPayload(input.ProviderPayload, input.ProviderPayloadS)
	payloadJSON := marshalPayloadStringMap(input.ProviderPayload)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return InboundSMSResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, nowText); err != nil {
		return InboundSMSResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_webhook_receipts(
			id, workspace_id, provider, provider_event_id, signature_valid, signature_value,
			payload_hash, event_id, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`, receiptID, workspaceID, providerNameTwilio, providerEventID, boolToInt(input.SignatureValid), strings.TrimSpace(input.SignatureValue), payloadHash, receivedAtText, nowText); err != nil {
		if !isUniqueConstraintError(err) {
			return InboundSMSResult{}, fmt.Errorf("insert webhook receipt: %w", err)
		}
		existing, loadErr := loadExistingReceipt(ctx, tx, workspaceID, providerNameTwilio, providerEventID)
		if loadErr != nil {
			return InboundSMSResult{}, loadErr
		}
		replayedThreadID := threadID
		if strings.TrimSpace(existing.EventID) != "" {
			if loadedThreadID, loadThreadErr := loadEventThreadID(ctx, tx, existing.EventID); loadThreadErr == nil && strings.TrimSpace(loadedThreadID) != "" {
				replayedThreadID = loadedThreadID
			}
		}
		if err := tx.Commit(); err != nil {
			return InboundSMSResult{}, fmt.Errorf("commit replay tx: %w", err)
		}
		return InboundSMSResult{
			ReceiptID: existing.ReceiptID,
			EventID:   existing.EventID,
			ThreadID:  replayedThreadID,
			Replayed:  true,
		}, nil
	}

	if err := upsertThread(ctx, tx, threadID, workspaceID, "message", connectorIDTwilio, threadExternalRef, fromAddress, nowText); err != nil {
		return InboundSMSResult{}, err
	}

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
	`, eventID, workspaceID, threadID, connectorIDTwilio, receivedAtText, nullable(bodyText), nowText); err != nil {
		return InboundSMSResult{}, fmt.Errorf("upsert inbound comm event: %w", err)
	}

	if err := upsertEventAddresses(ctx, tx, eventID, fromAddress, toAddress, nowText); err != nil {
		return InboundSMSResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_provider_messages(
			id, workspace_id, event_id, provider, provider_message_id, provider_account_id, channel, direction,
			from_address, to_address, status, payload_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'INBOUND', ?, ?, ?, ?, ?, ?)
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
	`, deterministicID("provider-message", workspaceID, providerNameTwilio, providerEventID), workspaceID, eventID, providerNameTwilio, providerEventID, nullable(strings.TrimSpace(input.ProviderAccount)), channelNameSMS, fromAddress, toAddress, nullable(strings.TrimSpace(input.ProviderStatus)), nullable(payloadJSON), nowText, nowText); err != nil {
		return InboundSMSResult{}, fmt.Errorf("upsert inbound provider message: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_webhook_receipts
		SET event_id = ?, signature_valid = ?, signature_value = ?, payload_hash = ?
		WHERE workspace_id = ? AND provider = ? AND provider_event_id = ?
	`, eventID, boolToInt(input.SignatureValid), strings.TrimSpace(input.SignatureValue), payloadHash, workspaceID, providerNameTwilio, providerEventID); err != nil {
		return InboundSMSResult{}, fmt.Errorf("update webhook receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return InboundSMSResult{}, fmt.Errorf("commit inbound sms tx: %w", err)
	}
	return InboundSMSResult{
		ReceiptID: receiptID,
		EventID:   eventID,
		ThreadID:  threadID,
		Replayed:  false,
	}, nil
}
