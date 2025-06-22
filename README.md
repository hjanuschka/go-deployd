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
- **🎨 Beautiful Dashboard** - CodeMirror editor with syntax highlighting
- **📊 Production Ready** - Built for scale with Go's performance
- **🔒 Security First** - Built-in authentication, validation, and CORS
- **🌐 MongoDB Native** - Full MongoDB query support with `$sort`, `$limit`, `$fields`

## Quick Start

```bash
# Start the server (uses MongoDB defaults: localhost:27017/deployd)
go run cmd/deployd/main.go

# Or with custom settings
go run cmd/deployd/main.go -port 3000 -db-name myapp -dev
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

## Sessions & Authentication

The server automatically manages sessions via cookies. In development mode (`-dev` flag), all sessions have root privileges.

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
- Modern Material UI dashboard instead of original dashboard
- Go's type system for better performance

### Planned Features
- [ ] JavaScript event hooks (using goja)
- [ ] File upload support
- [x] Modern admin dashboard with Material UI
- [ ] Real-time dashboard updates
- [ ] Clustering support
- [ ] Plugin system
- [ ] Full Socket.IO compatibility

## Building

```bash
# Build everything (server + dashboard)
make build

# Run in development mode (auto-starts MongoDB + builds dashboard)
make run

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

Apache 2.0 (same as original deployd)