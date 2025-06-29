package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// StorageConfig defines the configuration for file storage
type StorageConfig struct {
	// Type can be "local", "s3", or "minio"
	Type string `json:"type"`
	
	// Local storage configuration
	Local LocalStorageConfig `json:"local,omitempty"`
	
	// S3/MinIO configuration (S3-compatible)
	S3 S3StorageConfig `json:"s3,omitempty"`
	
	// Global settings
	MaxFileSize        int64  `json:"maxFileSize"`        // Max file size in bytes (default: 50MB)
	AllowedExtensions  []string `json:"allowedExtensions"` // Empty means all extensions allowed
	SignedURLExpiration int    `json:"signedUrlExpiration"` // Expiration in seconds (default: 3600)
}

// LocalStorageConfig for local file storage
type LocalStorageConfig struct {
	BasePath string `json:"basePath"` // Base directory for file storage
	URLPrefix string `json:"urlPrefix"` // URL prefix for serving files
}

// S3StorageConfig for S3 and MinIO storage
type S3StorageConfig struct {
	Endpoint        string `json:"endpoint"`        // S3/MinIO endpoint (empty for AWS S3)
	Region          string `json:"region"`          // AWS region
	Bucket          string `json:"bucket"`          // Bucket name
	AccessKeyID     string `json:"accessKeyId"`     // Access key
	SecretAccessKey string `json:"secretAccessKey"` // Secret key
	UseSSL          bool   `json:"useSSL"`          // Use SSL for MinIO (default: true)
	PathStyle       bool   `json:"pathStyle"`       // Use path-style URLs (for MinIO)
}

// DefaultStorageConfig returns the default storage configuration
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type: "local",
		Local: LocalStorageConfig{
			BasePath:  "uploads",
			URLPrefix: "/files",
		},
		MaxFileSize:         50 * 1024 * 1024, // 50MB
		AllowedExtensions:   []string{},       // All allowed by default
		SignedURLExpiration: 3600,              // 1 hour
	}
}

// LoadStorageConfig loads storage configuration from file
func LoadStorageConfig(configPath string) (*StorageConfig, error) {
	configFile := filepath.Join(configPath, ".deployd", "storage.json")
	
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Return default config
		return DefaultStorageConfig(), nil
	}
	
	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	
	var config StorageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	// Set defaults for missing values
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 50 * 1024 * 1024
	}
	if config.SignedURLExpiration == 0 {
		config.SignedURLExpiration = 3600
	}
	if config.Type == "" {
		config.Type = "local"
	}
	if config.Type == "local" && config.Local.BasePath == "" {
		config.Local.BasePath = "uploads"
	}
	if config.Type == "local" && config.Local.URLPrefix == "" {
		config.Local.URLPrefix = "/files"
	}
	
	// Load environment variables for sensitive data
	if config.Type == "s3" || config.Type == "minio" {
		if accessKey := os.Getenv("STORAGE_ACCESS_KEY"); accessKey != "" {
			config.S3.AccessKeyID = accessKey
		}
		if secretKey := os.Getenv("STORAGE_SECRET_KEY"); secretKey != "" {
			config.S3.SecretAccessKey = secretKey
		}
	}
	
	return &config, nil
}

// IsS3Compatible returns true if the storage type is S3 or MinIO
func (c *StorageConfig) IsS3Compatible() bool {
	return c.Type == "s3" || c.Type == "minio"
}

// IsLocal returns true if using local storage
func (c *StorageConfig) IsLocal() bool {
	return c.Type == "local"
}