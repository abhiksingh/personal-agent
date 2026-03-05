package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"personalagent/runtime/internal/transport"
	"personalagent/runtime/internal/workspaceid"
)

const (
	defaultInboundWatcherPollInterval  = 2 * time.Second
	defaultInboundWatcherWorkspaceID   = workspaceid.CanonicalDefault
	defaultInboundWatcherMessagesLimit = 100
	maxInboundWatcherMessagesLimit     = 500
	defaultInboundWatcherFileBatchSize = 100
	maxInboundWatcherFileBatchSize     = 500

	envInboundWatcherWorkspaceID          = "PA_INBOUND_WATCHER_WORKSPACE_ID"
	envInboundWatcherInboxDir             = "PA_INBOUND_WATCHER_INBOX_DIR"
	envInboundWatcherMessagesSourceScope  = "PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE"
	envInboundWatcherMessagesSourceDBPath = "PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH"
	envInboundWatcherMessagesLimit        = "PA_INBOUND_WATCHER_MESSAGES_LIMIT"
	envInboundWatcherPollInterval         = "PA_INBOUND_WATCHER_POLL_INTERVAL"
	envInboundWatcherFileBatchSize        = "PA_INBOUND_WATCHER_FILE_BATCH_SIZE"
)

type inboundWatcherCommService interface {
	IngestMessages(ctx context.Context, request transport.MessagesIngestRequest) (transport.MessagesIngestResponse, error)
	IngestMailRuleEvent(ctx context.Context, request transport.MailRuleIngestRequest) (transport.MailRuleIngestResponse, error)
	IngestCalendarChange(ctx context.Context, request transport.CalendarChangeIngestRequest) (transport.CalendarChangeIngestResponse, error)
	IngestBrowserEvent(ctx context.Context, request transport.BrowserEventIngestRequest) (transport.BrowserEventIngestResponse, error)
}

type InboundWatcherRuntimeOptions struct {
	PollInterval         time.Duration
	WorkspaceID          string
	ResolveWorkspaceID   func(ctx context.Context) (string, error)
	InboxDir             string
	MessagesSourceScope  string
	MessagesSourceDBPath string
	MessagesLimit        int
	FileBatchSize        int
	Now                  func() time.Time
}

type inboundWatcherFileIngestResult struct {
	WorkspaceID string
	SourceScope string
}

type inboundWatcherFileAdapter struct {
	name         string
	source       string
	subdir       string
	defaultScope string
	ingest       func(ctx context.Context, raw []byte, workspaceID string) (inboundWatcherFileIngestResult, error)
}

type InboundWatcherRuntime struct {
	db                   *sql.DB
	comm                 inboundWatcherCommService
	pollInterval         time.Duration
	workspaceID          string
	workspacePinned      bool
	resolveWorkspaceID   func(ctx context.Context) (string, error)
	inboxDir             string
	messagesSourceScope  string
	messagesSourceDBPath string
	messagesLimit        int
	fileBatchSize        int
	now                  func() time.Time
	fileAdapters         []inboundWatcherFileAdapter

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewInboundWatcherRuntime(db *sql.DB, comm inboundWatcherCommService, opts InboundWatcherRuntimeOptions) (*InboundWatcherRuntime, error) {
	if db == nil {
		return nil, fmt.Errorf("inbound watcher runtime db is required")
	}
	if comm == nil {
		return nil, fmt.Errorf("inbound watcher runtime comm service is required")
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}

	workspaceID, workspacePinned := resolveInboundWatcherWorkspaceID(opts.WorkspaceID)

	runtime := &InboundWatcherRuntime{
		db:                   db,
		comm:                 comm,
		pollInterval:         resolveInboundWatcherPollInterval(opts.PollInterval),
		workspaceID:          workspaceID,
		workspacePinned:      workspacePinned,
		resolveWorkspaceID:   opts.ResolveWorkspaceID,
		inboxDir:             resolveInboundWatcherInboxDir(opts.InboxDir),
		messagesSourceScope:  resolveInboundWatcherMessagesSourceScope(opts.MessagesSourceScope),
		messagesSourceDBPath: resolveInboundWatcherMessagesSourceDBPath(opts.MessagesSourceDBPath),
		messagesLimit:        resolveInboundWatcherMessagesLimit(opts.MessagesLimit),
		fileBatchSize:        resolveInboundWatcherFileBatchSize(opts.FileBatchSize),
		now:                  nowFn,
	}

	runtime.fileAdapters = []inboundWatcherFileAdapter{
		{
			name:         "mail",
			source:       mailRuleIngestSource,
			subdir:       "mail",
			defaultScope: resolveMailSourceScope(""),
			ingest:       runtime.ingestMailFile,
		},
		{
			name:         "calendar",
			source:       calendarChangeIngestSource,
			subdir:       "calendar",
			defaultScope: resolveCalendarSourceScope("", ""),
			ingest:       runtime.ingestCalendarFile,
		},
		{
			name:         "browser",
			source:       browserEventIngestSource,
			subdir:       "browser",
			defaultScope: resolveBrowserSourceScope("", ""),
			ingest:       runtime.ingestBrowserFile,
		},
	}
	return runtime, nil
}

func (r *InboundWatcherRuntime) Start(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return nil
	}

	runCtx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.running = true
	go r.runLoop(runCtx, done)
	return nil
}

func (r *InboundWatcherRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	r.running = false
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
