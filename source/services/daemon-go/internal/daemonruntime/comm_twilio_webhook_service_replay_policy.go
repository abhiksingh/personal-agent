package daemonruntime

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

type replayLookupNetIPFunc func(ctx context.Context, network string, host string) ([]netip.Addr, error)

type twilioReplayTargetAllowlist struct {
	hosts    map[string]struct{}
	ips      map[netip.Addr]struct{}
	prefixes []netip.Prefix
}

func parseTwilioReplayTargetAllowlist(raw string) twilioReplayTargetAllowlist {
	allowlist := twilioReplayTargetAllowlist{
		hosts: map[string]struct{}{},
		ips:   map[netip.Addr]struct{}{},
	}
	for _, value := range strings.Split(strings.TrimSpace(raw), ",") {
		entry := strings.ToLower(strings.TrimSpace(value))
		entry = strings.TrimSuffix(entry, ".")
		if entry == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			allowlist.prefixes = append(allowlist.prefixes, prefix)
			continue
		}
		if addr, err := netip.ParseAddr(entry); err == nil {
			allowlist.ips[addr] = struct{}{}
			continue
		}
		allowlist.hosts[entry] = struct{}{}
	}
	return allowlist
}

func (a twilioReplayTargetAllowlist) allowsHost(host string, addr netip.Addr, hasAddr bool) bool {
	if _, ok := a.hosts[strings.ToLower(strings.TrimSpace(host))]; ok {
		return true
	}
	if !hasAddr {
		return false
	}
	if _, ok := a.ips[addr]; ok {
		return true
	}
	for _, prefix := range a.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func validateTwilioWebhookReplayTarget(ctx context.Context, rawTargetURL string, allowlistRaw string) error {
	return validateTwilioWebhookReplayTargetWithLookup(ctx, rawTargetURL, allowlistRaw, defaultReplayLookupNetIP)
}

func validateTwilioWebhookReplayTargetWithLookup(
	ctx context.Context,
	rawTargetURL string,
	allowlistRaw string,
	lookup replayLookupNetIPFunc,
) error {
	trimmedTarget := strings.TrimSpace(rawTargetURL)
	if trimmedTarget == "" {
		return fmt.Errorf("replay target url is required")
	}
	parsed, err := url.Parse(trimmedTarget)
	if err != nil {
		return fmt.Errorf("parse replay target url: %w", err)
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("replay target url must be absolute")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("replay target url scheme %q is not supported (expected http or https)", scheme)
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return fmt.Errorf("replay target host is required")
	}

	allowlist := parseTwilioReplayTargetAllowlist(allowlistRaw)
	ip, hasIP := parseReplayTargetIP(host)
	if allowlist.allowsHost(host, ip, hasIP) {
		return nil
	}

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil
	}
	if hasIP && ip.IsLoopback() {
		return nil
	}

	resolvedAddrs := make([]netip.Addr, 0, 1)
	if hasIP {
		resolvedAddrs = append(resolvedAddrs, ip.Unmap())
	} else {
		if lookup == nil {
			return fmt.Errorf("resolve replay target host %q: resolver is required", host)
		}
		if ctx == nil {
			ctx = context.Background()
		}
		addrs, err := lookup(ctx, "ip", host)
		if err != nil {
			return fmt.Errorf("resolve replay target host %q: %w", host, err)
		}
		if len(addrs) == 0 {
			return fmt.Errorf("resolve replay target host %q: no addresses returned", host)
		}
		for _, candidate := range addrs {
			resolvedAddrs = append(resolvedAddrs, candidate.Unmap())
		}
	}

	for _, candidate := range resolvedAddrs {
		if allowlist.allowsHost(host, candidate, true) {
			continue
		}
		if isReplayTargetMetadataHost(host, candidate, true) {
			return fmt.Errorf(
				"replay target host %q resolves to metadata endpoint address %q and is blocked by default (set %s to an explicit allowlist to override)",
				host,
				candidate.String(),
				twilioWebhookReplayTargetAllowlistEnv,
			)
		}
		if isReplayTargetPrivateAddress(candidate, true) {
			return fmt.Errorf(
				"replay target host %q resolves to private or link-local address %q and is blocked by default (set %s to an explicit allowlist to override)",
				host,
				candidate.String(),
				twilioWebhookReplayTargetAllowlistEnv,
			)
		}
	}
	return nil
}

func defaultReplayLookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, network, host)
}

func parseReplayTargetIP(host string) (netip.Addr, bool) {
	ipCandidate := strings.TrimSpace(host)
	if base, _, found := strings.Cut(ipCandidate, "%"); found {
		ipCandidate = base
	}
	addr, err := netip.ParseAddr(ipCandidate)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr, true
}

func isReplayTargetPrivateAddress(addr netip.Addr, hasAddr bool) bool {
	if !hasAddr {
		return false
	}
	return addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}

func isReplayTargetMetadataHost(host string, addr netip.Addr, hasAddr bool) bool {
	switch host {
	case "metadata", "metadata.google.internal", "metadata.azure.internal", "instance-data", "instance-data.ec2.internal":
		return true
	}
	if !hasAddr {
		return false
	}
	if addr == netip.MustParseAddr("169.254.169.254") {
		return true
	}
	return addr == netip.MustParseAddr("fd00:ec2::254")
}
