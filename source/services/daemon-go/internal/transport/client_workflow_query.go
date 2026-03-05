package transport

import (
	"context"
	"net/http"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func (c *Client) ApprovalInbox(ctx context.Context, request ApprovalInboxRequest, correlationID string) (ApprovalInboxResponse, error) {
	var openapiResponse openapitypes.ApprovalInboxResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/approvals/list", toOpenAPIApprovalInboxRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ApprovalInboxResponse{}, err
	}
	return fromOpenAPIApprovalInboxResponse(openapiResponse), nil
}

func (c *Client) TaskRunList(ctx context.Context, request TaskRunListRequest, correlationID string) (TaskRunListResponse, error) {
	var openapiResponse openapitypes.TaskRunListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/tasks/list", toOpenAPITaskRunListRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return TaskRunListResponse{}, err
	}
	return fromOpenAPITaskRunListResponse(openapiResponse), nil
}

func (c *Client) CommThreadList(ctx context.Context, request CommThreadListRequest, correlationID string) (CommThreadListResponse, error) {
	var response CommThreadListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/threads/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommEventTimeline(ctx context.Context, request CommEventTimelineRequest, correlationID string) (CommEventTimelineResponse, error) {
	var response CommEventTimelineResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/events/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommCallSessionList(ctx context.Context, request CommCallSessionListRequest, correlationID string) (CommCallSessionListResponse, error) {
	var response CommCallSessionListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/call-sessions/list", request, correlationID, &response)
	return response, err
}
