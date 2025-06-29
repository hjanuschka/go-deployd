# Local File Uploads Directory

This directory is used for local file storage when using the "local" storage type in Go-Deployd.

## Structure

Files are organized by upload date:
```
uploads/
├── 2025/
│   └── 06/
│       └── 29/
│           ├── a1b2c3d4e5f6.jpg
│           ├── x7y8z9a1b2c3.pdf
│           └── ...
└── README.md
```

## Configuration

Local storage is configured in `.deployd/storage.json`:

```json
{
  "type": "local",
  "local": {
    "basePath": "uploads",
    "urlPrefix": "/files"
  }
}
```

## Security Notes

- This directory should be included in your `.gitignore` for production
- Files are served through the Go-Deployd API with authentication
- Direct file system access is controlled by the application

## Development

For local development, this directory will be created automatically when you upload files. No manual setup required!

## Production

In production, consider using S3 or MinIO for better scalability and performance. See the documentation at `docs/file-storage.md` for configuration details.