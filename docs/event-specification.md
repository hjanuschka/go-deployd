# Event System Specification

## Overview
The event system supports both Go and JavaScript runtimes with a unified pattern for consistency and maintainability.

## Event Types
- `get.js` / `get.go` - GET requests
- `post.js` / `post.go` - POST requests  
- `put.js` / `put.go` - PUT requests
- `delete.js` / `delete.go` - DELETE requests
- `validate.js` / `validate.go` - Data validation
- `beforerequest.js` / `beforerequest.go` - Before request processing
- `aftercommit.js` / `aftercommit.go` - After data commit

## Unified JavaScript Pattern

### Required Function: Run(context)
All JavaScript events MUST use the `Run(context)` function pattern:

```javascript
function Run(context) {
    // Your event logic here
    context.data.newField = "computed value";
    context.log("Event executed");
}
```

### Context Object
The `context` parameter provides:
- `context.data` - The document/request data (mutable)
- `context.log(message)` - Logging function
- `context.query` - Query parameters
- `context.me` - Current user (if authenticated)
- `context.isRoot` - Admin privileges flag

### Data Modification
- Modify `context.data` properties directly
- Changes are automatically extracted after Run() execution
- No need to return or assign to `this.*` properties

### Example JavaScript Event
```javascript
// resources/collection-name/get.js
function Run(context) {
    context.log("Processing GET request");
    
    // Add computed fields
    context.data.status = context.data.completed ? "Done" : "Pending";
    context.data.processedAt = new Date().toISOString();
    
    // Add metadata
    context.data.processedBy = "JavaScript Run(context)";
}
```

## Go Pattern

### Required Function: Run(context *context.Context) error
All Go events MUST use the `Run` function pattern:

```go
func Run(context *context.Context) error {
    // Your event logic here
    context.Data["newField"] = "computed value"
    context.Log("Event executed")
    return nil
}
```

### Example Go Event
```go
// resources/collection-name/get.go
package main

import (
    "github.com/hjanuschka/go-deployd/internal/context"
)

func Run(ctx *context.Context) error {
    ctx.Log("Processing GET request")
    
    // Add computed fields
    if completed, ok := ctx.Data["completed"].(bool); ok {
        if completed {
            ctx.Data["status"] = "Done"
        } else {
            ctx.Data["status"] = "Pending"
        }
    }
    
    // Add metadata
    ctx.Data["processedBy"] = "Go Run(context)"
    
    return nil
}
```

## JavaScript Event Requirements

### ONLY Supported Pattern
**Required pattern:**
```javascript
// ONLY supported pattern - Run(context) function
function Run(context) {
    context.data.status = "computed";
    context.data.processedAt = new Date();
}
```

**Deprecated patterns (NO LONGER SUPPORTED):**
```javascript
// DON'T USE - Legacy this.* pattern removed
this.status = "computed";
this.processedAt = new Date();
```

### Benefits of Unified Pattern
1. **Consistency** - Same pattern across Go and JavaScript
2. **Isolation** - No global variable pollution
3. **Clarity** - Explicit function signature and context
4. **Maintainability** - Easier to debug and modify
5. **Type Safety** - Better error detection

## NoStore Collections
Collections with `"noStore": true` in config.json are event-driven endpoints without database storage:
- Process incoming requests through events
- Return computed responses
- No data persistence
- Ideal for calculators, converters, APIs

## Error Handling
- JavaScript: Exceptions are caught and logged
- Go: Return error from Run() function
- Context isolation prevents data leakage between requests

## Testing
Test events with curl:
```bash
# Test JavaScript event
curl localhost:2403/collection-js/endpoint

# Test Go event  
curl localhost:2403/collection-go/endpoint
```

## Implementation Notes
- V8 engine executes JavaScript with context isolation
- Function wrapping preserves Run() in global scope
- Data extraction only from context.data (no global scope fallback)
- Context clearing prevents request data bleeding
- Legacy this.* patterns are completely removed