package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	connectorWorkerExecAddressKey        = "exec_address"
	defaultTwilioWorkerHTTPClientTimeout = 4 * time.Second
)

var twilioWorkerHTTPClient = &http.Client{Timeout: defaultTwilioWorkerHTTPClientTimeout}

type workerExecuteRequest struct {
	ExecutionContext connectorcontract.ExecutionContext `json:"execution_context"`
	Step             connectorcontract.TaskStep         `json:"step"`
}

type channelConnectorExecuteRequest struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

type channelConnectorExecuteResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type workerMessage struct {
	Type    string                  `json:"type"`
	Plugin  *shared.AdapterMetadata `json:"plugin,omitempty"`
	Healthy *bool                   `json:"healthy,omitempty"`
}

func runConnectorWorker(connectorType string, pluginID string, healthInterval time.Duration, dbPath string) error {
	if strings.EqualFold(strings.TrimSpace(connectorType), "cloudflared") {
		return runCloudflaredConnectorWorker(pluginID, healthInterval)
	}
	if strings.EqualFold(strings.TrimSpace(connectorType), "messages") {
		return runMessagesConnectorWorker(pluginID, healthInterval)
	}
	if strings.EqualFold(strings.TrimSpace(connectorType), "twilio") {
		return runTwilioConnectorWorker(pluginID, healthInterval)
	}

	adapter, err := connectorAdapterForWorker(connectorType, pluginID, dbPath)
	if err != nil {
		return err
	}
	execAuthToken, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		return err
	}
	if healthInterval <= 0 {
		healthInterval = 250 * time.Millisecond
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen worker execute endpoint: %w", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !authorizeWorkerExecuteRequest(request, execAuthToken) {
			writeWorkerUnauthorized(writer)
			return
		}

		var payload workerExecuteRequest
		if statusCode, err := decodeWorkerExecuteJSONPayload(writer, request, &payload, "execute"); err != nil {
			writeWorkerError(writer, statusCode, err)
			return
		}

		result, execErr := adapter.ExecuteStep(request.Context(), payload.ExecutionContext, payload.Step)
		if execErr != nil {
			writeWorkerError(writer, http.StatusInternalServerError, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(result); err != nil {
			writeWorkerError(writer, http.StatusInternalServerError, fmt.Errorf("encode execute response: %w", err))
			return
		}
	})

	server := newWorkerExecuteHTTPServer(mux)
	go func() {
		_ = server.Serve(listener)
	}()

	metadata := adapter.Metadata()
	if metadata.Runtime == nil {
		metadata.Runtime = map[string]string{}
	}
	metadata.Runtime[connectorWorkerExecAddressKey] = listener.Addr().String()

	if err := emitWorkerMessage(workerMessage{Type: "handshake", Plugin: &metadata}); err != nil {
		return fmt.Errorf("emit handshake: %w", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy := true
			if err := emitWorkerMessage(workerMessage{Type: "health", Healthy: &healthy}); err != nil {
				return fmt.Errorf("emit health: %w", err)
			}
		case <-signals:
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		}
	}
}

func runTwilioConnectorWorker(pluginID string, healthInterval time.Duration) error {
	trimmedPluginID := strings.TrimSpace(pluginID)
	if trimmedPluginID == "" {
		trimmedPluginID = "twilio.daemon"
	}
	if healthInterval <= 0 {
		healthInterval = 250 * time.Millisecond
	}
	execAuthToken, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen twilio connector worker execute endpoint: %w", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !authorizeWorkerExecuteRequest(request, execAuthToken) {
			writeWorkerUnauthorized(writer)
			return
		}

		var payload channelConnectorExecuteRequest
		if statusCode, err := decodeWorkerExecuteJSONPayload(writer, request, &payload, "twilio execute"); err != nil {
			writeWorkerError(writer, statusCode, err)
			return
		}

		result, execErr := executeTwilioConnectorWorkerOperation(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(channelConnectorExecuteResponse{Result: result}); err != nil {
			writeWorkerError(writer, http.StatusInternalServerError, fmt.Errorf("encode twilio execute response: %w", err))
		}
	})

	server := newWorkerExecuteHTTPServer(mux)
	go func() {
		_ = server.Serve(listener)
	}()

	metadata := shared.AdapterMetadata{
		ID:          trimmedPluginID,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Twilio Connector Worker",
		Version:     "v1",
		Capabilities: []shared.CapabilityDescriptor{
			{Key: "channel.twilio.check"},
			{Key: "channel.twilio.sms.send"},
			{Key: "channel.twilio.voice.start_call"},
		},
		Runtime: map[string]string{
			connectorWorkerExecAddressKey: listener.Addr().String(),
		},
	}

	if err := emitWorkerMessage(workerMessage{Type: "handshake", Plugin: &metadata}); err != nil {
		return fmt.Errorf("emit twilio connector handshake: %w", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy := true
			if err := emitWorkerMessage(workerMessage{Type: "health", Healthy: &healthy}); err != nil {
				return fmt.Errorf("emit twilio connector health: %w", err)
			}
		case <-signals:
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		}
	}
}

func runMessagesConnectorWorker(pluginID string, healthInterval time.Duration) error {
	trimmedPluginID := strings.TrimSpace(pluginID)
	if trimmedPluginID == "" {
		trimmedPluginID = "messages.daemon"
	}
	if healthInterval <= 0 {
		healthInterval = 250 * time.Millisecond
	}
	execAuthToken, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen imessage connector worker execute endpoint: %w", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !authorizeWorkerExecuteRequest(request, execAuthToken) {
			writeWorkerUnauthorized(writer)
			return
		}

		var payload channelConnectorExecuteRequest
		if statusCode, err := decodeWorkerExecuteJSONPayload(writer, request, &payload, "messages execute"); err != nil {
			writeWorkerError(writer, statusCode, err)
			return
		}

		result, execErr := executeMessagesConnectorWorkerOperation(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(channelConnectorExecuteResponse{Result: result}); err != nil {
			writeWorkerError(writer, http.StatusInternalServerError, fmt.Errorf("encode messages execute response: %w", err))
		}
	})

	server := newWorkerExecuteHTTPServer(mux)
	go func() {
		_ = server.Serve(listener)
	}()

	metadata := shared.AdapterMetadata{
		ID:          trimmedPluginID,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "iMessage Connector Worker",
		Version:     "v1",
		Capabilities: []shared.CapabilityDescriptor{
			{Key: "channel.messages.send"},
			{Key: "channel.messages.status"},
			{Key: "channel.messages.ingest_poll"},
		},
		Runtime: map[string]string{
			connectorWorkerExecAddressKey: listener.Addr().String(),
		},
	}

	if err := emitWorkerMessage(workerMessage{Type: "handshake", Plugin: &metadata}); err != nil {
		return fmt.Errorf("emit imessage connector handshake: %w", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy := true
			if err := emitWorkerMessage(workerMessage{Type: "health", Healthy: &healthy}); err != nil {
				return fmt.Errorf("emit imessage connector health: %w", err)
			}
		case <-signals:
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		}
	}
}

func executeTwilioConnectorWorkerOperation(ctx context.Context, operation string, payload json.RawMessage) (any, error) {
	switch operation {
	case "twilio_check":
		var request channelcheck.TwilioRequest
		if err := decodeChannelConnectorPayload(payload, &request); err != nil {
			return nil, err
		}
		return channelcheck.CheckTwilio(ctx, twilioWorkerHTTPClient, request)
	case "twilio_sms_send":
		var request twilioadapter.SMSAPIRequest
		if err := decodeChannelConnectorPayload(payload, &request); err != nil {
			return nil, err
		}
		return twilioadapter.SendSMS(ctx, twilioWorkerHTTPClient, request)
	case "twilio_voice_start_call":
		var request twilioadapter.VoiceCallRequest
		if err := decodeChannelConnectorPayload(payload, &request); err != nil {
			return nil, err
		}
		return twilioadapter.StartVoiceCall(ctx, twilioWorkerHTTPClient, request)
	default:
		return nil, fmt.Errorf("unsupported twilio connector worker operation %q", operation)
	}
}

func executeMessagesConnectorWorkerOperation(ctx context.Context, operation string, payload json.RawMessage) (any, error) {
	switch operation {
	case "messages_send":
		var request messagesadapter.SendRequest
		if err := decodeChannelConnectorPayload(payload, &request); err != nil {
			return nil, err
		}
		return messagesadapter.Send(ctx, request)
	case "messages_status":
		var request messagesadapter.StatusRequest
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &request); err != nil {
				return nil, fmt.Errorf("decode payload: %w", err)
			}
		}
		return messagesadapter.Status(request), nil
	case "messages_poll_inbound":
		var request messagesadapter.InboundPollRequest
		if err := decodeChannelConnectorPayload(payload, &request); err != nil {
			return nil, err
		}
		return messagesadapter.PollInbound(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported imessage connector worker operation %q", operation)
	}
}

func decodeChannelConnectorPayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return fmt.Errorf("payload is required")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

func connectorAdapterForWorker(connectorType string, pluginID string, dbPath string) (connectorcontract.Adapter, error) {
	trimmedType := strings.ToLower(strings.TrimSpace(connectorType))
	trimmedID := strings.TrimSpace(pluginID)
	if trimmedID == "" {
		trimmedID = "connector." + trimmedType + ".worker"
	}

	switch trimmedType {
	case "mail":
		return mailadapter.NewAdapterWithDBPath(trimmedID, dbPath), nil
	case "calendar":
		return calendaradapter.NewAdapter(trimmedID), nil
	case "browser":
		return browseradapter.NewAdapter(trimmedID), nil
	case "finder":
		return finderadapter.NewAdapter(trimmedID), nil
	default:
		return nil, fmt.Errorf("unsupported connector worker type %q", connectorType)
	}
}

func emitWorkerMessage(message workerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(payload))
	return err
}

func writeWorkerError(writer http.ResponseWriter, statusCode int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"error": err.Error(),
	})
}
