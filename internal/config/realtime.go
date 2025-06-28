package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RealtimeConfig holds real-time WebSocket and messaging configuration
type RealtimeConfig struct {
	Enabled     bool          `json:"enabled"`     // Enable/disable WebSocket support
	MessageTTL  int           `json:"messageTTL"`  // Message time-to-live in seconds
	Broker      BrokerConfig  `json:"broker"`      // Message broker configuration for multi-server deployments
	Limits      LimitsConfig  `json:"limits"`      // Connection and rate limits
}

// BrokerConfig holds message broker configuration for scaling across multiple servers
type BrokerConfig struct {
	Type     string       `json:"type"`     // "memory", "redis", "rabbitmq", "nats"
	Enabled  bool         `json:"enabled"`  // Enable message broker (required for multi-server deployments)
	Redis    RedisConfig  `json:"redis"`    // Redis configuration
	RabbitMQ RabbitConfig `json:"rabbitmq"` // RabbitMQ configuration
	NATS     NATSConfig   `json:"nats"`     // NATS configuration
}

// RedisConfig holds Redis configuration for message brokering
type RedisConfig struct {
	Host     string `json:"host"`     // Redis host
	Port     int    `json:"port"`     // Redis port
	Password string `json:"password"` // Redis password (optional)
	Database int    `json:"database"` // Redis database number
	Prefix   string `json:"prefix"`   // Key prefix for deployd messages
}

// RabbitConfig holds RabbitMQ configuration
type RabbitConfig struct {
	Host     string `json:"host"`     // RabbitMQ host
	Port     int    `json:"port"`     // RabbitMQ port
	Username string `json:"username"` // RabbitMQ username
	Password string `json:"password"` // RabbitMQ password
	VHost    string `json:"vhost"`    // RabbitMQ virtual host
	Exchange string `json:"exchange"` // Exchange name for deployd messages
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	Host     string `json:"host"`     // NATS host
	Port     int    `json:"port"`     // NATS port
	Username string `json:"username"` // NATS username (optional)
	Password string `json:"password"` // NATS password (optional)
	Subject  string `json:"subject"`  // Subject prefix for deployd messages
}

// LimitsConfig holds connection and rate limiting configuration
type LimitsConfig struct {
	MaxConnections    int `json:"maxConnections"`    // Maximum WebSocket connections per server
	MaxRoomsPerClient int `json:"maxRoomsPerClient"` // Maximum rooms a client can join
	MessageRateLimit  int `json:"messageRateLimit"`  // Messages per second per client
	PingInterval      int `json:"pingInterval"`      // WebSocket ping interval in seconds
	PongTimeout       int `json:"pongTimeout"`       // WebSocket pong timeout in seconds
}

// DefaultRealtimeConfig returns the default real-time configuration
func DefaultRealtimeConfig() *RealtimeConfig {
	return &RealtimeConfig{
		Enabled:    true, // WebSocket enabled by default
		MessageTTL: 3600, // 1 hour message TTL
		Broker: BrokerConfig{
			Type:    "memory", // In-memory only by default (single server)
			Enabled: false,    // Broker disabled by default
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				Database: 0,
				Prefix:   "deployd:",
			},
			RabbitMQ: RabbitConfig{
				Host:     "localhost",
				Port:     5672,
				Username: "guest",
				Password: "guest",
				VHost:    "/",
				Exchange: "deployd",
			},
			NATS: NATSConfig{
				Host:     "localhost",
				Port:     4222,
				Username: "",
				Password: "",
				Subject:  "deployd",
			},
		},
		Limits: LimitsConfig{
			MaxConnections:    10000, // 10k connections per server
			MaxRoomsPerClient: 100,   // Max 100 rooms per client
			MessageRateLimit:  100,   // 100 messages per second per client
			PingInterval:      54,    // Ping every 54 seconds
			PongTimeout:       10,    // 10 second pong timeout
		},
	}
}

// LoadRealtimeConfig loads real-time configuration from file or creates default
func LoadRealtimeConfig(configDir string) (*RealtimeConfig, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "realtime.json")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config
		config := DefaultRealtimeConfig()

		if err := SaveRealtimeConfig(config, configDir); err != nil {
			return nil, fmt.Errorf("failed to save default realtime config: %w", err)
		}

		fmt.Printf("📡 Created default real-time configuration at %s\n", configFile)
		fmt.Printf("   WebSocket: %v | Broker: %s\n", config.Enabled, config.Broker.Type)

		return config, nil
	}

	// Load existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read realtime config: %w", err)
	}

	var config RealtimeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse realtime config: %w", err)
	}

	return &config, nil
}

// SaveRealtimeConfig saves real-time configuration to file
func SaveRealtimeConfig(config *RealtimeConfig, configDir string) error {
	configFile := filepath.Join(configDir, "realtime.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal realtime config: %w", err)
	}

	// Write with restricted permissions (644 = owner read/write, group/others read)
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write realtime config: %w", err)
	}

	return nil
}

// IsMultiServerMode returns true if message broker is enabled for multi-server deployments
func (rc *RealtimeConfig) IsMultiServerMode() bool {
	return rc.Broker.Enabled && rc.Broker.Type != "memory"
}

// GetBrokerConnectionString returns the connection string for the configured broker
func (rc *RealtimeConfig) GetBrokerConnectionString() string {
	switch rc.Broker.Type {
	case "redis":
		if rc.Broker.Redis.Password != "" {
			return fmt.Sprintf("redis://:%s@%s:%d/%d",
				rc.Broker.Redis.Password,
				rc.Broker.Redis.Host,
				rc.Broker.Redis.Port,
				rc.Broker.Redis.Database)
		}
		return fmt.Sprintf("redis://%s:%d/%d",
			rc.Broker.Redis.Host,
			rc.Broker.Redis.Port,
			rc.Broker.Redis.Database)

	case "rabbitmq":
		return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
			rc.Broker.RabbitMQ.Username,
			rc.Broker.RabbitMQ.Password,
			rc.Broker.RabbitMQ.Host,
			rc.Broker.RabbitMQ.Port,
			rc.Broker.RabbitMQ.VHost)

	case "nats":
		if rc.Broker.NATS.Username != "" {
			return fmt.Sprintf("nats://%s:%s@%s:%d",
				rc.Broker.NATS.Username,
				rc.Broker.NATS.Password,
				rc.Broker.NATS.Host,
				rc.Broker.NATS.Port)
		}
		return fmt.Sprintf("nats://%s:%d",
			rc.Broker.NATS.Host,
			rc.Broker.NATS.Port)

	default:
		return ""
	}
}