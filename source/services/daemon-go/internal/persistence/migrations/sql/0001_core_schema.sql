PRAGMA foreign_keys = ON;

CREATE TABLE workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT,
  display_name TEXT,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE workspace_members (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE (workspace_id, user_id)
);

CREATE TABLE actors (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE workspace_principals (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id),
  UNIQUE (workspace_id, actor_id)
);

CREATE TABLE actor_handles (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  handle_value TEXT NOT NULL,
  is_primary INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id),
  UNIQUE (workspace_id, channel, handle_value)
);

CREATE TABLE user_actor_links (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id),
  UNIQUE (user_id, actor_id)
);

CREATE TABLE comm_threads (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  external_ref TEXT,
  title TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE comm_events (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  thread_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  direction TEXT NOT NULL,
  assistant_emitted INTEGER NOT NULL DEFAULT 0,
  occurred_at TEXT NOT NULL,
  body_text TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (thread_id) REFERENCES comm_threads(id)
);

CREATE TABLE comm_event_addresses (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  address_role TEXT NOT NULL,
  address_value TEXT NOT NULL,
  display_name TEXT,
  position INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (event_id) REFERENCES comm_events(id),
  CHECK (address_role IN ('FROM', 'TO', 'CC', 'BCC', 'REPLY_TO'))
);

CREATE TABLE comm_attachments (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  file_name TEXT,
  media_type TEXT,
  uri TEXT,
  content_hash TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (event_id) REFERENCES comm_events(id)
);

CREATE TABLE email_event_meta (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  message_id TEXT,
  in_reply_to TEXT,
  references_header TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (event_id) REFERENCES comm_events(id),
  UNIQUE (event_id)
);

CREATE TABLE thread_participants (
  id TEXT PRIMARY KEY,
  thread_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  FOREIGN KEY (thread_id) REFERENCES comm_threads(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id),
  UNIQUE (thread_id, actor_id)
);

CREATE TABLE thread_local_identities (
  id TEXT PRIMARY KEY,
  thread_id TEXT NOT NULL,
  handle_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (thread_id) REFERENCES comm_threads(id),
  FOREIGN KEY (handle_id) REFERENCES actor_handles(id),
  UNIQUE (thread_id, handle_id)
);

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  requested_by_actor_id TEXT NOT NULL,
  subject_principal_actor_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  state TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  deadline_at TEXT,
  channel TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (requested_by_actor_id) REFERENCES actors(id),
  FOREIGN KEY (subject_principal_actor_id) REFERENCES actors(id)
);

CREATE TABLE task_runs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  acting_as_actor_id TEXT NOT NULL,
  state TEXT NOT NULL,
  started_at TEXT,
  finished_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id),
  FOREIGN KEY (workspace_id, acting_as_actor_id) REFERENCES workspace_principals(workspace_id, actor_id)
);

CREATE TABLE task_plans (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  plan_text TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES task_runs(id)
);

CREATE TABLE task_steps (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  step_index INTEGER NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  interaction_level TEXT,
  capability_key TEXT,
  timeout_seconds INTEGER,
  retry_max INTEGER NOT NULL DEFAULT 0,
  retry_count INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES task_runs(id),
  UNIQUE (run_id, step_index)
);

CREATE TABLE task_step_recipients (
  id TEXT PRIMARY KEY,
  step_id TEXT NOT NULL,
  address_role TEXT NOT NULL,
  recipient_value TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (step_id) REFERENCES task_steps(id)
);

CREATE TABLE delivery_attempts (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  step_id TEXT,
  event_id TEXT,
  destination_endpoint TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  channel TEXT NOT NULL,
  provider_receipt TEXT,
  status TEXT NOT NULL,
  error TEXT,
  attempted_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (step_id) REFERENCES task_steps(id),
  FOREIGN KEY (event_id) REFERENCES comm_events(id)
);

CREATE TABLE run_artifacts (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  step_id TEXT,
  artifact_type TEXT NOT NULL,
  uri TEXT,
  content_hash TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES task_runs(id),
  FOREIGN KEY (step_id) REFERENCES task_steps(id)
);

CREATE TABLE approval_requests (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  run_id TEXT,
  step_id TEXT,
  requested_phrase TEXT NOT NULL,
  decision TEXT,
  decision_by_actor_id TEXT,
  requested_at TEXT NOT NULL,
  decided_at TEXT,
  rationale TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (run_id) REFERENCES task_runs(id),
  FOREIGN KEY (step_id) REFERENCES task_steps(id),
  FOREIGN KEY (decision_by_actor_id) REFERENCES actors(id),
  CHECK (
    (run_id IS NOT NULL AND step_id IS NULL)
    OR
    (run_id IS NULL AND step_id IS NOT NULL)
  )
);

CREATE TABLE audit_log_entries (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  run_id TEXT,
  step_id TEXT,
  event_type TEXT NOT NULL,
  actor_id TEXT,
  acting_as_actor_id TEXT,
  correlation_id TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (run_id) REFERENCES task_runs(id),
  FOREIGN KEY (step_id) REFERENCES task_steps(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id),
  FOREIGN KEY (acting_as_actor_id) REFERENCES actors(id)
);

CREATE TABLE directives (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  subject_principal_actor_id TEXT NOT NULL,
  title TEXT NOT NULL,
  instruction TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (workspace_id, subject_principal_actor_id) REFERENCES workspace_principals(workspace_id, actor_id)
);

CREATE TABLE automation_triggers (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  directive_id TEXT NOT NULL,
  trigger_type TEXT NOT NULL,
  is_enabled INTEGER NOT NULL DEFAULT 1,
  filter_json TEXT,
  cooldown_seconds INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (directive_id) REFERENCES directives(id)
);

CREATE TABLE trigger_fires (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  trigger_id TEXT NOT NULL,
  source_event_id TEXT NOT NULL,
  fired_at TEXT NOT NULL,
  task_id TEXT,
  outcome TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (trigger_id) REFERENCES automation_triggers(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE user_devices (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  device_type TEXT NOT NULL,
  platform TEXT NOT NULL,
  label TEXT,
  last_seen_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE device_sessions (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  session_token_hash TEXT NOT NULL,
  started_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  revoked_at TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES user_devices(id)
);

CREATE TABLE capability_grants (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  capability_key TEXT NOT NULL,
  scope_json TEXT,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id)
);

CREATE TABLE channel_connectors (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  connector_type TEXT NOT NULL,
  status TEXT NOT NULL,
  config_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE secret_refs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  owner_type TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  keychain_account TEXT NOT NULL,
  keychain_service TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  UNIQUE (workspace_id, owner_type, owner_id, keychain_account, keychain_service)
);

CREATE TABLE delegation_rules (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  from_actor_id TEXT NOT NULL,
  to_actor_id TEXT NOT NULL,
  scope_type TEXT NOT NULL,
  scope_key TEXT,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (from_actor_id) REFERENCES actors(id),
  FOREIGN KEY (to_actor_id) REFERENCES actors(id)
);

CREATE TABLE channel_delivery_policies (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  endpoint_pattern TEXT,
  policy_json TEXT NOT NULL,
  is_default INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE memory_items (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  owner_principal_actor_id TEXT NOT NULL,
  scope_type TEXT NOT NULL,
  key TEXT NOT NULL,
  value_json TEXT NOT NULL,
  status TEXT NOT NULL,
  source_summary TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (owner_principal_actor_id) REFERENCES actors(id)
);

CREATE TABLE memory_sources (
  id TEXT PRIMARY KEY,
  memory_item_id TEXT NOT NULL,
  source_type TEXT NOT NULL,
  source_ref TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (memory_item_id) REFERENCES memory_items(id)
);

CREATE TABLE memory_candidates (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  owner_principal_actor_id TEXT NOT NULL,
  candidate_json TEXT NOT NULL,
  score REAL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (owner_principal_actor_id) REFERENCES actors(id)
);

CREATE TABLE context_documents (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  owner_principal_actor_id TEXT,
  source_uri TEXT,
  checksum TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (owner_principal_actor_id) REFERENCES actors(id)
);

CREATE TABLE context_chunks (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  chunk_index INTEGER NOT NULL,
  text_body TEXT NOT NULL,
  token_count INTEGER,
  created_at TEXT NOT NULL,
  FOREIGN KEY (document_id) REFERENCES context_documents(id),
  UNIQUE (document_id, chunk_index)
);

CREATE INDEX idx_comm_events_thread_occurred ON comm_events(thread_id, occurred_at);
CREATE INDEX idx_comm_event_addresses_event_role ON comm_event_addresses(event_id, address_role);
CREATE INDEX idx_tasks_workspace_state_created ON tasks(workspace_id, state, created_at);
CREATE INDEX idx_task_runs_task_state_started ON task_runs(task_id, state, started_at);
CREATE INDEX idx_task_steps_run_index_status ON task_steps(run_id, step_index, status);
CREATE INDEX idx_memory_items_owner_scope_key_status ON memory_items(owner_principal_actor_id, scope_type, key, status);
CREATE UNIQUE INDEX uq_trigger_fires_workspace_trigger_source ON trigger_fires(workspace_id, trigger_id, source_event_id);
CREATE UNIQUE INDEX uq_delivery_attempts_endpoint_idempotency ON delivery_attempts(destination_endpoint, idempotency_key);
