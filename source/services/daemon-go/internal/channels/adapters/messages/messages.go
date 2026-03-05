package messages

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	SourceName              = "apple_messages_chatdb"
	envMessagesChatDBPath   = "PA_MESSAGES_CHAT_DB_PATH"
	envMessagesSendDryRun   = "PA_MESSAGES_SEND_DRY_RUN"
	transportMessagesNative = "messages_apple_events"
	transportMessagesDryRun = "messages_dry_run"
	defaultPollLimit        = 100
	maxPollLimit            = 500
)

type SendRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Destination string `json:"destination"`
	Message     string `json:"message"`
}

type SendResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Destination string `json:"destination"`
	MessageID   string `json:"message_id"`
	Channel     string `json:"channel"`
	Status      string `json:"status"`
	Transport   string `json:"transport"`
}

type StatusRequest struct {
	SourceDBPath string `json:"source_db_path,omitempty"`
	SourceScope  string `json:"source_scope,omitempty"`
}

type StatusResponse struct {
	Ready        bool   `json:"ready"`
	Source       string `json:"source"`
	SourceScope  string `json:"source_scope"`
	SourceDBPath string `json:"source_db_path"`
	Transport    string `json:"transport"`
	Error        string `json:"error,omitempty"`
}

type InboundPollRequest struct {
	WorkspaceID string `json:"workspace_id"`
	SourceDBPath string `json:"source_db_path,omitempty"`
	SourceScope string `json:"source_scope,omitempty"`
	SinceCursor string `json:"since_cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type InboundMessageEvent struct {
	SourceEventID   string `json:"source_event_id"`
	SourceCursor    string `json:"source_cursor"`
	ExternalThreadID string `json:"external_thread_id,omitempty"`
	SenderAddress   string `json:"sender_address,omitempty"`
	LocalAddress    string `json:"local_address,omitempty"`
	BodyText        string `json:"body_text,omitempty"`
	OccurredAt      string `json:"occurred_at"`
}

type InboundPollResponse struct {
	WorkspaceID string               `json:"workspace_id"`
	Source      string               `json:"source"`
	SourceScope string               `json:"source_scope"`
	SourceDBPath string              `json:"source_db_path"`
	CursorStart string               `json:"cursor_start,omitempty"`
	CursorEnd   string               `json:"cursor_end,omitempty"`
	Polled      int                  `json:"polled"`
	Events      []InboundMessageEvent `json:"events"`
}

type commandResult struct {
	output string
	err    error
}

var runCommand = func(ctx context.Context, name string, args ...string) commandResult {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return commandResult{
		output: strings.TrimSpace(string(output)),
		err:    err,
	}
}

var nowUTC = func() time.Time {
	return time.Now().UTC()
}

func Send(ctx context.Context, request SendRequest) (SendResponse, error) {
	destination := strings.TrimSpace(request.Destination)
	if destination == "" {
		return SendResponse{}, fmt.Errorf("messages destination is required")
	}
	message := strings.TrimSpace(request.Message)
	if message == "" {
		return SendResponse{}, fmt.Errorf("messages message is required")
	}

	response := SendResponse{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		Destination: destination,
		MessageID:   newMessageID("imessage"),
		Channel:     "imessage",
		Status:      "sent",
	}
	if isTruthyEnv(os.Getenv(envMessagesSendDryRun)) {
		response.Transport = transportMessagesDryRun
		return response, nil
	}
	if runtime.GOOS != "darwin" {
		return SendResponse{}, fmt.Errorf("messages Apple Events send requires macOS (set %s=1 for dry-run)", envMessagesSendDryRun)
	}

	scriptLines := []string{
		"on run argv",
		"set targetHandle to item 1 of argv",
		"set messageBody to item 2 of argv",
		"tell application \"Messages\"",
		"set targetService to 1st service whose service type = iMessage",
		"set targetBuddy to buddy targetHandle of targetService",
		"send messageBody to targetBuddy",
		"end tell",
		"end run",
	}

	args := make([]string, 0, len(scriptLines)*2+2)
	for _, line := range scriptLines {
		args = append(args, "-e", line)
	}
	args = append(args, destination, message)

	result := runCommand(ctx, "osascript", args...)
	if result.err != nil {
		if result.output != "" {
			return SendResponse{}, fmt.Errorf("messages Apple Events send failed: %s", result.output)
		}
		return SendResponse{}, fmt.Errorf("messages Apple Events send failed: %w", result.err)
	}

	response.Transport = transportMessagesNative
	return response, nil
}

func Status(request StatusRequest) StatusResponse {
	sourcePath := ResolveSourceDBPath(request.SourceDBPath)
	sourceScope := ResolveSourceScope(request.SourceScope, sourcePath)

	response := StatusResponse{
		Ready:        true,
		Source:       SourceName,
		SourceScope:  sourceScope,
		SourceDBPath: sourcePath,
		Transport:    transportMessagesNative,
	}
	if isTruthyEnv(os.Getenv(envMessagesSendDryRun)) {
		response.Transport = transportMessagesDryRun
	}
	if sourcePath == "" {
		response.Ready = false
		response.Error = "messages chat db path could not be resolved"
		return response
	}
	if err := ensureReadableSourceDBPath(sourcePath); err != nil {
		response.Ready = false
		response.Error = err.Error()
	}
	return response
}

func PollInbound(ctx context.Context, request InboundPollRequest) (InboundPollResponse, error) {
	sourcePath := ResolveSourceDBPath(request.SourceDBPath)
	if err := ensureReadableSourceDBPath(sourcePath); err != nil {
		return InboundPollResponse{}, err
	}
	sourceScope := ResolveSourceScope(request.SourceScope, sourcePath)
	limit := normalizePollLimit(request.Limit)
	since := parseCursor(request.SinceCursor)

	db, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		return InboundPollResponse{}, fmt.Errorf("open messages source db: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT
			m.ROWID,
			COALESCE(m.guid, ''),
			COALESCE(c.guid, ''),
			COALESCE(h.id, ''),
			COALESCE(m.text, ''),
			COALESCE(m.date, 0),
			COALESCE(m.is_from_me, 0)
		FROM message m
		LEFT JOIN chat_message_join cmj ON cmj.message_id = m.ROWID
		LEFT JOIN chat c ON c.ROWID = cmj.chat_id
		LEFT JOIN handle h ON h.ROWID = m.handle_id
		WHERE m.ROWID > ?
		  AND COALESCE(m.is_from_me, 0) = 0
		  AND TRIM(COALESCE(m.text, '')) <> ''
		  AND (COALESCE(m.service, '') = 'iMessage' OR COALESCE(m.service, '') = '')
		ORDER BY m.ROWID ASC
		LIMIT ?
	`, since, limit)
	if err != nil {
		if isMessagesDBAccessError(err) {
			return InboundPollResponse{}, fmt.Errorf(
				"query inbound messages: messages source db is not accessible (grant Full Disk Access to Personal Agent Daemon and verify %s): %w",
				sourcePath,
				err,
			)
		}
		return InboundPollResponse{}, fmt.Errorf("query inbound messages: %w", err)
	}
	defer rows.Close()

	events := make([]InboundMessageEvent, 0, limit)
	cursorEnd := strings.TrimSpace(request.SinceCursor)
	for rows.Next() {
		var (
			rowID        int64
			messageGUID  string
			chatGUID     string
			handleID     string
			bodyText     string
			rawDate      int64
			isFromMeFlag int
		)
		if err := rows.Scan(&rowID, &messageGUID, &chatGUID, &handleID, &bodyText, &rawDate, &isFromMeFlag); err != nil {
			return InboundPollResponse{}, fmt.Errorf("scan inbound message row: %w", err)
		}
		if isFromMeFlag != 0 {
			continue
		}

		sourceEventID := strings.TrimSpace(messageGUID)
		if sourceEventID == "" {
			sourceEventID = fmt.Sprintf("message-rowid:%d", rowID)
		}
		eventCursor := strconv.FormatInt(rowID, 10)
		event := InboundMessageEvent{
			SourceEventID:    sourceEventID,
			SourceCursor:     eventCursor,
			ExternalThreadID: strings.TrimSpace(chatGUID),
			SenderAddress:    strings.TrimSpace(handleID),
			BodyText:         strings.TrimSpace(bodyText),
			OccurredAt:       parseMessagesDate(rawDate).Format(time.RFC3339Nano),
		}
		events = append(events, event)
		cursorEnd = eventCursor
	}
	if err := rows.Err(); err != nil {
		return InboundPollResponse{}, fmt.Errorf("iterate inbound message rows: %w", err)
	}

	return InboundPollResponse{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		Source:      SourceName,
		SourceScope: sourceScope,
		SourceDBPath: sourcePath,
		CursorStart: strings.TrimSpace(request.SinceCursor),
		CursorEnd:   strings.TrimSpace(cursorEnd),
		Polled:      len(events),
		Events:      events,
	}, nil
}

func ResolveSourceDBPath(override string) string {
	if path := strings.TrimSpace(override); path != "" {
		if absolute, err := filepath.Abs(path); err == nil {
			return absolute
		}
		return path
	}
	if path := strings.TrimSpace(os.Getenv(envMessagesChatDBPath)); path != "" {
		if absolute, err := filepath.Abs(path); err == nil {
			return absolute
		}
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return ""
	}
	return filepath.Join(homeDir, "Library", "Messages", "chat.db")
}

func ResolveSourceScope(scope string, sourceDBPath string) string {
	if trimmed := strings.TrimSpace(scope); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(sourceDBPath)
}

func ensureReadableSourceDBPath(sourcePath string) error {
	trimmedPath := strings.TrimSpace(sourcePath)
	if trimmedPath == "" {
		return fmt.Errorf("messages source db path is required")
	}
	info, err := os.Stat(trimmedPath)
	if err != nil {
		return fmt.Errorf("messages source db path is not available: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("messages source db path must reference a file: %s", trimmedPath)
	}
	handle, err := os.Open(trimmedPath)
	if err != nil {
		return fmt.Errorf("messages source db path is not readable: %w", err)
	}
	_ = handle.Close()
	return nil
}

func isMessagesDBAccessError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	return strings.Contains(message, "unable to open database file") ||
		strings.Contains(message, "authorization denied") ||
		strings.Contains(message, "operation not permitted") ||
		strings.Contains(message, "out of memory (14)")
}

func normalizePollLimit(limit int) int {
	if limit <= 0 {
		return defaultPollLimit
	}
	if limit > maxPollLimit {
		return maxPollLimit
	}
	return limit
}

func parseCursor(raw string) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseMessagesDate(raw int64) time.Time {
	if raw == 0 {
		return nowUTC()
	}

	appleEpoch := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	candidate := appleEpoch.Add(time.Duration(raw))
	if candidate.Year() >= 2001 && candidate.Year() <= 2100 {
		return candidate.UTC()
	}
	secondsCandidate := time.Unix(raw, 0).UTC()
	if secondsCandidate.Year() >= 2001 && secondsCandidate.Year() <= 2100 {
		return secondsCandidate
	}
	return nowUTC()
}

func newMessageID(prefix string) string {
	tokenBytes := make([]byte, 4)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, nowUTC().UnixNano())
	}
	return fmt.Sprintf("%s-%d-%s", prefix, nowUTC().UnixNano(), hex.EncodeToString(tokenBytes))
}

func isTruthyEnv(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
