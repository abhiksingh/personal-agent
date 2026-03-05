package transport

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	config                ServerConfig
	realtimeOrigins       realtimeOriginPolicy
	authTokenScopes       []string
	authTokenScopeSet     map[string]struct{}
	backend               ControlBackend
	broker                *EventBroker
	lifecycle             DaemonLifecycleService
	workflowQueries       WorkflowQueryService
	secretRefs            SecretReferenceService
	providers             ProviderService
	models                ModelService
	chat                  ChatService
	agent                 AgentService
	delegation            DelegationService
	comm                  CommService
	twilio                TwilioChannelService
	cloudflared           CloudflaredConnectorService
	automation            AutomationService
	inspect               InspectService
	retention             RetentionService
	contextOps            ContextOpsService
	uiStatus              UIStatusService
	identityDirectory     IdentityDirectoryService
	controlRateLimiter    *controlEndpointRateLimiter
	realtimeMu            sync.Mutex
	realtimeConnections   int
	realtimeSubscriptions int
	httpServer            *http.Server
	listener              net.Listener
	unixPath              string
	closeOnce             sync.Once
	mu                    sync.RWMutex
}

const (
	defaultServerReadHeaderTimeout                                 = 5 * time.Second
	defaultServerReadTimeout                                       = 15 * time.Second
	defaultServerWriteTimeout                                      = 30 * time.Second
	defaultServerIdleTimeout                                       = 120 * time.Second
	defaultServerMaxHeaderBytes                                    = 1 << 20 // 1 MiB
	defaultRequestBodyBytesLimit                                   = int64(1 << 20)
	defaultControlRateLimitWindow                                  = 1 * time.Second
	defaultControlRateLimitMaxRequests                             = 30
	defaultRealtimeReadLimitBytes                                  = int64(64 * 1024)
	defaultRealtimeWriteTimeout                                    = 10 * time.Second
	defaultRealtimePongTimeout                                     = 30 * time.Second
	defaultRealtimePingInterval                                    = 10 * time.Second
	defaultRealtimeMaxConnections, defaultRealtimeMaxSubscriptions = 64, 64
	responseContentTypeJSON, responseContentTypeProblem            = "application/json", "application/problem+json"
	responseHeaderCorrelationID, responseHeaderAPIVersion          = "X-Correlation-ID", "X-PersonalAgent-API-Version"
	responseHeaderCurrentAPIVer                                    = "v1"
)

func NewServer(config ServerConfig, backend ControlBackend, broker *EventBroker) (*Server, error) {
	if backend == nil {
		return nil, errors.New("control backend is required")
	}
	if strings.TrimSpace(config.AuthToken) == "" {
		return nil, errors.New("auth token is required")
	}
	config.AuthTokenScopes = normalizeAuthTokenScopes(config.AuthTokenScopes)
	if config.ListenerMode == "" {
		config.ListenerMode = ListenerModeTCP
	}
	if config.ListenerMode == ListenerModeTCP && strings.TrimSpace(config.Address) == "" {
		config.Address = DefaultTCPAddress
	}
	if config.ListenerMode == ListenerModeNamedPipe {
		trimmed := strings.TrimSpace(config.Address)
		if trimmed == "" || trimmed == DefaultTCPAddress {
			config.Address = DefaultNamedPipeAddress
		}
	}
	if config.TLSConfig != nil && config.ListenerMode != ListenerModeTCP {
		return nil, errors.New("tls is only supported for tcp listener mode")
	}
	if config.ReadHeaderTimeout <= 0 {
		config.ReadHeaderTimeout = defaultServerReadHeaderTimeout
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = defaultServerReadTimeout
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = defaultServerWriteTimeout
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = defaultServerIdleTimeout
	}
	if config.MaxHeaderBytes <= 0 {
		config.MaxHeaderBytes = defaultServerMaxHeaderBytes
	}
	if config.RequestBodyBytesLimit <= 0 {
		config.RequestBodyBytesLimit = defaultRequestBodyBytesLimit
	}
	if config.ControlRateLimitWindow <= 0 {
		config.ControlRateLimitWindow = defaultControlRateLimitWindow
	}
	if config.ControlRateLimitMaxRequests <= 0 {
		config.ControlRateLimitMaxRequests = defaultControlRateLimitMaxRequests
	}
	if config.RealtimeReadLimitBytes <= 0 {
		config.RealtimeReadLimitBytes = defaultRealtimeReadLimitBytes
	}
	if config.RealtimeWriteTimeout <= 0 {
		config.RealtimeWriteTimeout = defaultRealtimeWriteTimeout
	}
	if config.RealtimePongTimeout <= 0 {
		config.RealtimePongTimeout = defaultRealtimePongTimeout
	}
	if config.RealtimePingInterval <= 0 {
		config.RealtimePingInterval = defaultRealtimePingInterval
	}
	if config.RealtimePingInterval >= config.RealtimePongTimeout {
		config.RealtimePingInterval = config.RealtimePongTimeout / 2
		if config.RealtimePingInterval <= 0 {
			config.RealtimePingInterval = time.Second
		}
	}
	if config.RealtimeMaxConnections <= 0 {
		config.RealtimeMaxConnections = defaultRealtimeMaxConnections
	}
	if config.RealtimeMaxSubscriptions <= 0 {
		config.RealtimeMaxSubscriptions = defaultRealtimeMaxSubscriptions
	}
	if broker == nil {
		broker = NewEventBroker()
	}
	realtimeOrigins, err := newRealtimeOriginPolicy(config.RuntimeProfile, config.WebSocketOriginAllowlist)
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:            config,
		realtimeOrigins:   realtimeOrigins,
		authTokenScopes:   append([]string(nil), config.AuthTokenScopes...),
		authTokenScopeSet: scopeSetFromList(config.AuthTokenScopes),
		backend:           backend,
		broker:            broker,
		lifecycle:         config.DaemonLifecycle,
		workflowQueries:   config.WorkflowQueries,
		secretRefs:        config.SecretReferences,
		providers:         config.Providers,
		models:            config.Models,
		chat:              config.Chat,
		agent:             config.Agent,
		delegation:        config.Delegation,
		comm:              config.Comm,
		twilio:            config.Twilio,
		cloudflared:       config.Cloudflared,
		automation:        config.Automation,
		inspect:           config.Inspect,
		retention:         config.Retention,
		contextOps:        config.ContextOps,
		uiStatus:          config.UIStatus,
		identityDirectory: config.IdentityDirectory,
		controlRateLimiter: newControlEndpointRateLimiter(
			config.ControlRateLimitMaxRequests,
			config.ControlRateLimitWindow,
			nil,
		),
	}

	mux := http.NewServeMux()
	server.registerRoutes(mux)

	server.httpServer = &http.Server{
		Handler:           server.wrapWithUnknownRouteFallback(mux),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}

	return server, nil
}

func (s *Server) Start() error {
	listener, unixPath, err := createListener(s.config)
	if err != nil {
		return err
	}
	if s.config.TLSConfig != nil {
		listener = tls.NewListener(listener, s.config.TLSConfig)
	}

	s.mu.Lock()
	s.listener = listener
	s.unixPath = unixPath
	s.mu.Unlock()

	go func() {
		_ = s.httpServer.Serve(listener)
	}()
	return nil
}

func (s *Server) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var shutdownErr error
	s.closeOnce.Do(func() {
		if s.broker != nil {
			s.broker.Close()
		}

		s.mu.RLock()
		listener := s.listener
		unixPath := s.unixPath
		s.mu.RUnlock()

		if s.httpServer != nil {
			shutdownErr = s.httpServer.Shutdown(ctx)
		}
		if listener != nil {
			_ = listener.Close()
		}
		if unixPath != "" {
			_ = os.Remove(unixPath)
		}
	})
	return shutdownErr
}

func (s *Server) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func createListener(config ServerConfig) (net.Listener, string, error) {
	switch config.ListenerMode {
	case ListenerModeTCP:
		listener, err := net.Listen("tcp", config.Address)
		if err != nil {
			return nil, "", fmt.Errorf("listen tcp %s: %w", config.Address, err)
		}
		return listener, "", nil
	case ListenerModeUnix:
		if strings.TrimSpace(config.Address) == "" {
			return nil, "", errors.New("unix listener requires address path")
		}
		if err := ensurePrivateUnixSocketParentDir(config.Address); err != nil {
			return nil, "", fmt.Errorf("create unix listener directory: %w", err)
		}
		_ = os.Remove(config.Address)
		listener, err := net.Listen("unix", config.Address)
		if err != nil {
			return nil, "", fmt.Errorf("listen unix %s: %w", config.Address, err)
		}
		if err := enforceUnixSocketFileMode(config.Address); err != nil {
			_ = listener.Close()
			_ = os.Remove(config.Address)
			return nil, "", fmt.Errorf("set unix listener socket permissions: %w", err)
		}
		return listener, config.Address, nil
	case ListenerModeNamedPipe:
		return createNamedPipeListener(config.Address)
	default:
		return nil, "", fmt.Errorf("unsupported listener mode %q", config.ListenerMode)
	}
}

func (s *Server) wrapWithUnknownRouteFallback(mux *http.ServeMux) http.Handler {
	if mux == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !isControlV1Path(request.URL.Path) {
			mux.ServeHTTP(writer, request)
			return
		}

		_, pattern := mux.Handler(request)
		if strings.TrimSpace(pattern) != "" {
			mux.ServeHTTP(writer, request)
			return
		}

		correlationID, ok := s.authorize(writer, request)
		if !ok {
			return
		}

		message := "unknown control route"
		errorEnvelope := buildTransportErrorEnvelope(
			http.StatusNotFound,
			message,
			correlationID,
			map[string]any{
				"path":   request.URL.Path,
				"method": request.Method,
			},
		)
		writeJSONWithContentType(writer, http.StatusNotFound, map[string]any{
			"error":          errorEnvelope.Error,
			"correlation_id": errorEnvelope.CorrelationID,
			"type":           errorEnvelope.Type,
			"title":          errorEnvelope.Title,
			"status":         errorEnvelope.Status,
			"detail":         errorEnvelope.Detail,
			"instance":       errorEnvelope.Instance,
			"path":           request.URL.Path,
			"method":         request.Method,
		}, correlationID, responseContentTypeProblem)
	})
}

func isControlV1Path(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "/v1" {
		return true
	}
	return strings.HasPrefix(trimmed, "/v1/")
}

func (s *Server) authorize(writer http.ResponseWriter, request *http.Request) (string, bool) {
	correlationID := strings.TrimSpace(request.Header.Get("X-Correlation-ID"))
	if correlationID == "" {
		correlationID = mustRandomID()
	}
	if !authorizeBearerToken(request, s.config.AuthToken) {
		writeJSONError(writer, http.StatusUnauthorized, "unauthorized", correlationID)
		return "", false
	}
	scopeAuthorized, requiredScopes := s.routeScopeAllowed(request)
	if !scopeAuthorized {
		writeJSONErrorWithDetails(writer, http.StatusForbidden, "forbidden: insufficient authorization scope", correlationID, map[string]any{
			"category":        "auth_scope",
			"required_scopes": requiredScopes,
			"granted_scopes":  append([]string(nil), s.authTokenScopes...),
			"path":            strings.TrimSpace(request.URL.Path),
			"method":          strings.ToUpper(strings.TrimSpace(request.Method)),
		})
		return "", false
	}
	return correlationID, true
}

func authorizeBearerToken(request *http.Request, expectedToken string) bool {
	if request == nil {
		return false
	}
	trimmedExpected := strings.TrimSpace(expectedToken)
	if trimmedExpected == "" {
		return false
	}
	authValue := strings.TrimSpace(request.Header.Get("Authorization"))
	expectedAuth := "Bearer " + trimmedExpected
	if len(authValue) != len(expectedAuth) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(authValue), []byte(expectedAuth)) == 1
}

func writeJSONError(writer http.ResponseWriter, statusCode int, message string, correlationID string) {
	writeJSONWithContentType(writer, statusCode, buildTransportErrorEnvelope(statusCode, message, correlationID, defaultTransportErrorDetails(statusCode, message)), correlationID, responseContentTypeProblem)
}

func writeMethodNotAllowed(writer http.ResponseWriter, method string) {
	writer.Header().Set("Allow", method)
	writer.WriteHeader(http.StatusMethodNotAllowed)
}

func firstNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
