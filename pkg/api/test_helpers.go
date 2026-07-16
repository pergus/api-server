package api

// Shared test helper types used across multiple test files

type testOrder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type testOrderResource struct {
	storage Storage
}

func (r *testOrderResource) Name() string {
	return "orders"
}

func (r *testOrderResource) NewObject() any {
	return &testOrder{}
}

func (r *testOrderResource) Storage() Storage {
	return r.storage
}
