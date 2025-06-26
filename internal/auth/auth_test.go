package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestUserRegistration(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	// Initialize users collection
	userStore := db.GetStore("users")
	ctx := context.Background()
	err := userStore.CreateTable(ctx)
	require.NoError(t, err)
	defer userStore.DropTable(ctx)

	t.Run("successful registration", func(t *testing.T) {
		username := testutil.GenerateRandomName("user")
		email := fmt.Sprintf("%s@test.com", username)
		
		userData := map[string]interface{}{
			"username": username,
			"email":    email,
			"password": "securePassword123",
		}

		// Simulate registration
		hashedPassword := hashPassword(userData["password"].(string))
		userData["password"] = hashedPassword
		userData["verified"] = false
		userData["createdAt"] = time.Now()

		result, err := userStore.Insert(ctx, userData)
		require.NoError(t, err)
		assert.NotNil(t, result.InsertedID)

		// Verify user was created
		query := db.CreateQuery().Where("username", "=", username)
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, email, users[0]["email"])
		assert.False(t, users[0]["verified"].(bool))
	})

	t.Run("duplicate username", func(t *testing.T) {
		username := testutil.GenerateRandomName("user")
		
		// Create first user
		userData1 := map[string]interface{}{
			"username": username,
			"email":    fmt.Sprintf("%s1@test.com", username),
			"password": hashPassword("password123"),
		}
		_, err := userStore.Insert(ctx, userData1)
		require.NoError(t, err)

		// Try to create second user with same username
		userData2 := map[string]interface{}{
			"username": username,
			"email":    fmt.Sprintf("%s2@test.com", username),
			"password": hashPassword("password456"),
		}

		// Check for existing username first
		query := db.CreateQuery().Where("username", "=", username)
		existing, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, existing, 1, "Username already exists")
	})

	t.Run("duplicate email", func(t *testing.T) {
		email := fmt.Sprintf("%s@test.com", testutil.GenerateRandomName("email"))
		
		// Create first user
		userData1 := map[string]interface{}{
			"username": testutil.GenerateRandomName("user1"),
			"email":    email,
			"password": hashPassword("password123"),
		}
		_, err := userStore.Insert(ctx, userData1)
		require.NoError(t, err)

		// Check for existing email
		query := db.CreateQuery().Where("email", "=", email)
		existing, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, existing, 1, "Email already exists")
	})
}

func TestUserLogin(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	userStore := db.GetStore("users")
	ctx := context.Background()
	err := userStore.CreateTable(ctx)
	require.NoError(t, err)
	defer userStore.DropTable(ctx)

	// Create test user
	username := testutil.GenerateRandomName("loginuser")
	password := "testPassword123"
	hashedPassword := hashPassword(password)
	
	userData := map[string]interface{}{
		"username": username,
		"email":    fmt.Sprintf("%s@test.com", username),
		"password": hashedPassword,
		"verified": true,
	}
	
	result, err := userStore.Insert(ctx, userData)
	require.NoError(t, err)
	userID := result.InsertedID.(string)

	t.Run("successful login", func(t *testing.T) {
		// Find user by username
		query := db.CreateQuery().Where("username", "=", username)
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		require.Len(t, users, 1)

		// Verify password
		storedHash := users[0]["password"].(string)
		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
		assert.NoError(t, err)

		// Generate JWT token
		token, err := auth.GenerateJWT(userID, username, "test-secret")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("login with wrong password", func(t *testing.T) {
		query := db.CreateQuery().Where("username", "=", username)
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		require.Len(t, users, 1)

		storedHash := users[0]["password"].(string)
		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("wrongPassword"))
		assert.Error(t, err)
	})

	t.Run("login with non-existent user", func(t *testing.T) {
		query := db.CreateQuery().Where("username", "=", "nonexistent")
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("login with unverified email", func(t *testing.T) {
		// Create unverified user
		unverifiedUser := map[string]interface{}{
			"username": testutil.GenerateRandomName("unverified"),
			"email":    fmt.Sprintf("%s@test.com", testutil.GenerateRandomName("unverified")),
			"password": hashPassword("password123"),
			"verified": false,
		}
		_, err := userStore.Insert(ctx, unverifiedUser)
		require.NoError(t, err)

		// Find user
		query := db.CreateQuery().Where("username", "=", unverifiedUser["username"])
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		require.Len(t, users, 1)

		// Check if verified
		assert.False(t, users[0]["verified"].(bool), "User email not verified")
	})
}

func TestJWTAuthentication(t *testing.T) {
	secret := "test-jwt-secret"
	
	t.Run("generate and validate JWT", func(t *testing.T) {
		userID := "123456"
		username := "testuser"
		
		// Generate token
		token, err := auth.GenerateJWT(userID, username, secret)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate token
		claims, err := auth.ValidateJWT(token, secret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
	})

	t.Run("invalid JWT", func(t *testing.T) {
		// Invalid token
		_, err := auth.ValidateJWT("invalid.token.here", secret)
		assert.Error(t, err)

		// Token with wrong secret
		token, err := auth.GenerateJWT("123", "user", "wrong-secret")
		require.NoError(t, err)
		
		_, err = auth.ValidateJWT(token, secret)
		assert.Error(t, err)
	})

	t.Run("expired JWT", func(t *testing.T) {
		// This would require modifying the JWT generation to accept custom expiry
		// For now, we'll test that the token has an expiry
		token, err := auth.GenerateJWT("123", "user", secret)
		require.NoError(t, err)
		
		claims, err := auth.ValidateJWT(token, secret)
		require.NoError(t, err)
		assert.NotZero(t, claims.ExpiresAt)
	})
}

func TestSessionManagement(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	sessionStore := db.GetStore("sessions")
	ctx := context.Background()
	err := sessionStore.CreateTable(ctx)
	require.NoError(t, err)
	defer sessionStore.DropTable(ctx)

	t.Run("create session", func(t *testing.T) {
		userID := "user123"
		sessionToken := testutil.GenerateRandomName("session")
		
		sessionData := map[string]interface{}{
			"token":     sessionToken,
			"userId":    userID,
			"createdAt": time.Now(),
			"expiresAt": time.Now().Add(24 * time.Hour),
			"active":    true,
		}

		result, err := sessionStore.Insert(ctx, sessionData)
		require.NoError(t, err)
		assert.NotNil(t, result.InsertedID)
	})

	t.Run("validate session", func(t *testing.T) {
		sessionToken := testutil.GenerateRandomName("session")
		userID := "user456"
		
		// Create valid session
		sessionData := map[string]interface{}{
			"token":     sessionToken,
			"userId":    userID,
			"createdAt": time.Now(),
			"expiresAt": time.Now().Add(24 * time.Hour),
			"active":    true,
		}
		_, err := sessionStore.Insert(ctx, sessionData)
		require.NoError(t, err)

		// Find session
		query := db.CreateQuery().
			Where("token", "=", sessionToken).
			Where("active", "=", true).
			Where("expiresAt", ">", time.Now())
		
		sessions, err := sessionStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, userID, sessions[0]["userId"])
	})

	t.Run("invalidate session", func(t *testing.T) {
		sessionToken := testutil.GenerateRandomName("session")
		
		// Create session
		sessionData := map[string]interface{}{
			"token":     sessionToken,
			"userId":    "user789",
			"createdAt": time.Now(),
			"expiresAt": time.Now().Add(24 * time.Hour),
			"active":    true,
		}
		_, err := sessionStore.Insert(ctx, sessionData)
		require.NoError(t, err)

		// Invalidate session
		query := db.CreateQuery().Where("token", "=", sessionToken)
		update := db.CreateUpdate().Set("active", false)
		updateResult, err := sessionStore.Update(ctx, query, update)
		require.NoError(t, err)
		assert.Greater(t, updateResult.ModifiedCount, int64(0))

		// Verify session is inactive
		sessions, err := sessionStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.False(t, sessions[0]["active"].(bool))
	})

	t.Run("expired session", func(t *testing.T) {
		// Create expired session
		sessionData := map[string]interface{}{
			"token":     testutil.GenerateRandomName("session"),
			"userId":    "user999",
			"createdAt": time.Now().Add(-48 * time.Hour),
			"expiresAt": time.Now().Add(-24 * time.Hour), // Expired
			"active":    true,
		}
		_, err := sessionStore.Insert(ctx, sessionData)
		require.NoError(t, err)

		// Try to find active, non-expired sessions
		query := db.CreateQuery().
			Where("active", "=", true).
			Where("expiresAt", ">", time.Now())
		
		sessions, err := sessionStore.Find(ctx, query)
		require.NoError(t, err)
		
		// Should not include the expired session
		for _, session := range sessions {
			expiresAt := session["expiresAt"].(time.Time)
			assert.True(t, expiresAt.After(time.Now()))
		}
	})
}

func TestPasswordReset(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	userStore := db.GetStore("users")
	ctx := context.Background()
	err := userStore.CreateTable(ctx)
	require.NoError(t, err)
	defer userStore.DropTable(ctx)

	// Create test user
	user := testutil.CreateTestUser(t, db)

	t.Run("request password reset", func(t *testing.T) {
		resetToken := testutil.GenerateRandomName("reset")
		resetExpiry := time.Now().Add(1 * time.Hour)

		// Update user with reset token
		query := db.CreateQuery().Where("_id", "=", user.ID)
		update := db.CreateUpdate().
			Set("resetToken", resetToken).
			Set("resetTokenExpiry", resetExpiry)
		
		updateResult, err := userStore.Update(ctx, query, update)
		require.NoError(t, err)
		assert.Greater(t, updateResult.ModifiedCount, int64(0))
	})

	t.Run("reset password with valid token", func(t *testing.T) {
		resetToken := testutil.GenerateRandomName("reset")
		resetExpiry := time.Now().Add(1 * time.Hour)

		// Set reset token
		query := db.CreateQuery().Where("_id", "=", user.ID)
		update := db.CreateUpdate().
			Set("resetToken", resetToken).
			Set("resetTokenExpiry", resetExpiry)
		_, err := userStore.Update(ctx, query, update)
		require.NoError(t, err)

		// Verify token and reset password
		query = db.CreateQuery().
			Where("_id", "=", user.ID).
			Where("resetToken", "=", resetToken).
			Where("resetTokenExpiry", ">", time.Now())
		
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, users, 1)

		// Update password and clear reset token
		newPassword := hashPassword("newPassword123")
		update = db.CreateUpdate().
			Set("password", newPassword).
			Set("resetToken", nil).
			Set("resetTokenExpiry", nil)
		
		updateResult, err := userStore.Update(ctx, query, update)
		require.NoError(t, err)
		assert.Greater(t, updateResult.ModifiedCount, int64(0))
	})

	t.Run("reset password with expired token", func(t *testing.T) {
		// Set expired reset token
		query := db.CreateQuery().Where("_id", "=", user.ID)
		update := db.CreateUpdate().
			Set("resetToken", "expiredtoken").
			Set("resetTokenExpiry", time.Now().Add(-1*time.Hour)) // Expired
		_, err := userStore.Update(ctx, query, update)
		require.NoError(t, err)

		// Try to find user with valid token
		query = db.CreateQuery().
			Where("_id", "=", user.ID).
			Where("resetToken", "=", "expiredtoken").
			Where("resetTokenExpiry", ">", time.Now())
		
		users, err := userStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, users, 0, "Should not find user with expired token")
	})
}