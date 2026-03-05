package twilio

import (
	"database/sql"
	"time"
)

const (
	providerNameTwilio = "twilio"
	channelNameSMS     = "sms"
	connectorIDTwilio  = "twilio"
)

type SMSPersistence struct {
	db  *sql.DB
	now func() time.Time
}

type InboundSMSInput struct {
	WorkspaceID      string
	ProviderEventID  string
	ProviderAccount  string
	SignatureValue   string
	FromAddress      string
	ToAddress        string
	BodyText         string
	ReceivedAt       time.Time
	SignatureValid   bool
	ProviderStatus   string
	ProviderPayload  map[string]string
	ProviderPayloadS string
}

type InboundSMSResult struct {
	ReceiptID string
	EventID   string
	ThreadID  string
	Replayed  bool
}

type OutboundSMSInput struct {
	WorkspaceID     string
	ProviderMessage string
	ProviderAccount string
	FromAddress     string
	ToAddress       string
	BodyText        string
	OccurredAt      time.Time
	ProviderStatus  string
	ProviderPayload map[string]any
}

type OutboundSMSResult struct {
	EventID  string
	ThreadID string
}

func NewSMSPersistence(db *sql.DB) *SMSPersistence {
	return &SMSPersistence{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}
