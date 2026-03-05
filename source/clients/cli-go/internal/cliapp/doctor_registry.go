package cliapp

import (
	"context"
	"io"

	"personalagent/runtime/internal/transport"
)

type doctorCheckRunner func(context.Context, *transport.Client, *doctorExecutionState) doctorCheck

func doctorFullCheckRegistry() []doctorCheckRunner {
	return []doctorCheckRunner{
		runDoctorProviderReadinessCheck,
		runDoctorModelRouteReadinessCheck,
		runDoctorChannelMappingReadinessCheck,
		runDoctorSecretReferenceCheck,
		runDoctorPluginRuntimeHealthCheck,
		runDoctorOptionalToolingCheck,
	}
}

func doctorWriteAndExit(stdout io.Writer, report *doctorReport, strict bool) int {
	if report == nil {
		return 1
	}
	finalizeDoctorReport(report)
	if code := writeDoctorReportResponse(stdout, *report); code != 0 {
		return code
	}
	if report.OverallStatus == doctorCheckStatusFail {
		return 1
	}
	if strict && report.OverallStatus == doctorCheckStatusWarn {
		return 1
	}
	return 0
}
