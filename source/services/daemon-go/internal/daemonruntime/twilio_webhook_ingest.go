package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
)

const (
	webhookSignatureModeStrict = "strict"
	webhookSignatureModeBypass = "bypass"
)

func IngestTwilioWebhookSMS(ctx context.Context, db *sql.DB, request TwilioWebhookSMSIngressRequest) TwilioWebhookSMSIngressResult {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	result := TwilioWebhookSMSIngressResult{
		WorkspaceID: workspace,
		MessageSID:  strings.TrimSpace(request.MessageSID),
		StatusCode:  http.StatusBadRequest,
	}

	if db == nil {
		result.Error = "worker db is not initialized"
		return result
	}

	messageSID := strings.TrimSpace(request.MessageSID)
	fromAddress := strings.TrimSpace(request.FromAddress)
	toAddress := strings.TrimSpace(request.ToAddress)
	bodyText := strings.TrimSpace(request.BodyText)
	if messageSID == "" || fromAddress == "" || toAddress == "" {
		result.Error = "missing required Twilio SMS fields (MessageSid/From/To)"
		return result
	}
	if configured := strings.TrimSpace(request.ConfiguredSMSNumber); configured != "" && !strings.EqualFold(configured, toAddress) {
		result.Error = fmt.Sprintf("webhook destination number %q does not match configured twilio sms number %q", toAddress, configured)
		return result
	}

	signatureMode := normalizeWebhookSignatureMode(request.SignatureMode)
	if signatureMode != webhookSignatureModeStrict && signatureMode != webhookSignatureModeBypass {
		result.Error = fmt.Sprintf("unsupported signature mode %q", request.SignatureMode)
		return result
	}
	if signatureMode == webhookSignatureModeStrict {
		requestURL := strings.TrimSpace(request.RequestURL)
		signature := strings.TrimSpace(request.SignatureValue)
		if requestURL == "" {
			result.Error = "request url is required for strict signature mode"
			return result
		}
		if signature == "" {
			result.Error = "signature is required for strict signature mode"
			return result
		}
		if err := twilioadapter.ValidateRequestSignature(strings.TrimSpace(request.AuthToken), requestURL, request.ProviderPayload, signature); err != nil {
			auditPayload := map[string]any{
				"provider":          "twilio",
				"channel":           "sms",
				"message_sid":       messageSID,
				"from":              fromAddress,
				"to":                toAddress,
				"signature_present": signature != "",
				"request_url":       requestURL,
				"error":             err.Error(),
			}
			_ = recordTwilioWebhookAudit(ctx, db, workspace, "twilio_webhook_rejected_invalid_signature", auditPayload)
			result.Error = err.Error()
			result.StatusCode = http.StatusUnauthorized
			return result
		}
	}

	persistence := twilioadapter.NewSMSPersistence(db)
	persisted, err := persistence.PersistInboundSMS(ctx, twilioadapter.InboundSMSInput{
		WorkspaceID:     workspace,
		ProviderEventID: messageSID,
		ProviderAccount: strings.TrimSpace(request.ProviderAccount),
		SignatureValue:  strings.TrimSpace(request.SignatureValue),
		FromAddress:     fromAddress,
		ToAddress:       toAddress,
		BodyText:        bodyText,
		ReceivedAt:      time.Now().UTC(),
		SignatureValid:  true,
		ProviderStatus:  "received",
		ProviderPayload: request.ProviderPayload,
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Accepted = true
	result.Replayed = persisted.Replayed
	result.EventID = persisted.EventID
	result.ThreadID = persisted.ThreadID
	result.StatusCode = http.StatusOK
	return result
}

func IngestTwilioWebhookVoice(ctx context.Context, db *sql.DB, request TwilioWebhookVoiceIngressRequest) TwilioWebhookVoiceIngressResult {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	callSID := strings.TrimSpace(request.CallSID)
	fromAddress := strings.TrimSpace(request.FromAddress)
	toAddress := strings.TrimSpace(request.ToAddress)
	direction := firstWebhookNonEmpty(strings.TrimSpace(request.Direction), "inbound")
	callStatus := firstWebhookNonEmpty(strings.TrimSpace(request.CallStatus), "initiated")
	transcript := strings.TrimSpace(request.TranscriptText)
	resolvedProviderEventID := strings.TrimSpace(request.ProviderEventID)
	if resolvedProviderEventID == "" {
		normalizedStatus := strings.ReplaceAll(strings.ToLower(callStatus), "-", "_")
		resolvedProviderEventID = fmt.Sprintf("voice:%s:%s", callSID, normalizedStatus)
		if transcript != "" {
			resolvedProviderEventID += ":transcript"
		}
	}

	result := TwilioWebhookVoiceIngressResult{
		WorkspaceID:     workspace,
		ProviderEventID: resolvedProviderEventID,
		CallSID:         callSID,
		StatusCode:      http.StatusBadRequest,
	}

	if db == nil {
		result.Error = "worker db is not initialized"
		return result
	}
	if callSID == "" || fromAddress == "" || toAddress == "" {
		result.Error = "missing required Twilio voice fields (CallSid/From/To)"
		return result
	}

	localValue := toAddress
	if strings.Contains(strings.ToLower(direction), "outbound") {
		localValue = fromAddress
	}
	if configured := strings.TrimSpace(request.ConfiguredVoiceNumber); configured != "" && !strings.EqualFold(configured, localValue) {
		result.Error = fmt.Sprintf("local call number %q does not match configured twilio voice number %q", localValue, configured)
		return result
	}

	signatureMode := normalizeWebhookSignatureMode(request.SignatureMode)
	if signatureMode != webhookSignatureModeStrict && signatureMode != webhookSignatureModeBypass {
		result.Error = fmt.Sprintf("unsupported signature mode %q", request.SignatureMode)
		return result
	}
	if signatureMode == webhookSignatureModeStrict {
		requestURL := strings.TrimSpace(request.RequestURL)
		signature := strings.TrimSpace(request.SignatureValue)
		if requestURL == "" {
			result.Error = "request url is required for strict signature mode"
			return result
		}
		if signature == "" {
			result.Error = "signature is required for strict signature mode"
			return result
		}
		if err := twilioadapter.ValidateRequestSignature(strings.TrimSpace(request.AuthToken), requestURL, request.ProviderPayload, signature); err != nil {
			auditPayload := map[string]any{
				"provider":          "twilio",
				"channel":           "voice",
				"provider_event_id": resolvedProviderEventID,
				"call_sid":          callSID,
				"from":              fromAddress,
				"to":                toAddress,
				"call_status":       callStatus,
				"signature_present": signature != "",
				"request_url":       requestURL,
				"error":             err.Error(),
			}
			_ = recordTwilioWebhookAudit(ctx, db, workspace, "twilio_webhook_rejected_invalid_signature", auditPayload)
			result.Error = err.Error()
			result.StatusCode = http.StatusUnauthorized
			return result
		}
	}

	persistence := twilioadapter.NewVoicePersistence(db)
	persisted, err := persistence.PersistInboundWebhook(ctx, twilioadapter.VoiceWebhookInput{
		WorkspaceID:               workspace,
		ProviderEventID:           resolvedProviderEventID,
		ProviderCallID:            callSID,
		ProviderAccount:           strings.TrimSpace(request.ProviderAccount),
		SignatureValue:            strings.TrimSpace(request.SignatureValue),
		FromAddress:               fromAddress,
		ToAddress:                 toAddress,
		Direction:                 direction,
		CallStatus:                callStatus,
		TranscriptText:            transcript,
		TranscriptDirection:       firstWebhookNonEmpty(strings.TrimSpace(request.TranscriptDirection), "INBOUND"),
		TranscriptAssistantEmited: request.TranscriptAssistantEmitted,
		ReceivedAt:                time.Now().UTC(),
		SignatureValid:            true,
		ProviderPayload:           request.ProviderPayload,
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Accepted = true
	result.Replayed = persisted.Replayed
	result.CallSessionID = persisted.CallSessionID
	result.ThreadID = persisted.ThreadID
	result.CallStatus = persisted.CallStatus
	result.StatusEventID = persisted.StatusEventID
	result.TranscriptEventID = persisted.TranscriptEventID
	result.StatusCode = http.StatusOK
	return result
}

func normalizeWebhookSignatureMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return webhookSignatureModeStrict
	}
	return normalized
}

func firstWebhookNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func recordTwilioWebhookAudit(ctx context.Context, db *sql.DB, workspaceID string, eventType string, payload map[string]any) error {
	workspace := normalizeWorkspaceID(workspaceID)
	if strings.TrimSpace(eventType) == "" {
		return fmt.Errorf("event type is required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	payloadJSON := ""
	if len(payload) > 0 {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal audit payload: %w", err)
		}
		payloadJSON = strings.TrimSpace(string(encoded))
	}
	auditID, err := randomWebhookAuditID()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin audit tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureCommWorkspace(ctx, tx, workspace, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id,
			correlation_id, payload_json, created_at
		) VALUES (?, ?, NULL, NULL, ?, NULL, NULL, NULL, ?, ?)
	`, auditID, workspace, strings.TrimSpace(eventType), nullableText(payloadJSON), now); err != nil {
		return fmt.Errorf("insert audit log entry: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit tx: %w", err)
	}
	return nil
}

func randomWebhookAuditID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
