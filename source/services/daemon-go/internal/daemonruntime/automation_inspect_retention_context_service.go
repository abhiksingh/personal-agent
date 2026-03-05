package daemonruntime

import (
	"fmt"

	"personalagent/runtime/internal/transport"
)

type AutomationInspectRetentionContextService struct {
	container *ServiceContainer
}

var _ transport.AutomationService = (*AutomationInspectRetentionContextService)(nil)
var _ transport.InspectService = (*AutomationInspectRetentionContextService)(nil)
var _ transport.RetentionService = (*AutomationInspectRetentionContextService)(nil)
var _ transport.ContextOpsService = (*AutomationInspectRetentionContextService)(nil)

const (
	defaultContextQueryListLimit = 25
	maxContextQueryListLimit     = 200
)

func NewAutomationInspectRetentionContextService(container *ServiceContainer) (*AutomationInspectRetentionContextService, error) {
	if container == nil || container.DB == nil {
		return nil, fmt.Errorf("service container with db is required")
	}
	return &AutomationInspectRetentionContextService{container: container}, nil
}
