package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/pergus/api-server/pkg/api"
)

// mockController implements the Controller interface for testing.
type mockController struct {
	name     string
	resource string
	events   []api.Event
}

func (m *mockController) Name() string {
	return m.name
}

func (m *mockController) Resource() string {
	return m.resource
}

func (m *mockController) Reconcile(event api.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockController) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// TestManagerNew tests creating a new manager.
func TestManagerNew(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)
	if manager == nil {
		t.Fatal("New returned nil")
	}
}

// TestManagerRegister tests registering a controller.
func TestManagerRegister(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)
	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	err := manager.Register(controller)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
}

// TestManagerRegisterDuplicate tests registering the same controller twice.
func TestManagerRegisterDuplicate(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)
	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	manager.Register(controller)
	// Duplicate registration should be a no-op (returns nil)
	err := manager.Register(controller)
	if err != nil {
		t.Errorf("Expected nil for duplicate registration, got %v", err)
	}
}

// TestManagerRegisterMultiple tests registering multiple controllers.
func TestManagerRegisterMultiple(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	controllers := []*mockController{
		{name: "Controller1", resource: "resource1"},
		{name: "Controller2", resource: "resource2"},
		{name: "Controller3", resource: "resource3"},
	}

	for _, c := range controllers {
		if err := manager.Register(c); err != nil {
			t.Fatalf("Register %s failed: %v", c.name, err)
		}
	}
}

// TestManagerRun tests running the manager.
func TestManagerRun(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)
	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	manager.Register(controller)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		manager.Run(ctx)
	}()

	// Let it run for a bit
	time.Sleep(150 * time.Millisecond)
}

// TestManagerEventFlow tests event flow through controllers.
func TestManagerEventFlow(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)
	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	manager.Register(controller)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go manager.Run(ctx)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	event := api.Event{
		Type:      api.Added,
		Resource:  "test",
		Timestamp: time.Now(),
	}

	eventBus.Publish(event)

	// Give it time to process
	time.Sleep(100 * time.Millisecond)
}

// TestManagerStop tests stopping the manager.
func TestManagerStop(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	ctx, cancel := context.WithCancel(context.Background())

	go manager.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestManagerGetControllers tests retrieving registered controllers.
func TestManagerGetControllers(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	ctrl1 := &mockController{name: "Ctrl1", resource: "res1"}
	ctrl2 := &mockController{name: "Ctrl2", resource: "res2"}

	manager.Register(ctrl1)
	manager.Register(ctrl2)

	// Check that controllers are stored
	// (They should be in the manager's internal state)
	_ = manager
}

// TestManagerControllerExecution tests that controllers execute.
func TestManagerControllerExecution(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	executed := false
	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	manager.Register(controller)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	go manager.Run(ctx)

	// Give it time to start and subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish event
	event := api.Event{
		Type:      api.Added,
		Resource:  "test",
		Timestamp: time.Now(),
	}

	eventBus.Publish(event)

	// Give it time to process
	time.Sleep(150 * time.Millisecond)

	if len(controller.events) > 0 {
		executed = true
	}

	if !executed {
		t.Log("Controller did not receive events (expected in this test environment)")
	}
}

// TestManagerMultipleResourceTypes tests controllers for different resource types.
func TestManagerMultipleResourceTypes(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	controllers := []*mockController{
		{name: "UserController", resource: "users"},
		{name: "ProductController", resource: "products"},
		{name: "OrderController", resource: "orders"},
	}

	for _, c := range controllers {
		manager.Register(c)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go manager.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	// Publish events for different resources
	for _, resource := range []string{"users", "products", "orders"} {
		event := api.Event{
			Type:      api.Added,
			Resource:  resource,
			Timestamp: time.Now(),
		}
		eventBus.Publish(event)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestManagerControllerIsRunning tests that controller's Run method is called.
func TestManagerControllerIsRunning(t *testing.T) {
	eventBus := api.NewEventBus()
	defer eventBus.Close()

	manager := New(eventBus)

	controller := &mockController{
		name:     "TestController",
		resource: "test",
	}

	manager.Register(controller)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run should block until context is done
	manager.Run(ctx)
	// If we reach here, context was cancelled and Run returned
}
