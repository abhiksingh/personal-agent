package daemonruntime

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"personalagent/runtime/internal/runtimepaths"
)

func (r *InboundWatcherRuntime) currentWorkspaceID(ctx context.Context) string {
	if r == nil {
		return normalizeWorkspaceID("")
	}
	if r.workspacePinned || r.resolveWorkspaceID == nil {
		return normalizeWorkspaceID(r.workspaceID)
	}

	if resolved, err := r.resolveWorkspaceID(ctx); err == nil {
		normalized := normalizeWorkspaceID(resolved)
		if strings.TrimSpace(normalized) != "" {
			r.workspaceID = normalized
			return normalized
		}
	}
	return normalizeWorkspaceID(r.workspaceID)
}

func resolveInboundWatcherWorkspaceID(override string) (string, bool) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return normalizeWorkspaceID(trimmed), true
	}
	if trimmed := strings.TrimSpace(os.Getenv(envInboundWatcherWorkspaceID)); trimmed != "" {
		return normalizeWorkspaceID(trimmed), true
	}
	return defaultInboundWatcherWorkspaceID, false
}

func resolveInboundWatcherInboxDir(override string) string {
	raw := strings.TrimSpace(override)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(envInboundWatcherInboxDir))
	}
	if raw == "" {
		if defaultInboundDir, err := runtimepaths.DefaultInboundDir(); err == nil && strings.TrimSpace(defaultInboundDir) != "" {
			raw = defaultInboundDir
		}
	}
	if raw == "" {
		raw = filepath.Join(os.TempDir(), "personal-agent", "inbound")
	}
	if absolute, err := filepath.Abs(raw); err == nil {
		return absolute
	}
	return raw
}

func resolveInboundWatcherMessagesSourceScope(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(os.Getenv(envInboundWatcherMessagesSourceScope))
}

func resolveInboundWatcherMessagesSourceDBPath(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(os.Getenv(envInboundWatcherMessagesSourceDBPath))
}

func resolveInboundWatcherPollInterval(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	raw := strings.TrimSpace(os.Getenv(envInboundWatcherPollInterval))
	if raw == "" {
		return defaultInboundWatcherPollInterval
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultInboundWatcherPollInterval
	}
	return parsed
}

func resolveInboundWatcherMessagesLimit(override int) int {
	value := override
	if value <= 0 {
		if parsed := strings.TrimSpace(os.Getenv(envInboundWatcherMessagesLimit)); parsed != "" {
			if resolved, err := strconv.Atoi(parsed); err == nil {
				value = resolved
			}
		}
	}
	if value <= 0 {
		value = defaultInboundWatcherMessagesLimit
	}
	if value > maxInboundWatcherMessagesLimit {
		return maxInboundWatcherMessagesLimit
	}
	return value
}

func resolveInboundWatcherFileBatchSize(override int) int {
	value := override
	if value <= 0 {
		if parsed := strings.TrimSpace(os.Getenv(envInboundWatcherFileBatchSize)); parsed != "" {
			if resolved, err := strconv.Atoi(parsed); err == nil {
				value = resolved
			}
		}
	}
	if value <= 0 {
		value = defaultInboundWatcherFileBatchSize
	}
	if value > maxInboundWatcherFileBatchSize {
		return maxInboundWatcherFileBatchSize
	}
	return value
}
