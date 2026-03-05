package transport

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultEventBrokerSubscriberBuffer        = 32
	defaultEventBrokerSlowSubscriberDropLimit = 128
	defaultEventBrokerOverflowPolicy          = "drop_event_for_subscriber"
	defaultEventBrokerSlowSubscriberAction    = "disconnect_after_consecutive_drops"
	defaultEventBrokerPublishQueueBuffer      = 64
	defaultEventBrokerPublishQueuePolicy      = "drop_oldest_queued_publish_event"
)

type EventBrokerOptions struct {
	DefaultSubscriberBuffer        int
	SlowSubscriberConsecutiveLimit int
	PublishQueueBuffer             int
}

type EventBrokerBackpressurePolicy struct {
	OverflowPolicy                 string `json:"overflow_policy"`
	SlowSubscriberAction           string `json:"slow_subscriber_action"`
	SlowSubscriberConsecutiveLimit int    `json:"slow_subscriber_consecutive_limit"`
	DefaultSubscriberBuffer        int    `json:"default_subscriber_buffer"`
	PublishQueueBuffer             int    `json:"publish_queue_buffer"`
	PublishQueuePolicy             string `json:"publish_queue_policy"`
}

type EventBrokerSubscriberDiagnostics struct {
	SubscriptionID   int       `json:"subscription_id"`
	BufferCapacity   int       `json:"buffer_capacity"`
	BufferDepth      int       `json:"buffer_depth"`
	DeliveredEvents  int64     `json:"delivered_events"`
	DroppedEvents    int64     `json:"dropped_events"`
	ConsecutiveDrops int       `json:"consecutive_drops"`
	LastDroppedAt    time.Time `json:"last_dropped_at,omitempty"`
}

type EventBrokerDiagnostics struct {
	Sequence                  int64                              `json:"sequence"`
	ActiveSubscribers         int                                `json:"active_subscribers"`
	PublishedEvents           int64                              `json:"published_events"`
	DeliveredEvents           int64                              `json:"delivered_events"`
	DroppedEvents             int64                              `json:"dropped_events"`
	DroppedPublishEvents      int64                              `json:"dropped_publish_events"`
	SlowSubscriberDisconnects int64                              `json:"slow_subscriber_disconnects"`
	PublishQueueDepth         int                                `json:"publish_queue_depth"`
	PublishQueueCapacity      int                                `json:"publish_queue_capacity"`
	Policy                    EventBrokerBackpressurePolicy      `json:"policy"`
	Subscribers               []EventBrokerSubscriberDiagnostics `json:"subscribers,omitempty"`
}

type EventBroker struct {
	subscribersMu                  sync.RWMutex
	subscribers                    map[int]*eventBrokerSubscriber
	nextSubID                      int
	sequence                       int64
	publishedEvents                int64
	deliveredEvents                int64
	droppedEvents                  int64
	droppedPublishEvents           int64
	slowSubscriberDisconnects      int64
	defaultSubscriberBuffer        int
	slowSubscriberConsecutiveLimit int
	publishQueueBuffer             int
	publishQueue                   chan RealtimeEventEnvelope
	publishAdmission               sync.RWMutex
	publishLoopDone                chan struct{}
	closeOnce                      sync.Once
	closed                         int32
	closeCallCountValue            int64
}

type eventBrokerSubscriber struct {
	mu               sync.Mutex
	channel          chan RealtimeEventEnvelope
	deliveredEvents  int64
	droppedEvents    int64
	consecutiveDrops int
	lastDroppedAt    time.Time
	closed           bool
}

type eventBrokerSubscriberPublishResult struct {
	delivered        bool
	dropped          bool
	shouldDisconnect bool
}

func (s *eventBrokerSubscriber) publish(
	event RealtimeEventEnvelope,
	now time.Time,
	slowSubscriberConsecutiveLimit int,
) eventBrokerSubscriberPublishResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return eventBrokerSubscriberPublishResult{}
	}

	select {
	case s.channel <- event:
		s.deliveredEvents++
		s.consecutiveDrops = 0
		return eventBrokerSubscriberPublishResult{
			delivered: true,
		}
	default:
		s.droppedEvents++
		s.consecutiveDrops++
		s.lastDroppedAt = now
		return eventBrokerSubscriberPublishResult{
			dropped:          true,
			shouldDisconnect: s.consecutiveDrops >= slowSubscriberConsecutiveLimit,
		}
	}
}

func (s *eventBrokerSubscriber) close() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}
	s.closed = true
	close(s.channel)
	return true
}

func (s *eventBrokerSubscriber) diagnostics(subscriptionID int) EventBrokerSubscriberDiagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return EventBrokerSubscriberDiagnostics{
		SubscriptionID:   subscriptionID,
		BufferCapacity:   cap(s.channel),
		BufferDepth:      len(s.channel),
		DeliveredEvents:  s.deliveredEvents,
		DroppedEvents:    s.droppedEvents,
		ConsecutiveDrops: s.consecutiveDrops,
		LastDroppedAt:    s.lastDroppedAt,
	}
}

type eventBrokerSubscriberSnapshot struct {
	subscriptionID int
	subscriber     *eventBrokerSubscriber
}

func NewEventBroker() *EventBroker {
	return NewEventBrokerWithOptions(EventBrokerOptions{})
}

func NewEventBrokerWithOptions(opts EventBrokerOptions) *EventBroker {
	defaultBuffer := opts.DefaultSubscriberBuffer
	if defaultBuffer <= 0 {
		defaultBuffer = defaultEventBrokerSubscriberBuffer
	}
	slowLimit := opts.SlowSubscriberConsecutiveLimit
	if slowLimit <= 0 {
		slowLimit = defaultEventBrokerSlowSubscriberDropLimit
	}
	publishQueueBuffer := opts.PublishQueueBuffer
	if publishQueueBuffer <= 0 {
		publishQueueBuffer = defaultEventBrokerPublishQueueBuffer
	}
	broker := &EventBroker{
		subscribers:                    map[int]*eventBrokerSubscriber{},
		defaultSubscriberBuffer:        defaultBuffer,
		slowSubscriberConsecutiveLimit: slowLimit,
		publishQueueBuffer:             publishQueueBuffer,
		publishQueue:                   make(chan RealtimeEventEnvelope, publishQueueBuffer),
		publishLoopDone:                make(chan struct{}),
	}
	go broker.runPublishLoop()
	return broker
}

func (b *EventBroker) Publish(event RealtimeEventEnvelope) error {
	if b == nil {
		return nil
	}

	b.publishAdmission.RLock()
	defer b.publishAdmission.RUnlock()

	if atomic.LoadInt32(&b.closed) != 0 {
		atomic.AddInt64(&b.droppedPublishEvents, 1)
		return nil
	}

	if b.tryEnqueuePublishEvent(event) {
		return nil
	}

	// Admission policy under pressure: drop the oldest queued publish event so
	// request paths stay non-blocking while preserving latest-state visibility.
	if b.dropOldestQueuedPublishEvent() {
		atomic.AddInt64(&b.droppedPublishEvents, 1)
	}

	if b.tryEnqueuePublishEvent(event) {
		return nil
	}
	atomic.AddInt64(&b.droppedPublishEvents, 1)
	return nil
}

func (b *EventBroker) runPublishLoop() {
	defer close(b.publishLoopDone)
	for event := range b.publishQueue {
		b.publish(event)
	}
}

func (b *EventBroker) publish(event RealtimeEventEnvelope) {
	now := time.Now().UTC()
	event.Sequence = atomic.AddInt64(&b.sequence, 1)
	atomic.AddInt64(&b.publishedEvents, 1)

	snapshots := b.snapshotSubscribers()
	slowSubscriberIDs := make([]int, 0, 2)
	var deliveredDelta int64
	var droppedDelta int64
	for _, snapshot := range snapshots {
		result := snapshot.subscriber.publish(event, now, b.slowSubscriberConsecutiveLimit)
		if result.delivered {
			deliveredDelta++
			continue
		}
		if result.dropped {
			droppedDelta++
		}
		if result.shouldDisconnect {
			slowSubscriberIDs = append(slowSubscriberIDs, snapshot.subscriptionID)
		}
	}
	if deliveredDelta > 0 {
		atomic.AddInt64(&b.deliveredEvents, deliveredDelta)
	}
	if droppedDelta > 0 {
		atomic.AddInt64(&b.droppedEvents, droppedDelta)
	}

	if len(slowSubscriberIDs) > 0 {
		b.disconnectSubscribers(slowSubscriberIDs, true)
	}
}

func (b *EventBroker) tryEnqueuePublishEvent(event RealtimeEventEnvelope) bool {
	select {
	case b.publishQueue <- event:
		return true
	default:
		return false
	}
}

func (b *EventBroker) dropOldestQueuedPublishEvent() bool {
	select {
	case <-b.publishQueue:
		return true
	default:
		return false
	}
}

func (b *EventBroker) Subscribe(buffer int) (int, <-chan RealtimeEventEnvelope) {
	if b == nil {
		return newClosedEventBrokerSubscription(defaultEventBrokerSubscriberBuffer)
	}
	if buffer <= 0 {
		buffer = b.defaultSubscriberBuffer
		if buffer <= 0 {
			buffer = defaultEventBrokerSubscriberBuffer
		}
	}
	if atomic.LoadInt32(&b.closed) != 0 {
		return newClosedEventBrokerSubscription(buffer)
	}

	b.subscribersMu.Lock()
	defer b.subscribersMu.Unlock()
	if atomic.LoadInt32(&b.closed) != 0 {
		return newClosedEventBrokerSubscription(buffer)
	}
	channel := make(chan RealtimeEventEnvelope, buffer)
	id := b.nextSubID
	b.nextSubID++
	b.subscribers[id] = &eventBrokerSubscriber{
		channel: channel,
	}
	return id, channel
}

func (b *EventBroker) Close() {
	if b == nil {
		return
	}

	atomic.AddInt64(&b.closeCallCountValue, 1)
	b.closeOnce.Do(func() {
		b.publishAdmission.Lock()
		if atomic.LoadInt32(&b.closed) == 0 {
			atomic.StoreInt32(&b.closed, 1)
			close(b.publishQueue)
		}
		b.publishAdmission.Unlock()

		if b.publishLoopDone != nil {
			<-b.publishLoopDone
		}
		b.disconnectAllSubscribers()
	})
}

func (b *EventBroker) Unsubscribe(subscriptionID int) {
	if b == nil {
		return
	}
	b.subscribersMu.Lock()
	subscriber, ok := b.subscribers[subscriptionID]
	if !ok {
		b.subscribersMu.Unlock()
		return
	}
	delete(b.subscribers, subscriptionID)
	b.subscribersMu.Unlock()
	subscriber.close()
}

func (b *EventBroker) BackpressurePolicy() EventBrokerBackpressurePolicy {
	if b == nil {
		return EventBrokerBackpressurePolicy{
			OverflowPolicy:                 defaultEventBrokerOverflowPolicy,
			SlowSubscriberAction:           defaultEventBrokerSlowSubscriberAction,
			SlowSubscriberConsecutiveLimit: defaultEventBrokerSlowSubscriberDropLimit,
			DefaultSubscriberBuffer:        defaultEventBrokerSubscriberBuffer,
			PublishQueueBuffer:             defaultEventBrokerPublishQueueBuffer,
			PublishQueuePolicy:             defaultEventBrokerPublishQueuePolicy,
		}
	}

	return EventBrokerBackpressurePolicy{
		OverflowPolicy:                 defaultEventBrokerOverflowPolicy,
		SlowSubscriberAction:           defaultEventBrokerSlowSubscriberAction,
		SlowSubscriberConsecutiveLimit: b.slowSubscriberConsecutiveLimit,
		DefaultSubscriberBuffer:        b.defaultSubscriberBuffer,
		PublishQueueBuffer:             b.publishQueueCapacity(),
		PublishQueuePolicy:             defaultEventBrokerPublishQueuePolicy,
	}
}

func (b *EventBroker) Diagnostics() EventBrokerDiagnostics {
	diagnostics := EventBrokerDiagnostics{
		Policy: b.BackpressurePolicy(),
	}
	if b == nil {
		return diagnostics
	}

	diagnostics.Sequence = atomic.LoadInt64(&b.sequence)
	diagnostics.PublishedEvents = atomic.LoadInt64(&b.publishedEvents)
	diagnostics.DeliveredEvents = atomic.LoadInt64(&b.deliveredEvents)
	diagnostics.DroppedEvents = atomic.LoadInt64(&b.droppedEvents)
	diagnostics.DroppedPublishEvents = atomic.LoadInt64(&b.droppedPublishEvents)
	diagnostics.SlowSubscriberDisconnects = atomic.LoadInt64(&b.slowSubscriberDisconnects)
	diagnostics.PublishQueueDepth = len(b.publishQueue)
	diagnostics.PublishQueueCapacity = b.publishQueueCapacity()
	snapshots := b.snapshotSubscribers()

	diagnostics.ActiveSubscribers = len(snapshots)
	if len(snapshots) == 0 {
		return diagnostics
	}

	diagnostics.Subscribers = make([]EventBrokerSubscriberDiagnostics, 0, len(snapshots))
	for _, snapshot := range snapshots {
		diagnostics.Subscribers = append(diagnostics.Subscribers, snapshot.subscriber.diagnostics(snapshot.subscriptionID))
	}
	sort.Slice(diagnostics.Subscribers, func(i, j int) bool {
		if diagnostics.Subscribers[i].DroppedEvents == diagnostics.Subscribers[j].DroppedEvents {
			return diagnostics.Subscribers[i].SubscriptionID < diagnostics.Subscribers[j].SubscriptionID
		}
		return diagnostics.Subscribers[i].DroppedEvents > diagnostics.Subscribers[j].DroppedEvents
	})
	return diagnostics
}

func (b *EventBroker) snapshotSubscribers() []eventBrokerSubscriberSnapshot {
	if b == nil {
		return nil
	}
	b.subscribersMu.RLock()
	defer b.subscribersMu.RUnlock()
	if len(b.subscribers) == 0 {
		return nil
	}
	snapshots := make([]eventBrokerSubscriberSnapshot, 0, len(b.subscribers))
	for subscriptionID, subscriber := range b.subscribers {
		snapshots = append(snapshots, eventBrokerSubscriberSnapshot{
			subscriptionID: subscriptionID,
			subscriber:     subscriber,
		})
	}
	return snapshots
}

func (b *EventBroker) disconnectSubscribers(subscriptionIDs []int, slowDisconnect bool) {
	if b == nil || len(subscriptionIDs) == 0 {
		return
	}
	subscribersToClose := make([]*eventBrokerSubscriber, 0, len(subscriptionIDs))
	var slowDisconnectDelta int64

	b.subscribersMu.Lock()
	for _, subscriptionID := range subscriptionIDs {
		subscriber, ok := b.subscribers[subscriptionID]
		if !ok {
			continue
		}
		delete(b.subscribers, subscriptionID)
		subscribersToClose = append(subscribersToClose, subscriber)
		if slowDisconnect {
			slowDisconnectDelta++
		}
	}
	b.subscribersMu.Unlock()
	if slowDisconnectDelta > 0 {
		atomic.AddInt64(&b.slowSubscriberDisconnects, slowDisconnectDelta)
	}

	for _, subscriber := range subscribersToClose {
		subscriber.close()
	}
}

func (b *EventBroker) disconnectAllSubscribers() {
	if b == nil {
		return
	}
	b.subscribersMu.Lock()
	subscribersToClose := make([]*eventBrokerSubscriber, 0, len(b.subscribers))
	for subscriptionID, subscriber := range b.subscribers {
		delete(b.subscribers, subscriptionID)
		subscribersToClose = append(subscribersToClose, subscriber)
	}
	b.subscribersMu.Unlock()
	for _, subscriber := range subscribersToClose {
		subscriber.close()
	}
}

func (b *EventBroker) publishQueueCapacity() int {
	if b == nil || b.publishQueueBuffer <= 0 {
		return defaultEventBrokerPublishQueueBuffer
	}
	return b.publishQueueBuffer
}

func (b *EventBroker) closeCallCount() int64 {
	if b == nil {
		return 0
	}
	return atomic.LoadInt64(&b.closeCallCountValue)
}

func newClosedEventBrokerSubscription(buffer int) (int, <-chan RealtimeEventEnvelope) {
	if buffer <= 0 {
		buffer = defaultEventBrokerSubscriberBuffer
	}
	channel := make(chan RealtimeEventEnvelope, buffer)
	close(channel)
	return -1, channel
}
