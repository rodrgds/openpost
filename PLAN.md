# OpenPost - Development Plan & Status

## Overview

OpenPost is a lightweight, self-hosted social media scheduler — an open-source alternative to Typefully, Buffer, and Hypefury. Single Go binary with embedded SvelteKit frontend. SQLite-backed everything. No external dependencies.

---

## Current Status

### Backend

| Area | Status | Details |
|------|--------|---------|
| API Framework | ✅ Done | Echo + Huma (OpenAPI). All handlers registered with middleware. |
| Database Schema | ✅ Done | Bun ORM models for Users, Workspaces, Posts, SocialAccounts, Jobs, MediaAttachments, PostMedia. SQLite with WAL pragma. |
| Background Queue | ✅ Done | SQLite-backed polling worker. Picks up scheduled jobs and publishes posts. |
| User Auth (JWT) | ✅ Done | Register, Login, `/me` endpoints. Bearer token auth middleware. |
| Workspace CRUD | ✅ Done | Create, list workspaces. Multi-tenant architecture. |
| Platform Adapters | ✅ Done | Unified `PlatformAdapter` interface in `internal/platform/`. All 5 platforms implemented. |
| Twitter/X | ✅ Done | OAuth 2.0 PKCE. Media upload (chunked). Threading via `reply.in_reply_to_tweet_id`. |
| Mastodon | ✅ Done | Multi-server OAuth. Media upload (async poll). Threading via `in_reply_to_id`. |
| Bluesky | ✅ Done | App password auth. Blob upload. Threading via `reply: {root, parent}` with uri+cid. |
| LinkedIn | ✅ Done | OAuth 2.0. Vector Assets media upload. Threading via Comments API. |
| Threads | ✅ Done | Meta OAuth 2.0. Public URL media. Threading via `reply_to_id`. |
| Token Encryption | ✅ Done | AES-256-GCM encryption for all OAuth tokens at rest. |
| Token Management | ✅ Done | Token refresh logic for expiring tokens. Uses adapter map (no switch). |
| Post Publishing | ✅ Done | Background worker publishes to all 5 platforms. Thread-aware sequential publishing. |
| Post Scheduling | ✅ Done | Posts saved with `scheduled_at`, jobs table drives the queue. |
| Media Upload | ✅ Done | `POST /media/upload` + `GET /media/:id`. Local filesystem storage. |
| Post Threading | ✅ Done | `POST /posts/thread` endpoint. Sequential publishing with parent tracking. |
| SPA Embedding | ✅ Done | SvelteKit static output embedded in Go binary via `go:embed`. |

### Frontend

| Area | Status | Details |
|------|--------|---------|
| SvelteKit Setup | ✅ Done | Svelte 5 runes, TailwindCSS 4, Paraglide i18n. |
| UI Components | ✅ Done | shadcn-svelte component library (button, dialog, select, drawer, tooltip, textarea, sheet, label, dropdown-menu, checkbox, calendar, avatar). |
| i18n Languages | ✅ Done | English (en), Spanish (es), Portuguese (pt) via Paraglide. |
| Auth UI | ✅ Done | Login (`/login`) and Register (`/register`) pages with JWT storage. |
| Workspace Switcher | ✅ Done | Dropdown to create and switch workspaces. |
| Accounts Page | ✅ Done | Connect all 5 platforms. Shows connected accounts (`/accounts`). |
| Post Composer | ✅ Done | Text editor with platform selection and scheduling. |
| Media Upload UI | ✅ Done | Drag-and-drop zone, previews, alt text editor. |
| Thread Builder | ✅ Done | Toggle thread mode, add/remove posts, connector lines. |
| Schedule Calendar | ✅ Done | Monthly overview with per-platform/workspace breakdown. |
| Dashboard | ✅ Done | Workspace overview, recent posts, schedule summary (`/`). |
| Mobile Support | ✅ Done | Capacitor wrapper for Android. Full Android project in `frontend/android/`. |

### Frontend Routes

| Route | Page |
|-------|------|
| `/` | Dashboard (home) |
| `/login` | Login page |
| `/register` | Register page |
| `/accounts` | Connected accounts management |
| `/connect` | Platform connection hub |
| `/accounts/mastodon/callback` | Mastodon OAuth callback |

### What's Working End-to-End

1. User registers → logs in → creates workspace
2. Connects accounts on all 5 platforms (X, Mastodon, Bluesky, LinkedIn, Threads)
3. Composes a post, attaches media, selects target accounts, picks a schedule time
4. Background worker publishes at the scheduled time (media uploaded to each platform)
5. Tokens auto-refresh when expiring
6. Create multi-post threads that publish sequentially with reply chains

---

## What's Left

### Features

- [ ] Post analytics (engagement metrics)
- [ ] Email notifications
- [ ] Webhook support
- [ ] Draft management
- [ ] Bulk scheduling (CSV import)
- [ ] Post templates & auto-plug
- [ ] Rate limit handling
- [ ] Team & role management
- [ ] iOS Capacitor support (Android done)

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

### Why a PlatformAdapter interface?

Eliminates 5 switch statements across the codebase. Adding a new platform = implement the interface in one file + register it in main.go. No changes to publisher, token manager, or OAuth handler.

### Why SQLite for everything?

Single binary, zero dependencies. SQLite with WAL mode handles concurrent reads/writes well for a self-hosted tool. The background queue uses the same SQLite database with row-level locking for job coordination.

---

## Known Complexities

1. **Bluesky AT Protocol:** Simple auth (app passwords), but posting requires correct record schemas (`app.bsky.feed.post`). Threading needs both root and parent AT-URIs with CIDs.
2. **Threads 60-Day Expiry:** Long-lived tokens require proactive refresh. The token manager must scan and renew before expiry.
3. **Threads Media:** Requires publicly accessible URLs. Won't work with localhost — needs ngrok for local dev.
4. **LinkedIn Threading:** Not native threads. Subsequent posts appear as comments on the first post via Comments API.
5. **Media Size Limits:** Each platform has different limits (X: 5MB images, 512MB video; Mastodon: varies by instance).
6. **X Media ID Expiry:** Media IDs expire ~2 hours after upload. Must upload at publish time, not creation time.
7. **Character Limits:** X (280), Mastodon (500 default, configurable), Threads (500), Bluesky (300), LinkedIn (3000).
