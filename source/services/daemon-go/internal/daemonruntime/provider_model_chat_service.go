package daemonruntime

import (
	"net/http"

	"personalagent/runtime/internal/transport"
)

type ProviderModelChatService struct {
	container           *ServiceContainer
	providerProbeClient *http.Client
	chatStreamClient    *http.Client
}

type providerStatus struct {
	Ready    bool
	Endpoint string
}

type routeCandidate struct {
	provider string
	model    string
}

const (
	modelRouteDecisionSelected    = "selected"
	modelRouteDecisionSkipped     = "skipped"
	modelRouteDecisionUnavailable = "unavailable"

	modelRouteReasonPrincipalNotProvided   = "principal_context_not_provided"
	modelRouteReasonPrincipalActive        = "principal_context_active"
	modelRouteReasonPrincipalPolicyMissing = "principal_policy_not_configured"

	modelRouteReasonTaskPolicySelected    = "task_class_policy_selected"
	modelRouteReasonTaskPolicyMissing     = "task_class_policy_missing"
	modelRouteReasonTaskPolicyUnavailable = "task_class_policy_unavailable"

	modelRouteReasonDefaultPolicySelected    = "default_policy_selected"
	modelRouteReasonDefaultPolicyMissing     = "default_policy_missing"
	modelRouteReasonDefaultPolicyUnavailable = "default_policy_unavailable"

	modelRouteReasonFallbackSelected = "fallback_selected"

	modelRouteReasonTaskClassCapabilityFiltered = "task_class_capability_filtered"
	modelRouteReasonTaskClassCapabilityMissing  = "task_class_capability_missing"
)

type modelRouteAnalysis struct {
	workspaceID      string
	taskClass        string
	principalActorID string
	selected         routeCandidate
	selectedSource   string
	notes            string
	reasonCodes      []string
	decisions        []transport.ModelRouteDecision
	fallbackChain    []transport.ModelRouteFallbackDecision
}

var _ transport.ProviderService = (*ProviderModelChatService)(nil)
var _ transport.ModelService = (*ProviderModelChatService)(nil)
var _ transport.ChatService = (*ProviderModelChatService)(nil)

func NewProviderModelChatService(container *ServiceContainer) *ProviderModelChatService {
	return &ProviderModelChatService{
		container:           container,
		providerProbeClient: newDaemonRuntimeHTTPClient(defaultProviderProbeHTTPTimeout),
		chatStreamClient:    newDaemonRuntimeHTTPClient(defaultProviderChatHTTPTimeout),
	}
}
