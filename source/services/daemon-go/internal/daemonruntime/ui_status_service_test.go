package daemonruntime

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/channelconfig"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

func TestUIStatusServiceChannelStatusWithoutTwilioConfig(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: twilioWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Channels) != 3 {
		t.Fatalf("expected 3 logical channels, got %d", len(response.Channels))
	}

	appChat, ok := findChannelCard(response.Channels, "app")
	if !ok {
		t.Fatalf("expected app channel card")
	}
	if appChat.Status != "degraded" {
		t.Fatalf("expected app status degraded when worker missing, got %s", appChat.Status)
	}
	if appChat.Worker != nil {
		t.Fatalf("expected no app worker snapshot when worker missing")
	}
	if _, ok := findRemediationAction(appChat.RemediationActions, "repair_daemon_runtime"); !ok {
		t.Fatalf("expected app status card to include repair_daemon_runtime remediation action")
	}

	imessage, ok := findChannelCard(response.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if imessage.Status != "degraded" {
		t.Fatalf("expected message status degraded when workers are not fully ready, got %s", imessage.Status)
	}
	if imessage.Worker != nil {
		t.Fatalf("expected no message worker snapshot when primary connector worker is missing")
	}

	twilioSMS, ok := findChannelCard(response.Channels, "voice")
	if !ok {
		t.Fatalf("expected voice channel card")
	}
	if twilioSMS.Configured {
		t.Fatalf("expected voice configured=false when config is missing")
	}
	if twilioSMS.Status != "not_configured" {
		t.Fatalf("expected voice status not_configured, got %s", twilioSMS.Status)
	}
	if twilioSMS.Worker == nil || twilioSMS.Worker.PluginID != twilioWorkerPluginID {
		t.Fatalf("expected voice to include worker snapshot")
	}
	if _, ok := findRemediationAction(twilioSMS.RemediationActions, "configure_twilio_channel"); !ok {
		t.Fatalf("expected voice status card to include configure_twilio_channel remediation action")
	}
}

func TestUIStatusServiceChannelStatusWithTwilioConfigAndWorker(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: appChatWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Capabilities: []shared.CapabilityDescriptor{
					{Key: "channel.app_chat.send"},
					{Key: "channel.app_chat.status"},
				},
			},
		},
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Capabilities: []shared.CapabilityDescriptor{
					{Key: "channel.messages.send"},
					{Key: "channel.messages.status"},
					{Key: "channel.messages.ingest_poll"},
				},
			},
		},
		{
			PluginID: twilioWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	store := channelconfig.NewSQLiteTwilioStore(container.DB)
	_, err := store.Upsert(context.Background(), channelconfig.TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "https://api.twilio.com",
	})
	if err != nil {
		t.Fatalf("upsert twilio config: %v", err)
	}

	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status: %v", err)
	}
	twilioSMS, ok := findChannelCard(response.Channels, "voice")
	if !ok {
		t.Fatalf("expected voice channel card")
	}
	if !twilioSMS.Configured {
		t.Fatalf("expected voice configured=true")
	}
	if twilioSMS.Status != "ready" {
		t.Fatalf("expected voice status ready, got %s", twilioSMS.Status)
	}
	if twilioSMS.Configuration["primary_connector_id"] != "twilio" {
		t.Fatalf("expected voice primary connector twilio in config map, got %+v", twilioSMS.Configuration)
	}

	appChat, ok := findChannelCard(response.Channels, "app")
	if !ok {
		t.Fatalf("expected app channel card")
	}
	if appChat.Status != "ready" {
		t.Fatalf("expected app status ready, got %s", appChat.Status)
	}
	if appChat.Worker == nil || appChat.Worker.PluginID != appChatWorkerPluginID {
		t.Fatalf("expected app worker snapshot")
	}
	if descriptor, ok := findConfigFieldDescriptor(appChat.ConfigFieldDescriptors, "transport"); !ok {
		t.Fatalf("expected app channel transport field descriptor")
	} else if descriptor.Type != "enum" || !descriptor.Editable || len(descriptor.EnumOptions) == 0 {
		t.Fatalf("unexpected app channel transport descriptor: %+v", descriptor)
	}
	if _, ok := findRemediationAction(appChat.RemediationActions, "refresh_channel_status"); !ok {
		t.Fatalf("expected app status card to include refresh_channel_status remediation action")
	}

	imessage, ok := findChannelCard(response.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if imessage.Status != "ready" {
		t.Fatalf("expected message status ready, got %s", imessage.Status)
	}
	if imessage.Worker == nil || imessage.Worker.PluginID != messagesWorkerPluginID {
		t.Fatalf("expected message worker snapshot")
	}
	if descriptor, ok := findConfigFieldDescriptor(imessage.ConfigFieldDescriptors, "primary_connector_id"); !ok {
		t.Fatalf("expected message channel primary_connector_id descriptor")
	} else if descriptor.Editable || descriptor.Type != "enum" {
		t.Fatalf("unexpected message channel primary connector descriptor: %+v", descriptor)
	}
	if _, ok := findRemediationAction(imessage.RemediationActions, "open_channel_logs"); !ok {
		t.Fatalf("expected message status card to include open_channel_logs remediation action")
	}
}

func TestUIStatusServiceChannelStatusTreatsStartupStatesAsReady(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: appChatWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateStarting,
		},
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRestarting,
		},
		{
			PluginID: twilioWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRegistered,
		},
	})
	store := channelconfig.NewSQLiteTwilioStore(container.DB)
	_, err := store.Upsert(context.Background(), channelconfig.TwilioUpsertInput{
		WorkspaceID:               "ws1",
		AccountSIDSecretName:      "TWILIO_ACCOUNT_SID",
		AuthTokenSecretName:       "TWILIO_AUTH_TOKEN",
		AccountSIDKeychainService: "personal-agent",
		AccountSIDKeychainAccount: "TWILIO_ACCOUNT_SID",
		AuthTokenKeychainService:  "personal-agent",
		AuthTokenKeychainAccount:  "TWILIO_AUTH_TOKEN",
		SMSNumber:                 "+15555550001",
		VoiceNumber:               "+15555550002",
		Endpoint:                  "https://api.twilio.com",
	})
	if err != nil {
		t.Fatalf("upsert twilio config: %v", err)
	}

	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status: %v", err)
	}

	appChat, ok := findChannelCard(response.Channels, "app")
	if !ok {
		t.Fatalf("expected app channel card")
	}
	if appChat.Status != "ready" {
		t.Fatalf("expected app startup state to classify as ready, got %s", appChat.Status)
	}
	if !strings.Contains(strings.ToLower(appChat.Summary), "available via") {
		t.Fatalf("expected app startup summary, got %q", appChat.Summary)
	}

	imessage, ok := findChannelCard(response.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if imessage.Status != "ready" {
		t.Fatalf("expected message startup state to classify as ready, got %s", imessage.Status)
	}

	twilioSMS, ok := findChannelCard(response.Channels, "voice")
	if !ok {
		t.Fatalf("expected voice channel card")
	}
	if twilioSMS.Status != "ready" {
		t.Fatalf("expected voice startup state to classify as ready, got %s", twilioSMS.Status)
	}
}

func TestUIStatusServiceImessageChannelStatusTracksIngestFailureAndRecovery(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}
	originalProbe := runMessagesStatusProbe
	t.Cleanup(func() {
		runMessagesStatusProbe = originalProbe
	})
	runMessagesStatusProbe = func(request messagesadapter.StatusRequest) messagesadapter.StatusResponse {
		return messagesadapter.StatusResponse{
			Ready:        false,
			Source:       messagesadapter.SourceName,
			SourceScope:  messagesadapter.ResolveSourceScope(request.SourceScope, "/Users/test/Library/Messages/chat.db"),
			SourceDBPath: "/Users/test/Library/Messages/chat.db",
			Transport:    "messages_apple_events",
			Error:        "messages source db path is not readable: open /Users/test/Library/Messages/chat.db: operation not permitted",
		}
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws1', 'ws1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO automation_source_subscriptions(
			id, workspace_id, source, source_scope, status, config_json,
			last_cursor, last_event_id, last_error, created_at, updated_at
		) VALUES (
			'sub.messages.ws1', 'ws1', ?, 'messages.chatdb.default', 'ACTIVE', '{}',
			'', '', 'Not authorized to read Messages database path /Users/test/Library/Messages/chat.db', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z'
		)
		ON CONFLICT(workspace_id, source, source_scope) DO UPDATE SET
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, messagesadapter.SourceName); err != nil {
		t.Fatalf("seed messages ingest failure subscription: %v", err)
	}

	degradedResponse, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status degraded: %v", err)
	}
	imessage, ok := findChannelCard(degradedResponse.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if imessage.Status != "degraded" {
		t.Fatalf("expected message channel degraded during ingest failure, got %s", imessage.Status)
	}
	if reason, _ := imessage.Configuration["status_reason"].(string); reason != channelReasonIngestFailure {
		t.Fatalf("expected imessage status reason %q, got %+v", channelReasonIngestFailure, imessage.Configuration["status_reason"])
	}
	if _, ok := imessage.Configuration["ingest_last_error"].(string); !ok {
		t.Fatalf("expected imessage ingest_last_error details, got %+v", imessage.Configuration)
	}
	if action, ok := findRemediationAction(imessage.RemediationActions, "repair_messages_ingest_access"); !ok {
		t.Fatalf("expected imessage remediation action repair_messages_ingest_access")
	} else if !action.Recommended {
		t.Fatalf("expected ingest remediation action to be recommended, got %+v", action)
	}
	openSettingsAction, ok := findRemediationAction(imessage.RemediationActions, "open_channel_system_settings")
	if !ok {
		t.Fatalf("expected imessage remediation action open_channel_system_settings")
	}
	if openSettingsAction.Intent != "open_system_settings" {
		t.Fatalf("expected open_channel_system_settings intent=open_system_settings, got %+v", openSettingsAction)
	}
	if openSettingsAction.Destination != "ui://system-settings/privacy/full-disk-access" {
		t.Fatalf("unexpected iMessage channel system settings destination: %s", openSettingsAction.Destination)
	}

	if _, err := container.DB.Exec(`
		UPDATE automation_source_subscriptions
		SET last_error = '', updated_at = '2026-02-25T00:10:00Z'
		WHERE workspace_id = 'ws1'
		  AND source = ?
		  AND source_scope = 'messages.chatdb.default'
	`, messagesadapter.SourceName); err != nil {
		t.Fatalf("clear messages ingest failure subscription error: %v", err)
	}

	readyResponse, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status ready: %v", err)
	}
	imessageReady, ok := findChannelCard(readyResponse.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card after recovery")
	}
	if imessageReady.Status != "ready" {
		t.Fatalf("expected message channel ready after ingest recovery, got %s", imessageReady.Status)
	}
	if reason, _ := imessageReady.Configuration["status_reason"].(string); reason != channelReasonReady {
		t.Fatalf("expected imessage status reason %q after recovery, got %+v", channelReasonReady, imessageReady.Configuration["status_reason"])
	}
	if _, exists := imessageReady.Configuration["ingest_last_error"]; exists {
		t.Fatalf("expected ingest_last_error to be cleared after recovery, got %+v", imessageReady.Configuration["ingest_last_error"])
	}
}

func TestUIStatusServiceImessageChannelStatusDoesNotFallbackToLegacyWorkspaceIngestState(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed legacy workspace: %v", err)
	}
	if _, err := container.DB.Exec(`
		INSERT INTO automation_source_subscriptions(
			id, workspace_id, source, source_scope, status, config_json,
			last_cursor, last_event_id, last_error, created_at, updated_at
		) VALUES (
			'sub.messages.default', 'default', ?, 'messages.chatdb.default', 'ACTIVE', '{}',
			'', '', 'legacy ingest failure: operation not permitted', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z'
		)
		ON CONFLICT(workspace_id, source, source_scope) DO UPDATE SET
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, messagesadapter.SourceName); err != nil {
		t.Fatalf("seed legacy ingest subscription: %v", err)
	}

	response, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status with legacy ingest fallback: %v", err)
	}
	imessage, ok := findChannelCard(response.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if imessage.Status != "ready" {
		t.Fatalf("expected message channel ready when only legacy workspace ingest state exists, got %s", imessage.Status)
	}
	if reason, _ := imessage.Configuration["status_reason"].(string); reason == channelReasonIngestFailure {
		t.Fatalf("expected no ingest failure reason from legacy workspace row, got %+v", imessage.Configuration["status_reason"])
	}
}

func TestUIStatusServiceConnectorStatusMapsWorkerStates(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
		{
			PluginID:  "calendar.daemon",
			Kind:      shared.AdapterKindConnector,
			State:     PluginWorkerStateFailed,
			LastError: "unexpected runtime crash",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Connectors) < 4 {
		t.Fatalf("expected at least 4 connector cards, got %d", len(response.Connectors))
	}

	mail, ok := findConnectorCard(response.Connectors, "mail")
	if !ok {
		t.Fatalf("expected mail connector card")
	}
	if mail.Status != "ready" {
		t.Fatalf("expected mail status ready, got %s", mail.Status)
	}
	if mail.Worker == nil || mail.Worker.PluginID != "mail.daemon" {
		t.Fatalf("expected mail worker snapshot")
	}
	if _, ok := findRemediationAction(mail.RemediationActions, "refresh_connector_status"); !ok {
		t.Fatalf("expected mail status card to include refresh_connector_status remediation action")
	}

	calendar, ok := findConnectorCard(response.Connectors, "calendar")
	if !ok {
		t.Fatalf("expected calendar connector card")
	}
	if calendar.Status != "failed" {
		t.Fatalf("expected calendar status failed, got %s", calendar.Status)
	}
	if _, ok := findRemediationAction(calendar.RemediationActions, "repair_daemon_runtime"); !ok {
		t.Fatalf("expected calendar status card to include repair_daemon_runtime remediation action")
	}

	browser, ok := findConnectorCard(response.Connectors, "browser")
	if !ok {
		t.Fatalf("expected browser connector card")
	}
	if browser.Status != "degraded" {
		t.Fatalf("expected browser status degraded when worker missing, got %s", browser.Status)
	}
	if _, ok := findRemediationAction(browser.RemediationActions, "open_connector_system_settings"); !ok {
		t.Fatalf("expected browser status card to include open_connector_system_settings remediation action")
	}
}

func TestUIStatusServiceConnectorStatusExecutePathProbeFailureIsDegraded(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: "127.0.0.1:19091",
				},
			},
			execAuthToken: "worker-token",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalProbe := runConnectorExecutePathProbe
	t.Cleanup(func() {
		runConnectorExecutePathProbe = originalProbe
	})
	runConnectorExecutePathProbe = func(_ context.Context, _ string, connectorID string, _ PluginWorkerStatus) (connectorExecuteProbeResult, error) {
		if strings.TrimSpace(connectorID) == "mail" {
			return connectorExecuteProbeResult{
				Ready: false,
				Error: "dial tcp 127.0.0.1:19091: connect: connection refused",
			}, fmt.Errorf("dial tcp 127.0.0.1:19091: connect: connection refused")
		}
		return connectorExecuteProbeResult{Ready: true, StatusCode: 400, Error: "unsupported"}, nil
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	mail, ok := findConnectorCard(response.Connectors, "mail")
	if !ok {
		t.Fatalf("expected mail connector card")
	}
	if mail.Status != "degraded" {
		t.Fatalf("expected mail status degraded after execute probe failure, got %s", mail.Status)
	}
	if reason, _ := mail.Configuration["status_reason"].(string); reason != connectorReasonExecutePathFailure {
		t.Fatalf("expected mail status reason %q, got %+v", connectorReasonExecutePathFailure, mail.Configuration["status_reason"])
	}
	if ready, _ := mail.Configuration["execute_path_probe_ready"].(bool); ready {
		t.Fatalf("expected execute_path_probe_ready=false on execute-path failure, got %+v", mail.Configuration["execute_path_probe_ready"])
	}
	if !strings.Contains(strings.ToLower(mail.Summary), "execute endpoint probe failed") {
		t.Fatalf("expected execute-path failure summary, got %q", mail.Summary)
	}
	if _, ok := findRemediationAction(mail.RemediationActions, "repair_daemon_runtime"); !ok {
		t.Fatalf("expected execute-path degraded connector to include repair_daemon_runtime action")
	}
	if mail.ActionReadiness != "degraded" {
		t.Fatalf("expected action_readiness degraded for execute-path failure, got %q", mail.ActionReadiness)
	}
	if len(mail.ActionBlockers) != 1 || mail.ActionBlockers[0].Code != "execute_path_unavailable" {
		t.Fatalf("expected execute_path_unavailable blocker, got %+v", mail.ActionBlockers)
	}
}

func TestUIStatusServiceConnectorStatusExecutePathPermissionFailureMarksPermissionMissing(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "browser.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: "127.0.0.1:19095",
				},
			},
			execAuthToken: "worker-token",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalProbe := runConnectorExecutePathProbe
	t.Cleanup(func() {
		runConnectorExecutePathProbe = originalProbe
	})
	runConnectorExecutePathProbe = func(_ context.Context, _ string, connectorID string, _ PluginWorkerStatus) (connectorExecuteProbeResult, error) {
		if strings.TrimSpace(connectorID) != "browser" {
			return connectorExecuteProbeResult{Ready: true, StatusCode: 400, Error: "unsupported"}, nil
		}
		return connectorExecuteProbeResult{
			Ready:      false,
			StatusCode: 500,
			Error:      "Browser connector execute failed: Allow JavaScript from Apple Events is disabled.",
		}, fmt.Errorf("Browser connector execute failed: Allow JavaScript from Apple Events is disabled.")
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	browser, ok := findConnectorCard(response.Connectors, "browser")
	if !ok {
		t.Fatalf("expected browser connector card")
	}
	if browser.Status != "degraded" {
		t.Fatalf("expected browser status degraded after execute probe permission failure, got %s", browser.Status)
	}
	if reason, _ := browser.Configuration["status_reason"].(string); reason != connectorReasonPermissionMissing {
		t.Fatalf("expected browser status reason %q, got %+v", connectorReasonPermissionMissing, browser.Configuration["status_reason"])
	}
	if permission, _ := browser.Configuration["permission_state"].(string); permission != "missing" {
		t.Fatalf("expected browser permission_state=missing, got %+v", browser.Configuration["permission_state"])
	}
	if browser.ActionReadiness != "blocked" {
		t.Fatalf("expected action_readiness blocked for permission-missing execute probe failure, got %q", browser.ActionReadiness)
	}
	if len(browser.ActionBlockers) != 1 || browser.ActionBlockers[0].Code != "permission_missing" {
		t.Fatalf("expected permission_missing blocker, got %+v", browser.ActionBlockers)
	}
}

func TestUIStatusServiceConnectorStatusExecutePathProbeHealthyRemainsReady(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: "127.0.0.1:19092",
				},
			},
			execAuthToken: "worker-token",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalProbe := runConnectorExecutePathProbe
	t.Cleanup(func() {
		runConnectorExecutePathProbe = originalProbe
	})
	runConnectorExecutePathProbe = func(_ context.Context, _ string, _ string, _ PluginWorkerStatus) (connectorExecuteProbeResult, error) {
		return connectorExecuteProbeResult{
			Ready:      true,
			StatusCode: 400,
			Error:      "unsupported connector execute probe operation \"__connector_execute_probe__\"",
		}, nil
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	mail, ok := findConnectorCard(response.Connectors, "mail")
	if !ok {
		t.Fatalf("expected mail connector card")
	}
	if mail.Status != "ready" {
		t.Fatalf("expected mail status ready when execute-path probe is healthy, got %s", mail.Status)
	}
	if reason, _ := mail.Configuration["status_reason"].(string); reason != connectorReasonReady {
		t.Fatalf("expected mail status reason %q, got %+v", connectorReasonReady, mail.Configuration["status_reason"])
	}
	if ready, _ := mail.Configuration["execute_path_probe_ready"].(bool); !ready {
		t.Fatalf("expected execute_path_probe_ready=true, got %+v", mail.Configuration["execute_path_probe_ready"])
	}
	if statusCode, _ := mail.Configuration["execute_path_probe_status_code"].(int); statusCode != 400 {
		t.Fatalf("expected execute_path_probe_status_code=400, got %+v", mail.Configuration["execute_path_probe_status_code"])
	}
}

func TestUIStatusServiceConnectorStatusIncludesLocalIngestBridgeReadiness(t *testing.T) {
	bridgeRoot := filepath.Join(t.TempDir(), "inbound-bridge")
	t.Setenv(envInboundWatcherInboxDir, bridgeRoot)

	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
		{
			PluginID: "calendar.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
		{
			PluginID: "browser.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	before, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status before setup: %v", err)
	}
	mailBefore, ok := findConnectorCard(before.Connectors, "mail")
	if !ok {
		t.Fatalf("expected mail connector card")
	}
	if ready, ok := mailBefore.Configuration["local_ingest_bridge_ready"].(bool); !ok || ready {
		t.Fatalf("expected local_ingest_bridge_ready=false before setup, got %+v", mailBefore.Configuration["local_ingest_bridge_ready"])
	}

	ensureStatus := EnsureInboundWatcherBridge("")
	if !ensureStatus.Ready {
		t.Fatalf("expected bridge ensure to report ready, got %+v", ensureStatus)
	}

	after, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status after setup: %v", err)
	}
	for _, connectorID := range []string{"mail", "calendar", "browser"} {
		card, ok := findConnectorCard(after.Connectors, connectorID)
		if !ok {
			t.Fatalf("expected connector card %s", connectorID)
		}
		ready, ok := card.Configuration["local_ingest_bridge_ready"].(bool)
		if !ok || !ready {
			t.Fatalf("expected connector %s local_ingest_bridge_ready=true, got %+v", connectorID, card.Configuration["local_ingest_bridge_ready"])
		}
		bridgeConfig, ok := card.Configuration["local_ingest_bridge"].(map[string]any)
		if !ok {
			t.Fatalf("expected connector %s local_ingest_bridge object, got %+v", connectorID, card.Configuration["local_ingest_bridge"])
		}
		if root := strings.TrimSpace(fmt.Sprintf("%v", bridgeConfig["inbox_root"])); root != bridgeRoot {
			t.Fatalf("expected connector %s bridge inbox_root %q, got %q", connectorID, bridgeRoot, root)
		}
	}
}

func TestUIStatusServiceConnectorStatusClassifiesCloudflaredMissingBinary(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: CloudflaredConnectorPluginID,
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalProbe := runCloudflaredConnectorVersionProbe
	t.Cleanup(func() {
		runCloudflaredConnectorVersionProbe = originalProbe
	})
	runCloudflaredConnectorVersionProbe = func(_ context.Context, _ string, _ PluginWorkerStatus) (transport.CloudflaredVersionResponse, error) {
		return transport.CloudflaredVersionResponse{
			WorkspaceID: "ws1",
			Available:   false,
			BinaryPath:  "/missing/cloudflared",
			ExitCode:    -1,
			DryRun:      false,
			Error:       "fork/exec /missing/cloudflared: no such file or directory",
		}, nil
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}

	cloudflared, ok := findConnectorCard(response.Connectors, "cloudflared")
	if !ok {
		t.Fatalf("expected cloudflared connector card")
	}
	if cloudflared.Status != "degraded" {
		t.Fatalf("expected cloudflared status degraded when binary missing, got %s", cloudflared.Status)
	}
	if !strings.Contains(strings.ToLower(cloudflared.Summary), "binary") {
		t.Fatalf("expected cloudflared missing-binary summary, got %q", cloudflared.Summary)
	}
	if reason, _ := cloudflared.Configuration["status_reason"].(string); reason != connectorReasonCloudflaredBinaryMissing {
		t.Fatalf("expected cloudflared status reason %q, got %+v", connectorReasonCloudflaredBinaryMissing, cloudflared.Configuration["status_reason"])
	}
	if action, ok := findRemediationAction(cloudflared.RemediationActions, "install_cloudflared_connector"); !ok {
		t.Fatalf("expected cloudflared remediation action install_cloudflared_connector")
	} else if !action.Recommended {
		t.Fatalf("expected cloudflared install action to be recommended, got %+v", action)
	}
	if _, ok := findRemediationAction(cloudflared.RemediationActions, "request_connector_permission"); ok {
		t.Fatalf("did not expect cloudflared permission remediation action")
	}
}

func TestUIStatusServiceConnectorStatusClassifiesPermissionFailure(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID:  "finder.daemon",
			Kind:      shared.AdapterKindConnector,
			State:     PluginWorkerStateFailed,
			LastError: "Not authorized to send Apple events to Finder. (-1743)",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	finder, ok := findConnectorCard(response.Connectors, "finder")
	if !ok {
		t.Fatalf("expected finder connector card")
	}
	if finder.Status != "degraded" {
		t.Fatalf("expected finder status degraded for permission blocker, got %s", finder.Status)
	}
	if reason, _ := finder.Configuration["status_reason"].(string); reason != connectorReasonPermissionMissing {
		t.Fatalf("expected finder status reason %q, got %+v", connectorReasonPermissionMissing, finder.Configuration["status_reason"])
	}
	requestPermission, ok := findRemediationAction(finder.RemediationActions, "request_connector_permission")
	if !ok {
		t.Fatalf("expected finder request_connector_permission remediation action")
	}
	if !requestPermission.Recommended {
		t.Fatalf("expected finder request permission action to be recommended, got %+v", requestPermission)
	}
	systemSettings, ok := findRemediationAction(finder.RemediationActions, "open_connector_system_settings")
	if !ok {
		t.Fatalf("expected finder open_connector_system_settings remediation action")
	}
	if !systemSettings.Recommended {
		t.Fatalf("expected finder system settings action to be recommended, got %+v", systemSettings)
	}
	if _, ok := findRemediationAction(finder.RemediationActions, "repair_daemon_runtime"); ok {
		t.Fatalf("did not expect repair_daemon_runtime action when permission remediation is required")
	}
}

func TestUIStatusServiceMessagesConnectorIncludesPermissionRequestAction(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	messages, ok := findConnectorCard(response.Connectors, "imessage")
	if !ok {
		t.Fatalf("expected imessage connector card")
	}
	if descriptor, ok := findConfigFieldDescriptor(messages.ConfigFieldDescriptors, "source_db_path"); !ok {
		t.Fatalf("expected messages connector source_db_path descriptor")
	} else if descriptor.Type != "path" || !descriptor.Editable {
		t.Fatalf("unexpected messages connector source_db_path descriptor: %+v", descriptor)
	}
	if permission, ok := messages.Configuration["permission_state"].(string); !ok || permission != "unknown" {
		t.Fatalf("expected messages permission_state=unknown by default, got %+v", messages.Configuration["permission_state"])
	}
	requestPermission, ok := findRemediationAction(messages.RemediationActions, "request_connector_permission")
	if !ok {
		t.Fatalf("expected messages request_connector_permission remediation action")
	}
	if !requestPermission.Enabled {
		t.Fatalf("expected messages request permission action to be enabled, got %+v", requestPermission)
	}
	if requestPermission.Label != "Request Messages Automation Permission" {
		t.Fatalf("expected messages request permission label to explain automation scope, got %q", requestPermission.Label)
	}
	systemSettings, ok := findRemediationAction(messages.RemediationActions, "open_connector_system_settings")
	if !ok {
		t.Fatalf("expected messages open_connector_system_settings remediation action")
	}
	if systemSettings.Destination != "ui://system-settings/privacy/automation" {
		t.Fatalf("expected messages automation system settings destination, got %s", systemSettings.Destination)
	}
	openFullDisk, ok := findRemediationAction(messages.RemediationActions, "open_imessage_system_settings")
	if !ok {
		t.Fatalf("expected messages open_imessage_system_settings remediation action")
	}
	if openFullDisk.Destination != "ui://system-settings/privacy/full-disk-access" {
		t.Fatalf("expected messages full-disk-access settings destination, got %s", openFullDisk.Destination)
	}
	twilio, ok := findConnectorCard(response.Connectors, "twilio")
	if !ok {
		t.Fatalf("expected twilio connector card")
	}
	if descriptor, ok := findConfigFieldDescriptor(twilio.ConfigFieldDescriptors, "auth_token_value"); !ok {
		t.Fatalf("expected twilio connector auth_token_value descriptor")
	} else if !descriptor.Secret || !descriptor.WriteOnly || !descriptor.Editable {
		t.Fatalf("expected twilio auth_token_value descriptor to be secret+write_only+editable, got %+v", descriptor)
	}
}

func TestUIStatusServiceMessagesConnectorFlagsPermissionMissingOnIngestAccessError(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}
	originalProbe := runMessagesStatusProbe
	t.Cleanup(func() {
		runMessagesStatusProbe = originalProbe
	})
	runMessagesStatusProbe = func(request messagesadapter.StatusRequest) messagesadapter.StatusResponse {
		return messagesadapter.StatusResponse{
			Ready:        false,
			Source:       messagesadapter.SourceName,
			SourceScope:  messagesadapter.ResolveSourceScope(request.SourceScope, "/Users/test/Library/Messages/chat.db"),
			SourceDBPath: "/Users/test/Library/Messages/chat.db",
			Transport:    "messages_apple_events",
			Error:        "messages source db path is not readable: open /Users/test/Library/Messages/chat.db: operation not permitted",
		}
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws1', 'ws1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO automation_source_subscriptions(
			id, workspace_id, source, source_scope, status, config_json,
			last_cursor, last_event_id, last_error, created_at, updated_at
		) VALUES (
			'sub.messages.ws1', 'ws1', ?, 'messages.chatdb.default', 'ACTIVE', '{}',
			'', '', 'messages source db path is not readable: open /Users/test/Library/Messages/chat.db: operation not permitted', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z'
		)
		ON CONFLICT(workspace_id, source, source_scope) DO UPDATE SET
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, messagesadapter.SourceName); err != nil {
		t.Fatalf("seed messages ingest failure subscription: %v", err)
	}

	response, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	messages, ok := findConnectorCard(response.Connectors, "imessage")
	if !ok {
		t.Fatalf("expected imessage connector card")
	}
	if reason, _ := messages.Configuration["status_reason"].(string); reason != connectorReasonPermissionMissing {
		t.Fatalf("expected messages status reason %q, got %+v", connectorReasonPermissionMissing, messages.Configuration["status_reason"])
	}
	if permission, _ := messages.Configuration["permission_state"].(string); permission != "missing" {
		t.Fatalf("expected messages permission_state=missing, got %+v", messages.Configuration["permission_state"])
	}
	if !strings.Contains(strings.ToLower(messages.Summary), "full disk access") {
		t.Fatalf("expected messages summary to reference full disk access remediation, got %q", messages.Summary)
	}
}

func TestUIStatusServiceMessagesConnectorClearsStalePermissionDeniedIngestErrorAfterFullDiskGrant(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: messagesWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws1', 'ws1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if _, err := container.DB.Exec(`
		INSERT INTO automation_source_subscriptions(
			id, workspace_id, source, source_scope, status, config_json,
			last_cursor, last_event_id, last_error, created_at, updated_at
		) VALUES (
			'sub.messages.ws1', 'ws1', ?, '/Users/test/Library/Messages/chat.db', 'ACTIVE', '{}',
			'', '', 'messages source db path is not readable: open /Users/test/Library/Messages/chat.db: operation not permitted', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z'
		)
		ON CONFLICT(workspace_id, source, source_scope) DO UPDATE SET
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, messagesadapter.SourceName); err != nil {
		t.Fatalf("seed messages ingest failure subscription: %v", err)
	}

	originalProbe := runMessagesStatusProbe
	t.Cleanup(func() {
		runMessagesStatusProbe = originalProbe
	})
	runMessagesStatusProbe = func(request messagesadapter.StatusRequest) messagesadapter.StatusResponse {
		return messagesadapter.StatusResponse{
			Ready:        true,
			Source:       messagesadapter.SourceName,
			SourceScope:  messagesadapter.ResolveSourceScope(request.SourceScope, "/Users/test/Library/Messages/chat.db"),
			SourceDBPath: "/Users/test/Library/Messages/chat.db",
			Transport:    "messages_apple_events",
		}
	}

	connectorResponse, err := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector status: %v", err)
	}
	imessage, ok := findConnectorCard(connectorResponse.Connectors, "imessage")
	if !ok {
		t.Fatalf("expected imessage connector card")
	}
	if imessage.Status != "ready" {
		t.Fatalf("expected imessage connector ready after full disk probe passes, got %s", imessage.Status)
	}
	if permission, _ := imessage.Configuration["permission_state"].(string); permission != "granted" {
		t.Fatalf("expected imessage permission_state=granted after stale error recovery, got %+v", imessage.Configuration["permission_state"])
	}
	if _, exists := imessage.Configuration["ingest_last_error"]; exists {
		t.Fatalf("expected stale ingest_last_error to be cleared from imessage connector card, got %+v", imessage.Configuration["ingest_last_error"])
	}

	channelResponse, err := service.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel status: %v", err)
	}
	messageChannel, ok := findChannelCard(channelResponse.Channels, "message")
	if !ok {
		t.Fatalf("expected message channel card")
	}
	if messageChannel.Status != "ready" {
		t.Fatalf("expected message channel ready after stale ingest recovery, got %s", messageChannel.Status)
	}

	var persistedLastError string
	if err := container.DB.QueryRow(`
		SELECT COALESCE(last_error, '')
		FROM automation_source_subscriptions
		WHERE workspace_id = 'ws1'
		  AND source = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, messagesadapter.SourceName).Scan(&persistedLastError); err != nil {
		t.Fatalf("load persisted messages ingest error: %v", err)
	}
	if strings.TrimSpace(persistedLastError) != "" {
		t.Fatalf("expected persisted messages ingest error to be cleared, got %q", persistedLastError)
	}
}

func TestUIStatusServiceChannelDiagnosticsIncludeWorkerHealthAndActions(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: twilioWorkerPluginID,
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListChannelDiagnostics(context.Background(), transport.ChannelDiagnosticsRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list channel diagnostics: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Diagnostics) != 3 {
		t.Fatalf("expected diagnostics for 3 logical channels, got %d", len(response.Diagnostics))
	}

	appChat, ok := findChannelDiagnostics(response.Diagnostics, "app")
	if !ok {
		t.Fatalf("expected app diagnostics")
	}
	if appChat.WorkerHealth.Registered {
		t.Fatalf("expected app worker registered=false when worker is missing")
	}
	if _, ok := findRemediationAction(appChat.RemediationActions, "repair_daemon_runtime"); !ok {
		t.Fatalf("expected app diagnostics to include repair_daemon_runtime remediation action")
	}
	if action, ok := findRemediationAction(appChat.RemediationActions, "repair_daemon_runtime"); !ok || action.Intent == "" || action.Label == "" {
		t.Fatalf("expected typed remediation metadata on app diagnostics action, got %+v", action)
	}

	twilioSMS, ok := findChannelDiagnostics(response.Diagnostics, "voice")
	if !ok {
		t.Fatalf("expected voice diagnostics")
	}
	if twilioSMS.WorkerHealth.Worker == nil || twilioSMS.WorkerHealth.Worker.PluginID != twilioWorkerPluginID {
		t.Fatalf("expected voice worker snapshot")
	}
	if _, ok := findRemediationAction(twilioSMS.RemediationActions, "configure_twilio_channel"); !ok {
		t.Fatalf("expected voice diagnostics to include configure_twilio_channel remediation action")
	}
	if action, ok := findRemediationAction(twilioSMS.RemediationActions, "configure_twilio_channel"); !ok || action.Intent != "navigate" {
		t.Fatalf("expected voice diagnostics action intent=navigate, got %+v", action)
	}

	filtered, err := service.ListChannelDiagnostics(context.Background(), transport.ChannelDiagnosticsRequest{
		WorkspaceID: "ws1",
		ChannelID:   "voice",
	})
	if err != nil {
		t.Fatalf("list filtered channel diagnostics: %v", err)
	}
	if len(filtered.Diagnostics) != 1 || filtered.Diagnostics[0].ChannelID != "voice" {
		t.Fatalf("unexpected filtered channel diagnostics payload: %+v", filtered)
	}
}

func TestUIStatusServiceConnectorDiagnosticsIncludeWorkerHealthAndActions(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "calendar.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateFailed,
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	response, err := service.ListConnectorDiagnostics(context.Background(), transport.ConnectorDiagnosticsRequest{WorkspaceID: "ws1"})
	if err != nil {
		t.Fatalf("list connector diagnostics: %v", err)
	}
	if response.WorkspaceID != "ws1" {
		t.Fatalf("expected workspace ws1, got %s", response.WorkspaceID)
	}
	if len(response.Diagnostics) < 4 {
		t.Fatalf("expected diagnostics for at least 4 connectors, got %d", len(response.Diagnostics))
	}

	calendar, ok := findConnectorDiagnostics(response.Diagnostics, "calendar")
	if !ok {
		t.Fatalf("expected calendar diagnostics")
	}
	if !calendar.WorkerHealth.Registered || calendar.WorkerHealth.Worker == nil {
		t.Fatalf("expected calendar worker snapshot")
	}
	if calendar.WorkerHealth.Worker.State != string(PluginWorkerStateFailed) {
		t.Fatalf("expected calendar worker state failed, got %s", calendar.WorkerHealth.Worker.State)
	}

	repairAction, ok := findRemediationAction(calendar.RemediationActions, "repair_daemon_runtime")
	if !ok {
		t.Fatalf("expected calendar diagnostics to include repair_daemon_runtime action")
	}
	if !repairAction.Recommended || !repairAction.Enabled {
		t.Fatalf("expected repair_daemon_runtime to be enabled+recommended, got %+v", repairAction)
	}

	systemSettingsAction, ok := findRemediationAction(calendar.RemediationActions, "open_connector_system_settings")
	if !ok {
		t.Fatalf("expected calendar diagnostics to include open_connector_system_settings action")
	}
	if systemSettingsAction.Destination != "ui://system-settings/privacy/automation" {
		t.Fatalf("unexpected calendar system settings destination: %s", systemSettingsAction.Destination)
	}
	if systemSettingsAction.Intent != "open_system_settings" {
		t.Fatalf("expected open_system_settings intent, got %+v", systemSettingsAction)
	}

	filtered, err := service.ListConnectorDiagnostics(context.Background(), transport.ConnectorDiagnosticsRequest{
		WorkspaceID: "ws1",
		ConnectorID: "calendar",
	})
	if err != nil {
		t.Fatalf("list filtered connector diagnostics: %v", err)
	}
	if len(filtered.Diagnostics) != 1 || filtered.Diagnostics[0].ConnectorID != "calendar" {
		t.Fatalf("unexpected filtered connector diagnostics payload: %+v", filtered)
	}
}

func TestUIStatusServiceRequestConnectorPermissionUsesDaemonRunner(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
	})

	var commandName string
	var commandArgs []string
	runConnectorPermissionCommand = func(_ context.Context, name string, args ...string) (string, error) {
		commandName = name
		commandArgs = append([]string{}, args...)
		return "ok", nil
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.WorkspaceID != "ws1" || response.ConnectorID != "mail" {
		t.Fatalf("unexpected connector permission response identifiers: %+v", response)
	}
	if response.PermissionState != "granted" {
		t.Fatalf("expected granted permission state, got %s", response.PermissionState)
	}
	if !strings.Contains(response.Message, "Personal Agent Daemon") {
		t.Fatalf("expected daemon-attributed success message, got %q", response.Message)
	}
	if commandName != "osascript" {
		t.Fatalf("expected osascript command, got %q", commandName)
	}
	if len(commandArgs) == 0 {
		t.Fatalf("expected osascript args to be populated")
	}
	statusResponse, statusErr := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if statusErr != nil {
		t.Fatalf("list connector status after permission request: %v", statusErr)
	}
	mailCard, found := findConnectorCard(statusResponse.Connectors, "mail")
	if !found {
		t.Fatalf("expected mail connector card in status response")
	}
	if permission, _ := mailCard.Configuration["permission_state"].(string); permission != "granted" {
		t.Fatalf("expected persisted mail permission_state=granted, got %+v", mailCard.Configuration["permission_state"])
	}
}

func TestUIStatusServiceRequestConnectorPermissionUnexpectedSuccessOutputMapsUnknown(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
	})

	runConnectorPermissionCommand = func(_ context.Context, _ string, _ ...string) (string, error) {
		return "permission pending", nil
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "unknown" {
		t.Fatalf("expected unknown permission state for unexpected osascript output, got %s", response.PermissionState)
	}
	if !strings.Contains(strings.ToLower(response.Message), "unexpected result") {
		t.Fatalf("expected unexpected-result remediation copy, got %q", response.Message)
	}
}

func TestUIStatusServiceRequestConnectorPermissionDeniedSignalInOutputMapsMissing(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
	})

	runConnectorPermissionCommand = func(_ context.Context, _ string, _ ...string) (string, error) {
		return "Not authorized to send Apple events to Mail. (-1743)", nil
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "missing" {
		t.Fatalf("expected missing permission state for denied output detail, got %s", response.PermissionState)
	}
	if !strings.Contains(strings.ToLower(response.Message), "allow personal agent daemon") {
		t.Fatalf("expected denied automation remediation message, got %q", response.Message)
	}
}

func TestUIStatusServiceRequestConnectorPermissionMapsAutomationDenied(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
	})

	runConnectorPermissionCommand = func(_ context.Context, _ string, _ ...string) (string, error) {
		return "Not authorized to send Apple events to System Events. (-1743)", fmt.Errorf("exit status 1")
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "browser",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "missing" {
		t.Fatalf("expected missing permission state for denied automation, got %s", response.PermissionState)
	}
	if !strings.Contains(strings.ToLower(response.Message), "allow personal agent daemon") {
		t.Fatalf("expected denied automation remediation message, got %q", response.Message)
	}
}

func TestUIStatusServiceRequestConnectorPermissionRetriesAfterWarmLaunchWhenTargetNotRunning(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
	})

	callSequence := make([]string, 0, 4)
	osascriptCalls := 0
	runConnectorPermissionCommand = func(_ context.Context, name string, _ ...string) (string, error) {
		callSequence = append(callSequence, name)
		switch name {
		case "open":
			return "", nil
		case "osascript":
			osascriptCalls++
			if osascriptCalls == 1 {
				return "Calendar got an error: Application isn’t running. (-600)", fmt.Errorf("exit status 1")
			}
			return "Not authorized to send Apple events to Calendar. (-1743)", fmt.Errorf("exit status 1")
		default:
			t.Fatalf("unexpected command invocation: %s", name)
			return "", nil
		}
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "calendar",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "missing" {
		t.Fatalf("expected missing permission state after warm-launch retry, got %s", response.PermissionState)
	}
	if !strings.Contains(strings.ToLower(response.Message), "allow personal agent daemon") {
		t.Fatalf("expected denied automation remediation guidance after retry, got %q", response.Message)
	}
	if len(callSequence) != 3 || callSequence[0] != "osascript" || callSequence[1] != "open" || callSequence[2] != "osascript" {
		t.Fatalf("expected osascript->open->osascript retry sequence, got %+v", callSequence)
	}
}

func TestUIStatusServiceRequestConnectorPermissionMessagesRequiresAutomationAndFullDiskAccess(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	originalMessagesProbe := runMessagesStatusProbe
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
		runMessagesStatusProbe = originalMessagesProbe
	})

	runnerCalled := false
	runConnectorPermissionCommand = func(_ context.Context, name string, _ ...string) (string, error) {
		if name != "osascript" {
			t.Fatalf("expected messages permission probe to execute osascript, got %s", name)
		}
		runnerCalled = true
		return "Not authorized to send Apple events to Messages. (-1743)", fmt.Errorf("exit status 1")
	}
	runMessagesStatusProbe = func(_ messagesadapter.StatusRequest) messagesadapter.StatusResponse {
		return messagesadapter.StatusResponse{
			Ready:        false,
			Source:       messagesadapter.SourceName,
			SourceScope:  "/Users/test/Library/Messages/chat.db",
			SourceDBPath: "/Users/test/Library/Messages/chat.db",
			Transport:    "messages_apple_events",
			Error:        "messages source db path is not readable: operation not permitted",
		}
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "imessage",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "missing" {
		t.Fatalf("expected messages permission_state=missing when automation + chat db access are denied, got %s", response.PermissionState)
	}
	messageLower := strings.ToLower(response.Message)
	if !strings.Contains(messageLower, "automation") || !strings.Contains(messageLower, "full disk access") {
		t.Fatalf("expected dual-permission remediation in message, got %q", response.Message)
	}
	if !runnerCalled {
		t.Fatalf("expected osascript runner invocation for messages permission request")
	}

	status, statusErr := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if statusErr != nil {
		t.Fatalf("list connector status after messages permission request: %v", statusErr)
	}
	messagesCard, ok := findConnectorCard(status.Connectors, "imessage")
	if !ok {
		t.Fatalf("expected imessage connector card in status response")
	}
	if permission, _ := messagesCard.Configuration["permission_state"].(string); permission != "missing" {
		t.Fatalf("expected persisted messages permission_state=missing, got %+v", messagesCard.Configuration["permission_state"])
	}
	if automationPermission, _ := messagesCard.Configuration["messages_automation_permission_state"].(string); automationPermission != "missing" {
		t.Fatalf("expected persisted messages_automation_permission_state=missing, got %+v", messagesCard.Configuration["messages_automation_permission_state"])
	}
	if fullDiskPermission, _ := messagesCard.Configuration["messages_full_disk_permission_state"].(string); fullDiskPermission != "missing" {
		t.Fatalf("expected persisted messages_full_disk_permission_state=missing, got %+v", messagesCard.Configuration["messages_full_disk_permission_state"])
	}
}

func TestUIStatusServiceRequestConnectorPermissionMessagesGrantedWhenBothChecksPass(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalRunner := runConnectorPermissionCommand
	originalMessagesProbe := runMessagesStatusProbe
	t.Cleanup(func() {
		runConnectorPermissionCommand = originalRunner
		runMessagesStatusProbe = originalMessagesProbe
	})

	runnerCalled := false
	runConnectorPermissionCommand = func(_ context.Context, name string, _ ...string) (string, error) {
		if name != "osascript" {
			t.Fatalf("expected messages permission probe to execute osascript, got %s", name)
		}
		runnerCalled = true
		return "4", nil
	}
	runMessagesStatusProbe = func(_ messagesadapter.StatusRequest) messagesadapter.StatusResponse {
		return messagesadapter.StatusResponse{
			Ready:        true,
			Source:       messagesadapter.SourceName,
			SourceScope:  "/Users/test/Library/Messages/chat.db",
			SourceDBPath: "/Users/test/Library/Messages/chat.db",
			Transport:    "messages_apple_events",
		}
	}

	response, err := service.RequestConnectorPermission(context.Background(), transport.ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "imessage",
	})
	if err != nil {
		t.Fatalf("request connector permission: %v", err)
	}
	if response.PermissionState != "granted" {
		t.Fatalf("expected messages permission_state=granted when chat db is readable, got %s", response.PermissionState)
	}
	messageLower := strings.ToLower(response.Message)
	if !strings.Contains(messageLower, "automation") || !strings.Contains(messageLower, "full disk access") {
		t.Fatalf("expected messages readiness guidance to reference automation + full disk access, got %q", response.Message)
	}
	if !runnerCalled {
		t.Fatalf("expected osascript runner invocation for messages permission request")
	}

	status, statusErr := service.ListConnectorStatus(context.Background(), transport.ConnectorStatusRequest{WorkspaceID: "ws1"})
	if statusErr != nil {
		t.Fatalf("list connector status after messages permission request: %v", statusErr)
	}
	messagesCard, ok := findConnectorCard(status.Connectors, "imessage")
	if !ok {
		t.Fatalf("expected imessage connector card in status response")
	}
	if permission, _ := messagesCard.Configuration["permission_state"].(string); permission != "granted" {
		t.Fatalf("expected persisted messages permission_state=granted, got %+v", messagesCard.Configuration["permission_state"])
	}
	if automationPermission, _ := messagesCard.Configuration["messages_automation_permission_state"].(string); automationPermission != "granted" {
		t.Fatalf("expected persisted messages_automation_permission_state=granted, got %+v", messagesCard.Configuration["messages_automation_permission_state"])
	}
	if fullDiskPermission, _ := messagesCard.Configuration["messages_full_disk_permission_state"].(string); fullDiskPermission != "granted" {
		t.Fatalf("expected persisted messages_full_disk_permission_state=granted, got %+v", messagesCard.Configuration["messages_full_disk_permission_state"])
	}
}

func TestUIStatusServiceConnectorTestOperationValidatesExecutePath(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: "127.0.0.1:19093",
				},
			},
			execAuthToken: "worker-token",
		},
	})
	service, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}

	originalProbe := runConnectorExecutePathProbe
	t.Cleanup(func() {
		runConnectorExecutePathProbe = originalProbe
	})

	runConnectorExecutePathProbe = func(_ context.Context, _ string, _ string, _ PluginWorkerStatus) (connectorExecuteProbeResult, error) {
		return connectorExecuteProbeResult{
			Ready: false,
			Error: "dial tcp 127.0.0.1:19093: connect: connection refused",
		}, fmt.Errorf("dial tcp 127.0.0.1:19093: connect: connection refused")
	}
	failed, err := service.TestConnectorOperation(context.Background(), transport.ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Operation:   "health",
	})
	if err != nil {
		t.Fatalf("connector test operation failed-path call: %v", err)
	}
	if failed.Success {
		t.Fatalf("expected connector test operation success=false when execute path probe fails")
	}
	if failed.Status != "degraded" {
		t.Fatalf("expected connector test operation status degraded, got %s", failed.Status)
	}
	if !strings.Contains(strings.ToLower(failed.Summary), "execute endpoint probe failed") {
		t.Fatalf("expected execute-path failure summary, got %q", failed.Summary)
	}
	if ready, _ := failed.Details.AsMap()["execute_path_ready"].(bool); ready {
		t.Fatalf("expected failed execute_path_ready=false, got %+v", failed.Details.AsMap()["execute_path_ready"])
	}

	runConnectorExecutePathProbe = func(_ context.Context, _ string, _ string, _ PluginWorkerStatus) (connectorExecuteProbeResult, error) {
		return connectorExecuteProbeResult{
			Ready:      true,
			StatusCode: 400,
			Error:      "unsupported connector execute probe operation",
		}, nil
	}
	healthy, err := service.TestConnectorOperation(context.Background(), transport.ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Operation:   "health",
	})
	if err != nil {
		t.Fatalf("connector test operation healthy-path call: %v", err)
	}
	if !healthy.Success {
		t.Fatalf("expected connector test operation success=true when execute path probe is reachable")
	}
	if healthy.Status != "ok" {
		t.Fatalf("expected connector test operation status ok, got %s", healthy.Status)
	}
	if ready, _ := healthy.Details.AsMap()["execute_path_ready"].(bool); !ready {
		t.Fatalf("expected healthy execute_path_ready=true, got %+v", healthy.Details.AsMap()["execute_path_ready"])
	}
	if code, _ := healthy.Details.AsMap()["execute_path_probe_status_code"].(int); code != 400 {
		t.Fatalf("expected healthy execute_path_probe_status_code=400, got %+v", healthy.Details.AsMap()["execute_path_probe_status_code"])
	}
}

func TestChannelActionReadinessClassification(t *testing.T) {
	readyReadiness, readyBlockers := channelActionReadiness(transport.ChannelStatusCard{
		Status: "ready",
		Configuration: map[string]any{
			"status_reason": channelReasonReady,
		},
	})
	if readyReadiness != "ready" {
		t.Fatalf("expected ready readiness, got %q", readyReadiness)
	}
	if len(readyBlockers) != 0 {
		t.Fatalf("expected no ready blockers, got %+v", readyBlockers)
	}

	blockedReadiness, blockedBlockers := channelActionReadiness(transport.ChannelStatusCard{
		Status:  "not_configured",
		Summary: "Twilio channel setup is incomplete.",
		RemediationActions: []transport.DiagnosticsRemediationAction{
			{
				Identifier:  "configure_twilio_channel",
				Recommended: true,
			},
		},
	})
	if blockedReadiness != "blocked" {
		t.Fatalf("expected blocked readiness, got %q", blockedReadiness)
	}
	if len(blockedBlockers) != 1 {
		t.Fatalf("expected one blocked readiness blocker, got %+v", blockedBlockers)
	}
	if blockedBlockers[0].Code != "config_incomplete" {
		t.Fatalf("expected config_incomplete blocker, got %+v", blockedBlockers[0])
	}
	if blockedBlockers[0].RemediationAction != "configure_twilio_channel" {
		t.Fatalf("expected configure_twilio_channel remediation action, got %+v", blockedBlockers[0])
	}
	if !strings.Contains(strings.ToLower(blockedBlockers[0].Message), "incomplete") {
		t.Fatalf("expected blocker message to include summary context, got %q", blockedBlockers[0].Message)
	}

	degradedReadiness, degradedBlockers := channelActionReadiness(transport.ChannelStatusCard{
		Status: "degraded",
		Configuration: map[string]any{
			"status_reason": channelReasonWorkerFailed,
		},
		RemediationActions: []transport.DiagnosticsRemediationAction{
			{
				Identifier: "repair_daemon_runtime",
			},
		},
	})
	if degradedReadiness != "degraded" {
		t.Fatalf("expected degraded readiness, got %q", degradedReadiness)
	}
	if len(degradedBlockers) != 1 || degradedBlockers[0].Code != "worker_unavailable" {
		t.Fatalf("expected worker_unavailable degraded blocker, got %+v", degradedBlockers)
	}
}

func TestConnectorActionReadinessClassification(t *testing.T) {
	blockedReadiness, blockedBlockers := connectorActionReadiness(transport.ConnectorStatusCard{
		Status: "failed",
		Configuration: map[string]any{
			"status_reason": connectorReasonPermissionMissing,
		},
		RemediationActions: []transport.DiagnosticsRemediationAction{
			{
				Identifier:  "open_connector_system_settings",
				Recommended: true,
			},
		},
	})
	if blockedReadiness != "blocked" {
		t.Fatalf("expected blocked readiness, got %q", blockedReadiness)
	}
	if len(blockedBlockers) != 1 {
		t.Fatalf("expected one blocked readiness blocker, got %+v", blockedBlockers)
	}
	if blockedBlockers[0].Code != "permission_missing" {
		t.Fatalf("expected permission_missing blocker, got %+v", blockedBlockers[0])
	}
	if blockedBlockers[0].RemediationAction != "open_connector_system_settings" {
		t.Fatalf("expected open_connector_system_settings remediation action, got %+v", blockedBlockers[0])
	}

	degradedReadiness, degradedBlockers := connectorActionReadiness(transport.ConnectorStatusCard{
		Status: "degraded",
		Configuration: map[string]any{
			"status_reason": connectorReasonWorkerFailed,
		},
	})
	if degradedReadiness != "degraded" {
		t.Fatalf("expected degraded readiness, got %q", degradedReadiness)
	}
	if len(degradedBlockers) != 1 || degradedBlockers[0].Code != "worker_unavailable" {
		t.Fatalf("expected worker_unavailable degraded blocker, got %+v", degradedBlockers)
	}

	executeDegradedReadiness, executeDegradedBlockers := connectorActionReadiness(transport.ConnectorStatusCard{
		Status: "degraded",
		Configuration: map[string]any{
			"status_reason": connectorReasonExecutePathFailure,
		},
	})
	if executeDegradedReadiness != "degraded" {
		t.Fatalf("expected execute-path degraded readiness, got %q", executeDegradedReadiness)
	}
	if len(executeDegradedBlockers) != 1 || executeDegradedBlockers[0].Code != "execute_path_unavailable" {
		t.Fatalf("expected execute_path_unavailable degraded blocker, got %+v", executeDegradedBlockers)
	}

	readyReadiness, readyBlockers := connectorActionReadiness(transport.ConnectorStatusCard{
		Status: "ready",
		Configuration: map[string]any{
			"status_reason": connectorReasonReady,
		},
	})
	if readyReadiness != "ready" {
		t.Fatalf("expected ready readiness, got %q", readyReadiness)
	}
	if len(readyBlockers) != 0 {
		t.Fatalf("expected no ready blockers, got %+v", readyBlockers)
	}
}

func findChannelCard(cards []transport.ChannelStatusCard, channelID string) (transport.ChannelStatusCard, bool) {
	for _, card := range cards {
		if card.ChannelID == channelID {
			return card, true
		}
	}
	return transport.ChannelStatusCard{}, false
}

func findConnectorCard(cards []transport.ConnectorStatusCard, connectorID string) (transport.ConnectorStatusCard, bool) {
	for _, card := range cards {
		if card.ConnectorID == connectorID {
			return card, true
		}
	}
	return transport.ConnectorStatusCard{}, false
}

func findChannelDiagnostics(
	diagnostics []transport.ChannelDiagnosticsSummary,
	channelID string,
) (transport.ChannelDiagnosticsSummary, bool) {
	for _, summary := range diagnostics {
		if summary.ChannelID == channelID {
			return summary, true
		}
	}
	return transport.ChannelDiagnosticsSummary{}, false
}

func findConnectorDiagnostics(
	diagnostics []transport.ConnectorDiagnosticsSummary,
	connectorID string,
) (transport.ConnectorDiagnosticsSummary, bool) {
	for _, summary := range diagnostics {
		if summary.ConnectorID == connectorID {
			return summary, true
		}
	}
	return transport.ConnectorDiagnosticsSummary{}, false
}

func findRemediationAction(
	actions []transport.DiagnosticsRemediationAction,
	identifier string,
) (transport.DiagnosticsRemediationAction, bool) {
	for _, action := range actions {
		if action.Identifier == identifier {
			return action, true
		}
	}
	return transport.DiagnosticsRemediationAction{}, false
}

func findConfigFieldDescriptor(
	descriptors []transport.ConfigFieldDescriptor,
	key string,
) (transport.ConfigFieldDescriptor, bool) {
	for _, descriptor := range descriptors {
		if descriptor.Key == key {
			return descriptor, true
		}
	}
	return transport.ConfigFieldDescriptor{}, false
}
