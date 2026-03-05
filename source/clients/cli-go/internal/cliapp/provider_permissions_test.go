package cliapp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenRuntimeDBTightensFilesystemPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime-dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed db dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("set seed db dir permissions: %v", err)
	}

	dbPath := filepath.Join(dir, "runtime.db")
	if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
		t.Fatalf("seed db file: %v", err)
	}
	if err := os.Chmod(dbPath, 0o644); err != nil {
		t.Fatalf("set seed db file permissions: %v", err)
	}

	db, err := openRuntimeDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open runtime db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat db dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected db directory permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected db file permissions 0600, got %o", got)
	}
}
