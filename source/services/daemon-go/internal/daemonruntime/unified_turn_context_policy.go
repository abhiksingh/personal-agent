package daemonruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	contextrepo "personalagent/runtime/internal/core/repository/context"
	contextservice "personalagent/runtime/internal/core/service/context"
	coretypes "personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *UnifiedTurnService) GetChatPersonaPolicy(ctx context.Context, request transport.ChatPersonaPolicyRequest) (transport.ChatPersonaPolicyResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	principal := strings.TrimSpace(request.PrincipalActorID)
	channel := normalizePersonaPolicyChannelScope(request.ChannelID)

	if s == nil || s.container == nil || s.container.DB == nil {
		return defaultPersonaPolicy(workspace, principal, channel, "default"), nil
	}

	candidates := [][2]string{{principal, channel}, {principal, ""}, {"", channel}, {"", ""}}
	for _, candidate := range candidates {
		policy, found, err := s.lookupPersonaPolicy(ctx, workspace, candidate[0], candidate[1])
		if err != nil {
			return transport.ChatPersonaPolicyResponse{}, err
		}
		if found {
			return policy, nil
		}
	}
	return defaultPersonaPolicy(workspace, principal, channel, "default"), nil
}

func (s *UnifiedTurnService) UpsertChatPersonaPolicy(ctx context.Context, request transport.ChatPersonaPolicyUpsertRequest) (transport.ChatPersonaPolicyResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.ChatPersonaPolicyResponse{}, fmt.Errorf("chat persona policy store is not configured")
	}
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	principal := strings.TrimSpace(request.PrincipalActorID)
	channel := normalizePersonaPolicyChannelScope(request.ChannelID)
	stylePrompt := strings.TrimSpace(request.StylePrompt)
	if stylePrompt == "" {
		return transport.ChatPersonaPolicyResponse{}, fmt.Errorf("style_prompt is required")
	}
	guardrails := normalizeGuardrails(request.Guardrails)
	guardrailsJSON, err := json.Marshal(guardrails)
	if err != nil {
		return transport.ChatPersonaPolicyResponse{}, fmt.Errorf("marshal persona guardrails: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.container.DB.ExecContext(ctx, `
		INSERT INTO chat_persona_policies(
			id,
			workspace_id,
			principal_actor_id,
			channel_id,
			style_prompt,
			guardrails_json,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, principal_actor_id, channel_id)
		DO UPDATE SET
			style_prompt = excluded.style_prompt,
			guardrails_json = excluded.guardrails_json,
			updated_at = excluded.updated_at
	`, fmt.Sprintf("persona.%s.%s.%s", workspace, safePolicyToken(principal, "default"), safePolicyToken(channel, "all")), workspace, principal, channel, stylePrompt, string(guardrailsJSON), now, now); err != nil {
		return transport.ChatPersonaPolicyResponse{}, fmt.Errorf("upsert chat persona policy: %w", err)
	}

	return transport.ChatPersonaPolicyResponse{
		WorkspaceID:      workspace,
		PrincipalActorID: principal,
		ChannelID:        channel,
		StylePrompt:      stylePrompt,
		Guardrails:       guardrails,
		Source:           "persisted",
		UpdatedAt:        now,
	}, nil
}

func (s *UnifiedTurnService) lookupPersonaPolicy(
	ctx context.Context,
	workspaceID string,
	principalActorID string,
	channelID string,
) (transport.ChatPersonaPolicyResponse, bool, error) {
	var (
		stylePrompt   string
		guardrailsRaw string
		updatedAt     string
	)
	err := s.container.DB.QueryRowContext(ctx, `
		SELECT
			style_prompt,
			COALESCE(guardrails_json, '[]'),
			updated_at
		FROM chat_persona_policies
		WHERE workspace_id = ?
		  AND principal_actor_id = ?
		  AND channel_id = ?
		LIMIT 1
	`, workspaceID, principalActorID, channelID).Scan(&stylePrompt, &guardrailsRaw, &updatedAt)
	if err == sql.ErrNoRows {
		return transport.ChatPersonaPolicyResponse{}, false, nil
	}
	if err != nil {
		return transport.ChatPersonaPolicyResponse{}, false, fmt.Errorf("lookup chat persona policy: %w", err)
	}
	guardrails := []string{}
	_ = json.Unmarshal([]byte(guardrailsRaw), &guardrails)
	return transport.ChatPersonaPolicyResponse{
		WorkspaceID:      workspaceID,
		PrincipalActorID: principalActorID,
		ChannelID:        channelID,
		StylePrompt:      strings.TrimSpace(stylePrompt),
		Guardrails:       normalizeGuardrails(guardrails),
		Source:           "persisted",
		UpdatedAt:        strings.TrimSpace(updatedAt),
	}, true, nil
}

func (s *UnifiedTurnService) resolvePersonaPolicy(ctx context.Context, request transport.ChatPersonaPolicyRequest) (transport.ChatPersonaPolicyResponse, error) {
	policy, err := s.GetChatPersonaPolicy(ctx, request)
	if err != nil {
		return transport.ChatPersonaPolicyResponse{}, err
	}
	if strings.TrimSpace(policy.StylePrompt) == "" {
		policy.StylePrompt = defaultChatPersonaStylePrompt
	}
	if len(policy.Guardrails) == 0 {
		policy.Guardrails = append([]string(nil), defaultChatPersonaGuardrails...)
	}
	return policy, nil
}

func defaultPersonaPolicy(workspaceID string, principalActorID string, channelID string, source string) transport.ChatPersonaPolicyResponse {
	return transport.ChatPersonaPolicyResponse{
		WorkspaceID:      normalizeWorkspaceID(workspaceID),
		PrincipalActorID: strings.TrimSpace(principalActorID),
		ChannelID:        normalizePersonaPolicyChannelScope(channelID),
		StylePrompt:      defaultChatPersonaStylePrompt,
		Guardrails:       append([]string(nil), defaultChatPersonaGuardrails...),
		Source:           strings.TrimSpace(source),
	}
}

func normalizeGuardrails(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func safePolicyToken(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	replacer := strings.NewReplacer("/", "_", " ", "_", ".", "_", "-", "_")
	resolved := strings.ToLower(replacer.Replace(trimmed))
	if resolved == "" {
		return fallback
	}
	return resolved
}

func (s *UnifiedTurnService) assembleContext(ctx context.Context, request transport.ChatTurnRequest) (contextAssembly, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return contextAssembly{}, nil
	}
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	taskClass := normalizeTaskClass(request.TaskClass)

	store := contextrepo.NewSQLiteBudgetTelemetryStore(s.container.DB)
	budgeter := contextservice.NewTaskClassBudgeter(store)
	budget, err := budgeter.Compute(ctx, workspace, taskClass, coretypes.ContextBudgetInput{
		ContextWindow: 8192,
		OutputLimit:   1024,
		DeepAnalysis:  false,
	})
	if err != nil {
		return contextAssembly{}, err
	}

	ownerActorID := strings.TrimSpace(request.ActingAsActorID)
	if ownerActorID == "" {
		ownerActorID = strings.TrimSpace(request.SubjectActorID)
	}
	if ownerActorID == "" {
		ownerActorID = strings.TrimSpace(request.RequestedByActorID)
	}
	if ownerActorID == "" {
		return contextAssembly{RetrievalTarget: budget.RetrievalTarget}, nil
	}

	rows, err := s.container.DB.QueryContext(ctx, `
		SELECT
			id,
			COALESCE(scope_type, ''),
			COALESCE(key, ''),
			COALESCE(value_json, ''),
			COALESCE(status, ''),
			COALESCE(source_summary, ''),
			COALESCE(updated_at, '')
		FROM memory_items
		WHERE workspace_id = ?
		  AND owner_principal_actor_id = ?
		  AND UPPER(COALESCE(status, '')) = 'ACTIVE'
		ORDER BY updated_at DESC, id DESC
		LIMIT 200
	`, workspace, ownerActorID)
	if err != nil {
		return contextAssembly{}, fmt.Errorf("query memory context items: %w", err)
	}
	defer rows.Close()

	retrievalLines := make([]string, 0, 64)
	retrievalTokens := 0
	memoryRecords := make([]coretypes.MemoryRecord, 0, 64)
	for rows.Next() {
		var (
			memoryID  string
			scopeType string
			key       string
			valueJSON string
			status    string
			sourceRef string
			updatedAt string
		)
		if err := rows.Scan(&memoryID, &scopeType, &key, &valueJSON, &status, &sourceRef, &updatedAt); err != nil {
			return contextAssembly{}, fmt.Errorf("scan memory context row: %w", err)
		}
		line := strings.TrimSpace(fmt.Sprintf("[%s] %s: %s", scopeType, key, valueJSON))
		if line == "" {
			continue
		}
		lineTokens := wordCount(line)
		if budget.RetrievalTarget > 0 && retrievalTokens+lineTokens > budget.RetrievalTarget {
			continue
		}
		retrievalLines = append(retrievalLines, line)
		retrievalTokens += lineTokens
		parsedUpdatedAt, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(updatedAt))
		memoryRecords = append(memoryRecords, coretypes.MemoryRecord{
			ID:            strings.TrimSpace(memoryID),
			Kind:          strings.TrimSpace(scopeType),
			Status:        strings.TrimSpace(status),
			IsCanonical:   strings.EqualFold(strings.TrimSpace(scopeType), "fact") || strings.EqualFold(strings.TrimSpace(scopeType), "rule"),
			TokenEstimate: lineTokens,
			Content:       line,
			SourceRef:     strings.TrimSpace(sourceRef),
			LastUpdatedAt: parsedUpdatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return contextAssembly{}, fmt.Errorf("iterate memory context rows: %w", err)
	}

	compactor := contextservice.NewMemoryCompactor(nil)
	compaction := compactor.Compact(memoryRecords, coretypes.MemoryCompactionConfig{
		TokenThreshold: maxInt(1, budget.RetrievalTarget),
		StaleAfter:     72 * time.Hour,
	})
	compactionTriggered := len(compaction.Summaries) > 0
	for _, summary := range compaction.Summaries {
		retrievalLines = append(retrievalLines, strings.TrimSpace(summary.Content))
		retrievalTokens += maxInt(1, summary.TokenEstimate)
	}

	return contextAssembly{
		Summary:             strings.Join(retrievalLines, "\n"),
		RetrievalTarget:     budget.RetrievalTarget,
		RetrievalUsed:       retrievalTokens,
		CompactionTriggered: compactionTriggered,
	}, nil
}

func (s *UnifiedTurnService) recordContextSample(
	ctx context.Context,
	workspaceID string,
	taskClass string,
	modelKey string,
	assembly contextAssembly,
	plannerText string,
	assistantText string,
) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return
	}
	store := contextrepo.NewSQLiteBudgetTelemetryStore(s.container.DB)
	_ = store.RecordContextBudgetSample(ctx, coretypes.ContextBudgetSample{
		WorkspaceID:      normalizeWorkspaceID(workspaceID),
		TaskClass:        normalizeTaskClass(taskClass),
		ModelKey:         strings.TrimSpace(modelKey),
		ContextWindow:    8192,
		OutputLimit:      1024,
		DeepAnalysis:     false,
		RemainingBudget:  maxInt(0, 8192-assembly.RetrievalUsed),
		RetrievalTarget:  maxInt(0, assembly.RetrievalTarget),
		RetrievalUsed:    maxInt(0, assembly.RetrievalUsed),
		PromptTokens:     wordCount(plannerText),
		CompletionTokens: wordCount(assistantText),
		RecordedAt:       time.Now().UTC(),
	})
}

func normalizeTurnChannelID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "message":
		return "message"
	case "voice":
		return "voice"
	default:
		return "app"
	}
}

func normalizePersonaPolicyChannelScope(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "", "all", "*":
		return ""
	default:
		return normalizeTurnChannelID(trimmed)
	}
}

func executionOriginForTurnChannel(channelID string) string {
	switch normalizeTurnChannelID(channelID) {
	case "voice":
		return string(coretypes.ExecutionOriginVoice)
	default:
		return string(coretypes.ExecutionOriginApp)
	}
}

func plannerRemediationHint(stopReason string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(stopReason)) {
	case "planner_output_invalid":
		return map[string]any{
			"code":             "planner_output_invalid",
			"domain":           "chat_orchestration",
			"summary":          "The planner output was malformed. Retry the turn or switch to a different chat model route.",
			"primary_action":   "retry_turn",
			"secondary_action": "open_models",
		}
	case "planner_no_action":
		return map[string]any{
			"code":             "planner_no_action",
			"domain":           "chat_orchestration",
			"summary":          "The planner did not return a tool action. Retry with a clearer request or adjust model routing.",
			"primary_action":   "retry_turn",
			"secondary_action": "open_models",
		}
	default:
		return nil
	}
}

func modelRouteRemediationHint(err error) map[string]any {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(lower, "no enabled action-capable models") || strings.Contains(lower, "no enabled models with ready provider configuration") {
		return map[string]any{
			"code":             "model_route_unavailable",
			"domain":           "models",
			"summary":          "No enabled and ready chat model route is available for action orchestration.",
			"primary_action":   "open_models",
			"secondary_action": "recheck_route",
		}
	}
	return nil
}

func toolFailureRemediationHint(tool modelToolDefinition, errorCode string) map[string]any {
	domain := "connectors"
	primaryAction := "open_connectors"
	secondaryAction := "retry_turn"
	for _, capability := range tool.CapabilityKeys {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(capability)), "messages_") ||
			strings.HasPrefix(strings.ToLower(strings.TrimSpace(capability)), "channel_") {
			domain = "channels"
			primaryAction = "open_channels"
			break
		}
	}
	summary := "Tool execution failed while dispatching through connector runtime."
	if strings.EqualFold(strings.TrimSpace(errorCode), "tool_executor_unavailable") {
		summary = "Tool execution is unavailable. Check daemon connector runtime and retry."
	}
	return map[string]any{
		"code":             "tool_execution_failure",
		"domain":           domain,
		"summary":          summary,
		"primary_action":   primaryAction,
		"secondary_action": secondaryAction,
	}
}

func remediationSummary(remediation map[string]any, fallback string) string {
	if remediation == nil {
		return strings.TrimSpace(fallback)
	}
	if summary := strings.TrimSpace(fmt.Sprintf("%v", remediation["summary"])); summary != "" {
		return summary
	}
	return strings.TrimSpace(fallback)
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func wordCount(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	return len(strings.Fields(trimmed))
}

func mustLocalRandomID(prefix string) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%s-%d", strings.TrimSpace(prefix), time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(prefix), hex.EncodeToString(random))
}
