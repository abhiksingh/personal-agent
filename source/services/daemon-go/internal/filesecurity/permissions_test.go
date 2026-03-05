package filesecurity

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsurePrivateDirTightensExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime-dir")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("seed directory: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("set seed permissions: %v", err)
	}

	if err := EnsurePrivateDir(path); err != nil {
		t.Fatalf("ensure private dir: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat directory: %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateDirMode {
		t.Fatalf("expected %o permissions, got %o", PrivateDirMode, got)
	}
}

func TestEnsurePrivateFileTightensExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("set seed file permissions: %v", err)
	}

	if err := EnsurePrivateFile(path); err != nil {
		t.Fatalf("ensure private file: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateFileMode {
		t.Fatalf("expected %o permissions, got %o", PrivateFileMode, got)
	}
}

func TestEnsurePrivateDirSkipsSharedRoots(t *testing.T) {
	if err := EnsurePrivateDir("."); err != nil {
		t.Fatalf("ensure private dir dot path: %v", err)
	}

	tempRoot := os.TempDir()
	if err := EnsurePrivateDir(tempRoot); err != nil {
		t.Fatalf("ensure private dir temp root: %v", err)
	}
}

func TestEnsurePrivateFileRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics are permission-dependent on windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "external.json")
	if err := os.WriteFile(target, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatalf("chmod target file: %v", err)
	}

	symlinkPath := filepath.Join(root, "symlink.json")
	if err := os.Symlink(target, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if err := EnsurePrivateFile(symlinkPath); err == nil {
		t.Fatalf("expected symlink rejection error")
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("expected target permissions unchanged at 0644, got %o", got)
	}
}
