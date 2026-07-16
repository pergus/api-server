package resources

import (
	"testing"

	"github.com/pergus/api-server/pkg/api"
)

// TestUserResource tests the User resource.
func TestUserResource(t *testing.T) {
	user := NewUserResource()

	if user.Name() != "users" {
		t.Errorf("Expected name 'users', got %s", user.Name())
	}

	// Test creating new object
	obj := user.NewObject()
	if obj == nil {
		t.Fatal("NewObject returned nil")
	}

	// Should be a User
	_, ok := obj.(*User)
	if !ok {
		t.Errorf("NewObject returned %T, expected *User", obj)
	}

	// Test storage
	storage := user.Storage()
	if storage == nil {
		t.Fatal("Storage is nil")
	}
}

// TestProductResource tests the Product resource.
func TestProductResource(t *testing.T) {
	product := NewProductResource()

	if product.Name() != "products" {
		t.Errorf("Expected name 'products', got %s", product.Name())
	}

	obj := product.NewObject()
	_, ok := obj.(*Product)
	if !ok {
		t.Errorf("NewObject returned %T, expected *Product", obj)
	}
}

// TestOrderResource tests the Order resource.
func TestOrderResource(t *testing.T) {
	order := NewOrderResource()

	if order.Name() != "orders" {
		t.Errorf("Expected name 'orders', got %s", order.Name())
	}

	obj := order.NewObject()
	_, ok := obj.(*Order)
	if !ok {
		t.Errorf("NewObject returned %T, expected *Order", obj)
	}
}

// TestUserStorage tests User storage operations.
func TestUserStorage(t *testing.T) {
	user := NewUserResource()
	storage := user.Storage()

	testUser := &User{
		ID:       "user-1",
		Name:     "Alice",
		Email:    "alice@example.com",
		IsActive: true,
	}

	// Create
	err := storage.Create(testUser)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get
	retrieved, err := storage.Get("user-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	retrievedUser := retrieved.(*User)
	if retrievedUser.Name != "Alice" {
		t.Errorf("Name mismatch: expected Alice, got %s", retrievedUser.Name)
	}

	// List
	all, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 1 {
		t.Errorf("Expected 1 user, got %d", len(all))
	}

	// Update
	testUser.IsActive = false
	err = storage.Update("user-1", testUser)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, _ := storage.Get("user-1")
	if updated.(*User).IsActive {
		t.Error("Update didn't work - user still active")
	}

	// Delete
	err = storage.Delete("user-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = storage.Get("user-1")
	if err == nil {
		t.Error("Delete didn't work - user still exists")
	}
}

// TestProductStorage tests Product storage operations.
func TestProductStorage(t *testing.T) {
	product := NewProductResource()
	storage := product.Storage()

	testProduct := &Product{
		ID:          "prod-1",
		Name:        "Widget",
		Description: "A useful widget",
		Price:       9.99,
		StockCount:  100,
	}

	err := storage.Create(testProduct)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	retrieved, err := storage.Get("prod-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	retrievedProduct := retrieved.(*Product)
	if retrievedProduct.Price != 9.99 {
		t.Errorf("Price mismatch: %f", retrievedProduct.Price)
	}
}

// TestOrderStorage tests Order storage operations.
func TestOrderStorage(t *testing.T) {
	order := NewOrderResource()
	storage := order.Storage()

	testOrder := &Order{
		ID:         "order-1",
		UserID:     "user-1",
		Total:      99.99,
		Status:     "draft",
	}

	err := storage.Create(testOrder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	all, _ := storage.List()
	if len(all) != 1 {
		t.Errorf("Expected 1 order, got %d", len(all))
	}
}

// TestMultipleResources tests interop between different resources.
func TestMultipleResources(t *testing.T) {
	users := NewUserResource().Storage()
	products := NewProductResource().Storage()
	orders := NewOrderResource().Storage()

	// Each should be independent
	user := &User{ID: "u1", Name: "Alice", Email: "alice@example.com", IsActive: true}
	product := &Product{ID: "p1", Name: "Widget", Description: "Test", Price: 10, StockCount: 5}
	order := &Order{ID: "o1", UserID: "u1", Total: 10, Status: "draft"}

	users.Create(user)
	products.Create(product)
	orders.Create(order)

	userList, _ := users.List()
	prodList, _ := products.List()
	orderList, _ := orders.List()

	if len(userList) != 1 || len(prodList) != 1 || len(orderList) != 1 {
		t.Error("Resources not independent")
	}
}

// TestResourceInterface tests that all resources implement Resource interface.
func TestResourceInterface(t *testing.T) {
	var _ api.Resource = NewUserResource()
	var _ api.Resource = NewProductResource()
	var _ api.Resource = NewOrderResource()
}
