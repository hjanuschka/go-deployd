# Security Policy

## Supported Versions

We actively maintain security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you believe you have found a security vulnerability in Go-Deployd, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please send an email to: security@go-deployd.dev (or create a private GitHub security advisory)

Include the following information:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 1 week
- **Fix Development**: Varies based on complexity
- **Public Disclosure**: After fix is released and users have had time to update

### Security Best Practices

When deploying Go-Deployd in production:

1. **Change Default Credentials**
   - Generate secure JWT secrets
   - Use strong master keys
   - Never use default passwords

2. **Secure Database Access**
   - Use authentication for all database connections
   - Enable SSL/TLS for database connections
   - Follow principle of least privilege

3. **Network Security**
   - Use HTTPS in production
   - Configure firewalls appropriately
   - Consider using a reverse proxy

4. **Regular Updates**
   - Keep Go-Deployd updated to the latest version
   - Monitor security advisories
   - Update dependencies regularly

5. **Configuration Security**
   - Store secrets in environment variables
   - Use secure file permissions
   - Enable audit logging

For detailed security configuration, see our [Production Deployment Guide](./docs/production-deployment.md).

## Known Security Considerations

### Authentication
- JWT tokens are stateless; consider token rotation for high-security environments
- Master key provides full admin access; protect it accordingly
- Rate limiting is not built-in; implement at the reverse proxy level

### Database Security
- SQLite files should have appropriate file permissions (600)
- MongoDB and MySQL connections should use authentication and SSL
- Consider database-level encryption for sensitive data

### Event System Security
- JavaScript events run in isolated V8 contexts
- Go events run as compiled plugins with full system access
- Validate all user inputs in custom events

## Vulnerability Disclosure Policy

When a security vulnerability is confirmed:

1. We will work to develop a fix
2. We will coordinate disclosure with the reporter
3. We will release a security advisory with details
4. We will credit the reporter (unless they prefer to remain anonymous)

Thank you for helping keep Go-Deployd secure!