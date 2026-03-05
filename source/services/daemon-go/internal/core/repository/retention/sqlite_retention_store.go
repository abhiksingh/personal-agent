package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SQLiteRetentionStore struct {
	db *sql.DB
}

func NewSQLiteRetentionStore(db *sql.DB) *SQLiteRetentionStore {
	return &SQLiteRetentionStore{db: db}
}

func (s *SQLiteRetentionStore) PurgeTraceDataBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
	total := int64(0)
	for index, stmt := range []string{
		`DELETE FROM run_artifacts WHERE created_at < ?`,
		`DELETE FROM audit_log_entries WHERE created_at < ?`,
	} {
		affected, err := s.executeRetentionDeleteStatement(ctx, stmt, cutoffText)
		if err != nil {
			return total, fmt.Errorf("purge trace data statement %d failed: %w", index+1, err)
		}
		total += affected
	}
	return total, nil
}

func (s *SQLiteRetentionStore) PurgeTranscriptDataBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
	total := int64(0)
	statements := []string{
		`DELETE FROM comm_event_addresses
		 WHERE event_id IN (SELECT id FROM comm_events WHERE created_at < ?)`,
		`DELETE FROM comm_attachments
		 WHERE event_id IN (SELECT id FROM comm_events WHERE created_at < ?)`,
		`DELETE FROM email_event_meta
		 WHERE event_id IN (SELECT id FROM comm_events WHERE created_at < ?)`,
		`DELETE FROM comm_events WHERE created_at < ?`,
	}
	for index, stmt := range statements {
		affected, err := s.executeRetentionDeleteStatement(ctx, stmt, cutoffText)
		if err != nil {
			return total, fmt.Errorf("purge transcript data statement %d failed: %w", index+1, err)
		}
		total += affected
	}
	return total, nil
}

func (s *SQLiteRetentionStore) PurgeMemoryDataBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
	total := int64(0)
	statements := []string{
		`DELETE FROM memory_sources
		 WHERE memory_item_id IN (SELECT id FROM memory_items WHERE created_at < ?)`,
		`DELETE FROM memory_items WHERE created_at < ?`,
		`DELETE FROM memory_candidates WHERE created_at < ?`,
		`DELETE FROM context_chunks
		 WHERE document_id IN (SELECT id FROM context_documents WHERE created_at < ?)`,
		`DELETE FROM context_documents WHERE created_at < ?`,
	}
	for index, stmt := range statements {
		affected, err := s.executeRetentionDeleteStatement(ctx, stmt, cutoffText)
		if err != nil {
			return total, fmt.Errorf("purge memory data statement %d failed: %w", index+1, err)
		}
		total += affected
	}
	return total, nil
}

func (s *SQLiteRetentionStore) executeRetentionDeleteStatement(ctx context.Context, statement string, cutoffText string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin retention delete transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, statement, cutoffText)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit retention delete transaction: %w", err)
	}
	return affected, nil
}
