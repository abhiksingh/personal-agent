package daemonruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	repodelivery "personalagent/runtime/internal/core/repository/delivery"
	deliveryservice "personalagent/runtime/internal/core/service/delivery"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) SendComm(ctx context.Context, request transport.CommSendRequest) (transport.CommSendResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	threadID := strings.TrimSpace(request.ThreadID)
	destination := strings.TrimSpace(request.Destination)
	sourceChannel := strings.TrimSpace(request.SourceChannel)
	connectorHint := normalizeCommConnectorHint(request.ConnectorID)

	if threadID != "" {
		replyContext, err := resolveCommThreadReplyContext(ctx, s.container.DB, workspace, threadID)
		if err != nil {
			return transport.CommSendResponse{}, err
		}
		if connectorHint != "" && replyContext.ConnectorID != "" && connectorHint != replyContext.ConnectorID {
			return transport.CommSendResponse{}, fmt.Errorf(
				"connector hint %q does not match thread %q connector %q",
				connectorHint,
				threadID,
				replyContext.ConnectorID,
			)
		}
		if connectorHint == "" {
			connectorHint = replyContext.ConnectorID
		}
		if sourceChannel == "" {
			sourceChannel = replyContext.SourceChannel
		}
		if destination == "" {
			destination = replyContext.Destination
		}
	}

	sourceChannel = applyCommConnectorHintToSource(sourceChannel, connectorHint)
	if sourceChannel == "" {
		sourceChannel = "message"
	}
	if !isSupportedCommSourceChannel(sourceChannel) {
		return transport.CommSendResponse{}, fmt.Errorf(
			"unsupported source channel %q (allowed: app|message|voice|sms with optional connector hint builtin.app|imessage|twilio)",
			sourceChannel,
		)
	}
	if destination == "" {
		return transport.CommSendResponse{}, fmt.Errorf("--destination is required (or provide --thread-id with resolvable reply destination)")
	}

	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		resolved, err := daemonRandomID()
		if err != nil {
			return transport.CommSendResponse{}, err
		}
		operationID = resolved
	}

	store := repodelivery.NewSQLiteDeliveryStore(s.container.DB)
	sender := newDaemonDeliverySender(
		s.container.DB,
		s.container.SecretResolver,
		s.channelDispatch,
		s.twilioStore,
		request.IMessagesFailure,
		request.SMSFailures,
	)
	service := deliveryservice.NewService(store, sender, deliveryservice.Options{})
	deliverySourceChannel := canonicalDeliveryPolicySourceChannel(sourceChannel)

	result, deliverErr := service.Deliver(ctx, types.DeliveryRequest{
		WorkspaceID:         workspace,
		OperationID:         operationID,
		StepID:              strings.TrimSpace(request.StepID),
		EventID:             strings.TrimSpace(request.EventID),
		SourceChannel:       deliverySourceChannel,
		DestinationEndpoint: destination,
		MessageBody:         strings.TrimSpace(request.Message),
	})

	resolvedConnector := connectorHint
	if resolvedConnector == "" {
		resolvedConnector = connectorHintFromSourceChannel(sourceChannel)
	}
	response := transport.CommSendResponse{
		WorkspaceID:           workspace,
		OperationID:           operationID,
		ThreadID:              threadID,
		ResolvedSourceChannel: sourceChannel,
		ResolvedConnectorID:   resolvedConnector,
		ResolvedDestination:   destination,
		Success:               deliverErr == nil,
		Result:                result,
	}
	if deliverErr != nil {
		response.Error = deliverErr.Error()
	}
	return response, nil
}

type commThreadReplyContext struct {
	ThreadID      string
	SourceChannel string
	ConnectorID   string
	Destination   string
}

func resolveCommThreadReplyContext(
	ctx context.Context,
	db *sql.DB,
	workspace string,
	threadID string,
) (commThreadReplyContext, error) {
	if db == nil {
		return commThreadReplyContext{}, fmt.Errorf("database is not configured")
	}
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return commThreadReplyContext{}, fmt.Errorf("thread_id is required")
	}

	var (
		threadChannel   string
		threadConnector string
		threadExternal  string
	)
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(channel, ''), COALESCE(connector_id, ''), COALESCE(external_ref, '')
		FROM comm_threads
		WHERE workspace_id = ?
		  AND id = ?
		LIMIT 1
	`, workspace, trimmedThreadID).Scan(&threadChannel, &threadConnector, &threadExternal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return commThreadReplyContext{}, fmt.Errorf("comm thread %q not found for workspace %q", trimmedThreadID, workspace)
		}
		return commThreadReplyContext{}, fmt.Errorf("load comm thread reply context: %w", err)
	}

	threadConnector = normalizeCommConnectorHint(threadConnector)
	sourceChannel := applyCommConnectorHintToSource(threadChannel, threadConnector)
	destination, err := resolveThreadReplyDestination(ctx, db, workspace, trimmedThreadID, strings.TrimSpace(threadExternal))
	if err != nil {
		return commThreadReplyContext{}, err
	}

	return commThreadReplyContext{
		ThreadID:      trimmedThreadID,
		SourceChannel: sourceChannel,
		ConnectorID:   threadConnector,
		Destination:   destination,
	}, nil
}

func resolveThreadReplyDestination(
	ctx context.Context,
	db *sql.DB,
	workspace string,
	threadID string,
	threadExternalRef string,
) (string, error) {
	var (
		lastDirection string
		lastFrom      string
		lastTo        string
	)
	err := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(ce.direction, ''),
			COALESCE((
				SELECT cea.address_value
				FROM comm_event_addresses cea
				WHERE cea.event_id = ce.id
				  AND cea.address_role = 'FROM'
				ORDER BY cea.position ASC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT cea.address_value
				FROM comm_event_addresses cea
				WHERE cea.event_id = ce.id
				  AND cea.address_role = 'TO'
				ORDER BY cea.position ASC
				LIMIT 1
			), '')
		FROM comm_events ce
		WHERE ce.workspace_id = ?
		  AND ce.thread_id = ?
		ORDER BY ce.occurred_at DESC, ce.created_at DESC, ce.id DESC
		LIMIT 1
	`, workspace, threadID).Scan(&lastDirection, &lastFrom, &lastTo)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("load latest thread event addresses: %w", err)
	}
	if destination := selectReplyDestination(lastDirection, lastFrom, lastTo); destination != "" {
		return destination, nil
	}

	var (
		callDirection string
		callFrom      string
		callTo        string
	)
	callErr := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(direction, ''),
			COALESCE(from_address, ''),
			COALESCE(to_address, '')
		FROM comm_call_sessions
		WHERE workspace_id = ?
		  AND thread_id = ?
		ORDER BY started_at DESC, updated_at DESC, id DESC
		LIMIT 1
	`, workspace, threadID).Scan(&callDirection, &callFrom, &callTo)
	if callErr != nil && !errors.Is(callErr, sql.ErrNoRows) {
		return "", fmt.Errorf("load latest call-session addresses: %w", callErr)
	}
	if destination := selectReplyDestination(callDirection, callFrom, callTo); destination != "" {
		return destination, nil
	}

	if destination := parseRemoteAddressFromThreadExternalRef(threadExternalRef); destination != "" {
		return destination, nil
	}

	return "", fmt.Errorf(
		"cannot derive reply destination for thread %q; provide explicit destination",
		strings.TrimSpace(threadID),
	)
}

func selectReplyDestination(direction string, fromAddress string, toAddress string) string {
	normalizedDirection := strings.ToUpper(strings.TrimSpace(direction))
	trimmedFrom := strings.TrimSpace(fromAddress)
	trimmedTo := strings.TrimSpace(toAddress)
	switch normalizedDirection {
	case "INBOUND":
		return firstNonEmpty(trimmedFrom, trimmedTo)
	case "OUTBOUND":
		return firstNonEmpty(trimmedTo, trimmedFrom)
	default:
		return firstNonEmpty(trimmedFrom, trimmedTo)
	}
}

func parseRemoteAddressFromThreadExternalRef(externalRef string) string {
	trimmed := strings.TrimSpace(externalRef)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 4 {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(parts[0]), "twilio") {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func normalizeCommConnectorHint(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "imessage":
		return "imessage"
	case "twilio":
		return "twilio"
	case "builtin.app":
		return "builtin.app"
	default:
		return normalized
	}
}

func applyCommConnectorHintToSource(sourceChannel string, connectorHint string) string {
	normalizedConnector := normalizeCommConnectorHint(connectorHint)
	if normalizedConnector == "" {
		return canonicalLogicalCommChannel(sourceChannel)
	}
	logical := canonicalLogicalCommChannel(sourceChannel)

	switch normalizedConnector {
	case "twilio":
		if logical == "app" {
			return "app"
		}
		if logical == "voice" {
			return "voice"
		}
		return "twilio"
	case "imessage":
		if logical == "voice" {
			return "voice"
		}
		if logical == "app" {
			return "app"
		}
		return "imessage"
	case "builtin.app":
		return "app"
	default:
		return canonicalLogicalCommChannel(sourceChannel)
	}
}

func canonicalLogicalCommChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "", "message":
		return "message"
	case "imessage", "i_message", "i-message", "apple_messages", "messages":
		return "imessage"
	case "twilio":
		return "twilio"
	case "sms", "text", "text_message", "text-message":
		return "sms"
	case "voice":
		return "voice"
	case "app", "builtin.app":
		return "app"
	default:
		return strings.ToLower(strings.TrimSpace(channel))
	}
}

func canonicalDeliveryPolicySourceChannel(sourceChannel string) string {
	switch strings.ToLower(strings.TrimSpace(sourceChannel)) {
	case "twilio":
		return "sms"
	case "builtin.app":
		return "app"
	default:
		return strings.ToLower(strings.TrimSpace(sourceChannel))
	}
}

func connectorHintFromSourceChannel(sourceChannel string) string {
	switch strings.ToLower(strings.TrimSpace(sourceChannel)) {
	case "twilio", "voice", "sms":
		return "twilio"
	case "imessage":
		return "imessage"
	case "app", "builtin.app":
		return "builtin.app"
	default:
		return ""
	}
}

func isSupportedCommSourceChannel(sourceChannel string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceChannel)) {
	case "app", "message", "voice", "builtin.app", "imessage", "twilio", "sms":
		return true
	default:
		return false
	}
}
