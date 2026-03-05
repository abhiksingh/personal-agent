package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type lifecycleServiceStub struct {
	statusResponse  DaemonLifecycleStatusResponse
	controlResponse DaemonLifecycleControlResponse
	historyResponse DaemonPluginLifecycleHistoryResponse
	lastAction      string
	lastHistoryReq  DaemonPluginLifecycleHistoryRequest
}

func (s *lifecycleServiceStub) DaemonLifecycleStatus(context.Context) (DaemonLifecycleStatusResponse, error) {
	return s.statusResponse, nil
}

func (s *lifecycleServiceStub) DaemonLifecycleControl(_ context.Context, request DaemonLifecycleControlRequest) (DaemonLifecycleControlResponse, error) {
	s.lastAction = request.Action
	response := s.controlResponse
	if strings.TrimSpace(response.Action) == "" {
		response.Action = request.Action
	}
	return response, nil
}

func (s *lifecycleServiceStub) DaemonPluginLifecycleHistory(_ context.Context, request DaemonPluginLifecycleHistoryRequest) (DaemonPluginLifecycleHistoryResponse, error) {
	s.lastHistoryReq = request
	return s.historyResponse, nil
}

type uiStatusServiceStub struct {
	channelConnectorMappingListResponse   ChannelConnectorMappingListResponse
	channelConnectorMappingUpsertResponse ChannelConnectorMappingUpsertResponse
	channelResponse                       ChannelStatusResponse
	connectorResponse                     ConnectorStatusResponse
	channelDiagnosticsResponse            ChannelDiagnosticsResponse
	connectorDiagnosticsResponse          ConnectorDiagnosticsResponse
	connectorPermissionResponse           ConnectorPermissionResponse
	channelConfigResponse                 ChannelConfigUpsertResponse
	connectorConfigResponse               ConnectorConfigUpsertResponse
	channelTestResponse                   ChannelTestOperationResponse
	connectorTestResponse                 ConnectorTestOperationResponse
	lastChannelConnectorMappingListReq    ChannelConnectorMappingListRequest
	lastChannelConnectorMappingUpsertReq  ChannelConnectorMappingUpsertRequest
	lastChannelReq                        ChannelStatusRequest
	lastConnectorReq                      ConnectorStatusRequest
	lastChannelDiagReq                    ChannelDiagnosticsRequest
	lastConnectorDiagReq                  ConnectorDiagnosticsRequest
	lastConnectorPermissionReq            ConnectorPermissionRequest
	lastChannelConfigReq                  ChannelConfigUpsertRequest
	lastConnectorConfigReq                ConnectorConfigUpsertRequest
	lastChannelTestReq                    ChannelTestOperationRequest
	lastConnectorTestReq                  ConnectorTestOperationRequest
}

func (s *uiStatusServiceStub) ListChannelConnectorMappings(_ context.Context, request ChannelConnectorMappingListRequest) (ChannelConnectorMappingListResponse, error) {
	s.lastChannelConnectorMappingListReq = request
	return s.channelConnectorMappingListResponse, nil
}

func (s *uiStatusServiceStub) UpsertChannelConnectorMapping(_ context.Context, request ChannelConnectorMappingUpsertRequest) (ChannelConnectorMappingUpsertResponse, error) {
	s.lastChannelConnectorMappingUpsertReq = request
	return s.channelConnectorMappingUpsertResponse, nil
}

func (s *uiStatusServiceStub) ListChannelStatus(_ context.Context, request ChannelStatusRequest) (ChannelStatusResponse, error) {
	s.lastChannelReq = request
	return s.channelResponse, nil
}

func (s *uiStatusServiceStub) ListConnectorStatus(_ context.Context, request ConnectorStatusRequest) (ConnectorStatusResponse, error) {
	s.lastConnectorReq = request
	return s.connectorResponse, nil
}

func (s *uiStatusServiceStub) ListChannelDiagnostics(_ context.Context, request ChannelDiagnosticsRequest) (ChannelDiagnosticsResponse, error) {
	s.lastChannelDiagReq = request
	return s.channelDiagnosticsResponse, nil
}

func (s *uiStatusServiceStub) ListConnectorDiagnostics(_ context.Context, request ConnectorDiagnosticsRequest) (ConnectorDiagnosticsResponse, error) {
	s.lastConnectorDiagReq = request
	return s.connectorDiagnosticsResponse, nil
}

func (s *uiStatusServiceStub) RequestConnectorPermission(_ context.Context, request ConnectorPermissionRequest) (ConnectorPermissionResponse, error) {
	s.lastConnectorPermissionReq = request
	return s.connectorPermissionResponse, nil
}

func (s *uiStatusServiceStub) UpsertChannelConfig(_ context.Context, request ChannelConfigUpsertRequest) (ChannelConfigUpsertResponse, error) {
	s.lastChannelConfigReq = request
	return s.channelConfigResponse, nil
}

func (s *uiStatusServiceStub) UpsertConnectorConfig(_ context.Context, request ConnectorConfigUpsertRequest) (ConnectorConfigUpsertResponse, error) {
	s.lastConnectorConfigReq = request
	return s.connectorConfigResponse, nil
}

func (s *uiStatusServiceStub) TestChannelOperation(_ context.Context, request ChannelTestOperationRequest) (ChannelTestOperationResponse, error) {
	s.lastChannelTestReq = request
	return s.channelTestResponse, nil
}

func (s *uiStatusServiceStub) TestConnectorOperation(_ context.Context, request ConnectorTestOperationRequest) (ConnectorTestOperationResponse, error) {
	s.lastConnectorTestReq = request
	return s.connectorTestResponse, nil
}

type workflowQueryServiceStub struct {
	approvalInboxResponse ApprovalInboxResponse
	lastApprovalInboxReq  ApprovalInboxRequest
	taskRunListResponse   TaskRunListResponse
	lastTaskRunListReq    TaskRunListRequest
	commThreadResponse    CommThreadListResponse
	lastCommThreadReq     CommThreadListRequest
	commEventResponse     CommEventTimelineResponse
	lastCommEventReq      CommEventTimelineRequest
	commCallResponse      CommCallSessionListResponse
	lastCommCallReq       CommCallSessionListRequest
}

func (s *workflowQueryServiceStub) ListApprovalInbox(_ context.Context, request ApprovalInboxRequest) (ApprovalInboxResponse, error) {
	s.lastApprovalInboxReq = request
	return s.approvalInboxResponse, nil
}

func (s *workflowQueryServiceStub) ListTaskRuns(_ context.Context, request TaskRunListRequest) (TaskRunListResponse, error) {
	s.lastTaskRunListReq = request
	return s.taskRunListResponse, nil
}

func (s *workflowQueryServiceStub) ListCommThreads(_ context.Context, request CommThreadListRequest) (CommThreadListResponse, error) {
	s.lastCommThreadReq = request
	return s.commThreadResponse, nil
}

func (s *workflowQueryServiceStub) ListCommEvents(_ context.Context, request CommEventTimelineRequest) (CommEventTimelineResponse, error) {
	s.lastCommEventReq = request
	return s.commEventResponse, nil
}

func (s *workflowQueryServiceStub) ListCommCallSessions(_ context.Context, request CommCallSessionListRequest) (CommCallSessionListResponse, error) {
	s.lastCommCallReq = request
	return s.commCallResponse, nil
}

type taskStatusContractBackendStub struct {
	statusResponse TaskStatusResponse
	statusErr      error
	cancelErr      error
	retryErr       error
	requeueErr     error
}

func (s *taskStatusContractBackendStub) SubmitTask(_ context.Context, _ SubmitTaskRequest, _ string) (SubmitTaskResponse, error) {
	return SubmitTaskResponse{}, errors.New("not implemented in test stub")
}

func (s *taskStatusContractBackendStub) TaskStatus(_ context.Context, taskID string, correlationID string) (TaskStatusResponse, error) {
	if s.statusErr != nil {
		return TaskStatusResponse{}, s.statusErr
	}
	response := s.statusResponse
	if strings.TrimSpace(response.TaskID) == "" {
		response.TaskID = strings.TrimSpace(taskID)
	}
	if strings.TrimSpace(response.CorrelationID) == "" {
		response.CorrelationID = strings.TrimSpace(correlationID)
	}
	return response, nil
}

func (s *taskStatusContractBackendStub) CancelTask(_ context.Context, _ TaskCancelRequest, _ string) (TaskCancelResponse, error) {
	if s.cancelErr != nil {
		return TaskCancelResponse{}, s.cancelErr
	}
	return TaskCancelResponse{}, errors.New("not implemented in test stub")
}

func (s *taskStatusContractBackendStub) RetryTask(_ context.Context, _ TaskRetryRequest, _ string) (TaskRetryResponse, error) {
	if s.retryErr != nil {
		return TaskRetryResponse{}, s.retryErr
	}
	return TaskRetryResponse{}, errors.New("not implemented in test stub")
}

func (s *taskStatusContractBackendStub) RequeueTask(_ context.Context, _ TaskRequeueRequest, _ string) (TaskRequeueResponse, error) {
	if s.requeueErr != nil {
		return TaskRequeueResponse{}, s.requeueErr
	}
	return TaskRequeueResponse{}, errors.New("not implemented in test stub")
}

func (s *taskStatusContractBackendStub) CapabilitySmoke(_ context.Context, correlationID string) (CapabilitySmokeResponse, error) {
	return CapabilitySmokeResponse{
		Healthy:       true,
		DaemonVersion: "test",
		Channels:      []string{"app"},
		Connectors:    []string{"apple.mail"},
		CorrelationID: correlationID,
	}, nil
}

type identityDirectoryServiceStub struct {
	workspacesResponse     IdentityWorkspacesResponse
	principalsResponse     IdentityPrincipalsResponse
	activeContextResponse  IdentityActiveContextResponse
	selectContextResponse  IdentityActiveContextResponse
	bootstrapResponse      IdentityBootstrapResponse
	devicesResponse        IdentityDeviceListResponse
	sessionsResponse       IdentitySessionListResponse
	sessionRevokeResponse  IdentitySessionRevokeResponse
	lastWorkspacesRequest  IdentityWorkspacesRequest
	lastPrincipalsRequest  IdentityPrincipalsRequest
	lastActiveContextReq   IdentityActiveContextRequest
	lastSelectWorkspaceReq IdentityWorkspaceSelectRequest
	lastBootstrapRequest   IdentityBootstrapRequest
	lastDevicesRequest     IdentityDeviceListRequest
	lastSessionsRequest    IdentitySessionListRequest
	lastSessionRevokeReq   IdentitySessionRevokeRequest
}

func (s *identityDirectoryServiceStub) ListWorkspaces(_ context.Context, request IdentityWorkspacesRequest) (IdentityWorkspacesResponse, error) {
	s.lastWorkspacesRequest = request
	return s.workspacesResponse, nil
}

func (s *identityDirectoryServiceStub) ListPrincipals(_ context.Context, request IdentityPrincipalsRequest) (IdentityPrincipalsResponse, error) {
	s.lastPrincipalsRequest = request
	return s.principalsResponse, nil
}

func (s *identityDirectoryServiceStub) GetActiveContext(_ context.Context, request IdentityActiveContextRequest) (IdentityActiveContextResponse, error) {
	s.lastActiveContextReq = request
	return s.activeContextResponse, nil
}

func (s *identityDirectoryServiceStub) SelectWorkspace(_ context.Context, request IdentityWorkspaceSelectRequest) (IdentityActiveContextResponse, error) {
	s.lastSelectWorkspaceReq = request
	return s.selectContextResponse, nil
}

func (s *identityDirectoryServiceStub) Bootstrap(_ context.Context, request IdentityBootstrapRequest) (IdentityBootstrapResponse, error) {
	s.lastBootstrapRequest = request
	return s.bootstrapResponse, nil
}

func (s *identityDirectoryServiceStub) ListDevices(_ context.Context, request IdentityDeviceListRequest) (IdentityDeviceListResponse, error) {
	s.lastDevicesRequest = request
	return s.devicesResponse, nil
}

func (s *identityDirectoryServiceStub) ListSessions(_ context.Context, request IdentitySessionListRequest) (IdentitySessionListResponse, error) {
	s.lastSessionsRequest = request
	return s.sessionsResponse, nil
}

func (s *identityDirectoryServiceStub) RevokeSession(_ context.Context, request IdentitySessionRevokeRequest) (IdentitySessionRevokeResponse, error) {
	s.lastSessionRevokeReq = request
	return s.sessionRevokeResponse, nil
}

type cloudflaredConnectorServiceStub struct {
	versionResponse CloudflaredVersionResponse
	execResponse    CloudflaredExecResponse
	lastVersionReq  CloudflaredVersionRequest
	lastExecReq     CloudflaredExecRequest
}

func (s *cloudflaredConnectorServiceStub) CloudflaredVersion(_ context.Context, request CloudflaredVersionRequest) (CloudflaredVersionResponse, error) {
	s.lastVersionReq = request
	return s.versionResponse, nil
}

func (s *cloudflaredConnectorServiceStub) CloudflaredExec(_ context.Context, request CloudflaredExecRequest) (CloudflaredExecResponse, error) {
	s.lastExecReq = request
	return s.execResponse, nil
}

type delegationServiceStub struct {
	grantResponse           DelegationRuleRecord
	listResponse            DelegationListResponse
	revokeResponse          DelegationRevokeResponse
	checkResponse           DelegationCheckResponse
	capabilityUpsertResp    CapabilityGrantRecord
	capabilityListResp      CapabilityGrantListResponse
	lastGrantReq            DelegationGrantRequest
	lastListReq             DelegationListRequest
	lastRevokeReq           DelegationRevokeRequest
	lastCheckReq            DelegationCheckRequest
	lastCapabilityUpsertReq CapabilityGrantUpsertRequest
	lastCapabilityListReq   CapabilityGrantListRequest
}

func (s *delegationServiceStub) GrantDelegation(_ context.Context, request DelegationGrantRequest) (DelegationRuleRecord, error) {
	s.lastGrantReq = request
	return s.grantResponse, nil
}

func (s *delegationServiceStub) ListDelegations(_ context.Context, request DelegationListRequest) (DelegationListResponse, error) {
	s.lastListReq = request
	return s.listResponse, nil
}

func (s *delegationServiceStub) RevokeDelegation(_ context.Context, request DelegationRevokeRequest) (DelegationRevokeResponse, error) {
	s.lastRevokeReq = request
	return s.revokeResponse, nil
}

func (s *delegationServiceStub) CheckDelegation(_ context.Context, request DelegationCheckRequest) (DelegationCheckResponse, error) {
	s.lastCheckReq = request
	return s.checkResponse, nil
}

func (s *delegationServiceStub) UpsertCapabilityGrant(_ context.Context, request CapabilityGrantUpsertRequest) (CapabilityGrantRecord, error) {
	s.lastCapabilityUpsertReq = request
	return s.capabilityUpsertResp, nil
}

func (s *delegationServiceStub) ListCapabilityGrants(_ context.Context, request CapabilityGrantListRequest) (CapabilityGrantListResponse, error) {
	s.lastCapabilityListReq = request
	return s.capabilityListResp, nil
}

type inspectServiceStub struct {
	inspectRunResponse InspectRunResponse
	lastInspectRunReq  InspectRunRequest
	queryResponse      InspectLogQueryResponse
	streamResponse     InspectLogStreamResponse
	lastQueryReq       InspectLogQueryRequest
	lastStreamReq      InspectLogStreamRequest
}

func (s *inspectServiceStub) InspectRun(_ context.Context, request InspectRunRequest) (InspectRunResponse, error) {
	s.lastInspectRunReq = request
	return s.inspectRunResponse, nil
}

func (s *inspectServiceStub) InspectTranscript(context.Context, InspectTranscriptRequest) (InspectTranscriptResponse, error) {
	return InspectTranscriptResponse{}, nil
}

func (s *inspectServiceStub) InspectMemory(context.Context, InspectMemoryRequest) (InspectMemoryResponse, error) {
	return InspectMemoryResponse{}, nil
}

func (s *inspectServiceStub) QueryInspectLogs(_ context.Context, request InspectLogQueryRequest) (InspectLogQueryResponse, error) {
	s.lastQueryReq = request
	return s.queryResponse, nil
}

func (s *inspectServiceStub) StreamInspectLogs(_ context.Context, request InspectLogStreamRequest) (InspectLogStreamResponse, error) {
	s.lastStreamReq = request
	return s.streamResponse, nil
}

type contextOpsServiceStub struct {
	samplesResponse            ContextSamplesResponse
	tuneResponse               ContextTuneResponse
	memoryInventoryResponse    ContextMemoryInventoryResponse
	memoryCandidatesResponse   ContextMemoryCandidatesResponse
	retrievalDocumentsResponse ContextRetrievalDocumentsResponse
	retrievalChunksResponse    ContextRetrievalChunksResponse
	lastSamplesReq             ContextSamplesRequest
	lastTuneReq                ContextTuneRequest
	lastMemoryInventoryReq     ContextMemoryInventoryRequest
	lastMemoryCandidatesReq    ContextMemoryCandidatesRequest
	lastRetrievalDocumentsReq  ContextRetrievalDocumentsRequest
	lastRetrievalChunksReq     ContextRetrievalChunksRequest
}

func (s *contextOpsServiceStub) ListContextSamples(_ context.Context, request ContextSamplesRequest) (ContextSamplesResponse, error) {
	s.lastSamplesReq = request
	return s.samplesResponse, nil
}

func (s *contextOpsServiceStub) TuneContext(_ context.Context, request ContextTuneRequest) (ContextTuneResponse, error) {
	s.lastTuneReq = request
	return s.tuneResponse, nil
}

func (s *contextOpsServiceStub) QueryContextMemoryInventory(_ context.Context, request ContextMemoryInventoryRequest) (ContextMemoryInventoryResponse, error) {
	s.lastMemoryInventoryReq = request
	return s.memoryInventoryResponse, nil
}

func (s *contextOpsServiceStub) QueryContextMemoryCandidates(_ context.Context, request ContextMemoryCandidatesRequest) (ContextMemoryCandidatesResponse, error) {
	s.lastMemoryCandidatesReq = request
	return s.memoryCandidatesResponse, nil
}

func (s *contextOpsServiceStub) QueryContextRetrievalDocuments(_ context.Context, request ContextRetrievalDocumentsRequest) (ContextRetrievalDocumentsResponse, error) {
	s.lastRetrievalDocumentsReq = request
	return s.retrievalDocumentsResponse, nil
}

func (s *contextOpsServiceStub) QueryContextRetrievalChunks(_ context.Context, request ContextRetrievalChunksRequest) (ContextRetrievalChunksResponse, error) {
	s.lastRetrievalChunksReq = request
	return s.retrievalChunksResponse, nil
}

type chatServiceStub struct {
	response              ChatTurnResponse
	turnErr               error
	streamDeltas          []string
	lastRequest           ChatTurnRequest
	lastCorrelationID     string
	historyResponse       ChatTurnHistoryResponse
	historyErr            error
	lastHistoryRequest    ChatTurnHistoryRequest
	personaGetResponse    ChatPersonaPolicyResponse
	personaGetErr         error
	lastPersonaGetRequest ChatPersonaPolicyRequest
	personaSetResponse    ChatPersonaPolicyResponse
	personaSetErr         error
	lastPersonaSetRequest ChatPersonaPolicyUpsertRequest
	explainResponse       ChatTurnExplainResponse
	explainErr            error
	lastExplainRequest    ChatTurnExplainRequest
}

func (s *chatServiceStub) ChatTurn(_ context.Context, request ChatTurnRequest, correlationID string, streamFn func(delta string)) (ChatTurnResponse, error) {
	s.lastRequest = request
	s.lastCorrelationID = correlationID
	for _, delta := range s.streamDeltas {
		if streamFn != nil {
			streamFn(delta)
		}
	}
	if s.turnErr != nil {
		return ChatTurnResponse{}, s.turnErr
	}
	return s.response, nil
}

func (s *chatServiceStub) QueryChatTurnHistory(_ context.Context, request ChatTurnHistoryRequest) (ChatTurnHistoryResponse, error) {
	s.lastHistoryRequest = request
	if s.historyErr != nil {
		return ChatTurnHistoryResponse{}, s.historyErr
	}
	return s.historyResponse, nil
}

func (s *chatServiceStub) GetChatPersonaPolicy(_ context.Context, request ChatPersonaPolicyRequest) (ChatPersonaPolicyResponse, error) {
	s.lastPersonaGetRequest = request
	if s.personaGetErr != nil {
		return ChatPersonaPolicyResponse{}, s.personaGetErr
	}
	return s.personaGetResponse, nil
}

func (s *chatServiceStub) UpsertChatPersonaPolicy(_ context.Context, request ChatPersonaPolicyUpsertRequest) (ChatPersonaPolicyResponse, error) {
	s.lastPersonaSetRequest = request
	if s.personaSetErr != nil {
		return ChatPersonaPolicyResponse{}, s.personaSetErr
	}
	return s.personaSetResponse, nil
}

func (s *chatServiceStub) ExplainChatTurn(_ context.Context, request ChatTurnExplainRequest) (ChatTurnExplainResponse, error) {
	s.lastExplainRequest = request
	if s.explainErr != nil {
		return ChatTurnExplainResponse{}, s.explainErr
	}
	return s.explainResponse, nil
}

type agentServiceStub struct {
	runResponse    AgentRunResponse
	runErr         error
	lastRunRequest AgentRunRequest
}

func (s *agentServiceStub) RunAgent(_ context.Context, request AgentRunRequest) (AgentRunResponse, error) {
	s.lastRunRequest = request
	if s.runErr != nil {
		return AgentRunResponse{}, s.runErr
	}
	return s.runResponse, nil
}

func (s *agentServiceStub) ApproveAgent(_ context.Context, _ AgentApproveRequest) (AgentRunResponse, error) {
	return AgentRunResponse{}, nil
}

type automationServiceStub struct {
	updateResponse      AutomationUpdateResponse
	deleteResponse      AutomationDeleteResponse
	fireHistoryResponse AutomationFireHistoryResponse
	metadataResponse    AutomationCommTriggerMetadataResponse
	validateResponse    AutomationCommTriggerValidateResponse
	lastUpdateReq       AutomationUpdateRequest
	lastDeleteReq       AutomationDeleteRequest
	lastCreateReq       AutomationCreateRequest
	lastListReq         AutomationListRequest
	lastFireHistoryReq  AutomationFireHistoryRequest
	lastRunSchedReq     AutomationRunScheduleRequest
	lastRunCommReq      AutomationRunCommEventRequest
	lastCommMetadataReq AutomationCommTriggerMetadataRequest
	lastCommValidateReq AutomationCommTriggerValidateRequest
}

func (s *automationServiceStub) CreateAutomation(_ context.Context, request AutomationCreateRequest) (AutomationTriggerRecord, error) {
	s.lastCreateReq = request
	return AutomationTriggerRecord{}, nil
}

func (s *automationServiceStub) ListAutomation(_ context.Context, request AutomationListRequest) (AutomationListResponse, error) {
	s.lastListReq = request
	return AutomationListResponse{}, nil
}

func (s *automationServiceStub) ListAutomationFireHistory(_ context.Context, request AutomationFireHistoryRequest) (AutomationFireHistoryResponse, error) {
	s.lastFireHistoryReq = request
	return s.fireHistoryResponse, nil
}

func (s *automationServiceStub) UpdateAutomation(_ context.Context, request AutomationUpdateRequest) (AutomationUpdateResponse, error) {
	s.lastUpdateReq = request
	return s.updateResponse, nil
}

func (s *automationServiceStub) DeleteAutomation(_ context.Context, request AutomationDeleteRequest) (AutomationDeleteResponse, error) {
	s.lastDeleteReq = request
	return s.deleteResponse, nil
}

func (s *automationServiceStub) RunAutomationSchedule(_ context.Context, request AutomationRunScheduleRequest) (AutomationRunScheduleResponse, error) {
	s.lastRunSchedReq = request
	return AutomationRunScheduleResponse{}, nil
}

func (s *automationServiceStub) RunAutomationCommEvent(_ context.Context, request AutomationRunCommEventRequest) (AutomationRunCommEventResponse, error) {
	s.lastRunCommReq = request
	return AutomationRunCommEventResponse{}, nil
}

func (s *automationServiceStub) AutomationCommTriggerMetadata(_ context.Context, request AutomationCommTriggerMetadataRequest) (AutomationCommTriggerMetadataResponse, error) {
	s.lastCommMetadataReq = request
	return s.metadataResponse, nil
}

func (s *automationServiceStub) AutomationCommTriggerValidate(_ context.Context, request AutomationCommTriggerValidateRequest) (AutomationCommTriggerValidateResponse, error) {
	s.lastCommValidateReq = request
	return s.validateResponse, nil
}

func startTestServer(t *testing.T, config ServerConfig) *Server {
	t.Helper()
	return startTestServerWithBackend(t, config, nil)
}

func startTestServerWithBackend(t *testing.T, config ServerConfig, backend ControlBackend) *Server {
	t.Helper()
	broker := NewEventBroker()
	if backend == nil {
		backend = NewInMemoryControlBackend(broker)
	}
	server, err := NewServer(config, backend, broker)
	if err != nil {
		t.Fatalf("create transport server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start transport server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server
}

func dialRealtimeWSWithHeaders(t *testing.T, address string, headers http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	return dialer.Dial("ws://"+address+"/v1/realtime/ws", headers)
}

func buildTLSServerAndClientConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()

	now := time.Now()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "personal-agent-test-ca"},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    now.Add(-1 * time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server certificate: %v", err)
	}

	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	serverCertificate, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("load server key pair: %v", err)
	}

	serverTLSConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCertificate},
	}

	clientRoots := x509.NewCertPool()
	if !clientRoots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})) {
		t.Fatalf("append CA cert to client roots")
	}
	clientTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    clientRoots,
	}
	return serverTLSConfig, clientTLSConfig
}

func TestServerCloseInvokesBrokerCloseExactlyOnce(t *testing.T) {
	broker := NewEventBroker()
	backend := NewInMemoryControlBackend(broker)

	server, err := NewServer(ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "close-token",
	}, backend, broker)
	if err != nil {
		t.Fatalf("create transport server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start transport server: %v", err)
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	if err := server.Close(closeCtx); err != nil {
		t.Fatalf("first server close: %v", err)
	}

	closeCtx2, closeCancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel2()
	if err := server.Close(closeCtx2); err != nil {
		t.Fatalf("second server close should be idempotent: %v", err)
	}

	if got := broker.closeCallCount(); got != 1 {
		t.Fatalf("expected broker close to be invoked exactly once, got %d", got)
	}
}

func TestTransportServerAndClientOverTCPTLS(t *testing.T) {
	serverTLSConfig, clientTLSConfig := buildTLSServerAndClientConfigs(t)
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "tls-token",
		TLSConfig:    serverTLSConfig,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "tls-token",
		TLSConfig:    clientTLSConfig,
	})
	if err != nil {
		t.Fatalf("create tls transport client: %v", err)
	}

	smoke, err := client.CapabilitySmoke(context.Background(), "corr-tls-smoke")
	if err != nil {
		t.Fatalf("tls capability smoke failed: %v", err)
	}
	if !smoke.Healthy {
		t.Fatalf("expected healthy tls smoke response")
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-tls-stream")
	if err != nil {
		t.Fatalf("connect realtime stream over tls: %v", err)
	}
	_ = stream.Close()
}

func TestTransportTLSVerificationFailure(t *testing.T) {
	serverTLSConfig, _ := buildTLSServerAndClientConfigs(t)
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "tls-token",
		TLSConfig:    serverTLSConfig,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "tls-token",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})
	if err != nil {
		t.Fatalf("create tls client: %v", err)
	}

	if _, err := client.CapabilitySmoke(context.Background(), "corr-tls-fail"); err == nil {
		t.Fatalf("expected tls certificate verification failure")
	}
}

func TestTransportTLSUnsupportedForUnixMode(t *testing.T) {
	_, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeUnix,
		Address:      "/tmp/personal-agent-transport-tls.sock",
		AuthToken:    "tls-token",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	if err == nil {
		t.Fatalf("expected tls unsupported error for unix mode client")
	}

	broker := NewEventBroker()
	backend := NewInMemoryControlBackend(broker)
	_, err = NewServer(ServerConfig{
		ListenerMode: ListenerModeUnix,
		Address:      "/tmp/personal-agent-transport-tls.sock",
		AuthToken:    "tls-token",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}, backend, broker)
	if err == nil {
		t.Fatalf("expected tls unsupported error for unix mode server")
	}
}

func TestTransportServerAndClientOverTCP(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "test-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "test-token",
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-stream")
	if err != nil {
		t.Fatalf("connect realtime stream: %v", err)
	}
	defer stream.Close()

	submitResponse, err := client.SubmitTask(context.Background(), SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor-requester",
		SubjectPrincipalActorID: "actor-subject",
		Title:                   "Test transport task",
		TaskClass:               "chat",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if submitResponse.State != "queued" {
		t.Fatalf("expected queued state, got %s", submitResponse.State)
	}

	event, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive stream event: %v", err)
	}
	if event.EventType != "task_submitted" {
		t.Fatalf("expected task_submitted event, got %s", event.EventType)
	}
	if event.Sequence <= 0 {
		t.Fatalf("expected monotonic sequence, got %d", event.Sequence)
	}
	if gotRunID := strings.TrimSpace(fmt.Sprintf("%v", event.Payload.AsMap()["run_id"])); gotRunID != submitResponse.RunID {
		t.Fatalf("expected task_submitted payload run_id %s, got %s", submitResponse.RunID, gotRunID)
	}

	lifecycleEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive lifecycle stream event: %v", err)
	}
	if lifecycleEvent.EventType != "task_run_lifecycle" {
		t.Fatalf("expected task_run_lifecycle event, got %s", lifecycleEvent.EventType)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued lifecycle state, got %s", gotState)
	}
	if gotRunID := strings.TrimSpace(fmt.Sprintf("%v", lifecycleEvent.Payload.AsMap()["run_id"])); gotRunID != submitResponse.RunID {
		t.Fatalf("expected lifecycle payload run_id %s, got %s", submitResponse.RunID, gotRunID)
	}
	if strings.TrimSpace(lifecycleEvent.CorrelationID) != "corr-submit" {
		t.Fatalf("expected lifecycle correlation corr-submit, got %s", lifecycleEvent.CorrelationID)
	}

	statusResponse, err := client.TaskStatus(context.Background(), submitResponse.TaskID, "corr-status")
	if err != nil {
		t.Fatalf("task status: %v", err)
	}
	if statusResponse.TaskID != submitResponse.TaskID {
		t.Fatalf("expected task id %s, got %s", submitResponse.TaskID, statusResponse.TaskID)
	}
	if statusResponse.RunID != submitResponse.RunID || statusResponse.State != "queued" || statusResponse.RunState != "queued" {
		t.Fatalf("unexpected queued task status payload: %+v", statusResponse)
	}
	if !statusResponse.Actions.CanCancel || statusResponse.Actions.CanRetry || !statusResponse.Actions.CanRequeue {
		t.Fatalf("unexpected queued action availability: %+v", statusResponse.Actions)
	}

	cancelResponse, err := client.CancelTask(context.Background(), TaskCancelRequest{
		RunID:  submitResponse.RunID,
		Reason: "transport cancel test",
	}, "corr-task-cancel")
	if err != nil {
		t.Fatalf("cancel task run: %v", err)
	}
	if !cancelResponse.Cancelled {
		t.Fatalf("expected cancelled=true response, got %+v", cancelResponse)
	}
	if cancelResponse.RunState != "cancelled" || cancelResponse.TaskState != "cancelled" {
		t.Fatalf("expected cancelled task/run states, got task=%s run=%s", cancelResponse.TaskState, cancelResponse.RunState)
	}

	cancelLifecycleEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive cancel lifecycle event: %v", err)
	}
	if cancelLifecycleEvent.EventType != "task_run_lifecycle" {
		t.Fatalf("expected task_run_lifecycle cancel event, got %s", cancelLifecycleEvent.EventType)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", cancelLifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "cancelled" {
		t.Fatalf("expected cancelled lifecycle state, got %s", gotState)
	}
	if strings.TrimSpace(cancelLifecycleEvent.CorrelationID) != "corr-task-cancel" {
		t.Fatalf("expected cancellation correlation corr-task-cancel, got %s", cancelLifecycleEvent.CorrelationID)
	}

	cancelledStatus, err := client.TaskStatus(context.Background(), submitResponse.TaskID, "corr-status-after-cancel")
	if err != nil {
		t.Fatalf("task status after cancel: %v", err)
	}
	if cancelledStatus.State != "cancelled" {
		t.Fatalf("expected cancelled task status after cancel, got %s", cancelledStatus.State)
	}
	if cancelledStatus.RunID != submitResponse.RunID || cancelledStatus.RunState != "cancelled" {
		t.Fatalf("expected cancelled run status for original run, got %+v", cancelledStatus)
	}
	if cancelledStatus.Actions.CanCancel || !cancelledStatus.Actions.CanRetry || cancelledStatus.Actions.CanRequeue {
		t.Fatalf("unexpected cancelled action availability: %+v", cancelledStatus.Actions)
	}

	retryResponse, err := client.RetryTask(context.Background(), TaskRetryRequest{
		RunID:  submitResponse.RunID,
		Reason: "transport retry test",
	}, "corr-task-retry")
	if err != nil {
		t.Fatalf("retry task run: %v", err)
	}
	if !retryResponse.Retried || retryResponse.PreviousRunID != submitResponse.RunID || retryResponse.RunID == "" {
		t.Fatalf("unexpected retry response payload: %+v", retryResponse)
	}
	if retryResponse.TaskState != "queued" || retryResponse.RunState != "queued" {
		t.Fatalf("expected retry response queued states, got task=%s run=%s", retryResponse.TaskState, retryResponse.RunState)
	}
	if !retryResponse.Actions.CanCancel || retryResponse.Actions.CanRetry || !retryResponse.Actions.CanRequeue {
		t.Fatalf("unexpected retry action availability: %+v", retryResponse.Actions)
	}

	retryLifecycleEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive retry lifecycle event: %v", err)
	}
	if retryLifecycleEvent.EventType != "task_run_lifecycle" {
		t.Fatalf("expected task_run_lifecycle retry event, got %s", retryLifecycleEvent.EventType)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", retryLifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued retry lifecycle state, got %s", gotState)
	}
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", retryLifecycleEvent.Payload.AsMap()["lifecycle_source"])); gotSource != "control_backend_retry" {
		t.Fatalf("expected retry lifecycle source control_backend_retry, got %s", gotSource)
	}
	if strings.TrimSpace(retryLifecycleEvent.CorrelationID) != "corr-task-retry" {
		t.Fatalf("expected retry correlation corr-task-retry, got %s", retryLifecycleEvent.CorrelationID)
	}

	requeueResponse, err := client.RequeueTask(context.Background(), TaskRequeueRequest{
		RunID:  retryResponse.RunID,
		Reason: "transport requeue test",
	}, "corr-task-requeue")
	if err != nil {
		t.Fatalf("requeue task run: %v", err)
	}
	if !requeueResponse.Requeued || requeueResponse.PreviousRunID != retryResponse.RunID || requeueResponse.RunID == "" {
		t.Fatalf("unexpected requeue response payload: %+v", requeueResponse)
	}
	if requeueResponse.TaskState != "queued" || requeueResponse.RunState != "queued" {
		t.Fatalf("expected requeue response queued states, got task=%s run=%s", requeueResponse.TaskState, requeueResponse.RunState)
	}
	if !requeueResponse.Actions.CanCancel || requeueResponse.Actions.CanRetry || !requeueResponse.Actions.CanRequeue {
		t.Fatalf("unexpected requeue action availability: %+v", requeueResponse.Actions)
	}

	requeueCancelLifecycleEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive requeue cancellation lifecycle event: %v", err)
	}
	if requeueCancelLifecycleEvent.EventType != "task_run_lifecycle" {
		t.Fatalf("expected task_run_lifecycle requeue cancellation event, got %s", requeueCancelLifecycleEvent.EventType)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", requeueCancelLifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "cancelled" {
		t.Fatalf("expected cancelled requeue lifecycle state for prior run, got %s", gotState)
	}
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", requeueCancelLifecycleEvent.Payload.AsMap()["lifecycle_source"])); gotSource != "control_backend_requeue" {
		t.Fatalf("expected requeue cancellation lifecycle source control_backend_requeue, got %s", gotSource)
	}

	requeueQueuedLifecycleEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive requeue queued lifecycle event: %v", err)
	}
	if requeueQueuedLifecycleEvent.EventType != "task_run_lifecycle" {
		t.Fatalf("expected task_run_lifecycle requeue queued event, got %s", requeueQueuedLifecycleEvent.EventType)
	}
	if gotState := strings.TrimSpace(fmt.Sprintf("%v", requeueQueuedLifecycleEvent.Payload.AsMap()["lifecycle_state"])); gotState != "queued" {
		t.Fatalf("expected queued requeue lifecycle state for new run, got %s", gotState)
	}
	if gotSource := strings.TrimSpace(fmt.Sprintf("%v", requeueQueuedLifecycleEvent.Payload.AsMap()["lifecycle_source"])); gotSource != "control_backend_requeue" {
		t.Fatalf("expected requeue queued lifecycle source control_backend_requeue, got %s", gotSource)
	}

	requeueStatus, err := client.TaskStatus(context.Background(), submitResponse.TaskID, "corr-status-after-requeue")
	if err != nil {
		t.Fatalf("task status after requeue: %v", err)
	}
	if requeueStatus.RunID != requeueResponse.RunID || requeueStatus.State != "queued" || requeueStatus.RunState != "queued" {
		t.Fatalf("expected status to track requeued run, got %+v", requeueStatus)
	}
	if !requeueStatus.Actions.CanCancel || requeueStatus.Actions.CanRetry || !requeueStatus.Actions.CanRequeue {
		t.Fatalf("unexpected action availability after requeue: %+v", requeueStatus.Actions)
	}

	if err := stream.SendSignal(ClientSignal{SignalType: "cancel", TaskID: submitResponse.TaskID, CorrelationID: "corr-cancel"}); err != nil {
		t.Fatalf("send client signal: %v", err)
	}
	signalEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive signal event: %v", err)
	}
	if signalEvent.EventType != "client_signal" {
		t.Fatalf("expected client_signal event, got %s", signalEvent.EventType)
	}
	var signalAckEvent RealtimeEventEnvelope
	for i := 0; i < 4; i++ {
		candidate, recvErr := stream.Receive()
		if recvErr != nil {
			t.Fatalf("receive signal ack candidate event: %v", recvErr)
		}
		if candidate.EventType == "client_signal_ack" {
			signalAckEvent = candidate
			break
		}
	}
	if signalAckEvent.EventType != "client_signal_ack" {
		t.Fatalf("expected client_signal_ack event, got %s", signalAckEvent.EventType)
	}
	if gotAccepted := fmt.Sprintf("%v", signalAckEvent.Payload.AsMap()["accepted"]); gotAccepted != "true" {
		t.Fatalf("expected client_signal_ack accepted=true, got %v", signalAckEvent.Payload.AsMap()["accepted"])
	}
}

func TestTransportTaskControlRoutesUseTypedDomainErrorMapping(t *testing.T) {
	backend := &taskStatusContractBackendStub{
		statusErr:  NewTaskControlNotFoundError("task not found: task-missing"),
		cancelErr:  NewTaskControlReferenceMismatchError("workspace mismatch for task/run"),
		retryErr:   NewTaskControlStateConflictError(`task run state "running" is not retryable`),
		requeueErr: NewTaskControlMissingReferenceError("task_id or run_id is required"),
	}
	server := startTestServerWithBackend(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "task-error-token",
	}, backend)
	baseURL := "http://" + server.Address()
	httpClient := &http.Client{Timeout: 2 * time.Second}

	cases := []struct {
		name           string
		method         string
		path           string
		body           string
		statusCode     int
		errorCode      string
		detailCategory string
	}{
		{
			name:           "task status not found",
			method:         http.MethodGet,
			path:           "/v1/tasks/task-missing",
			statusCode:     http.StatusNotFound,
			errorCode:      "resource_not_found",
			detailCategory: taskControlErrorCategoryLookup,
		},
		{
			name:           "task cancel reference mismatch",
			method:         http.MethodPost,
			path:           "/v1/tasks/cancel",
			body:           `{"workspace_id":"ws1","task_id":"task-1","run_id":"run-1"}`,
			statusCode:     http.StatusBadRequest,
			errorCode:      "invalid_request",
			detailCategory: taskControlErrorCategoryReferenceMismatch,
		},
		{
			name:           "task retry state conflict",
			method:         http.MethodPost,
			path:           "/v1/tasks/retry",
			body:           `{"workspace_id":"ws1","task_id":"task-1","run_id":"run-1"}`,
			statusCode:     http.StatusConflict,
			errorCode:      "resource_conflict",
			detailCategory: taskControlErrorCategoryStateConflict,
		},
		{
			name:           "task requeue missing reference",
			method:         http.MethodPost,
			path:           "/v1/tasks/requeue",
			body:           `{}`,
			statusCode:     http.StatusBadRequest,
			errorCode:      "missing_required_field",
			detailCategory: taskControlErrorCategoryValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(tc.method, baseURL+tc.path, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			request.Header.Set("Authorization", "Bearer task-error-token")
			if tc.method != http.MethodGet {
				request.Header.Set("Content-Type", "application/json")
			}

			response, err := httpClient.Do(request)
			if err != nil {
				t.Fatalf("execute request: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, response.StatusCode)
			}

			var payload map[string]any
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode error payload: %v", err)
			}
			errorObjectRaw, ok := payload["error"]
			if !ok {
				t.Fatalf("expected error object")
			}
			errorObject, ok := errorObjectRaw.(map[string]any)
			if !ok {
				t.Fatalf("expected error map, got %T", errorObjectRaw)
			}
			if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != tc.errorCode {
				t.Fatalf("expected error.code %q, got %q", tc.errorCode, got)
			}
			detailsRaw, ok := errorObject["details"]
			if !ok {
				t.Fatalf("expected error.details")
			}
			details, ok := detailsRaw.(map[string]any)
			if !ok {
				t.Fatalf("expected details map, got %T", detailsRaw)
			}
			if got := strings.TrimSpace(fmt.Sprint(details["category"])); got != tc.detailCategory {
				t.Fatalf("expected details.category %q, got %q", tc.detailCategory, got)
			}
		})
	}
}

func TestTransportDaemonLifecycleStatusAndControl(t *testing.T) {
	lifecycle := &lifecycleServiceStub{
		statusResponse: DaemonLifecycleStatusResponse{
			LifecycleState: "running",
			ProcessID:      1234,
			SetupState:     "ready",
			InstallState:   "installed",
			DatabaseReady:  true,
			ControlAuth: DaemonControlAuthState{
				State:  "configured",
				Source: "auth_token_file",
			},
			HealthClassification: DaemonLifecycleHealthClassification{
				OverallState:       "ready",
				CoreRuntimeState:   "ready",
				PluginRuntimeState: "healthy",
				Blocking:           false,
			},
			Controls: DaemonLifecycleControls{
				Start:   true,
				Stop:    true,
				Restart: true,
			},
		},
		controlResponse: DaemonLifecycleControlResponse{
			Accepted:       true,
			Idempotent:     false,
			LifecycleState: "restart_requested",
			Message:        "daemon restart requested",
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "lifecycle-token",
		DaemonLifecycle: lifecycle,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "lifecycle-token",
	})
	if err != nil {
		t.Fatalf("create lifecycle client: %v", err)
	}

	status, err := client.DaemonLifecycleStatus(context.Background(), "corr-lifecycle-status")
	if err != nil {
		t.Fatalf("daemon lifecycle status: %v", err)
	}
	if status.LifecycleState != "running" || status.SetupState != "ready" {
		t.Fatalf("unexpected lifecycle status payload: %+v", status)
	}
	if status.HealthClassification.OverallState != "ready" ||
		status.HealthClassification.CoreRuntimeState != "ready" ||
		status.HealthClassification.PluginRuntimeState != "healthy" ||
		status.HealthClassification.Blocking {
		t.Fatalf("unexpected lifecycle health classification payload: %+v", status.HealthClassification)
	}
	if status.ControlAuth.State != "configured" || status.ControlAuth.Source != "auth_token_file" {
		t.Fatalf("unexpected control auth payload: %+v", status.ControlAuth)
	}

	control, err := client.DaemonLifecycleControl(context.Background(), DaemonLifecycleControlRequest{
		Action: "restart",
		Reason: "manual test",
	}, "corr-lifecycle-control")
	if err != nil {
		t.Fatalf("daemon lifecycle control: %v", err)
	}
	if !control.Accepted || control.LifecycleState != "restart_requested" {
		t.Fatalf("unexpected lifecycle control payload: %+v", control)
	}
	if lifecycle.lastAction != "restart" {
		t.Fatalf("expected lifecycle action restart, got %q", lifecycle.lastAction)
	}
}

func TestTransportDaemonLifecyclePluginHistoryRoute(t *testing.T) {
	lifecycle := &lifecycleServiceStub{
		historyResponse: DaemonPluginLifecycleHistoryResponse{
			WorkspaceID: "daemon",
			Items: []DaemonPluginLifecycleHistoryRecord{
				{
					AuditID:       "audit-1",
					WorkspaceID:   "daemon",
					PluginID:      "messages.daemon",
					Kind:          "channel",
					State:         "running",
					EventType:     "PLUGIN_HANDSHAKE_ACCEPTED",
					ProcessID:     12345,
					RestartCount:  1,
					Reason:        "worker_recovered",
					RestartEvent:  false,
					FailureEvent:  false,
					RecoveryEvent: true,
					OccurredAt:    "2026-02-25T00:00:00Z",
				},
			},
			HasMore:             true,
			NextCursorCreatedAt: "2026-02-25T00:00:00Z",
			NextCursorID:        "audit-1",
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "lifecycle-token",
		DaemonLifecycle: lifecycle,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "lifecycle-token",
	})
	if err != nil {
		t.Fatalf("create lifecycle client: %v", err)
	}

	response, err := client.DaemonPluginLifecycleHistory(context.Background(), DaemonPluginLifecycleHistoryRequest{
		WorkspaceID: "daemon",
		PluginID:    "messages.daemon",
		State:       "running",
		Limit:       10,
	}, "corr-lifecycle-plugin-history")
	if err != nil {
		t.Fatalf("daemon lifecycle plugin history: %v", err)
	}
	if response.WorkspaceID != "daemon" || len(response.Items) != 1 || response.Items[0].AuditID != "audit-1" {
		t.Fatalf("unexpected daemon lifecycle plugin history payload: %+v", response)
	}
	if !response.Items[0].RecoveryEvent || response.Items[0].Reason != "worker_recovered" {
		t.Fatalf("expected recovery lifecycle history record metadata, got %+v", response.Items[0])
	}
	if lifecycle.lastHistoryReq.PluginID != "messages.daemon" || lifecycle.lastHistoryReq.State != "running" {
		t.Fatalf("unexpected lifecycle history request payload: %+v", lifecycle.lastHistoryReq)
	}
}

func TestTransportDaemonLifecycleRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "lifecycle-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "lifecycle-token",
	})
	if err != nil {
		t.Fatalf("create lifecycle client: %v", err)
	}

	_, err = client.DaemonLifecycleStatus(context.Background(), "corr-lifecycle-status")
	if err == nil {
		t.Fatalf("expected daemon lifecycle status error when service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.DaemonPluginLifecycleHistory(context.Background(), DaemonPluginLifecycleHistoryRequest{
		WorkspaceID: "daemon",
	}, "corr-lifecycle-plugin-history")
	if err == nil {
		t.Fatalf("expected daemon lifecycle plugin history error when service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}

	_, err = client.CapabilityGrantList(context.Background(), CapabilityGrantListRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	}, "corr-capability-list")
	if err == nil {
		t.Fatalf("expected capability grant list to fail when delegation service is not configured")
	}
	httpErr = HTTPError{}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportDaemonLifecycleInstallControlAction(t *testing.T) {
	lifecycle := &lifecycleServiceStub{
		statusResponse: DaemonLifecycleStatusResponse{
			LifecycleState: "running",
			SetupState:     "ready",
			InstallState:   "installed",
			DatabaseReady:  true,
			HealthClassification: DaemonLifecycleHealthClassification{
				OverallState:       "ready",
				CoreRuntimeState:   "ready",
				PluginRuntimeState: "healthy",
				Blocking:           false,
			},
			ControlOperation: DaemonLifecycleControlOperation{
				State: "idle",
			},
			Controls: DaemonLifecycleControls{
				Start:   true,
				Stop:    true,
				Restart: true,
				Install: true,
				Repair:  true,
			},
		},
		controlResponse: DaemonLifecycleControlResponse{
			Action:         "install",
			Accepted:       true,
			Idempotent:     false,
			LifecycleState: "running",
			OperationState: "in_progress",
			Message:        "daemon install operation started",
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "lifecycle-token",
		DaemonLifecycle: lifecycle,
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "lifecycle-token",
	})
	if err != nil {
		t.Fatalf("create lifecycle client: %v", err)
	}

	control, err := client.DaemonLifecycleControl(context.Background(), DaemonLifecycleControlRequest{
		Action: "install",
		Reason: "transport install control test",
	}, "corr-lifecycle-install")
	if err != nil {
		t.Fatalf("daemon lifecycle install control: %v", err)
	}
	if !control.Accepted || control.OperationState != "in_progress" {
		t.Fatalf("unexpected lifecycle install control payload: %+v", control)
	}
	if lifecycle.lastAction != "install" {
		t.Fatalf("expected lifecycle action install, got %q", lifecycle.lastAction)
	}
}

func TestTransportServiceNotConfiguredErrorsIncludeRemediationMetadata(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "service-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "service-token",
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}

	_, err = client.ListProviders(context.Background(), ProviderListRequest{
		WorkspaceID: "ws1",
	}, "corr-service-not-configured")
	if err == nil {
		t.Fatalf("expected provider list to fail when provider service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
	if httpErr.Code != "service_not_configured" {
		t.Fatalf("expected code service_not_configured, got %q", httpErr.Code)
	}
	if len(httpErr.DetailsPayload) == 0 {
		t.Fatalf("expected details payload for service_not_configured error")
	}

	var details map[string]any
	if err := json.Unmarshal(httpErr.DetailsPayload, &details); err != nil {
		t.Fatalf("decode service_not_configured details: %v", err)
	}
	if got := details["category"]; got != "service_not_configured" {
		t.Fatalf("expected details.category service_not_configured, got %v", got)
	}
	if got := details["domain"]; got != "providers" {
		t.Fatalf("expected details.domain providers, got %v", got)
	}

	serviceRaw, ok := details["service"]
	if !ok {
		t.Fatalf("expected details.service object")
	}
	service, ok := serviceRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected details.service map, got %T", serviceRaw)
	}
	if got := service["id"]; got != "provider" {
		t.Fatalf("expected details.service.id provider, got %v", got)
	}
	if got := service["config_field"]; got != "Providers" {
		t.Fatalf("expected details.service.config_field Providers, got %v", got)
	}

	remediationRaw, ok := details["remediation"]
	if !ok {
		t.Fatalf("expected details.remediation object")
	}
	remediation, ok := remediationRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected details.remediation map, got %T", remediationRaw)
	}
	if got := remediation["action"]; got != "configure_server_service" {
		t.Fatalf("expected remediation action configure_server_service, got %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(remediation["hint"])); got == "" {
		t.Fatalf("expected non-empty remediation hint")
	}
}

func TestTransportChannelAndConnectorStatusRoutes(t *testing.T) {
	uiStatus := &uiStatusServiceStub{
		channelConnectorMappingListResponse: ChannelConnectorMappingListResponse{
			WorkspaceID:    "ws1",
			ChannelID:      "message",
			FallbackPolicy: "priority_order",
			Bindings: []ChannelConnectorMappingRecord{
				{
					ChannelID:   "message",
					ConnectorID: "imessage",
					Enabled:     true,
					Priority:    1,
				},
				{
					ChannelID:   "message",
					ConnectorID: "twilio",
					Enabled:     true,
					Priority:    2,
				},
			},
		},
		channelConnectorMappingUpsertResponse: ChannelConnectorMappingUpsertResponse{
			WorkspaceID:    "ws1",
			ChannelID:      "message",
			ConnectorID:    "twilio",
			Enabled:        true,
			Priority:       1,
			FallbackPolicy: "priority_order",
			UpdatedAt:      "2026-02-26T00:00:00Z",
			Bindings: []ChannelConnectorMappingRecord{
				{
					ChannelID:   "message",
					ConnectorID: "twilio",
					Enabled:     true,
					Priority:    1,
				},
				{
					ChannelID:   "message",
					ConnectorID: "imessage",
					Enabled:     true,
					Priority:    2,
				},
			},
		},
		channelResponse: ChannelStatusResponse{
			WorkspaceID: "ws1",
			Channels: []ChannelStatusCard{
				{
					ChannelID:   "app_chat",
					DisplayName: "App Chat",
					Status:      "ready",
					Configured:  true,
					Enabled:     true,
					ConfigFieldDescriptors: []ConfigFieldDescriptor{
						{
							Key:      "enabled",
							Label:    "Enabled",
							Type:     "bool",
							Required: false,
							Editable: true,
						},
					},
					RemediationActions: []DiagnosticsRemediationAction{
						{
							Identifier:  "refresh_channel_status",
							Label:       "Refresh Channel Status",
							Intent:      "refresh_status",
							Destination: "/v1/channels/status",
							Enabled:     true,
							Recommended: false,
						},
					},
				},
			},
		},
		connectorResponse: ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []ConnectorStatusCard{
				{
					ConnectorID: "mail",
					PluginID:    "mail.daemon",
					DisplayName: "Mail Connector",
					Status:      "ready",
					Configured:  true,
					Enabled:     true,
					ConfigFieldDescriptors: []ConfigFieldDescriptor{
						{
							Key:      "scope",
							Label:    "Mail Scope",
							Type:     "enum",
							Required: false,
							Editable: true,
						},
					},
					RemediationActions: []DiagnosticsRemediationAction{
						{
							Identifier:  "refresh_connector_status",
							Label:       "Refresh Connector Status",
							Intent:      "refresh_status",
							Destination: "/v1/connectors/status",
							Enabled:     true,
							Recommended: false,
						},
					},
				},
			},
		},
		connectorPermissionResponse: ConnectorPermissionResponse{
			WorkspaceID:     "ws1",
			ConnectorID:     "mail",
			PermissionState: "granted",
			Message:         "Mail permission request dispatched via Personal Agent Daemon.",
		},
		channelConfigResponse: ChannelConfigUpsertResponse{
			WorkspaceID: "ws1",
			ChannelID:   "app_chat",
			Configuration: UIStatusConfigurationFromMap(map[string]any{
				"transport": "daemon_realtime",
				"enabled":   true,
			}),
			UpdatedAt: "2026-02-25T00:00:00Z",
		},
		connectorConfigResponse: ConnectorConfigUpsertResponse{
			WorkspaceID: "ws1",
			ConnectorID: "mail",
			Configuration: UIStatusConfigurationFromMap(map[string]any{
				"scope": "inbox",
			}),
			UpdatedAt: "2026-02-25T00:00:01Z",
		},
		channelTestResponse: ChannelTestOperationResponse{
			WorkspaceID: "ws1",
			ChannelID:   "app_chat",
			Operation:   "health",
			Success:     true,
			Status:      "ok",
			Summary:     "app_chat channel worker is healthy.",
			CheckedAt:   "2026-02-25T00:00:02Z",
			Details: UIStatusTestOperationDetailsFromMap(map[string]any{
				"plugin_id": "app_chat.daemon",
			}),
		},
		connectorTestResponse: ConnectorTestOperationResponse{
			WorkspaceID: "ws1",
			ConnectorID: "mail",
			Operation:   "health",
			Success:     true,
			Status:      "ok",
			Summary:     "mail connector worker is healthy.",
			CheckedAt:   "2026-02-25T00:00:03Z",
			Details: UIStatusTestOperationDetailsFromMap(map[string]any{
				"plugin_id": "mail.daemon",
			}),
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "ui-status-token",
		UIStatus:     uiStatus,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "ui-status-token",
	})
	if err != nil {
		t.Fatalf("create ui status client: %v", err)
	}

	mappingList, err := client.ChannelConnectorMappingsList(context.Background(), ChannelConnectorMappingListRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
	}, "corr-channel-mapping-list")
	if err != nil {
		t.Fatalf("channel connector mapping list: %v", err)
	}
	if mappingList.WorkspaceID != "ws1" || mappingList.ChannelID != "message" || len(mappingList.Bindings) != 2 {
		t.Fatalf("unexpected channel connector mapping list payload: %+v", mappingList)
	}
	if uiStatus.lastChannelConnectorMappingListReq.WorkspaceID != "ws1" || uiStatus.lastChannelConnectorMappingListReq.ChannelID != "message" {
		t.Fatalf("unexpected channel connector mapping list request payload: %+v", uiStatus.lastChannelConnectorMappingListReq)
	}

	mappingUpsert, err := client.ChannelConnectorMappingUpsert(context.Background(), ChannelConnectorMappingUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "message",
		ConnectorID: "twilio",
		Enabled:     true,
		Priority:    1,
	}, "corr-channel-mapping-upsert")
	if err != nil {
		t.Fatalf("channel connector mapping upsert: %v", err)
	}
	if mappingUpsert.WorkspaceID != "ws1" || mappingUpsert.ChannelID != "message" || mappingUpsert.ConnectorID != "twilio" || mappingUpsert.Priority != 1 {
		t.Fatalf("unexpected channel connector mapping upsert payload: %+v", mappingUpsert)
	}
	if uiStatus.lastChannelConnectorMappingUpsertReq.WorkspaceID != "ws1" ||
		uiStatus.lastChannelConnectorMappingUpsertReq.ChannelID != "message" ||
		uiStatus.lastChannelConnectorMappingUpsertReq.ConnectorID != "twilio" ||
		uiStatus.lastChannelConnectorMappingUpsertReq.Priority != 1 {
		t.Fatalf("unexpected channel connector mapping upsert request payload: %+v", uiStatus.lastChannelConnectorMappingUpsertReq)
	}

	channelStatus, err := client.ChannelStatus(context.Background(), ChannelStatusRequest{WorkspaceID: "ws1"}, "corr-channel-status")
	if err != nil {
		t.Fatalf("channel status: %v", err)
	}
	if len(channelStatus.Channels) != 1 || channelStatus.Channels[0].ChannelID != "app_chat" {
		t.Fatalf("unexpected channel status payload: %+v", channelStatus)
	}
	if len(channelStatus.Channels[0].RemediationActions) != 1 ||
		channelStatus.Channels[0].RemediationActions[0].Identifier != "refresh_channel_status" {
		t.Fatalf("expected channel status remediation actions to round-trip, got %+v", channelStatus.Channels[0].RemediationActions)
	}
	if len(channelStatus.Channels[0].ConfigFieldDescriptors) != 1 || channelStatus.Channels[0].ConfigFieldDescriptors[0].Key != "enabled" {
		t.Fatalf("expected channel config field descriptors to round-trip, got %+v", channelStatus.Channels[0].ConfigFieldDescriptors)
	}
	if uiStatus.lastChannelReq.WorkspaceID != "ws1" {
		t.Fatalf("expected channel status workspace ws1, got %s", uiStatus.lastChannelReq.WorkspaceID)
	}

	connectorStatus, err := client.ConnectorStatus(context.Background(), ConnectorStatusRequest{WorkspaceID: "ws1"}, "corr-connector-status")
	if err != nil {
		t.Fatalf("connector status: %v", err)
	}
	if len(connectorStatus.Connectors) != 1 || connectorStatus.Connectors[0].ConnectorID != "mail" {
		t.Fatalf("unexpected connector status payload: %+v", connectorStatus)
	}
	if len(connectorStatus.Connectors[0].RemediationActions) != 1 ||
		connectorStatus.Connectors[0].RemediationActions[0].Identifier != "refresh_connector_status" {
		t.Fatalf("expected connector status remediation actions to round-trip, got %+v", connectorStatus.Connectors[0].RemediationActions)
	}
	if len(connectorStatus.Connectors[0].ConfigFieldDescriptors) != 1 || connectorStatus.Connectors[0].ConfigFieldDescriptors[0].Key != "scope" {
		t.Fatalf("expected connector config field descriptors to round-trip, got %+v", connectorStatus.Connectors[0].ConfigFieldDescriptors)
	}
	if uiStatus.lastConnectorReq.WorkspaceID != "ws1" {
		t.Fatalf("expected connector status workspace ws1, got %s", uiStatus.lastConnectorReq.WorkspaceID)
	}

	connectorPermission, err := client.ConnectorPermissionRequest(context.Background(), ConnectorPermissionRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
	}, "corr-connector-permission")
	if err != nil {
		t.Fatalf("connector permission request: %v", err)
	}
	if connectorPermission.WorkspaceID != "ws1" || connectorPermission.ConnectorID != "mail" || connectorPermission.PermissionState != "granted" {
		t.Fatalf("unexpected connector permission payload: %+v", connectorPermission)
	}
	if uiStatus.lastConnectorPermissionReq.WorkspaceID != "ws1" || uiStatus.lastConnectorPermissionReq.ConnectorID != "mail" {
		t.Fatalf("unexpected connector permission request payload: %+v", uiStatus.lastConnectorPermissionReq)
	}

	channelConfig, err := client.ChannelConfigUpsert(context.Background(), ChannelConfigUpsertRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app_chat",
		Configuration: UIStatusConfigurationFromMap(map[string]any{
			"enabled": true,
		}),
		Merge: true,
	}, "corr-channel-config-upsert")
	if err != nil {
		t.Fatalf("channel config upsert: %v", err)
	}
	if channelConfig.WorkspaceID != "ws1" || channelConfig.ChannelID != "app_chat" {
		t.Fatalf("unexpected channel config upsert payload: %+v", channelConfig)
	}
	if uiStatus.lastChannelConfigReq.WorkspaceID != "ws1" || uiStatus.lastChannelConfigReq.ChannelID != "app_chat" || !uiStatus.lastChannelConfigReq.Merge {
		t.Fatalf("unexpected channel config request payload: %+v", uiStatus.lastChannelConfigReq)
	}

	connectorConfig, err := client.ConnectorConfigUpsert(context.Background(), ConnectorConfigUpsertRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Configuration: UIStatusConfigurationFromMap(map[string]any{
			"scope": "inbox",
		}),
	}, "corr-connector-config-upsert")
	if err != nil {
		t.Fatalf("connector config upsert: %v", err)
	}
	if connectorConfig.WorkspaceID != "ws1" || connectorConfig.ConnectorID != "mail" {
		t.Fatalf("unexpected connector config upsert payload: %+v", connectorConfig)
	}
	if uiStatus.lastConnectorConfigReq.WorkspaceID != "ws1" || uiStatus.lastConnectorConfigReq.ConnectorID != "mail" {
		t.Fatalf("unexpected connector config request payload: %+v", uiStatus.lastConnectorConfigReq)
	}

	channelTest, err := client.ChannelTestOperation(context.Background(), ChannelTestOperationRequest{
		WorkspaceID: "ws1",
		ChannelID:   "app_chat",
		Operation:   "health",
	}, "corr-channel-test")
	if err != nil {
		t.Fatalf("channel test operation: %v", err)
	}
	if !channelTest.Success || channelTest.Status != "ok" {
		t.Fatalf("unexpected channel test payload: %+v", channelTest)
	}
	if uiStatus.lastChannelTestReq.WorkspaceID != "ws1" || uiStatus.lastChannelTestReq.ChannelID != "app_chat" {
		t.Fatalf("unexpected channel test request payload: %+v", uiStatus.lastChannelTestReq)
	}

	connectorTest, err := client.ConnectorTestOperation(context.Background(), ConnectorTestOperationRequest{
		WorkspaceID: "ws1",
		ConnectorID: "mail",
		Operation:   "health",
	}, "corr-connector-test")
	if err != nil {
		t.Fatalf("connector test operation: %v", err)
	}
	if !connectorTest.Success || connectorTest.Status != "ok" {
		t.Fatalf("unexpected connector test payload: %+v", connectorTest)
	}
	if uiStatus.lastConnectorTestReq.WorkspaceID != "ws1" || uiStatus.lastConnectorTestReq.ConnectorID != "mail" {
		t.Fatalf("unexpected connector test request payload: %+v", uiStatus.lastConnectorTestReq)
	}
}

func TestTransportChannelAndConnectorStatusDescriptorMetadataDefaults(t *testing.T) {
	uiStatus := &uiStatusServiceStub{
		channelResponse: ChannelStatusResponse{
			WorkspaceID: "ws1",
			Channels: []ChannelStatusCard{
				{
					ChannelID:   "app_chat",
					DisplayName: "App Chat",
					Status:      "ready",
					Configured:  true,
					Enabled:     true,
					ConfigFieldDescriptors: []ConfigFieldDescriptor{
						{
							Key:      "enabled",
							Label:    "Enabled",
							Type:     "bool",
							Required: true,
							Editable: true,
						},
					},
				},
			},
		},
		connectorResponse: ConnectorStatusResponse{
			WorkspaceID: "ws1",
			Connectors: []ConnectorStatusCard{
				{
					ConnectorID: "mail",
					DisplayName: "Mail Connector",
					PluginID:    "mail.daemon",
					Status:      "ready",
					Configured:  true,
					Enabled:     true,
					ConfigFieldDescriptors: []ConfigFieldDescriptor{
						{
							Key:         "scope",
							Label:       "Scope",
							Type:        "enum",
							Required:    false,
							Editable:    true,
							EnumOptions: []string{"inbox", "all"},
							Secret:      true,
							WriteOnly:   true,
							HelpText:    "Choose which mailbox scope to query.",
						},
					},
				},
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "ui-status-token",
		UIStatus:     uiStatus,
	})

	httpClient := &http.Client{Timeout: 2 * time.Second}
	doStatusRequest := func(path string) map[string]any {
		t.Helper()

		requestBody := bytes.NewBufferString(`{"workspace_id":"ws1"}`)
		request, err := http.NewRequest(http.MethodPost, "http://"+server.Address()+path, requestBody)
		if err != nil {
			t.Fatalf("build %s request: %v", path, err)
		}
		request.Header.Set("Authorization", "Bearer ui-status-token")
		request.Header.Set("Content-Type", "application/json")

		response, err := httpClient.Do(request)
		if err != nil {
			t.Fatalf("execute %s request: %v", path, err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			var errPayload map[string]any
			_ = json.NewDecoder(response.Body).Decode(&errPayload)
			t.Fatalf("expected %s status 200, got %d payload=%v", path, response.StatusCode, errPayload)
		}

		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode %s payload: %v", path, err)
		}
		return payload
	}

	channelPayload := doStatusRequest("/v1/channels/status")
	channelsRaw, ok := channelPayload["channels"].([]any)
	if !ok || len(channelsRaw) == 0 {
		t.Fatalf("expected non-empty channels array, got %T (%+v)", channelPayload["channels"], channelPayload["channels"])
	}
	channelCard, ok := channelsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected channel card object, got %T", channelsRaw[0])
	}
	channelDescriptorsRaw, ok := channelCard["config_field_descriptors"].([]any)
	if !ok || len(channelDescriptorsRaw) == 0 {
		t.Fatalf("expected non-empty channel descriptor array, got %T (%+v)", channelCard["config_field_descriptors"], channelCard["config_field_descriptors"])
	}
	channelDescriptor, ok := channelDescriptorsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected channel descriptor object, got %T", channelDescriptorsRaw[0])
	}
	assertConfigFieldDescriptorPayload(t, channelDescriptor, configFieldDescriptorContractExpectation{
		Required:    true,
		EnumOptions: []string{},
		Secret:      false,
		WriteOnly:   false,
		HelpText:    "",
	})

	connectorPayload := doStatusRequest("/v1/connectors/status")
	connectorsRaw, ok := connectorPayload["connectors"].([]any)
	if !ok || len(connectorsRaw) == 0 {
		t.Fatalf("expected non-empty connectors array, got %T (%+v)", connectorPayload["connectors"], connectorPayload["connectors"])
	}
	connectorCard, ok := connectorsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected connector card object, got %T", connectorsRaw[0])
	}
	connectorDescriptorsRaw, ok := connectorCard["config_field_descriptors"].([]any)
	if !ok || len(connectorDescriptorsRaw) == 0 {
		t.Fatalf("expected non-empty connector descriptor array, got %T (%+v)", connectorCard["config_field_descriptors"], connectorCard["config_field_descriptors"])
	}
	connectorDescriptor, ok := connectorDescriptorsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected connector descriptor object, got %T", connectorDescriptorsRaw[0])
	}
	assertConfigFieldDescriptorPayload(t, connectorDescriptor, configFieldDescriptorContractExpectation{
		Required:    false,
		EnumOptions: []string{"inbox", "all"},
		Secret:      true,
		WriteOnly:   true,
		HelpText:    "Choose which mailbox scope to query.",
	})
}

func TestTransportChatTurnRouteIncludesTaskRunCorrelationMetadata(t *testing.T) {
	chat := &chatServiceStub{
		response: ChatTurnResponse{
			WorkspaceID: "ws1",
			TaskClass:   "chat",
			Provider:    "openai",
			ModelKey:    "gpt-4.1-mini",
			Items: []ChatTurnItem{
				{
					ItemID:   "item-assistant-1",
					Type:     "assistant_message",
					Role:     "assistant",
					Status:   "completed",
					Content:  "hello from stub",
					Metadata: ChatTurnItemMetadataFromMap(map[string]any{"source": "stub"}),
				},
			},
			TaskRunCorrelation: ChatTurnTaskRunCorrelation{
				Available: true,
				Source:    "audit_log_entry",
				TaskID:    "task-1",
				RunID:     "run-1",
				TaskState: "running",
				RunState:  "running",
			},
		},
	}

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	response, err := client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Status: "completed", Content: "hello"},
		},
	}, "corr-chat-turn")
	if err != nil {
		t.Fatalf("chat turn: %v", err)
	}
	if response.CorrelationID != "corr-chat-turn" {
		t.Fatalf("expected correlation id corr-chat-turn, got %s", response.CorrelationID)
	}
	if !response.TaskRunCorrelation.Available ||
		response.TaskRunCorrelation.TaskID != "task-1" ||
		response.TaskRunCorrelation.RunID != "run-1" {
		t.Fatalf("unexpected chat task/run correlation payload: %+v", response.TaskRunCorrelation)
	}
	if len(response.Items) != 1 || strings.TrimSpace(response.Items[0].Type) != "assistant_message" {
		t.Fatalf("expected canonical assistant turn item in response, got %+v", response.Items)
	}
	if chat.lastCorrelationID != "corr-chat-turn" {
		t.Fatalf("expected forwarded correlation id corr-chat-turn, got %s", chat.lastCorrelationID)
	}
	if chat.lastRequest.WorkspaceID != "ws1" || len(chat.lastRequest.Items) != 1 {
		t.Fatalf("unexpected forwarded chat request payload: %+v", chat.lastRequest)
	}
	if strings.TrimSpace(response.ContractVersion) != ChatTurnContractVersionV2 {
		t.Fatalf("expected chat turn contract version %q, got %q", ChatTurnContractVersionV2, response.ContractVersion)
	}
	if strings.TrimSpace(response.TurnItemSchemaVersion) != ChatTurnItemSchemaVersionV1 {
		t.Fatalf("expected turn item schema version %q, got %q", ChatTurnItemSchemaVersionV1, response.TurnItemSchemaVersion)
	}
	if strings.TrimSpace(response.RealtimeEventContractVersion) != ChatRealtimeLifecycleContractVersionV2 {
		t.Fatalf("expected realtime event contract version %q, got %q", ChatRealtimeLifecycleContractVersionV2, response.RealtimeEventContractVersion)
	}
}

func TestTransportChatTurnRouteNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	_, err = client.ChatTurn(context.Background(), ChatTurnRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
		Items: []ChatTurnItem{
			{Type: "user_message", Role: "user", Status: "completed", Content: "hello"},
		},
	}, "corr-chat-turn")
	if err == nil {
		t.Fatalf("expected chat turn to fail when chat service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportChatTurnExplainRoute(t *testing.T) {
	chat := &chatServiceStub{
		explainResponse: ChatTurnExplainResponse{
			WorkspaceID:     "ws1",
			TaskClass:       "chat",
			ContractVersion: ChatTurnExplainContractVersionV1,
			SelectedRoute: ModelRouteExplainResponse{
				WorkspaceID:      "ws1",
				TaskClass:        "chat",
				SelectedProvider: "openai",
				SelectedModelKey: "gpt-4.1-mini",
				SelectedSource:   "task_policy",
				Summary:          "selected route explanation",
				Explanations:     []string{"task-class policy selected openai/gpt-4.1-mini"},
			},
			ToolCatalog: []ChatTurnToolCatalogEntry{
				{
					Name: "mail_send",
					InputSchema: map[string]any{
						"type": "object",
					},
				},
			},
			PolicyDecisions: []ChatTurnToolPolicyDecision{
				{
					ToolName: "mail_send",
					Decision: "ALLOW",
					Reason:   "tool execution allowed",
				},
			},
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	response, err := client.ChatTurnExplain(context.Background(), ChatTurnExplainRequest{
		WorkspaceID:        "ws1",
		TaskClass:          "chat",
		RequestedByActorID: "actor.requester",
		ActingAsActorID:    "actor.requester",
		Channel: ChatTurnChannelContext{
			ChannelID: "app",
		},
	}, "corr-chat-explain")
	if err != nil {
		t.Fatalf("chat turn explain: %v", err)
	}
	if strings.TrimSpace(response.ContractVersion) != ChatTurnExplainContractVersionV1 {
		t.Fatalf("expected contract version %q, got %q", ChatTurnExplainContractVersionV1, response.ContractVersion)
	}
	if strings.TrimSpace(response.SelectedRoute.SelectedProvider) != "openai" {
		t.Fatalf("expected selected provider openai, got %q", response.SelectedRoute.SelectedProvider)
	}
	if len(response.ToolCatalog) != 1 || strings.TrimSpace(response.ToolCatalog[0].Name) != "mail_send" {
		t.Fatalf("unexpected tool catalog payload: %+v", response.ToolCatalog)
	}
	if len(response.PolicyDecisions) != 1 || strings.TrimSpace(response.PolicyDecisions[0].Decision) != "ALLOW" {
		t.Fatalf("unexpected policy decision payload: %+v", response.PolicyDecisions)
	}
	if strings.TrimSpace(chat.lastExplainRequest.WorkspaceID) != "ws1" || strings.TrimSpace(chat.lastExplainRequest.TaskClass) != "chat" {
		t.Fatalf("unexpected forwarded explain request payload: %+v", chat.lastExplainRequest)
	}
}

func TestTransportChatTurnExplainRouteNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	_, err = client.ChatTurnExplain(context.Background(), ChatTurnExplainRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	}, "corr-chat-explain")
	if err == nil {
		t.Fatalf("expected chat turn explain to fail when explain service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportChatTurnHistoryRoute(t *testing.T) {
	chat := &chatServiceStub{
		historyResponse: ChatTurnHistoryResponse{
			WorkspaceID: "ws1",
			Items: []ChatTurnHistoryRecord{
				{
					RecordID:      "turnitem-1",
					TurnID:        "turn-1",
					WorkspaceID:   "ws1",
					TaskClass:     "chat",
					CorrelationID: "corr-1",
					ChannelID:     "message",
					ConnectorID:   "twilio",
					ThreadID:      "thread-1",
					ItemIndex:     0,
					Item: ChatTurnItem{
						ItemID:  "item-1",
						Type:    "assistant_message",
						Role:    "assistant",
						Status:  "completed",
						Content: "history reply",
					},
					TaskRunReference: ChatTurnTaskRunCorrelation{
						Available: true,
						Source:    "turn_ledger",
						TaskID:    "task-1",
						RunID:     "run-1",
					},
					CreatedAt: "2026-02-27T00:00:00Z",
				},
			},
			HasMore:             true,
			NextCursorCreatedAt: "2026-02-27T00:00:00Z",
			NextCursorItemID:    "turnitem-1",
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	response, err := client.ChatTurnHistory(context.Background(), ChatTurnHistoryRequest{
		WorkspaceID:   "ws1",
		ChannelID:     "message",
		ConnectorID:   "twilio",
		ThreadID:      "thread-1",
		CorrelationID: "corr-1",
		Limit:         25,
	}, "corr-history")
	if err != nil {
		t.Fatalf("chat turn history: %v", err)
	}
	if response.WorkspaceID != "ws1" || len(response.Items) != 1 || !response.HasMore {
		t.Fatalf("unexpected history response payload: %+v", response)
	}
	if chat.lastHistoryRequest.WorkspaceID != "ws1" || chat.lastHistoryRequest.Limit != 25 {
		t.Fatalf("unexpected forwarded history request payload: %+v", chat.lastHistoryRequest)
	}
	if chat.lastHistoryRequest.ThreadID != "thread-1" || chat.lastHistoryRequest.CorrelationID != "corr-1" {
		t.Fatalf("expected forwarded thread/correlation filters, got %+v", chat.lastHistoryRequest)
	}
}

func TestTransportChatTurnHistoryRouteNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	_, err = client.ChatTurnHistory(context.Background(), ChatTurnHistoryRequest{WorkspaceID: "ws1"}, "corr-history")
	if err == nil {
		t.Fatalf("expected history route to fail when chat history service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}

func TestTransportChatPersonaPolicyRoutes(t *testing.T) {
	chat := &chatServiceStub{
		personaGetResponse: ChatPersonaPolicyResponse{
			WorkspaceID:      "ws1",
			PrincipalActorID: "actor.a",
			ChannelID:        "message",
			StylePrompt:      "Be concise.",
			Guardrails:       []string{"Confirm tool outcomes before claiming success."},
			Source:           "persisted",
			UpdatedAt:        "2026-02-27T00:00:00Z",
		},
		personaSetResponse: ChatPersonaPolicyResponse{
			WorkspaceID:      "ws1",
			PrincipalActorID: "actor.a",
			ChannelID:        "message",
			StylePrompt:      "Be concise.",
			Guardrails:       []string{"Confirm tool outcomes before claiming success."},
			Source:           "persisted",
			UpdatedAt:        "2026-02-27T00:01:00Z",
		},
	}
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
		Chat:         chat,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	getResponse, err := client.GetChatPersonaPolicy(context.Background(), ChatPersonaPolicyRequest{
		WorkspaceID:      "ws1",
		PrincipalActorID: "actor.a",
		ChannelID:        "message",
	}, "corr-persona-get")
	if err != nil {
		t.Fatalf("get chat persona policy: %v", err)
	}
	if getResponse.Source != "persisted" || getResponse.StylePrompt != "Be concise." {
		t.Fatalf("unexpected chat persona get response: %+v", getResponse)
	}
	if chat.lastPersonaGetRequest.WorkspaceID != "ws1" || chat.lastPersonaGetRequest.PrincipalActorID != "actor.a" {
		t.Fatalf("unexpected forwarded persona get request payload: %+v", chat.lastPersonaGetRequest)
	}

	setResponse, err := client.UpsertChatPersonaPolicy(context.Background(), ChatPersonaPolicyUpsertRequest{
		WorkspaceID:      "ws1",
		PrincipalActorID: "actor.a",
		ChannelID:        "message",
		StylePrompt:      "Be concise.",
		Guardrails:       []string{"Confirm tool outcomes before claiming success."},
	}, "corr-persona-set")
	if err != nil {
		t.Fatalf("upsert chat persona policy: %v", err)
	}
	if setResponse.Source != "persisted" || setResponse.UpdatedAt == "" {
		t.Fatalf("unexpected chat persona set response: %+v", setResponse)
	}
	if chat.lastPersonaSetRequest.WorkspaceID != "ws1" || chat.lastPersonaSetRequest.StylePrompt != "Be concise." {
		t.Fatalf("unexpected forwarded persona set request payload: %+v", chat.lastPersonaSetRequest)
	}
}

func TestTransportChatPersonaPolicyRoutesNotImplementedWithoutService(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "chat-token",
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "chat-token",
	})
	if err != nil {
		t.Fatalf("create chat client: %v", err)
	}

	_, err = client.GetChatPersonaPolicy(context.Background(), ChatPersonaPolicyRequest{WorkspaceID: "ws1"}, "corr-persona-get")
	if err == nil {
		t.Fatalf("expected persona get route to fail when persona policy service is not configured")
	}
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 501 {
		t.Fatalf("expected status 501, got %d", httpErr.StatusCode)
	}
}
