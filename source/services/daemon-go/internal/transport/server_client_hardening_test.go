package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTransportAuthIsRequired(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "expected-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "wrong-token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.CapabilitySmoke(context.Background(), "corr-smoke")
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}

	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", httpErr.StatusCode)
	}
	if httpErr.Code != "auth_unauthorized" {
		t.Fatalf("expected code auth_unauthorized, got %q", httpErr.Code)
	}
	if strings.TrimSpace(httpErr.Message) != "unauthorized" {
		t.Fatalf("expected unauthorized message, got %q", httpErr.Message)
	}
	if strings.TrimSpace(httpErr.CorrelationID) == "" {
		t.Fatalf("expected correlation id on unauthorized errors")
	}
}

func TestTransportRouteScopePolicyRejectsForbiddenRoute(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "scope-token",
		AuthTokenScopes: []string{"tasks:write"},
	})

	baseURL := "http://" + server.Address()
	httpClient := &http.Client{Timeout: 2 * time.Second}

	allowedRequest, err := http.NewRequest(http.MethodPost, baseURL+"/v1/tasks", bytes.NewReader([]byte(`{
		"workspace_id":"ws1",
		"requested_by_actor_id":"actor.requester",
		"subject_principal_actor_id":"actor.subject",
		"title":"scoped task"
	}`)))
	if err != nil {
		t.Fatalf("build scoped allowed request: %v", err)
	}
	allowedRequest.Header.Set("Authorization", "Bearer scope-token")
	allowedRequest.Header.Set("Content-Type", "application/json")

	allowedResponse, err := httpClient.Do(allowedRequest)
	if err != nil {
		t.Fatalf("execute scoped allowed request: %v", err)
	}
	if allowedResponse.StatusCode == http.StatusForbidden {
		var payload map[string]any
		_ = json.NewDecoder(allowedResponse.Body).Decode(&payload)
		_ = allowedResponse.Body.Close()
		t.Fatalf("expected /v1/tasks to remain authorized with tasks:write scope, payload=%v", payload)
	}
	_ = allowedResponse.Body.Close()

	forbiddenRequest, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/turn", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("build scoped forbidden request: %v", err)
	}
	forbiddenRequest.Header.Set("Authorization", "Bearer scope-token")
	forbiddenRequest.Header.Set("Content-Type", "application/json")

	forbiddenResponse, err := httpClient.Do(forbiddenRequest)
	if err != nil {
		t.Fatalf("execute scoped forbidden request: %v", err)
	}
	defer forbiddenResponse.Body.Close()
	if forbiddenResponse.StatusCode != http.StatusForbidden {
		var payload map[string]any
		_ = json.NewDecoder(forbiddenResponse.Body).Decode(&payload)
		t.Fatalf("expected scoped forbidden route status 403, got %d payload=%v", forbiddenResponse.StatusCode, payload)
	}

	var payload map[string]any
	if err := json.NewDecoder(forbiddenResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode scoped forbidden response: %v", err)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in forbidden response")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != "auth_forbidden" {
		t.Fatalf("expected auth_forbidden code, got %q", got)
	}

	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected forbidden response details")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected forbidden details map, got %T", detailsRaw)
	}
	requiredRaw, ok := details["required_scopes"]
	if !ok {
		t.Fatalf("expected required_scopes details")
	}
	requiredScopes, ok := requiredRaw.([]any)
	if !ok || len(requiredScopes) == 0 {
		t.Fatalf("expected non-empty required_scopes array, got %v", requiredRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(requiredScopes[0])); got != "chat:write" {
		t.Fatalf("expected required scope chat:write, got %q", got)
	}
	grantedRaw, ok := details["granted_scopes"]
	if !ok {
		t.Fatalf("expected granted_scopes details")
	}
	grantedScopes, ok := grantedRaw.([]any)
	if !ok || len(grantedScopes) == 0 {
		t.Fatalf("expected non-empty granted_scopes array, got %v", grantedRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(grantedScopes[0])); got != "tasks:write" {
		t.Fatalf("expected granted scope tasks:write, got %q", got)
	}
}

func TestTransportRouteScopePolicyRejectsPrivilegedReadRouteWithoutScope(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "scope-token",
		AuthTokenScopes: []string{"tasks:read"},
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+server.Address()+"/v1/daemon/lifecycle/status", nil)
	if err != nil {
		t.Fatalf("build scoped privileged-read request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer scope-token")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute scoped privileged-read request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		var payload map[string]any
		_ = json.NewDecoder(response.Body).Decode(&payload)
		t.Fatalf("expected privileged-read route status 403, got %d payload=%v", response.StatusCode, payload)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode scoped privileged-read response: %v", err)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in privileged-read forbidden response")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != "auth_forbidden" {
		t.Fatalf("expected auth_forbidden code, got %q", got)
	}

	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected privileged-read forbidden response details")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected privileged-read forbidden details map, got %T", detailsRaw)
	}
	requiredRaw, ok := details["required_scopes"]
	if !ok {
		t.Fatalf("expected required_scopes details for privileged-read route")
	}
	requiredScopes, ok := requiredRaw.([]any)
	if !ok || len(requiredScopes) == 0 {
		t.Fatalf("expected non-empty required_scopes array, got %v", requiredRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(requiredScopes[0])); got != "daemon:read" {
		t.Fatalf("expected required scope daemon:read, got %q", got)
	}
}

func TestTransportRouteScopePolicyRejectsCapabilitySmokeWithoutMetadataScope(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "scope-token",
		AuthTokenScopes: []string{"tasks:read"},
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+server.Address()+"/v1/capabilities/smoke", nil)
	if err != nil {
		t.Fatalf("build scoped capability-smoke request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer scope-token")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute scoped capability-smoke request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		var payload map[string]any
		_ = json.NewDecoder(response.Body).Decode(&payload)
		t.Fatalf("expected capability-smoke status 403, got %d payload=%v", response.StatusCode, payload)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode capability-smoke forbidden response: %v", err)
	}

	errorObjectRaw, ok := payload["error"]
	if !ok {
		t.Fatalf("expected typed error object in capability-smoke forbidden response")
	}
	errorObject, ok := errorObjectRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected error object map, got %T", errorObjectRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(errorObject["code"])); got != "auth_forbidden" {
		t.Fatalf("expected auth_forbidden code, got %q", got)
	}

	detailsRaw, ok := errorObject["details"]
	if !ok {
		t.Fatalf("expected capability-smoke forbidden response details")
	}
	details, ok := detailsRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected capability-smoke forbidden details map, got %T", detailsRaw)
	}
	requiredRaw, ok := details["required_scopes"]
	if !ok {
		t.Fatalf("expected required_scopes details for capability-smoke route")
	}
	requiredScopes, ok := requiredRaw.([]any)
	if !ok || len(requiredScopes) == 0 {
		t.Fatalf("expected non-empty required_scopes array, got %v", requiredRaw)
	}
	if got := strings.TrimSpace(fmt.Sprint(requiredScopes[0])); got != "metadata:read" {
		t.Fatalf("expected required scope metadata:read, got %q", got)
	}
}

func TestTransportRouteScopePolicyAllowsCapabilitySmokeWithMetadataScope(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:    ListenerModeTCP,
		Address:         "127.0.0.1:0",
		AuthToken:       "scope-token",
		AuthTokenScopes: []string{"metadata:read"},
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+server.Address()+"/v1/capabilities/smoke", nil)
	if err != nil {
		t.Fatalf("build scoped capability-smoke request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer scope-token")

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("execute scoped capability-smoke request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var payload map[string]any
		_ = json.NewDecoder(response.Body).Decode(&payload)
		t.Fatalf("expected capability-smoke status 200, got %d payload=%v", response.StatusCode, payload)
	}
}

func TestTransportSubmitTaskRejectsOversizedPayload(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:          ListenerModeTCP,
		Address:               "127.0.0.1:0",
		AuthToken:             "size-limit-token",
		RequestBodyBytesLimit: 128,
	})
	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "size-limit-token",
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}

	_, err = client.SubmitTask(context.Background(), SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor-requester",
		SubjectPrincipalActorID: "actor-subject",
		Title:                   strings.Repeat("x", 512),
		TaskClass:               "chat",
	}, "corr-submit-too-large")
	if err == nil {
		t.Fatalf("expected oversized submit task payload to fail")
	}

	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413 for oversized payload, got %d", httpErr.StatusCode)
	}
	if httpErr.Code != "request_payload_too_large" {
		t.Fatalf("expected code request_payload_too_large, got %q", httpErr.Code)
	}
	if !strings.Contains(httpErr.Message, "128") {
		t.Fatalf("expected oversized payload message to include byte limit, got %q", httpErr.Message)
	}
}

func TestTransportRealtimeRejectsNonAllowlistedOrigin(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "origin-token",
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")
	headers.Set("Origin", "https://evil.example.com")

	conn, response, err := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if err == nil {
		_ = conn.Close()
		t.Fatalf("expected websocket origin rejection")
	}
	if response == nil {
		t.Fatalf("expected origin rejection response")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected websocket origin rejection status 403, got %d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	body, _ := io.ReadAll(response.Body)
	httpErr := parseTransportHTTPError(response.StatusCode, body, strings.TrimSpace(response.Header.Get(responseHeaderCorrelationID)))
	if httpErr.Code != "auth_forbidden" {
		t.Fatalf("expected auth_forbidden code for rejected websocket origin, got %q", httpErr.Code)
	}
	if strings.TrimSpace(httpErr.Message) != "realtime websocket origin is not allowed" {
		t.Fatalf("unexpected websocket origin rejection message: %q", httpErr.Message)
	}
}

func TestTransportRealtimeAllowsLoopbackOriginInLocalProfile(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "local",
		AuthToken:      "origin-token",
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")
	headers.Set("Origin", "http://localhost:3000")

	conn, response, err := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if err != nil {
		if response != nil {
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("expected loopback websocket origin to be accepted, got status %d body=%s err=%v", response.StatusCode, strings.TrimSpace(string(body)), err)
		}
		t.Fatalf("expected loopback websocket origin to be accepted: %v", err)
	}
	defer conn.Close()
}

func TestTransportRealtimeAllowsMissingOriginInProdProfile(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:   ListenerModeTCP,
		Address:        "127.0.0.1:0",
		RuntimeProfile: "prod",
		AuthToken:      "origin-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "origin-token",
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-origin-prod-no-header")
	if err != nil {
		t.Fatalf("expected websocket connection without Origin header to remain supported in prod profile: %v", err)
	}
	_ = stream.Close()
}

func TestTransportRealtimeHonorsExplicitOriginAllowlist(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:             ListenerModeTCP,
		Address:                  "127.0.0.1:0",
		RuntimeProfile:           "prod",
		AuthToken:                "origin-token",
		WebSocketOriginAllowlist: []string{"https://console.example.com"},
	})

	allowedHeaders := http.Header{}
	allowedHeaders.Set("Authorization", "Bearer origin-token")
	allowedHeaders.Set("Origin", "https://console.example.com")

	allowedConn, allowedResponse, allowedErr := dialRealtimeWSWithHeaders(t, server.Address(), allowedHeaders)
	if allowedErr != nil {
		if allowedResponse != nil {
			defer allowedResponse.Body.Close()
			body, _ := io.ReadAll(allowedResponse.Body)
			t.Fatalf("expected explicitly allowlisted websocket origin to be accepted, got status %d body=%s err=%v", allowedResponse.StatusCode, strings.TrimSpace(string(body)), allowedErr)
		}
		t.Fatalf("expected explicitly allowlisted websocket origin to be accepted: %v", allowedErr)
	}
	_ = allowedConn.Close()

	deniedHeaders := http.Header{}
	deniedHeaders.Set("Authorization", "Bearer origin-token")
	deniedHeaders.Set("Origin", "https://denied.example.com")

	deniedConn, deniedResponse, deniedErr := dialRealtimeWSWithHeaders(t, server.Address(), deniedHeaders)
	if deniedErr == nil {
		_ = deniedConn.Close()
		t.Fatalf("expected non-allowlisted websocket origin to be rejected")
	}
	if deniedResponse == nil {
		t.Fatalf("expected response for non-allowlisted websocket origin rejection")
	}
	defer deniedResponse.Body.Close()
	if deniedResponse.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(deniedResponse.Body)
		t.Fatalf("expected 403 for non-allowlisted websocket origin, got %d body=%s", deniedResponse.StatusCode, strings.TrimSpace(string(body)))
	}
}

func TestTransportRealtimeRejectsWhenConnectionCapExceeded(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:             ListenerModeTCP,
		Address:                  "127.0.0.1:0",
		AuthToken:                "origin-token",
		RealtimeMaxConnections:   1,
		RealtimeMaxSubscriptions: 1,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")

	firstConn, firstResponse, firstErr := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if firstErr != nil {
		if firstResponse != nil {
			defer firstResponse.Body.Close()
			body, _ := io.ReadAll(firstResponse.Body)
			t.Fatalf("expected first websocket connection to succeed, got status %d body=%s err=%v", firstResponse.StatusCode, strings.TrimSpace(string(body)), firstErr)
		}
		t.Fatalf("expected first websocket connection to succeed: %v", firstErr)
	}
	defer firstConn.Close()

	secondConn, secondResponse, secondErr := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if secondErr == nil {
		_ = secondConn.Close()
		t.Fatalf("expected second websocket connection to be rejected by cap")
	}
	if secondResponse == nil {
		t.Fatalf("expected HTTP response for capped websocket rejection")
	}
	defer secondResponse.Body.Close()
	if secondResponse.StatusCode != http.StatusTooManyRequests {
		body, _ := io.ReadAll(secondResponse.Body)
		t.Fatalf("expected 429 for capped websocket rejection, got %d body=%s", secondResponse.StatusCode, strings.TrimSpace(string(body)))
	}

	body, _ := io.ReadAll(secondResponse.Body)
	httpErr := parseTransportHTTPError(secondResponse.StatusCode, body, strings.TrimSpace(secondResponse.Header.Get(responseHeaderCorrelationID)))
	if httpErr.Code != "rate_limit_exceeded" {
		t.Fatalf("expected rate_limit_exceeded code for capped websocket rejection, got %q", httpErr.Code)
	}
	var details map[string]any
	if err := json.Unmarshal(httpErr.DetailsPayload, &details); err != nil {
		t.Fatalf("decode capped websocket details payload: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["limit_type"])); got != "connections" {
		t.Fatalf("expected limit_type connections, got %q", got)
	}
}

func TestTransportRealtimeRejectsWhenSubscriptionCapExceeded(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:             ListenerModeTCP,
		Address:                  "127.0.0.1:0",
		AuthToken:                "origin-token",
		RealtimeMaxConnections:   2,
		RealtimeMaxSubscriptions: 1,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")

	firstConn, firstResponse, firstErr := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if firstErr != nil {
		if firstResponse != nil {
			defer firstResponse.Body.Close()
			body, _ := io.ReadAll(firstResponse.Body)
			t.Fatalf("expected first websocket connection to succeed, got status %d body=%s err=%v", firstResponse.StatusCode, strings.TrimSpace(string(body)), firstErr)
		}
		t.Fatalf("expected first websocket connection to succeed: %v", firstErr)
	}
	defer firstConn.Close()

	secondConn, secondResponse, secondErr := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if secondErr == nil {
		_ = secondConn.Close()
		t.Fatalf("expected second websocket connection to be rejected by subscription cap")
	}
	if secondResponse == nil {
		t.Fatalf("expected HTTP response for capped websocket rejection")
	}
	defer secondResponse.Body.Close()
	if secondResponse.StatusCode != http.StatusTooManyRequests {
		body, _ := io.ReadAll(secondResponse.Body)
		t.Fatalf("expected 429 for capped websocket rejection, got %d body=%s", secondResponse.StatusCode, strings.TrimSpace(string(body)))
	}

	body, _ := io.ReadAll(secondResponse.Body)
	httpErr := parseTransportHTTPError(secondResponse.StatusCode, body, strings.TrimSpace(secondResponse.Header.Get(responseHeaderCorrelationID)))
	if httpErr.Code != "rate_limit_exceeded" {
		t.Fatalf("expected rate_limit_exceeded code for capped websocket rejection, got %q", httpErr.Code)
	}
	var details map[string]any
	if err := json.Unmarshal(httpErr.DetailsPayload, &details); err != nil {
		t.Fatalf("decode capped websocket details payload: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(details["limit_type"])); got != "subscriptions" {
		t.Fatalf("expected limit_type subscriptions, got %q", got)
	}
}

func TestTransportRealtimeClosesConnectionOnOversizedSignalPayload(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:           ListenerModeTCP,
		Address:                "127.0.0.1:0",
		AuthToken:              "origin-token",
		RealtimeReadLimitBytes: 128,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")

	conn, response, err := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if err != nil {
		if response != nil {
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("expected websocket connection for oversized payload test, got status %d body=%s err=%v", response.StatusCode, strings.TrimSpace(string(body)), err)
		}
		t.Fatalf("expected websocket connection for oversized payload test: %v", err)
	}
	defer conn.Close()

	writeErr := conn.WriteJSON(map[string]any{
		"signal_type": "cancel",
		"reason":      strings.Repeat("x", 2048),
	})
	if writeErr == nil {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, readErr := conn.ReadMessage(); readErr == nil {
			t.Fatalf("expected oversized payload connection to be closed")
		}
	}
}

func TestTransportRealtimeClosesStaleClientWithoutPong(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode:             ListenerModeTCP,
		Address:                  "127.0.0.1:0",
		AuthToken:                "origin-token",
		RealtimePingInterval:     25 * time.Millisecond,
		RealtimePongTimeout:      80 * time.Millisecond,
		RealtimeWriteTimeout:     30 * time.Millisecond,
		RealtimeMaxConnections:   2,
		RealtimeMaxSubscriptions: 2,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer origin-token")

	conn, response, err := dialRealtimeWSWithHeaders(t, server.Address(), headers)
	if err != nil {
		if response != nil {
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("expected websocket connection for stale-client test, got status %d body=%s err=%v", response.StatusCode, strings.TrimSpace(string(body)), err)
		}
		t.Fatalf("expected websocket connection for stale-client test: %v", err)
	}
	defer conn.Close()

	time.Sleep(220 * time.Millisecond)
	conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	if _, _, readErr := conn.ReadMessage(); readErr == nil {
		t.Fatalf("expected stale websocket client to be disconnected")
	}
}

func TestTransportRealtimeClientSignalAckRejectsUnsupportedSignalType(t *testing.T) {
	server := startTestServer(t, ServerConfig{
		ListenerMode: ListenerModeTCP,
		Address:      "127.0.0.1:0",
		AuthToken:    "test-token",
	})

	client, err := NewClient(ClientConfig{
		ListenerMode: ListenerModeTCP,
		Address:      server.Address(),
		AuthToken:    "test-token",
	})
	if err != nil {
		t.Fatalf("create transport client: %v", err)
	}

	stream, err := client.ConnectRealtime(context.Background(), "corr-stream-ack")
	if err != nil {
		t.Fatalf("connect realtime stream: %v", err)
	}
	defer stream.Close()

	if err := stream.SendSignal(ClientSignal{
		SignalType:    "unsupported",
		CorrelationID: "corr-unsupported-signal",
	}); err != nil {
		t.Fatalf("send unsupported client signal: %v", err)
	}

	event, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive client signal event: %v", err)
	}
	if event.EventType != "client_signal" {
		t.Fatalf("expected client_signal event, got %s", event.EventType)
	}

	ackEvent, err := stream.Receive()
	if err != nil {
		t.Fatalf("receive client signal ack: %v", err)
	}
	if ackEvent.EventType != "client_signal_ack" {
		t.Fatalf("expected client_signal_ack event, got %s", ackEvent.EventType)
	}
	if gotAccepted := fmt.Sprintf("%v", ackEvent.Payload.AsMap()["accepted"]); gotAccepted != "false" {
		t.Fatalf("expected accepted=false for unsupported signal, got %v", ackEvent.Payload.AsMap()["accepted"])
	}
	if gotReason := strings.TrimSpace(fmt.Sprintf("%v", ackEvent.Payload.AsMap()["reason"])); !strings.Contains(gotReason, "unsupported") {
		t.Fatalf("expected unsupported reason in ack payload, got %q", gotReason)
	}
	if strings.TrimSpace(ackEvent.CorrelationID) != "corr-unsupported-signal" {
		t.Fatalf("expected signal correlation corr-unsupported-signal, got %s", ackEvent.CorrelationID)
	}
}
