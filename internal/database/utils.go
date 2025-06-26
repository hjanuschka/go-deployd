package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// generateUniqueID creates a unique identifier similar to MongoDB ObjectIDs
func generateUniqueID() string {
	// Create a 12-byte ID similar to MongoDB ObjectID
	// 4 bytes timestamp + 8 bytes random
	timestamp := uint32(time.Now().Unix())
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	id := make([]byte, 12)
	id[0] = byte(timestamp >> 24)
	id[1] = byte(timestamp >> 16)
	id[2] = byte(timestamp >> 8)
	id[3] = byte(timestamp)
	copy(id[4:], randomBytes)

	return hex.EncodeToString(id)
}

// ExtendedConfig provides additional configuration options for different database types
type ExtendedConfig struct {
	*Config
	Type     DatabaseType           `json:"type"`
	Options  map[string]interface{} `json:"options"`
	FilePath string                 `json:"file_path,omitempty"` // For SQLite
}

// NewExtendedConfig creates a new extended configuration with defaults
func NewExtendedConfig() *ExtendedConfig {
	return &ExtendedConfig{
		Config: &Config{
			Host: "localhost",
			Port: 27017,
			Name: "deployd",
		},
		Type:    DatabaseTypeMongoDB, // Default to MongoDB
		Options: make(map[string]interface{}),
	}
}

// Validate validates the configuration
func (c *ExtendedConfig) Validate() error {
	if c.Config == nil {
		return fmt.Errorf("base config is required")
	}

	if c.Config.Name == "" {
		return fmt.Errorf("database name is required")
	}

	switch c.Type {
	case DatabaseTypeMongoDB:
		if c.Config.Host == "" {
			c.Config.Host = "localhost"
		}
		if c.Config.Port == 0 {
			c.Config.Port = 27017
		}
	case DatabaseTypeSQLite:
		// For SQLite, we can use the Host field as the file path
		if c.Config.Host == "" {
			c.Config.Host = fmt.Sprintf("data/%s.db", c.Config.Name)
		}
	case DatabaseTypeMySQL:
		if c.Config.Host == "" {
			c.Config.Host = "localhost"
		}
		if c.Config.Port == 0 {
			c.Config.Port = 3306
		}
	case DatabaseTypePostgres:
		if c.Config.Host == "" {
			c.Config.Host = "localhost"
		}
		if c.Config.Port == 0 {
			c.Config.Port = 5432
		}
	default:
		return fmt.Errorf("unsupported database type: %s", c.Type)
	}

	return nil
}

// ToBasicConfig converts ExtendedConfig to basic Config for backwards compatibility
func (c *ExtendedConfig) ToBasicConfig() *Config {
	return c.Config
}
