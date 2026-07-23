// plugins/invoices/main.go
//
// Package main implements the invoices plugin.
//
// This is a complete example of a plugin that:
// - Implements the Plugin interface
// - Defines its own resource type (Invoice)
// - Registers itself with the API server when loaded
// - Becomes immediately available through the API
//
// To use this plugin:
// 1. Build it: go build -buildmode=plugin -o invoices.so ./plugins/invoices/main.go
// 2. Copy the .so file to the plugins/ directory while the server is running
// 3. The server will automatically load it and make /api/invoices available
//
// This demonstrates that new API resources can be introduced without:
// - Recompiling the server
// - Restarting the server
// - Rebuilding the HTTP router
// - Any framework changes
package main

import (
	"log"

	"github.com/pergus/api-server/pkg/api"
	"github.com/pergus/api-server/pkg/plugins"
)

// Invoice is the resource type defined by this plugin.
type Invoice struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
}

// InvoiceResource implements api.Resource.
type InvoiceResource struct {
	storage api.Storage
}

// NewInvoiceResource creates a new invoice resource.
func NewInvoiceResource() *InvoiceResource {
	return &InvoiceResource{
		storage: api.NewMemoryStorage(),
	}
}

// Name returns "invoices".
func (r *InvoiceResource) Name() string {
	return "invoices"
}

// NewObject returns an empty Invoice.
func (r *InvoiceResource) NewObject() any {
	return &Invoice{}
}

// Storage returns the storage implementation.
func (r *InvoiceResource) Storage() api.Storage {
	return r.storage
}

// InvoicePlugin implements the plugins.Plugin interface.
type InvoicePlugin struct {
	resource *InvoiceResource
}

// Name returns the plugin name.
func (p *InvoicePlugin) Name() string {
	return "invoices"
}

// Register adds the invoice resource to the server.
func (p *InvoicePlugin) Register(registry api.Registry, scheme api.Scheme) error {
	log.Println("[InvoicePlugin] Registering invoice resource and type")

	// Register the resource
	if err := registry.Register(p.resource); err != nil {
		return err
	}

	// Register the type factory
	if err := scheme.Register("invoices", func() any { return &Invoice{} }); err != nil {
		return err
	}

	log.Println("[InvoicePlugin] Successfully registered invoices")
	return nil
}

// Unregister removes the invoice resource from the server.
func (p *InvoicePlugin) Unregister(registry api.Registry) error {
	log.Println("[InvoicePlugin] Unregistering invoice resource")
	return registry.Unregister("invoices")
}

// Plugin is the symbol that the plugin loader looks for.
// It must be exported and of type plugins.Plugin.
var Plugin plugins.Plugin = &InvoicePlugin{
	resource: NewInvoiceResource(),
}
