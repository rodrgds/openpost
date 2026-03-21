<p align="center">
  <a href="https://github.com/openpost/openpost" target="_blank">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://via.placeholder.com/280x80/1a1a1a/22c55e?text=OpenPost">
    <img alt="OpenPost Logo" src="https://via.placeholder.com/280x80/ffffff/16a34a?text=OpenPost" width="280"/>
  </picture>
  </a>
</p>

<p align="center">
<a href="LICENSE">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
</a>
<a href="https://golang.org">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
</a>
<a href="https://svelte.dev">
  <img src="https://img.shields.io/badge/Svelte-5-FF3E00?style=flat&logo=svelte" alt="Svelte Version">
</a>
</p>

<div align="center">
  <strong>
  <h2>A lightweight, self-hosted social media scheduler</h2><br />
  </strong>
  The open-source alternative to Typefully, Buffer, and Hypefury.<br />
  One binary. No dependencies. Full control.
</div>

<div align="center">
  <br />
  <img alt="X (Twitter)" src="https://upload.wikimedia.org/wikipedia/commons/5/57/X_logo_2023_%28white%29.png" width="24" style="background: black; border-radius: 4px;">
  &nbsp;
  <img alt="Mastodon" src="https://upload.wikimedia.org/wikipedia/commons/4/48/Mastodon_logo_%28simple%29.svg" width="24">
  &nbsp;
  <img alt="Bluesky" src="https://upload.wikimedia.org/wikipedia/commons/e/e1/Bluesky_app_logo.png" width="24" style="border-radius: 4px;">
  &nbsp;
  <img alt="Threads" src="https://upload.wikimedia.org/wikipedia/commons/9/95/Threads_logo.svg" width="24">
  &nbsp;
  <img alt="LinkedIn" src="https://upload.wikimedia.org/wikipedia/commons/c/ca/LinkedIn_logo_initials.png" width="24">
</div>

<p align="center">
  <br />
  <a href="#-features"><strong>Features</strong></a>
  ·
  <a href="#-supported-platforms"><strong>Platforms</strong></a>
  ·
  <a href="#-quick-start"><strong>Quick Start</strong></a>
  ·
  <a href="#-configuration"><strong>Configuration</strong></a>
  ·
  <a href="#-tech-stack"><strong>Tech Stack</strong></a>
</p>

<br />

## 🚀 Features

- **Single Binary Deployment** - Everything compiled into one executable. No Docker required (but supported).
- **Multi-Platform Posting** - Schedule posts to X (Twitter), Mastodon, Bluesky, Threads, and LinkedIn.
- **Custom Mastodon Instances** - Connect accounts from any Mastodon server.
- **Workspaces & Teams** - Multi-tenant architecture with role-based access control.
- **Background Scheduling** - SQLite-backed job queue survives server restarts.
- **Encrypted Tokens** - AES-256-GCM encryption for all OAuth tokens at rest.
- **Modern UI** - SvelteKit frontend with TailwindCSS, responsive design.
- **Self-Hosted** - Your data stays on your server. No third-party tracking.

## 📱 Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| X (Twitter) | ✅ Supported | OAuth 2.0 PKCE |
| Mastodon | ✅ Supported | Custom instances |
| Bluesky | 🔜 Planned | AT Protocol + DPoP |
| Threads | 🔜 Planned | Instagram Graph API |
| LinkedIn | 🔜 Planned | OAuth 2.0 |

## ⚡ Quick Start

### Prerequisites

- Go 1.25+ (for building from source)
- Bun or npm (for frontend builds)
- OAuth credentials for the platforms you want to use

### Building from Source

```bash
# Clone the repository
git clone https://github.com/openpost/openpost.git
cd openpost

# Install frontend dependencies and build
cd web
bun install
bun run build

# Build the Go binary (includes embedded frontend)
cd ../backend
cp .env.example .env
# Edit .env with your credentials
go build -o openpost ./cmd/openpost

# Run
./openpost
```

The application will start on `http://localhost:8080`.

### One-Liner Build

```bash
# From project root
cd web && bun install && bun run build && cd ../backend && go build -o openpost ./cmd/openpost
```

### Development Mode

For development, you can run the frontend and backend separately:

```bash
# Terminal 1: Frontend (with hot reload)
cd web
bun run dev

# Terminal 2: Backend (with hot reload)
cd backend
go run ./cmd/openpost
```

The frontend dev server runs on `http://localhost:5173` and proxies API calls to the backend.

## 🐳 Docker

```bash
# Build the image
docker build -t openpost -f docker/Dockerfile .

# Run
docker run -d \
  -p 8080:8080 \
  -v openpost_data:/data \
  -e JWT_SECRET=your-secret \
  -e ENCRYPTION_KEY=your-encryption-key \
  openpost
```

## ⚙️ Configuration

All configuration is done via environment variables or a `.env` file:

| Variable | Required | Description |
|----------|----------|-------------|
| `JWT_SECRET` | ✅ Yes | Secret key for JWT tokens (32+ chars) |
| `ENCRYPTION_KEY` | ✅ Yes | AES-256 key for token encryption |
| `TWITTER_CLIENT_ID` | For X | Twitter OAuth client ID |
| `TWITTER_CLIENT_SECRET` | For X | Twitter OAuth secret |
| `MASTODON_CLIENT_ID` | For Mastodon | Mastodon OAuth client ID |
| `MASTODON_CLIENT_SECRET` | For Mastodon | Mastodon OAuth secret |
| `OPENPOST_PORT` | No | Server port (default: 8080) |
| `OPENPOST_DB_PATH` | No | SQLite database path |
| `OPENPOST_FRONTEND_URL` | No | CORS origin for frontend |

### Generating Secrets

```bash
# Generate JWT secret
openssl rand -base64 32

# Generate encryption key
openssl rand -base64 32
```

### OAuth Setup

#### Twitter/X

1. Go to [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard)
2. Create a new App
3. Enable OAuth 2.0
4. Set callback URL: `http://localhost:8080/api/v1/accounts/x/callback`
5. Copy Client ID and Secret to your `.env`

#### Mastodon

1. Go to your instance Settings → Development
2. Create a new Application
3. Set callback URL: `http://localhost:8080/api/v1/accounts/mastodon/callback`
4. Copy Client ID and Secret to your `.env`

Note: Mastodon supports **any instance** - users enter their instance URL when connecting.

## 🏗️ Tech Stack

**Frontend:**
- SvelteKit 5 (with runes)
- TailwindCSS 4
- Paraglide (i18n)
- Vitest (testing)

**Backend:**
- Go 1.25+ (Echo framework)
- SQLite (Bun ORM)
- Background job queue (SQLite-backed polling)

**Deployment:**
- Single Go binary with embedded static files
- Docker support
- Zero external dependencies (no Redis, no PostgreSQL required)

## 📁 Project Structure

```
openpost/
├── web/                    # SvelteKit frontend
│   ├── src/
│   │   ├── lib/           # Shared utilities, API client
│   │   └── routes/       # SvelteKit routes
│   └── package.json
│
├── backend/
│   ├── cmd/openpost/      # Main entry point
│   └── internal/
│       ├── api/           # HTTP handlers & middleware
│       ├── config/       # Configuration loading
│       ├── database/      # SQLite setup
│       ├── models/        # Bun ORM models
│       ├── queue/         # Background job worker
│       └── services/      # Business logic
│           ├── auth/      # JWT & password handling
│           ├── crypto/    # Token encryption
│           ├── oauth/     # Platform OAuth implementations
│           ├── publisher/ # Post publishing logic
│           └── tokenmanager/ # Token refresh management
│
├── AGENTS.md              # AI agent guidelines
├── PLAN.md                # Implementation roadmap
└── README.md
```

## 🔐 Security

- **Tokens are encrypted at rest** using AES-256-GCM
- **Passwords hashed** with bcrypt
- **JWT authentication** with configurable expiry
- **No external services** - all data stays on your server
- **OAuth PKCE** for Twitter authentication

## 🗺️ Roadmap

See [PLAN.md](PLAN.md) for the complete implementation status and roadmap.

### Current Status (MVP)

- [x] User authentication (register/login)
- [x] Workspace management (multi-tenant)
- [x] Twitter/X OAuth
- [x] Mastodon OAuth (custom instances)
- [x] Post scheduling with background worker
- [x] Single binary deployment

### Coming Soon

- [ ] Bluesky integration (AT Protocol + DPoP)
- [ ] Threads integration (Instagram Graph API)
- [ ] LinkedIn integration
- [ ] Media upload support
- [ ] Post analytics
- [ ] Email notifications
- [ ] Webhook support

## 🤝 Contributing

We welcome contributions! Please see [AGENTS.md](AGENTS.md) for development guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Postiz](https://github.com/gitroomhq/postiz-app) and [Typefully](https://typefully.com)
- Built with [Echo](https://echo.labstack.com/), [SvelteKit](https://kit.svelte.dev/), and [Bun ORM](https://bun.uptrace.dev/)

---

<p align="center">
  Made with ❤️ by the OpenPost community
</p>