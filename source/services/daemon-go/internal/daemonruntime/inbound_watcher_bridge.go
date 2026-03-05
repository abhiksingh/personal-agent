package daemonruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"personalagent/runtime/internal/filesecurity"
)

type InboundWatcherBridgeDirectoryStatus struct {
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Writable bool   `json:"writable"`
	Error    string `json:"error,omitempty"`
}

type InboundWatcherBridgeSourceStatus struct {
	Source    string                              `json:"source"`
	Ready     bool                                `json:"ready"`
	Pending   InboundWatcherBridgeDirectoryStatus `json:"pending"`
	Processed InboundWatcherBridgeDirectoryStatus `json:"processed"`
	Failed    InboundWatcherBridgeDirectoryStatus `json:"failed"`
}

type InboundWatcherBridgeStatus struct {
	InboxRoot string                             `json:"inbox_root"`
	Ready     bool                               `json:"ready"`
	Sources   []InboundWatcherBridgeSourceStatus `json:"sources"`
}

type inboundWatcherBridgeSourceSpec struct {
	source string
	subdir string
}

var inboundWatcherBridgeSources = []inboundWatcherBridgeSourceSpec{
	{source: "mail", subdir: "mail"},
	{source: "calendar", subdir: "calendar"},
	{source: "browser", subdir: "browser"},
}

func ResolveInboundWatcherInboxDir(override string) string {
	return resolveInboundWatcherInboxDir(override)
}

func InspectInboundWatcherBridge(inboxDirOverride string) InboundWatcherBridgeStatus {
	return inspectInboundWatcherBridge(inboxDirOverride, false)
}

func EnsureInboundWatcherBridge(inboxDirOverride string) InboundWatcherBridgeStatus {
	return inspectInboundWatcherBridge(inboxDirOverride, true)
}

func inspectInboundWatcherBridge(inboxDirOverride string, ensure bool) InboundWatcherBridgeStatus {
	inboxRoot := resolveInboundWatcherInboxDir(inboxDirOverride)
	status := InboundWatcherBridgeStatus{
		InboxRoot: inboxRoot,
		Ready:     true,
		Sources:   make([]InboundWatcherBridgeSourceStatus, 0, len(inboundWatcherBridgeSources)),
	}

	for _, spec := range inboundWatcherBridgeSources {
		pendingPath := filepath.Join(inboxRoot, spec.subdir, "pending")
		processedPath := filepath.Join(inboxRoot, spec.subdir, "processed")
		failedPath := filepath.Join(inboxRoot, spec.subdir, "failed")

		sourceStatus := InboundWatcherBridgeSourceStatus{
			Source:    spec.source,
			Pending:   inspectInboundWatcherBridgeDir(pendingPath, ensure),
			Processed: inspectInboundWatcherBridgeDir(processedPath, ensure),
			Failed:    inspectInboundWatcherBridgeDir(failedPath, ensure),
		}
		sourceStatus.Ready =
			sourceStatus.Pending.Exists && sourceStatus.Pending.Writable &&
				sourceStatus.Processed.Exists && sourceStatus.Processed.Writable &&
				sourceStatus.Failed.Exists && sourceStatus.Failed.Writable

		if !sourceStatus.Ready {
			status.Ready = false
		}
		status.Sources = append(status.Sources, sourceStatus)
	}
	return status
}

func inspectInboundWatcherBridgeDir(path string, ensure bool) InboundWatcherBridgeDirectoryStatus {
	status := InboundWatcherBridgeDirectoryStatus{
		Path: strings.TrimSpace(path),
	}

	if status.Path == "" {
		status.Error = "bridge path is empty"
		return status
	}

	if ensure {
		if err := filesecurity.EnsurePrivateDir(status.Path); err != nil {
			status.Error = fmt.Sprintf("create bridge directory: %v", err)
			return status
		}
	}

	info, err := os.Stat(status.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			status.Error = "bridge directory does not exist"
			return status
		}
		status.Error = fmt.Sprintf("stat bridge directory: %v", err)
		return status
	}
	if !info.IsDir() {
		status.Error = "bridge path is not a directory"
		return status
	}
	status.Exists = true

	writable, writableErr := inboundWatcherBridgeDirWritable(status.Path)
	status.Writable = writable
	if writableErr != nil {
		status.Error = fmt.Sprintf("bridge directory is not writable: %v", writableErr)
	}
	return status
}

func inboundWatcherBridgeDirWritable(path string) (bool, error) {
	tmp, err := os.CreateTemp(path, ".pa-bridge-write-check-*.tmp")
	if err != nil {
		return false, err
	}
	name := tmp.Name()
	closeErr := tmp.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return false, closeErr
	}
	if removeErr != nil {
		return false, removeErr
	}
	return true, nil
}

func InboundWatcherBridgeSourceByID(status InboundWatcherBridgeStatus, sourceID string) (InboundWatcherBridgeSourceStatus, bool) {
	target := strings.ToLower(strings.TrimSpace(sourceID))
	if target == "" {
		return InboundWatcherBridgeSourceStatus{}, false
	}
	for _, source := range status.Sources {
		if strings.EqualFold(strings.TrimSpace(source.Source), target) {
			return source, true
		}
	}
	return InboundWatcherBridgeSourceStatus{}, false
}
