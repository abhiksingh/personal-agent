package daemonruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (s *UnifiedTurnService) persistTurnItems(
	ctx context.Context,
	request transport.ChatTurnRequest,
	response transport.ChatTurnResponse,
	userItem transport.ChatTurnItem,
) error {
	if s == nil || s.container == nil || s.container.DB == nil {
		return nil
	}

	turnID := mustLocalRandomID("turn")
	baseCreatedAt := time.Now().UTC()
	createdAt := baseCreatedAt.Format(time.RFC3339Nano)
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	channelID := normalizeTurnChannelID(request.Channel.ChannelID)
	connectorID := strings.ToLower(strings.TrimSpace(request.Channel.ConnectorID))
	threadID := strings.TrimSpace(request.Channel.ThreadID)
	correlationID := strings.TrimSpace(response.CorrelationID)
	if correlationID == "" {
		correlationID = strings.TrimSpace(response.CorrelationID)
	}
	if correlationID == "" {
		correlationID = "chat-turn"
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin turn persistence tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspace, workspace, createdAt, createdAt); err != nil {
		return fmt.Errorf("ensure workspace for turn persistence: %w", err)
	}

	index := 0
	if strings.TrimSpace(userItem.Type) != "" {
		itemCreatedAt := baseCreatedAt.Add(time.Duration(index) * time.Nanosecond).Format(time.RFC3339Nano)
		if err := insertTurnItemRow(ctx, tx, turnItemInsertInput{
			RecordID:      mustLocalRandomID("turnitem"),
			TurnID:        turnID,
			WorkspaceID:   workspace,
			TaskClass:     normalizeTaskClass(request.TaskClass),
			CorrelationID: correlationID,
			ChannelID:     channelID,
			ConnectorID:   connectorID,
			ThreadID:      threadID,
			ItemIndex:     index,
			Item:          userItem,
			TaskRun:       transport.ChatTurnTaskRunCorrelation{Available: false, Source: "none"},
			CreatedAt:     itemCreatedAt,
		}); err != nil {
			return err
		}
		index++
	}

	for _, item := range response.Items {
		itemCreatedAt := baseCreatedAt.Add(time.Duration(index) * time.Nanosecond).Format(time.RFC3339Nano)
		if err := insertTurnItemRow(ctx, tx, turnItemInsertInput{
			RecordID:      mustLocalRandomID("turnitem"),
			TurnID:        turnID,
			WorkspaceID:   workspace,
			TaskClass:     normalizeTaskClass(response.TaskClass),
			CorrelationID: correlationID,
			ChannelID:     channelID,
			ConnectorID:   connectorID,
			ThreadID:      threadID,
			ItemIndex:     index,
			Item:          item,
			TaskRun:       response.TaskRunCorrelation,
			CreatedAt:     itemCreatedAt,
		}); err != nil {
			return err
		}
		index++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit turn persistence tx: %w", err)
	}
	return nil
}

type turnItemInsertInput struct {
	RecordID      string
	TurnID        string
	WorkspaceID   string
	TaskClass     string
	CorrelationID string
	ChannelID     string
	ConnectorID   string
	ThreadID      string
	ItemIndex     int
	Item          transport.ChatTurnItem
	TaskRun       transport.ChatTurnTaskRunCorrelation
	CreatedAt     string
}

func insertTurnItemRow(ctx context.Context, tx *sql.Tx, input turnItemInsertInput) error {
	argumentsJSON := marshalOptionalMap(input.Item.Arguments)
	outputJSON := marshalOptionalMap(input.Item.Output)
	metadataJSON := marshalOptionalMap(input.Item.Metadata.AsMap())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_turn_items(
			id,
			turn_id,
			workspace_id,
			task_class,
			correlation_id,
			channel_id,
			connector_id,
			thread_id,
			item_index,
			item_type,
			role,
			status,
			content,
			tool_name,
			tool_call_id,
			arguments_json,
			output_json,
			error_code,
			error_message,
			approval_request_id,
			metadata_json,
			task_id,
			run_id,
			task_state,
			run_state,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		input.RecordID,
		input.TurnID,
		input.WorkspaceID,
		input.TaskClass,
		input.CorrelationID,
		input.ChannelID,
		nullIfEmptyText(input.ConnectorID),
		nullIfEmptyText(input.ThreadID),
		input.ItemIndex,
		strings.ToLower(strings.TrimSpace(input.Item.Type)),
		nullIfEmptyText(strings.ToLower(strings.TrimSpace(input.Item.Role))),
		nullIfEmptyText(strings.ToLower(strings.TrimSpace(input.Item.Status))),
		nullIfEmptyText(strings.TrimSpace(input.Item.Content)),
		nullIfEmptyText(strings.TrimSpace(input.Item.ToolName)),
		nullIfEmptyText(strings.TrimSpace(input.Item.ToolCallID)),
		nullIfEmptyText(argumentsJSON),
		nullIfEmptyText(outputJSON),
		nullIfEmptyText(strings.TrimSpace(input.Item.ErrorCode)),
		nullIfEmptyText(strings.TrimSpace(input.Item.ErrorMessage)),
		nullIfEmptyText(strings.TrimSpace(input.Item.ApprovalRequestID)),
		nullIfEmptyText(metadataJSON),
		nullIfEmptyText(strings.TrimSpace(input.TaskRun.TaskID)),
		nullIfEmptyText(strings.TrimSpace(input.TaskRun.RunID)),
		nullIfEmptyText(strings.TrimSpace(input.TaskRun.TaskState)),
		nullIfEmptyText(strings.TrimSpace(input.TaskRun.RunState)),
		input.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert turn item row: %w", err)
	}
	return nil
}

func marshalOptionalMap(value map[string]any) string {
	if len(value) == 0 {
		return ""
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}

func decodeOptionalMap(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return map[string]any{"decode_error": "invalid_json"}
	}
	return decoded
}

func nullIfEmptyText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
