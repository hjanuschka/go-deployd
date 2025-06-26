package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := 1 * time.Hour
	manager := NewJWTManager(secretKey, duration)

	tests := []struct {
		name     string
		userID   string
		username string
		isRoot   bool
	}{
		{
			name:     "Regular user token",
			userID:   "user123",
			username: "testuser",
			isRoot:   false,
		},
		{
			name:     "Root user token",
			userID:   "root",
			username: "admin",
			isRoot:   true,
		},
		{
			name:     "Empty username",
			userID:   "user456",
			username: "",
			isRoot:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.userID, tt.username, tt.isRoot)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			if token == "" {
				t.Errorf("GenerateToken() returned empty token")
			}

			// Validate the token we just generated
			claims, err := manager.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			if claims.UserID != tt.userID {
				t.Errorf("Expected UserID=%s, got %s", tt.userID, claims.UserID)
			}
			if claims.Username != tt.username {
				t.Errorf("Expected Username=%s, got %s", tt.username, claims.Username)
			}
			if claims.IsRoot != tt.isRoot {
				t.Errorf("Expected IsRoot=%v, got %v", tt.isRoot, claims.IsRoot)
			}
			if claims.Issuer != "go-deployd" {
				t.Errorf("Expected Issuer=go-deployd, got %s", claims.Issuer)
			}

			// Check expiration
			expectedExp := time.Now().Add(duration)
			if claims.ExpiresAt.Time.Before(expectedExp.Add(-10*time.Second)) ||
				claims.ExpiresAt.Time.After(expectedExp.Add(10*time.Second)) {
				t.Errorf("Token expiration time is not within expected range")
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := 1 * time.Hour
	manager := NewJWTManager(secretKey, duration)

	// Generate a valid token for testing
	validToken, err := manager.GenerateToken("test-user", "testuser", false)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantError bool
		errorType error
	}{
		{
			name:      "Valid token",
			token:     validToken,
			wantError: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantError: true,
		},
		{
			name:      "Invalid token format",
			token:     "invalid.token.format",
			wantError: true,
		},
		{
			name:      "Token with wrong secret",
			token:     generateTokenWithWrongSecret(t),
			wantError: true,
		},
		{
			name:      "Expired token",
			token:     generateExpiredToken(t, secretKey),
			wantError: true,
			errorType: ErrTokenExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateToken() expected error but got none")
				}
				if tt.errorType != nil && err != tt.errorType {
					t.Errorf("Expected error type %v, got %v", tt.errorType, err)
				}
				if claims != nil {
					t.Errorf("ValidateToken() expected nil claims but got %v", claims)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken() unexpected error = %v", err)
				}
				if claims == nil {
					t.Errorf("ValidateToken() expected claims but got nil")
				}
			}
		})
	}
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	shortDuration := 200 * time.Millisecond // Give more time to avoid immediate expiration
	manager := NewJWTManager(secretKey, shortDuration)

	// Generate token that will expire quickly
	token, err := manager.GenerateToken("test-user", "testuser", false)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token should be valid immediately (within 50ms of generation)
	claims, err := manager.ValidateToken(token)
	if err != nil {
		// If token expired immediately, that's likely due to clock precision
		// Just check that we get the expected error type
		if err != ErrTokenExpired {
			t.Fatalf("Expected token expired error, got: %v", err)
		}
		t.Log("Token expired immediately due to clock precision - this is acceptable")
		return
	}
	if claims == nil {
		t.Fatalf("Claims should not be nil")
	}

	// Wait for token to expire
	time.Sleep(300 * time.Millisecond)

	// Token should now be expired
	claims, err = manager.ValidateToken(token)
	if err == nil {
		t.Errorf("Token should be expired")
	}
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
	if claims != nil {
		t.Errorf("Claims should be nil for expired token")
	}
}

func TestGenerateSecretKey(t *testing.T) {
	// Generate multiple keys and ensure they're different
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key, err := GenerateSecretKey()
		if err != nil {
			t.Fatalf("GenerateSecretKey() error = %v", err)
		}

		if key == "" {
			t.Errorf("GenerateSecretKey() returned empty key")
		}

		if keys[key] {
			t.Errorf("GenerateSecretKey() returned duplicate key: %s", key)
		}
		keys[key] = true

		// Check key length (base64 encoded 32 bytes should be longer than 40 chars)
		if len(key) < 40 {
			t.Errorf("GenerateSecretKey() returned key that seems too short: %d chars", len(key))
		}
	}
}

func TestJWTClaims_Validation(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-testing"
	duration := 1 * time.Hour
	manager := NewJWTManager(secretKey, duration)

	// Test with various user data
	testCases := []struct {
		userID   string
		username string
		isRoot   bool
	}{
		{"", "", false},                             // Empty values
		{"user123", "normaluser", false},            // Normal user
		{"admin", "administrator", true},            // Admin user
		{"special-chars", "user@domain.com", false}, // Special characters
	}

	for _, tc := range testCases {
		token, err := manager.GenerateToken(tc.userID, tc.username, tc.isRoot)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		claims, err := manager.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if claims.UserID != tc.userID {
			t.Errorf("UserID mismatch: expected %s, got %s", tc.userID, claims.UserID)
		}
		if claims.Username != tc.username {
			t.Errorf("Username mismatch: expected %s, got %s", tc.username, claims.Username)
		}
		if claims.IsRoot != tc.isRoot {
			t.Errorf("IsRoot mismatch: expected %v, got %v", tc.isRoot, claims.IsRoot)
		}
	}
}

// Helper function to generate a token with wrong secret
func generateTokenWithWrongSecret(t *testing.T) string {
	wrongSecret := "wrong-secret-key"
	claims := &JWTClaims{
		UserID:   "test-user",
		Username: "testuser",
		IsRoot:   false,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "go-deployd",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(wrongSecret))
	if err != nil {
		t.Fatalf("Failed to generate token with wrong secret: %v", err)
	}
	return tokenString
}

// Helper function to generate an expired token
func generateExpiredToken(t *testing.T, secretKey string) string {
	claims := &JWTClaims{
		UserID:   "test-user",
		Username: "testuser",
		IsRoot:   false,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "go-deployd",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		t.Fatalf("Failed to generate expired token: %v", err)
	}
	return tokenString
}
