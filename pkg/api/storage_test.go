package api

import (
	"testing"
)

// TestMemoryStorageCreate tests creating objects in storage.
func TestMemoryStorageCreate(t *testing.T) {
	storage := NewMemoryStorage()

	obj := map[string]interface{}{"id": "obj-1", "name": "Test"}
	err := storage.Create(obj)

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

// TestMemoryStorageCreateDuplicate tests creating with duplicate ID.
func TestMemoryStorageCreateDuplicate(t *testing.T) {
	storage := NewMemoryStorage()

	obj1 := map[string]interface{}{"id": "obj-1", "name": "First"}
	obj2 := map[string]interface{}{"id": "obj-1", "name": "Second"}

	storage.Create(obj1)
	err := storage.Create(obj2)

	if err == nil {
		t.Fatal("Expected error for duplicate create")
	}
}

// TestMemoryStorageGet tests retrieving objects.
func TestMemoryStorageGet(t *testing.T) {
	storage := NewMemoryStorage()

	original := map[string]interface{}{"id": "obj-1", "name": "Test"}
	storage.Create(original)

	retrieved, err := storage.Get("obj-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Get returned nil")
	}

	m, ok := retrieved.(map[string]interface{})
	if !ok {
		t.Errorf("Retrieved object is %T, expected map", retrieved)
	}

	if m["name"] != "Test" {
		t.Errorf("Retrieved name is %v, expected Test", m["name"])
	}
}

// TestMemoryStorageGetNotFound tests getting non-existent object.
func TestMemoryStorageGetNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	_, err := storage.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent object")
	}
}

// TestMemoryStorageList tests listing all objects.
func TestMemoryStorageList(t *testing.T) {
	storage := NewMemoryStorage()

	objects := []map[string]interface{}{
		{"id": "obj-1", "name": "First"},
		{"id": "obj-2", "name": "Second"},
		{"id": "obj-3", "name": "Third"},
	}

	for _, obj := range objects {
		storage.Create(obj)
	}

	all, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 objects, got %d", len(all))
	}
}

// TestMemoryStorageListEmpty tests listing empty storage.
func TestMemoryStorageListEmpty(t *testing.T) {
	storage := NewMemoryStorage()

	all, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected 0 objects, got %d", len(all))
	}
}

// TestMemoryStorageUpdate tests updating objects.
func TestMemoryStorageUpdate(t *testing.T) {
	storage := NewMemoryStorage()

	original := map[string]interface{}{"id": "obj-1", "name": "Original"}
	storage.Create(original)

	updated := map[string]interface{}{"id": "obj-1", "name": "Updated"}
	err := storage.Update("obj-1", updated)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := storage.Get("obj-1")
	m, _ := retrieved.(map[string]interface{})
	if m["name"] != "Updated" {
		t.Errorf("Updated name is %v, expected Updated", m["name"])
	}
}

// TestMemoryStorageUpdateNotFound tests updating non-existent object.
func TestMemoryStorageUpdateNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	updated := map[string]interface{}{"id": "nonexistent", "name": "Updated"}
	err := storage.Update("nonexistent", updated)
	if err == nil {
		t.Fatal("Expected error for updating non-existent object")
	}
}

// TestMemoryStorageDelete tests deleting objects.
func TestMemoryStorageDelete(t *testing.T) {
	storage := NewMemoryStorage()

	obj := map[string]interface{}{"id": "obj-1", "name": "Test"}
	storage.Create(obj)

	err := storage.Delete("obj-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = storage.Get("obj-1")
	if err == nil {
		t.Fatal("Expected error getting deleted object")
	}
}

// TestMemoryStorageDeleteNotFound tests deleting non-existent object.
func TestMemoryStorageDeleteNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	err := storage.Delete("nonexistent")
	if err == nil {
		t.Fatal("Expected error for deleting non-existent object")
	}
}

// TestMemoryStorageIndependence tests storage independence.
func TestMemoryStorageIndependence(t *testing.T) {
	storage1 := NewMemoryStorage()
	storage2 := NewMemoryStorage()

	storage1.Create(map[string]interface{}{"id": "obj-1", "name": "Storage1"})
	storage2.Create(map[string]interface{}{"id": "obj-1", "name": "Storage2"})

	obj1, _ := storage1.Get("obj-1")
	obj2, _ := storage2.Get("obj-1")

	m1, _ := obj1.(map[string]interface{})
	m2, _ := obj2.(map[string]interface{})

	if m1["name"] == m2["name"] {
		t.Error("Storages are not independent")
	}
}

// TestMemoryStorageConcurrentOperations tests multiple operations.
func TestMemoryStorageConcurrentOperations(t *testing.T) {
	storage := NewMemoryStorage()

	// Create multiple objects
	for i := 1; i <= 5; i++ {
		id := "obj-" + string(rune(48+i))
		storage.Create(map[string]interface{}{"id": id, "index": i})
	}

	// List and verify count
	all, _ := storage.List()
	if len(all) != 5 {
		t.Errorf("Expected 5 objects after creates, got %d", len(all))
	}

	// Update one
	storage.Update("obj-1", map[string]interface{}{"id": "obj-1", "index": 100})

	// Delete one
	storage.Delete("obj-2")

	// Verify final count
	all, _ = storage.List()
	if len(all) != 4 {
		t.Errorf("Expected 4 objects after delete, got %d", len(all))
	}
}
