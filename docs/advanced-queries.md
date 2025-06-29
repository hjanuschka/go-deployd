# Advanced Queries

Go-Deployd supports MongoDB-style queries across all database backends (MongoDB, MySQL, SQLite). This document covers query translation, advanced operators, and the $forceMongo option.

## Table of Contents

- [MongoDB-to-SQL Translation](#mongodb-to-sql-translation)
  - [Supported Operators](#supported-operators)
  - [Query Examples](#query-examples)
- [$forceMongo Option](#forcemongo-option)
- [POST /query Endpoint](#post-query-endpoint)
- [Complex Query Examples](#complex-query-examples)
- [Query Performance](#query-performance)

## MongoDB-to-SQL Translation

When using MySQL or SQLite backends, MongoDB-style queries are automatically translated to SQL. This provides a consistent API across all database types.

### Supported Operators

| MongoDB Operator | SQL Translation | Example |
|-----------------|-----------------|---------|
| `$eq` | `=` | `{name: "John"}` → `name = 'John'` |
| `$ne` | `!=` | `{age: {$ne: 25}}` → `age != 25` |
| `$gt` | `>` | `{age: {$gt: 18}}` → `age > 18` |
| `$gte` | `>=` | `{age: {$gte: 18}}` → `age >= 18` |
| `$lt` | `<` | `{age: {$lt: 65}}` → `age < 65` |
| `$lte` | `<=` | `{age: {$lte: 65}}` → `age <= 65` |
| `$in` | `IN` | `{status: {$in: ["active", "pending"]}}` → `status IN ('active', 'pending')` |
| `$nin` | `NOT IN` | `{status: {$nin: ["deleted"]}}` → `status NOT IN ('deleted')` |
| `$regex` | `LIKE` | `{name: {$regex: "john"}}` → `name LIKE '%john%'` |
| `$exists` | `IS [NOT] NULL` | `{email: {$exists: true}}` → `email IS NOT NULL` |
| `$or` | `OR` | `{$or: [{a: 1}, {b: 2}]}` → `(a = 1 OR b = 2)` |
| `$and` | `AND` | `{$and: [{a: 1}, {b: 2}]}` → `(a = 1 AND b = 2)` |

### Query Examples

#### Simple Queries

```javascript
// Exact match
GET /todos?title=Shopping
// SQL: SELECT * FROM todos WHERE title = 'Shopping'

// Not equal
GET /todos?status[$ne]=deleted
// SQL: SELECT * FROM todos WHERE status != 'deleted'

// Greater than
GET /todos?priority[$gt]=5
// SQL: SELECT * FROM todos WHERE priority > 5
```

#### Range Queries

```javascript
// Between values
GET /todos?priority[$gte]=3&priority[$lte]=7
// SQL: SELECT * FROM todos WHERE priority >= 3 AND priority <= 7
```

#### Pattern Matching

```javascript
// Contains pattern
GET /todos?title[$regex]=urgent
// SQL: SELECT * FROM todos WHERE title LIKE '%urgent%'

// Starts with pattern
GET /todos?title[$regex]=^Task
// SQL: SELECT * FROM todos WHERE title LIKE 'Task%'

// Ends with pattern
GET /todos?title[$regex]=completed$
// SQL: SELECT * FROM todos WHERE title LIKE '%completed'
```

#### IN Queries

```javascript
// Multiple values
GET /todos?status[$in]=active,pending,review
// SQL: SELECT * FROM todos WHERE status IN ('active', 'pending', 'review')
```

## $forceMongo Option

The `$forceMongo` option bypasses SQL translation and sends queries directly to MongoDB when available. This is useful for MongoDB-specific features that can't be translated to SQL.

### Usage

```javascript
// Force MongoDB query execution
POST /todos/query
{
  "query": {
    "title": {"$regex": "^urgent", "$options": "i"},
    "tags": {"$all": ["important", "deadline"]}
  },
  "options": {
    "$forceMongo": true
  }
}
```

### When to Use $forceMongo

- MongoDB-specific operators (`$all`, `$elemMatch`, `$size`)
- Complex aggregations
- Text search queries
- Geospatial queries
- When you need MongoDB's exact behavior

### Example with JavaScript

```javascript
// Using dpd.js with $forceMongo
dpd.todos.post('/query', {
  query: {
    tags: {$all: ['urgent', 'important']},
    location: {
      $near: {
        $geometry: {type: "Point", coordinates: [-73.9667, 40.78]},
        $maxDistance: 1000
      }
    }
  },
  options: {
    $forceMongo: true
  }
}, function(todos, error) {
  if (!error) {
    console.log('MongoDB-specific query results:', todos);
  }
});
```

## POST /query Endpoint

For complex queries that can't be expressed as URL parameters, use the POST /query endpoint:

```bash
POST /todos/query
Content-Type: application/json

{
  "query": {
    "$or": [
      {"priority": {"$gte": 8}},
      {"title": {"$regex": "urgent"}},
      {"tags": {"$in": ["critical", "important"]}}
    ],
    "completed": false
  },
  "options": {
    "$sort": {"priority": -1, "createdAt": -1},
    "$limit": 20,
    "$skip": 0,
    "$fields": {"title": 1, "priority": 1, "tags": 1}
  }
}
```

### Response

```json
[
  {
    "id": "doc123",
    "title": "Urgent: Fix production bug",
    "priority": 10,
    "tags": ["critical", "bug"]
  },
  {
    "id": "doc124",
    "title": "Review security audit",
    "priority": 9,
    "tags": ["important", "security"]
  }
]
```

## Complex Query Examples

### E-commerce Product Search

```javascript
// Find laptops or computers between $500-$2000 in stock
POST /products/query
{
  "query": {
    "$and": [
      {
        "$or": [
          {"title": {"$regex": "laptop"}},
          {"title": {"$regex": "computer"}}
        ]
      },
      {
        "price": {
          "$gte": 500,
          "$lte": 2000
        }
      },
      {
        "status": {"$in": ["available", "limited"]}
      }
    ]
  },
  "options": {
    "$sort": {"price": 1},
    "$limit": 10
  }
}
```

### User Activity Query

```javascript
// Find active users who either completed tasks or have high-priority items
POST /users/query
{
  "query": {
    "active": true,
    "$or": [
      {"completedTasks": {"$gte": 10}},
      {
        "$and": [
          {"pendingTasks.priority": {"$gte": 8}},
          {"role": {"$in": ["admin", "manager"]}}
        ]
      }
    ]
  }
}
```

### Date Range Query

```javascript
// Find all orders from last 30 days
const thirtyDaysAgo = new Date();
thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);

POST /orders/query
{
  "query": {
    "createdAt": {
      "$gte": thirtyDaysAgo.toISOString()
    },
    "status": {"$ne": "cancelled"}
  },
  "options": {
    "$sort": {"createdAt": -1}
  }
}
```

## Query Performance

### Optimization Tips

1. **Use Indexes**: Ensure frequently queried fields are indexed
2. **Limit Results**: Always use `$limit` for large collections
3. **Project Fields**: Use `$fields` to return only needed data
4. **Avoid Deep Nesting**: Simpler queries perform better
5. **Use $forceMongo Sparingly**: Only when SQL translation isn't sufficient

### Database-Specific Considerations

#### MongoDB
- Native query execution
- Best performance for document queries
- Supports all MongoDB operators

#### MySQL/SQLite
- Queries translated to SQL
- Column-based storage for better performance
- Some MongoDB operators not supported without $forceMongo
- Regex queries use LIKE operator (less powerful than MongoDB regex)

### Query Translation Examples

```javascript
// MongoDB Query
{
  "title": {"$regex": "^Task"},
  "priority": {"$gte": 5},
  "$or": [
    {"status": "active"},
    {"assignedTo": "user123"}
  ]
}

// Translated SQL (MySQL/SQLite)
SELECT * FROM todos 
WHERE title LIKE 'Task%' 
  AND priority >= 5 
  AND (status = 'active' OR assignedTo = 'user123')
```