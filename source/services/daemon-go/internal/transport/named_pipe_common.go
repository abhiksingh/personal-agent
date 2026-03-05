package transport

import "strings"

func normalizeNamedPipeAddress(address string) string {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return DefaultNamedPipeAddress
	}
	if strings.HasPrefix(trimmed, `\\.\pipe\`) || strings.HasPrefix(trimmed, `\\`) {
		return trimmed
	}
	trimmed = strings.ReplaceAll(trimmed, "/", `\`)
	trimmed = strings.TrimLeft(trimmed, `\`)
	if trimmed == "" {
		return DefaultNamedPipeAddress
	}
	return `\\.\pipe\` + trimmed
}
