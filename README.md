<p align="center">
  <a href="https://github.com/rodrgds/openpost" target="_blank">
    <img alt="OpenPost Logo" src="./frontend/static/logo.svg" width="280"/>
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
  <h2>A lightweight, self-hosted social media scheduler with web and Android app</h2><br />
  </strong>
  The open-source alternative to Typefully, Buffer, and Hypefury.<br />
  One lightweight binary. No dependencies. Full control.
</div>

<div align="center">
  <br />
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/x-white.svg">
    <img alt="X (Twitter)" src="./assets/logos/x.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/mastodon-white.svg">
    <img alt="Mastodon" src="./assets/logos/mastodon.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/bluesky-white.svg">
    <img alt="Bluesky" src="./assets/logos/bluesky.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/threads-white.svg">
    <img alt="Threads" src="./assets/logos/threads.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/linkedin-white.svg">
    <img alt="LinkedIn" src="./assets/logos/linkedin.svg" width="24">
  </picture>
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
- **Media Uploads** - Attach images and videos to posts. Platform-specific media requirements handled automatically.
- **Post Threading** - Create multi-post threads that publish sequentially across all platforms.
- **Custom Mastodon Instances** - Connect accounts from any number of Mastodon servers (configured via JSON env var).
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
| Bluesky | ✅ Implemented | App passwords (no setup) |
| Threads | ✅ Implemented | Meta Graph API |
| LinkedIn | ✅ Implemented | OAuth 2.0 + Posts API |

## ⚡ Quick Start

### Prerequisites

- Go 1.25+ (for building from source)
- Bun or npm (for frontend builds)
- OAuth credentials for the platforms you want to use

### Building from Source

```bash
# Clone the repository
git clone https://github.com/rodrgds/openpost.git
cd openpost

# Install frontend dependencies and build
cd frontend
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
cd frontend && bun install && bun run build && cd ../backend && go build -o openpost ./cmd/openpost
```

### Development Mode

For development, you can run the frontend and backend separately:

```bash
# Terminal 1: Frontend (with hot reload)
cd frontend
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

## 📦 Docker Registry

### Building and Pushing

```bash
# Build the image
docker build -t ghcr.io/rodrgds/openpost:latest -f docker/Dockerfile .

# Tag for version
docker tag ghcr.io/rodrgds/openpost:latest ghcr.io/rodrgds/openpost:v0.1.0

# Push to registry
docker push ghcr.io/rodrgds/openpost:latest
docker push ghcr.io/rodrgds/openpost:v0.1.0
```

### Using Pre-built Image

```yaml
# docker-compose.yml
services:
  openpost:
    image: ghcr.io/rodrgds/openpost:latest
    restart: unless-stopped
    environment:
      - JWT_SECRET=your-secret
      - ENCRYPTION_KEY=your-encryption-key
    volumes:
      - openpost_data:/data
    ports:
      - "8080:8080"
```

## ⚙️ Configuration

All configuration is done via environment variables or a `.env` file:

| Variable | Required | Description |
|----------|----------|-------------|
| `JWT_SECRET` | ✅ Yes | Secret key for JWT tokens (32+ chars) |
| `ENCRYPTION_KEY` | ✅ Yes | AES-256 key for token encryption |
| `OPENPOST_MEDIA_PATH` | No | Local media storage path (default: `./media`) |
| `OPENPOST_MEDIA_URL` | No | URL path for serving media (default: `/media`) |
| `TWITTER_CLIENT_ID` | For X | Twitter OAuth client ID |
| `TWITTER_CLIENT_SECRET` | For X | Twitter OAuth secret |
| `MASTODON_SERVERS` | For Mastodon | JSON array of Mastodon server configs |
| `MASTODON_REDIRECT_URI` | No | OAuth callback URI (default: OOB) |
| `LINKEDIN_CLIENT_ID` | For LinkedIn | LinkedIn OAuth client ID |
| `LINKEDIN_CLIENT_SECRET` | For LinkedIn | LinkedIn OAuth secret |
| `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES` | No | Disable LinkedIn thread child replies when app lacks `w_member_social_feed` |
| `THREADS_CLIENT_ID` | For Threads | Meta App ID |
| `THREADS_CLIENT_SECRET` | For Threads | Meta App Secret |
| `OPENPOST_PORT` | No | Server port (default: 8080) |
| `OPENPOST_DB_PATH` | No | SQLite database path |
| `OPENPOST_FRONTEND_URL` | No | CORS origin for frontend |

**Note:** Bluesky doesn't require any env vars - users connect directly with their handle and app password.

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

Mastodon supports **multiple servers** via the `MASTODON_SERVERS` environment variable. Each server needs its own OAuth app because Mastodon client credentials are per-instance.

1. For each Mastodon instance, go to Settings → Development → New Application
2. Set the redirect URI to: `urn:ietf:wg:oauth:2.0:oob` (or your callback URL)
3. Copy the Client ID and Secret
4. Add to your `.env`:

```env
MASTODON_REDIRECT_URI=urn:ietf:wg:oauth:2.0:oob
MASTODON_SERVERS='[
  {"name":"Personal","client_id":"abc123","client_secret":"xyz789","instance_url":"https://mastodon.social"},
  {"name":"Work","client_id":"def456","client_secret":"uvw012","instance_url":"https://fosstodon.org"}
]'
```

The `name` is a label shown in the UI. You can configure as many servers as you need.

#### Bluesky

Bluesky uses app passwords - no setup needed. Users just enter their handle and app password when connecting:

1. Go to [Bluesky App Passwords](https://bsky.app/settings/app-passwords)
2. Create a new app password
3. In OpenPost, click Connect on Bluesky and enter your handle + app password

See [docs/bluesky-integration.md](docs/bluesky-integration.md) for more details.

#### LinkedIn

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/apps)
2. Create a new app and request "Share on LinkedIn" product
3. Request approval for `w_member_social_feed` (Social Actions create) in your app products/permissions
4. Add redirect URL: `http://localhost:8080/api/v1/accounts/linkedin/callback`
5. Copy Client ID and Secret to your `.env`

**Important:** LinkedIn thread replies (posting comments on the first post) require `w_member_social_feed` approval. If your app only has `w_member_social`, first posts may succeed but replies/comments will fail with `ACCESS_DENIED: partnerApiSocialActions.CREATE`.

See [docs/linkedin-integration.md](docs/linkedin-integration.md) for detailed setup instructions.

#### Threads

1. Go to [Meta for Developers](https://developers.facebook.com/)
2. Create a new app (Business type) and add Threads API product
3. Add redirect URL: `http://localhost:8080/api/v1/accounts/threads/callback`
4. Copy App ID and App Secret to your `.env` as `THREADS_CLIENT_ID` and `THREADS_CLIENT_SECRET`

See [docs/threads-integration.md](docs/threads-integration.md) for detailed setup instructions.

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
├── frontend/                  # SvelteKit frontend (web + Android app)
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/       # API client (openapi-fetch)
│   │   │   ├── components/# UI components
│   │   │   └── stores/    # Auth, UI state
│   │   └── routes/        # SvelteKit routes
│   ├── android/            # Android native app (Capacitor)
│   └── package.json
│
├── backend/
│   ├── cmd/openpost/       # Main entry point
│   │   └── public/        # Embedded SvelteKit build output (not source)
│   ├── internal/
│   │   ├── api/            # HTTP handlers & middleware
│   │   │   ├── handlers/  # Posts, Auth, Media, OAuth handlers
│   │   │   └── middleware/
│   │   │       └── auth.go# JWT authentication middleware
│   │   ├── config/         # Configuration loading
│   │   ├── database/       # SQLite setup
│   │   ├── models/         # Bun ORM models
│   │   │   ├── models.go
│   │   │   └── models_test.go
│   │   ├── platform/       # Platform adapter interface + implementations
│   │   │   ├── adapter.go # PlatformAdapter interface
│   │   │   ├── http.go    # Shared HTTP helpers
│   │   │   ├── x.go       # Twitter/X adapter
│   │   │   ├── mastodon.go# Mastodon adapter
│   │   │   ├── bluesky.go # Bluesky adapter
│   │   │   ├── linkedin.go# LinkedIn adapter
│   │   │   └── threads.go # Threads adapter
│   │   ├── queue/          # Background job worker
│   │   └── services/       # Business logic
│   │       ├── auth/       # JWT & password handling
│   │       │   ├── auth.go
│   │       │   └── auth_test.go
│   │       ├── crypto/     # Token encryption
│   │       │   ├── encrypt.go
│   │       │   └── encrypt_test.go
│   │       ├── mediastore/ # Local/S3 media storage
│   │       ├── publisher/  # Post publishing logic
│   │       └── tokenmanager/ # Token refresh management
│   ├── .golangci.yml       # Linter configuration
│   ├── go.mod              # Go module definition
│   └── go.sum              # Go module checksums
│
├── docs/                   # Platform integration docs
├── AGENTS.md               # AI agent guidelines
├── PLAN.md                 # Implementation roadmap
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
- [x] Mastodon OAuth (multi-server support)
- [x] Bluesky OAuth (AT Protocol)
- [x] LinkedIn OAuth
- [x] Threads OAuth (Meta Graph API)
- [x] Post scheduling with background worker
- [x] Media upload support
- [x] Post threading (multi-post threads)
- [x] Single binary deployment
- [x] Token refresh for all platforms
- [x] Platform adapter architecture

### Coming Soon

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
