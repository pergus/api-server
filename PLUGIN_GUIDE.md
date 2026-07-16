# Writing a Plugin Guide

This guide explains how to write a plugin for the dynamic API server. Plugins allow you to add new resources at runtime without restarting the server.

## What is a Plugin?

A plugin is a compiled Go package loaded as a shared object (.so file) at runtime. It lets you:
- Define new resource types
- Add storage backends
- Extend the API without server restart
- Add business logic tied to specific resources

## Plugin Lifecycle

1. **Build** - Compile to .so file with `-buildmode=plugin`
2. **Deploy** - Copy to `plugins/` directory
3. **Load** - Server detects and loads automatically (2-second poll)
4. **Register** - Plugin registers resources with the server
5. **Use** - New resources immediately available via API
6. **Unload** - Optional: unregister when needed

## Plugin Structure

Every plugin must:
1. Define at least one resource type (struct)
2. Implement the `api.Resource` interface
3. Implement the `plugins.Plugin` interface
4. Export a `Plugin` symbol of type `plugins.Plugin`

## Step-by-Step Example: Invoice Plugin

### 1. Define Your Resource Type

```go
type Invoice struct {
    ID         string  `json:"id"`
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
    Status     string  `json:"status"`
}
```

Your resource type is just a regular Go struct. It will be marshalled to/from JSON by the API.

### 2. Implement the Resource Interface

Create a wrapper that implements `api.Resource`:

```go
type InvoiceResource struct {
    storage api.Storage
}

// Name returns the resource identifier used in URLs
// GET /api/invoices, POST /api/invoices, etc.
func (r *InvoiceResource) Name() string {
    return "invoices"
}

// NewObject creates a new empty instance of your type
// The framework calls this to create objects from JSON
func (r *InvoiceResource) NewObject() any {
    return &Invoice{}
}

// Storage returns the persistence layer
// Use api.NewMemoryStorage() for in-memory storage
func (r *InvoiceResource) Storage() api.Storage {
    return r.storage
}
```

### 3. Implement the Plugin Interface

Create a plugin type that implements `plugins.Plugin`:

```go
type InvoicePlugin struct {
    resource *InvoiceResource
}

// Name returns the plugin name for logging
func (p *InvoicePlugin) Name() string {
    return "invoices"
}

// Register is called when the plugin is loaded
// Register resources and types with the provided registries
func (p *InvoicePlugin) Register(registry api.Registry, scheme api.Scheme) error {
    log.Println("[InvoicePlugin] Registering...")
    
    // Register the resource
    if err := registry.Register(p.resource); err != nil {
        return err
    }
    
    // Register the type factory
    // This tells the framework how to construct new objects from JSON
    if err := scheme.Register("invoices", func() any {
        return &Invoice{}
    }); err != nil {
        return err
    }
    
    log.Println("[InvoicePlugin] Successfully registered")
    return nil
}

// Unregister is called when the plugin is unloaded
// Clean up: unregister resources
func (p *InvoicePlugin) Unregister(registry api.Registry) error {
    log.Println("[InvoicePlugin] Unregistering...")
    return registry.Unregister("invoices")
}
```

### 4. Export the Plugin Symbol

The loader looks for a symbol named `Plugin` of type `plugins.Plugin`:

```go
var Plugin plugins.Plugin = &InvoicePlugin{
    resource: &InvoiceResource{
        storage: api.NewMemoryStorage(),
    },
}
```

This is how the server discovers and loads your plugin.

## Complete Plugin Example

```go
package main

import (
    "log"
    "github.com/pergus/api-server/pkg/api"
    "github.com/pergus/api-server/pkg/plugins"
)

type Invoice struct {
    ID         string  `json:"id"`
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
    Status     string  `json:"status"`
}

type InvoiceResource struct {
    storage api.Storage
}

func (r *InvoiceResource) Name() string {
    return "invoices"
}

func (r *InvoiceResource) NewObject() any {
    return &Invoice{}
}

func (r *InvoiceResource) Storage() api.Storage {
    return r.storage
}

type InvoicePlugin struct {
    resource *InvoiceResource
}

func (p *InvoicePlugin) Name() string {
    return "invoices"
}

func (p *InvoicePlugin) Register(registry api.Registry, scheme api.Scheme) error {
    log.Println("[InvoicePlugin] Registering")
    
    if err := registry.Register(p.resource); err != nil {
        return err
    }
    
    if err := scheme.Register("invoices", func() any {
        return &Invoice{}
    }); err != nil {
        return err
    }
    
    log.Println("[InvoicePlugin] Registered successfully")
    return nil
}

func (p *InvoicePlugin) Unregister(registry api.Registry) error {
    return registry.Unregister("invoices")
}

var Plugin plugins.Plugin = &InvoicePlugin{
    resource: &InvoiceResource{
        storage: api.NewMemoryStorage(),
    },
}
```

## Building Your Plugin

Create a directory for your plugin:

```bash
mkdir -p plugins/myresource
cd plugins/myresource
```

Create `main.go` with your plugin code, then build:

```bash
# Build as a plugin (shared object)
go build -buildmode=plugin -o myresource.so main.go

# Move to plugins directory
cp myresource.so ../
```

## Deploying Your Plugin

### While Server is Running

```bash
# Copy the .so file to the plugins/ directory
cp myresource.so plugins/

# Server detects it within 2 seconds and loads automatically
# Watch the logs: you should see "[InvoicePlugin] Registering"
```

### Testing Your Plugin

Once loaded, test via the API:

```bash
# Discover resources
curl http://localhost:8080/api

# Create an object
curl -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{"id":"inv-1","customer_id":"cust-1","amount":100.50,"status":"draft"}'

# List objects
curl http://localhost:8080/api/invoices

# Get specific object
curl http://localhost:8080/api/invoices/inv-1

# Update object
curl -X PUT http://localhost:8080/api/invoices/inv-1 \
  -H "Content-Type: application/json" \
  -d '{"id":"inv-1","customer_id":"cust-1","amount":150.75,"status":"sent"}'

# Delete object
curl -X DELETE http://localhost:8080/api/invoices/inv-1
```

Or use the CLI client:

```bash
./apictl api-resources           # See your plugin listed
./apictl create -f invoice.json  # Create an invoice
./apictl get invoices            # List all invoices
./apictl get invoices inv-1      # Get specific invoice
./apictl delete invoices inv-1   # Delete invoice
```

## Storage Backends

By default, plugins use `api.NewMemoryStorage()` which stores data in memory. This data is lost when the server restarts.

### Using a Custom Storage Backend

Implement the `api.Storage` interface and pass it to your resource:

```go
type MyDatabase struct {
    // ... database connection
}

func (db *MyDatabase) Create(obj any) error {
    // Store obj in database
    return nil
}

func (db *MyDatabase) Get(id string) (any, error) {
    // Retrieve from database
    return nil, fmt.Errorf("not found")
}

func (db *MyDatabase) List() ([]any, error) {
    // Return all objects
    return nil, nil
}

func (db *MyDatabase) Update(id string, obj any) error {
    // Update in database
    return nil
}

func (db *MyDatabase) Delete(id string) error {
    // Delete from database
    return nil
}

// Use in your plugin:
type MyPlugin struct {
    resource *MyResource
}

func (p *MyPlugin) Register(registry api.Registry, scheme api.Scheme) error {
    p.resource.storage = &MyDatabase{}  // Use custom storage
    return registry.Register(p.resource)
}
```

## Naming Conventions

- **Plugin directory**: `plugins/myresource/`
- **Plugin binary**: `myresource.so`
- **Resource name**: Same as directory, used in URLs (`/api/myresource`)
- **Type name**: Singular, PascalCase (e.g., `Invoice`)
- **Resource struct**: TypeResource (e.g., `InvoiceResource`)
- **Plugin struct**: TypePlugin (e.g., `InvoicePlugin`)

## Common Patterns

### Plugin with Multiple Resources

```go
type MyPlugin struct {
    resource1 *Resource1
    resource2 *Resource2
}

func (p *MyPlugin) Register(registry api.Registry, scheme api.Scheme) error {
    if err := registry.Register(p.resource1); err != nil {
        return err
    }
    if err := registry.Register(p.resource2); err != nil {
        return err
    }
    // ... register types ...
    return nil
}

func (p *MyPlugin) Unregister(registry api.Registry) error {
    registry.Unregister(p.resource1.Name())
    registry.Unregister(p.resource2.Name())
    return nil
}
```

### Plugin with Initialization

```go
type MyPlugin struct {
    resource *MyResource
    config   Config
}

func NewMyPlugin(cfg Config) *MyPlugin {
    return &MyPlugin{
        config: cfg,
        resource: &MyResource{
            storage: api.NewMemoryStorage(),
        },
    }
}

func (p *MyPlugin) Register(registry api.Registry, scheme api.Scheme) error {
    // Use config during registration
    log.Printf("Registering with config: %v", p.config)
    return registry.Register(p.resource)
}

var Plugin plugins.Plugin = NewMyPlugin(Config{})
```

## Troubleshooting

### Plugin Not Loading

**Check:**
1. File is in `plugins/` directory
2. File has `.so` extension
3. File was built with `-buildmode=plugin`
4. Server logs show "Scanning for existing plugins"

**Watch logs:**
```bash
./api-server 2>&1 | grep -i plugin
```

### "Plugin symbol is not of type Plugin"

**Problem:** The `Plugin` variable doesn't match the `plugins.Plugin` interface

**Solution:** Ensure your type implements all interface methods:
- Name() string
- Register(Registry, Scheme) error
- Unregister(Registry) error

### Import Errors

**Problem:** `cannot find module`, `missing go.sum entry`

**Solution:** Ensure your plugin imports match the server's go.mod:
```go
import (
    "github.com/pergus/api-server/pkg/api"
    "github.com/pergus/api-server/pkg/plugins"
)
```

### Binary Size Too Large

Plugin .so files can be large (5-10 MB) because they include the Go runtime.

To reduce size, you could:
1. Use `upx` to compress (creates smaller but slower binary)
2. Accept the size (disk space is cheap)
3. Build with specific flags: `go build -ldflags="-s -w" -buildmode=plugin`

## Plugin vs CRD

Both plugins and CRDs add resources, but they differ:

| Aspect | Plugin | CRD |
|--------|--------|-----|
| Definition | Compiled Go code | JSON/YAML |
| Business Logic | Yes (Go code) | No (schema only) |
| Storage | Custom possible | Dynamic objects only |
| Performance | Fastest | Fast |
| Complexity | High | Low |
| Reload | Requires restart | No restart needed |

Use **plugins** for: custom storage, validation logic, complex behavior
Use **CRDs** for: simple schemas, rapid iteration, user-defined resources

## Next Steps

1. Create your plugin following this guide
2. Build it: `go build -buildmode=plugin -o yourplugin.so`
3. Deploy: `cp yourplugin.so plugins/`
4. Test via API or CLI
5. Add custom logic in your Reconcile methods (see CONTROLLER_GUIDE.md)

See the example plugin at `plugins/invoices/main.go` for a complete working implementation.
