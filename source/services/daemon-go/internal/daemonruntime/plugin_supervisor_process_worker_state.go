package daemonruntime

func (w *managedPluginWorker) snapshot() PluginWorkerStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

func (w *managedPluginWorker) updateStatus(update func(status *PluginWorkerStatus)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	update(&w.status)
}

func (w *managedPluginWorker) isLoopRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.loopRunning
}

func (w *managedPluginWorker) beginLoop() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.loopRunning {
		return false
	}
	w.loopRunning = true
	return true
}

func (w *managedPluginWorker) endLoop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.loopRunning = false
}
