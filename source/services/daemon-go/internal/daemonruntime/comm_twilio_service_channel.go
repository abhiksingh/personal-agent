package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/channelcheck"
	"personalagent/runtime/internal/channelconfig"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	repodelivery "personalagent/runtime/internal/core/repository/delivery"
	deliveryservice "personalagent/runtime/internal/core/service/delivery"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) SetTwilioChannel(ctx context.Context, request transport.TwilioSetRequest) (transport.TwilioConfigRecord, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	accountSecret := strings.TrimSpace(request.AccountSIDSecretName)
	authSecret := strings.TrimSpace(request.AuthTokenSecretName)
	if accountSecret == "" || authSecret == "" {
		return transport.TwilioConfigRecord{}, fmt.Errorf("--account-sid-secret and --auth-token-secret are required")
	}

	accountRef, _, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, accountSecret)
	if err != nil {
		return transport.TwilioConfigRecord{}, fmt.Errorf("resolve twilio account sid secret %q: %w", accountSecret, err)
	}
	authRef, _, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, authSecret)
	if err != nil {
		return transport.TwilioConfigRecord{}, fmt.Errorf("resolve twilio auth token secret %q: %w", authSecret, err)
	}

	config, err := s.twilioStore.Upsert(ctx, channelconfig.TwilioUpsertInput{
		WorkspaceID:               workspace,
		AccountSIDSecretName:      accountSecret,
		AuthTokenSecretName:       authSecret,
		AccountSIDKeychainService: accountRef.Service,
		AccountSIDKeychainAccount: accountRef.Account,
		AuthTokenKeychainService:  authRef.Service,
		AuthTokenKeychainAccount:  authRef.Account,
		SMSNumber:                 strings.TrimSpace(request.SMSNumber),
		VoiceNumber:               strings.TrimSpace(request.VoiceNumber),
		Endpoint:                  strings.TrimSpace(request.Endpoint),
	})
	if err != nil {
		return transport.TwilioConfigRecord{}, err
	}
	return twilioConfigRecord(config), nil
}

func (s *CommTwilioService) GetTwilioChannel(ctx context.Context, request transport.TwilioGetRequest) (transport.TwilioConfigRecord, error) {
	config, err := s.twilioStore.Get(ctx, normalizeWorkspaceID(request.WorkspaceID))
	if err != nil {
		return transport.TwilioConfigRecord{}, err
	}
	return twilioConfigRecord(config), nil
}

func (s *CommTwilioService) CheckTwilioChannel(ctx context.Context, request transport.TwilioCheckRequest) (transport.TwilioCheckResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioCheckResponse{}, err
	}

	creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
	if err != nil {
		return transport.TwilioCheckResponse{}, err
	}

	result, checkErr := s.channelDispatch.CheckTwilio(ctx, channelcheck.TwilioRequest{
		Endpoint:   creds.Config.Endpoint,
		AccountSID: creds.AccountSID,
		AuthToken:  creds.AuthToken,
	})
	response := transport.TwilioCheckResponse{
		WorkspaceID: workspace,
		Success:     checkErr == nil,
		Config:      twilioConfigRecord(creds.Config),
		Result: transport.TwilioCheckResult{
			Endpoint:   result.Endpoint,
			StatusCode: result.StatusCode,
			LatencyMS:  result.LatencyMS,
			Message:    result.Message,
		},
	}
	if checkErr != nil {
		response.Error = checkErr.Error()
	}
	return response, nil
}

func (s *CommTwilioService) ExecuteTwilioSMSChatTurn(ctx context.Context, request transport.TwilioSMSChatTurnRequest) (transport.TwilioSMSChatTurn, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	to := strings.TrimSpace(request.To)
	if to == "" {
		return transport.TwilioSMSChatTurn{}, fmt.Errorf("--to is required")
	}
	message := strings.TrimSpace(request.Message)
	if message == "" {
		return transport.TwilioSMSChatTurn{}, fmt.Errorf("--message is required")
	}

	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		resolved, err := daemonRandomID()
		if err != nil {
			return transport.TwilioSMSChatTurn{}, err
		}
		operationID = resolved
	}

	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioSMSChatTurn{}, err
	}
	creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
	if err != nil {
		return transport.TwilioSMSChatTurn{}, err
	}
	localSMSNumber := strings.TrimSpace(config.SMSNumber)
	if localSMSNumber == "" {
		return transport.TwilioSMSChatTurn{}, fmt.Errorf("twilio sms number is not configured")
	}

	inboundMessageSID := normalizeTwilioAssistantOperationID("twilio-direct-sms-inbound-" + operationID)
	ingestRequest := TwilioWebhookSMSIngressRequest{
		WorkspaceID:         workspace,
		SignatureMode:       twilioWebhookSignatureModeBypass,
		MessageSID:          inboundMessageSID,
		ProviderAccount:     strings.TrimSpace(creds.AccountSID),
		FromAddress:         to,
		ToAddress:           localSMSNumber,
		BodyText:            message,
		ConfiguredSMSNumber: localSMSNumber,
		ProviderPayload: map[string]string{
			"From":       to,
			"To":         localSMSNumber,
			"Body":       message,
			"MessageSid": inboundMessageSID,
			"AccountSid": strings.TrimSpace(creds.AccountSID),
		},
	}
	ingestResult := IngestTwilioWebhookSMS(ctx, s.container.DB, ingestRequest)
	s.evaluateAutomationForCommEvents(ctx, ingestResult.Accepted, ingestResult.Replayed, ingestResult.EventID)

	turn := transport.TwilioSMSChatTurn{
		OperationID:      operationID,
		Message:          message,
		Success:          ingestResult.Accepted,
		IdempotentReplay: ingestResult.Replayed,
		ThreadID:         ingestResult.ThreadID,
	}
	if !ingestResult.Accepted {
		turn.Error = ingestResult.Error
		return turn, nil
	}
	if ingestResult.Replayed {
		return turn, nil
	}

	replyOptions := twilioWebhookAssistantOptions{
		Enabled:      true,
		TaskClass:    normalizeTaskClass(request.TaskClass),
		SystemPrompt: strings.TrimSpace(request.SystemPrompt),
		MaxHistory:   request.MaxHistory,
		ReplyTimeout: time.Duration(request.ReplyTimeoutMS) * time.Millisecond,
	}
	if replyOptions.TaskClass == "" {
		replyOptions.TaskClass = "chat"
	}
	if replyOptions.MaxHistory <= 0 {
		replyOptions.MaxHistory = 20
	}
	if replyOptions.ReplyTimeout <= 0 {
		replyOptions.ReplyTimeout = 12 * time.Second
	}

	replyCtx, cancel := context.WithTimeout(ctx, replyOptions.ReplyTimeout)
	defer cancel()
	assistantReply, replyErr := s.generateThreadAssistantReply(
		replyCtx,
		workspace,
		"message",
		ingestResult.ThreadID,
		"twilio",
		replyOptions,
	)
	if replyErr != nil {
		turn.AssistantError = replyErr.Error()
		return turn, nil
	}
	if strings.TrimSpace(assistantReply) == "" {
		turn.AssistantError = "assistant reply was empty"
		return turn, nil
	}

	assistantOperationID := normalizeTwilioAssistantOperationID("twilio-direct-sms-reply-" + operationID)
	deliveryResult, threadID, deliverErr := s.executeTwilioSMSDelivery(replyCtx, workspace, to, assistantReply, assistantOperationID)
	turn.AssistantReply = assistantReply
	turn.AssistantOperationID = assistantOperationID
	turn.Delivered = deliveryResult.Delivered
	turn.Channel = deliveryResult.Channel
	turn.ProviderReceipt = deliveryResult.ProviderReceipt
	turn.IdempotentReplay = turn.IdempotentReplay || deliveryResult.IdempotentReplay
	if strings.TrimSpace(threadID) != "" {
		turn.ThreadID = threadID
	}
	if deliverErr != nil {
		turn.AssistantError = deliverErr.Error()
		return turn, nil
	}
	return turn, nil
}

func (s *CommTwilioService) StartTwilioCall(ctx context.Context, request transport.TwilioStartCallRequest) (transport.TwilioStartCallResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	to := strings.TrimSpace(request.To)
	if to == "" {
		return transport.TwilioStartCallResponse{}, fmt.Errorf("--to is required")
	}
	if strings.TrimSpace(request.TwimlURL) == "" {
		return transport.TwilioStartCallResponse{}, fmt.Errorf("--twiml-url is required")
	}

	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioStartCallResponse{}, err
	}
	creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
	if err != nil {
		return transport.TwilioStartCallResponse{}, err
	}

	fromValue := strings.TrimSpace(request.From)
	if fromValue == "" {
		fromValue = strings.TrimSpace(config.VoiceNumber)
	}
	if fromValue == "" {
		return transport.TwilioStartCallResponse{}, fmt.Errorf("--from is required when no voice number is configured")
	}

	callResponse, err := s.channelDispatch.StartTwilioVoiceCall(ctx, twilioadapter.VoiceCallRequest{
		Endpoint:   config.Endpoint,
		AccountSID: creds.AccountSID,
		AuthToken:  creds.AuthToken,
		From:       fromValue,
		To:         to,
		TwiMLURL:   strings.TrimSpace(request.TwimlURL),
	})
	if err != nil {
		return transport.TwilioStartCallResponse{}, err
	}

	persistence := twilioadapter.NewVoicePersistence(s.container.DB)
	persisted, err := persistence.PersistOutboundCall(ctx, twilioadapter.OutboundCallInput{
		WorkspaceID:     workspace,
		ProviderCallID:  callResponse.CallSID,
		ProviderAccount: firstNonEmpty(strings.TrimSpace(callResponse.AccountSID), strings.TrimSpace(creds.AccountSID)),
		FromAddress:     firstNonEmpty(strings.TrimSpace(callResponse.From), fromValue),
		ToAddress:       firstNonEmpty(strings.TrimSpace(callResponse.To), to),
		Direction:       firstNonEmpty(strings.TrimSpace(callResponse.Direction), "outbound"),
		CallStatus:      firstNonEmpty(strings.TrimSpace(callResponse.Status), "initiated"),
		ProviderPayload: map[string]any{
			"sid":           callResponse.CallSID,
			"account_sid":   callResponse.AccountSID,
			"status":        callResponse.Status,
			"direction":     callResponse.Direction,
			"from":          callResponse.From,
			"to":            callResponse.To,
			"error_code":    callResponse.ErrorCode,
			"error_message": callResponse.ErrorMessage,
			"status_code":   callResponse.StatusCode,
		},
	})
	if err != nil {
		return transport.TwilioStartCallResponse{}, err
	}

	return transport.TwilioStartCallResponse{
		WorkspaceID:   workspace,
		CallSID:       callResponse.CallSID,
		CallSessionID: persisted.CallSessionID,
		ThreadID:      persisted.ThreadID,
		Status:        persisted.CallStatus,
		Direction:     firstNonEmpty(strings.TrimSpace(callResponse.Direction), "outbound"),
	}, nil
}

func (s *CommTwilioService) ListTwilioCallStatus(ctx context.Context, request transport.TwilioCallStatusRequest) (transport.TwilioCallStatusResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := request.Limit
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT id, workspace_id, provider, provider_call_id, thread_id, direction,
		       COALESCE(from_address, ''), COALESCE(to_address, ''), status,
		       COALESCE(started_at, ''), COALESCE(ended_at, ''), updated_at
		FROM comm_call_sessions
		WHERE workspace_id = ?
	`
	params := []any{workspace}
	if strings.TrimSpace(request.CallSID) != "" {
		query += " AND provider_call_id = ?"
		params = append(params, strings.TrimSpace(request.CallSID))
	}
	query += " ORDER BY updated_at DESC LIMIT ?"
	params = append(params, maxInt(1, limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.TwilioCallStatusResponse{}, fmt.Errorf("query call sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]transport.TwilioCallStatusRecord, 0)
	for rows.Next() {
		var item transport.TwilioCallStatusRecord
		if err := rows.Scan(
			&item.SessionID,
			&item.WorkspaceID,
			&item.Provider,
			&item.ProviderCallID,
			&item.ThreadID,
			&item.Direction,
			&item.FromAddress,
			&item.ToAddress,
			&item.Status,
			&item.StartedAt,
			&item.EndedAt,
			&item.UpdatedAt,
		); err != nil {
			return transport.TwilioCallStatusResponse{}, fmt.Errorf("scan call session: %w", err)
		}
		sessions = append(sessions, item)
	}
	if err := rows.Err(); err != nil {
		return transport.TwilioCallStatusResponse{}, fmt.Errorf("iterate call sessions: %w", err)
	}

	return transport.TwilioCallStatusResponse{
		WorkspaceID: workspace,
		CallSID:     strings.TrimSpace(request.CallSID),
		Sessions:    sessions,
	}, nil
}

func (s *CommTwilioService) ListTwilioTranscript(ctx context.Context, request transport.TwilioTranscriptRequest) (transport.TwilioTranscriptResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := request.Limit
	if limit <= 0 {
		limit = 50
	}

	resolvedThreadID := strings.TrimSpace(request.ThreadID)
	if resolvedThreadID == "" && strings.TrimSpace(request.CallSID) != "" {
		threadID, err := lookupThreadByCallSID(ctx, s.container.DB, workspace, strings.TrimSpace(request.CallSID))
		if err != nil {
			return transport.TwilioTranscriptResponse{}, err
		}
		resolvedThreadID = threadID
	}

	query := `
		SELECT ce.id, ce.thread_id, ct.channel, ce.event_type, ce.direction, ce.assistant_emitted,
		       COALESCE(ce.body_text, ''),
		       COALESCE((
		         SELECT cea.address_value
		         FROM comm_event_addresses cea
		         WHERE cea.event_id = ce.id AND cea.address_role = 'FROM'
		         ORDER BY cea.position ASC
		         LIMIT 1
		       ), ''),
		       ce.occurred_at
		FROM comm_events ce
		JOIN comm_threads ct ON ct.id = ce.thread_id
		LEFT JOIN comm_provider_messages cpm
		  ON cpm.event_id = ce.id
		 AND cpm.provider = 'twilio'
		WHERE ce.workspace_id = ?
		  AND (
		    ct.channel = 'voice'
		    OR (
		      ct.channel = 'message'
		      AND cpm.id IS NOT NULL
		    )
		  )
	`
	params := []any{workspace}
	if resolvedThreadID != "" {
		query += " AND ce.thread_id = ?"
		params = append(params, resolvedThreadID)
	}
	query += " ORDER BY ce.occurred_at DESC, ce.id DESC LIMIT ?"
	params = append(params, maxInt(1, limit))

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.TwilioTranscriptResponse{}, fmt.Errorf("query transcript events: %w", err)
	}
	defer rows.Close()

	events := make([]transport.TwilioTranscriptEvent, 0)
	for rows.Next() {
		var (
			item          transport.TwilioTranscriptEvent
			assistantFlag int
		)
		if err := rows.Scan(
			&item.EventID,
			&item.ThreadID,
			&item.Channel,
			&item.EventType,
			&item.Direction,
			&assistantFlag,
			&item.BodyText,
			&item.SenderAddress,
			&item.OccurredAt,
		); err != nil {
			return transport.TwilioTranscriptResponse{}, fmt.Errorf("scan transcript event: %w", err)
		}
		item.AssistantEmitted = assistantFlag == 1
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return transport.TwilioTranscriptResponse{}, fmt.Errorf("iterate transcript events: %w", err)
	}

	return transport.TwilioTranscriptResponse{
		WorkspaceID: workspace,
		ThreadID:    resolvedThreadID,
		CallSID:     strings.TrimSpace(request.CallSID),
		Events:      events,
	}, nil
}

func (s *CommTwilioService) resolveTwilioWorkspaceCredentials(ctx context.Context, workspace string, config channelconfig.TwilioConfig) (twilioWorkspaceCredentials, error) {
	_, accountSID, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, config.AccountSIDSecretName)
	if err != nil {
		return twilioWorkspaceCredentials{}, fmt.Errorf("resolve twilio account sid secret %q: %w", config.AccountSIDSecretName, err)
	}
	_, authToken, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, config.AuthTokenSecretName)
	if err != nil {
		return twilioWorkspaceCredentials{}, fmt.Errorf("resolve twilio auth token secret %q: %w", config.AuthTokenSecretName, err)
	}
	return twilioWorkspaceCredentials{
		Config:     config,
		AccountSID: accountSID,
		AuthToken:  authToken,
	}, nil
}

func (s *CommTwilioService) executeTwilioSMSDelivery(ctx context.Context, workspace string, destination string, message string, operationID string) (types.DeliveryResult, string, error) {
	store := repodelivery.NewSQLiteDeliveryStore(s.container.DB)
	sender := newDaemonDeliverySender(s.container.DB, s.container.SecretResolver, s.channelDispatch, s.twilioStore, 0, 0)
	service := deliveryservice.NewService(store, sender, deliveryservice.Options{})

	result, err := service.Deliver(ctx, types.DeliveryRequest{
		WorkspaceID: workspace,
		OperationID: strings.TrimSpace(operationID),
		// Twilio-specific command flows use canonical SMS source-channel input.
		SourceChannel:       "sms",
		DestinationEndpoint: strings.TrimSpace(destination),
		MessageBody:         strings.TrimSpace(message),
	})
	if err != nil {
		return result, "", err
	}
	threadID := ""
	if strings.TrimSpace(result.ProviderReceipt) != "" {
		threadID, _ = lookupThreadByProviderMessage(ctx, s.container.DB, workspace, result.ProviderReceipt)
	}
	return result, threadID, nil
}

func twilioConfigRecord(config channelconfig.TwilioConfig) transport.TwilioConfigRecord {
	return transport.TwilioConfigRecord{
		WorkspaceID:           config.WorkspaceID,
		AccountSIDSecretName:  config.AccountSIDSecretName,
		AuthTokenSecretName:   config.AuthTokenSecretName,
		SMSNumber:             config.SMSNumber,
		VoiceNumber:           config.VoiceNumber,
		Endpoint:              config.Endpoint,
		AccountSIDConfigured:  config.AccountSIDConfigured,
		AuthTokenConfigured:   config.AuthTokenConfigured,
		CredentialsConfigured: config.CredentialsConfigured,
		UpdatedAt:             config.UpdatedAt,
	}
}
