package api

// Resource defines the interface that all API resources must implement.
//
// This is the contract between the generic framework and concrete resources.
// The framework never knows about specific types like User or Product—it only
// knows about resources through this interface.
//
// This design is how Kubernetes allows arbitrary CRDs (Custom Resource Definitions)
// to be added at runtime without changing the core API server code.
type Resource interface {
	// Name returns the singular name of this resource.
	// Used as the path component: /api/{name}
	// Examples: "users", "products", "orders", "invoices"
	Name() string

	// NewObject returns a new, zero-value instance of this resource type.
	// Called by generic handlers to create empty objects for JSON unmarshalling.
	// The handler then decodes incoming JSON into this object.
	//
	// Example: returns &User{} for users, &Product{} for products
	NewObject() any

	// Storage returns the persistence layer for this resource.
	// Each resource has its own storage instance.
	// Implementations might use memory, a database, cloud storage, etc.
	Storage() Storage
}
