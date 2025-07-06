# Message Brokers

go-deployd supports multiple message broker implementations for real-time WebSocket communication across multiple servers. This is essential for horizontal scaling in production environments.

## Overview

When running multiple go-deployd servers behind a load balancer, WebSocket connections need to communicate across servers. Message brokers solve this by providing a publish/subscribe system for event distribution.

## Available Brokers

### 1. Memory Broker (Default)
**Best for:** Development and single-server deployments

```json
{
  "broker": {
    "type": "memory",
    "enabled": false
  }
}
```

**Characteristics:**
- ✅ Zero configuration
- ✅ No external dependencies
- ✅ Lowest latency
- ❌ Single server only
- ❌ No persistence

### 2. Redis Broker
**Best for:** Multi-server production deployments

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

**Characteristics:**
- ✅ Battle-tested in production
- ✅ Built-in pub/sub support
- ✅ Can be used for caching/sessions too
- ✅ Supports Redis Cluster
- ⚡ Low latency (< 1ms typical)
- 💾 Optional persistence
- 🔄 Automatic reconnection

### 3. RabbitMQ Broker
**Best for:** High-volume, enterprise deployments

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

**Characteristics:**
- ✅ Enterprise-grade reliability
- ✅ Advanced routing capabilities
- ✅ Message durability options
- ✅ Built-in monitoring/management
- ⚡ Medium latency (1-5ms typical)
- 💾 Persistent message storage
- 🔄 Automatic failover with clustering

## Architecture Comparison

### Redis Architecture
```
┌─────────┐    Pub/Sub     ┌─────────┐
│Server 1 │◄──────────────►│  Redis  │
└─────────┘                 └─────────┘
     ▲                           ▲
     │                           │
     ▼                           ▼
┌─────────┐                 ┌─────────┐
│Client A │                 │Server 2 │
└─────────┘                 └─────────┘
                                 ▲
                                 │
                                 ▼
                            ┌─────────┐
                            │Client B │
                            └─────────┘
```

**Message Flow:**
1. Client A sends message to Server 1
2. Server 1 publishes to Redis channel
3. Redis broadcasts to all subscribed servers
4. Server 2 receives and delivers to Client B

### RabbitMQ Architecture
```
┌─────────┐    Publish     ┌─────────────┐
│Server 1 │───────────────►│  RabbitMQ   │
└─────────┘                 │   Exchange  │
     ▲                      │  (Fanout)   │
     │                      └──────┬──────┘
     ▼                             │
┌─────────┐                 ┌──────▼──────┐
│Client A │                 │   Queues    │
└─────────┘                 │ ┌─────────┐ │
                            │ │Server 1 │ │
                            │ │ Queue   │ │
                            │ └─────────┘ │
                            │ ┌─────────┐ │
                            │ │Server 2 │ │
                            │ │ Queue   │ │
                            │ └────┬────┘ │
                            └──────┼──────┘
                                   ▼
                            ┌─────────┐
                            │Server 2 │
                            └─────────┘
                                 ▲
                                 │
                                 ▼
                            ┌─────────┐
                            │Client B │
                            └─────────┘
```

**Message Flow:**
1. Client A sends message to Server 1
2. Server 1 publishes to RabbitMQ fanout exchange
3. Exchange copies message to all bound queues
4. Each server consumes from its exclusive queue
5. Server 2 delivers to Client B

## Performance Comparison

| Metric | Memory | Redis | RabbitMQ |
|--------|--------|-------|----------|
| Latency | < 0.1ms | < 1ms | 1-5ms |
| Throughput | Unlimited* | 100K msg/s | 50K msg/s |
| Memory Usage | Low | Medium | High |
| CPU Usage | Minimal | Low | Medium |
| Network Overhead | None | Low | Medium |

*Limited only by server resources

## Choosing a Broker

### Use Memory Broker when:
- Running a single server
- Developing locally
- Low traffic application
- Simplicity is priority

### Use Redis Broker when:
- Need multi-server support
- Already using Redis for caching
- Want minimal operational overhead
- Need good performance with reliability

### Use RabbitMQ Broker when:
- Need guaranteed message delivery
- Complex routing requirements
- Enterprise compliance requirements
- Already have RabbitMQ infrastructure

## Configuration Examples

### Development (Single Server)
```json
{
  "enabled": true,
  "messageTTL": 3600,
  "broker": {
    "type": "memory",
    "enabled": false
  }
}
```

### Production with Redis
```json
{
  "enabled": true,
  "messageTTL": 3600,
  "broker": {
    "type": "redis",
    "enabled": true,
    "redis": {
      "host": "redis.internal.example.com",
      "port": 6379,
      "password": "your-secure-password",
      "database": 0,
      "prefix": "deployd:prod:"
    }
  }
}
```

### Enterprise with RabbitMQ
```json
{
  "enabled": true,
  "messageTTL": 7200,
  "broker": {
    "type": "rabbitmq",
    "enabled": true,
    "rabbitmq": {
      "host": "rabbitmq.internal.example.com",
      "port": 5672,
      "username": "deployd",
      "password": "your-secure-password",
      "vhost": "/deployd",
      "exchange": "deployd.events"
    }
  }
}
```

## Monitoring

### Redis Monitoring
```bash
# Monitor Redis pub/sub channels
redis-cli MONITOR | grep deployd:

# Check connected clients
redis-cli CLIENT LIST

# View pub/sub statistics
redis-cli PUBSUB CHANNELS deployd:*
```

### RabbitMQ Monitoring
```bash
# Using management UI
http://localhost:15672

# Using CLI
rabbitmqctl list_exchanges
rabbitmqctl list_queues
rabbitmqctl list_connections
```

## Troubleshooting

### Connection Issues
1. Check network connectivity
2. Verify authentication credentials
3. Ensure firewall rules allow connections
4. Check broker logs for errors

### Message Loss
1. Enable broker persistence (Redis/RabbitMQ)
2. Implement acknowledgments (RabbitMQ)
3. Monitor broker memory usage
4. Check for network timeouts

### Performance Issues
1. Monitor broker CPU/memory
2. Check network latency
3. Review message sizes
4. Consider broker clustering

## Best Practices

1. **Use connection pooling** - Brokers maintain persistent connections
2. **Monitor broker health** - Set up alerts for connection failures
3. **Plan for failures** - Brokers will reconnect automatically
4. **Size appropriately** - Ensure broker can handle peak load
5. **Secure connections** - Use authentication and SSL/TLS in production
6. **Namespace messages** - Use prefixes to separate environments

## Implementation Details

### Topic Routing
All brokers use a topic-based routing system:
- `collection_changes` - Database operation events
- `user_events` - User-specific notifications
- `system_events` - System-wide announcements
- `custom_events` - Application-specific events

### Message Format
```json
{
  "type": "event_type",
  "event": "event_name",
  "data": { ... },
  "room": "optional_room",
  "server_id": "originating_server",
  "timestamp": 1234567890,
  "meta": { ... }
}
```

### Automatic Reconnection
All brokers implement automatic reconnection with exponential backoff:
- Initial retry: 1 second
- Max retry interval: 30 seconds
- Infinite retry attempts

This ensures resilience against temporary network issues or broker restarts.