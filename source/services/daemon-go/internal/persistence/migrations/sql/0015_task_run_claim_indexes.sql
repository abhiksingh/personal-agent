CREATE INDEX idx_task_runs_claim_queued_state_created_at
ON task_runs(state, created_at)
WHERE state = 'queued';

CREATE INDEX idx_task_runs_claim_running_state_updated_created
ON task_runs(state, updated_at, created_at)
WHERE state = 'running';
