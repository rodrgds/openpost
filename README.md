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
  <a href="SECURITY.md">
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
  <a href="#quickstart"><strong>Quickstart</strong></a>
  ·
  <a href="#deployment-options"><strong>Deployment</strong></a>
  ·
  <a href="#configuration"><strong>Configuration</strong></a>
  ·
  <a href="#provider-setup"><strong>Providers</strong></a>
  ·
  <a href="#operations"><strong>Operations</strong></a>
  ·
  <a href="#development"><strong>Development</strong></a>
  ·
  <a href="#contributing"><strong>Contributing</strong></a>
</p>

## Screenshots

_Screenshots coming soon. The UI includes a dashboard, compose flow, account management, and scheduled queue view._

## Why OpenPost

- Self-hosted: your data stays on your server.
- Single binary or container: no Redis, no Postgres, no external queue.
- SQLite-backed scheduling: queued posts survive restarts.
- Multi-platform publishing: X, Mastodon, Bluesky, Threads, and LinkedIn.
- Encrypted tokens: OAuth tokens are encrypted at rest with AES-256-GCM.
- Thread support: publish multi-post threads in sequence.

## Quickstart

Docker Compose is the recommended way to run OpenPost.

1. Create a `docker-compose.yml` file:

```yaml
services:
  openpost:
    image: ghcr.io/rodrgds/openpost:latest
    container_name: openpost
    restart: unless-stopped
    env_file:
      - .env
    ports:
      - "8080:8080"
    volumes:
      - openpost_data:/data
    environment:
      - OPENPOST_PORT=8080
      - OPENPOST_DB_PATH=/data/db/openpost.db
      - OPENPOST_MEDIA_PATH=/data/media
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  openpost_data:
```

2. Copy the example environment file:

```bash
cp backend/.env.example .env
```

3. Edit `.env` and set at least:

- `JWT_SECRET`
- `ENCRYPTION_KEY`

Generate both with:

```bash
openssl rand -base64 32
```

4. Start OpenPost:

```bash
docker compose up -d
```

5. Open `http://localhost:8080`.

6. Create your account and connect only the providers you need.

### Docker run

```bash
docker volume create openpost_data

docker run -d \
  --name openpost \
  --restart unless-stopped \
  -p 8080:8080 \
  --mount source=openpost_data,target=/data \
  --env-file .env \
  -e OPENPOST_DB_PATH=/data/db/openpost.db \
  -e OPENPOST_MEDIA_PATH=/data/media \
  ghcr.io/rodrgds/openpost:latest
```

### Single binary

Download the binary for your platform from [GitHub Releases](https://github.com/rodrgds/openpost/releases), make it executable, then run it with a `.env` file:

```bash
cp backend/.env.example .env
chmod +x ./openpost
./openpost
```

If you prefer, you can also build from source:

```bash
git clone https://github.com/rodrgds/openpost.git
cd openpost/frontend
bun install
bun run build

cd ../backend
cp .env.example .env
go build -o openpost ./cmd/openpost
./openpost
```

The app runs on `http://localhost:8080` by default.

## Deployment Options

| Method | Best for | Status |
|--------|----------|--------|
| [Docker Compose](#quickstart) | Single-host production, homelabs | Recommended |
| [Docker run](#docker-run) | Quick evaluation | Supported |
| [Single binary](#single-binary) | Bare-metal, VMs, simple installs | Supported |
| [Source build](#single-binary) | Contributors and local development | Supported |

## Configuration

See [backend/.env.example](backend/.env.example) for the complete template.

### Required secrets

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret key for JWT tokens |
| `ENCRYPTION_KEY` | AES-256 key for encrypting stored OAuth tokens |

### Common settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENPOST_PORT` | `8080` | HTTP server port |
| `OPENPOST_DB_PATH` | `file:openpost.db?cache=shared&mode=rwc` | SQLite database path |
| `OPENPOST_FRONTEND_URL` | `http://localhost:8080` | Frontend origin |
| `OPENPOST_CORS_EXTRA_ORIGINS` | empty | Additional allowed origins |
| `OPENPOST_MEDIA_PATH` | `./media` | Local media storage path |
| `OPENPOST_MEDIA_URL` | `http://localhost:8080/media` | Public media URL |

### Production notes

- Set `OPENPOST_FRONTEND_URL` to your real public URL.
- Set `OPENPOST_MEDIA_URL` to a public HTTPS URL if you use Threads.
- Update provider callback URLs to use your public domain.
- Persist both the database and media directories.
- Keep `.env` out of version control.

### Provider summary

| Provider | Setup needed | Notes |
|----------|--------------|-------|
| X | OAuth app | Uses callback URL and client credentials |
| Mastodon | App per instance | Multi-instance JSON config via `MASTODON_SERVERS` |
| Bluesky | App password | No provider env vars required |
| LinkedIn | OAuth app | Replies may require extra approval |
| Threads | Meta app | Public media URL required |

## Provider Setup

### X

1. Create an app in the [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard).
2. Enable OAuth 2.0.
3. Set callback URL to `https://your-domain.com/api/v1/accounts/x/callback`.
4. Add `TWITTER_CLIENT_ID` and `TWITTER_CLIENT_SECRET` to `.env`.

### Mastodon

Mastodon uses per-instance OAuth apps.

1. Create an app on each instance you want to support.
2. Set the redirect URI.
3. Add `MASTODON_REDIRECT_URI` and `MASTODON_SERVERS` to `.env`.

Example:

```env
MASTODON_REDIRECT_URI=https://your-domain.com/api/v1/accounts/mastodon/callback
MASTODON_SERVERS='[
  {"name":"Personal","client_id":"xxx","client_secret":"yyy","instance_url":"https://mastodon.social"}
]'
```

### Bluesky

No server-side OAuth setup is required. Users connect with their handle and app password from [Bluesky Settings](https://bsky.app/settings/app-passwords).

### LinkedIn

1. Create an app in the [LinkedIn Developer Portal](https://www.linkedin.com/developers/apps).
2. Request the required posting permissions.
3. Set callback URL to `https://your-domain.com/api/v1/accounts/linkedin/callback`.
4. Add `LINKEDIN_CLIENT_ID` and `LINKEDIN_CLIENT_SECRET` to `.env`.

If your app cannot get reply/comment approval, set `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES=true`.

### Threads

1. Create a Business app in [Meta for Developers](https://developers.facebook.com/).
2. Add the Threads API product.
3. Set callback URL to `https://your-domain.com/api/v1/accounts/threads/callback`.
4. Add `THREADS_CLIENT_ID`, `THREADS_CLIENT_SECRET`, and `THREADS_REDIRECT_URI` to `.env`.

Threads media uploads require `OPENPOST_MEDIA_URL` to be publicly reachable.

More details live in [docs/](docs/).

## Operations

### Data storage

OpenPost stores persistent data in:

- Database: `/data/db/openpost.db`
- Media: `/data/media/`

Back up both.

### Backup example

```bash
cp /data/db/openpost.db openpost-backup-$(date +%Y%m%d).db
tar -czf media-backup-$(date +%Y%m%d).tar.gz /data/media/
```

### Upgrades

1. Read [CHANGELOG.md](CHANGELOG.md).
2. Back up `/data`.
3. Pull the new image or binary.
4. Restart the service.
5. Check `http://localhost:8080/api/v1/health`.

### Security

- Do not commit `.env`.
- Rotate provider credentials periodically.
- Read [SECURITY.md](SECURITY.md) for reporting and disclosure.

### Public URL and callbacks

If you run OpenPost behind your own domain, make sure:

- `OPENPOST_FRONTEND_URL` matches the public URL users visit.
- `OPENPOST_MEDIA_URL` is publicly reachable if you use Threads media uploads.
- Provider callback URLs use your public HTTPS domain.

## Development

### Tech stack

- Frontend: SvelteKit, TailwindCSS, Paraglide
- Backend: Go, Echo, Bun ORM, SQLite
- Deployment: embedded static frontend in a single Go binary

### Project structure

```text
openpost/
├── frontend/               # SvelteKit frontend
├── backend/                # Go backend
├── docs/                   # Platform-specific docs
├── docker-compose.yml      # Recommended local/prod starter
├── CHANGELOG.md
└── README.md
```

## Contributing

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [AGENTS.md](AGENTS.md)
- [ROADMAP.md](ROADMAP.md)

Use Conventional Commits for changes that land in the repo.

## License

MIT. See [LICENSE](LICENSE).
