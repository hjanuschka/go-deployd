# Production Deployment Guide

## Overview

This guide covers best practices for deploying Go-Deployd in production environments, including security hardening, performance optimization, and monitoring.

## Security Hardening

### 1. Environment Variables (Required)

**Never use default values in production!**

```bash
# Security (REQUIRED)
export JWT_SECRET="your-cryptographically-secure-secret-256-bits"
export MASTER_KEY="mk_your_secure_master_key_for_admin_access"

# Database
export DATABASE_URL="your-production-database-url"

# SMTP (if using email features)
export SMTP_HOST="your-smtp-server.com"
export SMTP_PORT="587"
export SMTP_USER="your-smtp-username"
export SMTP_PASS="your-smtp-password"

# Server
export PORT="2403"
export PRODUCTION="true"
export DEVELOPMENT="false"

# Redis (for multi-server real-time events)
export REDIS_URL="redis://your-redis-server:6379"
```

### 2. Generate Secure Keys

```bash
# Generate JWT secret (256-bit)
openssl rand -hex 32

# Generate master key
echo "mk_$(openssl rand -hex 32)"
```

### 3. Database Security

#### SQLite Production Setup
```bash
# Use dedicated database file with restricted permissions
export DATABASE_URL="sqlite://./data/production.db"
mkdir -p data
chmod 700 data
```

#### MySQL Production Setup
```bash
# Use connection with SSL and limited privileges
export DATABASE_URL="mysql://app_user:secure_password@mysql-server:3306/app_db?tls=true"
```

#### MongoDB Production Setup
```bash
# Use authentication and SSL
export DATABASE_URL="mongodb://app_user:secure_password@mongo-server:27017/app_db?ssl=true"
```

### 4. Network Security

#### Firewall Configuration
```bash
# Allow only necessary ports
ufw allow 22/tcp    # SSH
ufw allow 2403/tcp  # Go-Deployd
ufw enable
```

#### Reverse Proxy (Recommended)
Use nginx or similar reverse proxy:

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name yourdomain.com;
    
    ssl_certificate /path/to/certificate.crt;
    ssl_certificate_key /path/to/private.key;
    
    location / {
        proxy_pass http://localhost:2403;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Performance Optimization

### 1. Database Performance

#### Enable Column-Based Storage
```json
{
  "properties": {
    "email": {"type": "string", "index": true},
    "status": {"type": "string", "index": true},
    "createdAt": {"type": "date", "index": true}
  },
  "options": {
    "useColumns": true
  }
}
```

#### Database Indexes
- Index frequently queried fields
- Use compound indexes for multi-field queries
- Monitor query performance

### 2. Caching Strategy

#### Redis for Real-time Events
```bash
# Multi-server WebSocket support
export REDIS_URL="redis://redis-server:6379"
```

#### HTTP Caching Headers
Configure appropriate cache headers for static assets.

### 3. Resource Limits

#### systemd Service
```ini
[Unit]
Description=Go-Deployd API Server
After=network.target

[Service]
Type=simple
User=deployd
WorkingDirectory=/opt/go-deployd
ExecStart=/opt/go-deployd/deployd
Restart=always
RestartSec=5

# Resource limits
LimitNOFILE=65536
MemoryLimit=1G
CPUQuota=200%

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/go-deployd/data

[Install]
WantedBy=multi-user.target
```

## Docker Deployment

### 1. Production Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

# Build application
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:latest

# Install CA certificates
RUN apk --no-cache add ca-certificates tzdata
RUN update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S deployd && \
    adduser -S deployd -u 1001 -G deployd

# Set up application directory
WORKDIR /app
RUN mkdir -p data uploads && \
    chown -R deployd:deployd /app

# Copy binary and resources
COPY --from=builder /app/bin/go-deployd /usr/local/bin/deployd
COPY --from=builder --chown=deployd:deployd /app/resources ./resources
COPY --from=builder --chown=deployd:deployd /app/web ./web

# Switch to non-root user
USER deployd

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:2403/health || exit 1

EXPOSE 2403
CMD ["deployd"]
```

### 2. Docker Compose (Production)

```yaml
version: '3.8'

services:
  go-deployd:
    build: .
    ports:
      - "2403:2403"
    environment:
      - DATABASE_URL=mysql://app:${MYSQL_PASSWORD}@mysql:3306/app
      - JWT_SECRET=${JWT_SECRET}
      - MASTER_KEY=${MASTER_KEY}
      - REDIS_URL=redis://redis:6379
      - PRODUCTION=true
    volumes:
      - ./data:/app/data
      - ./uploads:/app/uploads
    depends_on:
      - mysql
      - redis
    restart: unless-stopped
    
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=app
      - MYSQL_USER=app
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped
    
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - go-deployd
    restart: unless-stopped

volumes:
  mysql_data:
```

## Monitoring and Logging

### 1. Health Checks

Built-in health endpoint:
```bash
curl http://localhost:2403/health
```

### 2. Logging Configuration

```bash
# Structured JSON logging
export LOG_FORMAT="json"
export LOG_LEVEL="info"

# Log rotation (using logrotate)
/opt/go-deployd/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    create 644 deployd deployd
    postrotate
        systemctl reload go-deployd
    endscript
}
```

### 3. Monitoring Metrics

Monitor these key metrics:
- Response time
- Database connection pool usage
- Memory usage
- WebSocket connection count
- Error rates
- Authentication failures

## Backup Strategy

### 1. Database Backups

#### SQLite
```bash
# Automated backup script
#!/bin/bash
sqlite3 /opt/go-deployd/data/production.db ".backup /backups/deployd-$(date +%Y%m%d-%H%M%S).db"
```

#### MySQL
```bash
mysqldump -u app -p app > /backups/deployd-$(date +%Y%m%d-%H%M%S).sql
```

#### MongoDB
```bash
mongodump --db app --out /backups/deployd-$(date +%Y%m%d-%H%M%S)
```

### 2. Application Backups

```bash
# Backup configuration and uploads
tar -czf /backups/deployd-config-$(date +%Y%m%d).tar.gz \
  /opt/go-deployd/resources \
  /opt/go-deployd/uploads \
  /opt/go-deployd/.deployd
```

## Scaling and High Availability

### 1. Load Balancing

Use multiple Go-Deployd instances behind a load balancer:

```nginx
upstream go_deployd {
    server server1:2403;
    server server2:2403;
    server server3:2403;
}

server {
    location / {
        proxy_pass http://go_deployd;
    }
}
```

### 2. Database Scaling

#### Read Replicas
Configure read replicas for heavy read workloads.

#### Connection Pooling
Tune database connection pool settings for your workload.

### 3. WebSocket Scaling

Use Redis for cross-server real-time events:
```bash
export REDIS_URL="redis://redis-cluster:6379"
```

## Security Checklist

- [ ] Changed default JWT secret and master key
- [ ] Configured HTTPS with valid SSL certificates
- [ ] Set up firewall rules
- [ ] Configured secure database access
- [ ] Enabled audit logging
- [ ] Set up automated security updates
- [ ] Configured backup and restore procedures
- [ ] Set up monitoring and alerting
- [ ] Reviewed and secured all environment variables
- [ ] Disabled debug mode in production
- [ ] Configured rate limiting
- [ ] Set up intrusion detection

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Check for memory leaks in custom events
   - Monitor WebSocket connection count
   - Review database query efficiency

2. **Slow Database Queries**
   - Enable column-based storage for frequently queried fields
   - Add appropriate database indexes
   - Monitor query execution times

3. **WebSocket Connection Issues**
   - Check Redis connectivity
   - Verify firewall allows WebSocket traffic
   - Monitor connection pool limits

### Getting Help

- Check logs: `/var/log/go-deployd/`
- Enable debug mode temporarily: `DEBUG=true`
- Review documentation: [docs/](./index.md)
- Report issues: [GitHub Issues](https://github.com/hjanuschka/go-deployd/issues)

---

**Remember**: Security is an ongoing process. Regularly review and update your production configuration as your application grows.