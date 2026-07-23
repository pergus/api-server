// pkg/api/plugins.go
//
// This file defines the PluginProvider interface and related types for managing
// plugins in the API server. The PluginProvider interface allows the API server
// to provide information about loaded and failed plugins, enabling clients to
// query the server for plugin status. The PluginInfo struct contains public
// information about a loaded plugin, including its name, path, and the time it
// was loaded.

package api

// PluginInfo contains public plugin information.
type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Loaded string `json:"loaded"`
}

// FailedPluginInfo contains information about a plugin that failed to load.
type FailedPluginInfo struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// PluginProvider provides information about loaded plugins.
type PluginProvider interface {
	ListLoaded() []PluginInfo
	ListFailed() []FailedPluginInfo
}
