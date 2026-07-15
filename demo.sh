#!/bin/bash

# Demo script for the dynamic API server with CRDs
# Run this after starting the server in another terminal

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

step_num=1

step() {
    echo -e "\n${BLUE}=== Step $step_num: $1 ===${NC}"
    ((step_num++))
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check if server is running
check_server() {
    if ! curl -s http://localhost:8080/api > /dev/null 2>&1; then
        echo -e "${RED}Error: Server is not running on http://localhost:8080${NC}"
        echo "Start the server with: ./api-server"
        exit 1
    fi
}

# Demo sequence
echo -e "${BLUE}Dynamic API Server with CRDs - Demo${NC}"
echo "======================================"

check_server
success "Server is running"

step "List Built-in Resources"
info "Running: ./apictl api-resources"
./apictl api-resources
success "Built-in resources listed"

step "Check API Versions"
info "Running: ./apictl api-versions"
./apictl api-versions
success "API groups listed"

step "Apply a CRD"
info "Running: ./apictl apply -f examples/invoice-crd.yaml"
./apictl apply -f examples/invoice-crd.yaml
success "CRD applied - invoices resource now available!"

step "List Resources Again"
info "Running: ./apictl api-resources"
info "Notice 'invoices' now appears in the list!"
./apictl api-resources
success "invoices resource is available"

step "Create an Invoice"
info "Running: ./apictl create -f examples/invoice-1.json"
./apictl create -f examples/invoice-1.json
success "Invoice created"

step "List All Invoices"
info "Running: ./apictl get invoices"
./apictl get invoices
success "Listed all invoices"

step "Get a Specific Invoice"
info "Running: ./apictl get invoices inv-001"
./apictl get invoices inv-001
success "Retrieved specific invoice"

step "Explain the Resource Schema"
info "Running: ./apictl explain invoices"
./apictl explain invoices
success "Resource schema displayed"

step "Delete the CRD"
info "Running: ./apictl delete crd invoices.example.io"
./apictl delete crd invoices.example.io
success "CRD deleted - resource no longer available"

step "Verify Resource Disappeared"
info "Running: ./apictl api-resources"
info "Notice 'invoices' is gone!"
./apictl api-resources
success "invoices resource has been removed"

# Final summary
echo -e "\n${GREEN}=== Demo Complete ===${NC}"
echo -e "${YELLOW}Key Takeaways:${NC}"
echo "✓ Resources registered and unregistered without server restart"
echo "✓ No recompilation required"
echo "✓ CRD changes appear immediately in API discovery"
echo "✓ apictl client discovered APIs dynamically"
echo "✓ Generic handlers worked for all resource types"
echo ""
echo -e "${BLUE}Architecture Highlights:${NC}"
echo "• Generic HTTP handlers (set up once, never change)"
echo "• Dynamic resource lookup at runtime"
echo "• Thread-safe Registry for concurrent access"
echo "• Type factory pattern (Scheme) for object creation"
echo "• DynamicObject for schema-less data storage"
echo ""
echo "See CRD_ARCHITECTURE.md for implementation details"
