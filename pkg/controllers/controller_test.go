package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/pergus/api-server/pkg/api"
)

// TestOrderControllerReconcileAdded tests that controller sets status on new orders.
func TestOrderControllerReconcileAdded(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	registry := api.NewRegistry()
	storage := api.NewMemoryStorage().(*api.MemoryStorage)
	storage.SetEventBus(eventBus, "orders")

	resource := &testOrderResource{storage: storage}
	registry.Register(resource)

	controller := NewOrderController(eventBus, registry)

	// Pre-populate storage with an order (this happens before reconciliation)
	initialOrder := map[string]interface{}{
		"id":          "order-1",
		"customer_id": "alice",
		"total":       99.99,
		"status":      "draft",
	}
	storage.Create(initialOrder)

	// Create an ADDED event (normally from API, but we're simulating)
	event := api.Event{
		Type:      api.Added,
		Resource:  "orders",
		Object:    initialOrder,
		Timestamp: time.Now(),
	}

	// Reconcile should update the order status to "processing"
	err := controller.Reconcile(event)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Check that order was updated
	updated, err := storage.Get("order-1")
	if err != nil {
		t.Fatalf("Failed to get updated order: %v", err)
	}

	updatedMap := updated.(map[string]interface{})
	status, ok := updatedMap["status"].(string)
	if !ok || status != "processing" {
		t.Errorf("Expected status=processing, got %v", updatedMap["status"])
	}
}

// TestOrderControllerHandlesDelete tests DELETED event handling.
func TestOrderControllerHandlesDelete(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	registry := api.NewRegistry()
	storage := api.NewMemoryStorage().(*api.MemoryStorage)

	resource := &testOrderResource{storage: storage}
	registry.Register(resource)

	controller := NewOrderController(eventBus, registry)

	order := map[string]interface{}{
		"id":          "order-1",
		"customer_id": "alice",
		"total":       99.99,
		"status":      "processing",
	}

	event := api.Event{
		Type:      api.Deleted,
		Resource:  "orders",
		Object:    order,
		Timestamp: time.Now(),
	}

	// Should not panic or error on delete
	err := controller.Reconcile(event)
	if err != nil {
		t.Errorf("Reconcile on delete failed: %v", err)
	}
}

// TestControllerManager tests basic controller management.
func TestControllerManager(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	registry := api.NewRegistry()
	storage := api.NewMemoryStorage().(*api.MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)

	manager := New(eventBus)

	// Register controller
	controller := NewOrderController(eventBus, registry)
	err := manager.Register(controller)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Start manager in goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(ctx)
	}()

	// Wait for context to cancel
	<-ctx.Done()
	<-done

	// Manager should have exited cleanly
}

// TestControllerWithMultipleEvents tests event processing sequence.
func TestControllerWithMultipleEvents(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	registry := api.NewRegistry()
	storage := api.NewMemoryStorage().(*api.MemoryStorage)
	storage.SetEventBus(eventBus, "orders")

	resource := &testOrderResource{storage: storage}
	registry.Register(resource)

	controller := NewOrderController(eventBus, registry)

	// Simulate event sequence
	events := []api.Event{
		{
			Type:      api.Added,
			Resource:  "orders",
			Object:    map[string]interface{}{"id": "order-1", "status": "draft"},
			Timestamp: time.Now(),
		},
		{
			Type:      api.Modified,
			Resource:  "orders",
			Object:    map[string]interface{}{"id": "order-1", "status": "processing"},
			Timestamp: time.Now(),
		},
		{
			Type:      api.Deleted,
			Resource:  "orders",
			Object:    map[string]interface{}{"id": "order-1"},
			Timestamp: time.Now(),
		},
	}

	for _, event := range events {
		if err := controller.Reconcile(event); err != nil {
			t.Errorf("Reconcile failed for %s: %v", event.Type, err)
		}
	}
}

// Test helper - a simple order resource
type testOrderResource struct {
	storage api.Storage
}

func (r *testOrderResource) Name() string {
	return "orders"
}

func (r *testOrderResource) NewObject() any {
	return make(map[string]interface{})
}

func (r *testOrderResource) Storage() api.Storage {
	return r.storage
}
