package transport

import "strings"

type controlRouteRateLimitPolicyRule struct {
	method      string
	path        string
	prefix      bool
	endpointKey string
}

type controlRouteRateLimitAllowlistRule struct {
	method string
	path   string
	prefix bool
	reason string
}

type controlRouteRateLimitPolicyDecision struct {
	enforced    bool
	allowlisted bool
	endpointKey string
	reason      string
}

var controlRouteRateLimitPolicyRules = []controlRouteRateLimitPolicyRule{
	{method: "POST", path: "/v1/chat/turn", endpointKey: controlRateLimitKeyChatTurn},
	{method: "POST", path: "/v1/chat/persona/set", endpointKey: controlRateLimitKeyChatPersonaSet},
	{method: "POST", path: "/v1/agent/run", endpointKey: controlRateLimitKeyAgentRun},
	{method: "POST", path: "/v1/agent/approve", endpointKey: controlRateLimitKeyAgentApprove},
	{method: "POST", path: "/v1/tasks", endpointKey: controlRateLimitKeyTaskSubmit},
	{method: "POST", path: "/v1/tasks/cancel", endpointKey: controlRateLimitKeyTaskCancel},
	{method: "POST", path: "/v1/tasks/retry", endpointKey: controlRateLimitKeyTaskRetry},
	{method: "POST", path: "/v1/tasks/requeue", endpointKey: controlRateLimitKeyTaskRequeue},
	{method: "POST", path: "/v1/daemon/lifecycle/control", endpointKey: controlRateLimitKeyDaemonLifecycleControl},
	{method: "POST", path: "/v1/automation/create", endpointKey: controlRateLimitKeyAutomationCreate},
	{method: "POST", path: "/v1/automation/update", endpointKey: controlRateLimitKeyAutomationUpdate},
	{method: "POST", path: "/v1/automation/delete", endpointKey: controlRateLimitKeyAutomationDelete},
	{method: "POST", path: "/v1/automation/run/schedule", endpointKey: controlRateLimitKeyAutomationRunSchedule},
	{method: "POST", path: "/v1/automation/run/comm-event", endpointKey: controlRateLimitKeyAutomationRunCommEvent},
}

var controlRouteRateLimitPolicyAllowlist = []controlRouteRateLimitAllowlistRule{
	{method: "POST", path: "/v1/delegation/grant", reason: "delegation management routes are currently operator-mediated and lower traffic"},
	{method: "POST", path: "/v1/delegation/revoke", reason: "delegation management routes are currently operator-mediated and lower traffic"},
	{method: "POST", path: "/v1/delegation/capability-grants/upsert", reason: "delegation policy changes are low-frequency control operations"},
	{method: "POST", path: "/v1/identity/context/select-workspace", reason: "identity context selection is user-session scoped and low volume"},
	{method: "POST", path: "/v1/identity/bootstrap", reason: "identity bootstrap is setup-oriented and low frequency"},
	{method: "POST", path: "/v1/identity/sessions/revoke", reason: "session revoke is a deliberate operator action"},
	{method: "POST", path: "/v1/secrets/refs", reason: "secret metadata writes are low-frequency config operations"},
	{method: "DELETE", path: "/v1/secrets/refs/", prefix: true, reason: "secret metadata deletes are low-frequency config operations"},
	{method: "POST", path: "/v1/providers/set", reason: "provider config updates are low-frequency administrative operations"},
	{method: "POST", path: "/v1/models/add", reason: "model catalog administration is low-frequency"},
	{method: "POST", path: "/v1/models/remove", reason: "model catalog administration is low-frequency"},
	{method: "POST", path: "/v1/models/enable", reason: "model catalog administration is low-frequency"},
	{method: "POST", path: "/v1/models/disable", reason: "model catalog administration is low-frequency"},
	{method: "POST", path: "/v1/models/select", reason: "model route selection is explicit user control"},
	{method: "POST", path: "/v1/channels/config/upsert", reason: "channel config updates are setup/admin flows"},
	{method: "POST", path: "/v1/channels/test", reason: "channel tests are explicit operator diagnostics"},
	{method: "POST", path: "/v1/channels/mappings/upsert", reason: "channel mapping updates are low-frequency setup operations"},
	{method: "POST", path: "/v1/channels/twilio/set", reason: "twilio config updates are setup/admin flows"},
	{method: "POST", path: "/v1/channels/twilio/sms-chat-turn", reason: "direct twilio sms chat-turn path has downstream safeguards and low direct-call volume"},
	{method: "POST", path: "/v1/channels/twilio/start-call", reason: "voice-call initiation has downstream provider limits"},
	{method: "POST", path: "/v1/channels/twilio/ingest-sms", reason: "ingest paths are source-governed by webhook event volume"},
	{method: "POST", path: "/v1/channels/twilio/ingest-voice", reason: "ingest paths are source-governed by webhook event volume"},
	{method: "POST", path: "/v1/channels/twilio/webhook/serve", reason: "webhook serve lifecycle operation is operator-triggered"},
	{method: "POST", path: "/v1/channels/twilio/webhook/replay", reason: "webhook replay is explicit diagnostic operation"},
	{method: "POST", path: "/v1/connectors/config/upsert", reason: "connector config updates are setup/admin flows"},
	{method: "POST", path: "/v1/connectors/test", reason: "connector tests are explicit operator diagnostics"},
	{method: "POST", path: "/v1/connectors/permission/request", reason: "permission requests are user-mediated control actions"},
	{method: "POST", path: "/v1/connectors/cloudflared/exec", reason: "cloudflared exec is explicit operator command flow"},
	{method: "POST", path: "/v1/comm/send", reason: "comm send path is governed by downstream channel/provider throttles"},
	{method: "POST", path: "/v1/comm/policy/set", reason: "comm policy updates are low-frequency admin operations"},
	{method: "POST", path: "/v1/comm/messages/ingest", reason: "ingest paths are source-governed by upstream event volume"},
	{method: "POST", path: "/v1/comm/mail/ingest", reason: "ingest paths are source-governed by upstream event volume"},
	{method: "POST", path: "/v1/comm/calendar/ingest", reason: "ingest paths are source-governed by upstream event volume"},
	{method: "POST", path: "/v1/comm/browser/ingest", reason: "ingest paths are source-governed by upstream event volume"},
	{method: "POST", path: "/v1/retention/purge", reason: "retention operations are explicit operator maintenance actions"},
	{method: "POST", path: "/v1/retention/compact-memory", reason: "retention operations are explicit operator maintenance actions"},
	{method: "POST", path: "/v1/context/tune", reason: "context policy tuning is explicit low-frequency operator control"},
}

func controlRouteRateLimitPolicyDecisionForRoute(method string, path string) controlRouteRateLimitPolicyDecision {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	normalizedPath := strings.TrimSpace(path)
	for _, rule := range controlRouteRateLimitPolicyRules {
		if !controlRouteRateLimitRouteMatch(normalizedMethod, normalizedPath, rule.method, rule.path, rule.prefix) {
			continue
		}
		return controlRouteRateLimitPolicyDecision{
			enforced:    true,
			endpointKey: strings.TrimSpace(rule.endpointKey),
		}
	}
	for _, rule := range controlRouteRateLimitPolicyAllowlist {
		if !controlRouteRateLimitRouteMatch(normalizedMethod, normalizedPath, rule.method, rule.path, rule.prefix) {
			continue
		}
		return controlRouteRateLimitPolicyDecision{
			allowlisted: true,
			reason:      strings.TrimSpace(rule.reason),
		}
	}
	return controlRouteRateLimitPolicyDecision{}
}

func controlRouteRateLimitRouteMatch(method string, path string, ruleMethod string, rulePath string, prefix bool) bool {
	if strings.ToUpper(strings.TrimSpace(method)) != strings.ToUpper(strings.TrimSpace(ruleMethod)) {
		return false
	}
	normalizedPath := strings.TrimSpace(path)
	normalizedRulePath := strings.TrimSpace(rulePath)
	if prefix {
		return strings.HasPrefix(normalizedPath, normalizedRulePath)
	}
	return normalizedPath == normalizedRulePath
}
