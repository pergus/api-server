package api

import (
	"context"
	"testing"
	"time"
)

// serverTestRes implements Resource for testing
type serverTestRes struct {
	storage Storage
	name    string
}

func (r *serverTestRes) Name() string {
	return r.name
}

func (r *serverTestRes) NewObject() any {
	return map[string]interface{}{}
}

func (r *serverTestRes) Storage() Storage {
	return r.storage
}

// TestServerNew tests creating a new server.
func TestServerNew(t *testing.T) {
	server := NewServer(Config{Port: 9999})

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.port != 9999 {
		t.Errorf("Server port is %d, expected 9999", server.port)
	}
}

// TestServerRegistry tests accessing the registry.
func TestServerRegistry(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	registry := server.Registry()
	if registry == nil {
		t.Fatal("Registry is nil")
	}
}

// TestServerScheme tests accessing the scheme.
func TestServerScheme(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	scheme := server.Scheme()
	if scheme == nil {
		t.Fatal("Scheme is nil")
	}
}

// TestServerCRDRegistry tests accessing the CRD registry.
func TestServerCRDRegistry(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	crdRegistry := server.CRDRegistry()
	if crdRegistry == nil {
		t.Fatal("CRD Registry is nil")
	}
}

// TestServerEventBus tests accessing the event bus.
func TestServerEventBus(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	eventBus := server.EventBus()
	if eventBus == nil {
		t.Fatal("EventBus is nil")
	}
	eventBus.Close()
}

// TestServerRegisterResource tests registering a resource.
func TestServerRegisterResource(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	impl := &serverTestRes{
		storage: NewMemoryStorage(),
		name:    "test",
	}

	server.RegisterResource(impl)
	server.EventBus().Close()
}

// TestServerRegisterType tests registering a type.
func TestServerRegisterType(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	factory := func() any {
		return map[string]interface{}{}
	}

	err := server.RegisterType("TestType", factory)
	if err != nil {
		t.Fatalf("RegisterType failed: %v", err)
	}
	server.EventBus().Close()
}

// TestServerUnregisterResource tests unregistering a resource.
func TestServerUnregisterResource(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	impl := &serverTestRes{
		storage: NewMemoryStorage(),
		name:    "testres",
	}

	server.RegisterResource(impl)

	err := server.UnregisterResource("testres")
	if err != nil {
		t.Logf("UnregisterResource returned: %v", err)
	}

	server.EventBus().Close()
}

// TestServerStop tests graceful shutdown.
func TestServerStop(t *testing.T) {
	server := NewServer(Config{Port: 9999})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	if err != nil {
		t.Logf("Stop returned: %v", err)
	}

	server.EventBus().Close()
}

// TestServerStopNilServer tests stopping a server that was never started.
func TestServerStopNilServer(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop on unstarted server failed: %v", err)
	}

	server.EventBus().Close()
}

// TestServerMultipleResourceRegistration tests registering multiple resources.
func TestServerMultipleResourceRegistration(t *testing.T) {
	server := NewServer(Config{Port: 8080})

	resourceNames := []string{"users", "products", "orders"}
	for _, name := range resourceNames {
		n := name
		r := &serverTestRes{
			storage: NewMemoryStorage(),
			name:    n,
		}
		if err := server.RegisterResource(r); err != nil {
			t.Fatalf("RegisterResource %s failed: %v", name, err)
		}
	}

	server.EventBus().Close()
}
