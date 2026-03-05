package providerconfig

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupProviderConfigDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "provider-config.db")
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

func TestSQLiteStoreUpsertGetList(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID:      "ws1",
		Provider:         ProviderOpenAI,
		Endpoint:         "https://api.openai.com/v1",
		APIKeySecretName: "OPENAI_API_KEY",
		KeychainService:  "personal-agent.ws1",
		KeychainAccount:  "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("upsert openai config: %v", err)
	}
	if config.Provider != ProviderOpenAI {
		t.Fatalf("expected provider openai, got %s", config.Provider)
	}
	if !config.APIKeyConfigured {
		t.Fatalf("expected api key configured=true")
	}

	got, err := store.Get(ctx, "ws1", ProviderOpenAI)
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if got.APIKeySecretName != "OPENAI_API_KEY" {
		t.Fatalf("expected secret name OPENAI_API_KEY, got %s", got.APIKeySecretName)
	}

	list, err := store.List(ctx, "ws1")
	if err != nil {
		t.Fatalf("list configs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one provider config, got %d", len(list))
	}
}

func TestSQLiteStoreReplacesSecretRefWhenCleared(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID:      "ws1",
		Provider:         ProviderOllama,
		Endpoint:         "http://127.0.0.1:11434",
		APIKeySecretName: "OLLAMA_API_KEY",
		KeychainService:  "personal-agent.ws1",
		KeychainAccount:  "OLLAMA_API_KEY",
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	updated, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID: "ws1",
		Provider:    ProviderOllama,
		Endpoint:    "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("clear secret upsert: %v", err)
	}
	if updated.APIKeyConfigured {
		t.Fatalf("expected api key configured=false after clear")
	}
	if updated.APIKeySecretName != "" {
		t.Fatalf("expected empty secret name after clear, got %q", updated.APIKeySecretName)
	}
}

func TestSQLiteStoreGetReturnsNotFound(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Get(ctx, "ws1", ProviderOpenAI)
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestSQLiteStoreListDoesNotFallbackToLegacyWorkspaceRows(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed legacy workspace: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO channel_connectors(
			id, workspace_id, connector_type, status, config_json, created_at, updated_at
		) VALUES (
			'provider.default.openai',
			'default',
			'model_provider.openai',
			'ACTIVE',
			'{"provider":"openai","endpoint":"https://api.openai.com/v1","api_key_secret_name":"OPENAI_API_KEY"}',
			'2026-02-26T00:00:00Z',
			'2026-02-26T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed legacy provider connector: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO secret_refs(
			id, workspace_id, owner_type, owner_id, keychain_account, keychain_service, created_at
		) VALUES (
			'sref.default.openai',
			'default',
			'MODEL_PROVIDER',
			'openai',
			'OPENAI_API_KEY',
			'personal-agent.default',
			'2026-02-26T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed legacy provider secret ref: %v", err)
	}

	configs, err := store.List(ctx, "ws1")
	if err != nil {
		t.Fatalf("list provider configs for canonical workspace: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected no configs for ws1 when only default workspace rows exist, got %d", len(configs))
	}
}

func TestProviderMetadataParity(t *testing.T) {
	tests := []struct {
		provider        string
		defaultEndpoint string
		requiresAPIKey  bool
	}{
		{
			provider:        ProviderOpenAI,
			defaultEndpoint: "https://api.openai.com/v1",
			requiresAPIKey:  true,
		},
		{
			provider:        ProviderAnthropic,
			defaultEndpoint: "https://api.anthropic.com/v1",
			requiresAPIKey:  true,
		},
		{
			provider:        ProviderGoogle,
			defaultEndpoint: "https://generativelanguage.googleapis.com/v1beta",
			requiresAPIKey:  true,
		},
		{
			provider:        ProviderOllama,
			defaultEndpoint: "http://127.0.0.1:11434",
			requiresAPIKey:  false,
		},
	}

	for _, tc := range tests {
		normalized, err := NormalizeProvider(strings.ToUpper(tc.provider))
		if err != nil {
			t.Fatalf("normalize provider %s: %v", tc.provider, err)
		}
		if normalized != tc.provider {
			t.Fatalf("expected normalized provider %s, got %s", tc.provider, normalized)
		}
		if got := DefaultEndpoint(tc.provider); got != tc.defaultEndpoint {
			t.Fatalf("expected default endpoint %s for provider %s, got %s", tc.defaultEndpoint, tc.provider, got)
		}
		if got := ProviderRequiresAPIKey(tc.provider); got != tc.requiresAPIKey {
			t.Fatalf("expected requires api key=%v for provider %s, got %v", tc.requiresAPIKey, tc.provider, got)
		}
	}
}

func TestSQLiteStoreUpsertRejectsInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID: "ws1",
		Provider:    ProviderOpenAI,
		Endpoint:    "http://api.openai.com/v1",
	})
	if err == nil {
		t.Fatalf("expected insecure non-loopback endpoint to be rejected")
	}
}

func TestSQLiteStoreUpsertRejectsPrivateEndpointByDefault(t *testing.T) {
	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID: "ws1",
		Provider:    ProviderOpenAI,
		Endpoint:    "https://192.168.1.10/v1",
	})
	if err == nil {
		t.Fatalf("expected private endpoint to be rejected")
	}
}

func TestSQLiteStoreUpsertAllowsInsecureAndPrivateEndpointWithOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	db := setupProviderConfigDB(t)
	store := NewSQLiteStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := store.Upsert(ctx, UpsertInput{
		WorkspaceID: "ws1",
		Provider:    ProviderOpenAI,
		Endpoint:    "http://192.168.1.10/v1",
	})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
	if config.Endpoint != "http://192.168.1.10/v1" {
		t.Fatalf("expected endpoint override, got %q", config.Endpoint)
	}
}
