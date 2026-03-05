package transport

import "time"

type MessagesIngestRequest struct {
	WorkspaceID  string `json:"workspace_id"`
	SourceScope  string `json:"source_scope,omitempty"`
	SourceDBPath string `json:"source_db_path,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type MessagesIngestEventRecord struct {
	SourceEventID string `json:"source_event_id"`
	SourceCursor  string `json:"source_cursor"`
	EventID       string `json:"event_id"`
	ThreadID      string `json:"thread_id"`
	Replayed      bool   `json:"replayed"`
}

type MessagesIngestResponse struct {
	WorkspaceID  string                      `json:"workspace_id"`
	Source       string                      `json:"source"`
	SourceScope  string                      `json:"source_scope"`
	SourceDBPath string                      `json:"source_db_path,omitempty"`
	CursorStart  string                      `json:"cursor_start,omitempty"`
	CursorEnd    string                      `json:"cursor_end,omitempty"`
	Polled       int                         `json:"polled"`
	Accepted     int                         `json:"accepted"`
	Replayed     int                         `json:"replayed"`
	Events       []MessagesIngestEventRecord `json:"events,omitempty"`
}

type MailRuleIngestRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	SourceScope      string `json:"source_scope,omitempty"`
	SourceEventID    string `json:"source_event_id,omitempty"`
	SourceCursor     string `json:"source_cursor,omitempty"`
	MessageID        string `json:"message_id,omitempty"`
	ThreadRef        string `json:"thread_ref,omitempty"`
	InReplyTo        string `json:"in_reply_to,omitempty"`
	ReferencesHeader string `json:"references_header,omitempty"`
	FromAddress      string `json:"from_address"`
	ToAddress        string `json:"to_address,omitempty"`
	Subject          string `json:"subject,omitempty"`
	BodyText         string `json:"body_text,omitempty"`
	OccurredAt       string `json:"occurred_at,omitempty"`
}

type MailRuleIngestResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	Source        string `json:"source"`
	SourceScope   string `json:"source_scope"`
	SourceEventID string `json:"source_event_id"`
	SourceCursor  string `json:"source_cursor,omitempty"`
	Accepted      bool   `json:"accepted"`
	Replayed      bool   `json:"replayed"`
	EventID       string `json:"event_id,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type CalendarChangeIngestRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	SourceScope   string `json:"source_scope,omitempty"`
	SourceEventID string `json:"source_event_id,omitempty"`
	SourceCursor  string `json:"source_cursor,omitempty"`
	CalendarID    string `json:"calendar_id,omitempty"`
	CalendarName  string `json:"calendar_name,omitempty"`
	EventUID      string `json:"event_uid,omitempty"`
	ChangeType    string `json:"change_type,omitempty"`
	Title         string `json:"title,omitempty"`
	Notes         string `json:"notes,omitempty"`
	Location      string `json:"location,omitempty"`
	StartsAt      string `json:"starts_at,omitempty"`
	EndsAt        string `json:"ends_at,omitempty"`
	OccurredAt    string `json:"occurred_at,omitempty"`
}

type CalendarChangeIngestResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	Source        string `json:"source"`
	SourceScope   string `json:"source_scope"`
	SourceEventID string `json:"source_event_id"`
	SourceCursor  string `json:"source_cursor,omitempty"`
	Accepted      bool   `json:"accepted"`
	Replayed      bool   `json:"replayed"`
	EventID       string `json:"event_id,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	EventUID      string `json:"event_uid,omitempty"`
	ChangeType    string `json:"change_type,omitempty"`
	Error         string `json:"error,omitempty"`
}

type BrowserEventIngestRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	SourceScope   string `json:"source_scope,omitempty"`
	SourceEventID string `json:"source_event_id,omitempty"`
	SourceCursor  string `json:"source_cursor,omitempty"`
	WindowID      string `json:"window_id,omitempty"`
	TabID         string `json:"tab_id,omitempty"`
	PageURL       string `json:"page_url,omitempty"`
	PageTitle     string `json:"page_title,omitempty"`
	EventType     string `json:"event_type,omitempty"`
	PayloadText   string `json:"payload_text,omitempty"`
	OccurredAt    string `json:"occurred_at,omitempty"`
}

type BrowserEventIngestResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	Source        string `json:"source"`
	SourceScope   string `json:"source_scope"`
	SourceEventID string `json:"source_event_id"`
	SourceCursor  string `json:"source_cursor,omitempty"`
	Accepted      bool   `json:"accepted"`
	Replayed      bool   `json:"replayed"`
	EventID       string `json:"event_id,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	EventType     string `json:"event_type,omitempty"`
	PageURL       string `json:"page_url,omitempty"`
	Error         string `json:"error,omitempty"`
}

type TwilioSetRequest struct {
	WorkspaceID          string `json:"workspace_id"`
	AccountSIDSecretName string `json:"account_sid_secret_name"`
	AuthTokenSecretName  string `json:"auth_token_secret_name"`
	SMSNumber            string `json:"sms_number"`
	VoiceNumber          string `json:"voice_number"`
	Endpoint             string `json:"endpoint,omitempty"`
}

type TwilioGetRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type TwilioCheckRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type TwilioConfigRecord struct {
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

type TwilioCheckResult struct {
	Endpoint   string `json:"endpoint"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Message    string `json:"message"`
}

type TwilioCheckResponse struct {
	WorkspaceID string             `json:"workspace_id"`
	Success     bool               `json:"success"`
	Config      TwilioConfigRecord `json:"config"`
	Result      TwilioCheckResult  `json:"result"`
	Error       string             `json:"error,omitempty"`
}

type TwilioSMSChatTurnRequest struct {
	WorkspaceID    string `json:"workspace_id"`
	To             string `json:"to"`
	Message        string `json:"message"`
	OperationID    string `json:"operation_id"`
	TaskClass      string `json:"task_class,omitempty"`
	SystemPrompt   string `json:"system_prompt,omitempty"`
	MaxHistory     int    `json:"max_history,omitempty"`
	ReplyTimeoutMS int    `json:"reply_timeout_ms,omitempty"`
}

type TwilioSMSChatTurn struct {
	OperationID          string `json:"operation_id"`
	Message              string `json:"message"`
	Success              bool   `json:"success"`
	Delivered            bool   `json:"delivered"`
	Channel              string `json:"channel,omitempty"`
	ProviderReceipt      string `json:"provider_receipt,omitempty"`
	IdempotentReplay     bool   `json:"idempotent_replay,omitempty"`
	ThreadID             string `json:"thread_id,omitempty"`
	AssistantReply       string `json:"assistant_reply,omitempty"`
	AssistantOperationID string `json:"assistant_operation_id,omitempty"`
	AssistantError       string `json:"assistant_error,omitempty"`
	Error                string `json:"error,omitempty"`
}

type TwilioStartCallRequest struct {
	WorkspaceID string `json:"workspace_id"`
	To          string `json:"to"`
	From        string `json:"from,omitempty"`
	TwimlURL    string `json:"twiml_url"`
}

type TwilioStartCallResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	CallSID       string `json:"call_sid"`
	CallSessionID string `json:"call_session_id"`
	ThreadID      string `json:"thread_id"`
	Status        string `json:"status"`
	Direction     string `json:"direction"`
}

type TwilioCallStatusRequest struct {
	WorkspaceID string `json:"workspace_id"`
	CallSID     string `json:"call_sid,omitempty"`
	Limit       int    `json:"limit"`
}

type TwilioCallStatusRecord struct {
	SessionID      string `json:"session_id"`
	WorkspaceID    string `json:"workspace_id"`
	Provider       string `json:"provider"`
	ProviderCallID string `json:"provider_call_id"`
	ThreadID       string `json:"thread_id"`
	Direction      string `json:"direction"`
	FromAddress    string `json:"from_address,omitempty"`
	ToAddress      string `json:"to_address,omitempty"`
	Status         string `json:"status"`
	StartedAt      string `json:"started_at,omitempty"`
	EndedAt        string `json:"ended_at,omitempty"`
	UpdatedAt      string `json:"updated_at"`
}

type TwilioCallStatusResponse struct {
	WorkspaceID string                   `json:"workspace_id"`
	CallSID     string                   `json:"call_sid,omitempty"`
	Sessions    []TwilioCallStatusRecord `json:"sessions"`
}

type TwilioTranscriptRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id,omitempty"`
	CallSID     string `json:"call_sid,omitempty"`
	Limit       int    `json:"limit"`
}

type TwilioTranscriptEvent struct {
	EventID          string `json:"event_id"`
	ThreadID         string `json:"thread_id"`
	Channel          string `json:"channel"`
	EventType        string `json:"event_type"`
	Direction        string `json:"direction"`
	AssistantEmitted bool   `json:"assistant_emitted"`
	BodyText         string `json:"body_text,omitempty"`
	SenderAddress    string `json:"sender_address,omitempty"`
	OccurredAt       string `json:"occurred_at"`
}

type TwilioTranscriptResponse struct {
	WorkspaceID string                  `json:"workspace_id"`
	ThreadID    string                  `json:"thread_id,omitempty"`
	CallSID     string                  `json:"call_sid,omitempty"`
	Events      []TwilioTranscriptEvent `json:"events"`
}

type TwilioWebhookServeRequest struct {
	WorkspaceID                 string `json:"workspace_id"`
	ListenAddress               string `json:"listen_address"`
	SignatureMode               string `json:"signature_mode"`
	CloudflaredMode             string `json:"cloudflared_mode,omitempty"`
	CloudflaredStartupTimeoutMS int64  `json:"cloudflared_startup_timeout_ms,omitempty"`
	AssistantReplies            bool   `json:"assistant_replies"`
	AssistantTaskClass          string `json:"assistant_task_class"`
	AssistantSystemPrompt       string `json:"assistant_system_prompt,omitempty"`
	AssistantMaxHistory         int    `json:"assistant_max_history"`
	AssistantReplyTimeoutMS     int64  `json:"assistant_reply_timeout_ms"`
	VoiceResponseMode           string `json:"voice_response_mode"`
	VoiceGreeting               string `json:"voice_greeting,omitempty"`
	VoiceFallback               string `json:"voice_fallback,omitempty"`
	SMSPath                     string `json:"sms_path"`
	VoicePath                   string `json:"voice_path"`
	RunForMS                    int64  `json:"run_for_ms"`
}

type TwilioWebhookServeResponse struct {
	WorkspaceID           string `json:"workspace_id"`
	SignatureMode         string `json:"signature_mode"`
	ListenAddress         string `json:"listen_address"`
	LocalSMSWebhookURL    string `json:"local_sms_webhook_url,omitempty"`
	LocalVoiceWebhookURL  string `json:"local_voice_webhook_url,omitempty"`
	SMSWebhookURL         string `json:"sms_webhook_url"`
	VoiceWebhookURL       string `json:"voice_webhook_url"`
	PublicBaseURL         string `json:"public_base_url,omitempty"`
	CloudflaredMode       string `json:"cloudflared_mode,omitempty"`
	CloudflaredAvailable  bool   `json:"cloudflared_available,omitempty"`
	CloudflaredActive     bool   `json:"cloudflared_active,omitempty"`
	CloudflaredDryRun     bool   `json:"cloudflared_dry_run,omitempty"`
	CloudflaredBinaryPath string `json:"cloudflared_binary_path,omitempty"`
	AssistantReplies      bool   `json:"assistant_replies"`
	AssistantTaskClass    string `json:"assistant_task_class"`
	VoiceResponseMode     string `json:"voice_response_mode"`
	Warning               string `json:"warning,omitempty"`
}

type TwilioWebhookReplayRequest struct {
	WorkspaceID   string            `json:"workspace_id"`
	Kind          string            `json:"kind"`
	BaseURL       string            `json:"base_url"`
	RequestURL    string            `json:"request_url,omitempty"`
	SignatureMode string            `json:"signature_mode"`
	SMSPath       string            `json:"sms_path"`
	VoicePath     string            `json:"voice_path"`
	HTTPTimeoutMS int64             `json:"http_timeout_ms"`
	Params        map[string]string `json:"params"`
}

type TwilioWebhookReplayResponse struct {
	WorkspaceID      string `json:"workspace_id"`
	Kind             string `json:"kind"`
	TargetURL        string `json:"target_url"`
	RequestURL       string `json:"request_url"`
	SignatureMode    string `json:"signature_mode"`
	SignaturePresent bool   `json:"signature_present"`
	StatusCode       int    `json:"status_code"`
	ResponseBody     string `json:"response_body"`
}
