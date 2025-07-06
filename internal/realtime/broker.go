package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
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
	client   *redis.Client
	pubsub   *redis.PubSub
	handlers map[string]MessageHandler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRedisBroker creates a new Redis message broker
func NewRedisBroker(config *config.RedisConfig) *RedisBroker {
	return &RedisBroker{
		config:   config,
		handlers: make(map[string]MessageHandler),
	}
}

func (rb *RedisBroker) Connect(ctx context.Context) error {
	rb.ctx, rb.cancel = context.WithCancel(ctx)
	
	// Create Redis client
	rb.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", rb.config.Host, rb.config.Port),
		Password: rb.config.Password,
		DB:       rb.config.Database,
	})
	
	// Test connection
	if err := rb.client.Ping(rb.ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	// Create pubsub client
	rb.pubsub = rb.client.Subscribe(rb.ctx)
	
	logging.Info("Redis broker connected", "realtime", map[string]interface{}{
		"host": rb.config.Host,
		"port": rb.config.Port,
		"db":   rb.config.Database,
	})
	
	return nil
}

func (rb *RedisBroker) Disconnect() error {
	if rb.cancel != nil {
		rb.cancel()
	}
	
	if rb.pubsub != nil {
		if err := rb.pubsub.Close(); err != nil {
			logging.Error("Failed to close Redis pubsub", "realtime", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	if rb.client != nil {
		if err := rb.client.Close(); err != nil {
			logging.Error("Failed to close Redis client", "realtime", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	logging.Info("Redis broker disconnected", "realtime", nil)
	return nil
}

func (rb *RedisBroker) Publish(topic string, message *BrokerMessage) error {
	if rb.client == nil {
		return fmt.Errorf("Redis client not connected")
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	channel := rb.config.Prefix + topic
	
	// Publish to Redis channel
	if err := rb.client.Publish(rb.ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}
	
	logging.Debug("Published to Redis", "realtime", map[string]interface{}{
		"channel": channel,
		"event":   message.Event,
		"type":    message.Type,
	})
	
	return nil
}

func (rb *RedisBroker) Subscribe(topic string, handler MessageHandler) error {
	if rb.pubsub == nil {
		return fmt.Errorf("Redis pubsub not connected")
	}
	
	rb.mu.Lock()
	rb.handlers[topic] = handler
	rb.mu.Unlock()
	
	channel := rb.config.Prefix + topic
	
	// Subscribe to Redis channel
	if err := rb.pubsub.Subscribe(rb.ctx, channel); err != nil {
		return fmt.Errorf("failed to subscribe to Redis channel: %w", err)
	}
	
	// Start listening for messages in a goroutine
	go func() {
		ch := rb.pubsub.Channel()
		for msg := range ch {
			var message BrokerMessage
			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				logging.Error("Failed to unmarshal Redis message", "realtime", map[string]interface{}{
					"error":   err.Error(),
					"channel": msg.Channel,
				})
				continue
			}
			
			// Get handler for topic
			rb.mu.RLock()
			handler, exists := rb.handlers[topic]
			rb.mu.RUnlock()
			
			if exists && handler != nil {
				if err := handler(&message); err != nil {
					logging.Error("Redis broker handler error", "realtime", map[string]interface{}{
						"topic": topic,
						"error": err.Error(),
					})
				}
			}
		}
	}()
	
	logging.Info("Subscribed to Redis channel", "realtime", map[string]interface{}{
		"channel": channel,
		"topic":   topic,
	})
	
	return nil
}

func (rb *RedisBroker) Unsubscribe(topic string) error {
	if rb.pubsub == nil {
		return fmt.Errorf("Redis pubsub not connected")
	}
	
	rb.mu.Lock()
	delete(rb.handlers, topic)
	rb.mu.Unlock()
	
	channel := rb.config.Prefix + topic
	
	// Unsubscribe from Redis channel
	if err := rb.pubsub.Unsubscribe(rb.ctx, channel); err != nil {
		return fmt.Errorf("failed to unsubscribe from Redis channel: %w", err)
	}
	
	logging.Info("Unsubscribed from Redis channel", "realtime", map[string]interface{}{
		"channel": channel,
		"topic":   topic,
	})
	
	return nil
}

func (rb *RedisBroker) IsConnected() bool {
	if rb.client == nil {
		return false
	}
	
	// Check connection with ping
	if err := rb.client.Ping(rb.ctx).Err(); err != nil {
		return false
	}
	
	return true
}

// RabbitMQBroker implements a RabbitMQ-based message broker with fanout exchange
type RabbitMQBroker struct {
	config       *config.RabbitConfig
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchange     string
	queue        *amqp.Queue
	consumers    map[string]<-chan amqp.Delivery
	handlers     map[string]MessageHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewRabbitMQBroker creates a new RabbitMQ message broker
func NewRabbitMQBroker(config *config.RabbitConfig) *RabbitMQBroker {
	return &RabbitMQBroker{
		config:    config,
		exchange:  config.Exchange,
		consumers: make(map[string]<-chan amqp.Delivery),
		handlers:  make(map[string]MessageHandler),
	}
}

func (rmq *RabbitMQBroker) Connect(ctx context.Context) error {
	rmq.ctx, rmq.cancel = context.WithCancel(ctx)
	
	// Build connection URL
	connURL := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		rmq.config.Username,
		rmq.config.Password,
		rmq.config.Host,
		rmq.config.Port,
		rmq.config.VHost)
	
	// Connect to RabbitMQ
	conn, err := amqp.Dial(connURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	rmq.conn = conn
	
	// Create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create channel: %w", err)
	}
	rmq.channel = channel
	
	// Declare fanout exchange
	err = channel.ExchangeDeclare(
		rmq.exchange, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	
	// Declare exclusive queue for this server
	queue, err := channel.QueueDeclare(
		"",    // name (auto-generate)
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	rmq.queue = &queue
	
	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name,   // queue name
		"",           // routing key (not used for fanout)
		rmq.exchange, // exchange
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to bind queue: %w", err)
	}
	
	logging.Info("RabbitMQ broker connected", "realtime", map[string]interface{}{
		"host":     rmq.config.Host,
		"port":     rmq.config.Port,
		"exchange": rmq.exchange,
		"queue":    queue.Name,
	})
	
	return nil
}

func (rmq *RabbitMQBroker) Disconnect() error {
	if rmq.cancel != nil {
		rmq.cancel()
	}
	
	// Close all consumers
	rmq.mu.Lock()
	for topic := range rmq.consumers {
		if rmq.channel != nil {
			if err := rmq.channel.Cancel(topic, false); err != nil {
				logging.Error("Failed to cancel RabbitMQ consumer", "realtime", map[string]interface{}{
					"topic": topic,
					"error": err.Error(),
				})
			}
		}
	}
	rmq.consumers = make(map[string]<-chan amqp.Delivery)
	rmq.mu.Unlock()
	
	if rmq.channel != nil {
		if err := rmq.channel.Close(); err != nil {
			logging.Error("Failed to close RabbitMQ channel", "realtime", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	if rmq.conn != nil {
		if err := rmq.conn.Close(); err != nil {
			logging.Error("Failed to close RabbitMQ connection", "realtime", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	logging.Info("RabbitMQ broker disconnected", "realtime", nil)
	return nil
}

func (rmq *RabbitMQBroker) Publish(topic string, message *BrokerMessage) error {
	if rmq.channel == nil {
		return fmt.Errorf("RabbitMQ channel not connected")
	}
	
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Create AMQP message with topic as a header
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
		Headers: amqp.Table{
			"topic": topic,
		},
	}
	
	// Publish to fanout exchange
	err = rmq.channel.PublishWithContext(
		rmq.ctx,
		rmq.exchange, // exchange
		"",           // routing key (ignored for fanout)
		false,        // mandatory
		false,        // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("failed to publish to RabbitMQ: %w", err)
	}
	
	logging.Debug("Published to RabbitMQ", "realtime", map[string]interface{}{
		"exchange": rmq.exchange,
		"topic":    topic,
		"event":    message.Event,
		"type":     message.Type,
	})
	
	return nil
}

func (rmq *RabbitMQBroker) Subscribe(topic string, handler MessageHandler) error {
	if rmq.channel == nil || rmq.queue == nil {
		return fmt.Errorf("RabbitMQ not connected")
	}
	
	rmq.mu.Lock()
	rmq.handlers[topic] = handler
	rmq.mu.Unlock()
	
	// Only create one consumer for all topics (fanout receives all messages)
	if len(rmq.consumers) == 0 {
		// Start consuming from queue
		messages, err := rmq.channel.Consume(
			rmq.queue.Name, // queue
			"",             // consumer tag (auto-generate)
			true,           // auto-ack
			true,           // exclusive
			false,          // no-local
			false,          // no-wait
			nil,            // args
		)
		if err != nil {
			return fmt.Errorf("failed to start consuming: %w", err)
		}
		
		rmq.mu.Lock()
		rmq.consumers["_default"] = messages
		rmq.mu.Unlock()
		
		// Start message processor
		go rmq.processMessages(messages)
	}
	
	logging.Info("Subscribed to RabbitMQ topic", "realtime", map[string]interface{}{
		"topic":    topic,
		"exchange": rmq.exchange,
		"queue":    rmq.queue.Name,
	})
	
	return nil
}

// processMessages handles incoming messages from RabbitMQ
func (rmq *RabbitMQBroker) processMessages(messages <-chan amqp.Delivery) {
	for msg := range messages {
		// Extract topic from headers
		var topic string
		if topicHeader, exists := msg.Headers["topic"]; exists {
			topic, _ = topicHeader.(string)
		}
		
		// Unmarshal message
		var message BrokerMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			logging.Error("Failed to unmarshal RabbitMQ message", "realtime", map[string]interface{}{
				"error": err.Error(),
				"body":  string(msg.Body),
			})
			continue
		}
		
		// Get handler for topic
		rmq.mu.RLock()
		handler, exists := rmq.handlers[topic]
		rmq.mu.RUnlock()
		
		if exists && handler != nil {
			if err := handler(&message); err != nil {
				logging.Error("RabbitMQ broker handler error", "realtime", map[string]interface{}{
					"topic": topic,
					"error": err.Error(),
				})
			}
		}
	}
}

func (rmq *RabbitMQBroker) Unsubscribe(topic string) error {
	rmq.mu.Lock()
	delete(rmq.handlers, topic)
	
	// If no more handlers, stop consuming
	if len(rmq.handlers) == 0 && len(rmq.consumers) > 0 {
		if rmq.channel != nil {
			if err := rmq.channel.Cancel("_default", false); err != nil {
				logging.Error("Failed to cancel RabbitMQ consumer", "realtime", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
		rmq.consumers = make(map[string]<-chan amqp.Delivery)
	}
	rmq.mu.Unlock()
	
	logging.Info("Unsubscribed from RabbitMQ topic", "realtime", map[string]interface{}{
		"topic": topic,
	})
	
	return nil
}

func (rmq *RabbitMQBroker) IsConnected() bool {
	if rmq.conn == nil || rmq.channel == nil {
		return false
	}
	
	// Check if connection is closed
	if rmq.conn.IsClosed() {
		return false
	}
	
	return true
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