package daemonruntime

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func resolveDaemonTwilioRequestURL(request *http.Request) string {
	if allowTwilioReplayRequestURLOverride(request) {
		override := strings.TrimSpace(request.Header.Get(twilioWebhookRequestURLOverrideHeader))
		if override != "" {
			return override
		}
	}
	return canonicalDaemonTwilioRequestURL(request)
}

func canonicalDaemonTwilioRequestURL(request *http.Request) string {
	if request == nil {
		return ""
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, request.Host, request.URL.RequestURI())
}

func allowTwilioReplayRequestURLOverride(request *http.Request) bool {
	if request == nil {
		return false
	}
	if !parseWebhookTruthy(request.Header.Get(twilioWebhookReplayMarkerHeader)) {
		return false
	}
	return requestOriginatesFromLoopback(request.RemoteAddr)
}

func requestOriginatesFromLoopback(remoteAddr string) bool {
	trimmed := strings.TrimSpace(remoteAddr)
	if trimmed == "" {
		return false
	}

	host := trimmed
	if parsedHost, _, err := net.SplitHostPort(trimmed); err == nil {
		host = parsedHost
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func resolveDaemonTwilioActionURL(request *http.Request) string {
	requestURL := canonicalDaemonTwilioRequestURL(request)
	if requestURL != "" {
		return requestURL
	}
	if request != nil {
		return request.URL.RequestURI()
	}
	return ""
}

func resolveDaemonTwilioSignatureRequestURL(request *http.Request) string {
	requestURL := resolveDaemonTwilioRequestURL(request)
	if requestURL != "" {
		return requestURL
	}
	if request != nil {
		return request.URL.RequestURI()
	}
	return ""
}
