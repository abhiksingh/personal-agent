CREATE TABLE comm_call_sessions (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_call_id TEXT NOT NULL,
  provider_account_id TEXT,
  thread_id TEXT NOT NULL,
  direction TEXT NOT NULL,
  from_address TEXT,
  to_address TEXT,
  status TEXT NOT NULL,
  started_at TEXT,
  ended_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (thread_id) REFERENCES comm_threads(id),
  UNIQUE (workspace_id, provider, provider_call_id)
);

CREATE UNIQUE INDEX uq_comm_call_sessions_workspace_provider_call
  ON comm_call_sessions(workspace_id, provider, provider_call_id);

CREATE INDEX idx_comm_call_sessions_workspace_status_updated
  ON comm_call_sessions(workspace_id, status, updated_at);
