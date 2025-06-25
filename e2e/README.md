# E2E Tests for go-deployd

This directory contains end-to-end tests for go-deployd that validate functionality across different database backends.

## Available Test Suites

### 1. Multi-Database E2E Tests (`run-e2e.sh`)
Tests both SQLite and MongoDB with identical datasets to ensure consistency across database implementations.

**Features:**
- CRUD operations testing
- Query operations with MongoDB-style operators
- Authentication and authorization
- Data consistency validation between databases
- Real-world scenarios with users, products, and orders

**Usage:**
```bash
./e2e/scripts/run-e2e.sh
```

### 2. MySQL E2E Tests (`run-mysql-e2e.sh`)
Dedicated test suite for MySQL database support with comprehensive validation.

**Features:**
- MySQL-specific CRUD operations
- JSON column functionality testing
- Connection pooling under load
- Complex query operations
- Data integrity validation
- Table structure verification

**Usage:**
```bash
# With default settings (root user, no password, localhost)
./e2e/scripts/run-mysql-e2e.sh

# With custom MySQL credentials
./e2e/scripts/run-mysql-e2e.sh --mysql-user myuser --mysql-pass mypass

# With custom host and database
./e2e/scripts/run-mysql-e2e.sh --mysql-host myhost --mysql-db testdb
```

**Command Line Options:**
- `--mysql-user USER`: MySQL username (default: root)
- `--mysql-pass PASS`: MySQL password (default: empty)
- `--mysql-host HOST`: MySQL host (default: localhost)
- `--mysql-db DB`: Test database name (default: deployd_e2e_test)
- `--help`: Show help message

## Prerequisites

### System Requirements
- `jq` - JSON processor for parsing API responses
- `curl` - HTTP client for API testing
- `go` - Go compiler for building deployd

### Database Requirements

#### For SQLite tests:
- No additional setup required (SQLite is embedded)

#### For MongoDB tests:
- MongoDB server running on localhost:27017
- `mongo` or `mongosh` client available

#### For MySQL tests:
- MySQL server running and accessible
- `mysql` client available
- Database user with create/drop database privileges

## Test Data

The tests use fixture data located in `e2e/fixtures/`:
- `users.json` - Sample user accounts with various roles
- `products.json` - Product catalog with different categories
- `orders.json` - Order records linking users and products

## Test Collections

The tests create several collections with specific schemas:

### Users Collection
```json
{
  "username": "string (required)",
  "email": "string (required)",
  "password": "string (required)",
  "role": "string (default: user)",
  "name": "string",
  "age": "number",
  "active": "boolean (default: true)"
}
```

### Products Collection
```json
{
  "name": "string (required)",
  "price": "number (required)",
  "category": "string (required)",
  "inStock": "boolean (default: true)",
  "quantity": "number (default: 0)"
}
```

### Orders Collection
```json
{
  "userId": "string (required)",
  "status": "string (required)",
  "total": "number (required)",
  "items": "array"
}
```

### Private Docs Collection (for auth testing)
```json
{
  "title": "string (required)",
  "content": "string (required)",
  "userId": "string (required)",
  "private": "boolean (default: true)"
}
```

## What Gets Tested

### CRUD Operations
- **Create**: Insert new documents with validation
- **Read**: Retrieve single and multiple documents
- **Update**: Modify existing documents
- **Delete**: Remove documents and verify deletion

### Query Features
- Basic filtering and searching
- MongoDB-style operators (`$gte`, `$lte`, `$regex`, etc.)
- Sorting with multiple fields
- Pagination with limit and offset
- Field projection (include/exclude)

### Authentication & Authorization
- User registration and login
- Session management
- Role-based access control
- Document ownership filtering
- Master key (root) access
- Multi-user scenarios

### Database-Specific Features
- **SQLite**: File-based storage, JSON column support
- **MongoDB**: Native document operations, BSON handling
- **MySQL**: JSON columns, connection pooling, UTF8MB4 support

### Performance & Reliability
- Connection pooling under concurrent load
- Data integrity across operations
- Error handling and recovery
- Server responsiveness under stress

## Output and Results

Test results are stored in:
- `e2e/results/` - General test results and logs
- `e2e/mysql-results/` - MySQL-specific test results

Each test run generates:
- Server logs for debugging
- Request/response samples
- Performance metrics
- Error reports (if any)

## Example Test Run

```bash
$ ./e2e/scripts/run-mysql-e2e.sh

[MySQL E2E] Starting MySQL E2E tests for go-deployd
[MySQL E2E] Checking prerequisites...
[SUCCESS] Prerequisites check passed
[MySQL E2E] Checking MySQL availability...
[SUCCESS] MySQL is available and accessible
[MySQL E2E] Setting up MySQL test database...
[SUCCESS] MySQL test database 'deployd_e2e_test' created successfully
[MySQL E2E] Building deployd binary with MySQL support...
[SUCCESS] Deployd binary built successfully
[MySQL E2E] Setting up collection configurations...
[SUCCESS] Collection configurations created
[MySQL E2E] Starting deployd server with MySQL on port 9003...
[MySQL E2E] Waiting for MySQL server to start...
[SUCCESS] MySQL server started successfully (PID: 12345)
[MySQL E2E] Loading test data for collection: users
  ✓ Inserted item: alice
  ✓ Inserted item: bob
  ✓ Inserted item: charlie
[SUCCESS] Test data loaded for collection: users
[MySQL E2E] Testing CRUD operations for MySQL...
[SUCCESS] CREATE operation successful (ID: abc123)
[SUCCESS] READ operation successful
[SUCCESS] UPDATE operation successful
[SUCCESS] DELETE operation successful
[SUCCESS] DELETE verification successful
[MySQL E2E] Testing MySQL-specific features...
[SUCCESS] JSON query returned 5 users with age >= 25
[SUCCESS] Descending age sort - first user age: 45
[SUCCESS] Regex query returned 3 users with @example.com emails
[MySQL E2E] Testing data integrity in MySQL...
[MySQL E2E] Total users in MySQL: 8
[MySQL E2E] Total products in MySQL: 12
[MySQL E2E] Total orders in MySQL: 6
[SUCCESS] Data integrity check passed - all collections have data
[MySQL E2E] Testing MySQL connection pooling under concurrent load...
[SUCCESS] Connection pooling test passed - server remains responsive
[MySQL E2E] Verifying MySQL table structure...
[MySQL E2E] Found 3 tables in MySQL database
[SUCCESS] Users table exists with correct structure
[MySQL E2E] Active users (JSON query): 6
[SUCCESS] JSON column functionality verified
[MySQL E2E] Stopping MySQL server (PID: 12345)...
[SUCCESS] MySQL server stopped
[SUCCESS] All MySQL E2E tests completed successfully!
[MySQL E2E] Test results available in: /path/to/e2e/mysql-results
[MySQL E2E] Server logs available in: /path/to/e2e/mysql-results/mysql-server.log
```

## Troubleshooting

### Common Issues

1. **MySQL Connection Failed**
   - Ensure MySQL server is running
   - Verify credentials and host settings
   - Check that user has necessary privileges

2. **Port Already in Use**
   - Tests use ports 9001 (SQLite), 9002 (MongoDB), 9003 (MySQL)
   - Kill any existing processes or change ports in scripts

3. **Prerequisites Missing**
   - Install `jq`: `brew install jq` (macOS) or `apt-get install jq` (Ubuntu)
   - Install `curl`: Usually pre-installed on most systems

4. **Build Failures**
   - Ensure Go is installed and up to date
   - Run `go mod tidy` to update dependencies
   - Check that all imports are available

### Debug Mode

For debugging test failures, check the server logs:
```bash
# View MySQL server logs
cat e2e/mysql-results/mysql-server.log

# View general test logs
cat e2e/results/sqlite-server.log
cat e2e/results/mongodb-server.log
```

## Contributing

When adding new tests:
1. Follow the existing pattern for consistent output
2. Add both positive and negative test cases
3. Include cleanup in case of failures
4. Document any new prerequisites or setup steps
5. Ensure tests are idempotent (can be run multiple times)