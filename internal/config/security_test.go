package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSecurityConfig(t *testing.T) {
	t.Run("Default configuration values", func(t *testing.T) {
		cfg := config.DefaultSecurityConfig()

		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.MasterKey)
		assert.True(t, cfg.AllowRegistration)
		assert.Empty(t, cfg.JWTSecret)
		assert.Equal(t, "24h", cfg.JWTExpiration)
		assert.True(t, cfg.RequireVerification)

		// Test email config defaults
		assert.Equal(t, "smtp", cfg.Email.Provider)
		assert.Equal(t, "smtp.gmail.com", cfg.Email.SMTP.Host)
		assert.Equal(t, 587, cfg.Email.SMTP.Port)
		assert.True(t, cfg.Email.SMTP.TLS)
		assert.Equal(t, "us-east-1", cfg.Email.SES.Region)
		assert.Equal(t, "noreply@example.com", cfg.Email.From)
		assert.Equal(t, "Go-Deployd", cfg.Email.FromName)
	})
}

func TestGetConfigDir(t *testing.T) {
	t.Run("Returns default config directory", func(t *testing.T) {
		dir := config.GetConfigDir()
		assert.Equal(t, ".deployd", dir)
	})
}

func TestValidateMasterKey(t *testing.T) {
	t.Run("Valid master key", func(t *testing.T) {
		cfg := &config.SecurityConfig{
			MasterKey: "test_master_key_123",
		}

		assert.True(t, cfg.ValidateMasterKey("test_master_key_123"))
	})

	t.Run("Invalid master key", func(t *testing.T) {
		cfg := &config.SecurityConfig{
			MasterKey: "test_master_key_123",
		}

		assert.False(t, cfg.ValidateMasterKey("wrong_key"))
		assert.False(t, cfg.ValidateMasterKey(""))
	})

	t.Run("Empty master key in config", func(t *testing.T) {
		cfg := &config.SecurityConfig{
			MasterKey: "",
		}

		assert.False(t, cfg.ValidateMasterKey("any_key"))
		assert.False(t, cfg.ValidateMasterKey(""))
	})
}

func TestSaveSecurityConfig(t *testing.T) {
	t.Run("Save config to file", func(t *testing.T) {
		tempDir := t.TempDir()
		
		cfg := config.DefaultSecurityConfig()
		cfg.MasterKey = "test_master_key"
		cfg.JWTSecret = "test_jwt_secret"

		err := config.SaveSecurityConfig(cfg, tempDir)
		require.NoError(t, err)

		// Verify file was created
		configFile := filepath.Join(tempDir, "security.json")
		assert.FileExists(t, configFile)

		// Verify file content
		data, err := os.ReadFile(configFile)
		require.NoError(t, err)

		var saved config.SecurityConfig
		err = json.Unmarshal(data, &saved)
		require.NoError(t, err)

		assert.Equal(t, "test_master_key", saved.MasterKey)
		assert.Equal(t, "test_jwt_secret", saved.JWTSecret)
		assert.Equal(t, "24h", saved.JWTExpiration)
		assert.True(t, saved.AllowRegistration)
	})

	t.Run("Save config with custom email settings", func(t *testing.T) {
		tempDir := t.TempDir()
		
		cfg := config.DefaultSecurityConfig()
		cfg.Email.Provider = "ses"
		cfg.Email.From = "custom@example.com"
		cfg.Email.FromName = "Custom App"
		cfg.Email.SES.Region = "us-west-2"
		cfg.Email.SES.AccessKeyID = "test_access_key"

		err := config.SaveSecurityConfig(cfg, tempDir)
		require.NoError(t, err)

		// Verify saved content
		configFile := filepath.Join(tempDir, "security.json")
		data, err := os.ReadFile(configFile)
		require.NoError(t, err)

		var saved config.SecurityConfig
		err = json.Unmarshal(data, &saved)
		require.NoError(t, err)

		assert.Equal(t, "ses", saved.Email.Provider)
		assert.Equal(t, "custom@example.com", saved.Email.From)
		assert.Equal(t, "Custom App", saved.Email.FromName)
		assert.Equal(t, "us-west-2", saved.Email.SES.Region)
		assert.Equal(t, "test_access_key", saved.Email.SES.AccessKeyID)
	})
}

func TestLoadSecurityConfig(t *testing.T) {
	t.Run("Load config from non-existent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, "new_config")

		cfg, err := config.LoadSecurityConfig(configDir)
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Should create default config with generated master key
		assert.NotEmpty(t, cfg.MasterKey)
		assert.True(t, cfg.AllowRegistration)
		assert.Equal(t, "24h", cfg.JWTExpiration)

		// Verify config file was created
		configFile := filepath.Join(configDir, "security.json")
		assert.FileExists(t, configFile)
	})

	t.Run("Load existing config file", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create existing config
		existingCfg := config.DefaultSecurityConfig()
		existingCfg.MasterKey = "existing_master_key"
		existingCfg.JWTSecret = "existing_jwt_secret"
		existingCfg.AllowRegistration = false
		existingCfg.JWTExpiration = "48h"

		err := config.SaveSecurityConfig(existingCfg, tempDir)
		require.NoError(t, err)

		// Load the config
		cfg, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)

		assert.Equal(t, "existing_master_key", cfg.MasterKey)
		assert.Equal(t, "existing_jwt_secret", cfg.JWTSecret)
		assert.False(t, cfg.AllowRegistration)
		assert.Equal(t, "48h", cfg.JWTExpiration)
	})

	t.Run("Load config with missing master key", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create config without master key
		existingCfg := config.DefaultSecurityConfig()
		existingCfg.JWTSecret = "existing_jwt_secret"
		// MasterKey is left empty

		err := config.SaveSecurityConfig(existingCfg, tempDir)
		require.NoError(t, err)

		// Load the config - should generate master key
		cfg, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)

		assert.NotEmpty(t, cfg.MasterKey)
		assert.Contains(t, cfg.MasterKey, "mk_")
		assert.Equal(t, "existing_jwt_secret", cfg.JWTSecret)
	})

	t.Run("Load config with missing JWT secret", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create config without JWT secret
		existingCfg := config.DefaultSecurityConfig()
		existingCfg.MasterKey = "existing_master_key"
		// JWTSecret is left empty

		err := config.SaveSecurityConfig(existingCfg, tempDir)
		require.NoError(t, err)

		// Load the config - should generate JWT secret
		cfg, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)

		assert.Equal(t, "existing_master_key", cfg.MasterKey)
		assert.NotEmpty(t, cfg.JWTSecret)
		assert.NotContains(t, cfg.JWTSecret, "mk_") // JWT secret doesn't have prefix
	})

	t.Run("Load config with missing JWT expiration", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create config without JWT expiration
		existingCfg := config.DefaultSecurityConfig()
		existingCfg.MasterKey = "existing_master_key"
		existingCfg.JWTSecret = "existing_jwt_secret"
		existingCfg.JWTExpiration = ""

		err := config.SaveSecurityConfig(existingCfg, tempDir)
		require.NoError(t, err)

		// Load the config - should set default expiration
		cfg, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)

		assert.Equal(t, "existing_master_key", cfg.MasterKey)
		assert.Equal(t, "existing_jwt_secret", cfg.JWTSecret)
		assert.Equal(t, "24h", cfg.JWTExpiration) // Should set default
	})

	t.Run("Load invalid JSON config", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create invalid JSON file
		configFile := filepath.Join(tempDir, "security.json")
		err := os.WriteFile(configFile, []byte("invalid json"), 0600)
		require.NoError(t, err)

		// Should fail to load
		cfg, err := config.LoadSecurityConfig(tempDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse security config")
	})
}

func TestEmailConfigStruct(t *testing.T) {
	t.Run("Complete email configuration", func(t *testing.T) {
		cfg := &config.EmailConfig{
			Provider: "smtp",
			SMTP: config.SMTPConfig{
				Host:     "mail.example.com",
				Port:     465,
				Username: "user@example.com",
				Password: "secret",
				TLS:      true,
			},
			SES: config.SESConfig{
				Region:          "eu-west-1",
				AccessKeyID:     "AKIATEST",
				SecretAccessKey: "secretkey",
			},
			From:     "noreply@example.com",
			FromName: "Test App",
		}

		assert.Equal(t, "smtp", cfg.Provider)
		assert.Equal(t, "mail.example.com", cfg.SMTP.Host)
		assert.Equal(t, 465, cfg.SMTP.Port)
		assert.Equal(t, "user@example.com", cfg.SMTP.Username)
		assert.Equal(t, "secret", cfg.SMTP.Password)
		assert.True(t, cfg.SMTP.TLS)
		assert.Equal(t, "eu-west-1", cfg.SES.Region)
		assert.Equal(t, "AKIATEST", cfg.SES.AccessKeyID)
		assert.Equal(t, "secretkey", cfg.SES.SecretAccessKey)
		assert.Equal(t, "noreply@example.com", cfg.From)
		assert.Equal(t, "Test App", cfg.FromName)
	})
}

func TestMasterKeyGeneration(t *testing.T) {
	t.Run("Master key format and uniqueness", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Generate first config
		cfg1, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)
		
		// Remove the config and generate again
		os.RemoveAll(tempDir)
		os.MkdirAll(tempDir, 0700)
		
		cfg2, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)

		// Master keys should be different
		assert.NotEqual(t, cfg1.MasterKey, cfg2.MasterKey)
		
		// Both should have proper format
		assert.Contains(t, cfg1.MasterKey, "mk_")
		assert.Contains(t, cfg2.MasterKey, "mk_")
		
		// Should be reasonable length (mk_ + 96 hex chars = 99 total)
		assert.Len(t, cfg1.MasterKey, 99)
		assert.Len(t, cfg2.MasterKey, 99)
	})
}

func TestJWTSecretGeneration(t *testing.T) {
	t.Run("JWT secret format and uniqueness", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()
		
		// Create configs without JWT secrets
		cfg1Base := config.DefaultSecurityConfig()
		cfg1Base.MasterKey = "test_key_1"
		err := config.SaveSecurityConfig(cfg1Base, tempDir1)
		require.NoError(t, err)
		
		cfg2Base := config.DefaultSecurityConfig()
		cfg2Base.MasterKey = "test_key_2"
		err = config.SaveSecurityConfig(cfg2Base, tempDir2)
		require.NoError(t, err)
		
		// Load configs - should generate JWT secrets
		cfg1, err := config.LoadSecurityConfig(tempDir1)
		require.NoError(t, err)
		
		cfg2, err := config.LoadSecurityConfig(tempDir2)
		require.NoError(t, err)

		// JWT secrets should be different
		assert.NotEqual(t, cfg1.JWTSecret, cfg2.JWTSecret)
		
		// Should be hex strings of proper length (64 hex chars)
		assert.Len(t, cfg1.JWTSecret, 64)
		assert.Len(t, cfg2.JWTSecret, 64)
		
		// Should not contain mk_ prefix
		assert.NotContains(t, cfg1.JWTSecret, "mk_")
		assert.NotContains(t, cfg2.JWTSecret, "mk_")
	})
}

func TestCompleteConfigurationFlow(t *testing.T) {
	t.Run("Full configuration lifecycle", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// 1. Load config for first time (should create default)
		cfg1, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)
		
		assert.NotEmpty(t, cfg1.MasterKey)
		assert.True(t, cfg1.AllowRegistration)
		assert.Equal(t, "24h", cfg1.JWTExpiration)
		
		// 2. Modify config
		cfg1.AllowRegistration = false
		cfg1.JWTExpiration = "12h"
		cfg1.Email.Provider = "ses"
		cfg1.Email.From = "custom@myapp.com"
		
		err = config.SaveSecurityConfig(cfg1, tempDir)
		require.NoError(t, err)
		
		// 3. Load config again (should preserve changes)
		cfg2, err := config.LoadSecurityConfig(tempDir)
		require.NoError(t, err)
		
		assert.Equal(t, cfg1.MasterKey, cfg2.MasterKey)
		assert.False(t, cfg2.AllowRegistration)
		assert.Equal(t, "12h", cfg2.JWTExpiration)
		assert.Equal(t, "ses", cfg2.Email.Provider)
		assert.Equal(t, "custom@myapp.com", cfg2.Email.From)
		
		// 4. Test master key validation
		assert.True(t, cfg2.ValidateMasterKey(cfg1.MasterKey))
		assert.False(t, cfg2.ValidateMasterKey("wrong_key"))
	})
}

func TestConfigPermissions(t *testing.T) {
	t.Run("Config file has secure permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		
		cfg := config.DefaultSecurityConfig()
		cfg.MasterKey = "test_key"
		
		err := config.SaveSecurityConfig(cfg, tempDir)
		require.NoError(t, err)
		
		// Check file permissions
		configFile := filepath.Join(tempDir, "security.json")
		stat, err := os.Stat(configFile)
		require.NoError(t, err)
		
		// Should be 0600 (owner read/write only)
		assert.Equal(t, os.FileMode(0600), stat.Mode().Perm())
	})
}