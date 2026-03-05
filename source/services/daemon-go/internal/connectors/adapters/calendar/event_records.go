package calendar

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

var unsafeEventPathTokenRegex = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type operationRecord struct {
	OperationID      string `json:"operation_id"`
	EventID          string `json:"event_id,omitempty"`
	Connector        string `json:"connector"`
	CapabilityKey    string `json:"capability_key"`
	StepID           string `json:"step_id"`
	WorkspaceID      string `json:"workspace_id"`
	RunID            string `json:"run_id"`
	RequestedByActor string `json:"requested_by_actor_id,omitempty"`
	SubjectActor     string `json:"subject_principal_actor_id,omitempty"`
	ActingAsActor    string `json:"acting_as_actor_id,omitempty"`
	CreatedAt        string `json:"created_at"`
	Transport        string `json:"transport,omitempty"`
	CalendarName     string `json:"calendar_name,omitempty"`
	EventTitle       string `json:"event_title,omitempty"`
	EventNotes       string `json:"event_notes,omitempty"`
}

type calendarEventRecord struct {
	EventID           string `json:"event_id"`
	WorkspaceID       string `json:"workspace_id"`
	CalendarName      string `json:"calendar_name"`
	Title             string `json:"title"`
	Notes             string `json:"notes,omitempty"`
	Status            string `json:"status"`
	LastOperationID   string `json:"last_operation_id,omitempty"`
	LastOperationMode string `json:"last_operation_mode,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func buildCalendarOperationRecord(
	capabilityKey string,
	operationID string,
	eventID string,
	stepID string,
	execCtx connectorcontract.ExecutionContext,
	createdAt string,
	transport string,
	calendarName string,
	title string,
	notes string,
) operationRecord {
	return operationRecord{
		OperationID:      operationID,
		EventID:          eventID,
		Connector:        "calendar",
		CapabilityKey:    capabilityKey,
		StepID:           stepID,
		WorkspaceID:      execCtx.WorkspaceID,
		RunID:            execCtx.RunID,
		RequestedByActor: execCtx.RequestedByActor,
		SubjectActor:     execCtx.SubjectPrincipal,
		ActingAsActor:    execCtx.ActingAsActor,
		CreatedAt:        createdAt,
		Transport:        transport,
		CalendarName:     calendarName,
		EventTitle:       title,
		EventNotes:       notesPreview(notes),
	}
}

func writeCalendarOperationRecord(workspaceRoot string, capability string, operationID string, record operationRecord) (string, error) {
	return adapterscaffold.WriteOperationRecord(workspaceRoot, capability, operationID, record)
}

func calendarEventRecordPath(workspaceRoot string, eventID string) string {
	return filepath.Join(workspaceRoot, "events", eventRecordToken(eventID)+".json")
}

func eventRecordToken(eventID string) string {
	trimmed := strings.TrimSpace(eventID)
	cleaned := unsafeEventPathTokenRegex.ReplaceAllString(trimmed, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		cleaned = "event"
	}
	if len(cleaned) > 96 {
		cleaned = cleaned[:96]
	}
	sum := sha1.Sum([]byte(trimmed))
	return fmt.Sprintf("%s-%s", cleaned, hex.EncodeToString(sum[:4]))
}

func loadCalendarEventRecord(path string) (calendarEventRecord, error) {
	var record calendarEventRecord
	if err := localstate.ReadJSONFile(path, &record); err != nil {
		if os.IsNotExist(err) {
			return calendarEventRecord{}, err
		}
		return calendarEventRecord{}, fmt.Errorf("read calendar event record: %w", err)
	}
	if strings.TrimSpace(record.EventID) == "" {
		return calendarEventRecord{}, fmt.Errorf("calendar event record is invalid at %s", path)
	}
	return record, nil
}

func notesPreview(notes string) string {
	trimmed := strings.TrimSpace(notes)
	if len(trimmed) <= 120 {
		return trimmed
	}
	return trimmed[:120]
}
