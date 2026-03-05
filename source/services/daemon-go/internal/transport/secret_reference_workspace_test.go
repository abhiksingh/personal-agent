package transport

import (
	"context"
	"errors"
	"testing"
)

func TestInMemorySecretReferenceServiceNormalizesCanonicalWorkspace(t *testing.T) {
	service := NewInMemorySecretReferenceService()

	record, err := service.UpsertSecretReference(context.Background(), SecretReferenceUpsertRequest{
		WorkspaceID: "",
		Name:        "OPENAI_API_KEY",
		Backend:     "memory",
		Service:     "personal-agent.ws1",
		Account:     "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("upsert secret reference: %v", err)
	}
	if record.WorkspaceID != "ws1" {
		t.Fatalf("expected canonical workspace ws1, got %q", record.WorkspaceID)
	}

	loaded, err := service.GetSecretReference(context.Background(), "ws1", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("get secret reference by canonical workspace: %v", err)
	}
	if loaded.WorkspaceID != "ws1" {
		t.Fatalf("expected loaded workspace ws1, got %q", loaded.WorkspaceID)
	}

	if _, err := service.GetSecretReference(context.Background(), "default", "OPENAI_API_KEY"); !errors.Is(err, ErrSecretReferenceNotFound) {
		t.Fatalf("expected default workspace lookup to remain isolated, got %v", err)
	}

	deleted, err := service.DeleteSecretReference(context.Background(), "", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("delete secret reference by empty workspace: %v", err)
	}
	if deleted.WorkspaceID != "ws1" {
		t.Fatalf("expected deleted workspace ws1, got %q", deleted.WorkspaceID)
	}
}
