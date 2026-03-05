package transport

import (
	"context"
	"net/http"
)

func (c *Client) DaemonLifecycleStatus(ctx context.Context, correlationID string) (DaemonLifecycleStatusResponse, error) {
	var response DaemonLifecycleStatusResponse
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/v1/daemon/lifecycle/status", nil, correlationID, &response)
	return response, err
}

func (c *Client) DaemonLifecycleControl(ctx context.Context, request DaemonLifecycleControlRequest, correlationID string) (DaemonLifecycleControlResponse, error) {
	var response DaemonLifecycleControlResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/daemon/lifecycle/control", request, correlationID, &response)
	return response, err
}

func (c *Client) DaemonPluginLifecycleHistory(ctx context.Context, request DaemonPluginLifecycleHistoryRequest, correlationID string) (DaemonPluginLifecycleHistoryResponse, error) {
	var response DaemonPluginLifecycleHistoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/daemon/lifecycle/plugins/history", request, correlationID, &response)
	return response, err
}
