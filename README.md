<p align="center">
  <a href="https://github.com/rodrgds/openpost" target="_blank">
    <img alt="OpenPost Logo" src="./web/static/logo.svg" width="280"/>
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
  One lightweight binary. No dependencies. Full control.
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

## ✨ Nix Flakes

OpenPost is available as a Nix flake for easy installation and deployment.

### Prerequisites

The Nix build requires network access for `bun install`. Enable it with:

```bash
# Add to ~/.config/nix/nix.conf
experimental-features = nix-command flakes
```

### Quick Start

**Run directly** (with impure flag for network access):

```bash
nix run --impure github:rodrgds/openpost
```

**Build locally**:

```bash
nix build --impure github:rodrgds/openpost
./result/bin/openpost
```

> **Note:** The `--impure` flag is required because `bun install` needs network access to download dependencies. This is a Nix sandbox limitation.

### Use in Your Flake

Add OpenPost as an input to your `flake.nix`:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    openpost.url = "github:rodrgds/openpost";
  };

  outputs = { self, nixpkgs, openpost }: {
    # For NixOS system configuration
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        # Import the NixOS module
        openpost.nixosModules.default
        
        # Your configuration
        ({ config, pkgs, ... }: {
          services.openpost = {
            enable = true;
            dataDir = "/var/lib/openpost";
          };
          
          # Don't forget to set your secrets!
          environment.systemPackages = [ openpost.packages.x86_64-linux.default ];
        })
      ];
    };
    
    # For development shell
    devShells.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.mkShell {
      buildInputs = [ openpost.packages.x86_64-linux.default ];
    };
  };
}
```

### NixOS Module

For declarative NixOS deployment, import the module from the flake:

```nix
{ inputs, ... }:
{
  imports = [ inputs.openpost.nixosModules.default ];

  services.openpost = {
    enable = true;
    dataDir = "/var/lib/openpost";
    environment = {
      JWT_SECRET = "your-secret";
      ENCRYPTION_KEY = "your-encryption-key";
    };
  };
}
```

The module provides:
- Systemd service for running OpenPost
- Persistent data directory configuration
- Environment variable management

### Available Outputs

```bash
# Show all flake outputs
nix flake show github:rodrgds/openpost

# Build for specific platform
nix build github:rodrgds/openpost#packages.x86_64-linux.default

# Enter development shell
nix develop github:rodrgds/openpost
```

### Template for Your Projects

To use OpenPost as a template for your own project:

```bash
# Use as a template
nix flake init -t github:rodrgds/openpost
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
| `TWITTER_CLIENT_ID` | For X | Twitter OAuth client ID |
| `TWITTER_CLIENT_SECRET` | For X | Twitter OAuth secret |
| `MASTODON_SERVERS` | For Mastodon | JSON array of Mastodon server configs |
| `MASTODON_REDIRECT_URI` | No | OAuth callback URI (default: OOB) |
| `LINKEDIN_CLIENT_ID` | For LinkedIn | LinkedIn OAuth client ID |
| `LINKEDIN_CLIENT_SECRET` | For LinkedIn | LinkedIn OAuth secret |
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
3. Add redirect URL: `http://localhost:8080/api/v1/accounts/linkedin/callback`
4. Copy Client ID and Secret to your `.env`

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
- [x] Mastodon OAuth (multi-server support)
- [x] Bluesky OAuth (AT Protocol)
- [x] LinkedIn OAuth
- [x] Threads OAuth (Meta Graph API)
- [x] Post scheduling with background worker
- [x] Single binary deployment
- [x] Token refresh for all platforms

### Coming Soon

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