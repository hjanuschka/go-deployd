# Session vs JWT Authentication for CLI Tools

## Overview

When building a CLI tool for a REST API, the choice of authentication mechanism is crucial. This document compares session-based and JWT-based authentication for the go-deployd CLI.

## Session-Based Authentication in CLI

### How It Would Work
```bash
# 1. Login creates a session
curl -c cookies.txt -X POST http://localhost:2403/users/login \
  -d '{"username":"admin","password":"pass"}'

# 2. Subsequent requests send cookies
curl -b cookies.txt http://localhost:2403/users
```

### Challenges
1. **Cookie Management**: CLI tools must manually handle cookie storage
2. **State Persistence**: Sessions require server-side state storage
3. **Cross-Platform Issues**: Cookie file paths vary by OS
4. **Scripting Complexity**: Each script needs cookie file management
5. **Concurrency**: Multiple CLI instances need separate cookie files

### Implementation Example (Complex)
```go
// Session-based CLI would need:
type SessionCLI struct {
    cookieJar *cookiejar.Jar
    client    *http.Client
}

func (c *SessionCLI) login() error {
    // Complex cookie jar setup
    jar, _ := cookiejar.New(nil)
    c.client = &http.Client{Jar: jar}
    
    // Make login request
    resp, err := c.client.Post(...)
    
    // Save cookies to file (platform-specific)
    // Handle cookie expiration
    // Manage cookie file permissions
}
```

## JWT-Based Authentication in CLI

### How It Works
```bash
# 1. Login returns a token
TOKEN=$(curl -X POST http://localhost:2403/auth/login \
  -d '{"masterKey":"mk_xxx"}' | jq -r '.token')

# 2. Use token in headers
curl -H "Authorization: Bearer $TOKEN" http://localhost:2403/users
```

### Advantages
1. **Simplicity**: Just a string to store and send
2. **Stateless**: No server-side session management
3. **Portable**: Works identically across all platforms
4. **Scriptable**: Easy to use in automation
5. **Standard**: Industry-standard Bearer token format

### Implementation Example (Simple)
```go
// JWT-based CLI implementation:
type JWTCLI struct {
    token  string
    client *http.Client
}

func (c *JWTCLI) login(masterKey string) error {
    // Simple POST request
    resp, err := c.client.Post(...)
    
    // Extract token from response
    var auth AuthResponse
    json.NewDecoder(resp.Body).Decode(&auth)
    
    // Save token to file
    os.WriteFile("~/.token", []byte(auth.Token), 0600)
}
```

## Comparison Matrix

| Aspect | Session-Based | JWT-Based | Winner |
|--------|--------------|-----------|---------|
| **Setup Complexity** | High (cookie jars, storage) | Low (simple string) | JWT ✓ |
| **Cross-Platform** | Complex (cookie paths) | Simple (same everywhere) | JWT ✓ |
| **Scripting** | Difficult | Easy | JWT ✓ |
| **Security** | CSRF vulnerable | Bearer token standard | JWT ✓ |
| **State Management** | Server maintains | Client maintains | JWT ✓ |
| **Performance** | DB lookups | Self-contained | JWT ✓ |
| **Debugging** | Complex | Simple (visible token) | JWT ✓ |
| **Token Sharing** | Cookie files | Environment variables | JWT ✓ |

## Real-World Examples

### CI/CD Pipeline (JWT)
```yaml
# GitHub Actions example
steps:
  - name: Login to API
    run: |
      TOKEN=$(curl -X POST ${{ secrets.API_URL }}/auth/login \
        -d "{\"masterKey\":\"${{ secrets.MASTER_KEY }}\"}" | jq -r '.token')
      echo "API_TOKEN=$TOKEN" >> $GITHUB_ENV
  
  - name: Deploy
    run: |
      curl -H "Authorization: Bearer ${{ env.API_TOKEN }}" \
        -X POST ${{ secrets.API_URL }}/deploy
```

### Session-based would be much more complex:
```yaml
# Would require cookie file management, harder to debug
# Platform-specific cookie handling
# State persistence between steps
```

## Security Considerations

### JWT Advantages
1. **No CSRF**: Tokens aren't automatically sent like cookies
2. **Explicit**: Must be explicitly included in requests
3. **Expiration**: Self-contained expiration time
4. **Revocation**: Can implement token blacklists if needed

### Session Disadvantages
1. **CSRF Risk**: Cookies sent automatically
2. **Session Hijacking**: Cookie theft risks
3. **Complexity**: Secure cookie attributes needed

## Conclusion

For CLI tools, **JWT authentication is clearly superior** to session-based authentication because:

1. **Developer Experience**: Much simpler to implement and use
2. **Automation-Friendly**: Perfect for scripts and CI/CD
3. **Standard Practice**: Industry standard for API authentication
4. **Cross-Platform**: Works identically everywhere
5. **Debugging**: Tokens are visible and inspectable

The go-deployd CLI implementation with JWT provides a clean, secure, and developer-friendly experience that would be much more complex with session-based authentication.