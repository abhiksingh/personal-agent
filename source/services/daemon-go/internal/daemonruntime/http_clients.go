package daemonruntime

import (
	"net/http"
	"time"
)

const (
	defaultProviderProbeHTTPTimeout = 4 * time.Second
	defaultProviderChatHTTPTimeout  = 5 * time.Minute
)

func newDaemonRuntimeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultProviderProbeHTTPTimeout
	}
	return &http.Client{Timeout: timeout}
}
