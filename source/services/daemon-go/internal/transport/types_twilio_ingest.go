package transport

type TwilioIngestSMSRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	RequestURL    string `json:"request_url,omitempty"`
	Signature     string `json:"signature,omitempty"`
	SkipSignature bool   `json:"skip_signature"`
	FromAddress   string `json:"from_address"`
	ToAddress     string `json:"to_address"`
	BodyText      string `json:"body_text,omitempty"`
	MessageSID    string `json:"message_sid"`
	AccountSID    string `json:"account_sid,omitempty"`
}

type TwilioIngestSMSResponse struct {
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

type TwilioIngestVoiceRequest struct {
	WorkspaceID                string `json:"workspace_id"`
	RequestURL                 string `json:"request_url,omitempty"`
	Signature                  string `json:"signature,omitempty"`
	SkipSignature              bool   `json:"skip_signature"`
	ProviderEventID            string `json:"provider_event_id,omitempty"`
	CallSID                    string `json:"call_sid"`
	AccountSID                 string `json:"account_sid,omitempty"`
	FromAddress                string `json:"from_address"`
	ToAddress                  string `json:"to_address"`
	Direction                  string `json:"direction,omitempty"`
	CallStatus                 string `json:"call_status,omitempty"`
	Transcript                 string `json:"transcript,omitempty"`
	TranscriptDirection        string `json:"transcript_direction,omitempty"`
	TranscriptAssistantEmitted bool   `json:"transcript_assistant_emitted,omitempty"`
}

type TwilioIngestVoiceResponse struct {
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
