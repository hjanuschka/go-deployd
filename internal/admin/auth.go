package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles system-level authentication with master key
type AuthHandler struct {
	db         database.DatabaseInterface
	Security   *config.SecurityConfig
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db database.DatabaseInterface, security *config.SecurityConfig) *AuthHandler {
	// Parse JWT expiration duration
	jwtDuration, err := time.ParseDuration(security.JWTExpiration)
	if err != nil {
		jwtDuration = 24 * time.Hour // Default to 24 hours
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(security.JWTSecret, jwtDuration)

	return &AuthHandler{
		db:         db,
		Security:   security,
		jwtManager: jwtManager,
	}
}

// SystemLoginRequest represents a system login request using master key
type SystemLoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// SystemLoginResponse represents the response from system login
// DEPRECATED: Use JWT authentication endpoints instead
type SystemLoginResponse struct {
	Success   bool        `json:"success"`
	Token     string      `json:"token"`
	User      interface{} `json:"user"`
	Message   string      `json:"message"`
	ExpiresAt string      `json:"expiresAt"`
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

	var req struct {
		Email    string `json:"email"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}

	// Either email or username must be provided
	if req.Email == "" && req.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Either email or username must be provided",
		})
		return
	}

	// Find user by email or username
	store := ah.db.CreateStore("users")
	query := database.NewQueryBuilder()

	if req.Email != "" {
		query.Where("email", "=", req.Email)
	} else {
		query.Where("username", "=", req.Username)
	}

	userData, err := store.FindOne(context.Background(), query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "User not found",
		})
		return
	}

	// Extract user information
	userID, _ := userData["id"].(string)
	username, _ := userData["username"].(string)
	role, _ := userData["role"].(string)
	isRoot := (role == "admin")

	// Generate JWT token for the user
	token, err := ah.jwtManager.GenerateToken(userID, username, isRoot)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to generate token",
		})
		return
	}

	// Parse JWT expiration duration
	jwtDuration, err := time.ParseDuration(ah.Security.JWTExpiration)
	if err != nil {
		jwtDuration = 24 * time.Hour // Default to 24 hours
	}
	expiresAt := time.Now().Add(jwtDuration).Unix()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"token":     token,
		"expiresAt": expiresAt,
		"isRoot":    isRoot,
		"user": map[string]interface{}{
			"id":       userID,
			"username": username,
			"email":    userData["email"],
			"role":     role,
		},
	})
}

// HandleMasterKeyValidation validates a master key without performing authentication
func (ah *AuthHandler) HandleMasterKeyValidation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
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
			"valid":   false,
			"message": "Invalid JSON body",
		})
		return
	}

	valid := ah.Security.ValidateMasterKey(req.MasterKey)

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
	isAdmin := ah.Security.ValidateMasterKey(masterKey)

	response := map[string]interface{}{
		"jwtExpiration":     ah.Security.JWTExpiration,
		"allowRegistration": ah.Security.AllowRegistration,
	}

	// Only show master key info to authenticated admin
	if isAdmin {
		response["hasMasterKey"] = ah.Security.MasterKey != ""
		response["masterKeyPrefix"] = func() string {
			if len(ah.Security.MasterKey) > 10 {
				return ah.Security.MasterKey[:10] + "..."
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
	if !ah.Security.ValidateMasterKey(req.CurrentMasterKey) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid current master key",
		})
		return
	}

	// Generate new master key
	newMasterKey := generateNewMasterKey()
	ah.Security.MasterKey = newMasterKey

	// Save updated configuration
	if err := config.SaveSecurityConfig(ah.Security, config.GetConfigDir()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save new master key",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"message":      "Master key regenerated successfully",
		"newMasterKey": newMasterKey,
	})
}

// Middleware to require master key or JWT authentication
func (ah *AuthHandler) RequireMasterKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First check master key authentication
		masterKey := r.Header.Get("X-Master-Key")
		if masterKey == "" {
			// Also check cookie for dashboard requests
			if cookie, err := r.Cookie("masterKey"); err == nil {
				masterKey = cookie.Value
			}
		}

		if ah.Security.ValidateMasterKey(masterKey) {
			next(w, r)
			return
		}

		// Check JWT authentication with isRoot=true
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") && ah.jwtManager != nil {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := ah.jwtManager.ValidateToken(token); err == nil && claims.IsRoot {
				next(w, r)
				return
			}
		}

		// Neither master key nor valid JWT token with isRoot
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Authentication required",
			"message": "This endpoint requires a valid master key or root JWT token",
		})
		return
	}
}

// CreateUserRequest represents a request to create a user with master key
type CreateUserRequest struct {
	UserData map[string]interface{} `json:"userData"`
}

// HandleCreateUser creates a user with master key authentication
func (ah *AuthHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON body",
		})
		return
	}

	// Master key is validated by RequireMasterKey middleware
	// No need to validate it again here

	// Validate required user data
	email, hasEmail := req.UserData["email"].(string)
	username, hasUsername := req.UserData["username"].(string)
	password, hasPassword := req.UserData["password"].(string)

	if !hasEmail && !hasUsername {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Username or email is required",
		})
		return
	}

	if !hasPassword || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Password is required",
		})
		return
	}

	// Check if user already exists
	userStore := ah.db.CreateStore("users")

	if hasEmail {
		query := database.NewQueryBuilder().Where("email", "$eq", email)
		existing, err := userStore.FindOne(r.Context(), query)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Database error",
			})
			return
		}
		if existing != nil {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "User with this email already exists",
			})
			return
		}
	}

	if hasUsername {
		query := database.NewQueryBuilder().Where("username", "$eq", username)
		existing, err := userStore.FindOne(r.Context(), query)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Database error",
			})
			return
		}
		if existing != nil {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "User with this username already exists",
			})
			return
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to hash password",
		})
		return
	}

	// Set hashed password and default role
	req.UserData["password"] = string(hashedPassword)
	if _, hasRole := req.UserData["role"]; !hasRole {
		req.UserData["role"] = "user"
	}

	// Create user
	result, err := userStore.Insert(r.Context(), req.UserData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create user",
		})
		return
	}

	// Convert result to map and remove password from response
	var userResult map[string]interface{}
	if resultMap, ok := result.(map[string]interface{}); ok {
		userResult = resultMap
		delete(userResult, "password")
	} else {
		// Fallback - return the original user data without password
		userResult = make(map[string]interface{})
		for k, v := range req.UserData {
			if k != "password" {
				userResult[k] = v
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User created successfully",
		"user":    userResult,
	})
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
