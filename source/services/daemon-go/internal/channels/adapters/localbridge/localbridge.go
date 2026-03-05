package localbridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"personalagent/runtime/internal/filesecurity"
	"personalagent/runtime/internal/runtimepaths"
)

const (
	envChannelDataDir     = "PA_CHANNEL_DATA_DIR"
	transportAppChatLocal = "local_app_chat_artifact"
)

var unsafeTokenPattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type AppChatSendRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id,omitempty"`
	Message     string `json:"message"`
}

type AppChatSendResponse struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id"`
	MessageID   string `json:"message_id"`
	Status      string `json:"status"`
	Transport   string `json:"transport"`
	RecordPath  string `json:"record_path,omitempty"`
}

type AppChatStatusResponse struct {
	Ready     bool   `json:"ready"`
	Transport string `json:"transport"`
	OutboxDir string `json:"outbox_dir,omitempty"`
	Error     string `json:"error,omitempty"`
}

type appChatRecord struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id"`
	MessageID   string `json:"message_id"`
	Message     string `json:"message"`
	CreatedAt   string `json:"created_at"`
}

func SendAppChat(_ context.Context, request AppChatSendRequest) (AppChatSendResponse, error) {
	message := strings.TrimSpace(request.Message)
	if message == "" {
		return AppChatSendResponse{}, fmt.Errorf("app chat message is required")
	}

	workspaceToken := sanitizeToken(request.WorkspaceID, "workspace")
	threadToken := sanitizeToken(request.ThreadID, "app-chat-thread")
	messageID := newMessageID("appchat")

	recordPath := filepath.Join(baseDir(), "app_chat", workspaceToken, "threads", threadToken, "messages", messageID+".json")
	record := appChatRecord{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		ThreadID:    threadToken,
		MessageID:   messageID,
		Message:     message,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := writeJSONFile(recordPath, record); err != nil {
		return AppChatSendResponse{}, fmt.Errorf("persist app chat message: %w", err)
	}

	return AppChatSendResponse{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		ThreadID:    threadToken,
		MessageID:   messageID,
		Status:      "sent",
		Transport:   transportAppChatLocal,
		RecordPath:  recordPath,
	}, nil
}

func AppChatStatus() AppChatStatusResponse {
	outboxDir := filepath.Join(baseDir(), "app_chat")
	if err := ensureWritableDir(outboxDir); err != nil {
		return AppChatStatusResponse{
			Ready:     false,
			Transport: transportAppChatLocal,
			OutboxDir: outboxDir,
			Error:     err.Error(),
		}
	}
	return AppChatStatusResponse{
		Ready:     true,
		Transport: transportAppChatLocal,
		OutboxDir: outboxDir,
	}
}

func baseDir() string {
	if configured := strings.TrimSpace(os.Getenv(envChannelDataDir)); configured != "" {
		return filepath.Join(configured, "channels")
	}
	if defaultChannelsDir, err := runtimepaths.DefaultChannelsDir(); err == nil && strings.TrimSpace(defaultChannelsDir) != "" {
		return defaultChannelsDir
	}
	return filepath.Join(os.TempDir(), "personal-agent", "channels")
}

func sanitizeToken(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	cleaned := unsafeTokenPattern.ReplaceAllString(trimmed, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return fallback
	}
	return strings.ToLower(cleaned)
}

func newMessageID(prefix string) string {
	tokenBytes := make([]byte, 4)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(tokenBytes))
}

func ensureWritableDir(path string) error {
	if err := filesecurity.EnsurePrivateDir(path); err != nil {
		return err
	}
	probe, err := os.CreateTemp(path, ".probe-*.tmp")
	if err != nil {
		return err
	}
	probePath := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probePath)
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := filesecurity.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".local-bridge-*.json")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	cleanup := true
	defer func() {
		_ = tempFile.Close()
		if cleanup {
			_ = os.Remove(tempName)
		}
	}()

	if err := tempFile.Chmod(filesecurity.PrivateFileMode); err != nil {
		return err
	}
	if _, err := tempFile.Write(payload); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	if err := filesecurity.EnsurePrivateFile(path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
