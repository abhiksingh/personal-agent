package transport

import (
	"net/http"
	"strings"
)

var daemonCapabilityRouteGroups = []DaemonCapabilityRouteGroup{
	{ID: "meta", Prefix: "/v1/meta", Description: "daemon discovery metadata"},
	{ID: "tasks", Prefix: "/v1/tasks", Description: "task submit/list/status/cancel/retry/requeue workflows"},
	{ID: "approvals", Prefix: "/v1/approvals", Description: "approval inbox query workflows"},
	{ID: "lifecycle", Prefix: "/v1/daemon/lifecycle", Description: "daemon lifecycle status/control/history"},
	{ID: "secrets", Prefix: "/v1/secrets/refs", Description: "write-only SecretRef metadata operations"},
	{ID: "providers", Prefix: "/v1/providers", Description: "provider configuration and readiness checks"},
	{ID: "models", Prefix: "/v1/models", Description: "model catalog, route selection, and explainability"},
	{ID: "chat", Prefix: "/v1/chat", Description: "chat turn and realtime lifecycle workflows"},
	{ID: "agent", Prefix: "/v1/agent", Description: "agent run and approval orchestration"},
	{ID: "delegation", Prefix: "/v1/delegation", Description: "delegation and capability grant management"},
	{ID: "identity", Prefix: "/v1/identity", Description: "workspace/principal/session identity flows"},
	{ID: "comm", Prefix: "/v1/comm", Description: "communications send/ingest/policy/query workflows"},
	{ID: "channels", Prefix: "/v1/channels", Description: "channel status, config, mapping, and Twilio flows"},
	{ID: "connectors", Prefix: "/v1/connectors", Description: "connector status/config/permission/cloudflared flows"},
	{ID: "automation", Prefix: "/v1/automation", Description: "automation create/list/run and trigger metadata"},
	{ID: "inspect", Prefix: "/v1/inspect", Description: "inspect run/transcript/memory/log queries"},
	{ID: "retention", Prefix: "/v1/retention", Description: "retention purge and memory compaction"},
	{ID: "context", Prefix: "/v1/context", Description: "context budget and retrieval/memory introspection"},
	{ID: "realtime", Prefix: "/v1/realtime/ws", Description: "realtime WebSocket stream endpoint"},
	{ID: "smoke", Prefix: "/v1/capabilities/smoke", Description: "transport smoke check endpoint"},
}

var daemonRealtimeEventTypes = []string{
	"approval_recorded",
	"chat_completed",
	"chat_error",
	"client_signal",
	"client_signal_ack",
	"tool_call_completed",
	"tool_call_output",
	"tool_call_started",
	"task_run_lifecycle",
	"task_submitted",
	"turn_item_completed",
	"turn_item_delta",
	"turn_item_started",
}

var daemonClientSignalTypes = []string{
	"cancel",
}

var daemonProtocolModes = []string{
	"http_json",
	"websocket_json",
}

func (s *Server) handleMetaCapabilities(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}

	response := DaemonCapabilitiesResponse{
		APIVersion:             responseHeaderCurrentAPIVer,
		RouteGroups:            cloneRouteGroups(daemonCapabilityRouteGroups),
		RealtimeEventTypes:     append([]string(nil), daemonRealtimeEventTypes...),
		RealtimeBackpressure:   s.broker.BackpressurePolicy(),
		RealtimeDiagnostics:    s.broker.Diagnostics(),
		ClientSignalTypes:      append([]string(nil), daemonClientSignalTypes...),
		ProtocolModes:          append([]string(nil), daemonProtocolModes...),
		TransportListenerModes: []string{string(ListenerModeTCP), string(ListenerModeUnix), string(ListenerModeNamedPipe)},
		CorrelationID:          strings.TrimSpace(correlationID),
	}

	writeJSON(writer, http.StatusOK, response, correlationID)
}

func cloneRouteGroups(groups []DaemonCapabilityRouteGroup) []DaemonCapabilityRouteGroup {
	if len(groups) == 0 {
		return nil
	}
	cloned := make([]DaemonCapabilityRouteGroup, len(groups))
	copy(cloned, groups)
	return cloned
}
