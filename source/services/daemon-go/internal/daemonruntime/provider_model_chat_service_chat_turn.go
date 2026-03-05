package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"personalagent/runtime/internal/chatruntime"
	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func (s *ProviderModelChatService) ChatTurn(ctx context.Context, request transport.ChatTurnRequest, correlationID string, onToken func(delta string)) (transport.ChatTurnResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := normalizeTaskClass(request.TaskClass)

	routeProvider := strings.TrimSpace(request.ProviderOverride)
	routeModel := strings.TrimSpace(request.ModelOverride)

	resolvedProvider := ""
	resolvedModel := ""
	if routeProvider != "" || routeModel != "" {
		if routeProvider == "" || routeModel == "" {
			return transport.ChatTurnResponse{}, fmt.Errorf("both provider and model overrides are required together")
		}
		normalizedProvider, err := providerconfig.NormalizeProvider(routeProvider)
		if err != nil {
			return transport.ChatTurnResponse{}, err
		}
		if _, err := s.container.ModelPolicyStore.GetCatalogEntry(ctx, workspace, normalizedProvider, routeModel); err != nil {
			if errors.Is(err, modelpolicy.ErrModelNotFound) {
				return transport.ChatTurnResponse{}, modelpolicy.ErrModelNotFound
			}
			return transport.ChatTurnResponse{}, err
		}
		resolvedProvider = normalizedProvider
		resolvedModel = routeModel
	} else {
		resolved, err := s.resolveModelRoute(ctx, workspace, taskClass)
		if err != nil {
			return transport.ChatTurnResponse{}, err
		}
		resolvedProvider = resolved.Provider
		resolvedModel = resolved.ModelKey
	}

	providerConfig, err := s.container.ProviderConfigStore.Get(ctx, workspace, resolvedProvider)
	if err != nil {
		return transport.ChatTurnResponse{}, fmt.Errorf("load provider config for %s: %w", resolvedProvider, err)
	}

	apiKey := ""
	if secretName := strings.TrimSpace(providerConfig.APIKeySecretName); secretName != "" {
		_, secretValue, err := s.container.SecretResolver.ResolveSecret(ctx, workspace, secretName)
		if err != nil {
			return transport.ChatTurnResponse{}, fmt.Errorf("resolve secret %q: %w", secretName, err)
		}
		apiKey = secretValue
	}

	messages := make([]chatruntime.Message, 0, len(request.Items)+1)
	if strings.TrimSpace(request.SystemPrompt) != "" {
		messages = append(messages, chatruntime.Message{
			Role:    "system",
			Content: strings.TrimSpace(request.SystemPrompt),
		})
	}
	for _, item := range request.Items {
		itemType := strings.ToLower(strings.TrimSpace(item.Type))
		role := strings.ToLower(strings.TrimSpace(item.Role))
		switch itemType {
		case "system_message":
			if role == "" {
				role = "system"
			}
		case "user_message":
			if role == "" {
				role = "user"
			}
		case "assistant_message":
			if role == "" {
				role = "assistant"
			}
		default:
			continue
		}
		content := strings.TrimSpace(item.Content)
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, chatruntime.Message{
			Role:    role,
			Content: content,
		})
	}
	if len(messages) == 0 {
		return transport.ChatTurnResponse{}, fmt.Errorf("chat turn items are required")
	}
	toolSpecs, preferNativeToolCalling := chatRuntimeToolSpecsFromCatalog(request.ToolCatalog)

	builder := &strings.Builder{}
	if err := chatruntime.StreamAssistantResponse(ctx, s.chatHTTPClient(), chatruntime.StreamRequest{
		Provider:                resolvedProvider,
		Endpoint:                providerConfig.Endpoint,
		ModelKey:                resolvedModel,
		APIKey:                  apiKey,
		Messages:                messages,
		ToolSpecs:               toolSpecs,
		PreferNativeToolCalling: preferNativeToolCalling,
	}, func(delta string) error {
		if delta == "" {
			return nil
		}
		builder.WriteString(delta)
		if onToken != nil {
			onToken(delta)
		}
		return nil
	}); err != nil {
		return transport.ChatTurnResponse{}, err
	}

	assistantText := strings.TrimSpace(builder.String())
	return transport.ChatTurnResponse{
		WorkspaceID:   workspace,
		TaskClass:     taskClass,
		Provider:      resolvedProvider,
		ModelKey:      resolvedModel,
		CorrelationID: correlationID,
		Channel:       request.Channel,
		Items: []transport.ChatTurnItem{
			{
				Type:    "assistant_message",
				Role:    "assistant",
				Status:  "completed",
				Content: assistantText,
			},
		},
		TaskRunCorrelation: s.resolveChatTurnTaskRunCorrelation(ctx, correlationID),
	}, nil
}

func chatRuntimeToolSpecsFromCatalog(catalog []transport.ChatTurnToolCatalogEntry) ([]chatruntime.ToolSpec, bool) {
	if len(catalog) == 0 {
		return nil, false
	}
	specs := make([]chatruntime.ToolSpec, 0, len(catalog))
	for _, entry := range catalog {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		spec := chatruntime.ToolSpec{
			Name:        name,
			Description: strings.TrimSpace(entry.Description),
			InputSchema: cloneAnyMap(entry.InputSchema),
		}
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		return nil, false
	}
	return specs, true
}

func (s *ProviderModelChatService) chatHTTPClient() *http.Client {
	if s != nil && s.chatStreamClient != nil {
		return s.chatStreamClient
	}
	return newDaemonRuntimeHTTPClient(defaultProviderChatHTTPTimeout)
}

func (s *ProviderModelChatService) resolveChatTurnTaskRunCorrelation(
	ctx context.Context,
	correlationID string,
) transport.ChatTurnTaskRunCorrelation {
	correlation := strings.TrimSpace(correlationID)
	if correlation == "" || s.container == nil || s.container.DB == nil {
		return transport.ChatTurnTaskRunCorrelation{
			Available: false,
			Source:    "none",
		}
	}

	row := s.container.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE(tr.task_id, ''),
			COALESCE(tr.id, ''),
			COALESCE(t.state, ''),
			COALESCE(tr.state, '')
		FROM audit_log_entries al
		JOIN task_runs tr ON tr.id = al.run_id
		JOIN tasks t ON t.id = tr.task_id
		WHERE al.correlation_id = ?
		  AND TRIM(COALESCE(al.run_id, '')) <> ''
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT 1
	`, correlation)

	var taskID string
	var runID string
	var taskState string
	var runState string
	if err := row.Scan(&taskID, &runID, &taskState, &runState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return transport.ChatTurnTaskRunCorrelation{
				Available: false,
				Source:    "none",
			}
		}
		return transport.ChatTurnTaskRunCorrelation{
			Available: false,
			Source:    "lookup_error",
		}
	}

	taskID = strings.TrimSpace(taskID)
	runID = strings.TrimSpace(runID)
	taskState = strings.TrimSpace(taskState)
	runState = strings.TrimSpace(runState)
	if taskID == "" || runID == "" {
		return transport.ChatTurnTaskRunCorrelation{
			Available: false,
			Source:    "none",
		}
	}

	return transport.ChatTurnTaskRunCorrelation{
		Available: true,
		Source:    "audit_log_entry",
		TaskID:    taskID,
		RunID:     runID,
		TaskState: taskState,
		RunState:  runState,
	}
}
