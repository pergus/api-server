# Files Reference

Complete index of all files in the extended dynamic API server project.

## Core Framework Files

### Server Entry Point
- **`cmd/api-server/main.go`** - Server initialization
  - Registers built-in resources (users, products, orders)
  - Registers CRDs for built-in resources
  - Creates API framework
  - Initializes plugin system
  - Initializes controller manager
  - Starts HTTP server
  - Manages graceful shutdown

### API Framework

#### Generic Routing
- **`pkg/api/router.go`** - HTTP request dispatcher
  - Generic handlers (list, get, create, update, delete)
  - Discovery endpoints (/api, /apis)
  - CRD management endpoints (/crds)
  - API path discovery endpoints

#### Resource Management
- **`pkg/api/registry.go`** - Thread-safe resource registry
  - Register/unregister resources at runtime
  - Lookup by name (called on every request)
  - List all registered resources

- **`pkg/api/resource.go`** - Resource interface definition
  - Name() - Resource identifier
  - NewObject() - Object factory
  - Storage() - Persistence layer

#### Type Factory System
- **`pkg/api/scheme.go`** - Type factory registry
  - Register type constructors by name
  - Create objects without importing types
  - Thread-safe read/write access

#### Storage Interface
- **`pkg/api/storage.go`** - Abstract persistence layer
  - MemoryStorage implementation
  - List, Get, Create, Update, Delete operations
  - ID extraction from objects

#### Server Configuration
- **`pkg/api/server.go`** - HTTP server wrapper
  - Registry and Scheme accessors
  - CRDRegistry accessor
  - Startup and shutdown management

#### Common Types
- **`pkg/api/types.go`** - Shared response types
  - ListResponse, ErrorResponse
  - DiscoveryResponse
  - RequestTiming

#### Middleware
- **`pkg/api/middleware.go`** - HTTP middleware
  - LoggingMiddleware, CORSMiddleware, RecoveryMiddleware, TimingMiddleware
  - Middleware chaining support

#### Event System
- **`pkg/api/event.go`** - Event model and types
  - Event struct with Type, Resource, Object, Timestamp
  - EventType constants (Added, Modified, Deleted)
  - Subscription management

- **`pkg/api/eventbus.go`** - Event pub/sub system
  - EventBus interface
  - InProcessEventBus implementation
  - Thread-safe publish/subscribe with goroutine fan-out
  - Subscription with buffered event channels (100 buffer)

#### Watch API
- **`pkg/api/router.go`** includes watch() method
  - GET /api/{resource}?watch=true for Server-Sent Events
  - Keeps connection alive with keep-alive ticker (5 second interval)
  - SSE formatted event streaming

## Controller Framework Files

- **`pkg/controllers/controller.go`** - Controller interface
  - Name(), Resource(), Reconcile(Event), Run(ctx) methods
  - Pattern for event-driven reconciliation

- **`pkg/controllers/manager.go`** - ControllerManager
  - Register controllers at runtime
  - Concurrent execution in separate goroutines
  - Event subscription and delivery

- **`pkg/controllers/orders.go`** - OrderController example
  - Demonstrates reconciliation pattern
  - Reacts to ADDED (set status=processing)
  - Handles MODIFIED and DELETED events
  - Shows resource updates triggering new events

## CRD System Files (NEW)

### CRD Definitions and Management
- **`pkg/api/crd.go`** - Custom Resource Definition system
  - CRDDefinition struct
  - CRDRegistry interface and implementation
  - Thread-safe CRD storage
  - Validation and naming conventions

### Dynamic Objects
- **`pkg/api/dynamic.go`** - Generic object representation
  - DynamicObject struct for schema-less data
  - DynamicResource wrapper for CRDs
  - Custom JSON marshalling/unmarshalling
  - Metadata handling (ID extraction)

## Built-in Resources

- **`pkg/resources/users.go`** - User resource
- **`pkg/resources/products.go`** - Product resource
- **`pkg/resources/orders.go`** - Order resource

## Plugin System

- **`pkg/plugins/interface.go`** - Plugin interface
- **`pkg/plugins/loader.go`** - Plugin loading and watching
- **`pkg/api/plugins.go`** - Plugin provider interface
- **`plugins/invoices/main.go`** - Example invoice plugin
- **`plugins/build.sh`** - Script to build all plugins

## apictl Client

### Client Implementation
- **`cmd/apictl/main.go`** - CLI entry point and help
  - Command parsing
  - Usage information and documentation

- **`cmd/apictl/client.go`** - HTTP API client
  - REST communication with server
  - Discovery methods (GetAPIResources, GetAPIs)
  - CRUD operations (Create, Get, Update, Delete, List)
  - CRD management (CreateCRD, ListCRDs, DeleteCRD)
  - Watch streaming with SSE parsing
  - Error handling with separate error channel
  - Unlimited line size support (no 64KiB token limit)

- **`cmd/apictl/commands.go`** - Command implementations
  - `api-resources` - List available resources
  - `api-versions` - List API groups
  - `plugins` - List loaded plugins and count
  - `get` - Retrieve resources (list all or get specific)
  - `create` - Create resource from file
  - `delete` - Delete resource
  - `apply` - Apply CRDs and resources (create/upsert)
  - `explain` - Show resource schema
  - `watch` - Stream resource events
  - Helper functions: pluralize, extractID, convertMap

## Examples

- **`examples/invoice-crd.yaml`** - CRD definition in YAML format
  - Group: example.io
  - Version: v1
  - Kind: Invoice
  - Plural: invoices

- **`examples/invoice-1.json`** - Sample Invoice object
  - Demonstrates JSON format
  - Ready to use with create command


## Documentation

### Quick References
- **`QUICKSTART.md`** - 5-minute getting started
  - Build instructions
  - Demo sequence
  - Key concepts
  - Next steps

- **`IMPLEMENTATION_SUMMARY.md`** - Summary of changes
  - What was added
  - How features work
  - Educational value
  - Architecture diagrams

### Architecture Documentation
- **`ARCHITECTURE.md`** - Core framework architecture
  - Problem and solution
  - Component architecture
  - Design patterns

- **`CRD_ARCHITECTURE.md`** - CRD system deep dive
  - CRD system components
  - Lifecycle of CRDs
  - API discovery design
  - apitcl architecture
  - Thread safety analysis
  - Performance characteristics

### Build and Setup
- **`BUILD.md`** - Complete build instructions
  - Build procedures
  - Running examples
  - Testing via curl
  - Using apitcl client
  - Troubleshooting

- **`DEMO.md`** - Detailed demonstration guide
  - Step-by-step walkthrough
  - All features explained
  - How it works sections

- **`README.md`** - Main project documentation
  - Overview
  - Quick start (both CRD and plugin approaches)
  - Using the API
  - CRD system explanation
  - apitcl usage
  - Creating plugins
  - Architecture details

## Test Files

### API Tests
- **`pkg/api/eventbus_test.go`** - EventBus tests (5 tests)
- **`pkg/api/router_test.go`** - Router tests (3 tests)
- **`pkg/api/integration_test.go`** - Integration tests (5 tests + 2 benchmarks)
- **`pkg/api/crd_test.go`** - CRD system tests (7 tests)
- **`pkg/api/discovery_test.go`** - API discovery tests (6 tests)
- **`pkg/api/error_test.go`** - Error handling tests (9 tests)
- **`pkg/api/middleware_test.go`** - Middleware tests (14 tests)
- **`pkg/api/scheme_test.go`** - Type scheme tests (7 tests)
- **`pkg/api/server_test.go`** - Server tests (10 tests)
- **`pkg/api/storage_test.go`** - Storage tests (14 tests)
- **`pkg/api/test_helpers.go`** - Shared test utilities

### Resource Tests
- **`pkg/resources/resources_test.go`** - Built-in resource tests (8 tests)

### Controller Tests
- **`pkg/controllers/controller_test.go`** - Controller tests (4 tests)
- **`pkg/controllers/manager_test.go`** - ControllerManager tests (13 tests)

### CLI Tests
- **`cmd/apictl/client_test.go`** - Client HTTP method tests (11 tests)
- **`cmd/apictl/commands_test.go`** - CLI helper function tests (5 tests)

## Build and Development Files

- **`Makefile`** - Build automation with 30+ targets
  - Build targets: build, server, client, all, clean
  - Test targets: test, test-coverage
  - Quality targets: fmt, lint, staticcheck, deadcode
  - Utility targets: info, verify, help
  - Demo targets: api-resources, api-versions, create-crd, get-invoices

## Helper Scripts

- **`demo.sh`** - Automated demonstration
  - Runs complete demo sequence
  - Colored output
  - Validation steps
  - Summary and takeaways
  - Usage: `chmod +x demo.sh && ./demo.sh`

## Project Configuration

- **`go.mod`** - Go module definition
  - Module name: github.com/example/api-server
  - Go version: 1.21
  - Dependencies: gopkg.in/yaml.v2

## Directory Structure

```
api-server/
├── cmd/
│   ├── api-server/
│   │   └── main.go                    (Server entry point)
│   └── apictl/
│       ├── main.go                    (CLI entry point)
│       ├── client.go                  (HTTP API client)
│       ├── commands.go                (CLI command handlers)
│       ├── client_test.go             (Client tests)
│       └── commands_test.go           (Command tests)
│
├── pkg/
│   ├── api/
│   │   ├── router.go                  (HTTP request dispatcher)
│   │   ├── registry.go                (Resource registry)
│   │   ├── resource.go                (Resource interface)
│   │   ├── scheme.go                  (Type factory system)
│   │   ├── storage.go                 (Storage abstraction)
│   │   ├── server.go                  (Server wrapper)
│   │   ├── types.go                   (Response types)
│   │   ├── middleware.go              (HTTP middleware)
│   │   ├── crd.go                     (CRD system)
│   │   ├── dynamic.go                 (Dynamic resources)
│   │   ├── event.go                   (Event model)
│   │   ├── eventbus.go                (Event pub/sub)
│   │   ├── test_helpers.go            (Shared test utilities)
│   │   ├── eventbus_test.go
│   │   ├── router_test.go
│   │   ├── integration_test.go
│   │   ├── crd_test.go
│   │   ├── discovery_test.go
│   │   ├── error_test.go
│   │   ├── middleware_test.go
│   │   ├── scheme_test.go
│   │   ├── server_test.go
│   │   └── storage_test.go
│   │
│   ├── controllers/
│   │   ├── controller.go              (Controller interface)
│   │   ├── manager.go                 (ControllerManager)
│   │   ├── orders.go                  (OrderController example)
│   │   ├── controller_test.go
│   │   └── manager_test.go
│   │
│   ├── plugins/
│   │   ├── interface.go               (Plugin interface)
│   │   └── loader.go                  (Plugin loading system)
│   │
│   └── resources/
│       ├── resources.go               (Built-in resources)
│       ├── users.go
│       ├── products.go
│       ├── orders.go
│       └── resources_test.go
│
├── plugins/
│   ├── invoices/
│   │   └── main.go                    (Invoice plugin example)
│   └── build.sh                       (Plugin build script)
│
├── examples/
│   ├── invoice-crd.yaml               (CRD example in YAML)
│   └── invoice-1.json                 (Object example in JSON)
│
├── Makefile                           (Build automation)
├── go.mod                             (Go module definition)
├── go.sum                             (Go module checksums)
├── README.md                          (Main project documentation)
├── BUILD.md                           (Build and development guide)
├── QUICKSTART.md                      (Quick start guide)
├── BOOK.md                            (A book describing the implementation)
├── DEMO.md                            (Detailed demonstration)
├── ARCHITECTURE.md                    (Core architecture explanation)
├── CRD_ARCHITECTURE.md                (CRD system deep dive)
├── WATCH_ARCHITECTURE.md              (Watch API & event system)
├── IMPLEMENTATION_SUMMARY.md          (Feature implementation summary)
├── FILES_REFERENCE.md                 (This file)
└── demo.sh                            (Automated demo script)
```

## How to Maintain This Reference

This file should be updated when:
- New source files are added
- Files are deleted or significantly refactored
- New documentation is created
- Directory structure changes
- Major features are added

Do NOT update for:
- Line count changes (source code evolves constantly)
- Binary size changes (depends on Go version and optimizations)
- Minor code edits

## How to Use This Reference

1. **For quick start:** Read QUICKSTART.md
2. **For deep dive:** Read CRD_ARCHITECTURE.md
3. **For building:** Read BUILD.md
4. **For understanding architecture:** Read ARCHITECTURE.md
5. **For implementation details:** Read IMPLEMENTATION_SUMMARY.md or BOOK.md
6. **For running demo:** Execute `./demo.sh`

## Highlighted Files

### Must Read
1. **pkg/api/router.go** - The heart of extensibility
   - Shows how generic handlers work
   - Demonstrates resource lookup pattern
   - Illustrates CRD registration flow

2. **pkg/api/crd.go** - CRD system implementation
   - CRD definition structure
   - Thread-safe registry pattern
   - Validation logic

3. **pkg/api/dynamic.go** - Dynamic object system
   - Schema-less data storage
   - Flexible JSON handling
   - Object representation

4. **cmd/apitcl/client.go** - Discovery-based client
   - Shows how clients discover APIs
   - Demonstrates no hardcoded resources
   - REST API interaction pattern

### Educational Value
- **pkg/api/registry.go** - Thread-safe read/write with RWMutex
- **pkg/api/scheme.go** - Factory pattern and type abstraction
- **pkg/api/storage.go** - Interface-based persistence

## Building and Running

### Build Everything
```bash
make build
# or
go build -o bin/api-server ./cmd/api-server
go build -o bin/apictl ./cmd/apictl
```

### Run Server
```bash
make run
# or
./bin/api-server
```

### Run Tests
```bash
make test
# or
go test ./...
```

### Run All Quality Checks
```bash
make verify
# Runs: build → test → lint → staticcheck → deadcode
```

### Manual Testing
```bash
./bin/apictl api-resources
./bin/apictl api-versions
./bin/apictl plugins
./bin/apictl apply -f examples/invoice-crd.yaml
./bin/apictl create -f examples/invoice-1.json
./bin/apictl get invoices
./bin/apictl watch orders
```

## Testing Checklist

- [ ] Server builds without errors
- [ ] apitcl builds without errors
- [ ] Server starts and listens on port 8080
- [ ] `/api` endpoint returns built-in resources
- [ ] `/crds` POST creates new CRD
- [ ] `/crds` GET lists CRDs
- [ ] `/apis` lists API groups
- [ ] New resource available after CRD creation
- [ ] apitcl discovers resources dynamically
- [ ] apitcl create works with CRD resources
- [ ] apitcl get lists CRD resources
- [ ] CRD deletion removes resource from discovery

## Notes

- All code is Go 1.21+
- Uses standard library HTTP and JSON
- No external dependencies except YAML parser
- Thread-safe for concurrent access
- Extensible for future enhancements
- Educational focus on clarity over performance
