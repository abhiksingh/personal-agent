package connectorflow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/core/service/approval"
	"personalagent/runtime/internal/core/service/risk"
	"personalagent/runtime/internal/core/types"
	shared "personalagent/runtime/internal/shared/contracts"
)

type FinderHappyPathService struct {
	selector   ConnectorSelector
	classifier *risk.Classifier
	gate       *approval.Gate
}

func NewFinderHappyPathService(selector ConnectorSelector, classifier *risk.Classifier, gate *approval.Gate) *FinderHappyPathService {
	resolvedClassifier := classifier
	if resolvedClassifier == nil {
		resolvedClassifier = risk.NewClassifier()
	}
	resolvedGate := gate
	if resolvedGate == nil {
		resolvedGate = approval.NewGate(0.7)
	}

	return &FinderHappyPathService{
		selector:   selector,
		classifier: resolvedClassifier,
		gate:       resolvedGate,
	}
}

func (s *FinderHappyPathService) Execute(ctx context.Context, request types.FinderHappyPathRequest) (types.FinderHappyPathResult, error) {
	if s.selector == nil {
		return types.FinderHappyPathResult{}, fmt.Errorf("connector selector is required")
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return types.FinderHappyPathResult{}, fmt.Errorf("workspace id is required")
	}
	if strings.TrimSpace(request.RunID) == "" {
		return types.FinderHappyPathResult{}, fmt.Errorf("run id is required")
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

	targetPath := strings.TrimSpace(request.TargetPath)
	targetQuery := strings.TrimSpace(request.TargetQuery)
	searchRoot := strings.TrimSpace(request.SearchRootPath)
	findTrace := types.ConnectorStepTrace{}
	if targetQuery != "" || targetPath == "" {
		if targetQuery == "" && targetPath != "" {
			targetQuery = filepath.Base(targetPath)
		}
		if targetQuery == "" {
			return types.FinderHappyPathResult{}, fmt.Errorf("target path or target query is required")
		}
		findInput := map[string]any{
			"query": targetQuery,
		}
		if searchRoot != "" {
			findInput["root_path"] = searchRoot
		}
		var err error
		findTrace, err = s.executeFinderStep(ctx, request.PreferredAdapterID, execCtx, "finder_find", 0, "Find path", findInput)
		if err != nil {
			return types.FinderHappyPathResult{}, err
		}
		if targetPath == "" {
			targetPath = strings.TrimSpace(findTrace.Evidence["selected_path"])
		}
	}
	if strings.TrimSpace(targetPath) == "" {
		return types.FinderHappyPathResult{}, fmt.Errorf("finder did not resolve a target path")
	}

	listTrace, err := s.executeFinderStep(ctx, request.PreferredAdapterID, execCtx, "finder_list", 1, "List "+targetPath, map[string]any{
		"path": targetPath,
	})
	if err != nil {
		return types.FinderHappyPathResult{}, err
	}
	previewTrace, err := s.executeFinderStep(ctx, request.PreferredAdapterID, execCtx, "finder_preview", 2, "Preview delete "+targetPath, map[string]any{
		"path": targetPath,
	})
	if err != nil {
		return types.FinderHappyPathResult{}, err
	}

	riskClassification := s.classifier.ClassifyAction("delete file " + targetPath)
	gateDecision := s.gate.Evaluate(riskClassification)
	if gateDecision.RequireApproval && request.ApprovalPhrase != types.DestructiveApprovalPhrase {
		return types.FinderHappyPathResult{}, fmt.Errorf("destructive finder step requires exact approval phrase %q", types.DestructiveApprovalPhrase)
	}

	deleteTrace, err := s.executeFinderStep(ctx, request.PreferredAdapterID, execCtx, "finder_delete", 3, "Delete "+targetPath, map[string]any{
		"path": targetPath,
	})
	if err != nil {
		return types.FinderHappyPathResult{}, err
	}

	return types.FinderHappyPathResult{
		FindTrace:    findTrace,
		ListTrace:    listTrace,
		PreviewTrace: previewTrace,
		DeleteTrace:  deleteTrace,
		GateDecision: gateDecision,
	}, nil
}

func (s *FinderHappyPathService) executeFinderStep(
	ctx context.Context,
	preferredAdapterID string,
	execCtx connectorcontract.ExecutionContext,
	capability string,
	stepIndex int,
	stepName string,
	stepInput map[string]any,
) (types.ConnectorStepTrace, error) {
	adapter, err := s.selector.SelectByCapability(capability, preferredAdapterID)
	if err != nil {
		return types.ConnectorStepTrace{}, err
	}

	step := connectorcontract.TaskStep{
		ID:            fmt.Sprintf("finder-%s-%d", capability, stepIndex),
		RunID:         execCtx.RunID,
		StepIndex:     stepIndex,
		Name:          stepName,
		Status:        shared.TaskStepStatusPending,
		CapabilityKey: capability,
		Input:         cloneAnyMap(stepInput),
	}
	result, execErr := adapter.ExecuteStep(ctx, execCtx, step)
	if execErr != nil {
		return types.ConnectorStepTrace{}, execErr
	}
	if result.Status != shared.TaskStepStatusCompleted {
		return types.ConnectorStepTrace{}, fmt.Errorf("finder step %s did not complete", capability)
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

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
