package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hjanuschka/go-deployd/internal/admin"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/sessions"
)

type Config struct {
	Port           int
	DatabaseType   string
	DatabaseHost   string
	DatabasePort   int
	DatabaseName   string
	ConfigPath     string
	Development    bool
}

type Server struct {
	config      *Config
	db          database.DatabaseInterface
	sessions    *sessions.SessionStore
	router      *router.Router
	adminHandler *admin.AdminHandler
	upgrader    websocket.Upgrader
	httpMux     *mux.Router
}

func New(config *Config) (*Server, error) {
	dbConfig := &database.Config{
		Host: config.DatabaseHost,
		Port: config.DatabasePort,
		Name: config.DatabaseName,
	}

	// Determine database type
	var dbType database.DatabaseType
	switch config.DatabaseType {
	case "mongodb":
		dbType = database.DatabaseTypeMongoDB
	case "sqlite":
		dbType = database.DatabaseTypeSQLite
	case "mysql":
		dbType = database.DatabaseTypeMySQL
	case "postgres":
		dbType = database.DatabaseTypePostgres
	default:
		dbType = database.DatabaseTypeMongoDB // Default to MongoDB
	}

	db, err := database.NewDatabase(dbType, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sessionStore := sessions.New(db, config.Development)

	s := &Server{
		config:   config,
		db:       db,
		sessions: sessionStore,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: Implement proper origin checking
			},
		},
		httpMux: mux.NewRouter(),
	}

	s.router = router.New(s.db, s.sessions, config.Development, config.ConfigPath)

	// Create admin handler
	adminConfig := &admin.Config{
		Port:           config.Port,
		DatabaseHost:   config.DatabaseHost,
		DatabasePort:   config.DatabasePort,
		DatabaseName:   config.DatabaseName,
		Development:    config.Development,
	}
	s.adminHandler = admin.NewAdminHandler(s.db, s.router, adminConfig, s.sessions)

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	// WebSocket endpoint for real-time features
	s.httpMux.HandleFunc("/socket.io/", s.handleWebSocket)

	// Admin API routes
	s.adminHandler.RegisterRoutes(s.httpMux)

	// Built-in API routes (like original Deployd)
	s.setupBuiltinRoutes()

	// Serve dashboard static files
	dashboardPath := filepath.Join("web", "dashboard")
	s.httpMux.PathPrefix("/_dashboard/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/_dashboard/")
		if path == "" || path == "/" {
			// Serve index.html for dashboard root
			path = "index.html"
		}
		fullPath := filepath.Join(dashboardPath, path)
		
		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// If file doesn't exist and it's not an asset, serve index.html (SPA routing)
			if !strings.HasPrefix(path, "assets/") {
				fullPath = filepath.Join(dashboardPath, "index.html")
			} else {
				http.NotFound(w, r)
				return
			}
		}
		
		http.ServeFile(w, r, fullPath)
	})

	// Root route handling
	s.setupRootRoute()

	// API routes - delegate to our custom router (lowest priority)
	s.httpMux.PathPrefix("/").HandlerFunc(s.router.ServeHTTP)
}

func (s *Server) setupBuiltinRoutes() {
	// Built-in collections list endpoint (like original Deployd)
	s.httpMux.HandleFunc("/collections", s.handleCollections).Methods("GET")
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get collections from router
	resources := s.router.GetResources()
	collections := make([]string, 0)
	
	for _, resource := range resources {
		// Only include actual collections, not internal resources
		name := resource.GetName()
		if name != "" && !strings.HasPrefix(name, "_") {
			collections = append(collections, name)
		}
	}
	
	// Return collection names as a simple array (like original Deployd)
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		http.Error(w, "Failed to encode collections", http.StatusInternalServerError)
		return
	}
}


func (s *Server) setupRootRoute() {
	s.httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			s.router.ServeHTTP(w, r)
			return
		}
		
		if s.config.Development {
			// Redirect to dashboard in development
			http.Redirect(w, r, "/_dashboard/", http.StatusTemporaryRedirect)
		} else {
			// In production, try to serve index.html from public directory
			indexPath := filepath.Join("./public", "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
			} else {
				// No index.html, redirect to dashboard
				http.Redirect(w, r, "/_dashboard/", http.StatusTemporaryRedirect)
			}
		}
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Handle WebSocket connection
	// TODO: Implement full Socket.IO compatibility
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("WebSocket message received: %s", p)

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpMux.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Server) CreateStore(namespace string) database.StoreInterface {
	return s.db.CreateStore(namespace)
}