package providerconfig

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
	ProviderOpenAI               = "openai"
	ProviderAnthropic            = "anthropic"
	ProviderGoogle               = "google"
	ProviderOllama               = "ollama"
	ownerTypeModelProvider       = "MODEL_PROVIDER"
	connectorTypeProviderPrefix  = "model_provider."
	defaultOpenAIEndpointBaseURL = "https://api.openai.com/v1"
	defaultAnthropicEndpointURL  = "https://api.anthropic.com/v1"
	defaultGoogleEndpointURL     = "https://generativelanguage.googleapis.com/v1beta"
	defaultOllamaEndpointBaseURL = "http://127.0.0.1:11434"
)

var ErrProviderNotFound = errors.New("provider not configured")

type Config struct {
	WorkspaceID      string    `json:"workspace_id"`
	Provider         string    `json:"provider"`
	Endpoint         string    `json:"endpoint"`
	APIKeySecretName string    `json:"api_key_secret_name,omitempty"`
	APIKeyConfigured bool      `json:"api_key_configured"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpsertInput struct {
	WorkspaceID      string
	Provider         string
	Endpoint         string
	APIKeySecretName string
	KeychainService  string
	KeychainAccount  string
}

type SQLiteStore struct {
	db  *sql.DB
	now func() time.Time
}

type providerConnectorConfig struct {
	Provider         string `json:"provider"`
	Endpoint         string `json:"endpoint"`
	APIKeySecretName string `json:"api_key_secret_name,omitempty"`
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func NormalizeProvider(provider string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case ProviderOpenAI, ProviderAnthropic, ProviderGoogle, ProviderOllama:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
}

func ProviderRequiresAPIKey(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderOpenAI, ProviderAnthropic, ProviderGoogle:
		return true
	default:
		return false
	}
}

func DefaultEndpoint(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return defaultOpenAIEndpointBaseURL
	case ProviderAnthropic:
		return defaultAnthropicEndpointURL
	case ProviderGoogle:
		return defaultGoogleEndpointURL
	case ProviderOllama:
		return defaultOllamaEndpointBaseURL
	default:
		return ""
	}
}

func (s *SQLiteStore) Upsert(ctx context.Context, input UpsertInput) (Config, error) {
	workspaceID := normalizeWorkspaceID(input.WorkspaceID)
	provider, err := NormalizeProvider(input.Provider)
	if err != nil {
		return Config{}, err
	}

	secretName := strings.TrimSpace(input.APIKeySecretName)
	if secretName != "" {
		if strings.TrimSpace(input.KeychainService) == "" || strings.TrimSpace(input.KeychainAccount) == "" {
			return Config{}, fmt.Errorf("keychain service/account are required when api key secret name is provided")
		}
	}

	connectorConfig := providerConnectorConfig{
		Provider:         provider,
		APIKeySecretName: secretName,
	}
	resolvedEndpoint, err := normalizeEndpoint(provider, input.Endpoint)
	if err != nil {
		return Config{}, err
	}
	connectorConfig.Endpoint = resolvedEndpoint

	configJSON, err := json.Marshal(connectorConfig)
	if err != nil {
		return Config{}, fmt.Errorf("marshal provider config: %w", err)
	}

	now := s.now().Format(time.RFC3339Nano)
	connectorID := providerConnectorID(workspaceID, provider)
	connectorType := connectorTypeForProvider(provider)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Config{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspace(ctx, tx, workspaceID, now); err != nil {
		return Config{}, err
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
	`, connectorID, workspaceID, connectorType, string(configJSON), now, now); err != nil {
		return Config{}, fmt.Errorf("upsert provider config: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
		  AND owner_id = ?
	`, workspaceID, ownerTypeModelProvider, provider); err != nil {
		return Config{}, fmt.Errorf("clear provider secret refs: %w", err)
	}

	if secretName != "" {
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
		`, providerSecretRefID(workspaceID, provider, input.KeychainAccount), workspaceID, ownerTypeModelProvider, provider, input.KeychainAccount, input.KeychainService, now); err != nil {
			return Config{}, fmt.Errorf("insert provider secret ref: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Config{}, fmt.Errorf("commit tx: %w", err)
	}

	return s.Get(ctx, workspaceID, provider)
}

func (s *SQLiteStore) Get(ctx context.Context, workspaceID string, provider string) (Config, error) {
	normalizedWorkspace := normalizeWorkspaceID(workspaceID)
	normalizedProvider, err := NormalizeProvider(provider)
	if err != nil {
		return Config{}, err
	}

	configs, err := s.List(ctx, normalizedWorkspace)
	if err != nil {
		return Config{}, err
	}

	for _, config := range configs {
		if config.Provider == normalizedProvider {
			return config, nil
		}
	}
	return Config{}, ErrProviderNotFound
}

func (s *SQLiteStore) List(ctx context.Context, workspaceID string) ([]Config, error) {
	normalizedWorkspace := normalizeWorkspaceID(workspaceID)
	configs, err := s.listByWorkspace(ctx, normalizedWorkspace, normalizedWorkspace)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SQLiteStore) listByWorkspace(ctx context.Context, sourceWorkspace string, responseWorkspace string) ([]Config, error) {
	secretConfiguredByProvider, err := s.loadSecretConfiguredByProvider(ctx, sourceWorkspace)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			connector_type,
			COALESCE(config_json, ''),
			updated_at
		FROM channel_connectors
		WHERE workspace_id = ?
		  AND connector_type LIKE ?
		ORDER BY connector_type ASC
	`, sourceWorkspace, connectorTypeProviderPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query provider configs: %w", err)
	}
	defer rows.Close()

	configs := make([]Config, 0)
	for rows.Next() {
		var connectorType string
		var configJSON string
		var updatedAtRaw string
		if err := rows.Scan(&connectorType, &configJSON, &updatedAtRaw); err != nil {
			return nil, fmt.Errorf("scan provider config: %w", err)
		}

		provider := strings.TrimPrefix(connectorType, connectorTypeProviderPrefix)
		if _, err := NormalizeProvider(provider); err != nil {
			continue
		}

		connectorConfig := providerConnectorConfig{
			Provider: provider,
			Endpoint: DefaultEndpoint(provider),
		}
		if strings.TrimSpace(configJSON) != "" {
			if err := json.Unmarshal([]byte(configJSON), &connectorConfig); err != nil {
				return nil, fmt.Errorf("decode provider config JSON: %w", err)
			}
		}
		if connectorConfig.Provider == "" {
			connectorConfig.Provider = provider
		}
		connectorConfig.Provider = strings.ToLower(strings.TrimSpace(connectorConfig.Provider))
		normalizedEndpoint, err := normalizeEndpoint(provider, connectorConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		connectorConfig.Endpoint = normalizedEndpoint
		connectorConfig.APIKeySecretName = strings.TrimSpace(connectorConfig.APIKeySecretName)

		updatedAt, err := parseTimestamp(updatedAtRaw)
		if err != nil {
			return nil, err
		}

		configs = append(configs, Config{
			WorkspaceID:      responseWorkspace,
			Provider:         connectorConfig.Provider,
			Endpoint:         connectorConfig.Endpoint,
			APIKeySecretName: connectorConfig.APIKeySecretName,
			APIKeyConfigured: secretConfiguredByProvider[provider],
			UpdatedAt:        updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider config rows: %w", err)
	}
	return configs, nil
}

func (s *SQLiteStore) loadSecretConfiguredByProvider(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT owner_id
		FROM secret_refs
		WHERE workspace_id = ?
		  AND owner_type = ?
	`, workspaceID, ownerTypeModelProvider)
	if err != nil {
		return nil, fmt.Errorf("query provider secret refs: %w", err)
	}
	defer rows.Close()

	configured := map[string]bool{}
	for rows.Next() {
		var ownerID string
		if err := rows.Scan(&ownerID); err != nil {
			return nil, fmt.Errorf("scan provider secret ref: %w", err)
		}
		configured[strings.ToLower(strings.TrimSpace(ownerID))] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider secret refs: %w", err)
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
		return time.Time{}, fmt.Errorf("missing updated_at for provider config")
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

func normalizeEndpoint(provider string, endpoint string) (string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		trimmed = DefaultEndpoint(provider)
	}
	parsed, err := endpointpolicy.ParseAndValidate(trimmed, endpointpolicy.Options{
		Service: fmt.Sprintf("%s provider endpoint", strings.ToLower(strings.TrimSpace(provider))),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func connectorTypeForProvider(provider string) string {
	return connectorTypeProviderPrefix + provider
}

func providerConnectorID(workspaceID string, provider string) string {
	return fmt.Sprintf("provider:%s:%s", workspaceID, provider)
}

func providerSecretRefID(workspaceID string, provider string, account string) string {
	return fmt.Sprintf("provider-secret:%s:%s:%s", workspaceID, provider, strings.TrimSpace(account))
}
