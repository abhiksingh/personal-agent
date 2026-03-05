package mail

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

func (a *Adapter) executeUncached(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep, stepResultPath string) (connectorcontract.StepExecutionResult, error) {
	workspaceRoot := adapterscaffold.WorkspaceRootFromStepResultPath(stepResultPath)
	stepToken := adapterhelpers.StableStepToken(execCtx, step)
	nowText := time.Now().UTC().Format(time.RFC3339Nano)
	provider := "apple-mail"
	if isMailAutomationDryRunEnabled() {
		provider = "apple-mail-dry-run"
	}

	switch step.CapabilityKey {
	case CapabilityDraft:
		recipient, subject, body, inputErr := resolveMailStepInput(step, false)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail draft input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		draftID := "draft-" + stepToken
		operation, err := executeMailOperation(ctx, "draft", recipient, subject, body)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail draft automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute mail draft: %w", err)
		}
		record := buildMailOperationRecord(
			CapabilityDraft,
			draftID,
			stepToken,
			execCtx,
			nowText,
			operation.Transport,
			recipient,
			subject,
			body,
		)
		recordPath, err := writeMailOperationRecord(workspaceRoot, CapabilityDraft, draftID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail draft write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write mail draft record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "mail draft prepared via Apple Mail automation",
			Retryable: false,
			Evidence: map[string]string{
				"draft_id":              draftID,
				"workspace_id":          execCtx.WorkspaceID,
				"requested_by_actor_id": execCtx.RequestedByActor,
				"provider":              provider,
				"transport":             operation.Transport,
				"recipient":             recipient,
				"record_path":           recordPath,
			},
			Output: map[string]any{
				"draft_id":     draftID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilitySend:
		recipient, subject, body, inputErr := resolveMailStepInput(step, true)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail send input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		messageID := "message-" + stepToken
		operation, err := executeMailOperation(ctx, "send", recipient, subject, body)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail send automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute mail send: %w", err)
		}
		record := buildMailOperationRecord(
			CapabilitySend,
			messageID,
			stepToken,
			execCtx,
			nowText,
			operation.Transport,
			recipient,
			subject,
			body,
		)
		recordPath, err := writeMailOperationRecord(workspaceRoot, CapabilitySend, messageID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail send write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write mail send record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "mail sent via Apple Mail automation",
			Retryable: false,
			Evidence: map[string]string{
				"message_id":  messageID,
				"provider":    provider,
				"transport":   operation.Transport,
				"recipient":   recipient,
				"record_path": recordPath,
			},
			Output: map[string]any{
				"message_id":   messageID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilityReply:
		recipient, subject, body, inputErr := resolveMailStepInput(step, true)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail reply input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		replyID := "reply-" + stepToken
		threadID := "thread-" + adapterhelpers.StableToken(execCtx.RunID, "run")
		replySubject := subject
		if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
			replySubject = "Re: " + replySubject
		}
		operation, err := executeMailOperation(ctx, "reply", recipient, replySubject, body)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail reply automation failed",
				Retryable:   true,
				ErrorReason: "automation_unavailable",
			}, fmt.Errorf("execute mail reply: %w", err)
		}
		record := buildMailOperationRecord(
			CapabilityReply,
			replyID,
			stepToken,
			execCtx,
			nowText,
			operation.Transport,
			recipient,
			replySubject,
			body,
		)
		recordPath, err := writeMailOperationRecord(workspaceRoot, CapabilityReply, replyID, record)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail reply write failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, fmt.Errorf("write mail reply record: %w", err)
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "mail reply sent via Apple Mail automation",
			Retryable: false,
			Evidence: map[string]string{
				"reply_id":    replyID,
				"thread_id":   threadID,
				"provider":    provider,
				"transport":   operation.Transport,
				"recipient":   recipient,
				"record_path": recordPath,
			},
			Output: map[string]any{
				"reply_id":     replyID,
				"thread_id":    threadID,
				"operation_id": operation.OperationID,
			},
		}, nil
	case CapabilityUnreadSummary:
		limit, inputErr := resolveMailSummaryLimit(step.Input)
		if inputErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail unread-summary input is invalid",
				Retryable:   false,
				ErrorReason: "invalid_input",
			}, inputErr
		}
		summary, err := a.summarizeUnreadInbox(ctx, strings.TrimSpace(execCtx.WorkspaceID), limit)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "mail unread-summary query failed",
				Retryable:   true,
				ErrorReason: "storage_error",
			}, err
		}
		summaryText := fmt.Sprintf("summarized %d unread emails across %d threads", summary.UnreadCount, summary.ThreadCount)
		if summary.UnreadCount == 0 {
			summaryText = "no unread inbox emails found"
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   summaryText,
			Retryable: false,
			Evidence: map[string]string{
				"provider":       "mail_inbox_sqlite",
				"workspace_id":   strings.TrimSpace(execCtx.WorkspaceID),
				"unread_count":   strconv.Itoa(summary.UnreadCount),
				"thread_count":   strconv.Itoa(summary.ThreadCount),
				"returned_count": strconv.Itoa(len(summary.Items)),
				"limit":          strconv.Itoa(summary.Limit),
			},
			Output: map[string]any{
				"unread_count":      summary.UnreadCount,
				"thread_count":      summary.ThreadCount,
				"returned_count":    len(summary.Items),
				"limit":             summary.Limit,
				"items":             unreadSummaryItemsOutput(summary.Items),
				"unread_semantics":  "inbound mail without newer assistant outbound mail in same thread",
				"queried_workspace": strings.TrimSpace(execCtx.WorkspaceID),
			},
		}, nil
	default:
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "unsupported mail capability",
			Retryable:   false,
			ErrorReason: "unsupported_capability",
		}, fmt.Errorf("unsupported mail capability: %s", step.CapabilityKey)
	}
}
