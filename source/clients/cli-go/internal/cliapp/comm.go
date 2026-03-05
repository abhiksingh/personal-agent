package cliapp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"personalagent/runtime/internal/channelconfig"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/securestore"
)

type cliDeliverySender struct {
	mu       sync.Mutex
	failures map[string]int
	sent     map[string]int
	db       *sql.DB
	manager  *securestore.Manager
	client   *http.Client
}

func newCLIDeliverySender(db *sql.DB, manager *securestore.Manager, client *http.Client, imessageFailures int, smsFailures int) *cliDeliverySender {
	httpClient := client
	if httpClient == nil {
		httpClient = newCLIHTTPClient(0)
	}
	return &cliDeliverySender{
		failures: map[string]int{
			"imessage": maxInt(0, imessageFailures),
			"twilio":   maxInt(0, smsFailures),
		},
		sent:    map[string]int{},
		db:      db,
		manager: manager,
		client:  httpClient,
	}
}

func (s *cliDeliverySender) Send(ctx context.Context, channel string, request types.DeliveryRequest, _ string) (string, error) {
	routeChannel := normalizeCLIDeliveryRouteChannel(channel)
	if routeChannel == "twilio" {
		receipt, handled, err := s.sendTwilioSMS(ctx, request)
		if handled {
			return receipt, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failures[routeChannel] > 0 {
		s.failures[routeChannel]--
		return "", fmt.Errorf("simulated %s send failure", routeChannel)
	}

	s.sent[routeChannel]++
	return fmt.Sprintf("%s:%s:%d", routeChannel, strings.TrimSpace(request.OperationID), s.sent[routeChannel]), nil
}

func (s *cliDeliverySender) sendTwilioSMS(ctx context.Context, request types.DeliveryRequest) (string, bool, error) {
	if s.db == nil {
		return "", false, nil
	}
	workspace := normalizeWorkspace(request.WorkspaceID)
	store := channelconfig.NewSQLiteTwilioStore(s.db)
	config, err := store.Get(ctx, workspace)
	if err != nil {
		if errors.Is(err, channelconfig.ErrTwilioNotConfigured) {
			return "", false, nil
		}
		return "", true, fmt.Errorf("load twilio channel config: %w", err)
	}
	if s.manager == nil {
		return "", true, fmt.Errorf("twilio configured for workspace %q but secret manager is unavailable", workspace)
	}

	_, accountSID, err := s.manager.Get(workspace, config.AccountSIDSecretName)
	if err != nil {
		return "", true, fmt.Errorf("resolve twilio account sid secret %q: %w", config.AccountSIDSecretName, err)
	}
	_, authToken, err := s.manager.Get(workspace, config.AuthTokenSecretName)
	if err != nil {
		return "", true, fmt.Errorf("resolve twilio auth token secret %q: %w", config.AuthTokenSecretName, err)
	}

	providerResponse, err := twilioadapter.SendSMS(ctx, s.client, twilioadapter.SMSAPIRequest{
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

func parseFallbackChannels(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, channel := range raw {
		normalized := strings.ToLower(strings.TrimSpace(channel))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func commRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeCLIDeliveryRouteChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "sms", "twilio":
		return "twilio"
	case "", "imessage":
		return "imessage"
	case "app", "builtin.app":
		return "builtin.app"
	default:
		return strings.ToLower(strings.TrimSpace(channel))
	}
}
