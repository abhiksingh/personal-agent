ALTER TABLE comm_threads ADD COLUMN connector_id TEXT NOT NULL DEFAULT '';
ALTER TABLE comm_events ADD COLUMN connector_id TEXT NOT NULL DEFAULT '';
ALTER TABLE comm_call_sessions ADD COLUMN connector_id TEXT NOT NULL DEFAULT '';

UPDATE comm_threads
SET connector_id = CASE
  WHEN LOWER(TRIM(channel)) IN ('twilio_sms', 'twilio_voice', 'sms', 'voice', 'twilio') THEN 'twilio'
  WHEN LOWER(TRIM(channel)) IN ('imessage', 'imessage_sms', 'messages') THEN 'imessage'
  WHEN LOWER(TRIM(channel)) = 'message' THEN
    CASE
      WHEN EXISTS (
        SELECT 1
        FROM comm_provider_messages cpm
        JOIN comm_events ce ON ce.id = cpm.event_id
        WHERE ce.thread_id = comm_threads.id
          AND LOWER(COALESCE(cpm.provider, '')) = 'twilio'
      ) THEN 'twilio'
      ELSE 'imessage'
    END
  WHEN LOWER(TRIM(channel)) IN ('app', 'app_chat') THEN 'builtin.app'
  WHEN LOWER(TRIM(channel)) = 'mail' THEN 'mail'
  WHEN LOWER(TRIM(channel)) = 'calendar' THEN 'calendar'
  WHEN LOWER(TRIM(channel)) = 'browser' THEN 'browser'
  WHEN LOWER(TRIM(channel)) = 'finder' THEN 'finder'
  ELSE LOWER(TRIM(channel))
END
WHERE TRIM(COALESCE(connector_id, '')) = '';

UPDATE comm_events
SET connector_id = COALESCE((
  SELECT t.connector_id
  FROM comm_threads t
  WHERE t.id = comm_events.thread_id
), '')
WHERE TRIM(COALESCE(connector_id, '')) = '';

UPDATE comm_call_sessions
SET connector_id = CASE
  WHEN LOWER(TRIM(provider)) = 'twilio' THEN 'twilio'
  ELSE COALESCE((
    SELECT t.connector_id
    FROM comm_threads t
    WHERE t.id = comm_call_sessions.thread_id
  ), LOWER(TRIM(provider)))
END
WHERE TRIM(COALESCE(connector_id, '')) = '';

CREATE INDEX idx_comm_threads_workspace_channel_connector_updated
  ON comm_threads(workspace_id, channel, connector_id, updated_at, id);

CREATE INDEX idx_comm_events_workspace_connector_occurred
  ON comm_events(workspace_id, connector_id, occurred_at, id);

CREATE INDEX idx_comm_call_sessions_workspace_connector_updated
  ON comm_call_sessions(workspace_id, connector_id, updated_at, id);
