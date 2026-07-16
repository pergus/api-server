# Writing a Controller Guide

This guide explains how to write a controller for the dynamic API server. Controllers react to resource events and perform automated actions (reconciliation).

## What is a Controller?

A controller is a background process that:
- Watches for events on a specific resource (ADDED, MODIFIED, DELETED)
- Reacts to those events
- May update resources (generating more events)
- Implements business logic without user interaction

Controllers enable reactive, event-driven architecture where systems automatically respond to state changes.

## Controller Lifecycle

1. **Register** - Tell manager about the controller
2. **Start** - Manager launches controller in a goroutine
3. **Subscribe** - Controller subscribes to resource events
4. **Watch** - Controller waits for events
5. **Reconcile** - When event arrives, controller processes it
6. **React** - Controller may update state (generates more events)
7. **Stop** - Manager cancels context, controller exits

## Controller Interface

Every controller must implement `controllers.Controller`:

```go
type Controller interface {
    // Name returns the controller identifier
    Name() string
    
    // Resource returns which resource to watch
    Resource() string
    
    // Reconcile handles a single event
    Reconcile(event api.Event) error
    
    // Run starts the controller
    // Should block until context is cancelled
    Run(ctx context.Context) error
}
```

## Step-by-Step Example: Order Controller

### 1. Define Your Event Model

Think about what events your resource will have:

```
ORDER Lifecycle:
  - ADDED: New order created
  - MODIFIED: Order status changed
  - DELETED: Order cancelled
```

### 2. Create the Controller Type

```go
type OrderController struct {
    baseController  // Provides runLoop helper
    registry        api.Registry  // Access other resources
}

// NewOrderController creates a new controller
func NewOrderController(eventBus api.EventBus, registry api.Registry) *OrderController {
    return &OrderController{
        baseController: baseController{
            name:     "OrderController",
            resource: "orders",  // Watch "orders" resource
            eventBus: eventBus,
        },
        registry: registry,
    }
}
```

### 3. Implement Interface Methods

```go
// Name returns the controller identifier
func (oc *OrderController) Name() string {
    return oc.baseController.name
}

// Resource returns which resource to watch
func (oc *OrderController) Resource() string {
    return oc.baseController.resource
}

// Run starts the controller
// The baseController.runLoop handles subscription and event loop
func (oc *OrderController) Run(ctx context.Context) error {
    return oc.baseController.runLoop(ctx, oc.Reconcile)
}
```

### 4. Implement Reconciliation Logic

```go
// Reconcile handles an event
// Called by the event loop for each event
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

// reconcileAdded handles newly created orders
func (oc *OrderController) reconcileAdded(event api.Event) error {
    log.Printf("[%s] NEW ORDER", oc.Name())
    
    // Parse the object
    orderData, err := json.Marshal(event.Object)
    if err != nil {
        return err
    }
    
    var order map[string]interface{}
    if err := json.Unmarshal(orderData, &order); err != nil {
        return err
    }
    
    id := order["id"].(string)
    
    // Business logic: set status to processing
    order["status"] = "processing"
    
    // Update the order (generates MODIFIED event)
    resource, ok := oc.registry.Lookup("orders")
    if !ok {
        return nil  // Resource not available
    }
    
    if err := resource.Storage().Update(id, order); err != nil {
        log.Printf("[%s] Error updating %s: %v", oc.Name(), id, err)
        return nil  // Continue processing other events
    }
    
    log.Printf("[%s] Order %s reconciled", oc.Name(), id)
    return nil
}

// reconcileModified handles updated orders
func (oc *OrderController) reconcileModified(event api.Event) error {
    log.Printf("[%s] Order modified", oc.Name())
    // Log, audit, or trigger other actions
    return nil
}

// reconcileDeleted handles deleted orders
func (oc *OrderController) reconcileDeleted(event api.Event) error {
    log.Printf("[%s] Order deleted", oc.Name())
    // Cleanup, archive, or trigger other actions
    return nil
}
```

## Complete Controller Example

```go
package controllers

import (
    "context"
    "encoding/json"
    "log"
    "github.com/pergus/api-server/pkg/api"
)

type OrderController struct {
    baseController
    registry api.Registry
}

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

func (oc *OrderController) Name() string {
    return oc.baseController.name
}

func (oc *OrderController) Resource() string {
    return oc.baseController.resource
}

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

func (oc *OrderController) reconcileAdded(event api.Event) error {
    log.Printf("[%s] NEW ORDER", oc.Name())
    
    orderData, err := json.Marshal(event.Object)
    if err != nil {
        return err
    }
    
    var order map[string]interface{}
    if err := json.Unmarshal(orderData, &order); err != nil {
        return err
    }
    
    id := order["id"].(string)
    order["status"] = "processing"
    
    resource, ok := oc.registry.Lookup("orders")
    if !ok {
        return nil
    }
    
    if err := resource.Storage().Update(id, order); err != nil {
        log.Printf("[%s] Error: %v", oc.Name(), err)
        return nil
    }
    
    log.Printf("[%s] Order %s reconciled", oc.Name(), id)
    return nil
}

func (oc *OrderController) reconcileModified(event api.Event) error {
    log.Printf("[%s] Order modified", oc.Name())
    return nil
}

func (oc *OrderController) reconcileDeleted(event api.Event) error {
    log.Printf("[%s] Order deleted", oc.Name())
    return nil
}

func (oc *OrderController) Run(ctx context.Context) error {
    return oc.baseController.runLoop(ctx, oc.Reconcile)
}
```

## Registering Controllers

In `cmd/api-server/main.go`:

```go
import "github.com/pergus/api-server/pkg/controllers"

// In main():
manager := controllers.New(server.EventBus())

// Register your controller
if err := manager.Register(controllers.NewOrderController(
    server.EventBus(),
    server.Registry(),
)); err != nil {
    log.Fatalf("Failed to register controller: %v", err)
}

// Start the manager (runs all controllers concurrently)
go func() {
    ctx := context.Background()
    if err := manager.Run(ctx); err != nil {
        log.Printf("Controller manager error: %v", err)
    }
}()
```

## Event Model

Events have this structure:

```go
type Event struct {
    Type      EventType     // Added, Modified, or Deleted
    Resource  string        // "orders", "users", etc.
    Object    any           // The actual object
    Timestamp time.Time     // When it happened
}

type EventType string

const (
    Added    EventType = "ADDED"
    Modified EventType = "MODIFIED"
    Deleted  EventType = "DELETED"
)
```

### Event Flow

```
User creates order
        ↓
POST /api/orders → 201 Created
        ↓
Storage generates ADDED event
        ↓
EventBus publishes to subscribers
        ↓
OrderController receives ADDED event
        ↓
reconcileAdded() updates status to "processing"
        ↓
Update generates MODIFIED event
        ↓
EventBus publishes MODIFIED
        ↓
Other subscribers see the MODIFIED event
        ↓
Cycle continues...
```

## Common Patterns

### Simple Logging Controller

```go
type LoggingController struct {
    baseController
}

func (lc *LoggingController) Reconcile(event api.Event) error {
    log.Printf("[%s] %s: %v", lc.Name(), event.Type, event.Object)
    return nil
}
```

### Audit Trail Controller

```go
type AuditController struct {
    baseController
    auditLog []AuditEntry
}

type AuditEntry struct {
    Timestamp time.Time
    Type      string
    Resource  string
    Object    interface{}
}

func (ac *AuditController) Reconcile(event api.Event) error {
    ac.auditLog = append(ac.auditLog, AuditEntry{
        Timestamp: event.Timestamp,
        Type:      string(event.Type),
        Resource:  event.Resource,
        Object:    event.Object,
    })
    return nil
}
```

### Multi-Resource Controller

```go
type SyncController struct {
    baseController
    registry api.Registry
}

func NewSyncController(eventBus api.EventBus, registry api.Registry) *SyncController {
    return &SyncController{
        baseController: baseController{
            name:     "SyncController",
            resource: "orders",  // Still watch one resource
            eventBus: eventBus,
        },
        registry: registry,
    }
}

func (sc *SyncController) Reconcile(event api.Event) error {
    // When order changes, update related resources
    if event.Type == api.Modified {
        // Update inventory
        inventory, _ := sc.registry.Lookup("inventory")
        // ... update logic
        
        // Trigger billing
        billing, _ := sc.registry.Lookup("billing")
        // ... billing logic
    }
    return nil
}
```

### With Configuration

```go
type ConfigurableController struct {
    baseController
    config ControllerConfig
}

type ControllerConfig struct {
    ProcessingTimeout time.Duration
    RetryCount        int
    AlertEmail        string
}

func NewOrderController(eventBus api.EventBus, config ControllerConfig) *OrderController {
    return &OrderController{
        baseController: baseController{...},
        config: config,
    }
}
```

## Error Handling

### Return nil - Continue Processing

```go
func (oc *OrderController) Reconcile(event api.Event) error {
    resource, ok := oc.registry.Lookup("orders")
    if !ok {
        // Resource temporarily unavailable, but don't fail
        return nil
    }
    // ...
    return nil  // Success
}
```

### Return error - Log but Continue

```go
func (oc *OrderController) Reconcile(event api.Event) error {
    if err := someOperation(event); err != nil {
        log.Printf("[%s] Warning: %v", oc.Name(), err)
        return nil  // Continue, don't block
    }
    return nil
}
```

### Never Block

Controllers run in a shared event loop. Don't:
- Sleep for long periods
- Make blocking I/O without timeouts
- Call expensive operations synchronously

Instead:
- Use goroutines for heavy work
- Queue work for async processing
- Keep Reconcile fast (milliseconds)

## Testing Controllers

```go
func TestOrderController(t *testing.T) {
    // Create mock event bus and registry
    eventBus := api.NewEventBus()
    defer eventBus.Close()
    
    registry := api.NewRegistry()
    storage := api.NewMemoryStorage()
    
    resource := &OrderResource{storage: storage}
    registry.Register(resource)
    
    // Create controller
    controller := NewOrderController(eventBus, registry)
    
    // Create test event
    event := api.Event{
        Type:      api.Added,
        Resource:  "orders",
        Object:    Order{ID: "1", Status: "draft"},
        Timestamp: time.Now(),
    }
    
    // Reconcile
    err := controller.Reconcile(event)
    if err != nil {
        t.Fatalf("Reconcile failed: %v", err)
    }
    
    // Verify results
    order, _ := storage.Get("1")
    // Assert order.Status == "processing"
}
```

## Troubleshooting

### Controller Never Sees Events

**Check:**
1. Resource name matches exactly (case-sensitive)
2. Controller registered before manager starts
3. Events are actually being published
4. Event bus is not closed

**Debug:**
```go
// Add logging
log.Printf("[%s] Subscribed to %s", oc.Name(), oc.Resource())
log.Printf("[%s] Received event: %v", oc.Name(), event)
```

### Controller Blocks Event Loop

**Problem:** Reconcile is slow and blocks other controllers

**Solution:** Spawn goroutine for heavy work:
```go
func (oc *OrderController) Reconcile(event api.Event) error {
    go oc.slowOperation(event)  // Non-blocking
    return nil
}
```

### State Not Updated

**Problem:** Controller updates don't appear to take effect

**Check:**
1. You're updating via registry.Lookup().Storage().Update()
2. Update returns nil (no error)
3. Storage actually implements Update

## Best Practices

1. **Keep Reconcile Fast** - Aim for <100ms per event
2. **Don't Assume State** - Always look up current state
3. **Handle Errors Gracefully** - Log but don't crash
4. **Test Idempotence** - Same event should produce same result
5. **Use Logging** - Makes debugging event flows easy
6. **Document Business Logic** - Explain why you reconcile this way
7. **Consider Event Order** - Events arrive in order per resource
8. **Avoid Infinite Loops** - Your update shouldn't trigger itself

## Comparing Controllers and Plugins

| Aspect | Controller | Plugin |
|--------|-----------|--------|
| Purpose | React to events | Define resources |
| Timing | Always running | Only when loaded |
| Trigger | Automatic (events) | User action (API call) |
| State | Can change resources | Provides storage |
| Scope | Multiple resources | Own resource |
| Logic | Business rules | Resource definition |

Use **controllers** for: automation, orchestration, reactions
Use **plugins** for: custom resources, storage backends

## Next Steps

1. Create your controller type
2. Implement the Controller interface
3. Write your Reconcile logic
4. Register in main.go
5. Test with events
6. Monitor logs to verify behavior

See the example controller at `pkg/controllers/orders.go` for a complete working implementation.

## Resources

- [Event System Architecture](WATCH_ARCHITECTURE.md)
- [Plugin Guide](PLUGIN_GUIDE.md)
- [API Reference](FILES_REFERENCE.md)
