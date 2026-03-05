package types

import "time"

type DeliveryAttemptStatus string

const (
	DeliveryAttemptPending DeliveryAttemptStatus = "pending"
	DeliveryAttemptSent    DeliveryAttemptStatus = "sent"
	DeliveryAttemptFailed  DeliveryAttemptStatus = "failed"
)

type ChannelDeliveryPolicy struct {
	PrimaryChannel   string   `json:"primary_channel"`
	RetryCount       int      `json:"retry_count"`
	FallbackChannels []string `json:"fallback_channels"`
}

type DeliveryRequest struct {
	WorkspaceID         string
	OperationID         string
	StepID              string
	EventID             string
	SourceChannel       string
	DestinationEndpoint string
	MessageBody         string
}

type DeliveryAttemptRecord struct {
	AttemptID           string
	WorkspaceID         string
	StepID              string
	EventID             string
	DestinationEndpoint string
	IdempotencyKey      string
	Channel             string
	Status              DeliveryAttemptStatus
	ProviderReceipt     string
	ErrorText           string
	AttemptedAt         time.Time
}

type DeliveryResult struct {
	Delivered        bool
	Channel          string
	ProviderReceipt  string
	IdempotentReplay bool
	Attempts         []DeliveryAttemptRecord
}
