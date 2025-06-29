# Column-Based Storage Guide

## Overview

go-deployd features an advanced **column-based storage system** that dramatically improves query performance by storing frequently accessed fields as native database columns instead of JSON data. This hybrid approach combines the flexibility of document storage with the performance of relational databases.

## Performance Comparison

### Traditional JSON Storage
```sql
-- Slow: JSON extraction with no indexing
SELECT * FROM todos WHERE JSON_EXTRACT(data, '$.completed') = false;
```

### Column-Based Storage
```sql
-- Fast: Direct column access with native indexes
SELECT * FROM todos WHERE "completed" = false;
```

**Performance improvement: 5-50x faster queries on indexed fields**

## Enabling Column-Based Storage

Add `"useColumns": true` to your collection configuration:

```json
{
  "properties": {
    "title": {
      "type": "string",
      "required": true,
      "index": true
    },
    "completed": {
      "type": "boolean",
      "default": false,
      "index": true
    },
    "priority": {
      "type": "number",
      "default": 1
    },
    "metadata": {
      "type": "object"
    }
  },
  "options": {
    "useColumns": true
  }
}
```

## How It Works

### Hybrid Storage Architecture

1. **Primitive Fields** → Native database columns (string, number, boolean, date)
2. **Complex Objects** → JSON storage in `data` column
3. **Indexes** → Native SQL indexes for fast lookups

### Automatic Schema Management

When `useColumns: true` is enabled:

- Database columns are created automatically for primitive fields
- Existing data is migrated seamlessly
- Schema changes are handled gracefully
- Backward compatibility is maintained

### Field Type Mapping

| JSON Type | Database Column Type | Indexable |
|-----------|---------------------|-----------|
| `string` | `TEXT` | ✅ |
| `number` | `REAL` | ✅ |
| `boolean` | `BOOLEAN` | ✅ |
| `date` | `DATETIME` | ✅ |
| `object` | JSON (in data column) | ❌ |
| `array` | JSON (in data column) | ❌ |

## Performance Best Practices

### 1. Index Frequently Queried Fields

```json
{
  "properties": {
    "email": {
      "type": "string",
      "unique": true,
      "index": true
    },
    "status": {
      "type": "string", 
      "index": true
    },
    "createdAt": {
      "type": "date",
      "index": true
    }
  },
  "options": {
    "useColumns": true
  }
}
```

### 2. Use Appropriate Data Types

```json
{
  "properties": {
    "userId": {"type": "string"},     // For IDs and references
    "count": {"type": "number"},      // For numeric values
    "active": {"type": "boolean"},    // For flags
    "timestamp": {"type": "date"}     // For dates/times
  }
}
```

### 3. Keep Complex Data in JSON

```json
{
  "properties": {
    "name": {"type": "string", "index": true},  // Column
    "settings": {"type": "object"},             // JSON
    "tags": {"type": "array"}                   // JSON
  }
}
```

## Migration Guide

### Enabling for Existing Collections

1. **Backup your data** (recommended)
2. Add `"useColumns": true` to config.json
3. Restart the application
4. Schema migration happens automatically

### Example Migration

**Before (JSON-only):**
```json
{
  "properties": {
    "title": {"type": "string"},
    "completed": {"type": "boolean"}
  }
}
```

**After (Column-based):**
```json
{
  "properties": {
    "title": {"type": "string", "index": true},
    "completed": {"type": "boolean", "index": true}
  },
  "options": {
    "useColumns": true
  }
}
```

## Database Support

| Database | Column Storage | Indexes | Migration |
|----------|---------------|---------|-----------|
| SQLite | ✅ Full Support | ✅ | ✅ |
| MySQL | ✅ Full Support | ✅ | ✅ |
| MongoDB | ❌ JSON Only | ✅ | N/A |

## Monitoring and Debugging

### Enable Debug Logging

Set environment variable:
```bash
DEBUG_COLUMN_STORE=true
```

### Query Analysis

Check logs for column access patterns:
```
DEBUG: Field 'email' has column, using direct access: "email"
DEBUG: SQLQueryBuilder generated SQL: "email" = ?
```

### Performance Monitoring

```javascript
// Check if collection uses column storage
GET /collection-name?$debug=true

// Response includes storage type
{
  "meta": {
    "storageType": "ColumnStore",
    "useColumns": true,
    "indexedFields": ["email", "status"]
  }
}
```

## Advanced Configuration

### Custom Column Names

```json
{
  "properties": {
    "emailAddress": {
      "type": "string",
      "columnName": "email",
      "index": true
    }
  }
}
```

### Partial Column Storage

```json
{
  "properties": {
    "id": {"type": "string", "useColumn": true},
    "name": {"type": "string", "useColumn": true},
    "profile": {"type": "object", "useColumn": false}
  },
  "options": {
    "useColumns": true
  }
}
```

## Troubleshooting

### Common Issues

**1. Schema Lock Errors**
```
Error: Cannot modify column 'name' in SQLite table (not supported)
```
*Solution: SQLite doesn't support column modifications. Recreate the database or use a new collection.*

**2. Index Creation Failed**
```
Error: Index creation failed for field 'tags'
```
*Solution: Arrays and objects cannot be indexed. Move complex data to JSON storage.*

**3. Performance Not Improved**
```
DEBUG: Field 'field' no column, using JSON extraction
```
*Solution: Ensure `useColumns: true` is set and field type is primitive.*

### Best Practices for Production

1. **Test migrations** on development data first
2. **Monitor query performance** after enabling
3. **Use appropriate indexes** for your query patterns
4. **Keep complex data in JSON** for flexibility

## Example Collections

### High-Performance User Collection

```json
{
  "properties": {
    "username": {"type": "string", "unique": true, "index": true},
    "email": {"type": "string", "unique": true, "index": true},
    "active": {"type": "boolean", "default": true, "index": true},
    "role": {"type": "string", "index": true},
    "createdAt": {"type": "date", "default": "now", "index": true},
    "profile": {"type": "object"},
    "preferences": {"type": "object"}
  },
  "options": {
    "useColumns": true
  }
}
```

### E-commerce Product Collection

```json
{
  "properties": {
    "sku": {"type": "string", "unique": true, "index": true},
    "name": {"type": "string", "index": true},
    "price": {"type": "number", "index": true},
    "category": {"type": "string", "index": true},
    "inStock": {"type": "boolean", "index": true},
    "tags": {"type": "array"},
    "specifications": {"type": "object"},
    "reviews": {"type": "array"}
  },
  "options": {
    "useColumns": true
  }
}
```

---

**Column-based storage is the recommended approach for production deployments requiring high query performance.**