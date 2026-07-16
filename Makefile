.PHONY: help build server client all clean run run-server demo test install-deps fmt lint deadcode

# Variables
BINARY_SERVER := bin/api-server
BINARY_CLIENT := bin/apictl
GO := go
GOFLAGS := -v
COVERAGE_FILE := coverage.out

# Default target
help:
	@echo "Dynamic API Server with CRDs - Makefile"
	@echo "========================================"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          Build both api-server and apictl"
	@echo "  make server         Build api-server binary"
	@echo "  make client         Build apictl client"
	@echo "  make all            Build everything (api-server + apictl)"
	@echo "  make run            Run api-server (requires api-server binary)"
	@echo "  make run-server     Same as 'make run'"
	@echo "  make demo           Run automated demo (requires api-server + apictl)"
	@echo "  make test           Run tests"
	@echo "  make clean          Remove build artifacts"
	@echo "  make install-deps   Download Go dependencies"
	@echo "  make fmt            Format code"
	@echo "  make lint           Run go vet"
	@echo "  make deadcode       Check for dead code"
	@echo "  make help           Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  make build          # Build both binaries"
	@echo "  make run            # Start api-server (in Terminal 1)"
	@echo "  # In Terminal 2:"
	@echo "  ./apictl api-resources"
	@echo "  ./apictl apply -f examples/invoice-crd.yaml"
	@echo ""

# Build targets
build: server client
	@echo "✓ Build complete: $(BINARY_SERVER) and $(BINARY_CLIENT)"

all: build
	@echo "✓ All targets built successfully"

server:
	@echo "Building api-server..."
	$(GO) build $(GOFLAGS) -o $(BINARY_SERVER) ./cmd/api-server
	@echo "✓ api-server built: $(BINARY_SERVER)"

client:
	@echo "Building apictl client..."
	$(GO) build $(GOFLAGS) -o $(BINARY_CLIENT) ./cmd/apictl
	@echo "✓ apictl built: $(BINARY_CLIENT)"

# Run targets
run: server
	@echo "Starting server on http://localhost:8080"
	@echo "Press Ctrl+C to stop"
	./$(BINARY_SERVER)

run-server: run

demo: server client
	@echo "Running automated demo..."
	@echo ""
	@chmod +x demo.sh
	./demo.sh

# Test targets
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Code quality targets
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✓ Code formatted"

lint:
	@echo "Running linter..."
	$(GO) vet ./...
	@echo "✓ Linting complete"

deadcode:
	@echo "Checking for dead code..."
	deadcode ./...
	@echo "✓ Dead code check complete"

# Dependency targets
install-deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✓ Dependencies ready"

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_SERVER) $(BINARY_CLIENT)
	rm -f coverage.out coverage.html
	$(GO) clean
	@echo "✓ Clean complete"

clean-all: clean
	@echo "Cleaning all generated files..."
	$(GO) mod tidy
	@echo "✓ All clean"

# Quick start targets
quickstart: install-deps build
	@echo ""
	@echo "✓ Quick start build complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  Terminal 1:"
	@echo "    make run"
	@echo ""
	@echo "  Terminal 2:"
	@echo "    ./$(BINARY_CLIENT) api-resources"
	@echo "    ./$(BINARY_CLIENT) apply -f examples/invoice-crd.yaml"
	@echo "    make demo"

# Development targets
dev: install-deps fmt lint build
	@echo "✓ Development build complete!"

# Info targets
info:
	@echo "Project Information:"
	@echo "  Binary (server): $(BINARY_SERVER)"
	@echo "  Binary (client): $(BINARY_CLIENT)"
	@echo "  Go version:"
	@$(GO) version
	@echo "  Module:"
	@grep "^module" go.mod

# Verify targets
verify: build test lint deadcode
	@echo "✓ Verification complete - project is healthy!"

# Setup plugins directory (for original plugin system)
setup-plugins:
	@echo "Creating plugins directory..."
	mkdir -p plugins
	@echo "✓ Plugins directory ready"

# Build everything and prepare for demo
prepare-demo: install-deps build setup-plugins
	@echo ""
	@echo "✓ Project ready for demo!"
	@echo ""
	@echo "Run demo with: make demo"
	@echo "Or manually:"
	@echo "  Terminal 1: make run"
	@echo "  Terminal 2: ./$(BINARY_CLIENT) api-resources"

# Discover available resources
api-resources: client
	@./$(BINARY_CLIENT) api-resources

# Discover API versions
api-versions: client
	@./$(BINARY_CLIENT) api-versions

# Create invoice CRD (requires running server)
create-crd: client
	@./$(BINARY_CLIENT) apply -f examples/invoice-crd.yaml

# List invoices (requires running server and created CRD)
get-invoices: client
	@./$(BINARY_CLIENT) get invoices

# Create sample invoice (requires running server and created CRD)
create-invoice: client
	@./$(BINARY_CLIENT) create -f examples/invoice-1.json

# Run full integration flow (requires running server)
integration-flow: create-crd create-invoice get-invoices
	@echo "✓ Integration flow complete"

# Docker targets (optional - for future use)
docker-build:
	@echo "Building Docker image..."
	docker build -t api-server:latest .
	@echo "✓ Docker image built"

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 api-server:latest

# Help for specific commands
help-quickstart:
	@echo "Quick Start Guide"
	@echo "================"
	@echo ""
	@echo "1. Build: make quickstart"
	@echo "2. Terminal 1: make run"
	@echo "3. Terminal 2: make demo"
	@echo ""
	@echo "Or step by step:"
	@echo "  make build          - Build binaries"
	@echo "  make run            - Start server"
	@echo "  make api-resources  - List available resources"
	@echo "  make create-crd     - Create Invoice CRD"
	@echo "  make api-resources  - List resources again (invoices should appear)"
	@echo "  make create-invoice - Create sample invoice"
	@echo "  make get-invoices   - List invoices"

# Phony targets that don't correspond to files
.PHONY: help build server client all clean run run-server demo test test-coverage \
        fmt lint deadcode install-deps clean-all quickstart dev info verify setup-plugins \
        prepare-demo api-resources api-versions create-crd get-invoices create-invoice \
        integration-flow docker-build docker-run help-quickstart
