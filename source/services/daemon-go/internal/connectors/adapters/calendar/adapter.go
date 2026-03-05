package calendar

import (
	"context"
	"fmt"
	"runtime"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	CapabilityCreate             = "calendar_create"
	CapabilityUpdate             = "calendar_update"
	CapabilityCancel             = "calendar_cancel"
	capabilityExecuteProbe       = "__connector_execute_probe__"
	envCalendarAutomationDryRun  = "PA_CALENDAR_AUTOMATION_DRY_RUN"
	envCalendarDefaultName       = "PA_CALENDAR_DEFAULT_NAME"
	transportCalendarAppleEvents = "calendar_apple_events"
	transportCalendarDryRun      = "calendar_dry_run"
)

type Adapter struct {
	id string
}

func NewAdapter(id string) *Adapter {
	if id == "" {
		id = "calendar.default"
	}
	return &Adapter{id: id}
}

func (a *Adapter) Metadata() connectorcontract.Metadata {
	return connectorcontract.Metadata{
		ID:          a.id,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Calendar Connector",
		Version:     "0.3.1",
		Capabilities: []connectorcontract.CapabilityDescriptor{
			{Key: CapabilityCreate, Description: "Create event via macOS Calendar automation"},
			{Key: CapabilityUpdate, Description: "Update event via macOS Calendar automation"},
			{Key: CapabilityCancel, Description: "Cancel event via macOS Calendar automation"},
		},
	}
}

func (a *Adapter) HealthCheck(_ context.Context) error {
	if isCalendarAutomationDryRunEnabled() {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("calendar automation requires macOS (set %s=1 for dry-run)", envCalendarAutomationDryRun)
	}
	return nil
}

func (a *Adapter) ExecuteStep(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if step.CapabilityKey == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("step capability key is required")
	}
	if isCalendarExecuteProbeCapability(step.CapabilityKey) {
		return a.executeCalendarExecuteProbe(ctx)
	}

	return adapterscaffold.ExecuteStepWithCache("calendar", execCtx, step, func(stepResultPath string) (connectorcontract.StepExecutionResult, error) {
		return a.executeUncached(ctx, execCtx, step, stepResultPath)
	})
}
