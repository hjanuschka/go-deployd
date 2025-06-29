# Public Files Directory

This directory serves static files publicly at `/public/*` URLs.

## Usage

Place any files you want to serve publicly in this directory:

- **Images**: `/public/images/logo.png` → `http://localhost:2403/public/images/logo.png`
- **CSS**: `/public/styles/app.css` → `http://localhost:2403/public/styles/app.css`
- **JavaScript**: `/public/js/client.js` → `http://localhost:2403/public/js/client.js`
- **Documents**: `/public/docs/manual.pdf` → `http://localhost:2403/public/docs/manual.pdf`

## Security

- Files are served as-is without authentication
- Directory listing is disabled for security
- Hidden files (starting with `.`) are not served
- Paths are sanitized to prevent directory traversal attacks

## Examples

```html
<!-- In your HTML -->
<link rel="stylesheet" href="/public/styles/app.css">
<script src="/public/js/app.js"></script>
<img src="/public/images/logo.png" alt="Logo">
```

## File Types

All standard MIME types are supported and set automatically based on file extension.