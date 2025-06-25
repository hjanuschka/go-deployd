package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	MasterKey           string `json:"masterKey"`
	SessionTTL          int    `json:"sessionTTL"`          // in seconds
	TokenTTL            int    `json:"tokenTTL"`            // in seconds (deprecated, use JWTExpiration)
	AllowRegistration   bool   `json:"allowRegistration"`   // allow public user registration
	JWTSecret           string `json:"jwtSecret"`           // JWT signing secret
	JWTExpiration       string `json:"jwtExpiration"`       // JWT expiration duration (e.g., "24h", "1d")
}

// DefaultSecurityConfig returns the default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MasterKey:         "",
		SessionTTL:        86400,  // 24 hours
		TokenTTL:          2592000, // 30 days (deprecated)
		AllowRegistration: true,   // allow registration by default
		JWTSecret:         "",
		JWTExpiration:     "24h",  // 24 hours default
	}
}

// LoadSecurityConfig loads security configuration from file or creates default
func LoadSecurityConfig(configDir string) (*SecurityConfig, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	configFile := filepath.Join(configDir, "security.json")
	
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config with generated master key
		config := DefaultSecurityConfig()
		config.MasterKey = generateMasterKey()
		
		if err := SaveSecurityConfig(config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save default security config: %w", err)
		}
		
		fmt.Printf("🔐 Generated new master key and saved to %s\n", configFile)
		fmt.Printf("   Master Key: %s\n", config.MasterKey)
		fmt.Printf("   Keep this key secure! It provides administrative access.\n")
		
		return config, nil
	}
	
	// Load existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}
	
	var config SecurityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}
	
	// Generate master key if it's missing
	if config.MasterKey == "" {
		config.MasterKey = generateMasterKey()
		if err := SaveSecurityConfig(&config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save updated security config: %w", err)
		}
		fmt.Printf("🔐 Generated missing master key: %s\n", config.MasterKey)
	}
	
	// Generate JWT secret if it's missing
	if config.JWTSecret == "" {
		config.JWTSecret = generateJWTSecret()
		if err := SaveSecurityConfig(&config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save updated security config: %w", err)
		}
		fmt.Printf("🔑 Generated JWT secret\n")
	}
	
	// Set default JWT expiration if missing
	if config.JWTExpiration == "" {
		config.JWTExpiration = "24h"
		if err := SaveSecurityConfig(&config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save updated security config: %w", err)
		}
	}
	
	return &config, nil
}

// SaveSecurityConfig saves security configuration to file
func SaveSecurityConfig(config *SecurityConfig, configDir string) error {
	configFile := filepath.Join(configDir, "security.json")
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal security config: %w", err)
	}
	
	// Write with restricted permissions (600 = owner read/write only)
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write security config: %w", err)
	}
	
	return nil
}

// generateMasterKey generates a cryptographically secure master key
func generateMasterKey() string {
	// Generate 48 bytes (384 bits) of random data
	bytes := make([]byte, 48)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a deterministic but still reasonably secure method
		panic(fmt.Sprintf("failed to generate secure random key: %v", err))
	}
	
	return "mk_" + hex.EncodeToString(bytes)
}

// generateJWTSecret generates a cryptographically secure JWT secret
func generateJWTSecret() string {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate secure JWT secret: %v", err))
	}
	
	return hex.EncodeToString(bytes)
}

// ValidateMasterKey checks if the provided key matches the configured master key
func (sc *SecurityConfig) ValidateMasterKey(providedKey string) bool {
	return providedKey != "" && providedKey == sc.MasterKey
}

// GetConfigDir returns the default configuration directory
func GetConfigDir() string {
	// Use current directory + .deployd for configuration
	// In production, this could be /etc/deployd or ~/.deployd
	return ".deployd"
}