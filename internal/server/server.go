package server

import (
	"context"
	"embed"
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
	"github.com/hjanuschka/go-deployd/internal/auth"
	appconfig "github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/email"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/metrics"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/swagger"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Port             int
	DatabaseType     string
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseSSL      bool
	ConfigPath       string
	Development      bool
}

type Server struct {
	config         *Config
	db             database.DatabaseInterface
	router         *router.Router
	adminHandler   *admin.AdminHandler
	upgrader       websocket.Upgrader
	httpMux        *mux.Router
	jwtManager     *auth.JWTManager
	securityConfig *appconfig.SecurityConfig
	dashboardFS    *embed.FS
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

	// Initialize logging system with enhanced configuration
	logLevel := logging.INFO
	if config.Development {
		logLevel = logging.DEBUG
	}

	// Check for LOG_LEVEL environment variable override
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			logLevel = logging.DEBUG
		case "INFO":
			logLevel = logging.INFO
		case "WARN", "WARNING":
			logLevel = logging.WARN
		case "ERROR":
			logLevel = logging.ERROR
		}
	}

	if err := logging.InitializeLogger(logging.Config{
		LogDir:    "./logs",
		DevMode:   config.Development,
		MinLevel:  logLevel,
		Component: "server",
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Load security configuration
	configDir := appconfig.GetConfigDir()
	securityConfig, err := appconfig.LoadSecurityConfig(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load security config: %w", err)
	}

	// Parse JWT expiration duration
	jwtDuration, err := time.ParseDuration(securityConfig.JWTExpiration)
	if err != nil {
		jwtDuration = 24 * time.Hour // Default to 24 hours if parsing fails
		logging.Error("Failed to parse JWT expiration, using default 24h", "auth", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(securityConfig.JWTSecret, jwtDuration)

	// Log server startup
	logging.Info("Starting go-deployd server", "server", map[string]interface{}{
		"port":        config.Port,
		"database":    config.DatabaseType,
		"development": config.Development,
	})

	s := &Server{
		config: config,
		db:     db,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: Implement proper origin checking
			},
		},
		httpMux:        mux.NewRouter(),
		jwtManager:     jwtManager,
		securityConfig: securityConfig,
	}

	s.router = router.New(s.db, config.Development, config.ConfigPath)

	// Create admin handler
	adminConfig := &admin.Config{
		Port:         config.Port,
		DatabaseHost: config.DatabaseHost,
		DatabasePort: config.DatabasePort,
		DatabaseName: config.DatabaseName,
		Development:  config.Development,
	}
	s.adminHandler = admin.NewAdminHandler(s.db, s.router, adminConfig)

	s.setupRoutes()

	// Start background jobs
	go s.startUserCleanupJob()

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

	// Authentication routes
	s.setupAuthRoutes()

	// Metrics API routes
	s.setupMetricsRoutes()

	// API documentation routes
	s.setupSwaggerRoutes()

	// Add redirect for missing trailing slash on dashboard
	s.httpMux.HandleFunc("/_dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/_dashboard/", http.StatusMovedPermanently)
	})

	// Setup dashboard routes (embedded or filesystem based on availability)
	s.setupDashboardRoutes()

	// Serve self-test page
	s.httpMux.HandleFunc("/self-test.html", s.handleSelfTest)

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
					"error":   "Authentication required",
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

// validateDashboardAuth checks both master key and JWT authentication for dashboard routes
func (s *Server) validateDashboardAuth(r *http.Request) bool {
	// First check master key authentication
	masterKey := r.Header.Get("X-Master-Key")
	if masterKey == "" {
		if cookie, err := r.Cookie("masterKey"); err == nil {
			masterKey = cookie.Value
		}
	}

	if s.adminHandler.AuthHandler.Security.ValidateMasterKey(masterKey) {
		return true
	}

	// Check JWT authentication with isRoot=true
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") && s.jwtManager != nil {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := s.jwtManager.ValidateToken(token); err == nil && claims.IsRoot {
			return true
		}
	}

	return false
}

func (s *Server) handleDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
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
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
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
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
		})
		return
	}

	collector := metrics.GetGlobalCollector()
	stats := collector.GetSystemStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleCollectionsList(w http.ResponseWriter, r *http.Request) {
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
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
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
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
	if !s.validateDashboardAuth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Authentication required - master key or root JWT token needed",
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

func (s *Server) setupAuthRoutes() {
	// Login endpoint
	s.httpMux.HandleFunc("/auth/login", s.handleLogin).Methods("POST", "OPTIONS")
	// Token validation endpoint
	s.httpMux.HandleFunc("/auth/validate", s.handleTokenValidation).Methods("GET", "OPTIONS")
	// User info endpoint
	s.httpMux.HandleFunc("/auth/me", s.handleMe).Methods("GET", "OPTIONS")
	// Email verification endpoint
	s.httpMux.HandleFunc("/auth/verify", s.handleEmailVerification).Methods("POST", "GET", "OPTIONS")
	// Resend verification email endpoint
	s.httpMux.HandleFunc("/auth/resend-verification", s.handleResendVerification).Methods("POST", "OPTIONS")
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	MasterKey string `json:"masterKey,omitempty"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string                 `json:"token"`
	ExpiresAt int64                  `json:"expiresAt"`
	User      map[string]interface{} `json:"user,omitempty"`
	IsRoot    bool                   `json:"isRoot"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	var userID, username string
	var isRoot bool
	var userData map[string]interface{}

	// Check for master key authentication
	if req.MasterKey != "" {
		if s.securityConfig.ValidateMasterKey(req.MasterKey) {
			userID = "root"
			username = "root"
			isRoot = true
		} else {
			http.Error(w, `{"error": "Invalid master key"}`, http.StatusUnauthorized)
			return
		}
	} else if req.Username != "" && req.Password != "" {
		// Authenticate user with username/password
		user, err := s.authenticateUser(req.Username, req.Password)
		if err != nil {
			http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		userID = getStringFromMap(user, "id")
		username = getStringFromMap(user, "username")
		role := getStringFromMap(user, "role")
		isRoot = (role == "admin")

		// Remove password and other sensitive fields from user data
		userData = make(map[string]interface{})
		for k, v := range user {
			if k != "password" && k != "salt" {
				userData[k] = v
			}
		}
		userData["role"] = role
	} else {
		http.Error(w, `{"error": "Username/password or masterKey required"}`, http.StatusBadRequest)
		return
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(userID, username, isRoot)
	if err != nil {
		logging.Error("Failed to generate JWT token", "auth", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, `{"error": "Failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	// Calculate expiration time
	duration, _ := time.ParseDuration(s.securityConfig.JWTExpiration)
	expiresAt := time.Now().Add(duration).Unix()

	// Prepare response
	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		IsRoot:    isRoot,
		User:      userData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTokenValidation(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		http.Error(w, `{"error": "Bearer token required"}`, http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Invalid token: %s"}`, err.Error()), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":    true,
		"userID":   claims.UserID,
		"username": claims.Username,
		"isRoot":   claims.IsRoot,
		"exp":      claims.ExpiresAt.Unix(),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		http.Error(w, `{"error": "Bearer token required"}`, http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Invalid token: %s"}`, err.Error()), http.StatusUnauthorized)
		return
	}

	// For root users, return basic info
	if claims.IsRoot {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       claims.UserID,
			"username": claims.Username,
			"isRoot":   claims.IsRoot,
		})
		return
	}

	// For regular users, fetch user data from users collection
	// Fetch user data by ID
	store := s.db.CreateStore("users")

	// Create query to find user by ID
	query := database.NewQueryBuilder()
	query.Where("id", "=", claims.UserID)

	userData, err := store.FindOne(r.Context(), query)
	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userData)
}

func (s *Server) handleEmailVerification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		return
	}

	var token string

	if r.Method == "GET" {
		// GET request with token as query parameter (for email links)
		token = r.URL.Query().Get("token")
	} else if r.Method == "POST" {
		// POST request with token in JSON body
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		token = req.Token
	}

	if token == "" {
		http.Error(w, `{"error": "Verification token required"}`, http.StatusBadRequest)
		return
	}

	// Find user by verification token
	store := s.db.CreateStore("users")
	query := database.NewQueryBuilder()
	query.Where("verificationToken", "=", token)

	userData, err := store.FindOne(r.Context(), query)
	if err != nil {
		http.Error(w, `{"error": "Invalid or expired verification token"}`, http.StatusBadRequest)
		return
	}

	// Check if token has expired
	if expiresStr, ok := userData["verificationExpires"].(string); ok {
		if expires, err := time.Parse(time.RFC3339, expiresStr); err == nil {
			if time.Now().After(expires) {
				http.Error(w, `{"error": "Verification token has expired"}`, http.StatusBadRequest)
				return
			}
		}
	}

	// Update user to verified and active
	userID := userData["id"].(string)
	updateQuery := database.NewQueryBuilder()
	updateQuery.Where("id", "=", userID)

	updateBuilder := database.NewUpdateBuilder().
		Set("isVerified", true).
		Set("active", true).
		Unset("verificationToken").
		Unset("verificationExpires").
		Set("updatedAt", time.Now().Format(time.RFC3339))

	_, err = store.UpdateOne(r.Context(), updateQuery, updateBuilder)
	if err != nil {
		http.Error(w, `{"error": "Failed to verify user"}`, http.StatusInternalServerError)
		return
	}

	// Return success response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Email verified successfully",
		"user": map[string]interface{}{
			"id":       userData["id"],
			"username": userData["username"],
			"email":    userData["email"],
			"verified": true,
		},
	})
}

func (s *Server) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error": "Email address required"}`, http.StatusBadRequest)
		return
	}

	// Find user by email
	store := s.db.CreateStore("users")
	query := database.NewQueryBuilder()
	query.Where("email", "=", req.Email)

	userData, err := store.FindOne(r.Context(), query)
	if err != nil {
		// Don't reveal if email exists for security
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "If the email exists and is unverified, a verification email has been sent",
		})
		return
	}

	// Check if already verified
	if verified, ok := userData["isVerified"].(bool); ok && verified {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email is already verified",
		})
		return
	}

	// Generate new verification token and update user
	verificationToken, err := email.GenerateVerificationToken()
	if err != nil {
		http.Error(w, `{"error": "Failed to generate verification token"}`, http.StatusInternalServerError)
		return
	}

	// Update user with new verification token
	updateQuery := database.NewQueryBuilder().Where("email", "=", req.Email)
	updateBuilder := database.NewUpdateBuilder()
	updateBuilder.Set("verificationToken", verificationToken)
	updateBuilder.Set("verificationExpires", time.Now().Add(24*time.Hour))

	_, err = store.Update(r.Context(), updateQuery, updateBuilder)
	if err != nil {
		http.Error(w, `{"error": "Failed to update verification token"}`, http.StatusInternalServerError)
		return
	}

	// Send verification email
	baseURL := fmt.Sprintf("http://%s", r.Host)
	emailService := email.NewEmailService(&s.securityConfig.Email)

	username := getStringFromMap(userData, "username")
	if err := emailService.SendVerificationEmail(req.Email, username, verificationToken, baseURL); err != nil {
		// Log the error but still return success to avoid revealing email existence
		// TODO: Add proper logging
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "If the email exists and is unverified, a verification email has been sent",
	})
}

func (s *Server) setupSwaggerRoutes() {
	// Create swagger generator
	baseURL := fmt.Sprintf("http://localhost:%d", s.config.Port)
	generator := swagger.NewGenerator(baseURL, s.router.GetResources())

	// Overall API documentation
	s.httpMux.HandleFunc("/api/docs/openapi.json", s.handleOverallSwagger(generator)).Methods("GET")

	// Collection-specific API documentation
	s.httpMux.HandleFunc("/api/docs/{collection}/openapi.json", s.handleCollectionSwagger(generator)).Methods("GET")

	// Swagger UI for overall API
	s.httpMux.PathPrefix("/api/docs/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/api/docs/openapi.json"),
		httpSwagger.DeepLinking(true),
	))

	// Collection-specific Swagger UI
	s.httpMux.HandleFunc("/api/docs/{collection}/", s.handleCollectionSwaggerUI()).Methods("GET")
}

func (s *Server) handleOverallSwagger(generator *swagger.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		spec, err := generator.GenerateSpec()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to generate API spec: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(spec)
	}
}

func (s *Server) handleCollectionSwagger(generator *swagger.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		collectionName := vars["collection"]

		// Enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Find the collection
		var targetCollection resources.Resource
		for _, collection := range s.router.GetResources() {
			if collection.GetName() == collectionName {
				targetCollection = collection
				break
			}
		}

		if targetCollection == nil {
			http.Error(w, fmt.Sprintf(`{"error": "Collection '%s' not found"}`, collectionName), http.StatusNotFound)
			return
		}

		spec, err := generator.GenerateCollectionSpec(targetCollection)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to generate API spec: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(spec)
	}
}

func (s *Server) handleCollectionSwaggerUI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		collectionName := vars["collection"]

		// Redirect to Swagger UI with collection-specific spec
		swaggerURL := fmt.Sprintf("/api/docs/%s/openapi.json", collectionName)

		// Serve custom Swagger UI HTML
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '%s',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        }
    </script>
</body>
</html>
`, strings.Title(collectionName), swaggerURL)

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}

// authenticateUser validates username/password and returns user data
func (s *Server) authenticateUser(username, password string) (map[string]interface{}, error) {
	store := s.db.CreateStore("users")

	// Find user by username or email
	var query database.QueryBuilder
	if strings.Contains(username, "@") {
		query = database.NewQueryBuilder().Where("email", "$eq", username)
	} else {
		query = database.NewQueryBuilder().Where("username", "$eq", username)
	}

	user, err := store.FindOne(context.Background(), query)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Verify password
	hashedPassword, ok := user["password"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user data")
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// getStringFromMap safely extracts a string value from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// startUserCleanupJob runs a background job to remove unverified users after 24 hours
func (s *Server) startUserCleanupJob() {
	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Also run immediately on startup
	s.cleanupUnverifiedUsers()

	for range ticker.C {
		s.cleanupUnverifiedUsers()
	}
}

// cleanupUnverifiedUsers removes users who haven't verified their email within 24 hours
func (s *Server) cleanupUnverifiedUsers() {
	if !s.securityConfig.RequireVerification {
		// Email verification not required, no cleanup needed
		return
	}

	store := s.db.CreateStore("users")

	// Find unverified users where verification token expired
	query := database.NewQueryBuilder()
	query.Where("isVerified", "=", false)
	query.Where("verificationExpires", "<", time.Now())

	// Find expired unverified users
	users, err := store.Find(context.Background(), query, database.QueryOptions{})
	if err != nil {
		log.Printf("Error finding unverified users for cleanup: %v", err)
		return
	}

	// Delete each expired unverified user
	deletedCount := 0
	for _, user := range users {
		if userID, ok := user["id"].(string); ok {
			deleteQuery := database.NewQueryBuilder().Where("id", "=", userID)
			result, err := store.Remove(context.Background(), deleteQuery)
			if err != nil {
				log.Printf("Error deleting unverified user %s: %v", userID, err)
				continue
			}
			if result.DeletedCount() > 0 {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("🧹 Cleaned up %d unverified users", deletedCount)
	}
}

// handleSelfTest serves the API self-test page
func (s *Server) handleSelfTest(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for the self-test page
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	// Serve the self-test.html file
	selfTestPath := filepath.Join("web", "self-test.html")
	http.ServeFile(w, r, selfTestPath)
}

// SetDashboardFS sets the embedded dashboard filesystem
func (s *Server) SetDashboardFS(fs embed.FS) {
	s.dashboardFS = &fs
	// Re-setup dashboard routes with embedded FS
	if s.dashboardFS != nil {
		s.setupDashboardRoutes(*s.dashboardFS)
	}
}
