package finder

import (
	"context"
	"fmt"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	CapabilityFind    = "finder_find"
	CapabilityList    = "finder_list"
	CapabilityPreview = "finder_preview"
	CapabilityDelete  = "finder_delete"
)

const (
	defaultSearchMaxEntries = 8000
	defaultSearchMaxDepth   = 8
	maxFinderMatches        = 8
)

type Adapter struct {
	id string
}

func NewAdapter(id string) *Adapter {
	if id == "" {
		id = "finder.default"
	}
	return &Adapter{id: id}
}

func (a *Adapter) Metadata() connectorcontract.Metadata {
	return connectorcontract.Metadata{
		ID:          a.id,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Finder Connector",
		Version:     "0.3.0",
		Capabilities: []connectorcontract.CapabilityDescriptor{
			{Key: CapabilityFind, Description: "Find files by semantic query"},
			{Key: CapabilityList, Description: "List files in path"},
			{Key: CapabilityPreview, Description: "Preview file/folder details"},
			{Key: CapabilityDelete, Description: "Delete file path"},
		},
	}
}

func (a *Adapter) HealthCheck(_ context.Context) error {
	return nil
}

func (a *Adapter) ExecuteStep(_ context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if step.CapabilityKey == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("step capability key is required")
	}

	return adapterscaffold.ExecuteStepWithCache("finder", execCtx, step, func(_ string) (connectorcontract.StepExecutionResult, error) {
		return a.executeUncached(execCtx, step)
	})
}
