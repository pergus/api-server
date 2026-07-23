// pkg/plugins/interface.go
//
// Package plugins provides a plugin loading system for dynamic API extensibility.
//
// Plugins are compiled Go code that register resources with the API server at runtime.
//
// Key insights:
// - Plugins are loaded from .so files (compiled shared objects)
// - Each plugin is a separate Go package compiled to a plugin binary
// - When a plugin loads, it calls Register to add itself to the server
// - The server never needs to recompile or restart
//
// This demonstrates how to handle CustomResourceDefinitions (CRDs):
// - A CRD is like a plugin that adds a new resource type
// - Once registered, it works exactly like built-in resources
// - The API server code never changes

package plugins

import (
	"github.com/pergus/api-server/pkg/api"
)

// Plugin defines the interface that all plugins must implement.
//
// Each plugin is a separate Go package that exports a Plugin symbol.
// When the plugin loads, the plugin manager calls Register() to add the plugin's
// resources and types to the API server.
type Plugin interface {
	// Name returns the plugin name.
	Name() string

	// Register adds the plugin's resources to the server.
	// Called when the plugin is loaded.
	// The plugin receives the registry and scheme so it can register itself.
	Register(registry api.Registry, scheme api.Scheme) error

	// Unregister removes the plugin's resources from the server.
	// Called when the plugin is unloaded.
	Unregister(registry api.Registry) error
}
