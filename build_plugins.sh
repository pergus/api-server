#!/bin/bash

# Build script for plugins
# This builds all plugins in the plugins/ directory as .so files

set -e

echo "Building plugins..."

mkdir -p bin/plugins

for plugin_dir in plugins/*; do
    if [ -d "$plugin_dir" ]; then
        plugin_name=$(basename "$plugin_dir")

        echo "Building ${plugin_name} plugin..."
        go build -buildmode=plugin -o "bin/plugins/${plugin_name}.so" "./${plugin_dir}"
    fi
done

echo "All plugins built successfully!"
