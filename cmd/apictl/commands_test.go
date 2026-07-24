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
