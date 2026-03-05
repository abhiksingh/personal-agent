CREATE TABLE comm_ingest_receipts (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  source TEXT NOT NULL,
  source_scope TEXT NOT NULL,
  source_event_id TEXT NOT NULL,
  source_cursor TEXT,
  trust_state TEXT NOT NULL,
  event_id TEXT,
  payload_hash TEXT,
  received_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (event_id) REFERENCES comm_events(id)
);

CREATE TABLE comm_ingest_cursors (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  source TEXT NOT NULL,
  source_scope TEXT NOT NULL,
  cursor_value TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE automation_source_subscriptions (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  source TEXT NOT NULL,
  source_scope TEXT NOT NULL,
  status TEXT NOT NULL,
  config_json TEXT,
  last_cursor TEXT,
  last_event_id TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (last_event_id) REFERENCES comm_events(id)
);

CREATE UNIQUE INDEX uq_comm_ingest_receipts_workspace_source_event
  ON comm_ingest_receipts(workspace_id, source, source_event_id);

CREATE UNIQUE INDEX uq_comm_ingest_cursors_workspace_source_scope
  ON comm_ingest_cursors(workspace_id, source, source_scope);

CREATE UNIQUE INDEX uq_automation_source_subscriptions_workspace_source_scope
  ON automation_source_subscriptions(workspace_id, source, source_scope);

CREATE INDEX idx_automation_source_subscriptions_workspace_source
  ON automation_source_subscriptions(workspace_id, source, source_scope);
