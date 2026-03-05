package transport

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizeAuthTokenScopes(t *testing.T) {
	cases := []struct {
		name     string
		raw      []string
		expected []string
	}{
		{name: "empty defaults wildcard", raw: nil, expected: []string{"*"}},
		{name: "dedupe sort lowercase", raw: []string{" Tasks:Write ", "chat:READ", "tasks:write"}, expected: []string{"chat:read", "tasks:write"}},
		{name: "all keyword wildcard", raw: []string{"tasks:read", "all", "chat:write"}, expected: []string{"*"}},
		{name: "asterisk wildcard", raw: []string{"tasks:read", "*"}, expected: []string{"*"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeAuthTokenScopes(tc.raw)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("normalizeAuthTokenScopes(%v)=%v want %v", tc.raw, got, tc.expected)
			}
		})
	}
}

func TestRequiredScopesForRouteSensitiveControlMatrix(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		path     string
		expected []string
	}{
		{name: "metadata", method: http.MethodGet, path: "/v1/meta/capabilities", expected: []string{"metadata:read"}},
		{name: "realtime", method: http.MethodGet, path: "/v1/realtime/ws", expected: []string{"realtime:read"}},
		{name: "daemon status", method: http.MethodGet, path: "/v1/daemon/lifecycle/status", expected: []string{"daemon:read"}},
		{name: "daemon control", method: http.MethodPost, path: "/v1/daemon/lifecycle/control", expected: []string{"daemon:write"}},
		{name: "tasks list", method: http.MethodPost, path: "/v1/tasks/list", expected: []string{"tasks:read"}},
		{name: "task status", method: http.MethodGet, path: "/v1/tasks/task-1", expected: []string{"tasks:read"}},
		{name: "task submit", method: http.MethodPost, path: "/v1/tasks", expected: []string{"tasks:write"}},
		{name: "comm attempts", method: http.MethodPost, path: "/v1/comm/attempts", expected: []string{"comm:read"}},
		{name: "comm send", method: http.MethodPost, path: "/v1/comm/send", expected: []string{"comm:write"}},
		{name: "chat explain", method: http.MethodPost, path: "/v1/chat/turn/explain", expected: []string{"chat:read"}},
		{name: "chat turn", method: http.MethodPost, path: "/v1/chat/turn", expected: []string{"chat:write"}},
		{name: "delegation list", method: http.MethodPost, path: "/v1/delegation/list", expected: []string{"delegation:read"}},
		{name: "delegation grant", method: http.MethodPost, path: "/v1/delegation/grant", expected: []string{"delegation:write"}},
		{name: "identity context", method: http.MethodPost, path: "/v1/identity/context", expected: []string{"identity:read"}},
		{name: "identity select", method: http.MethodPost, path: "/v1/identity/context/select-workspace", expected: []string{"identity:write"}},
		{name: "secret get", method: http.MethodGet, path: "/v1/secrets/refs/ws1/OPENAI_API_KEY", expected: []string{"secrets:read"}},
		{name: "secret delete", method: http.MethodDelete, path: "/v1/secrets/refs/ws1/OPENAI_API_KEY", expected: []string{"secrets:write"}},
		{name: "provider list", method: http.MethodPost, path: "/v1/providers/list", expected: []string{"providers:read"}},
		{name: "provider set", method: http.MethodPost, path: "/v1/providers/set", expected: []string{"providers:write"}},
		{name: "model list", method: http.MethodPost, path: "/v1/models/list", expected: []string{"models:read"}},
		{name: "model add", method: http.MethodPost, path: "/v1/models/add", expected: []string{"models:write"}},
		{name: "channel status", method: http.MethodPost, path: "/v1/channels/status", expected: []string{"channels:read"}},
		{name: "channel upsert", method: http.MethodPost, path: "/v1/channels/config/upsert", expected: []string{"channels:write"}},
		{name: "connector status", method: http.MethodPost, path: "/v1/connectors/status", expected: []string{"connectors:read"}},
		{name: "connector execute", method: http.MethodPost, path: "/v1/connectors/cloudflared/exec", expected: []string{"connectors:write"}},
		{name: "automation list", method: http.MethodPost, path: "/v1/automation/list", expected: []string{"automation:read"}},
		{name: "automation update", method: http.MethodPost, path: "/v1/automation/update", expected: []string{"automation:write"}},
		{name: "inspect logs", method: http.MethodPost, path: "/v1/inspect/logs/query", expected: []string{"inspect:read"}},
		{name: "retention purge", method: http.MethodPost, path: "/v1/retention/purge", expected: []string{"retention:write"}},
		{name: "context samples", method: http.MethodPost, path: "/v1/context/samples", expected: []string{"context:read"}},
		{name: "context tune", method: http.MethodPost, path: "/v1/context/tune", expected: []string{"context:write"}},
		{name: "capability smoke", method: http.MethodGet, path: "/v1/capabilities/smoke", expected: []string{"metadata:read"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "http://localhost"+tc.path, nil)
			got := requiredScopesForRoute(request)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("requiredScopesForRoute(%s %s)=%v want %v", tc.method, tc.path, got, tc.expected)
			}
		})
	}
}

func TestRouteScopeAllowedMatrix(t *testing.T) {
	cases := []struct {
		name            string
		scopes          []string
		method          string
		path            string
		expectedAllowed bool
		expectedScope   []string
	}{
		{name: "tasks read allowed", scopes: []string{"tasks:read"}, method: http.MethodGet, path: "/v1/tasks/task-1", expectedAllowed: true, expectedScope: []string{"tasks:read"}},
		{name: "tasks write denied with read-only", scopes: []string{"tasks:read"}, method: http.MethodPost, path: "/v1/tasks", expectedAllowed: false, expectedScope: []string{"tasks:write"}},
		{name: "chat write allowed", scopes: []string{"chat:write"}, method: http.MethodPost, path: "/v1/chat/turn", expectedAllowed: true, expectedScope: []string{"chat:write"}},
		{name: "chat read denied with write-only", scopes: []string{"chat:write"}, method: http.MethodPost, path: "/v1/chat/history", expectedAllowed: false, expectedScope: []string{"chat:read"}},
		{name: "chat read allowed", scopes: []string{"chat:read"}, method: http.MethodPost, path: "/v1/chat/history", expectedAllowed: true, expectedScope: []string{"chat:read"}},
		{name: "wildcard allowed", scopes: []string{"*"}, method: http.MethodPost, path: "/v1/connectors/config/upsert", expectedAllowed: true, expectedScope: []string{"connectors:write"}},
		{name: "capability smoke denied with unrelated scope", scopes: []string{"tasks:write"}, method: http.MethodGet, path: "/v1/capabilities/smoke", expectedAllowed: false, expectedScope: []string{"metadata:read"}},
		{name: "capability smoke allowed with metadata scope", scopes: []string{"metadata:read"}, method: http.MethodGet, path: "/v1/capabilities/smoke", expectedAllowed: true, expectedScope: []string{"metadata:read"}},
		{name: "empty configured scopes allow", scopes: nil, method: http.MethodPost, path: "/v1/chat/turn", expectedAllowed: true, expectedScope: []string{"chat:write"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{
				authTokenScopes:   append([]string(nil), tc.scopes...),
				authTokenScopeSet: scopeSetFromList(tc.scopes),
			}
			request := httptest.NewRequest(tc.method, "http://localhost"+tc.path, nil)
			allowed, required := server.routeScopeAllowed(request)
			if allowed != tc.expectedAllowed {
				t.Fatalf("routeScopeAllowed(%s %s) allowed=%t want %t", tc.method, tc.path, allowed, tc.expectedAllowed)
			}
			if !reflect.DeepEqual(required, tc.expectedScope) {
				t.Fatalf("routeScopeAllowed(%s %s) required=%v want %v", tc.method, tc.path, required, tc.expectedScope)
			}
		})
	}
}

func TestRouteScopeAllowedNilServerDeniesScopedRoute(t *testing.T) {
	var server *Server
	request := httptest.NewRequest(http.MethodPost, "http://localhost/v1/chat/turn", nil)
	allowed, required := server.routeScopeAllowed(request)
	if allowed {
		t.Fatalf("expected nil server scope check to deny scoped route")
	}
	expected := []string{"chat:write"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("expected nil server required scopes %v, got %v", expected, required)
	}
}
