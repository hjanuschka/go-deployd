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

	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

type CollectionConfig struct {
	Properties                map[string]Property                  `json:"properties"`
	EventConfig               map[string]events.EventConfiguration `json:"eventConfig,omitempty"`
	AllowAdditionalProperties bool                                 `json:"allowAdditionalProperties,omitempty"`
	IsBuiltin                 bool                                 `json:"isBuiltin,omitempty"`
}

type Collection struct {
	*BaseResource
	config           *CollectionConfig
	store            database.StoreInterface
	db               database.DatabaseInterface
	scriptManager    *events.UniversalScriptManager
	hotReloadManager *events.HotReloadGoManager
	configPath       string
	realtimeEmitter  events.RealtimeEmitter
}

func NewCollection(name string, config *CollectionConfig, db database.DatabaseInterface) *Collection {
	// Ensure required timestamp fields are present
	if config == nil {
		config = &CollectionConfig{Properties: make(map[string]Property)}
	}
	if config.Properties == nil {
		config.Properties = make(map[string]Property)
	}

	// Add required timestamp fields if not present
	config.Properties["createdAt"] = Property{
		Type:     "date",
		Required: false,
		Default:  "now",
	}
	config.Properties["updatedAt"] = Property{
		Type:     "date",
		Required: false,
		Default:  "now",
	}

	return &Collection{
		BaseResource:     NewBaseResource(name),
		config:           config,
		store:            db.CreateStore(name),
		db:               db,
		scriptManager:    events.NewUniversalScriptManager(),
		hotReloadManager: nil, // Will be initialized when needed
		realtimeEmitter:  nil, // Will be set when available
	}
}

func LoadCollectionFromConfig(name, configPath string, db database.DatabaseInterface) (Resource, error) {
	return LoadCollectionFromConfigWithEmitter(name, configPath, db, nil)
}

func LoadCollectionFromConfigWithEmitter(name, configPath string, db database.DatabaseInterface, emitter events.RealtimeEmitter) (Resource, error) {
	configFile := filepath.Join(configPath, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CollectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Check if this is a user collection (special case)
	if name == "users" || name == "user" {
		userCollection := NewUserCollection(name, &config, db)
		userCollection.configPath = configPath

		// Set the realtime emitter if provided
		if emitter != nil {
			userCollection.scriptManager.SetRealtimeEmitter(emitter)
			userCollection.realtimeEmitter = emitter
		}

		// Load event scripts with configuration
		if err := userCollection.scriptManager.LoadScriptsWithConfig(configPath, config.EventConfig); err != nil {
			// Scripts are optional, so don't fail if they don't exist
			fmt.Printf("Warning: Failed to load scripts for %s: %v\n", name, err)
		}

		return userCollection, nil
	}

	// Regular collection
	collection := NewCollection(name, &config, db)
	collection.configPath = configPath

	// Set the realtime emitter if provided
	if emitter != nil {
		collection.scriptManager.SetRealtimeEmitter(emitter)
		collection.realtimeEmitter = emitter
	}

	// Load event scripts with configuration
	if err := collection.scriptManager.LoadScriptsWithConfig(configPath, config.EventConfig); err != nil {
		// Scripts are optional, so don't fail if they don't exist
		fmt.Printf("Warning: Failed to load scripts for %s: %v\n", name, err)
	}

	return collection, nil
}

func (c *Collection) Handle(ctx *appcontext.Context) error {
	id := ctx.GetID()
	
	// Handle special endpoints for POST requests
	if ctx.Method == "POST" && id == "query" {
		return c.handleQuery(ctx)
	}
	
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
		logging.Info("🔍 SINGLE DOCUMENT GET REQUEST", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"documentId": id,
			"query":      ctx.Query,
		})

		// Get single document
		query := database.NewQueryBuilder().Where("id", "$eq", id)
		doc, err := c.store.FindOne(ctx.Context(), query)
		if err != nil {
			return ctx.WriteError(500, err.Error())
		}
		if doc == nil {
			return ctx.WriteError(404, "Document not found")
		}

		logging.Info("📄 DOCUMENT RETRIEVED", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"documentId": id,
			"dataKeys":   getDataKeys(doc),
		})

		// Check for $skipEvents parameter in query to bypass events
		skipEvents := ctx.Query["$skipEvents"] == "true"

		logging.Info("🎯 EVENT DECISION", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"skipEvents":   skipEvents,
			"willRunEvent": !skipEvents,
		})

		// Run Get event for single document (skip if $skipEvents is true)
		if !skipEvents {
			if err := c.runGetEvent(ctx, doc); err != nil {
				if scriptErr, ok := err.(*events.ScriptError); ok {
					return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
				}
				return ctx.WriteError(500, err.Error())
			}
		}

		logging.Info("📤 RETURNING DOCUMENT", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"documentId": id,
			"finalData":  doc,
		})

		return ctx.WriteJSON(doc)
	}

	// Get multiple documents
	// Check for $skipEvents parameter in query to bypass events
	skipEvents := ctx.Query["$skipEvents"] == "true"

	// First extract query options like $sort, $limit, $skip
	opts, cleanQuery := c.extractQueryOptions(ctx.Query)

	// Debug logging
	fmt.Printf("DEBUG: Collection.handleGet - Original query: %+v\n", ctx.Query)
	fmt.Printf("DEBUG: Collection.handleGet - Clean query: %+v\n", cleanQuery)
	fmt.Printf("DEBUG: Collection.handleGet - Query options: %+v\n", opts)

	// Then sanitize the remaining query and convert to QueryBuilder
	sanitizedQuery := c.sanitizeQuery(cleanQuery)
	fmt.Printf("DEBUG: Collection.handleGet - Sanitized query: %+v\n", sanitizedQuery)
	
	query := c.mapToQueryBuilder(sanitizedQuery)
	fmt.Printf("DEBUG: Collection.handleGet - QueryBuilder created, calling store.Find\n")
	fmt.Printf("DEBUG: Collection.handleGet - Store type: %T\n", c.store)

	docs, err := c.store.Find(ctx.Context(), query, opts)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	// Run Get event for each document (skip if $skipEvents is true)
	filteredDocs := make([]map[string]interface{}, 0)
	for _, doc := range docs {
		if !skipEvents {
			// Create a copy of the document for event processing
			eventDoc := make(map[string]interface{})
			for k, v := range doc {
				eventDoc[k] = v
			}

			if err := c.runGetEvent(ctx, eventDoc); err != nil {
				// Skip documents that fail the Get event (any error type)
				// This includes script errors, cancellations, validation failures, etc.
				continue
			}

			// Use the event-processed document as the result
			filteredDocs = append(filteredDocs, eventDoc)
		} else {
			// No events, use original document
			filteredDocs = append(filteredDocs, doc)
		}
	}

	return ctx.WriteJSON(filteredDocs)
}

func (c *Collection) handlePost(ctx *appcontext.Context) error {
	logging.Debug("POST request started", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"requestBodyKeys": getDataKeys(ctx.Body),
		"requestBody":     ctx.Body,
	})

	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "POST"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}

	logging.Debug("Starting Go validation", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"bodyKeys": getDataKeys(ctx.Body),
		"body":     ctx.Body,
	})

	// Validate and sanitize body
	if err := c.validate(ctx.Body, true); err != nil {
		logging.Debug("Go validation failed", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"error": err.Error(),
		})
		return ctx.WriteError(400, err.Error())
	}

	logging.Debug("Go validation passed, starting sanitization", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"bodyKeys": getDataKeys(ctx.Body),
		"body":     ctx.Body,
	})

	// Check for $skipEvents parameter to bypass all events (before sanitization)
	skipEvents := false
	if val, exists := ctx.Body["$skipEvents"]; exists {
		if skip, ok := val.(bool); ok && skip {
			skipEvents = true
		}
		// Remove $skipEvents from body so it doesn't interfere with validation/sanitization
		delete(ctx.Body, "$skipEvents")
	}

	sanitized := c.sanitize(ctx.Body)

	logging.Debug("Sanitization complete", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"sanitizedKeys": getDataKeys(sanitized),
		"sanitized":     sanitized,
	})

	// Set default values
	c.setDefaults(sanitized)

	// Run Validate event (skip if $skipEvents is true)
	if !skipEvents {
		if err := c.runValidateEvent(ctx, sanitized); err != nil {
			if scriptErr, ok := err.(*events.ScriptError); ok {
				return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
			}
			if validationErr, ok := err.(*events.ValidationError); ok {
				return ctx.WriteError(400, validationErr.Error())
			}
			return ctx.WriteError(500, err.Error())
		}
	}

	// Run Post event (skip if $skipEvents is true)
	if !skipEvents {
		if err := c.runPostEvent(ctx, sanitized); err != nil {
			if scriptErr, ok := err.(*events.ScriptError); ok {
				return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
			}
			return ctx.WriteError(500, err.Error())
		}
	}

	// Set timestamps after events (cannot be overridden by events)
	c.setTimestamps(sanitized, true)

	// Insert document
	result, err := c.store.Insert(ctx.Context(), sanitized)
	if err != nil {
		logging.Error("Failed to insert document", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"error": err.Error(),
			"data":  sanitized,
		})
		return ctx.WriteError(500, err.Error())
	}

	// Log successful document creation
	logging.Info("Document created", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"documentId": result,
		"fields":     len(sanitized),
	})

	// Emit collection change event for real-time updates
	if c.realtimeEmitter != nil {
		logging.Debug("Emitting collection change event", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"event": "created",
			"hasEmitter": c.realtimeEmitter != nil,
		})
		c.realtimeEmitter.EmitCollectionChange(c.name, "created", result)
		logging.Debug("Realtime emission completed", fmt.Sprintf("collection:%s", c.name), nil)
	} else {
		logging.Debug("No realtime emitter available", fmt.Sprintf("collection:%s", c.name), nil)
	}

	// Run AfterCommit event synchronously (can modify the response document)
	if resultDoc, ok := result.(map[string]interface{}); ok {
		c.runAfterCommitEvent(ctx, resultDoc, "POST")
		// Use the potentially modified resultDoc for the response
		return ctx.WriteJSON(resultDoc)
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
	query := database.NewQueryBuilder().Where("id", "$eq", id)
	previous, err := c.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	if previous == nil {
		return ctx.WriteError(404, "Document not found")
	}

	// Check for $skipEvents parameter to bypass all events (before sanitization)
	skipEvents := false
	if val, exists := ctx.Body["$skipEvents"]; exists {
		if skip, ok := val.(bool); ok && skip {
			skipEvents = true
		}
		// Remove $skipEvents from body so it doesn't interfere with validation/sanitization
		delete(ctx.Body, "$skipEvents")
	}

	// Validate and sanitize body
	if err := c.validate(ctx.Body, false); err != nil {
		return ctx.WriteError(400, err.Error())
	}

	sanitized := c.sanitize(ctx.Body)

	// Check if there are any fields to update after sanitization
	if len(sanitized) == 0 {
		// If skipEvents was specified but no actual fields to update, return the existing document
		if skipEvents {
			return ctx.WriteJSON(previous)
		}
		return ctx.WriteError(400, "No fields to update")
	}

	// Merge with existing document
	merged := make(map[string]interface{})
	for k, v := range previous {
		merged[k] = v
	}
	for k, v := range sanitized {
		merged[k] = v
	}

	// Run Validate event (skip if $skipEvents is true)
	if !skipEvents {
		if err := c.runValidateEvent(ctx, merged); err != nil {
			if scriptErr, ok := err.(*events.ScriptError); ok {
				return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
			}
			if validationErr, ok := err.(*events.ValidationError); ok {
				return ctx.WriteError(400, validationErr.Error())
			}
			return ctx.WriteError(500, err.Error())
		}
	}

	// Run Put event (skip if $skipEvents is true)
	if !skipEvents {
		if err := c.runPutEvent(ctx, merged); err != nil {
			if scriptErr, ok := err.(*events.ScriptError); ok {
				return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
			}
			return ctx.WriteError(500, err.Error())
		}
	}

	// Set timestamps after events (cannot be overridden by events)
	c.setTimestamps(sanitized, false)

	// Update document - for SQLite we need to update individual fields, not set the entire data
	updateQuery := database.NewQueryBuilder().Where("id", "$eq", id)
	updateBuilder := database.NewUpdateBuilder()
	updateCount := 0
	for key, value := range sanitized {
		// Skip the id field - it should not be updated
		if key != "id" {
			updateBuilder.Set(key, value)
			updateCount++
		}
	}

	// Check if we have any fields to update
	if updateCount == 0 {
		return ctx.WriteError(400, "No valid fields to update")
	}

	_, err = c.store.Update(ctx.Context(), updateQuery, updateBuilder)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	// Note: We don't check ModifiedCount() because it can be 0 if the document
	// already has the same values, which is a successful operation

	// Return updated document
	findQuery := database.NewQueryBuilder().Where("id", "$eq", id)
	doc, err := c.store.FindOne(ctx.Context(), findQuery)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	// Emit collection change event for real-time updates
	if c.realtimeEmitter != nil {
		c.realtimeEmitter.EmitCollectionChange(c.name, "updated", doc)
	}

	// Run AfterCommit event synchronously (can modify the response document)
	c.runAfterCommitEvent(ctx, doc, "PUT")

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
	query := database.NewQueryBuilder().Where("id", "$eq", id)
	doc, err := c.store.FindOne(ctx.Context(), query)
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
	deleteQuery := database.NewQueryBuilder().Where("id", "$eq", id)
	result, err := c.store.Remove(ctx.Context(), deleteQuery)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	if result.DeletedCount() == 0 {
		return ctx.WriteError(404, "Document not found")
	}

	// Emit collection change event for real-time updates
	if c.realtimeEmitter != nil {
		c.realtimeEmitter.EmitCollectionChange(c.name, "deleted", doc)
	}

	// Run AfterCommit event synchronously (blocks HTTP response until complete)
	c.runAfterCommitEvent(ctx, doc, "DELETE")

	return ctx.WriteJSON(map[string]interface{}{
		"deleted": result.DeletedCount(),
	})
}

func (c *Collection) handleCount(ctx *appcontext.Context) error {
	if !ctx.IsRoot {
		return ctx.WriteError(403, "Must be root to count")
	}

	sanitizedQuery := c.sanitizeQuery(ctx.Query)
	delete(sanitizedQuery, "id") // Remove id from query for count
	countQuery := c.mapToQueryBuilder(sanitizedQuery)

	count, err := c.store.Count(ctx.Context(), countQuery)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	return ctx.WriteJSON(map[string]interface{}{
		"count": count,
	})
}

func (c *Collection) handleQuery(ctx *appcontext.Context) error {
	// Run BeforeRequest event
	if err := c.runBeforeRequestEvent(ctx, "QUERY"); err != nil {
		if scriptErr, ok := err.(*events.ScriptError); ok {
			return ctx.WriteError(scriptErr.StatusCode, scriptErr.Message)
		}
		return ctx.WriteError(500, err.Error())
	}

	// Parse the complex query from request body
	queryData, exists := ctx.Body["query"]
	if !exists {
		return ctx.WriteError(400, "Query object required in request body")
	}

	queryMap, ok := queryData.(map[string]interface{})
	if !ok {
		return ctx.WriteError(400, "Query must be an object")
	}

	// Extract query options from body (if provided)
	var opts database.QueryOptions
	if optsData, exists := ctx.Body["options"]; exists {
		if optsMap, ok := optsData.(map[string]interface{}); ok {
			opts = c.parseQueryOptions(optsMap)
		}
	} else {
		// Set default options
		opts = database.QueryOptions{
			Sort:   make(map[string]int),
			Fields: make(map[string]int),
		}
		defaultLimit := int64(50)
		opts.Limit = &defaultLimit
	}

	// Check for $skipEvents parameter to bypass events
	skipEvents := false
	if val, exists := ctx.Body["$skipEvents"]; exists {
		if skip, ok := val.(bool); ok && skip {
			skipEvents = true
		}
	}
	if optsData, exists := ctx.Body["options"]; exists {
		if optsMap, ok := optsData.(map[string]interface{}); ok {
			if val, exists := optsMap["$skipEvents"]; exists {
				if skip, ok := val.(bool); ok && skip {
					skipEvents = true
				}
			}
		}
	}

	// Check for $forceMongo parameter to use direct MongoDB queries (bypass SQL translation)
	forceMongo := false
	if val, exists := ctx.Body["$forceMongo"]; exists {
		if force, ok := val.(bool); ok && force {
			forceMongo = true
		}
	}
	if optsData, exists := ctx.Body["options"]; exists {
		if optsMap, ok := optsData.(map[string]interface{}); ok {
			if val, exists := optsMap["$forceMongo"]; exists {
				if force, ok := val.(bool); ok && force {
					forceMongo = true
				}
			}
		}
	}

	// Debug logging
	fmt.Printf("DEBUG: Collection.handleQuery - Original query: %+v\n", queryMap)
	fmt.Printf("DEBUG: Collection.handleQuery - Query options: %+v\n", opts)
	fmt.Printf("DEBUG: Collection.handleQuery - forceMongo: %v\n", forceMongo)

	var docs []map[string]interface{}
	var err error

	if forceMongo {
		// Use direct MongoDB-style query execution (bypassing SQL translation)
		fmt.Printf("DEBUG: Collection.handleQuery - Using direct MongoDB query\n")
		
		// Check if store supports raw query interface
		if rawQueryStore, ok := c.store.(interface {
			FindWithRawQuery(ctx context.Context, mongoQuery interface{}, options map[string]interface{}) ([]map[string]interface{}, error)
		}); ok {
			// Convert QueryOptions to map for raw query interface
			optsMap := make(map[string]interface{})
			if opts.Limit != nil {
				optsMap["$limit"] = *opts.Limit
			}
			if opts.Skip != nil {
				optsMap["$skip"] = *opts.Skip
			}
			if len(opts.Sort) > 0 {
				optsMap["$sort"] = opts.Sort
			}
			if len(opts.Fields) > 0 {
				optsMap["$fields"] = opts.Fields
			}
			
			docs, err = rawQueryStore.FindWithRawQuery(ctx.Context(), queryMap, optsMap)
		} else {
			return ctx.WriteError(500, "Raw MongoDB queries not supported by this store implementation")
		}
	} else {
		// Use standard SQL translation
		fmt.Printf("DEBUG: Collection.handleQuery - Using SQL translation\n")
		
		// Sanitize and convert the query
		sanitizedQuery := c.sanitizeQuery(queryMap)
		fmt.Printf("DEBUG: Collection.handleQuery - Sanitized query: %+v\n", sanitizedQuery)
		
		query := c.mapToQueryBuilder(sanitizedQuery)
		fmt.Printf("DEBUG: Collection.handleQuery - QueryBuilder created, calling store.Find\n")

		// Execute the query
		docs, err = c.store.Find(ctx.Context(), query, opts)
	}
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	// Run Get event for each document (skip if $skipEvents is true)
	filteredDocs := make([]map[string]interface{}, 0)
	for _, doc := range docs {
		if !skipEvents {
			eventDoc := make(map[string]interface{})
			for k, v := range doc {
				eventDoc[k] = v
			}

			if err := c.runGetEvent(ctx, eventDoc); err != nil {
				continue // Skip documents that fail the Get event
			}

			filteredDocs = append(filteredDocs, eventDoc)
		} else {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	return ctx.WriteJSON(filteredDocs)
}

// Helper method to parse query options from request body
func (c *Collection) parseQueryOptions(optsMap map[string]interface{}) database.QueryOptions {
	opts := database.QueryOptions{
		Sort:   make(map[string]int),
		Fields: make(map[string]int),
	}

	if sortData, exists := optsMap["$sort"]; exists {
		if sortMap, ok := sortData.(map[string]interface{}); ok {
			for k, v := range sortMap {
				if sortDir, ok := v.(float64); ok {
					opts.Sort[k] = int(sortDir)
				}
			}
		}
	}

	if limitData, exists := optsMap["$limit"]; exists {
		if limit, ok := limitData.(float64); ok {
			limitInt := int64(limit)
			opts.Limit = &limitInt
		}
	}

	if skipData, exists := optsMap["$skip"]; exists {
		if skip, ok := skipData.(float64); ok {
			skipInt := int64(skip)
			opts.Skip = &skipInt
		}
	}

	if fieldsData, exists := optsMap["$fields"]; exists {
		if fieldsMap, ok := fieldsData.(map[string]interface{}); ok {
			for k, v := range fieldsMap {
				if include, ok := v.(float64); ok {
					opts.Fields[k] = int(include)
				}
			}
		}
	}

	// Apply default pagination if no limit was specified
	if opts.Limit == nil {
		defaultLimit := int64(50)
		opts.Limit = &defaultLimit
	}

	return opts
}

func (c *Collection) validate(data map[string]interface{}, isCreate bool) error {
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
		// Format errors as field: message pairs
		var errorStrings []string
		for field, message := range errors {
			errorStrings = append(errorStrings, fmt.Sprintf("%s: %s", field, message))
		}
		return fmt.Errorf("validation errors: %s", strings.Join(errorStrings, ", "))
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
			_, ok = value.(map[string]interface{})
		}
		return ok
	}
	return false
}

func (c *Collection) sanitize(data map[string]interface{}) map[string]interface{} {
	if c.config == nil || c.config.Properties == nil {
		return data
	}

	sanitized := make(map[string]interface{})

	for name, prop := range c.config.Properties {
		if value, exists := data[name]; exists {
			sanitized[name] = c.coerceType(value, prop.Type)
		}
	}

	return sanitized
}

// Helper function to get map keys for logging
func getDataKeys(data map[string]interface{}) []string {
	if data == nil {
		return nil
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
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

func (c *Collection) sanitizeQuery(query map[string]interface{}) map[string]interface{} {
	if c.config == nil || c.config.Properties == nil {
		return query
	}

	sanitized := make(map[string]interface{})

	for key, value := range query {
		// Allow MongoDB operators and id field
		if strings.HasPrefix(key, "$") || key == "id" {
			sanitized[key] = c.sanitizeQueryValue(value)
			continue
		}

		// Handle field[operator] pattern (e.g., "title[$regex]")
		if strings.Contains(key, "[") && strings.HasSuffix(key, "]") {
			// Extract field name and operator
			parts := strings.SplitN(key, "[", 2)
			if len(parts) == 2 {
				fieldName := parts[0]
				operator := strings.TrimSuffix(parts[1], "]")
				
				// Check if the field exists in properties
				if prop, exists := c.config.Properties[fieldName]; exists {
					// Create nested structure: field: {operator: value}
					if existing, hasExisting := sanitized[fieldName]; hasExisting {
						if existingMap, ok := existing.(map[string]interface{}); ok {
							existingMap[operator] = c.coerceType(value, prop.Type)
						} else {
							sanitized[fieldName] = map[string]interface{}{operator: c.coerceType(value, prop.Type)}
						}
					} else {
						sanitized[fieldName] = map[string]interface{}{operator: c.coerceType(value, prop.Type)}
					}
				}
			}
			continue
		}

		// Only allow defined properties
		if prop, exists := c.config.Properties[key]; exists {
			sanitized[key] = c.coerceType(value, prop.Type)
		}
	}

	return sanitized
}

// sanitizeQueryValue recursively sanitizes complex query values like $or arrays
func (c *Collection) sanitizeQueryValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		// Recursively sanitize nested objects
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = c.sanitizeQueryValue(val)
		}
		return result
	case []interface{}:
		// Handle arrays (like $or conditions)
		result := make([]interface{}, len(v))
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result[i] = c.sanitizeQuery(itemMap)
			} else {
				result[i] = item
			}
		}
		return result
	default:
		return value
	}
}

func (c *Collection) extractQueryOptions(query map[string]interface{}) (database.QueryOptions, map[string]interface{}) {
	opts := database.QueryOptions{
		Sort:   make(map[string]int),
		Fields: make(map[string]int),
	}
	cleanQuery := make(map[string]interface{})

	for key, value := range query {
		switch key {
		case "$sort", "$orderby":
			// Handle different value types for sort specification
			if sortMap, ok := value.(map[string]interface{}); ok {
				for k, v := range sortMap {
					if sortDir, ok := v.(int); ok {
						opts.Sort[k] = sortDir
					} else if sortDir, ok := v.(float64); ok {
						opts.Sort[k] = int(sortDir)
					}
				}
			}
		case "$limit":
			if limit, ok := value.(int64); ok {
				opts.Limit = &limit
			} else if limitFloat, ok := value.(float64); ok {
				limit := int64(limitFloat)
				opts.Limit = &limit
			} else if limitStr, ok := value.(string); ok {
				if limit, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
					opts.Limit = &limit
				}
			}
		case "$skip":
			if skip, ok := value.(int64); ok {
				opts.Skip = &skip
			} else if skipFloat, ok := value.(float64); ok {
				skip := int64(skipFloat)
				opts.Skip = &skip
			} else if skipStr, ok := value.(string); ok {
				if skip, err := strconv.ParseInt(skipStr, 10, 64); err == nil {
					opts.Skip = &skip
				}
			}
		case "$fields":
			// Handle field projection - support both object and string formats
			if fieldsMap, ok := value.(map[string]interface{}); ok {
				for k, v := range fieldsMap {
					if include, ok := v.(int); ok {
						opts.Fields[k] = include
					} else if include, ok := v.(float64); ok {
						opts.Fields[k] = int(include)
					}
				}
			} else if fieldsStr, ok := value.(string); ok {
				// Handle comma-separated field list like "title,content,id"
				fields := strings.Split(fieldsStr, ",")
				for _, field := range fields {
					field = strings.TrimSpace(field)
					if field != "" {
						opts.Fields[field] = 1
					}
				}
			}
		default:
			cleanQuery[key] = value
		}
	}

	// Apply default pagination if no limit was specified
	if opts.Limit == nil {
		defaultLimit := int64(50) // Default to 50 records per page
		opts.Limit = &defaultLimit
	}

	return opts, cleanQuery
}

func (c *Collection) mapToQueryBuilder(query map[string]interface{}) database.QueryBuilder {
	builder := database.NewQueryBuilder()

	for field, value := range query {
		if strings.HasPrefix(field, "$") {
			// Handle special MongoDB operators at root level
			switch field {
			case "$or":
				if orConditions, ok := value.([]interface{}); ok {
					var orBuilders []database.QueryBuilder
					for _, condition := range orConditions {
						if condMap, ok := condition.(map[string]interface{}); ok {
							orBuilder := c.mapToQueryBuilder(condMap)
							orBuilders = append(orBuilders, orBuilder)
						}
					}
					if len(orBuilders) > 0 {
						builder.Or(orBuilders...)
					}
				}
			case "$and":
				if andConditions, ok := value.([]interface{}); ok {
					var andBuilders []database.QueryBuilder
					for _, condition := range andConditions {
						if condMap, ok := condition.(map[string]interface{}); ok {
							andBuilder := c.mapToQueryBuilder(condMap)
							andBuilders = append(andBuilders, andBuilder)
						}
					}
					if len(andBuilders) > 0 {
						builder.And(andBuilders...)
					}
				}
			case "$nor":
				// $nor is not supported in the current QueryBuilder interface
				// For now, log a warning and skip
				fmt.Printf("WARNING: $nor operator is not yet supported, skipping\n")
			}
			continue
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			// Field has operators like {"age": {"$gt": 18}}
			for op, opValue := range valueMap {
				switch op {
				case "$in":
					if values, ok := opValue.([]interface{}); ok {
						builder.WhereIn(field, values)
					}
				case "$nin":
					if values, ok := opValue.([]interface{}); ok {
						builder.WhereNotIn(field, values)
					}
				case "$exists":
					// Handle $exists operator - for now, treat as basic field presence check
					if exists, ok := opValue.(bool); ok {
						if exists {
							builder.WhereNotNull(field)
						} else {
							builder.WhereNull(field)
						}
					}
				case "$ne":
					builder.Where(field, "$ne", opValue)
				default:
					builder.Where(field, op, opValue)
				}
			}
		} else {
			// Simple equality
			builder.Where(field, "$eq", value)
		}
	}

	return builder
}

func (c *Collection) setDefaults(data map[string]interface{}) {
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

// setTimestamps sets createdAt and updatedAt fields, ensuring they cannot be overridden
func (c *Collection) setTimestamps(data map[string]interface{}, isCreate bool) {
	now := time.Now()

	if isCreate {
		// On creation, always set createdAt to now (cannot be overridden)
		data["createdAt"] = now
	}

	// Always set updatedAt to now (cannot be overridden)
	data["updatedAt"] = now
}

// Event runner methods
func (c *Collection) runBeforeRequestEvent(ctx *appcontext.Context, event string) error {
	data := map[string]interface{}{"event": event}
	return c.scriptManager.RunEvent(events.EventBeforeRequest, ctx, data)
}

func (c *Collection) runValidateEvent(ctx *appcontext.Context, data map[string]interface{}) error {
	logging.Debug("Running validate event", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"dataKeys":   getDataKeys(data),
		"dataValues": data,
		"hasData":    data != nil,
		"dataLen":    len(data),
	})

	err := c.scriptManager.RunEvent(events.EventValidate, ctx, data)

	if err != nil {
		logging.Debug("Validate event returned error", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"error":     err.Error(),
			"errorType": fmt.Sprintf("%T", err),
		})
	} else {
		logging.Debug("Validate event completed successfully", fmt.Sprintf("collection:%s", c.name), nil)
	}

	return err
}

func (c *Collection) runGetEvent(ctx *appcontext.Context, data map[string]interface{}) error {
	logging.Debug("🔥 RUNNING GET EVENT", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"documentId": data["id"],
		"email":      data["email"],
		"hasScript":  c.scriptManager != nil,
	})

	err := c.scriptManager.RunEvent(events.EventGet, ctx, data)

	if err != nil {
		logging.Error("❌ GET EVENT FAILED", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"error": err.Error(),
		})
	}

	return err
}

func (c *Collection) runPostEvent(ctx *appcontext.Context, data map[string]interface{}) error {
	return c.scriptManager.RunEvent(events.EventPost, ctx, data)
}

func (c *Collection) runPutEvent(ctx *appcontext.Context, data map[string]interface{}) error {
	return c.scriptManager.RunEvent(events.EventPut, ctx, data)
}

func (c *Collection) runDeleteEvent(ctx *appcontext.Context, data map[string]interface{}) error {
	return c.scriptManager.RunEvent(events.EventDelete, ctx, data)
}

func (c *Collection) runAfterCommitEvent(ctx *appcontext.Context, data map[string]interface{}, event string) {
	// AfterCommit runs asynchronously and errors are ignored
	logging.Debug("Running AfterCommit event", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
		"event": event,
		"hasData": data != nil,
	})
	
	err := c.scriptManager.RunEvent(events.EventAfterCommit, ctx, data)
	if err != nil {
		logging.Debug("AfterCommit event completed with error (ignored)", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"event": event,
			"error": err.Error(),
		})
	} else {
		logging.Debug("AfterCommit event completed successfully", fmt.Sprintf("collection:%s", c.name), map[string]interface{}{
			"event": event,
		})
	}
}

// Hot-reload methods
func (c *Collection) LoadHotReloadScript(eventType events.EventType, source string) error {
	return c.scriptManager.LoadHotReloadScript(eventType, source)
}

func (c *Collection) TestHotReloadScript(eventType events.EventType, ctx *appcontext.Context, data map[string]interface{}) error {
	return c.scriptManager.RunEvent(eventType, ctx, data)
}

func (c *Collection) TestScript(eventType events.EventType, ctx *appcontext.Context, data map[string]interface{}) error {
	return c.scriptManager.RunEvent(eventType, ctx, data)
}

func (c *Collection) GetHotReloadInfo() map[string]interface{} {
	return c.scriptManager.GetHotReloadInfo()
}

func (c *Collection) GetConfigPath() string {
	return c.configPath
}

func (c *Collection) SetConfigPath(path string) {
	c.configPath = path
}

func (c *Collection) GetScriptManager() *events.UniversalScriptManager {
	return c.scriptManager
}

func (c *Collection) ReloadScripts() error {
	return c.scriptManager.LoadScripts(c.configPath)
}

// SetRealtimeEmitter sets the realtime emitter for the collection
func (c *Collection) SetRealtimeEmitter(emitter events.RealtimeEmitter) {
	c.realtimeEmitter = emitter
	if c.scriptManager != nil {
		c.scriptManager.SetRealtimeEmitter(emitter)
	}
}

// isMongoCommand checks if the request body contains MongoDB operators
func (c *Collection) isMongoCommand(body map[string]interface{}) bool {
	for key := range body {
		if strings.HasPrefix(key, "$") && key != "$skipEvents" {
			return true
		}
	}
	return false
}

// handleMongoCommand processes MongoDB command operations
func (c *Collection) handleMongoCommand(ctx *appcontext.Context, id string) error {
	query := database.NewQueryBuilder().Where("id", "$eq", id)

	// Get the existing document for events
	previous, err := c.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}
	if previous == nil {
		return ctx.WriteError(404, "Document not found")
	}

	// Create a copy for the Put event (with anticipated changes)
	merged := make(map[string]interface{})
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

	// Execute the actual MongoDB operation - convert body to UpdateBuilder
	updateBuilder := database.NewUpdateBuilder()
	for op, value := range ctx.Body {
		if valueMap, ok := value.(map[string]interface{}); ok {
			for field, fieldValue := range valueMap {
				switch op {
				case "$set":
					updateBuilder.Set(field, fieldValue)
				case "$inc":
					updateBuilder.Inc(field, fieldValue)
				case "$unset":
					updateBuilder.Unset(field)
				}
			}
		}
	}
	result, err := c.store.Update(ctx.Context(), query, updateBuilder)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	if result.ModifiedCount() == 0 {
		return ctx.WriteError(404, "Document not found")
	}

	// Return updated document
	query = database.NewQueryBuilder().Where("id", "$eq", id)
	doc, err := c.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, err.Error())
	}

	// Run AfterCommit event synchronously (can modify the response document)
	c.runAfterCommitEvent(ctx, doc, "PUT")

	return ctx.WriteJSON(doc)
}

// simulateMongoOperations applies MongoDB operations to a document for validation
func (c *Collection) simulateMongoOperations(doc map[string]interface{}, operations map[string]interface{}) {
	for op, value := range operations {
		switch op {
		case "$inc":
			if incOps, ok := value.(map[string]interface{}); ok {
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
			if setOps, ok := value.(map[string]interface{}); ok {
				for field, setValue := range setOps {
					doc[field] = setValue
				}
			}
		case "$unset":
			if unsetOps, ok := value.(map[string]interface{}); ok {
				for field := range unsetOps {
					delete(doc, field)
				}
			}
		case "$push":
			if pushOps, ok := value.(map[string]interface{}); ok {
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
			if pullOps, ok := value.(map[string]interface{}); ok {
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
			if addOps, ok := value.(map[string]interface{}); ok {
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

// GetConfig returns the collection configuration
func (c *Collection) GetConfig() *CollectionConfig {
	return c.config
}
