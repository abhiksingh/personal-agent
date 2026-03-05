package transport

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RealtimeEventPayload captures typed externally-consumed realtime fields and
// preserves unknown keys in Additional.
type RealtimeEventPayload struct {
	WorkspaceID     string               `json:"workspace_id,omitempty"`
	TaskID          string               `json:"task_id,omitempty"`
	RunID           string               `json:"run_id,omitempty"`
	State           string               `json:"state,omitempty"`
	TaskState       string               `json:"task_state,omitempty"`
	RunState        string               `json:"run_state,omitempty"`
	LifecycleState  string               `json:"lifecycle_state,omitempty"`
	LifecycleSource string               `json:"lifecycle_source,omitempty"`
	LastError       string               `json:"last_error,omitempty"`
	SignalType      string               `json:"signal_type,omitempty"`
	Accepted        *bool                `json:"accepted,omitempty"`
	Reason          string               `json:"reason,omitempty"`
	Cancelled       *bool                `json:"cancelled,omitempty"`
	AlreadyTerminal *bool                `json:"already_terminal,omitempty"`
	ItemID          string               `json:"item_id,omitempty"`
	ItemIndex       *int                 `json:"item_index,omitempty"`
	ItemType        string               `json:"item_type,omitempty"`
	Status          string               `json:"status,omitempty"`
	Delta           string               `json:"delta,omitempty"`
	ToolName        string               `json:"tool_name,omitempty"`
	ToolCallID      string               `json:"tool_call_id,omitempty"`
	Arguments       map[string]any       `json:"arguments,omitempty"`
	Output          map[string]any       `json:"output,omitempty"`
	ErrorCode       string               `json:"error_code,omitempty"`
	Error           string               `json:"error,omitempty"`
	Metadata        ChatTurnItemMetadata `json:"metadata,omitempty"`
	TaskClass       string               `json:"task_class,omitempty"`
	Provider        string               `json:"provider,omitempty"`
	ModelKey        string               `json:"model_key,omitempty"`
	AssistantEmpty  *bool                `json:"assistant_empty,omitempty"`
	ItemCount       *int                 `json:"item_count,omitempty"`
	ToolCallCount   *int                 `json:"tool_call_count,omitempty"`
	ApprovalCount   *int                 `json:"approval_count,omitempty"`
	Message         string               `json:"message,omitempty"`
	Additional      map[string]any       `json:"-"`
}

func (p RealtimeEventPayload) IsZero() bool {
	return p.WorkspaceID == "" &&
		p.TaskID == "" &&
		p.RunID == "" &&
		p.State == "" &&
		p.TaskState == "" &&
		p.RunState == "" &&
		p.LifecycleState == "" &&
		p.LifecycleSource == "" &&
		p.LastError == "" &&
		p.SignalType == "" &&
		p.Accepted == nil &&
		p.Reason == "" &&
		p.Cancelled == nil &&
		p.AlreadyTerminal == nil &&
		p.ItemID == "" &&
		p.ItemIndex == nil &&
		p.ItemType == "" &&
		p.Status == "" &&
		p.Delta == "" &&
		p.ToolName == "" &&
		p.ToolCallID == "" &&
		len(p.Arguments) == 0 &&
		len(p.Output) == 0 &&
		p.ErrorCode == "" &&
		p.Error == "" &&
		p.Metadata.IsZero() &&
		p.TaskClass == "" &&
		p.Provider == "" &&
		p.ModelKey == "" &&
		p.AssistantEmpty == nil &&
		p.ItemCount == nil &&
		p.ToolCallCount == nil &&
		p.ApprovalCount == nil &&
		p.Message == "" &&
		len(p.Additional) == 0
}

func (p RealtimeEventPayload) AsMap() map[string]any {
	result := cloneAnyMapShallow(p.Additional)
	setStringField(result, "workspace_id", p.WorkspaceID)
	setStringField(result, "task_id", p.TaskID)
	setStringField(result, "run_id", p.RunID)
	setStringField(result, "state", p.State)
	setStringField(result, "task_state", p.TaskState)
	setStringField(result, "run_state", p.RunState)
	setStringField(result, "lifecycle_state", p.LifecycleState)
	setStringField(result, "lifecycle_source", p.LifecycleSource)
	setStringField(result, "last_error", p.LastError)
	setStringField(result, "signal_type", p.SignalType)
	setBoolPointerField(result, "accepted", p.Accepted)
	setStringField(result, "reason", p.Reason)
	setBoolPointerField(result, "cancelled", p.Cancelled)
	setBoolPointerField(result, "already_terminal", p.AlreadyTerminal)
	setStringField(result, "item_id", p.ItemID)
	setIntPointerField(result, "item_index", p.ItemIndex)
	setStringField(result, "item_type", p.ItemType)
	setStringField(result, "status", p.Status)
	setRawStringField(result, "delta", p.Delta)
	setStringField(result, "tool_name", p.ToolName)
	setStringField(result, "tool_call_id", p.ToolCallID)
	setAnyMapField(result, "arguments", p.Arguments)
	setAnyMapField(result, "output", p.Output)
	setStringField(result, "error_code", p.ErrorCode)
	setStringField(result, "error", p.Error)
	if !p.Metadata.IsZero() {
		result["metadata"] = p.Metadata.AsMap()
	}
	setStringField(result, "task_class", p.TaskClass)
	setStringField(result, "provider", p.Provider)
	setStringField(result, "model_key", p.ModelKey)
	setBoolPointerField(result, "assistant_empty", p.AssistantEmpty)
	setIntPointerField(result, "item_count", p.ItemCount)
	setIntPointerField(result, "tool_call_count", p.ToolCallCount)
	setIntPointerField(result, "approval_count", p.ApprovalCount)
	setStringField(result, "message", p.Message)
	return result
}

func RealtimeEventPayloadFromMap(value map[string]any) RealtimeEventPayload {
	if len(value) == 0 {
		return RealtimeEventPayload{}
	}
	result := RealtimeEventPayload{
		WorkspaceID:     readAnyString(value["workspace_id"]),
		TaskID:          readAnyString(value["task_id"]),
		RunID:           readAnyString(value["run_id"]),
		State:           readAnyString(value["state"]),
		TaskState:       readAnyString(value["task_state"]),
		RunState:        readAnyString(value["run_state"]),
		LifecycleState:  readAnyString(value["lifecycle_state"]),
		LifecycleSource: readAnyString(value["lifecycle_source"]),
		LastError:       readAnyString(value["last_error"]),
		SignalType:      readAnyString(value["signal_type"]),
		Accepted:        readAnyBoolPointer(value["accepted"]),
		Reason:          readAnyString(value["reason"]),
		Cancelled:       readAnyBoolPointer(value["cancelled"]),
		AlreadyTerminal: readAnyBoolPointer(value["already_terminal"]),
		ItemID:          readAnyString(value["item_id"]),
		ItemIndex:       readAnyIntPointer(value["item_index"]),
		ItemType:        readAnyString(value["item_type"]),
		Status:          readAnyString(value["status"]),
		Delta:           readAnyStringPreservingWhitespace(value["delta"]),
		ToolName:        readAnyString(value["tool_name"]),
		ToolCallID:      readAnyString(value["tool_call_id"]),
		Arguments:       readAnyMap(value["arguments"]),
		Output:          readAnyMap(value["output"]),
		ErrorCode:       readAnyString(value["error_code"]),
		Error:           readAnyString(value["error"]),
		TaskClass:       readAnyString(value["task_class"]),
		Provider:        readAnyString(value["provider"]),
		ModelKey:        readAnyString(value["model_key"]),
		AssistantEmpty:  readAnyBoolPointer(value["assistant_empty"]),
		ItemCount:       readAnyIntPointer(value["item_count"]),
		ToolCallCount:   readAnyIntPointer(value["tool_call_count"]),
		ApprovalCount:   readAnyIntPointer(value["approval_count"]),
		Message:         readAnyString(value["message"]),
	}
	if metadata := readAnyMap(value["metadata"]); len(metadata) > 0 {
		result.Metadata = ChatTurnItemMetadataFromMap(metadata)
	}
	result.Additional = removeKnownKeys(value,
		"workspace_id",
		"task_id",
		"run_id",
		"state",
		"task_state",
		"run_state",
		"lifecycle_state",
		"lifecycle_source",
		"last_error",
		"signal_type",
		"accepted",
		"reason",
		"cancelled",
		"already_terminal",
		"item_id",
		"item_index",
		"item_type",
		"status",
		"delta",
		"tool_name",
		"tool_call_id",
		"arguments",
		"output",
		"error_code",
		"error",
		"metadata",
		"task_class",
		"provider",
		"model_key",
		"assistant_empty",
		"item_count",
		"tool_call_count",
		"approval_count",
		"message",
	)
	return result
}

func (p RealtimeEventPayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.AsMap())
}

func (p *RealtimeEventPayload) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("nil RealtimeEventPayload")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*p = RealtimeEventPayload{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = RealtimeEventPayloadFromMap(decoded)
	return nil
}
