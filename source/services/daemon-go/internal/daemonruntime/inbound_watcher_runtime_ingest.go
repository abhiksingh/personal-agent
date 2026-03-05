package daemonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/filesecurity"
	"personalagent/runtime/internal/transport"
)

func (r *InboundWatcherRuntime) pollMessages(ctx context.Context, workspaceID string) (bool, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	sourcePath := strings.TrimSpace(r.messagesSourceDBPath)
	resolvedScope := messagesadapter.ResolveSourceScope(r.messagesSourceScope, messagesadapter.ResolveSourceDBPath(sourcePath))

	response, err := r.comm.IngestMessages(ctx, transport.MessagesIngestRequest{
		WorkspaceID:  workspace,
		SourceScope:  r.messagesSourceScope,
		SourceDBPath: sourcePath,
		Limit:        r.messagesLimit,
	})
	if err != nil {
		_ = upsertAutomationSourceSubscription(ctx, r.db, workspace, messagesadapter.SourceName, resolvedScope, "", "", err.Error())
		return false, err
	}
	return response.Accepted > 0 || response.Replayed > 0 || len(response.Events) > 0, nil
}

func (r *InboundWatcherRuntime) pollFileAdapter(ctx context.Context, adapter inboundWatcherFileAdapter, workspaceID string) (bool, error) {
	pendingDir := filepath.Join(r.inboxDir, adapter.subdir, "pending")
	if err := filesecurity.EnsurePrivateDir(pendingDir); err != nil {
		return false, fmt.Errorf("%s watcher pending dir: %w", adapter.name, err)
	}

	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		return false, fmt.Errorf("%s watcher read pending dir: %w", adapter.name, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	workDone := false
	var firstErr error
	processed := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return workDone, ctx.Err()
		}
		if processed >= r.fileBatchSize {
			break
		}
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if strings.ToLower(filepath.Ext(name)) != ".json" {
			continue
		}

		processed++
		filePath := filepath.Join(pendingDir, name)
		info, statErr := os.Lstat(filePath)
		if statErr != nil {
			workDone = true
			if err := r.failInboundFile(ctx, adapter, filePath, workspaceID, adapter.defaultScope, fmt.Errorf("lstat watcher payload: %w", statErr)); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			workDone = true
			if err := r.failInboundFile(ctx, adapter, filePath, workspaceID, adapter.defaultScope, fmt.Errorf("reject watcher payload symlink: %s", name)); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			workDone = true
			if err := r.failInboundFile(ctx, adapter, filePath, workspaceID, adapter.defaultScope, fmt.Errorf("reject watcher payload non-regular file: %s mode=%v", name, info.Mode())); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}

		raw, readErr := os.ReadFile(filePath)
		if readErr != nil {
			workDone = true
			if err := r.failInboundFile(ctx, adapter, filePath, workspaceID, adapter.defaultScope, fmt.Errorf("read watcher payload: %w", readErr)); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}

		result, ingestErr := adapter.ingest(ctx, raw, workspaceID)
		if ingestErr != nil {
			workDone = true
			scope := firstNonEmpty(strings.TrimSpace(result.SourceScope), adapter.defaultScope)
			if err := r.failInboundFile(ctx, adapter, filePath, workspaceID, scope, ingestErr); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}

		processedDir := filepath.Join(r.inboxDir, adapter.subdir, "processed")
		if _, moveErr := archiveInboundWatcherFile(filePath, processedDir, r.now()); moveErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("archive watcher %s file %s: %w", adapter.name, name, moveErr)
			}
			continue
		}
		workDone = true
	}
	return workDone, firstErr
}

func (r *InboundWatcherRuntime) failInboundFile(ctx context.Context, adapter inboundWatcherFileAdapter, filePath string, workspaceID string, sourceScope string, ingestErr error) error {
	failedDir := filepath.Join(r.inboxDir, adapter.subdir, "failed")
	if _, moveErr := archiveInboundWatcherFile(filePath, failedDir, r.now()); moveErr != nil {
		return fmt.Errorf("%s watcher move failed payload: %w", adapter.name, moveErr)
	}
	scope := firstNonEmpty(strings.TrimSpace(sourceScope), adapter.defaultScope)
	workspace := normalizeWorkspaceID(workspaceID)
	if subErr := upsertAutomationSourceSubscription(ctx, r.db, workspace, adapter.source, scope, "", "", strings.TrimSpace(ingestErr.Error())); subErr != nil {
		return fmt.Errorf("%s watcher upsert failed subscription: %w", adapter.name, subErr)
	}
	return ingestErr
}

func (r *InboundWatcherRuntime) ingestMailFile(ctx context.Context, raw []byte, workspaceID string) (inboundWatcherFileIngestResult, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	request := transport.MailRuleIngestRequest{}
	if err := json.Unmarshal(raw, &request); err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: workspace,
			SourceScope: resolveMailSourceScope(""),
		}, fmt.Errorf("decode mail watcher payload: %w", err)
	}
	request.WorkspaceID = firstNonEmpty(strings.TrimSpace(request.WorkspaceID), workspace)

	response, err := r.comm.IngestMailRuleEvent(ctx, request)
	if err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: normalizeWorkspaceID(request.WorkspaceID),
			SourceScope: resolveMailSourceScope(request.SourceScope),
		}, fmt.Errorf("ingest mail watcher payload: %w", err)
	}
	return inboundWatcherFileIngestResult{
		WorkspaceID: normalizeWorkspaceID(response.WorkspaceID),
		SourceScope: firstNonEmpty(strings.TrimSpace(response.SourceScope), resolveMailSourceScope(request.SourceScope)),
	}, nil
}

func (r *InboundWatcherRuntime) ingestCalendarFile(ctx context.Context, raw []byte, workspaceID string) (inboundWatcherFileIngestResult, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	request := transport.CalendarChangeIngestRequest{}
	if err := json.Unmarshal(raw, &request); err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: workspace,
			SourceScope: resolveCalendarSourceScope("", ""),
		}, fmt.Errorf("decode calendar watcher payload: %w", err)
	}
	request.WorkspaceID = firstNonEmpty(strings.TrimSpace(request.WorkspaceID), workspace)

	response, err := r.comm.IngestCalendarChange(ctx, request)
	if err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: normalizeWorkspaceID(request.WorkspaceID),
			SourceScope: resolveCalendarSourceScope(request.SourceScope, request.CalendarID),
		}, fmt.Errorf("ingest calendar watcher payload: %w", err)
	}
	return inboundWatcherFileIngestResult{
		WorkspaceID: normalizeWorkspaceID(response.WorkspaceID),
		SourceScope: firstNonEmpty(strings.TrimSpace(response.SourceScope), resolveCalendarSourceScope(request.SourceScope, request.CalendarID)),
	}, nil
}

func (r *InboundWatcherRuntime) ingestBrowserFile(ctx context.Context, raw []byte, workspaceID string) (inboundWatcherFileIngestResult, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	request := transport.BrowserEventIngestRequest{}
	if err := json.Unmarshal(raw, &request); err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: workspace,
			SourceScope: resolveBrowserSourceScope("", ""),
		}, fmt.Errorf("decode browser watcher payload: %w", err)
	}
	request.WorkspaceID = firstNonEmpty(strings.TrimSpace(request.WorkspaceID), workspace)

	response, err := r.comm.IngestBrowserEvent(ctx, request)
	if err != nil {
		return inboundWatcherFileIngestResult{
			WorkspaceID: normalizeWorkspaceID(request.WorkspaceID),
			SourceScope: resolveBrowserSourceScope(request.SourceScope, request.WindowID),
		}, fmt.Errorf("ingest browser watcher payload: %w", err)
	}
	return inboundWatcherFileIngestResult{
		WorkspaceID: normalizeWorkspaceID(response.WorkspaceID),
		SourceScope: firstNonEmpty(strings.TrimSpace(response.SourceScope), resolveBrowserSourceScope(request.SourceScope, request.WindowID)),
	}, nil
}

func archiveInboundWatcherFile(path string, targetDir string, now time.Time) (string, error) {
	if err := filesecurity.EnsurePrivateDir(targetDir); err != nil {
		return "", err
	}
	baseName := filepath.Base(path)
	targetPath := filepath.Join(targetDir, baseName)
	if _, err := os.Stat(targetPath); err == nil {
		ext := filepath.Ext(baseName)
		stem := strings.TrimSuffix(baseName, ext)
		targetPath = filepath.Join(targetDir, fmt.Sprintf("%s-%d%s", stem, now.UTC().UnixNano(), ext))
	}
	if err := os.Rename(path, targetPath); err != nil {
		return "", err
	}
	info, err := os.Lstat(targetPath)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return targetPath, nil
	}
	if err := filesecurity.EnsurePrivateFile(targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}
