package calendar

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

var (
	calendarEventIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,127}$`)
)

type calendarStepInput struct {
	EventID string
	Title   string
	Notes   string
}

func isCalendarExecuteProbeCapability(capability string) bool {
	return strings.EqualFold(strings.TrimSpace(capability), capabilityExecuteProbe)
}

func resolveCalendarName() string {
	if configured := strings.TrimSpace(os.Getenv(envCalendarDefaultName)); configured != "" {
		return configured
	}
	return "Home"
}

func resolveCalendarCreateInput(step connectorcontract.TaskStep) (calendarStepInput, error) {
	if len(step.Input) == 0 {
		return calendarStepInput{}, fmt.Errorf("calendar step input is required")
	}
	title, err := adapterhelpers.RequiredStringInput(step.Input, "title")
	if err != nil {
		return calendarStepInput{}, err
	}
	notes, err := adapterhelpers.OptionalStringInput(step.Input, "notes")
	if err != nil {
		return calendarStepInput{}, err
	}
	return calendarStepInput{
		Title: title,
		Notes: notes,
	}, nil
}

func resolveCalendarUpdateInput(step connectorcontract.TaskStep) (calendarStepInput, error) {
	if len(step.Input) == 0 {
		return calendarStepInput{}, fmt.Errorf("calendar step input is required")
	}
	eventIDRaw, err := adapterhelpers.RequiredStringInput(step.Input, "event_id")
	if err != nil {
		return calendarStepInput{}, err
	}
	eventID, err := normalizeCalendarEventID(eventIDRaw)
	if err != nil {
		return calendarStepInput{}, err
	}
	title, err := adapterhelpers.OptionalStringInput(step.Input, "title")
	if err != nil {
		return calendarStepInput{}, err
	}
	notes, err := adapterhelpers.OptionalStringInput(step.Input, "notes")
	if err != nil {
		return calendarStepInput{}, err
	}
	if strings.TrimSpace(title) == "" && strings.TrimSpace(notes) == "" {
		return calendarStepInput{}, fmt.Errorf("calendar update requires title or notes")
	}
	return calendarStepInput{
		EventID: eventID,
		Title:   title,
		Notes:   notes,
	}, nil
}

func resolveCalendarCancelInput(step connectorcontract.TaskStep) (calendarStepInput, error) {
	if len(step.Input) == 0 {
		return calendarStepInput{}, fmt.Errorf("calendar step input is required")
	}
	eventIDRaw, err := adapterhelpers.RequiredStringInput(step.Input, "event_id")
	if err != nil {
		return calendarStepInput{}, err
	}
	eventID, err := normalizeCalendarEventID(eventIDRaw)
	if err != nil {
		return calendarStepInput{}, err
	}
	return calendarStepInput{
		EventID: eventID,
	}, nil
}

func normalizeCalendarEventID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("event_id is required")
	}
	if !calendarEventIDPattern.MatchString(trimmed) {
		return "", fmt.Errorf("event_id must match %s", calendarEventIDPattern.String())
	}
	return trimmed, nil
}

func composeCalendarEventDescription(eventID string, notes string) string {
	marker := "PA_EVENT_ID:" + strings.TrimSpace(eventID)
	trimmedNotes := strings.TrimSpace(notes)
	if trimmedNotes == "" {
		return marker
	}
	return marker + "\n" + trimmedNotes
}

func isCalendarAutomationDryRunEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envCalendarAutomationDryRun))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
