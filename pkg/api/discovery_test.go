package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDiscoveryAPI tests the GET /api discovery endpoint.
func TestDiscoveryAPI(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Register test resources
	for _, name := range []string{"users", "products", "orders"} {
		storage := NewMemoryStorage().(*MemoryStorage)
		res := &testOrderResourceNamed{storage: storage, name: name}
		registry.Register(res)
		scheme.Register(name, func() any { return map[string]interface{}{} })
	}

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Test discovery endpoint
	resp, _ := http.Get(server.URL + "/api")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Discovery failed: %d", resp.StatusCode)
	}

	var result DiscoveryResponse
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if len(result.Resources) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(result.Resources))
	}

	// Check that all resources are present
	found := make(map[string]bool)
	for _, r := range result.Resources {
		found[r] = true
	}

	expected := []string{"users", "products", "orders"}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("Missing resource: %s", name)
		}
	}
}

// TestDiscoveryAPIs tests GET /apis endpoint.
func TestDiscoveryAPIs(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	// Register CRDs with different groups
	crdRegistry.RegisterCRD(&CRDDefinition{
		Group: "example.io", Version: "v1", Kind: "Invoice", Plural: "invoices",
	})
	crdRegistry.RegisterCRD(&CRDDefinition{
		Group: "billing.io", Version: "v1", Kind: "Payment", Plural: "payments",
	})

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	resp, _ := http.Get(server.URL + "/apis")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("APIs discovery failed: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("Invalid response format")
	}

	if len(groups) < 2 {
		t.Errorf("Expected at least 2 groups, got %d", len(groups))
	}
}

// TestDiscoveryAPIGroup tests GET /apis/{group}.
func TestDiscoveryAPIGroup(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	crdRegistry.RegisterCRD(&CRDDefinition{
		Group: "example.io", Version: "v1", Kind: "Invoice", Plural: "invoices",
	})
	crdRegistry.RegisterCRD(&CRDDefinition{
		Group: "example.io", Version: "v1", Kind: "Receipt", Plural: "receipts",
	})

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	resp, _ := http.Get(server.URL + "/apis/example.io")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Group discovery failed: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	// Should return resources for this group
	resources, ok := result["resources"].([]interface{})
	if !ok || len(resources) == 0 {
		t.Error("No resources returned for group")
	}
}

// TestDiscoveryAPIVersion tests GET /apis/{group}/{version}.
func TestDiscoveryAPIVersion(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	crdRegistry.RegisterCRD(&CRDDefinition{
		Group: "example.io", Version: "v1", Kind: "Invoice", Plural: "invoices",
	})

	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	resp, _ := http.Get(server.URL + "/apis/example.io/v1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Version discovery failed: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	// Should have resources for this version
	resources, ok := result["resources"].([]interface{})
	if !ok || len(resources) == 0 {
		t.Error("No resources returned for version")
	}
}

// TestDiscoveryBadGroup tests handling for group with no resources.
func TestDiscoveryBadGroup(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)
	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Try to get a group with no CRDs
	resp, _ := http.Get(server.URL + "/apis/nonexistent.io")
	// Should succeed but be empty or return empty list
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	// Should be empty or have no resources
	resources, _ := result["resources"].([]interface{})
	if len(resources) > 0 {
		t.Error("Non-existent group should have no resources")
	}
}

// testOrderResource with flexible name for testing
type testOrderResourceNamed struct {
	storage Storage
	name    string
}

func (r *testOrderResourceNamed) Name() string {
	return r.name
}

func (r *testOrderResourceNamed) NewObject() any {
	return &testOrder{}
}

func (r *testOrderResourceNamed) Storage() Storage {
	return r.storage
}
