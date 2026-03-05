package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	pluginEventWorkerStarted      = "PLUGIN_WORKER_STARTED"
	pluginEventHandshakeAccepted  = "PLUGIN_HANDSHAKE_ACCEPTED"
	pluginEventHealthTimeout      = "PLUGIN_HEALTH_TIMEOUT"
	pluginEventWorkerRestarting   = "PLUGIN_WORKER_RESTARTING"
	pluginEventWorkerExited       = "PLUGIN_WORKER_EXITED"
	pluginEventWorkerStopped      = "PLUGIN_WORKER_STOPPED"
	pluginEventWorkerRestartLimit = "PLUGIN_WORKER_RESTART_LIMIT_REACHED"

	pluginLifecycleHookDispatchQueueSize = 64

	pluginWorkerErrorStderrTailMaxLines = 8
	pluginWorkerErrorStderrTailMaxChars = 1024
)

var errManualRestartRequested = errors.New("manual restart requested")

type ProcessPluginSupervisor struct {
	mu      sync.RWMutex
	hooks   PluginLifecycleHooks
	workers map[string]*managedPluginWorker

	started bool
	runCtx  context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	hookDispatchQueue chan pluginLifecycleHookDispatch
	hookDispatchWG    sync.WaitGroup
	hookDispatchDrops uint64
}

type managedPluginWorker struct {
	spec PluginWorkerSpec

	mu     sync.RWMutex
	status PluginWorkerStatus

	loopRunning bool

	stopSignal    chan struct{}
	restartSignal chan struct{}
}

type pluginWorkerMessage struct {
	Type    string                  `json:"type"`
	Plugin  *shared.AdapterMetadata `json:"plugin,omitempty"`
	Healthy *bool                   `json:"healthy,omitempty"`
}

type pluginLifecycleHookDispatch struct {
	hooks     PluginLifecycleHooks
	eventType string
	pluginID  string
	processID int
	err       error
	event     PluginLifecycleEvent
}

func NewProcessPluginSupervisor() *ProcessPluginSupervisor {
	return &ProcessPluginSupervisor{
		workers: map[string]*managedPluginWorker{},
	}
}

func (s *ProcessPluginSupervisor) SetHooks(hooks PluginLifecycleHooks) {
	s.mu.Lock()
	s.hooks = hooks
	s.mu.Unlock()
}

func (s *ProcessPluginSupervisor) RegisterWorker(spec PluginWorkerSpec) error {
	normalized, err := normalizePluginWorkerSpec(spec)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.workers[normalized.PluginID]; exists {
		return fmt.Errorf("plugin worker already registered: %s", normalized.PluginID)
	}

	worker := &managedPluginWorker{
		spec: normalized,
		status: PluginWorkerStatus{
			PluginID:       normalized.PluginID,
			Kind:           normalized.Kind,
			State:          PluginWorkerStateRegistered,
			LastTransition: time.Now().UTC(),
		},
		stopSignal:    make(chan struct{}, 1),
		restartSignal: make(chan struct{}, 1),
	}
	s.workers[normalized.PluginID] = worker

	if s.started {
		s.wg.Add(1)
		go s.workerLoop(s.runCtx, worker)
	}
	return nil
}

func (s *ProcessPluginSupervisor) ListWorkers() []PluginWorkerStatus {
	s.mu.RLock()
	ids := make([]string, 0, len(s.workers))
	for id := range s.workers {
		ids = append(ids, id)
	}
	workers := make([]*managedPluginWorker, 0, len(ids))
	for _, id := range ids {
		workers = append(workers, s.workers[id])
	}
	s.mu.RUnlock()

	sort.Strings(ids)
	statuses := make([]PluginWorkerStatus, 0, len(ids))
	byID := map[string]*managedPluginWorker{}
	for _, worker := range workers {
		byID[worker.spec.PluginID] = worker
	}
	for _, id := range ids {
		statuses = append(statuses, byID[id].snapshot())
	}
	return statuses
}

func (s *ProcessPluginSupervisor) WorkerStatus(pluginID string) (PluginWorkerStatus, bool) {
	s.mu.RLock()
	worker, ok := s.workers[strings.TrimSpace(pluginID)]
	s.mu.RUnlock()
	if !ok {
		return PluginWorkerStatus{}, false
	}
	return worker.snapshot(), true
}

func (s *ProcessPluginSupervisor) RestartWorker(_ context.Context, pluginID string) error {
	trimmedID := strings.TrimSpace(pluginID)

	s.mu.RLock()
	worker, ok := s.workers[trimmedID]
	started := s.started
	runCtx := s.runCtx
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("plugin worker not registered: %s", pluginID)
	}
	if worker.isLoopRunning() {
		select {
		case worker.restartSignal <- struct{}{}:
		default:
		}
		return nil
	}
	if !started || runCtx == nil {
		return fmt.Errorf("plugin supervisor is not running")
	}

	s.wg.Add(1)
	go s.workerLoop(runCtx, worker)
	return nil
}

func (s *ProcessPluginSupervisor) StopWorker(_ context.Context, pluginID string) error {
	s.mu.RLock()
	worker, ok := s.workers[strings.TrimSpace(pluginID)]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("plugin worker not registered: %s", pluginID)
	}
	if !worker.isLoopRunning() {
		worker.updateStatus(func(status *PluginWorkerStatus) {
			status.State = PluginWorkerStateStopped
			status.ProcessID = 0
			status.LastTransition = time.Now().UTC()
		})
		return nil
	}
	select {
	case worker.stopSignal <- struct{}{}:
	default:
	}
	return nil
}

func (s *ProcessPluginSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.runCtx, s.cancel = context.WithCancel(ctx)
	s.started = true
	s.ensureHookDispatcherLocked()

	workers := make([]*managedPluginWorker, 0, len(s.workers))
	for _, worker := range s.workers {
		workers = append(workers, worker)
	}
	s.mu.Unlock()

	for _, worker := range workers {
		s.wg.Add(1)
		go s.workerLoop(s.runCtx, worker)
	}
	return nil
}

func (s *ProcessPluginSupervisor) Stop(_ context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.started = false
	s.cancel = nil
	s.runCtx = nil
	hookDispatchQueue := s.hookDispatchQueue
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
	if hookDispatchQueue != nil {
		close(hookDispatchQueue)
		s.hookDispatchWG.Wait()

		s.mu.Lock()
		if s.hookDispatchQueue == hookDispatchQueue {
			s.hookDispatchQueue = nil
		}
		s.mu.Unlock()
	}
	return nil
}

func (s *ProcessPluginSupervisor) HookDispatchDroppedCount() uint64 {
	if s == nil {
		return 0
	}
	return atomic.LoadUint64(&s.hookDispatchDrops)
}
