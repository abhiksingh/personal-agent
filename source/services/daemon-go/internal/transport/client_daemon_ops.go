package transport

import (
	"context"
	"net/http"
)

func (c *Client) AutomationCreate(ctx context.Context, request AutomationCreateRequest, correlationID string) (AutomationTriggerRecord, error) {
	var response AutomationTriggerRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/create", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationList(ctx context.Context, request AutomationListRequest, correlationID string) (AutomationListResponse, error) {
	var response AutomationListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/list", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationFireHistory(ctx context.Context, request AutomationFireHistoryRequest, correlationID string) (AutomationFireHistoryResponse, error) {
	var response AutomationFireHistoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/fire-history", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationUpdate(ctx context.Context, request AutomationUpdateRequest, correlationID string) (AutomationUpdateResponse, error) {
	var response AutomationUpdateResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/update", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationDelete(ctx context.Context, request AutomationDeleteRequest, correlationID string) (AutomationDeleteResponse, error) {
	var response AutomationDeleteResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/delete", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationRunSchedule(ctx context.Context, request AutomationRunScheduleRequest, correlationID string) (AutomationRunScheduleResponse, error) {
	var response AutomationRunScheduleResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/run/schedule", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationRunCommEvent(ctx context.Context, request AutomationRunCommEventRequest, correlationID string) (AutomationRunCommEventResponse, error) {
	var response AutomationRunCommEventResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/run/comm-event", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationCommTriggerMetadata(ctx context.Context, request AutomationCommTriggerMetadataRequest, correlationID string) (AutomationCommTriggerMetadataResponse, error) {
	var response AutomationCommTriggerMetadataResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/comm-trigger/metadata", request, correlationID, &response)
	return response, err
}

func (c *Client) AutomationCommTriggerValidate(ctx context.Context, request AutomationCommTriggerValidateRequest, correlationID string) (AutomationCommTriggerValidateResponse, error) {
	var response AutomationCommTriggerValidateResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/automation/comm-trigger/validate", request, correlationID, &response)
	return response, err
}

func (c *Client) InspectRun(ctx context.Context, request InspectRunRequest, correlationID string) (InspectRunResponse, error) {
	var response InspectRunResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/inspect/run", request, correlationID, &response)
	return response, err
}

func (c *Client) InspectTranscript(ctx context.Context, request InspectTranscriptRequest, correlationID string) (InspectTranscriptResponse, error) {
	var response InspectTranscriptResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/inspect/transcript", request, correlationID, &response)
	return response, err
}

func (c *Client) InspectMemory(ctx context.Context, request InspectMemoryRequest, correlationID string) (InspectMemoryResponse, error) {
	var response InspectMemoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/inspect/memory", request, correlationID, &response)
	return response, err
}

func (c *Client) InspectLogsQuery(ctx context.Context, request InspectLogQueryRequest, correlationID string) (InspectLogQueryResponse, error) {
	var response InspectLogQueryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/inspect/logs/query", request, correlationID, &response)
	return response, err
}

func (c *Client) InspectLogsStream(ctx context.Context, request InspectLogStreamRequest, correlationID string) (InspectLogStreamResponse, error) {
	var response InspectLogStreamResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/inspect/logs/stream", request, correlationID, &response)
	return response, err
}

func (c *Client) RetentionPurge(ctx context.Context, request RetentionPurgeRequest, correlationID string) (RetentionPurgeResponse, error) {
	var response RetentionPurgeResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/retention/purge", request, correlationID, &response)
	return response, err
}

func (c *Client) RetentionCompactMemory(ctx context.Context, request RetentionCompactMemoryRequest, correlationID string) (RetentionCompactMemoryResponse, error) {
	var response RetentionCompactMemoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/retention/compact-memory", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextSamples(ctx context.Context, request ContextSamplesRequest, correlationID string) (ContextSamplesResponse, error) {
	var response ContextSamplesResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/samples", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextTune(ctx context.Context, request ContextTuneRequest, correlationID string) (ContextTuneResponse, error) {
	var response ContextTuneResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/tune", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextMemoryInventory(ctx context.Context, request ContextMemoryInventoryRequest, correlationID string) (ContextMemoryInventoryResponse, error) {
	var response ContextMemoryInventoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/memory/inventory", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextMemoryCandidates(ctx context.Context, request ContextMemoryCandidatesRequest, correlationID string) (ContextMemoryCandidatesResponse, error) {
	var response ContextMemoryCandidatesResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/memory/compaction-candidates", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextRetrievalDocuments(ctx context.Context, request ContextRetrievalDocumentsRequest, correlationID string) (ContextRetrievalDocumentsResponse, error) {
	var response ContextRetrievalDocumentsResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/retrieval/documents", request, correlationID, &response)
	return response, err
}

func (c *Client) ContextRetrievalChunks(ctx context.Context, request ContextRetrievalChunksRequest, correlationID string) (ContextRetrievalChunksResponse, error) {
	var response ContextRetrievalChunksResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/context/retrieval/chunks", request, correlationID, &response)
	return response, err
}
