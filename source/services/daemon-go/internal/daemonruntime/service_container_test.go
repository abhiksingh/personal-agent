package daemonruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/securestore"
	shared "personalagent/runtime/internal/shared/contracts"
)

type recordingPluginSupervisor struct {
	hooks      PluginLifecycleHooks
	startCount int
	stopCount  int
	startErr   error
	stopErr    error
	workers    []PluginWorkerSpec
}

func (s *recordingPluginSupervisor) SetHooks(hooks PluginLifecycleHooks) {
	s.hooks = hooks
}

func (s *recordingPluginSupervisor) RegisterWorker(spec PluginWorkerSpec) error {
	s.workers = append(s.workers, spec)
	return nil
}

func (s *recordingPluginSupervisor) ListWorkers() []PluginWorkerStatus {
	return nil
}

func (s *recordingPluginSupervisor) WorkerStatus(_ string) (PluginWorkerStatus, bool) {
	return PluginWorkerStatus{}, false
}

func (s *recordingPluginSupervisor) RestartWorker(_ context.Context, _ string) error {
	return nil
}

func (s *recordingPluginSupervisor) StopWorker(_ context.Context, _ string) error {
	return nil
}

func (s *recordingPluginSupervisor) Start(_ context.Context) error {
	s.startCount++
	return s.startErr
}

func (s *recordingPluginSupervisor) Stop(_ context.Context) error {
	s.stopCount++
	return s.stopErr
}

func TestNewServiceContainerWiresDependencies(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := &recordingPluginSupervisor{}

	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: dbPath,
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	if container.DB == nil {
		t.Fatalf("expected db to be initialized")
	}
	if stats := container.DB.Stats(); stats.MaxOpenConnections != daemonSQLiteMaxOpenConns {
		t.Fatalf("expected MaxOpenConnections=%d, got %d", daemonSQLiteMaxOpenConns, stats.MaxOpenConnections)
	}
	if container.SecretResolver == nil {
		t.Fatalf("expected secret resolver to be initialized")
	}
	if container.ProviderConfigStore == nil {
		t.Fatalf("expected provider config store to be initialized")
	}
	if container.ModelPolicyStore == nil {
		t.Fatalf("expected model policy store to be initialized")
	}
	if container.ChannelRegistry == nil {
		t.Fatalf("expected channel registry to be initialized")
	}
	if container.ConnectorRegistry == nil {
		t.Fatalf("expected connector registry to be initialized")
	}
	if container.PluginSupervisor == nil {
		t.Fatalf("expected plugin supervisor to be initialized")
	}
	if supervisor.startCount != 1 {
		t.Fatalf("expected plugin supervisor start count 1, got %d", supervisor.startCount)
	}

	if _, err := container.DB.ExecContext(context.Background(), `SELECT COUNT(*) FROM schema_migrations`); err != nil {
		t.Fatalf("expected migrations to be applied: %v", err)
	}
}

func TestServiceContainerCloseStopsPluginSupervisor(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := &recordingPluginSupervisor{}

	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}

	if err := container.Close(context.Background()); err != nil {
		t.Fatalf("close service container: %v", err)
	}
	if supervisor.stopCount != 1 {
		t.Fatalf("expected plugin supervisor stop count 1, got %d", supervisor.stopCount)
	}
}

func TestNewServiceContainerFailsWhenPluginSupervisorStartFails(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := &recordingPluginSupervisor{startErr: context.DeadlineExceeded}

	_, err = NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
	})
	if err == nil {
		t.Fatalf("expected plugin supervisor start error")
	}
}

func TestServiceContainerWorkerLifetimeIsIndependentFromInitContext(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := NewProcessPluginSupervisor()
	spec := helperWorkerSpec(t, "stable", "plugin.channel.initctx", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)

	initCtx, cancelInit := context.WithCancel(context.Background())
	container, err := NewServiceContainer(initCtx, ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
		PluginWorkers:    []PluginWorkerSpec{spec},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning
	})

	cancelInit()
	time.Sleep(300 * time.Millisecond)

	status, ok := supervisor.WorkerStatus(spec.PluginID)
	if !ok {
		t.Fatalf("expected worker status for %s", spec.PluginID)
	}
	if status.State != PluginWorkerStateRunning {
		t.Fatalf("expected worker to remain running after init context cancellation, got %s", status.State)
	}
}

func TestServiceContainerSecretResolverResolvesSecureStoreValue(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.Put("ws1", "OPENAI_API_KEY", "sk-secret"); err != nil {
		t.Fatalf("seed secret: %v", err)
	}

	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	ref, value, err := container.SecretResolver.ResolveSecret(context.Background(), "ws1", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("resolve secret: %v", err)
	}
	if ref.WorkspaceID != "ws1" || ref.Name != "OPENAI_API_KEY" {
		t.Fatalf("unexpected secret reference: %+v", ref)
	}
	if value != "sk-secret" {
		t.Fatalf("unexpected secret value: %q", value)
	}
}

func TestResolveRuntimeDBPathUsesExplicitAndEnvInputs(t *testing.T) {
	explicitPath := filepath.Join(t.TempDir(), "explicit.db")
	resolved, err := resolveRuntimeDBPath(explicitPath)
	if err != nil {
		t.Fatalf("resolve explicit db path: %v", err)
	}
	if resolved != explicitPath {
		t.Fatalf("expected explicit db path %q, got %q", explicitPath, resolved)
	}

	envPath := filepath.Join(t.TempDir(), "env.db")
	t.Setenv("PERSONAL_AGENT_DB", envPath)
	resolved, err = resolveRuntimeDBPath("")
	if err != nil {
		t.Fatalf("resolve env db path: %v", err)
	}
	if resolved != envPath {
		t.Fatalf("expected env db path %q, got %q", envPath, resolved)
	}

	testRuntimeRoot := filepath.Join(t.TempDir(), "runtime-root")
	t.Setenv("PERSONAL_AGENT_DB", "")
	t.Setenv("PA_RUNTIME_ROOT_DIR", testRuntimeRoot)
	resolved, err = resolveRuntimeDBPath("")
	if err != nil {
		t.Fatalf("resolve runtime-root db path: %v", err)
	}
	expected := filepath.Join(testRuntimeRoot, "runtime.db")
	if resolved != expected {
		t.Fatalf("expected runtime-root db path %q, got %q", expected, resolved)
	}
}

func TestOpenRuntimeDBTightensFilesystemPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime-dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed db dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("set seed db dir permissions: %v", err)
	}

	dbPath := filepath.Join(dir, "runtime.db")
	if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
		t.Fatalf("seed runtime db file: %v", err)
	}
	if err := os.Chmod(dbPath, 0o644); err != nil {
		t.Fatalf("set seed db file permissions: %v", err)
	}

	db, err := openRuntimeDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open runtime db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if runtime.GOOS == "windows" {
		return
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat db dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected db directory permissions 0700, got %o", got)
	}
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected db file permissions 0600, got %o", got)
	}
}

func TestServiceContainerRegisterGetDeleteSecretReference(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	registered, err := container.RegisterSecretReference(context.Background(), securestore.SecretReference{
		WorkspaceID: "ws-secret",
		Name:        "OPENAI_API_KEY",
		Backend:     "memory",
		Service:     "personal-agent.ws-secret",
		Account:     "OPENAI_API_KEY",
	})
	if err != nil {
		t.Fatalf("register secret reference: %v", err)
	}
	if registered.WorkspaceID != "ws-secret" || registered.Name != "OPENAI_API_KEY" {
		t.Fatalf("unexpected registered reference: %+v", registered)
	}

	loaded, err := container.GetSecretReference(context.Background(), "ws-secret", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("get secret reference: %v", err)
	}
	if loaded.Service != "personal-agent.ws-secret" || loaded.Account != "OPENAI_API_KEY" {
		t.Fatalf("unexpected loaded reference: %+v", loaded)
	}

	deleted, err := container.DeleteSecretReference(context.Background(), "ws-secret", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("delete secret reference: %v", err)
	}
	if deleted.Name != "OPENAI_API_KEY" {
		t.Fatalf("unexpected deleted reference: %+v", deleted)
	}

	_, err = container.GetSecretReference(context.Background(), "ws-secret", "OPENAI_API_KEY")
	if !errors.Is(err, ErrSecretReferenceNotFound) {
		t.Fatalf("expected ErrSecretReferenceNotFound, got %v", err)
	}
}

func TestServiceContainerSecretReferenceDoesNotFallbackToLegacyWorkspaceRows(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})

	if _, err := container.DB.ExecContext(context.Background(), `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-26T00:00:00Z', '2026-02-26T00:00:00Z')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		t.Fatalf("seed legacy workspace: %v", err)
	}
	if _, err := container.DB.ExecContext(context.Background(), `
		INSERT INTO secret_refs(
			id, workspace_id, owner_type, owner_id, keychain_account, keychain_service, created_at
		) VALUES (
			'sref.default.openai',
			'default',
			'CLI_SECRET',
			'OPENAI_API_KEY',
			'OPENAI_API_KEY',
			'personal-agent.default',
			'2026-02-26T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed legacy secret ref: %v", err)
	}

	_, err = container.GetSecretReference(context.Background(), "ws1", "OPENAI_API_KEY")
	if !errors.Is(err, ErrSecretReferenceNotFound) {
		t.Fatalf("expected ErrSecretReferenceNotFound when only default workspace row exists, got %v", err)
	}

	if _, err := container.DeleteSecretReference(context.Background(), "ws1", "OPENAI_API_KEY"); !errors.Is(err, ErrSecretReferenceNotFound) {
		t.Fatalf("expected delete to fail with ErrSecretReferenceNotFound, got %v", err)
	}
}

func TestServiceContainerPersistsPluginLifecycleAuditEvents(t *testing.T) {
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := NewProcessPluginSupervisor()
	spec := helperWorkerSpec(t, "stable", "plugin.channel.audit", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)

	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
		PluginWorkers:    []PluginWorkerSpec{spec},
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}

	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning
	})
	waitUntil := func(timeout time.Duration, description string, check func() (bool, error)) {
		t.Helper()
		deadline := time.Now().Add(timeout)
		var lastErr error
		for time.Now().Before(deadline) {
			done, err := check()
			if err == nil && done {
				return
			}
			if err != nil {
				lastErr = err
			} else {
				lastErr = errors.New("condition not met")
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("%s: %v", description, lastErr)
	}

	var (
		pluginKind       string
		pluginDisplay    string
		pluginState      string
		capabilitiesJSON string
	)
	waitUntil(3*time.Second, "load runtime plugin record", func() (bool, error) {
		if err := container.DB.QueryRowContext(context.Background(), `
			SELECT kind, display_name, status, capabilities_json
			FROM runtime_plugins
			WHERE workspace_id = ?
			  AND plugin_id = ?
		`, daemonPluginAuditWorkspaceID, spec.PluginID).Scan(&pluginKind, &pluginDisplay, &pluginState, &capabilitiesJSON); err != nil {
			return false, err
		}
		return pluginState == string(PluginWorkerStateRunning), nil
	})
	if pluginKind != string(shared.AdapterKindChannel) {
		t.Fatalf("expected runtime plugin kind %q, got %q", shared.AdapterKindChannel, pluginKind)
	}
	if pluginDisplay != spec.PluginID {
		t.Fatalf("expected runtime plugin display name %q, got %q", spec.PluginID, pluginDisplay)
	}
	if pluginState != string(PluginWorkerStateRunning) {
		t.Fatalf("expected runtime plugin state %q, got %q", PluginWorkerStateRunning, pluginState)
	}
	if !strings.Contains(capabilitiesJSON, "channel.sms.send") {
		t.Fatalf("expected runtime plugin capabilities to include channel.sms.send, got %s", capabilitiesJSON)
	}

	var (
		processState  string
		processID     int
		restartCount  int
		lastEventType string
	)
	waitUntil(3*time.Second, "load runtime plugin process record", func() (bool, error) {
		if err := container.DB.QueryRowContext(context.Background(), `
			SELECT state, process_id, restart_count, event_type
			FROM runtime_plugin_processes
			WHERE workspace_id = ?
			  AND plugin_id = ?
		`, daemonPluginAuditWorkspaceID, spec.PluginID).Scan(&processState, &processID, &restartCount, &lastEventType); err != nil {
			return false, err
		}
		return processState == string(PluginWorkerStateRunning) && lastEventType == pluginEventHandshakeAccepted, nil
	})
	if processState != string(PluginWorkerStateRunning) {
		t.Fatalf("expected runtime plugin process state %q, got %q", PluginWorkerStateRunning, processState)
	}
	if processID <= 0 {
		t.Fatalf("expected runtime plugin process id to be set, got %d", processID)
	}
	if restartCount != 0 {
		t.Fatalf("expected runtime plugin restart_count=0, got %d", restartCount)
	}
	if lastEventType != pluginEventHandshakeAccepted {
		t.Fatalf("expected runtime plugin process event_type %q, got %q", pluginEventHandshakeAccepted, lastEventType)
	}

	var startedCount int
	waitUntil(3*time.Second, "count plugin start audit logs", func() (bool, error) {
		if err := container.DB.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM audit_log_entries
			WHERE workspace_id = ?
			  AND event_type = ?
		`, daemonPluginAuditWorkspaceID, pluginEventWorkerStarted).Scan(&startedCount); err != nil {
			return false, err
		}
		return startedCount > 0, nil
	})
	if startedCount == 0 {
		t.Fatalf("expected plugin start audit logs")
	}

	var handshakeCount int
	waitUntil(3*time.Second, "count plugin handshake audit logs", func() (bool, error) {
		if err := container.DB.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM audit_log_entries
			WHERE workspace_id = ?
			  AND event_type = ?
		`, daemonPluginAuditWorkspaceID, pluginEventHandshakeAccepted).Scan(&handshakeCount); err != nil {
			return false, err
		}
		return handshakeCount > 0, nil
	})
	if handshakeCount == 0 {
		t.Fatalf("expected plugin handshake audit logs")
	}

	if err := supervisor.StopWorker(context.Background(), spec.PluginID); err != nil {
		t.Fatalf("stop plugin worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateStopped
	})

	waitUntil(3*time.Second, "load runtime plugin process record after stop", func() (bool, error) {
		if err := container.DB.QueryRowContext(context.Background(), `
			SELECT state, process_id, event_type
			FROM runtime_plugin_processes
			WHERE workspace_id = ?
			  AND plugin_id = ?
		`, daemonPluginAuditWorkspaceID, spec.PluginID).Scan(&processState, &processID, &lastEventType); err != nil {
			return false, err
		}
		return processState == string(PluginWorkerStateStopped) && processID == 0 && lastEventType == pluginEventWorkerStopped, nil
	})
	if processState != string(PluginWorkerStateStopped) {
		t.Fatalf("expected runtime plugin process state %q after stop, got %q", PluginWorkerStateStopped, processState)
	}
	if processID != 0 {
		t.Fatalf("expected runtime plugin process id reset to 0 after stop, got %d", processID)
	}
	if lastEventType != pluginEventWorkerStopped {
		t.Fatalf("expected runtime plugin process event_type %q after stop, got %q", pluginEventWorkerStopped, lastEventType)
	}

	if err := container.Close(context.Background()); err != nil {
		t.Fatalf("close service container: %v", err)
	}
}
