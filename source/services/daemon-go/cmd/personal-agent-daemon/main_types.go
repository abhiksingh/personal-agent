package main

import "time"

type daemonLifecycleAction string

const (
	daemonLifecycleActionStop    daemonLifecycleAction = "stop"
	daemonLifecycleActionRestart daemonLifecycleAction = "restart"

	defaultDaemonAuthToken    = ""
	daemonRuntimeProfileLocal = "local"
	daemonRuntimeProfileProd  = "prod"

	daemonAuthTokenSourceFlag = "auth_token_flag"
	daemonAuthTokenSourceFile = "auth_token_file"
)

type daemonRunConfig struct {
	listenerMode              string
	listenAddress             string
	dbPath                    string
	pluginWorkersManifestPath string
	webSocketOriginAllowlist  string
	authToken                 string
	authTokenScopes           []string
	authTokenFile             string
	authTokenSource           string
	runtimeProfile            string
	allowNonLocal             bool
	lifecycleHostOpsMode      string
	readHeaderTimeout         time.Duration
	readTimeout               time.Duration
	writeTimeout              time.Duration
	idleTimeout               time.Duration
	maxHeaderBytes            int
	realtimeMaxConnections    int
	realtimeMaxSubscriptions  int
	tlsCertFile               string
	tlsKeyFile                string
	tlsClientCAFile           string
	tlsRequireClientCert      bool
}

type daemonCLIOptions struct {
	listenerMode              string
	listenAddress             string
	allowNonLocalBind         bool
	dbPath                    string
	runtimeProfile            string
	authToken                 string
	authTokenScopes           string
	authTokenFile             string
	webSocketOriginAllowlist  string
	lifecycleHostOpsMode      string
	readHeaderTimeout         time.Duration
	readTimeout               time.Duration
	writeTimeout              time.Duration
	idleTimeout               time.Duration
	maxHeaderBytes            int
	realtimeMaxConnections    int
	realtimeMaxSubscriptions  int
	pluginWorkersManifestPath string
	tlsCertFile               string
	tlsKeyFile                string
	tlsClientCAFile           string
	tlsRequireClientCert      bool
	connectorWorker           string
	channelWorker             string
	pluginID                  string
	workerHealthInterval      time.Duration
}
