package transport

import (
	"context"
	"errors"
	"testing"

	coretypes "personalagent/runtime/internal/core/types"
)

type commPolicyAttemptServiceStub struct {
	sendResponse         CommSendResponse
	policySetResponse    CommPolicyRecord
	attemptsResponse     CommAttemptsResponse
	webhookListResponse  CommWebhookReceiptListResponse
	ingestListResponse   CommIngestReceiptListResponse
	lastSendReq          CommSendRequest
	lastPolicyReq        CommPolicySetRequest
	lastAttemptsReq      CommAttemptsRequest
	lastWebhookListReq   CommWebhookReceiptListRequest
	lastIngestReceiptReq CommIngestReceiptListRequest
}

func (s *commPolicyAttemptServiceStub) SendComm(_ context.Context, request CommSendRequest) (CommSendResponse, error) {
	s.lastSendReq = request
	return s.sendResponse, nil
}

func (s *commPolicyAttemptServiceStub) ListCommAttempts(_ context.Context, request CommAttemptsRequest) (CommAttemptsResponse, error) {
	s.lastAttemptsReq = request
	return s.attemptsResponse, nil
}

func (s *commPolicyAttemptServiceStub) SetCommPolicy(_ context.Context, request CommPolicySetRequest) (CommPolicyRecord, error) {
	s.lastPolicyReq = request
	return s.policySetResponse, nil
}

func (s *commPolicyAttemptServiceStub) ListCommPolicies(context.Context, CommPolicyListRequest) (CommPolicyListResponse, error) {
	return CommPolicyListResponse{}, nil
}

func (s *commPolicyAttemptServiceStub) ListCommWebhookReceipts(_ context.Context, request CommWebhookReceiptListRequest) (CommWebhookReceiptListResponse, error) {
	s.lastWebhookListReq = request
	return s.webhookListResponse, nil
}

func (s *commPolicyAttemptServiceStub) ListCommIngestReceipts(_ context.Context, request CommIngestReceiptListRequest) (CommIngestReceiptListResponse, error) {
	s.lastIngestReceiptReq = request
	return s.ingestListResponse, nil
}

func (s *commPolicyAttemptServiceStub) IngestMessages(context.Context, MessagesIngestRequest) (MessagesIngestResponse, error) {
	return MessagesIngestResponse{}, nil
}

func (s *commPolicyAttemptServiceStub) IngestMailRuleEvent(context.Context, MailRuleIngestRequest) (MailRuleIngestResponse, error) {
	return MailRuleIngestResponse{}, nil
}

func (s *commPolicyAttemptServiceStub) IngestCalendarChange(context.Context, CalendarChangeIngestRequest) (CalendarChangeIngestResponse, error) {
	return CalendarChangeIngestResponse{}, nil
}

func (s *commPolicyAttemptServiceStub) IngestBrowserEvent(context.Context, BrowserEventIngestRequest) (BrowserEventIngestResponse, error) {
	return BrowserEventIngestResponse{}, nil
}

func TestTransportCommPolicyUpdateAndAttemptHistoryRoutes(t *testing.T) {
	commService := &commPolicyAttemptServiceStub{
		sendResponse: CommSendResponse{
			WorkspaceID:           "ws1",
			OperationID:           "op-send-1",
			ThreadID:              "thread-1",
			ResolvedSourceChannel: "twilio_sms",
			ResolvedConnectorID:   "twilio",
			ResolvedDestination:   "+15550001111",
			Success:               true,
			Result: coretypes.DeliveryResult{
				Delivered:       true,
				Channel:         "sms",
				ProviderReceipt: "SM123",
			},
		},
		policySetResponse: CommPolicyRecord{
			ID:            "policy-1",
			WorkspaceID:   "ws1",
			SourceChannel: "imessage",
			Policy: coretypes.ChannelDeliveryPolicy{
				PrimaryChannel:   "sms",
				RetryCount:       0,
				FallbackChannels: []string{"twilio_sms"},
			},
			IsDefault: true,
			CreatedAt: "2026-02-25T00:00:00Z",
			UpdatedAt: "2026-02-25T00:00:01Z",
		},
		attemptsResponse: CommAttemptsResponse{
			WorkspaceID: "ws1",
			OperationID: "op-1",
			ThreadID:    "thread-1",
			TaskID:      "task-1",
			RunID:       "run-1",
			HasMore:     true,
			NextCursor:  "2026-02-25T00:00:03Z|attempt-3",
			Attempts: []CommAttemptRecord{
				{
					AttemptID:           "attempt-3",
					WorkspaceID:         "ws1",
					OperationID:         "op-1",
					TaskID:              "task-1",
					RunID:               "run-1",
					StepID:              "step-1",
					EventID:             "event-1",
					ThreadID:            "thread-1",
					DestinationEndpoint: "+15550001111",
					IdempotencyKey:      "op-1|+15550001111|sms|2",
					Channel:             "sms",
					RouteIndex:          2,
					RoutePhase:          "fallback",
					RetryOrdinal:        0,
					FallbackFromChannel: "imessage",
					Status:              "sent",
					AttemptedAt:         "2026-02-25T00:00:03Z",
				},
			},
		},
		webhookListResponse: CommWebhookReceiptListResponse{
			WorkspaceID: "ws1",
			Provider:    "twilio",
			Items: []CommWebhookReceiptItem{
				{
					ReceiptID:             "wr-1",
					WorkspaceID:           "ws1",
					Provider:              "twilio",
					ProviderEventID:       "SM123",
					TrustState:            "accepted",
					SignatureValid:        true,
					SignatureValuePresent: true,
					EventID:               "event-1",
					ThreadID:              "thread-1",
					ReceivedAt:            "2026-02-25T00:00:01Z",
					CreatedAt:             "2026-02-25T00:00:01Z",
					AuditLinks: []ReceiptAuditLink{
						{AuditID: "audit-wr-1", EventType: "twilio_webhook_received", CreatedAt: "2026-02-25T00:00:01Z"},
					},
				},
			},
		},
		ingestListResponse: CommIngestReceiptListResponse{
			WorkspaceID: "ws1",
			Source:      "apple_mail_rule",
			Items: []CommIngestReceiptItem{
				{
					ReceiptID:     "ir-1",
					WorkspaceID:   "ws1",
					Source:        "apple_mail_rule",
					SourceScope:   "mail-rule-default",
					SourceEventID: "mail-event-1",
					SourceCursor:  "100",
					TrustState:    "accepted",
					EventID:       "event-2",
					ThreadID:      "thread-2",
					ReceivedAt:    "2026-02-25T00:00:02Z",
					CreatedAt:     "2026-02-25T00:00:02Z",
					AuditLinks: []ReceiptAuditLink{
						{AuditID: "audit-ir-1", EventType: "comm_ingest_received", CreatedAt: "2026-02-25T00:00:02Z"},
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "test-token",
		Comm:         commService,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "test-token",
	})
	if err != nil {
		t.Fatalf("create comm test client: %v", err)
	}

	policyResponse, err := client.CommPolicySet(context.Background(), CommPolicySetRequest{
		PolicyID:         "policy-1",
		WorkspaceID:      "ws1",
		SourceChannel:    "imessage",
		EndpointPattern:  "+1555%",
		PrimaryChannel:   "sms",
		RetryCount:       0,
		FallbackChannels: []string{"twilio_sms"},
		IsDefault:        true,
	}, "corr-policy-update")
	if err != nil {
		t.Fatalf("comm policy set: %v", err)
	}
	if policyResponse.ID != "policy-1" || policyResponse.Policy.PrimaryChannel != "sms" {
		t.Fatalf("unexpected policy response: %+v", policyResponse)
	}
	if commService.lastPolicyReq.PolicyID != "policy-1" || commService.lastPolicyReq.WorkspaceID != "ws1" {
		t.Fatalf("unexpected policy request payload: %+v", commService.lastPolicyReq)
	}

	sendResponse, err := client.CommSend(context.Background(), CommSendRequest{
		WorkspaceID:   "ws1",
		OperationID:   "op-send-1",
		ThreadID:      "thread-1",
		ConnectorID:   "twilio",
		SourceChannel: "message",
		Message:       "hello from transport test",
	}, "corr-comm-send")
	if err != nil {
		t.Fatalf("comm send: %v", err)
	}
	if !sendResponse.Success || sendResponse.ResolvedConnectorID != "twilio" || sendResponse.ResolvedDestination != "+15550001111" {
		t.Fatalf("unexpected comm send response payload: %+v", sendResponse)
	}
	if commService.lastSendReq.ThreadID != "thread-1" || commService.lastSendReq.ConnectorID != "twilio" || commService.lastSendReq.SourceChannel != "message" {
		t.Fatalf("unexpected comm send request payload: %+v", commService.lastSendReq)
	}

	attemptsResponse, err := client.CommAttempts(context.Background(), CommAttemptsRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
		TaskID:      "task-1",
		RunID:       "run-1",
		Channel:     "sms",
		Status:      "sent",
		Cursor:      "2026-02-25T00:00:04Z|attempt-4",
		Limit:       10,
	}, "corr-attempt-history")
	if err != nil {
		t.Fatalf("comm attempts history: %v", err)
	}
	if !attemptsResponse.HasMore || attemptsResponse.NextCursor == "" {
		t.Fatalf("expected has_more cursor fields, got %+v", attemptsResponse)
	}
	if len(attemptsResponse.Attempts) != 1 || attemptsResponse.Attempts[0].RoutePhase != "fallback" {
		t.Fatalf("unexpected attempts response payload: %+v", attemptsResponse)
	}
	if commService.lastAttemptsReq.ThreadID != "thread-1" || commService.lastAttemptsReq.TaskID != "task-1" || commService.lastAttemptsReq.RunID != "run-1" {
		t.Fatalf("unexpected attempts request payload: %+v", commService.lastAttemptsReq)
	}

	webhookReceipts, err := client.CommWebhookReceipts(context.Background(), CommWebhookReceiptListRequest{
		WorkspaceID:        "ws1",
		Provider:           "twilio",
		ProviderEventQuery: "sm",
		Limit:              5,
	}, "corr-webhook-receipts")
	if err != nil {
		t.Fatalf("comm webhook receipts: %v", err)
	}
	if len(webhookReceipts.Items) != 1 || webhookReceipts.Items[0].ReceiptID != "wr-1" {
		t.Fatalf("unexpected webhook receipts payload: %+v", webhookReceipts)
	}
	if len(webhookReceipts.Items[0].AuditLinks) != 1 {
		t.Fatalf("expected webhook receipt audit links, got %+v", webhookReceipts.Items[0].AuditLinks)
	}
	if commService.lastWebhookListReq.Provider != "twilio" || commService.lastWebhookListReq.Limit != 5 {
		t.Fatalf("unexpected webhook receipt request payload: %+v", commService.lastWebhookListReq)
	}

	ingestReceipts, err := client.CommIngestReceipts(context.Background(), CommIngestReceiptListRequest{
		WorkspaceID: "ws1",
		Source:      "apple_mail_rule",
		TrustState:  "accepted",
		Limit:       5,
	}, "corr-ingest-receipts")
	if err != nil {
		t.Fatalf("comm ingest receipts: %v", err)
	}
	if len(ingestReceipts.Items) != 1 || ingestReceipts.Items[0].ReceiptID != "ir-1" {
		t.Fatalf("unexpected ingest receipts payload: %+v", ingestReceipts)
	}
	if len(ingestReceipts.Items[0].AuditLinks) != 1 {
		t.Fatalf("expected ingest receipt audit links, got %+v", ingestReceipts.Items[0].AuditLinks)
	}
	if commService.lastIngestReceiptReq.Source != "apple_mail_rule" || commService.lastIngestReceiptReq.Limit != 5 {
		t.Fatalf("unexpected ingest receipt request payload: %+v", commService.lastIngestReceiptReq)
	}
}

func TestTransportCommPolicyUpdateAndAttemptHistoryRoutesNotImplementedWithoutCommService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "test-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "test-token",
	})
	if err != nil {
		t.Fatalf("create comm test client: %v", err)
	}

	_, err = client.CommPolicySet(context.Background(), CommPolicySetRequest{
		PolicyID:       "policy-1",
		WorkspaceID:    "ws1",
		SourceChannel:  "imessage",
		PrimaryChannel: "sms",
	}, "corr-policy-update")
	if err == nil {
		t.Fatalf("expected comm policy set error when comm service is not configured")
	}
	httpErr := HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.CommSend(context.Background(), CommSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
		Message:     "reply",
	}, "corr-comm-send")
	if err == nil {
		t.Fatalf("expected comm send error when comm service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501 for comm send, got %d", httpErr.StatusCode)
	}

	_, err = client.CommAttempts(context.Background(), CommAttemptsRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-1",
	}, "corr-attempt-history")
	if err == nil {
		t.Fatalf("expected comm attempts error when comm service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.CommWebhookReceipts(context.Background(), CommWebhookReceiptListRequest{
		WorkspaceID: "ws1",
		Limit:       5,
	}, "corr-webhook-receipts")
	if err == nil {
		t.Fatalf("expected comm webhook receipts error when comm service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}
