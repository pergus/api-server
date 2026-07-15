# Architecture: Dynamic API Server

This document explains how the dynamic API server achieves runtime extensibility without server restart or recompilation.

## Core Problem

Typical REST servers create routes at startup:

```go
// Traditional approach (NOT used here)
mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("DELETE /users/{id}", deleteUser)
// ... repeat for products, orders, etc.
```

**Problems:**
- Adding a new resource requires code changes
- Routes are hardcoded
- Router cannot be updated while running
- New resources need manual route registration

## Solution: Generic Routing

This server uses **only generic routes** registered once:

```go
// Dynamic approach (USED HERE)
mux.HandleFunc("/api", discovery)        // List all resources
mux.HandleFunc("/", route)               // Route everything else
```

Every request:
1. Extracts resource name from URL
2. Looks it up in Registry (which can be updated at runtime)
3. Routes to ONE generic handler

The router **never changes**. New resources are added to the Registry, and the next request handles them.

## Component Architecture

### 1. Registry

The Registry is the foundation of dynamic extensibility.

```go
type Registry interface {
    Register(resource Resource) error
    Unregister(name string) error
    Lookup(name string) (Resource, bool)
    List() []Resource
    Names() []string
}
```

**Key Implementation Detail:**

```go
type SimpleRegistry struct {
    mu        sync.RWMutex
    resources map[string]Resource
}
```

Uses sync.RWMutex for thread safety:
- **Multiple readers** can call Lookup() concurrently (happens on every HTTP request)
- **Single writer** can Register/Unregister (happens during startup or plugin loading)

This is the **critical path for performance**: Registry.Lookup() happens on every request and must be fast.

**Why RWMutex?**
- Read locks are cheap
- Many goroutines can call Lookup() simultaneously
- Resource registration is rare (startup or plugin load)

### 2. Scheme (Type Factory)

The Scheme prevents the framework from knowing about concrete types.

```go
type ObjectFactory func() any

type Scheme interface {
    Register(name string, factory ObjectFactory) error
    New(name string) (any, error)
    Has(name string) bool
}
```

**Why This Pattern?**

Generic handlers CANNOT do this:
```go
// WRONG: Handler imports concrete types
func create(w http.ResponseWriter, resource Resource) {
    var obj interface{}
    if resourceName == "users" {
        obj = &User{}
    } else if resourceName == "products" {
        obj = &Product{}
    }
    json.Unmarshal(req.Body, obj)
}
```

Instead, they do this:
```go
// RIGHT: Framework creates objects by name
func create(w http.ResponseWriter, resource Resource) {
    obj, _ := scheme.New(resource.Name())  // Handler doesn't know type!
    json.Unmarshal(req.Body, obj)
}
```

The Scheme.New() call returns:
- `&User{}` when called with "users"
- `&Product{}` when called with "products"
- `&Invoice{}` when called with "invoices" (if plugin loaded)

The handler never imports User, Product, or Invoice.

### 3. Resource Interface

Every resource implements:

```go
type Resource interface {
    Name() string
    NewObject() any
    Storage() Storage
}
```

This is the **ONLY** interface the framework knows about. Everything else is hidden behind it.

### 4. Storage Interface

Persistence is abstracted:

```go
type Storage interface {
    List() ([]any, error)
    Get(id string) (any, error)
    Create(any) error
    Update(id string, any) error
    Delete(id string) error
}
```

Each resource has its own Storage instance. The framework never knows:
- How data is stored
- Where it's stored
- What technology is used

An implementation could use:
- In-memory (provided here)
- PostgreSQL
- MongoDB
- S3
- Anything else

### 5. Router (The Magic)

The Router is the entire API server behavior in one place.

**Setup (happens once at startup):**

```go
func (r *Router) Setup() {
    r.mux.HandleFunc("/api", r.discovery)
    r.mux.HandleFunc("/", r.route)
}
```

**Request Handling (happens for EVERY request):**

```go
func (r *Router) route(w http.ResponseWriter, req *http.Request) {
    // 1. Extract resource name from URL
    resourceName := parseURL(req.URL.Path)
    
    // 2. Look it up in registry
    // This lookup can find resources added WHILE the server is running
    resource, ok := r.registry.Lookup(resourceName)
    if !ok {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    // 3. Route based on HTTP method
    switch req.Method {
    case http.MethodGet:
        r.list(w, req, resource)
    case http.MethodPost:
        r.create(w, req, resource)
    }
}
```

**Handler Examples (all generic):**

```go
// One list handler for ALL resources
func (r *Router) list(w http.ResponseWriter, resource Resource) {
    objects, _ := resource.Storage().List()
    json.Marshal(objects)
}

// One create handler for ALL resources
func (r *Router) create(w http.ResponseWriter, resource Resource) {
    obj, _ := r.scheme.New(resource.Name())  // Generic!
    json.Unmarshal(req.Body, obj)
    resource.Storage().Create(obj)
}
```

Notice:
- No type assertions
- No switch statements
- No resource-specific logic
- No imports of User, Product, Invoice, etc.

These handlers work for ANY resource that implements the Resource interface.

### 6. Plugin System

Plugins demonstrate runtime extensibility.

**Plugin Interface:**

```go
type Plugin interface {
    Name() string
    Register(registry Registry, scheme Scheme) error
    Unregister(registry Registry) error
}
```

**Plugin Loading:**

```go
func (l *Loader) LoadPlugin(path string) error {
    // 1. Open .so file
    handle, _ := plugin.Open(path)
    
    // 2. Find Plugin symbol
    sym, _ := handle.Lookup("Plugin")
    p := sym.(Plugin)
    
    // 3. Call Register
    p.Register(l.registry, l.scheme)
    
    // Now the plugin's resources are available!
}
```

**What a Plugin Does:**

```go
// invoices plugin
func (p *InvoicePlugin) Register(registry Registry, scheme Scheme) {
    // Register the resource
    registry.Register(&InvoiceResource{})
    
    // Register the type
    scheme.Register("invoices", func() any { return &Invoice{} })
}
```

**Result:**
- `/api` now includes "invoices"
- `/api/invoices` works immediately
- All CRUD operations work
- Zero server changes

## Request Flow Example

**Request:** `POST /api/invoices`

```
1. HTTP Request arrives
   ↓
2. Router.route() called
   ↓
3. Extract "invoices" from path
   ↓
4. registry.Lookup("invoices") -> InvoiceResource
   ↓
5. HTTP method is POST -> route to create()
   ↓
6. scheme.New("invoices") -> &Invoice{}
   ↓
7. json.Unmarshal(body, &Invoice)
   ↓
8. invoiceResource.Storage().Create(&Invoice)
   ↓
9. Return 201 Created response
```

At NO point does the handler know:
- The type is Invoice
- What fields Invoice has
- How Invoice data is stored
- Even that invoices exist (determined at runtime)

## Thread Safety Guarantees

### Registry.Lookup() (Hot Path)

```go
func (r *SimpleRegistry) Lookup(name string) (Resource, bool) {
    r.mu.RLock()          // Read lock (cheap!)
    defer r.mu.RUnlock()
    resource, exists := r.resources[name]
    return resource, exists
}
```

- Multiple goroutines call this simultaneously
- Read lock blocks only when a write is happening
- Write is rare (startup, plugin load)

### Registry.Register() (Cold Path)

```go
func (r *SimpleRegistry) Register(resource Resource) error {
    r.mu.Lock()            // Exclusive lock
    defer r.mu.Unlock()
    r.resources[name] = resource
    return nil
}
```

- Called once per resource at startup
- Blocks all readers and writers
- But registration is quick

### Requests During Plugin Load

When a plugin loads:

```go
1. Plugin manager calls registry.Register()
   - Acquires write lock
   - Adds resource to map
   - Releases lock
   
2. While lock is held, HTTP requests may wait
   - But only for microseconds
   - New requests are queued by Go runtime

3. Next HTTP request can use the new resource
```

Perfect timing: Registry immediately consistent after plugin load.

## Why This Design?

### Separation of Concerns

- **Framework** (Router, Registry, Scheme, Middleware) - Generic
- **Resources** (User, Product, Invoice) - Specific
- **Storage** (In-memory, PostgreSQL, etc.) - Implementation detail

Changes to resources don't affect framework. Changes to framework don't affect resources.

### Extensibility Without Modification

Adding a new resource:
- No framework code changes
- No router code changes
- No recompilation of framework
- No restart of HTTP listener

Just:
1. Implement Resource interface
2. Call registry.Register()
3. Resource is available

### Kubernetes Parallel

This exactly mirrors Kubernetes:

| Concept | This Project | Kubernetes |
|---------|--------------|-----------|
| Resource | User, Product | Pod, Service |
| Resource Interface | api.Resource | metav1.Object |
| Registry | Registry | API Server |
| Scheme | Scheme | scheme.Scheme |
| Storage | Storage | etcd |
| Plugin | Plugin | CRD |
| Router | router | API Server request handler |

When you create a Kubernetes CRD, it gets registered in the API server's registry. The next request includes it in discovery. Full CRUD works immediately. This project demonstrates the same thing at a smaller scale.

## Performance Considerations

### Request Path

```
HTTP Request
  ↓
Middleware (logging, recovery, timing)
  ↓
router.route()
  ↓
registry.Lookup()  <- CRITICAL
  ↓
Generic handler (list, get, create, update, delete)
  ↓
resource.Storage().*  <- Implementation dependent
  ↓
Response
```

**Critical Path:** Registry.Lookup() with RWMutex

For high performance:
- Use RWMutex (not regular Mutex)
- Keep Registry.Lookup() fast (one map lookup)
- Avoid allocations in hot path

Current design:
- Single map.lookup() call
- RWMutex (optimal for read-heavy workload)
- No allocations in critical path

### Scalability

The generic router scales with request volume, not resource count:
- 3 resources or 3000 resources = same handler
- Performance is O(1) per resource lookup
- Plugin loading is O(1) per plugin

## Alternative Architectures Considered

### Option 1: Code Generation (NOT USED)

Generate route handlers per resource at compile time.

**Pros:** Type-safe
**Cons:** Requires recompilation for new resources

### Option 2: Dynamic Route Builder (NOT USED)

Rebuild HTTP routes when resources register.

```go
func (r *Router) Register(resource Resource) {
    r.mux.HandleFunc(fmt.Sprintf("GET /api/%s", resource.Name()),
        func(w http.ResponseWriter, req *http.Request) {
            r.get(w, req, resource)
        })
}
```

**Pros:** Might be slightly faster
**Cons:** Requires lock on mux, mux not designed for concurrent mutation

### Option 3: Middleware Chain (CHOSE THIS)

Single catch-all route that determines resource per request.

**Pros:**
- Routes never change
- No locks on HTTP router
- Truly generic
- Mirrors Kubernetes design

**Cons:**
- Slightly more overhead per request (URL parsing, map lookup)
- But this is negligible compared to I/O

## Summary

The key insight:

> **Generic routing is more extensible than generated routing.**

By moving resource discovery from HTTP router setup to request handling, we enable runtime registration without server restart.

The cost is minimal: one Registry.Lookup() call per request (microseconds with RWMutex).

The benefit is enormous: new resources work without recompilation or server restart.

This is how Kubernetes achieves its legendary extensibility.
