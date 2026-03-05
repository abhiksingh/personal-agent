package migrator

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func objectExists(t *testing.T, db *sql.DB, objectType, name string) bool {
	t.Helper()
	var found string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = ? AND name = ?",
		objectType,
		name,
	).Scan(&found)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query sqlite_master for %s %s: %v", objectType, name, err)
	}
	return true
}

func TestApplyCreatesRequiredTablesAndIndexes(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	applied, err := Apply(ctx, db)
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if len(applied) == 0 {
		t.Fatalf("expected at least one migration to apply")
	}

	tables := []string{
		"workspaces",
		"users",
		"workspace_members",
		"actors",
		"workspace_principals",
		"actor_handles",
		"user_actor_links",
		"comm_threads",
		"comm_events",
		"comm_event_addresses",
		"comm_ingest_receipts",
		"comm_ingest_cursors",
		"comm_webhook_receipts",
		"comm_provider_messages",
		"comm_call_sessions",
		"automation_source_subscriptions",
		"comm_attachments",
		"email_event_meta",
		"thread_participants",
		"thread_local_identities",
		"tasks",
		"task_runs",
		"task_plans",
		"task_steps",
		"task_step_recipients",
		"delivery_attempts",
		"run_artifacts",
		"approval_requests",
		"audit_log_entries",
		"directives",
		"automation_triggers",
		"trigger_fires",
		"user_devices",
		"device_sessions",
		"capability_grants",
		"channel_connectors",
		"channel_connector_bindings",
		"secret_refs",
		"runtime_plugins",
		"runtime_plugin_processes",
		"delegation_rules",
		"channel_delivery_policies",
		"memory_items",
		"memory_sources",
		"memory_candidates",
		"context_documents",
		"context_chunks",
		"context_budget_samples",
		"context_budget_tuning_profiles",
		"model_catalog_entries",
		"model_catalog_tombstones",
		"model_routing_policies",
		"chat_turn_items",
		"chat_persona_policies",
		"schema_migrations",
	}
	for _, table := range tables {
		if !objectExists(t, db, "table", table) {
			t.Fatalf("expected table %s to exist", table)
		}
	}

	indexes := []string{
		"idx_comm_events_thread_occurred",
		"idx_comm_event_addresses_event_role",
		"idx_comm_threads_workspace_channel_connector_updated",
		"idx_comm_events_workspace_connector_occurred",
		"idx_comm_call_sessions_workspace_connector_updated",
		"uq_comm_ingest_receipts_workspace_source_event",
		"uq_comm_ingest_cursors_workspace_source_scope",
		"uq_automation_source_subscriptions_workspace_source_scope",
		"idx_automation_source_subscriptions_workspace_source",
		"uq_comm_webhook_receipts_workspace_provider_event",
		"uq_comm_provider_messages_workspace_provider_message",
		"idx_comm_provider_messages_event_id",
		"uq_comm_call_sessions_workspace_provider_call",
		"idx_comm_call_sessions_workspace_status_updated",
		"idx_tasks_workspace_state_created",
		"idx_task_runs_task_state_started",
		"idx_task_runs_claim_queued_state_created_at",
		"idx_task_runs_claim_running_state_updated_created",
		"idx_audit_log_entries_correlation_created_id",
		"idx_task_steps_run_index_status",
		"idx_memory_items_owner_scope_key_status",
		"idx_runtime_plugins_workspace_kind_status",
		"idx_runtime_plugin_processes_workspace_state_updated",
		"idx_context_budget_samples_workspace_class_created",
		"idx_model_catalog_workspace_provider_enabled",
		"idx_model_tombstones_workspace_provider",
		"idx_model_routing_workspace_task_class",
		"uq_trigger_fires_workspace_trigger_source",
		"uq_delivery_attempts_endpoint_idempotency",
		"uq_channel_connector_bindings_workspace_channel_connector",
		"uq_channel_connector_bindings_workspace_channel_priority",
		"idx_channel_connector_bindings_workspace_channel_enabled_priority",
		"uq_chat_turn_items_turn_index",
		"idx_chat_turn_items_workspace_channel_thread_created",
		"idx_chat_turn_items_workspace_correlation_created",
		"idx_chat_persona_policies_workspace_scope",
	}
	for _, index := range indexes {
		if !objectExists(t, db, "index", index) {
			t.Fatalf("expected index %s to exist", index)
		}
	}

	triggers := []string{
		"trg_seed_channel_connector_bindings_after_workspace_insert",
	}
	for _, trigger := range triggers {
		if !objectExists(t, db, "trigger", trigger) {
			t.Fatalf("expected trigger %s to exist", trigger)
		}
	}
}

func TestRuntimePluginProcessRequiresRegisteredPlugin(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('daemon', 'daemon', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("seed daemon workspace: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO runtime_plugin_processes(
			workspace_id,
			plugin_id,
			state,
			process_id,
			restart_count,
			last_error,
			last_heartbeat_at,
			last_transition_at,
			event_type,
			metadata_json,
			created_at,
			updated_at
		) VALUES (
			'daemon',
			'missing.plugin',
			'running',
			123,
			0,
			NULL,
			NULL,
			?,
			'PLUGIN_HANDSHAKE_ACCEPTED',
			'{}',
			?,
			?
		)
	`, now, now, now); err == nil {
		t.Fatalf("expected runtime_plugin_processes insert without runtime_plugins row to fail")
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := Apply(ctx, db); err != nil {
		t.Fatalf("first apply migrations: %v", err)
	}

	appliedSecond, err := Apply(ctx, db)
	if err != nil {
		t.Fatalf("second apply migrations: %v", err)
	}
	if len(appliedSecond) != 0 {
		t.Fatalf("expected no pending migrations on second run, got %d", len(appliedSecond))
	}
}

func applyMigrationsThrough(t *testing.T, db *sql.DB, throughVersion string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		t.Fatalf("ensure schema_migrations table: %v", err)
	}

	migrationsToApply, err := listMigrations()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}

	for _, migration := range migrationsToApply {
		version := versionFromName(migration.Name)
		if version > throughVersion {
			break
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin migration %s: %v", migration.Name, err)
		}

		if _, err := tx.ExecContext(ctx, migration.Body); err != nil {
			_ = tx.Rollback()
			t.Fatalf("apply migration %s: %v", migration.Name, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)",
			version,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			t.Fatalf("record migration %s: %v", migration.Name, err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("commit migration %s: %v", migration.Name, err)
		}
	}
}

func TestChannelConnectorBindingsMigrationPreservesLegacyData(t *testing.T) {
	db := openTestDB(t)
	applyMigrationsThrough(t, db, "0009")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws1', 'ws1', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	const legacyConfigJSON = `{"account_sid_secret_name":"TWILIO_ACCOUNT_SID","auth_token_secret_name":"TWILIO_AUTH_TOKEN","sms_number":"+15550001111","voice_number":"+15550002222","endpoint":"https://api.twilio.com"}`
	if _, err := db.ExecContext(ctx, `
		INSERT INTO channel_connectors(
			id, workspace_id, connector_type, status, config_json, created_at, updated_at
		) VALUES (
			'channel:twilio:ws1', 'ws1', 'channel.twilio', 'ACTIVE', ?, ?, ?
		)
	`, legacyConfigJSON, now, now); err != nil {
		t.Fatalf("seed legacy twilio connector config: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		VALUES
			('thread-imessage', 'ws1', 'imessage_sms', 'legacy:imessage:1', 'Legacy iMessage', ?, ?),
			('thread-twilio', 'ws1', 'twilio_sms', 'legacy:twilio:1', 'Legacy Twilio', ?, ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed legacy comm threads: %v", err)
	}

	applied, err := Apply(ctx, db)
	if err != nil {
		t.Fatalf("apply remaining migrations: %v", err)
	}
	if len(applied) != 7 ||
		applied[0] != "0010_channel_connector_bindings.sql" ||
		applied[1] != "0011_comm_connector_attribution.sql" ||
		applied[2] != "0012_imessage_connector_id_rename.sql" ||
		applied[3] != "0013_unified_turn_ledger_persona.sql" ||
		applied[4] != "0014_task_step_input_payload.sql" ||
		applied[5] != "0015_task_run_claim_indexes.sql" ||
		applied[6] != "0016_audit_log_correlation_lookup_index.sql" {
		t.Fatalf("expected 0010+0011+0012+0013+0014+0015+0016 migrations to apply, got %+v", applied)
	}

	var configJSON string
	if err := db.QueryRowContext(ctx, `
		SELECT config_json
		FROM channel_connectors
		WHERE id = 'channel:twilio:ws1'
	`).Scan(&configJSON); err != nil {
		t.Fatalf("query legacy twilio connector config: %v", err)
	}
	if configJSON != legacyConfigJSON {
		t.Fatalf("legacy twilio config changed unexpectedly: %s", configJSON)
	}

	var legacyThreadCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM comm_threads
		WHERE workspace_id = 'ws1'
		  AND channel IN ('imessage_sms', 'twilio_sms')
	`).Scan(&legacyThreadCount); err != nil {
		t.Fatalf("count legacy comm threads: %v", err)
	}
	if legacyThreadCount != 2 {
		t.Fatalf("expected legacy comm threads to remain intact, got %d", legacyThreadCount)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT channel_id, connector_id, enabled, priority
		FROM channel_connector_bindings
		WHERE workspace_id = 'ws1'
		ORDER BY channel_id, priority
	`)
	if err != nil {
		t.Fatalf("query channel connector bindings: %v", err)
	}
	defer rows.Close()

	type binding struct {
		channelID   string
		connectorID string
		enabled     int
		priority    int
	}
	seen := map[string]binding{}
	for rows.Next() {
		var item binding
		if err := rows.Scan(&item.channelID, &item.connectorID, &item.enabled, &item.priority); err != nil {
			t.Fatalf("scan channel connector binding: %v", err)
		}
		seen[item.channelID+"|"+item.connectorID] = item
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate channel connector bindings: %v", err)
	}

	expected := map[string]binding{
		"app|builtin.app":  {channelID: "app", connectorID: "builtin.app", enabled: 1, priority: 1},
		"message|imessage": {channelID: "message", connectorID: "imessage", enabled: 1, priority: 1},
		"message|twilio":   {channelID: "message", connectorID: "twilio", enabled: 1, priority: 2},
		"voice|twilio":     {channelID: "voice", connectorID: "twilio", enabled: 1, priority: 1},
	}
	if len(seen) != len(expected) {
		t.Fatalf("expected %d seeded bindings, got %d (%+v)", len(expected), len(seen), seen)
	}
	for key, want := range expected {
		got, ok := seen[key]
		if !ok {
			t.Fatalf("missing seeded binding %s", key)
		}
		if got.enabled != want.enabled || got.priority != want.priority {
			t.Fatalf("binding %s mismatch got=%+v want=%+v", key, got, want)
		}
	}
}

func TestChannelConnectorBindingsDefaultsSeedForNewWorkspace(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws-new', 'ws-new', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	var seededCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM channel_connector_bindings
		WHERE workspace_id = 'ws-new'
	`).Scan(&seededCount); err != nil {
		t.Fatalf("count seeded bindings: %v", err)
	}
	if seededCount != 4 {
		t.Fatalf("expected 4 seeded channel connector bindings, got %d", seededCount)
	}
}

func TestCommConnectorAttributionMigrationBackfillsLegacyRows(t *testing.T) {
	db := openTestDB(t)
	applyMigrationsThrough(t, db, "0010")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws2', 'ws2', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		VALUES
			('thread-legacy-message', 'ws2', 'message', 'legacy:message:1', 'Legacy Message', ?, ?),
			('thread-legacy-voice', 'ws2', 'twilio_voice', 'legacy:voice:1', 'Legacy Voice', ?, ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed legacy threads: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		VALUES
			('event-legacy-message', 'ws2', 'thread-legacy-message', 'MESSAGE', 'INBOUND', 0, ?, 'hello', ?),
			('event-legacy-voice', 'ws2', 'thread-legacy-voice', 'VOICE_TRANSCRIPT', 'INBOUND', 0, ?, 'voice hello', ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed legacy events: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_provider_messages(
			id, workspace_id, event_id, provider, provider_message_id, provider_account_id, channel, direction,
			from_address, to_address, status, payload_json, created_at, updated_at
		) VALUES (
			'provider-message-legacy',
			'ws2',
			'event-legacy-message',
			'twilio',
			'SM-LEGACY-1',
			NULL,
			'sms',
			'INBOUND',
			'+15550001111',
			'+15550002222',
			'received',
			NULL,
			?,
			?
		)
	`, now, now); err != nil {
		t.Fatalf("seed legacy provider message: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_call_sessions(
			id, workspace_id, provider, provider_call_id, provider_account_id, thread_id, direction,
			from_address, to_address, status, started_at, ended_at, last_error, created_at, updated_at
		) VALUES (
			'call-legacy-1',
			'ws2',
			'twilio',
			'CA-LEGACY-1',
			NULL,
			'thread-legacy-voice',
			'inbound',
			'+15550001111',
			'+15550002222',
			'in_progress',
			?,
			NULL,
			NULL,
			?,
			?
		)
	`, now, now, now); err != nil {
		t.Fatalf("seed legacy call session: %v", err)
	}

	applied, err := Apply(ctx, db)
	if err != nil {
		t.Fatalf("apply remaining migrations: %v", err)
	}
	if len(applied) != 6 ||
		applied[0] != "0011_comm_connector_attribution.sql" ||
		applied[1] != "0012_imessage_connector_id_rename.sql" ||
		applied[2] != "0013_unified_turn_ledger_persona.sql" ||
		applied[3] != "0014_task_step_input_payload.sql" ||
		applied[4] != "0015_task_run_claim_indexes.sql" ||
		applied[5] != "0016_audit_log_correlation_lookup_index.sql" {
		t.Fatalf("expected 0011+0012+0013+0014+0015+0016 to apply, got %+v", applied)
	}

	var (
		threadMessageConnector string
		threadVoiceConnector   string
		eventMessageConnector  string
		eventVoiceConnector    string
		callConnector          string
	)
	if err := db.QueryRowContext(ctx, `
		SELECT connector_id
		FROM comm_threads
		WHERE id = 'thread-legacy-message'
	`).Scan(&threadMessageConnector); err != nil {
		t.Fatalf("query message thread connector: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT connector_id
		FROM comm_threads
		WHERE id = 'thread-legacy-voice'
	`).Scan(&threadVoiceConnector); err != nil {
		t.Fatalf("query voice thread connector: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT connector_id
		FROM comm_events
		WHERE id = 'event-legacy-message'
	`).Scan(&eventMessageConnector); err != nil {
		t.Fatalf("query message event connector: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT connector_id
		FROM comm_events
		WHERE id = 'event-legacy-voice'
	`).Scan(&eventVoiceConnector); err != nil {
		t.Fatalf("query voice event connector: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT connector_id
		FROM comm_call_sessions
		WHERE id = 'call-legacy-1'
	`).Scan(&callConnector); err != nil {
		t.Fatalf("query call session connector: %v", err)
	}

	if threadMessageConnector != "twilio" {
		t.Fatalf("expected message thread connector=twilio via provider backfill, got %s", threadMessageConnector)
	}
	if threadVoiceConnector != "twilio" {
		t.Fatalf("expected voice thread connector=twilio, got %s", threadVoiceConnector)
	}
	if eventMessageConnector != "twilio" {
		t.Fatalf("expected message event connector=twilio, got %s", eventMessageConnector)
	}
	if eventVoiceConnector != "twilio" {
		t.Fatalf("expected voice event connector=twilio, got %s", eventVoiceConnector)
	}
	if callConnector != "twilio" {
		t.Fatalf("expected call session connector=twilio, got %s", callConnector)
	}
}

func TestImessageConnectorRenameMigrationRewritesLegacyConnectorIDs(t *testing.T) {
	db := openTestDB(t)
	applyMigrationsThrough(t, db, "0011")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws3', 'ws3', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE channel_connector_bindings
		SET connector_id = 'apple.messages',
		    id = 'ccb.ws3.message.apple_messages'
		WHERE workspace_id = 'ws3'
		  AND channel_id = 'message'
		  AND connector_id = 'imessage'
	`); err != nil {
		t.Fatalf("seed legacy channel connector binding: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at, connector_id)
		VALUES ('thread-imessage-legacy', 'ws3', 'message', 'legacy:imessage:2', 'Legacy iMessage', ?, ?, 'apple.messages')
	`, now, now); err != nil {
		t.Fatalf("seed legacy comm thread: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at, connector_id)
		VALUES ('event-imessage-legacy', 'ws3', 'thread-imessage-legacy', 'MESSAGE', 'INBOUND', 0, ?, 'hello', ?, 'apple.messages')
	`, now, now); err != nil {
		t.Fatalf("seed legacy comm event: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO comm_call_sessions(
			id, workspace_id, provider, provider_call_id, provider_account_id, thread_id, direction,
			from_address, to_address, status, started_at, ended_at, last_error, created_at, updated_at, connector_id
		) VALUES (
			'call-imessage-legacy', 'ws3', 'local', 'CALL-LEGACY-1', NULL, 'thread-imessage-legacy', 'inbound',
			'+15550001111', '+15550002222', 'in_progress', ?, NULL, NULL, ?, ?, 'apple.messages'
		)
	`, now, now, now); err != nil {
		t.Fatalf("seed legacy call session: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO delivery_attempts(
			id, workspace_id, step_id, event_id, destination_endpoint, idempotency_key, channel, provider_receipt,
			status, error, attempted_at
		) VALUES (
			'attempt-imessage-legacy', 'ws3', NULL, NULL, '+15550001111', 'idemp-imessage-legacy',
			'apple.messages', NULL, 'sent', NULL, ?
		)
	`, now); err != nil {
		t.Fatalf("seed legacy delivery attempt: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO channel_delivery_policies(
			id, workspace_id, channel, endpoint_pattern, policy_json, is_default, created_at, updated_at
		) VALUES (
			'policy-imessage-legacy', 'ws3', 'message', NULL,
			'{"primary_channel":"apple.messages","fallback_channels":["twilio"]}', 1, ?, ?
		)
	`, now, now); err != nil {
		t.Fatalf("seed legacy channel delivery policy: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO channel_connectors(
			id, workspace_id, connector_type, status, config_json, created_at, updated_at
		) VALUES (
			'connector-imessage-legacy', 'ws3', 'apple.messages', 'ACTIVE', '{}', ?, ?
		)
	`, now, now); err != nil {
		t.Fatalf("seed legacy channel connector: %v", err)
	}

	applied, err := Apply(ctx, db)
	if err != nil {
		t.Fatalf("apply remaining migrations: %v", err)
	}
	if len(applied) != 5 ||
		applied[0] != "0012_imessage_connector_id_rename.sql" ||
		applied[1] != "0013_unified_turn_ledger_persona.sql" ||
		applied[2] != "0014_task_step_input_payload.sql" ||
		applied[3] != "0015_task_run_claim_indexes.sql" ||
		applied[4] != "0016_audit_log_correlation_lookup_index.sql" {
		t.Fatalf("expected 0012+0013+0014+0015+0016 to apply, got %+v", applied)
	}

	assertConnectorID := func(query string, args ...any) {
		t.Helper()
		var connectorID string
		if err := db.QueryRowContext(ctx, query, args...).Scan(&connectorID); err != nil {
			t.Fatalf("query connector id: %v", err)
		}
		if connectorID != "imessage" {
			t.Fatalf("expected connector id imessage, got %q", connectorID)
		}
	}
	assertConnectorID(`SELECT connector_id FROM channel_connector_bindings WHERE workspace_id = 'ws3' AND channel_id = 'message' AND connector_id = 'imessage'`)
	assertConnectorID(`SELECT connector_id FROM comm_threads WHERE id = 'thread-imessage-legacy'`)
	assertConnectorID(`SELECT connector_id FROM comm_events WHERE id = 'event-imessage-legacy'`)
	assertConnectorID(`SELECT connector_id FROM comm_call_sessions WHERE id = 'call-imessage-legacy'`)

	var deliveryChannel string
	if err := db.QueryRowContext(ctx, `
		SELECT channel
		FROM delivery_attempts
		WHERE id = 'attempt-imessage-legacy'
	`).Scan(&deliveryChannel); err != nil {
		t.Fatalf("query delivery attempt channel: %v", err)
	}
	if deliveryChannel != "imessage" {
		t.Fatalf("expected delivery attempt channel imessage, got %q", deliveryChannel)
	}

	var policyJSON string
	if err := db.QueryRowContext(ctx, `
		SELECT policy_json
		FROM channel_delivery_policies
		WHERE id = 'policy-imessage-legacy'
	`).Scan(&policyJSON); err != nil {
		t.Fatalf("query delivery policy: %v", err)
	}
	if policyJSON != `{"primary_channel":"imessage","fallback_channels":["twilio"]}` {
		t.Fatalf("expected policy json to rewrite connector id, got %s", policyJSON)
	}

	var connectorType string
	if err := db.QueryRowContext(ctx, `
		SELECT connector_type
		FROM channel_connectors
		WHERE id = 'connector-imessage-legacy'
	`).Scan(&connectorType); err != nil {
		t.Fatalf("query channel connector type: %v", err)
	}
	if connectorType != "imessage" {
		t.Fatalf("expected connector_type imessage, got %q", connectorType)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws4', 'ws4', 'ACTIVE', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("seed workspace after rename migration: %v", err)
	}
	var seededImessageBindings int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM channel_connector_bindings
		WHERE workspace_id = 'ws4'
		  AND channel_id = 'message'
		  AND connector_id = 'imessage'
	`).Scan(&seededImessageBindings); err != nil {
		t.Fatalf("count ws4 imessage bindings: %v", err)
	}
	if seededImessageBindings != 1 {
		t.Fatalf("expected ws4 to seed one imessage binding, got %d", seededImessageBindings)
	}
}
