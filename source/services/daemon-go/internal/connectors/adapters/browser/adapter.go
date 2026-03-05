package browser

import (
	"context"
	"fmt"
	"runtime"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	CapabilityOpen             = "browser_open"
	CapabilityExtract          = "browser_extract"
	CapabilityClose            = "browser_close"
	envBrowserAutomationDryRun = "PA_BROWSER_AUTOMATION_DRY_RUN"
	envBrowserAppName          = "PA_BROWSER_APP_NAME"
	transportBrowserSafari     = "safari_apple_events"
	transportBrowserDryRun     = "safari_dry_run"
	browserExtractPrefix       = "pa_browser_extract::"
	maxBrowserExtractChars     = 12000
)

type Adapter struct {
	id string
}

func NewAdapter(id string) *Adapter {
	if id == "" {
		id = "browser.default"
	}
	return &Adapter{id: id}
}

func (a *Adapter) Metadata() connectorcontract.Metadata {
	return connectorcontract.Metadata{
		ID:          a.id,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Browser Connector",
		Version:     "0.3.0",
		Capabilities: []connectorcontract.CapabilityDescriptor{
			{Key: CapabilityOpen, Description: "Open URL through Safari automation"},
			{Key: CapabilityExtract, Description: "Extract page context through Safari automation"},
			{Key: CapabilityClose, Description: "Close Safari browser tab/session"},
		},
	}
}

func (a *Adapter) HealthCheck(_ context.Context) error {
	if isBrowserAutomationDryRunEnabled() {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("browser automation requires macOS (set %s=1 for dry-run)", envBrowserAutomationDryRun)
	}
	return nil
}

func (a *Adapter) ExecuteStep(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if step.CapabilityKey == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("step capability key is required")
	}

	stepInput, err := resolveBrowserStepInput(step)
	if err != nil {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "browser URL missing",
			Retryable:   false,
			ErrorReason: "invalid_url",
		}, err
	}
	if guardErr := enforceURLGuardrails(stepInput.URL); guardErr != nil {
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "browser guardrail denied URL",
			Retryable:   false,
			ErrorReason: "guardrail_denied",
			Evidence: map[string]string{
				"url": stepInput.URL,
			},
		}, guardErr
	}

	return adapterscaffold.ExecuteStepWithCache("browser", execCtx, step, func(stepResultPath string) (connectorcontract.StepExecutionResult, error) {
		return a.executeUncached(ctx, execCtx, step, stepInput, stepResultPath)
	})
}
