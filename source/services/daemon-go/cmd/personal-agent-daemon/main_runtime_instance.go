package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"personalagent/runtime/internal/daemonruntime"
	"personalagent/runtime/internal/transport"
)

type daemonRuntimeBundle struct {
	container             *daemonruntime.ServiceContainer
	server                *transport.Server
	lifecycleService      *daemonruntime.DaemonLifecycleService
	automationRuntime     *daemonruntime.AutomationRuntime
	queuedTaskRuntime     *daemonruntime.QueuedTaskRuntime
	inboundWatcherRuntime *daemonruntime.InboundWatcherRuntime
}

func runDaemonInstance(
	config daemonRunConfig,
	executable string,
	actionRequests chan daemonLifecycleAction,
	signals <-chan os.Signal,
) (bool, error) {
	webSocketOrigins, err := parseDaemonWebSocketOriginAllowlist(config.webSocketOriginAllowlist)
	if err != nil {
		return false, fmt.Errorf("invalid websocket origin allowlist: %w", err)
	}

	drainPendingDaemonLifecycleActions(actionRequests)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startupCancel()

	runtimeBundle, err := newDaemonRuntimeBundle(startupCtx, config, executable, actionRequests, webSocketOrigins)
	if err != nil {
		return false, err
	}

	if err := startDaemonLifecyclePhases(startupCtx, runtimeBundle.startupPhases()); err != nil {
		_ = runtimeBundle.container.Close(context.Background())
		return false, fmt.Errorf("start daemon lifecycle phases: %w", err)
	}

	fmt.Fprintf(
		os.Stdout,
		"personal-agent-daemon transport listening (mode=%s address=%s db=%s)\n",
		config.listenerMode,
		runtimeBundle.server.Address(),
		runtimeBundle.container.DBPath,
	)

	action, stopSignal := waitForDaemonLifecycleAction(actionRequests, signals)
	if stopSignal != nil {
		fmt.Fprintf(
			os.Stdout,
			"daemon shutdown requested via signal=%s; draining runtime phases\n",
			stopSignal.String(),
		)
	} else {
		fmt.Fprintf(
			os.Stdout,
			"daemon shutdown requested via lifecycle_action=%s; draining runtime phases\n",
			action,
		)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := stopDaemonLifecyclePhases(shutdownCtx, runtimeBundle.shutdownPhases()); err != nil {
		return false, fmt.Errorf("daemon shutdown error: %w", err)
	}
	return action == daemonLifecycleActionRestart, nil
}

func newDaemonRuntimeBundle(
	startupCtx context.Context,
	config daemonRunConfig,
	executable string,
	actionRequests chan daemonLifecycleAction,
	webSocketOrigins []string,
) (*daemonRuntimeBundle, error) {
	pluginWorkers, err := loadDaemonPluginWorkers(executable, config.dbPath, config.pluginWorkersManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load plugin workers: %w", err)
	}

	container, err := daemonruntime.NewServiceContainer(startupCtx, daemonruntime.ServiceContainerConfig{
		DBPath:        config.dbPath,
		PluginWorkers: pluginWorkers,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize daemon service container: %w", err)
	}
	closeContainer := func() {
		_ = container.Close(context.Background())
	}

	broker := transport.NewEventBroker()
	providerModelChat := daemonruntime.NewProviderModelChatService(container)
	agentDelegation, err := daemonruntime.NewAgentDelegationService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon agent/delegation services: %w", err)
	}
	backend, err := daemonruntime.NewPersistedControlBackend(container, agentDelegation, broker)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon control backend: %w", err)
	}
	commTwilio, err := daemonruntime.NewCommTwilioService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon comm/twilio services: %w", err)
	}
	cloudflaredConnector, err := daemonruntime.NewCloudflaredConnectorService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon cloudflared connector service: %w", err)
	}
	agentDelegation.SetCommService(commTwilio)
	opsService, err := daemonruntime.NewAutomationInspectRetentionContextService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon automation/inspect/retention/context services: %w", err)
	}
	uiStatusService, err := daemonruntime.NewUIStatusService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon ui status service: %w", err)
	}
	unifiedTurnService, err := daemonruntime.NewUnifiedTurnService(
		container,
		providerModelChat,
		agentDelegation,
		uiStatusService,
		agentDelegation,
	)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon unified turn service: %w", err)
	}
	commTwilio.SetAssistantChatService(unifiedTurnService)
	identityDirectoryService, err := daemonruntime.NewIdentityDirectoryService(container)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon identity directory service: %w", err)
	}
	lifecycleHooks, err := buildDaemonLifecycleSetupHooks(config, executable, container.DBPath)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon lifecycle host-operation hooks: %w", err)
	}
	lifecycleService, err := daemonruntime.NewDaemonLifecycleService(daemonruntime.DaemonLifecycleServiceConfig{
		Container:         container,
		RuntimeMode:       config.listenerMode,
		ConfiguredAddress: config.listenAddress,
		ExecutablePath:    executable,
		AuthToken:         config.authToken,
		AuthTokenSource:   config.authTokenSource,
		RequestStop:       requestDaemonLifecycleAction(actionRequests, daemonLifecycleActionStop),
		RequestRestart:    requestDaemonLifecycleAction(actionRequests, daemonLifecycleActionRestart),
		RequestInstall:    lifecycleHooks.install,
		RequestUninstall:  lifecycleHooks.uninstall,
		RequestRepair:     lifecycleHooks.repair,
	})
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon lifecycle service: %w", err)
	}
	tlsConfig, err := buildDaemonTLSServerConfig(config)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon tls config: %w", err)
	}

	server, err := transport.NewServer(transport.ServerConfig{
		ListenerMode:             transport.ListenerMode(config.listenerMode),
		Address:                  config.listenAddress,
		RuntimeProfile:           config.runtimeProfile,
		AuthToken:                config.authToken,
		AuthTokenScopes:          append([]string(nil), config.authTokenScopes...),
		WebSocketOriginAllowlist: webSocketOrigins,
		TLSConfig:                tlsConfig,
		ReadHeaderTimeout:        config.readHeaderTimeout,
		ReadTimeout:              config.readTimeout,
		WriteTimeout:             config.writeTimeout,
		IdleTimeout:              config.idleTimeout,
		MaxHeaderBytes:           config.maxHeaderBytes,
		RealtimeMaxConnections:   config.realtimeMaxConnections,
		RealtimeMaxSubscriptions: config.realtimeMaxSubscriptions,
		DaemonLifecycle:          lifecycleService,
		WorkflowQueries:          agentDelegation,
		SecretReferences: &daemonSecretReferenceService{
			container: container,
		},
		Providers:         providerModelChat,
		Models:            providerModelChat,
		Chat:              unifiedTurnService,
		Agent:             agentDelegation,
		Delegation:        agentDelegation,
		Comm:              commTwilio,
		Twilio:            commTwilio,
		Cloudflared:       cloudflaredConnector,
		Automation:        opsService,
		Inspect:           opsService,
		Retention:         opsService,
		ContextOps:        opsService,
		UIStatus:          uiStatusService,
		IdentityDirectory: identityDirectoryService,
	}, backend, broker)
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon transport: %w", err)
	}

	automationRuntime, err := daemonruntime.NewAutomationRuntime(container.DB, daemonruntime.AutomationRuntimeOptions{})
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon automation runtime: %w", err)
	}
	queuedTaskRuntime, err := daemonruntime.NewQueuedTaskRuntime(container.DB, agentDelegation, daemonruntime.QueuedTaskRuntimeOptions{
		EventBroker: broker,
	})
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon queued-task runtime: %w", err)
	}
	backend.SetQueuedTaskRunCanceller(queuedTaskRuntime)
	inboundWatcherRuntime, err := daemonruntime.NewInboundWatcherRuntime(container.DB, commTwilio, daemonruntime.InboundWatcherRuntimeOptions{
		ResolveWorkspaceID: func(ctx context.Context) (string, error) {
			response, err := identityDirectoryService.ListWorkspaces(ctx, transport.IdentityWorkspacesRequest{
				IncludeInactive: true,
			})
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(response.ActiveContext.WorkspaceID), nil
		},
	})
	if err != nil {
		closeContainer()
		return nil, fmt.Errorf("initialize daemon inbound watcher runtime: %w", err)
	}
	commTwilio.SetAutomationCommEventEvaluator(automationRuntime)

	return &daemonRuntimeBundle{
		container:             container,
		server:                server,
		lifecycleService:      lifecycleService,
		automationRuntime:     automationRuntime,
		queuedTaskRuntime:     queuedTaskRuntime,
		inboundWatcherRuntime: inboundWatcherRuntime,
	}, nil
}

func (bundle *daemonRuntimeBundle) startupPhases() []daemonLifecyclePhase {
	return []daemonLifecyclePhase{
		{
			name: "transport_control_plane",
			start: func(ctx context.Context) error {
				if err := bundle.server.Start(); err != nil {
					return err
				}
				bundle.lifecycleService.SetBoundAddress(bundle.server.Address())
				return nil
			},
			stop: bundle.server.Close,
		},
		{
			name:  "automation_runtime",
			start: bundle.automationRuntime.Start,
			stop:  bundle.automationRuntime.Stop,
		},
		{
			name:  "queued_task_runtime",
			start: bundle.queuedTaskRuntime.Start,
			stop:  bundle.queuedTaskRuntime.Stop,
		},
		{
			name:  "inbound_watcher_runtime",
			start: bundle.inboundWatcherRuntime.Start,
			stop:  bundle.inboundWatcherRuntime.Stop,
		},
	}
}

func (bundle *daemonRuntimeBundle) shutdownPhases() []daemonLifecyclePhase {
	return []daemonLifecyclePhase{
		{
			name: "inbound_watcher_runtime",
			stop: bundle.inboundWatcherRuntime.Stop,
		},
		{
			name: "queued_task_runtime",
			stop: bundle.queuedTaskRuntime.Stop,
		},
		{
			name: "automation_runtime",
			stop: bundle.automationRuntime.Stop,
		},
		{
			name: "transport_control_plane",
			stop: bundle.server.Close,
		},
		{
			name: "service_container",
			stop: bundle.container.Close,
		},
	}
}
