# Watch API and Controller Architecture

This document explains how the Watch API and Controller Framework enable event-driven extensibility in the dynamic API server.

## Overview

The Watch API and Controller Framework brings event-driven architecture to the dynamic API server:

```
HTTP Request → Generic Handler → Storage → Event Bus → Watch Clients
                                                     ↘ Controllers
```

Every resource operation (create, update, delete) generates an event. Watch clients and controllers receive these events asynchronously without polling.

## Event Model

### Event Types

```go
type EventType string

const (
    Added    EventType = "ADDED"      // Resource created
    Modified EventType = "MODIFIED"   // Resource updated
    Deleted  EventType = "DELETED"    // Resource deleted
)
```

### Event Structure

```go
type Event struct {
    Type      EventType         // What happened
    Resource  string            // Which resource (e.g., "orders")
    Object    any               // The resource object
    Timestamp time.Time         // When it happened
}
```

## Event Bus (EventBus)

The EventBus is the publish-subscribe system that connects storage operations to watchers and controllers.

### Architecture

```
Storage → EventBus → (Fan-out) → Watch Clients
                            ↘    → Controllers
```

### Key Features

- **Thread-safe**: Uses sync.RWMutex for subscriber management
- **Non-blocking publishers**: Events are queued and fanned out in separate goroutines
- **Buffered channels**: Each subscription has a 100-event buffer to handle brief delays
- **Multiple subscribers**: Many watchers/controllers can listen to the same resource simultaneously

### Implementation

```go
// SimpleEventBus runs:
// 1. publishLoop() - reads from publish queue
// 2. fanOut() per event - distributes to all subscribers
// 3. Each subscriber drains its channel in a separate goroutine
```

This ensures:
- A slow subscriber doesn't block publishers
- A slow subscriber doesn't block other subscribers
- Publishers never block

## Storage Integration

### Publishing Events

When storage operations complete, events are published:

```
Create() → Publish ADDED event
Update() → Publish MODIFIED event
Delete() → Publish DELETED event (before removal)
```

The operation completes **before** the event is published, ensuring consistency.

### Flow Diagram

```
POST /api/orders

    ↓

Generic Handler (router.create)

    ↓

Storage.Create(order)

    ↓

Order stored in memory

    ↓

EventBus.Publish(Event{Type: Added, ...})

    ↓

Event queued (non-blocking return)

    ↓

Handler returns HTTP 201

    ↓

EventBus fanOut() runs in background

    ↓

Watch clients receive event

Controllers receive event and reconcile
```

## Watch API

### Endpoints

```
GET /api/{resource}?watch=true     # Watch a resource
```

Example:
```bash
curl "http://localhost:8080/api/orders?watch=true"
```

### Server-Sent Events (SSE)

The watch endpoint uses Server-Sent Events for streaming:

```
event: ADDED
data: {"id":"order-1","customer":"Alice","total":99.99,...}

event: MODIFIED
data: {"id":"order-1","customer":"Alice","total":149.99,...}

event: DELETED
data: {"id":"order-1",...}
```

### How It Works

1. Client sends `GET /api/{resource}?watch=true`
2. Handler checks for `watch=true` query parameter
3. Handler subscribes to the EventBus for that resource
4. For each event received:
   - Serialize as JSON
   - Send as SSE (event type + data)
   - Flush to client
5. Connection stays open until:
   - Client disconnects
   - Server shuts down
   - Resource is deleted

### No Polling

Unlike REST list endpoints that return a snapshot, watch streams **continuous events** as they occur:

- No polling interval
- No missed updates
- No stale data
- Real-time notifications

## Controller Framework

### What is a Controller?

A controller is a business logic processor that:

1. Watches events for a resource
2. Reacts to state changes
3. Updates state (triggering more events)
4. Performs reconciliation

### Controller Interface

```go
type Controller interface {
    Name() string                  // "OrderController"
    Resource() string              // "orders"
    Reconcile(Event) error         // Business logic
    Run(context.Context) error     // Lifecycle
}
```

### Example: OrderController

```
Order created

    ↓

OrderController.Reconcile(ADDED event)

    ↓

Calculate totals and set status to "processing"

    ↓

Call Storage.Update() to persist changes

    ↓

EventBus.Publish(MODIFIED event)

    ↓

Watch clients notified of status change

Other controllers can react to the modified event
```

### Reconciliation Loop

The reconciliation pattern is idempotent - calling it multiple times with the same object should be safe:

```go
func (oc *OrderController) Reconcile(event api.Event) error {
    switch event.Type {
    case api.Added:
        // Calculate totals, set status = "processing"
        // Call Storage.Update() to persist
        
    case api.Modified:
        // Log or react to changes
        
    case api.Deleted:
        // Clean up or archive
    }
    return nil
}
```

## Controller Manager

The ControllerManager:

1. Registers controllers
2. Starts all controllers in separate goroutines
3. Handles subscriptions automatically
4. Manages graceful shutdown

```go
manager := controllers.New(eventBus)
manager.Register(OrderController)
manager.Run(ctx)  // Blocks until ctx cancelled
```

Each controller:
- Runs in its own goroutine
- Subscribes to its resource on startup
- Processes events asynchronously
- Never blocks other controllers

## Complete Event Flow Example

### Scenario: Create Order → Controller Updates → Watch Sees Update

#### Step 1: Client creates order

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{"id":"order-1","customer":"Alice","total":99.99}'
```

#### Step 2: Generic handler processes

```
router.create() called
  ↓
Scheme.New("orders") → empty Order struct
  ↓
Unmarshal JSON into Order
  ↓
Storage.Create(order)
```

#### Step 3: Storage publishes event

```
Storage.Create() completes
  ↓
EventBus.Publish(Event{
    Type: Added,
    Resource: "orders",
    Object: order,
    Timestamp: now,
})
  ↓
Returns to handler immediately
```

#### Step 4: Handler responds

```
HTTP 201 Created
{
    "message": "orders created",
    "id": "order-1"
}
```

*At this point, client has response.*

#### Step 5: EventBus fanOut (background)

```
publishLoop() reads from publish queue
  ↓
Starts fanOut(event) in new goroutine
  ↓
Gets list of subscribers for "orders"
  ↓
Sends event to each:
  - Watch client subscription
  - OrderController subscription
```

#### Step 6: Watch client receives

```
Watch connection streaming...
  ↓
Receives ADDED event
  ↓
Prints to stdout:
  event: ADDED
  data: {"id":"order-1",...}
```

#### Step 7: OrderController receives

```
OrderController.runLoop() waiting on subscription channel
  ↓
Receives ADDED event
  ↓
Calls Reconcile(event)
  ↓
Reconcile() logic:
  - Extract order object
  - Set status = "processing"
  - Call Storage.Update()
```

#### Step 8: Update generates new event

```
Storage.Update() completes
  ↓
EventBus.Publish(Event{
    Type: Modified,
    Resource: "orders",
    Object: order (updated),
    Timestamp: now,
})
```

#### Step 9: Watch client sees update

```
Receives MODIFIED event
  ↓
Prints to stdout:
  event: MODIFIED
  data: {"id":"order-1","status":"processing",...}
```

#### Step 10: Other controllers can react

```
AnyOtherController watching orders
  ↓
Receives MODIFIED event
  ↓
Calls Reconcile()
  ↓
Performs additional business logic
```

## Thread Safety

### EventBus Thread Safety

- **Subscribe/Unsubscribe**: Protected by RWMutex
- **Publish**: Non-blocking, queues event
- **fanOut**: Runs in separate goroutine
- **Subscriber channels**: Buffered (100 events) to prevent blocking

### Storage Thread Safety

- **Create/Update/Delete**: Protected by RWMutex
- **EventBus operations**: Non-blocking (happens after lock release)

### Controller Thread Safety

- Each controller runs in separate goroutine
- No shared state between controllers
- Event delivery is sequential per subscription
- No concurrent calls to Reconcile() for same controller

## CLI Watch Support

### Command

```bash
apictl watch orders
apictl watch users
apictl watch invoices
```

### Output

```
Watching events for orders (Ctrl+C to stop)...

EVENT: ADDED
{
  "id": "order-1",
  "customer": "Alice",
  "total": 99.99,
  ...
}

EVENT: MODIFIED
{
  "id": "order-1",
  "customer": "Alice",
  "total": 149.99,
  "status": "processing"
}

EVENT: DELETED
{
  "id": "order-1",
  ...
}
```

## Why Not Polling?

### Polling Problems

- ❌ Delay between change and observation
- ❌ Server wasted processing list queries
- ❌ Network overhead
- ❌ Scalability issues (N resources = N polls)
- ❌ Stale data windows

### Watch Solution

- ✓ Instant notification
- ✓ No wasted server resources
- ✓ Minimal network overhead
- ✓ Scalable (one connection per watcher)
- ✓ Always current data

## Extending with Controllers

To add a new controller:

1. Implement the Controller interface:
   ```go
   type MyController struct {
       baseController
       registry api.Registry
   }
   
   func (mc *MyController) Reconcile(event api.Event) error {
       // Your business logic
   }
   ```

2. Register it:
   ```go
   manager.Register(MyController{...})
   ```

3. It automatically:
   - Subscribes to events
   - Receives all ADDED/MODIFIED/DELETED events
   - Runs in its own goroutine
   - Receives exclusive delivery (no missed events)

## Production Considerations

### Current Implementation

- In-memory EventBus
- Buffered channels (100 per subscription)
- Simple round-robin event distribution

### For Production

- Persistent event log (etcd, Kafka, etc.)
- Event replay/recovery
- Event filtering and projection
- Metrics and tracing
- Rate limiting and backpressure
- Dead letter queues
- Controller failure handling
- Event ordering guarantees

## Debugging

### Enable Logging

Events are logged to stdout:
```
[OrderController] received ADDED event for orders
[OrderController] Order order-1: status=processing
[OrderController] Order order-1 RECONCILED
```

### Watch Endpoint Curl

```bash
# Watch orders in real-time
curl -N "http://localhost:8080/api/orders?watch=true"

# Watch in separate terminal while creating orders
curl -X POST http://localhost:8080/api/orders \
  -d '{"id":"order-1","customer":"Alice"}'
```

### Check Subscriber Count

Watch endpoint logs:
```
Subscribe: orders (now 1 watchers)
Subscribe: orders (now 2 watchers)
Unsubscribe: orders (now 1 watchers)
```

## Summary

The Watch API and Controller Framework enable:

1. **Real-time notifications** - Clients get instant updates without polling
2. **Event-driven logic** - Controllers react to events instead of being called directly
3. **Decoupled systems** - Storage knows nothing about watchers/controllers
4. **Scalable architecture** - One event triggers many reactions

This is the foundation for building reactive, event-driven systems in the API framework.
