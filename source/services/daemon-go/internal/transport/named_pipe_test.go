package transport

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNormalizeNamedPipeAddress(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "empty", input: "", expect: DefaultNamedPipeAddress},
		{name: "simple name", input: "personal-agent-custom", expect: `\\.\pipe\personal-agent-custom`},
		{name: "slash path", input: "folder/name", expect: `\\.\pipe\folder\name`},
		{name: "already canonical", input: `\\.\pipe\existing`, expect: `\\.\pipe\existing`},
		{name: "already unc", input: `\\server\pipe\name`, expect: `\\server\pipe\name`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := normalizeNamedPipeAddress(testCase.input); got != testCase.expect {
				t.Fatalf("normalizeNamedPipeAddress(%q)=%q, expected %q", testCase.input, got, testCase.expect)
			}
		})
	}
}

func TestNamedPipeModeDefaultsFromTCPAddressFallback(t *testing.T) {
	broker := NewEventBroker()
	backend := NewInMemoryControlBackend(broker)

	server, err := NewServer(ServerConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      DefaultTCPAddress,
		AuthToken:    "named-pipe-token",
	}, backend, broker)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if server.config.Address != DefaultNamedPipeAddress {
		t.Fatalf("expected named pipe default address %q, got %q", DefaultNamedPipeAddress, server.config.Address)
	}

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      DefaultTCPAddress,
		AuthToken:    "named-pipe-token",
		Timeout:      150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client.baseURL != "http://namedpipe" {
		t.Fatalf("expected named pipe base URL, got %q", client.baseURL)
	}
	if client.wsURL != "ws://namedpipe/v1/realtime/ws" {
		t.Fatalf("expected named pipe websocket URL, got %q", client.wsURL)
	}
}

func TestNamedPipeModeReturnsUnsupportedErrorsOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unsupported-path assertions are for non-Windows only")
	}

	broker := NewEventBroker()
	backend := NewInMemoryControlBackend(broker)
	server, err := NewServer(ServerConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      DefaultNamedPipeAddress,
		AuthToken:    "named-pipe-token",
	}, backend, broker)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	if err := server.Start(); err == nil {
		t.Fatalf("expected named pipe listener start error on non-Windows")
	} else if !strings.Contains(err.Error(), "only supported on windows") {
		t.Fatalf("expected unsupported named pipe listener error, got %v", err)
	}

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      DefaultNamedPipeAddress,
		AuthToken:    "named-pipe-token",
		Timeout:      150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.CapabilitySmoke(context.Background(), "corr-named-pipe")
	if err == nil {
		t.Fatalf("expected named pipe client request error on non-Windows")
	}
	if !strings.Contains(err.Error(), "only supported on windows") {
		t.Fatalf("expected unsupported named pipe client error, got %v", err)
	}
}
