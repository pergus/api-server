package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestWatchSSEFormatting tests that watch streams valid SSE format.
func TestWatchSSEFormatting(t *testing.T) {
	// Setup
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Create and register a test resource
	storage := NewMemoryStorage().(*MemoryStorage)
	storage.SetEventBus(eventBus, "orders")
	resource := &testResource{
		name:    "orders",
		storage: storage,
	}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Start watch in a goroutine
	watchDone := make(chan bool, 1)
	var watchedEvents []string

	go func() {
		resp, err := http.Get(server.URL + "/api/orders?watch=true")
		if err != nil {
			t.Errorf("Watch request failed: %v", err)
			watchDone <- false
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		// Read first few lines (connection message)
		for i := 0; i < 5; i++ {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				break
			}
			watchedEvents = append(watchedEvents, strings.TrimSpace(line))
		}
		watchDone <- true
	}()

	// Give watch time to connect
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	order := &testOrder{ID: "order-1", Name: "Test Order"}
	eventBus.Publish(Event{
		Type:      Added,
		Resource:  "orders",
		Object:    order,
		Timestamp: time.Now(),
	})

	// Wait for watch to receive
	<-watchDone

	// Verify we got the connection message
	if len(watchedEvents) == 0 {
		t.Fatal("No events received from watch stream")
	}

	// Check that we have proper SSE format
	foundEvent := false
	for _, evt := range watchedEvents {
		if strings.HasPrefix(evt, "event:") {
			foundEvent = true
			break
		}
	}

	if !foundEvent {
		t.Logf("Events: %v", watchedEvents)
		t.Error("No SSE event: line found in watch stream")
	}
}

// TestListWithoutWatch tests normal list operation.
func TestListWithoutWatch(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Create and register resource
	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testResource{
		name:    "orders",
		storage: storage,
	}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	// Add some data
	storage.Create(&testOrder{ID: "order-1", Name: "Order 1"})
	storage.Create(&testOrder{ID: "order-2", Name: "Order 2"})

	router.Setup()

	// Test regular list (no watch)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/orders")
	if err != nil {
		t.Fatalf("List request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Expected 2 items, got %d", result.Count)
	}
}

// TestCreatePublishesEvent tests that create publishes events.
func TestCreatePublishesEvent(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Setup
	storage := NewMemoryStorage().(*MemoryStorage)
	storage.SetEventBus(eventBus, "orders")
	resource := &testResource{
		name:    "orders",
		storage: storage,
	}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	// Subscribe to events
	sub := eventBus.Subscribe("orders")
	defer eventBus.Unsubscribe(sub)

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Create via HTTP
	orderJSON := `{"id":"order-1","name":"Test Order"}`
	resp, err := http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(orderJSON)),
	)
	if err != nil {
		t.Fatalf("Create request failed: %v", err)
	}
	resp.Body.Close()

	// Check we received event
	select {
	case event := <-sub.Events:
		if event.Type != Added {
			t.Errorf("Expected ADDED event, got %s", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// Test helper - temporary resource wrapper for testing
type testResource struct {
	name    string
	storage Storage
}

func (r *testResource) Name() string {
	return r.name
}

func (r *testResource) NewObject() any {
	return &testOrder{}
}

func (r *testResource) Storage() Storage {
	return r.storage
}
