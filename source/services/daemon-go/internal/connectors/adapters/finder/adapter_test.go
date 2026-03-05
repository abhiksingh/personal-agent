package finder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

func TestExecuteStepListPreviewDeleteAgainstFilesystem(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	targetFile := filepath.Join(workDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	adapter := NewAdapter("finder.test")
	baseCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		TaskID:      "task-1",
		RunID:       "run-1",
	}

	listCtx := baseCtx
	listCtx.StepID = "step-list-1"
	listStep := connectorcontract.TaskStep{
		ID:            "step-list-1",
		CapabilityKey: CapabilityList,
		Name:          "List path",
		Input: map[string]any{
			"path": targetFile,
		},
	}
	listResult, err := adapter.ExecuteStep(context.Background(), listCtx, listStep)
	if err != nil {
		t.Fatalf("execute list step: %v", err)
	}
	if got := listResult.Evidence["file_count"]; got != "1" {
		t.Fatalf("expected file_count=1, got %q", got)
	}
	if got := listResult.Evidence["exists"]; got != "true" {
		t.Fatalf("expected exists=true, got %q", got)
	}
	if got := listResult.Evidence["resolved_via"]; got != "path" {
		t.Fatalf("expected resolved_via=path, got %q", got)
	}

	previewCtx := baseCtx
	previewCtx.StepID = "step-preview-1"
	previewStep := connectorcontract.TaskStep{
		ID:            "step-preview-1",
		CapabilityKey: CapabilityPreview,
		Name:          "Preview path",
		Input: map[string]any{
			"path": targetFile,
		},
	}
	previewResult, err := adapter.ExecuteStep(context.Background(), previewCtx, previewStep)
	if err != nil {
		t.Fatalf("execute preview step: %v", err)
	}
	if previewResult.Evidence["preview_id"] == "" {
		t.Fatalf("expected preview_id evidence")
	}
	if previewResult.Evidence["exists"] != "true" {
		t.Fatalf("expected preview exists=true")
	}

	deleteCtx := baseCtx
	deleteCtx.StepID = "step-delete-1"
	deleteStep := connectorcontract.TaskStep{
		ID:            "step-delete-1",
		CapabilityKey: CapabilityDelete,
		Name:          "Delete path",
		Input: map[string]any{
			"path": targetFile,
		},
	}
	deleteFirst, err := adapter.ExecuteStep(context.Background(), deleteCtx, deleteStep)
	if err != nil {
		t.Fatalf("execute delete step: %v", err)
	}
	deleteSecond, err := adapter.ExecuteStep(context.Background(), deleteCtx, deleteStep)
	if err != nil {
		t.Fatalf("execute delete step second time: %v", err)
	}
	if deleteFirst.Evidence["delete_id"] != deleteSecond.Evidence["delete_id"] {
		t.Fatalf("expected idempotent delete_id, got %q vs %q", deleteFirst.Evidence["delete_id"], deleteSecond.Evidence["delete_id"])
	}
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("expected target file deleted, stat err=%v", err)
	}
}

func TestExecuteStepFindReturnsDeterministicMatches(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	bestMatch := filepath.Join(workDir, "budget-report-final.txt")
	otherMatch := filepath.Join(workDir, "budget-report-draft.txt")
	if err := os.WriteFile(bestMatch, []byte("final"), 0o644); err != nil {
		t.Fatalf("write best match file: %v", err)
	}
	if err := os.WriteFile(otherMatch, []byte("draft"), 0o644); err != nil {
		t.Fatalf("write other match file: %v", err)
	}

	adapter := NewAdapter("finder.test")
	result, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		StepID:      "step-find-1",
	}, connectorcontract.TaskStep{
		ID:            "step-find-1",
		CapabilityKey: CapabilityFind,
		Name:          "Find budget report",
		Input: map[string]any{
			"query":     "budget report final",
			"root_path": workDir,
		},
	})
	if err != nil {
		t.Fatalf("execute find step: %v", err)
	}
	if got := result.Evidence["match_count"]; got == "0" {
		t.Fatalf("expected at least one finder match, got %q", got)
	}
	if got := result.Evidence["selected_path"]; got != bestMatch {
		t.Fatalf("expected selected_path=%q, got %q", bestMatch, got)
	}
	matches, ok := result.Output["matches"].([]map[string]any)
	if !ok || len(matches) == 0 {
		t.Fatalf("expected structured matches output, got %#v", result.Output["matches"])
	}
}

func TestExecuteStepListResolvesQueryWhenPathMissing(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	targetFile := filepath.Join(workDir, "travel-checklist.txt")
	if err := os.WriteFile(targetFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	adapter := NewAdapter("finder.test")
	result, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		StepID:      "step-list-query",
	}, connectorcontract.TaskStep{
		ID:            "step-list-query",
		CapabilityKey: CapabilityList,
		Name:          "List by query",
		Input: map[string]any{
			"query":     "travel checklist",
			"root_path": workDir,
		},
	})
	if err != nil {
		t.Fatalf("execute list via query step: %v", err)
	}
	if got := result.Evidence["resolved_via"]; got != "query" {
		t.Fatalf("expected resolved_via=query, got %q", got)
	}
	if got := result.Evidence["selected_path"]; got != targetFile {
		t.Fatalf("expected selected_path=%q, got %q", targetFile, got)
	}
}

func TestExecuteStepDeleteQueryAmbiguousBlocked(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	first := filepath.Join(workDir, "report-one.txt")
	second := filepath.Join(workDir, "report-two.txt")
	if err := os.WriteFile(first, []byte("1"), 0o644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(second, []byte("2"), 0o644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	adapter := NewAdapter("finder.test")
	result, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		StepID:      "step-delete-query",
	}, connectorcontract.TaskStep{
		ID:            "step-delete-query",
		CapabilityKey: CapabilityDelete,
		Name:          "Delete by ambiguous query",
		Input: map[string]any{
			"query":     "report",
			"root_path": workDir,
		},
	})
	if err == nil {
		t.Fatalf("expected delete query guardrail error")
	}
	if got := result.ErrorReason; got != "guardrail_denied" {
		t.Fatalf("expected guardrail_denied error_reason, got %q", got)
	}
	if _, statErr := os.Stat(first); statErr != nil {
		t.Fatalf("expected first file to remain, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(second); statErr != nil {
		t.Fatalf("expected second file to remain, stat err=%v", statErr)
	}
}

func TestExecuteStepDeleteMissingPathIsNoOp(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	targetFile := filepath.Join(t.TempDir(), "missing.txt")
	adapter := NewAdapter("finder.test")
	result, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		StepID:      "step-delete-missing",
	}, connectorcontract.TaskStep{
		ID:            "step-delete-missing",
		CapabilityKey: CapabilityDelete,
		Name:          "Delete path",
		Input: map[string]any{
			"path": targetFile,
		},
	})
	if err != nil {
		t.Fatalf("execute delete missing step: %v", err)
	}
	if got := result.Evidence["existed"]; got != "false" {
		t.Fatalf("expected existed=false, got %q", got)
	}
}

func TestExecuteStepDeleteRejectsUnsafePath(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	adapter := NewAdapter("finder.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_finder",
		StepID:      "step-delete-root",
	}, connectorcontract.TaskStep{
		ID:            "step-delete-root",
		CapabilityKey: CapabilityDelete,
		Name:          "Delete path",
		Input: map[string]any{
			"path": "/",
		},
	})
	if err == nil {
		t.Fatalf("expected guardrail error for root delete")
	}
}
