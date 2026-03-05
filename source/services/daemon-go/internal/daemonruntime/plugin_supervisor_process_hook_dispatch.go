package daemonruntime

import (
	"sync/atomic"
	"time"
)

func (s *ProcessPluginSupervisor) emitEvent(status PluginWorkerStatus, eventType string, err error) {
	s.mu.RLock()
	hooks := s.hooks
	hookDispatchQueue := s.hookDispatchQueue
	s.mu.RUnlock()
	errorContext := pluginWorkerEventErrorContext(status, err)
	dispatch := pluginLifecycleHookDispatch{
		hooks:     hooks,
		eventType: eventType,
		pluginID:  status.PluginID,
		processID: status.ProcessID,
		err:       err,
		event: PluginLifecycleEvent{
			PluginID:       status.PluginID,
			Kind:           status.Kind,
			State:          status.State,
			EventType:      eventType,
			ProcessID:      status.ProcessID,
			RestartCount:   status.RestartCount,
			Error:          errorContext.Message,
			ErrorSource:    errorContext.Source,
			ErrorOperation: errorContext.Operation,
			ErrorStderr:    errorContext.Stderr,
			OccurredAt:     time.Now().UTC(),
			LastHeartbeat:  status.LastHeartbeat,
			LastTransition: status.LastTransition,
			Metadata:       status.Metadata,
		},
	}
	if hookDispatchQueue == nil {
		dispatch.invoke()
		return
	}
	s.enqueueHookDispatch(hookDispatchQueue, dispatch)
}

func (s *ProcessPluginSupervisor) enqueueHookDispatch(
	queue chan pluginLifecycleHookDispatch,
	dispatch pluginLifecycleHookDispatch,
) {
	select {
	case queue <- dispatch:
		return
	default:
	}

	// Queue saturation policy: drop the oldest buffered event to keep worker
	// loops progressing and preserve latest lifecycle state visibility.
	dropped := false
	select {
	case <-queue:
		dropped = true
	default:
	}
	if dropped {
		atomic.AddUint64(&s.hookDispatchDrops, 1)
	}

	select {
	case queue <- dispatch:
		return
	default:
		// If queue is still unavailable, drop the current event and account it.
		atomic.AddUint64(&s.hookDispatchDrops, 1)
	}
}

func (s *ProcessPluginSupervisor) ensureHookDispatcherLocked() {
	if s.hookDispatchQueue != nil {
		return
	}
	queue := make(chan pluginLifecycleHookDispatch, pluginLifecycleHookDispatchQueueSize)
	s.hookDispatchQueue = queue
	s.hookDispatchWG.Add(1)
	go s.runHookDispatcher(queue)
}

func (s *ProcessPluginSupervisor) runHookDispatcher(queue <-chan pluginLifecycleHookDispatch) {
	defer s.hookDispatchWG.Done()
	for dispatch := range queue {
		dispatch.invoke()
	}
}

func (d pluginLifecycleHookDispatch) invoke() {
	if d.hooks.OnWorkerStart != nil && d.eventType == pluginEventWorkerStarted {
		d.hooks.OnWorkerStart(d.pluginID, d.processID)
	}
	if d.hooks.OnWorkerExit != nil && (d.eventType == pluginEventWorkerExited || d.eventType == pluginEventWorkerStopped || d.eventType == pluginEventWorkerRestartLimit) {
		d.hooks.OnWorkerExit(d.pluginID, d.processID, d.err)
	}
	if d.hooks.OnEvent != nil {
		d.hooks.OnEvent(d.event)
	}
}
