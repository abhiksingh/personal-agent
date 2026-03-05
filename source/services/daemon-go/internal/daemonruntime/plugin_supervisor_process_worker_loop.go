package daemonruntime

import (
	"context"
	"errors"
	"time"
)

func (s *ProcessPluginSupervisor) workerLoop(runCtx context.Context, worker *managedPluginWorker) {
	if !worker.beginLoop() {
		s.wg.Done()
		return
	}
	defer worker.endLoop()
	defer s.wg.Done()

	for {
		select {
		case <-runCtx.Done():
			worker.updateStatus(func(status *PluginWorkerStatus) {
				status.State = PluginWorkerStateStopped
				status.ProcessID = 0
				status.execAuthToken = ""
				status.LastTransition = time.Now().UTC()
			})
			s.emitEvent(worker.snapshot(), pluginEventWorkerStopped, nil)
			return
		default:
		}

		restart, stopped, err := s.runWorkerProcess(runCtx, worker)
		if stopped {
			worker.updateStatus(func(status *PluginWorkerStatus) {
				status.State = PluginWorkerStateStopped
				status.ProcessID = 0
				status.execAuthToken = ""
				status.LastTransition = time.Now().UTC()
			})
			s.emitEvent(worker.snapshot(), pluginEventWorkerStopped, err)
			return
		}
		if !restart {
			worker.updateStatus(func(status *PluginWorkerStatus) {
				status.State = PluginWorkerStateFailed
				status.ProcessID = 0
				status.execAuthToken = ""
				status.LastTransition = time.Now().UTC()
				applyPluginWorkerErrorContext(status, err)
			})
			s.emitEvent(worker.snapshot(), pluginEventWorkerExited, err)
			return
		}

		manualRestart := errors.Is(err, errManualRestartRequested)
		eventErr := err
		if manualRestart {
			eventErr = nil
		}

		status := worker.snapshot()
		if !manualRestart && status.RestartCount >= worker.spec.RestartPolicy.MaxRestarts {
			worker.updateStatus(func(current *PluginWorkerStatus) {
				current.State = PluginWorkerStateFailed
				current.ProcessID = 0
				current.execAuthToken = ""
				current.LastTransition = time.Now().UTC()
				applyPluginWorkerErrorContext(current, err)
			})
			s.emitEvent(worker.snapshot(), pluginEventWorkerRestartLimit, eventErr)
			return
		}

		worker.updateStatus(func(current *PluginWorkerStatus) {
			if !manualRestart {
				current.RestartCount++
			}
			current.State = PluginWorkerStateRestarting
			current.ProcessID = 0
			current.execAuthToken = ""
			current.LastTransition = time.Now().UTC()
			if !manualRestart && err != nil {
				applyPluginWorkerErrorContext(current, err)
			} else {
				clearPluginWorkerErrorContext(current)
			}
		})
		s.emitEvent(worker.snapshot(), pluginEventWorkerRestarting, eventErr)

		select {
		case <-runCtx.Done():
			worker.updateStatus(func(current *PluginWorkerStatus) {
				current.State = PluginWorkerStateStopped
				current.execAuthToken = ""
				current.LastTransition = time.Now().UTC()
			})
			s.emitEvent(worker.snapshot(), pluginEventWorkerStopped, nil)
			return
		case <-time.After(worker.spec.RestartPolicy.Delay):
		}
	}
}
