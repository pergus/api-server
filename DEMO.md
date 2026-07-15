# Dynamic API Server - Complete Demonstration

This walkthrough demonstrates genuine runtime extensibility: adding new API resources while the server is running, without server restart or recompilation.

## What You'll See

By the end of this demo, you'll understand:

1. How the generic router works
2. How Registry enables runtime registration
3. How Scheme allows generic handlers
4. How plugins are loaded without restart
5. How the HTTP router never needs rebuilding

## Prerequisites

- Go 1.21+
- Unix-like system (Linux/macOS)
- curl for testing

## Step 1: Build the Server

```bash
cd /Users/go/Documents/go-api-server/api-server
go build -o server ./cmd/server
```

Output:
```
(no output means success)
```

Verify:
```bash
ls -lh server
# -rwxr-xr-x 1 user staff 12M ... server
```

## Step 2: Build Plugins

```bash
cd plugins/invoices
go build -buildmode=plugin -o invoices.so main.go
ls -lh invoices.so
# -rw-r--r-- 1 user staff 9.1M ... invoices.so
```

## Step 3: Start the Server

```bash
cd /Users/go/Documents/go-api-server/api-server
./server
```

Expected output:
```
Registering built-in resources...
Starting plugin system...
Scanning for existing plugins...
Setting up routes (generic, never change)
Registered resources: 3
  - orders
  - products
  - users
Starting server on http://localhost:8080
Discovery: GET http://localhost:8080/api
```

**Key observation:** Generic routes are set up ONCE. They will never change.

## Step 4: Test Initial Resources

**In another terminal:**

Discover available resources:
```bash
curl -s http://localhost:8080/api | jq
```

Output:
```json
{
  "resources": [
    "orders",
    "products",
    "users"
  ],
  "timestamp": "2024-..."
}
```

Note: invoices is NOT in the list (not loaded yet).

Create a user:
```bash
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "id": "alice",
    "name": "Alice Johnson",
    "email": "alice@example.com",
    "is_active": true
  }' | jq
```

Output:
```json
{
  "message": "users created",
  "id": "alice"
}
```

List users:
```bash
curl -s http://localhost:8080/api/users | jq
```

Output:
```json
{
  "items": [
    {
      "id": "alice",
      "name": "Alice Johnson",
      "email": "alice@example.com",
      "is_active": true
    }
  ],
  "count": 1
}
```

## Step 5: THE MAGIC - Load Plugin While Server Running

**Keep the server running. In the SAME terminal (different window if needed):**

```bash
cp /Users/go/Documents/go-api-server/api-server/plugins/invoices/invoices.so /Users/go/Documents/go-api-server/api-server/plugins/
```

Wait 2-3 seconds for the plugin watcher to detect the new file.

## Step 6: Verify Plugin Loaded

Check the server output (first terminal). You should see:
```
[InvoicePlugin] Registering invoice resource and type
[InvoicePlugin] Successfully registered invoices
Loading plugin from ./plugins/invoices.so
Successfully loaded plugin: invoices
Registered resource: invoices
```

Verify in discovery:
```bash
curl -s http://localhost:8080/api | jq
```

Output:
```json
{
  "resources": [
    "invoices",
    "orders",
    "products",
    "users"
  ],
  "timestamp": "2024-..."
}
```

**invoices is now in the list!**

## Step 7: Use the New Resource Immediately

No restart needed. The resource works immediately.

Create an invoice:
```bash
curl -s -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{
    "id": "inv-001",
    "customer_id": "alice",
    "amount": 99.99,
    "status": "draft"
  }' | jq
```

Output:
```json
{
  "message": "invoices created",
  "id": "inv-001"
}
```

List invoices:
```bash
curl -s http://localhost:8080/api/invoices | jq
```

Output:
```json
{
  "items": [
    {
      "id": "inv-001",
      "customer_id": "alice",
      "amount": 99.99,
      "status": "draft"
    }
  ],
  "count": 1
}
```

Get an invoice:
```bash
curl -s http://localhost:8080/api/invoices/inv-001 | jq
```

Update an invoice:
```bash
curl -s -X PUT http://localhost:8080/api/invoices/inv-001 \
  -H "Content-Type: application/json" \
  -d '{
    "id": "inv-001",
    "customer_id": "alice",
    "amount": 99.99,
    "status": "sent"
  }' | jq
```

Delete an invoice:
```bash
curl -s -X DELETE http://localhost:8080/api/invoices/inv-001 | jq
```

## The Key Insight

**The server never restarted.** The HTTP listener never restarted. The router never rebuilt.

Yet new endpoints appeared:
- GET /api/invoices
- POST /api/invoices
- GET /api/invoices/{id}
- PUT /api/invoices/{id}
- DELETE /api/invoices/{id}

This is how Kubernetes works with Custom Resource Definitions (CRDs).

## How It Works

### 1. Generic Router

The entire HTTP router consists of:
```
/api                 -> discovery endpoint
/                    -> catch-all router
```

That's it. Only 2 routes, registered once at startup.

### 2. Runtime Lookup

Every request:
1. Extracts resource name from URL
2. Looks it up in Registry
3. Calls generic handler

```
POST /api/invoices
  ↓ Extract "invoices"
  ↓ registry.Lookup("invoices") returns InvoiceResource
  ↓ Call generic create handler
  ↓ Handler uses Scheme to create &Invoice{}
  ↓ Handler uses InvoiceResource.Storage().Create()
  ↓ Response
```

The handler NEVER knows the type is Invoice.

### 3. No Type Assertions

The handlers never do:
```go
// NOT USED
if resourceName == "invoices" {
    // invoice-specific code
}
```

Instead they do:
```go
// USED
resource, _ := registry.Lookup(resourceName)
obj, _ := scheme.New(resourceName)
resource.Storage().Create(obj)
```

## Why This Matters

### Traditional REST Servers

```go
// At startup, all routes hardcoded
mux.HandleFunc("GET /api/users", listUsers)
mux.HandleFunc("GET /api/users/{id}", getUser)
mux.HandleFunc("GET /api/products", listProducts)
mux.HandleFunc("GET /api/products/{id}", getProduct)
mux.HandleFunc("GET /api/invoices", listInvoices)
// ... etc

// Adding a new resource requires:
// 1. Code change
// 2. Recompile
// 3. Restart
```

### This Server

```go
// At startup, only generic routes
mux.HandleFunc("/api", discovery)
mux.HandleFunc("/", route)

// Adding a new resource requires:
// 1. Build plugin (.so file)
// 2. Copy to plugins/
// 3. Server loads automatically (within 2 seconds)
// 4. No restart. No recompilation.
```

## Advanced: Create Another Plugin

To demonstrate further extensibility, let's create an orders plugin.

1. Create the plugin:

```bash
mkdir -p /Users/go/Documents/go-api-server/api-server/plugins/shipping
cat > /Users/go/Documents/go-api-server/api-server/plugins/shipping/main.go << 'EOF'
package main

import (
    "log"
    "github.com/example/api-server/pkg/api"
    "github.com/example/api-server/pkg/plugins"
)

type Shipment struct {
    ID      string `json:"id"`
    OrderID string `json:"order_id"`
    Status  string `json:"status"`
}

type ShipmentResource struct {
    storage api.Storage
}

func NewShipmentResource() *ShipmentResource {
    return &ShipmentResource{
        storage: api.NewMemoryStorage(),
    }
}

func (r *ShipmentResource) Name() string {
    return "shipments"
}

func (r *ShipmentResource) NewObject() any {
    return &Shipment{}
}

func (r *ShipmentResource) Storage() api.Storage {
    return r.storage
}

type ShipmentPlugin struct {
    resource *ShipmentResource
}

func (p *ShipmentPlugin) Name() string {
    return "shipments"
}

func (p *ShipmentPlugin) Register(registry api.Registry, scheme api.Scheme) error {
    log.Println("[ShipmentPlugin] Registering")
    if err := registry.Register(p.resource); err != nil {
        return err
    }
    return scheme.Register("shipments", func() any { return &Shipment{} })
}

func (p *ShipmentPlugin) Unregister(registry api.Registry) error {
    return registry.Unregister("shipments")
}

var Plugin plugins.Plugin = &ShipmentPlugin{
    resource: NewShipmentResource(),
}
EOF
```

2. Build it:

```bash
cd /Users/go/Documents/go-api-server/api-server/plugins/shipping
go build -buildmode=plugin -o shipments.so main.go
```

3. While server is running:

```bash
cp /Users/go/Documents/go-api-server/api-server/plugins/shipping/shipments.so /Users/go/Documents/go-api-server/api-server/plugins/
```

4. Check discovery:

```bash
curl -s http://localhost:8080/api | jq
```

Now you should see both "invoices" and "shipments" in the resources list!

5. Use the new resource:

```bash
curl -s -X POST http://localhost:8080/api/shipments \
  -H "Content-Type: application/json" \
  -d '{
    "id": "ship-001",
    "order_id": "inv-001",
    "status": "pending"
  }' | jq
```

## Summary

In this demonstration, you saw:

1. **Generic Routing** - Only 2 routes, never change
2. **Runtime Registry** - Resources can be added while running
3. **Dynamic Dispatch** - Requests look up resources at runtime
4. **Generic Handlers** - One handler per operation, works for all resources
5. **Plugin System** - New resources loaded automatically
6. **Kubernetes Pattern** - Exactly how Kubernetes extends itself

All without:
- Server restart
- Server recompilation
- Router rebuild
- HTTP listener restart
- Any framework changes

This is true architectural extensibility.

## Questions to Ponder

1. Why does the router never rebuild? 
   - Because resource discovery happens at REQUEST TIME, not startup time.

2. How can the handlers work for any resource?
   - They operate through Resource and Storage interfaces, never concrete types.

3. How does the framework avoid knowing about Invoice?
   - The Scheme creates objects by name; handlers never import concrete types.

4. Why is this thread-safe?
   - Registry uses RWMutex; lookups have read lock, registration has write lock.

5. How is this like Kubernetes?
   - Kubernetes does exactly this with CRDs; this is a scaled-down version.

## Next Steps

- Modify a plugin and rebuild to see hot-reload (restart server to reload)
- Add more fields to resource types
- Create a plugin with a different storage backend
- Implement authentication in middleware
- Add request validation
- Build metrics collection

The architecture supports all of these without changes.
