package messages

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSendDryRunReturnsSentResponse(t *testing.T) {
	t.Setenv(envMessagesSendDryRun, "1")

	response, err := Send(context.Background(), SendRequest{
		WorkspaceID: "ws1",
		Destination: "+15555550123",
		Message:     "hello from dry run",
	})
	if err != nil {
		t.Fatalf("send dry run: %v", err)
	}
	if response.Status != "sent" {
		t.Fatalf("expected sent status, got %s", response.Status)
	}
	if response.Channel != "imessage" {
		t.Fatalf("expected imessage channel, got %s", response.Channel)
	}
	if response.Transport != transportMessagesDryRun {
		t.Fatalf("expected dry-run transport, got %s", response.Transport)
	}
	if strings.TrimSpace(response.MessageID) == "" {
		t.Fatalf("expected message id")
	}
}

func TestSendRequiresMacOSWhenNotDryRun(t *testing.T) {
	t.Setenv(envMessagesSendDryRun, "")
	if runtime.GOOS == "darwin" {
		t.Skip("macOS runtime may have Apple Events available; non-macOS guard is not applicable")
	}

	_, err := Send(context.Background(), SendRequest{
		WorkspaceID: "ws1",
		Destination: "+15555550123",
		Message:     "hello",
	})
	if err == nil {
		t.Fatalf("expected macOS requirement error")
	}
	if !strings.Contains(err.Error(), "requires macOS") {
		t.Fatalf("expected macOS requirement error, got %v", err)
	}
}

func TestPollInboundReturnsOnlyInboundIMessageRows(t *testing.T) {
	chatDBPath := filepath.Join(t.TempDir(), "chat.db")
	createMessagesFixtureDB(t, chatDBPath)

	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	originalNow := nowUTC
	nowUTC = func() time.Time { return now }
	t.Cleanup(func() {
		nowUTC = originalNow
	})

	response, err := PollInbound(context.Background(), InboundPollRequest{
		WorkspaceID: "ws1",
		SourceDBPath: chatDBPath,
		SinceCursor: "0",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("poll inbound: %v", err)
	}
	if response.Source != SourceName {
		t.Fatalf("expected source %s, got %s", SourceName, response.Source)
	}
	if response.SourceDBPath != chatDBPath {
		t.Fatalf("expected source db path %s, got %s", chatDBPath, response.SourceDBPath)
	}
	if response.Polled != 2 {
		t.Fatalf("expected 2 inbound events, got %d", response.Polled)
	}
	if response.CursorEnd != "14" {
		t.Fatalf("expected cursor_end=14, got %s", response.CursorEnd)
	}

	first := response.Events[0]
	if first.SourceEventID != "imessage-guid-11" {
		t.Fatalf("unexpected first source_event_id: %s", first.SourceEventID)
	}
	if first.SourceCursor != "11" {
		t.Fatalf("unexpected first source_cursor: %s", first.SourceCursor)
	}
	if first.ExternalThreadID != "chat-guid-1" {
		t.Fatalf("unexpected first thread guid: %s", first.ExternalThreadID)
	}
	if first.SenderAddress != "+15555550100" {
		t.Fatalf("unexpected first sender: %s", first.SenderAddress)
	}
	if first.BodyText != "hello inbound" {
		t.Fatalf("unexpected first body: %s", first.BodyText)
	}

	second := response.Events[1]
	if second.SourceEventID != "imessage-guid-14" {
		t.Fatalf("unexpected second source_event_id: %s", second.SourceEventID)
	}
	if second.SourceCursor != "14" {
		t.Fatalf("unexpected second source_cursor: %s", second.SourceCursor)
	}
	if second.SenderAddress != "+15555550101" {
		t.Fatalf("unexpected second sender: %s", second.SenderAddress)
	}
}

func TestPollInboundHonorsCursor(t *testing.T) {
	chatDBPath := filepath.Join(t.TempDir(), "chat.db")
	createMessagesFixtureDB(t, chatDBPath)

	response, err := PollInbound(context.Background(), InboundPollRequest{
		WorkspaceID: "ws1",
		SourceDBPath: chatDBPath,
		SinceCursor: "11",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("poll inbound with cursor: %v", err)
	}
	if response.Polled != 1 {
		t.Fatalf("expected 1 inbound event after cursor, got %d", response.Polled)
	}
	if len(response.Events) != 1 || response.Events[0].SourceCursor != "14" {
		t.Fatalf("unexpected events after cursor: %+v", response.Events)
	}
}

func TestPollInboundReturnsReadablePathErrorWhenSourceMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-chat.db")
	_, err := PollInbound(context.Background(), InboundPollRequest{
		WorkspaceID: "ws1",
		SourceDBPath: missingPath,
	})
	if err == nil {
		t.Fatalf("expected missing source db path error")
	}
	if !strings.Contains(err.Error(), "messages source db path is not available") {
		t.Fatalf("expected missing source path error, got %v", err)
	}
}

func TestPollInboundRejectsDirectorySourceDBPath(t *testing.T) {
	dirPath := t.TempDir()
	_, err := PollInbound(context.Background(), InboundPollRequest{
		WorkspaceID: "ws1",
		SourceDBPath: dirPath,
	})
	if err == nil {
		t.Fatalf("expected directory source db path error")
	}
	if !strings.Contains(err.Error(), "must reference a file") {
		t.Fatalf("expected file path validation error, got %v", err)
	}
}

func TestStatusReportsReadablePathErrorWhenSourceMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-chat.db")
	status := Status(StatusRequest{SourceDBPath: missingPath})
	if status.Ready {
		t.Fatalf("expected status to be not ready when source path is missing")
	}
	if !strings.Contains(status.Error, "messages source db path is not available") {
		t.Fatalf("expected missing source path status error, got %s", status.Error)
	}
}

func createMessagesFixtureDB(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	statements := []string{
		`CREATE TABLE message (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT,
			text TEXT,
			date INTEGER,
			is_from_me INTEGER,
			handle_id INTEGER,
			service TEXT
		);`,
		`CREATE TABLE chat (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT
		);`,
		`CREATE TABLE chat_message_join (
			chat_id INTEGER,
			message_id INTEGER
		);`,
		`CREATE TABLE handle (
			ROWID INTEGER PRIMARY KEY,
			id TEXT
		);`,
		`INSERT INTO handle(ROWID, id) VALUES (1, '+15555550100');`,
		`INSERT INTO handle(ROWID, id) VALUES (2, '+15555550101');`,
		`INSERT INTO chat(ROWID, guid) VALUES (1, 'chat-guid-1');`,
		`INSERT INTO chat(ROWID, guid) VALUES (2, 'chat-guid-2');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (11, 'imessage-guid-11', 'hello inbound', 1000000000, 0, 1, 'iMessage');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (12, 'imessage-guid-12', 'outbound', 1000000001, 1, 1, 'iMessage');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (13, 'imessage-guid-13', '', 1000000002, 0, 1, 'iMessage');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (14, 'imessage-guid-14', 'second inbound', 1000000003, 0, 2, 'iMessage');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (15, 'sms-guid-15', 'sms inbound', 1000000004, 0, 2, 'SMS');`,
		`INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 11);`,
		`INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 12);`,
		`INSERT INTO chat_message_join(chat_id, message_id) VALUES (2, 14);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec fixture statement: %v", err)
		}
	}
}
