package transport

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	transportRuntimeProfileLocal = "local"
	transportRuntimeProfileProd  = "prod"
)

type realtimeOriginPolicy struct {
	runtimeProfile string
	allowlist      map[string]struct{}
}

func newRealtimeOriginPolicy(runtimeProfile string, rawAllowlist []string) (realtimeOriginPolicy, error) {
	normalizedProfile, err := normalizeTransportRuntimeProfile(runtimeProfile)
	if err != nil {
		return realtimeOriginPolicy{}, err
	}

	normalizedAllowlist, err := NormalizeWebSocketOriginAllowlist(rawAllowlist)
	if err != nil {
		return realtimeOriginPolicy{}, err
	}

	allowlist := make(map[string]struct{}, len(normalizedAllowlist))
	for _, origin := range normalizedAllowlist {
		allowlist[origin] = struct{}{}
	}
	return realtimeOriginPolicy{
		runtimeProfile: normalizedProfile,
		allowlist:      allowlist,
	}, nil
}

func normalizeTransportRuntimeProfile(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return transportRuntimeProfileLocal, nil
	}
	switch normalized {
	case transportRuntimeProfileLocal:
		return transportRuntimeProfileLocal, nil
	case transportRuntimeProfileProd:
		return transportRuntimeProfileProd, nil
	default:
		return "", fmt.Errorf("unsupported runtime profile %q", raw)
	}
}

func NormalizeWebSocketOriginAllowlist(rawAllowlist []string) ([]string, error) {
	if len(rawAllowlist) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(rawAllowlist))
	normalized := make([]string, 0, len(rawAllowlist))
	for _, rawOrigin := range rawAllowlist {
		trimmedOrigin := strings.TrimSpace(rawOrigin)
		if trimmedOrigin == "" {
			continue
		}
		canonicalOrigin, _, err := canonicalizeWebSocketOrigin(trimmedOrigin)
		if err != nil {
			return nil, fmt.Errorf("invalid websocket origin %q: %w", rawOrigin, err)
		}
		if _, exists := seen[canonicalOrigin]; exists {
			continue
		}
		seen[canonicalOrigin] = struct{}{}
		normalized = append(normalized, canonicalOrigin)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func (p realtimeOriginPolicy) allowsRequest(request *http.Request) bool {
	if request == nil {
		return false
	}

	originHeader := strings.TrimSpace(request.Header.Get("Origin"))
	if originHeader == "" {
		return true
	}

	canonicalOrigin, parsedOrigin, err := canonicalizeWebSocketOrigin(originHeader)
	if err != nil {
		return false
	}

	if len(p.allowlist) > 0 {
		_, allowed := p.allowlist[canonicalOrigin]
		return allowed
	}
	if p.runtimeProfile == transportRuntimeProfileLocal {
		return isLoopbackOrigin(parsedOrigin)
	}
	return false
}

func canonicalizeWebSocketOrigin(raw string) (string, *url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, fmt.Errorf("origin is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", nil, fmt.Errorf("parse origin: %w", err)
	}
	if parsed.User != nil {
		return "", nil, fmt.Errorf("origin userinfo is not allowed")
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", nil, fmt.Errorf("origin scheme must be http or https")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", nil, fmt.Errorf("origin query/fragment is not allowed")
	}

	path := strings.TrimSpace(parsed.EscapedPath())
	if path != "" && path != "/" {
		return "", nil, fmt.Errorf("origin path is not allowed")
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", nil, fmt.Errorf("origin host is required")
	}
	lowerHost := strings.ToLower(host)

	portValue := 0
	portText := strings.TrimSpace(parsed.Port())
	if portText != "" {
		value, err := strconv.Atoi(portText)
		if err != nil || value <= 0 || value > 65535 {
			return "", nil, fmt.Errorf("origin port must be between 1 and 65535")
		}
		switch {
		case scheme == "http" && value == 80:
			portText = ""
		case scheme == "https" && value == 443:
			portText = ""
		default:
			portValue = value
		}
	}
	if portText == "" {
		portValue = 0
	}

	canonicalHost := lowerHost
	if strings.Contains(lowerHost, ":") {
		canonicalHost = "[" + lowerHost + "]"
	}
	if portValue > 0 {
		canonicalHost = net.JoinHostPort(lowerHost, strconv.Itoa(portValue))
	}

	return scheme + "://" + canonicalHost, parsed, nil
}

func isLoopbackOrigin(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	host := strings.TrimSpace(parsed.Hostname())
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
