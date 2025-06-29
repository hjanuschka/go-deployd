# File Storage Documentation

Go-Deployd provides a comprehensive file storage system that supports local storage, Amazon S3, and MinIO with a unified API. This makes it easy to start with local development and scale to planet-scale cloud storage.

## 🚀 Quick Start

### Local Development (Zero Configuration)

By default, Go-Deployd uses local file storage - perfect for development:

```bash
# Start the server - files will be stored in ./uploads/
npm run dev

# Upload a file
curl -X POST http://localhost:2403/files \
  -F "file=@myimage.jpg" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Production with S3/MinIO

Configure cloud storage by copying the example configuration:

```bash
# Copy S3 example
cp .deployd/storage.s3.example.json .deployd/storage.json

# Or copy MinIO example  
cp .deployd/storage.minio.example.json .deployd/storage.json
```

## 📋 Configuration

Storage is configured via `.deployd/storage.json`:

### Local Storage Configuration

```json
{
  "type": "local",
  "local": {
    "basePath": "uploads",
    "urlPrefix": "/files"
  },
  "maxFileSize": 52428800,
  "allowedExtensions": ["jpg", "jpeg", "png", "gif", "pdf"],
  "signedUrlExpiration": 3600
}
```

### S3 Configuration

```json
{
  "type": "s3",
  "s3": {
    "region": "us-east-1",
    "bucket": "my-app-files",
    "accessKeyId": "AKIA...",
    "secretAccessKey": "...",
    "useSSL": true,
    "pathStyle": false
  },
  "maxFileSize": 52428800,
  "allowedExtensions": [],
  "signedUrlExpiration": 3600
}
```

### MinIO Configuration

```json
{
  "type": "minio",
  "s3": {
    "endpoint": "http://localhost:9000",
    "region": "us-east-1", 
    "bucket": "my-app-files",
    "accessKeyId": "minioadmin",
    "secretAccessKey": "minioadmin",
    "useSSL": false,
    "pathStyle": true
  },
  "maxFileSize": 104857600,
  "allowedExtensions": [],
  "signedUrlExpiration": 3600
}
```

### Environment Variables

For security, use environment variables for credentials:

```bash
export STORAGE_ACCESS_KEY="your-access-key"
export STORAGE_SECRET_KEY="your-secret-key"
```

## 🔌 API Endpoints

The files resource provides a RESTful API at `/files`:

### Upload File

**Direct Upload:**
```bash
POST /files
Content-Type: multipart/form-data

curl -X POST http://localhost:2403/files \
  -F "file=@myimage.jpg" \
  -F "description=My awesome image" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "id": "a1b2c3d4e5f6",
  "filename": "a1b2c3d4e5f6.jpg",
  "originalName": "myimage.jpg",
  "contentType": "image/jpeg",
  "size": 245760,
  "storageType": "local",
  "url": "http://localhost:2403/files/a1b2c3d4e5f6",
  "uploadedAt": "2025-06-29T12:00:00Z",
  "uploadedBy": "user123",
  "metadata": {
    "description": "My awesome image"
  }
}
```

### Generate Signed URL (for direct uploads)

```bash
POST /files/signed
Content-Type: application/json

{
  "filename": "myimage.jpg",
  "contentType": "image/jpeg",
  "expiresIn": 3600
}
```

**Response:**
```json
{
  "signedUrl": "https://s3.amazonaws.com/bucket/path?signature=...",
  "fileID": "a1b2c3d4e5f6",
  "filename": "myimage.jpg",
  "contentType": "image/jpeg",
  "expiresIn": 3600,
  "completeUrl": "http://localhost:2403/files/complete/a1b2c3d4e5f6"
}
```

### Download File

```bash
GET /files/{fileID}

curl http://localhost:2403/files/a1b2c3d4e5f6 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Get File Metadata

```bash
GET /files/{fileID}?info=true

curl http://localhost:2403/files/a1b2c3d4e5f6?info=true \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Get Signed Download URL

```bash
GET /files/{fileID}?signed=true&expires=3600

curl http://localhost:2403/files/a1b2c3d4e5f6?signed=true \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### List Files

```bash
GET /files?limit=50&offset=0

curl http://localhost:2403/files?limit=10 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "files": [
    {
      "id": "a1b2c3d4e5f6",
      "filename": "a1b2c3d4e5f6.jpg",
      "originalName": "myimage.jpg",
      "contentType": "image/jpeg",
      "size": 245760,
      "uploadedAt": "2025-06-29T12:00:00Z",
      "url": "http://localhost:2403/files/a1b2c3d4e5f6"
    }
  ],
  "count": 1,
  "limit": 10,
  "offset": 0
}
```

### Delete File

```bash
DELETE /files/{fileID}

curl -X DELETE http://localhost:2403/files/a1b2c3d4e5f6 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 🔒 Security & Permissions

### Authentication

All file operations require authentication unless configured otherwise:

- **JWT Authentication**: Include `Authorization: Bearer <token>` header
- **Master Key**: Include `X-Master-Key: <key>` header for admin access

### File Ownership

- Users can only access files they uploaded
- Root/admin users can access all files
- Anonymous users cannot upload or access files

### File Validation

Configure allowed file types and size limits:

```json
{
  "maxFileSize": 52428800,
  "allowedExtensions": ["jpg", "jpeg", "png", "gif", "pdf", "txt"]
}
```

Empty `allowedExtensions` array allows all file types.

## 🌐 JavaScript Client (dpd.js)

Use the dpd.js client for easy file operations:

```javascript
// Upload file
const fileInput = document.getElementById('fileInput');
const file = fileInput.files[0];

const formData = new FormData();
formData.append('file', file);
formData.append('description', 'My uploaded file');

dpd.files.post(formData, (err, result) => {
  if (err) {
    console.error('Upload failed:', err);
  } else {
    console.log('File uploaded:', result);
  }
});

// List files
dpd.files.get({ limit: 10 }, (err, result) => {
  if (err) {
    console.error('Failed to list files:', err);
  } else {
    console.log('Files:', result.files);
  }
});

// Download file
dpd.files.get(fileId, (err, blob) => {
  if (err) {
    console.error('Download failed:', err);
  } else {
    // Create download link
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'filename.jpg';
    a.click();
  }
});

// Delete file
dpd.files.delete(fileId, (err) => {
  if (err) {
    console.error('Delete failed:', err);
  } else {
    console.log('File deleted successfully');
  }
});
```

## 🔧 Advanced Features

### Custom Metadata

Add custom metadata to files during upload:

```bash
curl -X POST http://localhost:2403/files \
  -F "file=@document.pdf" \
  -F "category=legal" \
  -F "project=website-redesign" \
  -F "version=1.0" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Signed URLs for Direct Upload

For large files or client-side uploads, use signed URLs:

1. **Get signed URL**: `POST /files/signed`
2. **Upload directly**: Use the signed URL to upload to S3/MinIO
3. **Complete upload**: Call completion endpoint (for local storage)

### MinIO Development Setup

Run MinIO locally for S3-compatible development:

```bash
# Run MinIO in Docker
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  --name minio \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"

# Access MinIO Console at http://localhost:9001
# Login: minioadmin / minioadmin
```

## 📊 Scaling to Planet Scale

### Local → S3 Migration

1. **Start with local storage** for development
2. **Switch to S3** by updating `storage.json`
3. **Migrate existing files** using the migration script (coming soon)

### Multi-Region Deployment

For global applications:

- Use S3 with CloudFront for global CDN
- Configure regional S3 buckets
- Implement geo-routing in your application

### Performance Optimization

- **Use signed URLs** for large file uploads
- **Implement client-side compression** before upload
- **Configure S3 lifecycle policies** for cost optimization
- **Use S3 Transfer Acceleration** for faster uploads

## 🚨 Error Handling

Common error responses:

### File Too Large
```json
{
  "error": "Request Entity Too Large",
  "message": "file too large",
  "status": 413
}
```

### Invalid File Type
```json
{
  "error": "Unsupported Media Type", 
  "message": "invalid file type: .exe",
  "status": 415
}
```

### File Not Found
```json
{
  "error": "Not Found",
  "message": "File not found",
  "status": 404
}
```

### Permission Denied
```json
{
  "error": "Forbidden",
  "message": "Permission denied", 
  "status": 403
}
```

## 💡 Best Practices

1. **Use signed URLs** for large files (>10MB)
2. **Validate file types** on both client and server
3. **Implement progress indicators** for uploads
4. **Use environment variables** for credentials
5. **Set up proper CORS** for client-side uploads
6. **Monitor storage costs** with S3/MinIO
7. **Implement file cleanup** for temporary files
8. **Use CDN** for frequently accessed files