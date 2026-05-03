# Environment Variables

This is the low-level quick reference version of the configuration docs.

| Variable | Purpose |
|---|---|
| `OPENPOST_PORT` | Backend port |
| `OPENPOST_DATABASE_PATH` | SQLite path or DSN |
| `OPENPOST_APP_URL` | Public frontend URL |
| `OPENPOST_PUBLIC_URL` | Canonical browser origin used for WebAuthn/passkeys |
| `OPENPOST_EXTRA_CORS_ORIGINS` | Extra CORS allowlist |
| `OPENPOST_DISABLE_REGISTRATIONS` | Disable new signups after bootstrap |
| `OPENPOST_JWT_SECRET` | JWT signing secret |
| `OPENPOST_ENCRYPTION_KEY` | OAuth token encryption secret |
| `OPENPOST_MEDIA_PATH` | Local media directory |
| `OPENPOST_MEDIA_URL` | Public media base URL |
| `X_CLIENT_ID` | X client ID |
| `X_CLIENT_SECRET` | X client secret |
| `X_REDIRECT_URI` | X callback override |
| `MASTODON_REDIRECT_URI` | Mastodon callback override |
| `MASTODON_SERVERS` | Mastodon server JSON |
| `LINKEDIN_CLIENT_ID` | LinkedIn client ID |
| `LINKEDIN_CLIENT_SECRET` | LinkedIn client secret |
| `LINKEDIN_REDIRECT_URI` | LinkedIn callback override |
| `LINKEDIN_DISABLE_THREAD_REPLIES` | Disable LinkedIn thread replies |
| `THREADS_CLIENT_ID` | Threads client ID |
| `THREADS_CLIENT_SECRET` | Threads client secret |
| `THREADS_REDIRECT_URI` | Threads callback override |

Legacy aliases still work for upgrades: `OPENPOST_DB_PATH`, `OPENPOST_FRONTEND_URL`, `OPENPOST_CORS_EXTRA_ORIGINS`, `JWT_SECRET`, `ENCRYPTION_KEY`, `TWITTER_CLIENT_ID`, `TWITTER_CLIENT_SECRET`, `TWITTER_REDIRECT_URI`, and `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES`.
