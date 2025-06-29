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
func Run(ctx *context.Context) error {
    ctx.Data["newField"] = "value"
    ctx.Log("Event message")
    return nil
}
```

**DO NOT use legacy `this.*` patterns in JavaScript - they are no longer supported.**

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
2. Test events with curl after changes
3. Check server logs for debugging: `tail -f test-server.log`
4. JavaScript events are isolated per request (no data leakage)
5. Modify context.data for data changes in both JS and Go

## File Structure
- `/resources/collection-name/` - Collection definitions
- `/resources/collection-name/config.json` - Collection configuration
- `/resources/collection-name/*.js` - JavaScript events
- `/resources/collection-name/*.go` - Go events
- `/internal/events/` - Event system implementation
- `/dashboard/` - Admin interface

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