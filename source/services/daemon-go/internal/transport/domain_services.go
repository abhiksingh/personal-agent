package transport

import "context"

type ProviderService interface {
	SetProvider(ctx context.Context, request ProviderSetRequest) (ProviderConfigRecord, error)
	ListProviders(ctx context.Context, request ProviderListRequest) (ProviderListResponse, error)
	CheckProviders(ctx context.Context, request ProviderCheckRequest) (ProviderCheckResponse, error)
}

type ModelService interface {
	ListModels(ctx context.Context, request ModelListRequest) (ModelListResponse, error)
	DiscoverModels(ctx context.Context, request ModelDiscoverRequest) (ModelDiscoverResponse, error)
	AddModel(ctx context.Context, request ModelCatalogAddRequest) (ModelCatalogEntryRecord, error)
	RemoveModel(ctx context.Context, request ModelCatalogRemoveRequest) (ModelCatalogRemoveResponse, error)
	EnableModel(ctx context.Context, request ModelToggleRequest) (ModelCatalogEntryRecord, error)
	DisableModel(ctx context.Context, request ModelToggleRequest) (ModelCatalogEntryRecord, error)
	SelectModelRoute(ctx context.Context, request ModelSelectRequest) (ModelRoutingPolicyRecord, error)
	GetModelPolicy(ctx context.Context, request ModelPolicyRequest) (ModelPolicyResponse, error)
	ResolveModelRoute(ctx context.Context, request ModelResolveRequest) (ModelResolveResponse, error)
	SimulateModelRoute(ctx context.Context, request ModelRouteSimulationRequest) (ModelRouteSimulationResponse, error)
	ExplainModelRoute(ctx context.Context, request ModelRouteExplainRequest) (ModelRouteExplainResponse, error)
}

type ChatService interface {
	ChatTurn(ctx context.Context, request ChatTurnRequest, correlationID string, onToken func(delta string)) (ChatTurnResponse, error)
}

type AgentService interface {
	RunAgent(ctx context.Context, request AgentRunRequest) (AgentRunResponse, error)
	ApproveAgent(ctx context.Context, request AgentApproveRequest) (AgentRunResponse, error)
}

type DelegationService interface {
	GrantDelegation(ctx context.Context, request DelegationGrantRequest) (DelegationRuleRecord, error)
	ListDelegations(ctx context.Context, request DelegationListRequest) (DelegationListResponse, error)
	RevokeDelegation(ctx context.Context, request DelegationRevokeRequest) (DelegationRevokeResponse, error)
	CheckDelegation(ctx context.Context, request DelegationCheckRequest) (DelegationCheckResponse, error)
	UpsertCapabilityGrant(ctx context.Context, request CapabilityGrantUpsertRequest) (CapabilityGrantRecord, error)
	ListCapabilityGrants(ctx context.Context, request CapabilityGrantListRequest) (CapabilityGrantListResponse, error)
}

type CommService interface {
	SendComm(ctx context.Context, request CommSendRequest) (CommSendResponse, error)
	ListCommAttempts(ctx context.Context, request CommAttemptsRequest) (CommAttemptsResponse, error)
	SetCommPolicy(ctx context.Context, request CommPolicySetRequest) (CommPolicyRecord, error)
	ListCommPolicies(ctx context.Context, request CommPolicyListRequest) (CommPolicyListResponse, error)
	ListCommWebhookReceipts(ctx context.Context, request CommWebhookReceiptListRequest) (CommWebhookReceiptListResponse, error)
	ListCommIngestReceipts(ctx context.Context, request CommIngestReceiptListRequest) (CommIngestReceiptListResponse, error)
	IngestMessages(ctx context.Context, request MessagesIngestRequest) (MessagesIngestResponse, error)
	IngestMailRuleEvent(ctx context.Context, request MailRuleIngestRequest) (MailRuleIngestResponse, error)
	IngestCalendarChange(ctx context.Context, request CalendarChangeIngestRequest) (CalendarChangeIngestResponse, error)
	IngestBrowserEvent(ctx context.Context, request BrowserEventIngestRequest) (BrowserEventIngestResponse, error)
}

type TwilioChannelService interface {
	SetTwilioChannel(ctx context.Context, request TwilioSetRequest) (TwilioConfigRecord, error)
	GetTwilioChannel(ctx context.Context, request TwilioGetRequest) (TwilioConfigRecord, error)
	CheckTwilioChannel(ctx context.Context, request TwilioCheckRequest) (TwilioCheckResponse, error)
	ExecuteTwilioSMSChatTurn(ctx context.Context, request TwilioSMSChatTurnRequest) (TwilioSMSChatTurn, error)
	StartTwilioCall(ctx context.Context, request TwilioStartCallRequest) (TwilioStartCallResponse, error)
	ListTwilioCallStatus(ctx context.Context, request TwilioCallStatusRequest) (TwilioCallStatusResponse, error)
	ListTwilioTranscript(ctx context.Context, request TwilioTranscriptRequest) (TwilioTranscriptResponse, error)
	IngestTwilioSMS(ctx context.Context, request TwilioIngestSMSRequest) (TwilioIngestSMSResponse, error)
	IngestTwilioVoice(ctx context.Context, request TwilioIngestVoiceRequest) (TwilioIngestVoiceResponse, error)
	ServeTwilioWebhook(ctx context.Context, request TwilioWebhookServeRequest) (TwilioWebhookServeResponse, error)
	ReplayTwilioWebhook(ctx context.Context, request TwilioWebhookReplayRequest) (TwilioWebhookReplayResponse, error)
}

type CloudflaredConnectorService interface {
	CloudflaredVersion(ctx context.Context, request CloudflaredVersionRequest) (CloudflaredVersionResponse, error)
	CloudflaredExec(ctx context.Context, request CloudflaredExecRequest) (CloudflaredExecResponse, error)
}
