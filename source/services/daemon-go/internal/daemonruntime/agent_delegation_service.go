package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	repoauthz "personalagent/runtime/internal/core/repository/authz"
	"personalagent/runtime/internal/core/service/agentexec"
	authzservice "personalagent/runtime/internal/core/service/authz"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type AgentDelegationService struct {
	container  *ServiceContainer
	engine     *agentexec.SQLiteExecutionEngine
	authorizer *authzservice.ActingAsAuthorizer
}

var _ transport.AgentService = (*AgentDelegationService)(nil)
var _ transport.DelegationService = (*AgentDelegationService)(nil)

type agentMessageDispatcher struct {
	comm transport.CommService
}

func NewAgentDelegationService(container *ServiceContainer) (*AgentDelegationService, error) {
	if container == nil || container.DB == nil {
		return nil, fmt.Errorf("service container with db is required")
	}
	if container.ConnectorRegistry == nil {
		return nil, fmt.Errorf("connector registry is required")
	}
	if err := registerDaemonConnectorAdapters(container); err != nil {
		return nil, err
	}

	dispatcher := NewSupervisorConnectorStepDispatcher(container.PluginSupervisor, container.ConnectorRegistry)
	engine := agentexec.NewSQLiteExecutionEngine(container.DB, newDispatchConnectorSelector(dispatcher))
	engine.SetIntentInterpreter(agentexec.NewModelAssistedIntentInterpreter(
		newDaemonIntentModelExtractor(container),
		0.6,
	))
	authorizer := authzservice.NewActingAsAuthorizer(repoauthz.NewDelegationRuleStoreSQLite(container.DB))
	return &AgentDelegationService{
		container:  container,
		engine:     engine,
		authorizer: authorizer,
	}, nil
}

func (s *AgentDelegationService) SetCommService(comm transport.CommService) {
	if s == nil || s.engine == nil {
		return
	}
	s.engine.SetMessageDispatcher(agentMessageDispatcher{comm: comm})
}

func (d agentMessageDispatcher) DispatchMessage(ctx context.Context, request agentexec.MessageDispatchRequest) (agentexec.MessageDispatchResult, error) {
	if d.comm == nil {
		return agentexec.MessageDispatchResult{}, fmt.Errorf("comm service is not configured")
	}
	response, err := d.comm.SendComm(ctx, transport.CommSendRequest{
		WorkspaceID:   strings.TrimSpace(request.WorkspaceID),
		OperationID:   strings.TrimSpace(request.OperationID),
		SourceChannel: strings.TrimSpace(request.SourceChannel),
		Destination:   strings.TrimSpace(request.Destination),
		Message:       strings.TrimSpace(request.MessageBody),
	})
	if err != nil {
		return agentexec.MessageDispatchResult{}, err
	}
	if !response.Success {
		errText := strings.TrimSpace(response.Error)
		if errText == "" {
			errText = "comm send failed"
		}
		return agentexec.MessageDispatchResult{}, fmt.Errorf("%s", errText)
	}
	if !response.Result.Delivered {
		return agentexec.MessageDispatchResult{}, fmt.Errorf("comm send did not deliver message")
	}

	channel := strings.ToLower(strings.TrimSpace(response.Result.Channel))
	if channel == "" {
		channel = strings.ToLower(strings.TrimSpace(request.SourceChannel))
	}
	receipt := strings.TrimSpace(response.Result.ProviderReceipt)
	summary := fmt.Sprintf("message dispatched via %s", channel)
	if receipt != "" {
		summary = fmt.Sprintf("message dispatched via %s (%s)", channel, receipt)
	}
	return agentexec.MessageDispatchResult{
		Channel:         channel,
		ProviderReceipt: receipt,
		Summary:         summary,
	}, nil
}

func registerDaemonConnectorAdapters(container *ServiceContainer) error {
	registrations := []struct {
		registerErr error
	}{
		{registerErr: container.ConnectorRegistry.Register(mailadapter.NewAdapterWithDBPath("mail.daemon", container.DBPath))},
		{registerErr: container.ConnectorRegistry.Register(calendaradapter.NewAdapter("calendar.daemon"))},
		{registerErr: container.ConnectorRegistry.Register(browseradapter.NewAdapter("browser.daemon"))},
		{registerErr: container.ConnectorRegistry.Register(finderadapter.NewAdapter("finder.daemon"))},
	}

	for _, registration := range registrations {
		if registration.registerErr == nil {
			continue
		}
		if strings.Contains(registration.registerErr.Error(), "already registered") {
			continue
		}
		return registration.registerErr
	}
	return nil
}

func (s *AgentDelegationService) RunAgent(ctx context.Context, request transport.AgentRunRequest) (transport.AgentRunResponse, error) {
	if s.engine == nil {
		return transport.AgentRunResponse{}, fmt.Errorf("execution engine is not configured")
	}
	requestText := strings.TrimSpace(request.RequestText)
	nativeAction, nativeActionErr := mapTransportNativeAction(request.NativeAction)
	if nativeActionErr != nil {
		return transport.AgentRunResponse{}, nativeActionErr
	}
	if strings.EqualFold(strings.TrimSpace(request.Origin), "chat_unified_tool") && nativeAction == nil {
		return transport.AgentRunResponse{}, fmt.Errorf("native_action is required for chat unified tool execution")
	}
	if requestText == "" && nativeAction == nil {
		return transport.AgentRunResponse{}, fmt.Errorf("--request or --native-action is required")
	}

	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	requestedBy := normalizeActorID(request.RequestedByActorID, "actor.requester")
	subject := normalizeActorID(request.SubjectActorID, requestedBy)
	actingAs := normalizeActorID(request.ActingAsActorID, subject)

	decision, err := s.authorizer.CanActAs(ctx, types.ActingAsRequest{
		WorkspaceID:        workspaceID,
		RequestedByActorID: requestedBy,
		ActingAsActorID:    actingAs,
		ScopeType:          "EXECUTION",
	})
	if err != nil {
		return transport.AgentRunResponse{}, err
	}
	if !decision.Allowed {
		return transport.AgentRunResponse{}, fmt.Errorf("acting_as denied: %s", decision.Reason)
	}

	result, err := s.engine.Execute(ctx, agentexec.ExecuteRequest{
		WorkspaceID:            workspaceID,
		RequestText:            requestText,
		NativeAction:           nativeAction,
		RequestedByActorID:     requestedBy,
		SubjectActorID:         subject,
		ActingAsActorID:        actingAs,
		Origin:                 types.ExecutionOrigin(strings.TrimSpace(request.Origin)),
		InAppApprovalConfirmed: request.InAppApprovalConfirmed,
		CorrelationID:          strings.TrimSpace(request.CorrelationID),
		ApprovalPhrase:         strings.TrimSpace(request.ApprovalPhrase),
		PreferredAdapterID:     strings.TrimSpace(request.PreferredAdapterID),
	})
	if err != nil {
		return transport.AgentRunResponse{}, err
	}
	return agentRunResponse(result), nil
}

func (s *AgentDelegationService) ApproveAgent(ctx context.Context, request transport.AgentApproveRequest) (transport.AgentRunResponse, error) {
	if s.engine == nil {
		return transport.AgentRunResponse{}, fmt.Errorf("execution engine is not configured")
	}
	if strings.TrimSpace(request.ApprovalRequestID) == "" {
		return transport.AgentRunResponse{}, fmt.Errorf("--approval-id is required")
	}
	if strings.TrimSpace(request.DecisionByActorID) == "" {
		return transport.AgentRunResponse{}, fmt.Errorf("--actor-id is required")
	}

	result, err := s.engine.ResumeAfterApproval(ctx, agentexec.ResumeRequest{
		WorkspaceID:       normalizeWorkspaceID(request.WorkspaceID),
		ApprovalRequestID: strings.TrimSpace(request.ApprovalRequestID),
		DecisionByActorID: strings.TrimSpace(request.DecisionByActorID),
		Phrase:            request.Phrase,
		CorrelationID:     strings.TrimSpace(request.CorrelationID),
	})
	if err != nil {
		return transport.AgentRunResponse{}, err
	}
	return agentRunResponse(result), nil
}

func (s *AgentDelegationService) ExecuteQueuedTaskRun(ctx context.Context, runID string, correlationID string) (transport.AgentRunResponse, error) {
	if s.engine == nil {
		return transport.AgentRunResponse{}, fmt.Errorf("execution engine is not configured")
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return transport.AgentRunResponse{}, fmt.Errorf("run id is required")
	}

	var (
		workspaceID string
		taskID      string
		requestedBy string
		subject     string
		actingAs    string
		title       string
		description string
		channel     string
	)
	err := s.container.DB.QueryRowContext(ctx, `
		SELECT
			tr.workspace_id,
			tr.task_id,
			t.requested_by_actor_id,
			t.subject_principal_actor_id,
			tr.acting_as_actor_id,
			COALESCE(t.title, ''),
			COALESCE(t.description, ''),
			COALESCE(t.channel, '')
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE tr.id = ?
	`, trimmedRunID).Scan(
		&workspaceID,
		&taskID,
		&requestedBy,
		&subject,
		&actingAs,
		&title,
		&description,
		&channel,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return transport.AgentRunResponse{}, fmt.Errorf("task run not found: %s", trimmedRunID)
		}
		return transport.AgentRunResponse{}, fmt.Errorf("load queued task run: %w", err)
	}

	requestText := strings.TrimSpace(description)
	if requestText == "" {
		requestText = strings.TrimSpace(title)
	}
	decision, authErr := s.authorizer.CanActAs(ctx, types.ActingAsRequest{
		WorkspaceID:        normalizeWorkspaceID(workspaceID),
		RequestedByActorID: strings.TrimSpace(requestedBy),
		ActingAsActorID:    strings.TrimSpace(actingAs),
		ScopeType:          "EXECUTION",
	})
	if authErr != nil {
		return transport.AgentRunResponse{}, authErr
	}
	if !decision.Allowed {
		return transport.AgentRunResponse{}, fmt.Errorf("acting_as denied: %s", decision.Reason)
	}

	result, execErr := s.engine.ExecutePersistedRun(ctx, agentexec.PersistedRunRequest{
		WorkspaceID:        strings.TrimSpace(workspaceID),
		TaskID:             strings.TrimSpace(taskID),
		RunID:              trimmedRunID,
		RequestText:        requestText,
		RequestedByActorID: strings.TrimSpace(requestedBy),
		SubjectActorID:     strings.TrimSpace(subject),
		ActingAsActorID:    strings.TrimSpace(actingAs),
		SourceChannel:      strings.TrimSpace(channel),
		CorrelationID:      strings.TrimSpace(correlationID),
	})
	if execErr != nil {
		return transport.AgentRunResponse{}, execErr
	}
	return agentRunResponse(result), nil
}
