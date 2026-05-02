# Quickstart

This is the fastest path to a working OpenPost instance.

## 1. Create `docker-compose.yml`

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

volumes:
  openpost_data:
```

## 2. Create `.env`

```bash
cp backend/.env.example .env
```

## 3. Generate secrets

```bash
openssl rand -base64 32
openssl rand -base64 32
```

Set the generated values as `JWT_SECRET` and `ENCRYPTION_KEY`.

::: warning
Do not use placeholder secrets in production.
:::

## 4. Start OpenPost

```bash
docker compose up -d
```

## 5. Open the app

Visit `http://localhost:8080`.

## 6. Finish setup

1. Create your OpenPost account.
2. Create or select a workspace.
3. Connect your first provider.
4. Publish a test post.

## Next steps

- [Docker Compose details](/installation/docker-compose)
- [Environment variables](/configuration/environment-variables)
- [Provider setup](/providers/overview)
