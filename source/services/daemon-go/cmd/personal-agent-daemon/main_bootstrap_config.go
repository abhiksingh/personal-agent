package main

import (
	"flag"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/controlauth"
	"personalagent/runtime/internal/transport"
)

func parseDaemonCLIOptions(args []string, stderr io.Writer) (daemonCLIOptions, error) {
	flags := flag.NewFlagSet("personal-agent-daemon", flag.ContinueOnError)
	flags.SetOutput(stderr)

	listenerMode := flags.String("listen-mode", "tcp", "transport listener mode: tcp|unix|named_pipe")
	listenAddress := flags.String("listen-address", transport.DefaultTCPAddress, "listener address (host:port for tcp, path for unix, pipe path/name for named_pipe)")
	allowNonLocalBind := flags.Bool("allow-non-local-bind", false, "allow daemon control API bind on non-loopback TCP interfaces")
	dbPath := flags.String("db", "", "runtime sqlite path (defaults to $PERSONAL_AGENT_DB or user config path)")
	runtimeProfile := flags.String("runtime-profile", daemonRuntimeProfileLocal, "daemon runtime profile: local|prod")
	authToken := flags.String("auth-token", defaultDaemonAuthToken, "bearer auth token required for control and realtime endpoints")
	authTokenScopes := flags.String("auth-token-scopes", "", "comma-separated bearer authorization scopes for route-level policy (prod requires explicit scopes; local defaults to wildcard '*' when omitted)")
	authTokenFile := flags.String("auth-token-file", "", "path to file containing bearer auth token")
	webSocketOriginAllowlist := flags.String("ws-origin-allowlist", "", "comma-separated allowed browser Origin values for realtime websocket upgrades")
	lifecycleHostOpsMode := flags.String("lifecycle-host-ops-mode", string(daemonLifecycleHostOpsModeUnsupported), "daemon lifecycle setup-action host-ops mode: unsupported|dry-run|apply")
	readHeaderTimeout := flags.Duration("read-header-timeout", 0, "HTTP server read-header timeout override (0 uses transport default)")
	readTimeout := flags.Duration("read-timeout", 0, "HTTP server read timeout override (0 uses transport default)")
	writeTimeout := flags.Duration("write-timeout", 0, "HTTP server write timeout override (0 uses transport default)")
	idleTimeout := flags.Duration("idle-timeout", 0, "HTTP server idle timeout override (0 uses transport default)")
	maxHeaderBytes := flags.Int("max-header-bytes", 0, "HTTP server max header bytes override (0 uses transport default)")
	realtimeMaxConnections := flags.Int("realtime-max-connections", 0, "max concurrent realtime websocket connections (0 uses transport default)")
	realtimeMaxSubscriptions := flags.Int("realtime-max-subscriptions", 0, "max concurrent realtime websocket subscriptions (0 uses transport default)")
	pluginWorkersManifestPath := flags.String("plugin-workers-manifest", "", "path to plugin worker manifest JSON (defaults to embedded manifest; env override PA_DAEMON_PLUGIN_WORKERS_MANIFEST)")
	tlsCertFile := flags.String("tls-cert-file", "", "server TLS certificate PEM file for secure TCP mode")
	tlsKeyFile := flags.String("tls-key-file", "", "server TLS private key PEM file for secure TCP mode")
	tlsClientCAFile := flags.String("tls-client-ca-file", "", "client CA PEM bundle for optional mTLS verification")
	tlsRequireClientCert := flags.Bool("tls-require-client-cert", false, "require verified client certificate (mTLS)")
	connectorWorker := flags.String("connector-worker", "", "internal connector worker mode: messages|mail|calendar|browser|finder|cloudflared|twilio")
	channelWorker := flags.String("channel-worker", "", "internal channel worker mode: app_chat")
	pluginID := flags.String("plugin-id", "", "plugin id for connector/channel worker mode")
	workerHealthInterval := flags.Duration("worker-health-interval", 250*time.Millisecond, "worker heartbeat interval")

	if err := flags.Parse(args); err != nil {
		return daemonCLIOptions{}, err
	}

	return daemonCLIOptions{
		listenerMode:              *listenerMode,
		listenAddress:             *listenAddress,
		allowNonLocalBind:         *allowNonLocalBind,
		dbPath:                    *dbPath,
		runtimeProfile:            *runtimeProfile,
		authToken:                 *authToken,
		authTokenScopes:           *authTokenScopes,
		authTokenFile:             *authTokenFile,
		webSocketOriginAllowlist:  *webSocketOriginAllowlist,
		lifecycleHostOpsMode:      *lifecycleHostOpsMode,
		readHeaderTimeout:         *readHeaderTimeout,
		readTimeout:               *readTimeout,
		writeTimeout:              *writeTimeout,
		idleTimeout:               *idleTimeout,
		maxHeaderBytes:            *maxHeaderBytes,
		realtimeMaxConnections:    *realtimeMaxConnections,
		realtimeMaxSubscriptions:  *realtimeMaxSubscriptions,
		pluginWorkersManifestPath: *pluginWorkersManifestPath,
		tlsCertFile:               *tlsCertFile,
		tlsKeyFile:                *tlsKeyFile,
		tlsClientCAFile:           *tlsClientCAFile,
		tlsRequireClientCert:      *tlsRequireClientCert,
		connectorWorker:           *connectorWorker,
		channelWorker:             *channelWorker,
		pluginID:                  *pluginID,
		workerHealthInterval:      *workerHealthInterval,
	}, nil
}

func buildDaemonRunConfig(options daemonCLIOptions) (daemonRunConfig, []string, error) {
	resolvedAuthToken, err := controlauth.ResolveToken(options.authToken, options.authTokenFile)
	if err != nil {
		return daemonRunConfig{}, nil, err
	}
	resolvedRuntimeProfile, err := resolveDaemonRuntimeProfile(options.runtimeProfile)
	if err != nil {
		return daemonRunConfig{}, nil, err
	}
	authTokenSource := daemonAuthTokenSourceFlag
	if strings.TrimSpace(options.authTokenFile) != "" {
		authTokenSource = daemonAuthTokenSourceFile
	}

	config := daemonRunConfig{
		listenerMode:              options.listenerMode,
		listenAddress:             options.listenAddress,
		dbPath:                    options.dbPath,
		pluginWorkersManifestPath: resolveDaemonPluginWorkersManifestPath(options.pluginWorkersManifestPath),
		webSocketOriginAllowlist:  options.webSocketOriginAllowlist,
		authToken:                 resolvedAuthToken,
		authTokenScopes:           parseDaemonAuthTokenScopes(options.authTokenScopes),
		authTokenFile:             strings.TrimSpace(options.authTokenFile),
		authTokenSource:           authTokenSource,
		runtimeProfile:            resolvedRuntimeProfile,
		allowNonLocal:             options.allowNonLocalBind,
		lifecycleHostOpsMode:      options.lifecycleHostOpsMode,
		readHeaderTimeout:         options.readHeaderTimeout,
		readTimeout:               options.readTimeout,
		writeTimeout:              options.writeTimeout,
		idleTimeout:               options.idleTimeout,
		maxHeaderBytes:            options.maxHeaderBytes,
		realtimeMaxConnections:    options.realtimeMaxConnections,
		realtimeMaxSubscriptions:  options.realtimeMaxSubscriptions,
		tlsCertFile:               options.tlsCertFile,
		tlsKeyFile:                options.tlsKeyFile,
		tlsClientCAFile:           options.tlsClientCAFile,
		tlsRequireClientCert:      options.tlsRequireClientCert,
	}
	if err := validateDaemonRunConfig(config); err != nil {
		return daemonRunConfig{}, nil, err
	}
	return config, daemonAuthScopeWarnings(config), nil
}
