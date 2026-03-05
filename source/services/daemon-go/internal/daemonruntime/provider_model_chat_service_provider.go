package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"personalagent/runtime/internal/providercheck"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func (s *ProviderModelChatService) SetProvider(ctx context.Context, request transport.ProviderSetRequest) (transport.ProviderConfigRecord, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	normalizedProvider, err := providerconfig.NormalizeProvider(request.Provider)
	if err != nil {
		return transport.ProviderConfigRecord{}, err
	}

	existingConfig, existingErr := s.container.ProviderConfigStore.Get(ctx, workspace, normalizedProvider)
	if existingErr != nil && !errors.Is(existingErr, providerconfig.ErrProviderNotFound) {
		return transport.ProviderConfigRecord{}, existingErr
	}

	finalEndpoint := strings.TrimSpace(request.Endpoint)
	if finalEndpoint == "" {
		if existingErr == nil {
			finalEndpoint = existingConfig.Endpoint
		} else {
			finalEndpoint = providerconfig.DefaultEndpoint(normalizedProvider)
		}
	}

	finalSecretName := strings.TrimSpace(request.APIKeySecretName)
	if !request.ClearAPIKey && finalSecretName == "" && existingErr == nil {
		finalSecretName = existingConfig.APIKeySecretName
	}
	if request.ClearAPIKey {
		finalSecretName = ""
	}
	if providerconfig.ProviderRequiresAPIKey(normalizedProvider) && finalSecretName == "" {
		return transport.ProviderConfigRecord{}, fmt.Errorf("%s provider requires api key secret", normalizedProvider)
	}

	keychainService := ""
	keychainAccount := ""
	if finalSecretName != "" {
		ref, _, resolveErr := s.container.SecretResolver.ResolveSecret(ctx, workspace, finalSecretName)
		if resolveErr != nil {
			return transport.ProviderConfigRecord{}, fmt.Errorf("resolve provider api key secret %q: %w", finalSecretName, resolveErr)
		}
		keychainService = ref.Service
		keychainAccount = ref.Account
	}

	config, err := s.container.ProviderConfigStore.Upsert(ctx, providerconfig.UpsertInput{
		WorkspaceID:      workspace,
		Provider:         normalizedProvider,
		Endpoint:         finalEndpoint,
		APIKeySecretName: finalSecretName,
		KeychainService:  keychainService,
		KeychainAccount:  keychainAccount,
	})
	if err != nil {
		return transport.ProviderConfigRecord{}, err
	}
	return providerConfigRecord(config), nil
}

func (s *ProviderModelChatService) ListProviders(ctx context.Context, request transport.ProviderListRequest) (transport.ProviderListResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	configs, err := s.container.ProviderConfigStore.List(ctx, workspace)
	if err != nil {
		return transport.ProviderListResponse{}, err
	}

	providers := make([]transport.ProviderConfigRecord, 0, len(configs))
	for _, config := range configs {
		providers = append(providers, providerConfigRecord(config))
	}
	return transport.ProviderListResponse{
		WorkspaceID: workspace,
		Providers:   providers,
	}, nil
}

func (s *ProviderModelChatService) CheckProviders(ctx context.Context, request transport.ProviderCheckRequest) (transport.ProviderCheckResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	configs := make([]providerconfig.Config, 0)

	filterProvider := strings.TrimSpace(request.Provider)
	if filterProvider != "" {
		normalizedProvider, err := providerconfig.NormalizeProvider(filterProvider)
		if err != nil {
			return transport.ProviderCheckResponse{}, err
		}
		config, err := s.container.ProviderConfigStore.Get(ctx, workspace, normalizedProvider)
		if err != nil {
			return transport.ProviderCheckResponse{}, err
		}
		configs = append(configs, config)
	} else {
		var err error
		configs, err = s.container.ProviderConfigStore.List(ctx, workspace)
		if err != nil {
			return transport.ProviderCheckResponse{}, err
		}
		if len(configs) == 0 {
			return transport.ProviderCheckResponse{}, fmt.Errorf("no provider configuration found for workspace %q", workspace)
		}
	}

	allHealthy := true
	results := make([]transport.ProviderCheckItem, 0, len(configs))
	for _, config := range configs {
		apiKey := ""
		if strings.TrimSpace(config.APIKeySecretName) != "" {
			_, secretValue, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, config.APIKeySecretName)
			if err != nil {
				allHealthy = false
				results = append(results, transport.ProviderCheckItem{
					Provider: config.Provider,
					Endpoint: config.Endpoint,
					Success:  false,
					Message:  fmt.Sprintf("resolve secret %q failed: %v", config.APIKeySecretName, err),
				})
				continue
			}
			apiKey = secretValue
		}

		checkResult, err := providercheck.Check(ctx, s.providerHTTPClient(), providercheck.Request{
			Provider: config.Provider,
			Endpoint: config.Endpoint,
			APIKey:   apiKey,
		})
		item := transport.ProviderCheckItem{
			Provider:   config.Provider,
			Endpoint:   checkResult.Endpoint,
			Success:    err == nil,
			StatusCode: checkResult.StatusCode,
			LatencyMS:  checkResult.LatencyMS,
			Message:    checkResult.Message,
		}
		if err != nil {
			allHealthy = false
			if strings.TrimSpace(item.Message) == "" {
				item.Message = err.Error()
			}
		}
		results = append(results, item)
	}

	return transport.ProviderCheckResponse{
		WorkspaceID: workspace,
		Success:     allHealthy,
		Results:     results,
	}, nil
}

func (s *ProviderModelChatService) providerHTTPClient() *http.Client {
	if s != nil && s.providerProbeClient != nil {
		return s.providerProbeClient
	}
	return newDaemonRuntimeHTTPClient(defaultProviderProbeHTTPTimeout)
}

func providerStatusByProvider(ctx context.Context, store *providerconfig.SQLiteStore, workspaceID string) (map[string]providerStatus, error) {
	configs, err := store.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	status := map[string]providerStatus{}
	for _, config := range configs {
		ready := true
		if providerconfig.ProviderRequiresAPIKey(config.Provider) {
			ready = strings.TrimSpace(config.APIKeySecretName) != "" && config.APIKeyConfigured
		}
		status[config.Provider] = providerStatus{
			Ready:    ready,
			Endpoint: config.Endpoint,
		}
	}
	return status, nil
}
