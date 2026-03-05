package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	channelregistry "personalagent/runtime/internal/channels/registry"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/filesecurity"
	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/persistence/migrator"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/runtimepaths"
	"personalagent/runtime/internal/securestore"
	shared "personalagent/runtime/internal/shared/contracts"

	_ "modernc.org/sqlite"
)

type SecretReferenceResolver interface {
	ResolveSecret(ctx context.Context, workspaceID string, name string) (securestore.SecretReference, string, error)
}

type PluginRestartPolicy struct {
	MaxRestarts int
	Delay       time.Duration
}

type PluginWorkerSpec struct {
	PluginID         string
	Kind             shared.AdapterKind
	Command          string
	Args             []string
	Env              []string
	WorkingDirectory string
	HandshakeTimeout time.Duration
	HealthInterval   time.Duration
	HealthTimeout    time.Duration
	RestartPolicy    PluginRestartPolicy
}

type PluginWorkerState string

const (
	PluginWorkerStateRegistered PluginWorkerState = "registered"
	PluginWorkerStateStarting   PluginWorkerState = "starting"
	PluginWorkerStateRunning    PluginWorkerState = "running"
	PluginWorkerStateRestarting PluginWorkerState = "restarting"
	PluginWorkerStateStopped    PluginWorkerState = "stopped"
	PluginWorkerStateFailed     PluginWorkerState = "failed"
)

type PluginWorkerStatus struct {
	PluginID           string
	Kind               shared.AdapterKind
	State              PluginWorkerState
	ProcessID          int
	RestartCount       int
	LastError          string
	LastErrorSource    string
	LastErrorOperation string
	LastErrorStderr    string
	LastHeartbeat      time.Time
	LastTransition     time.Time
	Metadata           shared.AdapterMetadata
	execAuthToken      string
}

type PluginLifecycleEvent struct {
	PluginID       string
	Kind           shared.AdapterKind
	State          PluginWorkerState
	EventType      string
	ProcessID      int
	RestartCount   int
	Error          string
	ErrorSource    string
	ErrorOperation string
	ErrorStderr    string
	OccurredAt     time.Time
	LastHeartbeat  time.Time
	LastTransition time.Time
	Metadata       shared.AdapterMetadata
}

type PluginLifecycleHooks struct {
	OnWorkerStart func(pluginID string, processID int)
	OnWorkerExit  func(pluginID string, processID int, err error)
	OnEvent       func(event PluginLifecycleEvent)
}

type PluginSupervisor interface {
	SetHooks(hooks PluginLifecycleHooks)
	RegisterWorker(spec PluginWorkerSpec) error
	ListWorkers() []PluginWorkerStatus
	WorkerStatus(pluginID string) (PluginWorkerStatus, bool)
	RestartWorker(ctx context.Context, pluginID string) error
	StopWorker(ctx context.Context, pluginID string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type NoopPluginSupervisor struct {
	hooks PluginLifecycleHooks
}

func NewNoopPluginSupervisor() *NoopPluginSupervisor {
	return &NoopPluginSupervisor{}
}

func (s *NoopPluginSupervisor) SetHooks(hooks PluginLifecycleHooks) {
	s.hooks = hooks
}

func (s *NoopPluginSupervisor) RegisterWorker(_ PluginWorkerSpec) error {
	return nil
}

func (s *NoopPluginSupervisor) ListWorkers() []PluginWorkerStatus {
	return nil
}

func (s *NoopPluginSupervisor) WorkerStatus(_ string) (PluginWorkerStatus, bool) {
	return PluginWorkerStatus{}, false
}

func (s *NoopPluginSupervisor) RestartWorker(_ context.Context, _ string) error {
	return nil
}

func (s *NoopPluginSupervisor) StopWorker(_ context.Context, _ string) error {
	return nil
}

func (s *NoopPluginSupervisor) Start(_ context.Context) error {
	return nil
}

func (s *NoopPluginSupervisor) Stop(_ context.Context) error {
	return nil
}

type managerSecretResolver struct {
	manager *securestore.Manager
}

func (r *managerSecretResolver) ResolveSecret(_ context.Context, workspaceID string, name string) (securestore.SecretReference, string, error) {
	return r.manager.Get(workspaceID, name)
}

type ServiceContainer struct {
	DB                   *sql.DB
	DBPath               string
	SecretResolver       SecretReferenceResolver
	ProviderConfigStore  *providerconfig.SQLiteStore
	ModelPolicyStore     *modelpolicy.SQLiteStore
	ChannelRegistry      *channelregistry.Registry
	ConnectorRegistry    *connectorregistry.Registry
	PluginSupervisor     PluginSupervisor
	pluginLifecycleAudit *pluginLifecycleAuditRuntime
}

type ServiceContainerConfig struct {
	DBPath               string
	SecretManagerFactory func() (*securestore.Manager, error)
	PluginSupervisor     PluginSupervisor
	PluginWorkers        []PluginWorkerSpec
}

const (
	daemonSQLiteMaxOpenConns = 1
	daemonSQLiteMaxIdleConns = 1
)

func NewServiceContainer(ctx context.Context, config ServiceContainerConfig) (*ServiceContainer, error) {
	if config.SecretManagerFactory == nil {
		config.SecretManagerFactory = securestore.NewDefaultManager
	}
	if config.PluginSupervisor == nil {
		config.PluginSupervisor = NewProcessPluginSupervisor()
	}

	manager, err := config.SecretManagerFactory()
	if err != nil {
		return nil, fmt.Errorf("create secure store manager: %w", err)
	}

	dbPath, err := resolveRuntimeDBPath(config.DBPath)
	if err != nil {
		return nil, err
	}
	db, err := openRuntimeDB(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	container := &ServiceContainer{
		DB:                  db,
		DBPath:              dbPath,
		SecretResolver:      &managerSecretResolver{manager: manager},
		ProviderConfigStore: providerconfig.NewSQLiteStore(db),
		ModelPolicyStore:    modelpolicy.NewSQLiteStore(db),
		ChannelRegistry:     channelregistry.New(),
		ConnectorRegistry:   connectorregistry.New(),
		PluginSupervisor:    config.PluginSupervisor,
	}
	container.startPluginLifecycleAuditRuntime()
	container.PluginSupervisor.SetHooks(container.pluginLifecycleHooks())
	for _, worker := range config.PluginWorkers {
		if err := container.PluginSupervisor.RegisterWorker(worker); err != nil {
			container.stopPluginLifecycleAuditRuntime()
			_ = db.Close()
			return nil, fmt.Errorf("register plugin worker %s: %w", worker.PluginID, err)
		}
	}

	// Worker runtime lifetime is owned by ServiceContainer.Close, not by the
	// initialization context. Using the startup context here can stop all
	// workers when a startup timeout/cancel fires after successful boot.
	if err := container.PluginSupervisor.Start(context.Background()); err != nil {
		container.stopPluginLifecycleAuditRuntime()
		_ = db.Close()
		return nil, fmt.Errorf("start plugin supervisor: %w", err)
	}

	return container, nil
}

func (c *ServiceContainer) Close(ctx context.Context) error {
	var errs []string
	if c.PluginSupervisor != nil {
		if err := c.PluginSupervisor.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("stop plugin supervisor: %v", err))
		}
	}
	c.stopPluginLifecycleAuditRuntime()
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close db: %v", err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func resolveRuntimeDBPath(explicitPath string) (string, error) {
	trimmed := strings.TrimSpace(explicitPath)
	if trimmed != "" {
		return trimmed, nil
	}

	envPath := strings.TrimSpace(os.Getenv("PERSONAL_AGENT_DB"))
	if envPath != "" {
		return envPath, nil
	}

	defaultPath, err := runtimepaths.DefaultDBPath()
	if err != nil {
		return "", err
	}
	return defaultPath, nil
}

func openRuntimeDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("db path is required")
	}
	if err := filesecurity.EnsurePrivateDir(filepath.Dir(dbPath)); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	configureSingleWriterSQLite(db)
	if _, err := migrator.Apply(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	if err := filesecurity.EnsurePrivateFile(dbPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("harden db file permissions: %w", err)
	}
	return db, nil
}

func configureSingleWriterSQLite(db *sql.DB) {
	// MVP invariant: daemon persistence runs through a single SQLite connection.
	db.SetMaxOpenConns(daemonSQLiteMaxOpenConns)
	db.SetMaxIdleConns(daemonSQLiteMaxIdleConns)
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
}
