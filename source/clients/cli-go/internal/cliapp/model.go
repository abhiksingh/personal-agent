package cliapp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providerconfig"
)

type modelListItem struct {
	WorkspaceID      string `json:"workspace_id"`
	Provider         string `json:"provider"`
	ModelKey         string `json:"model_key"`
	Enabled          bool   `json:"enabled"`
	ProviderReady    bool   `json:"provider_ready"`
	ProviderEndpoint string `json:"provider_endpoint,omitempty"`
}

type modelListResponse struct {
	WorkspaceID string          `json:"workspace_id"`
	Models      []modelListItem `json:"models"`
}

type modelResolveResponse struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
	Source      string `json:"source"`
	Notes       string `json:"notes,omitempty"`
}

type routeCandidate struct {
	provider string
	model    string
}

type providerStatus struct {
	Ready    bool
	Endpoint string
}

func providerStatusByProvider(ctx context.Context, providerStore *providerconfig.SQLiteStore, workspaceID string) (map[string]providerStatus, error) {
	configs, err := providerStore.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	statusByProvider := map[string]providerStatus{}
	for _, config := range configs {
		ready := true
		if providerconfig.ProviderRequiresAPIKey(config.Provider) {
			ready = strings.TrimSpace(config.APIKeySecretName) != "" && config.APIKeyConfigured
		}
		statusByProvider[config.Provider] = providerStatus{
			Ready:    ready,
			Endpoint: config.Endpoint,
		}
	}
	return statusByProvider, nil
}

func resolveModelRoute(
	ctx context.Context,
	modelStore *modelpolicy.SQLiteStore,
	providerStore *providerconfig.SQLiteStore,
	workspaceID string,
	taskClass string,
) (modelResolveResponse, error) {
	workspace := normalizeWorkspace(workspaceID)
	normalizedTaskClass := normalizeTaskClass(taskClass)

	entries, err := modelStore.ListCatalog(ctx, workspace, "")
	if err != nil {
		return modelResolveResponse{}, err
	}

	providerStatusMap, err := providerStatusByProvider(ctx, providerStore, workspace)
	if err != nil {
		return modelResolveResponse{}, err
	}

	candidates := make([]routeCandidate, 0)
	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		status, ok := providerStatusMap[entry.Provider]
		if !ok || !status.Ready {
			continue
		}
		candidates = append(candidates, routeCandidate{
			provider: entry.Provider,
			model:    entry.ModelKey,
		})
	}

	if len(candidates) == 0 {
		return modelResolveResponse{}, fmt.Errorf("no enabled models with ready provider configuration for workspace %q", workspace)
	}

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

	selected := candidates[0]
	source := "fallback_enabled"
	notes := ""

	taskPolicy, err := modelStore.GetRoutingPolicy(ctx, workspace, normalizedTaskClass)
	if err == nil {
		if containsCandidate(candidates, taskPolicy.Provider, taskPolicy.ModelKey) {
			selected = routeCandidate{provider: taskPolicy.Provider, model: taskPolicy.ModelKey}
			source = "task_class_policy"
			notes = ""
		} else {
			notes = fmt.Sprintf("task_class policy %s/%s unavailable; using fallback", taskPolicy.Provider, taskPolicy.ModelKey)
		}
	} else if !errors.Is(err, modelpolicy.ErrRoutingPolicyNotFound) {
		return modelResolveResponse{}, err
	}

	if source != "task_class_policy" && normalizedTaskClass != modelpolicy.TaskClassDefault {
		defaultPolicy, defaultErr := modelStore.GetRoutingPolicy(ctx, workspace, modelpolicy.TaskClassDefault)
		if defaultErr == nil && containsCandidate(candidates, defaultPolicy.Provider, defaultPolicy.ModelKey) {
			selected = routeCandidate{provider: defaultPolicy.Provider, model: defaultPolicy.ModelKey}
			source = "default_policy"
			notes = ""
		}
	}

	return modelResolveResponse{
		WorkspaceID: workspace,
		TaskClass:   normalizedTaskClass,
		Provider:    selected.provider,
		ModelKey:    selected.model,
		Source:      source,
		Notes:       notes,
	}, nil
}

func containsCandidate(candidates []routeCandidate, provider string, modelKey string) bool {
	for _, candidate := range candidates {
		if candidate.provider == provider && candidate.model == modelKey {
			return true
		}
	}
	return false
}

func providerPriority(provider string) int {
	switch provider {
	case providerconfig.ProviderOpenAI:
		return 0
	case providerconfig.ProviderAnthropic:
		return 1
	case providerconfig.ProviderGoogle:
		return 2
	case providerconfig.ProviderOllama:
		return 3
	default:
		return 99
	}
}

func normalizeTaskClass(taskClass string) string {
	trimmed := strings.ToLower(strings.TrimSpace(taskClass))
	if trimmed == "" {
		return modelpolicy.TaskClassDefault
	}
	return trimmed
}
