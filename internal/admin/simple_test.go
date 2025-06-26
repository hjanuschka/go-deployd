package admin

import (
	"testing"
	"time"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestConfigStruct(t *testing.T) {
	t.Run("Config struct initialization", func(t *testing.T) {
		cfg := &Config{
			Port:         8080,
			DatabaseHost: "localhost",
			DatabasePort: 5432,
			DatabaseName: "test_db",
			Development:  true,
		}

		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "localhost", cfg.DatabaseHost)
		assert.Equal(t, 5432, cfg.DatabasePort)
		assert.Equal(t, "test_db", cfg.DatabaseName)
		assert.True(t, cfg.Development)
	})
}

func TestServerInfoStruct(t *testing.T) {
	t.Run("ServerInfo struct initialization", func(t *testing.T) {
		startTime := time.Now()
		info := ServerInfo{
			Version:     "1.0.0",
			GoVersion:   "go1.21",
			Uptime:      "2h 30m",
			Database:    "PostgreSQL - Connected",
			Environment: "production",
			StartTime:   startTime,
		}

		assert.Equal(t, "1.0.0", info.Version)
		assert.Equal(t, "go1.21", info.GoVersion)
		assert.Equal(t, "2h 30m", info.Uptime)
		assert.Equal(t, "PostgreSQL - Connected", info.Database)
		assert.Equal(t, "production", info.Environment)
		assert.Equal(t, startTime, info.StartTime)
	})
}

func TestCollectionInfoStruct(t *testing.T) {
	t.Run("CollectionInfo struct initialization", func(t *testing.T) {
		lastModified := time.Now()
		properties := map[string]interface{}{
			"name": map[string]interface{}{
				"type":     "string",
				"required": true,
			},
		}

		info := CollectionInfo{
			Name:          "users",
			DocumentCount: 100,
			Properties:    properties,
			LastModified:  lastModified,
		}

		assert.Equal(t, "users", info.Name)
		assert.Equal(t, int64(100), info.DocumentCount)
		assert.Equal(t, properties, info.Properties)
		assert.Equal(t, lastModified, info.LastModified)
	})
}

func TestEmailTemplateStruct(t *testing.T) {
	t.Run("EmailTemplate struct initialization", func(t *testing.T) {
		template := EmailTemplate{
			Name:      "test",
			Subject:   "Test Subject",
			HTMLBody:  "<p>Test HTML</p>",
			TextBody:  "Test Text",
			Variables: []string{"Username", "Email"},
		}

		assert.Equal(t, "test", template.Name)
		assert.Equal(t, "Test Subject", template.Subject)
		assert.Equal(t, "<p>Test HTML</p>", template.HTMLBody)
		assert.Equal(t, "Test Text", template.TextBody)
		assert.Equal(t, []string{"Username", "Email"}, template.Variables)
	})
}

func TestGetString(t *testing.T) {
	t.Run("getString extracts string from map", func(t *testing.T) {
		m := map[string]interface{}{
			"name":   "John",
			"age":    30,
			"active": true,
		}

		assert.Equal(t, "John", getString(m, "name"))
		assert.Equal(t, "", getString(m, "age"))     // Not a string
		assert.Equal(t, "", getString(m, "missing")) // Missing key
	})

	t.Run("getString handles nil and empty maps", func(t *testing.T) {
		var nilMap map[string]interface{}
		emptyMap := make(map[string]interface{})

		assert.Equal(t, "", getString(nilMap, "key"))
		assert.Equal(t, "", getString(emptyMap, "key"))
	})
}

func TestBuildPropertiesMap(t *testing.T) {
	t.Run("buildPropertiesMap converts collection properties", func(t *testing.T) {
		// Create a mock admin handler with security config
		securityConfig := config.DefaultSecurityConfig()
		handler := &AdminHandler{
			AuthHandler: &AuthHandler{
				Security: securityConfig,
			},
		}

		configProps := map[string]resources.Property{
			"name": {
				Type:     "string",
				Required: true,
				Order:    1,
			},
			"age": {
				Type:    "number",
				Default: 0,
			},
			"email": {
				Type:   "string",
				Unique: true,
			},
			"createdAt": {
				Type:   "string",
				System: true,
			},
		}

		result := handler.buildPropertiesMap(configProps)

		// Check name property
		nameProps := result["name"].(map[string]interface{})
		assert.Equal(t, "string", nameProps["type"])
		assert.Equal(t, true, nameProps["required"])
		assert.Equal(t, 1, nameProps["order"])

		// Check age property
		ageProps := result["age"].(map[string]interface{})
		assert.Equal(t, "number", ageProps["type"])
		assert.Equal(t, 0, ageProps["default"])

		// Check email property
		emailProps := result["email"].(map[string]interface{})
		assert.Equal(t, "string", emailProps["type"])
		assert.Equal(t, true, emailProps["unique"])

		// Check system property
		createdProps := result["createdAt"].(map[string]interface{})
		assert.Equal(t, "string", createdProps["type"])
		assert.Equal(t, true, createdProps["system"])
		assert.Equal(t, true, createdProps["readonly"])
	})

	t.Run("buildPropertiesMap handles empty properties", func(t *testing.T) {
		securityConfig := config.DefaultSecurityConfig()
		handler := &AdminHandler{
			AuthHandler: &AuthHandler{
				Security: securityConfig,
			},
		}

		result := handler.buildPropertiesMap(map[string]resources.Property{})
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})
}
