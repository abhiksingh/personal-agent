package mail

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/runtimepaths"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	CapabilityDraft          = "mail_draft"
	CapabilitySend           = "mail_send"
	CapabilityReply          = "mail_reply"
	CapabilityUnreadSummary  = "mail_unread_summary"
	envMailAutomationDryRun  = "PA_MAIL_AUTOMATION_DRY_RUN"
	transportMailAppleEvents = "mail_apple_events"
	transportMailDryRun      = "mail_dry_run"
	defaultMailSummaryLimit  = 5
	maxMailSummaryLimit      = 50
)

type Adapter struct {
	id     string
	dbPath string
}

func NewAdapter(id string) *Adapter {
	return NewAdapterWithDBPath(id, "")
}

func NewAdapterWithDBPath(id string, dbPath string) *Adapter {
	if id == "" {
		id = "mail.default"
	}
	return &Adapter{
		id:     id,
		dbPath: resolveMailAdapterDBPath(dbPath),
	}
}

func (a *Adapter) Metadata() connectorcontract.Metadata {
	return connectorcontract.Metadata{
		ID:          a.id,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Mail Connector",
		Version:     "0.3.0",
		Capabilities: []connectorcontract.CapabilityDescriptor{
			{Key: CapabilityDraft, Description: "Create a draft via macOS Mail automation"},
			{Key: CapabilitySend, Description: "Send mail via macOS Mail automation"},
			{Key: CapabilityReply, Description: "Send reply via macOS Mail automation"},
			{Key: CapabilityUnreadSummary, Description: "Summarize unread inbox messages from persisted mail events"},
		},
	}
}

func (a *Adapter) HealthCheck(_ context.Context) error {
	if isMailAutomationDryRunEnabled() {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("mail automation requires macOS (set %s=1 for dry-run)", envMailAutomationDryRun)
	}
	return nil
}

func (a *Adapter) ExecuteStep(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if step.CapabilityKey == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("step capability key is required")
	}

	return adapterscaffold.ExecuteStepWithCache("mail", execCtx, step, func(stepResultPath string) (connectorcontract.StepExecutionResult, error) {
		return a.executeUncached(ctx, execCtx, step, stepResultPath)
	})
}

func resolveMailAdapterDBPath(explicit string) string {
	trimmed := strings.TrimSpace(explicit)
	if trimmed != "" {
		return trimmed
	}
	if envPath := strings.TrimSpace(os.Getenv("PERSONAL_AGENT_DB")); envPath != "" {
		return envPath
	}
	defaultPath, err := runtimepaths.DefaultDBPath()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(defaultPath)
}
