# Authentication

Go-Deployd provides comprehensive authentication capabilities using JWT (JSON Web Tokens) for stateless authentication. This documentation covers the complete authentication flow, security features, and implementation examples.

## Table of Contents

- [JWT Authentication Flow](#jwt-authentication-flow)
  - [Overview](#overview)
  - [Step 1: Create User](#step-1-create-user-master-key-required)
  - [Step 2: Login and Get Token](#step-2-login-and-get-jwt-token)
  - [Step 3: Use Token for API Calls](#step-3-use-token-for-api-calls)
  - [Step 4: Get User Info with /auth/me](#step-4-get-user-info-with-authme)
- [Complete Example Script](#complete-example-script)
- [JWT Token Structure](#jwt-token-structure)
- [Security Features](#security-features)
- [JWT Token Management](#jwt-token-management)

## JWT Authentication Flow

### Overview

Go-Deployd uses JWT (JSON Web Tokens) for stateless authentication. This complete example shows how to create a user, login to get a JWT token, and use that token to access protected endpoints.

✅ Both master key and user/password authentication are fully supported with JWT tokens. You can authenticate with either a master key (for admin operations) or username/password (for user operations).

### Step 1: Create User (Master Key Required)

First, create a user using the master key:

```bash
curl -X POST "http://localhost:2403/_admin/auth/create-user" \
  -H "Content-Type: application/json" \
  -H "X-Master-Key: your_master_key_here" \
  -d '{
    "userData": {
      "username": "johndoe",
      "email": "john@example.com",
      "password": "securePassword123",
      "name": "John Doe",
      "role": "user"
    }
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "User created successfully",
  "user": {
    "id": "65f7a8b9c1234567890abcde",
    "username": "johndoe",
    "email": "john@example.com",
    "name": "John Doe",
    "role": "user",
    "createdAt": "2024-06-26T10:00:00Z"
  }
}
```

### Step 2: Login and Get JWT Token

Login with either master key OR username/password to get a JWT token:

#### Option A: Master Key Login (Root Privileges)

```bash
curl -X POST "http://localhost:2403/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "masterKey": "your_master_key_here"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiJyb290IiwidXNlcm5hbWUiOiJyb290IiwiaXNSb290Ijp0cnVlLCJleHAiOjE3MTk0ODk2MDB9.xyzabc123...",
  "expiresAt": 1719489600,
  "isRoot": true
}
```

#### Option B: Username/Password Login (User Privileges)

```bash
curl -X POST "http://localhost:2403/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "securePassword123"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI2NWY3YThiOWMxMjM0NTY3ODkwYWJjZGUiLCJ1c2VybmFtZSI6ImpvaG5kb2UiLCJpc1Jvb3QiOmZhbHNlLCJleHAiOjE3MTk0ODk2MDB9.abc123xyz...",
  "expiresAt": 1719489600,
  "isRoot": false
}
```

✅ Both master key and user/password authentication are fully supported!

### Step 3: Use Token for API Calls

Use the JWT token in the Authorization header for all subsequent API calls:

```bash
# Save the token from the login response
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

```bash
# Get all users
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:2403/users"

# Create a document
curl -X POST "http://localhost:2403/todos" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Complete JWT implementation",
    "completed": false
  }'

# Access admin endpoints
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:2403/_admin/collections"
```

### Step 4: Get User Info with /auth/me

Use the `/auth/me` endpoint to get information about the currently authenticated user:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:2403/auth/me"
```

**Response for root user:**
```json
{
  "id": "root",
  "username": "root",
  "isRoot": true
}
```

**Response for regular user:**
```json
{
  "id": "65f7a8b9c1234567890abcde",
  "username": "johndoe",
  "email": "john@example.com",
  "name": "John Doe",
  "role": "user",
  "createdAt": "2024-06-26T10:00:00Z",
  "updatedAt": "2024-06-26T10:00:00Z"
}
```

## Complete Example Script

Here's a complete bash script demonstrating the full authentication flow:

```bash
#!/bin/bash

# Configuration
SERVER_URL="http://localhost:2403"
MASTER_KEY="mk_your_master_key_here"

echo "1. Creating user..."
CREATE_RESPONSE=$(curl -s -X POST "$SERVER_URL/_admin/auth/create-user" \
  -H "Content-Type: application/json" \
  -H "X-Master-Key: $MASTER_KEY" \
  -d '{
    "userData": {
      "username": "testuser",
      "email": "test@example.com",
      "password": "testPassword123",
      "name": "Test User"
    }
  }')

echo "User created: $(echo $CREATE_RESPONSE | jq -r '.user.username')"

echo -e "\n2. Logging in with master key..."
LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"masterKey\": \"$MASTER_KEY\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo "Got JWT token (master key): ${TOKEN:0:50}..."

echo -e "\n2b. Alternative: Login with username/password..."
USER_LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testPassword123"
  }')

USER_TOKEN=$(echo $USER_LOGIN_RESPONSE | jq -r '.token')
echo "Got JWT token (user): ${USER_TOKEN:0:50}..."
echo "Using master key token for admin operations..."

echo -e "\n3. Using token to create a todo..."
TODO_RESPONSE=$(curl -s -X POST "$SERVER_URL/todos" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test todo from JWT",
    "completed": false
  }')

echo "Created todo: $(echo $TODO_RESPONSE | jq -r '.title')"

echo -e "\n4. Getting user info with /auth/me..."
ME_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "$SERVER_URL/auth/me")

echo "Current user: $(echo $ME_RESPONSE | jq '.')"

echo -e "\n5. Validating token..."
VALIDATE_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "$SERVER_URL/auth/validate")

echo "Token validation: $(echo $VALIDATE_RESPONSE | jq '.')"
```

## JWT Token Structure

The JWT tokens contain the following claims:

```json
{
  "userId": "root",           // User ID (or "root" for master key)
  "username": "root",         // Username
  "isRoot": true,            // Whether user has root privileges
  "exp": 1719489600,         // Expiration timestamp
  "iat": 1719403200          // Issued at timestamp
}
```

**Token Properties:**
- Tokens expire after 24 hours by default (configurable)
- Only minimal user data is stored in the token
- Full user data is fetched from the database when needed

## Security Features

Go-Deployd implements comprehensive security measures:

- ✅ **bcrypt password hashing** (cost 12)
- ✅ **JWT token authentication** with secure signing
- ✅ **Master key authentication** (96-char secure key)
- ✅ **File permissions** (600) for sensitive config
- ✅ **Role-based access control** (RBAC)
- ✅ **Document-level access filtering**
- ✅ **CORS protection**
- ✅ **Input validation and sanitization**

## JWT Token Management

Go-Deployd uses JWT (JSON Web Tokens) for stateless authentication. Tokens are validated on each request without requiring server-side session storage.

### JWT Token Properties

- **Expiration:** 24 hours (configurable via JWTExpiration setting)
- **Storage:** Client-side (localStorage, cookies, or environment variables)
- **Security:** HMAC-SHA256 signed with secret key
- **Claims:** User ID, username, isRoot flag, expiration time
- **Stateless:** No server-side storage required

### Token Validation

```bash
# Validate current token
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:2403/auth/validate"
```

### Using Tokens in Requests

```bash
# Standard Bearer token (recommended)
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:2403/api/endpoint"

# Alternative: Store in environment variable
export JWT_TOKEN="your_jwt_token_here"
curl -H "Authorization: Bearer $JWT_TOKEN" \
  "http://localhost:2403/api/endpoint"

# For CLI tools: Save to file
echo "your_jwt_token" > ~/.deployd-token
curl -H "Authorization: Bearer $(cat ~/.deployd-token)" \
  "http://localhost:2403/api/endpoint"
```

## Security Considerations

1. **Token Storage:** Store JWT tokens securely on the client side
2. **HTTPS:** Always use HTTPS in production to protect tokens in transit
3. **Token Expiration:** Tokens expire after 24 hours by default - implement token refresh if needed
4. **Master Key:** Keep master keys secure and rotate them regularly
5. **Password Policy:** Enforce strong passwords for user accounts
6. **Role-Based Access:** Use appropriate roles to limit user permissions

## Authentication Endpoints Summary

| Endpoint | Method | Purpose | Authentication |
|----------|--------|---------|----------------|
| `/auth/login` | POST | Login with master key or username/password | None |
| `/auth/me` | GET | Get current user info | JWT Token |
| `/auth/validate` | GET | Validate JWT token | JWT Token |
| `/_admin/auth/create-user` | POST | Create new user | Master Key |