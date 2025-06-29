# Event Collections (noStore)

Event Collections provide event-driven endpoints without data storage, similar to the original [dpd-event](https://github.com/deployd/dpd-event) module. These are perfect for creating webhooks, API proxies, data transformations, or custom business logic endpoints.

## Overview

Event Collections are special collections that:
- **Don't store data** in the database
- **Execute event scripts** when receiving HTTP requests  
- **Provide enhanced context** with URL routing and helper functions
- **Support all HTTP methods** (GET, POST, PUT, DELETE)
- **Offer dpd-event compatibility** for easy migration

## Configuration

Add `"noStore": true` to your collection's `config.json`:

```json
{
  "noStore": true,
  "eventConfig": {
    "get": {"enabled": true},
    "post": {"enabled": true},
    "put": {"enabled": true}, 
    "delete": {"enabled": true},
    "beforerequest": {"enabled": true}
  }
}
```

## Event Context

Event scripts have access to enhanced context variables:

### Core Variables

- **`url`** - Request path without collection base (e.g., `/user/123` → `/123`)
- **`parts`** - URL segments as array (e.g., `/user/123/profile` → `["123", "profile"]`)
- **`query`** - Query parameters object (e.g., `?name=john&age=25`)
- **`body`** - Request body object (for POST/PUT requests)

### Helper Functions

- **`setResult(result)`** - Set the response body (string or object)
- **`getHeader(name)`** - Get request header value (case insensitive)
- **`setHeader(name, value)`** - Set response header
- **`setStatusCode(code)`** - Set HTTP status code

## Usage Examples

### Simple GET Endpoint

**Collection:** `/api-status`
**Event:** `get.js`

```javascript
// GET /api-status
setResult({
  status: "healthy",
  timestamp: new Date().toISOString(),
  version: "1.0.0"
});
```

**Response:**
```json
{
  "status": "healthy", 
  "timestamp": "2024-01-15T10:30:00.000Z",
  "version": "1.0.0"
}
```

### URL Routing with Parts

**Collection:** `/calculator`
**Event:** `get.js`

```javascript
// GET /calculator/add/5/3
if (parts[0] === "add" && parts.length === 3) {
  const a = parseInt(parts[1]);
  const b = parseInt(parts[2]);
  setResult({
    operation: "add",
    operands: [a, b],
    result: a + b
  });
} else {
  setStatusCode(400);
  setResult({error: "Invalid operation. Use: /calculator/add/num1/num2"});
}
```

### Query Parameters

**Collection:** `/search`
**Event:** `get.js`

```javascript
// GET /search?q=deployd&type=docs&limit=10
const searchQuery = query.q;
const type = query.type || "all";
const limit = parseInt(query.limit) || 50;

if (!searchQuery) {
  setStatusCode(400);
  setResult({error: "Query parameter 'q' is required"});
  return;
}

setResult({
  query: searchQuery,
  type: type,
  limit: limit,
  results: [
    // Simulate search results
    {title: "Go-Deployd Documentation", url: "/docs"}
  ]
});
```

### Webhook Handler

**Collection:** `/webhook`
**Event:** `post.js`

```javascript
// POST /webhook/github
const eventType = getHeader("x-github-event");
const signature = getHeader("x-hub-signature-256");

if (!eventType) {
  setStatusCode(400);
  setResult({error: "Missing GitHub event type header"});
  return;
}

// Process webhook payload
console.log(`Received ${eventType} event:`, body);

// Forward to other systems
if (eventType === "push") {
  // Trigger deployment
  setResult({
    status: "deployment_triggered",
    commit: body.head_commit?.id,
    repository: body.repository?.name
  });
} else {
  setResult({status: "event_processed", type: eventType});
}
```

### API Proxy with Headers

**Collection:** `/proxy`
**Event:** `get.js`

```javascript
// GET /proxy/external-api/users
const apiKey = getHeader("x-api-key");

if (!apiKey) {
  setStatusCode(401);
  setResult({error: "API key required"});
  return;
}

// Set CORS headers
setHeader("Access-Control-Allow-Origin", "*");
setHeader("Content-Type", "application/json");

// Proxy to external API (simplified example)
setResult({
  message: "Proxied request",
  path: url,
  authenticated: true
});
```

### Error Handling

**Collection:** `/validator`
**Event:** `post.js`

```javascript
// POST /validator/email
const email = body.email;

if (!email) {
  setStatusCode(400);
  setResult({
    error: "Email is required",
    code: "MISSING_EMAIL"
  });
  return;
}

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
if (!emailRegex.test(email)) {
  setStatusCode(422);
  setResult({
    error: "Invalid email format",
    code: "INVALID_EMAIL",
    email: email
  });
  return;
}

setResult({
  valid: true,
  email: email,
  message: "Email is valid"
});
```

## Advanced Features

### BeforeRequest Event

The `beforerequest` event runs before all other events:

```javascript
// beforerequest.js - Authentication/validation
const authToken = getHeader("authorization");

if (!authToken && !query.public) {
  setStatusCode(401);
  setResult({error: "Authentication required"});
  cancel(); // Stop processing
}

// Add user context for other events
this.user = {id: "user123", role: "admin"};
```

### Content Negotiation

```javascript
// get.js - Support multiple formats
const acceptHeader = getHeader("accept");

const data = {
  message: "Hello World",
  timestamp: Date.now()
};

if (acceptHeader && acceptHeader.includes("application/xml")) {
  setHeader("Content-Type", "application/xml");
  setResult(`<response><message>${data.message}</message></response>`);
} else if (acceptHeader && acceptHeader.includes("text/plain")) {
  setHeader("Content-Type", "text/plain");
  setResult(`${data.message} at ${data.timestamp}`);
} else {
  setHeader("Content-Type", "application/json");
  setResult(data);
}
```

## Client Usage

### HTTP Requests

```bash
# GET request with query parameters
curl "http://localhost:2403/calculator/add/10/5"

# POST request with JSON body
curl -X POST http://localhost:2403/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{"repository":{"name":"my-repo"}}'
```

### dpd.js Client

```javascript
// GET request
dpd.calculator.get("add/10/5", function(result, error) {
  console.log("Result:", result); // {operation: "add", result: 15}
});

// POST request  
dpd.webhook.post("github", {
  repository: {name: "my-repo"}
}, function(result, error) {
  console.log("Webhook result:", result);
});
```

## Best Practices

### 1. Input Validation

Always validate inputs in event scripts:

```javascript
if (!body.email || typeof body.email !== 'string') {
  setStatusCode(400);
  setResult({error: "Valid email required"});
  return;
}
```

### 2. Error Handling

Use appropriate HTTP status codes:

```javascript
// Bad Request
setStatusCode(400);

// Unauthorized  
setStatusCode(401);

// Not Found
setStatusCode(404);

// Unprocessable Entity
setStatusCode(422);

// Internal Server Error
setStatusCode(500);
```

### 3. Security

Validate authentication and authorization:

```javascript
const apiKey = getHeader("x-api-key");
if (apiKey !== "your-secret-key") {
  setStatusCode(401);
  setResult({error: "Invalid API key"});
  return;
}
```

### 4. Logging

Use console.log for debugging (appears in server logs):

```javascript
console.log("Processing request:", {
  method: "POST",
  url: url,
  parts: parts,
  hasBody: !!body
});
```

## Migration from dpd-event

Event Collections are designed for compatibility with dpd-event:

1. **Add `"noStore": true`** to collection config
2. **Keep existing event scripts** - they should work unchanged
3. **Update collection configuration** - enable events as needed
4. **Test functionality** - verify URL routing and helper functions work

## Comparison

| Feature | Regular Collection | Event Collection |
|---------|-------------------|------------------|
| Data Storage | ✅ Yes | ❌ No |
| Event Scripts | ✅ Yes | ✅ Yes |
| URL Routing | ❌ Basic | ✅ Advanced |
| Helper Functions | ❌ Limited | ✅ Full dpd-event API |
| Use Cases | CRUD Operations | Webhooks, APIs, Logic |

Event Collections provide the flexibility of dpd-event while maintaining compatibility with the Go-Deployd ecosystem.