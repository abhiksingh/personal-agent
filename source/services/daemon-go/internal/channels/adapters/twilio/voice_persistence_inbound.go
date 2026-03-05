package twilio

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (p *VoicePersistence) PersistInboundWebhook(ctx context.Context, input VoiceWebhookInput) (VoiceWebhookResult, error) {
	if p.db == nil {
		return VoiceWebhookResult{}, fmt.Errorf("db is required")
	}
	workspaceID := normalizeWorkspace(input.WorkspaceID)
	providerEventID := strings.TrimSpace(input.ProviderEventID)
	providerCallID := strings.TrimSpace(input.ProviderCallID)
	fromAddress := normalizeAddress(input.FromAddress)
	toAddress := normalizeAddress(input.ToAddress)
	callDirection := normalizeCallDirection(input.Direction)
	callStatus := normalizeVoiceCallStatus(input.CallStatus)
	if providerEventID == "" {
		return VoiceWebhookResult{}, fmt.Errorf("provider event id is required")
	}
	if providerCallID == "" {
		return VoiceWebhookResult{}, fmt.Errorf("provider call id is required")
	}
	if fromAddress == "" {
		return VoiceWebhookResult{}, fmt.Errorf("from address is required")
	}
	if toAddress == "" {
		return VoiceWebhookResult{}, fmt.Errorf("to address is required")
	}
	if callStatus == "" {
		return VoiceWebhookResult{}, fmt.Errorf("call status is required")
	}
	if !input.SignatureValid {
		return VoiceWebhookResult{}, fmt.Errorf("signature must be validated before persisting inbound voice webhook")
	}

	receivedAt := input.ReceivedAt.UTC()
	if receivedAt.IsZero() {
		receivedAt = p.now().UTC()
	}
	now := p.now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	receivedAtText := receivedAt.Format(time.RFC3339Nano)

	localAddress := toAddress
	remoteAddress := fromAddress
	if callDirection == "outbound" {
		localAddress = fromAddress
		remoteAddress = toAddress
	}
	threadExternalRef := voiceExternalRef(localAddress, remoteAddress)
	threadID := deterministicID("thread", workspaceID, providerNameTwilio, channelNameVoice, threadExternalRef)
	callSessionID := deterministicID("call-session", workspaceID, providerNameTwilio, providerCallID)
	receiptID := deterministicID("voice-webhook", workspaceID, providerNameTwilio, providerEventID)
	statusEventID := deterministicID("voice-status-event", workspaceID, providerCallID, providerEventID)
	transcriptEventID := ""
	transcriptText := strings.TrimSpace(input.TranscriptText)
	if transcriptText != "" {
		transcriptEventID = deterministicID("voice-transcript-event", workspaceID, providerCallID, providerEventID)
	}
	payloadJSON := marshalPayloadStringMap(input.ProviderPayload)
	payloadHash := hashPayload(input.ProviderPayload, payloadJSON)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return VoiceWebhookResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, nowText); err != nil {
		return VoiceWebhookResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_webhook_receipts(
			id, workspace_id, provider, provider_event_id, signature_valid, signature_value,
			payload_hash, event_id, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`, receiptID, workspaceID, providerNameTwilio, providerEventID, boolToInt(input.SignatureValid), strings.TrimSpace(input.SignatureValue), payloadHash, receivedAtText, nowText); err != nil {
		if !isUniqueConstraintError(err) {
			return VoiceWebhookResult{}, fmt.Errorf("insert voice webhook receipt: %w", err)
		}
		existing, loadErr := loadExistingReceipt(ctx, tx, workspaceID, providerNameTwilio, providerEventID)
		if loadErr != nil {
			return VoiceWebhookResult{}, loadErr
		}
		replayedThreadID := threadID
		if existingSessionThreadID, loadSessionErr := loadCallSessionThreadID(ctx, tx, workspaceID, providerCallID); loadSessionErr == nil && existingSessionThreadID != "" {
			replayedThreadID = existingSessionThreadID
		}
		if err := tx.Commit(); err != nil {
			return VoiceWebhookResult{}, fmt.Errorf("commit replay tx: %w", err)
		}
		return VoiceWebhookResult{
			ReceiptID:         existing.ReceiptID,
			CallSessionID:     callSessionID,
			ThreadID:          replayedThreadID,
			CallStatus:        callStatus,
			StatusEventID:     existing.EventID,
			TranscriptEventID: "",
			Replayed:          true,
		}, nil
	}

	if err := upsertThread(ctx, tx, threadID, workspaceID, channelNameVoice, connectorIDTwilio, threadExternalRef, remoteAddress, nowText); err != nil {
		return VoiceWebhookResult{}, err
	}

	resolvedStatus, err := upsertCallSession(ctx, tx, callSessionUpsertInput{
		WorkspaceID:     workspaceID,
		SessionID:       callSessionID,
		Provider:        providerNameTwilio,
		ConnectorID:     connectorIDTwilio,
		ProviderCallID:  providerCallID,
		ProviderAccount: strings.TrimSpace(input.ProviderAccount),
		ThreadID:        threadID,
		Direction:       callDirection,
		FromAddress:     fromAddress,
		ToAddress:       toAddress,
		Status:          callStatus,
		OccurredAt:      receivedAt,
		Now:             now,
	})
	if err != nil {
		return VoiceWebhookResult{}, err
	}

	statusDirection := callEventDirection(callDirection)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(
			id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
		) VALUES (?, ?, ?, ?, 'VOICE_CALL_STATUS', ?, 0, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, statusEventID, workspaceID, threadID, connectorIDTwilio, statusDirection, receivedAtText, nullable(resolvedStatus), nowText); err != nil {
		return VoiceWebhookResult{}, fmt.Errorf("upsert voice status event: %w", err)
	}
	if err := upsertEventAddresses(ctx, tx, statusEventID, fromAddress, toAddress, nowText); err != nil {
		return VoiceWebhookResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_provider_messages(
			id, workspace_id, event_id, provider, provider_message_id, provider_account_id, channel, direction,
			from_address, to_address, status, payload_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	`, deterministicID("provider-message", workspaceID, providerNameTwilio, providerEventID+":status"), workspaceID, statusEventID, providerNameTwilio, providerEventID+":status", nullable(strings.TrimSpace(input.ProviderAccount)), channelNameVoice, statusDirection, fromAddress, toAddress, nullable(resolvedStatus), nullable(payloadJSON), nowText, nowText); err != nil {
		return VoiceWebhookResult{}, fmt.Errorf("upsert voice status provider message: %w", err)
	}

	primaryEventID := statusEventID
	if transcriptText != "" {
		transcriptDirection := normalizeEventDirection(input.TranscriptDirection)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_events(
				id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
			) VALUES (?, ?, ?, ?, 'VOICE_TRANSCRIPT', ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				workspace_id = excluded.workspace_id,
				thread_id = excluded.thread_id,
				connector_id = excluded.connector_id,
				event_type = excluded.event_type,
				direction = excluded.direction,
				assistant_emitted = excluded.assistant_emitted,
				occurred_at = excluded.occurred_at,
				body_text = excluded.body_text
		`, transcriptEventID, workspaceID, threadID, connectorIDTwilio, transcriptDirection, boolToInt(input.TranscriptAssistantEmited), receivedAtText, nullable(transcriptText), nowText); err != nil {
			return VoiceWebhookResult{}, fmt.Errorf("upsert voice transcript event: %w", err)
		}
		if err := upsertEventAddresses(ctx, tx, transcriptEventID, fromAddress, toAddress, nowText); err != nil {
			return VoiceWebhookResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO comm_provider_messages(
				id, workspace_id, event_id, provider, provider_message_id, provider_account_id, channel, direction,
				from_address, to_address, status, payload_json, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		`, deterministicID("provider-message", workspaceID, providerNameTwilio, providerEventID+":transcript"), workspaceID, transcriptEventID, providerNameTwilio, providerEventID+":transcript", nullable(strings.TrimSpace(input.ProviderAccount)), channelNameVoice, transcriptDirection, fromAddress, toAddress, nullable(resolvedStatus), nullable(payloadJSON), nowText, nowText); err != nil {
			return VoiceWebhookResult{}, fmt.Errorf("upsert voice transcript provider message: %w", err)
		}
		primaryEventID = transcriptEventID
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comm_webhook_receipts
		SET event_id = ?, signature_valid = ?, signature_value = ?, payload_hash = ?
		WHERE workspace_id = ? AND provider = ? AND provider_event_id = ?
	`, primaryEventID, boolToInt(input.SignatureValid), strings.TrimSpace(input.SignatureValue), payloadHash, workspaceID, providerNameTwilio, providerEventID); err != nil {
		return VoiceWebhookResult{}, fmt.Errorf("update voice webhook receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return VoiceWebhookResult{}, fmt.Errorf("commit voice webhook tx: %w", err)
	}
	return VoiceWebhookResult{
		ReceiptID:         receiptID,
		CallSessionID:     callSessionID,
		ThreadID:          threadID,
		CallStatus:        resolvedStatus,
		StatusEventID:     statusEventID,
		TranscriptEventID: transcriptEventID,
		Replayed:          false,
	}, nil
}
