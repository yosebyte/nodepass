# Security Policy

## Supported Versions

We provide security updates for the following versions of NodePass:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |

## Security Features

NodePass implements multiple security layers:

### TLS Encryption Modes

- **TLS Mode 0**: Unencrypted mode for trusted networks (highest performance)
- **TLS Mode 1**: Self-signed certificates with TLS 1.3 (balanced security)
- **TLS Mode 2**: Custom certificate validation for enterprise security

### Network Security

- Password-based tunnel authentication
- Connection pooling with capacity limits
- Graceful degradation under load
- Configurable timeout and retry mechanisms

## Reporting Security Vulnerabilities

We take security seriously. If you discover a security vulnerability in NodePass, please report it responsibly.

### Where to Report

- **Email**: team@mail.nodepass.eu
- **Subject**: [SECURITY] Brief description of the issue

### What to Include

Please provide the following information:

1. **Description** of the vulnerability
2. **Steps to reproduce** the issue
3. **Potential impact** and affected versions
4. **Your contact information** for follow-up
5. **Any proof-of-concept code** (if applicable)

### Response Process

1. **Acknowledgment**: We will acknowledge receipt within 48 hours
2. **Assessment**: Initial assessment within 5 business days
3. **Updates**: Regular updates on investigation progress
4. **Resolution**: Security patch and public disclosure coordination

### Responsible Disclosure

- Please **do not** create public GitHub issues for security vulnerabilities
- Give us reasonable time to investigate and patch the issue
- We will coordinate public disclosure timing with you
- Security researchers will be credited in our security advisories

## Security Best Practices

### For Users

- **Use TLS Mode 1 or 2** in production environments
- **Choose strong passwords** for tunnel authentication
- **Keep NodePass updated** to the latest version
- **Monitor logs** for suspicious activity
- **Limit network exposure** by binding to specific interfaces
- **Use firewall rules** to restrict access to tunnel ports

### For Developers

- **Validate all inputs** including URL parameters and network data
- **Use secure coding practices** following Go security guidelines
- **Implement proper error handling** without leaking sensitive information
- **Test security features** thoroughly before release
- **Follow the principle of least privilege** in code design

## Security Architecture

### Network Layer

- TLS 1.3 encryption for secure data transmission
- Certificate validation and auto-reload capabilities
- Protection against common network attacks

### Application Layer

- Input validation and sanitization
- Secure memory handling for sensitive data
- Proper resource cleanup and connection management

### Operational Security

- Minimal container image based on scratch
- No unnecessary dependencies or services
- Clear separation of concerns between components

## Known Security Considerations

### TLS Mode 0 Usage

- Only use in completely trusted networks
- Not recommended for internet-facing deployments
- Provides maximum performance at the cost of encryption

### Master API Security

- Secure the API endpoint with proper authentication
- Use reverse proxy for additional security layers
- Monitor API access and implement rate limiting

## Security Updates

Security updates are released as:

- **Patch releases** for critical vulnerabilities
- **Minor releases** for security enhancements
- **Documentation updates** for security best practices

Subscribe to our release notifications:
- [GitHub Releases](https://github.com/yosebyte/nodepass/releases)
- [Telegram Channel](https://t.me/NodePassChannel)

## Ecosystem Security

### NodePassProject Libraries

Our core dependencies are maintained by the NodePassProject organization:

- **cert**: Certificate generation and management
- **conn**: Secure connection handling
- **logs**: Secure logging with sensitive data protection
- **pool**: Connection pool management with resource limits

### Third-Party Dependencies

- We minimize external dependencies
- All dependencies are regularly audited for security issues
- Updates are applied promptly when security issues are discovered

## Contact Information

For security-related questions or concerns:

- **Security Team**: team@mail.nodepass.eu
- **General Issues**: [GitHub Issues](https://github.com/yosebyte/nodepass/issues)
- **Community**: [Telegram Group](https://t.me/NodePassGroup)

## Attribution

We appreciate security researchers who help improve NodePass security. Contributors to our security will be acknowledged in:

- Security advisories
- Release notes
- Our contributors list

---

**Note**: This security policy applies to the NodePass core project. For security issues in ecosystem projects (NodePassDash, NodePanel, etc.), please refer to their respective repositories in the [NodePassProject](https://github.com/NodePassProject) organization.
