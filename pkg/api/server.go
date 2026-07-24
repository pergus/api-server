// pkg/api/server.go
//
// This file implements the core HTTP API server for the dynamic API framework.
// The Server struct manages the resource registry, type scheme, router, and
// event bus. It provides methods to start and stop the server, register
// resources and types at runtime, and handle Custom Resource Definitions
// (CRDs). The server supports dynamic registration of resources and types
// without requiring a restart, enabling runtime extensibility through plugins.

package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server is the HTTP API server.
type Server struct {
	registry    Registry
	scheme      Scheme
	router      *Router
	httpServer  *http.Server
	port        int
	crdRegistry CRDRegistry
	eventBus    EventBus
}

// Config holds server configuration.
type Config struct {
	Port int
}

// NewServer creates a new server.
func NewServer(cfg Config) *Server {
	registry := NewRegistry()
	scheme := NewScheme()
	crdRegistry := NewCRDRegistry()
	eventBus := NewEventBus()
	router := NewRouter(registry, scheme, crdRegistry, eventBus)

	return &Server{
		registry:    registry,
		scheme:      scheme,
		router:      router,
		crdRegistry: crdRegistry,
		eventBus:    eventBus,
		port:        cfg.Port,
	}
}

// Registry returns the resource registry.
// Called by plugins and main to register resources.
func (s *Server) Registry() Registry {
	return s.registry
}

// Scheme returns the type scheme.
// Called by plugins and main to register types.
func (s *Server) Scheme() Scheme {
	return s.scheme
}

// CRDRegistry returns the CRD registry.
// Called to manage Custom Resource Definitions.
func (s *Server) CRDRegistry() CRDRegistry {
	return s.crdRegistry
}

// Router returns the HTTP router.
func (s *Server) Router() *Router {
	return s.router
}

// EventBus returns the event bus.
// Called by controllers and watch endpoints to subscribe to events.
func (s *Server) EventBus() EventBus {
	return s.eventBus
}

// Start begins listening.
// The router is set up here.
func (s *Server) Start() error {
	log.Printf("Setting up routes (generic, never change)")
	s.router.Setup()

	log.Printf("Registered resources: %d", s.registry.Count())
	for _, name := range s.registry.Names() {
		log.Printf("  - %s", name)
	}

	// Wrap router with middleware
	handler := Chain(
		s.router,
		RecoveryMiddleware,
		CORSMiddleware,
		LoggingMiddleware,
		TimingMiddleware,
	)

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.port),
		Handler:        handler,
		ReadTimeout:    5 * time.Minute,
		WriteTimeout:   5 * time.Minute,
		IdleTimeout:    1 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Starting server on http://localhost:%d", s.port)
	log.Printf("Discovery: GET http://localhost:%d/api", s.port)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// RegisterResource registers a resource at runtime.
// This makes the resource immediately available without restarting the server.
// Also attaches the event bus to the resource's storage so events are published.
func (s *Server) RegisterResource(resource Resource) error {
	// Attach event bus to storage if storage is MemoryStorage
	if ms, ok := resource.Storage().(*MemoryStorage); ok {
		ms.SetEventBus(s.eventBus, resource.Name())
	}

	err := s.registry.Register(resource)
	if err == nil {
		log.Printf("Registered resource: %s", resource.Name())
	}
	return err
}

// RegisterType registers a type factory at runtime.
func (s *Server) RegisterType(name string, factory ObjectFactory) error {
	return s.scheme.Register(name, factory)
}

// UnregisterResource unregisters a resource at runtime.
// Not used in the current code but provided for completeness.
func (s *Server) UnregisterResource(name string) error {
	err := s.registry.Unregister(name)
	if err == nil {
		log.Printf("Unregistered resource: %s", name)
	}
	return err
}
