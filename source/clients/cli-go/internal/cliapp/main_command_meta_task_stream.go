package cliapp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"personalagent/runtime/internal/transport"
)

func runMetaCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "meta subcommand required: schema|capabilities")
		return 2
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "schema":
		return writeJSON(stdout, buildCLISchemaDocument())
	default:
		writeUnknownCommandError(stderr, "meta subcommand", args[0], []string{"schema", "capabilities"})
		return 2
	}
}

func runMetaCapabilities(ctx context.Context, client *transport.Client, correlationID string, stdout io.Writer, stderr io.Writer) int {
	response, err := client.DaemonCapabilities(ctx, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	return writeJSON(stdout, response)
}

func runSmoke(ctx context.Context, client *transport.Client, correlationID string, stdout io.Writer, stderr io.Writer) int {
	response, err := client.CapabilitySmoke(ctx, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	return writeJSON(stdout, response)
}

func runTaskCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "task subcommand required: submit|status|cancel|retry|requeue")
		return 2
	}

	switch args[0] {
	case "submit":
		flags := flag.NewFlagSet("task submit", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		requestedBy := flags.String("requested-by", "", "requester actor id")
		subject := flags.String("subject", "", "subject principal actor id")
		title := flags.String("title", "", "task title")
		description := flags.String("description", "", "task description")
		taskClass := flags.String("task-class", "", "task class")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := normalizeWorkspace(*workspaceID)

		response, err := client.SubmitTask(ctx, transport.SubmitTaskRequest{
			WorkspaceID:             workspace,
			RequestedByActorID:      *requestedBy,
			SubjectPrincipalActorID: *subject,
			Title:                   *title,
			Description:             *description,
			TaskClass:               *taskClass,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "status":
		flags := flag.NewFlagSet("task status", flag.ContinueOnError)
		flags.SetOutput(stderr)

		taskID := flags.String("task-id", "", "task id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.TaskStatus(ctx, *taskID, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeTaskStatusResponse(stdout, response)
	case "cancel":
		flags := flag.NewFlagSet("task cancel", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id (optional guard)")
		taskID := flags.String("task-id", "", "task id")
		runID := flags.String("run-id", "", "run id")
		reason := flags.String("reason", "", "optional cancellation reason")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := strings.TrimSpace(*workspaceID)
		if strings.TrimSpace(*taskID) == "" && strings.TrimSpace(*runID) == "" {
			fmt.Fprintln(stderr, "request failed: --task-id or --run-id is required")
			return 1
		}

		response, err := client.CancelTask(ctx, transport.TaskCancelRequest{
			WorkspaceID: workspace,
			TaskID:      strings.TrimSpace(*taskID),
			RunID:       strings.TrimSpace(*runID),
			Reason:      strings.TrimSpace(*reason),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "retry":
		flags := flag.NewFlagSet("task retry", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id (optional guard)")
		taskID := flags.String("task-id", "", "task id")
		runID := flags.String("run-id", "", "run id")
		reason := flags.String("reason", "", "optional retry reason")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := strings.TrimSpace(*workspaceID)
		if strings.TrimSpace(*taskID) == "" && strings.TrimSpace(*runID) == "" {
			fmt.Fprintln(stderr, "request failed: --task-id or --run-id is required")
			return 1
		}

		response, err := client.RetryTask(ctx, transport.TaskRetryRequest{
			WorkspaceID: workspace,
			TaskID:      strings.TrimSpace(*taskID),
			RunID:       strings.TrimSpace(*runID),
			Reason:      strings.TrimSpace(*reason),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "requeue":
		flags := flag.NewFlagSet("task requeue", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id (optional guard)")
		taskID := flags.String("task-id", "", "task id")
		runID := flags.String("run-id", "", "run id")
		reason := flags.String("reason", "", "optional requeue reason")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := strings.TrimSpace(*workspaceID)
		if strings.TrimSpace(*taskID) == "" && strings.TrimSpace(*runID) == "" {
			fmt.Fprintln(stderr, "request failed: --task-id or --run-id is required")
			return 1
		}

		response, err := client.RequeueTask(ctx, transport.TaskRequeueRequest{
			WorkspaceID: workspace,
			TaskID:      strings.TrimSpace(*taskID),
			RunID:       strings.TrimSpace(*runID),
			Reason:      strings.TrimSpace(*reason),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownCommandError(stderr, "task subcommand", args[0], []string{"submit", "status", "cancel", "retry", "requeue"})
		return 2
	}
}

func runStreamCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("stream", flag.ContinueOnError)
	flags.SetOutput(stderr)

	duration := flags.Duration("duration", 10*time.Second, "stream duration")
	signalType := flags.String("signal-type", "", "optional signal to send on connect")
	taskID := flags.String("task-id", "", "optional task id for signal")
	runID := flags.String("run-id", "", "optional run id for signal")
	reason := flags.String("reason", "", "optional reason for signal")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	stream, err := client.ConnectRealtime(ctx, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	defer stream.Close()

	if strings.TrimSpace(*signalType) != "" {
		if err := stream.SendSignal(transport.ClientSignal{
			SignalType:    *signalType,
			TaskID:        *taskID,
			RunID:         *runID,
			Reason:        *reason,
			CorrelationID: correlationID,
		}); err != nil {
			return writeError(stderr, err)
		}
	}

	if *duration <= 0 {
		return 0
	}

	done := make(chan struct{})
	defer close(done)

	events := make(chan transport.RealtimeEventEnvelope)
	readErrors := make(chan error, 1)
	go func() {
		for {
			event, readErr := stream.Receive()
			if readErr != nil {
				select {
				case readErrors <- readErr:
				default:
				}
				return
			}
			select {
			case events <- event:
			case <-done:
				return
			}
		}
	}()

	timer := time.NewTimer(*duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return 0
		case event := <-events:
			if exitCode := writeJSON(stdout, event); exitCode != 0 {
				return exitCode
			}
		case err := <-readErrors:
			if isExpectedRealtimeReadTermination(err) {
				return 0
			}
			return writeError(stderr, err)
		}
	}
}

func isExpectedRealtimeReadTermination(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var closeErr *websocket.CloseError
	return errors.As(err, &closeErr)
}
