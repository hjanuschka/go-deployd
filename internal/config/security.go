package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hjanuschka/go-deployd/internal/logging"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	MasterKey           string      `json:"masterKey"`
	AllowRegistration   bool        `json:"allowRegistration"`   // allow public user registration
	JWTSecret           string      `json:"jwtSecret"`           // JWT signing secret
	JWTExpiration       string      `json:"jwtExpiration"`       // JWT expiration duration (e.g., "24h", "1d")
	RequireVerification bool        `json:"requireVerification"` // require email verification for new users
	Email               EmailConfig `json:"email"`               // email configuration for verification
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider string     `json:"provider"` // "smtp" or "ses"
	SMTP     SMTPConfig `json:"smtp"`     // SMTP configuration
	SES      SESConfig  `json:"ses"`      // AWS SES configuration
	From     string     `json:"from"`     // sender email address
	FromName string     `json:"fromName"` // sender display name
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string `json:"host"`     // SMTP server hostname
	Port     int    `json:"port"`     // SMTP server port
	Username string `json:"username"` // SMTP username
	Password string `json:"password"` // SMTP password
	TLS      bool   `json:"tls"`      // use TLS encryption
}

// SESConfig holds AWS SES configuration
type SESConfig struct {
	Region          string `json:"region"`          // AWS region
	AccessKeyID     string `json:"accessKeyId"`     // AWS access key ID
	SecretAccessKey string `json:"secretAccessKey"` // AWS secret access key
}

// DefaultSecurityConfig returns the default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MasterKey:           "",
		AllowRegistration:   true, // allow registration by default
		JWTSecret:           "",
		JWTExpiration:       "24h", // 24 hours default
		RequireVerification: true,  // require email verification by default
		Email: EmailConfig{
			Provider: "smtp", // SMTP is default
			SMTP: SMTPConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "",
				Password: "",
				TLS:      true,
			},
			SES: SESConfig{
				Region:          "us-east-1",
				AccessKeyID:     "",
				SecretAccessKey: "",
			},
			From:     "noreply@example.com",
			FromName: "Go-Deployd",
		},
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

		logging.GetLogger().Info("Generated new master key", logging.Fields{
			"config_file":       configFile,
			"master_key_length": len(config.MasterKey),
		})
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
		logging.GetLogger().Info("Generated missing master key", logging.Fields{
			"master_key_length": len(config.MasterKey),
		})
		fmt.Printf("🔐 Generated missing master key: [HIDDEN FOR SECURITY]\n")
	}

	// Generate JWT secret if it's missing
	if config.JWTSecret == "" {
		config.JWTSecret = generateJWTSecret()
		if err := SaveSecurityConfig(&config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save updated security config: %w", err)
		}
		logging.GetLogger().Info("Generated JWT secret", logging.Fields{
			"jwt_secret_length": len(config.JWTSecret),
		})
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
