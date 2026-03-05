package cliapp

import (
	"context"

	"personalagent/runtime/internal/transport"
)

type doctorCheckStatus string

const (
	doctorCheckStatusPass    doctorCheckStatus = "pass"
	doctorCheckStatusWarn    doctorCheckStatus = "warn"
	doctorCheckStatusFail    doctorCheckStatus = "fail"
	doctorCheckStatusSkipped doctorCheckStatus = "skipped"
)

type doctorCheck struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Status      doctorCheckStatus `json:"status"`
	Summary     string            `json:"summary"`
	Details     map[string]any    `json:"details,omitempty"`
	Remediation []string          `json:"remediation,omitempty"`
}

type doctorSummary struct {
	Pass    int `json:"pass"`
	Warn    int `json:"warn"`
	Fail    int `json:"fail"`
	Skipped int `json:"skipped"`
}

type doctorReport struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	WorkspaceID   string            `json:"workspace_id,omitempty"`
	OverallStatus doctorCheckStatus `json:"overall_status"`
	Summary       doctorSummary     `json:"summary"`
	Checks        []doctorCheck     `json:"checks"`
}

type doctorCommandOptions struct {
	RequestedWorkspace string
	Quick              bool
	IncludeOptional    bool
	Strict             bool
}

type doctorExecutionState struct {
	Workspace       string
	CorrelationID   string
	IncludeOptional bool

	connectorStatusLoaded bool
	connectorStatus       *transport.ConnectorStatusResponse
	connectorStatusErr    error

	channelStatusLoaded bool
	channelStatus       *transport.ChannelStatusResponse
	channelStatusErr    error
}

func (s *doctorExecutionState) loadConnectorStatus(ctx context.Context, client *transport.Client) (*transport.ConnectorStatusResponse, error) {
	if s == nil {
		return nil, nil
	}
	if s.connectorStatusLoaded {
		return s.connectorStatus, s.connectorStatusErr
	}
	response, err := client.ConnectorStatus(ctx, transport.ConnectorStatusRequest{
		WorkspaceID: s.Workspace,
	}, s.CorrelationID)
	s.connectorStatusLoaded = true
	s.connectorStatusErr = err
	if err == nil {
		s.connectorStatus = &response
	}
	return s.connectorStatus, s.connectorStatusErr
}

func (s *doctorExecutionState) loadChannelStatus(ctx context.Context, client *transport.Client) (*transport.ChannelStatusResponse, error) {
	if s == nil {
		return nil, nil
	}
	if s.channelStatusLoaded {
		return s.channelStatus, s.channelStatusErr
	}
	response, err := client.ChannelStatus(ctx, transport.ChannelStatusRequest{
		WorkspaceID: s.Workspace,
	}, s.CorrelationID)
	s.channelStatusLoaded = true
	s.channelStatusErr = err
	if err == nil {
		s.channelStatus = &response
	}
	return s.channelStatus, s.channelStatusErr
}
