package transport

import (
	"context"
	"net/http"
)

func (c *Client) TwilioIngestSMS(ctx context.Context, request TwilioIngestSMSRequest, correlationID string) (TwilioIngestSMSResponse, error) {
	var response TwilioIngestSMSResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/ingest-sms", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioIngestVoice(ctx context.Context, request TwilioIngestVoiceRequest, correlationID string) (TwilioIngestVoiceResponse, error) {
	var response TwilioIngestVoiceResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/ingest-voice", request, correlationID, &response)
	return response, err
}
