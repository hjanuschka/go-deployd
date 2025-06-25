# MySQL Support for go-deployd

This document describes the MySQL support that has been added to go-deployd, including setup, configuration, and testing.

## Overview

go-deployd now supports MySQL as a database backend alongside SQLite and MongoDB. The MySQL implementation provides:

- Full CRUD operations with MongoDB-style query syntax
- JSON document storage using MySQL's native JSON column type
- Connection pooling and UTF8MB4 character set support
- Complete compatibility with existing go-deployd APIs
- Environment-based configuration for secure credential management

## Quick Start

### 1. Configuration

Copy the example environment file and configure your MySQL settings:

```bash
cp .env.example .env
```

Edit `.env` with your MySQL configuration:

```bash
# MySQL Database Configuration
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=your_username
MYSQL_PASS=your_password
MYSQL_DB=your_database

# Server Configuration
SERVER_PORT=2403
DEVELOPMENT_MODE=true
```

**Important**: The `.env` file is already in `.gitignore` to prevent credential exposure.

### 2. Running with MySQL

Use the environment-based MySQL runner script:

```bash
# Using .env file configuration
./scripts/run_mysql.sh

# Check configuration without starting
./scripts/run_mysql.sh --check-config

# Build binary only
./scripts/run_mysql.sh --build-only
```

Or run directly with command-line arguments:

```bash
./deployd -db-type=mysql -db-host=localhost -db-user=myuser -db-pass=mypass -db-name=mydatabase
```

### 3. Environment Variables

You can also use environment variables directly:

```bash
export MYSQL_HOST=192.168.1.100
export MYSQL_USER=deployd_user
export MYSQL_PASS=secure_password
export MYSQL_DB=my_deployd_db

./scripts/run_mysql.sh
```

## Environment Configuration

### Primary Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MYSQL_HOST` | MySQL server hostname/IP | `localhost` |
| `MYSQL_PORT` | MySQL server port | `3306` |
| `MYSQL_USER` | MySQL username | `root` |
| `MYSQL_PASS` | MySQL password | (empty) |
| `MYSQL_DB` | MySQL database name | `deployd` |
| `SERVER_PORT` | go-deployd server port | `2403` |
| `DEVELOPMENT_MODE` | Enable development mode | `true` |

### E2E Testing Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `E2E_MYSQL_HOST` | MySQL host for E2E tests | `localhost` |
| `E2E_MYSQL_USER` | MySQL user for E2E tests | `root` |
| `E2E_MYSQL_PASS` | MySQL password for E2E tests | (empty) |
| `E2E_MYSQL_DB` | MySQL database for E2E tests | `deployd_e2e_test_{timestamp}` |

## Database Setup

### Prerequisites

1. **MySQL Server** (5.7+ for JSON support, 8.0+ recommended)
2. **MySQL User** with database creation privileges
3. **Go 1.21+** for building go-deployd

### MySQL User Setup

Create a dedicated user for go-deployd:

```sql
-- Create user
CREATE USER 'deployd_user'@'%' IDENTIFIED BY 'secure_password';

-- Grant privileges (adjust as needed for production)
GRANT CREATE, DROP, SELECT, INSERT, UPDATE, DELETE ON *.* TO 'deployd_user'@'%';

-- For specific database only
GRANT ALL PRIVILEGES ON deployd.* TO 'deployd_user'@'%';

FLUSH PRIVILEGES;
```

### Database Creation

go-deployd will automatically create tables as needed, but you may want to create the database first:

```sql
CREATE DATABASE deployd 
  CHARACTER SET utf8mb4 
  COLLATE utf8mb4_unicode_ci;
```

## Features

### JSON Document Storage

MySQL implementation stores documents as JSON in a `data` column:

```sql
CREATE TABLE collection_name (
    id VARCHAR(255) PRIMARY KEY,
    data JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### MongoDB-Style Queries

All MongoDB-style operators are supported:

```javascript
// Equality
GET /users?name=john

// Comparison operators
GET /products?price={"$gte":50,"$lte":200}

// Regular expressions
GET /users?email={"$regex":".*@example.com"}

// Array operations
GET /tags?categories={"$in":["tech","programming"]}

// Sorting and pagination
GET /users?$sort={"age":-1}&$limit=10&$skip=20
```

### Connection Management

- **Connection Pooling**: Configurable pool with max 25 connections
- **UTF8MB4 Support**: Full Unicode character support
- **Automatic Reconnection**: Built-in connection health checks
- **Query Optimization**: Indexed common fields for performance

## Testing

### E2E Tests

Run comprehensive MySQL tests:

```bash
# Using environment configuration
./e2e/scripts/run-mysql-e2e.sh

# With custom credentials (overrides .env)
./e2e/scripts/run-mysql-e2e.sh --mysql-user myuser --mysql-pass mypass --mysql-host myhost
```

The E2E tests verify:

- Database connection and table creation
- CRUD operations (Create, Read, Update, Delete)
- JSON query functionality
- Connection pooling under load
- Data integrity and persistence
- MongoDB-style operator support

### Manual Testing

Quick connection test:

```bash
# Check configuration
./scripts/run_mysql.sh --check-config

# Start server and test
./scripts/run_mysql.sh &
curl http://localhost:2403/collections
```

## Production Considerations

### Security

1. **Environment Variables**: Use `.env` file or environment variables, never hardcode credentials
2. **Database User**: Create dedicated MySQL user with minimal required privileges
3. **Network Security**: Use SSL connections in production
4. **Firewall**: Restrict MySQL port access to application servers only

### Performance

1. **Connection Pooling**: Adjust pool size based on load (default: 25 max connections)
2. **Indexing**: Add indexes on frequently queried JSON fields:
   ```sql
   ALTER TABLE users ADD INDEX idx_email ((JSON_EXTRACT(data, '$.email')));
   ```
3. **Database Tuning**: Configure MySQL for JSON workloads
4. **Monitoring**: Monitor connection pool usage and query performance

### Backup and Recovery

1. **Regular Backups**: Use `mysqldump` for consistent backups
2. **Point-in-Time Recovery**: Enable binary logging
3. **Testing**: Regularly test backup restoration procedures

## Migration

### From SQLite

The database schema is compatible. Export SQLite data and import to MySQL:

```bash
# Export from SQLite (custom script needed)
# Import to MySQL using go-deployd API or direct SQL
```

### From MongoDB

Data structure is compatible since go-deployd maintains document-style storage:

```bash
# Export MongoDB collections
mongoexport --collection=users --out=users.json

# Import via go-deployd API
curl -X POST http://localhost:2403/users -d @users.json
```

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   ```
   Error 1045: Access denied for user
   ```
   - Verify username/password in `.env`
   - Check user permissions: `SHOW GRANTS FOR 'username'@'host';`
   - Ensure user can connect from your IP

2. **Database Not Found**
   ```
   Error 1049: Unknown database
   ```
   - Create database manually: `CREATE DATABASE your_db_name;`
   - Verify database name in configuration

3. **Connection Refused**
   ```
   Error 2003: Can't connect to MySQL server
   ```
   - Check MySQL service status: `systemctl status mysql`
   - Verify host/port configuration
   - Check firewall settings

4. **JSON Column Errors**
   ```
   Error 1064: Syntax error near 'JSON'
   ```
   - Requires MySQL 5.7+ for JSON support
   - Upgrade MySQL or use alternative database

### Debug Mode

Enable verbose logging:

```bash
DEVELOPMENT_MODE=true ./scripts/run_mysql.sh
```

Check server logs for detailed error information.

### Performance Issues

1. **Slow Queries**: Add indexes on frequently accessed JSON fields
2. **Connection Pool Exhaustion**: Increase max connections or optimize query patterns  
3. **Memory Usage**: Monitor MySQL memory configuration for JSON operations

## Architecture

### Database Layer

```
┌─────────────────┐
│   go-deployd    │
│   Application   │
├─────────────────┤
│ Database        │
│ Interface       │
├─────────────────┤
│ MySQL Driver    │
│ (Go SQL Driver) │
├─────────────────┤
│ MySQL Server    │
│ (JSON Storage)  │
└─────────────────┘
```

### Query Translation

MongoDB-style queries are translated to MySQL JSON operations:

```javascript
// Input: MongoDB query
{ "age": { "$gte": 18 }, "status": "active" }

// Output: MySQL WHERE clause
WHERE JSON_EXTRACT(data, '$.age') >= ? AND JSON_EXTRACT(data, '$.status') = ?
```

## Contributing

When contributing MySQL-related features:

1. **Test Coverage**: Add tests for new functionality
2. **Environment Variables**: Use env vars for configuration
3. **Documentation**: Update this README for new features
4. **Compatibility**: Ensure compatibility with existing APIs
5. **Security**: Never commit credentials or expose sensitive data

## Support

For MySQL-specific issues:

1. Check this documentation
2. Review E2E test implementations for examples
3. Examine server logs for detailed error information
4. Test with minimal configuration to isolate issues

## Version Requirements

- **go-deployd**: Current version with MySQL support
- **MySQL**: 5.7+ (JSON support), 8.0+ recommended
- **Go**: 1.21+ for building
- **Operating System**: Cross-platform support (Linux, macOS, Windows)