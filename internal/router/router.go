package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/storage"
)

type Router struct {
	resources       []resources.Resource
	db              database.DatabaseInterface
	development     bool
	configPath      string
	jwtManager      *auth.JWTManager
	realtimeEmitter events.RealtimeEmitter
	storageManager  *storage.Manager
}

func New(db database.DatabaseInterface, development bool, configPath string) *Router {
	return NewWithEmitter(db, development, configPath, nil)
}

func NewWithEmitter(db database.DatabaseInterface, development bool, configPath string, emitter events.RealtimeEmitter) *Router {
	return NewWithStorage(db, development, configPath, emitter, nil)
}

func NewWithStorage(db database.DatabaseInterface, development bool, configPath string, emitter events.RealtimeEmitter, storageManager *storage.Manager) *Router {
	// Load security config to set up JWT
	var jwtManager *auth.JWTManager
	securityConfig, err := config.LoadSecurityConfig(config.GetConfigDir())
	if err == nil {
		jwtDuration, err := time.ParseDuration(securityConfig.JWTExpiration)
		if err != nil {
			jwtDuration = 24 * time.Hour
		}
		jwtManager = auth.NewJWTManager(securityConfig.JWTSecret, jwtDuration)
	}

	r := &Router{
		db:              db,
		development:     development,
		configPath:      configPath,
		jwtManager:      jwtManager,
		realtimeEmitter: emitter,
		storageManager:  storageManager,
	}

	r.loadResources()

	return r
}

func (r *Router) loadResources() {
	if r.configPath == "" {
		r.configPath = "./resources"
	}

	// Always create built-in users collection first
	r.createBuiltinUsersCollection()
	
	// Create built-in files resource if storage manager is available
	r.createBuiltinFilesResource()

	// Create default collection resources if config path exists
	if _, err := os.Stat(r.configPath); os.IsNotExist(err) {
		// Create a default "todos" collection for demo purposes
		todosCollection := resources.NewCollection("todos", &resources.CollectionConfig{
			Properties: map[string]resources.Property{
				"title": {
					Type:     "string",
					Required: true,
				},
				"completed": {
					Type:    "boolean",
					Default: false,
				},
				"createdAt": {
					Type:    "date",
					Default: "now",
				},
			},
		}, r.db)

		// Set the realtime emitter if available
		if r.realtimeEmitter != nil {
			todosCollection.SetRealtimeEmitter(r.realtimeEmitter)
		}

		r.resources = append(r.resources, todosCollection)
		return
	}

	// Load resources from config directory
	err := filepath.Walk(r.configPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != r.configPath {
			// This is a resource directory
			resourceName := filepath.Base(path)
			configFile := filepath.Join(path, "config.json")

			log.Printf("🔍 Found resource directory: %s", resourceName)
			if _, err := os.Stat(configFile); err == nil {
				log.Printf("📁 Loading collection %s from %s", resourceName, path)
				// Load collection resource with emitter
				collection, err := resources.LoadCollectionFromConfigWithEmitter(resourceName, path, r.db, r.realtimeEmitter)
				if err != nil {
					log.Printf("❌ Failed to load collection %s: %v", resourceName, err)
					return nil
				}

				log.Printf("✅ Successfully loaded collection %s", resourceName)
				r.resources = append(r.resources, collection)
			} else {
				log.Printf("⚠️  No config.json found for resource %s: %v", resourceName, err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Failed to load resources: %v", err)
	}

	r.sortResources()
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check for authentication (JWT token or master key only)
	isAuthenticated := false
	isRoot := false
	userID := ""
	username := ""

	// 1. Check JWT token authentication
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") && r.jwtManager != nil {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := r.jwtManager.ValidateToken(token); err == nil {
			isAuthenticated = true
			isRoot = claims.IsRoot
			userID = claims.UserID
			username = claims.Username
		}
	}

	// 2. Check for master key authentication (fallback for admin operations)
	if !isAuthenticated {
		masterKey := req.Header.Get("X-Master-Key")
		if masterKey != "" {
			// Load security config to validate master key
			securityConfig, err := config.LoadSecurityConfig(config.GetConfigDir())
			if err == nil && securityConfig.ValidateMasterKey(masterKey) {
				isAuthenticated = true
				isRoot = true
				userID = "root"
				username = "root"
			}
		}
	}

	// Find matching resource
	resource := r.findMatchingResource(req.URL.Path)
	if resource == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Create context with authentication data
	authData := &context.AuthData{
		UserID:          userID,
		Username:        username,
		IsRoot:          isRoot,
		IsAuthenticated: isAuthenticated,
	}
	ctx := context.New(req, w, resource, authData, r.development)

	// Handle the request
	if err := resource.Handle(ctx); err != nil {
		log.Printf("Resource handler error: %v", err)
		ctx.WriteError(500, err.Error())
	}
}

func (r *Router) findMatchingResource(path string) resources.Resource {
	for _, resource := range r.resources {
		if r.pathMatches(path, resource.GetPath()) {
			return resource
		}
	}
	return nil
}

func (r *Router) pathMatches(requestPath, resourcePath string) bool {
	if resourcePath == "/" {
		return true
	}

	// Remove trailing slash from resource path
	resourcePath = strings.TrimSuffix(resourcePath, "/")

	return strings.HasPrefix(requestPath, resourcePath)
}

func (r *Router) GetResources() []resources.Resource {
	return r.resources
}

func (r *Router) AddResource(resource resources.Resource) {
	r.resources = append(r.resources, resource)
	r.sortResources()
}

func (r *Router) UpdateResource(name string, resource resources.Resource) {
	for i, res := range r.resources {
		if res.GetName() == name {
			r.resources[i] = resource
			break
		}
	}
	r.sortResources()
}

func (r *Router) RemoveResource(name string) {
	for i, res := range r.resources {
		if res.GetName() == name {
			r.resources = append(r.resources[:i], r.resources[i+1:]...)
			break
		}
	}
}

func (r *Router) GetCollection(name string) *resources.Collection {
	for _, res := range r.resources {
		if res.GetName() == name {
			if collection, ok := res.(*resources.Collection); ok {
				return collection
			}
		}
	}
	return nil
}

func (r *Router) sortResources() {
	// Sort resources by path length first (longer paths first), then by path segments
	sort.Slice(r.resources, func(i, j int) bool {
		pathI := r.resources[i].GetPath()
		pathJ := r.resources[j].GetPath()
		
		// First, compare by actual path length (longer first)
		if len(pathI) != len(pathJ) {
			return len(pathI) > len(pathJ)
		}
		
		// If same length, compare by number of segments (more specific first)
		return len(strings.Split(pathI, "/")) > len(strings.Split(pathJ, "/"))
	})
}

// createBuiltinUsersCollection creates the built-in users collection with default fields
func (r *Router) createBuiltinUsersCollection() {
	// Define the current built-in schema for users collection
	currentBuiltinSchema := map[string]resources.Property{
		"username": {
			Type:     "string",
			Required: true,
			Unique:   true,
			System:   true, // Mark as system field
		},
		"email": {
			Type:     "string",
			Required: true,
			Unique:   true,
			System:   true, // Mark as system field
		},
		"password": {
			Type:     "string",
			Required: true,
			System:   true, // Mark as system field
		},
		"role": {
			Type:    "string",
			Default: "user",
			System:  true, // Mark as system field
		},
		"active": {
			Type:    "boolean",
			Default: false, // Users start inactive until email verification
			System:  true,  // Mark as system field
		},
		"isVerified": {
			Type:    "boolean",
			Default: false,
			System:  true, // Mark as system field
		},
		"verificationToken": {
			Type:   "string",
			System: true, // Mark as system field
		},
		"verificationExpires": {
			Type:   "date",
			System: true, // Mark as system field
		},
		"createdAt": {
			Type:    "date",
			Default: "now",
			System:  true, // Mark as system field
		},
		"updatedAt": {
			Type:    "date",
			Default: "now",
			System:  true, // Mark as system field
		},
	}

	// Check if users collection config already exists and migrate if needed
	usersConfigPath := filepath.Join(r.configPath, "users")
	configFile := filepath.Join(usersConfigPath, "config.json")

	var finalConfig *resources.CollectionConfig

	if _, err := os.Stat(configFile); err == nil {
		// Existing config found - perform migration
		log.Printf("🔄 Found existing users collection, checking for schema migration...")
		finalConfig = r.migrateBuiltinCollection(configFile, currentBuiltinSchema)
	} else {
		// No existing config - create new one with built-in schema
		log.Printf("📦 Creating new built-in users collection...")
		finalConfig = &resources.CollectionConfig{
			Properties:                currentBuiltinSchema,
			AllowAdditionalProperties: true,
			IsBuiltin:                 true,
		}

		// Save the initial config file
		if err := r.saveCollectionConfig(usersConfigPath, finalConfig); err != nil {
			log.Printf("Warning: Failed to save users collection config: %v", err)
		}
	}

	// Create UserCollection with the final config
	usersCollection := resources.NewUserCollection("users", finalConfig, r.db)

	// CRITICAL: Set the configPath and load event scripts for built-in users collection
	usersCollection.Collection.SetConfigPath(usersConfigPath)

	// Set the realtime emitter if available
	if r.realtimeEmitter != nil {
		usersCollection.Collection.SetRealtimeEmitter(r.realtimeEmitter)
	}

	// Load event scripts if the users directory exists
	if _, err := os.Stat(usersConfigPath); err == nil {
		log.Printf("🔥 LOADING EVENT SCRIPTS FOR BUILT-IN USERS COLLECTION...")
		if err := usersCollection.Collection.GetScriptManager().LoadScriptsWithConfig(usersConfigPath, finalConfig.EventConfig); err != nil {
			log.Printf("❌ Failed to load event scripts for users collection: %v", err)
		} else {
			log.Printf("✅ Successfully loaded event scripts for built-in users collection")
		}
	} else {
		log.Printf("⚠️ Users config directory not found: %s", usersConfigPath)
	}

	r.resources = append(r.resources, usersCollection)
}

// createBuiltinFilesResource creates the built-in files resource for file upload/management
func (r *Router) createBuiltinFilesResource() {
	// Only create files resource if storage manager is available
	if r.storageManager == nil {
		log.Printf("⚠️ Storage manager not available, skipping files resource creation")
		return
	}
	
	log.Printf("📁 Creating built-in files resource...")
	
	// Create files resource
	filesResource := resources.NewFilesResource("files", r.storageManager, r.db)
	
	log.Printf("✅ Successfully created built-in files resource")
	r.resources = append(r.resources, filesResource)
}

// migrateBuiltinCollection handles schema migration for built-in collections
func (r *Router) migrateBuiltinCollection(configFile string, currentBuiltinSchema map[string]resources.Property) *resources.CollectionConfig {
	// Read existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Warning: Failed to read existing config, creating new one: %v", err)
		return &resources.CollectionConfig{
			Properties:                currentBuiltinSchema,
			AllowAdditionalProperties: true,
			IsBuiltin:                 true,
		}
	}

	var existingConfig resources.CollectionConfig
	if err := json.Unmarshal(data, &existingConfig); err != nil {
		log.Printf("Warning: Failed to parse existing config, creating new one: %v", err)
		return &resources.CollectionConfig{
			Properties:                currentBuiltinSchema,
			AllowAdditionalProperties: true,
			IsBuiltin:                 true,
		}
	}

	// Ensure properties map exists
	if existingConfig.Properties == nil {
		existingConfig.Properties = make(map[string]resources.Property)
	}

	// Track if any changes were made
	migrationNeeded := false

	// Add or update built-in system properties
	for fieldName, builtinProperty := range currentBuiltinSchema {
		if existingProperty, exists := existingConfig.Properties[fieldName]; exists {
			// Property exists - check if it needs updating
			if !r.propertiesEqual(existingProperty, builtinProperty) {
				log.Printf("  📝 Updating system field '%s' in users collection", fieldName)
				existingConfig.Properties[fieldName] = builtinProperty
				migrationNeeded = true
			}
		} else {
			// Property doesn't exist - add it
			log.Printf("  ➕ Adding new system field '%s' to users collection", fieldName)
			existingConfig.Properties[fieldName] = builtinProperty
			migrationNeeded = true
		}
	}

	// Ensure collection is marked as built-in and allows additional properties
	if !existingConfig.IsBuiltin {
		log.Printf("  🏗️ Marking users collection as built-in")
		existingConfig.IsBuiltin = true
		migrationNeeded = true
	}
	if !existingConfig.AllowAdditionalProperties {
		log.Printf("  🔧 Enabling additional properties for users collection")
		existingConfig.AllowAdditionalProperties = true
		migrationNeeded = true
	}

	// Save updated config if migration was needed
	if migrationNeeded {
		if err := r.saveCollectionConfig(filepath.Dir(configFile), &existingConfig); err != nil {
			log.Printf("Warning: Failed to save migrated config: %v", err)
		} else {
			log.Printf("✅ Successfully migrated users collection schema")
		}
	} else {
		log.Printf("✅ Users collection schema is up to date")
	}

	return &existingConfig
}

// propertiesEqual compares two Property structs for equality
func (r *Router) propertiesEqual(a, b resources.Property) bool {
	return a.Type == b.Type &&
		a.Required == b.Required &&
		a.Unique == b.Unique &&
		a.System == b.System &&
		fmt.Sprintf("%v", a.Default) == fmt.Sprintf("%v", b.Default)
}

// saveCollectionConfig saves a collection configuration to disk
func (r *Router) saveCollectionConfig(configDir string, config *resources.CollectionConfig) error {
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config.json
	configFile := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
