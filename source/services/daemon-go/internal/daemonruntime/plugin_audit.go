package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	daemonPluginAuditWorkspaceID          = "daemon"
	pluginLifecycleAuditQueueSize         = 128
	pluginLifecycleAuditPersistTimeout    = 2 * time.Second
	pluginLifecycleAuditLastErrorMaxChars = 220
)

type pluginLifecycleAuditDiagnostics struct {
	QueueDepth      int
	QueueCapacity   int
	DroppedEvents   uint64
	PersistedEvents uint64
	PersistFailures uint64
	LastFailure     string
	LastFailureAt   time.Time
}

func (d pluginLifecycleAuditDiagnostics) degraded() bool {
	return d.DroppedEvents > 0 || d.PersistFailures > 0
}

type pluginLifecycleAuditRuntime struct {
	mu      sync.RWMutex
	queue   chan PluginLifecycleEvent
	wg      sync.WaitGroup
	persist func(event PluginLifecycleEvent) error

	droppedEvents   uint64
	persistedEvents uint64
	persistFailures uint64

	lastFailureMu sync.RWMutex
	lastFailure   string
	lastFailureAt time.Time
}

func newPluginLifecycleAuditRuntime(
	queueSize int,
	persistFn func(event PluginLifecycleEvent) error,
) *pluginLifecycleAuditRuntime {
	if queueSize <= 0 {
		queueSize = pluginLifecycleAuditQueueSize
	}
	return &pluginLifecycleAuditRuntime{
		queue:   make(chan PluginLifecycleEvent, queueSize),
		persist: persistFn,
	}
}

func (r *pluginLifecycleAuditRuntime) start() {
	if r == nil {
		return
	}
	r.wg.Add(1)
	go r.run()
}

func (r *pluginLifecycleAuditRuntime) stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	queue := r.queue
	r.queue = nil
	if queue != nil {
		close(queue)
	}
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *pluginLifecycleAuditRuntime) enqueue(event PluginLifecycleEvent) {
	if r == nil {
		return
	}
	r.mu.RLock()
	queue := r.queue
	if queue == nil {
		r.mu.RUnlock()
		atomic.AddUint64(&r.droppedEvents, 1)
		return
	}

	select {
	case queue <- event:
		r.mu.RUnlock()
		return
	default:
	}

	// Saturation policy: drop oldest buffered event and keep newest lifecycle
	// visibility, while never blocking hook-dispatch callers.
	dropped := false
	select {
	case <-queue:
		dropped = true
	default:
	}
	if dropped {
		atomic.AddUint64(&r.droppedEvents, 1)
	}

	select {
	case queue <- event:
		r.mu.RUnlock()
		return
	default:
		atomic.AddUint64(&r.droppedEvents, 1)
		r.mu.RUnlock()
		return
	}
}

func (r *pluginLifecycleAuditRuntime) diagnostics() pluginLifecycleAuditDiagnostics {
	if r == nil {
		return pluginLifecycleAuditDiagnostics{}
	}
	diagnostics := pluginLifecycleAuditDiagnostics{
		DroppedEvents:   atomic.LoadUint64(&r.droppedEvents),
		PersistedEvents: atomic.LoadUint64(&r.persistedEvents),
		PersistFailures: atomic.LoadUint64(&r.persistFailures),
	}

	r.mu.RLock()
	if r.queue != nil {
		diagnostics.QueueDepth = len(r.queue)
		diagnostics.QueueCapacity = cap(r.queue)
	}
	r.mu.RUnlock()

	r.lastFailureMu.RLock()
	diagnostics.LastFailure = r.lastFailure
	diagnostics.LastFailureAt = r.lastFailureAt
	r.lastFailureMu.RUnlock()
	return diagnostics
}

func (r *pluginLifecycleAuditRuntime) run() {
	defer r.wg.Done()

	r.mu.RLock()
	queue := r.queue
	r.mu.RUnlock()
	if queue == nil {
		return
	}

	for event := range queue {
		if r.persist == nil {
			continue
		}
		if err := r.persist(event); err != nil {
			r.recordPersistFailure(err)
			continue
		}
		atomic.AddUint64(&r.persistedEvents, 1)
	}
}

func (r *pluginLifecycleAuditRuntime) recordPersistFailure(err error) {
	if r == nil || err == nil {
		return
	}
	atomic.AddUint64(&r.persistFailures, 1)
	message := pluginLifecycleAuditFailureMessage(err)
	r.lastFailureMu.Lock()
	r.lastFailure = message
	r.lastFailureAt = time.Now().UTC()
	r.lastFailureMu.Unlock()
}

func (c *ServiceContainer) startPluginLifecycleAuditRuntime() {
	if c == nil || c.DB == nil || c.pluginLifecycleAudit != nil {
		return
	}
	runtime := newPluginLifecycleAuditRuntime(pluginLifecycleAuditQueueSize, c.persistPluginLifecycleEvent)
	c.pluginLifecycleAudit = runtime
	runtime.start()
}

func (c *ServiceContainer) stopPluginLifecycleAuditRuntime() {
	if c == nil || c.pluginLifecycleAudit == nil {
		return
	}
	c.pluginLifecycleAudit.stop()
	c.pluginLifecycleAudit = nil
}

func (c *ServiceContainer) pluginLifecycleAuditDiagnostics() pluginLifecycleAuditDiagnostics {
	if c == nil || c.pluginLifecycleAudit == nil {
		return pluginLifecycleAuditDiagnostics{}
	}
	return c.pluginLifecycleAudit.diagnostics()
}

func (c *ServiceContainer) pluginLifecycleHooks() PluginLifecycleHooks {
	return PluginLifecycleHooks{
		OnEvent: c.enqueuePluginLifecycleAuditEvent,
	}
}

func (c *ServiceContainer) enqueuePluginLifecycleAuditEvent(event PluginLifecycleEvent) {
	if c == nil {
		return
	}
	if c.pluginLifecycleAudit == nil {
		c.recordPluginLifecycleEvent(event)
		return
	}
	c.pluginLifecycleAudit.enqueue(event)
}

func (c *ServiceContainer) recordPluginLifecycleEvent(event PluginLifecycleEvent) {
	if c == nil {
		return
	}
	if err := c.persistPluginLifecycleEvent(event); err != nil && c.pluginLifecycleAudit != nil {
		c.pluginLifecycleAudit.recordPersistFailure(err)
	}
}

func (c *ServiceContainer) persistPluginLifecycleEvent(event PluginLifecycleEvent) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("plugin lifecycle audit persistence database is not configured")
	}

	pluginID := strings.TrimSpace(event.PluginID)
	if pluginID == "" {
		return fmt.Errorf("plugin id is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginLifecycleAuditPersistTimeout)
	defer cancel()

	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	timestamp := occurredAt.Format(time.RFC3339Nano)

	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin plugin lifecycle audit tx: %w", err)
	}
	defer tx.Rollback()

	if err := ensureWorkspaceRecord(ctx, tx, daemonPluginAuditWorkspaceID, timestamp); err != nil {
		return fmt.Errorf("ensure plugin lifecycle audit workspace: %w", err)
	}

	payload, err := pluginAuditPayload(event)
	if err != nil {
		return fmt.Errorf("marshal plugin audit payload: %w", err)
	}

	capabilitiesJSON, err := json.Marshal(event.Metadata.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal plugin capabilities: %w", err)
	}

	runtimeJSON, err := json.Marshal(event.Metadata.Runtime)
	if err != nil {
		return fmt.Errorf("marshal plugin runtime metadata: %w", err)
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal plugin metadata: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO runtime_plugins(
			workspace_id,
			plugin_id,
			kind,
			display_name,
			version,
			capabilities_json,
			runtime_json,
			status,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, plugin_id) DO UPDATE SET
			kind = excluded.kind,
			display_name = excluded.display_name,
			version = excluded.version,
			capabilities_json = excluded.capabilities_json,
			runtime_json = excluded.runtime_json,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, daemonPluginAuditWorkspaceID, pluginID, event.Kind, pluginDisplayName(event), strings.TrimSpace(event.Metadata.Version), string(capabilitiesJSON), string(runtimeJSON), event.State, timestamp, timestamp); err != nil {
		return fmt.Errorf("upsert runtime plugin %s: %w", pluginID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO runtime_plugin_processes(
			workspace_id,
			plugin_id,
			state,
			process_id,
			restart_count,
			last_error,
			last_heartbeat_at,
			last_transition_at,
			event_type,
			metadata_json,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, plugin_id) DO UPDATE SET
			state = excluded.state,
			process_id = excluded.process_id,
			restart_count = excluded.restart_count,
			last_error = excluded.last_error,
			last_heartbeat_at = excluded.last_heartbeat_at,
			last_transition_at = excluded.last_transition_at,
			event_type = excluded.event_type,
			metadata_json = excluded.metadata_json,
			updated_at = excluded.updated_at
	`, daemonPluginAuditWorkspaceID, pluginID, event.State, event.ProcessID, event.RestartCount, nullableString(event.Error), nullableTimestamp(event.LastHeartbeat), nullableTimestamp(event.LastTransition), event.EventType, string(metadataJSON), timestamp, timestamp); err != nil {
		return fmt.Errorf("upsert runtime plugin process %s: %w", pluginID, err)
	}

	auditID, err := randomAuditID("plugin_audit_")
	if err != nil {
		return fmt.Errorf("generate plugin audit id: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id,
			workspace_id,
			run_id,
			step_id,
			event_type,
			actor_id,
			acting_as_actor_id,
			correlation_id,
			payload_json,
			created_at
		) VALUES (?, ?, NULL, NULL, ?, NULL, NULL, ?, ?, ?)
	`, auditID, daemonPluginAuditWorkspaceID, event.EventType, pluginID, string(payload), timestamp)
	if err != nil {
		return fmt.Errorf("insert plugin lifecycle audit log %s: %w", pluginID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit plugin lifecycle audit tx: %w", err)
	}
	return nil
}

func pluginLifecycleAuditFailureMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) <= pluginLifecycleAuditLastErrorMaxChars {
		return message
	}
	if pluginLifecycleAuditLastErrorMaxChars <= 3 {
		return message[:pluginLifecycleAuditLastErrorMaxChars]
	}
	return strings.TrimSpace(message[:pluginLifecycleAuditLastErrorMaxChars-3]) + "..."
}

type sqlExecContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func pluginAuditPayload(event PluginLifecycleEvent) ([]byte, error) {
	return json.Marshal(map[string]any{
		"plugin_id":          event.PluginID,
		"kind":               event.Kind,
		"state":              event.State,
		"process_id":         event.ProcessID,
		"restart_count":      event.RestartCount,
		"error":              event.Error,
		"error_source":       event.ErrorSource,
		"error_operation":    event.ErrorOperation,
		"error_stderr":       event.ErrorStderr,
		"last_heartbeat_at":  nullableTimestamp(event.LastHeartbeat),
		"last_transition_at": nullableTimestamp(event.LastTransition),
		"occurred_at":        nullableTimestamp(event.OccurredAt),
		"metadata":           event.Metadata,
	})
}

func pluginDisplayName(event PluginLifecycleEvent) string {
	name := strings.TrimSpace(event.Metadata.DisplayName)
	if name == "" {
		return strings.TrimSpace(event.PluginID)
	}
	return name
}

func ensureWorkspaceRecord(ctx context.Context, exec sqlExecContext, workspaceID string, timestamp string) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, workspaceID, workspaceID, timestamp, timestamp); err != nil {
		return fmt.Errorf("ensure workspace %s: %w", workspaceID, err)
	}
	return nil
}

func nullableTimestamp(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func randomAuditID(prefix string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buffer), nil
}
