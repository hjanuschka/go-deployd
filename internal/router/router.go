package router

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/sessions"
)

type Router struct {
	resources     []resources.Resource
	db            *database.Database
	sessions      *sessions.SessionStore
	development   bool
	configPath    string
}

func New(db *database.Database, sessions *sessions.SessionStore, development bool, configPath string) *Router {
	r := &Router{
		db:          db,
		sessions:    sessions,
		development: development,
		configPath:  configPath,
	}
	
	r.loadResources()
	
	return r
}

func (r *Router) loadResources() {
	if r.configPath == "" {
		r.configPath = "./resources"
	}
	
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
			
			if _, err := os.Stat(configFile); err == nil {
				// Load collection resource
				collection, err := resources.LoadCollectionFromConfig(resourceName, path, r.db)
				if err != nil {
					log.Printf("Failed to load collection %s: %v", resourceName, err)
					return nil
				}
				
				r.resources = append(r.resources, collection)
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
	
	// Get or create session
	session, err := r.sessions.GetSessionFromRequest(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session error: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Set session cookie
	r.sessions.SetSessionCookie(w, session)
	
	// Find matching resource
	resource := r.findMatchingResource(req.URL.Path)
	if resource == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	
	// Create context
	ctx := context.New(req, w, resource, session)
	
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
	// Sort resources by path specificity (longer paths first)
	sort.Slice(r.resources, func(i, j int) bool {
		return len(strings.Split(r.resources[i].GetPath(), "/")) > len(strings.Split(r.resources[j].GetPath(), "/"))
	})
}