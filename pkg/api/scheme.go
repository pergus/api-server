// pkg/api/scheme.go
//
// This file defines the Scheme interface and its implementation for managing
// type registration and object creation in the dynamic API server. The Scheme
// allows the API server to create instances of registered types without
// directly importing or knowing about those types, enabling support for
// arbitrary resource types defined at runtime via Custom Resource Definitions
// (CRDs) or plugins.

package api

import (
	"fmt"
	"sync"
)

// ObjectFactory is a function that creates a new, empty instance of an object
// type.
//
// The generic HTTP handlers cannot directly reference types like User or
// Product. Instead, they ask the Scheme to create an empty instance by name.
// This is loaded into and marshalled with incoming JSON.
type ObjectFactory func() any

// Scheme is a type registry and factory.
//
// This maps between:
//   - Type names (strings) -> Constructor functions
//   - This allows the API server to create objects without importing or knowing
//     about types
//
// The Scheme is how the generic handlers avoid importing resource types.
// When a request arrives for /api/users, the handler asks:
//
//	obj, _ := scheme.New("users")
//
// This returns &User{} without the handler knowing anything about User.
//
// The Scheme is thread-safe for registration (which happens at startup or when
// plugins load) and lookups (which happen on every request).
type Scheme interface {
	// Register maps a type name to a factory function.
	// Called during server initialization or plugin loading.
	// Returns error if the type is already registered.
	Register(name string, factory ObjectFactory) error

	// Unregister removes a type factory from the registry.
	// Called when CRDs or plugins are unloaded.
	// Returns error if the type is not registered.
	Unregister(name string) error

	// New creates a new instance of a registered type.
	// Called by generic HTTP handlers to create empty objects for unmarshalling.
	// Returns error if the type is not registered.
	New(name string) (any, error)

	// Has checks if a type is registered.
	Has(name string) bool
}

// ResourceScheme implements the Scheme interface.
type ResourceScheme struct {
	mu        sync.RWMutex
	factories map[string]ObjectFactory
}

// NewScheme creates a new Scheme.
func NewScheme() Scheme {
	return &ResourceScheme{
		factories: make(map[string]ObjectFactory),
	}
}

// Register adds a factory for a type.
// Thread-safe for concurrent registration.
// Called during initialization or when plugins load.
func (s *ResourceScheme) Register(name string, factory ObjectFactory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.factories[name]; exists {
		return fmt.Errorf("type %q already registered", name)
	}

	s.factories[name] = factory
	return nil
}

// Unregister removes a factory for a type.
// Thread-safe for concurrent unregistration.
// Called when CRDs or plugins are unloaded.
func (s *ResourceScheme) Unregister(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.factories[name]; !exists {
		return fmt.Errorf("type %q not registered", name)
	}

	delete(s.factories, name)
	return nil
}

// New creates a new instance of a registered type.
// Thread-safe for concurrent lookups.
// Called on every HTTP request, so read-lock performance matters.
func (s *ResourceScheme) New(name string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	factory, exists := s.factories[name]
	if !exists {
		return nil, fmt.Errorf("unknown type: %q", name)
	}

	return factory(), nil
}

// Has checks if a type is registered.
// Thread-safe for concurrent lookups.
func (s *ResourceScheme) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.factories[name]
	return exists
}
