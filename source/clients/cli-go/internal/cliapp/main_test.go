package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	"personalagent/runtime/internal/controlauth"
	"personalagent/runtime/internal/daemonruntime"
	"personalagent/runtime/internal/securestore"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"

	_ "modernc.org/sqlite"
)

func setTestSecretManager(t *testing.T, manager *securestore.Manager) {
	t.Helper()
	previousFactory := newSecretManager
	newSecretManager = func() (*securestore.Manager, error) { return manager, nil }
	t.Cleanup(func() {
		newSecretManager = previousFactory
	})
}

func setTestChatInput(t *testing.T, input string) {
	t.Helper()
	previousInput := chatInput
	chatInput = strings.NewReader(input)
	t.Cleanup(func() {
		chatInput = previousInput
	})
}

func withDaemonArgs(server *transport.Server, args ...string) []string {
	base := []string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
	}
	return append(base, args...)
}

type cliDoctorLifecycleServiceStub struct {
	status transport.DaemonLifecycleStatusResponse
}

func (s *cliDoctorLifecycleServiceStub) DaemonLifecycleStatus(context.Context) (transport.DaemonLifecycleStatusResponse, error) {
	return s.status, nil
}

func (s *cliDoctorLifecycleServiceStub) DaemonLifecycleControl(context.Context, transport.DaemonLifecycleControlRequest) (transport.DaemonLifecycleControlResponse, error) {
	return transport.DaemonLifecycleControlResponse{}, fmt.Errorf("not implemented in test stub")
}

func (s *cliDoctorLifecycleServiceStub) DaemonPluginLifecycleHistory(context.Context, transport.DaemonPluginLifecycleHistoryRequest) (transport.DaemonPluginLifecycleHistoryResponse, error) {
	return transport.DaemonPluginLifecycleHistoryResponse{}, fmt.Errorf("not implemented in test stub")
}

func modelListContainsModel(items []any, provider string, modelKey string) bool {
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		recordProvider, _ := record["provider"].(string)
		recordModelKey, _ := record["model_key"].(string)
		if strings.EqualFold(strings.TrimSpace(recordProvider), strings.TrimSpace(provider)) &&
			strings.EqualFold(strings.TrimSpace(recordModelKey), strings.TrimSpace(modelKey)) {
			return true
		}
	}
	return false
}

func quickstartStepByID(t *testing.T, payload map[string]any, stepID string) map[string]any {
	t.Helper()
	stepsRaw, ok := payload["steps"].([]any)
	if !ok {
		t.Fatalf("expected quickstart steps array, got %T", payload["steps"])
	}
	for _, raw := range stepsRaw {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(record["id"])) == strings.TrimSpace(stepID) {
			return record
		}
	}
	t.Fatalf("quickstart step %q not found in payload: %+v", stepID, payload["steps"])
	return nil
}

func startCLITestServer(t *testing.T) *transport.Server {
	t.Helper()
	broker := transport.NewEventBroker()
	backend := transport.NewInMemoryControlBackend(broker)
	server, err := transport.NewServer(transport.ServerConfig{
		ListenerMode:     transport.ListenerModeTCP,
		Address:          "127.0.0.1:0",
		AuthToken:        "cli-test-token",
		SecretReferences: transport.NewInMemorySecretReferenceService(),
	}, backend, broker)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server
}

func startCLITestServerWithLifecycle(t *testing.T, lifecycle transport.DaemonLifecycleService) *transport.Server {
	t.Helper()
	broker := transport.NewEventBroker()
	backend := transport.NewInMemoryControlBackend(broker)
	server, err := transport.NewServer(transport.ServerConfig{
		ListenerMode:     transport.ListenerModeTCP,
		Address:          "127.0.0.1:0",
		AuthToken:        "cli-test-token",
		SecretReferences: transport.NewInMemorySecretReferenceService(),
		DaemonLifecycle:  lifecycle,
	}, backend, broker)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server
}

func startCLITestServerWithDaemonServices(t *testing.T, dbPath string, manager *securestore.Manager) *transport.Server {
	t.Helper()
	if strings.TrimSpace(dbPath) == "" {
		dbPath = filepath.Join(t.TempDir(), "daemon-runtime.db")
	}
	if manager == nil {
		defaultManager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
		if err != nil {
			t.Fatalf("new default secret manager: %v", err)
		}
		manager = defaultManager
	}
	workerSpecs := cliDispatchWorkerSpecs(t, dbPath)
	container, err := daemonruntime.NewServiceContainer(context.Background(), daemonruntime.ServiceContainerConfig{
		DBPath: dbPath,
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginWorkers: workerSpecs,
	})
	if err != nil {
		t.Fatalf("new daemon service container: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = container.Close(ctx)
	})

	service := daemonruntime.NewProviderModelChatService(container)
	agentDelegation, err := daemonruntime.NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new daemon agent/delegation service: %v", err)
	}
	commTwilio, err := daemonruntime.NewCommTwilioService(container)
	if err != nil {
		t.Fatalf("new daemon comm/twilio service: %v", err)
	}
	commTwilio.SetAssistantChatService(service)
	opsService, err := daemonruntime.NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new daemon automation/inspect/retention/context service: %v", err)
	}
	identityService, err := daemonruntime.NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new daemon identity directory service: %v", err)
	}
	uiStatusService, err := daemonruntime.NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new daemon ui status service: %v", err)
	}
	broker := transport.NewEventBroker()
	backend, err := daemonruntime.NewPersistedControlBackend(container, agentDelegation, broker)
	if err != nil {
		t.Fatalf("new daemon control backend: %v", err)
	}
	server, err := transport.NewServer(transport.ServerConfig{
		ListenerMode:      transport.ListenerModeTCP,
		Address:           "127.0.0.1:0",
		AuthToken:         "cli-test-token",
		SecretReferences:  transport.NewInMemorySecretReferenceService(),
		Providers:         service,
		Models:            service,
		Chat:              service,
		Agent:             agentDelegation,
		Delegation:        agentDelegation,
		Comm:              commTwilio,
		Twilio:            commTwilio,
		Automation:        opsService,
		Inspect:           opsService,
		Retention:         opsService,
		ContextOps:        opsService,
		IdentityDirectory: identityService,
		UIStatus:          uiStatusService,
	}, backend, broker)
	if err != nil {
		t.Fatalf("new daemon transport server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start daemon transport server: %v", err)
	}
	waitForCLITestWorkersReady(t, container.PluginSupervisor, workerSpecs, 4*time.Second)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server
}

func waitForCLITestWorkersReady(t *testing.T, supervisor daemonruntime.PluginSupervisor, specs []daemonruntime.PluginWorkerSpec, timeout time.Duration) {
	t.Helper()
	if supervisor == nil || len(specs) == 0 {
		return
	}
	if timeout <= 0 {
		timeout = 4 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		readyCount := 0
		for _, spec := range specs {
			status, ok := supervisor.WorkerStatus(spec.PluginID)
			if !ok {
				continue
			}
			if status.State != daemonruntime.PluginWorkerStateRunning {
				continue
			}
			if strings.TrimSpace(status.Metadata.Runtime[cliDispatchWorkerExecAddressKey]) == "" {
				continue
			}
			readyCount++
		}
		if readyCount == len(specs) {
			return
		}
		if time.Now().After(deadline) {
			statuses := supervisor.ListWorkers()
			t.Fatalf("timed out waiting for test workers readiness: ready=%d total=%d statuses=%+v", readyCount, len(specs), statuses)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

const (
	cliDispatchWorkerProcessEnv     = "PA_CLI_DISPATCH_WORKER_PROCESS"
	cliDispatchWorkerExecAddressKey = "exec_address"
	cliTwilioWorkerCapabilityCheck  = "channel.twilio.check"
	cliTwilioWorkerCapabilitySMS    = "channel.twilio.sms.send"
	cliTwilioWorkerCapabilityVoice  = "channel.twilio.voice.start_call"
	cliMessagesWorkerCapabilitySend = "channel.messages.send"
	cliMessagesWorkerCapabilityPoll = "channel.messages.ingest_poll"
)

func cliDispatchWorkerSpecs(t *testing.T, dbPath string) []daemonruntime.PluginWorkerSpec {
	t.Helper()
	_ = dbPath
	specs := []daemonruntime.PluginWorkerSpec{
		cliConnectorWorkerSpec("mail.daemon", []string{
			mailadapter.CapabilityDraft,
			mailadapter.CapabilitySend,
			mailadapter.CapabilityReply,
		}),
		cliConnectorWorkerSpec("calendar.daemon", []string{
			calendaradapter.CapabilityCreate,
			calendaradapter.CapabilityUpdate,
			calendaradapter.CapabilityCancel,
		}),
		cliConnectorWorkerSpec("browser.daemon", []string{
			browseradapter.CapabilityOpen,
			browseradapter.CapabilityExtract,
			browseradapter.CapabilityClose,
		}),
		cliConnectorWorkerSpec("finder.daemon", []string{
			finderadapter.CapabilityList,
			finderadapter.CapabilityPreview,
			finderadapter.CapabilityDelete,
		}),
	}

	twilioSpec := cliConnectorWorkerSpec("twilio.daemon", []string{
		cliTwilioWorkerCapabilityCheck,
		cliTwilioWorkerCapabilitySMS,
		cliTwilioWorkerCapabilityVoice,
	})
	specs = append(specs, twilioSpec)
	specs = append(specs, cliConnectorWorkerSpec("messages.daemon", []string{
		cliMessagesWorkerCapabilitySend,
		cliMessagesWorkerCapabilityPoll,
	}))
	return specs
}

func cliConnectorWorkerSpec(pluginID string, capabilities []string) daemonruntime.PluginWorkerSpec {
	return daemonruntime.PluginWorkerSpec{
		PluginID: pluginID,
		Kind:     shared.AdapterKindConnector,
		Command:  os.Args[0],
		Args: []string{
			"-test.run=TestCLIDispatchWorkerHelperProcess",
			"--",
			string(shared.AdapterKindConnector),
			pluginID,
			strings.Join(capabilities, ","),
		},
		Env: []string{
			cliDispatchWorkerProcessEnv + "=1",
			"PA_MAIL_AUTOMATION_DRY_RUN=1",
		},
		HandshakeTimeout: 4 * time.Second,
		HealthInterval:   500 * time.Millisecond,
		HealthTimeout:    2 * time.Second,
		RestartPolicy: daemonruntime.PluginRestartPolicy{
			MaxRestarts: 3,
			Delay:       100 * time.Millisecond,
		},
	}
}

func cliChannelWorkerSpec(pluginID string, capabilities []string) daemonruntime.PluginWorkerSpec {
	return daemonruntime.PluginWorkerSpec{
		PluginID: pluginID,
		Kind:     shared.AdapterKindChannel,
		Command:  os.Args[0],
		Args: []string{
			"-test.run=TestCLIDispatchWorkerHelperProcess",
			"--",
			string(shared.AdapterKindChannel),
			pluginID,
			strings.Join(capabilities, ","),
		},
		Env:              []string{cliDispatchWorkerProcessEnv + "=1"},
		HandshakeTimeout: 4 * time.Second,
		HealthInterval:   500 * time.Millisecond,
		HealthTimeout:    2 * time.Second,
		RestartPolicy: daemonruntime.PluginRestartPolicy{
			MaxRestarts: 3,
			Delay:       100 * time.Millisecond,
		},
	}
}

type cliDispatchWorkerMessage struct {
	Type    string                  `json:"type"`
	Plugin  *shared.AdapterMetadata `json:"plugin,omitempty"`
	Healthy *bool                   `json:"healthy,omitempty"`
}

type cliConnectorWorkerExecuteRequest struct {
	ExecutionContext connectorcontract.ExecutionContext `json:"execution_context"`
	Step             connectorcontract.TaskStep         `json:"step"`
}

type cliChannelWorkerExecuteRequest struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

type cliChannelWorkerExecuteResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func TestCLIDispatchWorkerHelperProcess(t *testing.T) {
	if os.Getenv(cliDispatchWorkerProcessEnv) != "1" {
		return
	}

	args := flag.Args()
	if len(args) < 3 {
		_, _ = fmt.Fprintf(os.Stderr, "invalid helper args: %v\n", args)
		os.Exit(2)
	}

	kind := shared.AdapterKind(strings.TrimSpace(args[0]))
	pluginID := strings.TrimSpace(args[1])
	capabilities := splitCapabilities(strings.TrimSpace(args[2]))

	var err error
	switch kind {
	case shared.AdapterKindConnector:
		err = runCLITestConnectorWorker(pluginID, capabilities)
	case shared.AdapterKindChannel:
		err = runCLITestChannelWorker(pluginID, capabilities)
	default:
		err = fmt.Errorf("unsupported helper worker kind %q", kind)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "helper worker failed: %v\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func runCLITestConnectorWorker(pluginID string, capabilityKeys []string) error {
	if strings.EqualFold(strings.TrimSpace(pluginID), "twilio.daemon") {
		return runCLITestTwilioConnectorWorker(pluginID, capabilityKeys)
	}
	if strings.EqualFold(strings.TrimSpace(pluginID), "messages.daemon") {
		return runCLITestMessagesConnectorWorker(pluginID, capabilityKeys)
	}

	adapter, err := cliConnectorAdapter(pluginID)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen connector worker: %w", err)
	}
	defer listener.Close()

	metadata := adapter.Metadata()
	metadata.ID = pluginID
	metadata.Kind = shared.AdapterKindConnector
	metadata.DisplayName = pluginID
	metadata.Version = "test"
	if len(capabilityKeys) > 0 {
		metadata.Capabilities = make([]shared.CapabilityDescriptor, 0, len(capabilityKeys))
		for _, capability := range capabilityKeys {
			trimmed := strings.TrimSpace(capability)
			if trimmed == "" {
				continue
			}
			metadata.Capabilities = append(metadata.Capabilities, shared.CapabilityDescriptor{Key: trimmed})
		}
	}
	if metadata.Runtime == nil {
		metadata.Runtime = map[string]string{}
	}
	metadata.Runtime[cliDispatchWorkerExecAddressKey] = listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var payload cliConnectorWorkerExecuteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": err.Error()})
			return
		}

		result, execErr := adapter.ExecuteStep(request.Context(), payload.ExecutionContext, payload.Step)
		if execErr != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": execErr.Error()})
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(result)
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	if err := emitCLITestWorkerMessage(cliDispatchWorkerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
		return err
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		healthy := true
		_ = emitCLITestWorkerMessage(cliDispatchWorkerMessage{Type: "health", Healthy: &healthy})
	}
}

func runCLITestTwilioConnectorWorker(pluginID string, capabilityKeys []string) error {
	return runCLITestChannelOperationConnectorWorker(pluginID, capabilityKeys)
}

func runCLITestMessagesConnectorWorker(pluginID string, capabilityKeys []string) error {
	return runCLITestChannelOperationConnectorWorker(pluginID, capabilityKeys)
}

func runCLITestChannelOperationConnectorWorker(pluginID string, capabilityKeys []string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen channel-operation connector worker: %w", err)
	}
	defer listener.Close()

	metadata := shared.AdapterMetadata{
		ID:          pluginID,
		Kind:        shared.AdapterKindConnector,
		DisplayName: pluginID,
		Version:     "test",
		Runtime: map[string]string{
			cliDispatchWorkerExecAddressKey: listener.Addr().String(),
		},
	}
	for _, capability := range capabilityKeys {
		trimmed := strings.TrimSpace(capability)
		if trimmed == "" {
			continue
		}
		metadata.Capabilities = append(metadata.Capabilities, shared.CapabilityDescriptor{Key: trimmed})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var payload cliChannelWorkerExecuteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeCLITestChannelWorkerError(writer, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}

		result, execErr := executeCLITestChannelOperation(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeCLITestChannelWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(cliChannelWorkerExecuteResponse{Result: result})
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	if err := emitCLITestWorkerMessage(cliDispatchWorkerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
		return err
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		healthy := true
		_ = emitCLITestWorkerMessage(cliDispatchWorkerMessage{Type: "health", Healthy: &healthy})
	}
}

func runCLITestChannelWorker(pluginID string, capabilityKeys []string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen channel worker: %w", err)
	}
	defer listener.Close()

	metadata := shared.AdapterMetadata{
		ID:          pluginID,
		Kind:        shared.AdapterKindChannel,
		DisplayName: pluginID,
		Version:     "test",
		Runtime: map[string]string{
			cliDispatchWorkerExecAddressKey: listener.Addr().String(),
		},
	}
	for _, capability := range capabilityKeys {
		trimmed := strings.TrimSpace(capability)
		if trimmed == "" {
			continue
		}
		metadata.Capabilities = append(metadata.Capabilities, shared.CapabilityDescriptor{Key: trimmed})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var payload cliChannelWorkerExecuteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeCLITestChannelWorkerError(writer, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}

		result, execErr := executeCLITestChannelOperation(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeCLITestChannelWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(cliChannelWorkerExecuteResponse{Result: result})
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	if err := emitCLITestWorkerMessage(cliDispatchWorkerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
		return err
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		healthy := true
		_ = emitCLITestWorkerMessage(cliDispatchWorkerMessage{Type: "health", Healthy: &healthy})
	}
}

func executeCLITestChannelOperation(ctx context.Context, operation string, payload json.RawMessage) (any, error) {
	switch operation {
	case "twilio_check":
		var request channelcheck.TwilioRequest
		if err := decodeCLITestPayload(payload, &request); err != nil {
			return nil, err
		}
		return channelcheck.CheckTwilio(ctx, http.DefaultClient, request)
	case "twilio_sms_send":
		var request twilioadapter.SMSAPIRequest
		if err := decodeCLITestPayload(payload, &request); err != nil {
			return nil, err
		}
		return twilioadapter.SendSMS(ctx, http.DefaultClient, request)
	case "twilio_voice_start_call":
		var request twilioadapter.VoiceCallRequest
		if err := decodeCLITestPayload(payload, &request); err != nil {
			return nil, err
		}
		return twilioadapter.StartVoiceCall(ctx, http.DefaultClient, request)
	case "messages_send":
		var request messagesadapter.SendRequest
		if err := decodeCLITestPayload(payload, &request); err != nil {
			return nil, err
		}
		return messagesadapter.SendResponse{
			WorkspaceID: strings.TrimSpace(request.WorkspaceID),
			Destination: strings.TrimSpace(request.Destination),
			MessageID:   "imessage-test-worker-1",
			Channel:     "imessage",
			Status:      "sent",
			Transport:   "worker-test",
		}, nil
	case "messages_poll_inbound":
		var request messagesadapter.InboundPollRequest
		if err := decodeCLITestPayload(payload, &request); err != nil {
			return nil, err
		}
		return messagesadapter.InboundPollResponse{
			WorkspaceID:  strings.TrimSpace(request.WorkspaceID),
			Source:       messagesadapter.SourceName,
			SourceScope:  firstNonEmpty(strings.TrimSpace(request.SourceScope), "cli-test-messages"),
			SourceDBPath: firstNonEmpty(strings.TrimSpace(request.SourceDBPath), "/tmp/cli-test-messages.db"),
			CursorStart:  strings.TrimSpace(request.SinceCursor),
			CursorEnd:    firstNonEmpty(strings.TrimSpace(request.SinceCursor), "0"),
			Polled:       0,
			Events:       nil,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported channel operation %q", operation)
	}
}

func cliConnectorAdapter(pluginID string) (connectorcontract.Adapter, error) {
	switch strings.TrimSpace(pluginID) {
	case "mail.daemon":
		return mailadapter.NewAdapter(pluginID), nil
	case "calendar.daemon":
		return calendaradapter.NewAdapter(pluginID), nil
	case "browser.daemon":
		return browseradapter.NewAdapter(pluginID), nil
	case "finder.daemon":
		return finderadapter.NewAdapter(pluginID), nil
	default:
		return nil, fmt.Errorf("unsupported connector worker plugin id %q", pluginID)
	}
}

func decodeCLITestPayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return fmt.Errorf("payload is required")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

func writeCLITestChannelWorkerError(writer http.ResponseWriter, statusCode int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(cliChannelWorkerExecuteResponse{
		Error: err.Error(),
	})
}

func emitCLITestWorkerMessage(message cliDispatchWorkerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(payload))
	return err
}

func splitCapabilities(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func TestRunSmokeCommand(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"smoke",
	}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", exitCode, stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode smoke response: %v", err)
	}
	if healthy, ok := response["healthy"].(bool); !ok || !healthy {
		t.Fatalf("expected healthy=true in smoke response, got %v", response["healthy"])
	}
}

func TestRunSmokeCommandJSONCompactOutput(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"smoke",
	}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.Contains(stdout.String(), "\n  ") {
		t.Fatalf("expected compact json output, got %q", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode compact smoke response: %v", err)
	}
	if healthy, ok := response["healthy"].(bool); !ok || !healthy {
		t.Fatalf("expected healthy=true in compact smoke response, got %v", response["healthy"])
	}
}

func TestRunTaskStatusCommandTextOutput(t *testing.T) {
	server := startCLITestServer(t)

	submitOut := &bytes.Buffer{}
	submitErr := &bytes.Buffer{}
	submitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"task", "submit",
		"--workspace", "ws1",
		"--requested-by", "actor.requester",
		"--subject", "actor.requester",
		"--title", "text output task",
	}, submitOut, submitErr)
	if submitCode != 0 {
		t.Fatalf("task submit failed: code=%d stderr=%s output=%s", submitCode, submitErr.String(), submitOut.String())
	}

	var submitResponse transport.SubmitTaskResponse
	if err := json.Unmarshal(submitOut.Bytes(), &submitResponse); err != nil {
		t.Fatalf("decode task submit response: %v", err)
	}
	if strings.TrimSpace(submitResponse.TaskID) == "" {
		t.Fatalf("expected submitted task id")
	}

	statusOut := &bytes.Buffer{}
	statusErr := &bytes.Buffer{}
	statusCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "text",
		"task", "status",
		"--task-id", submitResponse.TaskID,
	}, statusOut, statusErr)
	if statusCode != 0 {
		t.Fatalf("task status text failed: code=%d stderr=%s output=%s", statusCode, statusErr.String(), statusOut.String())
	}
	output := statusOut.String()
	if !strings.Contains(output, "task status") ||
		!strings.Contains(output, "task_id: "+submitResponse.TaskID) ||
		!strings.Contains(output, "actions:") {
		t.Fatalf("expected task status text output, got %q", output)
	}
	if json.Valid([]byte(strings.TrimSpace(output))) {
		t.Fatalf("expected text output for task status, got JSON payload %q", output)
	}
}

func TestRunProviderListCommandTextOutput(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	setTestSecretManager(t, manager)
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-text-provider"); err != nil {
		t.Fatalf("seed openai secret: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "provider-text-output.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, manager)

	setCode := run(withDaemonArgs(server,
		"provider", "set",
		"--workspace", "ws1",
		"--provider", "openai",
		"--endpoint", "http://127.0.0.1:18080/v1",
		"--api-key-secret", "OPENAI_API_KEY",
	), &bytes.Buffer{}, &bytes.Buffer{})
	if setCode != 0 {
		t.Fatalf("provider set failed for text-output test")
	}

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"--output", "text",
		"provider", "list",
		"--workspace", "ws1",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("provider list text failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}
	output := listOut.String()
	if !strings.Contains(output, "provider list") ||
		!strings.Contains(output, "workspace: ws1") ||
		!strings.Contains(output, "provider=openai") {
		t.Fatalf("expected provider text output, got %q", output)
	}
	if json.Valid([]byte(strings.TrimSpace(output))) {
		t.Fatalf("expected text output for provider list, got JSON payload %q", output)
	}
}

func TestRunModelListCommandTextOutput(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "model-text-output.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	listOut := &bytes.Buffer{}
	listErr := &bytes.Buffer{}
	listCode := run(withDaemonArgs(server,
		"--output", "text",
		"model", "list",
		"--workspace", "ws1",
	), listOut, listErr)
	if listCode != 0 {
		t.Fatalf("model list text failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}
	output := listOut.String()
	if !strings.Contains(output, "model list") ||
		!strings.Contains(output, "workspace: ws1") ||
		!strings.Contains(output, "models:") ||
		!strings.Contains(output, "provider=") {
		t.Fatalf("expected model text output, got %q", output)
	}
	if json.Valid([]byte(strings.TrimSpace(output))) {
		t.Fatalf("expected text output for model list, got JSON payload %q", output)
	}
}

func TestRunDoctorCommandTextOutput(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "doctor-text-output.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, nil)

	doctorOut := &bytes.Buffer{}
	doctorErr := &bytes.Buffer{}
	doctorCode := run(withDaemonArgs(server,
		"--output", "text",
		"doctor",
		"--workspace", "ws1",
	), doctorOut, doctorErr)
	if doctorCode != 0 && doctorCode != 1 {
		t.Fatalf("doctor text expected exit code 0/1, got %d stderr=%s output=%s", doctorCode, doctorErr.String(), doctorOut.String())
	}
	output := doctorOut.String()
	if !strings.Contains(output, "doctor report") ||
		!strings.Contains(output, "overall_status:") ||
		!strings.Contains(output, "daemon.connectivity") {
		t.Fatalf("expected doctor text output, got %q", output)
	}
	if json.Valid([]byte(strings.TrimSpace(output))) {
		t.Fatalf("expected text output for doctor report, got JSON payload %q", output)
	}
}

func TestRunSmokeCommandMachineErrorJSONOutput(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"--error-output", "json",
		"task", "status",
		"--task-id", "task-does-not-exist",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output on failed request, got %q", stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
		t.Fatalf("decode structured error output: %v; raw=%s", err, stderr.String())
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level error payload, got %v", payload["error"])
	}
	if strings.TrimSpace(fmt.Sprint(errorPayload["message"])) == "" {
		t.Fatalf("expected non-empty error message payload")
	}
	if _, ok := errorPayload["status_code"]; !ok {
		t.Fatalf("expected status_code in structured error payload, got %v", errorPayload)
	}
	if _, ok := errorPayload["code"]; !ok {
		t.Fatalf("expected code in structured error payload, got %v", errorPayload)
	}
}

func TestRunProviderListCommandTextErrorUsesActionableRemediation(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"provider", "list",
		"--workspace", "ws1",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected provider list exit code 1, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output on failed request, got %q", stdout.String())
	}

	errorText := stderr.String()
	for _, needle := range []string{
		"request failed",
		"what failed:",
		"why:",
		"do next:",
		"not configured",
		"--error-output json",
	} {
		if !strings.Contains(errorText, needle) {
			t.Fatalf("expected actionable error output to contain %q, got %q", needle, errorText)
		}
	}
	for _, disallowed := range []string{"status=", "code=", "correlation_id="} {
		if strings.Contains(errorText, disallowed) {
			t.Fatalf("expected text error output to avoid raw transport internals (%s), got %q", disallowed, errorText)
		}
	}
}

func TestRunSmokeCommandTextErrorUsesActionableUnauthorizedRemediation(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "wrong-token",
		"smoke",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected smoke unauthorized exit code 1, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output on failed request, got %q", stdout.String())
	}

	errorText := stderr.String()
	for _, needle := range []string{
		"request failed",
		"what failed: Daemon authentication was rejected.",
		"why:",
		"do next:",
		"--error-output json",
	} {
		if !strings.Contains(errorText, needle) {
			t.Fatalf("expected unauthorized remediation output to contain %q, got %q", needle, errorText)
		}
	}
	for _, disallowed := range []string{"status=", "code=", "correlation_id="} {
		if strings.Contains(errorText, disallowed) {
			t.Fatalf("expected text error output to avoid raw transport internals (%s), got %q", disallowed, errorText)
		}
	}
}

func TestRunProviderListCommandMachineErrorJSONKeepsStructuredDiagnostics(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"--error-output", "json",
		"provider", "list",
		"--workspace", "ws1",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected provider list exit code 1, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output on failed request, got %q", stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
		t.Fatalf("decode structured provider list error output: %v; raw=%s", err, stderr.String())
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level error payload, got %v", payload["error"])
	}
	if code := fmt.Sprint(errorPayload["code"]); code != "service_not_configured" {
		t.Fatalf("expected code service_not_configured, got %q", code)
	}
	if _, ok := errorPayload["details"]; !ok {
		t.Fatalf("expected details object for structured diagnostics, got %v", errorPayload)
	}
}

func TestRunUnknownCommandMachineErrorJSONOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"--error-output", "json",
		"unknown",
	}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output on unknown command failure, got %q", stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
		t.Fatalf("decode structured unknown-command error output: %v; raw=%s", err, stderr.String())
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level error payload, got %v", payload["error"])
	}
	if code := fmt.Sprint(errorPayload["code"]); code != "cli.command_failed" {
		t.Fatalf("expected code cli.command_failed, got %q", code)
	}
	if !strings.Contains(fmt.Sprint(errorPayload["message"]), `unknown command "unknown"`) {
		t.Fatalf("expected unknown command message, got %q", fmt.Sprint(errorPayload["message"]))
	}
}

func TestRunUnknownCommandSuggestsClosestCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"--auth-token", "cli-test-token", "provder"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	errorText := stderr.String()
	if !strings.Contains(errorText, `unknown command "provder"`) {
		t.Fatalf("expected unknown command output, got %q", errorText)
	}
	if !strings.Contains(errorText, `did you mean "provider"?`) {
		t.Fatalf("expected similar-command suggestion, got %q", errorText)
	}
	if !strings.Contains(errorText, "run `personal-agent help` to view available commands") {
		t.Fatalf("expected actionable help next step, got %q", errorText)
	}
}

func TestCLIExitCodeSemanticsAcrossHelpUsageAndRuntimeFailures(t *testing.T) {
	t.Setenv(cliProfilesPathEnvKey, filepath.Join(t.TempDir(), "profiles.json"))
	t.Setenv(cliWorkspaceEnvKey, "")

	helpOut := &bytes.Buffer{}
	helpErr := &bytes.Buffer{}
	helpCode := run([]string{"help"}, helpOut, helpErr)
	if helpCode != 0 {
		t.Fatalf("expected help exit code 0, got %d (stderr=%s)", helpCode, helpErr.String())
	}

	usageOut := &bytes.Buffer{}
	usageErr := &bytes.Buffer{}
	usageCode := run([]string{"profile", "set"}, usageOut, usageErr)
	if usageCode != 2 {
		t.Fatalf("expected usage exit code 2 for profile set missing required flags, got %d (stderr=%s)", usageCode, usageErr.String())
	}
	if !strings.Contains(usageErr.String(), "--name is required") {
		t.Fatalf("expected profile missing-name usage error, got %q", usageErr.String())
	}

	parseOut := &bytes.Buffer{}
	parseErr := &bytes.Buffer{}
	parseCode := run([]string{"profile", "set", "--unknown-flag"}, parseOut, parseErr)
	if parseCode != 2 {
		t.Fatalf("expected flag-parse exit code 2, got %d (stderr=%s)", parseCode, parseErr.String())
	}

	runtimeOut := &bytes.Buffer{}
	runtimeErr := &bytes.Buffer{}
	runtimeCode := run([]string{
		"--mode", "tcp",
		"--address", "127.0.0.1:1",
		"--auth-token", "cli-test-token",
		"smoke",
	}, runtimeOut, runtimeErr)
	if runtimeCode != 1 {
		t.Fatalf("expected runtime/request failure exit code 1, got %d (stderr=%s)", runtimeCode, runtimeErr.String())
	}
}

func TestRunHelpCommandPrintsSkimFirstUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("help exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	usage := stdout.String()
	for _, needle := range []string{
		"Quickstart workflows (copy/paste):",
		"Skim command groups:",
		"Full command reference (generated from schema):",
		"personal-agent quickstart",
		"help",
		"completion",
		"doctor",
		"Help tips:",
	} {
		if !strings.Contains(usage, needle) {
			t.Fatalf("expected help output to contain %q, output=%q", needle, usage)
		}
	}
}

func TestRootHelpWorkflowExamplesReferenceKnownCommands(t *testing.T) {
	schema := buildCLISchemaDocument()
	commandSet := make(map[string]struct{}, len(schema.Commands))
	for _, command := range schema.Commands {
		commandSet[strings.ToLower(strings.TrimSpace(command.Name))] = struct{}{}
	}
	for _, example := range rootHelpWorkflowExamples() {
		command := rootCommandToken(example.Command)
		if command == "" {
			t.Fatalf("workflow example command is empty: %+v", example)
		}
		if _, found := commandSet[command]; !found {
			t.Fatalf("workflow example references unknown root command %q: %+v", command, example)
		}
	}
}

func TestRootHelpSkimGroupsReferenceKnownCommands(t *testing.T) {
	schema := buildCLISchemaDocument()
	commandSet := make(map[string]struct{}, len(schema.Commands))
	for _, command := range schema.Commands {
		commandSet[strings.ToLower(strings.TrimSpace(command.Name))] = struct{}{}
	}
	for _, group := range rootHelpSkimGroups() {
		for _, entry := range group.Commands {
			command := rootCommandToken(entry)
			if command == "" {
				t.Fatalf("skim group entry command is empty: group=%q entry=%q", group.Title, entry)
			}
			if _, found := commandSet[command]; !found {
				t.Fatalf("skim group %q references unknown root command %q via entry %q", group.Title, command, entry)
			}
		}
	}
}

func TestRunHelpCommandGeneratedReferenceIncludesAllCommands(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("help exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	usage := stdout.String()
	referenceCommands := extractRootReferenceCommandNames(usage)
	schema := buildCLISchemaDocument()
	for _, command := range schema.Commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			continue
		}
		if _, found := referenceCommands[name]; !found {
			t.Fatalf("expected generated reference to include command %q, usage=%q", name, usage)
		}
	}
}

func extractRootReferenceCommandNames(helpText string) map[string]struct{} {
	names := map[string]struct{}{}
	lines := strings.Split(helpText, "\n")
	inReferenceSection := false
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "Full command reference (generated from schema):") {
			inReferenceSection = true
			continue
		}
		if !inReferenceSection {
			continue
		}
		if strings.EqualFold(trimmed, "Help tips:") {
			break
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		names[fields[0]] = struct{}{}
	}
	return names
}

func TestRunHelpCommandPrintsScopedCommandUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"help", "task"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("help task exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	usage := stdout.String()
	for _, needle := range []string{
		"Usage: personal-agent task <subcommand> [flags]",
		"Summary: Task submit/status/cancel/retry/requeue operations",
		"Subcommands:",
		"submit",
		"status",
	} {
		if !strings.Contains(usage, needle) {
			t.Fatalf("expected scoped task usage to contain %q, output=%q", needle, usage)
		}
	}
}

func TestRunCommandHelpFlagPrintsScopedUsageWithoutDaemonConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"task", "--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("task --help exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	usage := stdout.String()
	for _, needle := range []string{
		"Usage: personal-agent task <subcommand> [flags]",
		"Subcommands:",
	} {
		if !strings.Contains(usage, needle) {
			t.Fatalf("expected task --help output to contain %q, output=%q", needle, usage)
		}
	}
}

func TestRunSubcommandHelpFlagPrintsScopedUsageWithoutDaemonConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"task", "submit", "--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("task submit --help exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	usage := stdout.String()
	for _, needle := range []string{
		"Usage: personal-agent task submit [flags]",
		"Summary: Submit a task",
		"Required flags:",
		"--workspace",
		"--requested-by",
		"--subject",
		"--title",
	} {
		if !strings.Contains(usage, needle) {
			t.Fatalf("expected task submit --help output to contain %q, output=%q", needle, usage)
		}
	}
}

func TestRunSubcommandHelpFlagBypassesDaemonForMetaCapabilities(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"meta", "capabilities", "--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("meta capabilities --help exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage: personal-agent meta capabilities [flags]") {
		t.Fatalf("expected scoped meta capabilities usage, output=%q", stdout.String())
	}
}

func TestRunCompletionCommandGeneratesBashScript(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"completion", "--shell", "bash"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("completion bash exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	script := stdout.String()
	for _, needle := range []string{
		"_personal_agent_completion()",
		"_pa_subcommands_for_path()",
		"_pa_flags_for_path()",
		"_pa_resolve_path()",
		"complete -F _personal_agent_completion personal-agent",
		`"connector twilio webhook")`,
		"replay serve",
		"--requested-by",
		"--primary-channel",
		"--signature-mode",
		"--shell",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected bash completion script to contain %q, got %q", needle, script)
		}
	}
}

func TestRunCompletionCommandGeneratesZshScript(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"completion", "zsh"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("completion zsh exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	script := stdout.String()
	for _, needle := range []string{
		"#compdef personal-agent",
		"_personal_agent()",
		"_pa_subcommands_for_path()",
		"_pa_flags_for_path()",
		"_pa_resolve_path_zsh()",
		"compdef _personal_agent personal-agent",
		`"connector twilio webhook")`,
		"--requested-by",
		"--primary-channel",
		"--signature-mode",
		"--shell",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected zsh completion script to contain %q, got %q", needle, script)
		}
	}
}

func TestRunCompletionCommandGeneratesFishScript(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"completion", "--shell", "fish"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("completion fish exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	script := stdout.String()
	for _, needle := range []string{
		"function __pa_dynamic_completions",
		"function __pa_subcommands_for_path",
		"function __pa_flags_for_path",
		"complete -c personal-agent -f -a",
		`case "connector twilio webhook"`,
		"--requested-by",
		"--primary-channel",
		"--signature-mode",
		"--shell",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected fish completion script to contain %q, got %q", needle, script)
		}
	}
}

func TestBuildCompletionCommandIndexIncludesDeepSubcommandsAndContextualFlags(t *testing.T) {
	index := buildCompletionCommandIndex(buildCLISchemaDocument())

	pathChecks := []struct {
		path         string
		subcommands  []string
		contextFlags []string
	}{
		{
			path:        "channel mapping",
			subcommands: []string{"enable", "disable", "list", "prioritize"},
		},
		{
			path:         "channel mapping enable",
			contextFlags: []string{"--channel", "--connector", "--workspace"},
		},
		{
			path:        "connector twilio webhook",
			subcommands: []string{"serve", "replay"},
		},
		{
			path:         "connector twilio webhook serve",
			contextFlags: []string{"--signature-mode", "--listen", "--workspace"},
		},
		{
			path:         "task submit",
			contextFlags: []string{"--workspace", "--requested-by", "--subject", "--title", "--description", "--task-class"},
		},
		{
			path:         "provider set",
			contextFlags: []string{"--workspace", "--provider", "--endpoint", "--api-key-secret", "--clear-api-key"},
		},
		{
			path:         "model select",
			contextFlags: []string{"--workspace", "--task-class", "--provider", "--model"},
		},
		{
			path:         "comm policy set",
			contextFlags: []string{"--workspace", "--source-channel", "--endpoint-pattern", "--primary-channel", "--fallback-channels"},
		},
		{
			path:         "identity bootstrap",
			contextFlags: []string{"--workspace", "--principal", "--display-name", "--source"},
		},
	}

	for _, check := range pathChecks {
		subcommands := index.subcommandsByPath[check.path]
		for _, expected := range check.subcommands {
			if !completionTokenExists(subcommands, expected) {
				t.Fatalf("expected completion subcommand %q for path %q, got %v", expected, check.path, subcommands)
			}
		}
		flags := index.flagsByPath[check.path]
		for _, expected := range check.contextFlags {
			if !completionTokenExists(flags, expected) {
				t.Fatalf("expected completion flag %q for path %q, got %v", expected, check.path, flags)
			}
		}
	}
}

func TestRunCompletionCommandRejectsUnsupportedShell(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"completion", "--shell", "powershell"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for unsupported shell, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `unsupported shell "powershell"`) {
		t.Fatalf("expected unsupported shell error, got %q", stderr.String())
	}
}

func completionTokenExists(tokens []string, expected string) bool {
	needle := strings.TrimSpace(expected)
	for _, token := range tokens {
		if strings.TrimSpace(token) == needle {
			return true
		}
	}
	return false
}

func TestRunNoCommandPrintsSkimFirstUsageOnError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for missing command, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	usage := stderr.String()
	if !strings.Contains(usage, "Quickstart workflows (copy/paste):") {
		t.Fatalf("expected usage output in stderr for missing command, got %q", usage)
	}
}

func TestRunAssistantTaskSubmitFlowSupportsBackNavigation(t *testing.T) {
	server := startCLITestServer(t)
	setTestChatInput(t, strings.Join([]string{
		"actor.requester",
		"back",
		"actor.requester",
		"actor.subject",
		"Task from assistant",
		"Optional description",
		"chat",
	}, "\n")+"\n")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"assistant",
		"--workspace", "ws1",
		"--flow", "task_submit",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("assistant task flow exit code %d, stderr=%s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode assistant payload: %v", err)
	}
	if payload["flow"] != "task_submit" {
		t.Fatalf("expected flow task_submit, got %v", payload["flow"])
	}
	if payload["success"] != true || payload["cancelled"] != false {
		t.Fatalf("expected successful non-cancelled assistant response, got %+v", payload)
	}
	if fmt.Sprint(payload["backtracks"]) != "1" {
		t.Fatalf("expected backtracks=1, got %v", payload["backtracks"])
	}

	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", payload["result"])
	}
	if strings.TrimSpace(fmt.Sprint(result["task_id"])) == "" || strings.TrimSpace(fmt.Sprint(result["run_id"])) == "" {
		t.Fatalf("expected task_id/run_id in assistant task result, got %+v", result)
	}
}

func TestRunAssistantCommSendFlowSupportsCancel(t *testing.T) {
	server := startCLITestServer(t)
	setTestChatInput(t, "cancel\n")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"assistant",
		"--workspace", "ws1",
		"--flow", "comm_send",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("assistant cancel flow should exit 0, got %d (stderr=%s)", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode assistant payload: %v", err)
	}
	if payload["flow"] != "comm_send" {
		t.Fatalf("expected flow comm_send, got %v", payload["flow"])
	}
	if payload["cancelled"] != true {
		t.Fatalf("expected cancelled=true, got %v", payload["cancelled"])
	}
	if payload["success"] != false {
		t.Fatalf("expected success=false for cancelled flow, got %v", payload["success"])
	}
}

func TestRunMetaSchemaCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--output", "json-compact",
		"meta", "schema",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("meta schema exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode meta schema payload: %v", err)
	}
	if payload["program"] != "personal-agent" {
		t.Fatalf("expected program personal-agent, got %v", payload["program"])
	}
	if payload["schema_version"] != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %v", payload["schema_version"])
	}
	outputModesRaw, ok := payload["output_modes"].([]any)
	if !ok {
		t.Fatalf("expected output_modes array in schema payload, got %T", payload["output_modes"])
	}
	outputModes := map[string]bool{}
	for _, raw := range outputModesRaw {
		outputModes[strings.TrimSpace(fmt.Sprint(raw))] = true
	}
	for _, expectedMode := range []string{"json", "json-compact", "text"} {
		if !outputModes[expectedMode] {
			t.Fatalf("expected output mode %q in schema payload, got %v", expectedMode, outputModesRaw)
		}
	}
	globalFlags, ok := payload["global_flags"].([]any)
	if !ok || len(globalFlags) == 0 {
		t.Fatalf("expected non-empty global_flags list, got %v", payload["global_flags"])
	}
	flagNames := map[string]bool{}
	for _, raw := range globalFlags {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		flagName, _ := record["name"].(string)
		flagNames[flagName] = true
	}
	if !flagNames["--output"] || !flagNames["--error-output"] {
		t.Fatalf("expected --output and --error-output in global flag schema, got %v", flagNames)
	}

	commands, ok := payload["commands"].([]any)
	if !ok || len(commands) == 0 {
		t.Fatalf("expected commands list in meta schema payload")
	}
	var foundTaskSubmit bool
	var foundTaskRetry bool
	var foundTaskRequeue bool
	var foundMetaCapabilities bool
	var foundDoctor bool
	var foundQuickstart bool
	var foundHelp bool
	var foundCompletion bool
	var foundAssistant bool
	var foundChatContract bool
	var foundVersion bool
	var foundProfileActive bool
	var foundProfileDelete bool
	var foundProfileRename bool
	for _, raw := range commands {
		commandRecord, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		commandName, _ := commandRecord["name"].(string)
		if commandName == "help" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			if !requiresDaemon && machineOutputSafe {
				foundHelp = true
			}
		}
		if commandName == "quickstart" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			if !requiresDaemon && machineOutputSafe {
				foundQuickstart = true
			}
		}
		if commandName == "completion" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			if !requiresDaemon && machineOutputSafe {
				subcommands, _ := commandRecord["subcommands"].([]any)
				var hasBash bool
				var hasFish bool
				var hasZsh bool
				for _, sub := range subcommands {
					subRecord, ok := sub.(map[string]any)
					if !ok {
						continue
					}
					if subRecord["name"] == "bash" {
						hasBash = true
					}
					if subRecord["name"] == "fish" {
						hasFish = true
					}
					if subRecord["name"] == "zsh" {
						hasZsh = true
					}
				}
				if hasBash && hasFish && hasZsh {
					foundCompletion = true
				}
			}
		}
		if commandName == "assistant" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			if requiresDaemon && machineOutputSafe {
				foundAssistant = true
			}
		}
		if commandName == "chat" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			supportsStreaming, _ := commandRecord["supports_streaming"].(bool)
			if requiresDaemon && !machineOutputSafe && supportsStreaming {
				foundChatContract = true
			}
		}
		if commandName == "version" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			machineOutputSafe, _ := commandRecord["machine_output_safe"].(bool)
			if !requiresDaemon && machineOutputSafe {
				foundVersion = true
			}
		}
		if commandName == "doctor" {
			if requiresDaemon, ok := commandRecord["requires_daemon"].(bool); ok && requiresDaemon {
				foundDoctor = true
			}
		}
		if commandName == "meta" {
			subcommands, _ := commandRecord["subcommands"].([]any)
			for _, sub := range subcommands {
				subRecord, ok := sub.(map[string]any)
				if !ok {
					continue
				}
				if subRecord["name"] == "capabilities" {
					if requiresDaemon, ok := subRecord["requires_daemon"].(bool); ok && requiresDaemon {
						foundMetaCapabilities = true
					}
				}
			}
		}
		if commandName == "profile" {
			requiresDaemon, _ := commandRecord["requires_daemon"].(bool)
			if requiresDaemon {
				t.Fatalf("expected profile command to be local (requires_daemon=false)")
			}
			subcommands, _ := commandRecord["subcommands"].([]any)
			for _, sub := range subcommands {
				subRecord, ok := sub.(map[string]any)
				if !ok {
					continue
				}
				switch subRecord["name"] {
				case "active":
					foundProfileActive = true
				case "delete":
					requiredFlags, _ := subRecord["required_flags"].([]any)
					for _, requiredFlag := range requiredFlags {
						if fmt.Sprint(requiredFlag) == "--name" {
							foundProfileDelete = true
						}
					}
				case "rename":
					requiredFlags, _ := subRecord["required_flags"].([]any)
					requiredFlagSet := map[string]bool{}
					for _, requiredFlag := range requiredFlags {
						requiredFlagSet[fmt.Sprint(requiredFlag)] = true
					}
					if requiredFlagSet["--name"] && requiredFlagSet["--to"] {
						foundProfileRename = true
					}
				}
			}
		}
		if commandName != "task" {
			continue
		}
		subcommands, _ := commandRecord["subcommands"].([]any)
		for _, sub := range subcommands {
			subRecord, ok := sub.(map[string]any)
			if !ok {
				continue
			}
			if subRecord["name"] != "submit" {
				switch subRecord["name"] {
				case "retry":
					requiredFlags, _ := subRecord["required_flags"].([]any)
					for _, requiredFlag := range requiredFlags {
						if fmt.Sprint(requiredFlag) == "--task-id|--run-id" {
							foundTaskRetry = true
						}
					}
				case "requeue":
					requiredFlags, _ := subRecord["required_flags"].([]any)
					for _, requiredFlag := range requiredFlags {
						if fmt.Sprint(requiredFlag) == "--task-id|--run-id" {
							foundTaskRequeue = true
						}
					}
				}
				continue
			}
			requiredFlags, _ := subRecord["required_flags"].([]any)
			requiredFlagSet := map[string]bool{}
			for _, requiredFlag := range requiredFlags {
				requiredFlagSet[fmt.Sprint(requiredFlag)] = true
			}
			if requiredFlagSet["--workspace"] && requiredFlagSet["--requested-by"] && requiredFlagSet["--subject"] && requiredFlagSet["--title"] {
				foundTaskSubmit = true
			}
		}
	}
	if !foundTaskSubmit {
		t.Fatalf("expected task submit required_flags metadata in schema payload")
	}
	if !foundTaskRetry {
		t.Fatalf("expected task retry required_flags metadata in schema payload")
	}
	if !foundTaskRequeue {
		t.Fatalf("expected task requeue required_flags metadata in schema payload")
	}
	if !foundMetaCapabilities {
		t.Fatalf("expected meta capabilities subcommand metadata with requires_daemon=true in schema payload")
	}
	if !foundDoctor {
		t.Fatalf("expected top-level doctor command metadata with requires_daemon=true in schema payload")
	}
	if !foundQuickstart {
		t.Fatalf("expected quickstart command metadata with requires_daemon=false and machine_output_safe=true in schema payload")
	}
	if !foundHelp {
		t.Fatalf("expected help command metadata with requires_daemon=false and machine_output_safe=true in schema payload")
	}
	if !foundCompletion {
		t.Fatalf("expected completion command metadata with bash/fish/zsh subcommands in schema payload")
	}
	if !foundAssistant {
		t.Fatalf("expected assistant command metadata with requires_daemon=true and machine_output_safe=true in schema payload")
	}
	if !foundChatContract {
		t.Fatalf("expected chat command metadata with requires_daemon=true, machine_output_safe=false, and supports_streaming=true in schema payload")
	}
	if !foundVersion {
		t.Fatalf("expected version command metadata with requires_daemon=false and machine_output_safe=true in schema payload")
	}
	if !foundProfileActive {
		t.Fatalf("expected profile active subcommand metadata in schema payload")
	}
	if !foundProfileDelete {
		t.Fatalf("expected profile delete required_flags metadata in schema payload")
	}
	if !foundProfileRename {
		t.Fatalf("expected profile rename required_flags metadata in schema payload")
	}
}

func TestBuildCLISchemaDocumentRootRegistryParity(t *testing.T) {
	registry := cliRootCommandRegistry()
	catalog := cliSchemaCommandCatalog()
	schema := buildCLISchemaDocument()

	registryNames := sortedRootCommandNames(registry)
	schemaNames := make([]string, 0, len(schema.Commands))
	for _, command := range schema.Commands {
		trimmed := strings.TrimSpace(command.Name)
		if trimmed == "" {
			continue
		}
		schemaNames = append(schemaNames, trimmed)
	}
	sort.Strings(schemaNames)
	if len(schemaNames) != len(registryNames) {
		t.Fatalf("schema root command count %d does not match registry count %d (schema=%v registry=%v)", len(schemaNames), len(registryNames), schemaNames, registryNames)
	}
	for _, name := range registryNames {
		if _, found := catalog[name]; !found {
			t.Fatalf("schema catalog is missing root command metadata for %q", name)
		}
	}
	for _, name := range schemaNames {
		if _, found := registry[name]; !found {
			t.Fatalf("schema emitted unknown root command %q not present in registry", name)
		}
	}
	for name := range catalog {
		if _, found := registry[name]; !found {
			t.Fatalf("schema catalog contains unknown root command %q not present in registry", name)
		}
	}
}

func TestBuildCLISchemaDocumentIncludesNestedDiscoverabilityAndCapabilities(t *testing.T) {
	schema := buildCLISchemaDocument()
	requiredPaths := [][]string{
		{"task", "submit"},
		{"channel", "mapping", "enable"},
		{"connector", "bridge", "status"},
		{"connector", "twilio", "webhook", "serve"},
		{"comm", "policy", "set"},
		{"automation", "run", "comm-event"},
		{"identity", "revoke-session"},
	}
	for _, path := range requiredPaths {
		command, found := findSchemaCommandByPath(schema.Commands, path...)
		if !found {
			t.Fatalf("expected schema to include command path %q", strings.Join(path, " "))
		}
		if !command.MachineOutputSafe {
			t.Fatalf("expected command path %q to be machine-output safe", strings.Join(path, " "))
		}
	}

	streamCommand, found := findSchemaCommandByPath(schema.Commands, "stream")
	if !found {
		t.Fatalf("expected stream command in schema")
	}
	if !streamCommand.SupportsStreaming {
		t.Fatalf("expected stream command supports_streaming=true")
	}
	chatCommand, found := findSchemaCommandByPath(schema.Commands, "chat")
	if !found {
		t.Fatalf("expected chat command in schema")
	}
	if !chatCommand.SupportsStreaming {
		t.Fatalf("expected chat command supports_streaming=true")
	}
	if chatCommand.MachineOutputSafe {
		t.Fatalf("expected chat command machine_output_safe=false")
	}

	taskSubmit, found := findSchemaCommandByPath(schema.Commands, "task", "submit")
	if !found {
		t.Fatalf("expected task submit command in schema")
	}
	requiredSet := map[string]bool{}
	for _, flagName := range taskSubmit.RequiredFlags {
		requiredSet[strings.TrimSpace(flagName)] = true
	}
	for _, required := range []string{"--workspace", "--requested-by", "--subject", "--title"} {
		if !requiredSet[required] {
			t.Fatalf("expected task submit required flag %q in schema metadata", required)
		}
	}
}

func findSchemaCommandByPath(commands []cliCommandSchema, path ...string) (cliCommandSchema, bool) {
	if len(path) == 0 {
		return cliCommandSchema{}, false
	}
	currentCommands := commands
	var current cliCommandSchema
	for _, segment := range path {
		found := false
		for _, candidate := range currentCommands {
			if strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(segment)) {
				current = candidate
				currentCommands = candidate.Subcommands
				found = true
				break
			}
		}
		if !found {
			return cliCommandSchema{}, false
		}
	}
	return current, true
}

func TestRunMetaCapabilitiesCommand(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"meta", "capabilities",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("meta capabilities exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode meta capabilities payload: %v", err)
	}
	if payload["api_version"] != "v1" {
		t.Fatalf("expected api_version v1, got %v", payload["api_version"])
	}
	if signals, ok := payload["client_signal_types"].([]any); !ok || len(signals) == 0 {
		t.Fatalf("expected non-empty client_signal_types, got %v", payload["client_signal_types"])
	}
	if modes, ok := payload["protocol_modes"].([]any); !ok || len(modes) == 0 {
		t.Fatalf("expected non-empty protocol_modes, got %v", payload["protocol_modes"])
	}
	if groups, ok := payload["route_groups"].([]any); !ok || len(groups) == 0 {
		t.Fatalf("expected non-empty route_groups, got %v", payload["route_groups"])
	}
}

func TestRunVersionCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--output", "json-compact",
		"version",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("version exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode version payload: %v", err)
	}
	if payload["schema_version"] != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %v", payload["schema_version"])
	}
	if payload["program"] != "personal-agent" {
		t.Fatalf("expected program personal-agent, got %v", payload["program"])
	}
	if strings.TrimSpace(fmt.Sprint(payload["version"])) == "" {
		t.Fatalf("expected non-empty version, got %v", payload["version"])
	}
	if strings.TrimSpace(fmt.Sprint(payload["go_version"])) == "" {
		t.Fatalf("expected non-empty go_version, got %v", payload["go_version"])
	}
	if !strings.Contains(fmt.Sprint(payload["platform"]), "/") {
		t.Fatalf("expected platform os/arch value, got %v", payload["platform"])
	}
}

func TestRunVersionCommandTextOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--output", "text",
		"version",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("version --output text exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	output := stdout.String()
	for _, needle := range []string{
		"personal-agent version",
		"version:",
		"go_version:",
		"platform:",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("expected text version output to contain %q, got %q", needle, output)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("expected text output for version command, got JSON payload %q", output)
	}
}

func TestRunDoctorCommandDaemonUnavailable(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", "127.0.0.1:1",
		"--auth-token", "cli-test-token",
		"--timeout", "1s",
		"doctor",
		"--workspace", "ws1",
		"--include-optional=false",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when daemon is unreachable, got %d (stderr=%s)", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode doctor payload: %v", err)
	}
	if payload["schema_version"] != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %v", payload["schema_version"])
	}
	if payload["overall_status"] != "fail" {
		t.Fatalf("expected overall_status=fail for unreachable daemon, got %v", payload["overall_status"])
	}
	checks, ok := payload["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("expected non-empty checks list, got %v", payload["checks"])
	}

	statusByID := map[string]string{}
	for _, raw := range checks {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := record["id"].(string)
		status, _ := record["status"].(string)
		statusByID[id] = status
	}
	if statusByID["daemon.connectivity"] != "fail" {
		t.Fatalf("expected daemon.connectivity=fail, got %q", statusByID["daemon.connectivity"])
	}
	if statusByID["workspace.context"] != "skipped" {
		t.Fatalf("expected workspace.context=skipped when connectivity fails, got %q", statusByID["workspace.context"])
	}
	if statusByID["plugins.health"] != "skipped" {
		t.Fatalf("expected plugins.health=skipped when connectivity fails, got %q", statusByID["plugins.health"])
	}
}

func TestRunDoctorCommandQuickSkipsDeepChecks(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"doctor",
		"--workspace", "ws1",
		"--quick",
	}, stdout, stderr)
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("doctor --quick exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode doctor --quick payload: %v", err)
	}
	checks, ok := payload["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("expected checks array in doctor --quick payload, got %v", payload["checks"])
	}

	statusByID := map[string]string{}
	summaryByID := map[string]string{}
	for _, raw := range checks {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(fmt.Sprint(record["id"]))
		statusByID[id] = strings.TrimSpace(fmt.Sprint(record["status"]))
		summaryByID[id] = strings.TrimSpace(fmt.Sprint(record["summary"]))
	}

	for _, id := range []string{
		"providers.readiness",
		"models.route_readiness",
		"channels.mappings",
		"secrets.references",
		"plugins.health",
		"tooling.optional",
	} {
		if statusByID[id] != "skipped" {
			t.Fatalf("expected %s=skipped for doctor --quick, got %q", id, statusByID[id])
		}
	}
	if !strings.Contains(strings.ToLower(summaryByID["providers.readiness"]), "quick mode") {
		t.Fatalf("expected providers.readiness quick-mode summary, got %q", summaryByID["providers.readiness"])
	}

	for _, id := range []string{"daemon.connectivity", "daemon.lifecycle", "workspace.context"} {
		if statusByID[id] == "skipped" {
			t.Fatalf("expected %s to be evaluated in doctor --quick, got skipped", id)
		}
	}
}

func TestRunDoctorCommandCheckSchema(t *testing.T) {
	server := startCLITestServer(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"doctor",
		"--workspace", "ws1",
	}, stdout, stderr)
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("doctor exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode doctor payload: %v", err)
	}
	if payload["schema_version"] != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %v", payload["schema_version"])
	}
	if payload["workspace_id"] != "ws1" {
		t.Fatalf("expected workspace_id ws1, got %v", payload["workspace_id"])
	}

	expectedCheckIDs := []string{
		"daemon.connectivity",
		"daemon.lifecycle",
		"workspace.context",
		"providers.readiness",
		"models.route_readiness",
		"channels.mappings",
		"secrets.references",
		"plugins.health",
		"tooling.optional",
	}
	checks, ok := payload["checks"].([]any)
	if !ok || len(checks) != len(expectedCheckIDs) {
		t.Fatalf("expected %d checks, got %v", len(expectedCheckIDs), payload["checks"])
	}
	for idx, expectedID := range expectedCheckIDs {
		record, ok := checks[idx].(map[string]any)
		if !ok {
			t.Fatalf("expected check object at index %d", idx)
		}
		if got := fmt.Sprint(record["id"]); got != expectedID {
			t.Fatalf("expected check id[%d]=%s, got %s", idx, expectedID, got)
		}
	}
}

func TestRunDoctorCommandSurfacesMissingLifecycleControlAuthMetadata(t *testing.T) {
	server := startCLITestServerWithLifecycle(t, &cliDoctorLifecycleServiceStub{
		status: transport.DaemonLifecycleStatusResponse{
			LifecycleState: "running",
			SetupState:     "ready",
			InstallState:   "installed",
			DatabaseReady:  true,
			ControlAuth: transport.DaemonControlAuthState{
				State:            "missing",
				Source:           "auth_token_flag",
				RemediationHints: []string{"set daemon auth token"},
			},
			HealthClassification: transport.DaemonLifecycleHealthClassification{
				OverallState:       "ready",
				CoreRuntimeState:   "ready",
				PluginRuntimeState: "healthy",
				Blocking:           false,
			},
		},
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"--output", "json-compact",
		"doctor",
		"--workspace", "ws1",
	}, stdout, stderr)
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("doctor exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode doctor payload: %v", err)
	}
	checks, ok := payload["checks"].([]any)
	if !ok {
		t.Fatalf("expected checks array, got %T", payload["checks"])
	}

	var lifecycleCheck map[string]any
	for _, raw := range checks {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if fmt.Sprint(record["id"]) == "daemon.lifecycle" {
			lifecycleCheck = record
			break
		}
	}
	if lifecycleCheck == nil {
		t.Fatalf("expected daemon.lifecycle check in doctor payload")
	}
	if got := fmt.Sprint(lifecycleCheck["status"]); got != "fail" {
		t.Fatalf("expected daemon.lifecycle status fail for missing auth, got %s", got)
	}

	details, ok := lifecycleCheck["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected details object for daemon.lifecycle check, got %T", lifecycleCheck["details"])
	}
	controlAuth, ok := details["control_auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected control_auth details object, got %T", details["control_auth"])
	}
	if fmt.Sprint(controlAuth["state"]) != "missing" {
		t.Fatalf("expected control_auth.state missing, got %v", controlAuth["state"])
	}
	if fmt.Sprint(controlAuth["source"]) != "auth_token_flag" {
		t.Fatalf("expected control_auth.source auth_token_flag, got %v", controlAuth["source"])
	}
}

func TestRunQuickstartCommandDaemonUnavailableReturnsRemediation(t *testing.T) {
	t.Setenv(cliProfilesPathEnvKey, filepath.Join(t.TempDir(), "profiles.json"))
	t.Setenv("PA_RUNTIME_ROOT_DIR", filepath.Join(t.TempDir(), "runtime-root"))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"quickstart",
		"--workspace", "ws1",
		"--profile", "qs-daemon-unavailable",
		"--mode", "tcp",
		"--address", "127.0.0.1:1",
		"--provider", "ollama",
		"--skip-provider-setup=true",
		"--skip-model-route=true",
		"--skip-doctor=true",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected quickstart to fail when daemon is unreachable, got %d (stderr=%s)", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode quickstart payload: %v", err)
	}
	if payload["schema_version"] != "1.0.0" {
		t.Fatalf("expected schema_version 1.0.0, got %v", payload["schema_version"])
	}
	if payload["overall_status"] != "fail" {
		t.Fatalf("expected overall_status=fail, got %v", payload["overall_status"])
	}
	defaults, ok := payload["defaults"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults metadata, got %T", payload["defaults"])
	}
	workspaceSelection, ok := defaults["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.workspace metadata, got %T", defaults["workspace"])
	}
	if fmt.Sprint(workspaceSelection["value"]) != "ws1" || fmt.Sprint(workspaceSelection["source"]) != "explicit" || fmt.Sprint(workspaceSelection["override_flag"]) != "--workspace" {
		t.Fatalf("unexpected workspace defaults metadata: %+v", workspaceSelection)
	}
	profileSelection, ok := defaults["profile"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.profile metadata, got %T", defaults["profile"])
	}
	if fmt.Sprint(profileSelection["value"]) != "qs-daemon-unavailable" || fmt.Sprint(profileSelection["source"]) != "explicit" || fmt.Sprint(profileSelection["override_flag"]) != "--profile" {
		t.Fatalf("unexpected profile defaults metadata: %+v", profileSelection)
	}
	tokenFileSelection, ok := defaults["token_file"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.token_file metadata, got %T", defaults["token_file"])
	}
	if fmt.Sprint(tokenFileSelection["source"]) != "default" || fmt.Sprint(tokenFileSelection["override_flag"]) != "--token-file" || strings.TrimSpace(fmt.Sprint(tokenFileSelection["value"])) == "" {
		t.Fatalf("unexpected token_file defaults metadata: %+v", tokenFileSelection)
	}
	overrideHints, ok := defaults["override_hints"].([]any)
	if !ok || len(overrideHints) < 3 {
		t.Fatalf("expected defaults.override_hints entries, got %v", defaults["override_hints"])
	}

	authStep := quickstartStepByID(t, payload, "auth.bootstrap")
	if authStep["status"] != "pass" {
		t.Fatalf("expected auth.bootstrap status=pass, got %v", authStep["status"])
	}
	connectivityStep := quickstartStepByID(t, payload, "daemon.connectivity")
	if connectivityStep["status"] != "fail" {
		t.Fatalf("expected daemon.connectivity status=fail, got %v", connectivityStep["status"])
	}

	remediation, ok := payload["remediation"].(map[string]any)
	if !ok {
		t.Fatalf("expected remediation object, got %T", payload["remediation"])
	}
	nextSteps, ok := remediation["next_steps"].([]any)
	if !ok || len(nextSteps) == 0 {
		t.Fatalf("expected remediation.next_steps entries, got %v", remediation["next_steps"])
	}
	var hasDaemonCommand bool
	var hasSmokeCommand bool
	var hasProfileUseCommand bool
	for _, raw := range nextSteps {
		step := fmt.Sprint(raw)
		if strings.Contains(step, "personal-agent-daemon --listen-mode 'tcp' --listen-address '127.0.0.1:1' --auth-token-file '") {
			hasDaemonCommand = true
		}
		if strings.Contains(step, "Confirm daemon health with `personal-agent smoke`.") {
			hasSmokeCommand = true
		}
		if strings.Contains(step, "personal-agent profile use --name") {
			hasProfileUseCommand = true
		}
	}
	if !hasDaemonCommand {
		t.Fatalf("expected daemon remediation command with resolved mode/address/token path, got %+v", nextSteps)
	}
	if !hasSmokeCommand {
		t.Fatalf("expected smoke remediation command, got %+v", nextSteps)
	}
	if hasProfileUseCommand {
		t.Fatalf("did not expect profile-use remediation when quickstart activated profile by default, got %+v", nextSteps)
	}
	if !strings.Contains(fmt.Sprint(remediation["human_summary"]), "requires attention") {
		t.Fatalf("expected human remediation summary, got %v", remediation["human_summary"])
	}
}

func TestRunQuickstartCommandDaemonUnavailableWithInactiveProfileIncludesExplicitCLIHints(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv("PA_RUNTIME_ROOT_DIR", filepath.Join(t.TempDir(), "runtime-root"))
	if err := saveCLIProfilesState(profilesPath, cliProfilesState{
		ActiveProfile: "already-active",
		Profiles: map[string]cliProfileRecord{
			"already-active": {
				Name:         "already-active",
				ListenerMode: "tcp",
				Address:      "127.0.0.1:17101",
				WorkspaceID:  "ws1",
			},
		},
	}); err != nil {
		t.Fatalf("seed existing active profile: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"quickstart",
		"--workspace", "ws1",
		"--profile", "qs-inactive-profile",
		"--activate=false",
		"--mode", "tcp",
		"--address", "127.0.0.1:17099",
		"--provider", "ollama",
		"--skip-provider-setup=true",
		"--skip-model-route=true",
		"--skip-doctor=true",
	}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("expected quickstart to fail when daemon is unreachable, got %d (stderr=%s)", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode quickstart payload: %v", err)
	}
	if payload["overall_status"] != "fail" {
		t.Fatalf("expected overall_status=fail, got %v", payload["overall_status"])
	}

	remediation, ok := payload["remediation"].(map[string]any)
	if !ok {
		t.Fatalf("expected remediation object, got %T", payload["remediation"])
	}
	nextSteps, ok := remediation["next_steps"].([]any)
	if !ok || len(nextSteps) == 0 {
		t.Fatalf("expected remediation.next_steps entries, got %v", remediation["next_steps"])
	}

	var hasProfileUseCommand bool
	var hasExplicitSmokeCommand bool
	for _, raw := range nextSteps {
		step := fmt.Sprint(raw)
		if strings.Contains(step, "personal-agent profile use --name 'qs-inactive-profile'") {
			hasProfileUseCommand = true
		}
		if strings.Contains(step, "personal-agent --mode 'tcp' --address '127.0.0.1:17099' --auth-token-file '") && strings.Contains(step, " smoke") {
			hasExplicitSmokeCommand = true
		}
	}
	if !hasProfileUseCommand {
		t.Fatalf("expected profile-use remediation command when quickstart leaves profile inactive, got %+v", nextSteps)
	}
	if !hasExplicitSmokeCommand {
		t.Fatalf("expected explicit mode/address/token smoke remediation command, got %+v", nextSteps)
	}
}

func TestRunQuickstartCommandConfiguresProviderAndModelRoute(t *testing.T) {
	t.Setenv(cliProfilesPathEnvKey, filepath.Join(t.TempDir(), "profiles.json"))
	t.Setenv("PA_RUNTIME_ROOT_DIR", filepath.Join(t.TempDir(), "runtime-root"))

	secretManager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("create secret manager: %v", err)
	}
	setTestSecretManager(t, secretManager)

	dbPath := filepath.Join(t.TempDir(), "quickstart.db")
	server := startCLITestServerWithDaemonServices(t, dbPath, secretManager)

	tokenFile := filepath.Join(t.TempDir(), "quickstart.token")
	if err := controlauth.WriteTokenFile(tokenFile, "cli-test-token", false); err != nil {
		t.Fatalf("write quickstart token file: %v", err)
	}

	const apiKeyValue = "sk-quickstart-test-value"
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{
		"quickstart",
		"--workspace", "ws1",
		"--profile", "qs-success",
		"--mode", "tcp",
		"--address", server.Address(),
		"--token-file", tokenFile,
		"--provider", "openai",
		"--endpoint", "http://127.0.0.1:18080/v1",
		"--api-key", apiKeyValue,
		"--model", "gpt-4.1-mini",
		"--task-class", "chat",
		"--skip-doctor=true",
	}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("expected quickstart success, got %d (stderr=%s)", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	if strings.Contains(stdout.String(), apiKeyValue) {
		t.Fatalf("quickstart output leaked plaintext api key")
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode quickstart payload: %v", err)
	}
	if payload["overall_status"] != "pass" {
		t.Fatalf("expected overall_status=pass, got %v", payload["overall_status"])
	}
	if payload["success"] != true {
		t.Fatalf("expected success=true, got %v", payload["success"])
	}
	defaults, ok := payload["defaults"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults metadata, got %T", payload["defaults"])
	}
	workspaceSelection, ok := defaults["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.workspace metadata, got %T", defaults["workspace"])
	}
	if fmt.Sprint(workspaceSelection["value"]) != "ws1" || fmt.Sprint(workspaceSelection["source"]) != "explicit" {
		t.Fatalf("unexpected quickstart workspace defaults metadata: %+v", workspaceSelection)
	}
	profileSelection, ok := defaults["profile"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.profile metadata, got %T", defaults["profile"])
	}
	if fmt.Sprint(profileSelection["value"]) != "qs-success" || fmt.Sprint(profileSelection["source"]) != "explicit" {
		t.Fatalf("unexpected quickstart profile defaults metadata: %+v", profileSelection)
	}
	tokenFileSelection, ok := defaults["token_file"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults.token_file metadata, got %T", defaults["token_file"])
	}
	if fmt.Sprint(tokenFileSelection["value"]) != tokenFile || fmt.Sprint(tokenFileSelection["source"]) != "explicit" {
		t.Fatalf("unexpected quickstart token-file defaults metadata: %+v", tokenFileSelection)
	}
	overrideHints, ok := defaults["override_hints"].([]any)
	if !ok || len(overrideHints) < 3 {
		t.Fatalf("expected defaults.override_hints entries, got %v", defaults["override_hints"])
	}
	if quickstartStepByID(t, payload, "provider.configure")["status"] != "pass" {
		t.Fatalf("expected provider.configure status=pass")
	}
	if quickstartStepByID(t, payload, "model.route")["status"] != "pass" {
		t.Fatalf("expected model.route status=pass")
	}
	if quickstartStepByID(t, payload, "readiness.doctor")["status"] != "skipped" {
		t.Fatalf("expected readiness.doctor status=skipped when --skip-doctor=true")
	}

	client, err := transport.NewClient(transport.ClientConfig{
		ListenerMode: transport.ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "cli-test-token",
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}
	providers, err := client.ListProviders(context.Background(), transport.ProviderListRequest{WorkspaceID: "ws1"}, "quickstart-test.providers")
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	var openAIRecord *transport.ProviderConfigRecord
	for index := range providers.Providers {
		record := providers.Providers[index]
		if record.Provider == "openai" {
			openAIRecord = &record
			break
		}
	}
	if openAIRecord == nil {
		t.Fatalf("expected openai provider record after quickstart, got %+v", providers.Providers)
	}
	if !openAIRecord.APIKeyConfigured || openAIRecord.APIKeySecretName != "OPENAI_API_KEY" {
		t.Fatalf("expected openai api key configuration metadata, got %+v", *openAIRecord)
	}

	resolvedRoute, err := client.ResolveModelRoute(context.Background(), transport.ModelResolveRequest{
		WorkspaceID: "ws1",
		TaskClass:   "chat",
	}, "quickstart-test.resolve")
	if err != nil {
		t.Fatalf("resolve model route: %v", err)
	}
	if resolvedRoute.Provider != "openai" || resolvedRoute.ModelKey != "gpt-4.1-mini" {
		t.Fatalf("expected resolved route openai/gpt-4.1-mini, got %+v", resolvedRoute)
	}
}

func TestRunMetaSchemaRejectsUnknownSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"meta", "schem"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown meta subcommand") {
		t.Fatalf("expected unknown meta subcommand error, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), `did you mean "schema"?`) {
		t.Fatalf("expected meta subcommand suggestion, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "run `personal-agent help meta` to view available subcommands") {
		t.Fatalf("expected actionable meta help next step, got %q", stderr.String())
	}
}
