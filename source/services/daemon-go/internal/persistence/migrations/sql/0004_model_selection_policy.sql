CREATE TABLE model_catalog_entries (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  model_key TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  UNIQUE (workspace_id, provider, model_key)
);

CREATE TABLE model_routing_policies (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  task_class TEXT NOT NULL,
  provider TEXT NOT NULL,
  model_key TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  UNIQUE (workspace_id, task_class)
);

CREATE INDEX idx_model_catalog_workspace_provider_enabled
  ON model_catalog_entries(workspace_id, provider, enabled);

CREATE INDEX idx_model_routing_workspace_task_class
  ON model_routing_policies(workspace_id, task_class);
