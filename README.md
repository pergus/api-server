# Dynamic API Server

A truly dynamic, runtime-extensible REST API server in Go that demonstrates how Kubernetes achieves extensibility without server restarts or recompilation.

**Key Features:**
- New API resources can be added while the server is running
- No HTTP router rebuild required
- No server restart needed
- **Watch API** - Stream events in real-time with Server-Sent Events
- **Controller Framework** - Event-driven business logic (Kubernetes-style)
- Two ways to extend:
  1. **CRDs (Custom Resource Definitions)** - JSON POST to `/crds` endpoint
  2. **Plugins (Go .so files)** - Classic plugin system

## Quick Start with CRDs

The easiest way to extend the API:

```bash
# Build
go build -o api-server ./cmd/api-server
go build -o apictl ./cmd/apictl

# Terminal 1: Start server
./api-server

# Terminal 2: List resources
./apictl api-resources

# Apply a CRD
./apictl apply -f examples/invoice-crd.yaml

# Notice invoices now appears!
./apictl api-resources

# Create an invoice
./apictl create -f examples/invoice-1.json

# List invoices
./apictl get invoices
```

See `QUICKSTART.md` for the 5-minute demo or run `./demo.sh`.

## Watch API and Controllers

Stream real-time events and implement event-driven business logic:

**Terminal 1 - Watch for events:**
```bash
./apictl watch orders
```

**Terminal 2 - Create an order (in another terminal):**
```bash
./apictl create -f examples/order-1.json
```

**Terminal 1 - See events in real-time:**
```
EVENT: ADDED
{
  "id": "order-001",
  "status": "draft",
  ...
}

EVENT: MODIFIED
{
  "id": "order-001",
  "status": "processing",  ← OrderController set this
  ...
}
```

The OrderController automatically processes orders, calculating totals and updating status. See `WATCH_ARCHITECTURE.md` and `WATCH_DEMO.md` for detailed explanation and complete walkthrough.

## Quick Start with Plugins

The traditional plugin approach:

### Build

```bash
go build -o server ./cmd/server
```

### Create Plugins Directory

```bash
mkdir -p plugins
```

### Run

```bash
./api-server
```

Expected output:
```
2024/... Registering built-in resources...
2024/... Starting plugin system...
2024/... Scanning for existing plugins...
2024/... Setting up routes (generic, never change)
2024/... Registered resources: 3
2024/...   - orders
2024/...   - products
2024/...   - users
2024/... Starting server on http://localhost:8080
2024/... Discovery: GET http://localhost:8080/api
```

## Architecture

The server uses a truly generic HTTP router:

```
Only ONE set of routes is registered at startup:
  GET    /api                  - Discovery
  GET    /api/{resource}       - List
  POST   /api/{resource}       - Create
  GET    /api/{resource}/{id}  - Get
  PUT    /api/{resource}/{id}  - Update
  DELETE /api/{resource}/{id}  - Delete
```

Every request:
1. Extracts the resource name from the URL
2. Looks it up in the thread-safe registry
3. Routes to the appropriate generic handler

This means **new resources are immediately available** after registration without any router changes.

## Using the API

### Discover Available Resources

```bash
curl http://localhost:8080/api | jq
```

Response:
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

### List Users

```bash
curl http://localhost:8080/api/users | jq
```

### Create a User

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "id": "alice",
    "name": "Alice Johnson",
    "email": "alice@example.com",
    "is_active": true
  }' | jq
```

### Get a User

```bash
curl http://localhost:8080/api/users/alice | jq
```

### Update a User

```bash
curl -X PUT http://localhost:8080/api/users/alice \
  -H "Content-Type: application/json" \
  -d '{
    "id": "alice",
    "name": "Alice Smith",
    "email": "alice.smith@example.com",
    "is_active": true
  }' | jq
```

### Delete a User

```bash
curl -X DELETE http://localhost:8080/api/users/alice | jq
```

## Dynamic Plugin Loading

### Build a Plugin

```bash
cd plugins
chmod +x build.sh
./build.sh
```

This builds the invoices plugin as `invoices/invoices.so`.

### Load Plugin While Server Running

**In another terminal:**

```bash
cp plugins/invoices/invoices.so plugins/
```

The server will detect the new plugin within 2 seconds and load it automatically.

### Verify Plugin Loaded

```bash
curl http://localhost:8080/api | jq
```

You should now see `invoices` in the resources list.

### Use the New Resource

```bash
curl -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{
    "id": "inv-001",
    "customer_id": "alice",
    "amount": 99.99,
    "status": "draft"
  }' | jq

curl http://localhost:8080/api/invoices | jq
```

## Complete Demonstration

1. Start the server:
   ```bash
   ./api-server
   ```

2. In another terminal, verify initial resources:
   ```bash
   curl http://localhost:8080/api | jq
   # Shows: users, products, orders
   ```

3. Build plugins:
   ```bash
   cd plugins
   ./build.sh
   ```

4. While server is running, copy plugin:
   ```bash
   cp plugins/invoices/invoices.so plugins/
   ```

5. Verify invoices appear in discovery (within 2 seconds):
   ```bash
   curl http://localhost:8080/api | jq
   # Now shows: invoices, orders, products, users
   ```

6. Use the new resource immediately:
   ```bash
   curl -X POST http://localhost:8080/api/invoices \
     -H "Content-Type: application/json" \
     -d '{"id":"inv-001","customer_id":"alice","amount":99.99,"status":"draft"}' | jq
   ```

No server restart. No recompilation. No router rebuild.

## Custom Resource Definitions (CRDs)

Define new resources without writing Go code:

### Create a CRD via REST API

```bash
curl -X POST http://localhost:8080/crds \
  -H "Content-Type: application/json" \
  -d '{
    "group": "example.io",
    "version": "v1",
    "kind": "Invoice",
    "plural": "invoices",
    "schema": {
      "id": "string",
      "customer": "string",
      "amount": "number"
    }
  }'
```

### Or via apitcl

```bash
./apictl apply -f examples/invoice-crd.yaml
```

### Use the New Resource

```bash
# Create
./apictl create -f invoice.json

# List
./apictl get invoices

# Get specific
./apictl get invoices inv-001

# Delete
./apictl delete invoices inv-001
```

**Key Point**: Resources created via CRDs don't require compiled Go types. They use generic `DynamicObject` internally.

### CRD Endpoints

```
POST   /crds              - Create CRD
GET    /crds              - List CRDs
DELETE /crds/{fullName}   - Delete CRD
```

### API Discovery for CRDs

```
GET /api                  - List all resources (built-in + CRDs)
GET /apis                 - List API groups
GET /apis/{group}         - List resources in group
GET /apis/{group}/{version} - List resources in group/version
```

## apitcl Client

A discovery-based CLI that learns about resources from the server:

```bash
./apictl api-resources      # Discover all resources
./apictl api-versions       # Discover API groups
./apictl get <resource>     # List objects
./apictl get <resource> <id> # Get specific object
./apictl create -f <file>   # Create from JSON
./apictl apply -f <file>    # Apply CRD from YAML
./apictl delete <resource> <id> # Delete object
./apictl explain <resource> # Show schema
```

**No hardcoded resources** - All discovery happens at runtime.

## Creating Your Own Plugin

1. Create a new Go file in the plugins directory:

```go
package main

import (
    "github.com/example/api-server/pkg/api"
    "github.com/example/api-server/pkg/plugins"
)

type MyResource struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type MyResourceResource struct {
    storage api.Storage
}

func NewMyResourceResource() *MyResourceResource {
    return &MyResourceResource{
        storage: api.NewMemoryStorage(),
    }
}

func (r *MyResourceResource) Name() string {
    return "myresources"
}

func (r *MyResourceResource) NewObject() any {
    return &MyResource{}
}

func (r *MyResourceResource) Storage() api.Storage {
    return r.storage
}

type MyResourcePlugin struct {
    resource *MyResourceResource
}

func (p *MyResourcePlugin) Name() string {
    return "myresources"
}

func (p *MyResourcePlugin) Register(registry api.Registry, scheme api.Scheme) error {
    if err := registry.Register(p.resource); err != nil {
        return err
    }
    return scheme.Register("myresources", func() any { return &MyResource{} })
}

func (p *MyResourcePlugin) Unregister(registry api.Registry) error {
    return registry.Unregister("myresources")
}

var Plugin plugins.Plugin = &MyResourcePlugin{
    resource: NewMyResourceResource(),
}
```

2. Build as a plugin:
```bash
go build -buildmode=plugin -o myresources.so ./myresources/main.go
```

3. Copy to plugins directory:
```bash
cp myresources.so plugins/
```

The server will automatically load it.

## Architecture Details

### Core Components

**Registry** - Thread-safe resource storage
- Allows concurrent reads (Lookup happens on every request)
- Exclusive write (Register/Unregister)
- Uses sync.RWMutex

**Scheme** - Type factory registry
- Maps type names (strings) to constructor functions
- Allows generic handlers to create objects without knowing types
- Prevents direct type imports in framework

**Router** - Generic, never changes
- Only registers generic routes at startup
- Determines resource at request time
- Calls generic handlers (one implementation per operation)
- Never contains resource-specific logic

**Plugin System** - Runtime extensibility
- Loads .so files (Go plugins)
- Calls Plugin.Register() to add resources
- Watches directory for new plugins
- Supports plugin unloading

### Key Design Patterns

1. **Interface-Based** - Everything depends on interfaces, never concrete types
2. **Runtime Registration** - Resources can be added while server is running
3. **Generic Routing** - Routes never change; resource determination happens per-request
4. **Thread Safety** - Registry and Scheme use RWMutex for concurrent access
5. **Dependency Injection** - Explicit construction, minimal global state

## How This Mirrors Kubernetes

Kubernetes achieves extensibility through:
- **API Resources** - Like our Resource interface
- **Scheme** - Type factory registry (identical pattern)
- **API Server** - Generic request handling (like our Router)
- **Resource Registry** - All known resources (like our Registry)
- **Storage Interface** - Abstract persistence (like our Storage)
- **CRDs** - Custom Resource Definitions (like our Plugins)

When you define a CRD in Kubernetes:
1. It's registered with the API server
2. It appears in /api (discovery)
3. Full CRUD endpoints work immediately
4. No API server restart needed

This project demonstrates exactly those patterns.

## Project Structure

```
api-server/
├── go.mod                          # Go module file
├── cmd/
│   ├── server/main.go              # Server entry point
│   └── apitcl/               # CLI client
│       ├── main.go
│       ├── client.go
│       └── commands.go
├── pkg/
│   ├── api/                        # Core framework
│   │   ├── types.go                # Common types
│   │   ├── resource.go             # Resource interface
│   │   ├── storage.go              # Storage interface
│   │   ├── registry.go             # Resource registry
│   │   ├── scheme.go               # Type factory
│   │   ├── router.go               # Generic routing
│   │   ├── middleware.go           # Middleware
│   │   ├── server.go               # HTTP server
│   │   ├── crd.go                  # CRD definitions & registry
│   │   └── dynamic.go              # DynamicObject type
│   ├── plugins/                    # Plugin system
│   │   ├── interface.go            # Plugin interface
│   │   └── loader.go               # Plugin loader
│   └── resources/                  # Built-in resources
│       ├── users.go
│       ├── products.go
│       └── orders.go
├── plugins/                        # Plugin directory
│   ├── invoices/main.go
│   ├── build.sh
│   └── (loaded .so files go here)
├── examples/                       # Example files
│   ├── invoice-crd.yaml            # CRD definition
│   ├── invoice-1.json              # Sample object
│   └── DEMO.md                     # Detailed demo
├── README.md
├── QUICKSTART.md                   # 5-minute quickstart
├── ARCHITECTURE.md                 # Core architecture
├── CRD_ARCHITECTURE.md             # CRD system design
├── BUILD.md                        # Build instructions
└── demo.sh                         # Automated demo script
```

## Files Overview

- `pkg/api/router.go` - The most important file. Shows how generic routing works.
- `pkg/api/registry.go` - Thread-safe resource storage with RWMutex.
- `pkg/api/scheme.go` - Type factory pattern preventing direct type imports.
- `pkg/plugins/loader.go` - Plugin watching and loading system.
- `plugins/invoices/main.go` - Complete plugin example.
- `cmd/server/main.go` - Demonstrates registration flow.

## Advanced Topics

### Plugin Hot-Reload

The system supports hot-reloading:
- Remove .so file from plugins/
- Server detects removal but continues operating
- (Unregister not yet implemented; plugins are loaded once)

### Thread Safety

All critical sections use appropriate synchronization:
- Registry: RWMutex (read lock for lookups, write lock for registration)
- Scheme: RWMutex (read lock for New, write lock for Register)
- MemoryStorage: RWMutex (read lock for Get/List, write lock for Create/Update/Delete)

### Why No Switch Statements?

The handlers never check resource types:
- No: `switch resourceName { case "users": ... }`
- Instead: Look up in Registry, call through interfaces

This makes the framework truly extensible.

## Limitations

- Go plugins can only be loaded, not unloaded (Go limitation)
- Plugins must be written in Go
- Performance depends on Registry.Lookup() speed (uses RWMutex)

## Future Enhancements

- gRPC-based plugins for language independence
- Plugin authentication/validation
- Metrics and observability
- Better error handling in plugin loading
- Plugin versioning

## References

- [Kubernetes API Machinery](https://github.com/kubernetes/api)
- [Go Plugin System](https://golang.org/pkg/plugin/)
- [Kubernetes CRD Architecture](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
