package mail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

type mailOperationResult struct {
	OperationID string
	Transport   string
}

var commandRunner adapterscaffold.CommandRunner = adapterscaffold.DefaultCommandRunner

func executeMailOperation(ctx context.Context, mode string, recipient string, subject string, body string) (mailOperationResult, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		return mailOperationResult{}, fmt.Errorf("mail operation mode is required")
	}
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return mailOperationResult{}, fmt.Errorf("mail recipient is required")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "Personal Agent Update"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "Sent by Personal Agent."
	}

	operationID := newMailOperationID(normalizedMode)
	if isMailAutomationDryRunEnabled() {
		return mailOperationResult{
			OperationID: operationID,
			Transport:   transportMailDryRun,
		}, nil
	}
	if runtime.GOOS != "darwin" {
		return mailOperationResult{}, fmt.Errorf("mail automation requires macOS (set %s=1 for dry-run)", envMailAutomationDryRun)
	}

	scriptLines := []string{
		"on run argv",
		"set operationMode to item 1 of argv",
		"set targetAddress to item 2 of argv",
		"set subjectText to item 3 of argv",
		"set bodyText to item 4 of argv",
		"tell application \"Mail\"",
		"set outgoingMessage to make new outgoing message with properties {subject:subjectText, content:bodyText & return & return, visible:false}",
		"tell outgoingMessage",
		"make new to recipient at end of to recipients with properties {address:targetAddress}",
		"if operationMode is \"draft\" then",
		"save",
		"else",
		"send",
		"end if",
		"end tell",
		"end tell",
		"end run",
	}
	args := make([]string, 0, len(scriptLines)*2+4)
	for _, line := range scriptLines {
		args = append(args, "-e", line)
	}
	scriptMode := "send"
	if normalizedMode == "draft" {
		scriptMode = "draft"
	}
	args = append(args, scriptMode, recipient, subject, body)

	result := adapterscaffold.ExecuteCommand(ctx, commandRunner, "osascript", args...)
	if result.Err != nil {
		if result.Output != "" {
			return mailOperationResult{}, fmt.Errorf("mail automation failed: %s", result.Output)
		}
		return mailOperationResult{}, fmt.Errorf("mail automation failed: %w", result.Err)
	}

	return mailOperationResult{
		OperationID: operationID,
		Transport:   transportMailAppleEvents,
	}, nil
}

func resolveMailStepInput(step connectorcontract.TaskStep, requireRecipient bool) (string, string, string, error) {
	if len(step.Input) == 0 {
		return "", "", "", fmt.Errorf("mail step input is required")
	}
	recipient, err := adapterhelpers.OptionalStringInput(step.Input, "recipient")
	if err != nil {
		return "", "", "", err
	}
	subject, err := adapterhelpers.OptionalStringInput(step.Input, "subject")
	if err != nil {
		return "", "", "", err
	}
	body, err := adapterhelpers.RequiredStringInput(step.Input, "body")
	if err != nil {
		return "", "", "", err
	}
	if requireRecipient && recipient == "" {
		return "", "", "", fmt.Errorf("mail recipient is required")
	}
	return recipient, subject, body, nil
}

func newMailOperationID(prefix string) string {
	tokenBytes := make([]byte, 4)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(tokenBytes))
}

func isMailAutomationDryRunEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envMailAutomationDryRun))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
