//go:build windows

package transport

import (
	"context"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

func createNamedPipeListener(address string) (net.Listener, string, error) {
	normalized := normalizeNamedPipeAddress(address)
	listener, err := winio.ListenPipe(normalized, nil)
	if err != nil {
		return nil, "", fmt.Errorf("listen named pipe %s: %w", normalized, err)
	}
	return listener, "", nil
}

func dialNamedPipeContext(ctx context.Context, address string) (net.Conn, error) {
	normalized := normalizeNamedPipeAddress(address)
	conn, err := winio.DialPipeContext(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("dial named pipe %s: %w", normalized, err)
	}
	return conn, nil
}
