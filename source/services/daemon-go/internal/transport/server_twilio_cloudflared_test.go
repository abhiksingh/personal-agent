package transport

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type twilioWebhookModeServiceStub struct {
	serveCalled    bool
	replayCalled   bool
	serveRequest   TwilioWebhookServeRequest
	replayRequest  TwilioWebhookReplayRequest
	serveResponse  TwilioWebhookServeResponse
	replayResponse TwilioWebhookReplayResponse
}

func (s *twilioWebhookModeServiceStub) SetTwilioChannel(context.Context, TwilioSetRequest) (TwilioConfigRecord, error) {
	return TwilioConfigRecord{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) GetTwilioChannel(context.Context, TwilioGetRequest) (TwilioConfigRecord, error) {
	return TwilioConfigRecord{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) CheckTwilioChannel(context.Context, TwilioCheckRequest) (TwilioCheckResponse, error) {
	return TwilioCheckResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) ExecuteTwilioSMSChatTurn(context.Context, TwilioSMSChatTurnRequest) (TwilioSMSChatTurn, error) {
	return TwilioSMSChatTurn{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) StartTwilioCall(context.Context, TwilioStartCallRequest) (TwilioStartCallResponse, error) {
	return TwilioStartCallResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) ListTwilioCallStatus(context.Context, TwilioCallStatusRequest) (TwilioCallStatusResponse, error) {
	return TwilioCallStatusResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) ListTwilioTranscript(context.Context, TwilioTranscriptRequest) (TwilioTranscriptResponse, error) {
	return TwilioTranscriptResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) IngestTwilioSMS(context.Context, TwilioIngestSMSRequest) (TwilioIngestSMSResponse, error) {
	return TwilioIngestSMSResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) IngestTwilioVoice(context.Context, TwilioIngestVoiceRequest) (TwilioIngestVoiceResponse, error) {
	return TwilioIngestVoiceResponse{}, errors.New("not implemented in test stub")
}

func (s *twilioWebhookModeServiceStub) ServeTwilioWebhook(_ context.Context, request TwilioWebhookServeRequest) (TwilioWebhookServeResponse, error) {
	s.serveCalled = true
	s.serveRequest = request
	return s.serveResponse, nil
}

func (s *twilioWebhookModeServiceStub) ReplayTwilioWebhook(_ context.Context, request TwilioWebhookReplayRequest) (TwilioWebhookReplayResponse, error) {
	s.replayCalled = true
	s.replayRequest = request
	return s.replayResponse, nil
}

func TestTransportTwilioWebhookServeRejectsBypassInProdProfile(t *testing.T) {
	twilio := &twilioWebhookModeServiceStub{}
	server := startTestServer(t, ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "prod",
		AuthToken:      "twilio-token",
		Twilio:         twilio,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "twilio-token",
	})
	if err != nil {
		t.Fatalf("create twilio client: %v", err)
	}

	_, err = client.TwilioWebhookServe(context.Background(), TwilioWebhookServeRequest{
		WorkspaceID:       "ws1",
		ListenAddress:     "127.0.0.1:8080",
		SignatureMode:     "bypass",
		VoiceResponseMode: "empty",
		SMSPath:           "/sms",
		VoicePath:         "/voice",
	}, "corr-twilio-serve-bypass")
	if err == nil {
		t.Fatalf("expected prod profile to reject bypass signature mode")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", httpErr.StatusCode)
	}
	if httpErr.Message != "--runtime-profile=prod does not allow --signature-mode=bypass" {
		t.Fatalf("unexpected error message: %q", httpErr.Message)
	}
	if twilio.serveCalled {
		t.Fatalf("expected twilio webhook serve service not to be called on prod bypass reject")
	}
}

func TestTransportTwilioWebhookReplayRejectsBypassInProdProfile(t *testing.T) {
	twilio := &twilioWebhookModeServiceStub{}
	server := startTestServer(t, ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "prod",
		AuthToken:      "twilio-token",
		Twilio:         twilio,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "twilio-token",
	})
	if err != nil {
		t.Fatalf("create twilio client: %v", err)
	}

	_, err = client.TwilioWebhookReplay(context.Background(), TwilioWebhookReplayRequest{
		WorkspaceID:   "ws1",
		Kind:          "sms",
		BaseURL:       "http://localhost:7071",
		SignatureMode: "bypass",
		SMSPath:       "/sms",
		VoicePath:     "/voice",
		Params:        map[string]string{"From": "+15550000000"},
	}, "corr-twilio-replay-bypass")
	if err == nil {
		t.Fatalf("expected prod profile to reject bypass signature mode for replay")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", httpErr.StatusCode)
	}
	if httpErr.Message != "--runtime-profile=prod does not allow --signature-mode=bypass" {
		t.Fatalf("unexpected error message: %q", httpErr.Message)
	}
	if twilio.replayCalled {
		t.Fatalf("expected twilio webhook replay service not to be called on prod bypass reject")
	}
}

func TestTransportTwilioWebhookBypassAllowedInLocalProfile(t *testing.T) {
	twilio := &twilioWebhookModeServiceStub{
		serveResponse: TwilioWebhookServeResponse{
			WorkspaceID:        "ws1",
			SignatureMode:      "bypass",
			ListenAddress:      "127.0.0.1:8080",
			SMSWebhookURL:      "http://localhost:8080/sms",
			VoiceWebhookURL:    "http://localhost:8080/voice",
			AssistantReplies:   true,
			AssistantTaskClass: "chat",
			VoiceResponseMode:  "empty",
		},
		replayResponse: TwilioWebhookReplayResponse{
			WorkspaceID:      "ws1",
			Kind:             "sms",
			TargetURL:        "http://localhost:8080/sms",
			RequestURL:       "http://localhost:8080/sms",
			SignatureMode:    "bypass",
			SignaturePresent: false,
			StatusCode:       200,
			ResponseBody:     "ok",
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "local",
		AuthToken:      "twilio-token",
		Twilio:         twilio,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "twilio-token",
	})
	if err != nil {
		t.Fatalf("create twilio client: %v", err)
	}

	serveResponse, err := client.TwilioWebhookServe(context.Background(), TwilioWebhookServeRequest{
		WorkspaceID:        "ws1",
		ListenAddress:      "127.0.0.1:8080",
		SignatureMode:      "bypass",
		AssistantReplies:   true,
		AssistantTaskClass: "chat",
		VoiceResponseMode:  "empty",
		SMSPath:            "/sms",
		VoicePath:          "/voice",
	}, "corr-twilio-serve-local")
	if err != nil {
		t.Fatalf("twilio webhook serve in local profile: %v", err)
	}
	if serveResponse.SignatureMode != "bypass" || !twilio.serveCalled || twilio.serveRequest.SignatureMode != "bypass" {
		t.Fatalf("expected local profile to allow bypass signature mode, response=%+v request=%+v called=%v", serveResponse, twilio.serveRequest, twilio.serveCalled)
	}

	replayResponse, err := client.TwilioWebhookReplay(context.Background(), TwilioWebhookReplayRequest{
		WorkspaceID:   "ws1",
		Kind:          "sms",
		BaseURL:       "http://localhost:8080",
		SignatureMode: "bypass",
		SMSPath:       "/sms",
		VoicePath:     "/voice",
		Params:        map[string]string{"From": "+15550000000"},
	}, "corr-twilio-replay-local")
	if err != nil {
		t.Fatalf("twilio webhook replay in local profile: %v", err)
	}
	if replayResponse.SignatureMode != "bypass" || !twilio.replayCalled || twilio.replayRequest.SignatureMode != "bypass" {
		t.Fatalf("expected local profile replay to allow bypass signature mode, response=%+v request=%+v called=%v", replayResponse, twilio.replayRequest, twilio.replayCalled)
	}
}
