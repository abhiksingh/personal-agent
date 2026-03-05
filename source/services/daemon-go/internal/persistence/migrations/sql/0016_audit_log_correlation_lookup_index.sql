CREATE INDEX idx_audit_log_entries_correlation_created_id
ON audit_log_entries(correlation_id, created_at DESC, id DESC);
