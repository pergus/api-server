// cmd/apictl/client.go
//
// This file defines the Client struct and its methods for interacting with the
// dynamic API server. The Client provides functions to list resources, retrieve
// specific resources, create, update, and delete resources, as well as manage
// Custom Resource Definitions (CRDs) and plugins. It communicates with the API
// server over HTTP and handles JSON encoding/decoding of requests and responses.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PluginInfo contains public plugin information.
type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Loaded string `json:"loaded"`
}

// FailedPluginInfo contains information about a plugin that failed to load.
type FailedPluginInfo struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// PluginList bundles loaded and failed plugins.
// It is returned by the ListPlugins method of the Client.
type PluginList struct {
	Plugins []PluginInfo       `json:"plugins"`
	Count   int                `json:"count"`
	Failed  []FailedPluginInfo `json:"failed"`
}

// Client communicates with the dynamic API server.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a new client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

// GetAPIResources retrieves all available resources.
func (c *Client) GetAPIResources() ([]string, error) {
	resp, err := c.get("/api")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	resources, ok := result["resources"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	res := make([]string, 0, len(resources))
	for _, r := range resources {
		if s, ok := r.(string); ok {
			res = append(res, s)
		}
	}
	return res, nil
}

// GetAPIs retrieves all API groups.
func (c *Client) GetAPIs() ([]string, error) {
	resp, err := c.get("/apis")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	groups, ok := result["groups"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	res := make([]string, 0, len(groups))
	for _, g := range groups {
		if s, ok := g.(string); ok {
			res = append(res, s)
		}
	}
	return res, nil
}

// ListResources lists all objects of a resource type.
func (c *Client) ListResources(resource string) ([]map[string]interface{}, error) {
	resp, err := c.get(fmt.Sprintf("/api/%s", resource))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	items, ok := result["items"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	res := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			res = append(res, m)
		}
	}
	return res, nil
}

// GetResource retrieves a specific resource.
func (c *Client) GetResource(resource, id string) (map[string]interface{}, error) {
	resp, err := c.get(fmt.Sprintf("/api/%s/%s", resource, id))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateResource creates a new resource.
func (c *Client) CreateResource(resource string, obj map[string]interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	resp, err := c.post(fmt.Sprintf("/api/%s", resource), data)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}
	return "", fmt.Errorf("no id in response")
}

// UpdateResource updates an existing resource.
func (c *Client) UpdateResource(resource, id string, obj map[string]interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = c.put(fmt.Sprintf("/api/%s/%s", resource, id), data)
	return err
}

// DeleteResource deletes a resource.
func (c *Client) DeleteResource(resource, id string) error {
	_, err := c.delete(fmt.Sprintf("/api/%s/%s", resource, id))
	return err
}

// -----------------------------------------------------------------------------
// CRDs
//

// CreateCRD creates a new CRD.
func (c *Client) CreateCRD(crd map[string]interface{}) error {
	data, err := json.Marshal(crd)
	if err != nil {
		return err
	}

	_, err = c.post("/crds", data)
	return err
}

// ListCRDs lists all CRDs.
func (c *Client) ListCRDs() ([]map[string]interface{}, error) {
	resp, err := c.get("/crds")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	items, ok := result["items"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	res := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			res = append(res, m)
		}
	}
	return res, nil
}

// DeleteCRD deletes a CRD.
func (c *Client) DeleteCRD(crdName string) error {
	_, err := c.delete(fmt.Sprintf("/crds/%s", crdName))
	return err
}

// -----------------------------------------------------------------------------
// Plugins
//

// ListPlugins lists loaded and failed plugins.
func (c *Client) ListPlugins() (*PluginList, error) {
	resp, err := c.get("/plugins")
	if err != nil {
		return nil, err
	}

	var result PluginList
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// -----------------------------------------------------------------------------
// Watcher
//

// WatchEvent represents a single event from the watch stream.
type WatchEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WatchResult bundles event and error channels for watch streaming.
// Callers should range over Events and check Errors for problems.
type WatchResult struct {
	Events <-chan WatchEvent
	Errors <-chan error
}

// Watch streams events for a resource.
// Returns event and error channels.
// The caller should range over Events and check Errors for issues:
//
//	result := client.Watch("orders")
//	for {
//		select {
//		case event := <-result.Events:
//			// Handle event
//		case err := <-result.Errors:
//			// Handle error (connection closed, parse error, etc.)
//			return
//		}
//	}
//
// Errors include:
// - Connection failures
// - Server errors
// - Parse errors
// - Line size overruns (no 64 KiB limit)
func (c *Client) Watch(resource string) (*WatchResult, error) {
	url := c.baseURL + fmt.Sprintf("/api/%s?watch=true", resource)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Create channels for events and errors
	events := make(chan WatchEvent, 10)
	errors := make(chan error, 1)

	// Start goroutine to read events using bufio.Reader instead of Scanner
	// This avoids the 64 KiB token limit and gives us better control
	go func() {
		defer resp.Body.Close()
		defer close(events)
		defer close(errors)

		reader := bufio.NewReader(resp.Body)
		var currentType string

		for {
			// Read line with no size limit (unlike Scanner's 64 KiB default)
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Normal connection close
					return
				}
				// Network or parsing error
				select {
				case errors <- fmt.Errorf("read error: %w", err):
				default:
				}
				return
			}

			// Remove trailing newline
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			// SSE format: event: TYPE\ndata: JSON\n\n
			if strings.HasPrefix(line, "event: ") {
				currentType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				event := WatchEvent{
					Type: currentType,
					Data: json.RawMessage(data),
				}
				select {
				case events <- event:
				case <-time.After(100 * time.Millisecond):
					// Event channel full, drop and log
					select {
					case errors <- fmt.Errorf("event channel full, dropping event"):
					default:
					}
				}
			} else if line != "" {
				// Non-comment lines (not starting with ':') are skipped
				_ = strings.HasPrefix(line, ":")
			}
			// Ignore empty lines and comments (lines starting with :)
		}
	}()

	return &WatchResult{
		Events: events,
		Errors: errors,
	}, nil
}

// -----------------------------------------------------------------------------
// Helper methods
//

func (c *Client) get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

func (c *Client) post(path string, body []byte) ([]byte, error) {
	return c.request("POST", path, body)
}

func (c *Client) put(path string, body []byte) ([]byte, error) {
	return c.request("PUT", path, body)
}

func (c *Client) delete(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}

// request performs an HTTP request with the given method, path, and body.
func (c *Client) request(method, path string, body []byte) ([]byte, error) {
	url := c.baseURL + path
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			if msg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
