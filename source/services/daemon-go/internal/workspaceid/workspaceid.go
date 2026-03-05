package workspaceid

import "strings"

const (
	CanonicalDefault = "ws1"
)

func Normalize(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return CanonicalDefault
	}
	return trimmed
}
