package twilio

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func setupVoicePersistenceDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "twilio-voice-persistence.db")
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

func TestVoicePersistenceLifecycleAndReplay(t *testing.T) {
	db := setupVoicePersistenceDB(t)
	persistence := NewVoicePersistence(db)
	ctx := context.Background()

	outbound, err := persistence.PersistOutboundCall(ctx, OutboundCallInput{
		WorkspaceID:     "ws_voice",
		ProviderCallID:  "CA123",
		ProviderAccount: "AC123",
		FromAddress:     "+15550001111",
		ToAddress:       "+15550002222",
		Direction:       "outbound-api",
		CallStatus:      "queued",
		ProviderPayload: map[string]any{"sid": "CA123", "status": "queued"},
	})
	if err != nil {
		t.Fatalf("persist outbound call: %v", err)
	}
	if outbound.CallStatus != "initiated" {
		t.Fatalf("expected initiated status, got %s", outbound.CallStatus)
	}

	ringing, err := persistence.PersistInboundWebhook(ctx, VoiceWebhookInput{
		WorkspaceID:     "ws_voice",
		ProviderEventID: "voice-callback-1",
		ProviderCallID:  "CA123",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-1",
		FromAddress:     "+15550001111",
		ToAddress:       "+15550002222",
		Direction:       "outbound-api",
		CallStatus:      "ringing",
		SignatureValid:  true,
		ProviderPayload: map[string]string{
			"CallSid":    "CA123",
			"CallStatus": "ringing",
		},
	})
	if err != nil {
		t.Fatalf("persist ringing callback: %v", err)
	}
	if ringing.CallStatus != "ringing" {
		t.Fatalf("expected ringing status, got %s", ringing.CallStatus)
	}

	inProgress, err := persistence.PersistInboundWebhook(ctx, VoiceWebhookInput{
		WorkspaceID:               "ws_voice",
		ProviderEventID:           "voice-callback-2",
		ProviderCallID:            "CA123",
		ProviderAccount:           "AC123",
		SignatureValue:            "sig-2",
		FromAddress:               "+15550001111",
		ToAddress:                 "+15550002222",
		Direction:                 "outbound-api",
		CallStatus:                "in-progress",
		TranscriptText:            "Thanks for calling",
		TranscriptDirection:       "INBOUND",
		TranscriptAssistantEmited: false,
		SignatureValid:            true,
		ProviderPayload: map[string]string{
			"CallSid":      "CA123",
			"CallStatus":   "in-progress",
			"SpeechResult": "Thanks for calling",
		},
	})
	if err != nil {
		t.Fatalf("persist in-progress callback: %v", err)
	}
	if inProgress.CallStatus != "in_progress" {
		t.Fatalf("expected in_progress status, got %s", inProgress.CallStatus)
	}
	if inProgress.TranscriptEventID == "" {
		t.Fatalf("expected transcript event id")
	}

	completed, err := persistence.PersistInboundWebhook(ctx, VoiceWebhookInput{
		WorkspaceID:     "ws_voice",
		ProviderEventID: "voice-callback-3",
		ProviderCallID:  "CA123",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-3",
		FromAddress:     "+15550001111",
		ToAddress:       "+15550002222",
		Direction:       "outbound-api",
		CallStatus:      "completed",
		SignatureValid:  true,
		ProviderPayload: map[string]string{
			"CallSid":    "CA123",
			"CallStatus": "completed",
		},
	})
	if err != nil {
		t.Fatalf("persist completed callback: %v", err)
	}
	if completed.CallStatus != "completed" {
		t.Fatalf("expected completed status, got %s", completed.CallStatus)
	}

	regression, err := persistence.PersistInboundWebhook(ctx, VoiceWebhookInput{
		WorkspaceID:     "ws_voice",
		ProviderEventID: "voice-callback-4",
		ProviderCallID:  "CA123",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-4",
		FromAddress:     "+15550001111",
		ToAddress:       "+15550002222",
		Direction:       "outbound-api",
		CallStatus:      "ringing",
		SignatureValid:  true,
		ProviderPayload: map[string]string{
			"CallSid":    "CA123",
			"CallStatus": "ringing",
		},
	})
	if err != nil {
		t.Fatalf("persist regression callback: %v", err)
	}
	if regression.CallStatus != "completed" {
		t.Fatalf("expected terminal status to remain completed, got %s", regression.CallStatus)
	}

	replay, err := persistence.PersistInboundWebhook(ctx, VoiceWebhookInput{
		WorkspaceID:     "ws_voice",
		ProviderEventID: "voice-callback-4",
		ProviderCallID:  "CA123",
		ProviderAccount: "AC123",
		SignatureValue:  "sig-4",
		FromAddress:     "+15550001111",
		ToAddress:       "+15550002222",
		Direction:       "outbound-api",
		CallStatus:      "ringing",
		SignatureValid:  true,
		ProviderPayload: map[string]string{
			"CallSid":    "CA123",
			"CallStatus": "ringing",
		},
	})
	if err != nil {
		t.Fatalf("persist replay callback: %v", err)
	}
	if !replay.Replayed {
		t.Fatalf("expected replay=true for duplicate provider event id")
	}

	assertVoiceCount(t, db, "comm_call_sessions", 1)
	assertVoiceCount(t, db, "comm_webhook_receipts", 4)

	var transcriptCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comm_events WHERE event_type = 'VOICE_TRANSCRIPT'`).Scan(&transcriptCount); err != nil {
		t.Fatalf("count voice transcript events: %v", err)
	}
	if transcriptCount != 1 {
		t.Fatalf("expected 1 voice transcript event, got %d", transcriptCount)
	}

	var finalStatus string
	if err := db.QueryRow(`SELECT status FROM comm_call_sessions WHERE workspace_id = 'ws_voice' AND provider_call_id = 'CA123'`).Scan(&finalStatus); err != nil {
		t.Fatalf("query final call status: %v", err)
	}
	if finalStatus != "completed" {
		t.Fatalf("expected final session status completed, got %s", finalStatus)
	}

	var (
		threadConnector string
		callConnector   string
		statusConnector string
		transcriptConn  string
	)
	if err := db.QueryRow(`SELECT connector_id FROM comm_threads WHERE id = ?`, outbound.ThreadID).Scan(&threadConnector); err != nil {
		t.Fatalf("query voice thread connector: %v", err)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_call_sessions WHERE id = ?`, outbound.CallSessionID).Scan(&callConnector); err != nil {
		t.Fatalf("query voice call session connector: %v", err)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, outbound.StatusEventID).Scan(&statusConnector); err != nil {
		t.Fatalf("query voice status event connector: %v", err)
	}
	if err := db.QueryRow(`SELECT connector_id FROM comm_events WHERE id = ?`, inProgress.TranscriptEventID).Scan(&transcriptConn); err != nil {
		t.Fatalf("query voice transcript event connector: %v", err)
	}
	if threadConnector != "twilio" || callConnector != "twilio" || statusConnector != "twilio" || transcriptConn != "twilio" {
		t.Fatalf("expected twilio connector attribution, got thread=%s call=%s status=%s transcript=%s", threadConnector, callConnector, statusConnector, transcriptConn)
	}
}

func assertVoiceCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	query := "SELECT COUNT(*) FROM " + table
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, count)
	}
}
