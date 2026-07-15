package resources

import (
	"github.com/pergus/api-server/pkg/api"
)

// Product is a sample resource type.
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	StockCount  int     `json:"stock_count"`
}

// ProductResource implements the Resource interface.
type ProductResource struct {
	storage api.Storage
}

// NewProductResource creates a new product resource.
func NewProductResource() *ProductResource {
	return &ProductResource{
		storage: api.NewMemoryStorage(),
	}
}

// Name returns "products".
func (r *ProductResource) Name() string {
	return "products"
}

// NewObject returns an empty Product.
func (r *ProductResource) NewObject() any {
	return &Product{}
}

// Storage returns the storage implementation.
func (r *ProductResource) Storage() api.Storage {
	return r.storage
}
