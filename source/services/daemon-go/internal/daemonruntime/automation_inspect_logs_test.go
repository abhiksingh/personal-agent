package daemonruntime

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"personalagent/runtime/internal/transport"
)

func TestAutomationInspectLogsQueryReturnsLIFOAndSummaries(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	configureOllamaProviderForRouteTests(t, container)
	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	ctx := context.Background()
	insertAuditLogEntry(t, container.DB, "ws1", "audit-older", "STEP_EXECUTED", `{"status":"COMPLETED","summary":"older summary"}`, "2026-02-24T00:00:01Z")
	insertAuditLogEntry(t, container.DB, "ws1", "audit-newer", "APPROVAL_REQUESTED", `{"approval_request_id":"apr-1","requested_phrase":"GO AHEAD"}`, "2026-02-24T00:00:02Z")

	response, err := service.QueryInspectLogs(ctx, transport.InspectLogQueryRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("query inspect logs: %v", err)
	}
	if len(response.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(response.Logs))
	}
	if response.Logs[0].LogID != "audit-newer" {
		t.Fatalf("expected newest log first, got %s", response.Logs[0].LogID)
	}
	if response.Logs[0].Status != "awaiting_approval" {
		t.Fatalf("expected approval status, got %s", response.Logs[0].Status)
	}
	if response.Logs[0].InputSummary == "" {
		t.Fatalf("expected input summary for approval log")
	}
	if response.Logs[0].Route.TaskClass == "" || response.Logs[0].Route.TaskClassSource == "" || response.Logs[0].Route.RouteSource == "" {
		t.Fatalf("expected deterministic route metadata on query response log, got %+v", response.Logs[0].Route)
	}
	if response.Logs[1].OutputSummary != "older summary" {
		t.Fatalf("expected step summary output, got %q", response.Logs[1].OutputSummary)
	}
	if response.Logs[1].Route.TaskClass == "" || response.Logs[1].Route.TaskClassSource == "" || response.Logs[1].Route.RouteSource == "" {
		t.Fatalf("expected deterministic route metadata on older query response log, got %+v", response.Logs[1].Route)
	}
	if response.NextCursorID != "audit-older" {
		t.Fatalf("expected next cursor id audit-older, got %s", response.NextCursorID)
	}
}

func TestAutomationInspectLogsStreamReturnsNewEntriesAfterCursor(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	configureOllamaProviderForRouteTests(t, container)
	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	ctx := context.Background()
	insertAuditLogEntry(t, container.DB, "ws1", "audit-base", "STEP_EXECUTED", `{"status":"COMPLETED","summary":"base summary"}`, "2026-02-24T00:00:03Z")

	insertErr := make(chan error, 1)
	go func() {
		time.Sleep(150 * time.Millisecond)
		_, err := container.DB.Exec(`
			INSERT INTO audit_log_entries(
				id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id,
				correlation_id, payload_json, created_at
			) VALUES (?, ?, NULL, NULL, ?, NULL, NULL, ?, ?, ?)
		`, "audit-live", "ws1", "STEP_EXECUTED", "corr-audit-live", `{"status":"FAILED","error":"connector timeout"}`, "2026-02-24T00:00:04Z")
		insertErr <- err
	}()

	response, err := service.StreamInspectLogs(ctx, transport.InspectLogStreamRequest{
		WorkspaceID:     "ws1",
		CursorCreatedAt: "2026-02-24T00:00:03Z",
		CursorID:        "audit-base",
		Limit:           10,
		TimeoutMS:       2000,
		PollIntervalMS:  100,
	})
	if err != nil {
		t.Fatalf("stream inspect logs: %v", err)
	}
	if response.TimedOut {
		t.Fatalf("expected stream response with new data, got timeout")
	}
	if len(response.Logs) == 0 {
		t.Fatalf("expected at least one streamed log")
	}
	if response.Logs[0].LogID != "audit-live" {
		t.Fatalf("expected audit-live first, got %s", response.Logs[0].LogID)
	}
	if response.Logs[0].Status != "failed" {
		t.Fatalf("expected failed status from payload, got %s", response.Logs[0].Status)
	}
	if response.Logs[0].Route.TaskClass == "" || response.Logs[0].Route.TaskClassSource == "" || response.Logs[0].Route.RouteSource == "" {
		t.Fatalf("expected deterministic route metadata on stream response log, got %+v", response.Logs[0].Route)
	}
	if err := <-insertErr; err != nil {
		t.Fatalf("insert streamed audit row: %v", err)
	}
}

func TestAutomationInspectLogsQueryExtractsTypedMetadataFields(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	configureOllamaProviderForRouteTests(t, container)
	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	ctx := context.Background()
	insertAuditLogEntry(t, container.DB, "ws1", "audit-metadata", "PLUGIN_WORKER_STARTED", `{
		"status":"RUNNING",
		"summary":"worker started",
		"plugin_id":"twilio-plugin",
		"kind":"channel",
		"state":"started",
		"process_id":"12345",
		"restart_count":2,
		"approval_request_id":"apr-123",
		"accepted":true,
		"replayed":false,
		"ignored_field":"ignored"
	}`, "2026-02-24T00:00:05Z")

	response, err := service.QueryInspectLogs(ctx, transport.InspectLogQueryRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("query inspect logs: %v", err)
	}
	if len(response.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(response.Logs))
	}
	log := response.Logs[0]
	if log.Status != "running" {
		t.Fatalf("expected payload status to win, got %q", log.Status)
	}
	if log.InputSummary != "approval_request_id=apr-123" {
		t.Fatalf("expected deterministic input summary, got %q", log.InputSummary)
	}
	if log.OutputSummary != "worker started" {
		t.Fatalf("expected summary output, got %q", log.OutputSummary)
	}

	if got, ok := log.Metadata["plugin_id"].(string); !ok || got != "twilio-plugin" {
		t.Fatalf("expected plugin_id metadata, got %#v", log.Metadata["plugin_id"])
	}
	if got, ok := log.Metadata["kind"].(string); !ok || got != "channel" {
		t.Fatalf("expected kind metadata, got %#v", log.Metadata["kind"])
	}
	if got, ok := log.Metadata["state"].(string); !ok || got != "started" {
		t.Fatalf("expected state metadata, got %#v", log.Metadata["state"])
	}
	if got, ok := log.Metadata["process_id"].(string); !ok || got != "12345" {
		t.Fatalf("expected process_id metadata, got %#v", log.Metadata["process_id"])
	}
	if got, ok := log.Metadata["approval_request_id"].(string); !ok || got != "apr-123" {
		t.Fatalf("expected approval_request_id metadata, got %#v", log.Metadata["approval_request_id"])
	}
	if got, ok := log.Metadata["accepted"].(bool); !ok || !got {
		t.Fatalf("expected accepted=true metadata, got %#v", log.Metadata["accepted"])
	}
	if got, ok := log.Metadata["replayed"].(bool); !ok || got {
		t.Fatalf("expected replayed=false metadata, got %#v", log.Metadata["replayed"])
	}
	if got, ok := log.Metadata["restart_count"].(float64); !ok || got != 2 {
		t.Fatalf("expected restart_count=2 metadata, got %#v", log.Metadata["restart_count"])
	}
	if _, exists := log.Metadata["ignored_field"]; exists {
		t.Fatalf("expected unknown metadata key to be excluded")
	}
}

func TestAutomationInspectLogsQueryInvalidPayloadAddsParseErrorMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	configureOllamaProviderForRouteTests(t, container)
	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	ctx := context.Background()
	insertAuditLogEntry(t, container.DB, "ws1", "audit-invalid", "PLUGIN_HEALTH_TIMEOUT", `{invalid`, "2026-02-24T00:00:06Z")

	response, err := service.QueryInspectLogs(ctx, transport.InspectLogQueryRequest{
		WorkspaceID: "ws1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("query inspect logs: %v", err)
	}
	if len(response.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(response.Logs))
	}
	log := response.Logs[0]
	if log.Status != "failed" {
		t.Fatalf("expected event-derived fallback status for invalid payload, got %q", log.Status)
	}
	if log.InputSummary != "" {
		t.Fatalf("expected empty input summary for invalid payload, got %q", log.InputSummary)
	}
	if log.OutputSummary != "" {
		t.Fatalf("expected empty output summary for invalid payload, got %q", log.OutputSummary)
	}
	if got, ok := log.Metadata["payload_parse_error"].(string); !ok || got != "invalid json" {
		t.Fatalf("expected payload_parse_error metadata, got %#v", log.Metadata["payload_parse_error"])
	}
}

func insertAuditLogEntry(t *testing.T, db *sql.DB, workspaceID string, auditID string, eventType string, payloadJSON string, createdAt string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspaceID, workspaceID, createdAt, createdAt); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id,
			correlation_id, payload_json, created_at
		) VALUES (?, ?, NULL, NULL, ?, NULL, NULL, ?, ?, ?)
	`, auditID, workspaceID, eventType, "corr-"+auditID, payloadJSON, createdAt); err != nil {
		t.Fatalf("insert audit log entry %s: %v", auditID, err)
	}
}
