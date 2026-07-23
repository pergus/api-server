// pkg/api/storage.go
//
// This file defines the Storage interface and an in-memory implementation (MemoryStorage).
// The Storage interface abstracts the persistence layer for resources, allowing
// different backends (in-memory, SQL, NoSQL, etc.) to be used interchangeably.
// MemoryStorage is a simple thread-safe in-memory storage that supports basic CRUD operations
// and integrates with the EventBus to publish events on resource changes.

package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Storage defines the persistence interface for all resources.
//
// By depending on this interface rather than a concrete storage backend,
// the framework is agnostic to HOW data is stored. Implementations can be:
// - In-memory (provided here)
// - SQL databases (PostgreSQL, MySQL, etc.)
// - NoSQL databases (MongoDB, DynamoDB, etc.)
// - Cloud storage (S3, Google Cloud Storage, etc.)
// - Distributed systems (etcd, Consul, etc.)
//
// This is identical to how the API server abstracts storage behind StorageInterface.
type Storage interface {
	// List returns all stored objects.
	List() ([]any, error)

	// Get retrieves a single object by its ID.
	Get(id string) (any, error)

	// Create stores a new object.
	// The object should have an "id" field that serves as the unique key.
	Create(obj any) error

	// Update modifies an existing object.
	Update(id string, obj any) error

	// Delete removes an object by ID.
	Delete(id string) error
}

// MemoryStorage is a simple, thread-safe in-memory storage implementation.
//
// All objects are stored in a map protected by a sync.RWMutex.
// This provides basic ACID properties for this example.
//
// For production use, you would replace this with a real database.
//
// Integration with EventBus:
// When an object is created, updated, or deleted, MemoryStorage publishes
// an event to the EventBus. This allows watch clients and controllers
// to react to changes without polling.
type MemoryStorage struct {
	mu       sync.RWMutex
	items    map[string]any
	eventBus EventBus
	resource string
}

// NewMemoryStorage creates a new in-memory storage instance.
func NewMemoryStorage() Storage {
	return &MemoryStorage{
		items:    make(map[string]any),
		eventBus: nil,
		resource: "",
	}
}

// SetEventBus attaches an event bus to this storage.
// Events will be published when objects are created, updated, or deleted.
// This must be called after NewMemoryStorage and before using the storage.
func (s *MemoryStorage) SetEventBus(bus EventBus, resource string) {
	s.eventBus = bus
	s.resource = resource
}

// List returns a copy of all stored items.
func (s *MemoryStorage) List() ([]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]any, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

// Get retrieves an item by ID.
func (s *MemoryStorage) Get(id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	if !exists {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return item, nil
}

// Create stores a new item.
// Expects the item to have an "id" field in its JSON representation.
// After storing, publishes an ADDED event if an event bus is attached.
func (s *MemoryStorage) Create(obj any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := extractID(obj)
	if err != nil {
		return err
	}

	if _, exists := s.items[id]; exists {
		return fmt.Errorf("already exists: %s", id)
	}

	s.items[id] = obj

	// Publish ADDED event if event bus is attached
	if s.eventBus != nil {
		s.eventBus.Publish(Event{
			Type:      Added,
			Resource:  s.resource,
			Object:    obj,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// Update modifies an existing item.
// After updating, publishes a MODIFIED event if an event bus is attached.
func (s *MemoryStorage) Update(id string, obj any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[id]; !exists {
		return fmt.Errorf("not found: %s", id)
	}

	s.items[id] = obj

	// Publish MODIFIED event if event bus is attached
	if s.eventBus != nil {
		s.eventBus.Publish(Event{
			Type:      Modified,
			Resource:  s.resource,
			Object:    obj,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// Delete removes an item by ID.
// Before deleting, publishes a DELETED event if an event bus is attached.
// The event contains the last state of the object.
func (s *MemoryStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, exists := s.items[id]
	if !exists {
		return fmt.Errorf("not found: %s", id)
	}

	delete(s.items, id)

	// Publish DELETED event if event bus is attached
	if s.eventBus != nil {
		s.eventBus.Publish(Event{
			Type:      Deleted,
			Resource:  s.resource,
			Object:    obj,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// extractID pulls the ID from an object by marshalling to JSON.
// This works for any type that has an "id" JSON field.
func extractID(obj any) (string, error) {

	// Marshal the object to its JSON representation.
	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	// Unmarshal into a generic map for dynamic field lookup.
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("unmarshal error: %w", err)
	}

	// Look up the "id" field.
	idVal, exists := m["id"]
	if !exists {
		return "", fmt.Errorf("object missing 'id' field")
	}

	// Convert the ID to its string representation.
	id := fmt.Sprintf("%v", idVal)
	if id == "" || id == "<nil>" {
		return "", fmt.Errorf("id field is empty or nil")
	}

	return id, nil
}
