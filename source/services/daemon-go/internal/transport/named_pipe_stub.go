//go:build !windows

package transport

import (
	"context"
	"fmt"
	"net"
)

func createNamedPipeListener(address string) (net.Listener, string, error) {
	_ = address
	return nil, "", fmt.Errorf("named-pipe listener mode is only supported on windows")
}

func dialNamedPipeContext(ctx context.Context, address string) (net.Conn, error) {
	_ = ctx
	_ = address
	return nil, fmt.Errorf("named-pipe client mode is only supported on windows")
}
