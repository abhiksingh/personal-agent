package transport

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestNewServerAppliesHTTPHardeningDefaults(t *testing.T) {
	server, err := NewServer(ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "defaults-token",
	}, NewInMemoryControlBackend(NewEventBroker()), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	if server.httpServer.ReadHeaderTimeout != defaultServerReadHeaderTimeout {
		t.Fatalf("expected default read header timeout %s, got %s", defaultServerReadHeaderTimeout, server.httpServer.ReadHeaderTimeout)
	}
	if server.httpServer.ReadTimeout != defaultServerReadTimeout {
		t.Fatalf("expected default read timeout %s, got %s", defaultServerReadTimeout, server.httpServer.ReadTimeout)
	}
	if server.httpServer.WriteTimeout != defaultServerWriteTimeout {
		t.Fatalf("expected default write timeout %s, got %s", defaultServerWriteTimeout, server.httpServer.WriteTimeout)
	}
	if server.httpServer.IdleTimeout != defaultServerIdleTimeout {
		t.Fatalf("expected default idle timeout %s, got %s", defaultServerIdleTimeout, server.httpServer.IdleTimeout)
	}
	if server.httpServer.MaxHeaderBytes != defaultServerMaxHeaderBytes {
		t.Fatalf("expected default max header bytes %d, got %d", defaultServerMaxHeaderBytes, server.httpServer.MaxHeaderBytes)
	}
	if server.config.RequestBodyBytesLimit != defaultRequestBodyBytesLimit {
		t.Fatalf("expected default request body bytes limit %d, got %d", defaultRequestBodyBytesLimit, server.config.RequestBodyBytesLimit)
	}
	if server.config.ControlRateLimitWindow != defaultControlRateLimitWindow {
		t.Fatalf("expected default control rate limit window %s, got %s", defaultControlRateLimitWindow, server.config.ControlRateLimitWindow)
	}
	if server.config.ControlRateLimitMaxRequests != defaultControlRateLimitMaxRequests {
		t.Fatalf("expected default control rate limit max requests %d, got %d", defaultControlRateLimitMaxRequests, server.config.ControlRateLimitMaxRequests)
	}
	if server.config.RealtimeReadLimitBytes != defaultRealtimeReadLimitBytes {
		t.Fatalf("expected default realtime read limit bytes %d, got %d", defaultRealtimeReadLimitBytes, server.config.RealtimeReadLimitBytes)
	}
	if server.config.RealtimeWriteTimeout != defaultRealtimeWriteTimeout {
		t.Fatalf("expected default realtime write timeout %s, got %s", defaultRealtimeWriteTimeout, server.config.RealtimeWriteTimeout)
	}
	if server.config.RealtimePongTimeout != defaultRealtimePongTimeout {
		t.Fatalf("expected default realtime pong timeout %s, got %s", defaultRealtimePongTimeout, server.config.RealtimePongTimeout)
	}
	if server.config.RealtimePingInterval != defaultRealtimePingInterval {
		t.Fatalf("expected default realtime ping interval %s, got %s", defaultRealtimePingInterval, server.config.RealtimePingInterval)
	}
	if server.config.RealtimeMaxConnections != defaultRealtimeMaxConnections {
		t.Fatalf("expected default realtime max connections %d, got %d", defaultRealtimeMaxConnections, server.config.RealtimeMaxConnections)
	}
	if server.config.RealtimeMaxSubscriptions != defaultRealtimeMaxSubscriptions {
		t.Fatalf("expected default realtime max subscriptions %d, got %d", defaultRealtimeMaxSubscriptions, server.config.RealtimeMaxSubscriptions)
	}
}

func TestNewServerHonorsHTTPHardeningOverrides(t *testing.T) {
	server, err := NewServer(ServerConfig{
		ListenerMode:                ListenerModeTCP,
		Address:                     "127.0.0.1:0",
		AuthToken:                   "override-token",
		ReadHeaderTimeout:           2 * time.Second,
		ReadTimeout:                 3 * time.Second,
		WriteTimeout:                4 * time.Second,
		IdleTimeout:                 5 * time.Second,
		MaxHeaderBytes:              8192,
		RequestBodyBytesLimit:       16384,
		ControlRateLimitWindow:      2 * time.Second,
		ControlRateLimitMaxRequests: 17,
		RealtimeReadLimitBytes:      32768,
		RealtimeWriteTimeout:        6 * time.Second,
		RealtimePongTimeout:         14 * time.Second,
		RealtimePingInterval:        4 * time.Second,
		RealtimeMaxConnections:      9,
		RealtimeMaxSubscriptions:    7,
	}, NewInMemoryControlBackend(NewEventBroker()), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	if got := server.httpServer.ReadHeaderTimeout; got != 2*time.Second {
		t.Fatalf("expected read header timeout override 2s, got %s", got)
	}
	if got := server.httpServer.ReadTimeout; got != 3*time.Second {
		t.Fatalf("expected read timeout override 3s, got %s", got)
	}
	if got := server.httpServer.WriteTimeout; got != 4*time.Second {
		t.Fatalf("expected write timeout override 4s, got %s", got)
	}
	if got := server.httpServer.IdleTimeout; got != 5*time.Second {
		t.Fatalf("expected idle timeout override 5s, got %s", got)
	}
	if got := server.httpServer.MaxHeaderBytes; got != 8192 {
		t.Fatalf("expected max header bytes override 8192, got %d", got)
	}
	if got := server.config.RequestBodyBytesLimit; got != 16384 {
		t.Fatalf("expected request body bytes limit override 16384, got %d", got)
	}
	if got := server.config.ControlRateLimitWindow; got != 2*time.Second {
		t.Fatalf("expected control rate limit window override 2s, got %s", got)
	}
	if got := server.config.ControlRateLimitMaxRequests; got != 17 {
		t.Fatalf("expected control rate limit max requests override 17, got %d", got)
	}
	if got := server.config.RealtimeReadLimitBytes; got != 32768 {
		t.Fatalf("expected realtime read limit override 32768, got %d", got)
	}
	if got := server.config.RealtimeWriteTimeout; got != 6*time.Second {
		t.Fatalf("expected realtime write timeout override 6s, got %s", got)
	}
	if got := server.config.RealtimePongTimeout; got != 14*time.Second {
		t.Fatalf("expected realtime pong timeout override 14s, got %s", got)
	}
	if got := server.config.RealtimePingInterval; got != 4*time.Second {
		t.Fatalf("expected realtime ping interval override 4s, got %s", got)
	}
	if got := server.config.RealtimeMaxConnections; got != 9 {
		t.Fatalf("expected realtime max connections override 9, got %d", got)
	}
	if got := server.config.RealtimeMaxSubscriptions; got != 7 {
		t.Fatalf("expected realtime max subscriptions override 7, got %d", got)
	}
}

func TestNewServerRejectsUnsupportedRuntimeProfile(t *testing.T) {
	_, err := NewServer(ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "staging",
		AuthToken:      "profile-token",
	}, NewInMemoryControlBackend(NewEventBroker()), nil)
	if err == nil {
		t.Fatalf("expected unsupported runtime profile to be rejected")
	}
}

func TestNewServerRejectsInvalidWebSocketOriginAllowlist(t *testing.T) {
	_, err := NewServer(ServerConfig{
		ListenerMode:             ListenerModeTCP,
		Address:                  "127.0.0.1:0",
		RuntimeProfile:           "prod",
		AuthToken:                "origin-token",
		WebSocketOriginAllowlist: []string{"ftp://invalid.example.com"},
	}, NewInMemoryControlBackend(NewEventBroker()), nil)
	if err == nil {
		t.Fatalf("expected invalid websocket origin allowlist entry to be rejected")
	}
}

func TestNormalizeWebSocketOriginAllowlistCanonicalizesAndDeduplicates(t *testing.T) {
	origins, err := NormalizeWebSocketOriginAllowlist([]string{
		"https://console.example.com",
		"https://console.example.com/",
		"HTTP://LOCALHOST:80",
		"  ",
	})
	if err != nil {
		t.Fatalf("normalize websocket origin allowlist: %v", err)
	}
	expected := []string{
		"http://localhost",
		"https://console.example.com",
	}
	if !reflect.DeepEqual(origins, expected) {
		t.Fatalf("unexpected normalized websocket origin allowlist: got %v want %v", origins, expected)
	}
}

func TestCreateListenerUnixEnforcesPrivateFilesystemPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on windows")
	}

	root := filepath.Join(os.TempDir(), fmt.Sprintf("pa-sock-%d", time.Now().UTC().UnixNano()))
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create unix listener test root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	parentDir := filepath.Join(root, "s")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatalf("seed unix listener parent dir: %v", err)
	}
	if err := os.Chmod(parentDir, 0o755); err != nil {
		t.Fatalf("set seed parent permissions: %v", err)
	}

	socketPath := filepath.Join(parentDir, "pa.sock")
	listener, unixPath, err := createListener(ServerConfig{
		ListenerMode: ListenerModeUnix,
		Address:      socketPath,
	})
	if err != nil {
		t.Fatalf("create unix listener: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	})

	if unixPath != socketPath {
		t.Fatalf("expected unix listener path %q, got %q", socketPath, unixPath)
	}

	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("stat unix listener parent dir: %v", err)
	}
	if got := parentInfo.Mode().Perm(); got != unixSocketParentDirMode {
		t.Fatalf("expected unix listener parent dir mode %o, got %o", unixSocketParentDirMode, got)
	}

	socketInfo, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat unix socket path: %v", err)
	}
	if got := socketInfo.Mode().Perm(); got != unixSocketFileMode {
		t.Fatalf("expected unix socket mode %o, got %o", unixSocketFileMode, got)
	}
}

func TestShouldSkipUnixSocketPermissionTighteningSharedRootsOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on windows")
	}

	sharedCandidates := []string{
		os.TempDir(),
		"/tmp",
		"/private/tmp",
	}
	for _, candidate := range sharedCandidates {
		if candidate == "" {
			continue
		}
		if !shouldSkipUnixSocketPermissionTightening(candidate) {
			t.Fatalf("expected shared root %q to skip permission tightening", candidate)
		}
	}

	privateRoot := filepath.Join(t.TempDir(), "private-socket-root")
	if err := os.MkdirAll(privateRoot, 0o755); err != nil {
		t.Fatalf("create private socket root: %v", err)
	}
	if shouldSkipUnixSocketPermissionTightening(privateRoot) {
		t.Fatalf("expected private socket root %q not to skip permission tightening", privateRoot)
	}
}
