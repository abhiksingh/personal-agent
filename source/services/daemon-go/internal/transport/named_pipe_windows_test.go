//go:build windows

package transport

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestTransportServerAndClientOverNamedPipe(t *testing.T) {
	pipeAddress := fmt.Sprintf(`\\.\pipe\personal-agent-%d`, time.Now().UTC().UnixNano())

	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      pipeAddress,
		AuthToken:    "pipe-token",
	})

	if !strings.Contains(strings.ToLower(server.Address()), strings.ToLower("personal-agent-")) {
		t.Fatalf("expected named pipe server address, got %q", server.Address())
	}

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeNamedPipe,
		Address:      pipeAddress,
		AuthToken:    "pipe-token",
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("create named pipe client: %v", err)
	}

	smoke, err := client.CapabilitySmoke(context.Background(), "corr-pipe")
	if err != nil {
		t.Fatalf("capability smoke over named pipe: %v", err)
	}
	if !smoke.Healthy {
		t.Fatalf("expected healthy capability response over named pipe")
	}
}
