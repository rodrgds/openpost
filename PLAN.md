# OpenPost - Development Plan & Status

## Overview

OpenPost is a lightweight, self-hosted social media scheduler — an open-source alternative to Typefully, Buffer, and Hypefury. Single Go binary with embedded SvelteKit frontend. SQLite-backed everything. No external dependencies.

---

## Current Status

### Backend

| Area | Status | Details |
|------|--------|---------|
| API Framework | ✅ Done | Echo + Huma (OpenAPI). All handlers registered with middleware. |
| Database Schema | ✅ Done | Bun ORM models for Users, Workspaces, Posts, SocialAccounts, Jobs. SQLite with WAL pragma. |
| Background Queue | ✅ Done | SQLite-backed polling worker. Picks up scheduled jobs and publishes posts. |
| User Auth (JWT) | ✅ Done | Register, Login, `/me` endpoints. Bearer token auth middleware. |
| Workspace CRUD | ✅ Done | Create, list workspaces. Multi-tenant architecture. |
| Twitter OAuth | ✅ Done | OAuth 2.0 PKCE flow. Token exchange and storage. |
| Mastodon OAuth | ✅ Done | Multi-server support via `MASTODON_SERVERS` JSON env var. Per-instance credentials. |
| Token Encryption | ✅ Done | AES-256-GCM encryption for all OAuth tokens at rest. |
| Token Management | ✅ Done | Token refresh logic for expiring tokens. |
| Post Publishing | ✅ Done | Background worker publishes to X and Mastodon via their APIs. |
| Post Scheduling | ✅ Done | Posts saved with `scheduled_at`, jobs table drives the queue. |
| SPA Embedding | ✅ Done | SvelteKit static output embedded in Go binary via `go:embed`. |

### Frontend

| Area | Status | Details |
|------|--------|---------|
| SvelteKit Setup | ✅ Done | Svelte 5 runes, TailwindCSS 4, Paraglide i18n. |
| Auth UI | ✅ Done | Login and Register pages with JWT storage. |
| Workspace Switcher | ✅ Done | Dropdown to create and switch workspaces. |
| Accounts Page | ✅ Done | Connect Twitter, Mastodon (multi-server). Shows connected accounts. |
| Post Composer | ✅ Done | Rich text editor with platform selection and scheduling. |
| Schedule Calendar | ✅ Done | Monthly overview with per-platform breakdown. |
| Dashboard | ✅ Done | Workspace overview, recent posts, schedule summary. |

### What's Working End-to-End

1. User registers → logs in → creates workspace
2. Connects Twitter account via OAuth
3. Connects one or more Mastodon servers (configured in env)
4. Composes a post, selects target accounts, picks a schedule time
5. Background worker publishes at the scheduled time
6. Tokens auto-refresh when expiring

---

## What's Left

### Platform Integrations

| Platform | Status | Approach | Notes |
|----------|--------|----------|-------|
| X (Twitter) | ✅ Done | OAuth 2.0 PKCE | Single API, any account connects. |
| Mastodon | ✅ Done | OAuth 2.0 (multi-server) | Each instance needs its own app registration. JSON config in env. |
| Bluesky | 🔜 Planned | App Passwords (AT Protocol) | No OAuth needed. User provides handle + app password. PDS derived from handle. Simple single-config. |
| Threads | 🔜 Planned | Instagram Graph API | OAuth 2.0. 60-day token expiry requires active refresh. |
| LinkedIn | 🔜 Planned | OAuth 2.0 | Standard OAuth. |

### Features

- [ ] **Bluesky Integration**
  - App password auth (handle + password)
  - Post via AT Protocol API (`com.atproto.repo.createRecord`)
  - No multi-server config needed — handle determines PDS automatically

- [ ] **Threads Integration**
  - Instagram Graph API OAuth
  - 60-day token expiry with automatic refresh
  - Post creation via `/me/threads` endpoint

- [ ] **LinkedIn Integration**
  - OAuth 2.0 authorization code flow
  - Post creation via UGC API

- [ ] **Media Upload Support**
  - Image upload in post composer
  - Video upload support
  - Local file storage with configurable path (`OPENPOST_MEDIA_PATH`)
  - Optional S3-compatible storage backend
  - Platform-specific media requirements (size limits, formats)

- [ ] **Thread/Long-form Support**
  - Thread builder for X (multi-tweet threads)
  - Long-form post support for Mastodon
  - Character count per platform with live validation

- [ ] **Post Templates & Auto-Plug**
  - Save and reuse post templates
  - Auto-plug: append a call-to-action or link to posts automatically
  - Per-workspace template library

- [ ] **Post Analytics**
  - Fetch engagement metrics (likes, retweets, replies) from platform APIs
  - Display stats in dashboard
  - Track performance over time

- [ ] **Email Notifications**
  - Notify when a post is published
  - Notify on failed publishes
  - Configurable notification preferences

- [ ] **Webhook Support**
  - Outbound webhooks on post events (published, failed)
  - Webhook management API

- [ ] **Team & Role Management**
  - Invite users to workspaces
  - Role-based access (admin, editor, viewer)
  - Activity log

- [ ] **Rate Limit Handling**
  - Detect rate limit responses from platform APIs
  - Exponential backoff retry in background worker
  - User-facing rate limit warnings

- [ ] **Draft Management**
  - Save posts as drafts without scheduling
  - Draft list / management page
  - Convert draft to scheduled post

- [ ] **Bulk Scheduling**
  - CSV import for bulk post scheduling
  - Batch operations on posts

---

## Architecture Decisions

### Why JSON env vars for Mastodon?

Mastodon OAuth apps are per-instance. Unlike X (single API) or Bluesky (app passwords), each Mastodon server requires its own client ID and secret. A `MASTODON_SERVERS` JSON array is the cleanest way to support this without config files:

```env
MASTODON_SERVERS='[
  {"name":"Personal","client_id":"abc","client_secret":"xyz","instance_url":"https://mastodon.social"},
  {"name":"Work","client_id":"def","client_secret":"uvw","instance_url":"https://fosstodon.org"}
]'
```

Bluesky and X don't need this — they use a single set of credentials.

### Why SQLite for everything?

Single binary, zero dependencies. SQLite with WAL mode handles concurrent reads/writes well for a self-hosted tool. The background queue uses the same SQLite database with row-level locking for job coordination.

---

## Known Complexities

1. **Bluesky AT Protocol:** Simple auth (app passwords), but posting requires correct record schemas (`app.bsky.feed.post`). Rich text facets for mentions/links need careful byte-offset handling.
2. **Threads 60-Day Expiry:** Long-lived tokens require proactive refresh. The token manager must scan and renew before expiry.
3. **Media Size Limits:** Each platform has different limits (X: 5MB images, 512MB video; Mastodon: varies by instance). Validation must be per-platform.
4. **Character Limits:** X (280), Mastodon (500 default, configurable), Threads (500), LinkedIn (3000). Composer needs live per-platform counting.
