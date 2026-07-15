# Makefile Guide

The Makefile provides convenient commands for building, running, and testing the dynamic API server.

## Quick Start

```bash
# Build both binaries
make build

# Terminal 1: Run the server
make run

# Terminal 2: Run the demo
make demo
```

## Build Targets

### `make build`
Builds both the server and kubectl-lite client binaries.

```bash
make build
# Output: ✓ Build complete: server and kubectl-lite
```

### `make server`
Builds only the server binary.

```bash
make server
# Creates: ./server
```

### `make client`
Builds only the kubectl-lite client binary.

```bash
make client
# Creates: ./kubectl-lite
```

### `make all`
Same as `make build` - builds everything.

## Running

### `make run` or `make run-server`
Starts the server on http://localhost:8080

```bash
make run
# Listens on port 8080
# Press Ctrl+C to stop
```

### `make demo`
Runs the automated demonstration script.

Requires both binaries to be built first.

```bash
make demo
# Runs through 10 steps automatically
# Shows colored output
# Includes summary and takeaways
```

## Testing

### `make test`
Runs all tests with race detection and coverage.

```bash
make test
# Runs: go test -v -race -cover ./...
```

### `make test-coverage`
Generates a detailed coverage report.

```bash
make test-coverage
# Outputs: coverage.html
# Open in browser to see coverage visualization
```

## Code Quality

### `make fmt`
Formats all Go code according to standard conventions.

```bash
make fmt
# Runs: go fmt ./...
```

### `make lint`
Runs the Go linter (vet) to check for potential issues.

```bash
make lint
# Runs: go vet ./...
```

### `make verify`
Runs build, test, and lint in sequence.

```bash
make verify
# Checks everything is working correctly
```

## Dependency Management

### `make install-deps`
Downloads and tidies Go dependencies.

```bash
make install-deps
# Runs: go mod download && go mod tidy
```

## Cleanup

### `make clean`
Removes build artifacts and temporary files.

```bash
make clean
# Removes: server, kubectl-lite, coverage.out, coverage.html
```

### `make clean-all`
Performs full cleanup including tidying modules.

```bash
make clean-all
# Full cleanup
```

## Development

### `make dev`
Runs a development build: installs deps, formats, lints, and builds.

```bash
make dev
# Complete development build
```

## Project Info

### `make info`
Shows project information.

```bash
make info
# Displays: project name, binaries, Go version, module info
```

## Setup

### `make setup-plugins`
Creates the plugins directory for the plugin system.

```bash
make setup-plugins
# Creates: ./plugins/
```

### `make prepare-demo`
Prepares everything for running a demo.

```bash
make prepare-demo
# Installs deps, builds binaries, creates plugins dir
```

## API Shortcuts

These targets interact with a running server (started with `make run`).

### `make api-resources`
Lists all available resources.

```bash
make api-resources
# Runs: ./kubectl-lite api-resources
```

### `make api-versions`
Lists all API groups.

```bash
make api-versions
# Runs: ./kubectl-lite api-versions
```

### `make create-crd`
Creates the Invoice CRD from the example.

```bash
make create-crd
# Runs: ./kubectl-lite apply -f examples/invoice-crd.yaml
# Requires: running server
```

### `make create-invoice`
Creates a sample invoice object.

```bash
make create-invoice
# Runs: ./kubectl-lite create -f examples/invoice-1.json
# Requires: running server, CRD created
```

### `make get-invoices`
Lists all invoice objects.

```bash
make get-invoices
# Runs: ./kubectl-lite get invoices
# Requires: running server, CRD created
```

### `make integration-flow`
Runs full integration flow: creates CRD, creates invoice, lists invoices.

```bash
make integration-flow
# Requires: running server
```

## Help

### `make help`
Shows the main help message with all available targets.

```bash
make help
```

### `make help-quickstart`
Shows a quick start guide.

```bash
make help-quickstart
```

## Workflow Examples

### Complete Demo Workflow

Terminal 1:
```bash
make build     # Build binaries
make run       # Start server
```

Terminal 2:
```bash
make demo      # Run automated demo
```

### Manual Workflow

Terminal 1:
```bash
make run       # Start server
```

Terminal 2:
```bash
make api-resources         # List initial resources
make create-crd            # Create Invoice CRD
make api-resources         # List resources again (invoices appears!)
make create-invoice        # Create sample invoice
make get-invoices          # List invoices
```

### Development Workflow

```bash
make install-deps  # Get dependencies
make dev           # Full development build
make run           # Start server
```

In another terminal:
```bash
make test          # Run tests
make lint          # Check code quality
make fmt           # Format code
```

### Continuous Integration

```bash
make verify        # Build + test + lint
make test-coverage # Generate coverage report
```

## Advanced Usage

### Building from Scratch

```bash
make clean         # Remove old binaries
make build         # Build fresh
```

### Full Development Setup

```bash
make prepare-demo  # Install deps, build, setup
make run           # Start server
```

### Code Quality Check

```bash
make fmt           # Format
make lint          # Lint
make test          # Test
```

## Makefile Variables

You can override the default variables:

```bash
# Use a different binary name
make server BINARY_SERVER=my-server

# Use different Go compiler flags
make build GOFLAGS="-x -v"
```

## Common Issues

### "command not found: make"
Install make:
- macOS: `brew install make`
- Linux: `apt-get install make` or `yum install make`
- Windows: Use WSL or install GNU Make

### Port 8080 already in use
Change the port in the server code or kill the existing process:
```bash
lsof -i :8080
kill -9 <PID>
```

### "Cannot find package"
Run:
```bash
make install-deps
make clean
make build
```

## Makefile Organization

The Makefile is organized into sections:

1. **Variables** - Configuration (binary names, flags)
2. **Build targets** - Compiling binaries
3. **Run targets** - Starting server and demos
4. **Test targets** - Testing and coverage
5. **Code quality** - Formatting and linting
6. **Dependencies** - Managing Go modules
7. **Cleanup** - Removing artifacts
8. **Quick start** - Convenient shortcuts
9. **Development** - Development workflow
10. **Info** - Project information
11. **API shortcuts** - Interact with running server
12. **Help** - Documentation targets

## Tips

- Use `make` (no argument) to see all targets
- Use `make help` for detailed help
- Use tab completion: `make <TAB><TAB>` in bash
- Chain targets: `make clean build run`
- Run specific workflow: `make prepare-demo && make run`

## See Also

- `QUICKSTART.md` - Quick start guide
- `BUILD.md` - Detailed build instructions
- `TESTING.md` - Testing procedures
- `README.md` - Main documentation
