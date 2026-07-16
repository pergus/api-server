package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCmdGetList tests getting a list of resources.
func TestCmdGetList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/users" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "user-1", "name": "Alice"},
					{"id": "user-2", "name": "Bob"},
				},
				"count": 2,
			})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	items, err := client.ListResources("users")

	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

// TestCmdGetSingle tests getting a single resource.
func TestCmdGetSingle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/users/user-1" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "user-1",
				"name": "Alice",
			})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resource, err := client.GetResource("users", "user-1")

	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}

	if resource["name"] != "Alice" {
		t.Errorf("Expected 'Alice', got %v", resource["name"])
	}
}

// TestExtractIDFromObject tests the extractID helper function.
func TestExtractIDFromObject(t *testing.T) {
	obj := map[string]interface{}{
		"id":   "test-123",
		"name": "Test Object",
	}

	id := extractID(obj)
	if id != "test-123" {
		t.Errorf("extractID returned %s, expected test-123", id)
	}
}

// TestExtractIDFromMetadata tests extractID from metadata.
func TestExtractIDFromMetadata(t *testing.T) {
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "my-object",
		},
	}

	id := extractID(obj)
	if id != "my-object" {
		t.Errorf("extractID from metadata returned %s, expected my-object", id)
	}
}

// TestExtractIDMissing tests extractID with missing id field.
func TestExtractIDMissing(t *testing.T) {
	obj := map[string]interface{}{
		"name": "Test Object",
	}

	id := extractID(obj)
	if id != "unknown" {
		t.Errorf("extractID with no id field should return 'unknown', got '%s'", id)
	}
}

// TestPluralizeHelper tests the pluralize helper function.
func TestPluralizeHelper(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"Product", "products"},
		{"Order", "orders"},
		{"Invoice", "invoices"},
	}

	for _, tt := range tests {
		result := pluralize(tt.input)
		if result != tt.expected {
			t.Errorf("pluralize(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// TestExtractIDHelper tests the extractID helper function.
func TestExtractIDHelper(t *testing.T) {
	obj := map[string]interface{}{
		"id":   "test-123",
		"name": "Test Object",
	}

	id := extractID(obj)
	if id != "test-123" {
		t.Errorf("extractID returned %s, expected test-123", id)
	}
}

