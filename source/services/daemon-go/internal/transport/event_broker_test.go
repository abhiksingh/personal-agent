package transport

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBrokerBackpressurePolicyDefaults(t *testing.T) {
	broker := NewEventBroker()
	policy := broker.BackpressurePolicy()
	if policy.OverflowPolicy != defaultEventBrokerOverflowPolicy {
		t.Fatalf("expected overflow policy %q, got %q", defaultEventBrokerOverflowPolicy, policy.OverflowPolicy)
	}
	if policy.SlowSubscriberAction != defaultEventBrokerSlowSubscriberAction {
		t.Fatalf("expected slow-subscriber action %q, got %q", defaultEventBrokerSlowSubscriberAction, policy.SlowSubscriberAction)
	}
	if policy.SlowSubscriberConsecutiveLimit <= 0 {
		t.Fatalf("expected positive slow-subscriber limit, got %d", policy.SlowSubscriberConsecutiveLimit)
	}
	if policy.DefaultSubscriberBuffer <= 0 {
		t.Fatalf("expected positive default subscriber buffer, got %d", policy.DefaultSubscriberBuffer)
	}
	if policy.PublishQueueBuffer <= 0 {
		t.Fatalf("expected positive publish queue buffer, got %d", policy.PublishQueueBuffer)
	}
	if policy.PublishQueuePolicy != defaultEventBrokerPublishQueuePolicy {
		t.Fatalf("expected publish queue policy %q, got %q", defaultEventBrokerPublishQueuePolicy, policy.PublishQueuePolicy)
	}
}

func TestEventBrokerDisconnectsSlowSubscribersAndTracksDrops(t *testing.T) {
	broker := NewEventBrokerWithOptions(EventBrokerOptions{
		DefaultSubscriberBuffer:        2,
		SlowSubscriberConsecutiveLimit: 2,
	})

	fastSubID, fastSubscription := broker.Subscribe(8)
	t.Cleanup(func() { broker.Unsubscribe(fastSubID) })
	_, slowSubscription := broker.Subscribe(1)

	for idx := 0; idx < 5; idx++ {
		if err := broker.Publish(RealtimeEventEnvelope{EventID: "evt", EventType: "task_run_lifecycle"}); err != nil {
			t.Fatalf("publish event %d: %v", idx, err)
		}
	}

	receivedFast := 0
	for receivedFast < 5 {
		select {
		case _, ok := <-fastSubscription:
			if !ok {
				t.Fatalf("expected fast subscription to remain open")
			}
			receivedFast++
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for fast subscriber events; received=%d", receivedFast)
		}
	}

	slowClosed := false
	for !slowClosed {
		select {
		case _, ok := <-slowSubscription:
			if !ok {
				slowClosed = true
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for slow subscriber disconnect")
		}
	}

	diagnostics := broker.Diagnostics()
	if diagnostics.PublishedEvents != 5 {
		t.Fatalf("expected published_events=5, got %d", diagnostics.PublishedEvents)
	}
	if diagnostics.DroppedEvents == 0 {
		t.Fatalf("expected dropped_events to be positive")
	}
	if diagnostics.SlowSubscriberDisconnects != 1 {
		t.Fatalf("expected one slow subscriber disconnect, got %d", diagnostics.SlowSubscriberDisconnects)
	}
	if diagnostics.ActiveSubscribers != 1 {
		t.Fatalf("expected only fast subscriber to remain active, got %d", diagnostics.ActiveSubscribers)
	}
}

func TestEventBrokerConcurrentUnsubscribeDuringPublishPreservesSequence(t *testing.T) {
	broker := NewEventBrokerWithOptions(EventBrokerOptions{
		DefaultSubscriberBuffer:        1,
		SlowSubscriberConsecutiveLimit: 3,
	})

	subscriptionID, subscription := broker.Subscribe(1)

	publishComplete := make(chan struct{})
	go func() {
		for idx := 0; idx < 256; idx++ {
			_ = broker.Publish(RealtimeEventEnvelope{
				EventID:   fmt.Sprintf("evt-%03d", idx),
				EventType: "task_run_lifecycle",
			})
		}
		close(publishComplete)
	}()

	broker.Unsubscribe(subscriptionID)

	select {
	case <-publishComplete:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for concurrent publish loop to complete")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		diagnostics := broker.Diagnostics()
		if diagnostics.PublishedEvents == 256 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case _, ok := <-subscription:
		if ok {
			t.Fatalf("expected unsubscribed channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for unsubscribed channel close")
	}

	diagnostics := broker.Diagnostics()
	if diagnostics.Sequence != diagnostics.PublishedEvents {
		t.Fatalf("expected sequence and published_events to match, got sequence=%d published=%d", diagnostics.Sequence, diagnostics.PublishedEvents)
	}
	totalAccounted := diagnostics.PublishedEvents + diagnostics.DroppedPublishEvents
	if totalAccounted != 256 {
		t.Fatalf(
			"expected submitted publish events to be fully accounted (published+dropped=256), got published=%d dropped=%d",
			diagnostics.PublishedEvents,
			diagnostics.DroppedPublishEvents,
		)
	}
}

func TestEventBrokerPublishQueuePressureDropsOldestWithoutBlockingCallers(t *testing.T) {
	broker := NewEventBrokerWithOptions(EventBrokerOptions{
		PublishQueueBuffer: 1,
	})

	// Hold subscriber map write lock so the publish loop stalls in snapshot,
	// forcing publish-queue pressure while Publish callers continue.
	var droppedUnderPressure int64
	broker.subscribersMu.Lock()
	func() {
		defer broker.subscribersMu.Unlock()

		done := make(chan struct{})
		go func() {
			for idx := 0; idx < 256; idx++ {
				_ = broker.Publish(RealtimeEventEnvelope{
					EventID:   fmt.Sprintf("evt-pressure-%03d", idx),
					EventType: "task_run_lifecycle",
				})
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("publish callers blocked under publish-queue pressure")
		}

		droppedUnderPressure = atomic.LoadInt64(&broker.droppedPublishEvents)
	}()
	if droppedUnderPressure == 0 {
		t.Fatalf("expected dropped publish events under pressure, got %d", droppedUnderPressure)
	}

	diagnostics := broker.Diagnostics()
	if diagnostics.PublishQueueCapacity != 1 {
		t.Fatalf("expected publish queue capacity=1, got %+v", diagnostics)
	}
	if diagnostics.Policy.PublishQueuePolicy != defaultEventBrokerPublishQueuePolicy {
		t.Fatalf("expected publish queue policy %q, got %+v", defaultEventBrokerPublishQueuePolicy, diagnostics.Policy)
	}
}

func TestEventBrokerCloseClosesSubscribersAndStopsPublishLoop(t *testing.T) {
	broker := NewEventBrokerWithOptions(EventBrokerOptions{
		DefaultSubscriberBuffer: 4,
		PublishQueueBuffer:      4,
	})
	subscriptionID, subscription := broker.Subscribe(4)
	if subscriptionID < 0 {
		t.Fatalf("expected active subscription id, got %d", subscriptionID)
	}

	if err := broker.Publish(RealtimeEventEnvelope{
		EventID:   "evt-close",
		EventType: "task_run_lifecycle",
	}); err != nil {
		t.Fatalf("publish before close: %v", err)
	}

	broker.Close()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-subscription:
			if !ok {
				goto subscriptionClosed
			}
		case <-deadline:
			t.Fatalf("timeout waiting for subscription close during broker shutdown")
		}
	}

subscriptionClosed:
	select {
	case <-broker.publishLoopDone:
	default:
		t.Fatalf("expected publish loop done signal to be closed")
	}

	diagnostics := broker.Diagnostics()
	if diagnostics.ActiveSubscribers != 0 {
		t.Fatalf("expected no active subscribers after close, got %d", diagnostics.ActiveSubscribers)
	}
}

func TestEventBrokerPostClosePublishAndSubscribeSemantics(t *testing.T) {
	broker := NewEventBroker()
	broker.Close()

	for idx := 0; idx < 3; idx++ {
		if err := broker.Publish(RealtimeEventEnvelope{
			EventID:   fmt.Sprintf("evt-closed-%d", idx),
			EventType: "task_run_lifecycle",
		}); err != nil {
			t.Fatalf("publish after close (%d): %v", idx, err)
		}
	}

	subscriptionID, subscription := broker.Subscribe(8)
	if subscriptionID != -1 {
		t.Fatalf("expected closed broker subscription id -1, got %d", subscriptionID)
	}
	select {
	case _, ok := <-subscription:
		if ok {
			t.Fatalf("expected closed subscription channel after broker close")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for closed subscription channel")
	}

	diagnostics := broker.Diagnostics()
	if diagnostics.PublishedEvents != 0 {
		t.Fatalf("expected no published events after close, got %d", diagnostics.PublishedEvents)
	}
	if diagnostics.DroppedPublishEvents != 3 {
		t.Fatalf("expected dropped_publish_events=3 after post-close publishes, got %d", diagnostics.DroppedPublishEvents)
	}
}

func TestEventBrokerCloseAcrossRepeatedStartStopCyclesLeavesNoSubscribers(t *testing.T) {
	for cycle := 0; cycle < 20; cycle++ {
		broker := NewEventBrokerWithOptions(EventBrokerOptions{
			DefaultSubscriberBuffer: 2,
			PublishQueueBuffer:      2,
		})
		subscriptionID, subscription := broker.Subscribe(2)
		if subscriptionID < 0 {
			t.Fatalf("cycle %d: expected active subscription id, got %d", cycle, subscriptionID)
		}

		_ = broker.Publish(RealtimeEventEnvelope{
			EventID:   fmt.Sprintf("evt-cycle-%d", cycle),
			EventType: "task_run_lifecycle",
		})
		broker.Close()

		deadline := time.After(2 * time.Second)
		for {
			select {
			case _, ok := <-subscription:
				if !ok {
					goto closed
				}
			case <-deadline:
				t.Fatalf("cycle %d: timeout waiting for subscription close", cycle)
			}
		}

	closed:
		diagnostics := broker.Diagnostics()
		if diagnostics.ActiveSubscribers != 0 {
			t.Fatalf("cycle %d: expected active_subscribers=0 after close, got %d", cycle, diagnostics.ActiveSubscribers)
		}
	}
}

func BenchmarkEventBrokerPublishFanout(b *testing.B) {
	for _, subscriberCount := range []int{32, 128, 512} {
		b.Run(fmt.Sprintf("subscribers_%d", subscriberCount), func(b *testing.B) {
			broker := NewEventBrokerWithOptions(EventBrokerOptions{
				DefaultSubscriberBuffer:        64,
				SlowSubscriberConsecutiveLimit: 1024,
			})

			subscriptionIDs := make([]int, 0, subscriberCount)
			done := make(chan struct{})
			var drainWG sync.WaitGroup

			for idx := 0; idx < subscriberCount; idx++ {
				subID, subCh := broker.Subscribe(64)
				subscriptionIDs = append(subscriptionIDs, subID)
				drainWG.Add(1)
				go func(channel <-chan RealtimeEventEnvelope) {
					defer drainWG.Done()
					for {
						select {
						case <-done:
							return
						case _, ok := <-channel:
							if !ok {
								return
							}
						}
					}
				}(subCh)
			}

			b.ResetTimer()
			for idx := 0; idx < b.N; idx++ {
				_ = broker.Publish(RealtimeEventEnvelope{
					EventID:   "evt",
					EventType: "task_run_lifecycle",
				})
			}
			b.StopTimer()

			close(done)
			for _, subID := range subscriptionIDs {
				broker.Unsubscribe(subID)
			}
			drainWG.Wait()
		})
	}
}
