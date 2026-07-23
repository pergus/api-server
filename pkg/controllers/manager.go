// pkg/controllers/manager.go
//
// This file implements the ControllerManager, which manages the lifecycle of
// multiple controllers. The manager is responsible for registering controllers,
// starting them in separate goroutines, and handling subscriptions to the event
// bus. It ensures that controllers run concurrently, events are delivered
// asynchronously, and each controller sees all events for its resource. The
// manager also supports graceful shutdown of all controllers.

package controllers

import (
	"context"
	"log"
	"sync"

	"github.com/pergus/api-server/pkg/api"
)

// ControllerManager manages the lifecycle of multiple controllers.
//
// Responsibilities:
// 1. Register controllers
// 2. Start controllers in separate goroutines
// 3. Handle subscriptions to the event bus
// 4. Graceful shutdown
//
// The manager ensures:
// - Controllers run concurrently
// - Events are delivered asynchronously (non-blocking)
// - Each controller sees all events for its resource
// - Clean shutdown without lost events
type ControllerManager struct {
	controllers map[string]Controller
	eventBus    api.EventBus
	mu          sync.RWMutex
	subs        map[string]api.Subscription
}

// New creates a new controller manager.
func New(eventBus api.EventBus) *ControllerManager {
	return &ControllerManager{
		controllers: make(map[string]Controller),
		eventBus:    eventBus,
		subs:        make(map[string]api.Subscription),
	}
}

// Register registers a new controller.
// Can be called before Run().
func (cm *ControllerManager) Register(controller Controller) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.controllers[controller.Name()]; exists {
		return nil // Already registered
	}

	cm.controllers[controller.Name()] = controller
	log.Printf("Registered controller: %s", controller.Name())
	return nil
}

// Run starts all registered controllers.
// Blocks until context is cancelled.
// Each controller runs in its own goroutine.
func (cm *ControllerManager) Run(ctx context.Context) error {
	cm.mu.Lock()
	controllers := make([]Controller, 0, len(cm.controllers))
	for _, c := range cm.controllers {
		controllers = append(controllers, c)
	}
	cm.mu.Unlock()

	// Start all controllers
	wg := &sync.WaitGroup{}
	for _, controller := range controllers {
		wg.Add(1)
		go func(c Controller) {
			defer wg.Done()
			log.Printf("Starting controller: %s", c.Name())
			if err := c.Run(ctx); err != nil {
				log.Printf("Controller %s error: %v", c.Name(), err)
			}
			log.Printf("Stopped controller: %s", c.Name())
		}(controller)
	}

	// Wait for all controllers to finish
	wg.Wait()
	return nil
}

// baseController is a helper that implements common controller logic.
//
// Most controllers follow this pattern:
// 1. Subscribe to events for their resource
// 2. Process events in a loop
// 3. Unsubscribe on shutdown
//
// This base class handles steps 1 and 3, allowing concrete controllers
// to focus on step 2 (the Reconcile logic).
type baseController struct {
	name     string
	resource string
	eventBus api.EventBus
}

// runLoop runs the main event processing loop.
// Calls reconcile() for each event.
func (bc *baseController) runLoop(ctx context.Context, reconcile func(event api.Event) error) error {
	// Subscribe to events
	sub := bc.eventBus.Subscribe(bc.resource)
	defer bc.eventBus.Unsubscribe(sub)

	log.Printf("[%s] subscribed to %s events", bc.name, bc.resource)

	// Process events until context is cancelled
	for {
		select {
		case event := <-sub.Events:
			log.Printf("[%s] received %s event for %s", bc.name, event.Type, event.Resource)
			if err := reconcile(event); err != nil {
				log.Printf("[%s] reconcile error: %v", bc.name, err)
			}

		case <-ctx.Done():
			log.Printf("[%s] context cancelled, stopping", bc.name)
			return nil
		}
	}
}
