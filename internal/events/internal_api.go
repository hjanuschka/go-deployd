package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// InternalClient provides internal HTTP-like access to resources
type InternalClient struct {
	router      http.Handler
	development bool
}

// NewInternalClient creates a new internal client
func NewInternalClient(router http.Handler, development bool) *InternalClient {
	return &InternalClient{
		router:      router,
		development: development,
	}
}

// Get performs an internal GET request
func (c *InternalClient) Get(path string, query ...map[string]interface{}) (interface{}, error) {
	// Build query string
	queryStr := ""
	if len(query) > 0 && query[0] != nil {
		params := []string{}
		for k, v := range query[0] {
			if jsonVal, err := json.Marshal(v); err == nil {
				params = append(params, fmt.Sprintf("%s=%s", k, string(jsonVal)))
			}
		}
		if len(params) > 0 {
			queryStr = "?" + strings.Join(params, "&")
		}
	}

	req := httptest.NewRequest("GET", path+queryStr, nil)
	return c.executeRequest(req)
}

// Post performs an internal POST request
func (c *InternalClient) Post(path string, data interface{}) (interface{}, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return c.executeRequest(req)
}

// Put performs an internal PUT request
func (c *InternalClient) Put(path string, data interface{}) (interface{}, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("PUT", path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return c.executeRequest(req)
}

// Delete performs an internal DELETE request
func (c *InternalClient) Delete(path string) (interface{}, error) {
	req := httptest.NewRequest("DELETE", path, nil)
	return c.executeRequest(req)
}

// executeRequest executes the internal request
func (c *InternalClient) executeRequest(req *http.Request) (interface{}, error) {
	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	c.router.ServeHTTP(w, req)

	// Get the response
	resp := w.Result()
	defer resp.Body.Close()

	// Parse the response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If JSON decode fails, return the status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return map[string]interface{}{"status": resp.StatusCode}, nil
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Check for errors in response
	if respMap, ok := result.(map[string]interface{}); ok {
		if errMsg, hasError := respMap["error"]; hasError {
			if msg, ok := respMap["message"].(string); ok {
				return nil, fmt.Errorf("%s", msg)
			}
			return nil, fmt.Errorf("request error: %v", errMsg)
		}
	}

	return result, nil
}

// CollectionClient provides collection-specific operations
type CollectionClient struct {
	client     *InternalClient
	collection string
}

// Collection returns a client for a specific collection
func (c *InternalClient) Collection(name string) *CollectionClient {
	return &CollectionClient{
		client:     c,
		collection: name,
	}
}

// Get retrieves documents or a specific document
func (cc *CollectionClient) Get(id string, query ...map[string]interface{}) (interface{}, error) {
	path := "/" + cc.collection
	if id != "" {
		path += "/" + id
	}
	return cc.client.Get(path, query...)
}

// Post creates a new document
func (cc *CollectionClient) Post(data interface{}) (interface{}, error) {
	return cc.client.Post("/"+cc.collection, data)
}

// Put updates a document
func (cc *CollectionClient) Put(id string, data interface{}) (interface{}, error) {
	return cc.client.Put("/"+cc.collection+"/"+id, data)
}

// Delete removes a document
func (cc *CollectionClient) Delete(id string) (interface{}, error) {
	return cc.client.Delete("/" + cc.collection + "/" + id)
}

// InternalAPI provides access to all collections (compatibility wrapper)
type InternalAPI struct {
	client *InternalClient
}

// NewInternalAPI creates a new internal API instance
func NewInternalAPI(router http.Handler, development bool) *InternalAPI {
	return &InternalAPI{
		client: NewInternalClient(router, development),
	}
}

// Collection returns a collection client
func (api *InternalAPI) Collection(name string) *CollectionClient {
	return api.client.Collection(name)
}