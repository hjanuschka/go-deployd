package admin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/email"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/events"
)

type AdminHandler struct {
	db           database.DatabaseInterface
	router       *router.Router
	config       *Config
	resourcesDir string
	AuthHandler  *AuthHandler
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

func NewAdminHandler(db database.DatabaseInterface, router *router.Router, adminConfig *Config) *AdminHandler {
	// Load security configuration
	securityConfig, err := config.LoadSecurityConfig(config.GetConfigDir())
	if err != nil {
		fmt.Printf("Warning: Failed to load security config: %v\n", err)
		securityConfig = config.DefaultSecurityConfig()
	}
	
	// Log master key on startup
	fmt.Printf("🔐 Master Key: %s\n", securityConfig.MasterKey)
	
	authHandler := NewAuthHandler(db, securityConfig)
	
	return &AdminHandler{
		db:           db,
		router:       router,
		config:       adminConfig,
		resourcesDir: "./resources",
		AuthHandler:  authHandler,
	}
}

func (h *AdminHandler) RegisterRoutes(r *mux.Router) {
	admin := r.PathPrefix("/_admin").Subrouter()
	
	// Public authentication routes (no master key required)
	admin.HandleFunc("/auth/validate-master-key", h.AuthHandler.HandleMasterKeyValidation).Methods("POST")
	admin.HandleFunc("/auth/dashboard-login", h.handleDashboardLogin).Methods("POST")
	
	// Protected authentication routes (master key required)
	admin.HandleFunc("/auth/system-login", h.AuthHandler.RequireMasterKey(h.AuthHandler.HandleSystemLogin)).Methods("POST")
	admin.HandleFunc("/auth/security-info", h.AuthHandler.RequireMasterKey(h.AuthHandler.HandleGetSecurityInfo)).Methods("GET")
	admin.HandleFunc("/auth/regenerate-master-key", h.AuthHandler.RequireMasterKey(h.AuthHandler.HandleRegenerateMasterKey)).Methods("POST")
	admin.HandleFunc("/auth/create-user", h.AuthHandler.RequireMasterKey(h.AuthHandler.HandleCreateUser)).Methods("POST")
	
	// Protected admin routes (master key required)
	admin.HandleFunc("/info", h.AuthHandler.RequireMasterKey(h.getServerInfo)).Methods("GET")
	admin.HandleFunc("/collections", h.AuthHandler.RequireMasterKey(h.getCollections)).Methods("GET")
	admin.HandleFunc("/collections/{name}", h.AuthHandler.RequireMasterKey(h.getCollection)).Methods("GET")
	admin.HandleFunc("/collections/{name}", h.AuthHandler.RequireMasterKey(h.createCollection)).Methods("POST")
	admin.HandleFunc("/collections/{name}", h.AuthHandler.RequireMasterKey(h.updateCollection)).Methods("PUT")
	admin.HandleFunc("/collections/{name}", h.AuthHandler.RequireMasterKey(h.deleteCollection)).Methods("DELETE")
	
	// Protected event management endpoints (master key required)
	admin.HandleFunc("/collections/{name}/events", h.AuthHandler.RequireMasterKey(h.getEvents)).Methods("GET")
	admin.HandleFunc("/collections/{name}/events/{event}", h.AuthHandler.RequireMasterKey(h.updateEvent)).Methods("PUT")
	admin.HandleFunc("/collections/{name}/events/{event}/test", h.AuthHandler.RequireMasterKey(h.testEvent)).Methods("POST")
	
	// Security settings management (master key required)
	admin.HandleFunc("/settings/security", h.AuthHandler.RequireMasterKey(h.getSecuritySettings)).Methods("GET")
	admin.HandleFunc("/settings/security", h.AuthHandler.RequireMasterKey(h.updateSecuritySettings)).Methods("PUT")
	
	// Email settings management (master key required)
	admin.HandleFunc("/settings/email", h.AuthHandler.RequireMasterKey(h.getEmailSettings)).Methods("GET")
	admin.HandleFunc("/settings/email", h.AuthHandler.RequireMasterKey(h.updateEmailSettings)).Methods("PUT")
	admin.HandleFunc("/settings/email/test", h.AuthHandler.RequireMasterKey(h.testEmailSettings)).Methods("POST")
	admin.HandleFunc("/settings/email/templates", h.AuthHandler.RequireMasterKey(h.getEmailTemplates)).Methods("GET")
	admin.HandleFunc("/settings/email/templates", h.AuthHandler.RequireMasterKey(h.updateEmailTemplates)).Methods("PUT")
	
	// Logging endpoints (master key required)
	admin.HandleFunc("/logs", h.AuthHandler.RequireMasterKey(h.getLogs)).Methods("GET")
	admin.HandleFunc("/logs/files", h.AuthHandler.RequireMasterKey(h.getLogFiles)).Methods("GET")
	admin.HandleFunc("/logs/download", h.AuthHandler.RequireMasterKey(h.downloadLogs)).Methods("GET")
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
					
					// Convert properties to interface map and add hardcoded timestamp fields
					props := h.buildPropertiesMap(config.Properties)
					
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
	
	// Convert properties to interface map and add hardcoded timestamp fields
	props := h.buildPropertiesMap(config.Properties)
	
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

// buildPropertiesMap converts collection properties to interface map 
func (h *AdminHandler) buildPropertiesMap(configProperties map[string]resources.Property) map[string]interface{} {
	props := make(map[string]interface{})
	
	// Add properties from config
	for name, prop := range configProperties {
		propMap := map[string]interface{}{
			"type": prop.Type,
		}
		if prop.Required {
			propMap["required"] = true
		}
		if prop.Default != nil {
			propMap["default"] = prop.Default
		}
		if prop.Order != 0 {
			propMap["order"] = prop.Order
		}
		if prop.Unique {
			propMap["unique"] = true
		}
		if prop.System {
			propMap["system"] = true
			// Only set readonly for specific system fields that should never be edited
			if name == "createdAt" || name == "updatedAt" || name == "id" || name == "verificationToken" || name == "verificationExpires" {
				propMap["readonly"] = true
			}
		}
		props[name] = propMap
	}
	
	return props
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
			if order, exists := propMap["order"]; exists {
				if orderInt, ok := order.(float64); ok {
					prop.Order = int(orderInt)
				}
			}
			if unique, exists := propMap["unique"]; exists {
				if uniqueBool, ok := unique.(bool); ok {
					prop.Unique = uniqueBool
				}
			}
			if system, exists := propMap["system"]; exists {
				if systemBool, ok := system.(bool); ok {
					prop.System = systemBool
				}
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
	
	// Return created collection info with hardcoded timestamp fields
	// Convert the created properties to the proper format
	createdProps := h.buildPropertiesMap(configProps)
	response := CollectionInfo{
		Name:          name,
		DocumentCount: 0,
		Properties:    createdProps,
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
			if order, exists := propMap["order"]; exists {
				if orderInt, ok := order.(float64); ok {
					prop.Order = int(orderInt)
				}
			}
			if unique, exists := propMap["unique"]; exists {
				if uniqueBool, ok := unique.(bool); ok {
					prop.Unique = uniqueBool
				}
			}
			if system, exists := propMap["system"]; exists {
				if systemBool, ok := system.(bool); ok {
					prop.System = systemBool
				}
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
	
	// Convert the updated properties to include hardcoded timestamp fields
	updatedProps := h.buildPropertiesMap(configProps)
	response := CollectionInfo{
		Name:          name,
		DocumentCount: count,
		Properties:    updatedProps,
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

// handleDashboardLogin handles dashboard login with master key
func (h *AdminHandler) handleDashboardLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		MasterKey string `json:"masterKey"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// Validate master key
	if !h.AuthHandler.Security.ValidateMasterKey(req.MasterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid master key",
		})
		return
	}
	
	// Set master key cookie for dashboard access
	http.SetCookie(w, &http.Cookie{
		Name:     "masterKey",
		Value:    req.MasterKey,
		Path:     "/_dashboard",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	})
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Dashboard login successful",
	})
}

// getSecuritySettings returns the current security settings
func (h *AdminHandler) getSecuritySettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"jwtExpiration":     h.AuthHandler.Security.JWTExpiration,
		"allowRegistration": h.AuthHandler.Security.AllowRegistration,
		"hasMasterKey":      h.AuthHandler.Security.MasterKey != "",
	}
	
	json.NewEncoder(w).Encode(response)
}

// updateSecuritySettings updates the security settings
func (h *AdminHandler) updateSecuritySettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		JWTExpiration     string `json:"jwtExpiration"`
		AllowRegistration bool   `json:"allowRegistration"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// Update security config
	h.AuthHandler.Security.JWTExpiration = req.JWTExpiration
	h.AuthHandler.Security.AllowRegistration = req.AllowRegistration
	
	// Save updated configuration
	if err := config.SaveSecurityConfig(h.AuthHandler.Security, config.GetConfigDir()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save security settings",
		})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Security settings updated successfully",
	})
}

// getLogs returns application logs with optional filtering
func (h *AdminHandler) getLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get query parameters
	level := r.URL.Query().Get("level")
	filename := r.URL.Query().Get("file")
	limitStr := r.URL.Query().Get("limit")
	
	// Set default limit
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	logger := logging.GetLogger()
	logLevel := logging.LogLevel(level)
	
	logs, err := logger.ReadLogs(filename, logLevel)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to read logs",
			"error":   err.Error(),
		})
		return
	}
	
	// Apply limit (get most recent entries)
	if len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}
	
	response := map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
	
	json.NewEncoder(w).Encode(response)
}

// getLogFiles returns available log files
func (h *AdminHandler) getLogFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	logger := logging.GetLogger()
	files, err := logger.GetLogFiles()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to list log files",
			"error":   err.Error(),
		})
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"files":   files,
	}
	
	json.NewEncoder(w).Encode(response)
}

// downloadLogs allows downloading log files
func (h *AdminHandler) downloadLogs(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	level := r.URL.Query().Get("level")
	filename := r.URL.Query().Get("file")
	
	logger := logging.GetLogger()
	logPath := logger.GetLogPath(filename)
	
	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Log file not found"))
		return
	}
	
	// Set headers for file download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(logPath)))
	
	// If level filter is specified, filter the logs
	if level != "" {
		logs, err := logger.ReadLogs(filename, logging.LogLevel(level))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to read logs"))
			return
		}
		
		// Write filtered logs as JSONL
		for _, log := range logs {
			data, _ := json.Marshal(log)
			w.Write(data)
			w.Write([]byte("\n"))
		}
		return
	}
	
	// Otherwise, serve the raw file
	http.ServeFile(w, r, logPath)
}

// getEmailSettings returns the current email settings
func (h *AdminHandler) getEmailSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"provider": h.AuthHandler.Security.Email.Provider,
		"smtp": map[string]interface{}{
			"host":     h.AuthHandler.Security.Email.SMTP.Host,
			"port":     h.AuthHandler.Security.Email.SMTP.Port,
			"username": h.AuthHandler.Security.Email.SMTP.Username,
			"tls":      h.AuthHandler.Security.Email.SMTP.TLS,
			// Don't expose password
			"hasPassword": h.AuthHandler.Security.Email.SMTP.Password != "",
		},
		"ses": map[string]interface{}{
			"region": h.AuthHandler.Security.Email.SES.Region,
			// Don't expose credentials
			"hasAccessKeyId":     h.AuthHandler.Security.Email.SES.AccessKeyID != "",
			"hasSecretAccessKey": h.AuthHandler.Security.Email.SES.SecretAccessKey != "",
		},
		"from":                h.AuthHandler.Security.Email.From,
		"fromName":            h.AuthHandler.Security.Email.FromName,
		"requireVerification": h.AuthHandler.Security.RequireVerification,
	}
	
	json.NewEncoder(w).Encode(response)
}

// updateEmailSettings updates the email settings
func (h *AdminHandler) updateEmailSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Provider string `json:"provider"`
		SMTP     struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			TLS      bool   `json:"tls"`
		} `json:"smtp"`
		SES struct {
			Region          string `json:"region"`
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		} `json:"ses"`
		From                string `json:"from"`
		FromName            string `json:"fromName"`
		RequireVerification bool   `json:"requireVerification"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// Update email config
	h.AuthHandler.Security.Email.Provider = req.Provider
	h.AuthHandler.Security.Email.From = req.From
	h.AuthHandler.Security.Email.FromName = req.FromName
	h.AuthHandler.Security.RequireVerification = req.RequireVerification
	
	// Update SMTP settings
	h.AuthHandler.Security.Email.SMTP.Host = req.SMTP.Host
	h.AuthHandler.Security.Email.SMTP.Port = req.SMTP.Port
	h.AuthHandler.Security.Email.SMTP.Username = req.SMTP.Username
	h.AuthHandler.Security.Email.SMTP.TLS = req.SMTP.TLS
	// Only update password if provided
	if req.SMTP.Password != "" {
		h.AuthHandler.Security.Email.SMTP.Password = req.SMTP.Password
	}
	
	// Update SES settings
	h.AuthHandler.Security.Email.SES.Region = req.SES.Region
	// Only update credentials if provided
	if req.SES.AccessKeyID != "" {
		h.AuthHandler.Security.Email.SES.AccessKeyID = req.SES.AccessKeyID
	}
	if req.SES.SecretAccessKey != "" {
		h.AuthHandler.Security.Email.SES.SecretAccessKey = req.SES.SecretAccessKey
	}
	
	// Save updated configuration
	if err := config.SaveSecurityConfig(h.AuthHandler.Security, config.GetConfigDir()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save email settings",
		})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Email settings updated successfully",
	})
}

// testEmailSettings sends a test email
func (h *AdminHandler) testEmailSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		To string `json:"to"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	if req.To == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Recipient email address is required",
		})
		return
	}
	
	// Create email service and send test email
	emailService := email.NewEmailService(&h.AuthHandler.Security.Email)
	if err := emailService.TestEmail(req.To); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to send test email: %v", err),
		})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Test email sent successfully",
	})
}

// EmailTemplate represents a customizable email template
type EmailTemplate struct {
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	HTMLBody  string `json:"htmlBody"`
	TextBody  string `json:"textBody"`
	Variables []string `json:"variables"`
}

// getEmailTemplates returns available email templates
func (h *AdminHandler) getEmailTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Define available templates with their default content
	templates := []EmailTemplate{
		{
			Name:    "verification",
			Subject: "Verify your email address",
			HTMLBody: `<html>
<body>
	<h2>Welcome to Go-Deployd!</h2>
	<p>Hi {{.Username}},</p>
	<p>Please verify your email address by clicking the link below:</p>
	<p><a href="{{.VerificationURL}}" style="background-color: #4CAF50; color: white; padding: 14px 25px; text-decoration: none; display: inline-block;">Verify Email</a></p>
	<p>Or copy and paste this URL into your browser:</p>
	<p>{{.VerificationURL}}</p>
	<p>This link will expire in 24 hours.</p>
	<p>If you didn't create an account, please ignore this email.</p>
	<br>
	<p>Best regards,<br>Go-Deployd Team</p>
</body>
</html>`,
			TextBody: `Welcome to Go-Deployd!

Hi {{.Username}},

Please verify your email address by visiting this URL:
{{.VerificationURL}}

This link will expire in 24 hours.

If you didn't create an account, please ignore this email.

Best regards,
Go-Deployd Team`,
			Variables: []string{"Username", "VerificationURL"},
		},
		{
			Name:    "passwordReset",
			Subject: "Reset your password",
			HTMLBody: `<html>
<body>
	<h2>Password Reset Request</h2>
	<p>Hi {{.Username}},</p>
	<p>We received a request to reset your password. Click the link below to create a new password:</p>
	<p><a href="{{.ResetURL}}" style="background-color: #2196F3; color: white; padding: 14px 25px; text-decoration: none; display: inline-block;">Reset Password</a></p>
	<p>Or copy and paste this URL into your browser:</p>
	<p>{{.ResetURL}}</p>
	<p>This link will expire in 1 hour.</p>
	<p>If you didn't request a password reset, please ignore this email.</p>
	<br>
	<p>Best regards,<br>Go-Deployd Team</p>
</body>
</html>`,
			TextBody: `Password Reset Request

Hi {{.Username}},

We received a request to reset your password. Visit this URL to create a new password:
{{.ResetURL}}

This link will expire in 1 hour.

If you didn't request a password reset, please ignore this email.

Best regards,
Go-Deployd Team`,
			Variables: []string{"Username", "ResetURL"},
		},
	}
	
	// TODO: Load custom templates from storage if they exist
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

// updateEmailTemplates updates email templates
func (h *AdminHandler) updateEmailTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Templates []EmailTemplate `json:"templates"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// TODO: Save custom templates to storage
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Email templates updated successfully",
	})
}