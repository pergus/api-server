package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Router is the HTTP request dispatcher.
//
// THIS IS THE KEY TO DYNAMIC EXTENSIBILITY.
//
// Unlike typical REST servers that create routes for each resource at startup:
//   GET /users, POST /users, GET /users/{id}, etc.
//
// This router creates only GENERIC routes that determine the resource at runtime:
//   GET /api/{resource}, POST /api/{resource}, GET /api/{resource}/{id}, etc.
//
// Every request:
// 1. Extracts the resource name from the URL
// 2. Looks it up in the registry (which may have been updated while running)
// 3. Dispatches to ONE generic handler
//
// The handlers never know about specific resources. They work through:
// - The Resource interface
// - The Storage interface
// - The Scheme (for object creation)
//
// This means new resources are immediately available after registration—
// no router rebuild, no server restart, no HTTP listener restart.
// This is how Kubernetes achieves extensibility.
type Router struct {
	registry    Registry
	scheme      Scheme
	crdRegistry CRDRegistry
	mux         *http.ServeMux
}

// NewRouter creates a new router.
func NewRouter(registry Registry, scheme Scheme, crdRegistry CRDRegistry) *Router {
	return &Router{
		registry:    registry,
		scheme:      scheme,
		crdRegistry: crdRegistry,
		mux:         http.NewServeMux(),
	}
}

// Setup registers the generic routes.
// These routes are created ONCE and never change, even when new resources are added.
func (r *Router) Setup() {
	// Kubernetes-style discovery endpoints
	r.mux.HandleFunc("/api", r.discovery)
	r.mux.HandleFunc("/apis", r.discoverAPIs)
	r.mux.HandleFunc("/apis/", r.discoverAPIPath)

	// Catch-all handler for all resource and CRD operations
	r.mux.HandleFunc("/", r.route)
}

// ServeHTTP makes Router satisfy http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// discovery handles GET /api
// Returns all registered resources.
// This endpoint updates dynamically as resources are registered.
func (r *Router) discovery(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := DiscoveryResponse{
		Resources: r.registry.Names(),
		Time:      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// route is the main request dispatcher.
// This single handler routes ALL resource and CRD requests.
func (r *Router) route(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	// Handle /crds endpoints
	if strings.HasPrefix(path, "/crds") {
		r.routeCRD(w, req)
		return
	}

	// Handle /api/{resource} endpoints
	if !strings.HasPrefix(path, "/api/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Remove /api/ prefix and split
	parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	resourceName := parts[0]

	// Look up resource in registry
	// This happens on EVERY request, which is fine because Lookup uses a read lock.
	resource, ok := r.registry.Lookup(resourceName)
	if !ok {
		http.Error(w, fmt.Sprintf("resource %q not found", resourceName), http.StatusNotFound)
		return
	}

	// Determine which handler to call based on HTTP method and URL structure
	if len(parts) == 1 {
		// /api/{resource} - list or create
		r.routeListOrCreate(w, req, resource)
	} else if len(parts) == 2 && parts[1] != "" {
		// /api/{resource}/{id} - get, update, or delete
		id := parts[1]
		r.routeItemOp(w, req, resource, id)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// routeListOrCreate handles list (GET) or create (POST) operations.
func (r *Router) routeListOrCreate(w http.ResponseWriter, req *http.Request, resource Resource) {
	switch req.Method {
	case http.MethodGet:
		r.list(w, req, resource)
	case http.MethodPost:
		r.create(w, req, resource)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// routeItemOp handles get, update, or delete operations on a specific item.
func (r *Router) routeItemOp(w http.ResponseWriter, req *http.Request, resource Resource, id string) {
	switch req.Method {
	case http.MethodGet:
		r.get(w, req, resource, id)
	case http.MethodPut:
		r.update(w, req, resource, id)
	case http.MethodDelete:
		r.delete(w, req, resource, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// list handles GET /api/{resource}
// Generic handler that works for ALL resources.
func (r *Router) list(w http.ResponseWriter, req *http.Request, resource Resource) {
	objects, err := resource.Storage().List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := ListResponse{
		Items: objects,
		Count: len(objects),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// get handles GET /api/{resource}/{id}
// Generic handler that works for ALL resources.
func (r *Router) get(w http.ResponseWriter, req *http.Request, resource Resource, id string) {
	object, err := resource.Storage().Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(object)
}

// create handles POST /api/{resource}
// Generic handler that works for ALL resources.
// The key insight: we ask the Scheme to create an empty object.
// We don't know what type it is, but we can unmarshal JSON into it.
func (r *Router) create(w http.ResponseWriter, req *http.Request, resource Resource) {
	// Read and limit request body
	body := io.LimitReader(req.Body, 1024*1024)
	defer req.Body.Close()

	// Ask the Scheme to create an empty object
	// This is the magic that allows generic handlers:
	// The handler doesn't know it's creating a User, Product, or Order.
	// It just asks for an empty object by the resource name.
	obj, err := r.scheme.New(resource.Name())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal incoming JSON into the empty object
	if err := json.NewDecoder(body).Decode(obj); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Store the object
	if err := resource.Storage().Create(obj); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract ID for response
	id := extractIDFromObject(obj)

	response := CreatedResponse{
		Message: fmt.Sprintf("%s created", resource.Name()),
		ID:      id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// update handles PUT /api/{resource}/{id}
// Generic handler that works for ALL resources.
func (r *Router) update(w http.ResponseWriter, req *http.Request, resource Resource, id string) {
	body := io.LimitReader(req.Body, 1024*1024)
	defer req.Body.Close()

	obj, err := r.scheme.New(resource.Name())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewDecoder(body).Decode(obj); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := resource.Storage().Update(id, obj); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := UpdatedResponse{
		Message: fmt.Sprintf("%s updated", resource.Name()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// delete handles DELETE /api/{resource}/{id}
// Generic handler that works for ALL resources.
func (r *Router) delete(w http.ResponseWriter, req *http.Request, resource Resource, id string) {
	if err := resource.Storage().Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := DeletedResponse{
		Message: fmt.Sprintf("%s deleted", resource.Name()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// routeCRD handles all /crds endpoints
// Routes based on HTTP method to appropriate CRD handler
func (r *Router) routeCRD(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	if path == "/crds" || path == "/crds/" {
		// /crds - list or create
		switch req.Method {
		case http.MethodGet:
			r.listCRDs(w, req)
		case http.MethodPost:
			r.createCRD(w, req)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else if strings.HasPrefix(path, "/crds/") {
		// /crds/{name} - delete
		switch req.Method {
		case http.MethodDelete:
			r.deleteCRD(w, req)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// extractIDFromObject pulls the ID from an object by marshalling to JSON.
func extractIDFromObject(obj any) string {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Printf("error marshalling object: %v", err)
		return ""
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("error unmarshalling object: %v", err)
		return ""
	}

	if id, ok := m["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}

// discoverAPIs handles GET /apis
// Returns all API groups
func (r *Router) discoverAPIs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Collect unique groups from built-in and CRD resources
	groups := make(map[string]bool)

	// Add built-in resources to a default group
	if len(r.registry.List()) > 0 {
		groups["api.example.io"] = true
	}

	// Add CRD groups
	for _, crd := range r.crdRegistry.ListCRDs() {
		groups[crd.Group] = true
	}

	groupList := make([]string, 0, len(groups))
	for g := range groups {
		groupList = append(groupList, g)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"groups":    groupList,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// discoverAPIPath handles GET /apis/{group} and /apis/{group}/{version}
func (r *Router) discoverAPIPath(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/apis/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	group := parts[0]
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}

	// Filter CRDs by group and version
	resources := make([]map[string]interface{}, 0)
	for _, crd := range r.crdRegistry.ListCRDs() {
		if crd.Group == group && (version == "" || crd.Version == version) {
			resources = append(resources, map[string]interface{}{
				"name":    crd.Plural,
				"kind":    crd.Kind,
				"version": crd.Version,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"group":     group,
		"version":   version,
		"resources": resources,
	})
}

// createCRD handles POST /crds
func (r *Router) createCRD(w http.ResponseWriter, req *http.Request) {
	body := io.LimitReader(req.Body, 1024*1024)
	defer req.Body.Close()

	var crd CRDDefinition
	if err := json.NewDecoder(body).Decode(&crd); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Register the CRD
	if err := r.crdRegistry.RegisterCRD(&crd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create a dynamic resource for this CRD
	resource := NewDynamicResource(&crd)

	// Register the resource in the main registry
	if err := r.registry.Register(resource); err != nil {
		// Unregister the CRD if resource registration fails
		r.crdRegistry.UnregisterCRD(crd.FullName())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Register the object factory in the scheme
	plural := crd.Plural
	if err := r.scheme.Register(plural, func() any {
		return &DynamicObject{
			APIVersion: fmt.Sprintf("%s/%s", crd.Group, crd.Version),
			Kind:       crd.Kind,
			Metadata:   make(map[string]interface{}),
			Spec:       make(map[string]interface{}),
		}
	}); err != nil {
		// Unregister on failure
		r.registry.Unregister(plural)
		r.crdRegistry.UnregisterCRD(crd.FullName())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("CRD registered: %s", crd.FullName())

	response := map[string]interface{}{
		"message": fmt.Sprintf("CRD %s registered", crd.FullName()),
		"name":    crd.FullName(),
		"path":    crd.APIPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listCRDs handles GET /crds
func (r *Router) listCRDs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	crds := r.crdRegistry.ListCRDs()
	crdList := make([]map[string]interface{}, 0, len(crds))

	for _, crd := range crds {
		crdList = append(crdList, map[string]interface{}{
			"name":    crd.FullName(),
			"group":   crd.Group,
			"version": crd.Version,
			"kind":    crd.Kind,
			"plural":  crd.Plural,
			"schema":  crd.Schema,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": crdList,
		"count": len(crdList),
	})
}

// deleteCRD handles DELETE /crds/{name}
func (r *Router) deleteCRD(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/crds/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	crdName := path

	// Get the CRD to find the plural name
	crd, exists := r.crdRegistry.GetCRD(crdName)
	if !exists {
		http.Error(w, fmt.Sprintf("CRD %q not found", crdName), http.StatusNotFound)
		return
	}

	plural := crd.Plural

	// Unregister from all three places to ensure complete cleanup

	// 1. Unregister the resource from Resource Registry
	if err := r.registry.Unregister(plural); err != nil {
		// Log but don't fail - resource might not exist in registry
		log.Printf("Warning: could not unregister resource %q: %v", plural, err)
	}

	// 2. Unregister the type factory from Scheme
	if err := r.scheme.Unregister(plural); err != nil {
		// Log but don't fail - type might not exist in scheme
		log.Printf("Warning: could not unregister type %q: %v", plural, err)
	}

	// 3. Unregister the CRD from CRD Registry
	if err := r.crdRegistry.UnregisterCRD(crdName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("CRD unregistered: %s", crdName)

	response := map[string]interface{}{
		"message": fmt.Sprintf("CRD %s deleted", crdName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
