# Health Checks

OpenPost exposes:

```txt
GET /api/v1/health
```

Expected response:

```json
{"status":"ok"}
```

For container installs, this endpoint is the right target for Docker health checks and external uptime probes.
