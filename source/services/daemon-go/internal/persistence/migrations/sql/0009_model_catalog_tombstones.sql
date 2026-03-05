CREATE TABLE model_catalog_tombstones (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  model_key TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE (workspace_id, provider, model_key),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE INDEX idx_model_tombstones_workspace_provider
  ON model_catalog_tombstones(workspace_id, provider);
