# Security Policy

## Supported Versions

We release patches for security vulnerabilities. The following versions are currently supported:

| Version | Supported          |
| ------- | ------------------ |
| latest  | ✅ Yes             |
| v0.x    | ✅ Security fixes |

We recommend always using the latest release for security patches and improvements.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

1. **Do NOT** create a public GitHub issue for security vulnerabilities
2. Email the maintainer directly at: `openpost+security@rgo.pt`
3. Include in your report:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact
   - Any suggested fixes (optional)

### What to Expect

- **Acknowledgment:** We will acknowledge your report within 48 hours
- **Status Update:** We will provide regular updates on the progress of addressing the vulnerability
- **Disclosure:** We will publicly disclose the vulnerability after a fix is available
- **Credit:** We will credit reporters in the security advisory (unless you prefer to remain anonymous)

### Response Timeline

- **Initial response:** Within 48 hours
- **Severity assessment:** Within 7 days
- **Fix development:** Varies by complexity (typically 1-4 weeks)
- **Public disclosure:** After patch is available

## Security Best Practices for Self-Hosters

When running OpenPost, follow these security practices:

### Secrets Management

- **Never commit `.env`** files to version control
- Use strong, randomly generated secrets (`openssl rand -base64 32`)
- Rotate secrets periodically
- Use Docker secrets, Kubernetes secrets, or a secrets manager in production

### Network Security

- Run behind a reverse proxy with TLS/HTTPS
- Configure a proper firewall
- Don't expose the OpenPost port directly to the internet (except for OAuth callbacks)
- For Threads: ensure `/media/` endpoint is publicly accessible for media uploads

### Data Protection

- Back up your database and media directory regularly
- Store backups in a secure location
- Consider encrypting backups at rest

### OAuth Provider Security

- Regularly review connected accounts
- Rotate OAuth tokens and secrets periodically
- Revoke access for accounts no longer in use

## Security Features in OpenPost

OpenPost includes the following security measures:

- **Token encryption:** All OAuth tokens are encrypted at rest using AES-256-GCM
- **Password hashing:** User passwords are hashed with bcrypt
- **JWT authentication:** Secure token-based session management
- **OAuth PKCE:** Twitter authentication uses PKCE for improved security
- **No external dependencies:** Self-contained binary with no external service dependencies

## Third-Party Dependencies

We regularly update dependencies to address security vulnerabilities. We use:

- Go dependencies (via Go modules)
- Frontend dependencies (via Bun/npm)
- Docker base images from trusted sources

## Scope

This security policy applies to:

- The OpenPost server application
- The embedded SvelteKit frontend
- OAuth integration with supported platforms

This does not apply to:

- Third-party OAuth providers (Twitter, Mastodon, Bluesky, LinkedIn, Threads)
- Your deployment infrastructure (reverse proxy, firewall, etc.)
- External databases or storage systems you configure