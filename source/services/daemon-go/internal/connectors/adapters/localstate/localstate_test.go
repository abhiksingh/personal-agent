package localstate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestSaveStepResultTightensPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "connectors", "mail", "ws1", "steps")
	path := filepath.Join(dir, "step-1.json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed step dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("set seed step dir permissions: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"legacy":true}`), 0o644); err != nil {
		t.Fatalf("seed step file: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("set seed step file permissions: %v", err)
	}

	if err := SaveStepResult(path, shared.StepExecutionResult{
		Status:  shared.TaskStepStatusCompleted,
		Summary: "ok",
	}); err != nil {
		t.Fatalf("save step result: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat step dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected step directory permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat step file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected step file permissions 0600, got %o", got)
	}
}

func TestWriteJSONFileTightensPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "connectors", "mail", "ws1", "operations", "mail_send")
	path := filepath.Join(dir, "artifact.json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed artifact dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("set seed artifact dir permissions: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"legacy":true}`), 0o644); err != nil {
		t.Fatalf("seed artifact file: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("set seed artifact file permissions: %v", err)
	}

	if err := WriteJSONFile(path, map[string]any{
		"updated": true,
	}); err != nil {
		t.Fatalf("write json file: %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat artifact dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected artifact directory permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat artifact file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected artifact file permissions 0600, got %o", got)
	}
}
