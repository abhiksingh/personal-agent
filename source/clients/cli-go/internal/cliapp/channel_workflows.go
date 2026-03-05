package cliapp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	repodelivery "personalagent/runtime/internal/core/repository/delivery"
	deliveryservice "personalagent/runtime/internal/core/service/delivery"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/securestore"
)

type twilioSMSChatTurn struct {
	OperationID          string `json:"operation_id"`
	Message              string `json:"message"`
	Success              bool   `json:"success"`
	Delivered            bool   `json:"delivered"`
	Channel              string `json:"channel,omitempty"`
	ProviderReceipt      string `json:"provider_receipt,omitempty"`
	IdempotentReplay     bool   `json:"idempotent_replay,omitempty"`
	ThreadID             string `json:"thread_id,omitempty"`
	AssistantReply       string `json:"assistant_reply,omitempty"`
	AssistantOperationID string `json:"assistant_operation_id,omitempty"`
	AssistantError       string `json:"assistant_error,omitempty"`
	Error                string `json:"error,omitempty"`
}

func executeTwilioSMSDelivery(
	ctx context.Context,
	db *sql.DB,
	manager *securestore.Manager,
	workspace string,
	destination string,
	message string,
	operationID string,
) (types.DeliveryResult, string, error) {
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newCLIDeliverySender(db, manager, newCLIHTTPClientFromContext(ctx), 0, 0)
	service := deliveryservice.NewService(store, sender, deliveryservice.Options{})

	result, err := service.Deliver(ctx, types.DeliveryRequest{
		WorkspaceID:         workspace,
		OperationID:         strings.TrimSpace(operationID),
		SourceChannel:       "sms",
		DestinationEndpoint: strings.TrimSpace(destination),
		MessageBody:         strings.TrimSpace(message),
	})
	if err != nil {
		return result, "", err
	}
	threadID := ""
	if strings.TrimSpace(result.ProviderReceipt) != "" {
		threadID, _ = lookupThreadByProviderMessage(ctx, db, workspace, result.ProviderReceipt)
	}
	return result, threadID, nil
}

func lookupThreadByProviderMessage(ctx context.Context, db *sql.DB, workspace string, providerMessageID string) (string, error) {
	var threadID string
	err := db.QueryRowContext(ctx, `
		SELECT thread_id
		FROM comm_messages
		WHERE workspace_id = ?
		  AND provider_message_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, workspace, strings.TrimSpace(providerMessageID)).Scan(&threadID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup thread by provider message: %w", err)
	}
	return strings.TrimSpace(threadID), nil
}
