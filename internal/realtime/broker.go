package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// MessageBroker defines the interface for message brokers
type MessageBroker interface {
	Connect(ctx context.Context) error
	Disconnect() error
	Publish(topic string, message *BrokerMessage) error
	Subscribe(topic string, handler MessageHandler) error
	Unsubscribe(topic string) error
	IsConnected() bool
}

// MessageHandler is a function that handles incoming messages from the broker
type MessageHandler func(message *BrokerMessage) error

// BrokerMessage represents a message sent through the broker
type BrokerMessage struct {
	Type      string                 `json:"type"`      // Message type
	Event     string                 `json:"event"`     // Event name
	Data      interface{}            `json:"data"`      // Message data
	Room      string                 `json:"room"`      // Target room
	ServerID  string                 `json:"server_id"` // Originating server ID
	Timestamp int64                  `json:"timestamp"` // Unix timestamp
	Meta      map[string]interface{} `json:"meta"`      // Additional metadata
}

// MemoryBroker implements an in-memory message broker (single server only)
type MemoryBroker struct {
	handlers map[string][]MessageHandler
	mu       sync.RWMutex
}

// NewMemoryBroker creates a new in-memory message broker
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		handlers: make(map[string][]MessageHandler),
	}
}

func (mb *MemoryBroker) Connect(ctx context.Context) error {
	logging.Info("Memory broker connected", "realtime", nil)
	return nil
}

func (mb *MemoryBroker) Disconnect() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers = make(map[string][]MessageHandler)
	logging.Info("Memory broker disconnected", "realtime", nil)
	return nil
}

func (mb *MemoryBroker) Publish(topic string, message *BrokerMessage) error {
	mb.mu.RLock()
	handlers, exists := mb.handlers[topic]
	mb.mu.RUnlock()

	if !exists {
		return nil // No handlers for this topic
	}

	for _, handler := range handlers {
		go func(h MessageHandler) {
			if err := h(message); err != nil {
				logging.Error("Memory broker handler error", "realtime", map[string]interface{}{
					"topic": topic,
					"error": err.Error(),
				})
			}
		}(handler)
	}

	return nil
}

func (mb *MemoryBroker) Subscribe(topic string, handler MessageHandler) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[topic] = append(mb.handlers[topic], handler)
	return nil
}

func (mb *MemoryBroker) Unsubscribe(topic string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	delete(mb.handlers, topic)
	return nil
}

func (mb *MemoryBroker) IsConnected() bool {
	return true
}

// RedisBroker implements a Redis-based message broker for multi-server deployments
type RedisBroker struct {
	config   *config.RedisConfig
	client   interface{} // Redis client (would be *redis.Client in real implementation)
	handlers map[string]MessageHandler
	mu       sync.RWMutex
}

// NewRedisBroker creates a new Redis message broker
func NewRedisBroker(config *config.RedisConfig) *RedisBroker {
	return &RedisBroker{
		config:   config,
		handlers: make(map[string]MessageHandler),
	}
}

func (rb *RedisBroker) Connect(ctx context.Context) error {
	// TODO: Implement Redis connection
	// This would require adding Redis dependency (github.com/go-redis/redis/v8)
	logging.Info("Redis broker would connect here", "realtime", map[string]interface{}{
		"host": rb.config.Host,
		"port": rb.config.Port,
	})
	return fmt.Errorf("Redis broker not implemented yet - add redis dependency first")
}

func (rb *RedisBroker) Disconnect() error {
	// TODO: Implement Redis disconnection
	return nil
}

func (rb *RedisBroker) Publish(topic string, message *BrokerMessage) error {
	// TODO: Implement Redis publish
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	key := rb.config.Prefix + topic
	logging.Debug("Would publish to Redis", "realtime", map[string]interface{}{
		"key":  key,
		"data": string(data),
	})
	return nil
}

func (rb *RedisBroker) Subscribe(topic string, handler MessageHandler) error {
	// TODO: Implement Redis subscribe
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.handlers[topic] = handler
	return nil
}

func (rb *RedisBroker) Unsubscribe(topic string) error {
	// TODO: Implement Redis unsubscribe
	rb.mu.Lock()
	defer rb.mu.Unlock()
	delete(rb.handlers, topic)
	return nil
}

func (rb *RedisBroker) IsConnected() bool {
	// TODO: Check Redis connection status
	return false
}

// RabbitMQBroker implements a RabbitMQ-based message broker
type RabbitMQBroker struct {
	config *config.RabbitConfig
	conn   interface{} // RabbitMQ connection (would be *amqp.Connection in real implementation)
	mu     sync.RWMutex
}

// NewRabbitMQBroker creates a new RabbitMQ message broker
func NewRabbitMQBroker(config *config.RabbitConfig) *RabbitMQBroker {
	return &RabbitMQBroker{
		config: config,
	}
}

func (rmq *RabbitMQBroker) Connect(ctx context.Context) error {
	// TODO: Implement RabbitMQ connection
	// This would require adding RabbitMQ dependency (github.com/streadway/amqp)
	logging.Info("RabbitMQ broker would connect here", "realtime", map[string]interface{}{
		"host":     rmq.config.Host,
		"port":     rmq.config.Port,
		"exchange": rmq.config.Exchange,
	})
	return fmt.Errorf("RabbitMQ broker not implemented yet - add amqp dependency first")
}

func (rmq *RabbitMQBroker) Disconnect() error {
	// TODO: Implement RabbitMQ disconnection
	return nil
}

func (rmq *RabbitMQBroker) Publish(topic string, message *BrokerMessage) error {
	// TODO: Implement RabbitMQ publish
	return nil
}

func (rmq *RabbitMQBroker) Subscribe(topic string, handler MessageHandler) error {
	// TODO: Implement RabbitMQ subscribe
	return nil
}

func (rmq *RabbitMQBroker) Unsubscribe(topic string) error {
	// TODO: Implement RabbitMQ unsubscribe
	return nil
}

func (rmq *RabbitMQBroker) IsConnected() bool {
	// TODO: Check RabbitMQ connection status
	return false
}

// BrokerFactory creates message brokers based on configuration
func NewMessageBroker(config *config.RealtimeConfig) (MessageBroker, error) {
	if !config.Broker.Enabled {
		return NewMemoryBroker(), nil
	}

	switch config.Broker.Type {
	case "memory":
		return NewMemoryBroker(), nil
	case "redis":
		return NewRedisBroker(&config.Broker.Redis), nil
	case "rabbitmq":
		return NewRabbitMQBroker(&config.Broker.RabbitMQ), nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", config.Broker.Type)
	}
}

// MessageTopics defines the topics used for different message types
const (
	TopicCollectionChanges = "collection_changes"
	TopicUserEvents        = "user_events"
	TopicSystemEvents      = "system_events"
	TopicCustomEvents      = "custom_events"
)