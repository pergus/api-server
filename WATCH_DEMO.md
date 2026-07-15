# Watch API and Controller Demo

This walkthrough demonstrates the Watch API and Controller Framework in action.

## Prerequisites

Build the binaries:
```bash
make build
```

Or manually:
```bash
go build -o api-server ./cmd/api-server
go build -o apictl ./cmd/apictl
```

## Complete Demonstration Sequence

### Part 1: Start the Server

**Terminal 1 - Start the server:**
```bash
./api-server
```

Expected output:
```
Registering built-in resources...
Registered resource: users
Registered resource: products
Registered resource: orders
Starting plugin system...
Scanning for existing plugins...
Setting up routes (generic, never change)
Registered resources: 3
  - orders
  - products
  - users
Initializing controller manager...
Registered controller: OrderController
Starting controller on OrderController
Starting server on http://localhost:8080
Discovery: GET http://localhost:8080/api
```

Notice:
- OrderController is registered and running
- It's waiting for events on the orders resource
- Server is ready to accept connections

### Part 2: Watch Orders (Real-time Events)

**Terminal 2 - Start watching orders:**
```bash
./apictl watch orders
```

Expected output:
```
Watching events for orders (Ctrl+C to stop)...

(Connection stays open, waiting for events)
```

The connection is now open and streaming. Every time an order is created, modified, or deleted, the OrderController and this watch client will both receive events.

### Part 3: Create an Order (in another terminal)

**Terminal 3 - Create an order:**
```bash
./apictl create -f examples/order-1.json
```

Expected output:
```
orders created: order-001
```

### Observe the Results

#### Terminal 1 (Server logs):

```
[OrderController] subscribed to orders events
[OrderController] received ADDED event for orders
[OrderController] NEW ORDER - calculating totals and setting status
[OrderController] Order order-001: status=processing, total=$99.99
[OrderController] Order order-001 RECONCILED (status updated)
[OrderController] received MODIFIED event for orders
[OrderController] Order order-001 MODIFIED (status=processing)
```

Notice the flow:
1. Controller receives ADDED event
2. Controller reconciles (calculates totals, sets status)
3. Controller calls Storage.Update()
4. Storage publishes MODIFIED event
5. Controller receives and logs the MODIFIED event

#### Terminal 2 (Watch stream):

```
EVENT: ADDED
{
  "id": "order-001",
  "kind": "Order",
  "customer_id": "alice",
  "total": 99.99,
  "status": "draft",
  "created_at": "2026-07-15T10:30:00Z"
}

EVENT: MODIFIED
{
  "id": "order-001",
  "kind": "Order",
  "customer_id": "alice",
  "total": 99.99,
  "status": "processing",
  "created_at": "2026-07-15T10:30:00Z"
}
```

Notice:
1. First event is ADDED with status="draft"
2. Second event is MODIFIED with status="processing" (set by controller)
3. Both events appear instantly without polling
4. The watch client sees ALL changes in real-time

### Part 4: Continue Watching

Create more orders while the watch is running:

**Terminal 3:**
```bash
./apictl create -f examples/order-1.json
./apictl create -f examples/order-2.json
./apictl create -f examples/order-3.json
```

**Terminal 2 (watch):**
Each order will generate:
1. ADDED event (initial creation)
2. MODIFIED event (controller processes)

### Part 5: Delete an Order

**Terminal 3:**
```bash
./apictl delete orders order-001
```

**Terminal 2 (watch):**
```
EVENT: DELETED
{
  "id": "order-001",
  "kind": "Order",
  "customer_id": "alice",
  "total": 99.99,
  "status": "processing",
  "created_at": "2026-07-15T10:30:00Z"
}
```

**Terminal 1 (server):**
```
[OrderController] received DELETED event for orders
[OrderController] Order order-001 DELETED
```

### Part 6: Stop the Watch

Press Ctrl+C in Terminal 2 to disconnect from the watch stream:
```
^C
```

Expected output:
```
Watching events for orders (Ctrl+C to stop)...

EVENT: ADDED
...

(disconnects cleanly)
```

The server continues running, and other watch clients would still receive events.

## Detailed Event Flow Walkthrough

When you execute `./apictl create -f examples/order-1.json`, here's what happens:

### Step 1: Client → Server

```
POST /api/orders
Content-Type: application/json

{
  "id": "order-001",
  "kind": "Order",
  ...
}
```

### Step 2: Generic Handler

```
Router.route()
  → routeListOrCreate() 
    → POST method → create()
      → Scheme.New("orders") 
        → creates empty Order{}
      → json.Unmarshal(body, &order)
        → populates Order fields
      → Storage.Create(order)
        → writes to memory
        → publishes ADDED event ← THIS IS KEY
      → returns HTTP 201
```

**This all completes in <1ms**

### Step 3: Event Published (Non-blocking)

```
EventBus.Publish(Event{
  Type: Added,
  Resource: "orders",
  Object: order,
  Timestamp: now,
})
```

The handler returns immediately. Publishing happens in background.

### Step 4: EventBus FanOut

```
publishLoop() reads from queue
  → starts fanOut(event) goroutine
    → gets subscribers: [WatchClient, OrderController]
    → sends to both subscriptions
```

Both the watch client and controller receive the event **simultaneously**.

### Step 5: Watch Client

```
Watch channel reads event
  → prints "EVENT: ADDED"
  → unmarshals JSON
  → pretty-prints object
  → ready for next event
```

**Instant display in Terminal 2**

### Step 6: OrderController

```
Controller.runLoop() reads event
  → calls Reconcile(Added event)
    → extracts order data
    → sets status = "processing"
    → calls Storage.Update(id, order)
      → writes updated order
      → publishes MODIFIED event
```

### Step 7: Second Event Cycle

```
MODIFIED event published
  → WatchClient receives it
    → displays "EVENT: MODIFIED"
  → OrderController receives it
    → calls Reconcile(Modified event)
      → logs the change
```

**Total latency: <5ms from POST to watch receiving the event**

## REST API Examples

You can also use curl directly instead of apictl:

### Watch via curl

```bash
# Stream events as curl
curl -N "http://localhost:8080/api/orders?watch=true"

# -N flag disables buffering to see events immediately
```

### Create Order via curl

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "id":"order-002",
    "kind":"Order",
    "customer_id":"bob",
    "total":199.99,
    "status":"draft"
  }'
```

### Delete Order via curl

```bash
curl -X DELETE http://localhost:8080/api/orders/order-001
```

### List Orders

```bash
curl http://localhost:8080/api/orders | jq
```

## Multiple Watch Streams

Open multiple watch terminals to see that all clients receive events:

**Terminal 2:**
```bash
./apictl watch orders
```

**Terminal 4 (another watch):**
```bash
./apictl watch orders
```

**Terminal 5 (another watch):**
```bash
./apictl watch orders
```

**Terminal 3 (create order):**
```bash
./apictl create -f examples/order-1.json
```

Result: **All three watch terminals receive the same events simultaneously**.

No polling, no delays, no missed updates.

## Watch Different Resources

In the same server, you can watch different resources:

**Terminal 2:**
```bash
./apictl watch users
```

**Terminal 4:**
```bash
./apictl watch orders
```

**Terminal 5:**
```bash
./apictl watch products
```

Create users while watching orders - the watch streams don't interfere with each other.

## Custom Resource Example

Try watching a custom resource you create via CRD:

**Terminal 2:**
```bash
./apictl watch invoices
```

**Terminal 3:**
```bash
./apictl apply -f examples/invoice-crd.yaml
./apictl create -f examples/invoice-1.json
```

**Terminal 2 (watch):**
```
EVENT: ADDED
{
  "id": "inv-001",
  "customer": "Acme Corp",
  ...
}
```

The watch API works automatically for ANY resource - built-in or custom!

## Performance Notes

- **Event propagation**: <1ms
- **Network latency**: Depends on client connection
- **No polling**: Real-time, instant updates
- **Buffered channels**: 100 events per subscriber
- **Non-blocking publishers**: Never wait for slow subscribers

## Controller Behavior

The OrderController demonstrates the reconciliation pattern:

1. **ADDED** events:
   - Sets status = "processing"
   - Generates a MODIFIED event

2. **MODIFIED** events:
   - Logs the change
   - Could trigger additional logic

3. **DELETED** events:
   - Logs the deletion
   - Could trigger cleanup

This pattern is idempotent - calling Reconcile() multiple times with the same event is safe.

## Troubleshooting

### Watch doesn't show events

1. Check that server is running: `ps aux | grep api-server`
2. Check that OrderController started: Look for "Registered controller: OrderController" in logs
3. Try creating an order: `./apictl create -f examples/order-1.json`
4. Check server logs for errors

### Events appear but controller didn't process

The OrderController only updates **orders** resources. Other resources won't see controller-generated MODIFIED events.

### Watch connection closes

- Server shutdown
- Network issue
- Client Ctrl+C

Reconnect with `./apictl watch orders` again.

### Multiple MODIFIED events

1. MODIFIED #1: Controller sets status
2. MODIFIED #2: You explicitly update the order

This is correct - each update generates an event.

## Key Takeaways

1. **No Polling** - Events stream in real-time
2. **Decoupled** - Storage knows nothing about watchers/controllers
3. **Asynchronous** - Events processed in background
4. **Event-driven** - Business logic responds to events
5. **Scalable** - One update triggers many reactions
6. **Kubernetes-like** - Familiar patterns from Kubernetes

This architecture is the foundation for building reactive systems at scale.
