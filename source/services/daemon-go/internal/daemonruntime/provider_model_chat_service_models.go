package daemonruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providercheck"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func (s *ProviderModelChatService) ListModels(ctx context.Context, request transport.ModelListRequest) (transport.ModelListResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	entries, err := s.container.ModelPolicyStore.ListCatalog(ctx, workspace, request.Provider)
	if err != nil {
		return transport.ModelListResponse{}, err
	}
	statusMap, err := providerStatusByProvider(ctx, s.container.ProviderConfigStore, workspace)
	if err != nil {
		return transport.ModelListResponse{}, err
	}

	items := make([]transport.ModelListItem, 0, len(entries))
	for _, entry := range entries {
		status := statusMap[entry.Provider]
		items = append(items, transport.ModelListItem{
			WorkspaceID:      workspace,
			Provider:         entry.Provider,
			ModelKey:         entry.ModelKey,
			Enabled:          entry.Enabled,
			ProviderReady:    status.Ready,
			ProviderEndpoint: status.Endpoint,
		})
	}
	return transport.ModelListResponse{
		WorkspaceID: workspace,
		Models:      items,
	}, nil
}

func (s *ProviderModelChatService) DiscoverModels(ctx context.Context, request transport.ModelDiscoverRequest) (transport.ModelDiscoverResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	filterProvider := strings.TrimSpace(request.Provider)

	configs := make([]providerconfig.Config, 0)
	if filterProvider != "" {
		normalizedProvider, err := providerconfig.NormalizeProvider(filterProvider)
		if err != nil {
			return transport.ModelDiscoverResponse{}, err
		}
		config, err := s.container.ProviderConfigStore.Get(ctx, workspace, normalizedProvider)
		if err != nil {
			return transport.ModelDiscoverResponse{}, err
		}
		configs = append(configs, config)
	} else {
		var err error
		configs, err = s.container.ProviderConfigStore.List(ctx, workspace)
		if err != nil {
			return transport.ModelDiscoverResponse{}, err
		}
		if len(configs) == 0 {
			return transport.ModelDiscoverResponse{}, fmt.Errorf("no provider configuration found for workspace %q", workspace)
		}
	}

	statusMap, err := providerStatusByProvider(ctx, s.container.ProviderConfigStore, workspace)
	if err != nil {
		return transport.ModelDiscoverResponse{}, err
	}

	catalogEntries, err := s.container.ModelPolicyStore.ListCatalog(ctx, workspace, "")
	if err != nil {
		return transport.ModelDiscoverResponse{}, err
	}
	catalogByKey := map[string]modelpolicy.CatalogEntry{}
	for _, entry := range catalogEntries {
		catalogByKey[catalogLookupKey(entry.Provider, entry.ModelKey)] = entry
	}

	results := make([]transport.ModelDiscoverProviderResult, 0, len(configs))
	for _, config := range configs {
		apiKey := ""
		if strings.TrimSpace(config.APIKeySecretName) != "" {
			_, secretValue, resolveErr := s.container.SecretResolver.ResolveSecret(ctx, workspace, config.APIKeySecretName)
			if resolveErr != nil {
				results = append(results, transport.ModelDiscoverProviderResult{
					Provider:         config.Provider,
					ProviderReady:    statusMap[config.Provider].Ready,
					ProviderEndpoint: statusMap[config.Provider].Endpoint,
					Success:          false,
					Message:          fmt.Sprintf("resolve secret %q failed: %v", config.APIKeySecretName, resolveErr),
					Models:           []transport.ModelDiscoverItem{},
				})
				continue
			}
			apiKey = secretValue
		}

		discovery, discoverErr := providercheck.Discover(ctx, s.providerHTTPClient(), providercheck.Request{
			Provider: config.Provider,
			Endpoint: config.Endpoint,
			APIKey:   apiKey,
		})

		items := make([]transport.ModelDiscoverItem, 0, len(discovery.Models))
		for _, modelKey := range discovery.Models {
			lookupKey := catalogLookupKey(config.Provider, modelKey)
			entry, inCatalog := catalogByKey[lookupKey]
			items = append(items, transport.ModelDiscoverItem{
				Provider:    config.Provider,
				ModelKey:    modelKey,
				DisplayName: modelKey,
				Source:      "provider_discovery",
				InCatalog:   inCatalog,
				Enabled:     inCatalog && entry.Enabled,
			})
		}

		message := strings.TrimSpace(discovery.Message)
		if discoverErr != nil && message == "" {
			message = discoverErr.Error()
		}
		results = append(results, transport.ModelDiscoverProviderResult{
			Provider:         config.Provider,
			ProviderReady:    statusMap[config.Provider].Ready,
			ProviderEndpoint: statusMap[config.Provider].Endpoint,
			Success:          discoverErr == nil,
			Message:          message,
			Models:           items,
		})
	}

	slices.SortFunc(results, func(left transport.ModelDiscoverProviderResult, right transport.ModelDiscoverProviderResult) int {
		leftPriority := providerPriority(left.Provider)
		rightPriority := providerPriority(right.Provider)
		if leftPriority != rightPriority {
			if leftPriority < rightPriority {
				return -1
			}
			return 1
		}
		switch {
		case left.Provider < right.Provider:
			return -1
		case left.Provider > right.Provider:
			return 1
		default:
			return 0
		}
	})

	return transport.ModelDiscoverResponse{
		WorkspaceID: workspace,
		Results:     results,
	}, nil
}

func (s *ProviderModelChatService) AddModel(ctx context.Context, request transport.ModelCatalogAddRequest) (transport.ModelCatalogEntryRecord, error) {
	entry, err := s.container.ModelPolicyStore.AddCatalogEntry(
		ctx,
		normalizeWorkspaceID(request.WorkspaceID),
		request.Provider,
		request.ModelKey,
		request.Enabled,
	)
	if err != nil {
		return transport.ModelCatalogEntryRecord{}, err
	}
	return modelCatalogEntryRecord(entry), nil
}

func (s *ProviderModelChatService) RemoveModel(ctx context.Context, request transport.ModelCatalogRemoveRequest) (transport.ModelCatalogRemoveResponse, error) {
	entry, err := s.container.ModelPolicyStore.RemoveCatalogEntry(
		ctx,
		normalizeWorkspaceID(request.WorkspaceID),
		request.Provider,
		request.ModelKey,
	)
	if err != nil {
		return transport.ModelCatalogRemoveResponse{}, err
	}
	return transport.ModelCatalogRemoveResponse{
		WorkspaceID: entry.WorkspaceID,
		Provider:    entry.Provider,
		ModelKey:    entry.ModelKey,
		Removed:     true,
		RemovedAt:   entry.UpdatedAt,
	}, nil
}

func (s *ProviderModelChatService) EnableModel(ctx context.Context, request transport.ModelToggleRequest) (transport.ModelCatalogEntryRecord, error) {
	entry, err := s.container.ModelPolicyStore.SetModelEnabled(ctx, normalizeWorkspaceID(request.WorkspaceID), request.Provider, request.ModelKey, true)
	if err != nil {
		return transport.ModelCatalogEntryRecord{}, err
	}
	return modelCatalogEntryRecord(entry), nil
}

func (s *ProviderModelChatService) DisableModel(ctx context.Context, request transport.ModelToggleRequest) (transport.ModelCatalogEntryRecord, error) {
	entry, err := s.container.ModelPolicyStore.SetModelEnabled(ctx, normalizeWorkspaceID(request.WorkspaceID), request.Provider, request.ModelKey, false)
	if err != nil {
		return transport.ModelCatalogEntryRecord{}, err
	}
	return modelCatalogEntryRecord(entry), nil
}

func (s *ProviderModelChatService) SelectModelRoute(ctx context.Context, request transport.ModelSelectRequest) (transport.ModelRoutingPolicyRecord, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := normalizeTaskClass(request.TaskClass)
	if taskClass == "chat" {
		entry, err := s.container.ModelPolicyStore.GetCatalogEntry(ctx, workspace, request.Provider, request.ModelKey)
		if err != nil {
			return transport.ModelRoutingPolicyRecord{}, err
		}
		if !entry.Enabled {
			return transport.ModelRoutingPolicyRecord{}, fmt.Errorf("model %s/%s is disabled", entry.Provider, entry.ModelKey)
		}
		if !isActionCapableChatModel(entry.Provider, entry.ModelKey) {
			return transport.ModelRoutingPolicyRecord{}, fmt.Errorf("model %s/%s is not action-capable for chat tool orchestration", entry.Provider, entry.ModelKey)
		}
	}

	policy, err := s.container.ModelPolicyStore.SetRoutingPolicy(ctx, workspace, taskClass, request.Provider, request.ModelKey)
	if err != nil {
		return transport.ModelRoutingPolicyRecord{}, err
	}
	return modelRoutingPolicyRecord(policy), nil
}

func (s *ProviderModelChatService) GetModelPolicy(ctx context.Context, request transport.ModelPolicyRequest) (transport.ModelPolicyResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := strings.TrimSpace(request.TaskClass)
	if taskClass != "" {
		policy, err := s.container.ModelPolicyStore.GetRoutingPolicy(ctx, workspace, taskClass)
		if err != nil {
			return transport.ModelPolicyResponse{}, err
		}
		record := modelRoutingPolicyRecord(policy)
		return transport.ModelPolicyResponse{
			WorkspaceID: workspace,
			Policy:      &record,
		}, nil
	}

	policies, err := s.container.ModelPolicyStore.ListRoutingPolicies(ctx, workspace)
	if err != nil {
		return transport.ModelPolicyResponse{}, err
	}
	records := make([]transport.ModelRoutingPolicyRecord, 0, len(policies))
	for _, policy := range policies {
		records = append(records, modelRoutingPolicyRecord(policy))
	}
	return transport.ModelPolicyResponse{
		WorkspaceID: workspace,
		Policies:    records,
	}, nil
}

func (s *ProviderModelChatService) ResolveModelRoute(ctx context.Context, request transport.ModelResolveRequest) (transport.ModelResolveResponse, error) {
	return s.resolveModelRoute(ctx, normalizeWorkspaceID(request.WorkspaceID), normalizeTaskClass(request.TaskClass))
}
