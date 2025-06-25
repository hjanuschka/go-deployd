package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hjanuschka/go-deployd/internal/admin"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/metrics"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/sessions"
)

type Config struct {
	Port               int
	DatabaseType       string
	DatabaseHost       string
	DatabasePort       int
	DatabaseName       string
	DatabaseUsername   string
	DatabasePassword   string
	DatabaseSSL        bool
	ConfigPath         string
	Development        bool
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
		Host:     config.DatabaseHost,
		Port:     config.DatabasePort,
		Name:     config.DatabaseName,
		Username: config.DatabaseUsername,
		Password: config.DatabasePassword,
		SSL:      config.DatabaseSSL,
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

	// Initialize logging system
	if err := logging.InitializeLogger("./logs"); err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Log server startup
	logging.Info("Starting go-deployd server", "server", map[string]interface{}{
		"port":         config.Port,
		"database":     config.DatabaseType,
		"development":  config.Development,
	})

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
	// Apply metrics middleware to all routes
	s.httpMux.Use(metrics.HTTPMiddleware)

	// WebSocket endpoint for real-time features
	s.httpMux.HandleFunc("/socket.io/", s.handleWebSocket)

	// Admin API routes
	s.adminHandler.RegisterRoutes(s.httpMux)

	// Built-in API routes (like original Deployd)
	s.setupBuiltinRoutes()

	// Metrics API routes
	s.setupMetricsRoutes()

	// Serve dashboard static files with authentication
	dashboardPath := filepath.Join("web", "dashboard")
	s.httpMux.PathPrefix("/_dashboard/").HandlerFunc(s.serveDashboardWithAuth(dashboardPath))

	// Root route handling
	s.setupRootRoute()

	// API routes - delegate to our custom router (lowest priority)
	s.httpMux.PathPrefix("/").HandlerFunc(s.router.ServeHTTP)
}

func (s *Server) setupBuiltinRoutes() {
	// Built-in collections list endpoint (like original Deployd)
	s.httpMux.HandleFunc("/collections", s.handleCollections).Methods("GET")
}

func (s *Server) setupMetricsRoutes() {
	// Metrics API endpoints
	s.httpMux.HandleFunc("/_dashboard/api/metrics/detailed", s.handleDetailedMetrics).Methods("GET")
	s.httpMux.HandleFunc("/_dashboard/api/metrics/aggregated", s.handleAggregatedMetrics).Methods("GET")
	s.httpMux.HandleFunc("/_dashboard/api/metrics/system", s.handleSystemStats).Methods("GET")
	s.httpMux.HandleFunc("/_dashboard/api/metrics/collections", s.handleCollectionsList).Methods("GET")
	s.httpMux.HandleFunc("/_dashboard/api/metrics/events", s.handleEventMetrics).Methods("GET")
	s.httpMux.HandleFunc("/_dashboard/api/metrics/periods", s.handlePeriodsMetrics).Methods("GET")
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
	// Shutdown V8 pool for JavaScript events
	if v8Pool := events.GetV8Pool(); v8Pool != nil {
		v8Pool.Shutdown()
		logging.Info("V8 pool shut down", "server", nil)
	}
	
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Server) CreateStore(namespace string) database.StoreInterface {
	return s.db.CreateStore(namespace)
}

// serveDashboardWithAuth returns a handler that serves dashboard files with master key authentication
func (s *Server) serveDashboardWithAuth(dashboardPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path
		path := strings.TrimPrefix(r.URL.Path, "/_dashboard/")
		
		// Allow login page without authentication
		if path == "login" || path == "login/" || strings.HasPrefix(path, "assets/") {
			s.serveDashboardFile(w, r, dashboardPath, path)
			return
		}
		
		// Check for master key authentication
		masterKey := r.Header.Get("X-Master-Key")
		if masterKey == "" {
			// Also check cookie
			if cookie, err := r.Cookie("masterKey"); err == nil {
				masterKey = cookie.Value
			}
		}
		
		// Validate master key
		if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
			// Redirect to login page for dashboard requests
			if path == "" || path == "/" || !strings.HasPrefix(path, "assets/") {
				http.Redirect(w, r, "/_dashboard/login", http.StatusTemporaryRedirect)
				return
			}
			
			// For API requests, return 401
			if strings.HasPrefix(path, "api/") {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Authentication required",
					"message": "Master key required for dashboard access",
				})
				return
			}
			
			// For other requests, serve login page
			path = "login"
		}
		
		s.serveDashboardFile(w, r, dashboardPath, path)
	}
}

// serveDashboardFile serves a specific dashboard file
func (s *Server) serveDashboardFile(w http.ResponseWriter, r *http.Request, dashboardPath, path string) {
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
}

func (s *Server) handleDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collection := r.URL.Query().Get("collection")
	since := time.Now().Add(-24 * time.Hour) // Last 24 hours
	if sinceParam := r.URL.Query().Get("since"); sinceParam != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceParam); err == nil {
			since = parsedTime
		}
	}

	collector := metrics.GetGlobalCollector()
	var metricsData []metrics.Metric
	if collection != "" && collection != "all" {
		metricsData = collector.GetDetailedMetricsByCollection(collection, since)
	} else {
		metricsData = collector.GetDetailedMetrics(since)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metricsData,
		"since":   since,
		"count":   len(metricsData),
	})
}

func (s *Server) handleAggregatedMetrics(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collection := r.URL.Query().Get("collection")
	since := time.Now().Add(-7 * 24 * time.Hour) // Last 7 days
	if sinceParam := r.URL.Query().Get("since"); sinceParam != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceParam); err == nil {
			since = parsedTime
		}
	}

	collector := metrics.GetGlobalCollector()
	metricsData := collector.GetAggregatedMetrics("hourly", collection, since)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metricsData,
		"since":   since,
		"count":   len(metricsData),
	})
}

func (s *Server) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collector := metrics.GetGlobalCollector()
	stats := collector.GetSystemStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleCollectionsList(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collector := metrics.GetGlobalCollector()
	collections := collector.GetCollections()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"collections": collections,
	})
}

func (s *Server) handleEventMetrics(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collection := r.URL.Query().Get("collection")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "hourly"
	}

	// Default to last 24 hours for detailed, 7 days for others
	var since time.Time
	switch period {
	case "detailed":
		since = time.Now().Add(-24 * time.Hour)
	case "hourly":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "daily":
		since = time.Now().Add(-30 * 24 * time.Hour)
	case "monthly":
		since = time.Now().Add(-365 * 24 * time.Hour)
	}

	if sinceParam := r.URL.Query().Get("since"); sinceParam != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceParam); err == nil {
			since = parsedTime
		}
	}

	collector := metrics.GetGlobalCollector()
	eventMetrics := collector.GetEventMetrics(collection)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": eventMetrics,
		"since":  since,
		"period": period,
	})
}

func (s *Server) handlePeriodsMetrics(w http.ResponseWriter, r *http.Request) {
	// Check for master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}
	
	if !s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required",
		})
		return
	}

	collection := r.URL.Query().Get("collection")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "daily"
	}

	var since time.Time
	switch period {
	case "daily":
		since = time.Now().Add(-6 * 30 * 24 * time.Hour) // 6 months
	case "monthly":
		since = time.Now().Add(-12 * 30 * 24 * time.Hour) // 12 months
	default:
		since = time.Now().Add(-30 * 24 * time.Hour) // 30 days
	}

	if sinceParam := r.URL.Query().Get("since"); sinceParam != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceParam); err == nil {
			since = parsedTime
		}
	}

	collector := metrics.GetGlobalCollector()
	metricsData := collector.GetAggregatedMetrics(period, collection, since)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metricsData,
		"since":   since,
		"period":  period,
		"count":   len(metricsData),
	})
}