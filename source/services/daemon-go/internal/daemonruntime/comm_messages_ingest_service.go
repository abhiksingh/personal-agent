package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	"personalagent/runtime/internal/transport"
)

type messagesIngestPersistInput struct {
	WorkspaceID      string
	Source           string
	SourceScope      string
	SourceEventID    string
	SourceCursor     string
	ExternalThreadID string
	SenderAddress    string
	LocalAddress     string
	BodyText         string
	OccurredAt       string
}

type messagesIngestPersistResult struct {
	EventID  string
	ThreadID string
	Replayed bool
}

const messagesConnectorID = "imessage"

func (s *CommTwilioService) IngestMessages(ctx context.Context, request transport.MessagesIngestRequest) (transport.MessagesIngestResponse, error) {
	if s == nil || s.container == nil || s.container.DB == nil {
		return transport.MessagesIngestResponse{}, fmt.Errorf("comm service container db is required")
	}
	if s.channelDispatch == nil {
		return transport.MessagesIngestResponse{}, fmt.Errorf("messages worker dispatcher is not configured")
	}

	workspace := normalizeWorkspaceID(request.WorkspaceID)
	sourcePath := messagesadapter.ResolveSourceDBPath(request.SourceDBPath)
	sourceScope := messagesadapter.ResolveSourceScope(request.SourceScope, sourcePath)
	source := messagesadapter.SourceName

	cursorStart, err := loadCommIngestCursor(ctx, s.container.DB, workspace, source, sourceScope)
	if err != nil {
		return transport.MessagesIngestResponse{}, err
	}

	polled, err := s.channelDispatch.PollMessagesInbound(ctx, messagesadapter.InboundPollRequest{
		WorkspaceID:  workspace,
		SourceDBPath: sourcePath,
		SourceScope:  sourceScope,
		SinceCursor:  cursorStart,
		Limit:        request.Limit,
	})
	if err != nil {
		_ = upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, source, sourceScope, "", "", err.Error())
		return transport.MessagesIngestResponse{}, err
	}

	resolvedScope := strings.TrimSpace(polled.SourceScope)
	if resolvedScope == "" {
		resolvedScope = sourceScope
	}
	response := transport.MessagesIngestResponse{
		WorkspaceID:  workspace,
		Source:       firstNonEmpty(strings.TrimSpace(polled.Source), source),
		SourceScope:  resolvedScope,
		SourceDBPath: firstNonEmpty(strings.TrimSpace(polled.SourceDBPath), sourcePath),
		CursorStart:  cursorStart,
		CursorEnd:    firstNonEmpty(strings.TrimSpace(polled.CursorEnd), cursorStart),
		Polled:       polled.Polled,
		Events:       make([]transport.MessagesIngestEventRecord, 0, len(polled.Events)),
	}

	lastEventID := ""
	for _, event := range polled.Events {
		persisted, persistErr := persistMessagesInboundEvent(ctx, s.container.DB, messagesIngestPersistInput{
			WorkspaceID:      workspace,
			Source:           response.Source,
			SourceScope:      resolvedScope,
			SourceEventID:    strings.TrimSpace(event.SourceEventID),
			SourceCursor:     strings.TrimSpace(event.SourceCursor),
			ExternalThreadID: strings.TrimSpace(event.ExternalThreadID),
			SenderAddress:    strings.TrimSpace(event.SenderAddress),
			LocalAddress:     strings.TrimSpace(event.LocalAddress),
			BodyText:         strings.TrimSpace(event.BodyText),
			OccurredAt:       strings.TrimSpace(event.OccurredAt),
		})
		if persistErr != nil {
			_ = upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, response.Source, resolvedScope, response.CursorEnd, lastEventID, persistErr.Error())
			return transport.MessagesIngestResponse{}, persistErr
		}

		record := transport.MessagesIngestEventRecord{
			SourceEventID: strings.TrimSpace(event.SourceEventID),
			SourceCursor:  strings.TrimSpace(event.SourceCursor),
			EventID:       persisted.EventID,
			ThreadID:      persisted.ThreadID,
			Replayed:      persisted.Replayed,
		}
		response.Events = append(response.Events, record)
		lastEventID = persisted.EventID
		if persisted.Replayed {
			response.Replayed++
		} else {
			response.Accepted++
			s.evaluateAutomationForCommEvents(ctx, true, false, persisted.EventID)
		}
		if compareCursorValue(record.SourceCursor, response.CursorEnd) > 0 {
			response.CursorEnd = record.SourceCursor
		}
	}

	if strings.TrimSpace(response.CursorEnd) != "" {
		if err := upsertCommIngestCursor(ctx, s.container.DB, workspace, response.Source, resolvedScope, response.CursorEnd); err != nil {
			return transport.MessagesIngestResponse{}, err
		}
	}
	if err := upsertAutomationSourceSubscription(ctx, s.container.DB, workspace, response.Source, resolvedScope, response.CursorEnd, lastEventID, ""); err != nil {
		return transport.MessagesIngestResponse{}, err
	}
	return response, nil
}
