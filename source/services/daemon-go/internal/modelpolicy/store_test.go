package modelpolicy

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"personalagent/runtime/internal/providerconfig"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupModelPolicyDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "model-policy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

func TestNormalizeWorkspaceIDCanonicalizesEmptyWorkspace(t *testing.T) {
	if got := normalizeWorkspaceID(""); got != "ws1" {
		t.Fatalf("expected empty workspace to normalize to ws1, got %q", got)
	}
	if got := normalizeWorkspaceID("default"); got != "default" {
		t.Fatalf("expected explicit workspace default to be preserved, got %q", got)
	}
}

func TestSQLiteStoreSeedsDefaultsAndListsCatalog(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries, err := store.ListCatalog(ctx, "ws1", "")
	if err != nil {
		t.Fatalf("list catalog: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected seeded default catalog entries")
	}
}

func TestSQLiteStoreSetModelEnabled(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := store.SetModelEnabled(ctx, "ws1", providerconfig.ProviderOpenAI, "gpt-4.1-mini", false)
	if err != nil {
		t.Fatalf("disable model: %v", err)
	}
	if entry.Enabled {
		t.Fatalf("expected disabled model")
	}
}

func TestSQLiteStoreSetAndGetRoutingPolicy(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	policy, err := store.SetRoutingPolicy(ctx, "ws1", "chat", providerconfig.ProviderOllama, "llama3.2")
	if err != nil {
		t.Fatalf("set routing policy: %v", err)
	}
	if policy.TaskClass != "chat" {
		t.Fatalf("expected task class chat, got %s", policy.TaskClass)
	}

	got, err := store.GetRoutingPolicy(ctx, "ws1", "chat")
	if err != nil {
		t.Fatalf("get routing policy: %v", err)
	}
	if got.Provider != providerconfig.ProviderOllama || got.ModelKey != "llama3.2" {
		t.Fatalf("unexpected routing policy %+v", got)
	}
}

func TestSQLiteStoreRejectsPolicyForDisabledModel(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := store.SetModelEnabled(ctx, "ws1", providerconfig.ProviderOllama, "mistral", false); err != nil {
		t.Fatalf("disable model: %v", err)
	}

	_, err := store.SetRoutingPolicy(ctx, "ws1", "chat", providerconfig.ProviderOllama, "mistral")
	if err == nil {
		t.Fatalf("expected error when selecting disabled model")
	}
}

func TestSQLiteStoreGetRoutingPolicyNotFound(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.GetRoutingPolicy(ctx, "ws1", "missing")
	if !errors.Is(err, ErrRoutingPolicyNotFound) {
		t.Fatalf("expected ErrRoutingPolicyNotFound, got %v", err)
	}
}

func TestSQLiteStoreAddCatalogEntryAllowsDynamicModel(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := store.AddCatalogEntry(ctx, "ws1", providerconfig.ProviderOpenAI, "gpt-5-codex", false)
	if err != nil {
		t.Fatalf("add catalog entry: %v", err)
	}
	if entry.Enabled {
		t.Fatalf("expected added model to be disabled by default")
	}

	enabled, err := store.SetModelEnabled(ctx, "ws1", providerconfig.ProviderOpenAI, "gpt-5-codex", true)
	if err != nil {
		t.Fatalf("enable dynamic model: %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected enabled dynamic model")
	}
}

func TestSQLiteStoreRemoveCatalogEntryPreventsDefaultReseed(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := store.RemoveCatalogEntry(ctx, "ws1", providerconfig.ProviderOpenAI, "gpt-4.1-mini"); err != nil {
		t.Fatalf("remove seeded model: %v", err)
	}

	entries, err := store.ListCatalog(ctx, "ws1", providerconfig.ProviderOpenAI)
	if err != nil {
		t.Fatalf("list catalog after remove: %v", err)
	}
	for _, entry := range entries {
		if entry.ModelKey == "gpt-4.1-mini" {
			t.Fatalf("expected removed default model to stay removed")
		}
	}
}

func TestSQLiteStoreRemoveCatalogEntryRemovesRoutingPolicy(t *testing.T) {
	db := setupModelPolicyDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := store.SetRoutingPolicy(ctx, "ws1", "chat", providerconfig.ProviderOpenAI, "gpt-4.1"); err != nil {
		t.Fatalf("set routing policy: %v", err)
	}

	if _, err := store.RemoveCatalogEntry(ctx, "ws1", providerconfig.ProviderOpenAI, "gpt-4.1"); err != nil {
		t.Fatalf("remove policy model: %v", err)
	}

	_, err := store.GetRoutingPolicy(ctx, "ws1", "chat")
	if !errors.Is(err, ErrRoutingPolicyNotFound) {
		t.Fatalf("expected routing policy to be removed, got %v", err)
	}
}
