package transport

import (
	"context"
	"net/http"
)

func (c *Client) ChannelConnectorMappingsList(ctx context.Context, request ChannelConnectorMappingListRequest, correlationID string) (ChannelConnectorMappingListResponse, error) {
	var response ChannelConnectorMappingListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/mappings/list", request, correlationID, &response)
	return response, err
}

func (c *Client) ChannelConnectorMappingUpsert(ctx context.Context, request ChannelConnectorMappingUpsertRequest, correlationID string) (ChannelConnectorMappingUpsertResponse, error) {
	var response ChannelConnectorMappingUpsertResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/mappings/upsert", request, correlationID, &response)
	return response, err
}

func (c *Client) ChannelStatus(ctx context.Context, request ChannelStatusRequest, correlationID string) (ChannelStatusResponse, error) {
	var response ChannelStatusResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/status", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectorStatus(ctx context.Context, request ConnectorStatusRequest, correlationID string) (ConnectorStatusResponse, error) {
	var response ConnectorStatusResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/status", request, correlationID, &response)
	return response, err
}

func (c *Client) ChannelDiagnostics(ctx context.Context, request ChannelDiagnosticsRequest, correlationID string) (ChannelDiagnosticsResponse, error) {
	var response ChannelDiagnosticsResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/diagnostics", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectorDiagnostics(ctx context.Context, request ConnectorDiagnosticsRequest, correlationID string) (ConnectorDiagnosticsResponse, error) {
	var response ConnectorDiagnosticsResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/diagnostics", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectorPermissionRequest(ctx context.Context, request ConnectorPermissionRequest, correlationID string) (ConnectorPermissionResponse, error) {
	var response ConnectorPermissionResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/permission/request", request, correlationID, &response)
	return response, err
}

func (c *Client) ChannelConfigUpsert(ctx context.Context, request ChannelConfigUpsertRequest, correlationID string) (ChannelConfigUpsertResponse, error) {
	var response ChannelConfigUpsertResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/config/upsert", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectorConfigUpsert(ctx context.Context, request ConnectorConfigUpsertRequest, correlationID string) (ConnectorConfigUpsertResponse, error) {
	var response ConnectorConfigUpsertResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/config/upsert", request, correlationID, &response)
	return response, err
}

func (c *Client) ChannelTestOperation(ctx context.Context, request ChannelTestOperationRequest, correlationID string) (ChannelTestOperationResponse, error) {
	var response ChannelTestOperationResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/test", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectorTestOperation(ctx context.Context, request ConnectorTestOperationRequest, correlationID string) (ConnectorTestOperationResponse, error) {
	var response ConnectorTestOperationResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/test", request, correlationID, &response)
	return response, err
}
