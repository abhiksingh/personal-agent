package browser

import (
	"context"
	"testing"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

func TestExecuteStepOpenExtractCloseWithDryRunAndIdempotentCache(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")

	adapter := NewAdapter("browser.test")
	baseCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws_browser",
		TaskID:      "task-1",
		RunID:       "run-1",
	}

	openCtx := baseCtx
	openCtx.StepID = "step-open-1"
	openStep := connectorcontract.TaskStep{
		ID:            "step-open-1",
		CapabilityKey: CapabilityOpen,
		Name:          "Open browser URL",
		Input: map[string]any{
			"url": "https://example.com",
		},
	}
	openFirst, err := adapter.ExecuteStep(context.Background(), openCtx, openStep)
	if err != nil {
		t.Fatalf("execute open step: %v", err)
	}
	openSecond, err := adapter.ExecuteStep(context.Background(), openCtx, openStep)
	if err != nil {
		t.Fatalf("execute open step second time: %v", err)
	}
	if openFirst.Evidence["session_id"] == "" {
		t.Fatalf("expected session_id evidence")
	}
	if openFirst.Evidence["session_id"] != openSecond.Evidence["session_id"] {
		t.Fatalf("expected idempotent session_id, got %q vs %q", openFirst.Evidence["session_id"], openSecond.Evidence["session_id"])
	}
	if openFirst.Evidence["provider"] != "safari-automation-dry-run" {
		t.Fatalf("expected safari-automation-dry-run provider, got %q", openFirst.Evidence["provider"])
	}

	extractCtx := baseCtx
	extractCtx.StepID = "step-extract-1"
	extractStep := connectorcontract.TaskStep{
		ID:            "step-extract-1",
		CapabilityKey: CapabilityExtract,
		Name:          "Extract browser URL",
		Input: map[string]any{
			"url":   "https://example.com",
			"query": "what is this page about?",
		},
	}
	extractResult, err := adapter.ExecuteStep(context.Background(), extractCtx, extractStep)
	if err != nil {
		t.Fatalf("execute extract step: %v", err)
	}
	if extractResult.Evidence["title"] == "" {
		t.Fatalf("expected extract title evidence")
	}
	if extractResult.Evidence["content_chars"] == "" {
		t.Fatalf("expected deep extract content_chars evidence")
	}
	if extractResult.Evidence["query_answer"] == "" {
		t.Fatalf("expected query_answer evidence")
	}
	if extractResult.Evidence["provider"] != "safari-automation-dry-run" {
		t.Fatalf("expected safari-automation-dry-run extract provider, got %q", extractResult.Evidence["provider"])
	}
	contentText, ok := extractResult.Output["content_text"].(string)
	if !ok || contentText == "" {
		t.Fatalf("expected content_text in extract output, got %#v", extractResult.Output["content_text"])
	}
	queryAnswer, ok := extractResult.Output["query_answer"].(string)
	if !ok || queryAnswer == "" {
		t.Fatalf("expected query_answer in extract output, got %#v", extractResult.Output["query_answer"])
	}

	closeCtx := baseCtx
	closeCtx.StepID = "step-close-1"
	closeStep := connectorcontract.TaskStep{
		ID:            "step-close-1",
		CapabilityKey: CapabilityClose,
		Name:          "Close browser URL",
		Input: map[string]any{
			"url": "https://example.com",
		},
	}
	closeResult, err := adapter.ExecuteStep(context.Background(), closeCtx, closeStep)
	if err != nil {
		t.Fatalf("execute close step: %v", err)
	}
	if closeResult.Evidence["closed"] != "true" {
		t.Fatalf("expected close evidence closed=true")
	}
}

func TestExecuteStepRejectsNonLoopbackHTTPURL(t *testing.T) {
	t.Setenv("PA_CONNECTOR_DATA_DIR", t.TempDir())

	adapter := NewAdapter("browser.test")
	_, err := adapter.ExecuteStep(context.Background(), connectorcontract.ExecutionContext{
		WorkspaceID: "ws_browser",
		StepID:      "step-http-nonloopback",
	}, connectorcontract.TaskStep{
		ID:            "step-http-nonloopback",
		CapabilityKey: CapabilityOpen,
		Name:          "Open browser URL",
		Input: map[string]any{
			"url": "http://example.com",
		},
	})
	if err == nil {
		t.Fatalf("expected guardrail error for non-loopback http URL")
	}
}
