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

	localbridge "personalagent/runtime/internal/channels/adapters/localbridge"
	shared "personalagent/runtime/internal/shared/contracts"
)

type channelWorkerExecuteRequest struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

type channelWorkerExecuteResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type channelWorkerRuntime struct{}

const (
	channelWorkerTypeAppChat = "app_chat"
)

func runChannelWorker(workerType string, pluginID string, healthInterval time.Duration, dbPath string) error {
	_ = dbPath
	normalizedType := strings.ToLower(strings.TrimSpace(workerType))
	switch normalizedType {
	case channelWorkerTypeAppChat:
		// supported
	default:
		return fmt.Errorf("unsupported channel worker type %q", workerType)
	}

	trimmedPluginID := strings.TrimSpace(pluginID)
	if trimmedPluginID == "" {
		trimmedPluginID = defaultChannelWorkerPluginID(normalizedType)
	}
	if healthInterval <= 0 {
		healthInterval = 250 * time.Millisecond
	}
	execAuthToken, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		return err
	}
	runtime := &channelWorkerRuntime{}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen channel worker execute endpoint: %w", err)
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

		var payload channelWorkerExecuteRequest
		if statusCode, err := decodeWorkerExecuteJSONPayload(writer, request, &payload, "execute"); err != nil {
			writeChannelWorkerError(writer, statusCode, err)
			return
		}

		result, execErr := runtime.executeChannelWorkerOperation(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeChannelWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(channelWorkerExecuteResponse{Result: result}); err != nil {
			writeChannelWorkerError(writer, http.StatusInternalServerError, fmt.Errorf("encode worker response: %w", err))
		}
	})

	server := newWorkerExecuteHTTPServer(mux)
	go func() {
		_ = server.Serve(listener)
	}()

	metadata := shared.AdapterMetadata{
		ID:           trimmedPluginID,
		Kind:         shared.AdapterKindChannel,
		DisplayName:  channelWorkerDisplayName(normalizedType),
		Version:      "v1",
		Capabilities: channelWorkerCapabilities(normalizedType),
		Runtime: map[string]string{
			"exec_address": listener.Addr().String(),
		},
	}

	if err := emitWorkerMessage(workerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
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

func defaultChannelWorkerPluginID(workerType string) string {
	_ = workerType
	return "app_chat.daemon"
}

func channelWorkerDisplayName(workerType string) string {
	_ = workerType
	return "App Chat Channel Worker"
}

func channelWorkerCapabilities(workerType string) []shared.CapabilityDescriptor {
	switch strings.ToLower(strings.TrimSpace(workerType)) {
	case channelWorkerTypeAppChat:
		return []shared.CapabilityDescriptor{
			{Key: "channel.app_chat.send"},
			{Key: "channel.app_chat.status"},
		}
	default:
		return nil
	}
}

func (r *channelWorkerRuntime) executeChannelWorkerOperation(ctx context.Context, operation string, payload json.RawMessage) (any, error) {
	switch operation {
	case "app_chat_send":
		var request localbridge.AppChatSendRequest
		if err := decodeChannelWorkerPayload(payload, &request); err != nil {
			return nil, err
		}
		return localbridge.SendAppChat(ctx, request)
	case "app_chat_status":
		return localbridge.AppChatStatus(), nil
	default:
		return nil, fmt.Errorf("unsupported channel worker operation %q", operation)
	}
}

func decodeChannelWorkerPayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return fmt.Errorf("payload is required")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

func writeChannelWorkerError(writer http.ResponseWriter, statusCode int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(channelWorkerExecuteResponse{
		Error: err.Error(),
	})
}
