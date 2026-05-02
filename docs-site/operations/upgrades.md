# Upgrades

## Docker Compose

```bash
docker compose pull
docker compose up -d
docker compose logs -f openpost
```

## Checklist

- Read the changelog
- Back up `/data`
- Pull the new image or binary
- Restart OpenPost
- Check `/api/v1/health`
- Inspect the scheduled queue and recent logs
