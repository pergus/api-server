// pkg/plugins/loader.go
//
// This file implements the plugin loader for the API server. The Loader
// watches a specified directory for new plugin files (.so) and loads them
// dynamically at runtime. It tracks loaded plugins, handles registration of
// their resources, and provides information about loaded and failed plugins.
// The Loader demonstrates the server's runtime extensibility without requiring
// a restart when new plugins are added.

package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/pergus/api-server/pkg/api"
)

// Loader manages plugin loading and lifecycle.
//
// The Loader:
// - Watches a directory for new .so files
// - Loads plugins dynamically
// - Tracks loaded plugins
// - Handles plugin unloading
//
// This demonstrates runtime extensibility without server restart.
type Loader struct {
	pluginDir string
	registry  api.Registry
	scheme    api.Scheme
	mu        sync.RWMutex
	loaded    map[string]*LoadedPlugin
	failed    map[string]error
	stopOnce  sync.Once
	stopChan  chan struct{}
}

// LoadedPlugin tracks a loaded plugin.
type LoadedPlugin struct {
	Plugin Plugin
	Path   string
	Loaded time.Time
	Handle *plugin.Plugin
}

// NewLoader creates a plugin loader.
func NewLoader(pluginDir string, registry api.Registry, scheme api.Scheme) *Loader {
	return &Loader{
		pluginDir: pluginDir,
		registry:  registry,
		scheme:    scheme,
		loaded:    make(map[string]*LoadedPlugin),
		failed:    make(map[string]error),
		stopChan:  make(chan struct{}),
	}
}

// IsLoaded checks if a plugin is already loaded by path.
func (l *Loader) IsLoaded(path string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, plugin := range l.loaded {
		if plugin.Path == path {
			return true
		}
	}

	return false
}

// HasFailed checks if a plugin has failed to load previously.
func (l *Loader) HasFailed(path string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	_, exists := l.failed[path]
	return exists
}

// LoadPlugin loads a single plugin from a file.
// Returns error if the plugin is invalid.
func (l *Loader) LoadPlugin(path string) error {

	if l.IsLoaded(path) {
		return fmt.Errorf("plugin already loaded: %s", path)
	}

	log.Printf("Loading plugin from %s", path)

	// Open the plugin
	handle, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for a Plugin symbol
	pluginSym, err := handle.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin missing Plugin symbol: %w", err)
	}

	// Assert it's a Plugin
	pluginPtr, ok := pluginSym.(*Plugin)
	if !ok {
		return fmt.Errorf("Plugin symbol is not of type *Plugin")
	}

	p := *pluginPtr

	// Register the plugin
	if err := p.Register(l.registry, l.scheme); err != nil {
		return fmt.Errorf("plugin registration failed: %w", err)
	}

	// Track the loaded plugin
	l.mu.Lock()
	l.loaded[p.Name()] = &LoadedPlugin{
		Plugin: p,
		Path:   path,
		Loaded: time.Now(),
		Handle: handle,
	}
	l.mu.Unlock()

	log.Printf("Successfully loaded plugin: %s", p.Name())
	return nil
}

// UnloadPlugin unloads a plugin by name.
// This removes its resources from the registry.
func (l *Loader) UnloadPlugin(name string) error {
	l.mu.Lock()
	loaded, exists := l.loaded[name]
	if !exists {
		l.mu.Unlock()
		return fmt.Errorf("plugin %q not loaded", name)
	}
	delete(l.loaded, name)
	l.mu.Unlock()

	log.Printf("Unloading plugin: %s", name)

	// Call the plugin's Unregister
	return loaded.Plugin.Unregister(l.registry)
}

// Watch polls the plugin directory for new plugins.
// Runs in a goroutine and watches for changes.
// Call Stop() to stop watching.
func (l *Loader) Watch(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-l.stopChan:
				log.Println("Stopping plugin watcher")
				return
			case <-ticker.C:
				l.scanPlugins()
			}
		}
	}()
}

// scanPlugins looks for new .so files in the plugin directory.
func (l *Loader) scanPlugins() {
	// Check if directory exists
	_, err := os.Stat(l.pluginDir)
	if os.IsNotExist(err) {
		return
	}

	if err != nil {
		log.Printf("Error checking plugin directory: %v", err)
		return
	}

	// List files in the directory
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		log.Printf("Error reading plugin directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}

		path := filepath.Join(l.pluginDir, entry.Name())

		if l.IsLoaded(path) {
			continue
		}

		if l.HasFailed(path) {
			continue
		}

		if err := l.LoadPlugin(path); err != nil {
			l.mu.Lock()
			l.failed[path] = err
			l.mu.Unlock()
			log.Printf("Failed to load plugin... %s: %v", path, err)
			continue
		}
	}
}

// Scan scans the plugin directory for new plugins and loads them.
func (l *Loader) Scan() {
	l.scanPlugins()
}

// Stop stops the plugin watcher.
func (l *Loader) Stop() {
	// Signal the watcher to stop
	// Use sync.Once to ensure we only close the channel once
	l.stopOnce.Do(func() {
		close(l.stopChan)
	})
}

// ListLoaded returns a list of all loaded plugins.
func (l *Loader) ListLoaded() []api.PluginInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]api.PluginInfo, 0, len(l.loaded))

	for _, p := range l.loaded {
		result = append(result, api.PluginInfo{
			Name:   p.Plugin.Name(),
			Path:   p.Path,
			Loaded: p.Loaded.Format("2006-01-02T15:04:05"),
		})
	}

	return result
}

// ListFailed returns a list of all plugins that failed to load.
func (l *Loader) ListFailed() []api.FailedPluginInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]api.FailedPluginInfo, 0, len(l.failed))

	for path, err := range l.failed {
		result = append(result, api.FailedPluginInfo{
			Path:  path,
			Error: err.Error(),
		})
	}

	return result
}
