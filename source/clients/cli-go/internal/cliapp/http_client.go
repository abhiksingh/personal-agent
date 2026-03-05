package cliapp

import (
	"context"
	"net/http"
	"time"
)

const defaultCLIHTTPTimeout = 10 * time.Second

func newCLIHTTPClient(timeout time.Duration) *http.Client {
	resolvedTimeout := timeout
	if resolvedTimeout <= 0 {
		resolvedTimeout = defaultCLIHTTPTimeout
	}
	return &http.Client{Timeout: resolvedTimeout}
}

func newCLIHTTPClientFromContext(ctx context.Context) *http.Client {
	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			if remaining := time.Until(deadline); remaining > 0 {
				return newCLIHTTPClient(remaining)
			}
		}
	}
	return newCLIHTTPClient(0)
}
