package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	modeOpenAI = "openai"
	modeTwilio = "twilio"
	modeAll    = "all"
)

type runningServer struct {
	name     string
	server   *http.Server
	listener net.Listener
	done     chan error
}

func main() {
	mode := flag.String("mode", modeAll, "mock server mode: openai|twilio|all")
	openAIListen := flag.String("openai-listen", "127.0.0.1:18080", "listen address for OpenAI mock")
	twilioListen := flag.String("twilio-listen", "127.0.0.1:19080", "listen address for Twilio mock")
	flag.Parse()

	selectedMode := strings.ToLower(strings.TrimSpace(*mode))
	if selectedMode != modeOpenAI && selectedMode != modeTwilio && selectedMode != modeAll {
		fmt.Fprintf(os.Stderr, "unsupported --mode %q\n", *mode)
		os.Exit(2)
	}

	servers := make([]*runningServer, 0, 2)
	if selectedMode == modeOpenAI || selectedMode == modeAll {
		server, err := startServer("openai", strings.TrimSpace(*openAIListen), newOpenAIMockMux())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to start openai mock server: %v\n", err)
			os.Exit(1)
		}
		servers = append(servers, server)
	}
	if selectedMode == modeTwilio || selectedMode == modeAll {
		server, err := startServer("twilio", strings.TrimSpace(*twilioListen), newTwilioMockMux())
		if err != nil {
			for _, running := range servers {
				_ = running.listener.Close()
			}
			fmt.Fprintf(os.Stderr, "failed to start twilio mock server: %v\n", err)
			os.Exit(1)
		}
		servers = append(servers, server)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, len(servers))
	for _, server := range servers {
		go func(running *runningServer) {
			if err := <-running.done; err != nil {
				serverErr <- fmt.Errorf("%s mock server error: %w", running.name, err)
			}
		}(server)
	}

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		fmt.Fprintln(os.Stderr, err.Error())
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, server := range servers {
		_ = server.server.Shutdown(shutdownCtx)
	}
}

func startServer(name string, listenAddress string, handler http.Handler) (*runningServer, error) {
	if strings.TrimSpace(listenAddress) == "" {
		return nil, fmt.Errorf("%s listen address is required", name)
	}
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return nil, err
	}

	server := &http.Server{Handler: handler}
	done := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			done <- nil
			return
		}
		done <- err
	}()

	fmt.Fprintf(os.Stdout, "%s mock listening on http://%s\n", name, listener.Addr().String())
	return &runningServer{
		name:     name,
		server:   server,
		listener: listener,
		done:     done,
	}, nil
}

func newOpenAIMockMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"data": []map[string]string{
				{"id": "gpt-4.1-mini"},
			},
		})
	})
	mux.HandleFunc("/v1/chat/completions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.WriteHeader(http.StatusOK)

		flusher, ok := writer.(http.Flusher)
		if !ok {
			return
		}
		_, _ = writer.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"mock\"}}]}\n\n"))
		flusher.Flush()
		_, _ = writer.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" reply\"}}]}\n\n"))
		flusher.Flush()
		_, _ = writer.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	})
	return mux
}

type twilioMock struct {
	mu           sync.Mutex
	messageCount int
	callCount    int
}

func newTwilioMockMux() http.Handler {
	mock := &twilioMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/2010-04-01/Accounts/", mock.handleAccountRoute)
	return mux
}

func (mock *twilioMock) handleAccountRoute(writer http.ResponseWriter, request *http.Request) {
	cleanPath := path.Clean(strings.TrimSpace(request.URL.Path))
	if !strings.HasPrefix(cleanPath, "/2010-04-01/Accounts/") {
		http.NotFound(writer, request)
		return
	}

	segments := strings.Split(strings.TrimPrefix(cleanPath, "/2010-04-01/Accounts/"), "/")
	if len(segments) == 0 || strings.TrimSpace(segments[0]) == "" {
		http.NotFound(writer, request)
		return
	}

	accountToken := strings.TrimSpace(segments[0])
	accountSID := strings.TrimSuffix(accountToken, ".json")
	if accountSID == "" {
		http.NotFound(writer, request)
		return
	}

	// GET /2010-04-01/Accounts/{AccountSid}.json
	if request.Method == http.MethodGet && len(segments) == 1 && strings.HasSuffix(accountToken, ".json") {
		writeJSON(writer, http.StatusOK, map[string]any{
			"sid":    accountSID,
			"status": "active",
		})
		return
	}

	// POST /2010-04-01/Accounts/{AccountSid}/Messages.json
	if request.Method == http.MethodPost && len(segments) == 2 && strings.EqualFold(segments[1], "Messages.json") {
		if err := request.ParseForm(); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]any{"error": "invalid form payload"})
			return
		}
		mock.mu.Lock()
		mock.messageCount++
		messageSID := fmt.Sprintf("SMMOCK%04d", mock.messageCount)
		mock.mu.Unlock()

		writeJSON(writer, http.StatusOK, map[string]any{
			"sid":         messageSID,
			"account_sid": accountSID,
			"status":      "queued",
			"from":        firstNonEmpty(request.Form.Get("From"), "+15555550001"),
			"to":          firstNonEmpty(request.Form.Get("To"), "+15555550999"),
		})
		return
	}

	// POST /2010-04-01/Accounts/{AccountSid}/Calls.json
	if request.Method == http.MethodPost && len(segments) == 2 && strings.EqualFold(segments[1], "Calls.json") {
		if err := request.ParseForm(); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]any{"error": "invalid form payload"})
			return
		}
		mock.mu.Lock()
		mock.callCount++
		callSID := fmt.Sprintf("CAMOCK%04d", mock.callCount)
		mock.mu.Unlock()

		writeJSON(writer, http.StatusOK, map[string]any{
			"sid":         callSID,
			"account_sid": accountSID,
			"status":      "queued",
			"direction":   "outbound-api",
			"from":        firstNonEmpty(request.Form.Get("From"), "+15555550002"),
			"to":          firstNonEmpty(request.Form.Get("To"), "+15555550999"),
		})
		return
	}

	http.NotFound(writer, request)
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func firstNonEmpty(values ...string) string {
	for _, candidate := range values {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
