package daemonruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"personalagent/runtime/internal/channelconfig"
	"personalagent/runtime/internal/securestore"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

func TestUIStatusServiceConfigUpsertPersistsAndMerges(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	channelFirst, err := service.UpsertChannelConfig(context.Background(), transport.ChannelConfigUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"transport": "daemon_realtime",
			"enabled":   true,
		}),
	})
	if err != nil {
		t.Fatalf("upsert channel config: %v", err)
	}
	if channelFirst.ChannelID != "app" || channelFirst.WorkspaceID != "ws1" {
		t.Fatalf("unexpected channel config response: %+v", channelFirst)
	}

	channelMerged, err := service.UpsertChannelConfig(context.Background(), transport.ChannelConfigUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"enabled": false,
		}),
		Merge: true,
	})
	if err != nil {
		t.Fatalf("merge channel config: %v", err)
	}
	if enabled, ok := channelMerged.Configuration.AsMap()["enabled"].(bool); !ok || enabled {
		t.Fatalf("expected merged enabled=false, got %+v", channelMerged.Configuration)
	}
	if channelMerged.Configuration.AsMap()["transport"] != "daemon_realtime" {
		t.Fatalf("expected merged transport to be preserved, got %+v", channelMerged.Configuration)
	}

	connectorFirst, err := service.UpsertConnectorConfig(context.Background(), transport.ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"scope": "inbox",
		}),
	})
	if err != nil {
		t.Fatalf("upsert connector config: %v", err)
	}
	if connectorFirst.ConnectorID != "mail" || connectorFirst.WorkspaceID != "ws1" {
		t.Fatalf("unexpected connector config response: %+v", connectorFirst)
	}

	connectorMerged, err := service.UpsertConnectorConfig(context.Background(), transport.ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"mode": "read_only",
		}),
		Merge: true,
	})
	if err != nil {
		t.Fatalf("merge connector config: %v", err)
	}
	if connectorMerged.Configuration.AsMap()["scope"] != "inbox" || connectorMerged.Configuration.AsMap()["mode"] != "read_only" {
		t.Fatalf("expected merged connector config fields, got %+v", connectorMerged.Configuration)
	}
}

func TestUIStatusServiceChannelConnectorMappingListAndUpsert(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	initial, err := service.ListChannelConnectorMappings(context.Background(), transport.ChannelConnectorMappingListRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
	})
	if err != nil {
		t.Fatalf("list channel connector mappings: %v", err)
	}
	if initial.WorkspaceID != "ws1" || initial.ChannelID != "message" {
		t.Fatalf("unexpected mapping list identifiers: %+v", initial)
	}
	if initial.FallbackPolicy != "priority_order" {
		t.Fatalf("expected fallback policy priority_order, got %s", initial.FallbackPolicy)
	}
	if len(initial.Bindings) < 2 {
		t.Fatalf("expected seeded message channel mappings, got %+v", initial.Bindings)
	}

	prioritized, err := service.UpsertChannelConnectorMapping(context.Background(), transport.ChannelConnectorMappingUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
		ConnectorID: "twilio",
		Enabled:     true,
		Priority:    1,
	})
	if err != nil {
		t.Fatalf("upsert channel connector mapping prioritize: %v", err)
	}
	if prioritized.ChannelID != "message" || prioritized.ConnectorID != "twilio" || prioritized.Priority != 1 {
		t.Fatalf("unexpected prioritize response: %+v", prioritized)
	}
	twilioBinding, found := findChannelConnectorMappingRecord(prioritized.Bindings, "twilio")
	if !found {
		t.Fatalf("expected twilio binding in prioritize response: %+v", prioritized.Bindings)
	}
	if !twilioBinding.Enabled || twilioBinding.Priority != 1 {
		t.Fatalf("expected twilio binding enabled with priority=1, got %+v", twilioBinding)
	}

	disabled, err := service.UpsertChannelConnectorMapping(context.Background(), transport.ChannelConnectorMappingUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
		ConnectorID: "imessage",
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("upsert channel connector mapping disable: %v", err)
	}
	messagesBinding, found := findChannelConnectorMappingRecord(disabled.Bindings, "imessage")
	if !found {
		t.Fatalf("expected imessage binding in disable response: %+v", disabled.Bindings)
	}
	if messagesBinding.Enabled {
		t.Fatalf("expected imessage binding to be disabled, got %+v", messagesBinding)
	}
}

func TestUIStatusServiceChannelConnectorMappingListDoesNotFallbackToLegacyWorkspaceRows(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed default workspace: %v", err)
	}
	if _, err := container.DB.Exec(`
		UPDATE channel_connector_bindings
		SET enabled = 0, updated_at = '2026-02-26T00:00:01Z'
		WHERE workspace_id = 'default'
		  AND channel_id = 'message'
		  AND connector_id = 'imessage'
	`); err != nil {
		t.Fatalf("customize legacy message binding: %v", err)
	}
	response, err := service.ListChannelConnectorMappings(context.Background(), transport.ChannelConnectorMappingListRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
	})
	if err != nil {
		t.Fatalf("list channel connector mappings: %v", err)
	}
	appleBinding, found := findChannelConnectorMappingRecord(response.Bindings, "imessage")
	if !found {
		t.Fatalf("expected imessage binding in response: %+v", response.Bindings)
	}
	if !appleBinding.Enabled {
		t.Fatalf("expected canonical ws1 binding to remain enabled; legacy default row must not override: %+v", appleBinding)
	}
}

func TestUIStatusServiceChannelConnectorMappingRejectsUnsupportedCapabilityMap(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	_, err = service.UpsertChannelConnectorMapping(context.Background(), transport.ChannelConnectorMappingUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "voice",
		ConnectorID: "imessage",
		Enabled:     true,
		Priority:    1,
	})
	if err == nil {
		t.Fatalf("expected unsupported mapping upsert to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "does not") || !strings.Contains(strings.ToLower(err.Error()), "voice") {
		t.Fatalf("expected capability validation failure, got %v", err)
	}
}

func TestUIStatusServiceConfigUpsertMergeDoesNotReadLegacyWorkspaceUIConfigRows(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed default workspace: %v", err)
	}
	if _, err := container.DB.Exec(`
		INSERT INTO channel_connectors(
			id, workspace_id, connector_type, status, config_json, created_at, updated_at
		) VALUES (
			'ui.config.default.ui_connector_mail',
			'default',
			'ui.connector.mail',
			'ACTIVE',
			'{"scope":"legacy_inbox"}',
			'2026-02-26T00:00:00Z',
			'2026-02-26T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed legacy ui config row: %v", err)
	}

	updated, err := service.UpsertConnectorConfig(context.Background(), transport.ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"mode": "read_only",
		}),
		Merge: true,
	})
	if err != nil {
		t.Fatalf("upsert connector config merge: %v", err)
	}
	if _, exists := updated.Configuration.AsMap()["scope"]; exists {
		t.Fatalf("expected legacy workspace scope to be ignored, got %+v", updated.Configuration)
	}
	if updated.Configuration.AsMap()["mode"] != "read_only" {
		t.Fatalf("expected mode read_only, got %+v", updated.Configuration)
	}
}

func findChannelConnectorMappingRecord(bindings []transport.ChannelConnectorMappingRecord, connectorID string) (transport.ChannelConnectorMappingRecord, bool) {
	target := strings.ToLower(strings.TrimSpace(connectorID))
	for _, binding := range bindings {
		if strings.ToLower(strings.TrimSpace(binding.ConnectorID)) == target {
			return binding, true
		}
	}
	return transport.ChannelConnectorMappingRecord{}, false
}

func TestUIStatusServiceTwilioConnectorConfigUpsertCanonicalizesRuntimeConfig(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.RegisterSecretReference(context.Background(), securestore.SecretReference{
		WorkspaceID: "ws1",
		Name:        "TWILIO_ACCOUNT_SID",
		Backend:     "memory",
		Service:     "personal-agent.ws1",
		Account:     "TWILIO_ACCOUNT_SID",
	}); err != nil {
		t.Fatalf("register account sid secret reference: %v", err)
	}
	if _, err := container.RegisterSecretReference(context.Background(), securestore.SecretReference{
		WorkspaceID: "ws1",
		Name:        "TWILIO_AUTH_TOKEN",
		Backend:     "memory",
		Service:     "personal-agent.ws1",
		Account:     "TWILIO_AUTH_TOKEN",
	}); err != nil {
		t.Fatalf("register auth token secret reference: %v", err)
	}

	response, err := service.UpsertConnectorConfig(context.Background(), transport.ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "twilio",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"account_sid_secret_name": "TWILIO_ACCOUNT_SID",
			"auth_token_secret_name":  "TWILIO_AUTH_TOKEN",
			"number":                  "+15555550001",
			"endpoint":                "https://api.twilio.test",
		}),
		Merge: true,
	})
	if err != nil {
		t.Fatalf("upsert twilio connector config: %v", err)
	}
	if response.ConnectorID != "twilio" || response.WorkspaceID != "ws1" {
		t.Fatalf("unexpected connector config response identifiers: %+v", response)
	}
	if got := response.Configuration.AsMap()["number"]; got != "+15555550001" {
		t.Fatalf("expected normalized twilio number to match saved sms number, got %+v", response.Configuration)
	}
	if configured, ok := response.Configuration.AsMap()["credentials_configured"].(bool); !ok || !configured {
		t.Fatalf("expected credentials_configured=true in upsert response, got %+v", response.Configuration)
	}

	store := channelconfig.NewSQLiteTwilioStore(container.DB)
	canonical, err := store.Get(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("load canonical twilio config: %v", err)
	}
	if canonical.AccountSIDSecretName != "TWILIO_ACCOUNT_SID" || canonical.AuthTokenSecretName != "TWILIO_AUTH_TOKEN" {
		t.Fatalf("unexpected canonical twilio secret names: %+v", canonical)
	}
	if canonical.SMSNumber != "+15555550001" {
		t.Fatalf("expected canonical sms number, got %+v", canonical)
	}
	if canonical.VoiceNumber != "+15555550001" {
		t.Fatalf("expected canonical voice number fallback to sms number, got %+v", canonical)
	}
	if canonical.Endpoint != "https://api.twilio.test" {
		t.Fatalf("expected canonical endpoint https://api.twilio.test, got %+v", canonical)
	}
}

func TestUIStatusServiceTwilioChannelConfigUpsertCanonicalizesRuntimeConfig(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.RegisterSecretReference(context.Background(), securestore.SecretReference{
		WorkspaceID: "ws1",
		Name:        "TWILIO_ACCOUNT_SID",
		Backend:     "memory",
		Service:     "personal-agent.ws1",
		Account:     "TWILIO_ACCOUNT_SID",
	}); err != nil {
		t.Fatalf("register account sid secret reference: %v", err)
	}
	if _, err := container.RegisterSecretReference(context.Background(), securestore.SecretReference{
		WorkspaceID: "ws1",
		Name:        "TWILIO_AUTH_TOKEN",
		Backend:     "memory",
		Service:     "personal-agent.ws1",
		Account:     "TWILIO_AUTH_TOKEN",
	}); err != nil {
		t.Fatalf("register auth token secret reference: %v", err)
	}

	response, err := service.UpsertChannelConfig(context.Background(), transport.ChannelConfigUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "voice",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"account_sid_secret_name": "TWILIO_ACCOUNT_SID",
			"auth_token_secret_name":  "TWILIO_AUTH_TOKEN",
			"number":                  "+15555550009",
			"endpoint":                "https://api.twilio.voice.test",
		}),
		Merge: true,
	})
	if err != nil {
		t.Fatalf("upsert twilio channel config: %v", err)
	}
	if response.ChannelID != "voice" || response.WorkspaceID != "ws1" {
		t.Fatalf("unexpected channel config response identifiers: %+v", response)
	}
	if got := response.Configuration.AsMap()["number"]; got != "+15555550009" {
		t.Fatalf("expected normalized twilio number in channel config response, got %+v", response.Configuration)
	}
	if configured, ok := response.Configuration.AsMap()["credentials_configured"].(bool); !ok || !configured {
		t.Fatalf("expected credentials_configured=true in channel upsert response, got %+v", response.Configuration)
	}

	store := channelconfig.NewSQLiteTwilioStore(container.DB)
	canonical, err := store.Get(context.Background(), "ws1")
	if err != nil {
		t.Fatalf("load canonical twilio config: %v", err)
	}
	if canonical.SMSNumber != "+15555550009" || canonical.VoiceNumber != "+15555550009" {
		t.Fatalf("expected canonical twilio numbers from channel config upsert, got %+v", canonical)
	}
	if canonical.Endpoint != "https://api.twilio.voice.test" {
		t.Fatalf("expected canonical endpoint from channel config upsert, got %+v", canonical)
	}
}

func TestUIStatusServiceTwilioConnectorConfigUpsertErrorsForMissingSecretReference(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	_, err = service.UpsertConnectorConfig(context.Background(), transport.ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "twilio",
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{
			"account_sid_secret_name": "TWILIO_ACCOUNT_SID",
			"auth_token_secret_name":  "TWILIO_AUTH_TOKEN",
			"number":                  "+15555550001",
		}),
		Merge: true,
	})
	if err == nil {
		t.Fatalf("expected twilio upsert to fail when secret references are missing")
	}
	if !strings.Contains(err.Error(), "secret reference") {
		t.Fatalf("expected missing secret reference error, got %v", err)
	}

	store := channelconfig.NewSQLiteTwilioStore(container.DB)
	_, getErr := store.Get(context.Background(), "ws1")
	if !errors.Is(getErr, channelconfig.ErrTwilioNotConfigured) {
		t.Fatalf("expected canonical twilio config to remain missing, got err=%v", getErr)
	}
}

func TestUIStatusServiceTestOperationsReturnStructuredPayloads(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: appChatWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
		{
			PluginID: twilioWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	channelResult, err := service.TestChannelOperation(context.Background(), transport.ChannelTestOperationRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app",
	})
	if err != nil {
		t.Fatalf("channel test operation: %v", err)
	}
	if !channelResult.Success || channelResult.Status != "ok" || channelResult.Operation != "health" {
		t.Fatalf("unexpected channel test response: %+v", channelResult)
	}
	if channelResult.Details.AsMap()["plugin_id"] != appChatWorkerPluginID {
		t.Fatalf("expected app plugin details, got %+v", channelResult.Details)
	}

	twilioResult, err := service.TestChannelOperation(context.Background(), transport.ChannelTestOperationRequest{
		WorkspaceID: "ws1",
		ChannelID:   "voice",
	})
	if err != nil {
		t.Fatalf("twilio channel test operation: %v", err)
	}
	if twilioResult.Success || twilioResult.Status != "not_configured" {
		t.Fatalf("expected twilio test to report not_configured without config, got %+v", twilioResult)
	}

	connectorResult, err := service.TestConnectorOperation(context.Background(), transport.ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	})
	if err != nil {
		t.Fatalf("connector test operation: %v", err)
	}
	if !connectorResult.Success || connectorResult.Status != "ok" || connectorResult.Operation != "health" {
		t.Fatalf("unexpected connector test response: %+v", connectorResult)
	}
	if connectorResult.Details.AsMap()["plugin_id"] != "mail.daemon" {
		t.Fatalf("expected mail plugin details, got %+v", connectorResult.Details)
	}

	if _, err := service.TestChannelOperation(context.Background(), transport.ChannelTestOperationRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app",
		Operation:   "ping",
	}); err == nil {
		t.Fatalf("expected unsupported channel test operation to fail")
	}

	if _, err := service.TestConnectorOperation(context.Background(), transport.ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Operation:   "ping",
	}); err == nil {
		t.Fatalf("expected unsupported connector test operation to fail")
	}
}
