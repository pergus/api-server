# Dynamic API Server with CRDs - Demonstration

This demonstration shows how the server supports runtime API extensibility.

## Key Features

1. **Generic HTTP handlers** - Single set of handlers serve all resources
2. **Dynamic routing through resource lookup** - Resources are determined at runtime
3. **Resource registry** - Thread-safe, dynamically updated registry
4. **Scheme / factory registry** - Type creation without compiled types
5. **API discovery** - Discoverable via `/api`, `/apis` endpoints
6. **Custom Resource Definitions (CRDs)** - Register new resources at runtime
7. **Dynamic objects** - Generic JSON objects without compiled structs
8. **Discovery-based client** - `apictl` discovers available APIs

## Setup

### 1. Build the Server

```bash
cd api-server
go build -o server ./cmd/server
```

### 2. Build the apictl Client

```bash
go build -o apictl ./cmd/apictl
```

### 3. Start the Server

```bash
./api-server
```

Output should show:
```
Registering built-in resources...
Registered resource: users
Registered resource: products
Registered resource: orders
Starting server on http://localhost:8080
Discovery: GET http://localhost:8080/api
```

## Demonstration Sequence

### Step 1: List Available Resources

```bash
./apictl api-resources
```

Expected output:
```
NAME
orders
products
users
```

These are the built-in resources. No CRDs are registered yet.

### Step 2: Check API Groups

```bash
./apictl api-versions
```

### Step 3: Apply a CRD

```bash
./apictl apply -f examples/invoice-crd.yaml
```

Expected output:
```
CRD applied: invoices.example.io
```

**No server restart. No recompilation. The CRD is immediately registered.**

### Step 4: List Resources Again

```bash
./apictl api-resources
```

Expected output:
```
NAME
invoices
orders
products
users
```

Notice that `invoices` now appears. This resource was not present 10 seconds ago.

### Step 5: Create an Invoice

The new resource accepts objects without a compiled Go struct:

```bash
./apictl create -f examples/invoice-1.json
```

Expected output:
```
invoices created: inv-001
```

### Step 6: List All Invoices

```bash
./apictl get invoices
```

### Step 7: Get a Specific Invoice

```bash
./apictl get invoices inv-001
```

Output:
```json
{
  "apiVersion": "example.io/v1",
  "customer": "Acme Corp",
  "date": "2025-07-15",
  "id": "inv-001",
  "kind": "Invoice",
  "status": "sent",
  "amount": 5000
}
```

### Step 8: Explain the Resource

```bash
./apictl explain invoices
```

Shows the schema defined in the CRD.

### Step 9: Delete the CRD

```bash
./apictl delete crd invoices.example.io
```

Expected output:
```
CRD deleted: invoices.example.io
```

**Again, no server restart.**

### Step 10: Verify the Resource Disappeared

```bash
./apictl api-resources
```

Expected output:
```
NAME
orders
products
users
```

The `invoices` resource is gone. Any requests to `/api/invoices` will return 404.

## How It Works

### Generic Handlers

The router has exactly **two** route handlers:
- `GET /api` - Discovers all resources
- `GET /api/*` - Everything else (list, get, create, update, delete)

These two handlers are set up **once** when the server starts. They never change.

### Resource Lookup

On every HTTP request, the router:
1. Extracts the resource name from the URL path
2. Looks it up in the thread-safe Registry
3. Dispatches to the appropriate handler

If the resource is not found, returns 404. If registered, processes the request.

### CRD Registration

When a CRD is submitted:
1. Validate the definition
2. Create a `DynamicResource` wrapping the CRD
3. Register in the Resource Registry
4. Register in the Scheme (for object creation)
5. Expose via `/api/{plural}` endpoints

All of this happens while the server is running. No router rebuild. No restart.

### DynamicObject

The `DynamicObject` struct can hold any JSON data:

```go
type DynamicObject struct {
    APIVersion string                 `json:"apiVersion"`
    Kind       string                 `json:"kind"`
    Metadata   map[string]interface{} `json:"metadata"`
    Spec       map[string]interface{} `json:"spec"`
    Data       map[string]interface{} `json:"data,omitempty"`
}
```

This allows the generic handlers to work with any resource type without importing or knowing about it.

### apictl Discovery

The `apictl` client does not have any hardcoded resource names.

Instead:
1. `api-resources` calls `GET /api` to discover available resources
2. `api-versions` calls `GET /apis` to discover API groups
3. `create` and `get` look up the resource dynamically

If you add a new CRD, `apictl` automatically knows about it. No client rebuild. No recompilation.

## Key Insights

### Why This Architecture Matters

1. **No core server code needs to change** - New resource types are added via CRDs
2. **No router rebuild** - The HTTP dispatcher is generic
3. **No recompilation** - The server is one binary
4. **No restart** - Resources appear and disappear at runtime
5. **Client discovers APIs** - The client doesn't hardcode resource types

### Thread Safety

All critical sections use `sync.RWMutex`:
- Registry lookups (called on every request) use read locks
- Registration (rare) uses write locks
- This allows massive parallelism for the common case (lookups)

### Flexibility

The architecture supports:
- In-memory storage (as shown)
- SQL databases
- NoSQL databases
- Distributed systems (etcd, Consul)
- Cloud storage
- Anything that implements the `Storage` interface

## Advanced Examples

### Create Multiple Invoices

```bash
cat > invoice-2.json << 'EOF'
{
  "id": "inv-002",
  "customer": "TechCorp Ltd",
  "amount": 15000.00,
  "date": "2025-07-14",
  "status": "paid"
}
EOF

./apictl create -f invoice-2.json
./apictl get invoices
```

### Update via REST

```bash
curl -X PUT http://localhost:8080/api/invoices/inv-001 \
  -H "Content-Type: application/json" \
  -d '{
    "customer": "Acme Corp",
    "amount": 5500.00,
    "date": "2025-07-15",
    "status": "paid"
  }'
```

### Delete via REST

```bash
curl -X DELETE http://localhost:8080/api/invoices/inv-001
```

### List CRDs

```bash
./apictl get crds
```

Or via REST:

```bash
curl http://localhost:8080/crds
```

## Summary

This example demonstrates a full implementation of the API extensibility:

- ✓ Generic HTTP handlers
- ✓ Dynamic routing through resource lookup
- ✓ Resource registry (thread-safe)
- ✓ Scheme / factory registry
- ✓ API discovery
- ✓ Custom Resource Definitions (CRDs)
- ✓ Dynamic objects (no compiled structs)
- ✓ Declarative configuration (YAML CRDs)
- ✓ Discovery-based client (apictl)

No server restart. No recompilation. Pure runtime extensibility.
