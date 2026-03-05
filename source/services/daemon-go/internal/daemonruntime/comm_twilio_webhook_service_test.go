package daemonruntime

import (
	"context"
	"testing"

	"personalagent/runtime/internal/transport"
)

type twilioWebhookAssistantChatStub struct {
	request transport.ChatTurnRequest
	calls   int
}

func (s *twilioWebhookAssistantChatStub) ChatTurn(
	_ context.Context,
	request transport.ChatTurnRequest,
	_ string,
	_ func(delta string),
) (transport.ChatTurnResponse, error) {
	s.calls++
	s.request = request
	return transport.ChatTurnResponse{
		WorkspaceID: request.WorkspaceID,
		TaskClass:   request.TaskClass,
		Provider:    "stub-provider",
		ModelKey:    "stub-model",
		Items: []transport.ChatTurnItem{
			{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: "stubbed assistant reply",
			},
		},
	}, nil
}

func TestGenerateThreadAssistantReplyUsesInjectedChatServiceWithChannelContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new comm twilio service: %v", err)
	}

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, connector_id, external_ref, title, created_at, updated_at)
		 VALUES ('thread-1', 'ws1', 'message', 'twilio', 'thread:1', 'Thread 1', '2026-02-25T00:00:00Z', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-1', 'ws1', 'thread-1', 'twilio', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:01Z', 'hello inbound', '2026-02-25T00:00:01Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, connector_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-2', 'ws1', 'thread-1', 'twilio', 'MESSAGE', 'OUTBOUND', 1, '2026-02-25T00:00:02Z', 'older assistant message', '2026-02-25T00:00:02Z')`,
	}
	for _, statement := range statements {
		if _, err := container.DB.Exec(statement); err != nil {
			t.Fatalf("seed webhook assistant fixture failed: %v\nstatement: %s", err, statement)
		}
	}

	chatStub := &twilioWebhookAssistantChatStub{}
	service.SetAssistantChatService(chatStub)

	reply, err := service.generateThreadAssistantReply(
		context.Background(),
		"ws1",
		"message",
		"thread-1",
		"twilio",
		twilioWebhookAssistantOptions{
			TaskClass:    "chat",
			SystemPrompt: "reply like a human assistant",
			MaxHistory:   10,
		},
	)
	if err != nil {
		t.Fatalf("generate thread assistant reply: %v", err)
	}
	if reply != "stubbed assistant reply" {
		t.Fatalf("expected stubbed assistant reply, got %q", reply)
	}
	if chatStub.calls != 1 {
		t.Fatalf("expected one chat turn call, got %d", chatStub.calls)
	}
	if chatStub.request.Channel.ChannelID != "message" {
		t.Fatalf("expected channel_id=message, got %q", chatStub.request.Channel.ChannelID)
	}
	if chatStub.request.Channel.ConnectorID != "twilio" {
		t.Fatalf("expected connector_id=twilio, got %q", chatStub.request.Channel.ConnectorID)
	}
	if chatStub.request.Channel.ThreadID != "thread-1" {
		t.Fatalf("expected thread_id=thread-1, got %q", chatStub.request.Channel.ThreadID)
	}
	if chatStub.request.SystemPrompt != "reply like a human assistant" {
		t.Fatalf("expected system prompt forwarded, got %q", chatStub.request.SystemPrompt)
	}
	if chatStub.request.TaskClass != "chat" {
		t.Fatalf("expected task class chat, got %q", chatStub.request.TaskClass)
	}
	if len(chatStub.request.Items) != 2 {
		t.Fatalf("expected two historical turn items, got %d", len(chatStub.request.Items))
	}
	if chatStub.request.Items[0].Role != "user" || chatStub.request.Items[0].Content != "hello inbound" {
		t.Fatalf("expected first history item to be inbound user event, got %+v", chatStub.request.Items[0])
	}
	if chatStub.request.Items[1].Role != "assistant" || chatStub.request.Items[1].Content != "older assistant message" {
		t.Fatalf("expected second history item to be prior assistant event, got %+v", chatStub.request.Items[1])
	}
}
