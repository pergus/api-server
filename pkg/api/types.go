// Package api provides the core framework for a truly dynamic API server
// that can have resources registered and unregistered at runtime without
// rebuilding the HTTP router or restarting the server.
//
// This package demonstrates how Kubernetes achieves extensibility:
// - The router is completely generic and never changes
// - Resources are looked up dynamically on every request
// - New resources are immediately available once registered
// - The registry is thread-safe for concurrent access
//
// Key difference from static servers:
// - No routes are built during startup
// - Every request determines the target resource at runtime
// - The Scheme creates objects by name, not by type
// - Middleware and handlers never know specific resource types
package api

import "time"

// ListResponse wraps a list of objects.
type ListResponse struct {
	Items []any `json:"items"`
	Count int   `json:"count"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// CreatedResponse confirms object creation.
type CreatedResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// UpdatedResponse confirms object update.
type UpdatedResponse struct {
	Message string `json:"message"`
}

// DeletedResponse confirms object deletion.
type DeletedResponse struct {
	Message string `json:"message"`
}

// DiscoveryResponse lists available resources.
type DiscoveryResponse struct {
	Resources []string `json:"resources"`
	Time      string   `json:"timestamp"`
}

// RequestTiming captures request metrics.
type RequestTiming struct {
	StartTime  time.Time
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
}
