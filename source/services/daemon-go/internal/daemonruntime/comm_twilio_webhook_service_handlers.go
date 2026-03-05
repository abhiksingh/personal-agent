package daemonruntime

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"personalagent/runtime/internal/channelconfig"
	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) handleTwilioWebhookSMS(
	writer http.ResponseWriter,
	request *http.Request,
	workspace string,
	config channelconfig.TwilioConfig,
	signatureMode string,
	authToken string,
	options twilioWebhookAssistantOptions,
) (twilioWebhookSMSResponse, int) {
	if request.Method != http.MethodPost {
		return twilioWebhookSMSResponse{
			WorkspaceID: workspace,
			Accepted:    false,
			Error:       "method not allowed",
		}, http.StatusMethodNotAllowed
	}
	if statusCode, err := parseTwilioWebhookForm(writer, request); err != nil {
		return twilioWebhookSMSResponse{
			WorkspaceID: workspace,
			Accepted:    false,
			Error:       fmt.Sprintf("parse form: %v", err),
		}, statusCode
	}

	formParams := webhookFormToMap(request.PostForm)
	ingestRequest := TwilioWebhookSMSIngressRequest{
		WorkspaceID:         workspace,
		SignatureMode:       signatureMode,
		AuthToken:           authToken,
		RequestURL:          resolveDaemonTwilioSignatureRequestURL(request),
		SignatureValue:      strings.TrimSpace(request.Header.Get("X-Twilio-Signature")),
		MessageSID:          firstNonEmpty(strings.TrimSpace(request.PostForm.Get("MessageSid")), strings.TrimSpace(request.PostForm.Get("SmsSid"))),
		ProviderAccount:     strings.TrimSpace(request.PostForm.Get("AccountSid")),
		FromAddress:         strings.TrimSpace(request.PostForm.Get("From")),
		ToAddress:           strings.TrimSpace(request.PostForm.Get("To")),
		BodyText:            strings.TrimSpace(request.PostForm.Get("Body")),
		ConfiguredSMSNumber: strings.TrimSpace(config.SMSNumber),
		ProviderPayload:     formParams,
	}

	result := IngestTwilioWebhookSMS(request.Context(), s.container.DB, ingestRequest)

	response := twilioWebhookSMSResponse{
		WorkspaceID: workspace,
		Accepted:    result.Accepted,
		Replayed:    result.Replayed,
		EventID:     result.EventID,
		ThreadID:    result.ThreadID,
		MessageSID:  result.MessageSID,
		Error:       result.Error,
	}
	statusCode := resolveWebhookStatusCode(result.StatusCode)
	if !response.Accepted {
		return response, statusCode
	}

	if options.Enabled && !response.Replayed && strings.TrimSpace(ingestRequest.BodyText) != "" {
		replyCtx, cancel := context.WithTimeout(request.Context(), options.ReplyTimeout)
		defer cancel()

		assistantReply, replyErr := s.generateThreadAssistantReply(
			replyCtx,
			workspace,
			"message",
			response.ThreadID,
			"twilio",
			options,
		)
		if replyErr != nil {
			response.AssistantError = replyErr.Error()
			return response, http.StatusOK
		}
		if strings.TrimSpace(assistantReply) == "" {
			response.AssistantError = "assistant reply was empty"
			return response, http.StatusOK
		}

		operationID := normalizeTwilioAssistantOperationID("twilio-sms-reply-" + response.MessageSID)
		deliveryResult, _, deliveryErr := s.executeTwilioSMSDelivery(replyCtx, workspace, ingestRequest.FromAddress, assistantReply, operationID)
		if deliveryErr != nil {
			response.AssistantReply = assistantReply
			response.AssistantOperationID = operationID
			response.AssistantError = deliveryErr.Error()
			return response, http.StatusOK
		}
		response.AssistantReply = assistantReply
		response.AssistantOperationID = operationID
		response.AssistantDelivered = deliveryResult.Delivered
		response.AssistantProviderReceipt = deliveryResult.ProviderReceipt
	}

	return response, http.StatusOK
}

func (s *CommTwilioService) handleTwilioWebhookVoice(
	writer http.ResponseWriter,
	request *http.Request,
	workspace string,
	config channelconfig.TwilioConfig,
	signatureMode string,
	authToken string,
	options twilioWebhookAssistantOptions,
	voiceResponseMode string,
	voiceGreeting string,
	voiceFallback string,
) (twilioWebhookVoiceResponse, int, string) {
	if request.Method != http.MethodPost {
		return twilioWebhookVoiceResponse{
			WorkspaceID: workspace,
			Accepted:    false,
			Error:       "method not allowed",
		}, http.StatusMethodNotAllowed, ""
	}
	if statusCode, err := parseTwilioWebhookForm(writer, request); err != nil {
		return twilioWebhookVoiceResponse{
			WorkspaceID: workspace,
			Accepted:    false,
			Error:       fmt.Sprintf("parse form: %v", err),
		}, statusCode, ""
	}

	formParams := webhookFormToMap(request.PostForm)
	ingestRequest := TwilioWebhookVoiceIngressRequest{
		WorkspaceID:                workspace,
		SignatureMode:              signatureMode,
		AuthToken:                  authToken,
		RequestURL:                 resolveDaemonTwilioSignatureRequestURL(request),
		SignatureValue:             strings.TrimSpace(request.Header.Get("X-Twilio-Signature")),
		ProviderEventID:            firstNonEmpty(strings.TrimSpace(request.PostForm.Get("ProviderEventId")), strings.TrimSpace(request.PostForm.Get("EventSid"))),
		CallSID:                    strings.TrimSpace(request.PostForm.Get("CallSid")),
		ProviderAccount:            strings.TrimSpace(request.PostForm.Get("AccountSid")),
		FromAddress:                strings.TrimSpace(request.PostForm.Get("From")),
		ToAddress:                  strings.TrimSpace(request.PostForm.Get("To")),
		Direction:                  firstNonEmpty(strings.TrimSpace(request.PostForm.Get("Direction")), "inbound"),
		CallStatus:                 firstNonEmpty(strings.TrimSpace(request.PostForm.Get("CallStatus")), "initiated"),
		TranscriptText:             strings.TrimSpace(request.PostForm.Get("SpeechResult")),
		TranscriptDirection:        firstNonEmpty(strings.TrimSpace(request.PostForm.Get("TranscriptDirection")), "INBOUND"),
		TranscriptAssistantEmitted: parseWebhookTruthy(strings.TrimSpace(request.PostForm.Get("AssistantEmitted"))),
		ConfiguredVoiceNumber:      strings.TrimSpace(config.VoiceNumber),
		ProviderPayload:            formParams,
	}

	result := IngestTwilioWebhookVoice(request.Context(), s.container.DB, ingestRequest)

	response := twilioWebhookVoiceResponse{
		WorkspaceID:       workspace,
		Accepted:          result.Accepted,
		Replayed:          result.Replayed,
		ProviderEventID:   result.ProviderEventID,
		CallSID:           result.CallSID,
		CallSessionID:     result.CallSessionID,
		ThreadID:          result.ThreadID,
		CallStatus:        result.CallStatus,
		StatusEventID:     result.StatusEventID,
		TranscriptEventID: result.TranscriptEventID,
		Error:             result.Error,
	}
	statusCode := resolveWebhookStatusCode(result.StatusCode)
	if statusCode >= 400 || voiceResponseMode != twilioWebhookVoiceResponseTwiML {
		return response, statusCode, ""
	}
	if isTerminalVoiceCallStatus(result.CallStatus) {
		return response, http.StatusOK, buildTwiMLEmptyResponse()
	}

	assistantReply := strings.TrimSpace(voiceGreeting)
	assistantReplySource := "greeting"
	trimmedTranscript := strings.TrimSpace(ingestRequest.TranscriptText)
	if trimmedTranscript != "" {
		assistantReplySource = "transcript_echo"
		assistantReply = trimmedTranscript
		if options.Enabled && !result.Replayed {
			replyCtx, cancel := context.WithTimeout(request.Context(), options.ReplyTimeout)
			defer cancel()
			resolvedReply, replyErr := s.generateThreadAssistantReply(
				replyCtx,
				workspace,
				"voice",
				result.ThreadID,
				"twilio",
				options,
			)
			if replyErr != nil {
				response.AssistantError = replyErr.Error()
			} else if strings.TrimSpace(resolvedReply) != "" {
				assistantReply = strings.TrimSpace(resolvedReply)
				assistantReplySource = "assistant_model"
			}
		}
	}
	if strings.TrimSpace(assistantReply) == "" {
		assistantReply = strings.TrimSpace(voiceFallback)
		assistantReplySource = "fallback"
	}
	if strings.TrimSpace(assistantReply) == "" {
		assistantReply = defaultTwilioWebhookVoiceFallback
		assistantReplySource = "fallback_default"
	}

	response.AssistantReply = assistantReply
	response.AssistantReplySource = assistantReplySource
	response.AssistantReplyTranscript = trimmedTranscript
	actionURL := resolveDaemonTwilioActionURL(request)
	return response, http.StatusOK, buildTwiMLGatherResponse(actionURL, assistantReply, strings.TrimSpace(voiceFallback))
}

func (s *CommTwilioService) generateThreadAssistantReply(
	ctx context.Context,
	workspace string,
	channelID string,
	threadID string,
	connectorID string,
	options twilioWebhookAssistantOptions,
) (string, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return "", fmt.Errorf("missing thread id for assistant reply generation")
	}

	queryLimit := maxInt(1, options.MaxHistory)
	rows, err := s.container.DB.QueryContext(ctx, `
		SELECT assistant_emitted, COALESCE(body_text, '')
		FROM comm_events
		WHERE workspace_id = ?
		  AND thread_id = ?
		  AND COALESCE(body_text, '') != ''
		ORDER BY occurred_at DESC, id DESC
		LIMIT ?
	`, workspace, trimmedThreadID, queryLimit)
	if err != nil {
		return "", fmt.Errorf("load thread conversation events: %w", err)
	}
	defer rows.Close()

	reversed := make([]transport.ChatTurnItem, 0, queryLimit)
	for rows.Next() {
		var (
			assistantEmitted int
			bodyText         string
		)
		if err := rows.Scan(&assistantEmitted, &bodyText); err != nil {
			return "", fmt.Errorf("scan thread conversation event: %w", err)
		}
		role := "user"
		if assistantEmitted == 1 {
			role = "assistant"
		}
		itemType := "user_message"
		if role == "assistant" {
			itemType = "assistant_message"
		}
		reversed = append(reversed, transport.ChatTurnItem{
			Type:    itemType,
			Role:    role,
			Status:  "completed",
			Content: strings.TrimSpace(bodyText),
		})
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate thread conversation events: %w", err)
	}
	if len(reversed) == 0 {
		return "", fmt.Errorf("no conversation events available for assistant reply generation")
	}

	items := make([]transport.ChatTurnItem, 0, len(reversed))
	for index := len(reversed) - 1; index >= 0; index-- {
		if strings.TrimSpace(reversed[index].Content) == "" {
			continue
		}
		items = append(items, reversed[index])
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no usable conversation events available for assistant reply generation")
	}

	chatService := s.assistantChat
	if chatService == nil {
		return "", fmt.Errorf("assistant chat service is not configured")
	}
	turn, err := chatService.ChatTurn(ctx, transport.ChatTurnRequest{
		WorkspaceID:  workspace,
		TaskClass:    normalizeTaskClass(options.TaskClass),
		SystemPrompt: options.SystemPrompt,
		Channel: transport.ChatTurnChannelContext{
			ChannelID:   strings.TrimSpace(channelID),
			ConnectorID: strings.TrimSpace(connectorID),
			ThreadID:    trimmedThreadID,
		},
		Items: items,
	}, "", nil)
	if err != nil {
		return "", fmt.Errorf("generate assistant reply: %w", err)
	}
	reply := strings.TrimSpace(assistantMessageFromTwilioTurnItems(turn.Items))
	if reply == "" {
		return "", fmt.Errorf("assistant reply generation returned empty content")
	}
	return reply, nil
}

func assistantMessageFromTwilioTurnItems(items []transport.ChatTurnItem) string {
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		if strings.ToLower(strings.TrimSpace(item.Type)) != "assistant_message" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content != "" {
			return content
		}
	}
	return ""
}
