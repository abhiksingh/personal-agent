package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

type UnifiedTurnService struct {
	container *ServiceContainer
	modelChat transport.ChatService
	agent     transport.AgentService
	uiStatus  transport.UIStatusService
	policy    *ToolPolicyEngine
}

type plannerDirective struct {
	Type      string         `json:"type"`
	Content   string         `json:"content,omitempty"`
	ToolName  string         `json:"tool_name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolArgumentSpec struct {
	Type        string
	Required    bool
	EnumOptions []string
	Description string
}

type modelToolDefinition struct {
	Name              string
	CapabilityKeys    []string
	Description       string
	Arguments         map[string]toolArgumentSpec
	BuildNativeAction func(arguments map[string]any) (*transport.AgentNativeAction, error)
}

type contextAssembly struct {
	Summary             string
	RetrievalTarget     int
	RetrievalUsed       int
	CompactionTriggered bool
}

var defaultChatPersonaStylePrompt = "You are a reliable human-like assistant. Be clear, concise, and action-oriented."

var defaultChatPersonaGuardrails = []string{
	"Do not claim actions succeeded unless tool results confirm success.",
	"When approval is required, explicitly ask for approval and include the request id.",
	"Use deterministic, plain language with no hidden operational assumptions.",
}

var supportedModelToolCapabilities = []string{
	"mail_draft",
	"mail_send",
	"mail_reply",
	"mail_unread_summary",
	"calendar_create",
	"calendar_update",
	"calendar_cancel",
	"browser_open",
	"browser_extract",
	"browser_close",
	"finder_find",
	"finder_list",
	"finder_preview",
	"finder_delete",
	"channel_messages_send",
	"channel_twilio_sms_send",
	"messages_send_imessage",
	"messages_send_sms",
}

const unifiedTurnMaxToolCalls = 4
const unifiedTurnPlannerRepairMaxRetries = 2

var _ transport.ChatService = (*UnifiedTurnService)(nil)

func NewUnifiedTurnService(
	container *ServiceContainer,
	modelChat transport.ChatService,
	agent transport.AgentService,
	uiStatus transport.UIStatusService,
	delegation transport.DelegationService,
) (*UnifiedTurnService, error) {
	if modelChat == nil {
		return nil, fmt.Errorf("model chat service is required")
	}
	return &UnifiedTurnService{
		container: container,
		modelChat: modelChat,
		agent:     agent,
		uiStatus:  uiStatus,
		policy:    NewToolPolicyEngine(delegation),
	}, nil
}

func (s *UnifiedTurnService) QueryChatTurnHistory(ctx context.Context, request transport.ChatTurnHistoryRequest) (transport.ChatTurnHistoryResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.ChatTurnHistoryResponse{}, fmt.Errorf("chat history store is not configured")
	}
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	limit := request.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT
			id,
			turn_id,
			workspace_id,
			task_class,
			correlation_id,
			channel_id,
			COALESCE(connector_id, ''),
			COALESCE(thread_id, ''),
			item_index,
			item_type,
			COALESCE(role, ''),
			COALESCE(status, ''),
			COALESCE(content, ''),
			COALESCE(tool_name, ''),
			COALESCE(tool_call_id, ''),
			COALESCE(arguments_json, ''),
			COALESCE(output_json, ''),
			COALESCE(error_code, ''),
			COALESCE(error_message, ''),
			COALESCE(approval_request_id, ''),
			COALESCE(metadata_json, ''),
			COALESCE(task_id, ''),
			COALESCE(run_id, ''),
			COALESCE(task_state, ''),
			COALESCE(run_state, ''),
			created_at
		FROM chat_turn_items
		WHERE workspace_id = ?
	`
	params := []any{workspace}

	if channelID := strings.TrimSpace(request.ChannelID); channelID != "" {
		query += " AND channel_id = ?"
		params = append(params, strings.ToLower(channelID))
	}
	if connectorID := strings.TrimSpace(request.ConnectorID); connectorID != "" {
		query += " AND connector_id = ?"
		params = append(params, strings.ToLower(connectorID))
	}
	if threadID := strings.TrimSpace(request.ThreadID); threadID != "" {
		query += " AND thread_id = ?"
		params = append(params, threadID)
	}
	if correlationID := strings.TrimSpace(request.CorrelationID); correlationID != "" {
		query += " AND correlation_id = ?"
		params = append(params, correlationID)
	}

	beforeCreatedAt := strings.TrimSpace(request.BeforeCreatedAt)
	beforeItemID := strings.TrimSpace(request.BeforeItemID)
	if beforeCreatedAt != "" {
		if beforeItemID == "" {
			beforeItemID = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
		}
		query += " AND ((created_at < ?) OR (created_at = ? AND id < ?))"
		params = append(params, beforeCreatedAt, beforeCreatedAt, beforeItemID)
	}

	query += " ORDER BY created_at DESC, item_index DESC, id DESC LIMIT ?"
	params = append(params, limit+1)

	rows, err := s.container.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return transport.ChatTurnHistoryResponse{}, fmt.Errorf("query chat turn history: %w", err)
	}
	defer rows.Close()

	records := make([]transport.ChatTurnHistoryRecord, 0, limit+1)
	for rows.Next() {
		var (
			record        transport.ChatTurnHistoryRecord
			itemType      string
			argumentsJSON string
			outputJSON    string
			metadataJSON  string
			taskID        string
			runID         string
			taskState     string
			runState      string
		)
		if err := rows.Scan(
			&record.RecordID,
			&record.TurnID,
			&record.WorkspaceID,
			&record.TaskClass,
			&record.CorrelationID,
			&record.ChannelID,
			&record.ConnectorID,
			&record.ThreadID,
			&record.ItemIndex,
			&itemType,
			&record.Item.Role,
			&record.Item.Status,
			&record.Item.Content,
			&record.Item.ToolName,
			&record.Item.ToolCallID,
			&argumentsJSON,
			&outputJSON,
			&record.Item.ErrorCode,
			&record.Item.ErrorMessage,
			&record.Item.ApprovalRequestID,
			&metadataJSON,
			&taskID,
			&runID,
			&taskState,
			&runState,
			&record.CreatedAt,
		); err != nil {
			return transport.ChatTurnHistoryResponse{}, fmt.Errorf("scan chat turn history row: %w", err)
		}
		record.Item.Type = itemType
		record.Item.ItemID = record.RecordID
		record.TaskRunReference = transport.ChatTurnTaskRunCorrelation{
			Available: strings.TrimSpace(taskID) != "" && strings.TrimSpace(runID) != "",
			Source:    "turn_ledger",
			TaskID:    strings.TrimSpace(taskID),
			RunID:     strings.TrimSpace(runID),
			TaskState: strings.TrimSpace(taskState),
			RunState:  strings.TrimSpace(runState),
		}
		record.Item.Arguments = decodeOptionalMap(argumentsJSON)
		record.Item.Output = decodeOptionalMap(outputJSON)
		record.Item.Metadata = transport.ChatTurnItemMetadataFromMap(decodeOptionalMap(metadataJSON))
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return transport.ChatTurnHistoryResponse{}, fmt.Errorf("iterate chat turn history rows: %w", err)
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	response := transport.ChatTurnHistoryResponse{
		WorkspaceID: workspace,
		Items:       records,
		HasMore:     hasMore,
	}
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		response.NextCursorCreatedAt = strings.TrimSpace(last.CreatedAt)
		response.NextCursorItemID = strings.TrimSpace(last.RecordID)
	}
	return response, nil
}
