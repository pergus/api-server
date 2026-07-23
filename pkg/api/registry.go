// pkg/api/registry.go
//
// This file defines the Registry interface and its implementation for managing
// API resources in the dynamic API server. The Registry is responsible for
// keeping track of all known resources, allowing for registration, unregistration,
// lookup, and listing of resources. It is thread-safe and supports concurrent
// access, enabling dynamic resource management at runtime.

package api

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages all known API resources.
//
// This is THE key to extensibility. The registry:
// - Is consulted on every request to determine if a resource exists
// - Supports registration/unregistration at runtime
// - Is thread-safe for concurrent access
// - Never requires HTTP router rebuilding
//
// This is exactly how the API server manages resources dynamically.
// When you define a CRD (Custom Resource Definition), the API server registers it
// in the resource registry. The next request to /api includes it.
type Registry interface {
	// Register adds a resource to the registry.
	// Called by plugins or the main server during initialization.
	// Returns error if a resource with this name already exists.
	Register(resource Resource) error

	// Unregister removes a resource from the registry.
	// Called when plugins are unloaded.
	// Returns error if the resource doesn't exist.
	Unregister(name string) error

	// Lookup retrieves a resource by name.
	// Returns the resource and a boolean indicating if it was found.
	// This is called on every HTTP request to determine which resource to use.
	Lookup(name string) (Resource, bool)

	// List returns all registered resources in sorted order.
	// Used by the discovery endpoint.
	List() []Resource

	// Names returns just the names of all registered resources in sorted order.
	Names() []string

	// Count returns the number of registered resources.
	Count() int
}

// SimpleRegistry implements the Registry interface.
//
// It uses a sync.RWMutex to protect concurrent access.
// This allows:
// - Multiple readers (HTTP requests looking up resources)
// - Single writer (registering/unregistering resources)
// - Safe concurrent access without blocking readers unnecessarily
type SimpleRegistry struct {
	mu        sync.RWMutex
	resources map[string]Resource
}

// NewRegistry creates a new resource registry.
func NewRegistry() Registry {
	return &SimpleRegistry{
		resources: make(map[string]Resource),
	}
}

// Register adds a resource to the registry.
// Thread-safe; blocks write but allows concurrent reads.
func (r *SimpleRegistry) Register(resource Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := resource.Name()
	if _, exists := r.resources[name]; exists {
		return fmt.Errorf("resource %q already registered", name)
	}

	r.resources[name] = resource
	return nil
}

// Unregister removes a resource from the registry.
// Thread-safe; blocks write but allows concurrent reads.
func (r *SimpleRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[name]; !exists {
		return fmt.Errorf("resource %q not found", name)
	}

	delete(r.resources, name)
	return nil
}

// Lookup retrieves a resource by name.
// Thread-safe; allows concurrent reads.
// This is called on EVERY HTTP request, so read-lock performance matters.
func (r *SimpleRegistry) Lookup(name string) (Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resource, exists := r.resources[name]
	return resource, exists
}

// List returns all registered resources in sorted order.
// Thread-safe; allows concurrent reads.
// Used by the discovery endpoint.
func (r *SimpleRegistry) List() []Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]Resource, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}

	// Sort by name for deterministic output
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name() < resources[j].Name()
	})

	return resources
}

// Names returns just the resource names in sorted order.
func (r *SimpleRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.resources))
	for name := range r.resources {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Count returns the number of registered resources.
func (r *SimpleRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.resources)
}
