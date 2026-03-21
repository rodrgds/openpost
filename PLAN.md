# OpenPost - Comprehensive Research & Planning Document

## Executive Summary

OpenPost is envisioned as a lightweight, self-hosted, open-source alternative to tools like Typefully for scheduling posts across multiple social media platforms. Designed for extreme simplicity in deployment (a single Go binary) while maintaining high reliability through a custom SQLite-backed queue.

This document merges the foundational research with the **actual current state of the codebase** to outline what exists, what is half-finished, and a concrete roadmap to a production-ready v1.

---

## 1. Architecture Overview (Current & Target)

- **Frontend:** SvelteKit (SPA output embedded into Go binary). Styling via TailwindCSS.
- **Backend API:** Go + **Echo** framework (chosen over Fiber for standard HTTP compatibility & mature middleware ecosystem).
- **Database:** SQLite (configured with PRAGMA WAL) managed by **Bun** ORM.
- **Background Queue:** A custom **SQLite-backed polling worker** (avoids needing Redis or volatile in-memory cron).
- **Authentication:** Standard Email/Password for users, OAuth 2.0 / AT Protocol for connecting social accounts.
- **Multi-Tenant:** Full **Workspace UI** from day one to support collaborative teams, roles, and separated social accounts.

---

## 2. Current State vs. Missing Functionality

Based on an audit of the repository, here is the exact state of the project.

### 2.1 Backend / Infrastructure

| Area | Status | Notes / Missing Elements |
|---|---|---|
| **API Framework** | 🟡 Half-finished | `Echo` is set up in `main.go`. Some handlers (OAuth, Post, Workspace) are stubbed but missing core logic and imports (`context`, `time`, etc.). |
| **Database Schema** | 🟢 Mostly Complete | `models.go` has models for Users, Workspaces, Posts, SocialAccounts, Jobs. Missing some ORM tags for cascading deletes. |
| **Background Queue** | 🟡 Half-finished | `worker.go` correctly implements basic polling locking via SQLite. The actual `executeJob` delegates to empty/missing publishers. |
| **Authentication (User)** | 🔴 Missing | The `users` table exists, but there are no API routes for Register, Login (JWT/Session), or middleware to protect routes. |
| **Workspace Logic** | 🔴 Missing | Handlers are stubbed, but no complex logic exists to invite users, switch contexts, or manage roles ('admin', 'editor'). |
| **Social OAuth (Twitter)** | 🟡 Half-finished | `twitter.go` and `oauth.go` handlers exist but lack proper state validation and error handling. |
| **Social OAuth (Mastodon)**| 🟡 Half-finished | Similar to Twitter, stubbed but untested. Needs proper storage of instance URLs. |
| **Token Encryption** | 🟢 Implemented | `encrypt.go` provides AES-256-GCM encryption for storing OAuth tokens securely. |
| **Token Management** | 🔴 Missing | Need logic to actively refresh Twitter/Threads tokens before they expire and insert `refresh_token` jobs into the queue. |
| **Media Storage** | 🟡 Half-finished | Local storage interface stubbed. S3 interface completely missing. |

### 2.2 Frontend (SvelteKit)

| Area | Status | Notes / Missing Elements |
|---|---|---|
| **SvelteKit Init** | 🟢 Implemented | The `web/` directory has a basic SvelteKit skeleton with Vite and Tailwind/Paraglide setup. |
| **Authentication UI** | 🔴 Missing | No Login, Registration, or Password Reset forms. |
| **Workspace Management** | 🔴 Missing | No UI to create workspaces, switch between them, or manage team members. |
| **Social Connections UI** | 🔴 Missing | No dashboard to connect Twitter, Mastodon, etc., via the backend OAuth routes. |
| **Post Composer** | 🔴 Missing | The core feature! Needs a rich text editor, thread builder, media uploader, and platform selector. |
| **Schedule / Queue UI** | 🔴 Missing | Needs a list/calendar view to show upcoming posts from the database. |

---

## 3. Implementation Roadmap

To turn this repository into a "really big open source project," we need a systematic approach to complete the MVP.

### Phase 1: Core Foundation & Auth (Weeks 1-2)
*Goal: Fix compilation errors, solidify the database, and allow users to create accounts and workspaces.*

- [ ] **Fix Build Errors:** Clean up unused imports and syntax errors in `backend/internal/api/handlers/oauth.go` and `main.go`.
- [ ] **User Authentication API:** Implement `/api/v1/auth/register`, `/login`, and `/me` using JWT or cookie-based sessions.
- [ ] **Auth Middleware:** Create Echo middleware to extract the user session and inject the User ID and active Workspace ID into the request context.
- [ ] **Workspace API:** Complete CRUD operations for Workspaces and Workspace Members.
- [ ] **Frontend Auth:** Build Login/Register pages in SvelteKit and a Workspace switcher component.

### Phase 2: Social OAuth & Token Management (Weeks 3-4)
*Goal: Allow users to securely connect social accounts and maintain their access.*

- [ ] **Twitter & Mastodon Polish:** Finalize the OAuth handlers. Ensure the `workspace_id` state is securely passed and validated.
- [ ] **Frontend Integration:** Build the "Social Accounts" settings page in SvelteKit where users can click "Connect Twitter", etc.
- [ ] **Proactive Token Refresh:** Implement a chron job (via the SQLite queue or a simple backend goroutine) that scans for tokens expiring soon and refreshes them via the `TokenManager`.
- [ ] **Threads & Bluesky Support:** Expand the OAuth handlers to support Threads (Instagram Graph API) and Bluesky (AT Protocol with DPoP).

### Phase 3: The Post Composer & Queue (Weeks 5-6)
*Goal: The core workflow of writing a post, scheduling it, and having the background worker publish it.*

- [ ] **Post APIs:** Complete CRUD endpoints for `Posts` and `PostDestinations`.
- [ ] **Composer UI:** Build a Svelte component for drafting a post, selecting target platforms (from the connected social accounts), and picking a scheduled time.
- [ ] **Scheduler Integration:** When a post is scheduled, the backend must insert a `publish_post` job into the `jobs` table with `run_at` set to the scheduled time.
- [ ] **Publisher Service:** Implement the actual API calls to Twitter, Mastodon, etc., in `services/publisher`. Ensure it handles rate limits by throwing specific errors so the background worker can back off and retry.

### Phase 4: Media & Polish (Weeks 7-8)
*Goal: Support image/video uploads and make the app production-ready.*

- [ ] **Media Upload API:** Endpoint to accept multipart form uploads, save to local disk, and insert into `media_attachments`.
- [ ] **SvelteKit Uploader:** Drag-and-drop media attachment UI in the post composer.
- [ ] **S3 Integration (Optional MVP):** Add an S3-compatible backend for the BlobStorage interface for scalable deployments.
- [ ] **Single Binary Pipeline:** Finalize the Makefile / generate steps to ensure `bun run build` in `/web` perfectly embeds into `/backend/cmd/openpost/public` for the `go build`.

---

## 4. Known Complexities & "Gotchas"

1. **Bluesky (AT Protocol):** Implementing Bluesky requires handling DPoP (Demonstrating Proof of Possession), which requires rotating nonces and JWT proofs per request. This will be the hardest platform to integrate.
2. **Threads 60-Day Expiry:** Threads tokens max out at 60 days and must be actively refreshed. If a user doesn't log in, the backend must automatically renew it.
3. **SQLite Concurrency:** Because the app uses SQLite as a queue, we must rely on `PRAGMA WAL` and `busy_timeout` to avoid `database is locked` errors when the background worker and user APIs hit the DB simultaneously.
4. **Single Binary Static Assets:** SvelteKit routing can be tricky when embedded in Echo. We need to ensure the Echo static file handler correctly falls back to `index.html` for client-side routing (SPA fallback).