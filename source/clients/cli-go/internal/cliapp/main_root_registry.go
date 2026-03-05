package cliapp

import (
	"context"
	"io"

	"personalagent/runtime/internal/transport"
)

type daemonRootCommandRunner func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int

type daemonRootCommandRunnerWithInput func(ctx context.Context, client *transport.Client, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, correlationID string) int

func cliRootCommandRegistry() map[string]cliRootCommand {
	registry := make(map[string]cliRootCommand, 24)
	registerCLICoreCommands(registry)
	registerCLIConfigurationCommands(registry)
	registerCLIAgentCommands(registry)
	registerCLIOperationCommands(registry)
	return registry
}

func registerRootCommand(registry map[string]cliRootCommand, name string, run func(commandCtx cliRootCommandContext, args []string) int) {
	registry[name] = cliRootCommand{run: run}
}

func registerDaemonRootCommand(registry map[string]cliRootCommand, name string, runner daemonRootCommandRunner) {
	registerRootCommand(registry, name, func(commandCtx cliRootCommandContext, args []string) int {
		return commandCtx.withDaemonClient(func(client *transport.Client) int {
			return runner(commandCtx.ctx, client, args, commandCtx.correlationID, commandCtx.stdout, commandCtx.stderr)
		})
	})
}

func registerDaemonRootCommandWithInput(registry map[string]cliRootCommand, name string, runner daemonRootCommandRunnerWithInput) {
	registerRootCommand(registry, name, func(commandCtx cliRootCommandContext, args []string) int {
		return commandCtx.withDaemonClient(func(client *transport.Client) int {
			return runner(commandCtx.ctx, client, args, commandCtx.stdin, commandCtx.stdout, commandCtx.stderr, commandCtx.correlationID)
		})
	})
}
