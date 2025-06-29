# Claude AI Development Context

## Project Overview
go-deployd is a modern backend-as-a-service platform with event-driven architecture, real-time WebSocket support, and dual runtime support (Go + JavaScript).

## Event System - CRITICAL SPECIFICATION
**ALL JavaScript events MUST use the unified Run(context) pattern - see docs/event-specification.md**

### JavaScript Events Pattern (REQUIRED)
```javascript
function Run(context) {
    // Modify context.data directly
    context.data.newField = "value";
    context.log("Event message");
}
```

### Go Events Pattern (REQUIRED)  
```go
func Run(ctx *EventContext) error {
    ctx.Data["newField"] = "value"
    ctx.Log("Event message")
    return nil
}
```

**DO NOT use legacy `this.*` patterns in JavaScript - they are no longer supported.**

## Go Event Compilation System - CRITICAL
**Go events use a special compilation wrapper system - DO NOT add `package main`**

### Event Compilation Process
1. Go events are compiled using `internal/events/compile_wrapper.go`
2. The wrapper automatically adds `package main` and required imports
3. User Go events should ONLY contain:
   - Import statements (if needed)
   - The `Run(ctx *EventContext) error` function
   - Any helper functions

### EventContext Structure
```go
type EventContext struct {
    Data     map[string]interface{} // Document being processed
    Query    map[string]interface{} // Query parameters
    Me       map[string]interface{} // Current user (if authenticated)
    Method   string                 // HTTP method (GET, POST, etc.)
    IsRoot   bool                   // Admin privileges
    Internal bool                   // Internal request flag
    Errors   map[string]string      // Validation errors
    
    // Methods
    Cancel(message string, statusCode int) // Stop processing with error
    Log(message string, data ...map[string]interface{}) // Logging
    Emit(event string, data interface{}, room ...string) // WebSocket events
    Error(field, message string)          // Add validation error
    Hide(field string)                    // Remove field from response
}
```

## Key Architecture
- **Collections**: Resources with config.json (database-backed or noStore)
- **NoStore Collections**: Event-driven endpoints with `"noStore": true` (no DB storage)
- **V8 JavaScript Engine**: Isolated execution with context pooling
- **Real-time**: WebSocket support for live updates
- **Dashboard**: React-based admin interface at /_dashboard/

## Common Commands
- **Development**: `npm run dev` (starts both Go server and dashboard)
- **Testing**: `curl localhost:2403/collection/endpoint | jq`
- **Build**: `go build -o bin/go-deployd cmd/main.go`

## Event Development Guidelines
1. Always use the unified Run(context) pattern
2. **NEVER add `package main` to Go events** - handled by compile_wrapper.go
3. Test events with curl after changes
4. Check server logs for debugging: `tail -f test-server.log`
5. JavaScript events are isolated per request (no data leakage)
6. Modify context.data for data changes in both JS and Go
7. Go events are compiled as plugins using the wrapper system

## Go Event Examples
### Correct Go Event Structure
```go
// resources/collection/validate.go
import (
    "strings"
)

func Run(ctx *EventContext) error {
    title, ok := ctx.Data["title"].(string)
    if !ok || strings.TrimSpace(title) == "" {
        ctx.Cancel("Title is required", 400)
        return nil
    }
    ctx.Log("Validation passed")
    return nil
}
```

### Files Collection Events
Built-in files collection supports Go events for file processing:
```go
// resources/files/beforerequest.go
func Run(ctx *EventContext) error {
    if ctx.Method != "POST" {
        return nil // Only validate uploads
    }
    
    // Check file extension from headers
    contentType := ctx.Query["content-type"]
    if contentType == "application/exe" {
        ctx.Cancel("Executable files not allowed", 400)
        return nil
    }
    
    return nil
}
```

## File Structure
- `/resources/collection-name/` - Collection definitions
- `/resources/collection-name/config.json` - Collection configuration
- `/resources/collection-name/*.js` - JavaScript events
- `/resources/collection-name/*.go` - Go events (NO package main!)
- `/internal/events/` - Event system implementation
- `/internal/events/compile_wrapper.go` - Go event compilation wrapper
- `/internal/events/compile.go` - Event compilation logic
- `/resources/files/` - Built-in file storage with Go events
- `/dashboard/` - Admin interface

## Built-in Collections
### Files Collection (`/resources/files/`)
- File upload/download with Local, S3, MinIO backends
- Go events for validation: `beforerequest.go`, `post.go`, `get.go`, `delete.go`
- Real-time WebSocket notifications
- Automatic metadata extraction

### Users Collection (`/resources/users/`)
- JWT authentication system
- User registration and login
- Password hashing with bcrypt

## NoStore Collections
Perfect for API endpoints without database storage:
```json
{
  "type": "Collection",
  "noStore": true,
  "properties": {}
}
```

See calculator-js and calculator-go as reference implementations.