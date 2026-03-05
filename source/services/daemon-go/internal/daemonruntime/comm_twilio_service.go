package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"personalagent/runtime/internal/channelconfig"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

type CommTwilioService struct {
	container       *ServiceContainer
	twilioStore     *channelconfig.SQLiteTwilioStore
	channelDispatch ChannelWorkerDispatcher
	automationEval  AutomationCommEventEvaluator
	assistantChat   transport.ChatService
}

type twilioWorkspaceCredentials struct {
	Config     channelconfig.TwilioConfig
	AccountSID string
	AuthToken  string
}

type AutomationCommEventEvaluator interface {
	EvaluateCommEvent(ctx context.Context, eventID string) (types.CommTriggerEvaluationResult, error)
}

type daemonDeliverySender struct {
	mu       sync.Mutex
	failures map[string]int
	sent     map[string]int

	db        *sql.DB
	resolver  SecretReferenceResolver
	dispatch  ChannelWorkerDispatcher
	twilioCfg *channelconfig.SQLiteTwilioStore
}

var _ transport.CommService = (*CommTwilioService)(nil)
var _ transport.TwilioChannelService = (*CommTwilioService)(nil)

const (
	commAttemptHistoryDefaultLimit = 50
	commAttemptHistoryMaxLimit     = 200
)

func NewCommTwilioService(container *ServiceContainer) (*CommTwilioService, error) {
	if container == nil || container.DB == nil {
		return nil, fmt.Errorf("service container with db is required")
	}
	return &CommTwilioService{
		container:       container,
		twilioStore:     channelconfig.NewSQLiteTwilioStore(container.DB),
		channelDispatch: NewSupervisorChannelWorkerDispatcher(container.PluginSupervisor),
	}, nil
}

func (s *CommTwilioService) SetAutomationCommEventEvaluator(evaluator AutomationCommEventEvaluator) {
	s.automationEval = evaluator
}

func (s *CommTwilioService) SetAssistantChatService(chatService transport.ChatService) {
	s.assistantChat = chatService
}
