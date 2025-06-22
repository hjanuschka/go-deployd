package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/hjanuschka/go-deployd/internal/database"
	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/events"
)

type CollectionConfig struct {
	Properties map[string]Property `json:"properties"`
}

type Collection struct {
	*BaseResource
	config           *CollectionConfig
	store            *database.Store
	db               *database.Database
	scriptManager    *events.UniversalScriptManager
	hotReloadManager *events.HotReloadGoManager
	configPath       string
}

func NewCollection(name string, config *CollectionConfig, db *database.Database) *Collection {
	return &Collection{
		BaseResource:     NewBaseResource(name),
		config:           config,
		store:            db.CreateStore(name),
		db:               db,
		scriptManager:    events.NewUniversalScriptManager(),
		hotReloadManager: nil, // Will be initialized when needed
	}
}

func LoadCollectionFromConfig(name, configPath string, db *database.Database) (*Collection, error) {
	configFile := filepath.Join(configPath, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config CollectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	collection := NewCollection(name, &config, db)
	collection.configPath = configPath
	
	// Load event scripts
	if err := collection.scriptManager.LoadScripts(configPath); err != nil {
		// Scripts are optional, so don't fail if they don't exist
		fmt.Printf("Warning: Failed to load scripts for %s: %v\n", name, err)
	}
	
	return collection, nil
}

func (c *Collection) Handle(ctx *appcontext.Context) error {
	switch ctx.Method {
	case "GET":
		return c.handleGet(ctx)
	case "POST":
		return c.handlePost(ctx)
	case "PUT":
		return c.handlePut(ctx)
	case "DELETE":
		return c.handleDelete(ctx)
	default:
		return ctx.WriteError(405, "Method not allowed")
	}
}

func (c *Collection) handleGet(ctx *appcontext.Context) error {
	id := ctx.GetID()
	
	// Special endpoints
	if id == "count" {
		return c.handleCount(ctx)
	}
	
	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "GET"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	if id != "" {
		// Get single document
		doc, err := c.store.FindOne(ctx.Context(), bson.M{"id": id})
		if err != nil {
			return ctx.WriteError(500, err.Error())
		}
		if doc == nil {
			return ctx.WriteError(404, "Document not found")
		}
		
		// Run Get event for single document
		if err := c.runGetEvent(ctx, doc); err != nil {
			if scriptErr, ok := err.(*events.ScriptError); ok {
				return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
			}
			return ctx.WriteError(500, err.Error())
		}
		
		return ctx.WriteJSON(doc)
	}
	
	// Get multiple documents
	// First extract query options like $sort, $limit, $skip
	opts, cleanQuery := c.extractQueryOptions(ctx.Query)
	
	// Then sanitize the remaining query
	sanitizedQuery := c.sanitizeQuery(cleanQuery)
	
	docs, err := c.store.Find(ctx.Context(), sanitizedQuery, opts)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	// Run Get event for each document
	filteredDocs := make([]bson.M, 0)
	for _, doc := range docs {
		if err := c.runGetEvent(ctx, doc); err != nil {
			// Skip documents that fail the Get event
			if _, ok := err.(*events.ScriptError); ok {
				continue
			}
		}
		filteredDocs = append(filteredDocs, doc)
	}
	
	return ctx.WriteJSON(filteredDocs)
}

func (c *Collection) handlePost(ctx *appcontext.Context) error {
	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "POST"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}

	// Validate and sanitize body
	if err := c.validate(ctx.Body, true); err != nil {
		return ctx.WriteError(400, err.Error())
	}
	
	sanitized := c.sanitize(ctx.Body)
	
	// Set default values
	c.setDefaults(sanitized)
	
	// Run Validate event
	if err := c.runValidateEvent(ctx, sanitized); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		if validationErr, ok := err.(*events.ValidationError); ok {
			return ctx.WriteError(400, validationErr.Error())
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Run Post event
	if err := c.runPostEvent(ctx, sanitized); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Insert document
	result, err := c.store.Insert(ctx.Context(), sanitized)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	// Run AfterCommit event
	if resultDoc, ok := result.(bson.M); ok {
		go c.runAfterCommitEvent(ctx, resultDoc, "POST")
	}
	
	return ctx.WriteJSON(result)
}

func (c *Collection) handlePut(ctx *appcontext.Context) error {
	id := ctx.GetID()
	if id == "" {
		return ctx.WriteError(400, "ID is required for PUT requests")
	}
	
	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "PUT"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Check if this is a MongoDB command operation
	if c.isMongoCommand(ctx.Body) {
		return c.handleMongoCommand(ctx, id)
	}
	
	// Get the existing document for the 'previous' object
	previous, err := c.store.FindOne(ctx.Context(), bson.M{"id": id})
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	if previous == nil {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Validate and sanitize body
	if err := c.validate(ctx.Body, false); err != nil {
		return ctx.WriteError(400, err.Error())
	}
	
	sanitized := c.sanitize(ctx.Body)
	
	// Merge with existing document
	merged := make(bson.M)
	for k, v := range previous {
		merged[k] = v
	}
	for k, v := range sanitized {
		merged[k] = v
	}
	
	// Run Validate event
	if err := c.runValidateEvent(ctx, merged); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		if validationErr, ok := err.(*events.ValidationError); ok {
			return ctx.WriteError(400, validationErr.Error())
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Run Put event
	if err := c.runPutEvent(ctx, merged); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Update document
	update := bson.M{"$set": sanitized}
	result, err := c.store.Update(ctx.Context(), bson.M{"id": id}, update)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	if result.MatchedCount == 0 {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Return updated document
	doc, err := c.store.FindOne(ctx.Context(), bson.M{"id": id})
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	// Run AfterCommit event
	go c.runAfterCommitEvent(ctx, doc, "PUT")
	
	return ctx.WriteJSON(doc)
}

func (c *Collection) handleDelete(ctx *appcontext.Context) error {
	id := ctx.GetID()
	if id == "" {
		return ctx.WriteError(400, "ID is required for DELETE requests")
	}
	
	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "DELETE"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Get the document to delete
	doc, err := c.store.FindOne(ctx.Context(), bson.M{"id": id})
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	if doc == nil {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Run Delete event
	if err := c.runDeleteEvent(ctx, doc); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Delete the document
	result, err := c.store.Remove(ctx.Context(), bson.M{"id": id})
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	if result.DeletedCount == 0 {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Run AfterCommit event
	go c.runAfterCommitEvent(ctx, doc, "DELETE")
	
	return ctx.WriteJSON(map[string]interface{}{
		"deleted": result.DeletedCount,
	})
}

func (c *Collection) handleCount(ctx *appcontext.Context) error {
	if !ctx.Session.IsRoot() {
		return ctx.WriteError(403, "Must be root to count")
	}
	
	query := c.sanitizeQuery(ctx.Query)
	delete(query, "id") // Remove id from query for count
	
	count, err := c.store.Count(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	return ctx.WriteJSON(map[string]interface{}{
		"count": count,
	})
}

func (c *Collection) validate(data bson.M, isCreate bool) error {
	if c.config == nil || c.config.Properties == nil {
		return nil
	}
	
	errors := make(map[string]string)
	
	for name, prop := range c.config.Properties {
		value, exists := data[name]
		
		if !exists || value == nil {
			if prop.Required && (isCreate || data[name] != nil) {
				errors[name] = "is required"
			}
			continue
		}
		
		if !c.validateType(value, prop.Type) {
			errors[name] = fmt.Sprintf("must be a %s", prop.Type)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %v", errors)
	}
	
	return nil
}

func (c *Collection) validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return true
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "date":
		switch value.(type) {
		case time.Time, string:
			return true
		}
		return false
	case "array":
		return reflect.TypeOf(value).Kind() == reflect.Slice
	case "object":
		_, ok := value.(map[string]interface{})
		if !ok {
			_, ok = value.(bson.M)
		}
		return ok
	}
	return false
}

func (c *Collection) sanitize(data bson.M) bson.M {
	if c.config == nil || c.config.Properties == nil {
		return data
	}
	
	sanitized := make(bson.M)
	
	for name, prop := range c.config.Properties {
		if value, exists := data[name]; exists {
			sanitized[name] = c.coerceType(value, prop.Type)
		}
	}
	
	return sanitized
}

func (c *Collection) coerceType(value interface{}, targetType string) interface{} {
	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value)
	case "number":
		if str, ok := value.(string); ok {
			if num, err := strconv.ParseFloat(str, 64); err == nil {
				return num
			}
		}
		return value
	case "boolean":
		if str, ok := value.(string); ok {
			return str == "true"
		}
		return value
	case "date":
		if str, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
		}
		return value
	default:
		return value
	}
}

func (c *Collection) sanitizeQuery(query bson.M) bson.M {
	if c.config == nil || c.config.Properties == nil {
		return query
	}
	
	sanitized := make(bson.M)
	
	for key, value := range query {
		// Allow MongoDB operators and id field
		if strings.HasPrefix(key, "$") || key == "id" {
			sanitized[key] = value
			continue
		}
		
		// Only allow defined properties
		if prop, exists := c.config.Properties[key]; exists {
			sanitized[key] = c.coerceType(value, prop.Type)
		}
	}
	
	return sanitized
}

func (c *Collection) extractQueryOptions(query bson.M) (*options.FindOptions, bson.M) {
	opts := options.Find()
	cleanQuery := make(bson.M)
	
	for key, value := range query {
		switch key {
		case "$sort", "$orderby":
			// Handle different value types for sort specification
			if sortSpec, ok := value.(bson.M); ok {
				opts.SetSort(sortSpec)
			} else if sortMap, ok := value.(map[string]interface{}); ok {
				// Convert map[string]interface{} to bson.M
				sortSpec := make(bson.M)
				for k, v := range sortMap {
					sortSpec[k] = v
				}
				opts.SetSort(sortSpec)
			}
		case "$limit":
			if limit, ok := value.(int64); ok {
				opts.SetLimit(limit)
			} else if limitFloat, ok := value.(float64); ok {
				opts.SetLimit(int64(limitFloat))
			} else if limitStr, ok := value.(string); ok {
				if limit, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
					opts.SetLimit(limit)
				}
			}
		case "$skip":
			if skip, ok := value.(int64); ok {
				opts.SetSkip(skip)
			} else if skipFloat, ok := value.(float64); ok {
				opts.SetSkip(int64(skipFloat))
			} else if skipStr, ok := value.(string); ok {
				if skip, err := strconv.ParseInt(skipStr, 10, 64); err == nil {
					opts.SetSkip(skip)
				}
			}
		case "$fields":
			// Handle field projection - support both object and string formats
			if fieldsSpec, ok := value.(bson.M); ok {
				opts.SetProjection(fieldsSpec)
			} else if fieldsMap, ok := value.(map[string]interface{}); ok {
				// Convert map[string]interface{} to bson.M
				fieldsSpec := make(bson.M)
				for k, v := range fieldsMap {
					fieldsSpec[k] = v
				}
				opts.SetProjection(fieldsSpec)
			} else if fieldsStr, ok := value.(string); ok {
				// Handle comma-separated field list like "title,content,id"
				fieldsSpec := make(bson.M)
				fields := strings.Split(fieldsStr, ",")
				for _, field := range fields {
					field = strings.TrimSpace(field)
					if field != "" {
						fieldsSpec[field] = 1
					}
				}
				opts.SetProjection(fieldsSpec)
			}
		default:
			cleanQuery[key] = value
		}
	}
	
	return opts, cleanQuery
}

func (c *Collection) setDefaults(data bson.M) {
	if c.config == nil || c.config.Properties == nil {
		return
	}
	
	for name, prop := range c.config.Properties {
		if _, exists := data[name]; !exists && prop.Default != nil {
			if prop.Default == "now" && prop.Type == "date" {
				data[name] = time.Now()
			} else {
				data[name] = prop.Default
			}
		}
	}
}

// Event runner methods
func (c *Collection) runBeforeRequestEvent(ctx *appcontext.Context, event string) error {
	data := bson.M{"event": event}
	return c.scriptManager.RunEvent(events.EventBeforeRequest, ctx, data)
}

func (c *Collection) runValidateEvent(ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(events.EventValidate, ctx, data)
}

func (c *Collection) runGetEvent(ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(events.EventGet, ctx, data)
}

func (c *Collection) runPostEvent(ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(events.EventPost, ctx, data)
}

func (c *Collection) runPutEvent(ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(events.EventPut, ctx, data)
}

func (c *Collection) runDeleteEvent(ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(events.EventDelete, ctx, data)
}

func (c *Collection) runAfterCommitEvent(ctx *appcontext.Context, data bson.M, event string) {
	// AfterCommit runs asynchronously and errors are ignored
	c.scriptManager.RunEvent(events.EventAfterCommit, ctx, data)
}

// Hot-reload methods
func (c *Collection) LoadHotReloadScript(eventType events.EventType, source string) error {
	return c.scriptManager.LoadHotReloadScript(eventType, source)
}

func (c *Collection) TestHotReloadScript(eventType events.EventType, ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(eventType, ctx, data)
}

func (c *Collection) TestScript(eventType events.EventType, ctx *appcontext.Context, data bson.M) error {
	return c.scriptManager.RunEvent(eventType, ctx, data)
}

func (c *Collection) GetHotReloadInfo() map[string]interface{} {
	return c.scriptManager.GetHotReloadInfo()
}

func (c *Collection) GetConfigPath() string {
	return c.configPath
}

func (c *Collection) ReloadScripts() error {
	return c.scriptManager.LoadScripts(c.configPath)
}

// isMongoCommand checks if the request body contains MongoDB operators
func (c *Collection) isMongoCommand(body bson.M) bool {
	for key := range body {
		if strings.HasPrefix(key, "$") {
			return true
		}
	}
	return false
}

// handleMongoCommand processes MongoDB command operations
func (c *Collection) handleMongoCommand(ctx *appcontext.Context, id string) error {
	query := bson.M{"id": id}
	
	// Get the existing document for events
	previous, err := c.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	if previous == nil {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Create a copy for the Put event (with anticipated changes)
	merged := make(bson.M)
	for k, v := range previous {
		merged[k] = v
	}
	
	// Apply command operations for validation (simulate the changes)
	c.simulateMongoOperations(merged, ctx.Body)
	
	// Run Validate event with simulated changes
	if err := c.runValidateEvent(ctx, merged); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		if validationErr, ok := err.(*events.ValidationError); ok {
			return ctx.WriteError(400, validationErr.Error())
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Run Put event with simulated changes
	if err := c.runPutEvent(ctx, merged); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}
	
	// Execute the actual MongoDB operation
	result, err := c.store.UpdateOne(ctx.Context(), query, ctx.Body)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	if result.MatchedCount == 0 {
		return ctx.WriteError(404, "Document not found")
	}
	
	// Return updated document
	doc, err := c.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	
	// Run AfterCommit event
	go c.runAfterCommitEvent(ctx, doc, "PUT")
	
	return ctx.WriteJSON(doc)
}

// simulateMongoOperations applies MongoDB operations to a document for validation
func (c *Collection) simulateMongoOperations(doc bson.M, operations bson.M) {
	for op, value := range operations {
		switch op {
		case "$inc":
			if incOps, ok := value.(bson.M); ok {
				for field, incValue := range incOps {
					if currentVal, exists := doc[field]; exists {
						if currentNum, ok := currentVal.(float64); ok {
							if incNum, ok := incValue.(float64); ok {
								doc[field] = currentNum + incNum
							}
						}
					}
				}
			}
		case "$set":
			if setOps, ok := value.(bson.M); ok {
				for field, setValue := range setOps {
					doc[field] = setValue
				}
			}
		case "$unset":
			if unsetOps, ok := value.(bson.M); ok {
				for field := range unsetOps {
					delete(doc, field)
				}
			}
		case "$push":
			if pushOps, ok := value.(bson.M); ok {
				for field, pushValue := range pushOps {
					if currentVal, exists := doc[field]; exists {
						if currentArray, ok := currentVal.([]interface{}); ok {
							doc[field] = append(currentArray, pushValue)
						}
					} else {
						doc[field] = []interface{}{pushValue}
					}
				}
			}
		case "$pull":
			if pullOps, ok := value.(bson.M); ok {
				for field, pullValue := range pullOps {
					if currentVal, exists := doc[field]; exists {
						if currentArray, ok := currentVal.([]interface{}); ok {
							newArray := make([]interface{}, 0)
							for _, item := range currentArray {
								if !c.valuesEqual(item, pullValue) {
									newArray = append(newArray, item)
								}
							}
							doc[field] = newArray
						}
					}
				}
			}
		case "$addToSet":
			if addOps, ok := value.(bson.M); ok {
				for field, addValue := range addOps {
					if currentVal, exists := doc[field]; exists {
						if currentArray, ok := currentVal.([]interface{}); ok {
							found := false
							for _, item := range currentArray {
								if c.valuesEqual(item, addValue) {
									found = true
									break
								}
							}
							if !found {
								doc[field] = append(currentArray, addValue)
							}
						}
					} else {
						doc[field] = []interface{}{addValue}
					}
				}
			}
		}
	}
}

// valuesEqual compares two values for equality (simplified comparison)
func (c *Collection) valuesEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}