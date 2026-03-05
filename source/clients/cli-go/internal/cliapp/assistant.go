package cliapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

type assistantFlow string

const (
	assistantFlowSetup    assistantFlow = "setup"
	assistantFlowTask     assistantFlow = "task_submit"
	assistantFlowApproval assistantFlow = "approval_decision"
	assistantFlowCommSend assistantFlow = "comm_send"
)

type assistantResponse struct {
	SchemaVersion string        `json:"schema_version"`
	Flow          assistantFlow `json:"flow,omitempty"`
	WorkspaceID   string        `json:"workspace_id"`
	Success       bool          `json:"success"`
	Cancelled     bool          `json:"cancelled"`
	Backtracks    int           `json:"backtracks"`
	Summary       string        `json:"summary"`
	Result        any           `json:"result,omitempty"`
	Error         string        `json:"error,omitempty"`
	Remediation   []string      `json:"remediation,omitempty"`
}

type assistantPromptField struct {
	Key          string
	Label        string
	DefaultValue string
	Required     bool
	Validate     func(value string) error
}

type assistantPrompter struct {
	reader     *bufio.Reader
	writer     io.Writer
	backtracks int
}

func newAssistantPrompter(stdin io.Reader, writer io.Writer) *assistantPrompter {
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	if writer == nil {
		writer = io.Discard
	}
	return &assistantPrompter{
		reader: bufio.NewReader(stdin),
		writer: writer,
	}
}

func runAssistantCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	correlationID string,
) int {
	flags := flag.NewFlagSet("assistant", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	flowRaw := flags.String("flow", "", "flow to run: setup|task_submit|approval_decision|comm_send")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	workspace := normalizeWorkspace(*workspaceID)
	prompter := newAssistantPrompter(stdin, stderr)
	fmt.Fprintln(stderr, "Interactive assistant mode: type 'back' to edit previous input, 'cancel' to abort.")

	flow, cancelled, flowErr := resolveAssistantFlow(prompter, *flowRaw)
	if flowErr != nil {
		response := assistantResponse{
			SchemaVersion: "1.0.0",
			WorkspaceID:   workspace,
			Success:       false,
			Cancelled:     false,
			Backtracks:    prompter.backtracks,
			Summary:       "Failed to resolve interactive flow selection.",
			Error:         flowErr.Error(),
			Remediation: []string{
				"Use --flow with one of: setup|task_submit|approval_decision|comm_send.",
			},
		}
		_ = writeJSON(stdout, response)
		return 1
	}
	if cancelled {
		response := assistantResponse{
			SchemaVersion: "1.0.0",
			WorkspaceID:   workspace,
			Success:       false,
			Cancelled:     true,
			Backtracks:    prompter.backtracks,
			Summary:       "Interactive assistant cancelled before flow execution.",
		}
		_ = writeJSON(stdout, response)
		return 0
	}

	var (
		result  any
		summary string
		err     error
	)
	switch flow {
	case assistantFlowSetup:
		result, summary, cancelled, err = runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
	case assistantFlowTask:
		result, summary, cancelled, err = runAssistantTaskSubmitFlow(ctx, client, workspace, prompter, correlationID)
	case assistantFlowApproval:
		result, summary, cancelled, err = runAssistantApprovalFlow(ctx, client, workspace, prompter, correlationID)
	case assistantFlowCommSend:
		result, summary, cancelled, err = runAssistantCommSendFlow(ctx, client, workspace, prompter, correlationID)
	default:
		err = fmt.Errorf("unsupported assistant flow %q", flow)
	}

	if cancelled {
		response := assistantResponse{
			SchemaVersion: "1.0.0",
			Flow:          flow,
			WorkspaceID:   workspace,
			Success:       false,
			Cancelled:     true,
			Backtracks:    prompter.backtracks,
			Summary:       "Interactive assistant cancelled.",
		}
		_ = writeJSON(stdout, response)
		return 0
	}

	if err != nil {
		response := assistantResponse{
			SchemaVersion: "1.0.0",
			Flow:          flow,
			WorkspaceID:   workspace,
			Success:       false,
			Cancelled:     false,
			Backtracks:    prompter.backtracks,
			Summary:       firstNonEmpty(strings.TrimSpace(summary), "Interactive assistant flow failed."),
			Error:         err.Error(),
			Remediation:   assistantFlowRemediation(flow),
		}
		_ = writeJSON(stdout, response)
		return 1
	}

	response := assistantResponse{
		SchemaVersion: "1.0.0",
		Flow:          flow,
		WorkspaceID:   workspace,
		Success:       true,
		Cancelled:     false,
		Backtracks:    prompter.backtracks,
		Summary:       firstNonEmpty(strings.TrimSpace(summary), "Interactive assistant flow completed."),
		Result:        result,
	}
	if writeJSON(stdout, response) != 0 {
		return 1
	}
	return 0
}

func resolveAssistantFlow(prompter *assistantPrompter, raw string) (assistantFlow, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		flow, ok := normalizeAssistantFlow(trimmed)
		if !ok {
			return "", false, fmt.Errorf("unsupported --flow %q", raw)
		}
		return flow, false, nil
	}

	fmt.Fprintln(prompter.writer, "Select flow:")
	fmt.Fprintln(prompter.writer, "  1) setup")
	fmt.Fprintln(prompter.writer, "  2) task_submit")
	fmt.Fprintln(prompter.writer, "  3) approval_decision")
	fmt.Fprintln(prompter.writer, "  4) comm_send")
	for {
		value, action, err := prompter.prompt("Flow (name or 1-4)", "", true, nil)
		if err != nil {
			return "", false, err
		}
		if action == promptActionCancel {
			return "", true, nil
		}
		if action == promptActionBack {
			fmt.Fprintln(prompter.writer, "Already at flow selection.")
			continue
		}
		flow, ok := normalizeAssistantFlow(value)
		if ok {
			return flow, false, nil
		}
		fmt.Fprintln(prompter.writer, "Invalid flow. Choose setup|task_submit|approval_decision|comm_send or 1-4.")
	}
}

func normalizeAssistantFlow(raw string) (assistantFlow, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "1", string(assistantFlowSetup), "setup_flow":
		return assistantFlowSetup, true
	case "2", string(assistantFlowTask), "task", "submit_task":
		return assistantFlowTask, true
	case "3", string(assistantFlowApproval), "approval":
		return assistantFlowApproval, true
	case "4", string(assistantFlowCommSend), "comm", "send_comm":
		return assistantFlowCommSend, true
	default:
		return "", false
	}
}

func runAssistantSetupFlow(
	ctx context.Context,
	client *transport.Client,
	workspace string,
	prompter *assistantPrompter,
	correlationID string,
) (any, string, bool, error) {
	providerValue, action, err := prompter.prompt("Provider (openai|anthropic|google|ollama)", providerconfig.ProviderOpenAI, true, func(value string) error {
		_, err := providerconfig.NormalizeProvider(value)
		return err
	})
	if err != nil {
		return nil, "", false, err
	}
	if action == promptActionCancel {
		return nil, "", true, nil
	}
	if action == promptActionBack {
		return nil, "", false, errors.New("unexpected back action")
	}

	providerName, _ := providerconfig.NormalizeProvider(providerValue)
	endpointValue, action, err := prompter.prompt("Provider endpoint", providerconfig.DefaultEndpoint(providerName), false, nil)
	if err != nil {
		return nil, "", false, err
	}
	if action == promptActionCancel {
		return nil, "", true, nil
	}
	if action == promptActionBack {
		return runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
	}

	var (
		secretReference map[string]any
		secretName      string
	)
	if providerconfig.ProviderRequiresAPIKey(providerName) {
		apiKeyValue, apiKeyAction, apiKeyErr := prompter.prompt("Provider API key (write-only)", "", true, nil)
		if apiKeyErr != nil {
			return nil, "", false, apiKeyErr
		}
		if apiKeyAction == promptActionCancel {
			return nil, "", true, nil
		}
		if apiKeyAction == promptActionBack {
			return runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
		}

		secretDefault := quickstartDefaultAPIKeySecretName(providerName)
		secretValue, secretAction, secretErr := prompter.prompt("Secret reference name", secretDefault, true, nil)
		if secretErr != nil {
			return nil, "", false, secretErr
		}
		if secretAction == promptActionCancel {
			return nil, "", true, nil
		}
		if secretAction == promptActionBack {
			return runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
		}
		secretName = strings.TrimSpace(secretValue)

		manager, managerErr := newSecretManager()
		if managerErr != nil {
			return nil, "Failed to initialize secure-store manager.", false, managerErr
		}
		ref, putErr := manager.Put(workspace, secretName, apiKeyValue)
		if putErr != nil {
			return nil, "Failed to store provider API key securely.", false, putErr
		}
		registeredRef, upsertErr := client.UpsertSecretReference(ctx, transport.SecretReferenceUpsertRequest{
			WorkspaceID: ref.WorkspaceID,
			Name:        ref.Name,
			Backend:     ref.Backend,
			Service:     ref.Service,
			Account:     ref.Account,
		}, correlationID+".assistant.setup.secret")
		if upsertErr != nil {
			return nil, "Failed to register provider secret reference with daemon.", false, upsertErr
		}
		secretReference = map[string]any{
			"workspace_id": registeredRef.Reference.WorkspaceID,
			"name":         registeredRef.Reference.Name,
			"backend":      registeredRef.Reference.Backend,
			"service":      registeredRef.Reference.Service,
			"account":      registeredRef.Reference.Account,
		}
	}

	modelDefault := quickstartDefaultModelKey(providerName)
	modelValue, modelAction, modelErr := prompter.prompt("Model key", modelDefault, true, nil)
	if modelErr != nil {
		return nil, "", false, modelErr
	}
	if modelAction == promptActionCancel {
		return nil, "", true, nil
	}
	if modelAction == promptActionBack {
		return runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
	}

	taskClassValue, classAction, classErr := prompter.prompt("Task class", "chat", true, nil)
	if classErr != nil {
		return nil, "", false, classErr
	}
	if classAction == promptActionCancel {
		return nil, "", true, nil
	}
	if classAction == promptActionBack {
		return runAssistantSetupFlow(ctx, client, workspace, prompter, correlationID)
	}

	providerRecord, setErr := client.SetProvider(ctx, transport.ProviderSetRequest{
		WorkspaceID:      workspace,
		Provider:         providerName,
		Endpoint:         strings.TrimSpace(endpointValue),
		APIKeySecretName: secretName,
	}, correlationID+".assistant.setup.provider")
	if setErr != nil {
		return nil, "Provider configuration failed.", false, setErr
	}

	routeRecord, routeErr := client.SelectModelRoute(ctx, transport.ModelSelectRequest{
		WorkspaceID: workspace,
		TaskClass:   normalizeTaskClass(taskClassValue),
		Provider:    providerName,
		ModelKey:    strings.TrimSpace(modelValue),
	}, correlationID+".assistant.setup.route")
	if routeErr != nil {
		return nil, "Model route configuration failed.", false, routeErr
	}

	result := map[string]any{
		"provider_config": providerRecord,
		"model_route":     routeRecord,
	}
	if secretReference != nil {
		result["secret_reference"] = secretReference
	}
	return result, "Setup flow completed.", false, nil
}

func runAssistantTaskSubmitFlow(
	ctx context.Context,
	client *transport.Client,
	workspace string,
	prompter *assistantPrompter,
	correlationID string,
) (any, string, bool, error) {
	values, cancelled, err := prompter.promptFields([]assistantPromptField{
		{Key: "requested_by", Label: "Requested-by actor id", Required: true},
		{Key: "subject", Label: "Subject actor id", Required: true},
		{Key: "title", Label: "Task title", Required: true},
		{Key: "description", Label: "Task description", Required: false},
		{Key: "task_class", Label: "Task class", DefaultValue: "chat", Required: true},
	})
	if err != nil {
		return nil, "", false, err
	}
	if cancelled {
		return nil, "", true, nil
	}

	response, submitErr := client.SubmitTask(ctx, transport.SubmitTaskRequest{
		WorkspaceID:             workspace,
		RequestedByActorID:      strings.TrimSpace(values["requested_by"]),
		SubjectPrincipalActorID: strings.TrimSpace(values["subject"]),
		Title:                   strings.TrimSpace(values["title"]),
		Description:             strings.TrimSpace(values["description"]),
		TaskClass:               normalizeTaskClass(values["task_class"]),
	}, correlationID+".assistant.task.submit")
	if submitErr != nil {
		return nil, "Task submit flow failed.", false, submitErr
	}
	return response, "Task submit flow completed.", false, nil
}

func runAssistantApprovalFlow(
	ctx context.Context,
	client *transport.Client,
	workspace string,
	prompter *assistantPrompter,
	correlationID string,
) (any, string, bool, error) {
	values, cancelled, err := prompter.promptFields([]assistantPromptField{
		{Key: "approval_id", Label: "Approval ID", Required: true},
		{Key: "phrase", Label: "Approval phrase", DefaultValue: "GO AHEAD", Required: true},
		{Key: "actor_id", Label: "Decision actor ID", Required: true},
	})
	if err != nil {
		return nil, "", false, err
	}
	if cancelled {
		return nil, "", true, nil
	}

	response, decisionErr := client.AgentApprove(ctx, transport.AgentApproveRequest{
		WorkspaceID:       workspace,
		ApprovalRequestID: strings.TrimSpace(values["approval_id"]),
		Phrase:            strings.TrimSpace(values["phrase"]),
		DecisionByActorID: strings.TrimSpace(values["actor_id"]),
		CorrelationID:     correlationID + ".assistant.approval.approve",
	}, correlationID+".assistant.approval.approve")
	if decisionErr != nil {
		return nil, "Approval decision flow failed.", false, decisionErr
	}
	return response, "Approval decision flow completed.", false, nil
}

func runAssistantCommSendFlow(
	ctx context.Context,
	client *transport.Client,
	workspace string,
	prompter *assistantPrompter,
	correlationID string,
) (any, string, bool, error) {
	values, cancelled, err := prompter.promptFields([]assistantPromptField{
		{Key: "source_channel", Label: "Source channel", DefaultValue: "imessage", Required: true},
		{Key: "destination", Label: "Destination endpoint", Required: true},
		{Key: "message", Label: "Message body", Required: true},
		{Key: "operation_id", Label: "Operation ID (optional)", Required: false},
	})
	if err != nil {
		return nil, "", false, err
	}
	if cancelled {
		return nil, "", true, nil
	}

	response, sendErr := client.CommSend(ctx, transport.CommSendRequest{
		WorkspaceID:   workspace,
		OperationID:   strings.TrimSpace(values["operation_id"]),
		SourceChannel: strings.TrimSpace(values["source_channel"]),
		Destination:   strings.TrimSpace(values["destination"]),
		Message:       strings.TrimSpace(values["message"]),
	}, correlationID+".assistant.comm.send")
	if sendErr != nil {
		return nil, "Communication send flow failed.", false, sendErr
	}
	if !response.Success {
		return response, "Communication send flow failed.", false, fmt.Errorf("delivery was not successful: %s", strings.TrimSpace(response.Error))
	}
	return response, "Communication send flow completed.", false, nil
}

type promptAction string

const (
	promptActionSubmit promptAction = "submit"
	promptActionBack   promptAction = "back"
	promptActionCancel promptAction = "cancel"
)

func (p *assistantPrompter) promptFields(fields []assistantPromptField) (map[string]string, bool, error) {
	values := map[string]string{}
	if len(fields) == 0 {
		return values, false, nil
	}
	index := 0
	for index >= 0 && index < len(fields) {
		field := fields[index]
		defaultValue := field.DefaultValue
		if previous, ok := values[field.Key]; ok {
			defaultValue = previous
		}
		value, action, err := p.prompt(field.Label, defaultValue, field.Required, field.Validate)
		if err != nil {
			return nil, false, err
		}
		switch action {
		case promptActionCancel:
			return nil, true, nil
		case promptActionBack:
			if index == 0 {
				fmt.Fprintln(p.writer, "Already at the first prompt.")
				continue
			}
			index--
			p.backtracks++
			continue
		default:
			values[field.Key] = value
			index++
		}
	}
	return values, false, nil
}

func (p *assistantPrompter) prompt(
	label string,
	defaultValue string,
	required bool,
	validate func(value string) error,
) (string, promptAction, error) {
	for {
		if strings.TrimSpace(defaultValue) != "" {
			fmt.Fprintf(p.writer, "%s [%s]: ", strings.TrimSpace(label), strings.TrimSpace(defaultValue))
		} else {
			fmt.Fprintf(p.writer, "%s: ", strings.TrimSpace(label))
		}

		raw, err := p.reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", promptActionSubmit, err
		}

		trimmed := strings.TrimSpace(raw)
		if errors.Is(err, io.EOF) && trimmed == "" {
			return "", promptActionCancel, nil
		}
		switch strings.ToLower(trimmed) {
		case "cancel", "exit", "quit":
			return "", promptActionCancel, nil
		case "back":
			return "", promptActionBack, nil
		}
		if trimmed == "" && strings.TrimSpace(defaultValue) != "" {
			trimmed = strings.TrimSpace(defaultValue)
		}
		if required && strings.TrimSpace(trimmed) == "" {
			fmt.Fprintln(p.writer, "Value is required.")
			continue
		}
		if validate != nil {
			if err := validate(trimmed); err != nil {
				fmt.Fprintf(p.writer, "Invalid value: %v\n", err)
				continue
			}
		}
		return trimmed, promptActionSubmit, nil
	}
}

func assistantFlowRemediation(flow assistantFlow) []string {
	switch flow {
	case assistantFlowSetup:
		return []string{
			"Run `personal-agent provider list --workspace <id>` to inspect provider configuration.",
			"Run `personal-agent model resolve --workspace <id> --task-class <class>` to verify routing.",
		}
	case assistantFlowTask:
		return []string{
			"Run `personal-agent task submit ...` directly to retry with explicit flags.",
			"Run `personal-agent task status --task-id <id>` after submit succeeds.",
		}
	case assistantFlowApproval:
		return []string{
			"Run `personal-agent agent approve --workspace <id> --approval-id <id> --phrase \"GO AHEAD\" --actor-id <id>` directly.",
		}
	case assistantFlowCommSend:
		return []string{
			"Run `personal-agent comm send --workspace <id> --destination <endpoint> --message <text>` directly.",
		}
	default:
		return nil
	}
}
