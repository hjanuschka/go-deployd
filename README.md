<div align="center">
  <img src="dashboard/public/deployd-logo.png" alt="Go-Deployd Logo" width="120" height="120">
</div>

# 🚀 Go-Deployd

> **A high-performance, modern reimagining of Deployd in Go**  
> Build JSON APIs in seconds with zero configuration. Focus on your frontend while Go-Deployd handles the backend.

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![MongoDB](https://img.shields.io/badge/MongoDB-4.4+-47A248?style=flat&logo=mongodb&logoColor=white)](https://www.mongodb.com/)
[![React](https://img.shields.io/badge/Dashboard-React%2018-61DAFB?style=flat&logo=react)](https://reactjs.org/)

## ✨ What is Go-Deployd?

Go-Deployd is a **blazing-fast, zero-configuration backend** that transforms your MongoDB database into a full-featured REST API with a beautiful admin dashboard. Write business logic in JavaScript or Go, get instant hot-reload, and ship your app faster than ever.

```bash
# Start a full-stack app in 3 commands
go install github.com/hjanuschka/go-deployd@latest
go-deployd init my-app
go-deployd dev
# 🎉 Your API is ready at http://localhost:2403
```

### 🎯 **Core Philosophy**

- **⚡ Zero Config** - JSON APIs in seconds, not hours
- **🔥 Hot Reload** - JavaScript AND Go events with instant reload
- **🎨 Beautiful Dashboard** - Professional editor with syntax highlighting
- **📊 Production Ready** - Built for scale with Go's performance
- **🔒 Security First** - Built-in authentication, validation, and CORS
- **🌐 MongoDB Native** - Full MongoDB query support with `$sort`, `$limit`, `$fields`

## Quick Start

### With MongoDB (Default)
```bash
# Start the server (uses MongoDB defaults: localhost:27017/deployd)
go run cmd/deployd/main.go

# Or with custom settings
go run cmd/deployd/main.go -port 3000 -db-name myapp -dev

# Run development server with MongoDB (recommended)
make run
```

### With SQLite
```bash
# Run with SQLite database (no MongoDB required)
make run_sqlite

# Or directly
go run cmd/deployd/main.go -dev -database sqlite
```

The server will start at `http://localhost:2403` with a default "todos" collection and admin dashboard at `http://localhost:2403/_dashboard/`.

### 🎨 Dashboard Access

In development mode, visit `http://localhost:2403` and you'll be automatically redirected to the dashboard where you can:

- 📊 **View server stats** and collection overview
- 🗃️ **Manage collections** - create, edit schemas, delete
- 📋 **Browse and edit data** - add, modify, delete documents
- 🔧 **Test APIs** directly from the interface
- ⚙️ **Configure settings** and view server information

## Example Usage

### Create a Todo
```bash
curl -X POST http://localhost:2403/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go", "completed": false}'
```

### Get All Todos
```bash
curl http://localhost:2403/todos
```

### Get Single Todo
```bash
curl http://localhost:2403/todos/{id}
```

### Update Todo
```bash
curl -X PUT http://localhost:2403/todos/{id} \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'
```

### Delete Todo
```bash
curl -X DELETE http://localhost:2403/todos/{id}
```

## Configuration

Create a `resources/` directory with collection configurations:

```bash
mkdir -p resources/todos
```

Create `resources/todos/config.json`:
```json
{
  "properties": {
    "title": {
      "type": "string",
      "required": true
    },
    "completed": {
      "type": "boolean",
      "default": false
    },
    "createdAt": {
      "type": "date",
      "default": "now"
    },
    "priority": {
      "type": "number",
      "default": 1
    }
  }
}
```

## Supported Data Types

- `string` - Text values
- `number` - Numeric values (int/float)
- `boolean` - true/false values  
- `date` - ISO 8601 dates
- `array` - Lists of values
- `object` - Nested objects

## Property Options

- `type` - Data type (required)
- `required` - Must be provided (boolean)
- `default` - Default value (use "now" for current timestamp on dates)

## Query Features

### Filtering
```bash
# Get completed todos
curl "http://localhost:2403/todos?completed=true"

# Get todos with specific priority
curl "http://localhost:2403/todos?priority=1"
```

### MongoDB Query Operators
```bash
# Greater than
curl "http://localhost:2403/todos?\$gt[priority]=1"

# In array
curl "http://localhost:2403/todos?\$in[status]=todo,done"
```

### Special Endpoints

```bash
# Count documents (requires root session)
curl "http://localhost:2403/todos/count"
```

## Authentication & Security

### Master Key System

Go-Deployd uses a secure master key system for administrative access and programmatic user management:

- **Master key is auto-generated** on first startup and stored in `.deployd/security.json`
- **File permissions** are set to 600 (owner read/write only) for security
- **Dashboard access** requires master key authentication
- **Admin API endpoints** are protected by master key validation

#### Configuration Location

All security settings are stored in `.deployd/security.json`:

```json
{
  "masterKey": "mk_...",
  "sessionTTL": 86400,
  "tokenTTL": 2592000,
  "allowRegistration": false
}
```

#### Settings:
- `masterKey` - Auto-generated cryptographically secure key (96 hex chars)
- `sessionTTL` - Session timeout in seconds (default: 24 hours)
- `tokenTTL` - API token timeout in seconds (default: 30 days)  
- `allowRegistration` - Allow public user registration (default: true, set to false for admin-only user creation)

#### Dashboard Authentication

1. Visit `http://localhost:2403/_dashboard`
2. You'll be redirected to the login page
3. Enter your master key (displayed in console on first startup)
4. Access the dashboard with full administrative privileges

#### API Authentication

For programmatic access, include the master key in your requests:

```bash
# Via header
curl -H "X-Master-Key: mk_your_master_key_here" http://localhost:2403/_admin/info

# Via Authorization header
curl -H "Authorization: Bearer mk_your_master_key_here" http://localhost:2403/_admin/info
```

#### User Management

When `allowRegistration` is disabled, users can only be created via master key:

```bash
# Create user with master key
curl -X POST http://localhost:2403/_admin/auth/create-user \
  -H "Content-Type: application/json" \
  -d '{
    "masterKey": "mk_your_master_key_here",
    "userData": {
      "username": "admin",
      "email": "admin@example.com", 
      "password": "secure123",
      "role": "admin"
    }
  }'
```

#### Session Management

The server automatically manages sessions via cookies. With master key authentication:
- **isRoot** is automatically set to `true` for admin privileges
- Sessions persist across requests for seamless dashboard usage
- In development mode (`-dev` flag), additional debugging features are available

## Architecture

```
cmd/deployd/          # Main application entry point
internal/
├── server/           # HTTP server and WebSocket handling
├── router/           # Request routing to resources  
├── resources/        # Resource types (Collections, etc.)
├── database/         # MongoDB abstraction layer
├── context/          # Request context handling
└── sessions/         # Session management
```

## Differences from Original deployd

### What's the Same
- Resource-based architecture
- Collection CRUD operations
- Schema validation and sanitization
- Session management
- Development vs production modes

### What's Different  
- Built with Go instead of Node.js
- Uses native MongoDB driver instead of Mongoose
- WebSocket implementation instead of Socket.IO
- Modern Chakra UI dashboard instead of original dashboard
- Go's type system for better performance

### Completed Features
- [x] JavaScript and Go event hooks with hot reload
- [x] Modern admin dashboard with Chakra UI
- [x] Professional code editor with syntax highlighting  
- [x] Comprehensive event documentation with examples
- [x] MongoDB query operators ($sort, $limit, $skip, $fields)
- [x] Full CRUD operations with event lifecycle hooks
- [x] Session management and authentication support

## Building

```bash
# Build everything (server + dashboard)
make build

# Run in development mode (auto-starts MongoDB + builds dashboard)
make run

# Run with SQLite (no MongoDB required)
make run_sqlite

# Run just the dashboard dev server (for dashboard development)
make dashboard-dev

# Build dashboard for production
make dashboard-build

# Build binary only
go build -o bin/deployd cmd/deployd/main.go

# Run tests
go test ./...

# Get dependencies
go mod tidy
```

## Contributing

This project aims to maintain the spirit and simplicity of the original deployd while leveraging Go's strengths. Contributions welcome!

## License

MIT