package daemonruntime

import (
	"context"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

func TestCapabilityGrantUpsertAndListSupportsPaging(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	first, err := service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "messages_send_sms",
		ScopeJSON:     `{"channel":"sms"}`,
		Status:        "ACTIVE",
		ExpiresAt:     time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("upsert first capability grant: %v", err)
	}
	if first.GrantID == "" || first.Status != "ACTIVE" || first.ActorID != "actor.requester" {
		t.Fatalf("unexpected first capability grant payload: %+v", first)
	}

	second, err := service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "calendar_create",
		Status:        "ACTIVE",
	})
	if err != nil {
		t.Fatalf("upsert second capability grant: %v", err)
	}
	if second.GrantID == "" || second.CapabilityKey != "calendar_create" {
		t.Fatalf("unexpected second capability grant payload: %+v", second)
	}

	updated, err := service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID: "ws1",
		GrantID:     first.GrantID,
		Status:      "DISABLED",
	})
	if err != nil {
		t.Fatalf("update capability grant status: %v", err)
	}
	if updated.GrantID != first.GrantID || updated.Status != "DISABLED" || updated.ScopeJSON != `{"channel":"sms"}` {
		t.Fatalf("unexpected updated capability grant payload: %+v", updated)
	}

	page1, err := service.ListCapabilityGrants(context.Background(), transport.CapabilityGrantListRequest{
		WorkspaceID: "ws1",
		ActorID:     "actor.requester",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list capability grants page 1: %v", err)
	}
	if len(page1.Items) != 1 || page1.Items[0].GrantID != second.GrantID || !page1.HasMore {
		t.Fatalf("unexpected capability grant page 1 payload: %+v", page1)
	}

	page2, err := service.ListCapabilityGrants(context.Background(), transport.CapabilityGrantListRequest{
		WorkspaceID:     "ws1",
		ActorID:         "actor.requester",
		CursorCreatedAt: page1.NextCursorCreatedAt,
		CursorID:        page1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("list capability grants page 2: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].GrantID != first.GrantID {
		t.Fatalf("unexpected capability grant page 2 payload: %+v", page2)
	}

	filtered, err := service.ListCapabilityGrants(context.Background(), transport.CapabilityGrantListRequest{
		WorkspaceID: "ws1",
		Status:      "disabled",
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("list filtered capability grants: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].GrantID != first.GrantID || filtered.Items[0].Status != "DISABLED" {
		t.Fatalf("unexpected filtered capability grant payload: %+v", filtered)
	}

	var auditCount int
	if err := container.DB.QueryRow(`
		SELECT COUNT(*)
		FROM audit_log_entries
		WHERE workspace_id = 'ws1' AND event_type = 'CAPABILITY_GRANT_UPSERTED'
	`).Scan(&auditCount); err != nil {
		t.Fatalf("query capability-grant audit count: %v", err)
	}
	if auditCount < 3 {
		t.Fatalf("expected capability-grant audit entries >= 3, got %d", auditCount)
	}
}

func TestCapabilityGrantValidationErrors(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	_, err = service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "messages_send_sms",
		ScopeJSON:     "{invalid-json",
	})
	if err == nil || !strings.Contains(err.Error(), "scope_json must be valid json") {
		t.Fatalf("expected scope_json validation error, got %v", err)
	}

	_, err = service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID: "ws1",
		GrantID:     "missing-grant-id",
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing grant-id validation error, got %v", err)
	}

	_, err = service.UpsertCapabilityGrant(context.Background(), transport.CapabilityGrantUpsertRequest{
		WorkspaceID:   "ws1",
		ActorID:       "actor.requester",
		CapabilityKey: "messages_send_sms",
		Status:        "BROKEN",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported capability grant status") {
		t.Fatalf("expected status validation error, got %v", err)
	}

	_, err = service.ListCapabilityGrants(context.Background(), transport.CapabilityGrantListRequest{
		WorkspaceID: "ws1",
		CursorID:    "grant-1",
		Limit:       10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at is required") {
		t.Fatalf("expected cursor validation error, got %v", err)
	}
}
