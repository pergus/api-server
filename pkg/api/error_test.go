package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResourceNotFound tests 404 errors.
func TestResourceNotFound(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)
	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Try to get non-existent resource
	resp, _ := http.Get(server.URL + "/api/nonexistent")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestDuplicateCreate tests error on duplicate create.
func TestDuplicateCreate(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Create first order
	orderJSON := `{"id":"order-1","name":"Test"}`
	resp, _ := http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(orderJSON)),
	)
	resp.Body.Close()

	// Try to create duplicate
	resp, _ = http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(orderJSON)),
	)

	if resp.StatusCode < 400 {
		t.Errorf("Should fail on duplicate, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestInvalidJSON tests bad request handling.
func TestInvalidJSON(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Send invalid JSON
	invalidJSON := `{invalid json}`
	resp, _ := http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(invalidJSON)),
	)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestGetNotFound tests getting non-existent item.
func TestGetNotFound(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	resp, _ := http.Get(server.URL + "/api/orders/nonexistent")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestDeleteNotFound tests deleting non-existent item.
func TestDeleteNotFound(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/orders/nonexistent", nil)
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestUpdateNotFound tests updating non-existent item.
func TestUpdateNotFound(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	updateJSON := `{"id":"nonexistent","name":"Update"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		server.URL+"/api/orders/nonexistent",
		bytes.NewReader([]byte(updateJSON)),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestMethodNotAllowed tests wrong HTTP method.
func TestMethodNotAllowed(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Try PATCH on list endpoint (not allowed)
	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/api/orders", nil)
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestMissingIDField tests error when object lacks ID.
func TestMissingIDField(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	storage := NewMemoryStorage().(*MemoryStorage)
	resource := &testOrderResource{storage: storage}
	registry.Register(resource)
	scheme.Register("orders", func() any { return &testOrder{} })

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Missing required "id" field
	noIDJSON := `{"name":"Test"}`
	resp, _ := http.Post(
		server.URL+"/api/orders",
		"application/json",
		bytes.NewReader([]byte(noIDJSON)),
	)

	if resp.StatusCode < 400 {
		t.Errorf("Should fail without ID, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestInvalidCRDCreate tests CRD creation with missing fields.
func TestInvalidCRDCreate(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)
	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	tests := []struct {
		name string
		json string
	}{
		{"missing group", `{"version":"v1","kind":"Test","plural":"tests"}`},
		{"missing version", `{"group":"example.io","kind":"Test","plural":"tests"}`},
		{"missing kind", `{"group":"example.io","version":"v1","plural":"tests"}`},
		{"missing plural", `{"group":"example.io","version":"v1","kind":"Test"}`},
	}

	for _, tt := range tests {
		resp, _ := http.Post(
			server.URL+"/crds",
			"application/json",
			bytes.NewReader([]byte(tt.json)),
		)

		if resp.StatusCode < 400 {
			t.Errorf("Test %s: expected error, got %d", tt.name, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
