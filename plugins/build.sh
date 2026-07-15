#!/bin/bash

# Build script for plugins
# This builds all plugins in the plugins/ directory as .so files

set -e

echo "Building plugins..."

# Build invoices plugin
echo "Building invoices plugin..."
go build -buildmode=plugin -o invoices/invoices.so ./invoices/main.go

echo "All plugins built successfully!"
echo ""
echo "To use plugins:"
echo "1. Copy .so files to the plugins/ directory while the server is running"
echo "2. The server will automatically load them"
echo ""
echo "Example:"
echo "  cp invoices/invoices.so ../plugins/"
