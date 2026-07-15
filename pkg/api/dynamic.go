package api

import (
	"encoding/json"
	"fmt"
)

// DynamicObject represents a generic Kubernetes-like object.
// It can hold any JSON data without requiring a compiled Go struct.
// This is how Kubernetes stores Custom Resources.
type DynamicObject struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// GetID extracts the ID from metadata.name.
func (d *DynamicObject) GetID() (string, error) {
	if d.Metadata == nil {
		return "", fmt.Errorf("metadata is nil")
	}

	name, exists := d.Metadata["name"]
	if !exists {
		return "", fmt.Errorf("metadata.name not found")
	}

	id, ok := name.(string)
	if !ok {
		return "", fmt.Errorf("metadata.name is not a string")
	}

	if id == "" {
		return "", fmt.Errorf("metadata.name is empty")
	}

	return id, nil
}

// UnmarshalJSON implements custom JSON unmarshalling.
// This allows the object to accept flat JSON structures and normalize them.
func (d *DynamicObject) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// If the incoming JSON has an "id" field, move it to metadata.name
	if id, exists := raw["id"]; exists {
		if d.Metadata == nil {
			d.Metadata = make(map[string]interface{})
		}
		d.Metadata["name"] = id
		delete(raw, "id")
	}

	// Preserve apiVersion and kind if present
	if apiVersion, exists := raw["apiVersion"]; exists {
		d.APIVersion = apiVersion.(string)
		delete(raw, "apiVersion")
	}

	if kind, exists := raw["kind"]; exists {
		d.Kind = kind.(string)
		delete(raw, "kind")
	}

	// Everything else goes into spec (or data for backwards compatibility)
	if d.Spec == nil {
		d.Spec = make(map[string]interface{})
	}
	for k, v := range raw {
		if k != "metadata" {
			d.Spec[k] = v
		}
	}

	// Ensure metadata exists
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{})
	}

	return nil
}

// MarshalJSON implements custom JSON marshalling.
// Returns a flat structure for backwards compatibility.
func (d *DynamicObject) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	// Add apiVersion and kind if present
	if d.APIVersion != "" {
		result["apiVersion"] = d.APIVersion
	}
	if d.Kind != "" {
		result["kind"] = d.Kind
	}

	// Add id from metadata.name for backwards compatibility
	if d.Metadata != nil {
		if name, exists := d.Metadata["name"]; exists {
			result["id"] = name
		}
	}

	// Add all spec fields at the top level
	if d.Spec != nil {
		for k, v := range d.Spec {
			result[k] = v
		}
	}

	// Add data fields if present
	if d.Data != nil {
		for k, v := range d.Data {
			result[k] = v
		}
	}

	return json.Marshal(result)
}

// SimpleDynamicResource is a Resource implementation for CRD-based resources.
type SimpleDynamicResource struct {
	crd     *CRDDefinition
	storage Storage
}

// NewDynamicResource creates a new dynamic resource for a CRD.
func NewDynamicResource(crd *CRDDefinition) *SimpleDynamicResource {
	return &SimpleDynamicResource{
		crd:     crd,
		storage: NewMemoryStorage(),
	}
}

// Name returns the plural name of the resource.
func (r *SimpleDynamicResource) Name() string {
	return r.crd.Plural
}

// NewObject returns a new DynamicObject.
func (r *SimpleDynamicResource) NewObject() any {
	return &DynamicObject{
		APIVersion: fmt.Sprintf("%s/%s", r.crd.Group, r.crd.Version),
		Kind:       r.crd.Kind,
		Metadata:   make(map[string]interface{}),
		Spec:       make(map[string]interface{}),
	}
}

// Storage returns the storage backend.
func (r *SimpleDynamicResource) Storage() Storage {
	return r.storage
}

// CRD returns the CRD definition.
func (r *SimpleDynamicResource) CRD() *CRDDefinition {
	return r.crd
}
