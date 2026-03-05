package cliapp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteLocalIngressBridgePayloadTightensPermissions(t *testing.T) {
	pendingDir := filepath.Join(t.TempDir(), "inbox", "mail", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatalf("seed pending dir: %v", err)
	}
	if err := os.Chmod(pendingDir, 0o755); err != nil {
		t.Fatalf("set seed pending dir permissions: %v", err)
	}

	path, err := writeLocalIngressBridgePayload(pendingDir, "event.json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("write local ingress bridge payload: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat payload file: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(pendingDir)
	if err != nil {
		t.Fatalf("stat pending dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected pending dir permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat payload file for mode: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected payload file permissions 0600, got %o", got)
	}
}
