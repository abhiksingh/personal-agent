CREATE TABLE persistence_write_queue (
  id TEXT PRIMARY KEY,
  operation TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 5,
  available_at TEXT NOT NULL,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (status IN ('pending', 'in_progress', 'completed', 'failed'))
);

CREATE INDEX idx_persistence_write_queue_status_available
  ON persistence_write_queue(status, available_at, created_at);

CREATE INDEX idx_persistence_write_queue_operation_status
  ON persistence_write_queue(operation, status);
