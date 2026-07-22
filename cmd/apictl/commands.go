// cmd/apictl/commands.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v2"
)

// cmdAPIResources lists all available resources
func cmdAPIResources(c *Client) {
	resources, err := c.GetAPIResources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME")
	for _, r := range resources {
		fmt.Fprintf(w, "%s\n", r)
	}
	w.Flush()
}

// cmdAPIVersions lists all API versions
func cmdAPIVersions(c *Client) {
	groups, err := c.GetAPIs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(groups) == 0 {
		fmt.Println("No API groups found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "GROUP")
	for _, g := range groups {
		fmt.Fprintf(w, "%s\n", g)
	}
	w.Flush()
}

// cmdPlugins lists all loaded plugins
func cmdPlugins(c *Client) {
	plugins, count, err := c.ListPlugins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded Plugins: %d\n", count)
	if len(plugins) == 0 {
		fmt.Println("No plugins loaded")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tLOADED")
	for _, p := range plugins {
		name, _ := p["name"].(string)
		path, _ := p["path"].(string)
		loaded, _ := p["loaded"].(string)
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, path, loaded)
	}
	w.Flush()
}

// cmdGet lists or retrieves a resource
func cmdGet(c *Client, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: resource name required\n")
		os.Exit(1)
	}

	resource := args[0]
	var id string
	if len(args) > 1 {
		id = args[1]
	}

	if id == "" {
		// List all resources of this type
		items, err := c.ListResources(resource)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(items) == 0 {
			fmt.Printf("No %s found\n", resource)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		printTable(w, items)
		w.Flush()

	} else {
		// Get specific resource
		item, err := c.GetResource(resource, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(data))
	}
}

// -----------------------------------------------------------------------------
// Mutating commands
//

// cmdCreate creates a resource from a file
func cmdCreate(c *Client, args []string) {
	if len(args) < 2 || args[0] != "-f" {
		fmt.Fprintf(os.Stderr, "Usage: apitcl create -f <file>\n")
		os.Exit(1)
	}

	filename := args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Determine resource type from the object
	var resource string
	if kind, ok := obj["kind"].(string); ok {
		// Infer plural from kind (simplified)
		resource = pluralize(kind)
	} else {
		fmt.Fprintf(os.Stderr, "Error: object must have 'kind' field\n")
		os.Exit(1)
	}

	id, err := c.CreateResource(resource, obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s created: %s\n", resource, id)
}

// cmdDelete deletes a resource
func cmdDelete(c *Client, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: apitcl delete <resource> <id>\n")
		os.Exit(1)
	}

	resource := args[0]
	id := args[1]

	// Special case: delete CRDs
	if resource == "crd" || resource == "crds" {
		if err := c.DeleteCRD(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("CRD deleted: %s\n", id)
		return
	}

	if err := c.DeleteResource(resource, id); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s deleted: %s\n", resource, id)
}

// cmdApply applies a CRD or creates/updates a resource
func cmdApply(c *Client, args []string) {
	if len(args) < 2 || args[0] != "-f" {
		fmt.Fprintf(os.Stderr, "Usage: apitcl apply -f <file>\n")
		os.Exit(1)
	}

	filename := args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Try to parse as YAML first
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		// Fall back to JSON
		if err := json.Unmarshal(data, &obj); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
			os.Exit(1)
		}
	}

	kind, ok := obj["kind"].(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: object must have 'kind' field\n")
		os.Exit(1)
	}

	if kind == "CustomResourceDefinition" || kind == "CRD" {
		// Extract the spec
		spec, ok := obj["spec"].(map[interface{}]interface{})
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: CRD must have 'spec' field\n")
			os.Exit(1)
		}

		crd := convertMap(spec)
		if err := c.CreateCRD(crd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if fullName, ok := crd["name"].(string); ok {
			fmt.Printf("CRD applied: %s\n", fullName)
		} else {
			plural := crd["plural"]
			group := crd["group"]
			fmt.Printf("CRD applied: %s.%s\n", plural, group)
		}
	} else {
		// Regular resource
		resource := pluralize(kind)
		if _, err := c.CreateResource(resource, obj); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s applied\n", resource)
	}
}

// -----------------------------------------------------------------------------
// Watch
//

// cmdWatch streams events for a resource
func cmdWatch(c *Client, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: apictl watch <resource>\n")
		os.Exit(1)
	}

	resource := args[0]

	fmt.Printf("Watching events for %s (Ctrl+C to stop)...\n\n", resource)

	result, err := c.Watch(resource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for {
		select {
		case event, ok := <-result.Events:
			if !ok {
				// Events channel closed
				return
			}

			// Print event type
			fmt.Printf("EVENT: %s\n", event.Type)

			// Parse and pretty-print the object
			var obj interface{}
			if err := json.Unmarshal(event.Data, &obj); err != nil {
				fmt.Printf("Error parsing event data: %v\n", err)
				continue
			}

			data, _ := json.MarshalIndent(obj, "", "  ")
			fmt.Printf("%s\n\n", string(data))

		case err, ok := <-result.Errors:
			if !ok {
				// Errors channel closed
				return
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Exlain & Helpers
//

// cmdExplain shows resource schema
func cmdExplain(c *Client, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: apitcl explain <resource>\n")
		os.Exit(1)
	}

	resource := args[0]

	// Check if it's a registered resource (built-in or CRD)
	allResources, err := c.GetAPIResources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting resources: %v\n", err)
		os.Exit(1)
	}

	// Check if resource exists
	found := false
	for _, r := range allResources {
		if r == resource {
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Resource not found: %s\n", resource)
		os.Exit(1)
	}

	// Try to find CRD/schema information
	crds, err := c.ListCRDs()
	if err == nil {
		for _, crd := range crds {
			if crdPlural, ok := crd["plural"].(string); ok && crdPlural == resource {
				// Get CRD metadata
				kind, _ := crd["kind"].(string)
				group, _ := crd["group"].(string)
				version, _ := crd["version"].(string)

				fmt.Printf("Kind: %s\n", kind)
				fmt.Printf("API: %s/%s\n", group, version)
				fmt.Println()

				// Show schema
				schema, hasSchema := crd["schema"]
				if hasSchema && schema != nil {
					if schemaMap, ok := schema.(map[string]interface{}); ok && len(schemaMap) > 0 {
						fmt.Println("Schema:")
						data, _ := json.MarshalIndent(schemaMap, "", "  ")
						fmt.Printf("%s\n", string(data))
					}
				}

				// Show sample object if available
				items, err := c.ListResources(resource)
				if err == nil && len(items) > 0 {
					fmt.Println()
					fmt.Println("Sample object:")
					data, _ := json.MarshalIndent(items[0], "", "  ")
					fmt.Printf("%s\n", string(data))
				}
				return
			}
		}
	}

	// Fallback: show fields from sample objects
	items, err := c.ListResources(resource)
	if err == nil && len(items) > 0 {
		fmt.Printf("Resource: %s\n", resource)
		fmt.Printf("Available fields:\n")
		for _, field := range getFieldNames(items[0]) {
			fmt.Printf("  - %s\n", field)
		}
		fmt.Printf("\nSample object:\n")
		data, _ := json.MarshalIndent(items[0], "", "  ")
		fmt.Printf("%s\n", string(data))
	} else {
		fmt.Printf("Resource: %s\n", resource)
		fmt.Printf("(No schema or sample objects available)\n")
	}
}

// Helper functions

// extractID extracts the ID from a resource object.
// It first looks for the "id" field, then "metadata.name", and returns
// "unknown" if neither is found.
func extractID(obj map[string]interface{}) string {
	if id, ok := obj["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	if meta, ok := obj["metadata"].(map[string]interface{}); ok {
		if name, ok := meta["name"]; ok {
			return fmt.Sprintf("%v", name)
		}
	}
	return "unknown"
}

// getFieldNames returns the field names of a resource object as a slice of
// strings.
func getFieldNames(obj map[string]interface{}) []string {
	fields := make([]string, 0, len(obj))
	for k := range obj {
		fields = append(fields, k)
	}
	return fields
}

// pluralize returns the plural form of a resource kind.
// This is a simplified version and may not cover all cases.
func pluralize(kind string) string {
	// Simplified pluralization
	switch kind {
	case "User":
		return "users"
	case "Product":
		return "products"
	case "Order":
		return "orders"
	case "Invoice":
		return "invoices"
	default:
		return kind + "s"
	}
}

// convertMap converts a map with interface{} keys to a map with string keys.
// This is useful for converting YAML parsed maps to JSON-compatible maps.
func convertMap(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := fmt.Sprintf("%v", k)
		if mv, ok := v.(map[interface{}]interface{}); ok {
			result[key] = convertMap(mv)
		} else if av, ok := v.([]interface{}); ok {
			result[key] = av
		} else {
			result[key] = v
		}
	}
	return result
}

// printTable prints a slice of maps as a table.
func printTable(w *tabwriter.Writer, items []map[string]interface{}) {
	// The "id" field is always printed as the first column.
	// Other fields are printed in alphabetical order.
	// If a field is missing in an item, it will be printed as empty.
	// The table is printed to the provided tabwriter.Writer.

	if len(items) == 0 {
		return
	}

	// Collect all keys except "id".
	keySet := make(map[string]struct{})

	for _, item := range items {
		for key := range item {
			if key != "id" {
				keySet[key] = struct{}{}
			}
		}
	}

	columns := make([]string, 0, len(keySet)+1)
	columns = append(columns, "id")

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	columns = append(columns, keys...)

	// Header.
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.ToUpper(col))
	}
	fmt.Fprintln(w)

	// Rows.
	for _, item := range items {
		for i, col := range columns {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}

			fmt.Fprint(w, formatValue(item[col]))
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

// formatValue formats a value for display in the table.
func formatValue(value interface{}) string {
	// If the value is a map or slice, it is marshaled to JSON.
	// If the value is nil, it returns an empty string.
	// Otherwise, it returns the string representation of the value.

	if value == nil {
		return ""
	}

	switch value.(type) {
	case map[string]interface{}, []interface{}:
		if data, err := json.Marshal(value); err == nil {
			return string(data)
		}
	}

	return fmt.Sprint(value)
}
