package daemonruntime

import (
	"context"
	"errors"
	"time"

	"personalagent/runtime/internal/transport"
)

const (
	defaultControlCancelSettlementTimeout      = 5 * time.Second
	defaultControlCancelSettlementPollInterval = 50 * time.Millisecond
)

type PersistedControlBackend struct {
	container    *ServiceContainer
	agent        transport.AgentService
	eventBroker  *transport.EventBroker
	runCanceller queuedTaskRunCanceller

	version    string
	channels   []string
	connectors []string

	cancelSettlementTimeout      time.Duration
	cancelSettlementPollInterval time.Duration
}

var _ transport.ControlBackend = (*PersistedControlBackend)(nil)

type queuedTaskRunCanceller interface {
	CancelQueuedTaskRun(runID string, reason string) bool
}

func NewPersistedControlBackend(container *ServiceContainer, agent transport.AgentService, eventBroker *transport.EventBroker) (*PersistedControlBackend, error) {
	if container == nil || container.DB == nil {
		return nil, errors.New("service container with db is required")
	}
	if agent == nil {
		return nil, errors.New("agent service is required")
	}
	return &PersistedControlBackend{
		container:                    container,
		agent:                        agent,
		eventBroker:                  eventBroker,
		version:                      "0.1.0",
		channels:                     transport.DefaultCapabilitySmokeChannels(),
		connectors:                   transport.DefaultCapabilitySmokeConnectors(),
		cancelSettlementTimeout:      defaultControlCancelSettlementTimeout,
		cancelSettlementPollInterval: defaultControlCancelSettlementPollInterval,
	}, nil
}

func (b *PersistedControlBackend) SetQueuedTaskRunCanceller(canceller queuedTaskRunCanceller) {
	if b == nil {
		return
	}
	b.runCanceller = canceller
}

func (b *PersistedControlBackend) CapabilitySmoke(_ context.Context, correlationID string) (transport.CapabilitySmokeResponse, error) {
	return transport.CapabilitySmokeResponse{
		DaemonVersion: b.version,
		Channels:      append([]string{}, b.channels...),
		Connectors:    append([]string{}, b.connectors...),
		Healthy:       true,
		CorrelationID: correlationID,
	}, nil
}
