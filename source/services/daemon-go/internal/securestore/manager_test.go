package securestore

import "testing"

func TestManagerPutGetDelete(t *testing.T) {
	manager, err := NewManager("personal-agent", "memory", NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	ref, err := manager.Put("ws1", "OPENAI_API_KEY", "secret-value")
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	if ref.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", ref.WorkspaceID)
	}

	resolvedRef, value, err := manager.Get("ws1", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if resolvedRef.Service == "" || resolvedRef.Account == "" {
		t.Fatalf("expected populated secret reference")
	}
	if value != "secret-value" {
		t.Fatalf("expected secret value, got %q", value)
	}

	if _, err := manager.Delete("ws1", "OPENAI_API_KEY"); err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	if _, _, err := manager.Get("ws1", "OPENAI_API_KEY"); err == nil {
		t.Fatalf("expected not found error after delete")
	}
}

func TestManagerNormalizesDefaultWorkspace(t *testing.T) {
	manager, err := NewManager("personal-agent", "memory", NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	ref, err := manager.Put("", "OLLAMA_ENDPOINT", "http://localhost:11434")
	if err != nil {
		t.Fatalf("put default workspace secret: %v", err)
	}
	if ref.WorkspaceID != "ws1" {
		t.Fatalf("expected canonical default workspace ws1, got %s", ref.WorkspaceID)
	}
}
