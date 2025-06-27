<div align="center">
  <img src="dashboard/public/deployd-logo.png" alt="Go-Deployd Logo" width="120" height="120">
</div>

# 🚀 Go-Deployd

> **A high-performance, modern reimagining of Deployd in Go**  
> Build JSON APIs in seconds with zero configuration. Focus on your frontend while Go-Deployd handles the backend.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![SQLite](https://img.shields.io/badge/SQLite-3.0+-003B57?style=flat&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![React](https://img.shields.io/badge/Dashboard-React%2018-61DAFB?style=flat&logo=react)](https://reactjs.org/)

## ✨ What is Go-Deployd?

Go-Deployd is a **blazing-fast, zero-configuration backend** that transforms a simple SQLite database into a full-featured REST API with a beautiful admin dashboard. Write business logic in JavaScript or Go, get instant hot-reload, and ship your app faster than ever.

### 🎯 **Core Philosophy**

- **⚡ Zero Dependencies** - SQLite built-in, no external database required
- **🔥 Zero Config** - JSON APIs in seconds, not hours  
- **🎨 Beautiful Dashboard** - Professional editor with syntax highlighting
- **📊 Production Ready** - Built for scale with Go's performance
- **🔒 Security First** - Built-in JWT authentication, validation, and CORS
- **🌐 Multiple Databases** - SQLite (default), MongoDB, MySQL support

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
- ✅ A running REST API server
- ✅ SQLite database with sample collections (`users`, `todo-js`, `todo-go`)
- ✅ Beautiful admin dashboard
- ✅ JWT authentication system
- ✅ API testing interface at `/self-test.html`

## 📋 Sample Collections Included

Your fresh installation comes with working examples:

### **Users Collection** (Built-in)
- JWT authentication ready
- User registration and login
- Password hashing with bcrypt
- Email verification support

### **Todo-JS Collection** (JavaScript Events)
```javascript
// resources/todo-js/validate.js
if (!this.title || this.title.length < 1) {
    cancel("Title is required", 400);
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

- 📊 **Server Metrics** - Real-time performance stats
- 🗃️ **Collection Management** - Create, edit schemas, browse data
- 👥 **User Management** - Built-in user system with roles
- 📝 **Event Editor** - Write JavaScript/Go events with syntax highlighting  
- 📊 **Logs Viewer** - Real-time application logs with filtering
- ⚙️ **Settings** - Configure security, email, and more

## 💡 Example API Usage

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
```

## 🔧 Advanced Usage

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
# Install development tools (air for Go hot reload)
make install-dev-tools

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

## 🔍 Query Features

### MongoDB-Style Operators
```bash
# Comparison operators
curl "http://localhost:2403/todos?\$gt[priority]=1"
curl "http://localhost:2403/todos?\$in[status]=todo,done"

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
```

## 🛠️ Available Commands

```bash
# Quick start (SQLite, zero dependencies)
make run_sqlite          # Production-ready server

# Development (with hot reload)
make dev-sqlite          # Recommended for development
make dev                 # With MongoDB
make install-dev-tools   # Install air for hot reload

# Building
make build              # Build binary
make dashboard-build    # Build dashboard for production

# Database options
make run                # MongoDB (requires MongoDB server)
make run_mysql          # MySQL (requires .env config)

# Testing
make test               # Run Go tests
make e2e-test          # End-to-end tests

# Utilities
make clean             # Clean build artifacts
make fmt               # Format Go code
make help              # Show all commands
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
- **10x Faster** - Go performance vs Node.js
- **Zero Dependencies** - SQLite built-in, no MongoDB setup required
- **Modern Dashboard** - React 18 with Chakra UI
- **Hot Reload** - For both JavaScript AND Go events
- **JWT Authentication** - Modern token-based auth
- **Multi-Database** - SQLite, MongoDB, MySQL support
- **Production Ready** - Built for scale with proper error handling

## 🤝 Contributing

This project aims to maintain the simplicity of the original Deployd while leveraging Go's performance and modern web technologies. Contributions welcome!

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

<div align="center">
  <strong>Ready to build amazing APIs? Start with <code>make run_sqlite</code> and let Go-Deployd handle the rest! 🚀</strong>
</div>