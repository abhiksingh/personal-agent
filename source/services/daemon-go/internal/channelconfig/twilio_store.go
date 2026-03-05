package channelconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/endpointpolicy"
	"personalagent/runtime/internal/workspaceid"
)

const (
	connectorTypeTwilio          = "channel.twilio"
	ownerTypeChannelTwilio       = "CHANNEL_TWILIO"
	ownerIDChannelTwilio         = "twilio"
	defaultTwilioEndpointBaseURL = "https://api.twilio.com"
)

var ErrTwilioNotConfigured = errors.New("twilio channel not configured")

type TwilioConfig struct {
	WorkspaceID           string    `json:"workspace_id"`
	AccountSIDSecretName  string    `json:"account_sid_secret_name"`
	AuthTokenSecretName   string    `json:"auth_token_secret_name"`
	SMSNumber             string    `json:"sms_number"`
	VoiceNumber           string    `json:"voice_number"`
	Endpoint              string    `json:"endpoint"`
	AccountSIDConfigured  bool      `json:"account_sid_configured"`
	AuthTokenConfigured   bool      `json:"auth_token_configured"`
	CredentialsConfigured bool      `json:"credentials_configured"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type TwilioUpsertInput struct {
	WorkspaceID string

	AccountSIDSecretName string
	AuthTokenSecretName  string

	AccountSIDKeychainService string
	AccountSIDKeychainAccount string
	AuthTokenKeychainService  string
	AuthTokenKeychainAccount  string

	SMSNumber   string
	VoiceNumber string
	Endpoint    string
}

type twilioConnectorConfig struct {
	AccountSIDSecretName string `json:"account_sid_secret_name"`
	AuthTokenSecretName  string `json:"auth_token_secret_name"`
	SMSNumber            string `json:"sms_number"`
	VoiceNumber          string `json:"voice_number"`
	Endpoint             string `json:"endpoint"`
}

type SQLiteTwilioStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLiteTwilioStore(db *sql.DB) *SQLiteTwilioStore {
	return &SQLiteTwilioStore{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func DefaultTwilioEndpoint() string {
	return defaultTwilioEndpointBaseURL
}

func (s *SQLiteTwilioStore) Upsert(ctx context.Context, input TwilioUpsertInput) (TwilioConfig, error) {
	workspaceID := normalizeWorkspaceID(input.WorkspaceID)

	accountSIDSecretName := strings.TrimSpace(input.AccountSIDSecretName)
	authTokenSecretName := strings.TrimSpace(input.AuthTokenSecretName)
	if accountSIDSecretName == "" {
		return TwilioConfig{}, fmt.Errorf("account sid secret name is required")
	}
	if authTokenSecretName == "" {
		return TwilioConfig{}, fmt.Errorf("auth token secret name is required")
	}

	if strings.TrimSpace(input.AccountSIDKeychainService) == "" || strings.TrimSpace(input.AccountSIDKeychainAccount) == "" {
		return TwilioConfig{}, fmt.Errorf("account sid keychain service/account are required")
	}
	if strings.TrimSpace(input.AuthTokenKeychainService) == "" || strings.TrimSpace(input.AuthTokenKeychainAccount) == "" {
		return TwilioConfig{}, fmt.Errorf("auth token keychain service/account are required")
	}

	smsNumber := strings.TrimSpace(input.SMSNumber)
	voiceNumber := strings.TrimSpace(input.VoiceNumber)
	if smsNumber == "" {
		return TwilioConfig{}, fmt.Errorf("sms number is required")
	}
	if voiceNumber == "" {
		return TwilioConfig{}, fmt.Errorf("voice number is required")
	}

	connectorConfig := twilioConnectorConfig{
		AccountSIDSecretName: accountSIDSecretName,
		AuthTokenSecretName:  authTokenSecretName,
		SMSNumber:            smsNumber,
		VoiceNumber:          voiceNumber,
	}
	resolvedEndpoint, err := normalizeTwilioEndpoint(input.Endpoint)
	if err != nil {
		return TwilioConfig{}, err
	}
	connectorConfig.Endpoint = resolvedEndpoint
	configJSON, err := json.Marshal(connectorConfig)
	if err != nil {
		return TwilioConfig{}, fmt.Errorf("marshal twilio config: %w", err)
	}

	now := s.now().UTC().Format(time.RFC3339Nano)
	connectorID := twilioConnectorID(workspaceID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TwilioConfig{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, now); err != nil {
		return TwilioConfig{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO channel_connectors (
			id,
			workspace_id,
			connector_type,
			status,
			config_json,
			created_at,
			updated_at
		) VALUES (?, ?, ?, 'ACTIVE', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			config_json = excluded.config_json,
			updated_at = excluded.updated_at
	`, connectorID, workspaceID, connectorTypeTwilio, string(configJSON), now, now); err != nil {
		return TwilioConfig{}, fmt.Errorf("upsert twilio config: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
	`, workspaceID, ownerTypeChannelTwilio, ownerIDChannelTwilio); err != nil {
		return TwilioConfig{}, fmt.Errorf("clear twilio secret refs: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_refs (
			id,
			workspace_id,
			owner_type,
			owner_id,
			keychain_account,
			keychain_service,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, twilioSecretRefID(workspaceID, strings.TrimSpace(input.AccountSIDKeychainAccount)), workspaceID, ownerTypeChannelTwilio, ownerIDChannelTwilio, strings.TrimSpace(input.AccountSIDKeychainAccount), strings.TrimSpace(input.AccountSIDKeychainService), now); err != nil {
		return TwilioConfig{}, fmt.Errorf("insert twilio account sid secret ref: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_refs (
			id,
			workspace_id,
			owner_type,
			owner_id,
			keychain_account,
			keychain_service,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, twilioSecretRefID(workspaceID, strings.TrimSpace(input.AuthTokenKeychainAccount)), workspaceID, ownerTypeChannelTwilio, ownerIDChannelTwilio, strings.TrimSpace(input.AuthTokenKeychainAccount), strings.TrimSpace(input.AuthTokenKeychainService), now); err != nil {
		return TwilioConfig{}, fmt.Errorf("insert twilio auth token secret ref: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return TwilioConfig{}, fmt.Errorf("commit tx: %w", err)
	}

	return s.Get(ctx, workspaceID)
}

func (s *SQLiteTwilioStore) Get(ctx context.Context, workspaceID string) (TwilioConfig, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	return s.loadByWorkspace(ctx, workspace, workspace)
}

func (s *SQLiteTwilioStore) loadByWorkspace(ctx context.Context, sourceWorkspace string, responseWorkspace string) (TwilioConfig, error) {
	var (
		configJSON   string
		updatedAtRaw string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(config_json, ''), updated_at
		FROM channel_connectors
		WHERE workspace_id = ?
		  AND connector_type = ?
		LIMIT 1
	`, sourceWorkspace, connectorTypeTwilio).Scan(&configJSON, &updatedAtRaw)
	if err == sql.ErrNoRows {
		return TwilioConfig{}, ErrTwilioNotConfigured
	}
	if err != nil {
		return TwilioConfig{}, fmt.Errorf("query twilio config: %w", err)
	}

	parsedConfig := twilioConnectorConfig{Endpoint: defaultTwilioEndpointBaseURL}
	if strings.TrimSpace(configJSON) != "" {
		if err := json.Unmarshal([]byte(configJSON), &parsedConfig); err != nil {
			return TwilioConfig{}, fmt.Errorf("decode twilio config JSON: %w", err)
		}
	}
	parsedConfig.AccountSIDSecretName = strings.TrimSpace(parsedConfig.AccountSIDSecretName)
	parsedConfig.AuthTokenSecretName = strings.TrimSpace(parsedConfig.AuthTokenSecretName)
	parsedConfig.SMSNumber = strings.TrimSpace(parsedConfig.SMSNumber)
	parsedConfig.VoiceNumber = strings.TrimSpace(parsedConfig.VoiceNumber)
	resolvedEndpoint, err := normalizeTwilioEndpoint(parsedConfig.Endpoint)
	if err != nil {
		return TwilioConfig{}, err
	}
	parsedConfig.Endpoint = resolvedEndpoint

	configuredAccounts, err := s.loadConfiguredSecretAccounts(ctx, sourceWorkspace)
	if err != nil {
		return TwilioConfig{}, err
	}

	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return TwilioConfig{}, err
	}

	accountSIDConfigured := configuredAccounts[parsedConfig.AccountSIDSecretName]
	authTokenConfigured := configuredAccounts[parsedConfig.AuthTokenSecretName]

	return TwilioConfig{
		WorkspaceID:           responseWorkspace,
		AccountSIDSecretName:  parsedConfig.AccountSIDSecretName,
		AuthTokenSecretName:   parsedConfig.AuthTokenSecretName,
		SMSNumber:             parsedConfig.SMSNumber,
		VoiceNumber:           parsedConfig.VoiceNumber,
		Endpoint:              parsedConfig.Endpoint,
		AccountSIDConfigured:  accountSIDConfigured,
		AuthTokenConfigured:   authTokenConfigured,
		CredentialsConfigured: accountSIDConfigured && authTokenConfigured,
		UpdatedAt:             updatedAt,
	}, nil
}

func (s *SQLiteTwilioStore) loadConfiguredSecretAccounts(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT keychain_account
		FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
	`, workspaceID, ownerTypeChannelTwilio, ownerIDChannelTwilio)
	if err != nil {
		return nil, fmt.Errorf("query twilio secret refs: %w", err)
	}
	defer rows.Close()

	configured := map[string]bool{}
	for rows.Next() {
		var account string
		if err := rows.Scan(&account); err != nil {
			return nil, fmt.Errorf("scan twilio secret ref: %w", err)
		}
		configured[strings.TrimSpace(account)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate twilio secret refs: %w", err)
	}
	return configured, nil
}

func ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string, now string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces (
			id,
			name,
			status,
			created_at,
			updated_at
		) VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("missing updated_at for twilio config")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse updated_at %q: %w", value, err)
	}
	return parsed, nil
}

func normalizeWorkspaceID(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func normalizeTwilioEndpoint(endpoint string) (string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		trimmed = defaultTwilioEndpointBaseURL
	}
	parsed, err := endpointpolicy.ParseAndValidate(trimmed, endpointpolicy.Options{Service: "twilio endpoint"})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func twilioConnectorID(workspaceID string) string {
	return fmt.Sprintf("channel:twilio:%s", workspaceID)
}

func twilioSecretRefID(workspaceID string, keychainAccount string) string {
	return fmt.Sprintf("channel-secret:twilio:%s:%s", workspaceID, strings.TrimSpace(keychainAccount))
}
