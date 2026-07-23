// pkg/controllers/orders.go
//
// This file implements the OrderController, which watches order events and
// performs reconciliation. The controller demonstrates the reconciliation
// pattern: it reacts to events (Added, Modified, Deleted) and updates the order
// state accordingly. It is a simple example of how business logic can be
// decoupled from HTTP request handling, allowing controllers to operate
// independently of the API server's request lifecycle.


package controllers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/pergus/api-server/pkg/api"
)

// OrderController watches order events and performs reconciliation.
//
// Business Logic:
// - When an order is ADDED: Calculate totals, set status to "processing"
// - When an order is MODIFIED: Log the change
// - When an order is DELETED: Log the deletion
//
// This demonstrates the reconciliation pattern:
// 1. Watch for events
// 2. React to state changes
// 3. Update state (which generates more events)
// 4. Other systems react to your updates
//
// In a real system, this might also:
// - Update inventory
// - Send notifications
// - Trigger payment processing
// - Update analytics
type OrderController struct {
	baseController
	registry api.Registry
}

// NewOrderController creates a new order controller.
// The registry is used to update orders during reconciliation.
func NewOrderController(eventBus api.EventBus, registry api.Registry) *OrderController {
	return &OrderController{
		baseController: baseController{
			name:     "OrderController",
			resource: "orders",
			eventBus: eventBus,
		},
		registry: registry,
	}
}

// Name returns the controller name.
func (oc *OrderController) Name() string {
	return oc.baseController.name
}

// Resource returns the resource this controller watches.
func (oc *OrderController) Resource() string {
	return oc.baseController.resource
}

// Reconcile handles an order event.
// Implements the business logic for order processing.
func (oc *OrderController) Reconcile(event api.Event) error {
	switch event.Type {
	case api.Added:
		return oc.reconcileAdded(event)
	case api.Modified:
		return oc.reconcileModified(event)
	case api.Deleted:
		return oc.reconcileDeleted(event)
	}
	return nil
}

// reconcileAdded handles newly created orders.
// Sets status to "processing" and calculates totals.
func (oc *OrderController) reconcileAdded(event api.Event) error {
	log.Printf("[%s] NEW ORDER - calculating totals and setting status", oc.Name())

	// Parse the order object
	orderData, err := json.Marshal(event.Object)
	if err != nil {
		return err
	}

	var order map[string]interface{}
	if err := json.Unmarshal(orderData, &order); err != nil {
		return err
	}

	// Extract order ID
	id := order["id"].(string)

	// Set status to processing
	order["status"] = "processing"

	// Calculate total if not already set
	// (In a real system, this might sum item prices)
	if _, hasTotal := order["total"]; !hasTotal {
		order["total"] = 0
	}

	log.Printf("[%s] Order %s: status=processing, total=$%.2f", oc.Name(), id, order["total"])

	// Update the order in storage
	// This will generate a MODIFIED event which other watchers will see
	resource, ok := oc.registry.Lookup("orders")
	if !ok {
		log.Printf("[%s] orders resource not found", oc.Name())
		return nil // Resource not registered yet (CRD deletion scenario)
	}

	if err := resource.Storage().Update(id, order); err != nil {
		log.Printf("[%s] error updating order %s: %v", oc.Name(), id, err)
		return nil // Continue processing other events
	}

	log.Printf("[%s] Order %s RECONCILED (status updated)", oc.Name(), id)
	return nil
}

// reconcileModified handles updated orders.
// Logs the modification for debugging.
func (oc *OrderController) reconcileModified(event api.Event) error {
	orderData, _ := json.Marshal(event.Object)
	var order map[string]interface{}
	json.Unmarshal(orderData, &order)

	log.Printf("[%s] Order %s MODIFIED (status=%s)", oc.Name(), order["id"], order["status"])
	return nil
}

// reconcileDeleted handles deleted orders.
// Logs the deletion for debugging.
func (oc *OrderController) reconcileDeleted(event api.Event) error {
	orderData, _ := json.Marshal(event.Object)
	var order map[string]interface{}
	json.Unmarshal(orderData, &order)

	log.Printf("[%s] Order %s DELETED", oc.Name(), order["id"])
	return nil
}

// Run starts the order controller.
// Blocks until context is cancelled.
// Calls reconcile for each event.
func (oc *OrderController) Run(ctx context.Context) error {
	return oc.baseController.runLoop(ctx, oc.Reconcile)
}
