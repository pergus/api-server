#!/bin/bash

# Build script for plugins
# This builds all plugins in the plugins/ directory as .so files

set -e

echo "Building plugins..."

# Build invoices plugin
echo "Building invoices plugin..."
go build -buildmode=plugin -o bin/plugins/invoices.so ./plugins/invoices

echo "All plugins built successfully!"
