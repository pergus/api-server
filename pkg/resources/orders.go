// pkg/resources/orders.go
//
// This file defines the Order resource type and its associated Resource implementation.
// The Order resource represents an order in an e-commerce system, with fields for ID,
// user ID, product IDs, status, and total amount. The OrderResource struct implements
// the Resource interface, providing methods to create new Order objects and manage
// their storage using an in-memory storage backend.

package resources

import (
	"github.com/pergus/api-server/pkg/api"
)

// Order is a sample resource type.
type Order struct {
	ID         string   `json:"id"`
	UserID     string   `json:"user_id"`
	ProductIDs []string `json:"product_ids"`
	Status     string   `json:"status"`
	Total      float64  `json:"total"`
}

// OrderResource implements the Resource interface.
type OrderResource struct {
	storage api.Storage
}

// NewOrderResource creates a new order resource.
func NewOrderResource() *OrderResource {
	return &OrderResource{
		storage: api.NewMemoryStorage(),
	}
}

// Name returns "orders".
func (r *OrderResource) Name() string {
	return "orders"
}

// NewObject returns an empty Order.
func (r *OrderResource) NewObject() any {
	return &Order{}
}

// Storage returns the storage implementation.
func (r *OrderResource) Storage() api.Storage {
	return r.storage
}
