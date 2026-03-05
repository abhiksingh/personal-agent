package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

type inboundWatcherCommStub struct {
	mu sync.Mutex

	messagesCalls    int
	mailCalls        int
	calendarCalls    int
	browserCalls     int
	messagesRequests []transport.MessagesIngestRequest
	mailRequests     []transport.MailRuleIngestRequest
	calendarRequests []transport.CalendarChangeIngestRequest
	browserRequests  []transport.BrowserEventIngestRequest

	messagesResponse transport.MessagesIngestResponse
	messagesErr      error
	mailResponse     transport.MailRuleIngestResponse
	mailErr          error
	calendarResponse transport.CalendarChangeIngestResponse
	calendarErr      error
	browserResponse  transport.BrowserEventIngestResponse
	browserErr       error
}

func (s *inboundWatcherCommStub) IngestMessages(_ context.Context, request transport.MessagesIngestRequest) (transport.MessagesIngestResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messagesCalls++
	s.messagesRequests = append(s.messagesRequests, request)
	return s.messagesResponse, s.messagesErr
}

func (s *inboundWatcherCommStub) IngestMailRuleEvent(_ context.Context, request transport.MailRuleIngestRequest) (transport.MailRuleIngestResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mailCalls++
	s.mailRequests = append(s.mailRequests, request)
	return s.mailResponse, s.mailErr
}

func (s *inboundWatcherCommStub) IngestCalendarChange(_ context.Context, request transport.CalendarChangeIngestRequest) (transport.CalendarChangeIngestResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calendarCalls++
	s.calendarRequests = append(s.calendarRequests, request)
	return s.calendarResponse, s.calendarErr
}

func (s *inboundWatcherCommStub) IngestBrowserEvent(_ context.Context, request transport.BrowserEventIngestRequest) (transport.BrowserEventIngestResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.browserCalls++
	s.browserRequests = append(s.browserRequests, request)
	return s.browserResponse, s.browserErr
}

func (s *inboundWatcherCommStub) snapshotCounts() (int, int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messagesCalls, s.mailCalls, s.calendarCalls, s.browserCalls
}

func TestInboundWatcherRuntimeStartStopPollsMessages(t *testing.T) {
	db := newInboundWatcherTestDB(t)
	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-watch",
		},
	}

	runtime, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		PollInterval: 5 * time.Millisecond,
		WorkspaceID:  "ws-watch",
		InboxDir:     filepath.Join(t.TempDir(), "inbox"),
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start inbound watcher runtime: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Stop(context.Background())
	})

	waitForInboundWatcherCondition(t, 2*time.Second, func() bool {
		messagesCalls, _, _, _ := stub.snapshotCounts()
		return messagesCalls > 0
	})

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop inbound watcher runtime: %v", err)
	}
}

func TestInboundWatcherRuntimeUsesWorkspaceResolverWhenWorkspaceNotPinned(t *testing.T) {
	db := newInboundWatcherTestDB(t)
	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-resolved",
		},
	}

	resolverCalls := 0
	runtime, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		InboxDir: filepath.Join(t.TempDir(), "inbox"),
		ResolveWorkspaceID: func(_ context.Context) (string, error) {
			resolverCalls++
			return "ws-resolved", nil
		},
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, err := runtime.pollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if processed {
		t.Fatalf("expected no work when no events/files were ingested")
	}
	if resolverCalls == 0 {
		t.Fatalf("expected workspace resolver to be called at least once")
	}

	if len(stub.messagesRequests) != 1 {
		t.Fatalf("expected one messages ingest request, got %d", len(stub.messagesRequests))
	}
	if got := normalizeWorkspaceID(stub.messagesRequests[0].WorkspaceID); got != "ws-resolved" {
		t.Fatalf("expected resolved workspace ws-resolved, got %q", got)
	}
}

func TestInboundWatcherRuntimeWorkspaceEnvOverrideBeatsResolver(t *testing.T) {
	t.Setenv(envInboundWatcherWorkspaceID, "ws-env")
	db := newInboundWatcherTestDB(t)
	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-env",
		},
	}

	resolverCalls := 0
	runtime, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		InboxDir: filepath.Join(t.TempDir(), "inbox"),
		ResolveWorkspaceID: func(_ context.Context) (string, error) {
			resolverCalls++
			return "ws-resolved", nil
		},
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, err := runtime.pollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if processed {
		t.Fatalf("expected no work when no events/files were ingested")
	}
	if resolverCalls != 0 {
		t.Fatalf("expected resolver not to be called when env override pins workspace, got %d calls", resolverCalls)
	}
	if len(stub.messagesRequests) != 1 {
		t.Fatalf("expected one messages ingest request, got %d", len(stub.messagesRequests))
	}
	if got := normalizeWorkspaceID(stub.messagesRequests[0].WorkspaceID); got != "ws-env" {
		t.Fatalf("expected env-pinned workspace ws-env, got %q", got)
	}
}

func TestInboundWatcherRuntimePollsMailCalendarBrowserFileAdapters(t *testing.T) {
	db := newInboundWatcherTestDB(t)
	inboxDir := filepath.Join(t.TempDir(), "inbox")

	writeInboundWatcherJSONFile(t, filepath.Join(inboxDir, "mail", "pending", "001-mail.json"), transport.MailRuleIngestRequest{
		SourceScope:   "mailbox://watch",
		SourceEventID: "mail-file-1",
		FromAddress:   "sender@example.com",
		OccurredAt:    "2026-02-24T10:00:00Z",
	})
	writeInboundWatcherJSONFile(t, filepath.Join(inboxDir, "calendar", "pending", "001-calendar.json"), transport.CalendarChangeIngestRequest{
		SourceScope:   "calendar://watch",
		SourceEventID: "calendar-file-1",
		EventUID:      "calendar-uid-1",
		OccurredAt:    "2026-02-24T10:01:00Z",
	})
	writeInboundWatcherJSONFile(t, filepath.Join(inboxDir, "browser", "pending", "001-browser.json"), transport.BrowserEventIngestRequest{
		SourceScope:   "safari://window/watch",
		SourceEventID: "browser-file-1",
		EventType:     "navigation",
		PageURL:       "https://example.com",
		OccurredAt:    "2026-02-24T10:02:00Z",
	})

	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-watch",
		},
		mailResponse: transport.MailRuleIngestResponse{
			WorkspaceID: "ws-watch",
			SourceScope: "mailbox://watch",
			Accepted:    true,
		},
		calendarResponse: transport.CalendarChangeIngestResponse{
			WorkspaceID: "ws-watch",
			SourceScope: "calendar://watch",
			Accepted:    true,
		},
		browserResponse: transport.BrowserEventIngestResponse{
			WorkspaceID: "ws-watch",
			SourceScope: "safari://window/watch",
			Accepted:    true,
		},
	}

	runtime, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		WorkspaceID: "ws-watch",
		InboxDir:    inboxDir,
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, err := runtime.pollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if !processed {
		t.Fatalf("expected poll to process watcher files")
	}

	messagesCalls, mailCalls, calendarCalls, browserCalls := stub.snapshotCounts()
	if messagesCalls != 1 || mailCalls != 1 || calendarCalls != 1 || browserCalls != 1 {
		t.Fatalf("unexpected ingest call counts messages=%d mail=%d calendar=%d browser=%d", messagesCalls, mailCalls, calendarCalls, browserCalls)
	}

	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "mail", "pending")); count != 0 {
		t.Fatalf("expected mail pending dir to be empty, got %d file(s)", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "calendar", "pending")); count != 0 {
		t.Fatalf("expected calendar pending dir to be empty, got %d file(s)", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "browser", "pending")); count != 0 {
		t.Fatalf("expected browser pending dir to be empty, got %d file(s)", count)
	}

	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "mail", "processed")); count != 1 {
		t.Fatalf("expected one processed mail file, got %d", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "calendar", "processed")); count != 1 {
		t.Fatalf("expected one processed calendar file, got %d", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "browser", "processed")); count != 1 {
		t.Fatalf("expected one processed browser file, got %d", count)
	}

	if len(stub.mailRequests) != 1 || normalizeWorkspaceID(stub.mailRequests[0].WorkspaceID) != "ws-watch" {
		t.Fatalf("expected mail watcher payload to default workspace ws-watch, got %+v", stub.mailRequests)
	}
	if len(stub.calendarRequests) != 1 || normalizeWorkspaceID(stub.calendarRequests[0].WorkspaceID) != "ws-watch" {
		t.Fatalf("expected calendar watcher payload to default workspace ws-watch, got %+v", stub.calendarRequests)
	}
	if len(stub.browserRequests) != 1 || normalizeWorkspaceID(stub.browserRequests[0].WorkspaceID) != "ws-watch" {
		t.Fatalf("expected browser watcher payload to default workspace ws-watch, got %+v", stub.browserRequests)
	}
}

func TestInboundWatcherRuntimeMailFileReplaySafe(t *testing.T) {
	db := newInboundWatcherTestDB(t)
	inboxDir := filepath.Join(t.TempDir(), "inbox")

	dispatch := &stubMessagesPollDispatcher{
		pollResponse: messagesadapter.InboundPollResponse{
			WorkspaceID:  "ws-watch",
			Source:       messagesadapter.SourceName,
			SourceScope:  "scope://messages",
			SourceDBPath: "/tmp/messages.db",
		},
	}
	service := &CommTwilioService{
		container:       &ServiceContainer{DB: db},
		channelDispatch: dispatch,
	}

	writeInboundWatcherJSONFile(t, filepath.Join(inboxDir, "mail", "pending", "001-mail.json"), transport.MailRuleIngestRequest{
		WorkspaceID:   "ws-watch",
		SourceScope:   "mailbox://watch",
		SourceEventID: "mail-replay-1",
		MessageID:     "<mail-replay-1@example.com>",
		FromAddress:   "sender@example.com",
		ToAddress:     "recipient@example.com",
		Subject:       "watcher replay",
		BodyText:      "watcher replay body",
		OccurredAt:    "2026-02-24T11:00:00Z",
	})
	writeInboundWatcherJSONFile(t, filepath.Join(inboxDir, "mail", "pending", "002-mail.json"), transport.MailRuleIngestRequest{
		WorkspaceID:   "ws-watch",
		SourceScope:   "mailbox://watch",
		SourceEventID: "mail-replay-1",
		MessageID:     "<mail-replay-1@example.com>",
		FromAddress:   "sender@example.com",
		ToAddress:     "recipient@example.com",
		Subject:       "watcher replay",
		BodyText:      "watcher replay body",
		OccurredAt:    "2026-02-24T11:00:00Z",
	})

	runtime, err := NewInboundWatcherRuntime(db, service, InboundWatcherRuntimeOptions{
		WorkspaceID: "ws-watch",
		InboxDir:    inboxDir,
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, err := runtime.pollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if !processed {
		t.Fatalf("expected poll to process watcher files")
	}

	if count := inboundWatcherCount(t, db, "comm_events"); count != 1 {
		t.Fatalf("expected one mail comm_event after replay-safe ingest, got %d", count)
	}
	if count := inboundWatcherCount(t, db, "comm_ingest_receipts"); count != 1 {
		t.Fatalf("expected one ingest receipt after replay-safe ingest, got %d", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "mail", "processed")); count != 2 {
		t.Fatalf("expected both mail files archived to processed, got %d", count)
	}
}

func TestInboundWatcherRuntimeInvalidPayloadMovesToFailedAndTracksError(t *testing.T) {
	db := newInboundWatcherTestDB(t)
	inboxDir := filepath.Join(t.TempDir(), "inbox")

	writeInboundWatcherRawFile(t, filepath.Join(inboxDir, "browser", "pending", "001-browser.json"), "{invalid json")

	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-watch",
		},
	}
	runtime, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		WorkspaceID: "ws-watch",
		InboxDir:    inboxDir,
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, err := runtime.pollOnce(context.Background())
	if !processed {
		t.Fatalf("expected poll to process invalid browser payload file")
	}
	if err == nil {
		t.Fatalf("expected poll error for invalid browser payload")
	}

	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "browser", "pending")); count != 0 {
		t.Fatalf("expected browser pending dir to be empty, got %d file(s)", count)
	}
	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "browser", "failed")); count != 1 {
		t.Fatalf("expected invalid browser payload moved to failed dir, got %d file(s)", count)
	}

	var lastError string
	if err := db.QueryRow(
		`SELECT COALESCE(last_error, '') FROM automation_source_subscriptions WHERE workspace_id = ? AND source = ? AND source_scope = ? LIMIT 1`,
		"ws-watch",
		browserEventIngestSource,
		resolveBrowserSourceScope("", ""),
	).Scan(&lastError); err != nil {
		t.Fatalf("query browser source-subscription error: %v", err)
	}
	if !strings.Contains(strings.ToLower(lastError), "decode browser watcher payload") {
		t.Fatalf("expected browser watcher decode error in source subscription, got %q", lastError)
	}
}

func TestInboundWatcherRuntimeSymlinkPayloadIsQuarantinedWithoutTouchingTargetPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior is permission-dependent on windows")
	}

	db := newInboundWatcherTestDB(t)
	inboxDir := filepath.Join(t.TempDir(), "inbox")
	targetPath := filepath.Join(t.TempDir(), "outside-target.json")
	if err := os.WriteFile(targetPath, []byte(`{"workspace_id":"ws-watch","source_event_id":"mail-symlink-1"}`), 0o644); err != nil {
		t.Fatalf("write symlink target file: %v", err)
	}
	if err := os.Chmod(targetPath, 0o644); err != nil {
		t.Fatalf("chmod symlink target file: %v", err)
	}

	pendingPath := filepath.Join(inboxDir, "mail", "pending", "001-mail.json")
	if err := os.MkdirAll(filepath.Dir(pendingPath), 0o755); err != nil {
		t.Fatalf("create pending dir: %v", err)
	}
	if err := os.Symlink(targetPath, pendingPath); err != nil {
		t.Fatalf("create watcher symlink payload: %v", err)
	}

	stub := &inboundWatcherCommStub{
		messagesResponse: transport.MessagesIngestResponse{
			WorkspaceID: "ws-watch",
		},
	}
	watcher, err := NewInboundWatcherRuntime(db, stub, InboundWatcherRuntimeOptions{
		WorkspaceID: "ws-watch",
		InboxDir:    inboxDir,
	})
	if err != nil {
		t.Fatalf("new inbound watcher runtime: %v", err)
	}

	processed, pollErr := watcher.pollOnce(context.Background())
	if !processed {
		t.Fatalf("expected poll to process symlink payload")
	}
	if pollErr == nil || !strings.Contains(strings.ToLower(pollErr.Error()), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", pollErr)
	}

	_, mailCalls, _, _ := stub.snapshotCounts()
	if mailCalls != 0 {
		t.Fatalf("expected no mail ingest calls for symlink payload, got %d", mailCalls)
	}

	if count := countInboundWatcherJSONFiles(t, filepath.Join(inboxDir, "mail", "pending")); count != 0 {
		t.Fatalf("expected mail pending dir to be empty, got %d file(s)", count)
	}
	failedDir := filepath.Join(inboxDir, "mail", "failed")
	entries, err := os.ReadDir(failedDir)
	if err != nil {
		t.Fatalf("read failed dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one failed watcher payload entry, got %d", len(entries))
	}
	failedPath := filepath.Join(failedDir, entries[0].Name())
	failedInfo, err := os.Lstat(failedPath)
	if err != nil {
		t.Fatalf("lstat failed payload: %v", err)
	}
	if failedInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected failed payload to remain quarantined symlink, mode=%v", failedInfo.Mode())
	}

	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat symlink target file: %v", err)
	}
	if got := targetInfo.Mode().Perm(); got != 0o644 {
		t.Fatalf("expected symlink target permissions unchanged at 0644, got %o", got)
	}

	var lastError string
	if err := db.QueryRow(
		`SELECT COALESCE(last_error, '') FROM automation_source_subscriptions WHERE workspace_id = ? AND source = ? AND source_scope = ? LIMIT 1`,
		"ws-watch",
		mailRuleIngestSource,
		resolveMailSourceScope(""),
	).Scan(&lastError); err != nil {
		t.Fatalf("query mail source-subscription error: %v", err)
	}
	if !strings.Contains(strings.ToLower(lastError), "symlink") {
		t.Fatalf("expected symlink quarantine error in source subscription, got %q", lastError)
	}
}

func TestArchiveInboundWatcherFileTightensTargetPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "inbox")
	pendingDir := filepath.Join(root, "mail", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatalf("create pending dir: %v", err)
	}
	sourcePath := filepath.Join(pendingDir, "001-mail.json")
	if err := os.WriteFile(sourcePath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.Chmod(sourcePath, 0o644); err != nil {
		t.Fatalf("set source file permissions: %v", err)
	}

	targetDir := filepath.Join(root, "mail", "processed")
	archivedPath, err := archiveInboundWatcherFile(sourcePath, targetDir, time.Now().UTC())
	if err != nil {
		t.Fatalf("archive inbound file: %v", err)
	}
	if archivedPath == "" {
		t.Fatalf("expected archived path")
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("stat archived file: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(targetDir)
	if err != nil {
		t.Fatalf("stat target dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected processed dir permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(archivedPath)
	if err != nil {
		t.Fatalf("stat archived file for mode: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected archived file permissions 0600, got %o", got)
	}
}

func newInboundWatcherTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := openRuntimeDB(context.Background(), filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatalf("open runtime db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func writeInboundWatcherJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal watcher payload %s: %v", path, err)
	}
	writeInboundWatcherRawFile(t, path, string(payload))
}

func writeInboundWatcherRawFile(t *testing.T, path string, raw string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create watcher dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write watcher file %s: %v", path, err)
	}
}

func countInboundWatcherJSONFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		count++
	}
	return count
}

func inboundWatcherCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func waitForInboundWatcherCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
