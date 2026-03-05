package transport

import "context"

type AutomationService interface {
	CreateAutomation(ctx context.Context, request AutomationCreateRequest) (AutomationTriggerRecord, error)
	ListAutomation(ctx context.Context, request AutomationListRequest) (AutomationListResponse, error)
	ListAutomationFireHistory(ctx context.Context, request AutomationFireHistoryRequest) (AutomationFireHistoryResponse, error)
	UpdateAutomation(ctx context.Context, request AutomationUpdateRequest) (AutomationUpdateResponse, error)
	DeleteAutomation(ctx context.Context, request AutomationDeleteRequest) (AutomationDeleteResponse, error)
	RunAutomationSchedule(ctx context.Context, request AutomationRunScheduleRequest) (AutomationRunScheduleResponse, error)
	RunAutomationCommEvent(ctx context.Context, request AutomationRunCommEventRequest) (AutomationRunCommEventResponse, error)
	AutomationCommTriggerMetadata(ctx context.Context, request AutomationCommTriggerMetadataRequest) (AutomationCommTriggerMetadataResponse, error)
	AutomationCommTriggerValidate(ctx context.Context, request AutomationCommTriggerValidateRequest) (AutomationCommTriggerValidateResponse, error)
}

type InspectService interface {
	InspectRun(ctx context.Context, request InspectRunRequest) (InspectRunResponse, error)
	InspectTranscript(ctx context.Context, request InspectTranscriptRequest) (InspectTranscriptResponse, error)
	InspectMemory(ctx context.Context, request InspectMemoryRequest) (InspectMemoryResponse, error)
	QueryInspectLogs(ctx context.Context, request InspectLogQueryRequest) (InspectLogQueryResponse, error)
	StreamInspectLogs(ctx context.Context, request InspectLogStreamRequest) (InspectLogStreamResponse, error)
}

type RetentionService interface {
	PurgeRetention(ctx context.Context, request RetentionPurgeRequest) (RetentionPurgeResponse, error)
	CompactRetentionMemory(ctx context.Context, request RetentionCompactMemoryRequest) (RetentionCompactMemoryResponse, error)
}

type ContextOpsService interface {
	ListContextSamples(ctx context.Context, request ContextSamplesRequest) (ContextSamplesResponse, error)
	TuneContext(ctx context.Context, request ContextTuneRequest) (ContextTuneResponse, error)
	QueryContextMemoryInventory(ctx context.Context, request ContextMemoryInventoryRequest) (ContextMemoryInventoryResponse, error)
	QueryContextMemoryCandidates(ctx context.Context, request ContextMemoryCandidatesRequest) (ContextMemoryCandidatesResponse, error)
	QueryContextRetrievalDocuments(ctx context.Context, request ContextRetrievalDocumentsRequest) (ContextRetrievalDocumentsResponse, error)
	QueryContextRetrievalChunks(ctx context.Context, request ContextRetrievalChunksRequest) (ContextRetrievalChunksResponse, error)
}
