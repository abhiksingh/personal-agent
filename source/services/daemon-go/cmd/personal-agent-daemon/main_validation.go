package main

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func validateDaemonRunConfig(config daemonRunConfig) error {
	profile, err := normalizeDaemonRuntimeProfile(config.runtimeProfile)
	if err != nil {
		return err
	}
	if _, err := parseDaemonWebSocketOriginAllowlist(config.webSocketOriginAllowlist); err != nil {
		return err
	}
	if _, err := normalizeDaemonLifecycleHostOpsMode(config.lifecycleHostOpsMode); err != nil {
		return err
	}
	if err := validateDaemonHTTPServerConfig(config); err != nil {
		return err
	}
	if err := validateDaemonTLSConfig(config); err != nil {
		return err
	}
	if err := validateDaemonListenAddress(config.listenerMode, config.listenAddress, config.allowNonLocal, hasDaemonTLSServerConfig(config)); err != nil {
		return err
	}

	authToken := strings.TrimSpace(config.authToken)
	if authToken == "" {
		return fmt.Errorf("--auth-token is required")
	}
	nonLocalTCPBind, bindErr := isNonLocalTCPBind(config.listenerMode, config.listenAddress)
	if bindErr != nil {
		return bindErr
	}
	if nonLocalTCPBind && len(authToken) < 24 {
		return fmt.Errorf(
			"tcp non-local --listen-address requires auth token length >= 24 (use --auth-token-file or generate a stronger token)",
		)
	}
	if profile == daemonRuntimeProfileProd {
		if len(config.authTokenScopes) == 0 {
			return fmt.Errorf("--runtime-profile=prod requires explicit --auth-token-scopes least-privilege policy (example: --auth-token-scopes chat:write,tasks:read,tasks:write)")
		}
		if config.authTokenSource != daemonAuthTokenSourceFile {
			return fmt.Errorf("--runtime-profile=prod requires --auth-token-file")
		}
		if len(authToken) < 24 {
			return fmt.Errorf("--runtime-profile=prod requires auth token length >= 24")
		}
		if strings.ToLower(strings.TrimSpace(config.listenerMode)) != string(transport.ListenerModeTCP) {
			return fmt.Errorf("--runtime-profile=prod requires --listen-mode=tcp")
		}
		if !hasDaemonTLSServerConfig(config) {
			return fmt.Errorf("--runtime-profile=prod requires --tls-cert-file and --tls-key-file")
		}
		if !config.tlsRequireClientCert || strings.TrimSpace(config.tlsClientCAFile) == "" {
			return fmt.Errorf("--runtime-profile=prod requires mTLS (--tls-require-client-cert with --tls-client-ca-file)")
		}
	}
	return nil
}

func daemonAuthScopeWarnings(config daemonRunConfig) []string {
	profile := strings.ToLower(strings.TrimSpace(config.runtimeProfile))
	scopes := append([]string(nil), config.authTokenScopes...)
	if len(scopes) == 0 {
		if profile == daemonRuntimeProfileProd {
			return nil
		}
		return []string{
			"--auth-token-scopes omitted; defaulting to wildcard '*' (full control-plane access). Configure least-privilege scopes explicitly (example: --auth-token-scopes chat:write,tasks:read,tasks:write).",
		}
	}
	if hasWildcardDaemonAuthScope(scopes) {
		message := "--auth-token-scopes contains wildcard '*' (full control-plane access). Prefer least-privilege scoped access."
		if profile == daemonRuntimeProfileProd {
			message = "--runtime-profile=prod configured with wildcard '*' auth scope (full control-plane access). Prefer least-privilege scoped access."
		}
		return []string{message}
	}
	return nil
}

func validateDaemonHTTPServerConfig(config daemonRunConfig) error {
	if config.readHeaderTimeout < 0 {
		return fmt.Errorf("--read-header-timeout must be >= 0")
	}
	if config.readTimeout < 0 {
		return fmt.Errorf("--read-timeout must be >= 0")
	}
	if config.writeTimeout < 0 {
		return fmt.Errorf("--write-timeout must be >= 0")
	}
	if config.idleTimeout < 0 {
		return fmt.Errorf("--idle-timeout must be >= 0")
	}
	if config.maxHeaderBytes < 0 {
		return fmt.Errorf("--max-header-bytes must be >= 0")
	}
	if config.realtimeMaxConnections < 0 {
		return fmt.Errorf("--realtime-max-connections must be >= 0")
	}
	if config.realtimeMaxSubscriptions < 0 {
		return fmt.Errorf("--realtime-max-subscriptions must be >= 0")
	}
	return nil
}

func resolveDaemonRuntimeProfile(runtimeProfile string) (string, error) {
	return normalizeDaemonRuntimeProfile(runtimeProfile)
}

func normalizeDaemonRuntimeProfile(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return daemonRuntimeProfileLocal, nil
	}
	switch normalized {
	case daemonRuntimeProfileLocal:
		return daemonRuntimeProfileLocal, nil
	case daemonRuntimeProfileProd:
		return daemonRuntimeProfileProd, nil
	default:
		return "", fmt.Errorf("unsupported --runtime-profile %q", raw)
	}
}

func parseDaemonWebSocketOriginAllowlist(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	return transport.NormalizeWebSocketOriginAllowlist(strings.Split(trimmed, ","))
}

func parseDaemonAuthTokenScopes(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	normalized := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		scope := strings.ToLower(strings.TrimSpace(part))
		if scope == "" {
			continue
		}
		if scope == "*" || scope == "all" {
			return []string{"*"}
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func hasWildcardDaemonAuthScope(scopes []string) bool {
	for _, scope := range scopes {
		normalized := strings.ToLower(strings.TrimSpace(scope))
		if normalized == "*" {
			return true
		}
	}
	return false
}

func validateDaemonListenAddress(listenerMode string, listenAddress string, allowNonLocal bool, tlsEnabled bool) error {
	nonLocalTCPBind, err := isNonLocalTCPBind(listenerMode, listenAddress)
	if err != nil {
		return err
	}
	if !nonLocalTCPBind {
		return nil
	}

	if !allowNonLocal {
		return fmt.Errorf("tcp --listen-address %q is non-local; use --allow-non-local-bind to override", listenAddress)
	}
	if !tlsEnabled {
		return fmt.Errorf("tcp --listen-address %q is non-local; secure mode requires --tls-cert-file and --tls-key-file", listenAddress)
	}
	return nil
}

func isNonLocalTCPBind(listenerMode string, listenAddress string) (bool, error) {
	mode := strings.ToLower(strings.TrimSpace(listenerMode))
	if mode == "" {
		mode = string(transport.ListenerModeTCP)
	}
	if mode != string(transport.ListenerModeTCP) {
		return false, nil
	}

	address := strings.TrimSpace(listenAddress)
	if address == "" {
		return false, nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false, fmt.Errorf("invalid tcp --listen-address %q: %w", listenAddress, err)
	}
	host = strings.TrimSpace(host)
	return host == "" || !isLocalBindHost(host), nil
}

func isLocalBindHost(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return false
	}
	if strings.EqualFold(trimmed, "localhost") {
		return true
	}
	ip := net.ParseIP(trimmed)
	return ip != nil && ip.IsLoopback()
}
