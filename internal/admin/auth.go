package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/sessions"
)

// AuthHandler handles system-level authentication with master key
type AuthHandler struct {
	db       database.DatabaseInterface
	sessions *sessions.SessionStore
	security *config.SecurityConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db database.DatabaseInterface, sessions *sessions.SessionStore, security *config.SecurityConfig) *AuthHandler {
	return &AuthHandler{
		db:       db,
		sessions: sessions,
		security: security,
	}
}

// SystemLoginRequest represents a system login request using master key
type SystemLoginRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	MasterKey string `json:"masterKey"`
}

// SystemLoginResponse represents the response from system login
type SystemLoginResponse struct {
	Success     bool   `json:"success"`
	SessionID   string `json:"sessionId"`
	Token       string `json:"token"`
	User        interface{} `json:"user"`
	Message     string `json:"message"`
	ExpiresAt   string `json:"expiresAt"`
}

// HandleSystemLogin performs authentication using master key and user identifier
func (ah *AuthHandler) HandleSystemLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}
	
	var req SystemLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// Validate master key
	if !ah.security.ValidateMasterKey(req.MasterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid master key",
		})
		return
	}
	
	// Validate user identifier
	if req.Username == "" && req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Username or email is required",
		})
		return
	}
	
	// Find user in database
	userStore := ah.db.CreateStore("users")
	
	var query database.QueryBuilder
	if req.Email != "" {
		query = database.NewQueryBuilder().Where("email", "$eq", req.Email)
	} else {
		query = database.NewQueryBuilder().Where("username", "$eq", req.Username)
	}
	
	user, err := userStore.FindOne(r.Context(), query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Database error",
		})
		return
	}
	
	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "User not found",
		})
		return
	}
	
	// Create or get session
	session, err := ah.sessions.CreateSession("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create session",
		})
		return
	}
	
	// Set session data
	sessionData := map[string]interface{}{
		"userId":     user["id"],
		"username":   getStringField(user, "username"),
		"email":      getStringField(user, "email"),
		"role":       getStringField(user, "role"),
		"loginTime":  time.Now(),
		"loginType":  "master_key",
	}
	
	session.Set("user", sessionData)
	session.Set("isRoot", getStringField(user, "role") == "admin")
	
	// Save session
	if err := session.Save(ah.sessions); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save session",
		})
		return
	}
	
	// Set session cookie
	ah.sessions.SetSessionCookie(w, session)
	
	// Prepare user response (without password)
	userResponse := make(map[string]interface{})
	for k, v := range user {
		if k != "password" {
			userResponse[k] = v
		}
	}
	
	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(ah.security.SessionTTL) * time.Second)
	
	response := SystemLoginResponse{
		Success:   true,
		SessionID: session.GetID(),
		Token:     session.GetID(), // Use session ID as token for now
		User:      userResponse,
		Message:   "Authentication successful",
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleMasterKeyValidation validates a master key without performing authentication
func (ah *AuthHandler) HandleMasterKeyValidation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"message": "Method not allowed",
		})
		return
	}
	
	var req struct {
		MasterKey string `json:"masterKey"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	valid := ah.security.ValidateMasterKey(req.MasterKey)
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid,
		"message": func() string {
			if valid {
				return "Master key is valid"
			}
			return "Invalid master key"
		}(),
	})
}

// HandleGetSecurityInfo returns non-sensitive security configuration info
func (ah *AuthHandler) HandleGetSecurityInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}
	
	// Check if master key is provided for admin access
	masterKey := r.Header.Get("X-Master-Key")
	isAdmin := ah.security.ValidateMasterKey(masterKey)
	
	response := map[string]interface{}{
		"sessionTTL":  ah.security.SessionTTL,
		"tokenTTL":    ah.security.TokenTTL,
		"enableSSO":   ah.security.EnableSSO,
		"ssoEndpoint": ah.security.SSOEndpoint,
	}
	
	// Only show master key info to authenticated admin
	if isAdmin {
		response["hasMasterKey"] = ah.security.MasterKey != ""
		response["masterKeyPrefix"] = func() string {
			if len(ah.security.MasterKey) > 10 {
				return ah.security.MasterKey[:10] + "..."
			}
			return "***"
		}()
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleRegenerateMasterKey regenerates the master key (requires current master key)
func (ah *AuthHandler) HandleRegenerateMasterKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}
	
	var req struct {
		CurrentMasterKey string `json:"currentMasterKey"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}
	
	// Validate current master key
	if !ah.security.ValidateMasterKey(req.CurrentMasterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid current master key",
		})
		return
	}
	
	// Generate new master key
	newMasterKey := generateNewMasterKey()
	ah.security.MasterKey = newMasterKey
	
	// Save updated configuration
	if err := config.SaveSecurityConfig(ah.security, config.GetConfigDir()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save new master key",
		})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Master key regenerated successfully",
		"newMasterKey": newMasterKey,
	})
}

// Middleware to require master key authentication
func (ah *AuthHandler) RequireMasterKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masterKey := r.Header.Get("X-Master-Key")
		if masterKey == "" {
			// Also check Authorization header with Bearer format
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				masterKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		
		if !ah.security.ValidateMasterKey(masterKey) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Master key required",
				"message": "This endpoint requires a valid master key",
			})
			return
		}
		
		next(w, r)
	}
}

// Helper functions
func getStringField(data map[string]interface{}, field string) string {
	if val, ok := data[field].(string); ok {
		return val
	}
	return ""
}

func generateNewMasterKey() string {
	// Use the same generation logic as in config package
	return "mk_regenerated_" + strings.Replace(time.Now().Format("20060102150405"), " ", "", -1)
}