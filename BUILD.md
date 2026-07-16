# Build Instructions

## Prerequisites

- Go 1.21 or later
- Unix-like system (Linux, macOS)
  - Go plugin system uses shared objects (.so files)
  - Not supported on Windows

## Building the Server and Client

### Build Both Binaries

```bash
# Build api-server
go build -o api-server ./cmd/api-server

# Build apictl client
go build -o apictl ./cmd/apictl
```

This creates:
- `api-server` - The API server with CRD support
- `apictl` - The discovery-based CLI client

## Building Plugins

### Build Single Plugin

```bash
cd plugins/invoices
go build -buildmode=plugin -o invoices.so main.go
```

This creates `invoices.so` which can be loaded at runtime.

### Build All Plugins

```bash
cd plugins
chmod +x build.sh
./build.sh
```

This builds all plugins in the plugins directory.

## Creating a Plugin Directory

```bash
mkdir -p plugins
```

The server watches this directory for new .so files.

## Running the Server

```bash
./api-server
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

## Complete Build-and-Run Example (CRD System)

### Quick Start with CRDs

```bash
# 1. Build server and client
go build -o server ./cmd/server
go build -o apictl ./cmd/apictl

# 2. Start the server (in one terminal)
./api-server

# 3. In another terminal, use apictl
# List built-in resources
./apictl api-resources

# 4. Apply a CRD
./apictl apply -f examples/invoice-crd.yaml

# 5. Verify the resource appeared
./apictl api-resources

# 6. Create an invoice
./apictl create -f examples/invoice-1.json

# 7. List invoices
./apictl get invoices

# 8. Get a specific invoice
./apictl get invoices inv-001

# 9. Delete the CRD
./apictl delete crd invoices.example.io

# 10. Verify it's gone
./apictl api-resources
```

See `examples/DEMO.md` for a detailed walkthrough.

## Classic Plugin System Build

```bash
# 1. Build server
go build -o server ./cmd/server

# 2. Create plugins directory
mkdir -p plugins

# 3. Build plugins
cd plugins/invoices
go build -buildmode=plugin -o invoices.so main.go
cd ../..

# 4. Run server
./api-server
```

## Runtime Plugin Loading

While the server is running:

```bash
# In another terminal
cp plugins/invoices/invoices.so plugins/
```

The server will detect the new plugin within 2 seconds and load it automatically.

## Verifying the Build

### Check Server Binary

```bash
./api-server --help  # May show no output (server has no flags)
file server      # Should show: ELF 64-bit executable
```

### Check Plugin Binary

```bash
file plugins/invoices/invoices.so
# Should show: ELF 64-bit shared object
```

### Test API via curl

Start the server in one terminal:

```bash
./api-server
```

In another terminal, test the API:

```bash
# Discovery - list all resources
curl http://localhost:8080/api | jq

# Create user
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"id":"test","name":"Test","email":"test@example.com","is_active":true}' | jq

# List users
curl http://localhost:8080/api/users | jq

# Create a CRD
curl -X POST http://localhost:8080/crds \
  -H "Content-Type: application/json" \
  -d '{
    "group": "example.io",
    "version": "v1",
    "kind": "Invoice",
    "plural": "invoices",
    "schema": {"id": "string", "customer": "string"}
  }' | jq

# List CRDs
curl http://localhost:8080/crds | jq

# Create an invoice
curl -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{"id":"inv-1","customer":"Acme"}' | jq

# List invoices
curl http://localhost:8080/api/invoices | jq
```

### Using apictl Client

After building both server and client:

```bash
# Start server
./api-server

# In another terminal, discover resources
./apictl api-resources
./apictl api-versions

# Create a CRD from YAML
./apictl apply -f examples/invoice-crd.yaml

# Create an object
./apictl create -f examples/invoice-1.json

# List objects
./apictl get invoices

# Get specific object
./apictl get invoices inv-001

# Explain a resource
./apictl explain invoices

# Delete an object
./apictl delete invoices inv-001

# Delete a CRD
./apictl delete crd invoices.example.io
```

## Creating Your Own Plugin

1. Create plugin directory:
```bash
mkdir -p plugins/myplugin
```

2. Create main.go in that directory with a plugin implementation

3. Build the plugin:
```bash
cd plugins/myplugin
go build -buildmode=plugin -o myplugin.so main.go
```

4. Copy to plugins directory:
```bash
cp myplugin.so ../ 
```

The server will load it automatically.

See `plugins/invoices/main.go` for a complete example.

## Troubleshooting

### "unsupported GOOS/GOARCH pair windows/amd64"

The plugin system only works on Unix-like systems. Use Linux or macOS.

### "plugin already loaded"

Go's plugin system caches loaded plugins. Restart the server to reload a modified plugin.

### "Plugin symbol is not of type Plugin"

Ensure your plugin exports a `Plugin` symbol:
```go
var Plugin plugins.Plugin = &MyPlugin{}
```

### Plugin not loading

Check that:
1. .so file is in the `plugins/` directory
2. File extension is `.so`
3. Server is running (check logs for "Scanning for existing plugins")
4. Plugin imports are correct

The server logs plugin loading. Watch the logs to see if plugins are detected.

## Build Targets

### Source Files

```
cmd/server/main.go              ~150 lines
pkg/api/router.go               ~250 lines
pkg/api/registry.go             ~120 lines
pkg/api/scheme.go               ~80 lines
pkg/api/server.go               ~100 lines
pkg/api/storage.go              ~150 lines
pkg/api/resource.go             ~30 lines
pkg/api/middleware.go           ~100 lines
pkg/api/types.go                ~50 lines
pkg/plugins/loader.go           ~200 lines
pkg/plugins/interface.go        ~30 lines
pkg/resources/users.go          ~50 lines
pkg/resources/products.go       ~50 lines
pkg/resources/orders.go         ~50 lines
plugins/invoices/main.go        ~130 lines
```

Total: ~1,600 lines of Go code

### Binary Sizes

- Server binary: ~12 MB
- Invoices plugin: ~9 MB

(These sizes include Go runtime. Not heavily optimized.)

## Development

### Makefile Targets

The project includes a comprehensive Makefile for common development tasks:

```bash
# Build binaries
make build

# Run the server
make run

# Run tests
make test

# Generate coverage report
make test-coverage

# Format code
make fmt

# Lint code
make lint

# Advanced linting (requires install)
make staticcheck

# Check for dead code (requires install)
make deadcode

# Run all verification checks
make verify

# Show all available targets
make help
```

### Running Tests

```bash
go test ./...
```

Comprehensive tests are included with 79%+ coverage across core packages.

### Formatting Code

```bash
go fmt ./...
```

### Linting

```bash
go vet ./...
```

### Optional Code Quality Tools

The project includes Makefile targets for advanced static analysis. These tools are **optional** but recommended for code quality:

#### Installing Optional Tools

**staticcheck** - Advanced Go linter with additional checks:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

**deadcode** - Find unused code:

```bash
go install golang.org/x/tools/cmd/deadcode@latest
```

#### Running Code Quality Checks

Once installed, run the tools via Make:

```bash
# Run just staticcheck
make staticcheck

# Run just deadcode
make deadcode

# Run all quality checks (build, test, lint, staticcheck, deadcode)
make verify
```

Without these tools installed, you can still run:
- `make lint` - Uses go vet (built-in)
- `make test` - Run tests
- `make build` - Build binaries

## Performance

### Build Time

```
Server: ~1-2 seconds
Plugin: ~1-2 seconds
```

### Start Time

Server starts immediately. Plugin loading happens in background (2-second poll).

### Request Performance

- Discovery: <1ms
- List: <1ms
- Get: <1ms  
- Create: <1ms
- Update: <1ms
- Delete: <1ms

(In-memory storage; actual performance depends on storage backend.)

## Notes

- Go plugins are only supported on Linux and macOS
- Windows users can modify the code to use gRPC-based plugins instead
- Plugin API is stable; can be extended with versioning

## References

- [Go Plugin Documentation](https://golang.org/pkg/plugin/)
- [Go Build Modes](https://golang.org/cmd/go/#hdr-Build_modes)
