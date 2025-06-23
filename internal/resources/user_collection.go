package resources

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	"golang.org/x/crypto/bcrypt"
)

// UserCollection extends Collection with authentication capabilities
type UserCollection struct {
	*Collection
	securityConfig *config.SecurityConfig
}

// UserSessionData represents the session data stored for authenticated users
type UserSessionData struct {
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	LoginTime time.Time `json:"loginTime"`
	Token     string    `json:"token"`
}

// NewUserCollection creates a new user collection with authentication capabilities
func NewUserCollection(name string, collectionConfig *CollectionConfig, db database.DatabaseInterface) *UserCollection {
	collection := NewCollection(name, collectionConfig, db)
	
	// Load security configuration
	securityConfig, err := config.LoadSecurityConfig(config.GetConfigDir())
	if err != nil {
		// Use default config if loading fails
		securityConfig = config.DefaultSecurityConfig()
	}
	
	return &UserCollection{
		Collection:     collection,
		securityConfig: securityConfig,
	}
}

// Handle extends the base collection handler with authentication endpoints
func (uc *UserCollection) Handle(ctx *appcontext.Context) error {
	// Handle special authentication endpoints
	if ctx.Method == "POST" {
		switch ctx.GetID() {
		case "login":
			return uc.handleLogin(ctx)
		case "logout":
			return uc.handleLogout(ctx)
		case "generate-token":
			return uc.handleGenerateToken(ctx)
		}
	}
	
	// Handle /me endpoint for current user
	if ctx.Method == "GET" && ctx.GetID() == "me" {
		return uc.handleMe(ctx)
	}
	
	// Handle user registration (POST to collection without ID)
	if ctx.Method == "POST" && ctx.GetID() == "" {
		return uc.handleRegister(ctx)
	}
	
	// For all other requests, delegate to base collection
	return uc.Collection.Handle(ctx)
}

// handleLogin authenticates a user and creates a session
func (uc *UserCollection) handleLogin(ctx *appcontext.Context) error {
	// Get login credentials from context body
	body := ctx.Body
	if body == nil || len(body) == 0 {
		return ctx.WriteError(400, "Request body is required")
	}
	
	password, ok := body["password"].(string)
	if !ok || password == "" {
		return ctx.WriteError(400, "Password is required")
	}
	
	username, hasUsername := body["username"].(string)
	email, hasEmail := body["email"].(string)
	
	if !hasUsername && !hasEmail {
		return ctx.WriteError(400, "Username or email is required")
	}
	
	// Find user by username or email
	var query database.QueryBuilder
	if hasEmail && email != "" {
		query = database.NewQueryBuilder().Where("email", "$eq", email)
	} else if hasUsername && username != "" {
		query = database.NewQueryBuilder().Where("username", "$eq", username)
	} else {
		return ctx.WriteError(400, "Username or email is required")
	}
	
	user, err := uc.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, "Database error")
	}
	
	if user == nil {
		return ctx.WriteError(401, "Invalid credentials")
	}
	
	// Verify password
	hashedPassword, ok := user["password"].(string)
	if !ok {
		return ctx.WriteError(500, "Invalid user data")
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return ctx.WriteError(401, "Invalid credentials")
	}
	
	// Generate session token
	token := uc.generateToken()
	
	// Store session data
	sessionData := UserSessionData{
		UserID:    getStringField(user, "id"),
		Username:  getStringField(user, "username"),
		Email:     getStringField(user, "email"),
		Role:      getStringField(user, "role"),
		LoginTime: time.Now(),
		Token:     token,
	}
	
	// Set session data
	ctx.Session.Set("user", sessionData)
	ctx.Session.Set("isRoot", sessionData.Role == "admin")
	
	// Save session
	if err := ctx.Session.Save(ctx.SessionStore); err != nil {
		return ctx.WriteError(500, "Failed to save session")
	}
	
	// Return user data (without password) and token
	userResponse := make(map[string]interface{})
	for k, v := range user {
		if k != "password" {
			userResponse[k] = v
		}
	}
	userResponse["token"] = token
	
	return ctx.WriteJSON(userResponse)
}

// handleLogout clears the user session
func (uc *UserCollection) handleLogout(ctx *appcontext.Context) error {
	// Clear session data
	ctx.Session.Set("user", nil)
	ctx.Session.Set("isRoot", false)
	
	// Save session
	if err := ctx.Session.Save(ctx.SessionStore); err != nil {
		return ctx.WriteError(500, "Failed to save session")
	}
	
	return ctx.WriteJSON(map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// handleMe returns the current user's information
func (uc *UserCollection) handleMe(ctx *appcontext.Context) error {
	userData := ctx.Session.Get("user")
	if userData == nil {
		return ctx.WriteError(401, "Not authenticated")
	}
	
	// Handle both UserSessionData struct and map[string]interface{} formats
	var userID string
	switch data := userData.(type) {
	case UserSessionData:
		userID = data.UserID
	case map[string]interface{}:
		// Try multiple possible field names for compatibility
		if id, ok := data["userId"].(string); ok {
			userID = id
		} else if id, ok := data["UserID"].(string); ok {
			userID = id
		} else if id, ok := data["userid"].(string); ok {
			// MongoDB may convert field names to lowercase
			userID = id
		} else {
			// Check if there's a nested user object (from admin login)
			if userObj, ok := data["user"].(map[string]interface{}); ok {
				if id, ok := userObj["userId"].(string); ok {
					userID = id
				} else if id, ok := userObj["userid"].(string); ok {
					userID = id
				}
			}
			
			if userID == "" {
				return ctx.WriteError(500, "Invalid session data: missing userId")
			}
		}
	default:
		return ctx.WriteError(500, "Invalid session data type")
	}
	
	if userID == "" {
		return ctx.WriteError(500, "Invalid session data: empty userId")
	}
	
	// Get fresh user data from database
	query := database.NewQueryBuilder().Where("id", "$eq", userID)
	user, err := uc.store.FindOne(ctx.Context(), query)
	if err != nil {
		return ctx.WriteError(500, "Database error")
	}
	
	if user == nil {
		return ctx.WriteError(404, "User not found")
	}
	
	// Return user data (without password)
	userResponse := make(map[string]interface{})
	for k, v := range user {
		if k != "password" {
			userResponse[k] = v
		}
	}
	
	return ctx.WriteJSON(userResponse)
}

// handleRegister creates a new user account
func (uc *UserCollection) handleRegister(ctx *appcontext.Context) error {
	// Check if registration is allowed
	if !uc.securityConfig.AllowRegistration {
		return ctx.WriteError(403, "User registration is disabled. Please contact administrator.")
	}
	
	// Get registration data from context body (already parsed)
	userData := ctx.Body
	if userData == nil || len(userData) == 0 {
		return ctx.WriteError(400, "Request body is required")
	}
	
	// Validate required fields
	password, ok := userData["password"].(string)
	if !ok || password == "" {
		return ctx.WriteError(400, "Password is required")
	}
	
	email, hasEmail := userData["email"].(string)
	username, hasUsername := userData["username"].(string)
	
	if !hasEmail && !hasUsername {
		return ctx.WriteError(400, "Username or email is required")
	}
	
	// Check if user already exists
	if hasEmail {
		query := database.NewQueryBuilder().Where("email", "$eq", email)
		existing, err := uc.store.FindOne(ctx.Context(), query)
		if err != nil {
			return ctx.WriteError(500, "Database error")
		}
		if existing != nil {
			return ctx.WriteError(409, "User with this email already exists")
		}
	}
	
	if hasUsername {
		query := database.NewQueryBuilder().Where("username", "$eq", username)
		existing, err := uc.store.FindOne(ctx.Context(), query)
		if err != nil {
			return ctx.WriteError(500, "Database error")
		}
		if existing != nil {
			return ctx.WriteError(409, "User with this username already exists")
		}
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return ctx.WriteError(500, "Failed to hash password")
	}
	
	userData["password"] = string(hashedPassword)
	
	// Set default role if not provided
	if _, hasRole := userData["role"]; !hasRole {
		userData["role"] = "user"
	}
	
	// Delegate to base collection for validation and creation
	ctx.Body = userData
	return uc.Collection.handlePost(ctx)
}

// handleGenerateToken generates a static API token for a user
func (uc *UserCollection) handleGenerateToken(ctx *appcontext.Context) error {
	userData := ctx.Session.Get("user")
	if userData == nil {
		return ctx.WriteError(401, "Not authenticated")
	}
	
	sessionData, ok := userData.(UserSessionData)
	if !ok {
		return ctx.WriteError(500, "Invalid session data")
	}
	
	// Generate new static token
	staticToken := uc.generateStaticToken()
	
	// Update user with new token
	query := database.NewQueryBuilder().Where("id", "$eq", sessionData.UserID)
	update := database.NewUpdateBuilder().Set("apiToken", staticToken)
	
	_, err := uc.store.Update(ctx.Context(), query, update)
	if err != nil {
		return ctx.WriteError(500, "Failed to update user")
	}
	
	return ctx.WriteJSON(map[string]interface{}{
		"token": staticToken,
		"message": "Static API token generated successfully",
	})
}

// generateToken generates a session token
func (uc *UserCollection) generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateStaticToken generates a longer-lived API token
func (uc *UserCollection) generateStaticToken() string {
	bytes := make([]byte, 48)
	rand.Read(bytes)
	return "tk_" + hex.EncodeToString(bytes)
}

// getStringField safely gets a string field from a map
func getStringField(data map[string]interface{}, field string) string {
	if val, ok := data[field].(string); ok {
		return val
	}
	return ""
}

// AuthenticateToken validates a static API token and returns user data
func (uc *UserCollection) AuthenticateToken(token string) (map[string]interface{}, error) {
	if token == "" {
		return nil, errors.New("token is required")
	}
	
	query := database.NewQueryBuilder().Where("apiToken", "$eq", token)
	user, err := uc.store.FindOne(nil, query)
	if err != nil {
		return nil, err
	}
	
	if user == nil {
		return nil, errors.New("invalid token")
	}
	
	return user, nil
}