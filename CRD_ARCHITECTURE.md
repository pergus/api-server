# CRD and Dynamic Object Architecture

This document explains the extensions for Custom Resource Definitions (CRDs) and dynamic objects.

## New Components

### 1. CRDDefinition

**File**: `pkg/api/crd.go`

Represents a Custom Resource Definition:

```go
type CRDDefinition struct {
    Group    string                 // API group (e.g., "example.io")
    Version  string                 // API version (e.g., "v1")
    Kind     string                 // Resource kind (e.g., "Invoice")
    Plural   string                 // Plural name (e.g., "invoices")
    Schema   map[string]interface{} // JSON schema for validation
}
```

Methods:
- `Validate()` - Checks required fields
- `FullName()` - Returns "plural.group" (e.g., "invoices.example.io")
- `APIPath()` - Returns endpoint path (e.g., "/apis/example.io/v1/invoices")

### 2. CRDRegistry

Manages all registered CRDs:

```go
type CRDRegistry interface {
    RegisterCRD(crd *CRDDefinition) error
    UnregisterCRD(fullName string) error
    GetCRD(fullName string) (*CRDDefinition, bool)
    ListCRDs() []*CRDDefinition
    FindByPlural(plural string) (*CRDDefinition, bool)
}
```

Key feature: Thread-safe with `RWMutex` for concurrent access.

### 3. DynamicObject

**File**: `pkg/api/dynamic.go`

A generic object that can hold any JSON data without a compiled Go struct:

```go
type DynamicObject struct {
    APIVersion string                 `json:"apiVersion"`
    Kind       string                 `json:"kind"`
    Metadata   map[string]interface{} `json:"metadata"`
    Spec       map[string]interface{} `json:"spec"`
    Data       map[string]interface{} `json:"data,omitempty"`
}
```

Features:
- **Custom JSON unmarshalling**: Maps "id" field to "metadata.name"
- **Custom JSON marshalling**: Flattens back to simple JSON for backwards compatibility
- **Metadata extraction**: `GetID()` extracts ID from metadata.name

Example unmarshalling:

```json
// Input JSON
{
  "id": "inv-001",
  "customer": "Acme Corp",
  "amount": 5000.00
}

// Internal representation
{
  "metadata": { "name": "inv-001" },
  "spec": {
    "customer": "Acme Corp",
    "amount": 5000.00
  }
}
```

### 4. SimpleDynamicResource

Wraps a CRD to provide the Resource interface:

```go
type SimpleDynamicResource struct {
    crd     *CRDDefinition
    storage Storage
}
```

This allows CRD-based resources to integrate seamlessly with the existing framework.

## CRD Lifecycle

### 1. Creating a CRD

**Endpoint**: `POST /crds`

```bash
curl -X POST http://localhost:8080/crds \
  -H "Content-Type: application/json" \
  -d '{
    "group": "example.io",
    "version": "v1",
    "kind": "Invoice",
    "plural": "invoices",
    "schema": {
      "id": "string",
      "customer": "string",
      "amount": "number"
    }
  }'
```

**Server actions**:
1. Validate CRD definition
2. Register in CRDRegistry
3. Create SimpleDynamicResource
4. Register in main Resource Registry
5. Register object factory in Scheme
6. Respond 201 Created

**Result**: Resource immediately available at:
- `GET /api/invoices` - List
- `POST /api/invoices` - Create
- `GET /api/invoices/{id}` - Get
- `PUT /api/invoices/{id}` - Update
- `DELETE /api/invoices/{id}` - Delete

### 2. Listing CRDs

**Endpoint**: `GET /crds`

Returns all registered CRDs with metadata.

### 3. Deleting a CRD

**Endpoint**: `DELETE /crds/{fullName}`

Example:
```bash
curl -X DELETE http://localhost:8080/crds/invoices.example.io
```

**Server actions**:
1. Find CRD by full name
2. Unregister from Resource Registry
3. Unregister from CRDRegistry
4. All requests to `/api/invoices` return 404

## API Discovery

The server supports Kubernetes-style discovery:

### Discovery Endpoints

```
GET /api                    List built-in and core resources
GET /apis                   List all API groups
GET /apis/{group}           List resources in a group
GET /apis/{group}/{version} List resources in a group/version
```

### Response Format

`GET /api`:
```json
{
  "resources": ["users", "products", "orders", "invoices"],
  "timestamp": "2025-07-15T12:34:56Z"
}
```

`GET /apis`:
```json
{
  "groups": ["api.example.io", "example.io"],
  "timestamp": "2025-07-15T12:34:56Z"
}
```

`GET /apis/example.io`:
```json
{
  "group": "example.io",
  "version": "",
  "resources": [
    {"name": "invoices", "kind": "Invoice", "version": "v1"}
  ]
}
```

## apictl Client

**Directory**: `cmd/apictl`

A minimal CLI client that discovers APIs from the server.

### Architecture

**Client**: `client.go`
- HTTP communication with server
- API discovery methods
- CRUD operations

**Commands**: `commands.go`
- `api-resources` - List all available resources
- `api-versions` - List all API groups
- `get` - Retrieve resources
- `create` - Create from file
- `delete` - Delete resources
- `apply` - Apply CRDs or create/update resources
- `explain` - Show resource schema

### Key Feature: No Hardcoded Resources

```go
// apictl does NOT hardcode resource names
resources, err := client.GetAPIResources() // Discovered at runtime

// For each discovered resource:
// - Can list: apictl get <resource>
// - Can create: apictl create -f <file>
// - Can delete: apictl delete <resource> <id>
```

If you add a new CRD:
1. No client rebuild needed
2. No recompilation needed
3. Client automatically discovers it
4. All commands work immediately

### YAML Support

The client can parse YAML CRD definitions:

```yaml
apiVersion: api.example.io/v1
kind: CustomResourceDefinition
metadata:
  name: invoices.example.io
spec:
  group: example.io
  version: v1
  kind: Invoice
  plural: invoices
  schema:
    properties:
      id:
        type: string
      customer:
        type: string
      amount:
        type: number
```

Used with:
```bash
apictl apply -f crd.yaml
```

## Request Flow Examples

### Example 1: Create an Invoice via apictl

```bash
apictl create -f invoice.json
```

Flow:
1. Client reads `invoice.json`
2. Parses JSON to extract `kind: "Invoice"`
3. Infers plural: "invoices"
4. POSTs to `http://localhost:8080/api/invoices`
5. Server:
   - Looks up "invoices" in Registry → finds SimpleDynamicResource
   - Calls Scheme.New("invoices") → returns DynamicObject
   - Unmarshals JSON into DynamicObject
   - Stores in SimpleDynamicResource.Storage()
   - Returns ID
6. Client displays: "invoices created: inv-001"

### Example 2: Apply a CRD

```bash
apictl apply -f invoice-crd.yaml
```

Flow:
1. Client reads YAML
2. Parses to detect `kind: "CustomResourceDefinition"`
3. Extracts spec with group, version, kind, plural
4. POSTs to `http://localhost:8080/crds`
5. Server:
   - Validates CRD definition
   - Creates SimpleDynamicResource
   - Registers in all registries
   - Returns endpoint path
6. Client displays: "CRD applied: invoices.example.io"
7. Next request to `GET /api/invoices` succeeds

### Example 3: Discover and List Resources

```bash
apictl api-resources
```

Flow:
1. Client calls `GET http://localhost:8080/api`
2. Server returns list of all registered resources
3. Client displays in table format
4. If new CRD was registered while running, it appears in the list

## Thread Safety

All registries protect shared state with `RWMutex`:

```go
// CRDRegistry example
type SimpleCRDRegistry struct {
    mu    sync.RWMutex
    crds  map[string]*CRDDefinition
    byKey map[string]string
}

// Registration (rare)
func (r *SimpleCRDRegistry) RegisterCRD(crd *CRDDefinition) error {
    r.mu.Lock()              // Exclusive access
    defer r.mu.Unlock()
    // ... register
}

// Lookup (millions of times)
func (r *SimpleCRDRegistry) FindByPlural(plural string) (*CRDDefinition, bool) {
    r.mu.RLock()             // Concurrent reads allowed
    defer r.mu.RUnlock()
    // ... lookup
}
```

Benefits:
- Millions of concurrent resource lookups
- Safe registration while serving requests
- No busy-wait or polling
- Minimal contention

## Integration with Existing Framework

### Server Integration

```go
// In NewServer()
crdRegistry := NewCRDRegistry()
router := NewRouter(registry, scheme, crdRegistry)
```

### Router Integration

```go
// In router.Setup()
r.mux.HandleFunc("POST /crds", r.createCRD)
r.mux.HandleFunc("GET /crds", r.listCRDs)
r.mux.HandleFunc("DELETE /crds/", r.deleteCRD)
r.mux.HandleFunc("/apis", r.discoverAPIs)
r.mux.HandleFunc("/apis/", r.discoverAPIPath)
```

### Handler Implementation

Generic handlers already work with DynamicObject:

```go
// create() handler - works for ANY resource
obj, _ := r.scheme.New(resource.Name())
json.NewDecoder(body).Decode(obj)
resource.Storage().Create(obj)
```

No changes needed to handler code.

## Performance Characteristics

| Operation | Time | Notes |
|-----------|------|-------|
| CRD registration | ~1ms | Write lock, infrequent |
| Resource lookup | ~100ns | Read lock, called per request |
| CRD discovery | ~1μs | Read lock, small list |
| Object creation | ~1μs | factory() + unmarshalling |
| Storage operation | ~10μs | Hashtable + RWMutex |

Total per-request overhead: negligible vs. network/serialization overhead.

## Limitations and Future Work

1. **Schema Validation**
   - Schemas are stored but not enforced
   - Could add validation middleware

2. **Versioning**
   - CRDs support multiple versions
   - Could route by version

3. **Sub-resources**
   - Could add `/status` and `/scale` support

4. **Webhooks**
   - Could add pre/post operation hooks

5. **Watch/Stream**
   - Could add WebSocket for streaming events

## Summary

The CRD and DynamicObject extensions enable:

✓ Custom Resource Definitions without recompilation
✓ Generic object representation
✓ Runtime resource registration/unregistration
✓ API discovery for clients
✓ YAML-based declarative configuration
✓ Discovery-based CLI client
✓ Complete integration with existing framework

**Key principle**: Objects are just maps of data. Handlers don't care what type they are.
