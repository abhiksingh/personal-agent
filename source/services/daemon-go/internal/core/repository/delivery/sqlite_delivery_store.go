package delivery

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
)

type SQLiteDeliveryStore struct {
	db *sql.DB
}

func NewSQLiteDeliveryStore(db *sql.DB) *SQLiteDeliveryStore {
	return &SQLiteDeliveryStore{db: db}
}

func (s *SQLiteDeliveryStore) ResolveDeliveryPolicy(
	ctx context.Context,
	workspaceID string,
	sourceChannel string,
	destinationEndpoint string,
) (types.ChannelDeliveryPolicy, bool, error) {
	normalizedSource, err := normalizeDeliveryPolicySourceChannel(sourceChannel)
	if err != nil {
		return types.ChannelDeliveryPolicy{}, false, err
	}

	for _, candidate := range deliveryPolicyLookupCandidates(normalizedSource) {
		policy, found, err := s.resolveDeliveryPolicyForChannel(ctx, workspaceID, candidate, destinationEndpoint)
		if err != nil {
			return types.ChannelDeliveryPolicy{}, false, err
		}
		if found {
			return policy, true, nil
		}
	}

	boundPolicy, found, err := s.resolveBoundChannelPolicy(ctx, workspaceID, normalizedSource)
	if err != nil {
		return types.ChannelDeliveryPolicy{}, false, err
	}
	if found {
		return boundPolicy, true, nil
	}

	return types.ChannelDeliveryPolicy{}, false, nil
}

func (s *SQLiteDeliveryStore) resolveDeliveryPolicyForChannel(
	ctx context.Context,
	workspaceID string,
	sourceChannel string,
	destinationEndpoint string,
) (types.ChannelDeliveryPolicy, bool, error) {
	var policyJSON string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT policy_json
		 FROM channel_delivery_policies
		 WHERE workspace_id = ?
		   AND channel = ?
		   AND (
			 endpoint_pattern IS NULL
			 OR endpoint_pattern = ''
			 OR ? LIKE endpoint_pattern
		   )
		 ORDER BY is_default DESC, updated_at DESC, created_at DESC
		 LIMIT 1`,
		workspaceID,
		normalizeChannel(sourceChannel),
		destinationEndpoint,
	).Scan(&policyJSON)
	if err == sql.ErrNoRows {
		return types.ChannelDeliveryPolicy{}, false, nil
	}
	if err != nil {
		return types.ChannelDeliveryPolicy{}, false, fmt.Errorf("resolve delivery policy: %w", err)
	}

	policy := types.ChannelDeliveryPolicy{}
	if unmarshalErr := json.Unmarshal([]byte(policyJSON), &policy); unmarshalErr != nil {
		return types.ChannelDeliveryPolicy{}, false, fmt.Errorf("parse delivery policy json: %w", unmarshalErr)
	}
	policy = normalizePolicy(policy)
	return policy, true, nil
}

func (s *SQLiteDeliveryStore) resolveBoundChannelPolicy(
	ctx context.Context,
	workspaceID string,
	sourceChannel string,
) (types.ChannelDeliveryPolicy, bool, error) {
	logicalChannel := canonicalLogicalChannel(sourceChannel)
	if logicalChannel == "" {
		return types.ChannelDeliveryPolicy{}, false, nil
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT connector_id
		 FROM channel_connector_bindings
		 WHERE workspace_id = ?
		   AND channel_id = ?
		   AND enabled = 1
		 ORDER BY priority ASC, connector_id ASC`,
		workspaceID,
		logicalChannel,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table: channel_connector_bindings") {
			return types.ChannelDeliveryPolicy{}, false, nil
		}
		return types.ChannelDeliveryPolicy{}, false, fmt.Errorf("resolve bound channel policy: %w", err)
	}
	defer rows.Close()

	routes := make([]string, 0)
	for rows.Next() {
		var connectorID string
		if err := rows.Scan(&connectorID); err != nil {
			return types.ChannelDeliveryPolicy{}, false, fmt.Errorf("scan channel connector binding: %w", err)
		}
		normalized := normalizeChannel(connectorID)
		if normalized == "" {
			continue
		}
		routes = append(routes, normalized)
	}
	if err := rows.Err(); err != nil {
		return types.ChannelDeliveryPolicy{}, false, fmt.Errorf("iterate channel connector bindings: %w", err)
	}
	if len(routes) == 0 {
		return types.ChannelDeliveryPolicy{}, false, nil
	}

	if logicalChannel == "message" {
		switch normalizeChannel(sourceChannel) {
		case "imessage":
			routes = prioritizeConnectorRoute(routes, "imessage")
		}
	}

	normalizedRoutes := make([]string, 0, len(routes))
	for _, route := range routes {
		normalizedRoute := normalizeBoundRouteChannel(logicalChannel, route)
		if normalizedRoute == "" {
			continue
		}
		normalizedRoutes = append(normalizedRoutes, normalizedRoute)
	}
	if len(normalizedRoutes) == 0 {
		return types.ChannelDeliveryPolicy{}, false, nil
	}

	retryCount := 0
	if logicalChannel == "message" && normalizedRoutes[0] == "imessage" {
		retryCount = 1
	}

	return normalizePolicy(types.ChannelDeliveryPolicy{
		PrimaryChannel:   normalizedRoutes[0],
		RetryCount:       retryCount,
		FallbackChannels: append([]string{}, normalizedRoutes[1:]...),
	}), true, nil
}

func (s *SQLiteDeliveryStore) ReserveDeliveryAttempt(ctx context.Context, attempt types.DeliveryAttemptRecord) (types.DeliveryAttemptRecord, bool, error) {
	attemptID := strings.TrimSpace(attempt.AttemptID)
	if attemptID == "" {
		randomValue, err := randomID()
		if err != nil {
			return types.DeliveryAttemptRecord{}, false, err
		}
		attemptID = randomValue
	}
	attemptedAt := attempt.AttemptedAt.UTC()
	if attemptedAt.IsZero() {
		attemptedAt = time.Now().UTC()
	}

	attempt.AttemptID = attemptID
	attempt.AttemptedAt = attemptedAt
	attempt.Channel = normalizeChannel(attempt.Channel)
	if attempt.Status == "" {
		attempt.Status = types.DeliveryAttemptPending
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO delivery_attempts(
			id,
			workspace_id,
			step_id,
			event_id,
			destination_endpoint,
			idempotency_key,
			channel,
			provider_receipt,
			status,
			error,
			attempted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attempt.AttemptID,
		attempt.WorkspaceID,
		nullIfEmpty(attempt.StepID),
		nullIfEmpty(attempt.EventID),
		attempt.DestinationEndpoint,
		attempt.IdempotencyKey,
		attempt.Channel,
		nullIfEmpty(attempt.ProviderReceipt),
		string(attempt.Status),
		nullIfEmpty(attempt.ErrorText),
		attempt.AttemptedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, getErr := s.getAttemptByIdempotency(ctx, attempt.DestinationEndpoint, attempt.IdempotencyKey)
			if getErr != nil {
				return types.DeliveryAttemptRecord{}, false, getErr
			}
			return existing, false, nil
		}
		return types.DeliveryAttemptRecord{}, false, fmt.Errorf("insert delivery attempt: %w", err)
	}
	return attempt, true, nil
}

func (s *SQLiteDeliveryStore) MarkDeliveryAttemptResult(
	ctx context.Context,
	attemptID string,
	status types.DeliveryAttemptStatus,
	providerReceipt string,
	errorText string,
	attemptedAt time.Time,
) error {
	attemptTime := attemptedAt.UTC()
	if attemptTime.IsZero() {
		attemptTime = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`UPDATE delivery_attempts
		 SET status = ?, provider_receipt = ?, error = ?, attempted_at = ?
		 WHERE id = ?`,
		string(status),
		nullIfEmpty(providerReceipt),
		nullIfEmpty(errorText),
		attemptTime.Format(time.RFC3339Nano),
		attemptID,
	)
	if err != nil {
		return fmt.Errorf("mark delivery attempt result: %w", err)
	}
	return nil
}

func (s *SQLiteDeliveryStore) getAttemptByIdempotency(ctx context.Context, destinationEndpoint string, idempotencyKey string) (types.DeliveryAttemptRecord, error) {
	var attempt types.DeliveryAttemptRecord
	var status string
	var attemptedAt string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			id,
			workspace_id,
			COALESCE(step_id, ''),
			COALESCE(event_id, ''),
			destination_endpoint,
			idempotency_key,
			channel,
			COALESCE(provider_receipt, ''),
			status,
			COALESCE(error, ''),
			attempted_at
		 FROM delivery_attempts
		 WHERE destination_endpoint = ?
		   AND idempotency_key = ?`,
		destinationEndpoint,
		idempotencyKey,
	).Scan(
		&attempt.AttemptID,
		&attempt.WorkspaceID,
		&attempt.StepID,
		&attempt.EventID,
		&attempt.DestinationEndpoint,
		&attempt.IdempotencyKey,
		&attempt.Channel,
		&attempt.ProviderReceipt,
		&status,
		&attempt.ErrorText,
		&attemptedAt,
	)
	if err != nil {
		return types.DeliveryAttemptRecord{}, fmt.Errorf("get attempt by idempotency: %w", err)
	}
	attempt.Status = types.DeliveryAttemptStatus(status)
	attempt.AttemptedAt, _ = time.Parse(time.RFC3339Nano, attemptedAt)
	return attempt, nil
}

func normalizePolicy(policy types.ChannelDeliveryPolicy) types.ChannelDeliveryPolicy {
	policy.PrimaryChannel = normalizeDeliveryRouteChannel(policy.PrimaryChannel)
	if policy.RetryCount < 0 {
		policy.RetryCount = 0
	}

	fallback := make([]string, 0, len(policy.FallbackChannels))
	seen := map[string]struct{}{}
	if policy.PrimaryChannel != "" {
		seen[policy.PrimaryChannel] = struct{}{}
	}
	for _, channel := range policy.FallbackChannels {
		normalized := normalizeDeliveryRouteChannel(channel)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		fallback = append(fallback, normalized)
	}
	policy.FallbackChannels = fallback
	return policy
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func deliveryPolicyLookupCandidates(sourceChannel string) []string {
	source := normalizeChannel(sourceChannel)
	if source == "" {
		source = "message"
	}
	candidates := []string{source}
	if source == "imessage" {
		candidates = append(candidates, "message")
	}
	return candidates
}

func canonicalLogicalChannel(source string) string {
	switch normalizeChannel(source) {
	case "", "message", "imessage":
		return "message"
	case "sms":
		return "sms"
	case "voice":
		return "voice"
	case "app":
		return "app"
	default:
		return ""
	}
}

func normalizeDeliveryPolicySourceChannel(sourceChannel string) (string, error) {
	normalized := normalizeChannel(sourceChannel)
	if normalized == "" {
		return "message", nil
	}
	switch normalized {
	case "message", "imessage", "sms", "voice", "app":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported source channel %q (allowed: message|imessage|sms|voice|app)", strings.TrimSpace(sourceChannel))
	}
}

func prioritizeConnectorRoute(routes []string, preferred string) []string {
	preferred = normalizeChannel(preferred)
	if len(routes) <= 1 || preferred == "" {
		return routes
	}
	matchedIndex := -1
	for index, route := range routes {
		if normalizeChannel(route) == preferred {
			matchedIndex = index
			break
		}
	}
	if matchedIndex <= 0 {
		return routes
	}
	reordered := make([]string, 0, len(routes))
	reordered = append(reordered, routes[matchedIndex])
	reordered = append(reordered, routes[:matchedIndex]...)
	reordered = append(reordered, routes[matchedIndex+1:]...)
	return reordered
}

func normalizeBoundRouteChannel(logicalChannel string, route string) string {
	normalizedRoute := normalizeDeliveryRouteChannel(route)
	switch logicalChannel {
	case "message":
		switch normalizedRoute {
		case "imessage":
			return "imessage"
		case "twilio":
			return "twilio"
		}
	case "sms":
		switch normalizedRoute {
		case "twilio":
			return "twilio"
		}
	case "voice":
		switch normalizedRoute {
		case "twilio":
			return "twilio"
		}
	case "app":
		switch normalizedRoute {
		case "builtin.app":
			return "builtin.app"
		}
	}
	return normalizedRoute
}

func normalizeDeliveryRouteChannel(route string) string {
	switch normalizeChannel(route) {
	case "", "imessage":
		return "imessage"
	case "twilio", "sms":
		return "twilio"
	case "builtin.app", "app":
		return "builtin.app"
	default:
		return normalizeChannel(route)
	}
}
