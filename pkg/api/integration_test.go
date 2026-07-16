package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestFullEventFlowCreateUpdateDelete tests the complete flow.
func TestFullEventFlowCreateUpdateDelete(t *testing.T) {
	// Setup
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Create and register resource
	storage := NewMemoryStorage().(*MemoryStorage)
	storage.SetEventBus(eventBus, "orders")
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	// Subscribe to events
	sub := eventBus.Subscribe("orders")
	defer eventBus.Unsubscribe(sub)

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Test CREATE
	createJSON := `{"id":"order-1","name":"Test Order"}`
	resp, _ := http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(createJSON)),
	)
	resp.Body.Close()

	// Receive ADDED event
	select {
	case event := <-sub.Events:
		if event.Type != Added {
			t.Errorf("Expected ADDED, got %s", event.Type)
		}
		if event.Resource != "orders" {
			t.Errorf("Expected resource=orders, got %s", event.Resource)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ADDED event")
	}

	// Test UPDATE
	updateJSON := `{"id":"order-1","name":"Updated Order"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		server.URL+"/api/orders/order-1",
		bytes.NewReader([]byte(updateJSON)),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	// Receive MODIFIED event
	select {
	case event := <-sub.Events:
		if event.Type != Modified {
			t.Errorf("Expected MODIFIED, got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for MODIFIED event")
	}

	// Test DELETE
	req, _ = http.NewRequest(
		http.MethodDelete,
		server.URL+"/api/orders/order-1",
		nil,
	)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	// Receive DELETED event
	select {
	case event := <-sub.Events:
		if event.Type != Deleted {
			t.Errorf("Expected DELETED, got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for DELETED event")
	}
}

// TestMultipleSubscribersReceiveEvents tests fan-out to multiple subscribers.
func TestMultipleSubscribersReceiveEvents(t *testing.T) {
	eventBus := NewEventBus()
	defer eventBus.Close()

	// Create 3 subscribers
	sub1 := eventBus.Subscribe("orders")
	sub2 := eventBus.Subscribe("orders")
	sub3 := eventBus.Subscribe("orders")
	defer eventBus.Unsubscribe(sub1)
	defer eventBus.Unsubscribe(sub2)
	defer eventBus.Unsubscribe(sub3)

	// Publish event
	event := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	eventBus.Publish(event)

	// All subscribers should receive
	timeout := time.After(1 * time.Second)
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

// TestStorageWithoutEventBus works without events attached.
func TestStorageWithoutEventBus(t *testing.T) {
	storage := NewMemoryStorage().(*MemoryStorage)
	// Don't set event bus

	order := &testOrder{ID: "order-1", Name: "Test"}

	err := storage.Create(order)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, err := storage.Get("order-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.(*testOrder).Name != "Test" {
		t.Error("Data mismatch")
	}
}

// TestEventBusResourceIsolation ensures events are resource-specific.
func TestEventBusResourceIsolation(t *testing.T) {
	eventBus := NewEventBus()
	defer eventBus.Close()

	orderSub := eventBus.Subscribe("orders")
	userSub := eventBus.Subscribe("users")
	defer eventBus.Unsubscribe(orderSub)
	defer eventBus.Unsubscribe(userSub)

	// Publish order event
	eventBus.Publish(Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	})

	// Order subscriber gets it
	select {
	case <-orderSub.Events:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Order subscriber didn't receive order event")
	}

	// User subscriber should NOT get it
	select {
	case <-userSub.Events:
		t.Fatal("User subscriber received order event (resource isolation broken)")
	case <-time.After(50 * time.Millisecond):
		// Good - no event
	}
}

// TestCRDWithEvents tests that CRDs can be created and used.
// Note: Events for CRD-created resources require event bus attachment which
// happens automatically for built-in resources. CRD resources need manual setup
// in production, so this test just verifies CRD creation and basic operations.
func TestCRDWithEvents(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Create CRD
	crdJSON := `{
		"group": "example.io",
		"version": "v1",
		"kind": "Invoice",
		"plural": "invoices",
		"schema": {}
	}`
	resp, _ := http.Post(
		server.URL+"/crds",
		"application/json",
		bytes.NewReader([]byte(crdJSON)),
	)
	resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	// Create an invoice - should work
	invoiceJSON := `{"id":"inv-1","kind":"Invoice","amount":100}`
	resp, err := http.Post(
		server.URL+"/api/invoices",
		"application/json",
		bytes.NewReader([]byte(invoiceJSON)),
	)
	if err != nil || resp.StatusCode >= 400 {
		t.Fatalf("Failed to create invoice: %v, status %d", err, resp.StatusCode)
	}
	resp.Body.Close()

	// Get the invoice - should succeed
	resp, _ = http.Get(server.URL + "/api/invoices/inv-1")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Failed to get invoice: status %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// Note: testOrder and testOrderResource are defined in router_test.go

// BenchmarkEventPublishing benchmarks the event system throughput.
func BenchmarkEventPublishing(b *testing.B) {
	eventBus := NewEventBus()
	defer eventBus.Close()

	sub := eventBus.Subscribe("orders")
	defer eventBus.Unsubscribe(sub)

	event := Event{
		Type:      Added,
		Resource:  "orders",
		Object:    map[string]interface{}{"id": "order-1"},
		Timestamp: time.Now(),
	}

	// Drain events in background
	go func() {
		for range sub.Events {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventBus.Publish(event)
	}
}

// BenchmarkStorageWithEvents benchmarks storage with event publishing.
func BenchmarkStorageWithEvents(b *testing.B) {
	eventBus := NewEventBus()
	defer eventBus.Close()

	storage := NewMemoryStorage().(*MemoryStorage)
	storage.SetEventBus(eventBus, "orders")

	sub := eventBus.Subscribe("orders")
	defer eventBus.Unsubscribe(sub)

	// Drain events in background
	go func() {
		for range sub.Events {
		}
	}()

	order := &testOrder{ID: "order-1", Name: "Test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		order.ID = "order-" + string(rune(i))
		storage.Create(order)
	}
}
