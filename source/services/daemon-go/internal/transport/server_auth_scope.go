package transport

import (
	"net/http"
	"sort"
	"strings"
)

type routeScopePolicyRule struct {
	method   string
	path     string
	prefix   bool
	required []string
}

var controlRouteScopePolicy = []routeScopePolicyRule{
	// Metadata + realtime control surfaces.
	{method: http.MethodGet, path: "/v1/meta/capabilities", required: []string{"metadata:read"}},
	{method: http.MethodGet, path: "/v1/capabilities/smoke", required: []string{"metadata:read"}},
	{method: http.MethodGet, path: "/v1/realtime/ws", required: []string{"realtime:read"}},

	// Daemon lifecycle control-plane.
	{method: http.MethodGet, path: "/v1/daemon/lifecycle/status", required: []string{"daemon:read"}},
	{method: http.MethodPost, path: "/v1/daemon/lifecycle/plugins/history", required: []string{"daemon:read"}},
	{method: http.MethodPost, path: "/v1/daemon/lifecycle/control", required: []string{"daemon:write"}},

	// Task, approval, and communication timeline routes.
	{method: http.MethodPost, path: "/v1/tasks/list", required: []string{"tasks:read"}},
	{method: http.MethodGet, path: "/v1/tasks/", prefix: true, required: []string{"tasks:read"}},
	{method: http.MethodPost, path: "/v1/approvals/list", required: []string{"tasks:read"}},
	{method: http.MethodPost, path: "/v1/tasks", required: []string{"tasks:write"}},
	{method: http.MethodPost, path: "/v1/tasks/cancel", required: []string{"tasks:write"}},
	{method: http.MethodPost, path: "/v1/tasks/retry", required: []string{"tasks:write"}},
	{method: http.MethodPost, path: "/v1/tasks/requeue", required: []string{"tasks:write"}},

	{method: http.MethodPost, path: "/v1/comm/threads/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/events/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/call-sessions/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/attempts", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/webhook-receipts/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/ingest-receipts/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/policy/list", required: []string{"comm:read"}},
	{method: http.MethodPost, path: "/v1/comm/send", required: []string{"comm:write"}},
	{method: http.MethodPost, path: "/v1/comm/policy/set", required: []string{"comm:write"}},
	{method: http.MethodPost, path: "/v1/comm/messages/ingest", required: []string{"comm:write"}},
	{method: http.MethodPost, path: "/v1/comm/mail/ingest", required: []string{"comm:write"}},
	{method: http.MethodPost, path: "/v1/comm/calendar/ingest", required: []string{"comm:write"}},
	{method: http.MethodPost, path: "/v1/comm/browser/ingest", required: []string{"comm:write"}},

	// Chat + agent orchestration routes.
	{method: http.MethodPost, path: "/v1/chat/history", required: []string{"chat:read"}},
	{method: http.MethodPost, path: "/v1/chat/turn/explain", required: []string{"chat:read"}},
	{method: http.MethodPost, path: "/v1/chat/persona/get", required: []string{"chat:read"}},
	{method: http.MethodPost, path: "/v1/chat/turn", required: []string{"chat:write"}},
	{method: http.MethodPost, path: "/v1/chat/persona/set", required: []string{"chat:write"}},
	{method: http.MethodPost, path: "/v1/agent/run", required: []string{"agent:write"}},
	{method: http.MethodPost, path: "/v1/agent/approve", required: []string{"agent:write"}},

	// Delegation + identity administration.
	{method: http.MethodPost, path: "/v1/delegation/list", required: []string{"delegation:read"}},
	{method: http.MethodPost, path: "/v1/delegation/check", required: []string{"delegation:read"}},
	{method: http.MethodPost, path: "/v1/delegation/capability-grants/list", required: []string{"delegation:read"}},
	{method: http.MethodPost, path: "/v1/delegation/grant", required: []string{"delegation:write"}},
	{method: http.MethodPost, path: "/v1/delegation/revoke", required: []string{"delegation:write"}},
	{method: http.MethodPost, path: "/v1/delegation/capability-grants/upsert", required: []string{"delegation:write"}},

	{method: http.MethodPost, path: "/v1/identity/workspaces", required: []string{"identity:read"}},
	{method: http.MethodPost, path: "/v1/identity/principals", required: []string{"identity:read"}},
	{method: http.MethodPost, path: "/v1/identity/context", required: []string{"identity:read"}},
	{method: http.MethodPost, path: "/v1/identity/devices/list", required: []string{"identity:read"}},
	{method: http.MethodPost, path: "/v1/identity/sessions/list", required: []string{"identity:read"}},
	{method: http.MethodPost, path: "/v1/identity/context/select-workspace", required: []string{"identity:write"}},
	{method: http.MethodPost, path: "/v1/identity/bootstrap", required: []string{"identity:write"}},
	{method: http.MethodPost, path: "/v1/identity/sessions/revoke", required: []string{"identity:write"}},

	// Secrets, providers, and model controls.
	{method: http.MethodGet, path: "/v1/secrets/refs/", prefix: true, required: []string{"secrets:read"}},
	{method: http.MethodPost, path: "/v1/secrets/refs", required: []string{"secrets:write"}},
	{method: http.MethodDelete, path: "/v1/secrets/refs/", prefix: true, required: []string{"secrets:write"}},

	{method: http.MethodPost, path: "/v1/providers/list", required: []string{"providers:read"}},
	{method: http.MethodPost, path: "/v1/providers/check", required: []string{"providers:read"}},
	{method: http.MethodPost, path: "/v1/providers/set", required: []string{"providers:write"}},

	{method: http.MethodPost, path: "/v1/models/list", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/discover", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/policy", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/resolve", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/route/simulate", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/route/explain", required: []string{"models:read"}},
	{method: http.MethodPost, path: "/v1/models/add", required: []string{"models:write"}},
	{method: http.MethodPost, path: "/v1/models/remove", required: []string{"models:write"}},
	{method: http.MethodPost, path: "/v1/models/enable", required: []string{"models:write"}},
	{method: http.MethodPost, path: "/v1/models/disable", required: []string{"models:write"}},
	{method: http.MethodPost, path: "/v1/models/select", required: []string{"models:write"}},

	// Channel + connector administration.
	{method: http.MethodPost, path: "/v1/channels/status", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/diagnostics", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/mappings/list", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/get", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/check", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/call-status", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/transcript", required: []string{"channels:read"}},
	{method: http.MethodPost, path: "/v1/channels/config/upsert", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/test", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/mappings/upsert", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/set", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/sms-chat-turn", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/start-call", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/ingest-sms", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/ingest-voice", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/webhook/serve", required: []string{"channels:write"}},
	{method: http.MethodPost, path: "/v1/channels/twilio/webhook/replay", required: []string{"channels:write"}},

	{method: http.MethodPost, path: "/v1/connectors/status", required: []string{"connectors:read"}},
	{method: http.MethodPost, path: "/v1/connectors/diagnostics", required: []string{"connectors:read"}},
	{method: http.MethodPost, path: "/v1/connectors/cloudflared/version", required: []string{"connectors:read"}},
	{method: http.MethodPost, path: "/v1/connectors/config/upsert", required: []string{"connectors:write"}},
	{method: http.MethodPost, path: "/v1/connectors/test", required: []string{"connectors:write"}},
	{method: http.MethodPost, path: "/v1/connectors/permission/request", required: []string{"connectors:write"}},
	{method: http.MethodPost, path: "/v1/connectors/cloudflared/exec", required: []string{"connectors:write"}},

	// Automation, inspect, retention, and context tuning.
	{method: http.MethodPost, path: "/v1/automation/list", required: []string{"automation:read"}},
	{method: http.MethodPost, path: "/v1/automation/fire-history", required: []string{"automation:read"}},
	{method: http.MethodPost, path: "/v1/automation/comm-trigger/metadata", required: []string{"automation:read"}},
	{method: http.MethodPost, path: "/v1/automation/comm-trigger/validate", required: []string{"automation:read"}},
	{method: http.MethodPost, path: "/v1/automation/create", required: []string{"automation:write"}},
	{method: http.MethodPost, path: "/v1/automation/update", required: []string{"automation:write"}},
	{method: http.MethodPost, path: "/v1/automation/delete", required: []string{"automation:write"}},
	{method: http.MethodPost, path: "/v1/automation/run/schedule", required: []string{"automation:write"}},
	{method: http.MethodPost, path: "/v1/automation/run/comm-event", required: []string{"automation:write"}},

	{method: http.MethodPost, path: "/v1/inspect/run", required: []string{"inspect:read"}},
	{method: http.MethodPost, path: "/v1/inspect/transcript", required: []string{"inspect:read"}},
	{method: http.MethodPost, path: "/v1/inspect/memory", required: []string{"inspect:read"}},
	{method: http.MethodPost, path: "/v1/inspect/logs/query", required: []string{"inspect:read"}},
	{method: http.MethodPost, path: "/v1/inspect/logs/stream", required: []string{"inspect:read"}},

	{method: http.MethodPost, path: "/v1/retention/purge", required: []string{"retention:write"}},
	{method: http.MethodPost, path: "/v1/retention/compact-memory", required: []string{"retention:write"}},

	{method: http.MethodPost, path: "/v1/context/samples", required: []string{"context:read"}},
	{method: http.MethodPost, path: "/v1/context/memory/inventory", required: []string{"context:read"}},
	{method: http.MethodPost, path: "/v1/context/memory/compaction-candidates", required: []string{"context:read"}},
	{method: http.MethodPost, path: "/v1/context/retrieval/documents", required: []string{"context:read"}},
	{method: http.MethodPost, path: "/v1/context/retrieval/chunks", required: []string{"context:read"}},
	{method: http.MethodPost, path: "/v1/context/tune", required: []string{"context:write"}},
}

func normalizeAuthTokenScopes(raw []string) []string {
	normalized := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, scope := range raw {
		trimmed := strings.ToLower(strings.TrimSpace(scope))
		if trimmed == "" {
			continue
		}
		if trimmed == "*" || trimmed == "all" {
			return []string{"*"}
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return []string{"*"}
	}
	sort.Strings(normalized)
	return normalized
}

func scopeSetFromList(scopes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		normalized := strings.ToLower(strings.TrimSpace(scope))
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func requiredScopesForRoute(request *http.Request) []string {
	if request == nil {
		return nil
	}
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	path := strings.TrimSpace(request.URL.Path)
	for _, rule := range controlRouteScopePolicy {
		if method != strings.ToUpper(strings.TrimSpace(rule.method)) {
			continue
		}
		if rule.prefix {
			if strings.HasPrefix(path, strings.TrimSpace(rule.path)) {
				return append([]string(nil), rule.required...)
			}
			continue
		}
		if path == strings.TrimSpace(rule.path) {
			return append([]string(nil), rule.required...)
		}
	}
	return nil
}

func (s *Server) routeScopeAllowed(request *http.Request) (bool, []string) {
	requiredScopes := requiredScopesForRoute(request)
	if len(requiredScopes) == 0 {
		return true, nil
	}
	if s == nil {
		return false, requiredScopes
	}
	if len(s.authTokenScopeSet) == 0 {
		return true, requiredScopes
	}
	if _, wildcard := s.authTokenScopeSet["*"]; wildcard {
		return true, requiredScopes
	}
	for _, required := range requiredScopes {
		normalized := strings.ToLower(strings.TrimSpace(required))
		if normalized == "" {
			continue
		}
		if _, ok := s.authTokenScopeSet[normalized]; ok {
			return true, requiredScopes
		}
	}
	return false, requiredScopes
}
