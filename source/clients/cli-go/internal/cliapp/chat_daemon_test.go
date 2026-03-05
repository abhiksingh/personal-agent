package cliapp

import (
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestIsAssistantRealtimeDeltaEvent(t *testing.T) {
	base := transport.RealtimeEventEnvelope{
		EventType:     "turn_item_delta",
		CorrelationID: "corr-1",
		Payload: transport.RealtimeEventPayload{
			ItemType: "assistant_message",
			Delta:    " ",
		},
	}
	if !isAssistantRealtimeDeltaEvent(base, "corr-1") {
		t.Fatalf("expected assistant turn-item delta event to be accepted")
	}

	notAssistant := base
	notAssistant.Payload.ItemType = "tool_result"
	if isAssistantRealtimeDeltaEvent(notAssistant, "corr-1") {
		t.Fatalf("expected non-assistant turn-item delta event to be rejected")
	}

	otherEvent := base
	otherEvent.EventType = "chat_token"
	if isAssistantRealtimeDeltaEvent(otherEvent, "corr-1") {
		t.Fatalf("expected non turn_item_delta event to be rejected")
	}

	missingDelta := base
	missingDelta.Payload.Delta = ""
	if isAssistantRealtimeDeltaEvent(missingDelta, "corr-1") {
		t.Fatalf("expected empty delta payload to be rejected")
	}
}

func TestAssistantMessageFromTurnItemsPreservesWhitespace(t *testing.T) {
	items := []transport.ChatTurnItem{
		{Type: "assistant_message", Content: "  first  "},
		{Type: "assistant_message", Content: "line 1\nline 2"},
	}
	got := assistantMessageFromTurnItems(items)
	if got != "line 1\nline 2" {
		t.Fatalf("expected latest assistant message with exact spacing/newlines, got %q", got)
	}
}
