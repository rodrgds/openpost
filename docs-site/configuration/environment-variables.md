# Environment Variables

This page summarizes the env vars used by the backend. Some values in `backend/.env.example` are recommended deployment examples; code defaults may differ.

## Core settings

| Variable | Required | Default | Description |
|---|---:|---|---|
| `OPENPOST_PORT` | No | `8080` | HTTP server port. |
| `OPENPOST_DB_PATH` | No | `file:openpost.db?cache=shared&mode=rwc` | SQLite database path or DSN. |
| `OPENPOST_FRONTEND_URL` | No, but set it in real deployments | `http://localhost:5173` | Public frontend origin used for CORS and auth flow assumptions. |
| `OPENPOST_CORS_EXTRA_ORIGINS` | No | empty | Extra comma-separated origins to allow. |
| `JWT_SECRET` | Yes for production | development fallback in code | Secret used to sign JWTs. |
| `ENCRYPTION_KEY` | Yes for production | development fallback in code | Secret used to encrypt stored OAuth tokens. |
| `OPENPOST_MEDIA_PATH` | No | `./media` | Local directory for uploaded media. |
| `OPENPOST_MEDIA_URL` | No, but required for Threads production use | `/media` | Public base URL for media files. |
| `OPENPOST_ENV` | No | empty | Set to `production` or `prod` to enforce production secret validation. |

## X

| Variable | Required | Default | Description |
|---|---:|---|---|
| `TWITTER_CLIENT_ID` | Yes for X | empty | X OAuth client ID. |
| `TWITTER_CLIENT_SECRET` | Yes for X | empty | X OAuth client secret. |
| `TWITTER_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/x/callback` | X callback URL override. |

## Mastodon

| Variable | Required | Default | Description |
|---|---:|---|---|
| `MASTODON_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/mastodon/callback` | Mastodon callback URL override. |
| `MASTODON_SERVERS` | Yes for Mastodon | empty | JSON array of configured Mastodon apps and instance URLs. |

## LinkedIn

| Variable | Required | Default | Description |
|---|---:|---|---|
| `LINKEDIN_CLIENT_ID` | Yes for LinkedIn | empty | LinkedIn OAuth client ID. |
| `LINKEDIN_CLIENT_SECRET` | Yes for LinkedIn | empty | LinkedIn OAuth client secret. |
| `LINKEDIN_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/linkedin/callback` | LinkedIn callback URL override. |
| `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES` | No | `false` | Disable LinkedIn comment-style child replies for thread posts. |

## Threads

| Variable | Required | Default | Description |
|---|---:|---|---|
| `THREADS_CLIENT_ID` | Yes for Threads | empty | Meta app ID. |
| `THREADS_CLIENT_SECRET` | Yes for Threads | empty | Meta app secret. |
| `THREADS_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/threads/callback` | Threads callback URL override. |

## Notes

- `backend/.env.example` is still the best copy-paste starting point.
- Set explicit public URLs in production even when defaults exist.
- For Threads, treat `OPENPOST_MEDIA_URL` as mandatory.
