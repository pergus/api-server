package api

import (
	"fmt"
	"sync"
)

// CRDDefinition represents a Custom Resource Definition.
// This is how Kubernetes allows arbitrary new resources to be registered at runtime.
type CRDDefinition struct {
	Group    string                 `json:"group"`
	Version  string                 `json:"version"`
	Kind     string                 `json:"kind"`
	Plural   string                 `json:"plural"`
	Schema   map[string]interface{} `json:"schema"`
	metadata CRDMetadata            `json:"-"`
}

// CRDMetadata holds metadata about a CRD.
type CRDMetadata struct {
	Name      string
	FullName  string // e.g., "invoices.example.io"
	CreatedAt string
}

// Validate checks if the CRD definition is valid.
func (c *CRDDefinition) Validate() error {
	if c.Group == "" {
		return fmt.Errorf("group is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if c.Plural == "" {
		return fmt.Errorf("plural is required")
	}
	return nil
}

// FullName returns the fully qualified name: plural.group
func (c *CRDDefinition) FullName() string {
	return fmt.Sprintf("%s.%s", c.Plural, c.Group)
}

// APIPath returns the API endpoint path for this CRD.
// e.g., /apis/example.io/v1/invoices
func (c *CRDDefinition) APIPath() string {
	return fmt.Sprintf("/apis/%s/%s/%s", c.Group, c.Version, c.Plural)
}

// CRDRegistry manages Custom Resource Definitions.
type CRDRegistry interface {
	// RegisterCRD registers a new CRD.
	RegisterCRD(crd *CRDDefinition) error

	// UnregisterCRD removes a CRD.
	UnregisterCRD(fullName string) error

	// GetCRD retrieves a CRD by its full name.
	GetCRD(fullName string) (*CRDDefinition, bool)

	// ListCRDs returns all registered CRDs.
	ListCRDs() []*CRDDefinition

	// FindByPlural finds a CRD by its plural name.
	FindByPlural(plural string) (*CRDDefinition, bool)
}

// SimpleCRDRegistry implements CRDRegistry.
type SimpleCRDRegistry struct {
	mu    sync.RWMutex
	crds  map[string]*CRDDefinition // fullName -> CRD
	byKey map[string]string         // plural -> fullName (for fast lookup)
}

// NewCRDRegistry creates a new CRD registry.
func NewCRDRegistry() CRDRegistry {
	return &SimpleCRDRegistry{
		crds:  make(map[string]*CRDDefinition),
		byKey: make(map[string]string),
	}
}

// RegisterCRD registers a new CRD.
func (r *SimpleCRDRegistry) RegisterCRD(crd *CRDDefinition) error {
	if err := crd.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	fullName := crd.FullName()
	if _, exists := r.crds[fullName]; exists {
		return fmt.Errorf("CRD %q already registered", fullName)
	}

	r.crds[fullName] = crd
	r.byKey[crd.Plural] = fullName
	return nil
}

// UnregisterCRD removes a CRD.
func (r *SimpleCRDRegistry) UnregisterCRD(fullName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	crd, exists := r.crds[fullName]
	if !exists {
		return fmt.Errorf("CRD %q not found", fullName)
	}

	delete(r.crds, fullName)
	delete(r.byKey, crd.Plural)
	return nil
}

// GetCRD retrieves a CRD by its full name.
func (r *SimpleCRDRegistry) GetCRD(fullName string) (*CRDDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	crd, exists := r.crds[fullName]
	return crd, exists
}

// ListCRDs returns all registered CRDs in sorted order.
func (r *SimpleCRDRegistry) ListCRDs() []*CRDDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	crds := make([]*CRDDefinition, 0, len(r.crds))
	for _, crd := range r.crds {
		crds = append(crds, crd)
	}
	return crds
}

// FindByPlural finds a CRD by its plural name.
func (r *SimpleCRDRegistry) FindByPlural(plural string) (*CRDDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fullName, exists := r.byKey[plural]
	if !exists {
		return nil, false
	}
	return r.crds[fullName], true
}
