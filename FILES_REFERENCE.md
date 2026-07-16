# Files Reference

Complete index of all files in the extended dynamic API server project.

## Core Framework Files

### Server Entry Point
- **`cmd/server/main.go`** - Server initialization
  - Registers built-in resources
  - Creates API framework
  - Starts HTTP server
  - Manages graceful shutdown

### API Framework

#### Generic Routing
- **`pkg/api/router.go`** (~350 lines) - HTTP request dispatcher
  - Generic handlers (list, get, create, update, delete)
  - Discovery endpoints (/api, /apis)
  - CRD management endpoints (/crds)
  - API path discovery endpoints

#### Resource Management
- **`pkg/api/registry.go`** (~150 lines) - Thread-safe resource registry
  - Register/unregister resources at runtime
  - Lookup by name (called on every request)
  - List all registered resources

- **`pkg/api/resource.go`** (~30 lines) - Resource interface definition
  - Name() - Resource identifier
  - NewObject() - Object factory
  - Storage() - Persistence layer

#### Type Factory System
- **`pkg/api/scheme.go`** (~100 lines) - Type factory registry
  - Register type constructors by name
  - Create objects without importing types
  - Thread-safe read/write access

#### Storage Interface
- **`pkg/api/storage.go`** (~150 lines) - Abstract persistence layer
  - MemoryStorage implementation
  - List, Get, Create, Update, Delete operations
  - ID extraction from objects

#### Server Configuration
- **`pkg/api/server.go`** (~120 lines) - HTTP server wrapper
  - Registry and Scheme accessors
  - CRDRegistry accessor
  - Startup and shutdown management

#### Common Types
- **`pkg/api/types.go`** (~60 lines) - Shared response types
  - ListResponse, ErrorResponse
  - DiscoveryResponse
  - RequestTiming

#### Middleware
- **`pkg/api/middleware.go`** (~100 lines) - HTTP middleware
  - Logging, CORS, recovery, timing

## CRD System Files (NEW)

### CRD Definitions and Management
- **`pkg/api/crd.go`** (~140 lines) - Custom Resource Definition system
  - CRDDefinition struct
  - CRDRegistry interface and implementation
  - Thread-safe CRD storage
  - Validation and naming conventions

### Dynamic Objects
- **`pkg/api/dynamic.go`** (~180 lines) - Generic object representation
  - DynamicObject struct for schema-less data
  - SimpleDynamicResource wrapper for CRDs
  - Custom JSON marshalling/unmarshalling
  - Metadata handling (ID extraction)

## Built-in Resources

- **`pkg/resources/users.go`** (~40 lines) - User resource
- **`pkg/resources/products.go`** (~40 lines) - Product resource
- **`pkg/resources/orders.go`** (~40 lines) - Order resource

## Plugin System

- **`pkg/plugins/interface.go`** (~30 lines) - Plugin interface
- **`pkg/plugins/loader.go`** (~200 lines) - Plugin loading and watching
- **`plugins/invoices/main.go`** (~130 lines) - Example invoice plugin
- **`plugins/build.sh`** - Script to build all plugins

## apitcl Client (NEW)

### Client Implementation
- **`cmd/apitcl/main.go`** (~50 lines) - CLI entry point and help
  - Command parsing
  - Usage information

- **`cmd/apitcl/client.go`** (~180 lines) - HTTP API client
  - REST communication with server
  - Discovery methods
  - CRUD operations
  - CRD management

- **`cmd/apitcl/commands.go`** (~300 lines) - Command implementations
  - `api-resources` - List resources
  - `api-versions` - List API groups
  - `get` - Retrieve resources
  - `create` - Create from file
  - `delete` - Delete resources
  - `apply` - Apply CRDs and resources
  - `explain` - Show resource schema

## Examples

- **`examples/invoice-crd.yaml`** - CRD definition in YAML format
  - Group: example.io
  - Version: v1
  - Kind: Invoice
  - Plural: invoices

- **`examples/invoice-1.json`** - Sample Invoice object
  - Demonstrates JSON format
  - Ready to use with create command

- **`examples/DEMO.md`** (~200 lines) - Detailed demonstration guide
  - Step-by-step walkthrough
  - All features explained
  - How it works sections

## Documentation

### Quick References
- **`QUICKSTART.md`** (~120 lines) - 5-minute getting started
  - Build instructions
  - Demo sequence
  - Key concepts
  - Next steps

- **`IMPLEMENTATION_SUMMARY.md`** (~300 lines) - Summary of changes
  - What was added
  - How features work
  - Educational value
  - Architecture diagrams

### Architecture Documentation
- **`ARCHITECTURE.md`** (~280 lines) - Core framework architecture
  - Problem and solution
  - Component architecture
  - Design patterns
  - How it mirrors Kubernetes

- **`CRD_ARCHITECTURE.md`** (~280 lines) - CRD system deep dive
  - CRD system components
  - Lifecycle of CRDs
  - API discovery design
  - apitcl architecture
  - Thread safety analysis
  - Performance characteristics

### Build and Setup
- **`BUILD.md`** (~300 lines) - Complete build instructions
  - Build procedures
  - Running examples
  - Testing via curl
  - Using apitcl client
  - Troubleshooting

- **`README.md`** (~470 lines) - Main project documentation
  - Overview
  - Quick start (both CRD and plugin approaches)
  - Using the API
  - CRD system explanation
  - apitcl usage
  - Creating plugins
  - Architecture details

## Helper Scripts

- **`demo.sh`** (~100 lines) - Automated demonstration
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
│   ├── server/
│   │   └── main.go                    (Server entry)
│   └── apitcl/
│       ├── main.go                    (CLI entry)
│       ├── client.go                  (HTTP client)
│       └── commands.go                (CLI commands)
│
├── pkg/
│   ├── api/
│   │   ├── router.go                  (HTTP routing)
│   │   ├── registry.go                (Resource registry)
│   │   ├── resource.go                (Resource interface)
│   │   ├── scheme.go                  (Type factory)
│   │   ├── storage.go                 (Storage interface)
│   │   ├── server.go                  (Server wrapper)
│   │   ├── types.go                   (Common types)
│   │   ├── middleware.go              (HTTP middleware)
│   │   ├── crd.go                     (CRD system) NEW
│   │   └── dynamic.go                 (Dynamic objects) NEW
│   │
│   ├── plugins/
│   │   ├── interface.go               (Plugin interface)
│   │   └── loader.go                  (Plugin loading)
│   │
│   └── resources/
│       ├── users.go
│       ├── products.go
│       └── orders.go
│
├── plugins/
│   ├── invoices/
│   │   └── main.go                    (Invoice plugin)
│   └── build.sh
│
├── examples/
│   ├── invoice-crd.yaml               (CRD example)
│   ├── invoice-1.json                 (Object example)
│   └── DEMO.md                        (Detailed demo)
│
├── go.mod                             (Module definition)
├── README.md                          (Main documentation)
├── QUICKSTART.md                      (Quick start guide) NEW
├── ARCHITECTURE.md                    (Core architecture)
├── CRD_ARCHITECTURE.md                (CRD design) NEW
├── IMPLEMENTATION_SUMMARY.md          (Implementation summary) NEW
├── BUILD.md                           (Build instructions)
├── FILES_REFERENCE.md                 (This file) NEW
└── demo.sh                            (Demo script) NEW
```

## Key Statistics

### Code
- New code: ~1,900 lines
- Total framework: ~2,500 lines
- Documentation: ~2,000 lines
- Examples: ~100 lines

### Files
- New source files: 5
- New documentation: 5
- Modified files: 6
- Example files: 3
- Helper scripts: 1

### Binary Sizes
- Server: ~12 MB (includes Go runtime)
- apitcl: ~8.7 MB (includes Go runtime)

## How to Use This Reference

1. **For quick start:** Read QUICKSTART.md
2. **For deep dive:** Read CRD_ARCHITECTURE.md
3. **For building:** Read BUILD.md
4. **For understanding architecture:** Read ARCHITECTURE.md
5. **For implementation details:** Read IMPLEMENTATION_SUMMARY.md
6. **For running demo:** Execute `./demo.sh`

## Highlighted Files

### Must Read
1. **pkg/api/router.go** - The heart of extensibility
   - Shows how generic handlers work
   - Demonstrates resource lookup pattern
   - Illustrates CRD registration flow

2. **pkg/api/crd.go** - CRD system implementation
   - Simple CRD definition structure
   - Thread-safe registry pattern
   - Validation logic

3. **pkg/api/dynamic.go** - Dynamic object system
   - Schema-less data storage
   - Flexible JSON handling
   - Kubernetes-like object representation

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
go build -o server ./cmd/server
go build -o apitcl ./cmd/apitcl
```

### Run Server
```bash
./server
```

### Run Demo
```bash
./demo.sh
```

### Manual Testing
```bash
./apitcl api-resources
./apitcl apply -f examples/invoice-crd.yaml
./apitcl create -f examples/invoice-1.json
./apitcl get invoices
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
