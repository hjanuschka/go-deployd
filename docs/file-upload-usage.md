# File Upload & Storage Guide

This guide covers the complete file upload and storage system in go-deployd, including setup, configuration, and usage examples.

## 🎯 Overview

The file storage system in go-deployd provides:
- **Multiple storage backends**: Local, S3, and MinIO
- **Zero-configuration local development**: Works out of the box
- **Scalable cloud storage**: Easy migration to S3/MinIO for production
- **Built-in security**: File validation, size limits, and user-based access control
- **REST API**: Full CRUD operations for files
- **Metadata storage**: File information stored in database for fast queries
- **Real-time integration**: WebSocket events for file operations
- **Event system**: Customizable validation and processing via JavaScript events

## 🚀 Quick Start

### 1. Default Local Storage

The system works immediately without any configuration:

```bash
# Start the server
npm run dev

# Upload a file
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@example.jpg" \
  http://localhost:2403/files
```

### 2. Configuration

Storage configuration is loaded from `.deployd/storage.json`:

**Default (Local Storage):**
```json
{
  "type": "local",
  "local": {
    "basePath": "./uploads"
  },
  "maxFileSize": 10485760,
  "allowedExtensions": ["*"],
  "signedUrlExpiration": 3600
}
```

**S3 Configuration:**
```json
{
  "type": "s3",
  "s3": {
    "bucket": "my-bucket",
    "region": "us-east-1",
    "accessKeyId": "${AWS_ACCESS_KEY_ID}",
    "secretAccessKey": "${AWS_SECRET_ACCESS_KEY}"
  },
  "maxFileSize": 104857600,
  "allowedExtensions": ["jpg", "jpeg", "png", "pdf", "doc", "docx"],
  "signedUrlExpiration": 3600
}
```

**MinIO Configuration:**
```json
{
  "type": "s3",
  "s3": {
    "bucket": "uploads",
    "region": "us-east-1",
    "endpoint": "http://localhost:9000",
    "accessKeyId": "${MINIO_ACCESS_KEY}",
    "secretAccessKey": "${MINIO_SECRET_KEY}",
    "forcePathStyle": true
  },
  "maxFileSize": 104857600,
  "allowedExtensions": ["*"],
  "signedUrlExpiration": 3600
}
```

### 3. Environment Variables

For production, use environment variables for credentials:

```bash
# AWS S3
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"

# MinIO
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
```

## 📚 API Reference

### Upload File

**POST /files**

Upload a file using multipart form data.

```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf" \
  http://localhost:2403/files
```

**Response:**
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "originalName": "document.pdf",
  "storedName": "f47ac10b-58cc-4372-a567-0e02b2c3d479.pdf",
  "mimeType": "application/pdf",
  "size": 1048576,
  "uploadedAt": "2025-06-29T10:30:00Z",
  "uploadedBy": "user123",
  "storageType": "local",
  "path": "2025/06/29/f47ac10b-58cc-4372-a567-0e02b2c3d479.pdf"
}
```

### List Files

**GET /files**

List files with optional query parameters.

```bash
# List all files
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:2403/files

# List with pagination and sorting
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:2403/files?limit=10&offset=0&sort=uploadedAt&order=desc"
```

**Query Parameters:**
- `limit` (default: 50): Number of files to return
- `offset` (default: 0): Number of files to skip
- `sort` (default: "uploadedAt"): Field to sort by
- `order` (default: "desc"): Sort order ("asc" or "desc")

### Download File

**GET /files/{id}**

Download a file by its ID.

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:2403/files/f47ac10b-58cc-4372-a567-0e02b2c3d479 \
  --output downloaded-file.pdf
```

### Get File Info

**GET /files/{id}/info**

Get file metadata without downloading.

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:2403/files/f47ac10b-58cc-4372-a567-0e02b2c3d479/info
```

### Delete File

**DELETE /files/{id}**

Delete a file.

```bash
curl -X DELETE \
  -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:2403/files/f47ac10b-58cc-4372-a567-0e02b2c3d479
```

### Generate Signed URL

**POST /files/signed**

Generate a pre-signed URL for direct upload (S3/MinIO only).

```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"filename": "upload.jpg", "mimeType": "image/jpeg"}' \
  http://localhost:2403/files/signed
```

**Response:**
```json
{
  "uploadUrl": "https://s3.amazonaws.com/bucket/...",
  "fileId": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "expiresAt": "2025-06-29T11:30:00Z"
}
```

## 💻 JavaScript (dpd.js) Usage

### Upload File

```javascript
// HTML file input
const fileInput = document.getElementById('fileInput');
const file = fileInput.files[0];

if (file) {
    const formData = new FormData();
    formData.append('file', file);
    
    fetch('/files', {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer ' + dpd.token
        },
        body: formData
    })
    .then(response => response.json())
    .then(result => {
        console.log('File uploaded:', result);
    })
    .catch(error => {
        console.error('Upload error:', error);
    });
}
```

### List Files

```javascript
dpd.files.get({}, (err, files) => {
    if (err) {
        console.error('Error:', err);
        return;
    }
    
    console.log('Files:', files);
    files.forEach(file => {
        console.log(`${file.originalName} (${file.size} bytes)`);
    });
});
```

### Download File

```javascript
const fileId = 'f47ac10b-58cc-4372-a567-0e02b2c3d479';

dpd.files.get(fileId, (err, fileInfo) => {
    if (err) {
        console.error('Error:', err);
        return;
    }
    
    // Create download link
    const link = document.createElement('a');
    link.href = '/files/' + fileId;
    link.download = fileInfo.originalName;
    link.click();
});
```

### Delete File

```javascript
const fileId = 'f47ac10b-58cc-4372-a567-0e02b2c3d479';

dpd.files.del(fileId, (err) => {
    if (err) {
        console.error('Delete error:', err);
        return;
    }
    
    console.log('File deleted successfully');
});
```

## 🔧 Configuration Options

### Storage Types

| Type | Description | Use Case |
|------|-------------|----------|
| `local` | Local file system storage | Development, small deployments |
| `s3` | AWS S3 or S3-compatible storage | Production, scalable cloud storage |

### File Validation

```json
{
  "maxFileSize": 10485760,
  "allowedExtensions": ["jpg", "jpeg", "png", "gif", "pdf", "doc", "docx"],
  "signedUrlExpiration": 3600
}
```

- **maxFileSize**: Maximum file size in bytes (default: 10MB)
- **allowedExtensions**: Array of allowed file extensions, or `["*"]` for all
- **signedUrlExpiration**: Signed URL expiration time in seconds (S3/MinIO only)

### Local Storage Options

```json
{
  "type": "local",
  "local": {
    "basePath": "./uploads",
    "createDirectories": true,
    "dateBasedPath": true
  }
}
```

- **basePath**: Base directory for file storage
- **createDirectories**: Auto-create directories if they don't exist
- **dateBasedPath**: Organize files in date-based subdirectories (YYYY/MM/DD)

### S3/MinIO Options

```json
{
  "type": "s3",
  "s3": {
    "bucket": "my-bucket",
    "region": "us-east-1",
    "endpoint": "http://localhost:9000",
    "accessKeyId": "${AWS_ACCESS_KEY_ID}",
    "secretAccessKey": "${AWS_SECRET_ACCESS_KEY}",
    "forcePathStyle": true,
    "ssl": false
  }
}
```

- **bucket**: S3 bucket name
- **region**: AWS region
- **endpoint**: Custom endpoint URL (for MinIO)
- **accessKeyId**: AWS access key ID (supports env vars)
- **secretAccessKey**: AWS secret access key (supports env vars)
- **forcePathStyle**: Use path-style URLs (required for MinIO)
- **ssl**: Use SSL/TLS (default: true)

## 🔒 Security Features

### Authentication

All file operations require authentication:
- Users can only access their own files
- Admin users can access all files
- API keys and JWT tokens are supported

### File Validation

- File size limits prevent abuse
- Extension filtering blocks dangerous files
- MIME type validation ensures file integrity

### Access Control

```go
// Files are automatically associated with the authenticated user
file.UploadedBy = ctx.UserID

// Only file owner or admin can access
if file.UploadedBy != ctx.UserID && !ctx.IsRoot {
    return ErrUnauthorized
}
```

## 🚀 Deployment Scenarios

### Local Development

```json
{
  "type": "local",
  "local": {
    "basePath": "./uploads"
  },
  "maxFileSize": 10485760
}
```

### Production with AWS S3

```json
{
  "type": "s3",
  "s3": {
    "bucket": "myapp-files-prod",
    "region": "us-east-1",
    "accessKeyId": "${AWS_ACCESS_KEY_ID}",
    "secretAccessKey": "${AWS_SECRET_ACCESS_KEY}"
  },
  "maxFileSize": 104857600
}
```

### Self-hosted with MinIO

```json
{
  "type": "s3",
  "s3": {
    "bucket": "uploads",
    "region": "us-east-1",
    "endpoint": "https://minio.mycompany.com",
    "accessKeyId": "${MINIO_ACCESS_KEY}",
    "secretAccessKey": "${MINIO_SECRET_KEY}",
    "forcePathStyle": true
  },
  "maxFileSize": 104857600
}
```

## 🎯 Event System

The files collection supports custom Go events for validation, processing, and access control. Events are stored in `/resources/files/`:

### Available Events

| Event | File | Purpose | When Triggered |
|-------|------|---------|----------------|
| **post.go** | `/resources/files/post.go` | Validate uploads before processing | Before file upload |
| **aftercommit.go** | `/resources/files/aftercommit.go` | Process after successful upload | After file is saved |
| **get.go** | `/resources/files/get.go` | Control file access | Before file download/list |
| **delete.go** | `/resources/files/delete.go` | Validate file deletion | Before file deletion |

### Event Examples

#### post.go - Upload Validation
```go
package main

import (
    "fmt"
    "strings"
)

func Run(ctx *EventContext) error {
    // Reject files larger than 5MB
    if size, ok := ctx.Data["size"].(float64); ok && size > 5*1024*1024 {
        ctx.Cancel("File size exceeds 5MB limit", 400)
        return nil
    }
    
    // Only allow specific file types
    allowedTypes := []string{
        "image/jpeg",
        "image/png", 
        "application/pdf",
    }
    
    if contentType, ok := ctx.Data["contentType"].(string); ok {
        allowed := false
        for _, t := range allowedTypes {
            if t == contentType {
                allowed = true
                break
            }
        }
        if !allowed {
            ctx.Cancel(fmt.Sprintf("File type not allowed. Allowed types: %s", 
                strings.Join(allowedTypes, ", ")), 400)
            return nil
        }
    }
    
    // Add custom metadata
    ctx.Data["category"] = "user-upload"
    ctx.Data["processed"] = false
    
    return nil
}
```

#### aftercommit.go - Post-Processing
```go
package main

import (
    "fmt"
    "strings"
)

func Run(ctx *EventContext) error {
    // Log file upload
    ctx.Log("File uploaded", map[string]interface{}{
        "id":   ctx.Data["id"],
        "name": ctx.Data["originalName"],
        "user": ctx.Data["uploadedBy"],
    })
    
    // Emit real-time event
    if ctx.Emit != nil {
        ctx.Emit("file-uploaded", ctx.Data)
    }
    
    // Trigger image processing for images
    if contentType, ok := ctx.Data["contentType"].(string); ok {
        if strings.HasPrefix(contentType, "image/") {
            // Queue for thumbnail generation
            ctx.Data["needsThumbnail"] = true
        }
    }
    
    return nil
}
```

#### get.go - Access Control
```go
package main

import "fmt"

func Run(ctx *EventContext) error {
    // Require authentication
    if ctx.Me == nil || ctx.Me["id"] == nil {
        ctx.Cancel("Authentication required", 401)
        return nil
    }
    
    // For specific file access
    if ctx.Data != nil && ctx.Data["id"] != nil {
        // Only owner or admin can access
        if !ctx.IsRoot && ctx.Data["uploadedBy"] != ctx.Me["id"] {
            ctx.Cancel("Access denied", 403)
            return nil
        }
        
        // Log file access
        ctx.Log("File accessed", map[string]interface{}{
            "fileId": ctx.Data["id"],
            "userId": ctx.Me["id"],
        })
    }
    
    return nil
}
```

#### delete.go - Deletion Control
```go
package main

import (
    "fmt"
    "time"
)

func Run(ctx *EventContext) error {
    // Only file owner or admin can delete
    if !ctx.IsRoot && ctx.Data["uploadedBy"] != ctx.Me["id"] {
        ctx.Cancel("Only file owner can delete", 403)
        return nil
    }
    
    // Prevent deletion of recent files
    if uploadedAt, ok := ctx.Data["uploadedAt"].(string); ok {
        uploadTime, err := time.Parse(time.RFC3339, uploadedAt)
        if err == nil {
            hourAgo := time.Now().Add(-time.Hour)
            if uploadTime.After(hourAgo) && !ctx.IsRoot {
                ctx.Cancel("Cannot delete files uploaded within the last hour", 400)
                return nil
            }
        }
    }
    
    ctx.Log("File deletion authorized", map[string]interface{}{
        "id":        ctx.Data["id"],
        "name":      ctx.Data["originalName"],
        "deletedBy": ctx.Me["id"],
    })
    
    return nil
}
```

### Event Context

Events receive an `*EventContext` struct with:

| Property | Type | Description |
|----------|------|-------------|
| `ctx.Data` | `map[string]interface{}` | File metadata (modifiable) |
| `ctx.Me` | `map[string]interface{}` | Current user info |
| `ctx.IsRoot` | `bool` | Admin user flag |
| `ctx.Method` | `string` | HTTP method |
| `ctx.Query` | `map[string]interface{}` | Query parameters |
| `ctx.Cancel(msg, code)` | `func(string, int)` | Cancel operation with error |
| `ctx.Log(msg, data)` | `func(string, map[string]interface{})` | Log messages |
| `ctx.Emit(event, data)` | `func(string, interface{})` | Emit real-time events |

### Creating Custom Events

1. Create a Go file in `/resources/files/`
2. Name it according to the event type: `post.go`, `get.go`, `delete.go`, or `aftercommit.go`
3. Implement the `Run(ctx *EventContext) error` function
4. Restart the server to compile and load new events

### Real-time Events

File operations emit WebSocket events automatically:
- `created` - When a file is uploaded
- `deleted` - When a file is deleted

Listen for these events on the client:
```javascript
dpd.on('files:created', function(file) {
    console.log('New file uploaded:', file.originalName);
});

dpd.on('files:deleted', function(file) {
    console.log('File deleted:', file.originalName);
});
```

## 🔄 Migration Between Storage Types

Files can be migrated between storage types using the built-in migration tools:

```bash
# Export file metadata
curl -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  http://localhost:2403/files/export > files.json

# Switch storage configuration in .deployd/storage.json

# Import to new storage
curl -X POST \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @files.json \
  http://localhost:2403/files/import
```

## 🐛 Troubleshooting

### Common Issues

**File Upload Fails**
- Check file size limits in configuration
- Verify file extension is allowed
- Ensure proper authentication headers

**S3/MinIO Connection Issues**
- Verify credentials in environment variables
- Check bucket permissions
- Test endpoint connectivity

**Permission Denied**
- Ensure user is authenticated
- Check if user owns the file
- Verify admin permissions for global access

### Debug Mode

Enable debug logging for file operations:

```bash
export DEPLOYD_LOG_LEVEL=debug
npm run dev
```

## 📈 Performance Considerations

### Large Files

For files larger than 10MB, consider:
- Using signed URLs for direct upload to S3/MinIO
- Implementing chunked uploads
- Adding progress tracking

### High Volume

For high-volume scenarios:
- Use S3/MinIO for better scalability
- Implement CDN for file delivery
- Add caching for file metadata

### Storage Optimization

- Enable compression for text files
- Use appropriate S3 storage classes
- Implement lifecycle policies for old files

## 🌐 Web Interface

The built-in file management interface is available at:
- **Main Dashboard**: `http://localhost:2403/_dashboard/`
- **Self-Test Interface**: `http://localhost:2403/web/self-test.html`

The self-test interface includes:
- File upload with progress tracking
- File listing and management
- Download and delete operations
- dpd.js code examples

## 📋 Best Practices

1. **Always set file size limits** to prevent abuse
2. **Use environment variables** for production credentials
3. **Implement proper error handling** in client code
4. **Validate file types** on both client and server
5. **Use signed URLs** for large file uploads
6. **Monitor storage usage** and implement cleanup policies
7. **Test file operations** in your deployment environment

## 🔗 Related Documentation

- [Authentication Guide](./authentication.md)
- [Storage Configuration](./storage-config.md)
- [API Reference](./api-reference.md)
- [Deployment Guide](./deployment.md)