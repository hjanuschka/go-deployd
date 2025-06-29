# Events System

The Go-Deployd events system provides powerful server-side hooks for implementing business logic, validation, and data transformations. Events can be written in either JavaScript or Go.

## Table of Contents

- [Event Lifecycle](#event-lifecycle)
  - [Validate Events](#validate-events)
  - [BeforeCommit Events](#beforecommit-events)
  - [AfterCommit Events](#aftercommit-events)
- [Event Context](#event-context)
  - [Available Variables](#available-variables)
  - [Available Functions](#available-functions)
- [JavaScript Events](#javascript-events)
  - [Basic Validation Example](#basic-validation-example)
  - [Using npm Modules](#using-npm-modules)
  - [Logging and Debugging](#logging-and-debugging)
- [Go Events](#go-events)
  - [Basic Validation Example](#basic-validation-example-1)
  - [Using Third-Party Packages](#using-third-party-packages)
  - [Logging and Debugging](#logging-and-debugging-1)
- [Bypassing Events](#bypassing-events)
- [Performance Considerations](#performance-considerations)

## Event Lifecycle

### Validate Events

Run before any data modifications to validate incoming data and enforce business rules.

```javascript
// validate.js
if (!this.title || this.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}

if (this.priority > 10) {
  error('priority', 'Priority cannot exceed 10');
}
```

### BeforeCommit Events

Run after validation but before database commit. Use for data transformations and modifications.

```javascript
// beforecommit.js
// Auto-generate slug from title
if (this.title && !this.slug) {
  this.slug = this.title.toLowerCase().replace(/\s+/g, '-');
}

// Set timestamps
if (!this.createdAt) {
  this.createdAt = new Date();
}
this.updatedAt = new Date();
```

### AfterCommit Events

Run after successful database commit. Use for side effects, notifications, and response modifications.

```javascript
// aftercommit.js
// Send notification
if (this.priority >= 8) {
  emit('urgent_task_created', {
    taskId: this.id,
    title: this.title,
    priority: this.priority
  });
}

// Modify response
setResponseData({
  ...this,
  message: 'Task created successfully!',
  timestamp: new Date()
});
```

## Event Context

### Available Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `this` | Current document data | `this.title`, `this.id` |
| `me` | Authenticated user | `me.username`, `me.role` |
| `query` | Request query parameters | `query.filter`, `query.limit` |
| `internal` | Internal API client | `internal.users.get()` |
| `ctx` | Request context | `ctx.req.headers` |
| `isRoot` | Master key authentication | `if (isRoot) { ... }` |

### Available Functions

| Function | Description | Example |
|----------|-------------|---------|
| `error(field, message)` | Add validation error | `error('email', 'Invalid email')` |
| `hide(field)` | Remove field from response | `hide('password')` |
| `protect(field)` | Remove field from data | `protect('internalId')` |
| `cancel(message, code)` | Cancel operation | `cancel('Forbidden', 403)` |
| `emit(event, data)` | Send real-time event | `emit('user_online', {userId: me.id})` |
| `setResponseData(data)` | Replace response (aftercommit) | `setResponseData({...this, extra: 'data'})` |

## JavaScript Events

JavaScript events run in a V8 engine with support for npm modules and ES6+ features.

### Basic Validation Example

```javascript
// validate.js
if (!this.title || this.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}

if (this.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(this.email)) {
  error('email', 'Please enter a valid email address');
}

// Hide sensitive fields
hide('password');
```

### Using npm Modules

```javascript
// post.js - Using external libraries
const bcrypt = require('bcrypt');
const uuid = require('uuid');

if (this.password) {
  // Hash password before saving
  this.password = bcrypt.hashSync(this.password, 10);
}

// Add unique ID
this.externalId = uuid.v4();

// Send welcome email (example)
const nodemailer = require('nodemailer');
// ... email setup and sending logic
```

### Logging and Debugging

JavaScript events have access to `deployd.log()` for structured logging that integrates with the server's logging system.

```javascript
// Basic logging
deployd.log("User action performed");

// Structured logging with data
deployd.log("Pet created", {
    name: data.name,
    species: data.species,
    user: me,
    timestamp: new Date()
});

// Conditional logging
if (me && me.role === 'admin') {
    deployd.log("Admin action", {
        action: "bulk_update",
        affectedDocs: updateCount,
        adminUser: me.username
    });
}
```

💡 **Note**: Logging is automatically disabled in production mode for performance. Logs appear in server output with source identification (e.g., "js:todos").

### Available Global Functions

- `deployd.log(message, data)` - Structured logging (development only)
- `error(field, message)` - Add validation error
- `hide(field)` - Remove field from response
- `protect(field)` - Remove field from data
- `cancel(message, statusCode)` - Cancel operation
- `isMe(userId)` - Check if user owns resource

## Go Events

Go events are compiled as plugins and offer better performance for complex logic. They support any Go module available on the Go module proxy.

### Basic Validation Example

```go
// validate.go
package main

import (
    "strings"
    "regexp"
)

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Validate title
    if title, ok := eventCtx.Data["title"].(string); !ok || len(title) < 3 {
        eventCtx.Error("title", "Title must be at least 3 characters")
    }
    
    // Validate email format
    if email, ok := eventCtx.Data["email"].(string); ok && email != "" {
        emailRegex := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
        if !emailRegex.MatchString(email) {
            eventCtx.Error("email", "Please enter a valid email address")
        }
    }
    
    // Hide sensitive field
    eventCtx.Hide("password")
    
    return nil
}

var EventHandler = &EventHandler{}
```

### Using Third-Party Packages

```go
// post.go - Using external libraries
package main

import (
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "golang.org/x/crypto/bcrypt"
)

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Hash password if provided
    if password, ok := eventCtx.Data["password"].(string); ok && password != "" {
        hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
            eventCtx.Error("password", "Failed to process password")
            return nil
        }
        eventCtx.Data["password"] = string(hashed)
    }
    
    // Add unique external ID
    eventCtx.Data["externalId"] = uuid.New().String()
    
    // Handle decimal calculations
    if priceStr, ok := eventCtx.Data["price"].(string); ok {
        price, err := decimal.NewFromString(priceStr)
        if err == nil {
            tax := price.Mul(decimal.NewFromFloat(0.1)) // 10% tax
            eventCtx.Data["taxAmount"] = tax.String()
            eventCtx.Data["totalPrice"] = price.Add(tax).String()
        }
    }
    
    return nil
}

var EventHandler = &EventHandler{}
```

### Logging and Debugging

Go events have access to `deployd.Log()` for structured logging that integrates with the server's logging system.

```go
// Basic logging
deployd.Log("User action performed")

// Structured logging with data
deployd.Log("Pet created", map[string]interface{}{
    "name":      ctx.Data["name"],
    "species":   ctx.Data["species"],
    "user":      ctx.Me,
    "timestamp": time.Now(),
})

// Conditional logging
if ctx.IsRoot {
    deployd.Log("Admin action", map[string]interface{}{
        "action":       "bulk_update",
        "affectedDocs": updateCount,
        "adminUser":    ctx.Me["username"],
    })
}
```

💡 **Note**: Logging is automatically disabled in production mode for performance. Logs appear in server output with source identification (e.g., "go:todos").

### Available EventContext Methods

- `deployd.Log(message, data)` - Structured logging (development only)
- `Error(field, message)` - Add validation error
- `Hide(field)` - Remove field from response
- `Protect(field)` - Remove field from data
- `Cancel(message, statusCode)` - Cancel operation
- `IsMe(userId)` - Check if user owns resource
- `HasErrors()` - Check if validation errors exist

## Bypassing Events

When using the master key for administrative operations, you can bypass all events using the special `$skipEvents` parameter. This is useful for data migrations, bulk operations, or emergency fixes.

### Using $skipEvents in Request Body

```javascript
// POST/PUT request with $skipEvents in payload
var payload = {
  userId: user_id,
  title: "Admin Created Item",
  $skipEvents: true
};

fetch("/api/collection", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + masterKey
  },
  body: JSON.stringify(payload)
});
```

### Using $skipEvents as Query Parameter

```bash
# GET request bypassing events
curl -X GET "http://localhost:2403/users?$skipEvents=true" \
  -H "Authorization: Bearer ${MASTER_KEY}"

# POST request bypassing events  
curl -X POST "http://localhost:2403/users?$skipEvents=true" \
  -H "Authorization: Bearer ${MASTER_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "role": "administrator"}'
```

### Security Notes

- ⚠️ Only works with valid master key authentication
- ⚠️ Bypasses ALL events (validate, post, put, get)
- ⚠️ Use carefully - no validation or business logic will run
- ✅ Ideal for administrative data operations and migrations

## Performance Considerations

### Event Performance
- **Go events**: ~50-100x faster than JavaScript for CPU-intensive tasks
- **JavaScript events**: Better for simple validations and npm ecosystem
- Event compilation happens once at startup or file change
- Use Go events for complex business logic and calculations

### Best Practices
1. Use JavaScript for simple validations and when you need npm packages
2. Use Go for complex calculations, heavy processing, or performance-critical paths
3. Keep event logic focused and minimal
4. Use logging judiciously - it's disabled in production
5. Cache expensive calculations when possible
6. Consider async operations for non-critical side effects