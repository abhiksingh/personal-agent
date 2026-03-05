package daemonruntime

import (
	"context"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

func TestModelRouteSimulationAndExplainabilityTaskPolicySelected(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)
	seedModelRoutePrincipalFixture(t, container, "ws1", "actor.requester")

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if _, err := service.AddModel(context.Background(), transport.ModelCatalogAddRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2-route",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("add model: %v", err)
	}
	if _, err := service.SelectModelRoute(context.Background(), transport.ModelSelectRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Provider:    "ollama",
		ModelKey:    "llama3.2-route",
	}); err != nil {
		t.Fatalf("select model route: %v", err)
	}

	simulate, err := service.SimulateModelRoute(context.Background(), transport.ModelRouteSimulationRequest{
		WorkspaceID:      "ws1",
		TaskClass:        "chat",
		PrincipalActorID: "actor.requester",
	})
	if err != nil {
		t.Fatalf("simulate model route: %v", err)
	}
	if simulate.SelectedProvider != "ollama" || simulate.SelectedModelKey != "llama3.2-route" {
		t.Fatalf("expected selected ollama/llama3.2-route, got %+v", simulate)
	}
	if simulate.SelectedSource != "task_class_policy" {
		t.Fatalf("expected selected source task_class_policy, got %s", simulate.SelectedSource)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonPrincipalActive) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonPrincipalActive, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonPrincipalPolicyMissing) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonPrincipalPolicyMissing, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonTaskPolicySelected) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonTaskPolicySelected, simulate.ReasonCodes)
	}
	if !hasSelectedFallback(simulate.FallbackChain, simulate.SelectedProvider, simulate.SelectedModelKey) {
		t.Fatalf("expected selected fallback chain entry for selected route, got %+v", simulate.FallbackChain)
	}

	resolve, err := service.ResolveModelRoute(context.Background(), transport.ModelResolveRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	})
	if err != nil {
		t.Fatalf("resolve model route: %v", err)
	}
	if resolve.Provider != simulate.SelectedProvider || resolve.ModelKey != simulate.SelectedModelKey || resolve.Source != simulate.SelectedSource {
		t.Fatalf("expected resolve to match simulate selected route, resolve=%+v simulate=%+v", resolve, simulate)
	}

	explain, err := service.ExplainModelRoute(context.Background(), transport.ModelRouteExplainRequest{
		WorkspaceID:      "ws1",
		TaskClass:        "chat",
		PrincipalActorID: "actor.requester",
	})
	if err != nil {
		t.Fatalf("explain model route: %v", err)
	}
	if explain.SelectedProvider != simulate.SelectedProvider || explain.SelectedModelKey != simulate.SelectedModelKey || explain.SelectedSource != simulate.SelectedSource {
		t.Fatalf("expected explain selected route to match simulate, explain=%+v simulate=%+v", explain, simulate)
	}
	if strings.TrimSpace(explain.Summary) == "" {
		t.Fatalf("expected non-empty explain summary")
	}
	if len(explain.Explanations) == 0 {
		t.Fatalf("expected non-empty explain explanations")
	}
	if !hasReasonCode(explain.ReasonCodes, modelRouteReasonTaskPolicySelected) {
		t.Fatalf("expected explain reason code %s, got %+v", modelRouteReasonTaskPolicySelected, explain.ReasonCodes)
	}
}

func TestModelRouteSimulationTaskPolicyUnavailableFallsBack(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if _, err := service.SelectModelRoute(context.Background(), transport.ModelSelectRequest{
		WorkspaceID: "ws1",
		TaskClass:   "automation",
		Provider:    "openai",
		ModelKey:    "gpt-4.1",
	}); err != nil {
		t.Fatalf("select model route: %v", err)
	}

	simulate, err := service.SimulateModelRoute(context.Background(), transport.ModelRouteSimulationRequest{
		WorkspaceID: "ws1",
		TaskClass:   "automation",
	})
	if err != nil {
		t.Fatalf("simulate model route: %v", err)
	}
	if simulate.SelectedSource != "fallback_enabled" {
		t.Fatalf("expected selected source fallback_enabled, got %s", simulate.SelectedSource)
	}
	if simulate.SelectedProvider != "ollama" {
		t.Fatalf("expected fallback provider ollama, got %+v", simulate)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonTaskPolicyUnavailable) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonTaskPolicyUnavailable, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonDefaultPolicyMissing) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonDefaultPolicyMissing, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonFallbackSelected) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonFallbackSelected, simulate.ReasonCodes)
	}

	taskPolicyUnavailable := false
	for _, decision := range simulate.Decisions {
		if decision.Step == "task_class_policy" &&
			decision.Decision == modelRouteDecisionUnavailable &&
			decision.ReasonCode == modelRouteReasonTaskPolicyUnavailable &&
			decision.Provider == "openai" &&
			decision.ModelKey == "gpt-4.1" {
			taskPolicyUnavailable = true
			break
		}
	}
	if !taskPolicyUnavailable {
		t.Fatalf("expected task_class_policy unavailable decision entry, got %+v", simulate.Decisions)
	}
	if !strings.Contains(simulate.Notes, "unavailable") {
		t.Fatalf("expected fallback notes to mention unavailable policy, got %q", simulate.Notes)
	}
	if !hasSelectedFallback(simulate.FallbackChain, simulate.SelectedProvider, simulate.SelectedModelKey) {
		t.Fatalf("expected fallback chain selected entry for chosen route, got %+v", simulate.FallbackChain)
	}
}

func TestModelRouteSimulationRejectsInactivePrincipal(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}

	_, err := service.SimulateModelRoute(context.Background(), transport.ModelRouteSimulationRequest{
		WorkspaceID:      "ws1",
		TaskClass:        "chat",
		PrincipalActorID: "actor.missing",
	})
	if err == nil {
		t.Fatalf("expected inactive principal error")
	}
	if !strings.Contains(err.Error(), "not an active workspace principal") {
		t.Fatalf("expected inactive principal error message, got %v", err)
	}
}

func TestModelRouteSelectRejectsNonActionCapableChatPolicy(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}

	_, err := service.SelectModelRoute(context.Background(), transport.ModelSelectRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Provider:    "ollama",
		ModelKey:    "mistral",
	})
	if err == nil {
		t.Fatalf("expected non-action-capable chat route selection to fail")
	}
	if !strings.Contains(err.Error(), "not action-capable") {
		t.Fatalf("expected action-capable selection error, got %v", err)
	}

	if _, err := service.SelectModelRoute(context.Background(), transport.ModelSelectRequest{
		WorkspaceID: "ws1",
		TaskClass:   "automation",
		Provider:    "ollama",
		ModelKey:    "mistral",
	}); err != nil {
		t.Fatalf("expected non-chat route selection to allow mistral, got %v", err)
	}
}

func TestModelRouteSimulationChatPolicyMisconfiguredFallsBackToActionCapableCandidate(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}

	if _, err := container.ModelPolicyStore.SetRoutingPolicy(context.Background(), "ws1", "chat", "ollama", "mistral"); err != nil {
		t.Fatalf("seed chat routing policy: %v", err)
	}

	simulate, err := service.SimulateModelRoute(context.Background(), transport.ModelRouteSimulationRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	})
	if err != nil {
		t.Fatalf("simulate model route: %v", err)
	}
	if simulate.SelectedSource != "fallback_enabled" || simulate.SelectedProvider != "ollama" || simulate.SelectedModelKey != "llama3.2" {
		t.Fatalf("expected fallback route selection to ollama/llama3.2, got %+v", simulate)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonTaskClassCapabilityFiltered) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonTaskClassCapabilityFiltered, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonTaskPolicyUnavailable) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonTaskPolicyUnavailable, simulate.ReasonCodes)
	}
	if !hasReasonCode(simulate.ReasonCodes, modelRouteReasonFallbackSelected) {
		t.Fatalf("expected reason code %s, got %+v", modelRouteReasonFallbackSelected, simulate.ReasonCodes)
	}
}

func TestModelRouteResolveChatFailsWithoutActionCapableCandidates(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service := NewProviderModelChatService(container)

	if _, err := service.SetProvider(context.Background(), transport.ProviderSetRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		Endpoint:    "http://127.0.0.1:11434",
	}); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if _, err := service.DisableModel(context.Background(), transport.ModelToggleRequest{
		WorkspaceID: "ws1",
		Provider:    "ollama",
		ModelKey:    "llama3.2",
	}); err != nil {
		t.Fatalf("disable llama3.2: %v", err)
	}

	_, err := service.ResolveModelRoute(context.Background(), transport.ModelResolveRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	})
	if err == nil {
		t.Fatalf("expected chat route resolve failure without action-capable candidates")
	}
	if !strings.Contains(err.Error(), "no enabled action-capable models") {
		t.Fatalf("unexpected chat route resolve error: %v", err)
	}

	automationResolve, err := service.ResolveModelRoute(context.Background(), transport.ModelResolveRequest{
		WorkspaceID: "ws1",
		TaskClass:   "automation",
	})
	if err != nil {
		t.Fatalf("expected non-chat route to resolve with remaining model, got %v", err)
	}
	if automationResolve.Provider != "ollama" || automationResolve.ModelKey != "mistral" {
		t.Fatalf("expected automation route fallback to ollama/mistral, got %+v", automationResolve)
	}
}

func seedModelRoutePrincipalFixture(t *testing.T, container *ServiceContainer, workspaceID string, actorID string) {
	t.Helper()
	if container == nil || container.DB == nil {
		t.Fatalf("test container db is required")
	}
	tx, err := container.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	if err := ensureDelegationWorkspace(context.Background(), tx, workspaceID, nowText); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	if err := ensureDelegationActorPrincipal(context.Background(), tx, workspaceID, actorID, nowText); err != nil {
		t.Fatalf("ensure actor principal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}
}

func hasReasonCode(reasonCodes []string, reasonCode string) bool {
	for _, candidate := range reasonCodes {
		if strings.TrimSpace(candidate) == strings.TrimSpace(reasonCode) {
			return true
		}
	}
	return false
}

func hasSelectedFallback(chain []transport.ModelRouteFallbackDecision, provider string, modelKey string) bool {
	for _, item := range chain {
		if item.Provider == provider && item.ModelKey == modelKey && item.Selected {
			return true
		}
	}
	return false
}
