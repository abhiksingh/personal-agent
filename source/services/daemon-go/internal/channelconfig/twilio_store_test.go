package channelconfig

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupTwilioConfigDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "twilio-config.db")
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

func TestSQLiteTwilioStoreUpsertGet(t *testing.T) {
	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := store.Upsert(ctx, TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent.ws1",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent.ws1",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "https://api.twilio.test",
	})
	if err != nil {
		t.Fatalf("upsert twilio config: %v", err)
	}
	if config.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", config.WorkspaceID)
	}
	if config.Endpoint != "https://api.twilio.test" {
		t.Fatalf("expected endpoint override, got %s", config.Endpoint)
	}
	if !config.CredentialsConfigured {
		t.Fatalf("expected credentials configured=true")
	}

	got, err := store.Get(ctx, "ws1")
	if err != nil {
		t.Fatalf("get twilio config: %v", err)
	}
	if got.AccountSIDSecretName != "TWILIO_ACCOUNT_SID" {
		t.Fatalf("expected account sid secret name, got %s", got.AccountSIDSecretName)
	}
	if got.AuthTokenSecretName != "TWILIO_AUTH_TOKEN" {
		t.Fatalf("expected auth token secret name, got %s", got.AuthTokenSecretName)
	}
}

func TestSQLiteTwilioStoreGetNotFound(t *testing.T) {
	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Get(ctx, "ws1")
	if !errors.Is(err, ErrTwilioNotConfigured) {
		t.Fatalf("expected ErrTwilioNotConfigured, got %v", err)
	}
}

func TestSQLiteTwilioStoreGetDoesNotFallbackToLegacyWorkspaceRows(t *testing.T) {
	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

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
			'twilio.default',
			'default',
			'channel.twilio',
			'ACTIVE',
			'{"account_sid_secret_name":"TWILIO_ACCOUNT_SID","auth_token_secret_name":"TWILIO_AUTH_TOKEN","sms_number":"+15555550001","voice_number":"+15555550002","endpoint":"https://api.twilio.com"}',
			'2026-02-26T00:00:00Z',
			'2026-02-26T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed legacy twilio connector: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO secret_refs(
			id, workspace_id, owner_type, owner_id, keychain_account, keychain_service, created_at
		) VALUES
			('sref.default.twilio.sid', 'default', 'CHANNEL_TWILIO', 'twilio', 'TWILIO_ACCOUNT_SID', 'personal-agent.default', '2026-02-26T00:00:00Z'),
			('sref.default.twilio.token', 'default', 'CHANNEL_TWILIO', 'twilio', 'TWILIO_AUTH_TOKEN', 'personal-agent.default', '2026-02-26T00:00:00Z')
	`); err != nil {
		t.Fatalf("seed legacy twilio secret refs: %v", err)
	}

	_, err := store.Get(ctx, "ws1")
	if !errors.Is(err, ErrTwilioNotConfigured) {
		t.Fatalf("expected ErrTwilioNotConfigured when only default workspace rows exist, got %v", err)
	}
}

func TestSQLiteTwilioStoreUpsertRejectsInsecureNonLoopbackEndpointByDefault(t *testing.T) {
	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Upsert(ctx, TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent.ws1",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent.ws1",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "http://api.twilio.com",
	})
	if err == nil {
		t.Fatalf("expected insecure non-loopback endpoint to be rejected")
	}
}

func TestSQLiteTwilioStoreUpsertRejectsPrivateEndpointByDefault(t *testing.T) {
	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Upsert(ctx, TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent.ws1",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent.ws1",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "https://10.0.0.20",
	})
	if err == nil {
		t.Fatalf("expected private endpoint to be rejected")
	}
}

func TestSQLiteTwilioStoreUpsertAllowsPrivateInsecureEndpointWithOptIns(t *testing.T) {
	t.Setenv("PA_ALLOW_INSECURE_ENDPOINTS", "1")
	t.Setenv("PA_ALLOW_PRIVATE_ENDPOINTS", "1")

	db := setupTwilioConfigDB(t)
	store := NewSQLiteTwilioStore(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := store.Upsert(ctx, TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent.ws1",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent.ws1",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "http://10.0.0.20",
	})
	if err != nil {
		t.Fatalf("expected endpoint with explicit opt-ins to be accepted: %v", err)
	}
	if config.Endpoint != "http://10.0.0.20" {
		t.Fatalf("expected endpoint override, got %q", config.Endpoint)
	}
}
