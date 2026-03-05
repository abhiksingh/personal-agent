package transport

import (
	"context"
	"errors"
	"testing"
	"time"
)

type modelServiceStub struct {
	listResponse     ModelListResponse
	discoverResponse ModelDiscoverResponse
	addResponse      ModelCatalogEntryRecord
	removeResponse   ModelCatalogRemoveResponse
	policyResponse   ModelPolicyResponse
	resolveResponse  ModelResolveResponse
	simulateResponse ModelRouteSimulationResponse
	explainResponse  ModelRouteExplainResponse

	lastListReq     ModelListRequest
	lastDiscoverReq ModelDiscoverRequest
	lastAddReq      ModelCatalogAddRequest
	lastRemoveReq   ModelCatalogRemoveRequest
	lastToggleReq   ModelToggleRequest
	lastSelectReq   ModelSelectRequest
	lastPolicyReq   ModelPolicyRequest
	lastResolveReq  ModelResolveRequest
	lastSimulateReq ModelRouteSimulationRequest
	lastExplainReq  ModelRouteExplainRequest
}

func (s *modelServiceStub) ListModels(_ context.Context, request ModelListRequest) (ModelListResponse, error) {
	s.lastListReq = request
	return s.listResponse, nil
}

func (s *modelServiceStub) DiscoverModels(_ context.Context, request ModelDiscoverRequest) (ModelDiscoverResponse, error) {
	s.lastDiscoverReq = request
	return s.discoverResponse, nil
}

func (s *modelServiceStub) AddModel(_ context.Context, request ModelCatalogAddRequest) (ModelCatalogEntryRecord, error) {
	s.lastAddReq = request
	return s.addResponse, nil
}

func (s *modelServiceStub) RemoveModel(_ context.Context, request ModelCatalogRemoveRequest) (ModelCatalogRemoveResponse, error) {
	s.lastRemoveReq = request
	return s.removeResponse, nil
}

func (s *modelServiceStub) EnableModel(_ context.Context, request ModelToggleRequest) (ModelCatalogEntryRecord, error) {
	s.lastToggleReq = request
	return ModelCatalogEntryRecord{}, nil
}

func (s *modelServiceStub) DisableModel(_ context.Context, request ModelToggleRequest) (ModelCatalogEntryRecord, error) {
	s.lastToggleReq = request
	return ModelCatalogEntryRecord{}, nil
}

func (s *modelServiceStub) SelectModelRoute(_ context.Context, request ModelSelectRequest) (ModelRoutingPolicyRecord, error) {
	s.lastSelectReq = request
	return ModelRoutingPolicyRecord{}, nil
}

func (s *modelServiceStub) GetModelPolicy(_ context.Context, request ModelPolicyRequest) (ModelPolicyResponse, error) {
	s.lastPolicyReq = request
	return s.policyResponse, nil
}

func (s *modelServiceStub) ResolveModelRoute(_ context.Context, request ModelResolveRequest) (ModelResolveResponse, error) {
	s.lastResolveReq = request
	return s.resolveResponse, nil
}

func (s *modelServiceStub) SimulateModelRoute(_ context.Context, request ModelRouteSimulationRequest) (ModelRouteSimulationResponse, error) {
	s.lastSimulateReq = request
	return s.simulateResponse, nil
}

func (s *modelServiceStub) ExplainModelRoute(_ context.Context, request ModelRouteExplainRequest) (ModelRouteExplainResponse, error) {
	s.lastExplainReq = request
	return s.explainResponse, nil
}

func TestTransportModelDiscoverAddRemoveRoutes(t *testing.T) {
	models := &modelServiceStub{
		discoverResponse: ModelDiscoverResponse{
			WorkspaceID: "ws1",
			Results: []ModelDiscoverProviderResult{
				{
					Provider:      "ollama",
					ProviderReady: true,
					Success:       true,
					Models: []ModelDiscoverItem{
						{
							Provider:    "ollama",
							ModelKey:    "llama3.2",
							DisplayName: "llama3.2",
							Source:      "provider_discovery",
							InCatalog:   false,
							Enabled:     false,
						},
					},
				},
			},
		},
		addResponse: ModelCatalogEntryRecord{
			WorkspaceID: "ws1",
			Provider:    "ollama",
			ModelKey:    "llama3.2",
			Enabled:     false,
			UpdatedAt:   time.Now().UTC(),
		},
		removeResponse: ModelCatalogRemoveResponse{
			WorkspaceID: "ws1",
			Provider:    "ollama",
			ModelKey:    "llama3.2",
			Removed:     true,
			RemovedAt:   time.Now().UTC(),
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "model-token",
		Models:       models,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "model-token",
	})
	if err != nil {
		t.Fatalf("create model client: %v", err)
	}

	discover, err := client.DiscoverModels(context.Background(), ModelDiscoverRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
	}, "corr-model-discover")
	if err != nil {
		t.Fatalf("discover models: %v", err)
	}
	if discover.WorkspaceID != "ws1" || len(discover.Results) != 1 {
		t.Fatalf("unexpected discover payload: %+v", discover)
	}
	if models.lastDiscoverReq.Provider != "ollama" {
		t.Fatalf("unexpected discover request payload: %+v", models.lastDiscoverReq)
	}

	added, err := client.AddModel(context.Background(), ModelCatalogAddRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2",
		Enabled:     false,
	}, "corr-model-add")
	if err != nil {
		t.Fatalf("add model: %v", err)
	}
	if added.Provider != "ollama" || added.ModelKey != "llama3.2" {
		t.Fatalf("unexpected add payload: %+v", added)
	}
	if models.lastAddReq.ModelKey != "llama3.2" {
		t.Fatalf("unexpected add request payload: %+v", models.lastAddReq)
	}

	removed, err := client.RemoveModel(context.Background(), ModelCatalogRemoveRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2",
	}, "corr-model-remove")
	if err != nil {
		t.Fatalf("remove model: %v", err)
	}
	if !removed.Removed || removed.ModelKey != "llama3.2" {
		t.Fatalf("unexpected remove payload: %+v", removed)
	}
	if models.lastRemoveReq.ModelKey != "llama3.2" {
		t.Fatalf("unexpected remove request payload: %+v", models.lastRemoveReq)
	}
}

func TestTransportModelDiscoverAddRemoveRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "model-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "model-token",
	})
	if err != nil {
		t.Fatalf("create model client: %v", err)
	}

	_, err = client.DiscoverModels(context.Background(), ModelDiscoverRequest{
		WorkspaceID: "ws1",
	}, "corr-model-discover")
	if err == nil {
		t.Fatalf("expected discover models error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.AddModel(context.Background(), ModelCatalogAddRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2",
	}, "corr-model-add")
	if err == nil {
		t.Fatalf("expected add model error when service is not configured")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.RemoveModel(context.Background(), ModelCatalogRemoveRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2",
	}, "corr-model-remove")
	if err == nil {
		t.Fatalf("expected remove model error when service is not configured")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportModelRouteSimulationExplainRoutes(t *testing.T) {
	models := &modelServiceStub{
		simulateResponse: ModelRouteSimulationResponse{
			WorkspaceID:      "ws1",
			TaskClass:        "chat",
			PrincipalActorID: "actor.requester",
			SelectedProvider: "ollama",
			SelectedModelKey: "llama3.2",
			SelectedSource:   "task_class_policy",
			ReasonCodes:      []string{"task_class_policy_selected"},
			Decisions: []ModelRouteDecision{
				{
					Step:       "task_class_policy",
					Decision:   "selected",
					ReasonCode: "task_class_policy_selected",
					Provider:   "ollama",
					ModelKey:   "llama3.2",
				},
			},
			FallbackChain: []ModelRouteFallbackDecision{
				{
					Rank:       1,
					Provider:   "ollama",
					ModelKey:   "llama3.2",
					Selected:   true,
					ReasonCode: "fallback_selected",
				},
			},
		},
		explainResponse: ModelRouteExplainResponse{
			WorkspaceID:      "ws1",
			TaskClass:        "chat",
			PrincipalActorID: "actor.requester",
			SelectedProvider: "ollama",
			SelectedModelKey: "llama3.2",
			SelectedSource:   "task_class_policy",
			Summary:          "selected ollama/llama3.2 using task_class_policy",
			Explanations:     []string{"task_class_policy selected"},
			ReasonCodes:      []string{"task_class_policy_selected"},
			Decisions: []ModelRouteDecision{
				{
					Step:       "task_class_policy",
					Decision:   "selected",
					ReasonCode: "task_class_policy_selected",
					Provider:   "ollama",
					ModelKey:   "llama3.2",
				},
			},
			FallbackChain: []ModelRouteFallbackDecision{
				{
					Rank:       1,
					Provider:   "ollama",
					ModelKey:   "llama3.2",
					Selected:   true,
					ReasonCode: "fallback_selected",
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "model-token",
		Models:       models,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "model-token",
	})
	if err != nil {
		t.Fatalf("create model client: %v", err)
	}

	simulate, err := client.SimulateModelRoute(context.Background(), ModelRouteSimulationRequest{
		WorkspaceID:      "ws1",
		TaskClass:        "chat",
		PrincipalActorID: "actor.requester",
	}, "corr-model-route-simulate")
	if err != nil {
		t.Fatalf("simulate model route: %v", err)
	}
	if simulate.SelectedProvider != "ollama" || simulate.SelectedModelKey != "llama3.2" || simulate.SelectedSource != "task_class_policy" {
		t.Fatalf("unexpected model route simulation payload: %+v", simulate)
	}
	if models.lastSimulateReq.TaskClass != "chat" || models.lastSimulateReq.PrincipalActorID != "actor.requester" {
		t.Fatalf("unexpected model route simulate request payload: %+v", models.lastSimulateReq)
	}

	explain, err := client.ExplainModelRoute(context.Background(), ModelRouteExplainRequest{
		WorkspaceID:      "ws1",
		TaskClass:        "chat",
		PrincipalActorID: "actor.requester",
	}, "corr-model-route-explain")
	if err != nil {
		t.Fatalf("explain model route: %v", err)
	}
	if explain.SelectedProvider != "ollama" || explain.SelectedModelKey != "llama3.2" || explain.SelectedSource != "task_class_policy" {
		t.Fatalf("unexpected model route explain payload: %+v", explain)
	}
	if explain.Summary == "" || len(explain.ReasonCodes) == 0 || len(explain.Decisions) == 0 {
		t.Fatalf("expected explain payload fields to be populated, got %+v", explain)
	}
	if models.lastExplainReq.TaskClass != "chat" || models.lastExplainReq.PrincipalActorID != "actor.requester" {
		t.Fatalf("unexpected model route explain request payload: %+v", models.lastExplainReq)
	}
}

func TestTransportModelRouteSimulationExplainRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "model-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "model-token",
	})
	if err != nil {
		t.Fatalf("create model client: %v", err)
	}

	_, err = client.SimulateModelRoute(context.Background(), ModelRouteSimulationRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	}, "corr-model-route-simulate")
	if err == nil {
		t.Fatalf("expected simulate model route error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.ExplainModelRoute(context.Background(), ModelRouteExplainRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	}, "corr-model-route-explain")
	if err == nil {
		t.Fatalf("expected explain model route error when service is not configured")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}
