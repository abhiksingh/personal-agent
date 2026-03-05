package cliapp

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"personalagent/runtime/internal/controlauth"
	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"
)

var newSecretManager = securestore.NewDefaultManager
var chatInput io.Reader = os.Stdin

const (
	defaultCLIControlAuthToken = ""
	cliRuntimeProfileLocal     = "local"
	cliRuntimeProfileProd      = "prod"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	return run(args, stdout, stderr)
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	root := flag.NewFlagSet("personal-agent", flag.ContinueOnError)
	root.SetOutput(stderr)

	listenerMode := root.String("mode", "tcp", "transport mode: tcp|unix|named_pipe")
	address := root.String("address", transport.DefaultTCPAddress, "transport address (host:port for tcp, path for unix, pipe path/name for named_pipe)")
	runtimeProfile := root.String("runtime-profile", cliRuntimeProfileLocal, "runtime profile: local|prod")
	authToken := root.String("auth-token", defaultCLIControlAuthToken, "bearer auth token")
	authTokenFile := root.String("auth-token-file", "", "path to file containing bearer auth token")
	dbPath := root.String("db", "", "SQLite path for local CLI state (used by local webhook/ingest utilities)")
	correlationID := root.String("correlation-id", "", "optional correlation id")
	outputMode := root.String("output", string(cliOutputModeJSON), "output mode: json|json-compact|text")
	errorOutputMode := root.String("error-output", string(cliErrorOutputModeText), "error output mode: text|json")
	timeout := root.Duration("timeout", 10*time.Second, "request timeout")
	tlsCAFile := root.String("tls-ca-file", "", "CA bundle PEM for daemon TLS certificate verification")
	tlsClientCertFile := root.String("tls-client-cert-file", "", "client certificate PEM for daemon mTLS auth")
	tlsClientKeyFile := root.String("tls-client-key-file", "", "client private key PEM for daemon mTLS auth")
	tlsServerName := root.String("tls-server-name", "", "TLS server-name override for daemon certificate verification")
	tlsInsecureSkipVerify := root.Bool("tls-insecure-skip-verify", false, "skip daemon TLS certificate verification (testing only)")

	if err := root.Parse(args); err != nil {
		printUsage(stderr)
		return 2
	}

	rest := root.Args()
	if len(rest) == 0 {
		printUsage(stderr)
		return 2
	}

	outputConfig, err := resolveCLIOutputConfig(*outputMode, *errorOutputMode)
	if err != nil {
		fmt.Fprintf(stderr, "invalid CLI configuration: %v\n", err)
		return 2
	}
	restoreOutputConfig := setCLIOutputConfig(outputConfig)
	defer restoreOutputConfig()

	commandStderr := stderr
	var commandStderrCapture *bytes.Buffer
	if outputConfig.errorOutputMode == cliErrorOutputModeJSON {
		commandStderrCapture = &bytes.Buffer{}
		commandStderr = commandStderrCapture
	}
	finish := func(code int) int {
		return finalizeCLIErrorOutput(stderr, commandStderrCapture, code)
	}

	command := strings.TrimSpace(rest[0])
	commandArgs := rest[1:]
	metaNeedsDaemon := command == "meta" && len(commandArgs) > 0 && strings.EqualFold(strings.TrimSpace(commandArgs[0]), "capabilities")
	connectorNeedsDaemon := command == "connector" && connectorSubcommandRequiresDaemon(commandArgs)
	registry := cliRootCommandRegistry()
	commandCtx := cliRootCommandContext{
		stdout:        stdout,
		stderr:        commandStderr,
		correlationID: *correlationID,
		dbPath:        *dbPath,
		stdin:         chatInput,
	}

	if strings.EqualFold(command, "help") {
		return finish(runHelpCommand(commandArgs, commandCtx.stdout, commandCtx.stderr))
	}
	if helpPath, helpRequested := resolveInlineHelpPath(command, commandArgs); helpRequested {
		return finish(renderCLIHelpForPath(helpPath, buildCLISchemaDocument(), commandCtx.stdout, commandCtx.stderr))
	}

	if command == "auth" || command == "profile" || command == "quickstart" || command == "completion" || command == "version" || (command == "meta" && !metaNeedsDaemon) || (command == "connector" && !connectorNeedsDaemon) {
		handler, found := registry[command]
		if !found {
			writeUnknownCommandError(commandStderr, "command", command, sortedRootCommandNames(registry))
			printUsage(commandStderr)
			return finish(2)
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		commandCtx.ctx = ctx
		return finish(handler.run(commandCtx, commandArgs))
	}

	explicitFlags := visitedFlagNames(root)
	resolvedRuntimeProfile, err := resolveCLIRuntimeProfile(*runtimeProfile)
	if err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}
	if err := applyActiveCLIProfileDefaults(explicitFlags, listenerMode, address, authToken, authTokenFile); err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}

	resolvedAuthToken, err := controlauth.ResolveToken(*authToken, *authTokenFile)
	if err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}
	if err := validateCLIAuthTokenByRuntimeProfile(resolvedRuntimeProfile, resolvedAuthToken, strings.TrimSpace(*authTokenFile)); err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}
	if err := validateCLITransportByRuntimeProfile(resolvedRuntimeProfile, *listenerMode, *tlsInsecureSkipVerify); err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}
	transportTLSConfig, err := buildCLITransportTLSConfig(cliTransportTLSOptions{
		ListenerMode:        *listenerMode,
		TLSCAFile:           *tlsCAFile,
		TLSClientCertFile:   *tlsClientCertFile,
		TLSClientKeyFile:    *tlsClientKeyFile,
		TLSServerName:       *tlsServerName,
		TLSInsecureSkipCert: *tlsInsecureSkipVerify,
	})
	if err != nil {
		fmt.Fprintf(commandStderr, "invalid CLI configuration: %v\n", err)
		return finish(2)
	}
	commandCtx.clientFactory = newCLIClientFactory(transport.ClientConfig{
		ListenerMode: transport.ListenerMode(*listenerMode),
		Address:      *address,
		AuthToken:    resolvedAuthToken,
		Timeout:      *timeout,
		TLSConfig:    transportTLSConfig,
	})

	handler, found := registry[command]
	if !found {
		writeUnknownCommandError(commandStderr, "command", command, sortedRootCommandNames(registry))
		printUsage(commandStderr)
		return finish(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	commandCtx.ctx = ctx
	return finish(handler.run(commandCtx, commandArgs))
}

type cliRootCommandContext struct {
	ctx           context.Context
	stdout        io.Writer
	stderr        io.Writer
	correlationID string
	dbPath        string
	stdin         io.Reader
	clientFactory func() (*transport.Client, error)
}

func (c cliRootCommandContext) daemonClient() (*transport.Client, int) {
	if c.clientFactory == nil {
		fmt.Fprintln(c.stderr, "request failed: daemon client is not configured")
		return nil, 1
	}
	client, err := c.clientFactory()
	if err != nil {
		fmt.Fprintf(c.stderr, "client setup failed: %v\n", err)
		return nil, 1
	}
	return client, 0
}

func (c cliRootCommandContext) withDaemonClient(runner func(client *transport.Client) int) int {
	client, exitCode := c.daemonClient()
	if exitCode != 0 {
		return exitCode
	}
	return runner(client)
}

type cliRootCommand struct {
	run func(commandCtx cliRootCommandContext, args []string) int
}

func connectorSubcommandRequiresDaemon(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "mail", "calendar", "browser":
		if len(args) < 2 {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(args[1]), "ingest")
	case "cloudflared", "twilio":
		return true
	default:
		return false
	}
}

func newCLIClientFactory(config transport.ClientConfig) func() (*transport.Client, error) {
	var (
		once   sync.Once
		client *transport.Client
		err    error
	)
	return func() (*transport.Client, error) {
		once.Do(func() {
			client, err = transport.NewClient(config)
		})
		return client, err
	}
}
