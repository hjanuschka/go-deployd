package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// Message types for WebSocket communication
const (
	MessageTypeConnect     = "connect"
	MessageTypeDisconnect  = "disconnect"
	MessageTypeJoin        = "join"
	MessageTypeLeave       = "leave"
	MessageTypeEmit        = "emit"
	MessageTypeAuth        = "auth"
	MessageTypeError       = "error"
	MessageTypeCollectionChange = "collection:change"
)

// Event types for collection changes
const (
	EventTypeCreate = "created"
	EventTypeUpdate = "updated"
	EventTypeDelete = "deleted"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Event   string                 `json:"event,omitempty"`
	Data    interface{}            `json:"data,omitempty"`
	Room    string                 `json:"room,omitempty"`
	Token   string                 `json:"token,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	Rooms    map[string]bool
	User     interface{} // Authenticated user data
	IsRoot   bool        // Admin privileges
	LastSeen time.Time
	mu       sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	upgrader   websocket.Upgrader
	jwtManager *auth.JWTManager
	config     *config.RealtimeConfig
	broker     MessageBroker
	serverID   string
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(jwtManager *auth.JWTManager, realtimeConfig *config.RealtimeConfig) *Hub {
	// Create message broker
	broker, err := NewMessageBroker(realtimeConfig)
	if err != nil {
		logging.Error("Failed to create message broker, using memory broker", "realtime", map[string]interface{}{
			"error": err.Error(),
		})
		broker = NewMemoryBroker()
	}

	hub := &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		jwtManager: jwtManager,
		config:     realtimeConfig,
		broker:     broker,
		serverID:   generateServerID(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development, restrict in production
				return true // TODO: Implement proper origin checking
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Initialize broker and subscribe to topics
	if err := hub.initializeBroker(); err != nil {
		logging.Error("Failed to initialize message broker", "realtime", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return hub
}

// Run starts the hub and handles client connections
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			
			logging.Info("WebSocket client connected", "realtime", map[string]interface{}{
				"client_id": client.ID,
				"clients_count": len(h.clients),
			})
			
			// Send connection confirmation
			client.Send <- h.createMessage(MessageTypeConnect, "", map[string]interface{}{
				"client_id": client.ID,
				"timestamp": time.Now().Unix(),
			}, "")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				
				// Remove client from all rooms
				for room := range client.Rooms {
					h.removeFromRoom(client, room)
				}
			}
			h.mu.Unlock()
			
			logging.Info("WebSocket client disconnected", "realtime", map[string]interface{}{
				"client_id": client.ID,
				"clients_count": len(h.clients),
			})

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("WebSocket upgrade failed", "realtime", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	client := &Client{
		ID:       generateClientID(),
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      h,
		Rooms:    make(map[string]bool),
		LastSeen: time.Now(),
	}

	client.Hub.register <- client

	// Start goroutines for handling client
	go client.writePump()
	go client.readPump()
}

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Error("WebSocket read error", "realtime", map[string]interface{}{
					"client_id": c.ID,
					"error":     err.Error(),
				})
			}
			break
		}

		c.LastSeen = time.Now()
		c.handleMessage(message)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(data []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case MessageTypeAuth:
		c.handleAuth(msg.Token)
	case MessageTypeJoin:
		c.handleJoin(msg.Room)
	case MessageTypeLeave:
		c.handleLeave(msg.Room)
	case MessageTypeEmit:
		c.handleEmit(msg.Event, msg.Data, msg.Room)
	default:
		c.sendError(fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleAuth authenticates the client using JWT token
func (c *Client) handleAuth(token string) {
	if c.Hub.jwtManager == nil {
		c.sendError("Authentication not available")
		return
	}

	claims, err := c.Hub.jwtManager.ValidateToken(token)
	if err != nil {
		c.sendError("Invalid authentication token")
		return
	}

	c.mu.Lock()
	c.User = claims
	c.IsRoot = claims.IsRoot
	c.mu.Unlock()

	logging.Info("WebSocket client authenticated", "realtime", map[string]interface{}{
		"client_id": c.ID,
		"user_id":   claims.UserID,
		"is_root":   claims.IsRoot,
	})

	// Send auth success
	c.Send <- c.Hub.createMessage(MessageTypeAuth, "", map[string]interface{}{
		"authenticated": true,
		"user_id":       claims.UserID,
		"is_root":       claims.IsRoot,
	}, "")
}

// handleJoin adds client to a room
func (c *Client) handleJoin(room string) {
	if room == "" {
		c.sendError("Room name required")
		return
	}

	c.Hub.addToRoom(c, room)
	
	logging.Debug("Client joined room", "realtime", map[string]interface{}{
		"client_id": c.ID,
		"room":      room,
	})
}

// handleLeave removes client from a room
func (c *Client) handleLeave(room string) {
	if room == "" {
		c.sendError("Room name required")
		return
	}

	c.Hub.removeFromRoom(c, room)
	
	logging.Debug("Client left room", "realtime", map[string]interface{}{
		"client_id": c.ID,
		"room":      room,
	})
}

// handleEmit handles custom events emitted by clients
func (c *Client) handleEmit(event string, data interface{}, room string) {
	if event == "" {
		c.sendError("Event name required")
		return
	}

	// TODO: Implement permission checking for events
	message := c.Hub.createMessage(MessageTypeEmit, event, data, room)
	
	if room != "" {
		c.Hub.EmitToRoom(room, event, data)
	} else {
		c.Hub.broadcast <- message
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(message string) {
	errorMsg := c.Hub.createMessage(MessageTypeError, "", nil, "")
	var parsed WebSocketMessage
	json.Unmarshal(errorMsg, &parsed)
	parsed.Error = message
	
	if data, err := json.Marshal(parsed); err == nil {
		c.Send <- data
	}
}

// addToRoom adds a client to a room
func (h *Hub) addToRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true
	client.Rooms[room] = true
}

// removeFromRoom removes a client from a room
func (h *Hub) removeFromRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if clients, ok := h.rooms[room]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
	delete(client.Rooms, room)
}

// EmitToRoom sends a message to all clients in a specific room
func (h *Hub) EmitToRoom(room, event string, data interface{}) {
	message := h.createMessage(MessageTypeEmit, event, data, room)
	
	h.mu.RLock()
	if clients, ok := h.rooms[room]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
				delete(clients, client)
			}
		}
	}
	h.mu.RUnlock()
}

// EmitToAll sends a message to all connected clients
func (h *Hub) EmitToAll(event string, data interface{}) {
	message := h.createMessage(MessageTypeEmit, event, data, "")
	h.broadcast <- message
}

// EmitCollectionChange sends collection change notifications
func (h *Hub) EmitCollectionChange(collection, eventType string, data interface{}) {
	// Send to local clients
	collectionRoom := fmt.Sprintf("collection:%s", collection)
	h.EmitToRoom(collectionRoom, eventType, data)
	
	// Also send to global listeners
	h.EmitToRoom("collections", eventType, map[string]interface{}{
		"collection": collection,
		"data":       data,
	})

	// Publish to broker for multi-server distribution
	h.publishToBroker(TopicCollectionChanges, MessageTypeCollectionChange, eventType, data, collectionRoom)
}

// createMessage creates a WebSocket message
func (h *Hub) createMessage(msgType, event string, data interface{}, room string) []byte {
	msg := WebSocketMessage{
		Type:  msgType,
		Event: event,
		Data:  data,
		Room:  room,
		Meta: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}
	
	if bytes, err := json.Marshal(msg); err == nil {
		return bytes
	}
	return []byte(`{"type":"error","error":"Failed to marshal message"}`)
}

// initializeBroker connects to the message broker and subscribes to topics
func (h *Hub) initializeBroker() error {
	if h.broker == nil {
		return fmt.Errorf("no message broker configured")
	}

	// Connect to broker
	ctx := context.Background()
	if err := h.broker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to message broker: %w", err)
	}

	// Subscribe to collection changes from other servers
	if err := h.broker.Subscribe(TopicCollectionChanges, h.handleBrokerMessage); err != nil {
		return fmt.Errorf("failed to subscribe to collection changes: %w", err)
	}

	// Subscribe to custom events from other servers
	if err := h.broker.Subscribe(TopicCustomEvents, h.handleBrokerMessage); err != nil {
		return fmt.Errorf("failed to subscribe to custom events: %w", err)
	}

	logging.Info("Message broker initialized", "realtime", map[string]interface{}{
		"server_id":      h.serverID,
		"broker_type":    h.config.Broker.Type,
		"multi_server":   h.config.IsMultiServerMode(),
	})

	return nil
}

// handleBrokerMessage handles messages received from the message broker
func (h *Hub) handleBrokerMessage(message *BrokerMessage) error {
	// Don't process messages from our own server
	if message.ServerID == h.serverID {
		return nil
	}

	logging.Debug("Received broker message", "realtime", map[string]interface{}{
		"type":      message.Type,
		"event":     message.Event,
		"room":      message.Room,
		"server_id": message.ServerID,
	})

	// Convert broker message to WebSocket message and broadcast locally
	wsMessage := h.createMessage(message.Type, message.Event, message.Data, message.Room)
	
	if message.Room != "" {
		h.EmitToRoom(message.Room, message.Event, message.Data)
	} else {
		h.broadcast <- wsMessage
	}

	return nil
}

// publishToBroker publishes a message to the broker for multi-server distribution
func (h *Hub) publishToBroker(topic string, msgType, event string, data interface{}, room string) {
	if !h.config.IsMultiServerMode() {
		return
	}

	brokerMessage := &BrokerMessage{
		Type:      msgType,
		Event:     event,
		Data:      data,
		Room:      room,
		ServerID:  h.serverID,
		Timestamp: time.Now().Unix(),
		Meta: map[string]interface{}{
			"source": "websocket_hub",
		},
	}

	if err := h.broker.Publish(topic, brokerMessage); err != nil {
		logging.Error("Failed to publish to broker", "realtime", map[string]interface{}{
			"topic": topic,
			"error": err.Error(),
		})
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// generateServerID generates a unique server ID
func generateServerID() string {
	return fmt.Sprintf("server_%d", time.Now().UnixNano())
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetRooms returns information about active rooms
func (h *Hub) GetRooms() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	rooms := make(map[string]int)
	for room, clients := range h.rooms {
		rooms[room] = len(clients)
	}
	return rooms
}