// pkg/controllers/controller.go
//
// This file defines the Controller interface, which represents a reconciliation controller
// in the dynamic API framework. Controllers are responsible for watching events related to
// specific resources and performing business logic to reconcile the current state with the
// desired state. This decouples business logic from HTTP request handling, allowing controllers
// to operate independently of the API server's request lifecycle.

package controllers

import (
	"context"

	"github.com/pergus/api-server/pkg/api"
)

// Controller is the interface for a reconciliation controller.
//
// Controllers are the "brain" of the system - they watch events and
// perform business logic in response.
//
// Example controllers:
// - OrderController reconciles orders (e.g., calculate totals, update status)
// - Future: InvoiceController, UserController, etc.
//
// The key insight: Controllers do NOT respond to HTTP requests.
// They respond to events. This decouples business logic from API handling.
type Controller interface {
	// Name returns the name of this controller.
	// Used for logging and debugging.
	Name() string

	// Resource returns the name of the resource this controller watches.
	// (e.g., "orders", "invoices", "users")
	Resource() string

	// Reconcile processes an event and performs reconciliation.
	// Called when a resource is Added, Modified, or Deleted.
	//
	// Reconciliation is the process of observing the current state
	// and taking action to achieve the desired state.
	//
	// Example: When an Order is created, the controller might:
	// 1. Calculate totals
	// 2. Update order status to "processing"
	// 3. Call storage.Update() which generates another event
	// 4. Other systems react to the modified event
	//
	// Reconcile should be idempotent - calling it multiple times
	// with the same object should be safe.
	Reconcile(event api.Event) error

	// Run starts the controller.
	// Should block until context is cancelled.
	Run(ctx context.Context) error
}
