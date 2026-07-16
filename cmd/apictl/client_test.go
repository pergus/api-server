package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClientGetAPIResources tests resource discovery.
func TestClientGetAPIResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"resources": []string{"users", "products", "orders"},
			"timestamp": "2026-07-16T00:00:00Z",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resources, err := client.GetAPIResources()

	if err != nil {
		t.Fatalf("GetAPIResources failed: %v", err)
	}

	if len(resources) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(resources))
	}

	if resources[0] != "users" {
		t.Errorf("Expected 'users', got %s", resources[0])
	}
}

// TestClientGetAPIs tests API group discovery.
func TestClientGetAPIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis" {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"groups": []string{"example.io", "billing.io"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	groups, err := client.GetAPIs()

	if err != nil {
		t.Fatalf("GetAPIs failed: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	if groups[0] != "example.io" {
		t.Errorf("Expected 'example.io', got %s", groups[0])
	}
}

// TestClientListResources tests resource listing.
func TestClientListResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"id": "user-1", "name": "Alice"},
				{"id": "user-2", "name": "Bob"},
			},
			"count": 2,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resources, err := client.ListResources("users")

	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}

// TestClientGetResource tests getting a single resource.
func TestClientGetResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/user-1" {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "user-1",
			"name": "Alice",
			"email": "alice@example.com",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resource, err := client.GetResource("users", "user-1")

	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}

	if resource["id"] != "user-1" {
		t.Errorf("Expected id 'user-1', got %v", resource["id"])
	}
}

// TestClientCreateResource tests creating a resource.
func TestClientCreateResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "users created",
			"id":      "user-3",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	id, err := client.CreateResource("users", map[string]interface{}{
		"id":   "user-3",
		"name": "Charlie",
	})

	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	if id != "user-3" {
		t.Errorf("Expected id 'user-3', got %s", id)
	}
}

// TestClientDeleteResource tests deleting a resource.
func TestClientDeleteResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/user-1" || r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "users deleted",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteResource("users", "user-1")

	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}
}

// TestClientCreateCRD tests CRD creation.
func TestClientCreateCRD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/crds" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "CRD registered",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.CreateCRD(map[string]interface{}{
		"group":   "example.io",
		"version": "v1",
		"kind":    "Invoice",
		"plural":  "invoices",
	})

	if err != nil {
		t.Fatalf("CreateCRD failed: %v", err)
	}
}

// TestClientListCRDs tests CRD listing.
func TestClientListCRDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/crds" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "invoices.example.io", "kind": "Invoice", "plural": "invoices"},
			},
			"count": 1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	crds, err := client.ListCRDs()

	if err != nil {
		t.Fatalf("ListCRDs failed: %v", err)
	}

	if len(crds) != 1 {
		t.Errorf("Expected 1 CRD, got %d", len(crds))
	}
}

// TestClientDeleteCRD tests CRD deletion.
func TestClientDeleteCRD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/crds/invoices.example.io" || r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "CRD unregistered",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteCRD("invoices.example.io")

	if err != nil {
		t.Fatalf("DeleteCRD failed: %v", err)
	}
}

// TestClientServerError tests error handling for server errors.
func TestClientServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetAPIResources()

	if err == nil {
		t.Fatal("Expected error for server error response")
	}
}

// TestClientConnectionError tests handling of connection errors.
func TestClientConnectionError(t *testing.T) {
	client := NewClient("http://invalid-server-that-does-not-exist:9999")

	_, err := client.GetAPIResources()

	if err == nil {
		t.Fatal("Expected error for connection failure")
	}
}
