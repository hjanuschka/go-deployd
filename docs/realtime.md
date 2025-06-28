# Real-time WebSocket System

go-deployd includes a powerful real-time WebSocket system that enables live data synchronization and custom events. This system is designed to scale from single-server deployments to multi-server, multi-pod environments.

## Overview

The real-time system provides:
- **WebSocket connections** for real-time client communication
- **Automatic collection change notifications** for live data updates
- **Custom event emission** from server-side event scripts
- **Room-based messaging** for targeted communication
- **Multi-server support** via message brokers for horizontal scaling

## Configuration

### Basic Configuration

Real-time features are configured in `config/realtime.json`:

```json
{
  "enabled": true,
  "messageTTL": 3600,
  "broker": {
    "type": "memory",
    "enabled": false
  },
  "limits": {
    "maxConnections": 10000,
    "maxRoomsPerClient": 100,
    "messageRateLimit": 100,
    "pingInterval": 54,
    "pongTimeout": 10
  }
}
```

### Configuration Options

- **enabled**: Enable/disable WebSocket support globally
- **messageTTL**: Message time-to-live in seconds (for broker persistence)
- **limits**: Connection and rate limiting settings

## Message Brokers for Multi-Server Deployments

For deployments across multiple servers or Kubernetes pods, configure a message broker to synchronize real-time events.

### Broker Types

#### 1. Memory Broker (Default)
```json
{
  "broker": {
    "type": "memory",
    "enabled": false
  }
}
```
- **Use case**: Single server, local development
- **Scaling**: Cannot scale horizontally
- **Setup**: No additional setup required

#### 2. Redis Broker
```json
{
  "broker": {
    "type": "redis",
    "enabled": true,
    "redis": {
      "host": "localhost",
      "port": 6379,
      "password": "",
      "database": 0,
      "prefix": "deployd:"
    }
  }
}
```

**Redis Setup:**
```bash
# Docker
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Or with password
docker run -d --name redis -p 6379:6379 redis:7-alpine redis-server --requirepass yourpassword
```

**Production Redis Cluster:**
```bash
# Redis Cluster for high availability
redis-cli --cluster create \
  192.168.1.10:7000 192.168.1.10:7001 \
  192.168.1.11:7000 192.168.1.11:7001 \
  192.168.1.12:7000 192.168.1.12:7001 \
  --cluster-replicas 1
```

#### 3. RabbitMQ Broker
```json
{
  "broker": {
    "type": "rabbitmq",
    "enabled": true,
    "rabbitmq": {
      "host": "localhost",
      "port": 5672,
      "username": "guest",
      "password": "guest",
      "vhost": "/",
      "exchange": "deployd"
    }
  }
}
```

**RabbitMQ Setup:**
```bash
# Docker
docker run -d --name rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=admin \
  -e RABBITMQ_DEFAULT_PASS=password \
  rabbitmq:3-management

# Access management UI at http://localhost:15672
```

**Production RabbitMQ:**
```bash
# Install RabbitMQ
sudo apt-get install rabbitmq-server

# Enable management plugin
sudo rabbitmq-plugins enable rabbitmq_management

# Create virtual host and user
sudo rabbitmqctl add_vhost deployd_prod
sudo rabbitmqctl add_user deployd_user secure_password
sudo rabbitmqctl set_permissions -p deployd_prod deployd_user ".*" ".*" ".*"
```

#### 4. NATS Broker
```json
{
  "broker": {
    "type": "nats",
    "enabled": true,
    "nats": {
      "host": "localhost",
      "port": 4222,
      "username": "",
      "password": "",
      "subject": "deployd"
    }
  }
}
```

**NATS Setup:**
```bash
# Docker
docker run -d --name nats -p 4222:4222 -p 8222:8222 nats:2-alpine

# With authentication
docker run -d --name nats -p 4222:4222 -p 8222:8222 \
  nats:2-alpine --user admin --pass password
```

## Client Usage

### JavaScript Client (dpd.js)

```javascript
// Initialize with real-time enabled (default)
const dpd = new Deployd('http://localhost:2403');

// Listen for collection changes
dpd.todos.on('created', function(todo) {
  console.log('New todo created:', todo);
  updateTodoList();
});

dpd.todos.on('updated', function(todo) {
  console.log('Todo updated:', todo);
  updateTodoItem(todo);
});

dpd.todos.on('deleted', function(todo) {
  console.log('Todo deleted:', todo);
  removeTodoItem(todo.id);
});

// Listen for custom events
dpd.on('notification', function(data) {
  showNotification(data.message);
});

// Join specific rooms
dpd.socket.join('admin-notifications');
```

### Raw WebSocket Usage

```javascript
const ws = new WebSocket('ws://localhost:2403/socket.io/');

ws.onopen = function() {
  // Authenticate with JWT
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-jwt-token'
  }));
  
  // Join rooms
  ws.send(JSON.stringify({
    type: 'join',
    room: 'collection:todos'
  }));
};

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

## Server-Side Event Emission

### From Event Scripts (Go)

```go
func Run(ctx *EventContext) error {
    // Emit to all connected clients
    ctx.Emit("user-logged-in", map[string]interface{}{
        "username": ctx.Data["username"],
        "timestamp": time.Now(),
    })
    
    // Emit to specific room
    ctx.Emit("admin-alert", map[string]interface{}{
        "message": "Critical system event",
        "level": "error",
    }, "admin-room")
    
    return nil
}
```

### From Application Code

```go
// Access the realtime hub from your application
if hub := server.GetRealtimeHub(); hub != nil {
    // Emit to all clients
    hub.EmitToAll("system-maintenance", map[string]interface{}{
        "message": "Server will restart in 5 minutes",
        "countdown": 300,
    })
    
    // Emit to specific room
    hub.EmitToRoom("admin-dashboard", "server-stats", serverStats)
    
    // Emit collection change (automatic for CRUD operations)
    hub.EmitCollectionChange("users", "created", newUser)
}
```

## Room Management

### Automatic Rooms

The system automatically creates rooms for:
- **Collection changes**: `collection:todos`, `collection:users`
- **Global changes**: `collections` (all collection changes)

### Custom Rooms

```javascript
// Client joins custom room
dpd.socket.join('chat-room-123');
dpd.socket.join('admin-notifications');

// Server emits to custom room
ctx.Emit("chat-message", {
    user: "john",
    message: "Hello everyone!"
}, "chat-room-123")
```

## Deployment Scenarios

### Single Server (Development/Small Scale)

```json
{
  "enabled": true,
  "broker": {
    "type": "memory",
    "enabled": false
  }
}
```

### Multi-Server with Redis

```json
{
  "enabled": true,
  "broker": {
    "type": "redis",
    "enabled": true,
    "redis": {
      "host": "redis-cluster.internal",
      "port": 6379,
      "password": "${REDIS_PASSWORD}",
      "database": 0,
      "prefix": "deployd:prod:"
    }
  }
}
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-deployd
spec:
  replicas: 3
  selector:
    matchLabels:
      app: go-deployd
  template:
    metadata:
      labels:
        app: go-deployd
    spec:
      containers:
      - name: go-deployd
        image: go-deployd:latest
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        ports:
        - containerPort: 2403
---
apiVersion: v1
kind: Service
metadata:
  name: redis
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
```

### Large Scale Deployment

For very large deployments (10k+ concurrent connections):

1. **Disable WebSocket** on API servers:
```json
{
  "enabled": false
}
```

2. **Use dedicated WebSocket servers** with load balancer sticky sessions:
```json
{
  "enabled": true,
  "broker": {
    "type": "rabbitmq",
    "enabled": true
  },
  "limits": {
    "maxConnections": 50000
  }
}
```

## Monitoring and Debugging

### Health Check

```bash
# Check WebSocket endpoint
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Key: SGVsbG8gV2ViU29ja2V0IQ==" \
     -H "Sec-WebSocket-Version: 13" \
     http://localhost:2403/socket.io/
```

### Metrics

Real-time metrics available at `/_dashboard/api/metrics/detailed`:
- Active WebSocket connections
- Messages per second
- Room statistics
- Broker connection status

### Debugging

Enable debug logging:
```bash
export LOG_LEVEL=DEBUG
./deployd
```

### Testing

Use the built-in WebSocket test interface:
```
http://localhost:2403/self-test.html
```

## Security Considerations

1. **Authentication**: Always authenticate WebSocket connections
2. **Rate Limiting**: Configure appropriate message rate limits
3. **Origin Checking**: Implement proper origin validation in production
4. **Room Authorization**: Validate room access permissions
5. **Message Validation**: Sanitize and validate all WebSocket messages

## Performance Tuning

### Connection Limits
```json
{
  "limits": {
    "maxConnections": 10000,
    "maxRoomsPerClient": 100,
    "messageRateLimit": 100
  }
}
```

### Broker Optimization

**Redis:**
- Use Redis Cluster for high availability
- Configure appropriate `maxmemory-policy`
- Monitor connection pool sizes

**RabbitMQ:**
- Use multiple exchanges for different message types
- Configure appropriate TTL policies
- Monitor queue depths

**NATS:**
- Use clustering for high availability
- Configure JetStream for persistence
- Monitor connection counts

## Troubleshooting

### Common Issues

1. **WebSocket connection fails**
   - Check if `enabled: true` in configuration
   - Verify port accessibility
   - Check for proxy/firewall issues

2. **Messages not received across servers**
   - Verify broker configuration
   - Check broker connectivity
   - Monitor broker logs

3. **High memory usage**
   - Check for message broker memory leaks
   - Verify connection cleanup
   - Monitor room membership

### Logs

Real-time system logs with `realtime` component:
```bash
# Filter real-time logs
./deployd 2>&1 | grep '"component":"realtime"'
```