package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"personalagent/runtime/internal/channelconfig"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/core/types"
)

func newDaemonDeliverySender(
	db *sql.DB,
	resolver SecretReferenceResolver,
	dispatch ChannelWorkerDispatcher,
	twilioStore *channelconfig.SQLiteTwilioStore,
	imessageFailures int,
	smsFailures int,
) *daemonDeliverySender {
	return &daemonDeliverySender{
		failures: map[string]int{
			"imessage": maxInt(0, imessageFailures),
			"sms":      maxInt(0, smsFailures),
		},
		sent:      map[string]int{},
		db:        db,
		resolver:  resolver,
		dispatch:  dispatch,
		twilioCfg: twilioStore,
	}
}

func (s *daemonDeliverySender) Send(ctx context.Context, channel string, request types.DeliveryRequest, _ string) (string, error) {
	normalizedChannel := strings.ToLower(strings.TrimSpace(channel))

	if isMessagesRoute(normalizedChannel) {
		s.mu.Lock()
		if s.failures["imessage"] > 0 {
			s.failures["imessage"]--
			s.mu.Unlock()
			return "", fmt.Errorf("simulated imessage send failure")
		}
		s.mu.Unlock()

		receipt, handled, err := s.sendMessages(ctx, request)
		if handled {
			return receipt, err
		}
	}

	if isTwilioSMSRoute(normalizedChannel) {
		s.mu.Lock()
		if s.failures["sms"] > 0 {
			s.failures["sms"]--
			s.mu.Unlock()
			return "", fmt.Errorf("simulated sms send failure")
		}
		s.mu.Unlock()

		receipt, handled, err := s.sendTwilioSMS(ctx, request)
		if handled {
			return receipt, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failures[normalizedChannel] > 0 {
		s.failures[normalizedChannel]--
		return "", fmt.Errorf("simulated %s send failure", normalizedChannel)
	}

	s.sent[normalizedChannel]++
	return fmt.Sprintf("%s:%s:%d", normalizedChannel, strings.TrimSpace(request.OperationID), s.sent[normalizedChannel]), nil
}

func isMessagesRoute(channel string) bool {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "imessage":
		return true
	default:
		return false
	}
}

func isTwilioSMSRoute(channel string) bool {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "twilio", "sms":
		return true
	default:
		return false
	}
}

func (s *daemonDeliverySender) sendMessages(ctx context.Context, request types.DeliveryRequest) (string, bool, error) {
	if s.dispatch == nil {
		return "", false, nil
	}

	response, err := s.dispatch.SendMessages(ctx, messagesadapter.SendRequest{
		WorkspaceID: normalizeWorkspaceID(request.WorkspaceID),
		Destination: strings.TrimSpace(request.DestinationEndpoint),
		Message:     strings.TrimSpace(request.MessageBody),
	})
	if err != nil {
		return "", true, fmt.Errorf("messages send failed: %w", err)
	}

	receipt := strings.TrimSpace(response.MessageID)
	if receipt == "" {
		return "", true, fmt.Errorf("messages send failed: empty message id")
	}
	return receipt, true, nil
}

func (s *daemonDeliverySender) sendTwilioSMS(ctx context.Context, request types.DeliveryRequest) (string, bool, error) {
	if s.db == nil || s.twilioCfg == nil {
		return "", false, nil
	}
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	config, err := s.twilioCfg.Get(ctx, workspace)
	if err != nil {
		if errors.Is(err, channelconfig.ErrTwilioNotConfigured) {
			return "", false, nil
		}
		return "", true, fmt.Errorf("load twilio channel config: %w", err)
	}
	if s.resolver == nil {
		return "", true, fmt.Errorf("twilio configured for workspace %q but secret resolver is unavailable", workspace)
	}

	_, accountSID, err := s.resolver.ResolveSecret(ctx, workspace, config.AccountSIDSecretName)
	if err != nil {
		return "", true, fmt.Errorf("resolve twilio account sid secret %q: %w", config.AccountSIDSecretName, err)
	}
	_, authToken, err := s.resolver.ResolveSecret(ctx, workspace, config.AuthTokenSecretName)
	if err != nil {
		return "", true, fmt.Errorf("resolve twilio auth token secret %q: %w", config.AuthTokenSecretName, err)
	}

	providerResponse, err := s.dispatch.SendTwilioSMS(ctx, twilioadapter.SMSAPIRequest{
		Endpoint:   config.Endpoint,
		AccountSID: accountSID,
		AuthToken:  authToken,
		From:       config.SMSNumber,
		To:         strings.TrimSpace(request.DestinationEndpoint),
		Body:       strings.TrimSpace(request.MessageBody),
	})
	if err != nil {
		return "", true, fmt.Errorf("twilio sms send failed: %w", err)
	}

	persistence := twilioadapter.NewSMSPersistence(s.db)
	if _, err := persistence.PersistOutboundSMS(ctx, twilioadapter.OutboundSMSInput{
		WorkspaceID:     workspace,
		ProviderMessage: providerResponse.MessageSID,
		ProviderAccount: firstNonEmpty(strings.TrimSpace(providerResponse.AccountSID), strings.TrimSpace(accountSID)),
		FromAddress:     firstNonEmpty(strings.TrimSpace(providerResponse.From), config.SMSNumber),
		ToAddress:       firstNonEmpty(strings.TrimSpace(providerResponse.To), strings.TrimSpace(request.DestinationEndpoint)),
		BodyText:        strings.TrimSpace(request.MessageBody),
		ProviderStatus:  providerResponse.Status,
		ProviderPayload: map[string]any{
			"sid":           providerResponse.MessageSID,
			"account_sid":   providerResponse.AccountSID,
			"status":        providerResponse.Status,
			"from":          providerResponse.From,
			"to":            providerResponse.To,
			"error_code":    providerResponse.ErrorCode,
			"error_message": providerResponse.ErrorMessage,
			"status_code":   providerResponse.StatusCode,
		},
	}); err != nil {
		return "", true, fmt.Errorf("persist twilio outbound message: %w", err)
	}

	return providerResponse.MessageSID, true, nil
}
