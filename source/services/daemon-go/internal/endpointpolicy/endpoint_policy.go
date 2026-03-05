package endpointpolicy

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
)

const (
	EnvAllowInsecureEndpoints = "PA_ALLOW_INSECURE_ENDPOINTS"
	EnvAllowPrivateEndpoints  = "PA_ALLOW_PRIVATE_ENDPOINTS"
)

type Options struct {
	Service       string
	AllowInsecure bool
	AllowPrivate  bool
}

type lookupNetIPFunc func(ctx context.Context, network string, host string) ([]netip.Addr, error)

func ParseAndValidate(raw string, options Options) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is required", endpointLabel(options.Service))
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", endpointLabel(options.Service), err)
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("%s must be an absolute URL", endpointLabel(options.Service))
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("%s scheme %q is not supported (expected http or https)", endpointLabel(options.Service), scheme)
	}

	host, loopback, private := hostSecurityFlags(parsed.Hostname())
	if host == "" {
		return nil, fmt.Errorf("%s host is required", endpointLabel(options.Service))
	}

	allowInsecure := options.AllowInsecure || allowInsecureEndpointsFromEnv()
	if scheme == "http" && !loopback && !allowInsecure {
		return nil, fmt.Errorf(
			"%s must use https for non-loopback hosts (set %s=1 to allow insecure endpoints)",
			endpointLabel(options.Service),
			EnvAllowInsecureEndpoints,
		)
	}

	allowPrivate := options.AllowPrivate || allowPrivateEndpointsFromEnv()
	if private && !loopback && !allowPrivate {
		return nil, fmt.Errorf(
			"%s host %q is private or link-local (set %s=1 to allow private endpoints)",
			endpointLabel(options.Service),
			host,
			EnvAllowPrivateEndpoints,
		)
	}

	return parsed, nil
}

func ParseAndValidateResolved(ctx context.Context, raw string, options Options) (*url.URL, error) {
	parsed, err := ParseAndValidate(raw, options)
	if err != nil {
		return nil, err
	}
	if err := validateResolvedURLWithLookup(ctx, parsed, options, defaultLookupNetIP); err != nil {
		return nil, err
	}
	return parsed, nil
}

func ValidateResolvedURL(ctx context.Context, target *url.URL, options Options) error {
	return validateResolvedURLWithLookup(ctx, target, options, defaultLookupNetIP)
}

func validateResolvedURLWithLookup(ctx context.Context, target *url.URL, options Options, lookup lookupNetIPFunc) error {
	if target == nil {
		return fmt.Errorf("%s is required", endpointLabel(options.Service))
	}
	if lookup == nil {
		return fmt.Errorf("resolve %s: resolver is required", endpointLabel(options.Service))
	}

	host := strings.ToLower(strings.TrimSpace(target.Hostname()))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return fmt.Errorf("%s host is required", endpointLabel(options.Service))
	}

	_, loopback, _ := hostSecurityFlags(host)
	if loopback {
		return nil
	}
	if literalIP, hasLiteralIP := parseHostLiteralIP(host); hasLiteralIP {
		allowPrivate := options.AllowPrivate || allowPrivateEndpointsFromEnv()
		if isSensitiveResolvedAddr(literalIP) && !allowPrivate {
			return fmt.Errorf(
				"%s host %q resolves to private or link-local address %q (set %s=1 to allow private endpoints)",
				endpointLabel(options.Service),
				host,
				literalIP.String(),
				EnvAllowPrivateEndpoints,
			)
		}
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	addrs, err := lookup(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve %s host %q: %w", endpointLabel(options.Service), host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve %s host %q: no addresses returned", endpointLabel(options.Service), host)
	}

	allowPrivate := options.AllowPrivate || allowPrivateEndpointsFromEnv()
	if allowPrivate {
		return nil
	}
	for _, candidate := range addrs {
		addr := candidate.Unmap()
		if !isSensitiveResolvedAddr(addr) {
			continue
		}
		return fmt.Errorf(
			"%s host %q resolves to private or link-local address %q (set %s=1 to allow private endpoints)",
			endpointLabel(options.Service),
			host,
			addr.String(),
			EnvAllowPrivateEndpoints,
		)
	}
	return nil
}

func defaultLookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, network, host)
}

func parseHostLiteralIP(host string) (netip.Addr, bool) {
	ipCandidate := strings.TrimSpace(host)
	if base, _, found := strings.Cut(ipCandidate, "%"); found {
		ipCandidate = base
	}
	addr, err := netip.ParseAddr(ipCandidate)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func isSensitiveResolvedAddr(addr netip.Addr) bool {
	return addr.IsPrivate() ||
		addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}

func endpointLabel(service string) string {
	trimmed := strings.TrimSpace(service)
	if trimmed == "" {
		return "endpoint"
	}
	return trimmed
}

func hostSecurityFlags(rawHost string) (host string, loopback bool, private bool) {
	trimmed := strings.TrimSpace(rawHost)
	trimmed = strings.TrimSuffix(trimmed, ".")
	trimmed = strings.ToLower(trimmed)
	if trimmed == "" {
		return "", false, false
	}

	if trimmed == "localhost" || strings.HasSuffix(trimmed, ".localhost") {
		return trimmed, true, true
	}

	ipCandidate := trimmed
	if base, _, found := strings.Cut(trimmed, "%"); found {
		ipCandidate = base
	}

	parsedIP, err := netip.ParseAddr(ipCandidate)
	if err != nil {
		return trimmed, false, false
	}

	loopback = parsedIP.IsLoopback()
	private = parsedIP.IsPrivate() ||
		loopback ||
		parsedIP.IsLinkLocalUnicast() ||
		parsedIP.IsLinkLocalMulticast() ||
		parsedIP.IsMulticast() ||
		parsedIP.IsUnspecified()
	return trimmed, loopback, private
}

func allowInsecureEndpointsFromEnv() bool {
	return isTruthyEnv(EnvAllowInsecureEndpoints)
}

func allowPrivateEndpointsFromEnv() bool {
	return isTruthyEnv(EnvAllowPrivateEndpoints)
}

func isTruthyEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(strings.TrimSpace(key)))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
