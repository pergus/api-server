package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCRDRegistration tests CRD creation and retrieval.
func TestCRDRegistration(t *testing.T) {
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
		"schema": {"amount": "number"}
	}`

	resp, _ := http.Post(
		server.URL+"/crds",
		"application/json",
		bytes.NewReader([]byte(crdJSON)),
	)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 200/201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// List CRDs
	resp, _ = http.Get(server.URL + "/crds")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List CRDs failed: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	crds, ok := result["items"].([]interface{})
	if !ok || len(crds) == 0 {
		t.Error("No CRDs returned")
	}
}

// TestCRDDeletion tests CRD deletion.
func TestCRDDeletion(t *testing.T) {
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

	// Delete CRD
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/crds/invoices.example.io", nil)
	resp, _ = http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Delete failed: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify it's gone
	resp, _ = http.Get(server.URL + "/crds")
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	crds, ok := result["crds"].([]interface{})
	if ok && len(crds) > 0 {
		t.Error("CRD still exists after deletion")
	}
}

// TestCRDValidation tests CRD schema validation.
func TestCRDValidation(t *testing.T) {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	defer eventBus.Close()

	router := NewRouter(registry, scheme, crdRegistry, eventBus)
	router.Setup()
	server := httptest.NewServer(router)
	defer server.Close()

	// Missing required fields
	invalidCRD := `{"group": "example.io"}`
	resp, _ := http.Post(
		server.URL+"/crds",
		"application/json",
		bytes.NewReader([]byte(invalidCRD)),
	)

	// Should fail with 400
	if resp.StatusCode < 400 {
		t.Errorf("Should reject invalid CRD, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestCRDSchema tests schema handling.
func TestCRDSchema(t *testing.T) {
	crdRegistry := NewCRDRegistry()

	schema := map[string]interface{}{
		"id":       "string",
		"amount":   "number",
		"status":   "string",
		"items":    "integer",
	}

	crd := &CRDDefinition{
		Group:   "example.io",
		Version: "v1",
		Kind:    "Invoice",
		Plural:  "invoices",
		Schema:  schema,
	}

	crdRegistry.RegisterCRD(crd)

	retrieved, exists := crdRegistry.GetCRD("invoices.example.io")
	if !exists {
		t.Fatalf("Failed to get CRD")
	}

	if retrieved.Kind != "Invoice" {
		t.Errorf("Kind mismatch: %s", retrieved.Kind)
	}

	if len(retrieved.Schema) != 4 {
		t.Errorf("Schema mismatch: got %d fields", len(retrieved.Schema))
	}
}

// TestCRDRegistryListCRDs tests listing all CRDs.
func TestCRDRegistryListCRDs(t *testing.T) {
	crdRegistry := NewCRDRegistry()

	crds := []struct {
		name string
		kind string
	}{
		{"invoices", "Invoice"},
		{"orders", "Order"},
		{"users", "User"},
	}

	for _, c := range crds {
		crd := &CRDDefinition{
			Group:   "example.io",
			Version: "v1",
			Kind:    c.kind,
			Plural:  c.name,
		}
		crdRegistry.RegisterCRD(crd)
	}

	list := crdRegistry.ListCRDs()
	if len(list) != 3 {
		t.Errorf("Expected 3 CRDs, got %d", len(list))
	}
}

// TestCRDFindByPlural tests finding CRD by plural name.
func TestCRDFindByPlural(t *testing.T) {
	crdRegistry := NewCRDRegistry()

	crd := &CRDDefinition{
		Group:   "example.io",
		Version: "v1",
		Kind:    "Invoice",
		Plural:  "invoices",
	}
	crdRegistry.RegisterCRD(crd)

	found, exists := crdRegistry.FindByPlural("invoices")
	if !exists {
		t.Fatalf("Failed to find CRD")
	}

	if found.Kind != "Invoice" {
		t.Errorf("Wrong CRD found")
	}

	// Try non-existent
	_, exists = crdRegistry.FindByPlural("nonexistent")
	if exists {
		t.Error("Should not find non-existent CRD")
	}
}
