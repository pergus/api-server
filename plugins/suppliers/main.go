// plugins/supplier/main.go
//
// Package main implements the suppliers plugin.
//
// This plugin demonstrates that multiple plugins can be loaded at runtime
// and register independent API resources.
//
// It adds:
//
//	GET    /api/suppliers
//	POST   /api/suppliers
//	GET    /api/suppliers/{id}
//	PUT    /api/suppliers/{id}
//	DELETE /api/suppliers/{id}
//
// without modifying or restarting the API server.
package main

import (
	"log"

	"github.com/pergus/api-server/pkg/api"
	"github.com/pergus/api-server/pkg/plugins"
)

// Supplier is the resource type defined by this plugin.
type Supplier struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Country string `json:"country"`
	Active  bool   `json:"active"`
}

// SupplierResource implements api.Resource.
type SupplierResource struct {
	storage api.Storage
}

// NewSupplierResource creates a new supplier resource.
func NewSupplierResource() *SupplierResource {
	return &SupplierResource{
		storage: api.NewMemoryStorage(),
	}
}

// Name returns the resource name used in the API.
func (r *SupplierResource) Name() string {
	return "suppliers"
}

// NewObject returns an empty Supplier.
func (r *SupplierResource) NewObject() any {
	return &Supplier{}
}

// Storage returns the storage implementation.
func (r *SupplierResource) Storage() api.Storage {
	return r.storage
}

// SupplierPlugin implements plugins.Plugin.
type SupplierPlugin struct {
	resource *SupplierResource
}

// Name returns the plugin name.
func (p *SupplierPlugin) Name() string {
	return "suppliers"
}

// Register adds the supplier resource and type.
func (p *SupplierPlugin) Register(registry api.Registry, scheme api.Scheme) error {
	log.Println("[SupplierPlugin] Registering supplier resource and type")

	if err := registry.Register(p.resource); err != nil {
		return err
	}

	if err := scheme.Register("suppliers", func() any {
		return &Supplier{}
	}); err != nil {
		return err
	}

	log.Println("[SupplierPlugin] Successfully registered suppliers")

	return nil
}

// Unregister removes the supplier resource.
func (p *SupplierPlugin) Unregister(registry api.Registry) error {
	log.Println("[SupplierPlugin] Unregistering supplier resource")

	return registry.Unregister("suppliers")
}

// Plugin is discovered by the plugin loader.
var Plugin plugins.Plugin = &SupplierPlugin{
	resource: NewSupplierResource(),
}
