# Admin API Documentation

The Admin API provides administrative functionality for managing your deployd server, including server information, collection management, and security settings. All admin endpoints require authentication via master key.

## Table of Contents

- [Authentication](#authentication)
- [Server Information](#server-information)
- [Collection Management](#collection-management)
  - [List Collections](#list-collections)
  - [Get Collection Details](#get-collection-details)
  - [Create Collection](#create-collection)
- [Security Settings Management](#security-settings-management)
  - [Get Security Settings](#get-security-settings)
  - [Update Security Settings](#update-security-settings)
  - [Validate Master Key](#validate-master-key)

## Authentication

All admin API endpoints require authentication using the master key. Include the master key in the request header:

```
X-Master-Key: your_master_key_here
```

## Server Information

Get information about the deployd server, including version, uptime, and database status.

### Endpoint
```
GET /_admin/info
```

### Request
```bash
curl -H "X-Master-Key: your_master_key_here" \
  "https://your-server.com/_admin/info"
```

### Response
```json
{
  "version": "1.0.0",
  "goVersion": "go1.21",
  "uptime": "2h 15m",
  "database": "Connected",
  "environment": "development"
}
```

## Collection Management

### List Collections

Retrieve a list of all collections in your deployd application.

#### Endpoint
```
GET /_admin/collections
```

#### Request
```bash
curl -H "X-Master-Key: your_master_key_here" \
  "https://your-server.com/_admin/collections"
```

#### Response
```json
[
  {
    "name": "users",
    "properties": {
      "username": {"type": "string", "required": true},
      "email": {"type": "string", "required": true}
    }
  },
  {
    "name": "products",
    "properties": {
      "name": {"type": "string", "required": true},
      "price": {"type": "number", "required": true}
    }
  }
]
```

### Get Collection Details

Get detailed information about a specific collection, including its schema and properties.

#### Endpoint
```
GET /_admin/collections/{collection_name}
```

#### Request
```bash
curl -H "X-Master-Key: your_master_key_here" \
  "https://your-server.com/_admin/collections/users"
```

#### Response
```json
{
  "name": "users",
  "properties": {
    "username": {"type": "string", "required": true},
    "email": {"type": "string", "required": true},
    "createdAt": {"type": "date", "default": "now"}
  },
  "events": {
    "beforeCreate": true,
    "afterCreate": false
  }
}
```

### Create Collection

Create a new collection with specified properties and schema.

#### Endpoint
```
POST /_admin/collections/{collection_name}
```

#### Request
```bash
curl -X POST "https://your-server.com/_admin/collections/products" \
  -H "X-Master-Key: your_master_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "name": {"type": "string", "required": true},
    "price": {"type": "number", "required": true},
    "category": {"type": "string", "default": "general"}
  }'
```

#### Response
```json
{
  "success": true,
  "message": "Collection 'products' created successfully",
  "collection": {
    "name": "products",
    "properties": {
      "name": {"type": "string", "required": true},
      "price": {"type": "number", "required": true},
      "category": {"type": "string", "default": "general"}
    }
  }
}
```

## Security Settings Management

### Get Security Settings

Retrieve current security settings for the deployd server.

#### Endpoint
```
GET /_admin/settings/security
```

#### Request
```bash
curl -H "X-Master-Key: your_master_key_here" \
  "https://your-server.com/_admin/settings/security"
```

#### Response
```json
{
  "jwtExpiration": "24h",
  "allowRegistration": true,
  "requireEmailVerification": false,
  "maxLoginAttempts": 5,
  "lockoutDuration": "15m"
}
```

### Update Security Settings

Update security settings for the deployd server.

#### Endpoint
```
PUT /_admin/settings/security
```

#### Request
```bash
curl -X PUT "https://your-server.com/_admin/settings/security" \
  -H "X-Master-Key: your_master_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "jwtExpiration": "24h",
    "allowRegistration": false
  }'
```

#### Response
```json
{
  "success": true,
  "message": "Security settings updated successfully",
  "settings": {
    "jwtExpiration": "24h",
    "allowRegistration": false,
    "requireEmailVerification": false,
    "maxLoginAttempts": 5,
    "lockoutDuration": "15m"
  }
}
```

### Validate Master Key

Validate a master key to ensure it's correct and active.

#### Endpoint
```
POST /_admin/auth/validate-master-key
```

#### Request
```bash
curl -X POST "https://your-server.com/_admin/auth/validate-master-key" \
  -H "Content-Type: application/json" \
  -d '{
    "masterKey": "your_master_key_here"
  }'
```

#### Response
```json
{
  "valid": true,
  "message": "Master key is valid"
}
```

## Additional Admin Endpoints

### Dashboard Login

Authenticate for dashboard access.

#### Endpoint
```
POST /_admin/auth/dashboard-login
```

#### Request
```bash
curl -X POST "https://your-server.com/_admin/auth/dashboard-login" \
  -H "Content-Type: application/json" \
  -d '{
    "masterKey": "your_master_key_here"
  }'
```

### System Login

Authenticate for system-level access.

#### Endpoint
```
POST /_admin/auth/system-login
```

#### Request
```bash
curl -X POST "https://your-server.com/_admin/auth/system-login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your_password"
  }'
```

### Create User (Admin)

Create a new user account via admin API.

#### Endpoint
```
POST /_admin/auth/create-user
```

#### Request
```bash
curl -X POST "https://your-server.com/_admin/auth/create-user" \
  -H "X-Master-Key: your_master_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "email": "user@example.com",
    "password": "securepassword"
  }'
```

## Error Responses

All admin API endpoints return appropriate HTTP status codes and error messages:

- `200 OK` - Request successful
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Missing or invalid master key
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

### Error Response Format
```json
{
  "error": true,
  "message": "Description of the error",
  "code": "ERROR_CODE"
}
```

## Security Considerations

1. **Master Key Protection**: Always keep your master key secure and never expose it in client-side code
2. **HTTPS Only**: Always use HTTPS in production to protect the master key in transit
3. **Access Control**: Restrict access to admin endpoints to authorized personnel only
4. **Monitoring**: Monitor admin API usage for security purposes
5. **Rate Limiting**: Consider implementing rate limiting for admin endpoints to prevent abuse

## Best Practices

1. **Regular Security Audits**: Regularly review and update security settings
2. **Backup Collections**: Use collection management endpoints to backup your data schema
3. **Environment Separation**: Use different master keys for development, staging, and production
4. **Logging**: Monitor admin API access logs for security and debugging purposes
5. **Version Control**: Keep track of collection schema changes through version control