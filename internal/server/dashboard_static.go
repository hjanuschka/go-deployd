package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// setupDashboardRoutes sets up dashboard routes with embedded fallback
func (s *Server) setupDashboardRoutes(dashboardFS ...embed.FS) {
	// Check if web/dashboard exists (development mode)
	dashboardPath := filepath.Join("web", "dashboard")
	if _, err := os.Stat(dashboardPath); err == nil {
		// Development mode - use filesystem
		s.httpMux.PathPrefix("/_dashboard/").HandlerFunc(s.serveDashboardWithAuth(dashboardPath))
	} else if len(dashboardFS) > 0 {
		// Production mode - use embedded filesystem
		s.httpMux.PathPrefix("/_dashboard/").HandlerFunc(s.serveDashboardEmbedded(dashboardFS[0]))
	} else {
		// Fallback - serve 404
		s.httpMux.PathPrefix("/_dashboard/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Dashboard not available", http.StatusNotFound)
		})
	}
}

// serveDashboardEmbedded serves dashboard from embedded filesystem
func (s *Server) serveDashboardEmbedded(dashboardFS embed.FS) http.HandlerFunc {
	// Get the subdirectory from the embedded filesystem
	dashboardSubFS, err := fs.Sub(dashboardFS, "web/dashboard")
	if err != nil {
		// Try without web prefix
		dashboardSubFS = dashboardFS
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path
		path := strings.TrimPrefix(r.URL.Path, "/_dashboard/")

		// Allow login page and assets without authentication
		if path == "login" || path == "login/" || strings.HasPrefix(path, "assets/") {
			s.serveDashboardEmbeddedFile(w, r, dashboardSubFS, path)
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

		s.serveDashboardEmbeddedFile(w, r, dashboardSubFS, path)
	}
}

// serveDashboardEmbeddedFile serves a specific dashboard file from embedded FS
func (s *Server) serveDashboardEmbeddedFile(w http.ResponseWriter, r *http.Request, dashboardFS fs.FS, path string) {
	if path == "" || path == "/" {
		// Serve index.html for dashboard root
		path = "index.html"
	}

	// Set appropriate content type
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	}

	// Read file content and serve it
	content, err := fs.ReadFile(dashboardFS, path)
	if err != nil {
		// Try index.html for SPA routing
		if !strings.HasPrefix(path, "assets/") {
			content, err = fs.ReadFile(dashboardFS, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			http.NotFound(w, r)
			return
		}
	}

	w.Write(content)
}