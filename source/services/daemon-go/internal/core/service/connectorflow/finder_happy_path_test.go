package connectorflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/types"
)

func TestExecuteFinderHappyPathRequiresAndAcceptsDestructiveApproval(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	targetPath := filepath.Join(workDir, "report.txt")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write finder fixture file: %v", err)
	}

	registry := connectorregistry.New()
	if err := registry.Register(finderadapter.NewAdapter("finder.mock")); err != nil {
		t.Fatalf("register finder adapter: %v", err)
	}

	service := NewFinderHappyPathService(registry, nil, nil)
	result, err := service.Execute(context.Background(), types.FinderHappyPathRequest{
		WorkspaceID:      "ws_finder",
		RunID:            "run_finder_happy_path",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_finder_happy_path",
		TargetQuery:      "report",
		SearchRootPath:   workDir,
		ApprovalPhrase:   "GO AHEAD",
	})
	if err != nil {
		t.Fatalf("execute finder happy path: %v", err)
	}

	if !result.GateDecision.RequireApproval {
		t.Fatalf("expected destructive delete step to require approval")
	}
	if result.FindTrace.CapabilityKey != "finder_find" {
		t.Fatalf("expected finder_find trace, got %+v", result.FindTrace)
	}
	if result.FindTrace.Evidence["selected_path"] != targetPath {
		t.Fatalf("expected selected path %q, got %q", targetPath, result.FindTrace.Evidence["selected_path"])
	}
	if result.DeleteTrace.Evidence["deleted_path"] != targetPath {
		t.Fatalf("expected deleted path evidence for approved flow")
	}
}

func TestExecuteFinderHappyPathBlocksWhenApprovalPhraseMissing(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	workDir := t.TempDir()
	targetPath := filepath.Join(workDir, "report.txt")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write finder fixture file: %v", err)
	}

	registry := connectorregistry.New()
	if err := registry.Register(finderadapter.NewAdapter("finder.mock")); err != nil {
		t.Fatalf("register finder adapter: %v", err)
	}

	service := NewFinderHappyPathService(registry, nil, nil)
	_, err := service.Execute(context.Background(), types.FinderHappyPathRequest{
		WorkspaceID:      "ws_finder",
		RunID:            "run_finder_blocked",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_finder_blocked",
		TargetPath:       targetPath,
		ApprovalPhrase:   "yes",
	})
	if err == nil {
		t.Fatalf("expected missing approval phrase error")
	}
	if !strings.Contains(err.Error(), "GO AHEAD") {
		t.Fatalf("expected error to mention GO AHEAD, got %v", err)
	}
}
