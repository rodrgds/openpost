# Environment Variables

This page summarizes the env vars used by the backend. Some values in `backend/.env.example` are recommended deployment examples; code defaults may differ.

## Core settings

| Variable | Required | Default | Description |
|---|---:|---|---|
| `OPENPOST_PORT` | No | `8080` | HTTP server port. |
| `OPENPOST_DATABASE_PATH` | No | `file:openpost.db?cache=shared&mode=rwc` | SQLite database path or DSN. |
| `OPENPOST_APP_URL` | No, but set it in real deployments | `http://localhost:8080` | Public frontend origin used for CORS and auth flow assumptions. |
| `OPENPOST_PUBLIC_URL` | No | falls back to `OPENPOST_APP_URL` | Canonical browser origin used when configuring WebAuthn/passkeys. Set this to your real app URL in production. |
| `OPENPOST_EXTRA_CORS_ORIGINS` | No | empty | Extra comma-separated origins to allow. |
| `OPENPOST_DISABLE_REGISTRATIONS` | No | `false` | Disables new self-service signups after setup. The first account on a fresh instance is still allowed and becomes the instance admin automatically. |
| `OPENPOST_JWT_SECRET` | Yes | none | Secret used to sign JWTs. Must be at least 32 characters. |
| `OPENPOST_ENCRYPTION_KEY` | Yes | none | Secret used to encrypt stored OAuth tokens. Must be at least 32 characters. |
| `OPENPOST_MEDIA_PATH` | No | `./media` | Local directory for uploaded media. |
| `OPENPOST_MEDIA_URL` | No, but required for Threads production use | `/media` | Public base URL for media files. |
| `OPENPOST_ENV` | No | empty | Optional deployment label. Secret validation is enforced regardless of environment mode. |

## X

| Variable | Required | Default | Description |
|---|---:|---|---|
| `X_CLIENT_ID` | Yes for X | empty | X OAuth client ID. |
| `X_CLIENT_SECRET` | Yes for X | empty | X OAuth client secret. |
| `X_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/x/callback` | X OAuth 1.0a callback URL override. |

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
| `LINKEDIN_DISABLE_THREAD_REPLIES` | No | `false` | Disable LinkedIn comment-style child replies for thread posts. |

## Threads

| Variable | Required | Default | Description |
|---|---:|---|---|
| `THREADS_CLIENT_ID` | Yes for Threads | empty | Meta app ID. |
| `THREADS_CLIENT_SECRET` | Yes for Threads | empty | Meta app secret. |
| `THREADS_REDIRECT_URI` | No | `http://localhost:8080/api/v1/accounts/threads/callback` | Threads callback URL override. |

## Notes

- The preferred names above are what new deployments should use.
- Backward-compatible aliases still work for existing installs: `OPENPOST_DB_PATH`, `OPENPOST_FRONTEND_URL`, `OPENPOST_CORS_EXTRA_ORIGINS`, `JWT_SECRET`, `ENCRYPTION_KEY`, `TWITTER_CLIENT_ID`, `TWITTER_CLIENT_SECRET`, `TWITTER_REDIRECT_URI`, and `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES`.
- `backend/.env.example` is still the best copy-paste starting point.
- Set explicit public URLs in production even when defaults exist.
- For Threads, treat `OPENPOST_MEDIA_URL` as mandatory.
