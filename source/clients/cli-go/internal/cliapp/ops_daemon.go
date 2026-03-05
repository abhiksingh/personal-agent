package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runInspectDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "inspect subcommand required: run|transcript|memory")
		return 2
	}

	switch args[0] {
	case "run":
		flags := flag.NewFlagSet("inspect run", flag.ContinueOnError)
		flags.SetOutput(stderr)

		runID := flags.String("run-id", "", "task run id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.InspectRun(ctx, transport.InspectRunRequest{
			RunID: strings.TrimSpace(*runID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "transcript":
		flags := flag.NewFlagSet("inspect transcript", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		threadID := flags.String("thread-id", "", "optional thread id filter")
		limit := flags.Int("limit", 20, "max events")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.InspectTranscript(ctx, transport.InspectTranscriptRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			ThreadID:    strings.TrimSpace(*threadID),
			Limit:       *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "memory":
		flags := flag.NewFlagSet("inspect memory", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		ownerActor := flags.String("owner", "", "optional owner actor id filter")
		status := flags.String("status", "", "optional status filter")
		limit := flags.Int("limit", 50, "max memory rows")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.InspectMemory(ctx, transport.InspectMemoryRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			OwnerActor:  strings.TrimSpace(*ownerActor),
			Status:      strings.TrimSpace(*status),
			Limit:       *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "inspect subcommand", args[0])
		return 2
	}
}

func runRetentionDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "retention subcommand required: purge|compact-memory")
		return 2
	}

	switch args[0] {
	case "purge":
		flags := flag.NewFlagSet("retention purge", flag.ContinueOnError)
		flags.SetOutput(stderr)

		traceDays := flags.Int("trace-days", 0, "optional trace retention days override")
		transcriptDays := flags.Int("transcript-days", 0, "optional transcript retention days override")
		memoryDays := flags.Int("memory-days", 0, "optional memory retention days override")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.RetentionPurge(ctx, transport.RetentionPurgeRequest{
			TraceDays:      *traceDays,
			TranscriptDays: *transcriptDays,
			MemoryDays:     *memoryDays,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "compact-memory":
		flags := flag.NewFlagSet("retention compact-memory", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		ownerActor := flags.String("owner", "", "owner principal actor id")
		tokenThreshold := flags.Int("token-threshold", 1000, "token threshold")
		staleAfterHours := flags.Int("stale-after-hours", 168, "stale age threshold in hours")
		limit := flags.Int("limit", 500, "max memory items to consider")
		apply := flags.Bool("apply", false, "persist compaction result")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.RetentionCompactMemory(ctx, transport.RetentionCompactMemoryRequest{
			WorkspaceID:     normalizeWorkspace(*workspaceID),
			OwnerActor:      strings.TrimSpace(*ownerActor),
			TokenThreshold:  *tokenThreshold,
			StaleAfterHours: *staleAfterHours,
			Limit:           *limit,
			Apply:           *apply,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "retention subcommand", args[0])
		return 2
	}
}

func runContextDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "context subcommand required: samples|tune")
		return 2
	}

	switch args[0] {
	case "samples":
		flags := flag.NewFlagSet("context samples", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		taskClass := flags.String("task-class", "", "task class")
		limit := flags.Int("limit", 20, "max samples")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ContextSamples(ctx, transport.ContextSamplesRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			TaskClass:   strings.TrimSpace(*taskClass),
			Limit:       *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "tune":
		flags := flag.NewFlagSet("context tune", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		taskClass := flags.String("task-class", "", "task class")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ContextTune(ctx, transport.ContextTuneRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			TaskClass:   strings.TrimSpace(*taskClass),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "context subcommand", args[0])
		return 2
	}
}
