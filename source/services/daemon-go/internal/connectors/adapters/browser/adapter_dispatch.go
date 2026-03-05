package browser

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

type browserSession struct {
	SessionID   string `json:"session_id"`
	WorkspaceID string `json:"workspace_id"`
	RunID       string `json:"run_id"`
	TargetURL   string `json:"target_url"`
	OpenedAt    string `json:"opened_at"`
	Transport   string `json:"transport,omitempty"`
	BrowserApp  string `json:"browser_app,omitempty"`
}

type browserStepInput struct {
	URL   string
	Query string
}

func (a *Adapter) executeUncached(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep, stepInput browserStepInput, stepResultPath string) (connectorcontract.StepExecutionResult, error) {
	workspaceRoot := adapterscaffold.WorkspaceRootFromStepResultPath(stepResultPath)
	sessionID := "browser-session-" + adapterhelpers.StableToken(execCtx.RunID, "run")
	sessionPath := filepath.Join(workspaceRoot, "sessions", sessionID+".json")
	browserApp := resolveBrowserAppName()
	provider := "safari-automation"
	if isBrowserAutomationDryRunEnabled() {
		provider = "safari-automation-dry-run"
	}

	switch step.CapabilityKey {
	case CapabilityOpen:
		operation, err := executeBrowserOperation(ctx, "open", stepInput.URL, browserApp)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "browser open automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute browser open: %w", err)
		}

		session := browserSession{
			SessionID:   sessionID,
			WorkspaceID: execCtx.WorkspaceID,
			RunID:       execCtx.RunID,
			TargetURL:   stepInput.URL,
			OpenedAt:    time.Now().UTC().Format(time.RFC3339Nano),
			Transport:   operation.Transport,
			BrowserApp:  browserApp,
		}
		if err := localstate.WriteJSONFile(sessionPath, session); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "browser session write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write browser session file: %w", err)
		}

		summary := strings.TrimSpace(operation.Summary)
		if summary == "" {
			summary = "browser page opened via Safari automation"
		}

		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   summary,
			Retryable: false,
			Evidence: map[string]string{
				"url":          stepInput.URL,
				"session_id":   sessionID,
				"session_path": sessionPath,
				"provider":     provider,
				"transport":    operation.Transport,
				"browser_app":  browserApp,
			},
			Output: map[string]any{
				"url":          stepInput.URL,
				"session_id":   sessionID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilityExtract:
		operation, err := executeBrowserOperation(ctx, "extract", stepInput.URL, browserApp)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "browser extract automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute browser extract: %w", err)
		}

		extractSnapshot := parseBrowserExtractSnapshot(stepInput.URL, operation.Summary)
		queryAnswer := resolveBrowserQueryAnswer(stepInput.Query, extractSnapshot.Content)
		contentPreview := browserContentPreview(extractSnapshot.Content)
		contentChars := len(extractSnapshot.Content)
		extractSummary := "browser content extracted via Safari automation"
		if strings.TrimSpace(queryAnswer) != "" {
			extractSummary = "browser content extracted and query answered via Safari automation"
		}
		evidence := map[string]string{
			"extraction_id":   "browser-extract-" + adapterhelpers.StableStepToken(execCtx, step),
			"title":           extractSnapshot.Title,
			"url":             extractSnapshot.URL,
			"content_chars":   strconv.Itoa(contentChars),
			"content_preview": contentPreview,
			"provider":        provider,
			"transport":       operation.Transport,
			"browser_app":     browserApp,
		}
		output := map[string]any{
			"title":           extractSnapshot.Title,
			"url":             extractSnapshot.URL,
			"content_text":    extractSnapshot.Content,
			"content_preview": contentPreview,
			"content_chars":   contentChars,
			"operation_id":    operation.OperationID,
		}
		if strings.TrimSpace(stepInput.Query) != "" {
			evidence["query"] = strings.TrimSpace(stepInput.Query)
			output["query"] = strings.TrimSpace(stepInput.Query)
		}
		if strings.TrimSpace(queryAnswer) != "" {
			evidence["query_answer"] = queryAnswer
			output["query_answer"] = queryAnswer
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   extractSummary,
			Retryable: false,
			Evidence:  evidence,
			Output:    output,
		}, nil
	case CapabilityClose:
		operation, err := executeBrowserOperation(ctx, "close", stepInput.URL, browserApp)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "browser close automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute browser close: %w", err)
		}
		if err := removeIfExists(sessionPath); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "browser session close failed",
				Retryable:   true,
				ErrorReason: "io_error",
				Evidence: map[string]string{
					"session_id":   sessionID,
					"session_path": sessionPath,
				},
			}, fmt.Errorf("remove browser session file: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "browser session closed via Safari automation",
			Retryable: false,
			Evidence: map[string]string{
				"close_id":     "browser-close-" + adapterhelpers.StableStepToken(execCtx, step),
				"closed":       "true",
				"session_id":   sessionID,
				"session_path": sessionPath,
				"provider":     provider,
				"transport":    operation.Transport,
				"browser_app":  browserApp,
			},
			Output: map[string]any{
				"closed":       true,
				"operation_id": operation.OperationID,
			},
		}, nil
	default:
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "unsupported browser capability",
			Retryable:   false,
			ErrorReason: "unsupported_capability",
		}, fmt.Errorf("unsupported browser capability: %s", step.CapabilityKey)
	}
}
