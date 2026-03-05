package transport

import (
	"context"
	"net/http"
)

func (c *Client) IdentityWorkspaces(ctx context.Context, request IdentityWorkspacesRequest, correlationID string) (IdentityWorkspacesResponse, error) {
	var response IdentityWorkspacesResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/workspaces", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentityPrincipals(ctx context.Context, request IdentityPrincipalsRequest, correlationID string) (IdentityPrincipalsResponse, error) {
	var response IdentityPrincipalsResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/principals", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentityActiveContext(ctx context.Context, request IdentityActiveContextRequest, correlationID string) (IdentityActiveContextResponse, error) {
	var response IdentityActiveContextResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/context", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentitySelectWorkspace(ctx context.Context, request IdentityWorkspaceSelectRequest, correlationID string) (IdentityActiveContextResponse, error) {
	var response IdentityActiveContextResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/context/select-workspace", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentityBootstrap(ctx context.Context, request IdentityBootstrapRequest, correlationID string) (IdentityBootstrapResponse, error) {
	var response IdentityBootstrapResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/bootstrap", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentityDevices(ctx context.Context, request IdentityDeviceListRequest, correlationID string) (IdentityDeviceListResponse, error) {
	var response IdentityDeviceListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/devices/list", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentitySessions(ctx context.Context, request IdentitySessionListRequest, correlationID string) (IdentitySessionListResponse, error) {
	var response IdentitySessionListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/sessions/list", request, correlationID, &response)
	return response, err
}

func (c *Client) IdentitySessionRevoke(ctx context.Context, request IdentitySessionRevokeRequest, correlationID string) (IdentitySessionRevokeResponse, error) {
	var response IdentitySessionRevokeResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/identity/sessions/revoke", request, correlationID, &response)
	return response, err
}
