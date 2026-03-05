package connectorflow

import (
	"context"
	"testing"

	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/types"
)

func TestExecuteMailHappyPathDraftSendReplyWithTraceEvidence(t *testing.T) {
	registry := connectorregistry.New()
	if err := registry.Register(mailadapter.NewAdapter("mail.mock")); err != nil {
		t.Fatalf("register mail adapter: %v", err)
	}

	service := NewMailHappyPathService(registry)
	result, err := service.Execute(context.Background(), types.MailHappyPathRequest{
		WorkspaceID:      "ws_mail",
		RunID:            "run_mail_happy_path",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_mail_happy_path",
	})
	if err != nil {
		t.Fatalf("execute mail happy path: %v", err)
	}

	if result.DraftTrace.CapabilityKey != "mail_draft" {
		t.Fatalf("expected draft capability mail_draft, got %s", result.DraftTrace.CapabilityKey)
	}
	if result.SendTrace.CapabilityKey != "mail_send" {
		t.Fatalf("expected send capability mail_send, got %s", result.SendTrace.CapabilityKey)
	}
	if result.ReplyTrace.CapabilityKey != "mail_reply" {
		t.Fatalf("expected reply capability mail_reply, got %s", result.ReplyTrace.CapabilityKey)
	}

	if result.DraftTrace.AdapterID != "mail.mock" || result.SendTrace.AdapterID != "mail.mock" || result.ReplyTrace.AdapterID != "mail.mock" {
		t.Fatalf("expected all steps to execute via mail.mock adapter")
	}

	if result.DraftTrace.Evidence["draft_id"] == "" {
		t.Fatalf("expected draft trace evidence to include draft_id")
	}
	if result.SendTrace.Evidence["message_id"] == "" {
		t.Fatalf("expected send trace evidence to include message_id")
	}
	if result.ReplyTrace.Evidence["reply_id"] == "" || result.ReplyTrace.Evidence["thread_id"] == "" {
		t.Fatalf("expected reply trace evidence to include reply_id and thread_id")
	}
}
