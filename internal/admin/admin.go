package admin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/sessions"
)

type AdminHandler struct {
	db           database.DatabaseInterface
	router       *router.Router
	config       *Config
	resourcesDir string
	authHandler  *AuthHandler
}

type Config struct {
	Port           int
	DatabaseHost   string
	DatabasePort   int
	DatabaseName   string
	Development    bool
}

type ServerInfo struct {
	Version     string    `json:"version"`
	GoVersion   string    `json:"goVersion"`
	Uptime      string    `json:"uptime"`
	Database    string    `json:"database"`
	Environment string    `json:"environment"`
	StartTime   time.Time `json:"startTime"`
}

type CollectionInfo struct {
	Name          string                 `json:"name"`
	DocumentCount int64                  `json:"documentCount"`
	Properties    map[string]interface{} `json:"properties"`
	LastModified  time.Time              `json:"lastModified"`
}

func NewAdminHandler(db database.DatabaseInterface, router *router.Router, adminConfig *Config, sessions *sessions.SessionStore) *AdminHandler {
	// Load security configuration
	securityConfig, err := config.LoadSecurityConfig(config.GetConfigDir())
	if err != nil {
		fmt.Printf("Warning: Failed to load security config: %v\n", err)
		securityConfig = config.DefaultSecurityConfig()
	}
	
	authHandler := NewAuthHandler(db, sessions, securityConfig)
	
	return &AdminHandler{
		db:           db,
		router:       router,
		config:       adminConfig,
		resourcesDir: "./resources",
		authHandler:  authHandler,
	}
}

func (h *AdminHandler) RegisterRoutes(r *mux.Router) {
	admin := r.PathPrefix("/_admin").Subrouter()
	
	// System authentication routes (master key based)
	admin.HandleFunc("/auth/system-login", h.authHandler.HandleSystemLogin).Methods("POST")
	admin.HandleFunc("/auth/validate-master-key", h.authHandler.HandleMasterKeyValidation).Methods("POST")
	admin.HandleFunc("/auth/security-info", h.authHandler.HandleGetSecurityInfo).Methods("GET")
	admin.HandleFunc("/auth/regenerate-master-key", h.authHandler.HandleRegenerateMasterKey).Methods("POST")
	
	// Existing admin routes
	admin.HandleFunc("/info", h.getServerInfo).Methods("GET")
	admin.HandleFunc("/collections", h.getCollections).Methods("GET")
	admin.HandleFunc("/collections/{name}", h.getCollection).Methods("GET")
	admin.HandleFunc("/collections/{name}", h.createCollection).Methods("POST")
	admin.HandleFunc("/collections/{name}", h.updateCollection).Methods("PUT")
	admin.HandleFunc("/collections/{name}", h.deleteCollection).Methods("DELETE")
	
	// Event management endpoints
	admin.HandleFunc("/collections/{name}/events", h.getEvents).Methods("GET")
	admin.HandleFunc("/collections/{name}/events/{event}", h.updateEvent).Methods("PUT")
	admin.HandleFunc("/collections/{name}/events/{event}/test", h.testEvent).Methods("POST")
}

func (h *AdminHandler) getServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	info := ServerInfo{
		Version:     "1.0.0",
		GoVersion:   runtime.Version(),
		Database:    string(h.db.GetType()) + " - Connected",
		Environment: "development",
		StartTime:   time.Now().Add(-2 * time.Hour), // Mock uptime
		Uptime:      "2h 15m",
	}
	
	if h.config.Development {
		info.Environment = "development"
	} else {
		info.Environment = "production"
	}
	
	json.NewEncoder(w).Encode(info)
}

func (h *AdminHandler) getCollections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	collections := []CollectionInfo{}
	
	// Scan resources directory for actual collections
	if _, err := os.Stat(h.resourcesDir); err == nil {
		err := filepath.WalkDir(h.resourcesDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			
			if d.IsDir() && path != h.resourcesDir {
				configFile := filepath.Join(path, "config.json")
				if _, err := os.Stat(configFile); err == nil {
					// Load collection config
					data, err := os.ReadFile(configFile)
					if err != nil {
						return nil
					}
					
					var config resources.CollectionConfig
					if err := json.Unmarshal(data, &config); err != nil {
						return nil
					}
					
					// Get document count from database
					collectionName := filepath.Base(path)
					store := h.db.CreateStore(collectionName)
					count, _ := store.Count(r.Context(), database.NewQueryBuilder())
					
					// Get file modification time
					stat, _ := os.Stat(configFile)
					
					// Convert properties to interface map
					props := make(map[string]interface{})
					for name, prop := range config.Properties {
						propMap := map[string]interface{}{
							"type": prop.Type,
						}
						if prop.Required {
							propMap["required"] = true
						}
						if prop.Default != nil {
							propMap["default"] = prop.Default
						}
						props[name] = propMap
					}
					
					collections = append(collections, CollectionInfo{
						Name:          collectionName,
						DocumentCount: count,
						Properties:    props,
						LastModified:  stat.ModTime(),
					})
				}
			}
			
			return nil
		})
		
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan collections: %v", err), http.StatusInternalServerError)
			return
		}
	}
	
	json.NewEncoder(w).Encode(collections)
}

func (h *AdminHandler) getCollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	w.Header().Set("Content-Type", "application/json")
	
	collectionDir := filepath.Join(h.resourcesDir, name)
	configFile := filepath.Join(collectionDir, "config.json")
	
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	
	// Load collection config
	data, err := os.ReadFile(configFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read config: %v", err), http.StatusInternalServerError)
		return
	}
	
	var config resources.CollectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse config: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get document count from database
	store := h.db.CreateStore(name)
	count, _ := store.Count(r.Context(), database.NewQueryBuilder())
	
	// Get file modification time
	stat, _ := os.Stat(configFile)
	
	// Convert properties to interface map
	props := make(map[string]interface{})
	for propName, prop := range config.Properties {
		propMap := map[string]interface{}{
			"type": prop.Type,
		}
		if prop.Required {
			propMap["required"] = true
		}
		if prop.Default != nil {
			propMap["default"] = prop.Default
		}
		props[propName] = propMap
	}
	
	collection := CollectionInfo{
		Name:          name,
		DocumentCount: count,
		Properties:    props,
		LastModified:  stat.ModTime(),
	}
	
	json.NewEncoder(w).Encode(collection)
}

func getString(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (h *AdminHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	w.Header().Set("Content-Type", "application/json")
	
	var properties map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&properties); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Create collection directory
	collectionDir := filepath.Join(h.resourcesDir, name)
	if err := os.MkdirAll(collectionDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert properties to proper format
	configProps := make(map[string]resources.Property)
	for propName, propData := range properties {
		if propMap, ok := propData.(map[string]interface{}); ok {
			prop := resources.Property{
				Type: getString(propMap, "type"),
			}
			if required, exists := propMap["required"]; exists {
				if reqBool, ok := required.(bool); ok {
					prop.Required = reqBool
				}
			}
			if defaultVal, exists := propMap["default"]; exists {
				prop.Default = defaultVal
			}
			configProps[propName] = prop
		}
	}
	
	// Create config structure
	config := resources.CollectionConfig{
		Properties: configProps,
	}
	
	// Write config.json
	configFile := filepath.Join(collectionDir, "config.json")
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal config: %v", err), http.StatusInternalServerError)
		return
	}
	
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write config: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Load and register the collection with router
	collection, err := resources.LoadCollectionFromConfig(name, collectionDir, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load collection: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Add to router (we need to implement this)
	h.router.AddResource(collection)
	
	// Return created collection info
	response := CollectionInfo{
		Name:          name,
		DocumentCount: 0,
		Properties:    properties,
		LastModified:  time.Now(),
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	w.Header().Set("Content-Type", "application/json")
	
	var properties map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&properties); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	collectionDir := filepath.Join(h.resourcesDir, name)
	configFile := filepath.Join(collectionDir, "config.json")
	
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	
	// Convert properties to proper format
	configProps := make(map[string]resources.Property)
	for propName, propData := range properties {
		if propMap, ok := propData.(map[string]interface{}); ok {
			prop := resources.Property{
				Type: getString(propMap, "type"),
			}
			if required, exists := propMap["required"]; exists {
				if reqBool, ok := required.(bool); ok {
					prop.Required = reqBool
				}
			}
			if defaultVal, exists := propMap["default"]; exists {
				prop.Default = defaultVal
			}
			configProps[propName] = prop
		}
	}
	
	// Create config structure
	config := resources.CollectionConfig{
		Properties: configProps,
	}
	
	// Write updated config.json
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal config: %v", err), http.StatusInternalServerError)
		return
	}
	
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write config: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Reload collection in router
	collection, err := resources.LoadCollectionFromConfig(name, collectionDir, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reload collection: %v", err), http.StatusInternalServerError)
		return
	}
	
	h.router.UpdateResource(name, collection)
	
	// Get document count from database
	store := h.db.CreateStore(name)
	count, _ := store.Count(r.Context(), database.NewQueryBuilder())
	
	response := CollectionInfo{
		Name:          name,
		DocumentCount: count,
		Properties:    properties,
		LastModified:  time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	w.Header().Set("Content-Type", "application/json")
	
	collectionDir := filepath.Join(h.resourcesDir, name)
	
	// Remove from router first
	h.router.RemoveResource(name)
	
	// Delete the directory
	if err := os.RemoveAll(collectionDir); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete collection: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"deleted": name,
		"success": true,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Event management methods
func (h *AdminHandler) getEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionName := vars["name"]
	
	w.Header().Set("Content-Type", "application/json")
	
	collectionDir := filepath.Join(h.resourcesDir, collectionName)
	
	if _, err := os.Stat(collectionDir); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	
	scripts := make(map[string]string)
	types := make(map[string]string)
	
	eventFiles := []string{"get", "post", "put", "delete", "validate", "beforerequest", "aftercommit"}
	
	for _, eventName := range eventFiles {
		// Check for JavaScript file
		jsFile := filepath.Join(collectionDir, eventName+".js")
		if data, err := os.ReadFile(jsFile); err == nil {
			scripts[eventName] = string(data)
			types[eventName] = "js"
		}
		
		// Check for Go file (only if no JS file)
		if _, exists := scripts[eventName]; !exists {
			goFile := filepath.Join(collectionDir, eventName+".go")
			if data, err := os.ReadFile(goFile); err == nil {
				scripts[eventName] = string(data)
				types[eventName] = "go"
			}
		}
	}
	
	// Get hot-reload info if available
	hotReload := make(map[string]interface{})
	if collection := h.router.GetCollection(collectionName); collection != nil {
		hotReload = collection.GetHotReloadInfo()
	}
	
	response := map[string]interface{}{
		"scripts":    scripts,
		"types":      types,
		"hotReload":  hotReload,
		"collection": collectionName,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) updateEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionName := vars["name"]
	eventName := vars["event"]
	
	w.Header().Set("Content-Type", "application/json")
	
	var request struct {
		Script string `json:"script"`
		Type   string `json:"type"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	collectionDir := filepath.Join(h.resourcesDir, collectionName)
	
	if _, err := os.Stat(collectionDir); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	
	// Remove existing event files
	jsFile := filepath.Join(collectionDir, eventName+".js")
	goFile := filepath.Join(collectionDir, eventName+".go")
	os.Remove(jsFile)
	os.Remove(goFile)
	
	var filePath string
	if request.Type == "go" {
		filePath = goFile
	} else {
		filePath = jsFile
	}
	
	// Write the script file
	if err := os.WriteFile(filePath, []byte(request.Script), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write script: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Reload scripts in collection
	if collection := h.router.GetCollection(collectionName); collection != nil {
		if request.Type == "go" {
			// Use hot-reload for Go scripts
			eventType := events.EventType(strings.ToUpper(eventName))
			if err := collection.LoadHotReloadScript(eventType, request.Script); err != nil {
				http.Error(w, fmt.Sprintf("Failed to load Go script: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			// Reload all scripts for JS
			if err := collection.ReloadScripts(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to reload scripts: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}
	
	response := map[string]interface{}{
		"success":    true,
		"message":    eventName + " event updated successfully",
		"type":       request.Type,
		"hotReload":  request.Type == "go",
		"collection": collectionName,
		"event":      eventName,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) testEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionName := vars["name"]
	eventName := vars["event"]
	
	w.Header().Set("Content-Type", "application/json")
	
	var request struct {
		Data       map[string]interface{} `json:"data"`
		User       map[string]interface{} `json:"user"`
		Query      map[string]interface{} `json:"query"`
		ScriptType string                 `json:"scriptType"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	collection := h.router.GetCollection(collectionName)
	if collection == nil {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	
	// Create a mock context for testing
	// TODO: Create proper test context
	response := map[string]interface{}{
		"success":    true,
		"duration":   50,
		"data":       request.Data,
		"collection": collectionName,
		"event":      eventName,
		"scriptType": request.ScriptType,
	}
	
	// Simulate some validation
	if eventName == "validate" {
		if title, exists := request.Data["title"]; !exists || title == "" {
			response["success"] = false
			response["errors"] = map[string]string{
				"title": "Title is required",
			}
		}
	}
	
	json.NewEncoder(w).Encode(response)
}