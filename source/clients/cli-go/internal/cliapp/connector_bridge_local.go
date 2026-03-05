package cliapp

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"personalagent/runtime/internal/daemonruntime"
	"personalagent/runtime/internal/filesecurity"
)

var bridgeFileTokenSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type connectorBridgeCommandResponse struct {
	WorkspaceID   string                                   `json:"workspace_id"`
	EnsureApplied bool                                     `json:"ensure_applied,omitempty"`
	Status        daemonruntime.InboundWatcherBridgeStatus `json:"status"`
}

type connectorBridgeEnqueueResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	Source        string `json:"source"`
	SourceEventID string `json:"source_event_id"`
	InboxRoot     string `json:"inbox_root"`
	PendingDir    string `json:"pending_dir"`
	FilePath      string `json:"file_path"`
	Queued        bool   `json:"queued"`
}

func runConnectorBridgeLocalCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector bridge subcommand required: status|setup")
		return 2
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "status":
		flags := flag.NewFlagSet("connector bridge status", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		inboxDir := flags.String("inbox-dir", "", "optional override for local ingress bridge inbox root")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response := connectorBridgeCommandResponse{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Status:      daemonruntime.InspectInboundWatcherBridge(strings.TrimSpace(*inboxDir)),
		}
		return writeJSON(stdout, response)
	case "setup":
		flags := flag.NewFlagSet("connector bridge setup", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		inboxDir := flags.String("inbox-dir", "", "optional override for local ingress bridge inbox root")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response := connectorBridgeCommandResponse{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			EnsureApplied: true,
			Status:        daemonruntime.EnsureInboundWatcherBridge(strings.TrimSpace(*inboxDir)),
		}
		if code := writeJSON(stdout, response); code != 0 {
			return code
		}
		if !response.Status.Ready {
			return 1
		}
		return 0
	default:
		writeUnknownSubcommandError(stderr, "connector bridge subcommand", args[0])
		return 2
	}
}

func enqueueLocalIngressBridgePayload(
	_ context.Context,
	workspaceID string,
	source string,
	sourceEventID string,
	payload any,
	inboxDirOverride string,
) (connectorBridgeEnqueueResponse, error) {
	workspace := normalizeWorkspace(workspaceID)
	normalizedSource := strings.ToLower(strings.TrimSpace(source))
	if normalizedSource == "" {
		return connectorBridgeEnqueueResponse{}, fmt.Errorf("bridge source is required")
	}

	status := daemonruntime.EnsureInboundWatcherBridge(strings.TrimSpace(inboxDirOverride))
	sourceStatus, found := daemonruntime.InboundWatcherBridgeSourceByID(status, normalizedSource)
	if !found {
		return connectorBridgeEnqueueResponse{}, fmt.Errorf("unsupported bridge source %q", source)
	}
	if !sourceStatus.Ready {
		return connectorBridgeEnqueueResponse{}, fmt.Errorf("local ingest bridge source %s is not ready: %s", normalizedSource, localIngressBridgeSourceFailureReason(sourceStatus))
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return connectorBridgeEnqueueResponse{}, fmt.Errorf("encode %s bridge payload: %w", normalizedSource, err)
	}

	eventID := strings.TrimSpace(sourceEventID)
	if eventID == "" {
		eventID = fmt.Sprintf("%s-%d", normalizedSource, time.Now().UTC().UnixNano())
	}
	fileToken := sanitizeBridgeFileToken(eventID)
	if fileToken == "" {
		fileToken = fmt.Sprintf("%s-%d", normalizedSource, time.Now().UTC().UnixNano())
	}
	fileName := fmt.Sprintf("%d-%s-%s.json", time.Now().UTC().UnixNano(), sanitizeBridgeFileToken(normalizedSource), fileToken)

	path, err := writeLocalIngressBridgePayload(sourceStatus.Pending.Path, fileName, payloadBytes)
	if err != nil {
		return connectorBridgeEnqueueResponse{}, err
	}

	return connectorBridgeEnqueueResponse{
		WorkspaceID:   workspace,
		Source:        normalizedSource,
		SourceEventID: eventID,
		InboxRoot:     status.InboxRoot,
		PendingDir:    sourceStatus.Pending.Path,
		FilePath:      path,
		Queued:        true,
	}, nil
}

func writeLocalIngressBridgePayload(pendingDir string, fileName string, payload []byte) (string, error) {
	if strings.TrimSpace(pendingDir) == "" {
		return "", fmt.Errorf("bridge pending directory is empty")
	}
	if strings.TrimSpace(fileName) == "" {
		return "", fmt.Errorf("bridge payload filename is empty")
	}
	if err := filesecurity.EnsurePrivateDir(pendingDir); err != nil {
		return "", fmt.Errorf("ensure bridge pending directory: %w", err)
	}
	targetPath := filepath.Join(pendingDir, fileName)
	tmpFile, err := os.CreateTemp(pendingDir, ".pa-bridge-*.json")
	if err != nil {
		return "", fmt.Errorf("create bridge payload temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	written := false
	defer func() {
		if !written {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(filesecurity.PrivateFileMode); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("set bridge payload temp file permissions: %w", err)
	}
	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("write bridge payload temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close bridge payload temp file: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return "", fmt.Errorf("move bridge payload into pending queue: %w", err)
	}
	if err := filesecurity.EnsurePrivateFile(targetPath); err != nil {
		return "", fmt.Errorf("harden bridge payload file permissions: %w", err)
	}
	written = true
	return targetPath, nil
}

func sanitizeBridgeFileToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	sanitized := bridgeFileTokenSanitizer.ReplaceAllString(trimmed, "-")
	sanitized = strings.Trim(sanitized, "-")
	return strings.ToLower(sanitized)
}

func localIngressBridgeSourceFailureReason(status daemonruntime.InboundWatcherBridgeSourceStatus) string {
	reasons := []string{}
	for _, detail := range []string{
		strings.TrimSpace(status.Pending.Error),
		strings.TrimSpace(status.Processed.Error),
		strings.TrimSpace(status.Failed.Error),
	} {
		if detail != "" {
			reasons = append(reasons, detail)
		}
	}
	if len(reasons) == 0 {
		return "queue path checks failed"
	}
	return strings.Join(reasons, "; ")
}
