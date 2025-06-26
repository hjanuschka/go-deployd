# go-deployd CLI

A command-line interface for interacting with go-deployd APIs using JWT authentication.

## Why JWT for CLI?

The CLI uses JWT (JSON Web Tokens) instead of session-based authentication because:

1. **Stateless**: No need to maintain session state between commands
2. **Portable**: Tokens can be easily stored in files or environment variables
3. **Scriptable**: Perfect for automation and CI/CD pipelines
4. **Standard**: Uses standard `Authorization: Bearer <token>` headers

## Installation

```bash
go install ./cmd/deployd-cli
```

## Usage

### Authentication

First, authenticate with your master key:

```bash
deployd-cli -cmd=login -master-key=mk_your_master_key_here
```

This will:
- Send a POST request to `/auth/login` with the master key
- Receive a JWT token valid for 24 hours
- Store the token in `~/.deployd-token`

### Making API Requests

Once authenticated, you can make API requests:

#### List all items in a collection
```bash
deployd-cli -cmd=get -resource=users
```

#### Get a specific item
```bash
deployd-cli -cmd=get -resource=users -id=123
```

#### Create a new item
```bash
deployd-cli -cmd=post -resource=users -data='{"name":"John","email":"john@example.com"}'
```

#### Update an item
```bash
deployd-cli -cmd=put -resource=users -id=123 -data='{"name":"John Doe"}'
```

#### Delete an item
```bash
deployd-cli -cmd=delete -resource=users -id=123
```

### Custom Server URL

By default, the CLI connects to `http://localhost:2403`. To use a different server:

```bash
deployd-cli -host=https://api.example.com -cmd=get -resource=users
```

## Token Management

- Tokens are stored in `~/.deployd-token` with 600 permissions
- Tokens expire after 24 hours (configurable on server)
- Re-run the login command to get a new token

## Scripting Examples

### Backup all users
```bash
#!/bin/bash
deployd-cli -cmd=get -resource=users > users-backup.json
```

### Import data from file
```bash
#!/bin/bash
cat users.json | jq -c '.[]' | while read user; do
    deployd-cli -cmd=post -resource=users -data="$user"
done
```

### Monitor API health
```bash
#!/bin/bash
if deployd-cli -cmd=get -resource=users >/dev/null 2>&1; then
    echo "API is healthy"
else
    echo "API is down"
    exit 1
fi
```

## Session vs JWT Comparison

| Feature | Session-Based | JWT-Based |
|---------|--------------|-----------|
| State Management | Server maintains session | Stateless tokens |
| Storage | Cookie files | Simple text file |
| Expiration | Server-side control | Token contains expiry |
| Scripting | Complex cookie handling | Simple header addition |
| Security | CSRF concerns | Bearer token standard |
| Performance | Database lookups | Self-contained validation |

## Future Enhancements

- [ ] Support for user/password authentication
- [ ] Token refresh mechanism
- [ ] Interactive mode with command history
- [ ] Output format options (JSON, CSV, Table)
- [ ] Batch operations support
- [ ] WebSocket support for real-time features