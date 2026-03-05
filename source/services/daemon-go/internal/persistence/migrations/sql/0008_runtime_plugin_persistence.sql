CREATE TABLE runtime_plugins (
  workspace_id TEXT NOT NULL,
  plugin_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  display_name TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  capabilities_json TEXT NOT NULL DEFAULT '[]',
  runtime_json TEXT,
  status TEXT NOT NULL DEFAULT 'registered',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (workspace_id, plugin_id),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  CHECK (kind IN ('channel', 'connector')),
  CHECK (status IN ('registered', 'starting', 'running', 'restarting', 'stopped', 'failed'))
);

CREATE TABLE runtime_plugin_processes (
  workspace_id TEXT NOT NULL,
  plugin_id TEXT NOT NULL,
  state TEXT NOT NULL,
  process_id INTEGER NOT NULL DEFAULT 0,
  restart_count INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  last_heartbeat_at TEXT,
  last_transition_at TEXT NOT NULL,
  event_type TEXT NOT NULL,
  metadata_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (workspace_id, plugin_id),
  FOREIGN KEY (workspace_id, plugin_id) REFERENCES runtime_plugins(workspace_id, plugin_id),
  CHECK (state IN ('registered', 'starting', 'running', 'restarting', 'stopped', 'failed')),
  CHECK (event_type IN (
    'PLUGIN_WORKER_STARTED',
    'PLUGIN_HANDSHAKE_ACCEPTED',
    'PLUGIN_HEALTH_TIMEOUT',
    'PLUGIN_WORKER_RESTARTING',
    'PLUGIN_WORKER_EXITED',
    'PLUGIN_WORKER_STOPPED',
    'PLUGIN_WORKER_RESTART_LIMIT_REACHED'
  )),
  CHECK (restart_count >= 0)
);

CREATE INDEX idx_runtime_plugins_workspace_kind_status
ON runtime_plugins(workspace_id, kind, status);

CREATE INDEX idx_runtime_plugin_processes_workspace_state_updated
ON runtime_plugin_processes(workspace_id, state, updated_at);
