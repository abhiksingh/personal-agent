package connectorflow

import (
	"context"
	"testing"

	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/types"
)

func TestExecuteBrowserHappyPathWithGuardrailsAndTraceEvidence(t *testing.T) {
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	registry := connectorregistry.New()
	if err := registry.Register(browseradapter.NewAdapter("browser.mock")); err != nil {
		t.Fatalf("register browser adapter: %v", err)
	}

	service := NewBrowserHappyPathService(registry)
	result, err := service.Execute(context.Background(), types.BrowserHappyPathRequest{
		WorkspaceID:      "ws_browser",
		RunID:            "run_browser_happy_path",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_browser_happy_path",
		TargetURL:        "https://example.com",
	})
	if err != nil {
		t.Fatalf("execute browser happy path: %v", err)
	}

	if result.OpenTrace.CapabilityKey != "browser_open" || result.ExtractTrace.CapabilityKey != "browser_extract" || result.CloseTrace.CapabilityKey != "browser_close" {
		t.Fatalf("expected browser open/extract/close capability sequence")
	}
	if result.OpenTrace.Evidence["url"] != "https://example.com" {
		t.Fatalf("expected open evidence url to match target")
	}
	if result.ExtractTrace.Evidence["extraction_id"] == "" {
		t.Fatalf("expected extraction evidence")
	}
	if result.ExtractTrace.Evidence["content_chars"] == "" {
		t.Fatalf("expected deep extraction content_chars evidence")
	}
	if result.ExtractTrace.Evidence["query_answer"] == "" {
		t.Fatalf("expected query-grounded extraction answer evidence")
	}
	if result.CloseTrace.Evidence["closed"] != "true" {
		t.Fatalf("expected close evidence to indicate closed=true")
	}
}

func TestExecuteBrowserHappyPathBlocksNonHTTPSURL(t *testing.T) {
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")

	registry := connectorregistry.New()
	if err := registry.Register(browseradapter.NewAdapter("browser.mock")); err != nil {
		t.Fatalf("register browser adapter: %v", err)
	}

	service := NewBrowserHappyPathService(registry)
	_, err := service.Execute(context.Background(), types.BrowserHappyPathRequest{
		WorkspaceID:      "ws_browser",
		RunID:            "run_browser_guardrail",
		RequestedByActor: "actor_requester",
		SubjectPrincipal: "actor_subject",
		ActingAsActor:    "actor_subject",
		CorrelationID:    "corr_browser_guardrail",
		TargetURL:        "http://example.com",
	})
	if err == nil {
		t.Fatalf("expected guardrail error for non-https URL")
	}
}
