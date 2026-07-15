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
	router := NewRouter(registry, scheme, crdRegistry)

	return &Server{
		registry:    registry,
		scheme:      scheme,
		router:      router,
		crdRegistry: crdRegistry,
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
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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
func (s *Server) RegisterResource(resource Resource) error {
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
// This is called when plugins are unloaded.
func (s *Server) UnregisterResource(name string) error {
	err := s.registry.Unregister(name)
	if err == nil {
		log.Printf("Unregistered resource: %s", name)
	}
	return err
}
