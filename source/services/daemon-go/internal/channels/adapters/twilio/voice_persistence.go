package twilio

import (
	"database/sql"
	"time"
)

const (
	channelNameVoice = "voice"
)

type VoicePersistence struct {
	db  *sql.DB
	now func() time.Time
}

type VoiceWebhookInput struct {
	WorkspaceID               string
	ProviderEventID           string
	ProviderCallID            string
	ProviderAccount           string
	SignatureValue            string
	FromAddress               string
	ToAddress                 string
	Direction                 string
	CallStatus                string
	TranscriptText            string
	TranscriptDirection       string
	TranscriptAssistantEmited bool
	ReceivedAt                time.Time
	SignatureValid            bool
	ProviderPayload           map[string]string
}

type VoiceWebhookResult struct {
	ReceiptID         string
	CallSessionID     string
	ThreadID          string
	CallStatus        string
	StatusEventID     string
	TranscriptEventID string
	Replayed          bool
}

type OutboundCallInput struct {
	WorkspaceID     string
	ProviderCallID  string
	ProviderAccount string
	FromAddress     string
	ToAddress       string
	Direction       string
	CallStatus      string
	OccurredAt      time.Time
	ProviderPayload map[string]any
}

type OutboundCallResult struct {
	CallSessionID string
	ThreadID      string
	CallStatus    string
	StatusEventID string
}

func NewVoicePersistence(db *sql.DB) *VoicePersistence {
	return &VoicePersistence{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}
