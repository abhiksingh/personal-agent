PRAGMA foreign_keys = ON;

DELETE FROM channel_connector_bindings
WHERE LOWER(TRIM(connector_id)) = 'apple.messages'
  AND EXISTS (
    SELECT 1
    FROM channel_connector_bindings existing
    WHERE existing.workspace_id = channel_connector_bindings.workspace_id
      AND existing.channel_id = channel_connector_bindings.channel_id
      AND LOWER(TRIM(existing.connector_id)) = 'imessage'
  );

UPDATE channel_connector_bindings
SET connector_id = 'imessage'
WHERE LOWER(TRIM(connector_id)) = 'apple.messages';

UPDATE channel_connector_bindings
SET id = REPLACE(id, '.message.apple_messages', '.message.imessage')
WHERE id LIKE 'ccb.%.message.apple_messages';

DROP TRIGGER IF EXISTS trg_seed_channel_connector_bindings_after_workspace_insert;

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

UPDATE comm_threads
SET connector_id = 'imessage'
WHERE LOWER(TRIM(connector_id)) = 'apple.messages';

UPDATE comm_events
SET connector_id = 'imessage'
WHERE LOWER(TRIM(connector_id)) = 'apple.messages';

UPDATE comm_call_sessions
SET connector_id = 'imessage'
WHERE LOWER(TRIM(connector_id)) = 'apple.messages';

UPDATE delivery_attempts
SET channel = 'imessage'
WHERE LOWER(TRIM(channel)) = 'apple.messages';

UPDATE channel_delivery_policies
SET policy_json = REPLACE(policy_json, '"apple.messages"', '"imessage"')
WHERE policy_json LIKE '%apple.messages%';

UPDATE channel_connectors
SET connector_type = 'imessage'
WHERE LOWER(TRIM(connector_type)) IN ('apple.messages', 'messages', 'channel.messages');
