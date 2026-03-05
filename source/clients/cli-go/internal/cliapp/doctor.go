package cliapp

import (
	"context"
	"flag"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func runDoctorCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	options, exitCode := parseDoctorCommandOptions(args, stderr)
	if exitCode != 0 {
		return exitCode
	}

	report := doctorReport{
		SchemaVersion: "1.0.0",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		OverallStatus: doctorCheckStatusPass,
		Checks:        []doctorCheck{},
	}

	checks := make([]doctorCheck, 0, 10)

	connectivityCheck, _, connectivityErr := runDoctorConnectivityCheck(ctx, client, correlationID)
	checks = append(checks, connectivityCheck)
	if connectivityErr != nil {
		checks = append(checks, buildDoctorConnectivityFailureChecks(options.IncludeOptional)...)
		report.Checks = checks
		return doctorWriteAndExit(stdout, &report, options.Strict)
	}

	checks = append(checks, runDoctorLifecycleHealthCheck(ctx, client, correlationID))

	workspaceCheck, workspace := runDoctorWorkspaceContextCheck(ctx, client, options.RequestedWorkspace, correlationID)
	checks = append(checks, workspaceCheck)
	report.WorkspaceID = workspace

	state := &doctorExecutionState{
		Workspace:       workspace,
		CorrelationID:   correlationID,
		IncludeOptional: options.IncludeOptional,
	}

	if options.Quick {
		checks = append(checks, buildDoctorQuickModeSkippedChecks(options.IncludeOptional)...)
		report.Checks = checks
		return doctorWriteAndExit(stdout, &report, options.Strict)
	}

	for _, runner := range doctorFullCheckRegistry() {
		checks = append(checks, runner(ctx, client, state))
	}

	report.Checks = checks
	return doctorWriteAndExit(stdout, &report, options.Strict)
}

func parseDoctorCommandOptions(args []string, stderr io.Writer) (doctorCommandOptions, int) {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id override")
	quick := flags.Bool("quick", false, "run only first-line diagnostics (connectivity/lifecycle/context)")
	includeOptional := flags.Bool("include-optional", true, "include optional tooling checks (cloudflared/local bridge)")
	strict := flags.Bool("strict", false, "treat warnings as non-zero exit code")
	if err := flags.Parse(args); err != nil {
		return doctorCommandOptions{}, 2
	}

	return doctorCommandOptions{
		RequestedWorkspace: strings.TrimSpace(*workspaceID),
		Quick:              *quick,
		IncludeOptional:    *includeOptional,
		Strict:             *strict,
	}, 0
}
