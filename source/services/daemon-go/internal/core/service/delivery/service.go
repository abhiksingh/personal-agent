package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/contract"
	"personalagent/runtime/internal/core/types"
)

type Service struct {
	store  contract.DeliveryStore
	sender contract.DeliverySender
	now    func() time.Time
}

type Options struct {
	Now func() time.Time
}

func NewService(store contract.DeliveryStore, sender contract.DeliverySender, opts Options) *Service {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, sender: sender, now: nowFn}
}

func (s *Service) Deliver(ctx context.Context, request types.DeliveryRequest) (types.DeliveryResult, error) {
	if s.store == nil {
		return types.DeliveryResult{}, fmt.Errorf("delivery store is required")
	}
	if s.sender == nil {
		return types.DeliveryResult{}, fmt.Errorf("delivery sender is required")
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return types.DeliveryResult{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(request.OperationID) == "" {
		return types.DeliveryResult{}, fmt.Errorf("operation id is required")
	}
	if strings.TrimSpace(request.DestinationEndpoint) == "" {
		return types.DeliveryResult{}, fmt.Errorf("destination endpoint is required")
	}

	policy := s.defaultPolicy(request.SourceChannel)
	resolvedPolicy, found, err := s.store.ResolveDeliveryPolicy(ctx, request.WorkspaceID, request.SourceChannel, request.DestinationEndpoint)
	if err != nil {
		return types.DeliveryResult{}, err
	}
	if found {
		policy = normalizePolicy(resolvedPolicy)
	}

	channels := buildDeliveryChannels(policy)
	if len(channels) == 0 {
		return types.DeliveryResult{}, fmt.Errorf("delivery policy produced no channels")
	}

	result := types.DeliveryResult{Attempts: []types.DeliveryAttemptRecord{}}
	var lastError error
	for index, channel := range channels {
		idempotencyKey := composeIdempotencyKey(
			request.OperationID,
			request.DestinationEndpoint,
			request.SourceChannel,
			channel,
			index,
		)
		attempt := types.DeliveryAttemptRecord{
			WorkspaceID:         request.WorkspaceID,
			StepID:              request.StepID,
			EventID:             request.EventID,
			DestinationEndpoint: request.DestinationEndpoint,
			IdempotencyKey:      idempotencyKey,
			Channel:             channel,
			Status:              types.DeliveryAttemptPending,
			AttemptedAt:         s.now().UTC(),
		}

		reserved, created, reserveErr := s.store.ReserveDeliveryAttempt(ctx, attempt)
		if reserveErr != nil {
			return result, reserveErr
		}
		if !created {
			result.Attempts = append(result.Attempts, reserved)
			if reserved.Status == types.DeliveryAttemptSent {
				result.Delivered = true
				result.Channel = reserved.Channel
				result.ProviderReceipt = reserved.ProviderReceipt
				result.IdempotentReplay = true
				return result, nil
			}
			if reserved.Status == types.DeliveryAttemptPending {
				lastError = fmt.Errorf("delivery attempt in progress for channel %s", reserved.Channel)
				continue
			}
			continue
		}

		receipt, sendErr := s.sender.Send(ctx, channel, request, idempotencyKey)
		if sendErr != nil {
			attempt.Status = types.DeliveryAttemptFailed
			attempt.ErrorText = sendErr.Error()
			if err := s.store.MarkDeliveryAttemptResult(ctx, reserved.AttemptID, attempt.Status, "", attempt.ErrorText, s.now().UTC()); err != nil {
				return result, err
			}
			reserved.Status = attempt.Status
			reserved.ErrorText = attempt.ErrorText
			result.Attempts = append(result.Attempts, reserved)
			lastError = sendErr
			continue
		}

		attempt.Status = types.DeliveryAttemptSent
		attempt.ProviderReceipt = receipt
		if err := s.store.MarkDeliveryAttemptResult(ctx, reserved.AttemptID, attempt.Status, receipt, "", s.now().UTC()); err != nil {
			return result, err
		}
		reserved.Status = attempt.Status
		reserved.ProviderReceipt = receipt
		result.Attempts = append(result.Attempts, reserved)
		result.Delivered = true
		result.Channel = channel
		result.ProviderReceipt = receipt
		return result, nil
	}

	if lastError == nil {
		lastError = fmt.Errorf("delivery failed")
	}
	return result, lastError
}

func (s *Service) defaultPolicy(sourceChannel string) types.ChannelDeliveryPolicy {
	normalized := normalizeLogicalSourceChannel(sourceChannel)
	switch normalized {
	case "", "message", "imessage":
		return types.ChannelDeliveryPolicy{PrimaryChannel: "imessage", RetryCount: 1, FallbackChannels: []string{"twilio"}}
	case "sms", "voice":
		return types.ChannelDeliveryPolicy{PrimaryChannel: "twilio", RetryCount: 0, FallbackChannels: nil}
	case "app":
		return types.ChannelDeliveryPolicy{PrimaryChannel: "builtin.app", RetryCount: 0, FallbackChannels: nil}
	default:
		route := normalizeDeliveryRouteChannel(normalized)
		return types.ChannelDeliveryPolicy{PrimaryChannel: route, RetryCount: 1, FallbackChannels: nil}
	}
}

func buildDeliveryChannels(policy types.ChannelDeliveryPolicy) []string {
	normalized := normalizePolicy(policy)
	channels := make([]string, 0, normalized.RetryCount+1+len(normalized.FallbackChannels))
	for index := 0; index <= normalized.RetryCount; index++ {
		channels = append(channels, normalized.PrimaryChannel)
	}
	channels = append(channels, normalized.FallbackChannels...)
	return channels
}

func normalizePolicy(policy types.ChannelDeliveryPolicy) types.ChannelDeliveryPolicy {
	policy.PrimaryChannel = normalizeDeliveryRouteChannel(policy.PrimaryChannel)
	if policy.RetryCount < 0 {
		policy.RetryCount = 0
	}

	fallback := make([]string, 0, len(policy.FallbackChannels))
	seen := map[string]struct{}{}
	if policy.PrimaryChannel != "" {
		seen[policy.PrimaryChannel] = struct{}{}
	}
	for _, channel := range policy.FallbackChannels {
		normalized := normalizeDeliveryRouteChannel(channel)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		fallback = append(fallback, normalized)
	}
	policy.FallbackChannels = fallback

	if policy.PrimaryChannel == "" {
		policy.PrimaryChannel = "imessage"
	}
	return policy
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func composeIdempotencyKey(operationID string, destinationEndpoint string, sourceChannel string, routeChannel string, routeIndex int) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%d",
		strings.TrimSpace(operationID),
		strings.TrimSpace(destinationEndpoint),
		normalizeLogicalSourceChannel(sourceChannel),
		normalizeDeliveryRouteChannel(routeChannel),
		routeIndex,
	)
}

func normalizeLogicalSourceChannel(sourceChannel string) string {
	switch normalizeChannel(sourceChannel) {
	case "", "message", "imessage":
		return "message"
	case "voice":
		return "voice"
	case "app":
		return "app"
	case "sms":
		return "sms"
	default:
		return normalizeChannel(sourceChannel)
	}
}

func normalizeDeliveryRouteChannel(route string) string {
	switch normalizeChannel(route) {
	case "", "imessage":
		return "imessage"
	case "twilio", "sms":
		return "twilio"
	case "builtin.app", "app":
		return "builtin.app"
	default:
		return normalizeChannel(route)
	}
}
