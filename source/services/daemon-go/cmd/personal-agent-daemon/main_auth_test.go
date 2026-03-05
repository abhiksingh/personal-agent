package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/controlauth"
)

func TestNormalizeDaemonRuntimeProfile(t *testing.T) {
	value, err := normalizeDaemonRuntimeProfile("")
	if err != nil {
		t.Fatalf("normalize empty runtime profile: %v", err)
	}
	if value != daemonRuntimeProfileLocal {
		t.Fatalf("expected local default, got %q", value)
	}

	value, err = normalizeDaemonRuntimeProfile("PrOd")
	if err != nil {
		t.Fatalf("normalize prod runtime profile: %v", err)
	}
	if value != daemonRuntimeProfileProd {
		t.Fatalf("expected prod profile, got %q", value)
	}

	if _, err := normalizeDaemonRuntimeProfile("PrOdUcTiOn"); err == nil {
		t.Fatalf("expected unsupported runtime profile for legacy production alias")
	}

	if _, err := normalizeDaemonRuntimeProfile("staging"); err == nil {
		t.Fatalf("expected invalid runtime profile error")
	}
}

func TestValidateDaemonRunConfig(t *testing.T) {
	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "127.0.0.1:7071",
		runtimeProfile: daemonRuntimeProfileProd,
		authToken:      defaultDaemonAuthToken,
	}); err == nil {
		t.Fatalf("expected prod empty token to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:         "tcp",
		listenAddress:        "127.0.0.1:7071",
		runtimeProfile:       daemonRuntimeProfileLocal,
		authToken:            "dev-token",
		lifecycleHostOpsMode: "invalid-mode",
	}); err == nil {
		t.Fatalf("expected invalid lifecycle host-ops mode to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "127.0.0.1:7071",
		runtimeProfile: daemonRuntimeProfileProd,
		authToken:      "custom-prod-token",
	}); err == nil {
		t.Fatalf("expected prod token via --auth-token flag to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "127.0.0.1:7071",
		runtimeProfile: daemonRuntimeProfileLocal,
		authToken:      defaultDaemonAuthToken,
	}); err == nil {
		t.Fatalf("expected local empty token to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "127.0.0.1:7071",
		runtimeProfile: daemonRuntimeProfileLocal,
		authToken:      "short-local-token",
	}); err != nil {
		t.Fatalf("expected loopback local bind with short token to pass: %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:             "tcp",
		listenAddress:            "127.0.0.1:7071",
		runtimeProfile:           daemonRuntimeProfileLocal,
		authToken:                "dev-token",
		webSocketOriginAllowlist: "ftp://invalid.example.com",
	}); err == nil {
		t.Fatalf("expected invalid websocket origin allowlist to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:    "tcp",
		listenAddress:   "127.0.0.1:7071",
		runtimeProfile:  daemonRuntimeProfileProd,
		authToken:       "prod-token-value-0123456789012345",
		authTokenSource: daemonAuthTokenSourceFile,
	}); err == nil {
		t.Fatalf("expected prod TLS requirements to reject missing tls config")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:             "tcp",
		listenAddress:            "127.0.0.1:7071",
		runtimeProfile:           daemonRuntimeProfileProd,
		authToken:                "prod-token-value-0123456789012345",
		authTokenSource:          daemonAuthTokenSourceFile,
		authTokenScopes:          []string{"daemon:read"},
		tlsCertFile:              "/tmp/server.crt",
		tlsKeyFile:               "/tmp/server.key",
		tlsClientCAFile:          "/tmp/clients.pem",
		tlsRequireClientCert:     true,
		webSocketOriginAllowlist: "https://console.example.com",
	}); err != nil {
		t.Fatalf("expected valid prod config to pass: %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:         "tcp",
		listenAddress:        "127.0.0.1:7071",
		runtimeProfile:       daemonRuntimeProfileProd,
		authToken:            "prod-token-value-0123456789012345",
		authTokenSource:      daemonAuthTokenSourceFile,
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
		tlsClientCAFile:      "/tmp/clients.pem",
		tlsRequireClientCert: true,
	}); err == nil || !strings.Contains(err.Error(), "--auth-token-scopes") {
		t.Fatalf("expected missing prod auth scopes to be rejected with remediation, got %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "0.0.0.0:7071",
		runtimeProfile: daemonRuntimeProfileLocal,
		authToken:      "dev-token",
		allowNonLocal:  true,
	}); err == nil {
		t.Fatalf("expected non-local daemon bind without tls to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "0.0.0.0:7071",
		runtimeProfile: daemonRuntimeProfileLocal,
		authToken:      "dev-token",
		allowNonLocal:  true,
		tlsCertFile:    "/tmp/server.crt",
		tlsKeyFile:     "/tmp/server.key",
	}); err == nil || !strings.Contains(err.Error(), "auth token length >= 24") {
		t.Fatalf("expected non-local daemon bind with weak token to be rejected with remediation, got %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:   "tcp",
		listenAddress:  "0.0.0.0:7071",
		runtimeProfile: daemonRuntimeProfileLocal,
		authToken:      "local-strong-token-0123456789",
		allowNonLocal:  true,
		tlsCertFile:    "/tmp/server.crt",
		tlsKeyFile:     "/tmp/server.key",
	}); err != nil {
		t.Fatalf("expected non-local daemon bind with strong token and tls to be accepted: %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:           "tcp",
		listenAddress:          "127.0.0.1:7071",
		runtimeProfile:         daemonRuntimeProfileLocal,
		authToken:              "dev-token",
		realtimeMaxConnections: -1,
	}); err == nil {
		t.Fatalf("expected negative realtime max connections to be rejected")
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:             "tcp",
		listenAddress:            "127.0.0.1:7071",
		runtimeProfile:           daemonRuntimeProfileLocal,
		authToken:                "dev-token",
		realtimeMaxSubscriptions: -1,
	}); err == nil {
		t.Fatalf("expected negative realtime max subscriptions to be rejected")
	}
}

func TestValidateDaemonRunConfigRejectsWeakTokenForNonLocalProdBind(t *testing.T) {
	err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:         "tcp",
		listenAddress:        "0.0.0.0:7071",
		allowNonLocal:        true,
		runtimeProfile:       daemonRuntimeProfileProd,
		authToken:            "prod-short-token",
		authTokenSource:      daemonAuthTokenSourceFile,
		authTokenScopes:      []string{"daemon:read"},
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
		tlsClientCAFile:      "/tmp/clients.pem",
		tlsRequireClientCert: true,
	})
	if err == nil || !strings.Contains(err.Error(), "auth token length >= 24") {
		t.Fatalf("expected weak auth token to be rejected for non-local prod bind, got %v", err)
	}
}

func TestValidateDaemonListenAddress(t *testing.T) {
	if err := validateDaemonListenAddress("tcp", "127.0.0.1:7071", false, false); err != nil {
		t.Fatalf("expected loopback tcp address to pass: %v", err)
	}

	if err := validateDaemonListenAddress("tcp", "localhost:7071", false, false); err != nil {
		t.Fatalf("expected localhost tcp address to pass: %v", err)
	}

	if err := validateDaemonListenAddress("tcp", "0.0.0.0:7071", false, false); err == nil {
		t.Fatalf("expected non-local tcp address to be rejected by default")
	}

	if err := validateDaemonListenAddress("tcp", "0.0.0.0:7071", true, false); err == nil {
		t.Fatalf("expected non-local override without tls to fail")
	}

	if err := validateDaemonListenAddress("tcp", "0.0.0.0:7071", true, true); err != nil {
		t.Fatalf("expected allow-non-local override with tls to pass: %v", err)
	}

	if err := validateDaemonListenAddress("unix", "/tmp/personal-agent-daemon.sock", false, false); err != nil {
		t.Fatalf("expected unix socket address to pass: %v", err)
	}
}

func TestValidateDaemonRunConfigWithAuthTokenFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "daemon-control.token")
	if err := os.WriteFile(tokenFile, []byte("prod-token-from-file-0123456789012345\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	token, err := controlauth.ResolveToken(defaultDaemonAuthToken, tokenFile)
	if err != nil {
		t.Fatalf("resolve token file: %v", err)
	}

	if err := validateDaemonRunConfig(daemonRunConfig{
		listenerMode:         "tcp",
		listenAddress:        "127.0.0.1:7071",
		runtimeProfile:       daemonRuntimeProfileProd,
		authToken:            token,
		authTokenSource:      daemonAuthTokenSourceFile,
		authTokenScopes:      []string{"daemon:read"},
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
		tlsClientCAFile:      "/tmp/clients.pem",
		tlsRequireClientCert: true,
	}); err != nil {
		t.Fatalf("expected production auth token file to pass: %v", err)
	}
}

func TestValidateDaemonTLSConfig(t *testing.T) {
	if err := validateDaemonTLSConfig(daemonRunConfig{
		listenerMode: "tcp",
		tlsCertFile:  "/tmp/server.crt",
	}); err == nil {
		t.Fatalf("expected cert/key pairing validation error")
	}

	if err := validateDaemonTLSConfig(daemonRunConfig{
		listenerMode:         "unix",
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
		tlsClientCAFile:      "/tmp/clients.pem",
		tlsRequireClientCert: true,
	}); err == nil {
		t.Fatalf("expected non-tcp tls usage to fail")
	}

	if err := validateDaemonTLSConfig(daemonRunConfig{
		listenerMode:         "tcp",
		tlsRequireClientCert: true,
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
	}); err == nil {
		t.Fatalf("expected mTLS client CA requirement error")
	}

	if err := validateDaemonTLSConfig(daemonRunConfig{
		listenerMode:         "tcp",
		tlsCertFile:          "/tmp/server.crt",
		tlsKeyFile:           "/tmp/server.key",
		tlsClientCAFile:      "/tmp/clients.pem",
		tlsRequireClientCert: true,
	}); err != nil {
		t.Fatalf("expected valid mTLS config: %v", err)
	}
}

func TestParseDaemonWebSocketOriginAllowlist(t *testing.T) {
	allowlist, err := parseDaemonWebSocketOriginAllowlist("https://console.example.com, http://localhost:3000, https://console.example.com/")
	if err != nil {
		t.Fatalf("parse websocket origin allowlist: %v", err)
	}
	if len(allowlist) != 2 {
		t.Fatalf("expected deduplicated websocket origin allowlist length 2, got %d (%v)", len(allowlist), allowlist)
	}
	if allowlist[0] != "http://localhost:3000" || allowlist[1] != "https://console.example.com" {
		t.Fatalf("unexpected normalized websocket origin allowlist: %v", allowlist)
	}

	if _, err := parseDaemonWebSocketOriginAllowlist("ftp://invalid.example.com"); err == nil {
		t.Fatalf("expected websocket origin parser to reject non-http(s) origins")
	}
}

func TestParseDaemonAuthTokenScopes(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		expected []string
	}{
		{name: "empty", raw: "", expected: nil},
		{name: "only separators", raw: " , ,, ", expected: nil},
		{name: "dedupe lowercase sort", raw: " Tasks:Write , chat:READ, tasks:write ", expected: []string{"chat:read", "tasks:write"}},
		{name: "wildcard all keyword", raw: "tasks:read,all,chat:write", expected: []string{"*"}},
		{name: "wildcard star", raw: "tasks:read,*", expected: []string{"*"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDaemonAuthTokenScopes(tc.raw)
			if len(got) != len(tc.expected) {
				t.Fatalf("parseDaemonAuthTokenScopes(%q) length=%d want=%d (%v)", tc.raw, len(got), len(tc.expected), got)
			}
			for index := range got {
				if got[index] != tc.expected[index] {
					t.Fatalf("parseDaemonAuthTokenScopes(%q)[%d]=%q want=%q (full=%v)", tc.raw, index, got[index], tc.expected[index], got)
				}
			}
		})
	}
}

func TestDaemonAuthScopeWarnings(t *testing.T) {
	localDefaultWarning := daemonAuthScopeWarnings(daemonRunConfig{
		runtimeProfile:  daemonRuntimeProfileLocal,
		authTokenScopes: nil,
	})
	if len(localDefaultWarning) != 1 || !strings.Contains(localDefaultWarning[0], "defaulting to wildcard '*'") {
		t.Fatalf("expected deterministic local default wildcard warning, got %v", localDefaultWarning)
	}

	prodMissingScopesWarning := daemonAuthScopeWarnings(daemonRunConfig{
		runtimeProfile:  daemonRuntimeProfileProd,
		authTokenScopes: nil,
	})
	if len(prodMissingScopesWarning) != 0 {
		t.Fatalf("expected no prod warning when scopes are missing (validation should hard-fail), got %v", prodMissingScopesWarning)
	}

	prodWildcardWarning := daemonAuthScopeWarnings(daemonRunConfig{
		runtimeProfile:  daemonRuntimeProfileProd,
		authTokenScopes: []string{"*"},
	})
	if len(prodWildcardWarning) != 1 || !strings.Contains(prodWildcardWarning[0], "full control-plane access") {
		t.Fatalf("expected deterministic prod wildcard warning, got %v", prodWildcardWarning)
	}

	scopedNoWarning := daemonAuthScopeWarnings(daemonRunConfig{
		runtimeProfile:  daemonRuntimeProfileProd,
		authTokenScopes: []string{"chat:write", "tasks:read"},
	})
	if len(scopedNoWarning) != 0 {
		t.Fatalf("expected no warning for explicit least-privilege scopes, got %v", scopedNoWarning)
	}
}
