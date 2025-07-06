# Claude AI Development Context

## Project Overview
go-deployd is a modern backend-as-a-service platform with event-driven architecture, real-time WebSocket support, and dual runtime support (Go + JavaScript).

## Event System - CRITICAL SPECIFICATION
**ALL JavaScript events MUST use the unified Run(context) pattern - see docs/event-specification.md**

### JavaScript Events Pattern (REQUIRED)
```javascript
// Synchronous event
function Run(context) {
    // Modify context.data directly
    context.data.newField = "value";
    context.log("Event message");
}

// Async event (with async/await support)
async function Run(context) {
    try {
        const user = await dpd.users.get(context.data.userId);
        context.data.author = user;
    } catch (err) {
        context.error("userId", "User not found");
    }
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

### EventContext Structure (JavaScript & Go)
Both JavaScript and Go events use the same context API:

#### JavaScript Context Properties & Methods:
```javascript
context.data      // Document being processed (mutable)
context.query     // Query parameters
context.me        // Current user object (null if not authenticated)
context.method    // HTTP method (GET, POST, etc.)
context.isRoot    // Admin privileges (boolean)

// Methods
context.cancel(message, statusCode)  // Stop execution with error
context.log(message, data)           // Logging
context.emit(event, data, room)      // WebSocket events  
context.error(field, message)        // Add validation error
context.hasErrors()                  // Check if errors exist
context.hide(field)                  // Remove field from response
```

#### Go EventContext Structure:
```go
type EventContext struct {
    Data     map[string]interface{} // Document being processed
    Query    map[string]interface{} // Query parameters
    Me       map[string]interface{} // Current user (if authenticated)
    Method   string                 // HTTP method (GET, POST, etc.)
    IsRoot   bool                   // Admin privileges
    Internal bool                   // Internal request flag
    Errors   map[string]string      // Validation errors
    Dpd      *InternalClient        // Internal API access
    
    // Methods
    Cancel(message string, statusCode int) // Stop processing with error
    Log(message string, data ...map[string]interface{}) // Logging
    Emit(event string, data interface{}, room ...string) // WebSocket events
    Error(field, message string)          // Add validation error
    Hide(field string)                    // Remove field from response
    HasErrors() bool                      // Check if validation errors exist
}
```

## Key Architecture
- **Collections**: Resources with config.json (database-backed or noStore)
- **NoStore Collections**: Event-driven endpoints with `"noStore": true` (no DB storage)
- **V8 JavaScript Engine**: Isolated execution with context pooling
- **Real-time**: WebSocket support for live updates
- **Dashboard**: React-based admin interface at /_dashboard/
- **Column Storage**: SQL databases can use `"useColumns": true` for better query performance

## Common Commands
- **Development**: `npm run dev` (starts both Go server and dashboard)
- **Testing**: `curl localhost:2403/collection/endpoint | jq`
- **Build**: `go build -o bin/go-deployd cmd/deployd/main.go`
- **Go Only**: `go run cmd/deployd/main.go -dev -db-type sqlite`

## Event Development Guidelines
1. Always use the unified Run(context) pattern
2. **NEVER add `package main` to Go events** - handled by compile_wrapper.go
3. Test events with curl after changes
4. Check server logs for debugging (development mode shows event logs)
5. JavaScript events are isolated per request (no data leakage)
6. Modify context.data for data changes in both JS and Go
7. Go events are compiled as plugins using the wrapper system
8. **ALWAYS return after ctx.Cancel()** to prevent further execution
9. JavaScript async functions are supported - use `async function Run(context)`
10. The dpd object dynamically creates collection proxies - no hardcoded list

## Internal API - Accessing Other Collections

The internal API allows events to make HTTP-like requests to other collections,
executing the full request pipeline including all events and validation.

### JavaScript Events (Async/Await)
Access other collections using `dpd` with promises:
```javascript
async function Run(context) {
    try {
        // GET requests
        const allUsers = await dpd.users.get();  // Get all users
        const user = await dpd.users.get("user123");  // Get user by ID
        const activeUsers = await dpd.users.get({ active: true });  // Query users
        
        // POST request (create)
        const newTodo = await dpd.todos.post({
            title: "New task",
            userId: context.me.id,
            completed: false
        });
        
        // PUT request (update)
        const updated = await dpd.todos.put("todo123", {
            completed: true
        });
        
        // DELETE request
        await dpd.todos.del("todo123");
    } catch (err) {
        context.error("operation", err.message);
    }
}
```

### Go Events
Access other collections using `ctx.Dpd`:
```go
func Run(ctx *EventContext) error {
    // GET requests
    allUsers, err := ctx.Dpd.Collection("users").Get("", nil)  // Get all
    user, err := ctx.Dpd.Collection("users").Get("user123", nil)  // By ID
    activeUsers, err := ctx.Dpd.Collection("users").Get("", map[string]interface{}{
        "active": true,
    })  // With query
    
    // POST request (create)
    newTodo, err := ctx.Dpd.Collection("todos").Post(map[string]interface{}{
        "title": "New task",
        "userId": ctx.Me["id"],
        "completed": false,
    })
    
    // PUT request (update)
    updated, err := ctx.Dpd.Collection("todos").Put("todo123", map[string]interface{}{
        "completed": true,
    })
    
    // DELETE request
    _, err = ctx.Dpd.Collection("todos").Delete("todo123")
    
    return err
}
```

Available methods:
- `get(id?)` / `Get()` - Get all documents or by ID
- `get(query)` / `Get()` - Query documents
- `post(data)` / `Post()` - Create new document
- `put(id, data)` / `Put()` - Update existing document
- `del(id)` / `Delete()` - Delete document

### Key Features
- Full HTTP semantics (GET, POST, PUT, DELETE)
- Executes all events (validate, beforeRequest, get, post, put, delete, aftercommit)
- Respects collection validation rules
- Works with both regular and noStore collections
- Returns same response as HTTP requests
- Honors `$skipEvents` parameter to bypass event execution when needed
- **Dynamic collection access** - dpd.anyCollectionName works automatically

### Event Execution
The internal API runs the complete event pipeline:
1. `beforeRequest` - Before any operation
2. `validate` - For POST/PUT operations
3. `get`/`post`/`put`/`delete` - Method-specific events
4. `afterCommit` - After successful database operations

To skip events (useful to avoid infinite loops):
```javascript
// JavaScript
await dpd.users.post({ name: "Admin", $skipEvents: true });

// Go
ctx.Dpd.Collection("users").Post(map[string]interface{}{
    "name": "Admin",
    "$skipEvents": true,
})
```

**Important**: When an event in collection A triggers operations on collection B, 
collection B's events will also run. Use `$skipEvents` to prevent infinite loops
or unwanted cascading effects.

## Best Practices for Event Handlers

### 1. Error Handling
```go
// Always return after Cancel
if !isValid {
    ctx.Cancel("Invalid data", 400)
    return nil  // CRITICAL: Must return
}

// Handle internal API errors gracefully
user, err := ctx.Dpd.Collection("users").Get(userId, nil)
if err != nil {
    // Don't fail the whole request, handle gracefully
    ctx.Log("User lookup failed", map[string]interface{}{"error": err.Error()})
    ctx.Data["author"] = map[string]interface{}{"error": "User not found"}
}
```

### 2. Input Validation
```go
// Type-safe validation with proper checks
title, ok := ctx.Data["title"].(string)
if !ok || strings.TrimSpace(title) == "" {
    ctx.Error("title", "Title is required")
}

// Check HasErrors before proceeding
if ctx.HasErrors() {
    return nil
}
```

### 3. Security Considerations
- **Always validate file paths** to prevent directory traversal
- **Check authentication** before sensitive operations
- **Validate content types** for file uploads
- **Use rate limiting** for expensive operations
- **Hide sensitive fields** using ctx.Hide()

### 4. Performance Tips
- **Avoid N+1 queries** - batch fetch related data
- **Use $limit** for large result sets
- **Cache frequently accessed data** in context
- **Use goroutines carefully** - V8 contexts are not thread-safe

### 5. Logging Best Practices
```go
// Structured logging with context
ctx.Log("Operation completed", map[string]interface{}{
    "userId": ctx.Me["id"],
    "action": "update",
    "duration": time.Since(start).Milliseconds(),
})
```

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
- `/internal/events/script_dpd.go` - Dynamic dpd object implementation
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

## Known Issues & Workarounds

### 1. CORS/Origin Checking
**Issue**: WebSocket CheckOrigin returns true for all origins
**Location**: `internal/server/server.go:160`
**Workaround**: In production, implement proper origin validation

### 2. Unimplemented Features
- **AWS SES**: Use SMTP for email sending

### 3. Common Pitfalls
- Forgetting to return after `ctx.Cancel()`
- Not handling async errors in JavaScript events
- Creating infinite loops with internal API calls
- Not using `$skipEvents` when needed
- Hardcoding collection names instead of dynamic access

## Development Tips
1. **Use `npm run dev`** for automatic recompilation
2. **Check compilation errors** in the terminal, not just the API response
3. **Use structured logging** for better debugging
4. **Test with curl** for quick iteration
5. **Monitor `metrics.json`** for performance data
6. **Use column storage** (`useColumns: true`) for better SQL query performance
7. **JavaScript events support async/await** - use it for cleaner code
8. **The dpd object is dynamic** - any collection name works automatically