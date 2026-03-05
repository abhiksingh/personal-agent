package cliapp

import (
	"errors"
	"strings"

	"personalagent/runtime/internal/transport"
)

func doctorErrorDetails(err error) map[string]any {
	details := map[string]any{
		"error": err.Error(),
	}
	var httpErr transport.HTTPError
	if errors.As(err, &httpErr) {
		details["status_code"] = httpErr.StatusCode
		details["code"] = strings.TrimSpace(httpErr.Code)
		details["message"] = strings.TrimSpace(httpErr.Message)
		details["correlation_id"] = strings.TrimSpace(httpErr.CorrelationID)
		if httpErr.DetailsPayload != nil {
			details["problem_details"] = httpErr.DetailsPayload
		}
	}
	return details
}

func finalizeDoctorReport(report *doctorReport) {
	if report == nil {
		return
	}
	report.Summary = doctorSummary{}
	report.OverallStatus = doctorCheckStatusPass
	for _, check := range report.Checks {
		switch check.Status {
		case doctorCheckStatusPass:
			report.Summary.Pass++
		case doctorCheckStatusWarn:
			report.Summary.Warn++
		case doctorCheckStatusFail:
			report.Summary.Fail++
		case doctorCheckStatusSkipped:
			report.Summary.Skipped++
		}
	}
	if report.Summary.Fail > 0 {
		report.OverallStatus = doctorCheckStatusFail
		return
	}
	if report.Summary.Warn > 0 {
		report.OverallStatus = doctorCheckStatusWarn
		return
	}
	if report.Summary.Pass == 0 && report.Summary.Skipped > 0 {
		report.OverallStatus = doctorCheckStatusWarn
	}
}

func doctorFindConnectorCard(cards []transport.ConnectorStatusCard, connectorID string) (transport.ConnectorStatusCard, bool) {
	target := strings.ToLower(strings.TrimSpace(connectorID))
	if target == "" {
		return transport.ConnectorStatusCard{}, false
	}
	for _, card := range cards {
		if strings.EqualFold(strings.TrimSpace(card.ConnectorID), target) {
			return card, true
		}
	}
	return transport.ConnectorStatusCard{}, false
}

func appendDoctorRemediation(existing []string, candidates ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existing)+len(candidates))
	for _, value := range existing {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	for _, value := range candidates {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
