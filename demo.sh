#!/bin/bash

# Demo script for the dynamic API server with CRDs
# Run this after starting the server in another terminal

set -e

###############################################################################
# Configuration
###############################################################################

APICTL="./bin/apictl"
API_SERVER="./bin/api-server"
SERVER_URL="http://localhost:8080"

###############################################################################
# Colors
###############################################################################

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

step_num=1

step()
{
    echo -e "\n${BLUE}=== Step $step_num: $1 ===${NC}"
    (( step_num++ ))
}

success()
{
    echo -e "${GREEN}✓ $1${NC}"
}

info()
{
    echo -e "${YELLOW}→ $1${NC}"
}

###############################################################################
# Helpers
###############################################################################

check_server()
{
    if ! curl -s "${SERVER_URL}/api" > /dev/null 2>&1; then
        echo -e "${RED}Error: Server is not running on ${SERVER_URL}${NC}"
        echo "Start the server with: ${API_SERVER}"
        exit 1
    fi
}

###############################################################################
# Demo
###############################################################################

echo -e "${BLUE}Dynamic API Server with CRDs - Demo${NC}"
echo "======================================"

check_server
success "Server is running"

step "List Built-in Resources"
info "Running: ${APICTL} api-resources"
"${APICTL}" api-resources
success "Built-in resources listed"

step "Check API Versions"
info "Running: ${APICTL} api-versions"
"${APICTL}" api-versions
success "API groups listed"

step "Apply a CRD"
info "Running: ${APICTL} apply -f examples/invoice-crd.yaml"
"${APICTL}" apply -f examples/invoice-crd.yaml
success "CRD applied - invoices resource now available!"

step "List Resources Again"
info "Running: ${APICTL} api-resources"
info "Notice 'invoices' now appears in the list!"
"${APICTL}" api-resources
success "invoices resource is available"

step "Create an Invoice"
info "Running: ${APICTL} create -f examples/invoice-1.json"
"${APICTL}" create -f examples/invoice-1.json
success "Invoice created"

step "List All Invoices"
info "Running: ${APICTL} get invoices"
"${APICTL}" get invoices
success "Listed all invoices"

step "Get a Specific Invoice"
info "Running: ${APICTL} get invoices inv-001"
"${APICTL}" get invoices inv-001
success "Retrieved specific invoice"

step "Explain the Resource Schema"
info "Running: ${APICTL} explain invoices"
"${APICTL}" explain invoices
success "Resource schema displayed"

step "Delete the CRD"
info "Running: ${APICTL} delete crd invoices.example.io"
"${APICTL}" delete crd invoices.example.io
success "CRD deleted - resource no longer available"

step "Verify Resource Disappeared"
info "Running: ${APICTL} api-resources"
info "Notice 'invoices' is gone!"
"${APICTL}" api-resources
success "invoices resource has been removed"

###############################################################################
# Summary
###############################################################################

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