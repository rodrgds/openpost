# Docker Compose Reference

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
      - OPENPOST_DATABASE_PATH=/data/db/openpost.db
      - OPENPOST_MEDIA_PATH=/data/media
      - OPENPOST_MEDIA_URL=https://openpost.example.com/media
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  openpost_data:
```
