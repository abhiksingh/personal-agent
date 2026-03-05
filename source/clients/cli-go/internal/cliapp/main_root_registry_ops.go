package cliapp

import (
	"context"
	"io"

	"personalagent/runtime/internal/transport"
)

func registerCLIOperationCommands(registry map[string]cliRootCommand) {
	registerDaemonRootCommand(registry, "comm", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runCommDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "automation", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runAutomationDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "inspect", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runInspectDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "retention", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runRetentionDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "context", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runContextDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "smoke", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runSmoke(ctx, client, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "doctor", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runDoctorCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "task", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runTaskCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "stream", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runStreamCommand(ctx, client, args, correlationID, stdout, stderr)
	})
}
