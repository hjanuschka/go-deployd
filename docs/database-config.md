# Database Configuration

Go-Deployd supports MongoDB, MySQL, and SQLite databases. Choose based on your deployment needs and scale requirements.

## Table of Contents

- [MongoDB Configuration](#mongodb-configuration)
- [MySQL Configuration](#mysql-configuration)
- [SQLite Configuration](#sqlite-configuration)
- [Switching Databases](#switching-databases)
- [Feature Comparison](#feature-comparison)
- [Performance Considerations](#performance-considerations)

## MongoDB Configuration

Best for: Production environments, horizontal scaling, complex queries

```bash
# Set MongoDB connection string
export DATABASE_URL="mongodb://localhost:27017/deployd"

# Or with authentication
export DATABASE_URL="mongodb://username:password@localhost:27017/deployd"

# MongoDB Atlas (cloud)
export DATABASE_URL="mongodb+srv://username:password@cluster.mongodb.net/deployd"
```

## MySQL Configuration

Best for: Production environments, relational data, ACID compliance, enterprise deployments

```bash
# Set MySQL connection string
export DATABASE_URL="mysql://username:password@localhost:3306/deployd"

# With additional options
export DATABASE_URL="mysql://username:password@localhost:3306/deployd?charset=utf8mb4&parseTime=True&loc=Local"

# Using environment variables
export MYSQL_USER=deployd_user
export MYSQL_PASSWORD=secure_password
export MYSQL_DATABASE=deployd
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
```

## SQLite Configuration

Best for: Development, testing, small applications, single-server deployments

```bash
# Set SQLite database file path
export DATABASE_URL="sqlite:./data/deployd.db"

# Or use in-memory database (for testing)
export DATABASE_URL="sqlite::memory:"
```

## Switching Databases

Simply change the DATABASE_URL environment variable and restart the server. All databases support the same API and event system features.

```bash
# Development with SQLite
export DATABASE_URL="sqlite:./data/dev.db"
./deployd

# Production with MongoDB
export DATABASE_URL="mongodb://localhost:27017/deployd_prod"
./deployd

# Production with MySQL
export DATABASE_URL="mysql://user:pass@localhost:3306/deployd_prod"
./deployd
```

## Feature Comparison

### MongoDB
- ✅ Horizontal scaling
- ✅ Advanced indexing
- ✅ Replica sets
- ✅ Aggregation pipeline
- ✅ Document-based storage
- ❌ Requires separate server

### MySQL
- ✅ ACID transactions
- ✅ Relational data model
- ✅ Mature ecosystem
- ✅ Replication support
- ✅ Column-based storage with go-deployd
- ❌ Requires separate server

### SQLite
- ✅ Zero configuration
- ✅ Single file database
- ✅ ACID transactions
- ✅ Embedded in application
- ✅ Column-based storage with go-deployd
- ❌ Single writer limitation

## Performance Considerations

### Database Performance
- **SQLite**: Excellent for read-heavy workloads, single-server deployments
- **MongoDB**: Better for write-heavy workloads, multiple servers, document-based queries
- **MySQL**: Great for complex relational queries, transactions, and enterprise workloads
- All databases support efficient indexing on collection properties
- MySQL and SQLite use column-based storage for better query performance
- Consider database-specific optimizations in your events

### Event Performance
- **Go events**: ~50-100x faster than JavaScript for CPU-intensive tasks
- **JavaScript events**: Better for simple validations and npm ecosystem
- Event compilation happens once at startup or file change
- Use Go events for complex business logic and calculations