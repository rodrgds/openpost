# Docker Compose

Docker Compose is the recommended installation path for long-running OpenPost deployments.

## Prerequisites

- Docker Engine
- Docker Compose
- A writable persistent volume or bind mount for `/data`

## Create `docker-compose.yml`

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
      - OPENPOST_MEDIA_URL=http://localhost:8080/media
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  openpost_data:
```

## Create `.env`

```bash
cp backend/.env.example .env
```

Set at least:

- `JWT_SECRET`
- `ENCRYPTION_KEY`
- Provider credentials for the networks you want to enable

## Generate secrets

```bash
openssl rand -base64 32
```

Generate one value for `JWT_SECRET` and another for `ENCRYPTION_KEY`.

## Start OpenPost

```bash
docker compose up -d
```

## Check health

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:

```json
{"status":"ok"}
```

## Where data is stored

- Database: `/data/db/openpost.db`
- Media: `/data/media`

Do not store either on ephemeral container storage.

## Upgrade flow

```bash
docker compose pull
docker compose up -d
docker compose logs -f openpost
```

## Production warnings

- Put OpenPost behind HTTPS before enabling OAuth in production.
- Set `OPENPOST_FRONTEND_URL` and `OPENPOST_MEDIA_URL` to public URLs.
- Back up both the SQLite database and media directory.

## Next steps

- [Reverse proxy](/installation/reverse-proxy)
- [Production checklist](/configuration/production-checklist)
- [Provider setup](/providers/overview)
