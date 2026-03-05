package twilio

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupSMSPersistenceDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "twilio-sms-persistence.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := migrator.Apply(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func TestPersistInboundSMSCreatesCommEventAndIsReplaySafe(t *testing.T) {
	db := setupSMSPersistenceDB(t)
	persistence := NewSMSPersistence(db)

	input := InboundSMSInput{
		WorkspaceID:     "ws_twilio",
		ProviderEventID: "SMINBOUND1",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-1",
		FromAddress:     "+15551110000",
		ToAddress:       "+15552220000",
		BodyText:        "Hello inbound",
		SignatureValid:  true,
		ProviderStatus:  "received",
		ProviderPayload: map[string]string{
			"From":       "+15551110000",
			"To":         "+15552220000",
			"Body":       "Hello inbound",
			"MessageSid": "SMINBOUND1",
		},
	}

	first, err := persistence.PersistInboundSMS(context.Background(), input)
	if err != nil {
		t.Fatalf("persist inbound sms: %v", err)
	}
	if first.Replayed {
		t.Fatalf("expected first insert to not be replay")
	}
	if first.EventID == "" || first.ThreadID == "" || first.ReceiptID == "" {
		t.Fatalf("expected receipt/event/thread ids to be set, got %+v", first)
	}

	second, err := persistence.PersistInboundSMS(context.Background(), input)
	if err != nil {
		t.Fatalf("persist inbound sms replay: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replay on second persist")
	}
	if second.EventID != first.EventID {
		t.Fatalf("expected replay event id %s, got %s", first.EventID, second.EventID)
	}

	assertCount(t, db, "comm_events", 1)
	assertCount(t, db, "comm_webhook_receipts", 1)
	assertCount(t, db, "comm_provider_messages", 1)

	var (
		threadChannel   string
		threadConnector string
	)
	if err := db.QueryRow(`SELECT channel, connector_id FROM comm_threads WHERE id = ?`, first.ThreadID).Scan(&threadChannel, &threadConnector); err != nil {
		t.Fatalf("query inbound thread channel/connector: %v", err)
	}
	if threadChannel != "message" {
		t.Fatalf("expected inbound twilio sms thread channel=message, got %s", threadChannel)
	}
	if threadConnector != "twilio" {
		t.Fatalf("expected inbound twilio sms thread connector_id=twilio, got %s", threadConnector)
	}
	var eventConnector string
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, first.EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query inbound event connector: %v", err)
	}
	if eventConnector != "twilio" {
		t.Fatalf("expected inbound twilio sms event connector_id=twilio, got %s", eventConnector)
	}
}

func TestPersistOutboundSMSMaintainsThreadContinuity(t *testing.T) {
	db := setupSMSPersistenceDB(t)
	persistence := NewSMSPersistence(db)

	inbound, err := persistence.PersistInboundSMS(context.Background(), InboundSMSInput{
		WorkspaceID:     "ws_twilio",
		ProviderEventID: "SMINBOUND1",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-1",
		FromAddress:     "+15551110000",
		ToAddress:       "+15552220000",
		BodyText:        "Hello inbound",
		SignatureValid:  true,
		ProviderStatus:  "received",
		ProviderPayload: map[string]string{
			"From":       "+15551110000",
			"To":         "+15552220000",
			"Body":       "Hello inbound",
			"MessageSid": "SMINBOUND1",
		},
	})
	if err != nil {
		t.Fatalf("persist inbound sms: %v", err)
	}

	outbound, err := persistence.PersistOutboundSMS(context.Background(), OutboundSMSInput{
		WorkspaceID:     "ws_twilio",
		ProviderMessage: "SMOUTBOUND1",
		ProviderAccount: "AC123",
		FromAddress:     "+15552220000",
		ToAddress:       "+15551110000",
		BodyText:        "Hello outbound",
		ProviderStatus:  "queued",
		ProviderPayload: map[string]any{
			"sid":         "SMOUTBOUND1",
			"account_sid": "AC123",
			"status":      "queued",
		},
	})
	if err != nil {
		t.Fatalf("persist outbound sms: %v", err)
	}

	if outbound.ThreadID != inbound.ThreadID {
		t.Fatalf("expected outbound thread %s to match inbound thread %s", outbound.ThreadID, inbound.ThreadID)
	}
	if outbound.EventID == "" {
		t.Fatalf("expected outbound event id")
	}

	assertCount(t, db, "comm_events", 2)
	assertCount(t, db, "comm_provider_messages", 2)

	var (
		direction string
		assistant int
	)
	if err := db.QueryRow(`SELECT direction, assistant_emitted FROM comm_events WHERE id = ?`, outbound.EventID).Scan(&direction, &assistant); err != nil {
		t.Fatalf("query outbound event direction: %v", err)
	}
	if direction != "OUTBOUND" || assistant != 1 {
		t.Fatalf("expected outbound assistant event, got direction=%s assistant=%d", direction, assistant)
	}

	var (
		threadChannel   string
		threadConnector string
	)
	if err := db.QueryRow(`SELECT channel, connector_id FROM comm_threads WHERE id = ?`, outbound.ThreadID).Scan(&threadChannel, &threadConnector); err != nil {
		t.Fatalf("query outbound thread channel/connector: %v", err)
	}
	if threadChannel != "message" {
		t.Fatalf("expected outbound twilio sms thread channel=message, got %s", threadChannel)
	}
	if threadConnector != "twilio" {
		t.Fatalf("expected outbound twilio sms thread connector_id=twilio, got %s", threadConnector)
	}
	var eventConnector string
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, outbound.EventID).Scan(&eventConnector); err != nil {
		t.Fatalf("query outbound event connector: %v", err)
	}
	if eventConnector != "twilio" {
		t.Fatalf("expected outbound twilio sms event connector_id=twilio, got %s", eventConnector)
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	query := "SELECT COUNT(*) FROM " + table
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("count table %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, count)
	}
}
