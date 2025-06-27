package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"go.mongodb.org/mongo-driver/bson"
)

// EventsHandler handles event script management
type EventsHandler struct {
	collections map[string]*resources.Collection
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(collections map[string]*resources.Collection) *EventsHandler {
	return &EventsHandler{
		collections: collections,
	}
}

// GetEvents returns all event scripts for a collection
func (eh *EventsHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	collectionName := strings.TrimPrefix(r.URL.Path, "/api/_admin/collections/")
	collectionName = strings.Split(collectionName, "/")[0]

	collection, exists := eh.collections[collectionName]
	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Get script information
	scripts := make(map[string]interface{})
	types := make(map[string]string)

	// Check for existing script files
	eventTypes := []string{"get", "validate", "post", "put", "delete", "aftercommit", "beforerequest"}

	for _, eventType := range eventTypes {
		// Check for .js file
		jsPath := filepath.Join(collection.GetConfigPath(), eventType+".js")
		if content, err := os.ReadFile(jsPath); err == nil {
			scripts[eventType] = string(content)
			types[eventType] = "js"
		}

		// Check for .go file
		goPath := filepath.Join(collection.GetConfigPath(), eventType+".go")
		if content, err := os.ReadFile(goPath); err == nil {
			scripts[eventType] = string(content)
			types[eventType] = "go"
		}
	}

	// Get hot-reload info
	hotReloadInfo := collection.GetHotReloadInfo()

	response := map[string]interface{}{
		"scripts":    scripts,
		"types":      types,
		"hotReload":  hotReloadInfo,
		"collection": collectionName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateEvent updates a specific event script
func (eh *EventsHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/_admin/collections/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	collectionName := parts[0]
	eventName := parts[2]

	collection, exists := eh.collections[collectionName]
	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var request struct {
		Script string `json:"script"`
		Type   string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate event type
	validEvents := map[string]events.EventType{
		"get":           events.EventGet,
		"validate":      events.EventValidate,
		"post":          events.EventPost,
		"put":           events.EventPut,
		"delete":        events.EventDelete,
		"aftercommit":   events.EventAfterCommit,
		"beforerequest": events.EventBeforeRequest,
	}

	eventType, validEvent := validEvents[eventName]
	if !validEvent {
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	// Handle different script types
	switch request.Type {
	case "go":
		// Hot-reload Go script using interpreter
		if err := collection.LoadHotReloadScript(eventType, request.Script); err != nil {
			http.Error(w, fmt.Sprintf("Failed to load Go script: %v", err), http.StatusBadRequest)
			return
		}

		// Optionally save to file
		filePath := filepath.Join(collection.GetConfigPath(), eventName+".go")
		if err := eh.saveScriptToFile(filePath, request.Script, "go"); err != nil {
			// Log warning but don't fail the request
			logging.GetLogger().WithComponent("events").Warn("Failed to save Go script to file", logging.Fields{
				"collection": collectionName,
				"event":      eventName,
				"file_path":  filePath,
				"error":      err.Error(),
			})
		}

	case "js":
		// Save JavaScript file and reload
		filePath := filepath.Join(collection.GetConfigPath(), eventName+".js")
		if err := eh.saveScriptToFile(filePath, request.Script, "js"); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save script: %v", err), http.StatusInternalServerError)
			return
		}

		// Reload JavaScript scripts
		if err := collection.ReloadScripts(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reload scripts: %v", err), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Invalid script type", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("%s event updated successfully", eventName),
		"type":       request.Type,
		"hotReload":  request.Type == "go",
		"collection": collectionName,
		"event":      eventName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestEvent tests an event script with mock data
func (eh *EventsHandler) TestEvent(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/_admin/collections/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	collectionName := parts[0]
	eventName := parts[2]

	collection, exists := eh.collections[collectionName]
	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var request struct {
		Data       bson.M `json:"data"`
		User       bson.M `json:"user,omitempty"`
		Query      bson.M `json:"query,omitempty"`
		ScriptType string `json:"scriptType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create mock context
	mockCtx := eh.createMockContext(request.Data, request.User, request.Query)

	// Map event name to event type
	eventTypeMap := map[string]events.EventType{
		"get":           events.EventGet,
		"validate":      events.EventValidate,
		"post":          events.EventPost,
		"put":           events.EventPut,
		"delete":        events.EventDelete,
		"aftercommit":   events.EventAfterCommit,
		"beforerequest": events.EventBeforeRequest,
	}

	eventType, exists := eventTypeMap[eventName]
	if !exists {
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	// Test the event
	startTime := now()
	var testErr error
	var resultData bson.M = request.Data

	if request.ScriptType == "go" {
		// Test hot-reloaded Go script
		testErr = collection.TestHotReloadScript(eventType, mockCtx, resultData)
	} else {
		// Test JavaScript script
		testErr = collection.TestScript(eventType, mockCtx, resultData)
	}

	duration := int(now().Sub(startTime).Milliseconds())

	// Prepare response
	response := map[string]interface{}{
		"success":    testErr == nil,
		"duration":   duration,
		"data":       resultData,
		"collection": collectionName,
		"event":      eventName,
		"scriptType": request.ScriptType,
	}

	if testErr != nil {
		if scriptErr, ok := testErr.(*events.ScriptError); ok {
			response["error"] = scriptErr.Message
			response["statusCode"] = scriptErr.StatusCode
		} else if validationErr, ok := testErr.(*events.ValidationError); ok {
			response["errors"] = validationErr.Errors
		} else {
			response["error"] = testErr.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// saveScriptToFile saves a script to the filesystem
func (eh *EventsHandler) saveScriptToFile(filePath, content, scriptType string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Add package declaration for Go files
	if scriptType == "go" && !strings.Contains(content, "package ") {
		content = "package main\n\n" + content
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// createMockContext creates a mock context for testing
func (eh *EventsHandler) createMockContext(data, user, query bson.M) *appcontext.Context {
	// This is a simplified mock context for testing
	// In a real implementation, you'd create a proper mock HTTP request/response
	return &appcontext.Context{
		Query:  query,
		Body:   data,
		Method: "POST", // Default method
		// Session: mockSession with user data
	}
}

// Helper function to get current time (for easier testing)
var now = func() time.Time {
	return time.Now()
}
