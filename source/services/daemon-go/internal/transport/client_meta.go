package transport

import (
	"context"
	"net/http"
)

func (c *Client) DaemonCapabilities(ctx context.Context, correlationID string) (DaemonCapabilitiesResponse, error) {
	var response DaemonCapabilitiesResponse
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/v1/meta/capabilities", nil, correlationID, &response)
	return response, err
}
