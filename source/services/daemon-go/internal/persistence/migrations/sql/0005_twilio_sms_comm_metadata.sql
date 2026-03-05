CREATE TABLE comm_webhook_receipts (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_event_id TEXT NOT NULL,
  signature_valid INTEGER NOT NULL DEFAULT 0,
  signature_value TEXT,
  payload_hash TEXT,
  event_id TEXT,
  received_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (event_id) REFERENCES comm_events(id),
  UNIQUE (workspace_id, provider, provider_event_id)
);

CREATE TABLE comm_provider_messages (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_message_id TEXT NOT NULL,
  provider_account_id TEXT,
  channel TEXT NOT NULL,
  direction TEXT NOT NULL,
  from_address TEXT,
  to_address TEXT,
  status TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (event_id) REFERENCES comm_events(id),
  UNIQUE (workspace_id, provider, provider_message_id)
);

CREATE UNIQUE INDEX uq_comm_webhook_receipts_workspace_provider_event
  ON comm_webhook_receipts(workspace_id, provider, provider_event_id);

CREATE UNIQUE INDEX uq_comm_provider_messages_workspace_provider_message
  ON comm_provider_messages(workspace_id, provider, provider_message_id);

CREATE INDEX idx_comm_provider_messages_event_id
  ON comm_provider_messages(event_id);
