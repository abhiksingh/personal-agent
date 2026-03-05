package calendar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"time"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
)

type calendarOperationResult struct {
	OperationID string
	Transport   string
}

type calendarOperationRequest struct {
	Mode         string
	CalendarName string
	EventID      string
	Title        string
	Notes        string
}

var commandRunner adapterscaffold.CommandRunner = adapterscaffold.DefaultCommandRunner

func executeCalendarPermissionProbe(ctx context.Context) error {
	if isCalendarAutomationDryRunEnabled() {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("calendar automation requires macOS (set %s=1 for dry-run)", envCalendarAutomationDryRun)
	}

	scriptLines := []string{
		`tell application id "com.apple.iCal"`,
		`set calendarCount to count of calendars`,
		`return calendarCount as text`,
		`end tell`,
	}
	args := make([]string, 0, len(scriptLines)*2)
	for _, line := range scriptLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		args = append(args, "-e", trimmed)
	}

	result := adapterscaffold.ExecuteCommand(ctx, commandRunner, "osascript", args...)
	if result.Err != nil {
		if strings.TrimSpace(result.Output) != "" {
			return fmt.Errorf("calendar automation probe failed: %s", result.Output)
		}
		return fmt.Errorf("calendar automation probe failed: %w", result.Err)
	}
	return nil
}

func executeCalendarOperation(ctx context.Context, request calendarOperationRequest) (calendarOperationResult, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(request.Mode))
	if normalizedMode == "" {
		return calendarOperationResult{}, fmt.Errorf("calendar operation mode is required")
	}
	switch normalizedMode {
	case "create", "update", "cancel":
	default:
		return calendarOperationResult{}, fmt.Errorf("calendar operation mode %q is not supported", normalizedMode)
	}
	calendarName := strings.TrimSpace(request.CalendarName)
	if calendarName == "" {
		calendarName = "Home"
	}
	eventID, err := normalizeCalendarEventID(request.EventID)
	if err != nil {
		return calendarOperationResult{}, err
	}
	title := strings.TrimSpace(request.Title)
	if normalizedMode == "create" && title == "" {
		title = "Personal Agent Event"
	}
	notes := strings.TrimSpace(request.Notes)
	description := composeCalendarEventDescription(eventID, notes)

	operationID := newCalendarOperationID(normalizedMode)
	if isCalendarAutomationDryRunEnabled() {
		return calendarOperationResult{
			OperationID: operationID,
			Transport:   transportCalendarDryRun,
		}, nil
	}
	if runtime.GOOS != "darwin" {
		return calendarOperationResult{}, fmt.Errorf("calendar automation requires macOS (set %s=1 for dry-run)", envCalendarAutomationDryRun)
	}

	scriptLines := []string{
		"on run argv",
		"set operationMode to item 1 of argv",
		"set targetCalendarName to item 2 of argv",
		"set targetEventID to item 3 of argv",
		"set eventTitle to item 4 of argv",
		"set eventDescription to item 5 of argv",
		"set eventMarker to \"PA_EVENT_ID:\" & targetEventID",
		"tell application \"Calendar\"",
		"set targetCalendar to first calendar whose name is targetCalendarName",
		"if operationMode is \"create\" then",
		"set startDate to (current date)",
		"set endDate to startDate + (60 * minutes)",
		"make new event at end of events of targetCalendar with properties {summary:eventTitle, start date:startDate, end date:endDate, description:eventDescription}",
		"else",
		"set matchingEvents to (every event of targetCalendar whose description contains eventMarker)",
		"if (count of matchingEvents) is 0 then error \"event_not_found\" number 50100",
		"set targetEvent to item 1 of matchingEvents",
		"if operationMode is \"update\" then",
		"set summary of targetEvent to eventTitle",
		"set description of targetEvent to eventDescription",
		"else if operationMode is \"cancel\" then",
		"delete targetEvent",
		"end if",
		"end if",
		"end tell",
		"end run",
	}
	args := make([]string, 0, len(scriptLines)*2+5)
	for _, line := range scriptLines {
		args = append(args, "-e", line)
	}
	args = append(args, normalizedMode, calendarName, eventID, title, description)

	result := adapterscaffold.ExecuteCommand(ctx, commandRunner, "osascript", args...)
	if result.Err != nil {
		if strings.Contains(strings.ToLower(result.Output), "event_not_found") {
			return calendarOperationResult{}, fmt.Errorf("calendar event %s not found", eventID)
		}
		if result.Output != "" {
			return calendarOperationResult{}, fmt.Errorf("calendar automation failed: %s", result.Output)
		}
		return calendarOperationResult{}, fmt.Errorf("calendar automation failed: %w", result.Err)
	}

	return calendarOperationResult{
		OperationID: operationID,
		Transport:   transportCalendarAppleEvents,
	}, nil
}

func newCalendarOperationID(prefix string) string {
	tokenBytes := make([]byte, 4)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(tokenBytes))
}
