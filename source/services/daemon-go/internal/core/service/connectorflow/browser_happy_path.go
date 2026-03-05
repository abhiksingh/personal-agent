package connectorflow

import (
	"context"
	"fmt"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type BrowserHappyPathService struct {
	selector ConnectorSelector
}

func NewBrowserHappyPathService(selector ConnectorSelector) *BrowserHappyPathService {
	return &BrowserHappyPathService{selector: selector}
}

func (s *BrowserHappyPathService) Execute(ctx context.Context, request types.BrowserHappyPathRequest) (types.BrowserHappyPathResult, error) {
	if s.selector == nil {
		return types.BrowserHappyPathResult{}, fmt.Errorf("connector selector is required")
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return types.BrowserHappyPathResult{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(request.RunID) == "" {
		return types.BrowserHappyPathResult{}, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(request.TargetURL) == "" {
		return types.BrowserHappyPathResult{}, fmt.Errorf("target URL is required")
	}

	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID:      request.WorkspaceID,
		RunID:            request.RunID,
		CorrelationID:    request.CorrelationID,
		RequestedByActor: request.RequestedByActor,
		SubjectPrincipal: request.SubjectPrincipal,
		ActingAsActor:    request.ActingAsActor,
		SourceChannel:    "app_chat",
	}

	openTrace, err := s.executeBrowserStep(ctx, request.PreferredAdapterID, execCtx, "browser_open", 0, "Open "+request.TargetURL, request.TargetURL, "")
	if err != nil {
		return types.BrowserHappyPathResult{}, err
	}
	extractTrace, err := s.executeBrowserStep(ctx, request.PreferredAdapterID, execCtx, "browser_extract", 1, "Extract summary "+request.TargetURL, request.TargetURL, "summarize key facts from this page")
	if err != nil {
		return types.BrowserHappyPathResult{}, err
	}
	closeTrace, err := s.executeBrowserStep(ctx, request.PreferredAdapterID, execCtx, "browser_close", 2, "Close "+request.TargetURL, request.TargetURL, "")
	if err != nil {
		return types.BrowserHappyPathResult{}, err
	}

	return types.BrowserHappyPathResult{
		OpenTrace:    openTrace,
		ExtractTrace: extractTrace,
		CloseTrace:   closeTrace,
	}, nil
}

func (s *BrowserHappyPathService) executeBrowserStep(
	ctx context.Context,
	preferredAdapterID string,
	execCtx connectorcontract.ExecutionContext,
	capability string,
	stepIndex int,
	stepName string,
	targetURL string,
	query string,
) (types.ConnectorStepTrace, error) {
	adapter, err := s.selector.SelectByCapability(capability, preferredAdapterID)
	if err != nil {
		return types.ConnectorStepTrace{}, err
	}

	step := connectorcontract.TaskStep{
		ID:            fmt.Sprintf("browser-%s-%d", capability, stepIndex),
		RunID:         execCtx.RunID,
		StepIndex:     stepIndex,
		Name:          stepName,
		Status:        shared.TaskStepStatusPending,
		CapabilityKey: capability,
		Input: map[string]any{
			"url": targetURL,
		},
	}
	if capability == "browser_extract" && strings.TrimSpace(query) != "" {
		step.Input["query"] = strings.TrimSpace(query)
	}
	result, execErr := adapter.ExecuteStep(ctx, execCtx, step)
	if execErr != nil {
		return types.ConnectorStepTrace{}, execErr
	}
	if result.Status != shared.TaskStepStatusCompleted {
		return types.ConnectorStepTrace{}, fmt.Errorf("browser step %s did not complete", capability)
	}

	trace := types.ConnectorStepTrace{
		StepName:      step.Name,
		CapabilityKey: capability,
		AdapterID:     adapter.Metadata().ID,
		Summary:       result.Summary,
		Evidence:      map[string]string{},
	}
	for key, value := range result.Evidence {
		trace.Evidence[key] = value
	}
	return trace, nil
}
