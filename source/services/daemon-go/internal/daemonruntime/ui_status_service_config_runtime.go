package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/channelconfig"
	"personalagent/runtime/internal/transport"
)

func (s *UIStatusService) upsertCanonicalTwilioChannelConfig(
	ctx context.Context,
	workspaceID string,
	channelID string,
	config map[string]any,
) (map[string]any, string, error) {
	if s == nil || s.twilioStore == nil || !isTwilioChannelConfigID(channelID) {
		return nil, "", nil
	}
	return s.upsertCanonicalTwilioUIConfig(
		ctx,
		workspaceID,
		config,
		uiChannelConfigPrefix+strings.ToLower(strings.TrimSpace(channelID)),
	)
}

func (s *UIStatusService) upsertCanonicalTwilioConnectorConfig(
	ctx context.Context,
	workspaceID string,
	config map[string]any,
) (map[string]any, string, error) {
	if s == nil || s.twilioStore == nil {
		return nil, "", nil
	}
	return s.upsertCanonicalTwilioUIConfig(ctx, workspaceID, config, uiConnectorConfigPrefix+"twilio")
}

func (s *UIStatusService) upsertCanonicalTwilioUIConfig(
	ctx context.Context,
	workspaceID string,
	config map[string]any,
	uiConfigID string,
) (map[string]any, string, error) {
	existing, existingConfigured, err := s.loadTwilioConfig(ctx, workspaceID)
	if err != nil {
		return nil, "", err
	}

	accountSIDSecretName := uiConfigString(config, "account_sid_secret_name")
	if accountSIDSecretName == "" {
		accountSIDSecretName = strings.TrimSpace(existing.AccountSIDSecretName)
	}
	authTokenSecretName := uiConfigString(config, "auth_token_secret_name")
	if authTokenSecretName == "" {
		authTokenSecretName = strings.TrimSpace(existing.AuthTokenSecretName)
	}
	endpoint := uiConfigString(config, "endpoint")
	if endpoint == "" {
		endpoint = strings.TrimSpace(existing.Endpoint)
	}

	smsNumber := uiConfigString(config, "sms_number")
	voiceNumber := uiConfigString(config, "voice_number")
	number := uiConfigString(config, "number")
	if number != "" && smsNumber == "" {
		smsNumber = number
	}
	if number != "" && voiceNumber == "" {
		voiceNumber = number
	}
	if smsNumber == "" {
		smsNumber = strings.TrimSpace(existing.SMSNumber)
	}
	if voiceNumber == "" {
		voiceNumber = strings.TrimSpace(existing.VoiceNumber)
	}
	if smsNumber == "" && voiceNumber != "" {
		smsNumber = voiceNumber
	}
	if voiceNumber == "" && smsNumber != "" {
		voiceNumber = smsNumber
	}
	if endpoint == "" {
		endpoint = channelconfig.DefaultTwilioEndpoint()
	}

	if accountSIDSecretName == "" && authTokenSecretName == "" && smsNumber == "" && voiceNumber == "" {
		return nil, "", nil
	}
	if accountSIDSecretName == "" || authTokenSecretName == "" || smsNumber == "" || voiceNumber == "" {
		if existingConfigured {
			return nil, "", nil
		}
		return nil, "", nil
	}

	accountRef, err := s.container.GetSecretReference(ctx, workspaceID, accountSIDSecretName)
	if err != nil {
		if errors.Is(err, ErrSecretReferenceNotFound) {
			return nil, "", fmt.Errorf(
				"twilio account sid secret reference %q is not registered for workspace %q",
				accountSIDSecretName,
				workspaceID,
			)
		}
		return nil, "", fmt.Errorf("resolve twilio account sid secret reference %q: %w", accountSIDSecretName, err)
	}
	authRef, err := s.container.GetSecretReference(ctx, workspaceID, authTokenSecretName)
	if err != nil {
		if errors.Is(err, ErrSecretReferenceNotFound) {
			return nil, "", fmt.Errorf(
				"twilio auth token secret reference %q is not registered for workspace %q",
				authTokenSecretName,
				workspaceID,
			)
		}
		return nil, "", fmt.Errorf("resolve twilio auth token secret reference %q: %w", authTokenSecretName, err)
	}

	canonicalConfig, err := s.twilioStore.Upsert(ctx, channelconfig.TwilioUpsertInput{
		WorkspaceID:               workspaceID,
		AccountSIDSecretName:      accountSIDSecretName,
		AuthTokenSecretName:       authTokenSecretName,
		AccountSIDKeychainService: accountRef.Service,
		AccountSIDKeychainAccount: accountRef.Account,
		AuthTokenKeychainService:  authRef.Service,
		AuthTokenKeychainAccount:  authRef.Account,
		SMSNumber:                 smsNumber,
		VoiceNumber:               voiceNumber,
		Endpoint:                  endpoint,
	})
	if err != nil {
		return nil, "", fmt.Errorf("upsert canonical twilio config: %w", err)
	}

	normalizedConfig := map[string]any{
		"account_sid_secret_name": canonicalConfig.AccountSIDSecretName,
		"auth_token_secret_name":  canonicalConfig.AuthTokenSecretName,
		"endpoint":                canonicalConfig.Endpoint,
		"sms_number":              canonicalConfig.SMSNumber,
		"voice_number":            canonicalConfig.VoiceNumber,
		"number":                  canonicalConfig.SMSNumber,
		"credentials_configured":  canonicalConfig.CredentialsConfigured,
		"account_sid_configured":  canonicalConfig.AccountSIDConfigured,
		"auth_token_configured":   canonicalConfig.AuthTokenConfigured,
	}

	uiConfig, updatedAt, err := s.upsertUIConfig(
		ctx,
		workspaceID,
		strings.ToLower(strings.TrimSpace(uiConfigID)),
		normalizedConfig,
		true,
	)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(updatedAt) == "" {
		updatedAt = canonicalConfig.UpdatedAt.Format(time.RFC3339Nano)
	}
	return uiConfig, updatedAt, nil
}

func uiConfigString(config map[string]any, key string) string {
	if len(config) == 0 {
		return ""
	}
	value, ok := config[strings.TrimSpace(key)]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case float32:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case int:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int32:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int16:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int8:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case uint:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case uint64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case uint32:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case uint16:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case uint8:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func (s *UIStatusService) upsertUIConfig(
	ctx context.Context,
	workspaceID string,
	connectorType string,
	configuration map[string]any,
	merge bool,
) (map[string]any, string, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return nil, "", fmt.Errorf("database is not configured")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	normalizedType := strings.ToLower(strings.TrimSpace(connectorType))
	if normalizedType == "" {
		return nil, "", fmt.Errorf("connector type is required")
	}
	recordID := uiConfigRecordID(workspaceID, normalizedType)

	mergedConfig := map[string]any{}
	if merge {
		existing, err := s.readUIConfigByID(ctx, workspaceID, recordID)
		if err != nil {
			return nil, "", err
		}
		for key, value := range existing {
			mergedConfig[key] = value
		}
	}
	for key, value := range cloneUIAnyMap(configuration) {
		mergedConfig[key] = value
	}

	configJSON, err := json.Marshal(mergedConfig)
	if err != nil {
		return nil, "", fmt.Errorf("marshal ui config: %w", err)
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, now, now); err != nil {
		return nil, "", fmt.Errorf("ensure workspace: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO channel_connectors(
			id, workspace_id, connector_type, status, config_json, created_at, updated_at
		) VALUES (?, ?, ?, 'ACTIVE', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			config_json = excluded.config_json,
			updated_at = excluded.updated_at
	`, recordID, workspaceID, normalizedType, string(configJSON), now, now); err != nil {
		return nil, "", fmt.Errorf("upsert ui config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("commit tx: %w", err)
	}

	return mergedConfig, now, nil
}

func (s *UIStatusService) readUIConfigByID(ctx context.Context, workspaceID string, recordID string) (map[string]any, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	record := strings.TrimSpace(recordID)
	config := map[string]any{}
	var rawJSON string
	err := s.container.DB.QueryRowContext(ctx, `
		SELECT COALESCE(config_json, '')
		FROM channel_connectors
		WHERE workspace_id = ?
		  AND id = ?
		LIMIT 1
	`, workspace, record).Scan(&rawJSON)
	if err == sql.ErrNoRows {
		return config, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read ui config: %w", err)
	}
	if strings.TrimSpace(rawJSON) == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(rawJSON), &config); err != nil {
		return nil, fmt.Errorf("decode ui config json: %w", err)
	}
	return cloneUIAnyMap(config), nil
}

func cloneUIAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func normalizeUITestOperation(raw string) (string, error) {
	operation := strings.ToLower(strings.TrimSpace(raw))
	if operation == "" {
		operation = "health"
	}
	if operation != "health" {
		return "", fmt.Errorf("operation must be health")
	}
	return operation, nil
}

func uiConfigRecordID(workspaceID string, connectorType string) string {
	return uiConfigRecordIDPrefix + normalizeWorkspaceID(workspaceID) + "." + strings.ReplaceAll(strings.TrimSpace(connectorType), ".", "_")
}

func channelConfigFieldDescriptors(channelID string) []transport.ConfigFieldDescriptor {
	switch strings.ToLower(strings.TrimSpace(channelID)) {
	case "app":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "enabled",
				Label:    "Enabled",
				Type:     "bool",
				Editable: true,
				HelpText: "Enable or disable app-channel dispatch handling for this workspace.",
			},
			{
				Key:         "transport",
				Label:       "Transport",
				Type:        "enum",
				Editable:    true,
				EnumOptions: []string{"daemon_realtime"},
				HelpText:    "Daemon realtime transport mode used for app-channel chat flows.",
			},
		}
	case "message":
		return []transport.ConfigFieldDescriptor{
			{
				Key:         "primary_connector_id",
				Label:       "Primary Connector",
				Type:        "enum",
				Editable:    false,
				EnumOptions: []string{"imessage", "twilio"},
				HelpText:    "Primary message connector is derived from channel-mapping priority settings.",
			},
		}
	case "voice":
		return []transport.ConfigFieldDescriptor{
			{
				Key:         "primary_connector_id",
				Label:       "Primary Connector",
				Type:        "enum",
				Editable:    false,
				EnumOptions: []string{"twilio"},
				HelpText:    "Voice connector selection is mapping-driven for this MVP.",
			},
		}
	default:
		return nil
	}
}

func connectorConfigFieldDescriptors(connectorID string) []transport.ConfigFieldDescriptor {
	switch normalizeChannelMappingConnectorID(connectorID) {
	case "twilio":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "account_sid_secret_name",
				Label:    "Account SID Secret Ref",
				Type:     "secret_ref",
				Required: true,
				Editable: true,
				Secret:   true,
				HelpText: "Name of a registered SecretRef that stores the Twilio Account SID value.",
			},
			{
				Key:       "account_sid_value",
				Label:     "Account SID Value",
				Type:      "string",
				Editable:  true,
				Secret:    true,
				WriteOnly: true,
				HelpText:  "Optional raw Account SID input for client-side secure-store registration; daemon APIs do not return this value.",
			},
			{
				Key:      "auth_token_secret_name",
				Label:    "Auth Token Secret Ref",
				Type:     "secret_ref",
				Required: true,
				Editable: true,
				Secret:   true,
				HelpText: "Name of a registered SecretRef that stores the Twilio Auth Token value.",
			},
			{
				Key:       "auth_token_value",
				Label:     "Auth Token Value",
				Type:      "string",
				Editable:  true,
				Secret:    true,
				WriteOnly: true,
				HelpText:  "Optional raw Auth Token input for client-side secure-store registration; daemon APIs do not return this value.",
			},
			{
				Key:      "sms_number",
				Label:    "SMS Number",
				Type:     "string",
				Required: true,
				Editable: true,
				HelpText: "E.164 phone number used for SMS send/receive workflows.",
			},
			{
				Key:      "voice_number",
				Label:    "Voice Number",
				Type:     "string",
				Required: true,
				Editable: true,
				HelpText: "E.164 phone number used for outbound/inbound voice call workflows.",
			},
			{
				Key:      "endpoint",
				Label:    "API Endpoint",
				Type:     "url",
				Editable: true,
				HelpText: "Twilio API endpoint base URL (defaults to https://api.twilio.com).",
			},
		}
	case "imessage":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "source_db_path",
				Label:    "Messages Database Path",
				Type:     "path",
				Editable: true,
				HelpText: "Optional override for the Messages `chat.db` path used during inbound ingest polling.",
			},
			{
				Key:         "permission_state",
				Label:       "Permission State",
				Type:        "enum",
				Editable:    false,
				EnumOptions: []string{"unknown", "missing", "granted"},
				HelpText:    "Derived runtime permission state based on daemon probes and ingest errors.",
			},
		}
	case "mail":
		return []transport.ConfigFieldDescriptor{
			{
				Key:         "scope",
				Label:       "Mail Scope",
				Type:        "enum",
				Editable:    true,
				EnumOptions: []string{"inbox", "all"},
				HelpText:    "Connector-defined mailbox scope for inbound/outbound mail helpers.",
			},
			{
				Key:      "local_ingest_bridge_ready",
				Label:    "Local Ingest Bridge Ready",
				Type:     "bool",
				Editable: false,
				HelpText: "Derived daemon readiness for the local watcher queue bridge paths used by Mail handoffs.",
			},
			{
				Key:      "local_ingest_bridge",
				Label:    "Local Ingest Bridge Details",
				Type:     "object",
				Editable: false,
				HelpText: "Resolved queue-path details (pending/processed/failed) for Mail local-ingest bridge handoff.",
			},
		}
	case "calendar":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "calendar_id",
				Label:    "Default Calendar ID",
				Type:     "string",
				Editable: true,
				HelpText: "Optional calendar identifier used as default target for create/update flows.",
			},
			{
				Key:      "local_ingest_bridge_ready",
				Label:    "Local Ingest Bridge Ready",
				Type:     "bool",
				Editable: false,
				HelpText: "Derived daemon readiness for the local watcher queue bridge paths used by Calendar handoffs.",
			},
			{
				Key:      "local_ingest_bridge",
				Label:    "Local Ingest Bridge Details",
				Type:     "object",
				Editable: false,
				HelpText: "Resolved queue-path details (pending/processed/failed) for Calendar local-ingest bridge handoff.",
			},
		}
	case "browser":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "source_scope",
				Label:    "Source Scope",
				Type:     "string",
				Editable: true,
				HelpText: "Optional browser source scope token used for ingest/automation filtering.",
			},
			{
				Key:      "local_ingest_bridge_ready",
				Label:    "Local Ingest Bridge Ready",
				Type:     "bool",
				Editable: false,
				HelpText: "Derived daemon readiness for the local watcher queue bridge paths used by Browser handoffs.",
			},
			{
				Key:      "local_ingest_bridge",
				Label:    "Local Ingest Bridge Details",
				Type:     "object",
				Editable: false,
				HelpText: "Resolved queue-path details (pending/processed/failed) for Browser local-ingest bridge handoff.",
			},
		}
	case "finder":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "root_path",
				Label:    "Root Path",
				Type:     "path",
				Editable: true,
				HelpText: "Optional default root path for finder list/preview/delete operations.",
			},
		}
	case "cloudflared":
		return []transport.ConfigFieldDescriptor{
			{
				Key:      "binary_path",
				Label:    "Binary Path",
				Type:     "path",
				Editable: true,
				HelpText: "Optional cloudflared binary path override for connector-managed CLI proxy operations.",
			},
		}
	default:
		return nil
	}
}

func channelPluginForID(channelID string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(channelID)) {
	case "app":
		return appChatWorkerPluginID, true
	case "message":
		return messagesWorkerPluginID, true
	case "voice":
		return twilioWorkerPluginID, true
	default:
		return "", false
	}
}

func connectorPluginForID(connectorID string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(connectorID))
	switch normalized {
	case "builtin.app":
		return appChatWorkerPluginID, true
	case "imessage":
		return messagesWorkerPluginID, true
	case "twilio":
		return twilioWorkerPluginID, true
	case "mail", "calendar", "browser", "finder":
		return normalized + ".daemon", true
	case "cloudflared":
		return CloudflaredConnectorPluginID, true
	default:
		return "", false
	}
}
