package localbridge

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSendAppChatPersistsArtifact(t *testing.T) {
	t.Setenv(envChannelDataDir, t.TempDir())

	response, err := SendAppChat(context.Background(), AppChatSendRequest{
		WorkspaceID: "ws1",
		ThreadID:    "thread-a",
		Message:     "hello",
	})
	if err != nil {
		t.Fatalf("send app chat: %v", err)
	}
	if response.Status != "sent" {
		t.Fatalf("expected sent status, got %s", response.Status)
	}
	if response.Transport != transportAppChatLocal {
		t.Fatalf("expected app chat local transport marker, got %s", response.Transport)
	}
	if response.RecordPath == "" {
		t.Fatalf("expected record path")
	}
	if _, err := os.Stat(response.RecordPath); err != nil {
		t.Fatalf("expected app chat record file at %s: %v", response.RecordPath, err)
	}
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(response.RecordPath)
		if err != nil {
			t.Fatalf("stat record file: %v", err)
		}
		if got := fileInfo.Mode().Perm(); got != 0o600 {
			t.Fatalf("expected app chat record permissions 0600, got %o", got)
		}
		dirInfo, err := os.Stat(filepath.Dir(response.RecordPath))
		if err != nil {
			t.Fatalf("stat record directory: %v", err)
		}
		if got := dirInfo.Mode().Perm(); got != 0o700 {
			t.Fatalf("expected app chat record directory permissions 0700, got %o", got)
		}
	}
}

func TestStatusReportsWritableOutbox(t *testing.T) {
	base := t.TempDir()
	t.Setenv(envChannelDataDir, base)

	appStatus := AppChatStatus()
	if !appStatus.Ready {
		t.Fatalf("expected app chat status ready, got error: %s", appStatus.Error)
	}
	expectedAppDir := filepath.Join(base, "channels", "app_chat")
	if appStatus.OutboxDir != expectedAppDir {
		t.Fatalf("expected app chat outbox %s, got %s", expectedAppDir, appStatus.OutboxDir)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(appStatus.OutboxDir)
		if err != nil {
			t.Fatalf("stat app outbox dir: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("expected app outbox dir permissions 0700, got %o", got)
		}
	}
}
