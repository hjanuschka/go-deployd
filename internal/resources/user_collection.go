package resources

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	emailpkg "github.com/hjanuschka/go-deployd/internal/email"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

// UserCollection extends Collection with authentication capabilities
type UserCollection struct {
	*Collection
	securityConfig *config.SecurityConfig
	emailService   *emailpkg.EmailService
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
	
	// Create email service
	emailService := emailpkg.NewEmailService(&securityConfig.Email)
	
	return &UserCollection{
		Collection:     collection,
		securityConfig: securityConfig,
		emailService:   emailService,
	}
}

// Handle extends the base collection handler with authentication endpoints
func (uc *UserCollection) Handle(ctx *appcontext.Context) error {
	logging.Info("🔥 USER COLLECTION HANDLE", "user-collection", map[string]interface{}{
		"method": ctx.Method,
		"path":   ctx.Request.URL.Path,
		"id":     ctx.GetID(),
	})
	
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
// DEPRECATED: This endpoint is deprecated. Use /auth/login with JWT tokens instead.
func (uc *UserCollection) handleLogin(ctx *appcontext.Context) error {
	ctx.Response.WriteHeader(410)
	return ctx.WriteJSON(map[string]interface{}{
		"error":   "Gone",
		"message": "This login endpoint is deprecated. Please use /auth/login with JWT authentication instead.",
		"redirect": "/auth/login",
	})
}

// handleLogout clears the user session
// DEPRECATED: This endpoint is deprecated. JWT tokens expire automatically.
func (uc *UserCollection) handleLogout(ctx *appcontext.Context) error {
	ctx.Response.WriteHeader(410)
	return ctx.WriteJSON(map[string]interface{}{
		"error":   "Gone",
		"message": "This logout endpoint is deprecated. JWT tokens expire automatically. Simply delete the client-side token.",
		"info":    "JWT tokens are stateless and expire automatically based on server configuration.",
	})
}

// handleMe returns the current user's information
// DEPRECATED: This endpoint is deprecated. Use /auth/me instead.
func (uc *UserCollection) handleMe(ctx *appcontext.Context) error {
	ctx.Response.WriteHeader(410)
	return ctx.WriteJSON(map[string]interface{}{
		"error":   "Gone", 
		"message": "This endpoint is deprecated. Please use /auth/me with JWT authentication instead.",
		"redirect": "/auth/me",
	})
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
	
	// Email is required for verification
	if !hasEmail || email == "" {
		return ctx.WriteError(400, "Email is required for user registration")
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
	
	// If email verification is required, set user as inactive and generate verification token
	if uc.securityConfig.RequireVerification && uc.emailService.IsConfigured() {
		// Generate verification token
		verificationToken, err := emailpkg.GenerateVerificationToken()
		if err != nil {
			return ctx.WriteError(500, "Failed to generate verification token")
		}
		
		// Set verification fields
		userData["active"] = false
		userData["isVerified"] = false
		userData["verificationToken"] = verificationToken
		userData["verificationExpires"] = time.Now().Add(24 * time.Hour)
		
		// Delegate to base collection for validation and creation
		ctx.Body = userData
		if err := uc.Collection.handlePost(ctx); err != nil {
			return err
		}
		
		// Send verification email (do this after successful user creation)
		baseURL := "http://localhost:" + ctx.Request.Host // Get base URL from request
		if err := uc.emailService.SendVerificationEmail(email, username, verificationToken, baseURL); err != nil {
			// Log the error but don't fail the registration
			// User can request resend later
			// TODO: Add proper logging
		}
		
		return nil
	} else {
		// No email verification required, set user as active
		userData["active"] = true
		userData["isVerified"] = true
		
		// Delegate to base collection for validation and creation
		ctx.Body = userData
		return uc.Collection.handlePost(ctx)
	}
}

// handleGenerateToken generates a static API token for a user
func (uc *UserCollection) handleGenerateToken(ctx *appcontext.Context) error {
	if !ctx.IsAuthenticated {
		return ctx.WriteError(401, "Not authenticated")
	}
	
	// DEPRECATED: This endpoint is deprecated. Use JWT-based authentication instead.
	ctx.Response.WriteHeader(410)
	return ctx.WriteJSON(map[string]interface{}{
		"error":   "Gone",
		"message": "This endpoint is deprecated. Use JWT authentication for API access.",
		"info":    "JWT tokens provide secure, stateless authentication without requiring server-side token management.",
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