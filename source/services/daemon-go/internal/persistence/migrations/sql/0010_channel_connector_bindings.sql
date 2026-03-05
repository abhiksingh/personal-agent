PRAGMA foreign_keys = ON;

CREATE TABLE channel_connector_bindings (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  connector_id TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  priority INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  CHECK (channel_id IN ('app', 'message', 'voice')),
  CHECK (length(trim(connector_id)) > 0),
  CHECK (enabled IN (0, 1)),
  CHECK (priority >= 1)
);

CREATE UNIQUE INDEX uq_channel_connector_bindings_workspace_channel_connector
  ON channel_connector_bindings(workspace_id, channel_id, connector_id);

CREATE UNIQUE INDEX uq_channel_connector_bindings_workspace_channel_priority
  ON channel_connector_bindings(workspace_id, channel_id, priority);

CREATE INDEX idx_channel_connector_bindings_workspace_channel_enabled_priority
  ON channel_connector_bindings(workspace_id, channel_id, enabled, priority);

CREATE TRIGGER trg_seed_channel_connector_bindings_after_workspace_insert
AFTER INSERT ON workspaces
BEGIN
  INSERT OR IGNORE INTO channel_connector_bindings (
    id,
    workspace_id,
    channel_id,
    connector_id,
    enabled,
    priority,
    created_at,
    updated_at
  ) VALUES (
    'ccb.' || NEW.id || '.app.builtin_app',
    NEW.id,
    'app',
    'builtin.app',
    1,
    1,
    NEW.updated_at,
    NEW.updated_at
  );

  INSERT OR IGNORE INTO channel_connector_bindings (
    id,
    workspace_id,
    channel_id,
    connector_id,
    enabled,
    priority,
    created_at,
    updated_at
  ) VALUES (
    'ccb.' || NEW.id || '.message.imessage',
    NEW.id,
    'message',
    'imessage',
    1,
    1,
    NEW.updated_at,
    NEW.updated_at
  );

  INSERT OR IGNORE INTO channel_connector_bindings (
    id,
    workspace_id,
    channel_id,
    connector_id,
    enabled,
    priority,
    created_at,
    updated_at
  ) VALUES (
    'ccb.' || NEW.id || '.message.twilio',
    NEW.id,
    'message',
    'twilio',
    1,
    2,
    NEW.updated_at,
    NEW.updated_at
  );

  INSERT OR IGNORE INTO channel_connector_bindings (
    id,
    workspace_id,
    channel_id,
    connector_id,
    enabled,
    priority,
    created_at,
    updated_at
  ) VALUES (
    'ccb.' || NEW.id || '.voice.twilio',
    NEW.id,
    'voice',
    'twilio',
    1,
    1,
    NEW.updated_at,
    NEW.updated_at
  );
END;

INSERT OR IGNORE INTO channel_connector_bindings (
  id,
  workspace_id,
  channel_id,
  connector_id,
  enabled,
  priority,
  created_at,
  updated_at
)
SELECT
  'ccb.' || w.id || '.app.builtin_app',
  w.id,
  'app',
  'builtin.app',
  1,
  1,
  w.updated_at,
  w.updated_at
FROM workspaces w;

INSERT OR IGNORE INTO channel_connector_bindings (
  id,
  workspace_id,
  channel_id,
  connector_id,
  enabled,
  priority,
  created_at,
  updated_at
)
SELECT
  'ccb.' || w.id || '.message.imessage',
  w.id,
  'message',
  'imessage',
  1,
  1,
  w.updated_at,
  w.updated_at
FROM workspaces w;

INSERT OR IGNORE INTO channel_connector_bindings (
  id,
  workspace_id,
  channel_id,
  connector_id,
  enabled,
  priority,
  created_at,
  updated_at
)
SELECT
  'ccb.' || w.id || '.message.twilio',
  w.id,
  'message',
  'twilio',
  1,
  2,
  w.updated_at,
  w.updated_at
FROM workspaces w;

INSERT OR IGNORE INTO channel_connector_bindings (
  id,
  workspace_id,
  channel_id,
  connector_id,
  enabled,
  priority,
  created_at,
  updated_at
)
SELECT
  'ccb.' || w.id || '.voice.twilio',
  w.id,
  'voice',
  'twilio',
  1,
  1,
  w.updated_at,
  w.updated_at
FROM workspaces w;
