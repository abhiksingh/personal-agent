package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"personalagent/runtime/internal/channelconfig"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

const (
	appChatWorkerPluginID   = "app_chat.daemon"
	messagesWorkerPluginID  = "messages.daemon"
	twilioWorkerPluginID    = "twilio.daemon"
	uiConfigRecordIDPrefix  = "ui.config."
	uiChannelConfigPrefix   = "ui.channel."
	uiConnectorConfigPrefix = "ui.connector."

	channelReasonReady          = "ready"
	channelReasonNotConfigured  = "not_configured"
	channelReasonWorkerMissing  = "worker_missing"
	channelReasonWorkerStarting = "worker_starting"
	channelReasonWorkerStopped  = "worker_stopped"
	channelReasonWorkerFailed   = "worker_failed"
	channelReasonIngestFailure  = "ingest_failure"

	connectorReasonReady                     = "ready"
	connectorReasonWorkerMissing             = "worker_missing"
	connectorReasonWorkerStarting            = "worker_starting"
	connectorReasonWorkerStopped             = "worker_stopped"
	connectorReasonWorkerFailed              = "worker_failed"
	connectorReasonNotConfigured             = "not_configured"
	connectorReasonCredentialsIncomplete     = "credentials_incomplete"
	connectorReasonIngestFailure             = "ingest_failure"
	connectorReasonRuntimeFailure            = "runtime_failure"
	connectorReasonExecutePathFailure        = "execute_path_failure"
	connectorReasonPermissionMissing         = "permission_missing"
	connectorReasonCloudflaredBinaryMissing  = "cloudflared_binary_missing"
	connectorReasonCloudflaredRuntimeFailure = "cloudflared_runtime_failure"

	channelConnectorFallbackPolicyPriorityOrder = "priority_order"
)

var channelConnectorMappingDefaultCapabilities = map[string][]string{
	"builtin.app": {
		"channel.app_chat.send",
		"channel.app_chat.status",
	},
	"imessage": {
		"channel.messages.send",
		"channel.messages.status",
		"channel.messages.ingest_poll",
	},
	"twilio": {
		"channel.twilio.check",
		"channel.twilio.sms.send",
		"channel.twilio.voice.start_call",
	},
}

var channelConnectorMappingCapabilityRequirements = map[string]map[string][]string{
	"app": {
		"builtin.app": {"channel.app_chat.send"},
	},
	"message": {
		"imessage": {"channel.messages.send"},
		"twilio":   {"channel.twilio.sms.send"},
	},
	"voice": {
		"twilio": {"channel.twilio.voice.start_call"},
	},
}

type channelConnectorBindingRecord struct {
	ID          string
	WorkspaceID string
	ChannelID   string
	ConnectorID string
	Enabled     bool
	Priority    int
	CreatedAt   string
	UpdatedAt   string
}

var runConnectorPermissionCommand = func(ctx context.Context, name string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

var runMessagesStatusProbe = func(request messagesadapter.StatusRequest) messagesadapter.StatusResponse {
	return messagesadapter.Status(request)
}

var runCloudflaredConnectorVersionProbe = func(ctx context.Context, workspaceID string, worker PluginWorkerStatus) (transport.CloudflaredVersionResponse, error) {
	return probeCloudflaredConnectorVersion(ctx, workspaceID, worker)
}

type connectorPermissionProbeSpec struct {
	connectorID  string
	displayName  string
	systemTarget string
	launchApp    string
	appleScript  []string
}

var connectorPermissionProbeSpecs = map[string]connectorPermissionProbeSpec{
	"imessage": {
		connectorID:  "imessage",
		displayName:  "iMessage",
		systemTarget: "Privacy & Security > Full Disk Access",
		appleScript: []string{
			`tell application id "com.apple.MobileSMS"`,
			`set chatCount to count of chats`,
			`return chatCount as text`,
			`end tell`,
		},
	},
	"mail": {
		connectorID:  "mail",
		displayName:  "Mail",
		systemTarget: "Privacy & Security > Automation",
		launchApp:    "Mail",
		appleScript: []string{
			`tell application id "com.apple.mail"`,
			`set mailboxCount to count of mailboxes`,
			`return mailboxCount as text`,
			`end tell`,
		},
	},
	"calendar": {
		connectorID:  "calendar",
		displayName:  "Calendar",
		systemTarget: "Privacy & Security > Automation",
		launchApp:    "Calendar",
		appleScript: []string{
			`tell application id "com.apple.iCal"`,
			`set calendarCount to count of calendars`,
			`return calendarCount as text`,
			`end tell`,
		},
	},
	"browser": {
		connectorID:  "browser",
		displayName:  "Safari",
		systemTarget: "Privacy & Security > Automation",
		launchApp:    "Safari",
		appleScript: []string{
			`tell application id "com.apple.Safari"`,
			`set windowCount to count of windows`,
			`return windowCount as text`,
			`end tell`,
		},
	},
	"finder": {
		connectorID:  "finder",
		displayName:  "Finder",
		systemTarget: "Privacy & Security > Automation",
		launchApp:    "Finder",
		appleScript: []string{
			`tell application id "com.apple.finder"`,
			`set windowCount to count of Finder windows`,
			`return windowCount as text`,
			`end tell`,
		},
	},
}

type UIStatusService struct {
	container   *ServiceContainer
	twilioStore *channelconfig.SQLiteTwilioStore
}

var _ transport.UIStatusService = (*UIStatusService)(nil)

func NewUIStatusService(container *ServiceContainer) (*UIStatusService, error) {
	if container == nil || container.DB == nil {
		return nil, fmt.Errorf("service container with db is required")
	}
	return &UIStatusService{
		container:   container,
		twilioStore: channelconfig.NewSQLiteTwilioStore(container.DB),
	}, nil
}

func (s *UIStatusService) loadTwilioConfig(ctx context.Context, workspace string) (channelconfig.TwilioConfig, bool, error) {
	config, err := s.twilioStore.Get(ctx, workspace)
	if err == nil {
		return config, true, nil
	}
	if errors.Is(err, channelconfig.ErrTwilioNotConfigured) {
		return channelconfig.TwilioConfig{
			WorkspaceID: workspace,
			Endpoint:    channelconfig.DefaultTwilioEndpoint(),
		}, false, nil
	}
	return channelconfig.TwilioConfig{}, false, fmt.Errorf("load twilio config: %w", err)
}

func isTwilioChannelConfigID(channelID string) bool {
	switch strings.ToLower(strings.TrimSpace(channelID)) {
	case "twilio", "message", "voice":
		return true
	default:
		return false
	}
}
