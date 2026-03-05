PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS chat_turn_items (
  id TEXT PRIMARY KEY,
  turn_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  task_class TEXT NOT NULL,
  correlation_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  connector_id TEXT,
  thread_id TEXT,
  item_index INTEGER NOT NULL,
  item_type TEXT NOT NULL,
  role TEXT,
  status TEXT,
  content TEXT,
  tool_name TEXT,
  tool_call_id TEXT,
  arguments_json TEXT,
  output_json TEXT,
  error_code TEXT,
  error_message TEXT,
  approval_request_id TEXT,
  metadata_json TEXT,
  task_id TEXT,
  run_id TEXT,
  task_state TEXT,
  run_state TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  CHECK (channel_id IN ('app', 'message', 'voice')),
  CHECK (item_type IN ('user_message', 'assistant_message', 'tool_call', 'tool_result', 'approval_request', 'approval_decision'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_turn_items_turn_index ON chat_turn_items(turn_id, item_index);
CREATE INDEX IF NOT EXISTS idx_chat_turn_items_workspace_channel_thread_created ON chat_turn_items(workspace_id, channel_id, thread_id, created_at, item_index);
CREATE INDEX IF NOT EXISTS idx_chat_turn_items_workspace_correlation_created ON chat_turn_items(workspace_id, correlation_id, created_at, item_index);

CREATE TABLE IF NOT EXISTS chat_persona_policies (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  principal_actor_id TEXT NOT NULL DEFAULT '',
  channel_id TEXT NOT NULL DEFAULT '',
  style_prompt TEXT NOT NULL,
  guardrails_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  UNIQUE (workspace_id, principal_actor_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_persona_policies_workspace_scope ON chat_persona_policies(workspace_id, principal_actor_id, channel_id, updated_at);
