package daemonruntime

import (
	"errors"
	"os/exec"
	"sync"
	"time"
)

const (
	twilioWebhookSignatureModeStrict = "strict"
	twilioWebhookSignatureModeBypass = "bypass"
	twilioWebhookVoiceResponseJSON   = "json"
	twilioWebhookVoiceResponseTwiML  = "twiml"
)

const (
	twilioWebhookAPIVersion                = "v1"
	twilioWebhookDefaultProjectName        = "personalagent"
	twilioWebhookProjectNameEnvKey         = "PA_PROJECT_NAME"
	twilioWebhookCloudflaredModeAuto       = "auto"
	twilioWebhookCloudflaredModeOff        = "off"
	twilioWebhookCloudflaredModeRequired   = "required"
	twilioWebhookCloudflaredStartupTimeout = 2 * time.Second
	twilioCloudflaredDryRunPublicBaseURL   = "https://dry-run.trycloudflare.com"
	twilioCloudflaredBinaryName            = "cloudflared"
	twilioCloudflaredBinaryOverrideEnv     = "PA_CLOUDFLARED_BINARY"
	twilioWebhookReplayTargetAllowlistEnv  = "PA_TWILIO_WEBHOOK_REPLAY_TARGET_ALLOWLIST"
	twilioWebhookRequestURLOverrideHeader  = "X-Twilio-Request-URL"
	twilioWebhookReplayMarkerHeader        = "X-PersonalAgent-Twilio-Replay"
)

const (
	defaultTwilioWebhookVoiceGreeting = "Personal Agent here. How can I help you today?"
	defaultTwilioWebhookVoiceFallback = "I did not catch that. Please repeat your request."
)

const (
	defaultTwilioWebhookReadHeaderTimeout = 2 * time.Second
	defaultTwilioWebhookReadTimeout       = 10 * time.Second
	defaultTwilioWebhookWriteTimeout      = 10 * time.Second
	defaultTwilioWebhookIdleTimeout       = 60 * time.Second
	defaultTwilioWebhookMaxRequestBytes   = int64(64 * 1024)
	defaultTwilioWebhookMaxFormFields     = 128
	twilioWebhookReplayMaxResponseBytes   = int64(1 * 1024 * 1024)
)

var errTwilioCloudflaredNotInstalled = errors.New("cloudflared binary is not installed")

type twilioWebhookAssistantOptions struct {
	Enabled      bool
	TaskClass    string
	SystemPrompt string
	MaxHistory   int
	ReplyTimeout time.Duration
}

type twilioWebhookSMSResponse struct {
	WorkspaceID              string `json:"workspace_id"`
	Accepted                 bool   `json:"accepted"`
	Replayed                 bool   `json:"replayed"`
	EventID                  string `json:"event_id,omitempty"`
	ThreadID                 string `json:"thread_id,omitempty"`
	MessageSID               string `json:"message_sid,omitempty"`
	AssistantReply           string `json:"assistant_reply,omitempty"`
	AssistantOperationID     string `json:"assistant_operation_id,omitempty"`
	AssistantDelivered       bool   `json:"assistant_delivered,omitempty"`
	AssistantProviderReceipt string `json:"assistant_provider_receipt,omitempty"`
	AssistantError           string `json:"assistant_error,omitempty"`
	Error                    string `json:"error,omitempty"`
}

type twilioWebhookVoiceResponse struct {
	WorkspaceID              string `json:"workspace_id"`
	Accepted                 bool   `json:"accepted"`
	Replayed                 bool   `json:"replayed"`
	ProviderEventID          string `json:"provider_event_id"`
	CallSID                  string `json:"call_sid"`
	CallSessionID            string `json:"call_session_id,omitempty"`
	ThreadID                 string `json:"thread_id,omitempty"`
	CallStatus               string `json:"call_status,omitempty"`
	StatusEventID            string `json:"status_event_id,omitempty"`
	TranscriptEventID        string `json:"transcript_event_id,omitempty"`
	AssistantReply           string `json:"assistant_reply,omitempty"`
	AssistantReplySource     string `json:"assistant_reply_source,omitempty"`
	AssistantReplyTranscript string `json:"assistant_reply_transcript,omitempty"`
	AssistantError           string `json:"assistant_error,omitempty"`
	Error                    string `json:"error,omitempty"`
}

type twilioCloudflaredTunnelSession struct {
	BinaryPath    string
	PublicBaseURL string
	DryRun        bool

	mu        sync.Mutex
	cmd       *exec.Cmd
	waitCh    chan error
	closeOnce sync.Once
}
