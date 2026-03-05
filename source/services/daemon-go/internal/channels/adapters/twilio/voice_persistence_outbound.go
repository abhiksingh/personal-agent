package twilio

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (p *VoicePersistence) PersistOutboundCall(ctx context.Context, input OutboundCallInput) (OutboundCallResult, error) {
	if p.db == nil {
		return OutboundCallResult{}, fmt.Errorf("db is required")
	}
	workspaceID := normalizeWorkspace(input.WorkspaceID)
	providerCallID := strings.TrimSpace(input.ProviderCallID)
	fromAddress := normalizeAddress(input.FromAddress)
	toAddress := normalizeAddress(input.ToAddress)
	callDirection := normalizeCallDirection(input.Direction)
	callStatus := normalizeVoiceCallStatus(input.CallStatus)
	if providerCallID == "" {
		return OutboundCallResult{}, fmt.Errorf("provider call id is required")
	}
	if fromAddress == "" {
		return OutboundCallResult{}, fmt.Errorf("from address is required")
	}
	if toAddress == "" {
		return OutboundCallResult{}, fmt.Errorf("to address is required")
	}
	if callStatus == "" {
		callStatus = "initiated"
	}

	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = p.now().UTC()
	}
	now := p.now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	occurredAtText := occurredAt.Format(time.RFC3339Nano)

	localAddress := fromAddress
	remoteAddress := toAddress
	if callDirection == "inbound" {
		localAddress = toAddress
		remoteAddress = fromAddress
	}
	threadExternalRef := voiceExternalRef(localAddress, remoteAddress)
	threadID := deterministicID("thread", workspaceID, providerNameTwilio, channelNameVoice, threadExternalRef)
	callSessionID := deterministicID("call-session", workspaceID, providerNameTwilio, providerCallID)
	statusEventID := deterministicID("voice-status-event", workspaceID, providerCallID, "outbound-start")
	payloadJSON := marshalPayloadAnyMap(input.ProviderPayload)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return OutboundCallResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, nowText); err != nil {
		return OutboundCallResult{}, err
	}
	if err := upsertThread(ctx, tx, threadID, workspaceID, channelNameVoice, connectorIDTwilio, threadExternalRef, remoteAddress, nowText); err != nil {
		return OutboundCallResult{}, err
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
		OccurredAt:      occurredAt,
		Now:             now,
	})
	if err != nil {
		return OutboundCallResult{}, err
	}

	statusDirection := callEventDirection(callDirection)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comm_events(
			id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at
		) VALUES (?, ?, ?, ?, 'VOICE_CALL_STATUS', ?, 1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			thread_id = excluded.thread_id,
			connector_id = excluded.connector_id,
			event_type = excluded.event_type,
			direction = excluded.direction,
			assistant_emitted = excluded.assistant_emitted,
			occurred_at = excluded.occurred_at,
			body_text = excluded.body_text
	`, statusEventID, workspaceID, threadID, connectorIDTwilio, statusDirection, occurredAtText, nullable(resolvedStatus), nowText); err != nil {
		return OutboundCallResult{}, fmt.Errorf("upsert outbound voice status event: %w", err)
	}
	if err := upsertEventAddresses(ctx, tx, statusEventID, fromAddress, toAddress, nowText); err != nil {
		return OutboundCallResult{}, err
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
	`, deterministicID("provider-message", workspaceID, providerNameTwilio, providerCallID+":start"), workspaceID, statusEventID, providerNameTwilio, providerCallID+":start", nullable(strings.TrimSpace(input.ProviderAccount)), channelNameVoice, statusDirection, fromAddress, toAddress, nullable(resolvedStatus), nullable(payloadJSON), nowText, nowText); err != nil {
		return OutboundCallResult{}, fmt.Errorf("upsert outbound voice provider message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return OutboundCallResult{}, fmt.Errorf("commit outbound voice tx: %w", err)
	}
	return OutboundCallResult{
		CallSessionID: callSessionID,
		ThreadID:      threadID,
		CallStatus:    resolvedStatus,
		StatusEventID: statusEventID,
	}, nil
}
