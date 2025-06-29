# Collections API

The Collections API provides a RESTful interface for performing CRUD (Create, Read, Update, Delete) operations on your data collections. This API follows standard HTTP conventions and supports advanced querying capabilities using MongoDB-style operators.

## Table of Contents

- [Basic CRUD Operations](#basic-crud-operations)
  - [GET All Documents](#get-all-documents)
  - [GET Single Document](#get-single-document)
  - [POST Create Document](#post-create-document)
  - [PUT Update Document](#put-update-document)
  - [DELETE Document](#delete-document)
- [Advanced Queries](#advanced-queries)
  - [Filtering](#filtering)
  - [MongoDB-Style Operators](#mongodb-style-operators)
  - [Sorting & Pagination](#sorting--pagination)

## Basic CRUD Operations

All examples below use `{collection}` as a placeholder for your actual collection name (e.g., `users`, `posts`, `products`).

### GET All Documents

Retrieve all documents from a collection.

**Request:**
```bash
curl -X GET "http://localhost:8080/{collection}"
```

**Response:**
```json
[
  {
    "id": "doc123",
    "title": "Example Document",
    "createdAt": "2024-06-22T10:00:00Z",
    "updatedAt": "2024-06-22T10:00:00Z"
  }
]
```

### GET Single Document

Retrieve a specific document by its ID.

**Request:**
```bash
curl -X GET "http://localhost:8080/{collection}/doc123"
```

**Response:**
```json
{
  "id": "doc123",
  "title": "Example Document",
  "content": "Document content here",
  "createdAt": "2024-06-22T10:00:00Z",
  "updatedAt": "2024-06-22T10:00:00Z"
}
```

### POST Create Document

Create a new document in the collection.

**Request:**
```bash
curl -X POST "http://localhost:8080/{collection}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "New Document",
    "content": "Document content here",
    "tags": ["example", "api"]
  }'
```

**Response:**
```json
{
  "id": "newly-generated-id",
  "title": "New Document",
  "content": "Document content here",
  "tags": ["example", "api"],
  "createdAt": "2024-06-22T10:00:00Z",
  "updatedAt": "2024-06-22T10:00:00Z"
}
```

### PUT Update Document

Update an existing document by its ID.

**Request:**
```bash
curl -X PUT "http://localhost:8080/{collection}/doc123" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Document",
    "content": "Updated content"
  }'
```

**Response:**
```json
{
  "id": "doc123",
  "title": "Updated Document",
  "content": "Updated content",
  "createdAt": "2024-06-22T10:00:00Z",
  "updatedAt": "2024-06-22T10:30:00Z"
}
```

### DELETE Document

Delete a document by its ID.

**Request:**
```bash
curl -X DELETE "http://localhost:8080/{collection}/doc123"
```

**Response:**
```json
{
  "message": "Document deleted successfully"
}
```

## Advanced Queries

The Collections API supports advanced querying capabilities through URL parameters, enabling complex data retrieval operations.

### Filtering

Filter documents using simple field-value pairs.

**Simple filtering:**
```bash
# Filter by field value
curl "http://localhost:8080/{collection}?status=active"

# Multiple filters (AND operation)
curl "http://localhost:8080/{collection}?status=active&priority=high"
```

### MongoDB-Style Operators

Use MongoDB-style operators for advanced filtering conditions.

#### Comparison Operators

**Greater than / Less than:**
```bash
# Greater than
curl "http://localhost:8080/{collection}?age={\"$gt\":18}"

# Less than or equal
curl "http://localhost:8080/{collection}?price={\"$lte\":100}"
```

**In array:**
```bash
curl "http://localhost:8080/{collection}?status={\"$in\":[\"active\",\"pending\"]}"
```

**Not equal:**
```bash
curl "http://localhost:8080/{collection}?status={\"$ne\":\"deleted\"}"
```

**Field exists:**
```bash
curl "http://localhost:8080/{collection}?email={\"$exists\":true}"
```

#### Supported Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `$gt` | Greater than | `?age={"$gt":18}` |
| `$gte` | Greater than or equal | `?age={"$gte":18}` |
| `$lt` | Less than | `?price={"$lt":100}` |
| `$lte` | Less than or equal | `?price={"$lte":100}` |
| `$eq` | Equal to | `?status={"$eq":"active"}` |
| `$ne` | Not equal to | `?status={"$ne":"deleted"}` |
| `$in` | Value in array | `?status={"$in":["active","pending"]}` |
| `$nin` | Value not in array | `?status={"$nin":["deleted","archived"]}` |
| `$exists` | Field exists | `?email={"$exists":true}` |

### Sorting & Pagination

Control the order and quantity of returned results.

#### Sorting

**Sort ascending:**
```bash
curl "http://localhost:8080/{collection}?$sort={\"createdAt\":1}"
```

**Sort descending:**
```bash
curl "http://localhost:8080/{collection}?$sort={\"createdAt\":-1}"
```

**Multiple field sorting:**
```bash
curl "http://localhost:8080/{collection}?$sort={\"priority\":-1,\"createdAt\":1}"
```

#### Pagination

**Limit results:**
```bash
curl "http://localhost:8080/{collection}?$limit=10"
```

**Skip results (for pagination):**
```bash
curl "http://localhost:8080/{collection}?$skip=20&$limit=10"
```

#### Field Selection

**Select specific fields:**
```bash
curl "http://localhost:8080/{collection}?$fields={\"title\":1,\"status\":1}"
```

**Exclude specific fields:**
```bash
curl "http://localhost:8080/{collection}?$fields={\"sensitiveData\":0}"
```

#### Special Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `$sort` | Sort results | `?$sort={"createdAt":-1}` |
| `$limit` | Limit number of results | `?$limit=10` |
| `$skip` | Skip number of results | `?$skip=20` |
| `$fields` | Select/exclude fields | `?$fields={"title":1,"content":1}` |

## Complex Query Examples

**Paginated, filtered, and sorted results:**
```bash
curl "http://localhost:8080/users?status=active&age={\"$gte\":18}&$sort={\"createdAt\":-1}&$limit=10&$skip=0"
```

**Get recent active posts with specific fields:**
```bash
curl "http://localhost:8080/posts?status=published&createdAt={\"$gte\":\"2024-06-01\"}&$sort={\"createdAt\":-1}&$fields={\"title\":1,\"author\":1,\"createdAt\":1}&$limit=5"
```

## Response Format

All API responses follow a consistent JSON format:

### Successful Responses

**Single document:**
```json
{
  "id": "document-id",
  "field1": "value1",
  "field2": "value2",
  "createdAt": "2024-06-22T10:00:00Z",
  "updatedAt": "2024-06-22T10:00:00Z"
}
```

**Multiple documents:**
```json
[
  {
    "id": "document-id-1",
    "field1": "value1"
  },
  {
    "id": "document-id-2", 
    "field1": "value2"
  }
]
```

### Error Responses

```json
{
  "error": "Error message describing what went wrong",
  "code": "ERROR_CODE",
  "details": "Additional error details if available"
}
```

Common HTTP status codes:
- `200` - Success (GET, PUT)
- `201` - Created (POST)
- `204` - No Content (DELETE)
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error

## Authentication

Some operations may require authentication depending on your server configuration. Authentication can be provided via:

1. **Master Key** (in headers): `X-Master-Key: your-master-key`
2. **User Authentication** (session-based)
3. **API Keys** (if configured)

Refer to the Authentication documentation for detailed information on securing your API endpoints.