<p align="center">
  <a href="https://github.com/rodrgds/openpost">
    <img alt="OpenPost Logo" src="./frontend/static/logo.svg" width="280"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/rodrgds/openpost/releases">
    <img src="https://img.shields.io/github/v/release/rodrgds/openpost?sort=semver&label=Release" alt="Latest Release">
  </a>
  <a href="https://github.com/rodrgds/openpost/pkgs/container/openpost">
    <img src="https://img.shields.io/github/v/release/rodrgds/openpost?sort=semver&label=Image&include_prereleases" alt="Container Image">
  </a>
  <a href="https://github.com/rodrgds/openpost/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/rodrgds/openpost/ci.yml?label=CI" alt="CI Status">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT">
  </a>
  <a href="https://github.com/rodrgds/openpost/blob/main/SECURITY.md">
    <img src="https://img.shields.io/badge/Security-Security%20Policy-blue" alt="Security Policy">
  </a>
</p>

<div align="center">
  <strong>
  <h2>A lightweight, self-hosted social media scheduler</h2>
  </strong>
  Post to X, Mastodon, Bluesky, Threads, and LinkedIn from your own server.<br/>
  One binary or container. Your data stays on your machine.
</div>

<div align="center">
  <br/>
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
  <br/>
  <a href="#-quickstart"><strong>Quickstart</strong></a>
  ·
  <a href="#-deployment-options"><strong>Deployment</strong></a>
  ·
  <a href="#-configuration"><strong>Configuration</strong></a>
  ·
  <a href="#-provider-setup"><strong>Providers</strong></a>
  ·
  <a href="#-security-and-operations"><strong>Operations</strong></a>
  ·
  <a href="#-contributing"><strong>Contributing</strong></a>
</p>

<br/>

## Screenshots

<!-- Add screenshots here. Recommended: dashboard, compose post, accounts page, scheduled queue -->
<!-- ![OpenPost Dashboard](./assets/screenshots/dashboard.png) -->
<!-- ![Compose Post](./assets/screenshots/compose.png) -->

_Screenshots coming soon. The UI includes a dashboard, compose post flow, account management, and scheduled queue view._

## Why OpenPost

- **Self-hosted** — Your data stays on your server. No third-party tracking.
- **Single binary** — Everything compiled into one executable. No external dependencies.
- **SQLite-backed** — The scheduling queue survives restarts. No Redis required.
- **Multi-platform** — Post to X, Mastodon, Bluesky, Threads, and LinkedIn.
- **Encrypted tokens** — All OAuth tokens encrypted at rest with AES-256-GCM.
- **Threading support** — Create multi-post threads that publish sequentially.

## Quickstart

### Fastest path with Docker Compose

1. Create a `docker-compose.yml` file:

```yaml
services:
  openpost:
    image: ghcr.io/rodrgds/openpost:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file:
      - .env
    volumes:
      - openpost_data:/data
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  openpost_data:
```

2. Copy the example environment file:

```bash
curl -L -o .env https://raw.githubusercontent.com/rodrgds/openpost/main/backend/.env.example
```

3. Edit `.env` and set required secrets:

```bash
# Generate secure secrets
openssl rand -base64 32
```

Required variables:
- `JWT_SECRET` — Secret key for JWT tokens (32+ chars)
- `ENCRYPTION_KEY` — AES-256 key for token encryption

Add only the provider credentials you actually need (see [Provider Setup](#-provider-setup)).

4. Start the stack:

```bash
docker compose up -d
```

5. Open `http://localhost:8080` and create your account.

### Docker run

```bash
# Create persistent volume
docker volume create openpost_data

# Run the container
docker run -d \
  --name openpost \
  --restart unless-stopped \
  -p 8080:8080 \
  --mount source=openpost_data,target=/data \
  --env-file .env \
  ghcr.io/rodrgds/openpost:latest
```

### Single binary

Download a release from [GitHub Releases](https://github.com/rodrgds/openpost/releases), or build from source:

```bash
# Clone and build
git clone https://github.com/rodrgds/openpost.git
cd openpost

# Build frontend
cd frontend && bun install && bun run build && cd ..

# Build backend (embeds frontend)
cd backend
cp .env.example .env
# Edit .env with your credentials
go build -o openpost ./cmd/openpost

# Run
./openpost
```

The application runs on `http://localhost:8080`.

## Deployment Options

| Method | Best for | Status |
|--------|----------|--------|
| [Docker Compose](#quickstart) | Single-host production, homelabs | **Recommended** |
| [Docker run](#docker-run) | Quick evaluation | Supported |
| [Single binary](#single-binary) | Bare-metal, VMs | Supported |
| [Source build](#single-binary) | Contributors, development | Supported |
| [Kubernetes](#kubernetes) | Clustered environments | Reference manifest |

### Kubernetes

A baseline reference manifest for Kubernetes:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: openpost
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: openpost-config
  namespace: openpost
data:
  OPENPOST_PORT: "8080"
  OPENPOST_DB_PATH: "/data/db/openpost.db"
  OPENPOST_MEDIA_PATH: "/data/media"
---
apiVersion: v1
kind: Secret
metadata:
  name: openpost-secrets
  namespace: openpost
type: Opaque
stringData:
  JWT_SECRET: "replace-me"
  ENCRYPTION_KEY: "replace-me"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: openpost-data
  namespace: openpost
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: openpost
  namespace: openpost
spec:
  replicas: 1
  selector:
    matchLabels:
      app: openpost
  template:
    metadata:
      labels:
        app: openpost
    spec:
      containers:
        - name: openpost
          image: ghcr.io/rodrgds/openpost:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: openpost-config
            - secretRef:
                name: openpost-secrets
          volumeMounts:
            - name: data
              mountPath: /data
          startupProbe:
            httpGet:
              path: /api/v1/health
              port: 8080
            failureThreshold: 30
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /api/v1/health
              port: 8080
          readinessProbe:
            httpGet:
              path: /api/v1/health
              port: 8080
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            fsGroup: 1000
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: openpost-data
---
apiVersion: v1
kind: Service
metadata:
  name: openpost
  namespace: openpost
spec:
  selector:
    app: openpost
  ports:
    - port: 80
      targetPort: 8080
```

**Note:** This is a baseline reference. Production clusters should add ingress/TLS, registry authentication, backup jobs, resource limits, and secret management integration.

## Configuration

### Required secrets

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret key for JWT tokens. Generate with `openssl rand -base64 32` |
| `ENCRYPTION_KEY` | AES-256 key for encrypting OAuth tokens at rest. Generate with `openssl rand -base64 32` |

### Common settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENPOST_PORT` | `8080` | Server port |
| `OPENPOST_DB_PATH` | `openpost.db` | SQLite database path (relative to `/data/db/`) |
| `OPENPOST_MEDIA_PATH` | `./media` | Media storage path |
| `OPENPOST_MEDIA_URL` | `/media` | URL path for serving media |
| `OPENPOST_FRONTEND_URL` | `http://localhost:8080` | CORS origin for frontend |
| `OPENPOST_CORS_EXTRA_ORIGINS` | (none) | Additional CORS origins (comma-separated) |

### Provider configuration

| Provider | Variables | Setup Required |
|----------|-----------|----------------|
| X (Twitter) | `TWITTER_CLIENT_ID`, `TWITTER_CLIENT_SECRET` | OAuth app |
| Mastodon | `MASTODON_SERVERS` (JSON) | OAuth app per instance |
| Bluesky | (none) | App password only |
| LinkedIn | `LINKEDIN_CLIENT_ID`, `LINKEDIN_CLIENT_SECRET` | OAuth app + approval |
| Threads | `THREADS_CLIENT_ID`, `THREADS_CLIENT_SECRET` | Meta app |

Bluesky requires no environment variables — users connect with their handle and app password directly in the UI.

See [backend/.env.example](backend/.env.example) for the full configuration template with provider-specific details.

## Provider Setup

### X (Twitter)

1. Go to [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard)
2. Create a new App with OAuth 2.0 enabled
3. Set callback URL: `https://your-domain.com/api/v1/accounts/x/callback`
4. Request scopes: `tweet.read`, `tweet.write`, `users.read`, `offline.access`
5. Copy Client ID and Secret to your `.env`

### Mastodon

Mastodon requires per-instance OAuth apps because client credentials are server-specific.

1. For each Mastodon instance, go to **Settings → Development → New Application**
2. Set redirect URI: `urn:ietf:wg:oauth:2.0:oob` (or your production callback URL)
3. Request scopes: `read`, `write`
4. Add to your `.env`:

```env
MASTODON_REDIRECT_URI=https://your-domain.com/api/v1/accounts/mastodon/callback
MASTODON_SERVERS='[
  {"name":"Personal","client_id":"xxx","client_secret":"yyy","instance_url":"https://mastodon.social"},
  {"name":"Work","client_id":"aaa","client_secret":"bbb","instance_url":"https://fosstodon.org"}
]'
```

The `name` is your label for the server. Add as many as you need.

### Bluesky

No setup required. Users create an app password in [Bluesky Settings](https://bsky.app/settings/app-passwords) and enter it directly in OpenPost.

### LinkedIn

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/apps)
2. Create an app and request **Share on LinkedIn** product
3. Request approval for `w_member_social_feed` in your app permissions
4. Add redirect URL: `https://your-domain.com/api/v1/accounts/linkedin/callback`
5. Copy Client ID and Secret to your `.env`

**Important:** Thread replies (posting as comments on the first post) require `w_member_social_feed` approval. Without it, replies fail with `ACCESS_DENIED`. If you can't get approval, set `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES=true`.

### Threads

1. Go to [Meta for Developers](https://developers.facebook.com/)
2. Create a Business app and add **Threads API** product
3. Add redirect URL: `https://your-domain.com/api/v1/accounts/threads/callback`
4. Copy App ID and App Secret as `THREADS_CLIENT_ID` and `THREADS_CLIENT_SECRET`

**Important:** Threads requires publicly accessible URLs for media uploads. Your `/media/{id}` endpoint must be reachable from the internet.

See [docs/](docs/) for detailed platform-specific documentation.

## Security and Operations

### Secrets

- **Never commit `.env`** to version control
- Use Docker secrets, Kubernetes secrets, or a secrets manager in production
- Rotate OAuth client secrets and app passwords periodically

### Persistence

OpenPost stores all data under `/data`:

- `/data/db/openpost.db` — SQLite database
- `/data/media/` — Uploaded images and videos

**Backup both the database and media directory.**

### Backups

```bash
# Backup database
cp /data/db/openpost.db openpost-backup-$(date +%Y%m%d).db

# Backup media
tar -czf media-backup-$(date +%Y%m%d).tar.gz /data/media/
```

### Upgrades

1. Check the [Changelog](CHANGELOG.md) for breaking changes
2. Back up your `/data` directory
3. Pull the new image or download the new binary
4. Restart the service
5. Verify health: `curl http://localhost:8080/api/v1/health`

### Vulnerability Reporting

See [SECURITY.md](SECURITY.md) for the security policy and disclosure process.

## Reverse Proxy and TLS

### Caddy (recommended)

```Caddyfile
openpost.yourdomain.com {
    reverse_proxy localhost:8080
    encode gzip
}
```

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name openpost.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /media/ {
        proxy_pass http://localhost:8080;
    }
}
```

### Traefik

```yaml
services:
  openpost:
    image: ghcr.io/rodrgds/openpost:latest
    # ...
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.openpost.rule=Host(`openpost.yourdomain.com`)"
      - "traefik.http.routers.openpost.tls=true"
      - "traefik.http.services.openpost.loadbalancer.server.port=8080"
```

### OAuth callback URLs

When using a reverse proxy, update your OAuth callback URLs:

| Provider | Callback URL |
|----------|---------------|
| X | `https://your-domain.com/api/v1/accounts/x/callback` |
| Mastodon | `https://your-domain.com/api/v1/accounts/mastodon/callback` |
| LinkedIn | `https://your-domain.com/api/v1/accounts/linkedin/callback` |
| Threads | `https://your-domain.com/api/v1/accounts/threads/callback` |

Update the corresponding environment variables (`TWITTER_REDIRECT_URI`, etc.) in your `.env`.

## Tech Stack

**Frontend:**
- SvelteKit 5 (with runes)
- TailwindCSS 4
- Paraglide (i18n)

**Backend:**
- Go 1.25+ (Echo framework)
- SQLite (Bun ORM)
- Background job queue (SQLite-backed)

**Deployment:**
- Single Go binary with embedded static files
- Docker container image
- No external dependencies (no Redis, no PostgreSQL)

## Project Structure

```
openpost/
├── frontend/               # SvelteKit frontend
│   ├── src/
│   │   ├── lib/           # API client, components, stores
│   │   └── routes/        # SvelteKit routes
│   └── android/           # Android app (Capacitor)
├── backend/
│   ├── cmd/openpost/       # Main entry point
│   └── internal/
│       ├── api/            # HTTP handlers
│       ├── config/         # Configuration
│       ├── database/       # SQLite setup
│       ├── models/         # ORM models
│       ├── platform/       # Platform adapters (X, Mastodon, Bluesky, etc.)
│       ├── queue/          # Background worker
│       └── services/       # Business logic
├── docs/                   # Platform integration docs
├── docker-compose.yml      # Quickstart Compose file
├── CHANGELOG.md            # Version history
└── README.md
```

## Contributing

We welcome contributions! Please read our contributing guidelines:

- [CONTRIBUTING.md](CONTRIBUTING.md) — How to contribute
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — Community standards
- [AGENTS.md](AGENTS.md) — Developer guidelines and architecture

### Quick contribution steps

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features
   - `fix:` for bug fixes
   - `chore:` for maintenance
   - `refactor:` for code improvements
6. Open a Pull Request

## Roadmap and Releases

- [CHANGELOG.md](CHANGELOG.md) — What's changed in each release
- [ROADMAP.md](ROADMAP.md) — Upcoming features and status
- [GitHub Releases](https://github.com/rodrgds/openpost/releases) — Download binaries and images

## License

MIT — see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Postiz](https://github.com/gitroomhq/postiz-app) and [Typefully](https://typefully.com)
- Built with [Echo](https://echo.labstack.com/), [SvelteKit](https://kit.svelte.dev/), and [Bun ORM](https://bun.uptrace.dev/)

---

<p align="center">
  Made with ❤️ by the OpenPost community
</p>