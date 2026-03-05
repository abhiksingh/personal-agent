package cliapp

import (
	"fmt"
	"io"
	"strings"
)

func appendQuickstartStep(report *quickstartReport, step quickstartStep) {
	if report == nil {
		return
	}
	report.Steps = append(report.Steps, step)
}

func finalizeQuickstartAndWrite(stdout io.Writer, report *quickstartReport) int {
	if report == nil {
		return 1
	}
	finalizeQuickstartReport(report)
	exitCode := writeJSON(stdout, report)
	if exitCode != 0 {
		return exitCode
	}
	if report.OverallStatus == quickstartStepStatusFail {
		return 1
	}
	return 0
}

func finalizeQuickstartReport(report *quickstartReport) {
	report.Summary = quickstartSummary{}
	for _, step := range report.Steps {
		switch step.Status {
		case quickstartStepStatusPass:
			report.Summary.Pass++
		case quickstartStepStatusWarn:
			report.Summary.Warn++
		case quickstartStepStatusFail:
			report.Summary.Fail++
		default:
			report.Summary.Skipped++
		}
	}

	report.OverallStatus = quickstartStepStatusPass
	if report.Summary.Fail > 0 {
		report.OverallStatus = quickstartStepStatusFail
	} else if report.Summary.Warn > 0 {
		report.OverallStatus = quickstartStepStatusWarn
	}
	report.Success = report.OverallStatus != quickstartStepStatusFail

	nextSteps := []string{}
	seen := map[string]struct{}{}
	for _, step := range report.Steps {
		if step.Status != quickstartStepStatusWarn && step.Status != quickstartStepStatusFail {
			continue
		}
		for _, item := range step.Remediation {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			nextSteps = append(nextSteps, trimmed)
		}
	}
	report.Remediation.NextSteps = nextSteps
	switch report.OverallStatus {
	case quickstartStepStatusFail:
		report.Remediation.HumanSummary = fmt.Sprintf("Quickstart requires attention: %d blocking step(s) failed.", report.Summary.Fail)
	case quickstartStepStatusWarn:
		report.Remediation.HumanSummary = fmt.Sprintf("Quickstart completed with warnings: %d step(s) need follow-up.", report.Summary.Warn)
	default:
		report.Remediation.HumanSummary = "Quickstart completed successfully."
	}
}
