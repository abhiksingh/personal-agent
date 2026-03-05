package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) IngestTwilioSMS(ctx context.Context, request transport.TwilioIngestSMSRequest) (transport.TwilioIngestSMSResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioIngestSMSResponse{}, err
	}

	signatureMode := twilioWebhookSignatureModeStrict
	if request.SkipSignature {
		signatureMode = twilioWebhookSignatureModeBypass
	}

	authToken := ""
	if signatureMode == twilioWebhookSignatureModeStrict {
		creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
		if err != nil {
			return transport.TwilioIngestSMSResponse{}, err
		}
		authToken = creds.AuthToken
	}

	providerPayload := map[string]string{
		"From":       strings.TrimSpace(request.FromAddress),
		"To":         strings.TrimSpace(request.ToAddress),
		"Body":       strings.TrimSpace(request.BodyText),
		"MessageSid": strings.TrimSpace(request.MessageSID),
	}
	if account := strings.TrimSpace(request.AccountSID); account != "" {
		providerPayload["AccountSid"] = account
	}

	ingestRequest := TwilioWebhookSMSIngressRequest{
		WorkspaceID:         workspace,
		SignatureMode:       signatureMode,
		AuthToken:           authToken,
		RequestURL:          strings.TrimSpace(request.RequestURL),
		SignatureValue:      strings.TrimSpace(request.Signature),
		MessageSID:          strings.TrimSpace(request.MessageSID),
		ProviderAccount:     strings.TrimSpace(request.AccountSID),
		FromAddress:         strings.TrimSpace(request.FromAddress),
		ToAddress:           strings.TrimSpace(request.ToAddress),
		BodyText:            strings.TrimSpace(request.BodyText),
		ConfiguredSMSNumber: strings.TrimSpace(config.SMSNumber),
		ProviderPayload:     providerPayload,
	}

	result := IngestTwilioWebhookSMS(ctx, s.container.DB, ingestRequest)
	s.evaluateAutomationForCommEvents(ctx, result.Accepted, result.Replayed, result.EventID)

	return transport.TwilioIngestSMSResponse{
		WorkspaceID: workspace,
		Accepted:    result.Accepted,
		Replayed:    result.Replayed,
		EventID:     result.EventID,
		ThreadID:    result.ThreadID,
		MessageSID:  firstNonEmpty(result.MessageSID, ingestRequest.MessageSID),
		Error:       result.Error,
	}, nil
}

func (s *CommTwilioService) IngestTwilioVoice(ctx context.Context, request transport.TwilioIngestVoiceRequest) (transport.TwilioIngestVoiceResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioIngestVoiceResponse{}, err
	}

	signatureMode := twilioWebhookSignatureModeStrict
	if request.SkipSignature {
		signatureMode = twilioWebhookSignatureModeBypass
	}

	authToken := ""
	if signatureMode == twilioWebhookSignatureModeStrict {
		creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
		if err != nil {
			return transport.TwilioIngestVoiceResponse{}, err
		}
		authToken = creds.AuthToken
	}

	callSID := strings.TrimSpace(request.CallSID)
	if callSID == "" {
		return transport.TwilioIngestVoiceResponse{}, fmt.Errorf("--call-sid is required")
	}
	normalizedStatus := strings.ToLower(strings.TrimSpace(request.CallStatus))
	if normalizedStatus == "" {
		normalizedStatus = "initiated"
	}
	normalizedStatus = strings.ReplaceAll(normalizedStatus, "-", "_")

	resolvedProviderEventID := strings.TrimSpace(request.ProviderEventID)
	if resolvedProviderEventID == "" {
		resolvedProviderEventID = fmt.Sprintf("voice:%s:%s", callSID, normalizedStatus)
		if strings.TrimSpace(request.Transcript) != "" {
			resolvedProviderEventID += ":transcript"
		}
	}

	providerPayload := map[string]string{
		"CallSid":    callSID,
		"CallStatus": strings.TrimSpace(request.CallStatus),
		"From":       strings.TrimSpace(request.FromAddress),
		"To":         strings.TrimSpace(request.ToAddress),
		"Direction":  strings.TrimSpace(request.Direction),
	}
	if account := strings.TrimSpace(request.AccountSID); account != "" {
		providerPayload["AccountSid"] = account
	}
	if transcript := strings.TrimSpace(request.Transcript); transcript != "" {
		providerPayload["SpeechResult"] = transcript
	}

	ingestRequest := TwilioWebhookVoiceIngressRequest{
		WorkspaceID:                workspace,
		SignatureMode:              signatureMode,
		AuthToken:                  authToken,
		RequestURL:                 strings.TrimSpace(request.RequestURL),
		SignatureValue:             strings.TrimSpace(request.Signature),
		ProviderEventID:            resolvedProviderEventID,
		CallSID:                    callSID,
		ProviderAccount:            strings.TrimSpace(request.AccountSID),
		FromAddress:                strings.TrimSpace(request.FromAddress),
		ToAddress:                  strings.TrimSpace(request.ToAddress),
		Direction:                  strings.TrimSpace(request.Direction),
		CallStatus:                 strings.TrimSpace(request.CallStatus),
		TranscriptText:             strings.TrimSpace(request.Transcript),
		TranscriptDirection:        strings.TrimSpace(request.TranscriptDirection),
		TranscriptAssistantEmitted: request.TranscriptAssistantEmitted,
		ConfiguredVoiceNumber:      strings.TrimSpace(config.VoiceNumber),
		ProviderPayload:            providerPayload,
	}

	result := IngestTwilioWebhookVoice(ctx, s.container.DB, ingestRequest)
	s.evaluateAutomationForCommEvents(ctx, result.Accepted, result.Replayed, result.TranscriptEventID, result.StatusEventID)

	return transport.TwilioIngestVoiceResponse{
		WorkspaceID:       workspace,
		Accepted:          result.Accepted,
		Replayed:          result.Replayed,
		ProviderEventID:   firstNonEmpty(result.ProviderEventID, ingestRequest.ProviderEventID),
		CallSID:           firstNonEmpty(result.CallSID, ingestRequest.CallSID),
		CallSessionID:     result.CallSessionID,
		ThreadID:          result.ThreadID,
		CallStatus:        result.CallStatus,
		StatusEventID:     result.StatusEventID,
		TranscriptEventID: result.TranscriptEventID,
		Error:             result.Error,
	}, nil
}

func (s *CommTwilioService) evaluateAutomationForCommEvents(ctx context.Context, accepted bool, replayed bool, eventIDs ...string) {
	if !accepted || replayed {
		return
	}
	if s.automationEval == nil {
		return
	}
	for _, eventID := range eventIDs {
		trimmed := strings.TrimSpace(eventID)
		if trimmed == "" {
			continue
		}
		_, _ = s.automationEval.EvaluateCommEvent(ctx, trimmed)
	}
}
