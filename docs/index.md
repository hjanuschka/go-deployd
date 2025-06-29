# Go-Deployd Documentation

Welcome to the Go-Deployd documentation! Go-Deployd is a modern API framework that combines the simplicity of REST with real-time capabilities, flexible databases, and a powerful event system.

## Quick Links

- [Collections API](./collections-api.md) - RESTful API for data operations
- [Authentication](./authentication.md) - JWT-based authentication system
- [Admin API](./admin-api.md) - Server administration endpoints
- [Events System](./events-system.md) - Server-side business logic
- [Database Configuration](./database-config.md) - MongoDB, MySQL, and SQLite setup
- [Column-Based Storage](./column-based-storage.md) - High-performance SQLite/MySQL queries
- [WebSocket & Real-time](./websocket-realtime.md) - Real-time event broadcasting
- [dpd.js Client](./dpd-js-client.md) - JavaScript client library
- [Advanced Queries](./advanced-queries.md) - MongoDB-style queries and SQL translation
- [Event Collections](./event-collections.md) - Event-driven endpoints without data storage (noStore)

## Overview

Go-Deployd provides:

- **RESTful API** with automatic CRUD operations
- **Real-time WebSocket** events and broadcasting
- **Flexible Database Support** (MongoDB, MySQL, SQLite)
- **Powerful Event System** in JavaScript or Go
- **Built-in Authentication** with JWT tokens
- **Admin Dashboard** for easy management
- **MongoDB-style Queries** across all databases

## Getting Started

### Installation

```bash
# Download the latest release
wget https://github.com/hjanuschka/go-deployd/releases/latest/download/deployd-linux-amd64
chmod +x deployd-linux-amd64
./deployd-linux-amd64
```

### Create Your First Collection

1. Access the admin dashboard at `http://localhost:2403/_admin`
2. Create a new collection (e.g., "todos")
3. Define properties with validation rules
4. Start making API requests!

### Basic API Usage

```bash
# Create a document
curl -X POST http://localhost:2403/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go-Deployd", "completed": false}'

# Get all documents
curl http://localhost:2403/todos

# Update a document
curl -X PUT http://localhost:2403/todos/doc123 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'

# Delete a document
curl -X DELETE http://localhost:2403/todos/doc123
```

## Key Features

### 1. Collections & REST API

Automatically generated REST endpoints for your data collections with:
- CRUD operations (Create, Read, Update, Delete)
- MongoDB-style queries
- Pagination, sorting, and field selection
- Validation and type checking

[Learn more →](./collections-api.md)

### 2. Real-time WebSocket

Built-in WebSocket support for real-time applications:
- Automatic broadcasting of collection changes
- Custom event emission from server events
- Multi-server scaling with Redis
- Zero configuration required

[Learn more →](./websocket-realtime.md)

### 3. Event System

Powerful server-side hooks for business logic:
- **Validate**: Input validation and sanitization
- **BeforeCommit**: Data transformation before save
- **AfterCommit**: Side effects and notifications
- Write in JavaScript (with npm modules) or Go
- Hot reloading in development

[Learn more →](./events-system.md)

### 4. Flexible Databases

Support for multiple database backends:
- **MongoDB**: For document storage and horizontal scaling
- **MySQL**: For relational data and enterprise deployments
- **SQLite**: For development and single-server apps
- **Column-based storage**: High-performance queries with native SQL indexes
- Same API across all databases
- Automatic query translation

[Learn more →](./database-config.md) | [Column Storage →](./column-based-storage.md)

### 5. Authentication & Security

Built-in authentication system:
- JWT-based authentication
- Master key for admin operations
- User sessions and permissions
- CORS and security headers
- bcrypt password hashing

[Learn more →](./authentication.md)

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  REST API   │────▶│  Database   │
│  (Browser)  │     │  (HTTP/WS)  │     │ (Mongo/SQL) │
└─────────────┘     └─────────────┘     └─────────────┘
       │                    │
       │                    ▼
       │            ┌─────────────┐
       └───────────▶│   Events    │
         WebSocket  │  (JS/Go)    │
                    └─────────────┘
```

## Development Workflow

1. **Define Collections**: Create collections with properties and validation
2. **Add Business Logic**: Write event handlers for validation and processing
3. **Build UI**: Use dpd.js or any HTTP client to interact with the API
4. **Add Real-time**: Connect to WebSocket for live updates
5. **Deploy**: Single binary deployment with your chosen database

## Production Deployment

### Environment Variables

```bash
# Database configuration
export DATABASE_URL="mongodb://localhost:27017/myapp"

# Security
export JWT_SECRET="your-secret-key"
export MASTER_KEY="mk_your_master_key"

# Redis for multi-server WebSocket
export REDIS_URL="redis://localhost:6379"

# Server configuration
export PORT=2403
export PRODUCTION=true
```

### Docker Deployment

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY deployd /usr/local/bin/
COPY resources /app/resources
WORKDIR /app
EXPOSE 2403
CMD ["deployd"]
```

## Community & Support

- [GitHub Repository](https://github.com/hjanuschka/go-deployd)
- [Issue Tracker](https://github.com/hjanuschka/go-deployd/issues)
- [Discussions](https://github.com/hjanuschka/go-deployd/discussions)

## License

Go-Deployd is open source software licensed under the MIT license.