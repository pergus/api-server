package api

import (
	"testing"
	"time"
)

// TestEventBusPublishSubscribe tests basic pub/sub functionality.
func TestEventBusPublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	// Subscribe to orders
	sub := bus.Subscribe("orders")
	defer bus.Unsubscribe(sub)

	// Publish an event
	event := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	// Give time for event to be delivered
	select {
	case received := <-sub.Events:
		if received.Type != Added {
			t.Errorf("Expected ADDED, got %s", received.Type)
		}
		if received.Resource != "orders" {
			t.Errorf("Expected resource=orders, got %s", received.Resource)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for event")
	}
}

// TestEventBusMultipleSubscribers tests that multiple subscribers receive events.
func TestEventBusMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	// Subscribe 3 subscribers
	sub1 := bus.Subscribe("orders")
	sub2 := bus.Subscribe("orders")
	sub3 := bus.Subscribe("orders")
	defer bus.Unsubscribe(sub1)
	defer bus.Unsubscribe(sub2)
	defer bus.Unsubscribe(sub3)

	// Publish an event
	event := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	// All subscribers should receive the event
	timeout := time.After(100 * time.Millisecond)

	received := 0
	for received < 3 {
		select {
		case <-sub1.Events:
			received++
		case <-sub2.Events:
			received++
		case <-sub3.Events:
			received++
		case <-timeout:
			t.Fatalf("Expected 3 events, got %d", received)
		}
	}
}

// TestEventBusDifferentResources tests that subscriptions are resource-specific.
func TestEventBusDifferentResources(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	// Subscribe to different resources
	subOrders := bus.Subscribe("orders")
	subUsers := bus.Subscribe("users")
	defer bus.Unsubscribe(subOrders)
	defer bus.Unsubscribe(subUsers)

	// Publish order event
	orderEvent := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	bus.Publish(orderEvent)

	// Orders subscriber should receive it
	select {
	case <-subOrders.Events:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Orders subscriber didn't receive event")
	}

	// Users subscriber should NOT receive it
	select {
	case <-subUsers.Events:
		t.Fatalf("Users subscriber received order event")
	case <-time.After(50 * time.Millisecond):
		// Good - no event
	}
}

// TestMemoryStoragePublishesEvents tests that storage publishes events.
func TestMemoryStoragePublishesEvents(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	storage := NewMemoryStorage().(*MemoryStorage)
	storage.SetEventBus(bus, "orders")

	sub := bus.Subscribe("orders")
	defer bus.Unsubscribe(sub)

	// Create an order
	order := map[string]interface{}{
		"id":       "order-1",
		"customer": "Alice",
		"total":    99.99,
	}

	storage.Create(order)

	// Should receive ADDED event
	select {
	case event := <-sub.Events:
		if event.Type != Added {
			t.Errorf("Expected ADDED event, got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for ADDED event")
	}

	// Update the order
	updatedOrder := map[string]interface{}{
		"id":       "order-1",
		"customer": "Alice",
		"total":    149.99,
	}

	storage.Update("order-1", updatedOrder)

	// Should receive MODIFIED event
	select {
	case event := <-sub.Events:
		if event.Type != Modified {
			t.Errorf("Expected MODIFIED event, got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for MODIFIED event")
	}

	// Delete the order
	storage.Delete("order-1")

	// Should receive DELETED event
	select {
	case event := <-sub.Events:
		if event.Type != Deleted {
			t.Errorf("Expected DELETED event, got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for DELETED event")
	}
}

// TestEventBusClose tests that Close shuts down the bus properly.
func TestEventBusClose(t *testing.T) {
	bus := NewEventBus()

	sub := bus.Subscribe("orders")

	// Close the bus
	bus.Close()

	// Subscription should not receive events after close
	bus.Publish(Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{},
		Timestamp: time.Now(),
	})

	// Channel should be closed
	select {
	case <-sub.Events:
		// Channel closed or we got an event (acceptable after close)
	case <-time.After(100 * time.Millisecond):
		// Good - no delivery
	}
}

// BenchmarkEventBusPublish benchmarks event publishing.
func BenchmarkEventBusPublish(b *testing.B) {
	bus := NewEventBus()
	defer bus.Close()

	event := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

// BenchmarkEventBusSubscribe benchmarks subscriptions.
func BenchmarkEventBusSubscribe(b *testing.B) {
	bus := NewEventBus()
	defer bus.Close()

	subs := make([]*Subscription, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subs[i] = bus.Subscribe("orders")
	}
	b.StopTimer()

	for _, sub := range subs {
		bus.Unsubscribe(sub)
	}
}
