package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	openapitypes "personalagent/runtime/internal/transport/openapitypes"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientConfig struct {
	ListenerMode ListenerMode
	Address      string
	AuthToken    string
	Timeout      time.Duration
	TLSConfig    *tls.Config
}

type Client struct {
	httpClient *http.Client
	wsDialer   *websocket.Dialer
	baseURL    string
	wsURL      string
	authToken  string
}

type HTTPError struct {
	StatusCode     int
	Body           string
	Code           string
	Message        string
	CorrelationID  string
	DetailsPayload json.RawMessage
}

func (e HTTPError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.Body)
	}
	if strings.TrimSpace(e.Code) != "" {
		return fmt.Sprintf("http %d %s: %s", e.StatusCode, strings.TrimSpace(e.Code), message)
	}
	return fmt.Sprintf("http %d: %s", e.StatusCode, message)
}

func NewClient(config ClientConfig) (*Client, error) {
	if strings.TrimSpace(config.AuthToken) == "" {
		return nil, fmt.Errorf("auth token is required")
	}
	if config.ListenerMode == "" {
		config.ListenerMode = ListenerModeTCP
	}
	if config.ListenerMode == ListenerModeNamedPipe {
		trimmed := strings.TrimSpace(config.Address)
		if trimmed == "" || trimmed == DefaultTCPAddress {
			config.Address = DefaultNamedPipeAddress
		}
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}

	client := &Client{authToken: config.AuthToken}
	transport := &http.Transport{}
	dialer := websocket.Dialer{HandshakeTimeout: config.Timeout}

	switch config.ListenerMode {
	case ListenerModeTCP:
		if strings.TrimSpace(config.Address) == "" {
			return nil, fmt.Errorf("tcp client requires address")
		}
		if config.TLSConfig != nil {
			transport.TLSClientConfig = config.TLSConfig
			dialer.TLSClientConfig = config.TLSConfig
			client.baseURL = "https://" + config.Address
			client.wsURL = "wss://" + config.Address + "/v1/realtime/ws"
		} else {
			client.baseURL = "http://" + config.Address
			client.wsURL = "ws://" + config.Address + "/v1/realtime/ws"
		}
	case ListenerModeUnix:
		if config.TLSConfig != nil {
			return nil, fmt.Errorf("tls is only supported for tcp transport mode")
		}
		if strings.TrimSpace(config.Address) == "" {
			return nil, fmt.Errorf("unix client requires socket path")
		}
		dialContext := func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", config.Address)
		}
		transport.DialContext = dialContext
		dialer.NetDialContext = dialContext
		client.baseURL = "http://unix"
		client.wsURL = "ws://unix/v1/realtime/ws"
	case ListenerModeNamedPipe:
		if config.TLSConfig != nil {
			return nil, fmt.Errorf("tls is only supported for tcp transport mode")
		}
		dialContext := func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialNamedPipeContext(ctx, config.Address)
		}
		transport.DialContext = dialContext
		dialer.NetDialContext = dialContext
		client.baseURL = "http://namedpipe"
		client.wsURL = "ws://namedpipe/v1/realtime/ws"
	default:
		return nil, fmt.Errorf("unsupported listener mode %q", config.ListenerMode)
	}

	client.httpClient = &http.Client{Timeout: config.Timeout, Transport: transport}
	client.wsDialer = &dialer
	return client, nil
}

func (c *Client) SubmitTask(ctx context.Context, request SubmitTaskRequest, correlationID string) (SubmitTaskResponse, error) {
	var openapiResponse openapitypes.TaskSubmitResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/tasks", toOpenAPITaskSubmitRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return SubmitTaskResponse{}, err
	}
	return fromOpenAPITaskSubmitResponse(openapiResponse), nil
}

func (c *Client) TaskStatus(ctx context.Context, taskID string, correlationID string) (TaskStatusResponse, error) {
	var response TaskStatusResponse
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/v1/tasks/"+taskID, nil, correlationID, &response)
	return response, err
}

func (c *Client) CancelTask(ctx context.Context, request TaskCancelRequest, correlationID string) (TaskCancelResponse, error) {
	var response TaskCancelResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/tasks/cancel", request, correlationID, &response)
	return response, err
}

func (c *Client) RetryTask(ctx context.Context, request TaskRetryRequest, correlationID string) (TaskRetryResponse, error) {
	var response TaskRetryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/tasks/retry", request, correlationID, &response)
	return response, err
}

func (c *Client) RequeueTask(ctx context.Context, request TaskRequeueRequest, correlationID string) (TaskRequeueResponse, error) {
	var response TaskRequeueResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/tasks/requeue", request, correlationID, &response)
	return response, err
}

func (c *Client) CapabilitySmoke(ctx context.Context, correlationID string) (CapabilitySmokeResponse, error) {
	var response CapabilitySmokeResponse
	err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/v1/capabilities/smoke", nil, correlationID, &response)
	return response, err
}

func (c *Client) UpsertSecretReference(ctx context.Context, request SecretReferenceUpsertRequest, correlationID string) (SecretReferenceResponse, error) {
	var response SecretReferenceResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/secrets/refs", request, correlationID, &response)
	return response, err
}

func (c *Client) GetSecretReference(ctx context.Context, workspaceID string, name string, correlationID string) (SecretReferenceResponse, error) {
	var response SecretReferenceResponse
	path := c.baseURL + "/v1/secrets/refs/" + url.PathEscape(strings.TrimSpace(workspaceID)) + "/" + url.PathEscape(strings.TrimSpace(name))
	err := c.doJSON(ctx, http.MethodGet, path, nil, correlationID, &response)
	return response, err
}

func (c *Client) DeleteSecretReference(ctx context.Context, workspaceID string, name string, correlationID string) (SecretReferenceDeleteResponse, error) {
	var response SecretReferenceDeleteResponse
	path := c.baseURL + "/v1/secrets/refs/" + url.PathEscape(strings.TrimSpace(workspaceID)) + "/" + url.PathEscape(strings.TrimSpace(name))
	err := c.doJSON(ctx, http.MethodDelete, path, nil, correlationID, &response)
	return response, err
}

func (c *Client) SetProvider(ctx context.Context, request ProviderSetRequest, correlationID string) (ProviderConfigRecord, error) {
	var openapiResponse openapitypes.ProviderConfigRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/providers/set", toOpenAPIProviderSetRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ProviderConfigRecord{}, err
	}
	return fromOpenAPIProviderConfigRecord(openapiResponse), nil
}

func (c *Client) ListProviders(ctx context.Context, request ProviderListRequest, correlationID string) (ProviderListResponse, error) {
	var openapiResponse openapitypes.ProviderListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/providers/list", toOpenAPIProviderListRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ProviderListResponse{}, err
	}
	return fromOpenAPIProviderListResponse(openapiResponse), nil
}

func (c *Client) CheckProviders(ctx context.Context, request ProviderCheckRequest, correlationID string) (ProviderCheckResponse, error) {
	var openapiResponse openapitypes.ProviderCheckResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/providers/check", toOpenAPIProviderCheckRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ProviderCheckResponse{}, err
	}
	return fromOpenAPIProviderCheckResponse(openapiResponse), nil
}

func (c *Client) ListModels(ctx context.Context, request ModelListRequest, correlationID string) (ModelListResponse, error) {
	var openapiResponse openapitypes.ModelListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/list", toOpenAPIModelListRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ModelListResponse{}, err
	}
	return fromOpenAPIModelListResponse(openapiResponse), nil
}

func (c *Client) DiscoverModels(ctx context.Context, request ModelDiscoverRequest, correlationID string) (ModelDiscoverResponse, error) {
	var openapiResponse openapitypes.ModelDiscoverResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/discover", toOpenAPIModelDiscoverRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ModelDiscoverResponse{}, err
	}
	return fromOpenAPIModelDiscoverResponse(openapiResponse), nil
}

func (c *Client) AddModel(ctx context.Context, request ModelCatalogAddRequest, correlationID string) (ModelCatalogEntryRecord, error) {
	var response ModelCatalogEntryRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/add", request, correlationID, &response)
	return response, err
}

func (c *Client) RemoveModel(ctx context.Context, request ModelCatalogRemoveRequest, correlationID string) (ModelCatalogRemoveResponse, error) {
	var response ModelCatalogRemoveResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/remove", request, correlationID, &response)
	return response, err
}

func (c *Client) EnableModel(ctx context.Context, request ModelToggleRequest, correlationID string) (ModelCatalogEntryRecord, error) {
	var response ModelCatalogEntryRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/enable", request, correlationID, &response)
	return response, err
}

func (c *Client) DisableModel(ctx context.Context, request ModelToggleRequest, correlationID string) (ModelCatalogEntryRecord, error) {
	var response ModelCatalogEntryRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/disable", request, correlationID, &response)
	return response, err
}

func (c *Client) SelectModelRoute(ctx context.Context, request ModelSelectRequest, correlationID string) (ModelRoutingPolicyRecord, error) {
	var response ModelRoutingPolicyRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/select", request, correlationID, &response)
	return response, err
}

func (c *Client) ModelPolicy(ctx context.Context, request ModelPolicyRequest, correlationID string) (ModelPolicyResponse, error) {
	var response ModelPolicyResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/policy", request, correlationID, &response)
	return response, err
}

func (c *Client) ResolveModelRoute(ctx context.Context, request ModelResolveRequest, correlationID string) (ModelResolveResponse, error) {
	var response ModelResolveResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/resolve", request, correlationID, &response)
	return response, err
}

func (c *Client) SimulateModelRoute(ctx context.Context, request ModelRouteSimulationRequest, correlationID string) (ModelRouteSimulationResponse, error) {
	var response ModelRouteSimulationResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/route/simulate", request, correlationID, &response)
	return response, err
}

func (c *Client) ExplainModelRoute(ctx context.Context, request ModelRouteExplainRequest, correlationID string) (ModelRouteExplainResponse, error) {
	var openapiResponse openapitypes.ModelRouteExplainResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/models/route/explain", toOpenAPIModelRouteExplainRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ModelRouteExplainResponse{}, err
	}
	return fromOpenAPIModelRouteExplainResponse(openapiResponse), nil
}

func (c *Client) ChatTurn(ctx context.Context, request ChatTurnRequest, correlationID string) (ChatTurnResponse, error) {
	var openapiResponse openapitypes.ChatTurnResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/chat/turn", toOpenAPIChatTurnRequest(request), correlationID, &openapiResponse)
	if err != nil {
		return ChatTurnResponse{}, err
	}
	return fromOpenAPIChatTurnResponse(openapiResponse), nil
}

func (c *Client) ChatTurnExplain(ctx context.Context, request ChatTurnExplainRequest, correlationID string) (ChatTurnExplainResponse, error) {
	var response ChatTurnExplainResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/chat/turn/explain", request, correlationID, &response)
	return response, err
}

func (c *Client) ChatTurnHistory(ctx context.Context, request ChatTurnHistoryRequest, correlationID string) (ChatTurnHistoryResponse, error) {
	var response ChatTurnHistoryResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/chat/history", request, correlationID, &response)
	return response, err
}

func (c *Client) GetChatPersonaPolicy(ctx context.Context, request ChatPersonaPolicyRequest, correlationID string) (ChatPersonaPolicyResponse, error) {
	var response ChatPersonaPolicyResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/chat/persona/get", request, correlationID, &response)
	return response, err
}

func (c *Client) UpsertChatPersonaPolicy(ctx context.Context, request ChatPersonaPolicyUpsertRequest, correlationID string) (ChatPersonaPolicyResponse, error) {
	var response ChatPersonaPolicyResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/chat/persona/set", request, correlationID, &response)
	return response, err
}

func (c *Client) AgentRun(ctx context.Context, request AgentRunRequest, correlationID string) (AgentRunResponse, error) {
	var response AgentRunResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/agent/run", request, correlationID, &response)
	return response, err
}

func (c *Client) AgentApprove(ctx context.Context, request AgentApproveRequest, correlationID string) (AgentRunResponse, error) {
	var response AgentRunResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/agent/approve", request, correlationID, &response)
	return response, err
}

func (c *Client) DelegationGrant(ctx context.Context, request DelegationGrantRequest, correlationID string) (DelegationRuleRecord, error) {
	var response DelegationRuleRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/grant", request, correlationID, &response)
	return response, err
}

func (c *Client) DelegationList(ctx context.Context, request DelegationListRequest, correlationID string) (DelegationListResponse, error) {
	var response DelegationListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/list", request, correlationID, &response)
	return response, err
}

func (c *Client) DelegationRevoke(ctx context.Context, request DelegationRevokeRequest, correlationID string) (DelegationRevokeResponse, error) {
	var response DelegationRevokeResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/revoke", request, correlationID, &response)
	return response, err
}

func (c *Client) DelegationCheck(ctx context.Context, request DelegationCheckRequest, correlationID string) (DelegationCheckResponse, error) {
	var response DelegationCheckResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/check", request, correlationID, &response)
	return response, err
}

func (c *Client) CapabilityGrantUpsert(ctx context.Context, request CapabilityGrantUpsertRequest, correlationID string) (CapabilityGrantRecord, error) {
	var response CapabilityGrantRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/capability-grants/upsert", request, correlationID, &response)
	return response, err
}

func (c *Client) CapabilityGrantList(ctx context.Context, request CapabilityGrantListRequest, correlationID string) (CapabilityGrantListResponse, error) {
	var response CapabilityGrantListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/delegation/capability-grants/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommSend(ctx context.Context, request CommSendRequest, correlationID string) (CommSendResponse, error) {
	var response CommSendResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/send", request, correlationID, &response)
	return response, err
}

func (c *Client) CommAttempts(ctx context.Context, request CommAttemptsRequest, correlationID string) (CommAttemptsResponse, error) {
	var response CommAttemptsResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/attempts", request, correlationID, &response)
	return response, err
}

func (c *Client) CommWebhookReceipts(ctx context.Context, request CommWebhookReceiptListRequest, correlationID string) (CommWebhookReceiptListResponse, error) {
	var response CommWebhookReceiptListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/webhook-receipts/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommIngestReceipts(ctx context.Context, request CommIngestReceiptListRequest, correlationID string) (CommIngestReceiptListResponse, error) {
	var response CommIngestReceiptListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/ingest-receipts/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommPolicySet(ctx context.Context, request CommPolicySetRequest, correlationID string) (CommPolicyRecord, error) {
	var response CommPolicyRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/policy/set", request, correlationID, &response)
	return response, err
}

func (c *Client) CommPolicyList(ctx context.Context, request CommPolicyListRequest, correlationID string) (CommPolicyListResponse, error) {
	var response CommPolicyListResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/policy/list", request, correlationID, &response)
	return response, err
}

func (c *Client) CommMessagesIngest(ctx context.Context, request MessagesIngestRequest, correlationID string) (MessagesIngestResponse, error) {
	var response MessagesIngestResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/messages/ingest", request, correlationID, &response)
	return response, err
}

func (c *Client) CommMailRuleIngest(ctx context.Context, request MailRuleIngestRequest, correlationID string) (MailRuleIngestResponse, error) {
	var response MailRuleIngestResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/mail/ingest", request, correlationID, &response)
	return response, err
}

func (c *Client) CommCalendarIngest(ctx context.Context, request CalendarChangeIngestRequest, correlationID string) (CalendarChangeIngestResponse, error) {
	var response CalendarChangeIngestResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/calendar/ingest", request, correlationID, &response)
	return response, err
}

func (c *Client) CommBrowserIngest(ctx context.Context, request BrowserEventIngestRequest, correlationID string) (BrowserEventIngestResponse, error) {
	var response BrowserEventIngestResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/comm/browser/ingest", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioSet(ctx context.Context, request TwilioSetRequest, correlationID string) (TwilioConfigRecord, error) {
	var response TwilioConfigRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/set", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioGet(ctx context.Context, request TwilioGetRequest, correlationID string) (TwilioConfigRecord, error) {
	var response TwilioConfigRecord
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/get", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioCheck(ctx context.Context, request TwilioCheckRequest, correlationID string) (TwilioCheckResponse, error) {
	var response TwilioCheckResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/check", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioSMSChatTurn(ctx context.Context, request TwilioSMSChatTurnRequest, correlationID string) (TwilioSMSChatTurn, error) {
	var response TwilioSMSChatTurn
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/sms-chat-turn", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioStartCall(ctx context.Context, request TwilioStartCallRequest, correlationID string) (TwilioStartCallResponse, error) {
	var response TwilioStartCallResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/start-call", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioCallStatus(ctx context.Context, request TwilioCallStatusRequest, correlationID string) (TwilioCallStatusResponse, error) {
	var response TwilioCallStatusResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/call-status", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioTranscript(ctx context.Context, request TwilioTranscriptRequest, correlationID string) (TwilioTranscriptResponse, error) {
	var response TwilioTranscriptResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/transcript", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioWebhookServe(ctx context.Context, request TwilioWebhookServeRequest, correlationID string) (TwilioWebhookServeResponse, error) {
	var response TwilioWebhookServeResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/webhook/serve", request, correlationID, &response)
	return response, err
}

func (c *Client) TwilioWebhookReplay(ctx context.Context, request TwilioWebhookReplayRequest, correlationID string) (TwilioWebhookReplayResponse, error) {
	var response TwilioWebhookReplayResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/channels/twilio/webhook/replay", request, correlationID, &response)
	return response, err
}

func (c *Client) CloudflaredVersion(ctx context.Context, request CloudflaredVersionRequest, correlationID string) (CloudflaredVersionResponse, error) {
	var response CloudflaredVersionResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/cloudflared/version", request, correlationID, &response)
	return response, err
}

func (c *Client) CloudflaredExec(ctx context.Context, request CloudflaredExecRequest, correlationID string) (CloudflaredExecResponse, error) {
	var response CloudflaredExecResponse
	err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/v1/connectors/cloudflared/exec", request, correlationID, &response)
	return response, err
}

func (c *Client) ConnectRealtime(ctx context.Context, correlationID string) (*RealtimeClientConnection, error) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.authToken)
	if strings.TrimSpace(correlationID) != "" {
		headers.Set("X-Correlation-ID", correlationID)
	}
	conn, response, err := c.wsDialer.DialContext(ctx, c.wsURL, headers)
	if err != nil {
		if response != nil {
			body, _ := io.ReadAll(response.Body)
			return nil, HTTPError{StatusCode: response.StatusCode, Body: string(body)}
		}
		return nil, err
	}
	return &RealtimeClientConnection{conn: conn}, nil
}

func (c *Client) doJSON(ctx context.Context, method string, url string, requestPayload any, correlationID string, responsePayload any) error {
	var bodyReader io.Reader
	if requestPayload != nil {
		body, err := json.Marshal(requestPayload)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(body)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.authToken)
	httpRequest.Header.Set("Accept", "application/json")
	if requestPayload != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(correlationID) != "" {
		httpRequest.Header.Set("X-Correlation-ID", correlationID)
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResponse.Body)
		return parseTransportHTTPError(httpResponse.StatusCode, body, strings.TrimSpace(httpResponse.Header.Get("X-Correlation-ID")))
	}

	if responsePayload == nil {
		return nil
	}
	return json.NewDecoder(httpResponse.Body).Decode(responsePayload)
}

type RealtimeClientConnection struct {
	conn    *websocket.Conn
	readMu  sync.Mutex
	readErr error
}

func (c *RealtimeClientConnection) Receive() (RealtimeEventEnvelope, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if c.readErr != nil {
		return RealtimeEventEnvelope{}, c.readErr
	}
	var event RealtimeEventEnvelope
	if err := c.conn.ReadJSON(&event); err != nil {
		c.readErr = err
		return RealtimeEventEnvelope{}, err
	}
	return event, nil
}

func (c *RealtimeClientConnection) ReceiveWithTimeout(timeout time.Duration) (RealtimeEventEnvelope, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if c.readErr != nil {
		return RealtimeEventEnvelope{}, c.readErr
	}
	if timeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
		defer c.conn.SetReadDeadline(time.Time{})
	}
	var event RealtimeEventEnvelope
	if err := c.conn.ReadJSON(&event); err != nil {
		c.readErr = err
		return RealtimeEventEnvelope{}, err
	}
	return event, nil
}

func (c *RealtimeClientConnection) SendSignal(signal ClientSignal) error {
	return c.conn.WriteJSON(signal)
}

func (c *RealtimeClientConnection) Close() error {
	return c.conn.Close()
}
