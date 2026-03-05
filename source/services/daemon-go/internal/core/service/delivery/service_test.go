package delivery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	repodelivery "personalagent/runtime/internal/core/repository/delivery"
	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

type scriptedResponse struct {
	receipt string
	err     error
}

type scriptedSender struct {
	mu        sync.Mutex
	responses map[string][]scriptedResponse
	calls     []string
}

func newScriptedSender() *scriptedSender {
	return &scriptedSender{responses: map[string][]scriptedResponse{}, calls: []string{}}
}

func (s *scriptedSender) Send(_ context.Context, channel string, _ types.DeliveryRequest, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, channel)

	queue := s.responses[channel]
	if len(queue) == 0 {
		return "", fmt.Errorf("no scripted response for channel %s", channel)
	}
	response := queue[0]
	s.responses[channel] = queue[1:]
	if response.err != nil {
		return "", response.err
	}
	return response.receipt, nil
}

func (s *scriptedSender) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func setupDeliveryDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "delivery.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := migrator.Apply(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO workspaces(id, name, status, created_at, updated_at)
		 VALUES ('ws_delivery', 'Delivery Workspace', 'ACTIVE', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	return db
}

func countDeliveryAttempts(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_attempts`).Scan(&count); err != nil {
		t.Fatalf("count delivery attempts: %v", err)
	}
	return count
}

func TestDeliverRetriesMessagesThenFallsBackToTwilio(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	sender.responses["imessage"] = []scriptedResponse{{err: errors.New("messages attempt 1 failed")}, {err: errors.New("messages attempt 2 failed")}}
	sender.responses["twilio"] = []scriptedResponse{{receipt: "twilio-receipt-1"}}

	deliveryService := NewService(store, sender, Options{})
	result, err := deliveryService.Deliver(context.Background(), types.DeliveryRequest{
		WorkspaceID:         "ws_delivery",
		OperationID:         "op-001",
		SourceChannel:       "message",
		DestinationEndpoint: "+15551234567",
		MessageBody:         "hello",
	})
	if err != nil {
		t.Fatalf("deliver message with fallback: %v", err)
	}
	if !result.Delivered {
		t.Fatalf("expected delivery to succeed")
	}
	if result.Channel != "twilio" {
		t.Fatalf("expected twilio fallback route, got %s", result.Channel)
	}
	if sender.CallCount() != 3 {
		t.Fatalf("expected 3 send attempts, got %d", sender.CallCount())
	}
	if attempts := countDeliveryAttempts(t, db); attempts != 3 {
		t.Fatalf("expected 3 delivery attempt rows, got %d", attempts)
	}
}

func TestDeliverIsIdempotentForReplay(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	sender.responses["imessage"] = []scriptedResponse{{receipt: "messages-receipt-1"}}

	deliveryService := NewService(store, sender, Options{})
	request := types.DeliveryRequest{
		WorkspaceID:         "ws_delivery",
		OperationID:         "op-idempotent",
		SourceChannel:       "message",
		DestinationEndpoint: "+15550002222",
		MessageBody:         "idempotent message",
	}

	first, err := deliveryService.Deliver(context.Background(), request)
	if err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if !first.Delivered {
		t.Fatalf("expected first delivery to succeed")
	}

	second, err := deliveryService.Deliver(context.Background(), request)
	if err != nil {
		t.Fatalf("second delivery replay: %v", err)
	}
	if !second.Delivered {
		t.Fatalf("expected replay delivery to report success")
	}
	if !second.IdempotentReplay {
		t.Fatalf("expected replay to be idempotent hit")
	}
	if sender.CallCount() != 1 {
		t.Fatalf("expected one sender call total, got %d", sender.CallCount())
	}
	if attempts := countDeliveryAttempts(t, db); attempts != 1 {
		t.Fatalf("expected one delivery attempt row, got %d", attempts)
	}
}

func TestDeliverUsesConfiguredPolicyOverride(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	sender.responses["twilio"] = []scriptedResponse{{receipt: "twilio-policy-receipt"}}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO channel_delivery_policies(
			id, workspace_id, channel, endpoint_pattern, policy_json, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"policy_sms_override",
		"ws_delivery",
		"message",
		nil,
		`{"primary_channel":"twilio","retry_count":0,"fallback_channels":[]}`,
		1,
		now,
		now,
	); err != nil {
		t.Fatalf("insert policy override: %v", err)
	}

	deliveryService := NewService(store, sender, Options{})
	result, err := deliveryService.Deliver(context.Background(), types.DeliveryRequest{
		WorkspaceID:         "ws_delivery",
		OperationID:         "op-policy",
		SourceChannel:       "message",
		DestinationEndpoint: "+15553334444",
		MessageBody:         "policy override",
	})
	if err != nil {
		t.Fatalf("deliver with policy override: %v", err)
	}
	if result.Channel != "twilio" {
		t.Fatalf("expected override to route via twilio, got %s", result.Channel)
	}
	if sender.CallCount() != 1 {
		t.Fatalf("expected one send call under policy override, got %d", sender.CallCount())
	}
}

func TestDeliverResolvesCanonicalImessageSourceAndMessagePolicyLookup(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	sender.responses["builtin.app"] = []scriptedResponse{{receipt: "app-policy-receipt"}}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO channel_delivery_policies(
			id, workspace_id, channel, endpoint_pattern, policy_json, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"policy_message_override",
		"ws_delivery",
		"message",
		nil,
		`{"primary_channel":"builtin.app","retry_count":0,"fallback_channels":[]}`,
		1,
		now,
		now,
	); err != nil {
		t.Fatalf("insert message policy override: %v", err)
	}

	deliveryService := NewService(store, sender, Options{})
	result, err := deliveryService.Deliver(context.Background(), types.DeliveryRequest{
		WorkspaceID:         "ws_delivery",
		OperationID:         "op-canonical-source",
		SourceChannel:       "imessage",
		DestinationEndpoint: "+15550003333",
		MessageBody:         "canonical source candidate lookup",
	})
	if err != nil {
		t.Fatalf("deliver with canonical source lookup: %v", err)
	}
	if result.Channel != "builtin.app" {
		t.Fatalf("expected canonical imessage source to route via message policy override, got %s", result.Channel)
	}
	if sender.CallCount() != 1 {
		t.Fatalf("expected one send call for canonical source lookup, got %d", sender.CallCount())
	}
}

func TestDeliverRejectsNonCanonicalSourceChannelAliases(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	deliveryService := NewService(store, sender, Options{})

	for _, alias := range []string{"twilio", "builtin.app", "text"} {
		_, err := deliveryService.Deliver(context.Background(), types.DeliveryRequest{
			WorkspaceID:         "ws_delivery",
			OperationID:         "op-alias-" + strings.ReplaceAll(alias, ".", "-"),
			SourceChannel:       alias,
			DestinationEndpoint: "+15550009999",
			MessageBody:         "alias should fail",
		})
		if err == nil {
			t.Fatalf("expected source channel alias %q to fail validation", alias)
		}
		if !strings.Contains(err.Error(), `unsupported source channel "`) || !strings.Contains(err.Error(), "allowed: message|imessage|sms|voice|app") {
			t.Fatalf("expected deterministic canonical source-channel validation error for %q, got %v", alias, err)
		}
	}
	if sender.CallCount() != 0 {
		t.Fatalf("expected no sender calls for rejected aliases, got %d", sender.CallCount())
	}
}

func TestDeliverResolvesPrimaryRouteFromChannelConnectorBindings(t *testing.T) {
	db := setupDeliveryDB(t)
	store := repodelivery.NewSQLiteDeliveryStore(db)
	sender := newScriptedSender()
	sender.responses["twilio"] = []scriptedResponse{{receipt: "twilio-binding-receipt"}}

	// Reorder seeded message bindings so twilio is primary and messages is secondary.
	if _, err := db.Exec(`
		UPDATE channel_connector_bindings
		SET priority = 99
		WHERE workspace_id = 'ws_delivery'
		  AND channel_id = 'message'
		  AND connector_id = 'twilio'
	`); err != nil {
		t.Fatalf("set temporary twilio binding priority: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE channel_connector_bindings
		SET priority = 2
		WHERE workspace_id = 'ws_delivery'
		  AND channel_id = 'message'
		  AND connector_id = 'imessage'
	`); err != nil {
		t.Fatalf("set messages binding priority: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE channel_connector_bindings
		SET priority = 1
		WHERE workspace_id = 'ws_delivery'
		  AND channel_id = 'message'
		  AND connector_id = 'twilio'
	`); err != nil {
		t.Fatalf("set twilio binding priority: %v", err)
	}

	deliveryService := NewService(store, sender, Options{})
	result, err := deliveryService.Deliver(context.Background(), types.DeliveryRequest{
		WorkspaceID:         "ws_delivery",
		OperationID:         "op-binding-priority",
		SourceChannel:       "message",
		DestinationEndpoint: "+15554445555",
		MessageBody:         "binding driven route",
	})
	if err != nil {
		t.Fatalf("deliver with binding-priority route: %v", err)
	}
	if result.Channel != "twilio" {
		t.Fatalf("expected binding-priority route twilio, got %s", result.Channel)
	}
	if sender.CallCount() != 1 {
		t.Fatalf("expected one send call for binding-priority route, got %d", sender.CallCount())
	}
}
