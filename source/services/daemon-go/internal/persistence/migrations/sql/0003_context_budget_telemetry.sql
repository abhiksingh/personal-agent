CREATE TABLE context_budget_samples (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  task_class TEXT NOT NULL,
  model_key TEXT,
  context_window INTEGER NOT NULL,
  output_limit INTEGER,
  deep_analysis INTEGER NOT NULL DEFAULT 0,
  remaining_budget INTEGER NOT NULL,
  retrieval_target INTEGER NOT NULL,
  retrieval_used INTEGER NOT NULL,
  prompt_tokens INTEGER NOT NULL,
  completion_tokens INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE INDEX idx_context_budget_samples_workspace_class_created
  ON context_budget_samples(workspace_id, task_class, created_at DESC);

CREATE TABLE context_budget_tuning_profiles (
  workspace_id TEXT NOT NULL,
  task_class TEXT NOT NULL,
  retrieval_multiplier REAL NOT NULL,
  sample_count INTEGER NOT NULL DEFAULT 0,
  avg_retrieval_utilization REAL NOT NULL DEFAULT 0,
  avg_prompt_utilization REAL NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (workspace_id, task_class),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);
