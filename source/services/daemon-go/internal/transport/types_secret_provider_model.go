package transport

import "time"

type SecretReferenceUpsertRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Backend     string `json:"backend"`
	Service     string `json:"service"`
	Account     string `json:"account"`
}

type SecretReferenceRecord struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Backend     string `json:"backend,omitempty"`
	Service     string `json:"service"`
	Account     string `json:"account"`
}

type SecretReferenceResponse struct {
	Reference     SecretReferenceRecord `json:"reference"`
	CorrelationID string                `json:"correlation_id"`
}

type SecretReferenceDeleteResponse struct {
	Reference     SecretReferenceRecord `json:"reference"`
	Deleted       bool                  `json:"deleted"`
	CorrelationID string                `json:"correlation_id"`
}

type ProviderSetRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	Provider         string `json:"provider"`
	Endpoint         string `json:"endpoint,omitempty"`
	APIKeySecretName string `json:"api_key_secret_name,omitempty"`
	ClearAPIKey      bool   `json:"clear_api_key"`
}

type ProviderListRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type ProviderConfigRecord struct {
	WorkspaceID      string    `json:"workspace_id"`
	Provider         string    `json:"provider"`
	Endpoint         string    `json:"endpoint"`
	APIKeySecretName string    `json:"api_key_secret_name,omitempty"`
	APIKeyConfigured bool      `json:"api_key_configured"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProviderListResponse struct {
	WorkspaceID string                 `json:"workspace_id"`
	Providers   []ProviderConfigRecord `json:"providers"`
}

type ProviderCheckRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider,omitempty"`
}

type ProviderCheckItem struct {
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Message    string `json:"message"`
}

type ProviderCheckResponse struct {
	WorkspaceID string              `json:"workspace_id"`
	Success     bool                `json:"success"`
	Results     []ProviderCheckItem `json:"results"`
}

type ModelListRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider,omitempty"`
}

type ModelListItem struct {
	WorkspaceID      string `json:"workspace_id"`
	Provider         string `json:"provider"`
	ModelKey         string `json:"model_key"`
	Enabled          bool   `json:"enabled"`
	ProviderReady    bool   `json:"provider_ready"`
	ProviderEndpoint string `json:"provider_endpoint,omitempty"`
}

type ModelListResponse struct {
	WorkspaceID string          `json:"workspace_id"`
	Models      []ModelListItem `json:"models"`
}

type ModelDiscoverRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider,omitempty"`
}

type ModelDiscoverItem struct {
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`
	InCatalog   bool   `json:"in_catalog"`
	Enabled     bool   `json:"enabled"`
}

type ModelDiscoverProviderResult struct {
	Provider         string              `json:"provider"`
	ProviderReady    bool                `json:"provider_ready"`
	ProviderEndpoint string              `json:"provider_endpoint,omitempty"`
	Success          bool                `json:"success"`
	Message          string              `json:"message,omitempty"`
	Models           []ModelDiscoverItem `json:"models"`
}

type ModelDiscoverResponse struct {
	WorkspaceID string                        `json:"workspace_id"`
	Results     []ModelDiscoverProviderResult `json:"results"`
}

type ModelCatalogAddRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
	Enabled     bool   `json:"enabled"`
}

type ModelCatalogRemoveRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
}

type ModelCatalogRemoveResponse struct {
	WorkspaceID string    `json:"workspace_id"`
	Provider    string    `json:"provider"`
	ModelKey    string    `json:"model_key"`
	Removed     bool      `json:"removed"`
	RemovedAt   time.Time `json:"removed_at"`
}

type ModelToggleRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
}

type ModelCatalogEntryRecord struct {
	WorkspaceID string    `json:"workspace_id"`
	Provider    string    `json:"provider"`
	ModelKey    string    `json:"model_key"`
	Enabled     bool      `json:"enabled"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ModelSelectRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
}

type ModelRoutingPolicyRecord struct {
	WorkspaceID string    `json:"workspace_id"`
	TaskClass   string    `json:"task_class"`
	Provider    string    `json:"provider"`
	ModelKey    string    `json:"model_key"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ModelPolicyRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class,omitempty"`
}

type ModelPolicyResponse struct {
	WorkspaceID string                     `json:"workspace_id"`
	Policy      *ModelRoutingPolicyRecord  `json:"policy,omitempty"`
	Policies    []ModelRoutingPolicyRecord `json:"policies,omitempty"`
}

type ModelResolveRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class,omitempty"`
}

type ModelResolveResponse struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class"`
	Provider    string `json:"provider"`
	ModelKey    string `json:"model_key"`
	Source      string `json:"source"`
	Notes       string `json:"notes,omitempty"`
}

type ModelRouteSimulationRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	TaskClass        string `json:"task_class,omitempty"`
	PrincipalActorID string `json:"principal_actor_id,omitempty"`
}

type ModelRouteDecision struct {
	Step       string `json:"step"`
	Decision   string `json:"decision"`
	ReasonCode string `json:"reason_code"`
	Provider   string `json:"provider,omitempty"`
	ModelKey   string `json:"model_key,omitempty"`
	Note       string `json:"note,omitempty"`
}

type ModelRouteFallbackDecision struct {
	Rank       int    `json:"rank"`
	Provider   string `json:"provider"`
	ModelKey   string `json:"model_key"`
	Selected   bool   `json:"selected"`
	ReasonCode string `json:"reason_code"`
}

type ModelRouteSimulationResponse struct {
	WorkspaceID      string                       `json:"workspace_id"`
	TaskClass        string                       `json:"task_class"`
	PrincipalActorID string                       `json:"principal_actor_id,omitempty"`
	SelectedProvider string                       `json:"selected_provider"`
	SelectedModelKey string                       `json:"selected_model_key"`
	SelectedSource   string                       `json:"selected_source"`
	Notes            string                       `json:"notes,omitempty"`
	ReasonCodes      []string                     `json:"reason_codes"`
	Decisions        []ModelRouteDecision         `json:"decisions"`
	FallbackChain    []ModelRouteFallbackDecision `json:"fallback_chain"`
}

type ModelRouteExplainRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	TaskClass        string `json:"task_class,omitempty"`
	PrincipalActorID string `json:"principal_actor_id,omitempty"`
}

type ModelRouteExplainResponse struct {
	WorkspaceID      string                       `json:"workspace_id"`
	TaskClass        string                       `json:"task_class"`
	PrincipalActorID string                       `json:"principal_actor_id,omitempty"`
	SelectedProvider string                       `json:"selected_provider"`
	SelectedModelKey string                       `json:"selected_model_key"`
	SelectedSource   string                       `json:"selected_source"`
	Summary          string                       `json:"summary"`
	Explanations     []string                     `json:"explanations"`
	ReasonCodes      []string                     `json:"reason_codes"`
	Decisions        []ModelRouteDecision         `json:"decisions"`
	FallbackChain    []ModelRouteFallbackDecision `json:"fallback_chain"`
}
