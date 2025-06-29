<div align="center">
  <img src="dashboard/public/deployd-logo.png" alt="Go-Deployd Logo" width="120" height="120">
</div>

# 🚀 Go-Deployd

> **A high-performance, modern reimagining of Deployd in Go**  
> Build JSON APIs in seconds with zero configuration. Focus on your frontend while Go-Deployd handles the backend.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![WebSocket](https://img.shields.io/badge/WebSocket-Real--time-4CAF50?style=flat&logo=websocket)](./docs/websocket-realtime.md)
[![SQLite](https://img.shields.io/badge/SQLite-3.0+-003B57?style=flat&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![MongoDB](https://img.shields.io/badge/MongoDB-4.4+-47A248?style=flat&logo=mongodb&logoColor=white)](https://www.mongodb.com/)
[![MySQL](https://img.shields.io/badge/MySQL-8.0+-4479A1?style=flat&logo=mysql&logoColor=white)](https://www.mysql.com/)
[![V8](https://img.shields.io/badge/V8-JavaScript-4B8BF5?style=flat&logo=v8&logoColor=white)](https://v8.dev/)
[![React](https://img.shields.io/badge/Dashboard-React%2018-61DAFB?style=flat&logo=react)](https://reactjs.org/)

## ✨ What is Go-Deployd?

Go-Deployd is a **blazing-fast, zero-configuration backend** that transforms a simple SQLite database into a full-featured REST API with real-time WebSocket events and a beautiful admin dashboard. Write business logic in JavaScript or Go, get instant hot-reload, and ship your app faster than ever.

### 🎯 **Core Philosophy**

- **⚡ Zero Dependencies** - SQLite built-in, no external database required
- **🔥 Zero Config** - JSON APIs in seconds, not hours  
- **🎨 Beautiful Dashboard** - Professional editor with syntax highlighting
- **⚡ Real-time Events** - Built-in WebSocket support for live updates
- **📊 Production Ready** - Built for scale with Go's performance
- **🔒 Security First** - Built-in JWT authentication, validation, and CORS
- **🌐 Multiple Databases** - SQLite (default), MongoDB, MySQL support
- **🔍 MongoDB-style Queries** - Familiar query syntax across all databases
- **📁 File Storage** - Local, S3, MinIO support with event-driven processing
- **🚀 NoStore Collections** - Event-driven endpoints without database storage

## 🚀 Quick Start (3 Commands)

```bash
# 1. Clone the repository
git clone https://github.com/hjanuschka/go-deployd.git
cd go-deployd

# 2. Run with SQLite (no dependencies required)
make run_sqlite

# 3. Open your browser
# 🎉 Your API is ready at http://localhost:2403
# 📊 Dashboard available at http://localhost:2403/_dashboard
```

That's it! You now have:
- ✅ A running REST API server with real-time WebSocket events
- ✅ SQLite database with sample collections (`users`, `todo-js`, `todo-go`)
- ✅ Beautiful admin dashboard with comprehensive documentation
- ✅ JWT authentication system with secure token handling
- ✅ MongoDB-style queries across all database backends
- ✅ API testing interface at `/self-test.html`

## 📋 Sample Collections Included

Your fresh installation comes with working examples:

### **Users Collection** (Built-in)
- JWT authentication ready
- User registration and login
- Password hashing with bcrypt
- Email verification support

### **Files Collection** (Built-in File Storage)
- File upload/download with Local, S3, MinIO backends
- Event-driven processing with Go events for validation
- Automatic metadata extraction and storage
- Real-time WebSocket notifications for file operations

### **Calculator Collections** (NoStore Examples)
- `calculator-js` - JavaScript-based computation endpoint
- `calculator-go` - Go-based computation endpoint
- No database storage - pure event-driven processing
- Perfect examples of NoStore collections

### **Todo-JS Collection** (JavaScript Events)
```javascript
// resources/todo-js/validate.js
function Run(context) {
    if (!context.data.title || context.data.title.length < 1) {
        context.cancel("Title is required", 400);
    }
}
```

### **Todo-Go Collection** (Go Events)
```go
// resources/todo-go/validate.go
func Run(ctx *EventContext) error {
    title, ok := ctx.Data["title"].(string)
    if !ok || strings.TrimSpace(title) == "" {
        ctx.Cancel("Title is required", 400)
        return nil
    }
    return nil
}
```

## 🎨 Dashboard Features

Visit `http://localhost:2403/_dashboard` to access:

- 📊 **Server Metrics** - Real-time performance stats and monitoring
- 🗃️ **Collection Management** - Create, edit schemas, browse data with visual query builder
- 👥 **User Management** - Built-in user system with JWT authentication
- 📝 **Event Editor** - Write JavaScript/Go events with syntax highlighting and hot reload
- 📊 **Logs Viewer** - Real-time application logs with filtering and search
- ⚙️ **Settings** - Configure security, database connections, and more
- 📚 **Documentation** - Comprehensive guides accessible from GitHub

## 📚 Documentation

Comprehensive documentation is available in the [docs/](./docs/) directory:

- **[Getting Started Guide](./docs/index.md)** - Complete overview and quick start
- **[Collections API](./docs/collections-api.md)** - REST API reference with examples
- **[Authentication](./docs/authentication.md)** - JWT authentication and security
- **[Events System](./docs/events-system.md)** - JavaScript and Go event handlers
- **[WebSocket & Real-time](./docs/websocket-realtime.md)** - Real-time events and broadcasting
- **[Database Configuration](./docs/database-config.md)** - MongoDB, MySQL, and SQLite setup
- **[dpd.js Client](./docs/dpd-js-client.md)** - JavaScript client library
- **[Advanced Queries](./docs/advanced-queries.md)** - MongoDB-style queries and SQL translation

## ⚡ Real-time Features

Go-Deployd includes built-in WebSocket support for real-time applications:

```javascript
// Connect to WebSocket for live updates
const ws = new WebSocket('ws://localhost:2403/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'collection_change') {
    console.log(`${data.action} in ${data.collection}:`, data.document);
    // Update your UI in real-time!
  }
};
```

**Real-time Capabilities:**
- 🔄 **Automatic Broadcasting** - Collection changes broadcast to all clients
- 🎯 **Custom Events** - Emit custom events from server events with `emit()`
- 🏗️ **Multi-server Scaling** - Redis support for horizontal scaling
- 🔌 **Zero Configuration** - WebSocket works out of the box

## 💡 Example API Usage

### File Storage Operations
```bash
# Upload a file
curl -X POST http://localhost:2403/files \
  -F "file=@document.pdf" \
  -F "metadata={\"category\":\"documents\"}"

# List all files
curl http://localhost:2403/files

# Download a file
curl http://localhost:2403/files/[file-id]/download

# Delete a file
curl -X DELETE http://localhost:2403/files/[file-id]
```

### NoStore Collections (Event-Only Endpoints)
```bash
# Calculator endpoint (no database storage)
curl -X POST http://localhost:2403/calculator-js \
  -H "Content-Type: application/json" \
  -d '{"operation": "add", "a": 10, "b": 5}'

# Response: {"result": 15, "operation": "add"}
```

### Create a Todo
```bash
curl -X POST http://localhost:2403/todo-js \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go-Deployd", "completed": false, "priority": 1}'
```

### Get All Todos
```bash
curl http://localhost:2403/todo-js
```

### Query with Parameters
```bash
# Get completed todos
curl "http://localhost:2403/todo-js?completed=true"

# Sort by priority, limit results
curl "http://localhost:2403/todo-js?\$sort[priority]=-1&\$limit=10"

# MongoDB-style operators work across all databases
curl "http://localhost:2403/todo-js?priority[\$gte]=5&title[\$regex]=urgent"

# Complex queries with POST /query endpoint
curl -X POST http://localhost:2403/todo-js/query \
  -H "Content-Type: application/json" \
  -d '{"query": {"$or": [{"priority": {"$gte": 8}}, {"title": {"$regex": "urgent"}}]}}'
```

## 🔧 Advanced Features

### 📁 File Storage System
Go-Deployd includes a built-in file storage system supporting multiple backends:

```bash
# Configure storage backend in environment
export STORAGE_TYPE=s3
export AWS_BUCKET=my-bucket
export AWS_REGION=us-east-1

# Or use MinIO
export STORAGE_TYPE=minio
export MINIO_ENDPOINT=localhost:9000
export MINIO_BUCKET=uploads

# Local storage (default)
export STORAGE_TYPE=local
export STORAGE_PATH=./uploads
```

**File Storage Features:**
- **Multiple Backends** - Local filesystem, AWS S3, MinIO support
- **Event-Driven Processing** - Custom validation and processing with Go events
- **Metadata Management** - Automatic extraction and custom metadata storage
- **Real-time Notifications** - WebSocket events for file operations
- **Secure Access** - Built-in authentication and authorization

### 🚀 NoStore Collections
Create event-driven endpoints without database storage:

```json
// resources/api-endpoint/config.json
{
  "type": "Collection",
  "noStore": true,
  "properties": {}
}
```

```javascript
// resources/api-endpoint/post.js
function Run(context) {
    const { operation, a, b } = context.data;
    
    let result;
    switch (operation) {
        case 'add': result = a + b; break;
        case 'multiply': result = a * b; break;
        default: result = 'Invalid operation';
    }
    
    context.data = { operation, result, timestamp: new Date() };
}
```

**NoStore Use Cases:**
- API endpoints and webhooks
- Calculators and converters
- External service integrations
- Data transformation pipelines
- Serverless-style functions

### Multiple Database Support

```bash
# SQLite (default, no setup required)
make run_sqlite

# MongoDB (requires MongoDB server)
make run

# MySQL (requires MySQL server and .env file)
make run_mysql
```

### Development with Hot Reload

```bash
# Install development dependencies
npm install

# Start with hot reload (recommended for development)
make dev-sqlite
```

### Custom Configuration

Create collections by adding folders to `resources/`:

```bash
mkdir resources/my-collection
```

Create `resources/my-collection/config.json`:
```json
{
  "properties": {
    "name": { "type": "string", "required": true },
    "email": { "type": "string", "required": true },
    "active": { "type": "boolean", "default": true },
    "createdAt": { "type": "date", "default": "now" }
  }
}
```

Add event handlers:
- `resources/my-collection/validate.js` - Input validation
- `resources/my-collection/post.js` - After creation logic
- `resources/my-collection/get.go` - Custom response formatting

## 🔐 Authentication & Security

### Auto-Generated Security

On first startup, Go-Deployd automatically:
- ✅ Generates a secure master key (displayed in console)
- ✅ Creates JWT signing keys
- ✅ Sets up user authentication system
- ✅ Configures secure file permissions

### Master Key Authentication

Use the displayed master key to access the dashboard:
```
🔐 Generated new master key and saved to .deployd/security.json
   Master Key: mk_abc123...
   Keep this key secure! It provides administrative access.
```

### JWT Authentication

Users can register and login to get JWT tokens:
```bash
# Register a new user
curl -X POST http://localhost:2403/users \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "email": "alice@example.com", "password": "secure123"}'

# Login to get JWT token
curl -X POST http://localhost:2403/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "secure123"}'
```

## 🏗️ Supported Data Types

- `string` - Text values
- `number` - Numeric values (int/float)  
- `boolean` - true/false values
- `date` - ISO 8601 dates (use "now" for current timestamp)
- `array` - Lists of values
- `object` - Nested objects

## 🔍 Advanced Query Features

### MongoDB-Style Operators Across All Databases
```bash
# Comparison operators (work with SQLite, MySQL, and MongoDB)
curl "http://localhost:2403/todos?priority[\$gt]=1"
curl "http://localhost:2403/todos?status[\$in]=todo,done"
curl "http://localhost:2403/todos?title[\$regex]=urgent"

# Complex nested queries
curl -X POST http://localhost:2403/todos/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "$and": [
        {"priority": {"$gte": 5}},
        {"$or": [
          {"title": {"$regex": "urgent"}},
          {"assignedTo": "admin"}
        ]}
      ]
    }
  }'

# Force MongoDB queries (when using MongoDB backend)
curl -X POST http://localhost:2403/todos/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": {"tags": {"$all": ["urgent", "important"]}},
    "options": {"$forceMongo": true}
  }'

# Sorting and pagination
curl "http://localhost:2403/todos?\$sort[createdAt]=-1&\$limit=5&\$skip=10"

# Field selection
curl "http://localhost:2403/todos?\$fields=title,completed"
```

### Special Endpoints
```bash
# Count documents
curl "http://localhost:2403/todos/count"

# Collection list
curl "http://localhost:2403/collections"

# Health check and server info
curl "http://localhost:2403/_admin/info"
```

## 🛠️ Build & Deploy Commands

### 🏗️ **Building**

```bash
# Build development binary (includes web/ folder dependency)
make build

# Build standalone production binary (NO web/ folder needed!)
make dist
```

### 🔥 **Development Workflow**

```bash
# Install development tools first
make install-dev-tools

# Development with hot reload (recommended)
make dev-sqlite          # SQLite + Go hot reload + React dev server
make dev-mongo           # MongoDB + Go hot reload + React dev server

# Alternative: Simple development (no hot reload)
make dev-simple          # Faster startup, manual restarts needed

# Dashboard development
make dashboard-dev       # React-only dev server (port 3001)
make dashboard-build-dev # Build dashboard with debug symbols
```

**Development Features:**
- 🔥 **Go hot reload** with `nodemon` - changes trigger automatic rebuilds
- 🎨 **React hot reload** with Vite - instant UI updates  
- 🐛 **Debug symbols** - Unminified JS (3.3MB) + 7MB sourcemaps
- ⚡ **Fast rebuilds** - Optimized for development speed
- 📝 **Config exclusion** - Changes to `resources/*/config.json` don't trigger reload

### 🚀 **Production Deployment**

```bash
# Method 1: Standalone Binary (RECOMMENDED)
make dist                    # Creates bin/deployd-dist
./bin/deployd-dist -port 80  # Only needs resources/ folder!

# Method 2: Standard Binary + Web Assets
make build                   # Creates bin/deployd
make dashboard-build         # Creates optimized web/dashboard/
./bin/deployd -port 80       # Needs both binary + web/ folder
```

**Production Features:**
- 📦 **Standalone deployment** - Single binary + `resources/` folder
- ⚡ **Optimized dashboard** - Minified (1.6MB), code-split, compressed
- 🔒 **Production security** - No debug symbols or sourcemaps
- 🚀 **Fast startup** - Pre-built assets, no build step needed

### 🎯 **Database Options**

```bash
# SQLite (recommended, zero dependencies)
make run_sqlite          # or ./bin/deployd-dist -db-type sqlite

# MongoDB (requires MongoDB server)
make run                 # or ./bin/deployd-dist -db-type mongodb

# MySQL (requires MySQL + .env config)
make run_mysql           # or ./bin/deployd-dist -db-type mysql
```

### 🧪 **Testing & Utilities**

```bash
# Testing
make test               # Run Go unit tests
make e2e-test          # End-to-end API tests
make test-coverage     # Tests with coverage report

# Utilities
make clean             # Clean build artifacts
make fmt               # Format Go code
make lint              # Lint Go code (requires golangci-lint)
make help              # Show all available commands
```

## 📁 Project Structure

```
go-deployd/
├── cmd/deployd/           # Main application
├── resources/             # Your collections and events
│   ├── users/            # Built-in user system
│   ├── todo-js/          # JavaScript event examples
│   └── todo-go/          # Go event examples
├── dashboard/            # React admin dashboard
├── internal/             # Core Go packages
└── web/                  # Built dashboard assets
```

## 🚀 Production Deployment

1. Build the application:
```bash
make build
```

2. The binary includes everything needed:
```bash
./bin/deployd -port 80 -db-type sqlite
```

3. For additional security, set environment variables:
```bash
DEPLOYD_MASTER_KEY=your-secure-key ./bin/deployd
```

## 🆚 Differences from Original Deployd

### What's the Same
- Resource-based architecture with collections
- Event lifecycle hooks (validate, post, get, put, delete)
- Dashboard for managing data and events
- Zero-configuration philosophy

### What's Better
- **10x Faster** - Go performance vs Node.js with optimized query engines
- **Zero Dependencies** - SQLite built-in, no MongoDB setup required
- **Real-time Built-in** - WebSocket events with zero configuration
- **Modern Dashboard** - React 18 with Chakra UI and comprehensive documentation
- **Hot Reload** - For both JavaScript AND Go events with instant updates
- **JWT Authentication** - Modern token-based auth with secure defaults
- **Multi-Database** - SQLite, MongoDB, MySQL support with unified query API
- **MongoDB-style Queries** - Familiar query syntax across all database backends
- **Production Ready** - Built for scale with proper error handling and monitoring
- **Developer Experience** - Visual query builder, comprehensive docs, and testing tools

## 🤝 Contributing

This project aims to maintain the simplicity of the original Deployd while leveraging Go's performance and modern web technologies. Contributions welcome!

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

<div align="center">
  <strong>Ready to build amazing APIs? Start with <code>make run_sqlite</code> and let Go-Deployd handle the rest! 🚀</strong>
</div>