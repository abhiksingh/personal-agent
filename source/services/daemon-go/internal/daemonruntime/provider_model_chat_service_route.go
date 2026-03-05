package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func (s *ProviderModelChatService) SimulateModelRoute(ctx context.Context, request transport.ModelRouteSimulationRequest) (transport.ModelRouteSimulationResponse, error) {
	analysis, err := s.computeModelRouteAnalysis(
		ctx,
		normalizeWorkspaceID(request.WorkspaceID),
		normalizeTaskClass(request.TaskClass),
		strings.TrimSpace(request.PrincipalActorID),
	)
	if err != nil {
		return transport.ModelRouteSimulationResponse{}, err
	}
	return transport.ModelRouteSimulationResponse{
		WorkspaceID:      analysis.workspaceID,
		TaskClass:        analysis.taskClass,
		PrincipalActorID: analysis.principalActorID,
		SelectedProvider: analysis.selected.provider,
		SelectedModelKey: analysis.selected.model,
		SelectedSource:   analysis.selectedSource,
		Notes:            analysis.notes,
		ReasonCodes:      analysis.reasonCodes,
		Decisions:        analysis.decisions,
		FallbackChain:    analysis.fallbackChain,
	}, nil
}

func (s *ProviderModelChatService) ExplainModelRoute(ctx context.Context, request transport.ModelRouteExplainRequest) (transport.ModelRouteExplainResponse, error) {
	analysis, err := s.computeModelRouteAnalysis(
		ctx,
		normalizeWorkspaceID(request.WorkspaceID),
		normalizeTaskClass(request.TaskClass),
		strings.TrimSpace(request.PrincipalActorID),
	)
	if err != nil {
		return transport.ModelRouteExplainResponse{}, err
	}

	explanations := make([]string, 0, len(analysis.decisions)+2)
	for _, decision := range analysis.decisions {
		description := fmt.Sprintf("%s: %s (%s)", decision.Step, decision.Decision, decision.ReasonCode)
		if strings.TrimSpace(decision.Provider) != "" && strings.TrimSpace(decision.ModelKey) != "" {
			description += fmt.Sprintf(" %s/%s", decision.Provider, decision.ModelKey)
		}
		if strings.TrimSpace(decision.Note) != "" {
			description += fmt.Sprintf(" - %s", strings.TrimSpace(decision.Note))
		}
		explanations = append(explanations, description)
	}
	if strings.TrimSpace(analysis.notes) != "" {
		explanations = append(explanations, analysis.notes)
	}
	explanations = append(explanations, fmt.Sprintf("selected route: %s/%s via %s", analysis.selected.provider, analysis.selected.model, analysis.selectedSource))

	return transport.ModelRouteExplainResponse{
		WorkspaceID:      analysis.workspaceID,
		TaskClass:        analysis.taskClass,
		PrincipalActorID: analysis.principalActorID,
		SelectedProvider: analysis.selected.provider,
		SelectedModelKey: analysis.selected.model,
		SelectedSource:   analysis.selectedSource,
		Summary:          fmt.Sprintf("selected %s/%s using %s", analysis.selected.provider, analysis.selected.model, analysis.selectedSource),
		Explanations:     explanations,
		ReasonCodes:      analysis.reasonCodes,
		Decisions:        analysis.decisions,
		FallbackChain:    analysis.fallbackChain,
	}, nil
}

func (s *ProviderModelChatService) resolveModelRoute(ctx context.Context, workspaceID string, taskClass string) (transport.ModelResolveResponse, error) {
	analysis, err := s.computeModelRouteAnalysis(ctx, workspaceID, taskClass, "")
	if err != nil {
		return transport.ModelResolveResponse{}, err
	}
	return transport.ModelResolveResponse{
		WorkspaceID: analysis.workspaceID,
		TaskClass:   analysis.taskClass,
		Provider:    analysis.selected.provider,
		ModelKey:    analysis.selected.model,
		Source:      analysis.selectedSource,
		Notes:       analysis.notes,
	}, nil
}

func (s *ProviderModelChatService) computeModelRouteAnalysis(
	ctx context.Context,
	workspaceID string,
	taskClass string,
	principalActorID string,
) (modelRouteAnalysis, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	resolvedTaskClass := normalizeTaskClass(taskClass)
	analysis := modelRouteAnalysis{
		workspaceID:      workspace,
		taskClass:        resolvedTaskClass,
		principalActorID: strings.TrimSpace(principalActorID),
		reasonCodes:      make([]string, 0, 8),
		decisions:        make([]transport.ModelRouteDecision, 0, 8),
		fallbackChain:    make([]transport.ModelRouteFallbackDecision, 0),
	}

	if analysis.principalActorID == "" {
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "principal_context",
			Decision:   modelRouteDecisionSkipped,
			ReasonCode: modelRouteReasonPrincipalNotProvided,
		})
	} else {
		active, err := isActiveWorkspacePrincipal(ctx, s.container.DB, workspace, analysis.principalActorID)
		if err != nil {
			return modelRouteAnalysis{}, err
		}
		if !active {
			return modelRouteAnalysis{}, fmt.Errorf("principal_actor_id %q is not an active workspace principal", analysis.principalActorID)
		}
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "principal_context",
			Decision:   modelRouteDecisionSelected,
			ReasonCode: modelRouteReasonPrincipalActive,
			Note:       "principal context is active in workspace",
		})
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "principal_policy",
			Decision:   modelRouteDecisionSkipped,
			ReasonCode: modelRouteReasonPrincipalPolicyMissing,
			Note:       "principal-specific model policy is not configured; using workspace routing policy",
		})
	}

	entries, err := s.container.ModelPolicyStore.ListCatalog(ctx, workspace, "")
	if err != nil {
		return modelRouteAnalysis{}, err
	}
	statusMap, err := providerStatusByProvider(ctx, s.container.ProviderConfigStore, workspace)
	if err != nil {
		return modelRouteAnalysis{}, err
	}

	candidates := make([]routeCandidate, 0)
	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		providerStatus, ok := statusMap[entry.Provider]
		if !ok || !providerStatus.Ready {
			continue
		}
		candidates = append(candidates, routeCandidate{
			provider: entry.Provider,
			model:    entry.ModelKey,
		})
	}
	if len(candidates) == 0 {
		return modelRouteAnalysis{}, fmt.Errorf("no enabled models with ready provider configuration for workspace %q", workspace)
	}

	filteredCandidates, filteredOutCount := filterRouteCandidatesForTaskClass(resolvedTaskClass, candidates)
	if filteredOutCount > 0 {
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "task_class_capability",
			Decision:   modelRouteDecisionSelected,
			ReasonCode: modelRouteReasonTaskClassCapabilityFiltered,
			Note:       fmt.Sprintf("filtered %d route candidate(s) that are ineligible for task class %s", filteredOutCount, resolvedTaskClass),
		})
	}
	if len(filteredCandidates) == 0 {
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "task_class_capability",
			Decision:   modelRouteDecisionUnavailable,
			ReasonCode: modelRouteReasonTaskClassCapabilityMissing,
			Note:       fmt.Sprintf("no enabled ready model candidates satisfy task class %s capability requirements", resolvedTaskClass),
		})
		return modelRouteAnalysis{}, fmt.Errorf("no enabled action-capable models with ready provider configuration for task class %q in workspace %q", resolvedTaskClass, workspace)
	}
	candidates = filteredCandidates

	slices.SortFunc(candidates, func(left routeCandidate, right routeCandidate) int {
		leftPriority := providerPriority(left.provider)
		rightPriority := providerPriority(right.provider)
		if leftPriority != rightPriority {
			if leftPriority < rightPriority {
				return -1
			}
			return 1
		}
		if left.model < right.model {
			return -1
		}
		if left.model > right.model {
			return 1
		}
		return 0
	})

	for idx, candidate := range candidates {
		analysis.fallbackChain = append(analysis.fallbackChain, transport.ModelRouteFallbackDecision{
			Rank:       idx + 1,
			Provider:   candidate.provider,
			ModelKey:   candidate.model,
			Selected:   false,
			ReasonCode: "fallback_candidate_ready",
		})
	}

	selected := routeCandidate{}
	selectedSource := ""
	selectedSet := false
	notes := ""

	taskPolicy, err := s.container.ModelPolicyStore.GetRoutingPolicy(ctx, workspace, resolvedTaskClass)
	if err == nil {
		if containsCandidate(candidates, taskPolicy.Provider, taskPolicy.ModelKey) {
			selected = routeCandidate{provider: taskPolicy.Provider, model: taskPolicy.ModelKey}
			selectedSource = "task_class_policy"
			selectedSet = true
			appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
				Step:       "task_class_policy",
				Decision:   modelRouteDecisionSelected,
				ReasonCode: modelRouteReasonTaskPolicySelected,
				Provider:   taskPolicy.Provider,
				ModelKey:   taskPolicy.ModelKey,
			})
		} else {
			notes = fmt.Sprintf("task_class policy %s/%s unavailable; using fallback", taskPolicy.Provider, taskPolicy.ModelKey)
			appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
				Step:       "task_class_policy",
				Decision:   modelRouteDecisionUnavailable,
				ReasonCode: modelRouteReasonTaskPolicyUnavailable,
				Provider:   taskPolicy.Provider,
				ModelKey:   taskPolicy.ModelKey,
				Note:       "configured policy model/provider is not currently enabled and ready",
			})
		}
	} else if !errors.Is(err, modelpolicy.ErrRoutingPolicyNotFound) {
		return modelRouteAnalysis{}, err
	} else {
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "task_class_policy",
			Decision:   modelRouteDecisionSkipped,
			ReasonCode: modelRouteReasonTaskPolicyMissing,
		})
	}

	if !selectedSet && resolvedTaskClass != modelpolicy.TaskClassDefault {
		defaultPolicy, defaultErr := s.container.ModelPolicyStore.GetRoutingPolicy(ctx, workspace, modelpolicy.TaskClassDefault)
		if defaultErr == nil && containsCandidate(candidates, defaultPolicy.Provider, defaultPolicy.ModelKey) {
			selected = routeCandidate{provider: defaultPolicy.Provider, model: defaultPolicy.ModelKey}
			selectedSource = "default_policy"
			selectedSet = true
			appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
				Step:       "default_policy",
				Decision:   modelRouteDecisionSelected,
				ReasonCode: modelRouteReasonDefaultPolicySelected,
				Provider:   defaultPolicy.Provider,
				ModelKey:   defaultPolicy.ModelKey,
			})
		} else if defaultErr == nil {
			if notes == "" {
				notes = fmt.Sprintf("default policy %s/%s unavailable; using fallback", defaultPolicy.Provider, defaultPolicy.ModelKey)
			}
			appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
				Step:       "default_policy",
				Decision:   modelRouteDecisionUnavailable,
				ReasonCode: modelRouteReasonDefaultPolicyUnavailable,
				Provider:   defaultPolicy.Provider,
				ModelKey:   defaultPolicy.ModelKey,
				Note:       "default policy model/provider is not currently enabled and ready",
			})
		} else if errors.Is(defaultErr, modelpolicy.ErrRoutingPolicyNotFound) {
			appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
				Step:       "default_policy",
				Decision:   modelRouteDecisionSkipped,
				ReasonCode: modelRouteReasonDefaultPolicyMissing,
			})
		} else {
			return modelRouteAnalysis{}, defaultErr
		}
	}

	if !selectedSet {
		selected = candidates[0]
		selectedSource = "fallback_enabled"
		appendModelRouteDecision(&analysis, transport.ModelRouteDecision{
			Step:       "fallback_enabled",
			Decision:   modelRouteDecisionSelected,
			ReasonCode: modelRouteReasonFallbackSelected,
			Provider:   selected.provider,
			ModelKey:   selected.model,
		})
	}

	for idx := range analysis.fallbackChain {
		selectedFallback := analysis.fallbackChain[idx].Provider == selected.provider && analysis.fallbackChain[idx].ModelKey == selected.model
		analysis.fallbackChain[idx].Selected = selectedFallback
		if selectedFallback {
			analysis.fallbackChain[idx].ReasonCode = modelRouteReasonFallbackSelected
		}
	}

	analysis.workspaceID = workspace
	analysis.taskClass = resolvedTaskClass
	analysis.selected = selected
	analysis.selectedSource = selectedSource
	analysis.notes = notes
	return analysis, nil
}

func appendModelRouteDecision(analysis *modelRouteAnalysis, decision transport.ModelRouteDecision) {
	if analysis == nil {
		return
	}
	analysis.decisions = append(analysis.decisions, decision)
	if strings.TrimSpace(decision.ReasonCode) == "" {
		return
	}
	for _, existing := range analysis.reasonCodes {
		if existing == decision.ReasonCode {
			return
		}
	}
	analysis.reasonCodes = append(analysis.reasonCodes, decision.ReasonCode)
}

func containsCandidate(candidates []routeCandidate, provider string, modelKey string) bool {
	for _, candidate := range candidates {
		if candidate.provider == provider && candidate.model == modelKey {
			return true
		}
	}
	return false
}

func filterRouteCandidatesForTaskClass(taskClass string, candidates []routeCandidate) ([]routeCandidate, int) {
	if len(candidates) == 0 {
		return []routeCandidate{}, 0
	}
	filtered := make([]routeCandidate, 0, len(candidates))
	filteredOut := 0
	for _, candidate := range candidates {
		if taskClass == "chat" && !isActionCapableChatModel(candidate.provider, candidate.model) {
			filteredOut++
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered, filteredOut
}

func isActionCapableChatModel(provider string, modelKey string) bool {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	normalizedModel := strings.ToLower(strings.TrimSpace(modelKey))
	if normalizedModel == "" {
		return false
	}
	switch {
	case strings.Contains(normalizedModel, "lite"):
		return false
	case strings.Contains(normalizedModel, "haiku"):
		return false
	case normalizedProvider == providerconfig.ProviderOllama && normalizedModel == "mistral":
		return false
	default:
		return true
	}
}
