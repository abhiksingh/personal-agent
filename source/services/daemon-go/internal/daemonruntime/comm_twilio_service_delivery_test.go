package daemonruntime

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"
)

type stubChannelWorkerDispatcher struct {
	twilioResponse   twilioadapter.SMSAPIResponse
	twilioErr        error
	twilioCalls      int
	lastTwilioReq    twilioadapter.SMSAPIRequest
	messagesResponse messagesadapter.SendResponse
	messagesErr      error
	messagesCalls    int
	lastMessagesReq  messagesadapter.SendRequest
}

type failingAssistantChatStub struct {
	err   error
	calls int
}

func (s *failingAssistantChatStub) ChatTurn(
	_ context.Context,
	_ transport.ChatTurnRequest,
	_ string,
	_ func(delta string),
) (transport.ChatTurnResponse, error) {
	s.calls++
	if s.err != nil {
		return transport.ChatTurnResponse{}, s.err
	}
	return transport.ChatTurnResponse{}, fmt.Errorf("assistant chat failure")
}

func (s *stubChannelWorkerDispatcher) CheckTwilio(_ context.Context, _ channelcheck.TwilioRequest) (channelcheck.TwilioResult, error) {
	return channelcheck.TwilioResult{}, fmt.Errorf("not implemented")
}

func (s *stubChannelWorkerDispatcher) SendTwilioSMS(_ context.Context, request twilioadapter.SMSAPIRequest) (twilioadapter.SMSAPIResponse, error) {
	s.twilioCalls++
	s.lastTwilioReq = request
	if s.twilioErr != nil {
		return twilioadapter.SMSAPIResponse{}, s.twilioErr
	}
	return s.twilioResponse, nil
}

func (s *stubChannelWorkerDispatcher) StartTwilioVoiceCall(_ context.Context, _ twilioadapter.VoiceCallRequest) (twilioadapter.VoiceCallResponse, error) {
	return twilioadapter.VoiceCallResponse{}, fmt.Errorf("not implemented")
}

func (s *stubChannelWorkerDispatcher) SendMessages(_ context.Context, request messagesadapter.SendRequest) (messagesadapter.SendResponse, error) {
	s.lastMessagesReq = request
	s.messagesCalls++
	return s.messagesResponse, s.messagesErr
}

func (s *stubChannelWorkerDispatcher) PollMessagesInbound(_ context.Context, _ messagesadapter.InboundPollRequest) (messagesadapter.InboundPollResponse, error) {
	return messagesadapter.InboundPollResponse{}, fmt.Errorf("not implemented")
}

func TestDaemonDeliverySenderUsesMessagesDispatch(t *testing.T) {
	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			MessageID:   "imessage-worker-1",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	sender := newDaemonDeliverySender(nil, nil, dispatch, nil, 0, 0)

	receipt, err := sender.Send(context.Background(), "imessage", types.DeliveryRequest{
		WorkspaceID:         "ws1",
		OperationID:         "op1",
		DestinationEndpoint: "+15555550999",
		MessageBody:         "hello",
	}, "")
	if err != nil {
		t.Fatalf("send imessage: %v", err)
	}
	if receipt != "imessage-worker-1" {
		t.Fatalf("expected imessage receipt, got %s", receipt)
	}
	if dispatch.messagesCalls != 1 {
		t.Fatalf("expected one messages dispatch call, got %d", dispatch.messagesCalls)
	}
}

func TestDaemonDeliverySenderReturnsErrorForMissingMessagesReceipt(t *testing.T) {
	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	sender := newDaemonDeliverySender(nil, nil, dispatch, nil, 0, 0)

	receipt, err := sender.Send(context.Background(), "imessage", types.DeliveryRequest{
		WorkspaceID:         "ws1",
		OperationID:         "op2",
		DestinationEndpoint: "+15555550999",
		MessageBody:         "hello",
	}, "")
	if err == nil {
		t.Fatalf("expected missing receipt error")
	}
	if !strings.Contains(err.Error(), "empty message id") {
		t.Fatalf("expected empty receipt error, got %v", err)
	}
	if receipt != "" {
		t.Fatalf("expected empty receipt on error, got %s", receipt)
	}
}

func TestDaemonDeliverySenderPreservesSimulatedIMessageFailures(t *testing.T) {
	dispatch := &stubChannelWorkerDispatcher{
		messagesResponse: messagesadapter.SendResponse{
			WorkspaceID: "ws1",
			MessageID:   "imessage-worker-3",
			Channel:     "imessage",
			Status:      "sent",
		},
	}
	sender := newDaemonDeliverySender(nil, nil, dispatch, nil, 1, 0)

	_, firstErr := sender.Send(context.Background(), "imessage", types.DeliveryRequest{
		WorkspaceID:         "ws1",
		OperationID:         "op3",
		DestinationEndpoint: "+15555550999",
		MessageBody:         "hello",
	}, "")
	if firstErr == nil || !strings.Contains(firstErr.Error(), "simulated imessage send failure") {
		t.Fatalf("expected simulated imessage failure, got %v", firstErr)
	}
	if dispatch.messagesCalls != 0 {
		t.Fatalf("expected no dispatch call while simulated failure active, got %d", dispatch.messagesCalls)
	}

	receipt, secondErr := sender.Send(context.Background(), "imessage", types.DeliveryRequest{
		WorkspaceID:         "ws1",
		OperationID:         "op3",
		DestinationEndpoint: "+15555550999",
		MessageBody:         "hello",
	}, "")
	if secondErr != nil {
		t.Fatalf("expected second imessage send to dispatch successfully: %v", secondErr)
	}
	if receipt != "imessage-worker-3" {
		t.Fatalf("expected imessage receipt from dispatcher, got %s", receipt)
	}
	if dispatch.messagesCalls != 1 {
		t.Fatalf("expected one dispatch call after simulated failure consumed, got %d", dispatch.messagesCalls)
	}
}

func TestExecuteTwilioSMSChatTurnUsesAssistantReplyFlow(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm twilio service: %v", err)
	}

	dispatch := &stubChannelWorkerDispatcher{
		twilioResponse: twilioadapter.SMSAPIResponse{
			MessageSID: "SMOUT1",
			AccountSID: "AC123",
			From:       "+15555550001",
			To:         "+15555550999",
			Status:     "queued",
		},
	}
	service.channelDispatch = dispatch
	chatStub := &twilioWebhookAssistantChatStub{}
	service.SetAssistantChatService(chatStub)

	if _, err := manager.Put("ws1", "TWILIO_ACCOUNT_SID", "AC123"); err != nil {
		t.Fatalf("put account sid secret: %v", err)
	}
	if _, err := manager.Put("ws1", "TWILIO_AUTH_TOKEN", "twilio-token"); err != nil {
		t.Fatalf("put auth token secret: %v", err)
	}

	if _, err := service.SetTwilioChannel(context.Background(), transport.TwilioSetRequest{
		WorkspaceID:          "ws1",
		AccountSIDSecretName: "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:  "TWILIO_AUTH_TOKEN",
		SMSNumber:            "+15555550001",
		VoiceNumber:          "+15555550002",
		Endpoint:             "https://example.test",
	}); err != nil {
		t.Fatalf("set twilio channel: %v", err)
	}

	turn, err := service.ExecuteTwilioSMSChatTurn(context.Background(), transport.TwilioSMSChatTurnRequest{
		WorkspaceID: "ws1",
		To:          "+15555550999",
		Message:     "hello from inbound sender",
		OperationID: "turn-1",
	})
	if err != nil {
		t.Fatalf("execute twilio sms chat turn: %v", err)
	}
	if !turn.Success || !turn.Delivered {
		t.Fatalf("expected successful assistant delivery turn, got %+v", turn)
	}
	if turn.ProviderReceipt != "SMOUT1" {
		t.Fatalf("expected provider receipt SMOUT1, got %q", turn.ProviderReceipt)
	}
	if turn.AssistantReply != "stubbed assistant reply" {
		t.Fatalf("expected assistant reply from chat stub, got %q", turn.AssistantReply)
	}
	if turn.AssistantOperationID != "twilio-direct-sms-reply-turn-1" {
		t.Fatalf("expected deterministic assistant operation id, got %q", turn.AssistantOperationID)
	}
	if chatStub.calls != 1 {
		t.Fatalf("expected one assistant chat call, got %d", chatStub.calls)
	}
	if chatStub.request.Channel.ChannelID != "message" || chatStub.request.Channel.ConnectorID != "twilio" {
		t.Fatalf("expected twilio message channel context, got %+v", chatStub.request.Channel)
	}
	if len(chatStub.request.Items) != 1 || chatStub.request.Items[0].Role != "user" || chatStub.request.Items[0].Content != "hello from inbound sender" {
		t.Fatalf("expected one inbound user item in assistant request, got %+v", chatStub.request.Items)
	}
	if dispatch.twilioCalls != 1 {
		t.Fatalf("expected one outbound twilio send call, got %d", dispatch.twilioCalls)
	}
	if dispatch.lastTwilioReq.Body != "stubbed assistant reply" {
		t.Fatalf("expected assistant reply body in outbound send, got %q", dispatch.lastTwilioReq.Body)
	}
}

func TestExecuteTwilioSMSChatTurnSkipsAssistantReplayForSameOperationID(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm twilio service: %v", err)
	}

	dispatch := &stubChannelWorkerDispatcher{
		twilioResponse: twilioadapter.SMSAPIResponse{
			MessageSID: "SMOUT2",
			AccountSID: "AC123",
			From:       "+15555550001",
			To:         "+15555550999",
			Status:     "queued",
		},
	}
	service.channelDispatch = dispatch
	chatStub := &twilioWebhookAssistantChatStub{}
	service.SetAssistantChatService(chatStub)

	if _, err := manager.Put("ws1", "TWILIO_ACCOUNT_SID", "AC123"); err != nil {
		t.Fatalf("put account sid secret: %v", err)
	}
	if _, err := manager.Put("ws1", "TWILIO_AUTH_TOKEN", "twilio-token"); err != nil {
		t.Fatalf("put auth token secret: %v", err)
	}

	if _, err := service.SetTwilioChannel(context.Background(), transport.TwilioSetRequest{
		WorkspaceID:          "ws1",
		AccountSIDSecretName: "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:  "TWILIO_AUTH_TOKEN",
		SMSNumber:            "+15555550001",
		VoiceNumber:          "+15555550002",
		Endpoint:             "https://example.test",
	}); err != nil {
		t.Fatalf("set twilio channel: %v", err)
	}

	first, err := service.ExecuteTwilioSMSChatTurn(context.Background(), transport.TwilioSMSChatTurnRequest{
		WorkspaceID: "ws1",
		To:          "+15555550999",
		Message:     "same operation inbound",
		OperationID: "turn-replay",
	})
	if err != nil {
		t.Fatalf("execute first twilio sms chat turn: %v", err)
	}
	if !first.Success || !first.Delivered {
		t.Fatalf("expected first turn to deliver assistant reply, got %+v", first)
	}

	second, err := service.ExecuteTwilioSMSChatTurn(context.Background(), transport.TwilioSMSChatTurnRequest{
		WorkspaceID: "ws1",
		To:          "+15555550999",
		Message:     "same operation inbound",
		OperationID: "turn-replay",
	})
	if err != nil {
		t.Fatalf("execute replay twilio sms chat turn: %v", err)
	}
	if !second.Success || !second.IdempotentReplay {
		t.Fatalf("expected replay turn success with idempotent replay, got %+v", second)
	}
	if second.Delivered {
		t.Fatalf("expected replay turn to skip outbound delivery, got %+v", second)
	}
	if chatStub.calls != 1 {
		t.Fatalf("expected replay to skip extra assistant calls, got %d", chatStub.calls)
	}
	if dispatch.twilioCalls != 1 {
		t.Fatalf("expected replay to skip extra twilio sends, got %d", dispatch.twilioCalls)
	}
}

func TestExecuteTwilioSMSChatTurnAssistantErrorSkipsOutboundFallbackSend(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm twilio service: %v", err)
	}

	dispatch := &stubChannelWorkerDispatcher{
		twilioResponse: twilioadapter.SMSAPIResponse{
			MessageSID: "SMOUT3",
			AccountSID: "AC123",
			From:       "+15555550001",
			To:         "+15555550999",
			Status:     "queued",
		},
	}
	service.channelDispatch = dispatch
	chatStub := &failingAssistantChatStub{err: fmt.Errorf("assistant unavailable")}
	service.SetAssistantChatService(chatStub)

	if _, err := manager.Put("ws1", "TWILIO_ACCOUNT_SID", "AC123"); err != nil {
		t.Fatalf("put account sid secret: %v", err)
	}
	if _, err := manager.Put("ws1", "TWILIO_AUTH_TOKEN", "twilio-token"); err != nil {
		t.Fatalf("put auth token secret: %v", err)
	}

	if _, err := service.SetTwilioChannel(context.Background(), transport.TwilioSetRequest{
		WorkspaceID:          "ws1",
		AccountSIDSecretName: "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:  "TWILIO_AUTH_TOKEN",
		SMSNumber:            "+15555550001",
		VoiceNumber:          "+15555550002",
		Endpoint:             "https://example.test",
	}); err != nil {
		t.Fatalf("set twilio channel: %v", err)
	}

	turn, err := service.ExecuteTwilioSMSChatTurn(context.Background(), transport.TwilioSMSChatTurnRequest{
		WorkspaceID: "ws1",
		To:          "+15555550999",
		Message:     "hello when assistant is down",
		OperationID: "turn-assistant-error",
	})
	if err != nil {
		t.Fatalf("execute twilio sms chat turn: %v", err)
	}
	if !turn.Success {
		t.Fatalf("expected accepted ingest success despite assistant error, got %+v", turn)
	}
	if turn.Delivered {
		t.Fatalf("expected no outbound delivery on assistant error, got %+v", turn)
	}
	if !strings.Contains(strings.ToLower(turn.AssistantError), "assistant") {
		t.Fatalf("expected assistant error details in response, got %+v", turn)
	}
	if turn.Error != "" {
		t.Fatalf("expected top-level error empty on assistant failure path, got %q", turn.Error)
	}
	if dispatch.twilioCalls != 0 {
		t.Fatalf("expected zero twilio outbound calls on assistant error path, got %d", dispatch.twilioCalls)
	}
	if chatStub.calls != 1 {
		t.Fatalf("expected exactly one assistant chat attempt, got %d", chatStub.calls)
	}
}
