// cmd/api-server/main.go
//
// Command server starts the dynamic API server.
//
// This server demonstrates true runtime extensibility:
// - Resources can be registered while the server is running
// - New resources are immediately available through the API
// - The HTTP router never changes
// - The server never restarts
//
// The server:
// 1. Creates the API framework (Registry, Scheme, Router)
// 2. Registers built-in resources (users, products, orders)
// 3. Starts a plugin watcher to load new resources dynamically
// 4. Begins listening for HTTP requests
//
// At this point, plugins can be added to the plugins/ directory,
// and they will be loaded automatically without restarting the server.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pergus/api-server/pkg/api"
	"github.com/pergus/api-server/pkg/controllers"
	"github.com/pergus/api-server/pkg/plugins"
	"github.com/pergus/api-server/pkg/resources"
)

func main() {
	// Create the API server
	server := api.NewServer(api.Config{
		Port: 8080,
	})

	// Register built-in resources
	log.Println("Registering built-in resources...")

	// Register Users
	userResource := resources.NewUserResource()
	if err := server.RegisterResource(userResource); err != nil {
		log.Fatalf("Failed to register users: %v", err)
	}
	if err := server.RegisterType("users", func() any { return &resources.User{} }); err != nil {
		log.Fatalf("Failed to register users type: %v", err)
	}
	// Register User schema as CRD
	userCRD := &api.CRDDefinition{
		Group:   "api.example.io",
		Version: "v1",
		Kind:    "User",
		Plural:  "users",
		Schema: map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Unique user identifier",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "User's full name",
				},
				"email": map[string]interface{}{
					"type":        "string",
					"description": "User's email address",
				},
				"is_active": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether user is active",
				},
			},
		},
	}
	if err := server.CRDRegistry().RegisterCRD(userCRD); err != nil {
		log.Fatalf("Failed to register user schema: %v", err)
	}

	// Register Products
	productResource := resources.NewProductResource()
	if err := server.RegisterResource(productResource); err != nil {
		log.Fatalf("Failed to register products: %v", err)
	}
	if err := server.RegisterType("products", func() any { return &resources.Product{} }); err != nil {
		log.Fatalf("Failed to register products type: %v", err)
	}
	// Register Product schema as CRD
	productCRD := &api.CRDDefinition{
		Group:   "api.example.io",
		Version: "v1",
		Kind:    "Product",
		Plural:  "products",
		Schema: map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Product identifier",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Product name",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Product description",
				},
				"price": map[string]interface{}{
					"type":        "number",
					"description": "Product price in USD",
				},
				"stock": map[string]interface{}{
					"type":        "integer",
					"description": "Items in stock",
				},
			},
		},
	}
	if err := server.CRDRegistry().RegisterCRD(productCRD); err != nil {
		log.Fatalf("Failed to register product schema: %v", err)
	}

	// Register Orders
	orderResource := resources.NewOrderResource()
	if err := server.RegisterResource(orderResource); err != nil {
		log.Fatalf("Failed to register orders: %v", err)
	}
	if err := server.RegisterType("orders", func() any { return &resources.Order{} }); err != nil {
		log.Fatalf("Failed to register orders type: %v", err)
	}
	// Register Order schema as CRD
	orderCRD := &api.CRDDefinition{
		Group:   "api.example.io",
		Version: "v1",
		Kind:    "Order",
		Plural:  "orders",
		Schema: map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Order identifier",
				},
				"customer_id": map[string]interface{}{
					"type":        "string",
					"description": "Customer identifier",
				},
				"total": map[string]interface{}{
					"type":        "number",
					"description": "Order total in USD",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Order status (draft, processing, shipped, delivered)",
				},
				"created_at": map[string]interface{}{
					"type":        "string",
					"description": "ISO 8601 timestamp",
				},
			},
		},
	}
	if err := server.CRDRegistry().RegisterCRD(orderCRD); err != nil {
		log.Fatalf("Failed to register order schema: %v", err)
	}

	// Create plugin loader
	// This watches the plugins/ directory for new .so files
	log.Println("Starting plugin system...")
	loader := plugins.NewLoader("./plugins", server.Registry(), server.Scheme())

	// Load any existing plugins
	log.Println("Scanning for existing plugins...")
	entries, err := os.ReadDir("./plugins")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && entry.Name()[len(entry.Name())-3:] == ".so" {
				pluginPath := "./plugins/" + entry.Name()
				if err := loader.LoadPlugin(pluginPath); err != nil {
					log.Printf("Warning: failed to load %s: %v", pluginPath, err)
				}
			}
		}
	}

	// Start watching for new plugins
	// Poll every 2 seconds for new plugins
	loader.Watch(2 * time.Second)

	// Initialize the controller manager and register controllers
	log.Println("Initializing controller manager...")
	manager := controllers.New(server.EventBus())

	// Register the order controller
	// It will watch for order events and perform reconciliation
	if err := manager.Register(controllers.NewOrderController(server.EventBus(), server.Registry())); err != nil {
		log.Printf("Warning: failed to register OrderController: %v", err)
	}

	// Start the controller manager in a goroutine
	// Controllers run concurrently and respond to events
	go func() {
		ctx := context.Background()
		if err := manager.Run(ctx); err != nil {
			log.Printf("Controller manager error: %v", err)
		}
	}()

	// Start the server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader.Stop()
	if err := server.Stop(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
