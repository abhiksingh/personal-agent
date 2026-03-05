package mail

import (
	"strings"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

type operationRecord struct {
	OperationID      string `json:"operation_id"`
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
	Recipient        string `json:"recipient,omitempty"`
	Subject          string `json:"subject,omitempty"`
	BodyPreview      string `json:"body_preview,omitempty"`
}

func buildMailOperationRecord(
	capabilityKey string,
	operationID string,
	stepID string,
	execCtx connectorcontract.ExecutionContext,
	createdAt string,
	transport string,
	recipient string,
	subject string,
	body string,
) operationRecord {
	return operationRecord{
		OperationID:      operationID,
		Connector:        "mail",
		CapabilityKey:    capabilityKey,
		StepID:           stepID,
		WorkspaceID:      execCtx.WorkspaceID,
		RunID:            execCtx.RunID,
		RequestedByActor: execCtx.RequestedByActor,
		SubjectActor:     execCtx.SubjectPrincipal,
		ActingAsActor:    execCtx.ActingAsActor,
		CreatedAt:        createdAt,
		Transport:        transport,
		Recipient:        recipient,
		Subject:          subject,
		BodyPreview:      bodyPreview(body),
	}
}

func writeMailOperationRecord(workspaceRoot string, capabilityKey string, operationID string, record operationRecord) (string, error) {
	return adapterscaffold.WriteOperationRecord(workspaceRoot, capabilityKey, operationID, record)
}

func bodyPreview(body string) string {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) <= 120 {
		return trimmed
	}
	return trimmed[:120]
}
